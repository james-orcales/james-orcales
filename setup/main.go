// Package main binds the operating-system capabilities for the setup command.
package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"local/james-orcales/setup/internal"
	sysio "local/james-orcales/shared/io"
	system_io "local/james-orcales/shared/io/default"
	jlogcore "local/james-orcales/shared/jlog"
	jlog "local/james-orcales/shared/jlog/default"
	timeos "local/james-orcales/shared/time/default"
)

// EXIT_USAGE identifies an invalid setup execution environment.
const EXIT_USAGE = 2

// DIRECTORY_PERMISSIONS gives new parent directories owner-write access.
const DIRECTORY_PERMISSIONS = 0o755

// COPY_BYTES_MAX bounds one copied font file at 64 MiB.
const COPY_BYTES_MAX = 67108864

// ROOT_REFUSAL prevents the bootstrap from creating root-owned home files.
const ROOT_REFUSAL = "run as your normal user, not root"

func main() {
	clock, _ := timeos.New_Operating_System_Clock()
	logger := jlogcore.New_Console_Logger(jlogcore.New_Console_Logger_Input{
		Console: os.Stderr, Color: is_terminal(os.Stderr),
		Floor: jlogcore.LEVEL_DEBUG, Clock: clock,
	})
	if os.Geteuid() == 0 {
		jlog.Logger_Error(logger, ROOT_REFUSAL)
		os.Exit(EXIT_USAGE)
	}
	home, home_err := os.UserHomeDir()
	if home_err != nil {
		jlog.Logger_Error(logger, "cannot resolve home directory", jlog.Err(home_err))
		os.Exit(EXIT_USAGE)
	}
	loop, driver := system_io.New_Operating_System_IO(clock)
	shell := setup.Shell{
		Spawn:  spawn_command(loop, driver),
		Stdout: os.Stdout, Stderr: os.Stderr, Logger: logger,
	}
	steps := setup.Bootstrap_Steps(&setup.Bootstrap_Steps_Input{
		Home_Directory: home, Operating_System: runtime.GOOS,
		Cargo_Directory: os.Getenv("CARGO_HOME"),
		Data_Directory:  os.Getenv("XDG_DATA_HOME"),
		File_System:     setup.File_System{Loop: loop, Run_Until: driver.Run_Until},
		Shell:           shell, File_Present: file_present, Copy_File: copy_file,
	})
	os.Exit(setup.Bootstrap(&setup.Bootstrap_Input{Steps: steps, Logger: logger}))
}

// Reports whether file is a terminal without an external terminal dependency.
func is_terminal(file *os.File) (terminal bool) {
	info, stat_err := file.Stat()
	if stat_err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// Returns the synchronous process operation that only the composition root drives.
func spawn_command(loop sysio.IO, driver sysio.Driver) (spawn setup.Spawn) {
	return func(request sysio.Process_Request) (result sysio.Process_Result) {
		var completion sysio.Completion
		done := false
		complete := func(
			_ *sysio.Completion, spawned sysio.Process_Result, spawn_err error,
		) {
			if spawn_err != nil {
				spawned.Exit = 1
			}
			result = spawned
			done = true
		}
		loop.Spawn(&completion, complete, request)
		driver.Run_Until(func() (finished bool) { return done }, sysio.FOREVER)
		return result
	}
}

// Reports whether path identifies an existing file.
func file_present(path string) (present bool) {
	_, stat_err := os.Stat(path)
	return stat_err == nil
}

// Copies one bounded file after it creates the destination parent directory.
func copy_file(input *setup.File_Copy_Input) (err error) {
	source, open_err := os.Open(input.Source)
	if open_err != nil {
		return open_err
	}
	defer source.Close()
	mkdir_err := os.MkdirAll(filepath.Dir(input.Destination), DIRECTORY_PERMISSIONS)
	if mkdir_err != nil {
		return mkdir_err
	}
	destination, create_err := os.Create(input.Destination)
	if create_err != nil {
		return create_err
	}
	_, copy_err := io.CopyN(destination, source, COPY_BYTES_MAX)
	if copy_err == nil {
		destination.Close()
		return errors.New("source exceeds the maximum copy size")
	}
	if !errors.Is(copy_err, io.EOF) {
		destination.Close()
		return copy_err
	}
	return destination.Close()
}
