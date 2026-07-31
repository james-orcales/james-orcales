// Package main binds the operating-system capabilities for the setup command.
package main

import (
	"errors"
	"os"
	"runtime"

	"local/james-orcales/setup/internal"
	sysio "local/james-orcales/shared/io"
	system_io "local/james-orcales/shared/io/default"
	jlog "local/james-orcales/shared/jlog/default"
	system_time "local/james-orcales/shared/time/default"
)

func main() {
	clock, _ := system_time.New_Operating_System_Clock()
	home, home_err := os.UserHomeDir()
	color := false
	if console, console_err := os.Stderr.Stat(); console_err == nil {
		color = console.Mode()&os.ModeCharDevice != 0
	}
	environment, status := setup.New_Environment(&setup.Environment_Input{
		Clock: clock, Console: os.Stderr, Effective_User_Identifier: os.Geteuid(),
		Color: color, Home_Directory: home, Home_Error: home_err,
		Operating_System: runtime.GOOS, Cargo_Directory: os.Getenv("CARGO_HOME"),
		Data_Directory: os.Getenv("XDG_DATA_HOME"), Stdout: os.Stdout, Stderr: os.Stderr,
	})
	if status != 0 {
		os.Exit(status)
	}
	loop, driver, loop_err := system_io.New_Operating_System_IO(
		environment.Clock, setup.SCHEDULER_ENTRY_COUNT, setup.SCHEDULER_FLAGS)
	if loop_err != nil {
		jlog.Logger_Error(environment.Logger, "cannot initialize IO", jlog.Err(loop_err))
		os.Exit(setup.EXIT_USAGE)
	}
	// Internal code owns each transition. This root only advances one callback-owned operation.
	drive := func(operation setup.Operation) (err error) {
		for !operation.Complete() {
			completed, drive_err := driver.Run_Until(operation.Ready, sysio.FOREVER)
			if drive_err != nil {
				return drive_err
			}
			if !completed {
				return errors.New("the IO operation did not complete")
			}
			if !operation.Complete() {
				if operation.Rearm != nil {
					operation.Rearm()
				}
			}
		}
		return nil
	}
	file_system := setup.File_System{Read_Directory: loop.Read_Directory, Status: loop.Status}
	file_system.Read = func(path string, buffer_size int) (
		contents []byte, found bool, err error) {
		operation := setup.Read_File(loop, path, buffer_size)
		if drive_err := drive(operation.Operation); drive_err != nil {
			return nil, false, drive_err
		}
		return setup.File_Read_Result(operation)
	}
	file_system.Write = func(path string, contents []byte) (err error) {
		operation := setup.Write_File(loop, path, contents)
		if drive_err := drive(operation.Operation); drive_err != nil {
			return drive_err
		}
		return setup.File_Write_Error(operation)
	}
	shell := setup.Shell{Stdout: environment.Stdout, Stderr: environment.Stderr}
	shell.Logger = environment.Logger
	shell.Spawn = func(request sysio.Process_Request) (result sysio.Process_Result) {
		operation := setup.Spawn_Process(loop, request)
		if drive_err := drive(operation.Operation); drive_err != nil {
			return sysio.Process_Result{Exit: 1}
		}
		return setup.Process_Operation_Result(operation)
	}
	input := setup.Main_Input{Environment: environment, File_System: file_system, Shell: shell}
	status = setup.Main(&input)
	driver.Deinit()
	os.Exit(status)
}
