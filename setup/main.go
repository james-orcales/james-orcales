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
	systime "local/james-orcales/shared/time"
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
	system := draining_io(loop, driver)
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
		}, IO: system,
	}
	status := setup.Main(&input)
	driver.Deinit()
	os.Exit(int(status))
}

// The root drains each submission before it returns because setup policy is sequential. The
// copied value still has the shared IO shape, and no internal package receives the Driver.
func draining_io(loop sysio.IO, driver sysio.Driver) (system sysio.IO) {
	system = loop
	system.Read = func(
		completion *sysio.Completion, callback sysio.Callback,
		file sysio.File, buffer []byte, offset int64,
	) {
		retired, count, operation_err := false, 0, error(nil)
		actual := sysio.Completion{}
		loop.Read(&actual, func(_ *sysio.Completion, completed int, err error) {
			count, operation_err, retired = completed, err, true
		}, file, buffer, offset)
		drive_err := drive(driver, func() (done bool) {
			return retired
		})
		callback(completion, count, errors.Join(operation_err, drive_err))
	}
	system.Write = func(
		completion *sysio.Completion, callback sysio.Callback,
		file sysio.File, buffer []byte, offset int64,
	) {
		retired, count, operation_err := false, 0, error(nil)
		actual := sysio.Completion{}
		loop.Write(&actual, func(_ *sysio.Completion, completed int, err error) {
			count, operation_err, retired = completed, err, true
		}, file, buffer, offset)
		drive_err := drive(driver, func() (done bool) {
			return retired
		})
		callback(completion, count, errors.Join(operation_err, drive_err))
	}
	system.Close = func(
		completion *sysio.Completion, callback sysio.Timeout_Callback, file sysio.File,
	) {
		retired, operation_err := false, error(nil)
		actual := sysio.Completion{}
		loop.Close(&actual, func(_ *sysio.Completion, err error) {
			operation_err, retired = err, true
		}, file)
		drive_err := drive(driver, func() (done bool) {
			return retired
		})
		callback(completion, errors.Join(operation_err, drive_err))
	}
	system.Spawn = func(
		completion *sysio.Completion, callback sysio.Process_Callback,
		request sysio.Process_Request, deadline systime.Duration,
	) {
		retired, result, operation_err := false, sysio.Process_Result{}, error(nil)
		actual := sysio.Completion{}
		loop.Spawn(&actual, func(
			_ *sysio.Completion, completed sysio.Process_Result, err error,
		) {
			result, operation_err, retired = completed, err, true
		}, request, deadline)
		drive_err := drive(driver, func() (done bool) {
			return retired
		})
		callback(completion, result, errors.Join(operation_err, drive_err))
	}
	return system
}

// Only the composition root can advance callback-owned transitions.
func drive(driver sysio.Driver, done func() (finished bool)) (err error) {
	completed, drive_err := driver.Run_Until(done, setup.PROCESS_DURATION_MAX)
	if drive_err != nil {
		return drive_err
	}
	if !completed {
		return errors.New("the IO operation did not complete")
	}
	return nil
}
