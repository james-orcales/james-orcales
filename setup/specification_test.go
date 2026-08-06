package main

import (
	"bytes"
	"io"
	"os"
	"testing"
)

// The root owns timeline advancement, so no internal package can drive an injected IO loop.
func Test_Main_Owns_Driver(t *testing.T) {
	source := main_source(t)
	if !bytes.Contains(source, []byte("driver.Run_Until(")) {
		t.Fatal("main.go does not own the IO Driver")
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
