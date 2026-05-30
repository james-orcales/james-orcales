package io_test

import (
	"os"
	"testing"

	"github.com/james-orcales/james-orcales/shared/io"
	iodefault "github.com/james-orcales/james-orcales/shared/io/default"
	"github.com/james-orcales/james-orcales/shared/time"
	timeos "github.com/james-orcales/james-orcales/shared/time/default"
)

// Test_Operating_System_IO_Read writes a temp file and reads it back through the
// real backend, confirming the read runs in the loop and reports the bytes.
func Test_Operating_System_IO_Read(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "io")
	if err != nil {
		t.Fatal(err)
	}
	_, write_err := file.WriteAt([]byte("hello"), 0)
	if write_err != nil {
		t.Fatal(write_err)
	}

	loop := iodefault.New_Operating_System_IO(timeos.New_Operating_System_Clock())
	buffer := make([]byte, 5)
	count := -1
	var completion io.Completion
	loop.Read(&completion, func(_ *io.Completion, bytes int, read_err error) {
		if read_err != nil {
			t.Errorf("read error: %v", read_err)
		}
		count = bytes
	}, io.File(file.Fd()), buffer, 0)
	loop.Run()

	if count != 5 {
		t.Fatalf("read %d bytes, want 5", count)
	}
	if string(buffer) != "hello" {
		t.Fatalf("read %q, want hello", buffer)
	}
}

// Test_Operating_System_IO_Timeout verifies a timeout fires once real time passes
// its deadline.
func Test_Operating_System_IO_Timeout(t *testing.T) {
	loop := iodefault.New_Operating_System_IO(timeos.New_Operating_System_Clock())
	fired := false
	var completion io.Completion
	loop.Timeout(&completion, func(_ *io.Completion, err error) {
		fired = true
	}, time.Millisecond)
	loop.Run_For(50 * time.Millisecond)
	if !fired {
		t.Fatal("timeout did not fire")
	}
}
