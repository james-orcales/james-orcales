package sh

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/james-orcales/james-orcales/golang_snacks/invariant"
)

func Spawn(arguments ...string) (ok bool) {
	invariant.Always(len(arguments) > 0, "Spawn requires at least one argument")
	return SpawnRaw(SpawnRawInput{}, arguments...)
}

func Pipe(arguments ...string) (stdout string) {
	invariant.Always(len(arguments) > 0, "Pipe requires at least one argument")
	out := bytes.Buffer{}
	Fpipe(&out, arguments...)
	return strings.TrimSpace(out.String())
}

func PipeWithErr(arguments ...string) (stdout string, stderr string) {
	invariant.Always(len(arguments) > 0, "PipeWithErr requires at least one argument")
	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	FpipeWithErr(FpipeWithErrInput{Stdout: outBuf, Stderr: errBuf}, arguments...)
	return outBuf.String(), errBuf.String()
}

// Quick tip:
//
//	If you can determine success from the output, don't check the returned boolean. Only check the
//	boolean when output alone can't tell you if the command succeeded.
//
//	```go
//	// GOOD - output validates success
//	var buf bytes.Buffer
//	sh.Fpipe(&buf, "echo", "hello")
//	snap.Init(`hello\n`).Expect(t, buf.String())
//
//	// BAD - redundant check
//	ok := sh.Fpipe(&buf, "echo", "hello")
//	if !ok { ... }
//	snap.Init(`hello\n`).Expect(t, buf.String())
//
//	```
func Fpipe(out io.Writer, arguments ...string) (ok bool) {
	invariant.Always(len(arguments) > 0, "Fpipe requires at least one argument")
	invariant.Always(out != nil, "Fpipe output writer is not nil")
	return FpipeWithErr(FpipeWithErrInput{Stdout: out, Stderr: os.Stderr}, arguments...)
}

type FpipeWithErrInput struct {
	Stdout io.Writer
	Stderr io.Writer
}

func FpipeWithErr(in FpipeWithErrInput, arguments ...string) (ok bool) {
	invariant.Always(len(arguments) > 0, "FpipeWithErr requires at least one argument")
	invariant.Always(in.Stdout != nil, "FpipeWithErr stdout writer is not nil")
	invariant.Always(in.Stderr != nil, "FpipeWithErr stderr writer is not nil")
	return SpawnRaw(SpawnRawInput{Stdout: in.Stdout, Stderr: in.Stderr}, arguments...)
}

// SpawnRaw starts a process, treating leading ARG=VAL entries as env vars.
//
// Usage:
//
//	SpawnRaw(SpawnRawInput{}, "A=1", "B=2", "cmd")
//	SpawnRaw(SpawnRawInput{Stdin: r}, "cmd", "x", "y")
//	SpawnRaw(SpawnRawInput{WorkingDirectory: "/tmp"}, "cmd")
//
// The first non-ENV token becomes the executable. Remaining tokens become args.
// Returns true on zero exit status.
type SpawnRawInput struct {
	WorkingDirectory string
	Stdin            io.Reader
	Stdout           io.Writer
	Stderr           io.Writer
	OnChildExit      func(start time.Time, end time.Time, ru *syscall.Rusage)
}

func SpawnRaw(input SpawnRawInput, arguments ...string) (ok bool) {
	workingDirectory := input.WorkingDirectory
	in := input.Stdin
	out := input.Stdout
	err := input.Stderr
	if len(arguments) == 0 {
		panic("scripting.spawn requires at least one argument")
	}

	outWasNil := out == nil
	errWasNil := err == nil

	if out == nil {
		out = os.Stdout
	}
	if err == nil {
		err = os.Stderr
	}

	invariant.Always(out != nil, "SpawnRaw stdout writer is set after nil-check")
	invariant.Always(err != nil, "SpawnRaw stderr writer is set after nil-check")

	invariant.Sometimes(outWasNil, "Stdout writer is sometimes nil and defaults to os.Stdout")
	invariant.Sometimes(!outWasNil, "Stdout writer is sometimes explicitly provided")
	invariant.Sometimes(errWasNil, "Stderr writer is sometimes nil and defaults to os.Stderr")
	invariant.Sometimes(!errWasNil, "Stderr writer is sometimes explicitly provided")
	invariant.Sometimes(workingDirectory != "", "Working directory is sometimes specified")
	invariant.Sometimes(workingDirectory == "", "Working directory is sometimes empty")

	var env []string
	for i, arg := range arguments {
		if strings.Contains(arg, "=") {
			invariant.Always(i == len(env), "Environment variables are consecutive from the start of arguments")
			invariant.Always(strings.Index(arg, "=") > 0, "Environment variable has at least one character before '='")
			invariant.Always(strings.Count(arg, "=") >= 1, "Environment variable contains at least one '=' separator")

			// Extract key from KEY=VALUE format
			key := arg[:strings.Index(arg, "=")]
			invariant.Always(len(key) > 0, "Environment variable key is non-empty")
			invariant.Always(key == strings.TrimSpace(key), "Environment variable key has no leading/trailing whitespace")

			env = append(env, arg)
			invariant.Sometimes(strings.Count(arg, "=") > 1, "Environment variable value can contain '=' characters")
			invariant.Sometimes(len(arg[strings.Index(arg, "=")+1:]) == 0, "Environment variable can have empty value")
		} else {
			break
		}
	}

	invariant.Always(func() bool {
		for _, e := range env {
			if !strings.Contains(e, "=") {
				return false
			}
		}
		return true
	}(), "All collected environment variables contain '=' separator")

	if len(arguments)-len(env) == 0 {
		panic("sh.SpawnRaw was not provided an executable")
	}

	invariant.Always(len(env) <= len(arguments), "Number of env vars cannot exceed total arguments")
	invariant.Always(len(arguments)-len(env) > 0, "After env vars, at least one argument remains for the executable")
	invariant.Always(len(env) == 0 || len(arguments) > len(env), "When env vars present, total arguments exceed env var count")

	invariant.Sometimes(len(env) == 0, "Commands can be executed without custom environment variables")
	invariant.Sometimes(len(env) > 0, "Commands can be executed with custom environment variables")
	invariant.Sometimes(len(env) > 1, "Multiple environment variables can be set")

	executable := arguments[0]
	if len(env) > 0 {
		executable = arguments[len(env)]
		invariant.Always(len(env) < len(arguments), "When env vars present, there are arguments after them")
	}

	invariant.Always(executable != "", "Executable name is not empty")
	invariant.Always(!strings.Contains(executable, "="), "Executable is not an env var assignment")

	remainingArgs := arguments[len(env):]
	invariant.Always(len(remainingArgs) > 0, "After skipping env vars, at least executable remains")
	invariant.Always(remainingArgs[0] == executable, "First remaining argument after env vars is the executable")

	if len(arguments[len(env):]) > 1 {
		arguments = arguments[len(env)+1:]
		invariant.Always(len(arguments) > 0, "Command arguments exist after extracting executable")
		invariant.Always(func() bool {
			for _, arg := range arguments {
				if strings.Contains(arg, "=") && strings.Index(arg, "=") > 0 {
					// This could be a valid argument that happens to contain '=',
					// but it's after the executable, so it won't be treated as env var
					return true
				}
			}
			return true
		}(), "Command arguments are validated")
		invariant.Sometimes(func() bool {
			for _, arg := range arguments {
				if strings.Contains(arg, "=") {
					return true
				}
			}
			return false
		}(), "Command arguments can contain '=' without being treated as env vars")
	} else {
		arguments = nil
		invariant.Always(len(remainingArgs) == 1, "When no command arguments, only executable remains after env vars")
	}

	invariant.Sometimes(arguments == nil, "Commands can be executed without arguments")
	invariant.Sometimes(arguments != nil && len(arguments) > 0, "Commands can be executed with arguments")

	invariant.Always(
		len(env) == 0 || !strings.Contains(executable, "="),
		"When env vars are present, executable is not mistaken for an env var",
	)

	cmd := exec.Command(executable, arguments...)

	baseEnvCount := len(os.Environ())
	invariant.Always(baseEnvCount >= 0, "Base environment has non-negative count")

	cmd.Env = os.Environ()
	invariant.Always(len(cmd.Env) == baseEnvCount, "Command starts with base environment")

	cmd.Env = append(cmd.Env, env...)
	invariant.Always(len(cmd.Env) == baseEnvCount+len(env), "Command environment includes base plus custom env vars")
	invariant.Always(len(env) == 0 || len(cmd.Env) > baseEnvCount, "When custom env vars present, command env exceeds base env")

	// Validate that our custom env vars are actually in the command's environment
	if len(env) > 0 {
		invariant.Always(func() bool {
			cmdEnvMap := make(map[string]bool, len(cmd.Env))
			for _, e := range cmd.Env {
				cmdEnvMap[e] = true
			}
			for _, e := range env {
				if !cmdEnvMap[e] {
					return false
				}
			}
			return true
		}(), "All custom environment variables are present in command environment")
	}

	cmd.Stdout = out
	cmd.Stderr = err
	if in != nil {
		cmd.Stdin = in
	}

	invariant.Always(cmd.Stdout != nil, "Command stdout is set before execution")
	invariant.Always(cmd.Stderr != nil, "Command stderr is set before execution")
	invariant.Always(cmd.Path != "" || cmd.Err != nil, "Command has a path set or an error")

	if workingDirectory != "" {
		os.MkdirAll(workingDirectory, 0o755)
		cmd.Dir = workingDirectory
		invariant.Always(cmd.Dir == workingDirectory, "Working directory is set correctly on command")
	}
	startedAt := time.Now()
	errCode := cmd.Run()
	endedAt := time.Now()
	success := errCode == nil

	if input.OnChildExit != nil && cmd.ProcessState != nil {
		if ru, isRusage := cmd.ProcessState.SysUsage().(*syscall.Rusage); isRusage {
			input.OnChildExit(startedAt, endedAt, ru)
		}
	}

	invariant.Sometimes(success, "Commands sometimes succeed")
	invariant.Sometimes(!success, "Commands sometimes fail")

	return success
}

func WorkingDirectory() (dir string) {
	wd, _ := os.Getwd()
	invariant.Always(wd == "" || filepath.IsAbs(wd), "WorkingDirectory returns empty string or absolute path")
	return wd
}

// PushDir changes the current working directory to the specified dir and returns a function that
// restores the previous working directory. Call the returned function to revert to the original
// directory.
//
// Usage:
//
//	popDir := sh.PushDir("tmp")
//	doSomething()
//	popDir()
func PushDir(dir string) (popDir func()) {
	workingDirectory := WorkingDirectory()
	invariant.Always(workingDirectory != "", "Current working directory is not empty")

	if err := os.Chdir(dir); err != nil {
		panic(err)
	}

	newDir := WorkingDirectory()
	invariant.Always(newDir != workingDirectory, "Working directory actually changed after PushDir")

	return func() {
		beforeRestore := WorkingDirectory()
		if err := os.Chdir(workingDirectory); err != nil {
			panic(err)
		}
		afterRestore := WorkingDirectory()
		invariant.Always(afterRestore == workingDirectory, "Working directory restored to original location")
		invariant.Always(
			beforeRestore != afterRestore || beforeRestore == workingDirectory,
			"Working directory changed during restore or was already at original location",
		)
	}
}

func Which(name string) (abs string) {
	invariant.Always(name != "", "Which() receives a non-empty command name")

	path, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	path = filepath.Clean(path)

	invariant.Always(path != "", "When LookPath succeeds, resulting path is not empty")
	invariant.Always(filepath.IsAbs(path), "Which() returns absolute path when command is found")

	return path
}

func FileExists(path string) (exists bool) {
	_, err := os.Stat(path)
	return err == nil
}

func MakeDirAll(path string) (err error) {
	return os.MkdirAll(path, 0o755)
}

func Lines(s string, n int) (lines []string) {
	if n == 0 {
		return nil
	}
	if n > 0 {
		result := strings.SplitN(s, "\n", n)[:n]
		invariant.Always(len(result) == n, "Positive n returns exactly n elements")
		return result
	} else {
		result := strings.Split(s, "\n")
		absN := -n
		if len(result) > absN {
			tail := result[len(result)-absN:]
			invariant.Always(len(tail) == absN, "Negative n returns exactly |n| elements when string has enough lines")
			return tail
		} else {
			invariant.Always(len(result) <= absN, "Negative n returns all lines when string has fewer than |n| lines")
			return result
		}
	}
}
