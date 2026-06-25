package io_test

import (
	"bytes"
	"crypto/tls"
	"net"
	"os"
	"path/filepath"
	"syscall"
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

	clock, _ := timeos.New_Operating_System_Clock()
	loop, driver := iodefault.New_Operating_System_IO(clock)
	buffer := make([]byte, 5)
	count := -1
	var completion io.Completion
	loop.Read(&completion, func(_ *io.Completion, bytes int, read_err error) {
		if read_err != nil {
			t.Errorf("read error: %v", read_err)
		}
		count = bytes
	}, io.File(file.Fd()), buffer, 0)
	driver.Run()

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
	clock, _ := timeos.New_Operating_System_Clock()
	loop, driver := iodefault.New_Operating_System_IO(clock)
	fired := false
	var completion io.Completion
	loop.Timeout(&completion, func(_ *io.Completion, err error) {
		fired = true
	}, time.Millisecond)
	driver.Run_For(50 * time.Millisecond)
	if !fired {
		t.Fatal("timeout did not fire")
	}
}

// Test_Operating_System_IO_Socket runs a TCP loopback round-trip through the real
// backend: a client connects to a listener, sends bytes, and the accepted server
// socket receives them — all driven by the single event loop.
func Test_Operating_System_IO_Socket(t *testing.T) {
	port := free_port(t)
	clock, _ := timeos.New_Operating_System_Clock()
	loop, driver := iodefault.New_Operating_System_IO(clock)

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

	driver.Run_For(500 * time.Millisecond)
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

	driver.Run_For(500 * time.Millisecond)
	if received != 4 {
		t.Fatalf("received %d bytes, want 4", received)
	}
	if string(buffer[:4]) != "ping" {
		t.Fatalf("received %q, want ping", buffer[:4])
	}
}

// Test_Operating_System_IO_Cancel verifies cancelling a pending timeout fires its
// callback exactly once, with the Cancelled error, and promptly rather than at its
// far-off deadline.
func Test_Operating_System_IO_Cancel(t *testing.T) {
	clock, _ := timeos.New_Operating_System_Clock()
	loop, driver := iodefault.New_Operating_System_IO(clock)

	got := error(nil)
	fired := 0
	var completion io.Completion
	loop.Timeout(&completion, func(_ *io.Completion, err error) {
		fired++
		got = err
	}, time.Second)

	loop.Cancel(&completion)
	driver.Run_For(10 * time.Millisecond)

	if fired != 1 {
		t.Fatalf("callback fired %d times, want exactly 1", fired)
	}
	if got != io.Cancelled {
		t.Fatalf("cancel error = %v, want io.Cancelled", got)
	}
}

// Test_Operating_System_IO_Cancel_Accept verifies cancelling a socket operation armed
// on the poll drops the waiter and delivers the Cancelled error exactly once.
func Test_Operating_System_IO_Cancel_Accept(t *testing.T) {
	port := free_port(t)
	clock, _ := timeos.New_Operating_System_Clock()
	loop, driver := iodefault.New_Operating_System_IO(clock)

	listener, listen_err := loop.Listen("127.0.0.1", port)
	if listen_err != nil {
		t.Fatalf("listen: %v", listen_err)
	}

	got := error(nil)
	fired := 0
	var completion io.Completion
	loop.Accept(&completion, func(_ *io.Completion, socket io.File, err error) {
		fired++
		got = err
	}, listener)

	loop.Cancel(&completion)
	driver.Run_For(10 * time.Millisecond)

	if fired != 1 {
		t.Fatalf("accept callback fired %d times, want exactly 1", fired)
	}
	if got != io.Cancelled {
		t.Fatalf("cancelled accept error = %v, want io.Cancelled", got)
	}
}

// Test_Operating_System_IO_Open opens a file through the loop and reads it back.
func Test_Operating_System_IO_Open(t *testing.T) {
	source, err := os.CreateTemp(t.TempDir(), "io")
	if err != nil {
		t.Fatal(err)
	}
	if _, write_err := source.WriteAt([]byte("hello"), 0); write_err != nil {
		t.Fatal(write_err)
	}

	clock, _ := timeos.New_Operating_System_Clock()
	loop, driver := iodefault.New_Operating_System_IO(clock)
	file, open_err := loop.Open(source.Name())
	if open_err != nil {
		t.Fatalf("open: %v", open_err)
	}
	buffer := make([]byte, 5)
	count := -1
	var completion io.Completion
	loop.Read(&completion, func(_ *io.Completion, bytes int, read_err error) {
		count = bytes
	}, file, buffer, 0)
	driver.Run()

	if count != 5 {
		t.Fatalf("read %d bytes, want 5", count)
	}
	if string(buffer) != "hello" {
		t.Fatalf("read %q, want hello", buffer)
	}
}

// Test_Operating_System_IO_Create creates a file through the loop and writes to it.
func Test_Operating_System_IO_Create(t *testing.T) {
	source, err := os.CreateTemp(t.TempDir(), "io")
	if err != nil {
		t.Fatal(err)
	}

	clock, _ := timeos.New_Operating_System_Clock()
	loop, driver := iodefault.New_Operating_System_IO(clock)
	file, create_err := loop.Create(source.Name())
	if create_err != nil {
		t.Fatalf("create: %v", create_err)
	}
	count := -1
	var completion io.Completion
	loop.Write(&completion, func(_ *io.Completion, bytes int, write_err error) {
		count = bytes
	}, file, []byte("world"), 0)
	driver.Run()

	if count != 5 {
		t.Fatalf("wrote %d bytes, want 5", count)
	}
	verify, open_err := loop.Open(source.Name())
	if open_err != nil {
		t.Fatalf("open: %v", open_err)
	}
	buffer := make([]byte, 5)
	read := -1
	var read_completion io.Completion
	loop.Read(&read_completion, func(_ *io.Completion, bytes int, read_err error) {
		read = bytes
	}, verify, buffer, 0)
	driver.Run()
	if read != 5 {
		t.Fatalf("read back %d bytes, want 5", read)
	}
	if string(buffer) != "world" {
		t.Fatalf("file holds %q, want world", buffer)
	}
}

// Test_Operating_System_IO_Peer_Address reports the remote address of an accepted
// loopback connection.
func Test_Operating_System_IO_Peer_Address(t *testing.T) {
	port := free_port(t)
	clock, _ := timeos.New_Operating_System_Clock()
	loop, driver := iodefault.New_Operating_System_IO(clock)
	listener, listen_err := loop.Listen("127.0.0.1", port)
	if listen_err != nil {
		t.Fatalf("listen: %v", listen_err)
	}

	accepted := io.File(-1)
	var accept_completion io.Completion
	loop.Accept(&accept_completion, func(_ *io.Completion, socket io.File, err error) {
		accepted = socket
	}, listener)
	var connect_completion io.Completion
	loop.Connect(
		&connect_completion,
		func(_ *io.Completion, socket io.File, err error) {},
		"127.0.0.1", port,
	)
	driver.Run_For(500 * time.Millisecond)

	if accepted <= 0 {
		t.Fatalf("accept did not complete, got %d", accepted)
	}
	address, address_err := loop.Peer_Address(accepted)
	if address_err != nil {
		t.Fatalf("peer address: %v", address_err)
	}
	if address != "127.0.0.1" {
		t.Fatalf("peer address = %q, want 127.0.0.1", address)
	}
}

// Test_Operating_System_IO_Compute runs work on the pool and delivers on the loop.
func Test_Operating_System_IO_Compute(t *testing.T) {
	clock, _ := timeos.New_Operating_System_Clock()
	loop, driver := iodefault.New_Operating_System_IO(clock)
	ran := false
	fired := false
	var completion io.Completion
	loop.Compute(&completion, func(_ *io.Completion) {
		fired = true
	}, func() { ran = true })
	driver.Run_For(500 * time.Millisecond)

	if !ran {
		t.Fatal("compute work did not run")
	}
	if !fired {
		t.Fatal("compute callback did not fire on the loop")
	}
}

// Test_Operating_System_IO_Watch_Signal delivers a real SIGTERM onto the loop.
func Test_Operating_System_IO_Watch_Signal(t *testing.T) {
	clock, _ := timeos.New_Operating_System_Clock()
	loop, driver := iodefault.New_Operating_System_IO(clock)
	got := io.Signal(-1)
	fired := 0
	var completion io.Completion
	loop.Watch_Signal(&completion, func(_ *io.Completion, signal io.Signal) {
		fired++
		got = signal
	}, io.Signal_Terminate)
	if kill_err := syscall.Kill(os.Getpid(), syscall.SIGTERM); kill_err != nil {
		t.Fatalf("kill: %v", kill_err)
	}
	driver.Run_For(500 * time.Millisecond)

	if fired != 1 {
		t.Fatalf("signal callback fired %d times, want 1", fired)
	}
	if got != io.Signal_Terminate {
		t.Fatalf("signal = %d, want Signal_Terminate", got)
	}
}

// Test_Operating_System_IO_TLS runs a TLS loopback: a client connects (skipping
// verification) to a secure listener and exchanges plaintext through the tunnel.
func Test_Operating_System_IO_TLS(t *testing.T) {
	port := free_port(t)
	certificate := self_signed(t)
	clock, _ := timeos.New_Operating_System_Clock()
	loop, driver := iodefault.New_Operating_System_IO(clock)
	listener, listen_err := loop.Listen("127.0.0.1", port)
	if listen_err != nil {
		t.Fatalf("listen: %v", listen_err)
	}

	server := io.File(-1)
	var accept_completion io.Completion
	loop.Accept_Secure(&accept_completion, func(_ *io.Completion, socket io.File, err error) {
		if err != nil {
			t.Errorf("accept secure: %v", err)
		}
		server = socket
	}, listener, func() (value any) { return &certificate })

	client := io.File(-1)
	var connect_completion io.Completion
	loop.Connect_Insecure(&connect_completion,
		func(_ *io.Completion, socket io.File, err error) {
			if err != nil {
				t.Errorf("connect insecure: %v", err)
			}
			client = socket
		}, "127.0.0.1", port, "localhost")

	driver.Run_For(3 * time.Second)
	if server <= 0 {
		t.Fatalf("secure accept did not complete, got %d", server)
	}
	if client <= 0 {
		t.Fatalf("insecure connect did not complete, got %d", client)
	}

	var send_completion io.Completion
	loop.Send(&send_completion, func(_ *io.Completion, count int, err error) {
		if err != nil {
			t.Errorf("send: %v", err)
		}
	}, client, []byte("ping"))
	buffer := make([]byte, 16)
	received := -1
	var receive_completion io.Completion
	loop.Receive(&receive_completion, func(_ *io.Completion, count int, err error) {
		if err != nil {
			t.Errorf("receive: %v", err)
		}
		received = count
	}, server, buffer)
	driver.Run_For(3 * time.Second)

	if received != 4 {
		t.Fatalf("received %d bytes, want 4", received)
	}
	if string(buffer[:4]) != "ping" {
		t.Fatalf("received %q, want ping", buffer[:4])
	}
}

// The static self-signed certificate the TLS loopback test presents. Connect_Insecure
// skips verification, so its fixed identity and far-future expiry are all it needs, and
// baking it in keeps the test off stdlib time (the time/default gateway's alone).
const tls_test_certificate = `-----BEGIN CERTIFICATE-----
MIIBKzCB0qADAgECAgEBMAoGCCqGSM49BAMCMBQxEjAQBgNVBAMTCWxvY2FsaG9z
dDAgFw03MDAxMDEwMDAwMDBaGA8zMDAwMDEwMTAwMDAwMFowFDESMBAGA1UEAxMJ
bG9jYWxob3N0MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEZiPPVV+KvWqtXWUV
FwqGmclWo4flNdoEF7LXW7rcRyEfdETN8D7eZHsc1EszCqX/J7TUz5qt+EBqZnvD
EEjmKKMTMBEwDwYDVR0RBAgwBocEfwAAATAKBggqhkjOPQQDAgNIADBFAiEA1wL/
Db1CQuIeXErn5BukOvBoo9dQXKNzmQhQ6H2uFcgCICaZ/Oc5XKE3zbXmXn8joWKU
xB+G3LVmykY+fgvMWA80
-----END CERTIFICATE-----
-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIIzFOeQBJZexjKqtf7Adz+DzZpwi6njSth1YMrDm8uUYoAoGCCqGSM49
AwEHoUQDQgAEZiPPVV+KvWqtXWUVFwqGmclWo4flNdoEF7LXW7rcRyEfdETN8D7e
ZHsc1EszCqX/J7TUz5qt+EBqZnvDEEjmKA==
-----END EC PRIVATE KEY-----
`

// Parses the static test certificate for the TLS loopback test.
func self_signed(t *testing.T) (certificate tls.Certificate) {
	pair, err := tls.X509KeyPair([]byte(tls_test_certificate), []byte(tls_test_certificate))
	if err != nil {
		t.Fatal(err)
	}
	return pair
}

// Test_Operating_System_IO_Spawn runs real commands through the loop: a success with
// captured output, and a non-zero exit reported without a start error.
func Test_Operating_System_IO_Spawn(t *testing.T) {
	clock, _ := timeos.New_Operating_System_Clock()
	loop, driver := iodefault.New_Operating_System_IO(clock)

	echo := io.Process_Result{}
	echoed := false
	var echo_completion io.Completion
	loop.Spawn(&echo_completion, func(_ *io.Completion, result io.Process_Result, err error) {
		if err != nil {
			t.Errorf("echo spawn: %v", err)
		}
		echo = result
		echoed = true
	}, io.Process_Request{Path: "/bin/echo", Arguments: []string{"hi"}})
	driver.Run_For(2 * time.Second)

	if !echoed {
		t.Fatal("echo did not complete")
	}
	if echo.Exit != 0 {
		t.Fatalf("echo exit = %d, want 0", echo.Exit)
	}
	if string(echo.Output) != "hi\n" {
		t.Fatalf("echo output = %q, want hi", echo.Output)
	}

	fail := io.Process_Result{}
	failed := false
	var fail_completion io.Completion
	loop.Spawn(&fail_completion, func(_ *io.Completion, result io.Process_Result, err error) {
		if err != nil {
			t.Errorf("false spawn: %v", err)
		}
		fail = result
		failed = true
	}, io.Process_Request{Path: "/bin/sh", Arguments: []string{"-c", "exit 1"}})
	driver.Run_For(2 * time.Second)

	if !failed {
		t.Fatal("false did not complete")
	}
	if fail.Exit != 1 {
		t.Fatalf("false exit = %d, want 1", fail.Exit)
	}
}

// Test_Operating_System_IO_Spawn_Streams_To_Sink runs a command with a live stdout sink,
// confirming the backend streams the child's output to the writer as it runs instead of
// capturing it — the affordance a long build needs — and leaves Output empty.
func Test_Operating_System_IO_Spawn_Streams_To_Sink(t *testing.T) {
	clock, _ := timeos.New_Operating_System_Clock()
	loop, driver := iodefault.New_Operating_System_IO(clock)

	streamed := bytes.Buffer{}
	result := io.Process_Result{}
	done := false
	var completion io.Completion
	loop.Spawn(&completion, func(_ *io.Completion, spawned io.Process_Result, err error) {
		if err != nil {
			t.Errorf("echo spawn: %v", err)
		}
		result = spawned
		done = true
	}, io.Process_Request{Path: "/bin/echo", Arguments: []string{"hi"}, Stdout: &streamed})
	driver.Run_For(2 * time.Second)

	if !done {
		t.Fatal("echo did not complete")
	}
	if streamed.String() != "hi\n" {
		t.Fatalf("streamed output = %q, want hi", streamed.String())
	}
	if len(result.Output) != 0 {
		t.Fatalf("Output = %q, want empty when streamed to a sink", result.Output)
	}
}

// Test_Operating_System_IO_Directory exercises the filesystem-traversal ops on a real temp
// tree: Make_Directory builds a nested path (into which the fixture file is seeded), and
// Status and Read_Directory then report the tree's shape, including an absent path.
func Test_Operating_System_IO_Directory(t *testing.T) {
	clock, _ := timeos.New_Operating_System_Clock()
	loop, _ := iodefault.New_Operating_System_IO(clock)

	root := t.TempDir()
	nested := filepath.Join(root, "a", "b")
	if make_err := loop.Make_Directory(nested); make_err != nil {
		t.Fatalf("make directory: %v", make_err)
	}
	file_path := filepath.Join(nested, "file.txt")
	if seed_err := os.WriteFile(file_path, []byte("hello"), 0o644); seed_err != nil {
		t.Fatalf("seed file (proves Make_Directory built the parents): %v", seed_err)
	}

	directory_status, _ := loop.Status(nested)
	if !directory_status.Exists {
		t.Fatalf("nested status = %+v, want an existing path", directory_status)
	}
	if !directory_status.Is_Directory {
		t.Fatalf("nested status = %+v, want a directory", directory_status)
	}
	regular_status, _ := loop.Status(file_path)
	if !regular_status.Exists {
		t.Fatalf("file status = %+v, want an existing path", regular_status)
	}
	if regular_status.Is_Directory {
		t.Fatalf("file status = %+v, want a non-directory", regular_status)
	}
	absent_status, _ := loop.Status(filepath.Join(root, "nope"))
	if absent_status.Exists {
		t.Fatalf("absent status = %+v, want not exists", absent_status)
	}

	entries, read_err := loop.Read_Directory(nested)
	if read_err != nil {
		t.Fatalf("read directory: %v", read_err)
	}
	found := false
	for _, entry := range entries {
		if entry.Name != "file.txt" {
			continue
		}
		found = true
		if entry.Is_Directory {
			t.Fatal("file.txt reported as a directory")
		}
	}
	if !found {
		t.Fatalf("read directory %q missing file.txt, got %v", nested, entries)
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
