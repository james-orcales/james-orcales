package main

import (
	"errors"
	"testing"

	sysio "local/james-orcales/shared/io"
	systime "local/james-orcales/shared/time"
)

// The composition root must keep the Driver while it gives setup only synchronous file
// operations. This test prevents a future adapter from returning the Driver to the library.
func Test_File_System_Reads_And_Writes(t *testing.T) {
	loop, driver, _ := sysio.New_Sim(1)
	system := file_system(loop, driver)
	want := []byte("set -o nounset\n")
	write_err := system.Write("/source/profile", want)
	if write_err != nil {
		t.Fatalf("write file: %v", write_err)
	}
	got, found, read_err := system.Read("/source/profile")
	if read_err != nil {
		t.Fatalf("read file: %v", read_err)
	}
	if !found {
		t.Fatal("the written file is absent")
	}
	if string(got) != string(want) {
		t.Fatalf("file contents = %q, want %q", got, want)
	}
}

// A Driver error means that the root cannot prove completion. The adapter must report the
// error and must not present a partial write as a successful synchronous operation.
func Test_File_System_Reports_Driver_Error(t *testing.T) {
	drive_failure := errors.New("drive failure")
	loop := sysio.IO{
		Make_Directory: func(path string) (err error) { return nil },
		Create:         func(path string) (file sysio.File, err error) { return 1, nil },
		Write: func(
			completion *sysio.Completion, callback sysio.Callback,
			file sysio.File, buffer []byte, offset int64,
		) {
		},
	}
	driver := sysio.Driver{
		Run_Until: func(
			done func() (finished bool), timeout systime.Duration,
		) (completed bool, err error) {
			return false, drive_failure
		},
	}
	system := file_system(loop, driver)
	write_err := system.Write("/source/profile", []byte("content"))
	if !errors.Is(write_err, drive_failure) {
		t.Fatalf("write error = %v, want %v", write_err, drive_failure)
	}
}

// Setup builds can run for a long time, but a finite limit lets shared/io stop a process group
// that cannot finish. This test keeps that limit on each command that the root submits.
func Test_Spawn_Command_Uses_Finite_Duration(t *testing.T) {
	duration := systime.Duration(0)
	loop := sysio.IO{
		Spawn: func(
			completion *sysio.Completion, callback sysio.Process_Callback,
			request sysio.Process_Request, deadline systime.Duration,
		) {
			duration = deadline
			callback(completion, sysio.Process_Result{}, nil)
		},
	}
	driver := sysio.Driver{
		Run_Until: func(
			done func() (finished bool), timeout systime.Duration,
		) (completed bool, err error) {
			return done(), nil
		},
	}
	result := spawn_command(loop, driver)(sysio.Process_Request{Path: "build"})
	if result.Exit != 0 {
		t.Fatalf("spawn exit = %d, want 0", result.Exit)
	}
	if duration != PROCESS_DURATION_MAX {
		t.Fatalf("spawn duration = %d, want %d", duration, PROCESS_DURATION_MAX)
	}
	if duration <= 0 {
		t.Fatalf("spawn duration = %d, want a positive duration", duration)
	}
}

// The synchronous command boundary cannot return success when its Driver fails before the
// callback. The process result uses the existing nonzero exit contract for this root error.
func Test_Spawn_Command_Reports_Driver_Error(t *testing.T) {
	drive_failure := errors.New("drive failure")
	loop := sysio.IO{
		Spawn: func(
			completion *sysio.Completion, callback sysio.Process_Callback,
			request sysio.Process_Request, deadline systime.Duration,
		) {
		},
	}
	driver := sysio.Driver{
		Run_Until: func(
			done func() (finished bool), timeout systime.Duration,
		) (completed bool, err error) {
			return false, drive_failure
		},
	}
	result := spawn_command(loop, driver)(sysio.Process_Request{Path: "build"})
	if result.Exit == 0 {
		t.Fatal("a Driver error returned a successful process result")
	}
}
