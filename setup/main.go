// Package main binds the operating-system capabilities for the setup command.
package main

import (
	"os"
	"runtime"

	"local/james-orcales/setup/internal"
	sysio "local/james-orcales/shared/io"
	system_io "local/james-orcales/shared/io/default"
	system_time "local/james-orcales/shared/time/default"
)

func main() {
	clock, _ := system_time.New_Operating_System_Clock()
	home, home_err := os.UserHomeDir()
	console, console_err := os.Stderr.Stat()
	color := console_err == nil && console.Mode()&os.ModeCharDevice != 0
	stdout := file_stream(os.Stdout)
	stderr := file_stream(os.Stderr)
	loop, driver, loop_err := system_io.New_Operating_System_IO(
		clock, setup.SCHEDULER_ENTRY_COUNT, setup.SCHEDULER_FLAGS)
	if loop_err != nil {
		diagnostic := []byte("cannot initialize IO: " + loop_err.Error() + "\n")
		written, write_err := sysio.Write(stderr, diagnostic)
		if write_err != nil {
			os.Exit(int(setup.EXIT_USAGE))
		}
		if written != int64(len(diagnostic)) {
			os.Exit(int(setup.EXIT_USAGE))
		}
		os.Exit(int(setup.EXIT_USAGE))
	}
	input := setup.Main_Input{
		Environment: &setup.Environment_Input{
			Clock: clock, Console: stderr,
			Effective_User_Identifier: setup.Effective_User_Identifier(os.Geteuid()),
			Color:                     setup.Console_Color(color),
			Home_Directory:            setup.Home_Directory(home), Home_Error: home_err,
			Operating_System: setup.Operating_System(runtime.GOOS),
			Cargo_Directory:  setup.Cargo_Directory(os.Getenv("CARGO_HOME")),
			Data_Directory:   setup.Data_Directory(os.Getenv("XDG_DATA_HOME")),
			// Shared IO gives a child with no environment nothing at all, so the
			// builds below need the root to read the ambient values one time.
			Process_Environment: setup.Process_Environment(os.Environ()),
			Stdout:              stdout, Stderr: stderr,
		}, IO: loop,
	}
	runner := setup.Main(&input)
	status := drive_runner(driver, runner, stderr)
	deinitialize_driver(driver, runner)
	os.Exit(int(status))
}

// A failed backend remains process-owned because Deinit requires every submission to be joined.
func deinitialize_driver(driver sysio.Driver, runner setup.Runner) {
	if runner.Stopped() {
		driver.Deinit()
	}
}

// The composition root alone binds an operating-system file to the shared stream type.
func file_stream(file *os.File) (stream sysio.Stream) {
	return sysio.Stream{
		Data: file,
		Procedure: func(
			data any, mode sysio.Stream_Mode, buffer []byte,
			_ int64, _ sysio.Seek_From,
		) (count int64, err error) {
			if mode == sysio.STREAM_MODE_QUERY {
				return int64(sysio.Mode_Set_Add(0, sysio.STREAM_MODE_WRITE)), nil
			}
			// Unsupported modes fail because console output is a write-only capability.
			if mode != sysio.STREAM_MODE_WRITE {
				return 0, sysio.Stream_Empty
			}
			output := data.(*os.File)
			written, write_err := output.Write(buffer)
			return int64(written), write_err
		},
	}
}

// The root alternates recorded continuations and timeline advancement until setup stops.
func drive_runner(
	driver sysio.Driver, runner setup.Runner, diagnostics sysio.Stream,
) (status setup.Exit_Code) {
	for !runner.Stopped() {
		for runner.Rearm() {
		}
		if runner.Stopped() {
			break
		}
		completed, drive_err := driver.Run_Until(func() (finished bool) {
			if runner.Stopped() {
				return true
			}
			return bool(runner.Work_Queued())
		}, setup.PUMP_DURATION_MAX)
		if drive_err != nil {
			write_drive_failure(diagnostics)
			return setup.EXIT_FAILURE
		}
		if !completed {
			write_drive_failure(diagnostics)
			return setup.EXIT_FAILURE
		}
	}
	return runner.Status()
}

// Write_drive_failure keeps the driver result bounded before it crosses the output boundary.
func write_drive_failure(diagnostics sysio.Stream) {
	message := []byte(setup.IO_OPERATION_INCOMPLETE + "\n")
	written, write_err := sysio.Write(diagnostics, message)
	if write_err != nil {
		return
	}
	if written != int64(len(message)) {
		return
	}
}
