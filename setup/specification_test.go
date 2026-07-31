package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"testing"

	setup "local/james-orcales/setup/internal"
	sysio "local/james-orcales/shared/io"
	systime "local/james-orcales/shared/time"
)

// The root exists only to construct capabilities and advance the loop. This limit prevents setup
// policy and operating-system adapters from moving back out of internal packages.
func Test_Main_Stays_Thin(t *testing.T) {
	source := main_source(t)
	line_count := bytes.Count(source, []byte{'\n'})
	if line_count >= 90 {
		t.Fatalf("main.go has %d lines, want fewer than 90", line_count)
	}
}

// One internal entry point keeps the ordered bootstrap policy out of the composition root.
func Test_Main_Calls_Internal_Main(t *testing.T) {
	source := main_source(t)
	if !bytes.Contains(source, []byte("setup.Main(")) {
		t.Fatal("main.go does not call internal.Main")
	}
	if bytes.Contains(source, []byte("setup.Bootstrap(")) {
		t.Fatal("main.go owns bootstrap orchestration")
	}
}

// The composition root must keep the Driver while it gives setup only synchronous file
// operations. This test prevents a future adapter from returning the Driver to the library.
func Test_File_System_Reads_And_Writes(t *testing.T) {
	loop, driver, _ := sysio.New_Sim(1)
	want := []byte("set -o nounset\n")
	write := setup.Write_File(loop, "/source/profile", want)
	write_err := drive_operation(driver, write.Operation)
	if write_err != nil {
		t.Fatalf("write file: %v", write_err)
	}
	read := setup.Read_File(loop, "/source/profile", setup.DOTFILE_BYTES_MAX)
	read_err := drive_operation(driver, read.Operation)
	if read_err != nil {
		t.Fatalf("read file: %v", read_err)
	}
	got, found, result_err := setup.File_Read_Result(read)
	if result_err != nil {
		t.Fatalf("read result: %v", result_err)
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
	write := setup.Write_File(loop, "/source/profile", []byte("content"))
	write_err := drive_operation(driver, write.Operation)
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
	operation := setup.Spawn_Process(loop, sysio.Process_Request{Path: "build"})
	drive_err := drive_operation(driver, operation.Operation)
	if drive_err != nil {
		t.Fatalf("drive process: %v", drive_err)
	}
	result := setup.Process_Operation_Result(operation)
	if result.Exit != 0 {
		t.Fatalf("spawn exit = %d, want 0", result.Exit)
	}
	if duration != setup.PROCESS_DURATION_MAX {
		t.Fatalf("spawn duration = %d, want %d", duration, setup.PROCESS_DURATION_MAX)
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
	operation := setup.Spawn_Process(loop, sysio.Process_Request{Path: "build"})
	result := sysio.Process_Result{}
	if drive_operation(driver, operation.Operation) != nil {
		result.Exit = 1
	} else {
		result = setup.Process_Operation_Result(operation)
	}
	if result.Exit == 0 {
		t.Fatal("a Driver error returned a successful process result")
	}
}

const MAIN_SOURCE_BYTES_MAX = 16384

// A fixed buffer makes the source inspection independent of the file size and keeps the
// specification test within the same bounded-read policy as production code.
func main_source(t *testing.T) (source []byte) {
	file, open_err := os.Open("main.go")
	if open_err != nil {
		t.Fatalf("open main.go: %v", open_err)
	}
	t.Cleanup(func() {
		if close_err := file.Close(); close_err != nil {
			t.Errorf("close main.go: %v", close_err)
		}
	})
	buffer := make([]byte, MAIN_SOURCE_BYTES_MAX)
	count, read_err := io.ReadFull(file, buffer)
	if read_err != nil {
		if read_err != io.ErrUnexpectedEOF {
			t.Fatalf("read main.go: %v", read_err)
		}
	}
	return buffer[:count]
}

// The test owns the simulated Driver. The helper matches the root loop without giving that
// capability to setup, so each operation test proves the same ownership boundary.
func drive_operation(driver sysio.Driver, operation setup.Operation) (err error) {
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
