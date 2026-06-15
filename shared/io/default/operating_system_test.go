package io_test

import (
	"net"
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

// Test_Operating_System_IO_Socket runs a TCP loopback round-trip through the real
// backend: a client connects to a listener, sends bytes, and the accepted server
// socket receives them — all driven by the single event loop.
func Test_Operating_System_IO_Socket(t *testing.T) {
	port := free_port(t)
	loop := iodefault.New_Operating_System_IO(timeos.New_Operating_System_Clock())

	listener, listen_err := loop.Listen("127.0.0.1", port)
	if listen_err != nil {
		t.Fatalf("listen: %v", listen_err)
	}

	accepted := io.File(-1)
	var accept_completion io.Completion
	loop.Accept(&accept_completion, func(_ *io.Completion, socket io.File, accept_err error) {
		if accept_err != nil {
			t.Errorf("accept: %v", accept_err)
		}
		accepted = socket
	}, listener)

	connected := io.File(-1)
	var connect_completion io.Completion
	loop.Connect(
		&connect_completion,
		func(_ *io.Completion, socket io.File, connect_err error) {
			if connect_err != nil {
				t.Errorf("connect: %v", connect_err)
			}
			connected = socket
		},
		"127.0.0.1", port,
	)

	loop.Run_For(500 * time.Millisecond)
	if accepted <= 0 {
		t.Fatalf("accept did not complete, got %d", accepted)
	}
	if connected <= 0 {
		t.Fatalf("connect did not complete, got %d", connected)
	}

	var send_completion io.Completion
	loop.Send(&send_completion, func(_ *io.Completion, count int, send_err error) {
		if send_err != nil {
			t.Errorf("send: %v", send_err)
		}
	}, connected, []byte("ping"))

	buffer := make([]byte, 16)
	received := -1
	var receive_completion io.Completion
	loop.Receive(&receive_completion, func(_ *io.Completion, count int, receive_err error) {
		if receive_err != nil {
			t.Errorf("receive: %v", receive_err)
		}
		received = count
	}, accepted, buffer)

	loop.Run_For(500 * time.Millisecond)
	if received != 4 {
		t.Fatalf("received %d bytes, want 4", received)
	}
	if string(buffer[:4]) != "ping" {
		t.Fatalf("received %q, want ping", buffer[:4])
	}
}

// Returns a probably-free TCP port by binding and releasing one through the standard
// library, used only to pick a target for the backend under test.
func free_port(t *testing.T) (port int) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port = listener.Addr().(*net.TCPAddr).Port
	if close_err := listener.Close(); close_err != nil {
		t.Fatal(close_err)
	}
	return port
}
