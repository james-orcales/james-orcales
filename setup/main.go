// Package main binds the operating-system capabilities for the setup command.
package main

import (
	"errors"
	"fmt"
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
	loop, driver, loop_err := system_io.New_Operating_System_IO(
		clock, setup.SCHEDULER_ENTRY_COUNT, setup.SCHEDULER_FLAGS)
	if loop_err != nil {
		fmt.Fprintln(os.Stderr, "cannot initialize IO:", loop_err)
		os.Exit(int(setup.EXIT_USAGE))
	}
	file_system := setup.File_System{Read_Directory: loop.Read_Directory, Status: loop.Status}
	read := setup.File_Read_Adapter(loop)
	file_system.Read = func(path string, size int) (contents []byte, found bool, err error) {
		ready, done, rearm, result := read(path, size)
		if drive_err := drive(driver, ready, done, rearm); drive_err != nil {
			return nil, false, drive_err
		}
		return result()
	}
	write := setup.File_Write_Adapter(loop)
	file_system.Write = func(path string, contents []byte) (err error) {
		ready, done, rearm, result := write(path, contents)
		if drive_err := drive(driver, ready, done, rearm); drive_err != nil {
			return drive_err
		}
		return result()
	}
	process := setup.Process_Adapter(loop)
	shell := setup.Shell{Spawn: func(
		request sysio.Process_Request,
	) (result sysio.Process_Result) {
		ready, done, rearm, process_result := process(request)
		if drive_err := drive(driver, ready, done, rearm); drive_err != nil {
			return sysio.Process_Result{Exit: 1}
		}
		return process_result()
	}}
	input := setup.Main_Input{
		Environment: &setup.Environment_Input{
			Clock: clock, Console: os.Stderr,
			Effective_User_Identifier: setup.Effective_User_Identifier(os.Geteuid()),
			Color:                     setup.Console_Color(color),
			Home_Directory:            setup.Home_Directory(home), Home_Error: home_err,
			Operating_System: setup.Operating_System(runtime.GOOS),
			Cargo_Directory:  setup.Cargo_Directory(os.Getenv("CARGO_HOME")),
			Data_Directory:   setup.Data_Directory(os.Getenv("XDG_DATA_HOME")),
			Stdout:           os.Stdout, Stderr: os.Stderr,
		}, File_System: file_system, Shell: shell,
	}
	status := setup.Main(&input)
	driver.Deinit()
	os.Exit(int(status))
}

// Only the composition root can advance callback-owned transitions.
func drive(
	driver sysio.Driver, ready setup.Transition_Ready,
	done setup.Transition_Complete, rearm setup.Transition_Rearm,
) (err error) {
	for !done() {
		completed, drive_err := driver.Run_Until(ready, setup.PROCESS_DURATION_MAX)
		if drive_err != nil {
			return drive_err
		}
		if !completed {
			return errors.New("the IO operation did not complete")
		}
		if !done() {
			rearm()
		}
	}
	return nil
}
