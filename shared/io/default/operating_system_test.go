package io_test

import (
	"bytes"
	"errors"
	stdio "io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"local/james-orcales/shared/io"
	system_io "local/james-orcales/shared/io/default"
	"local/james-orcales/shared/time"
	timeos "local/james-orcales/shared/time/default"
)

// The Run_Until cap for the real-backend tests: generous, since a completion returns the
// pump the instant it fires — this bound only bites a genuine hang, failing the test
// instead of blocking until the package timeout.
const REAL_DEADLINE = 5 * time.SECOND

// The short finite operation deadline used to prove a dormant kernel wait retires promptly.
const REAL_OPERATION_DEADLINE = 25 * time.MILLISECOND

// Creates the 32-entry test scheduler and fails at the composition root if initialization fails.
func operating_system_loop(t *testing.T, clock time.Clock) (loop io.IO, driver io.Driver) {
	t.Helper()
	loop, driver, err := system_io.New_Operating_System_IO(clock, 32, 0)
	if err != nil {
		t.Fatalf("initialize io: %v", err)
	}
	return loop, driver
}

// Drives a real test predicate and fails immediately on a backend scheduler error.
func operating_system_run_until(
	t *testing.T, driver io.Driver, done func() (finished bool),
) (completed bool) {
	t.Helper()
	completed, err := driver.Run_Until(done, REAL_DEADLINE)
	if err != nil {
		t.Fatalf("drive io: %v", err)
	}
	return completed
}

// Returns the explicit profile used by real-backend socket tests.
func test_tcp_options() (options io.TCP_Options) {
	return io.TCP_Options{
		Receive_Buffer: 4 * 1024 * 1024,
		Send_Buffer:    2 * 1024 * 1024,
		Keepalive: &io.TCP_Keepalive{
			Idle_Seconds: 5, Interval_Seconds: 4, Count: 3,
		},
		User_Timeout_Milliseconds: 17 * 1000,
		No_Delay:                  true,
	}
}

// Opens the caller-owned IPv4 TCP socket used by backend tests.
func test_open_socket(loop io.IO) (socket io.File, err error) {
	return loop.Open_Socket_TCP(io.FAMILY_IPV4, test_tcp_options())
}

// Opens and binds one caller-owned IPv4 TCP listener.
func test_listen(loop io.IO, host string, port int) (listener io.File, err error) {
	address, address_err := io.Address_Parse(host, port)
	if address_err != nil {
		return io.File(-1), address_err
	}
	listener, open_err := test_open_socket(loop)
	if open_err != nil {
		return io.File(-1), open_err
	}
	_, listen_err := loop.Listen(listener, address, io.Listen_Options{Backlog: 65535})
	if listen_err != nil {
		loop.Close_Socket(listener)
		return io.File(-1), listen_err
	}
	return listener, nil
}

// Converts an IP literal for the explicit-address Connect surface.
func test_connect(
	loop io.IO, completion *io.Completion, callback io.Timeout_Callback,
	socket io.File, host string, port int,
) {
	address, err := io.Address_Parse(host, port)
	if err != nil {
		panic(err)
	}
	loop.Connect(completion, callback, socket, address, REAL_DEADLINE)
}

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
	loop, driver := operating_system_loop(t, clock)
	buffer := make([]byte, 5)
	count := -1
	read_done := false
	var completion io.Completion
	loop.Read(&completion, func(_ *io.Completion, bytes int, read_err error) {
		if read_err != nil {
			t.Errorf("read error: %v", read_err)
		}
		count = bytes
		read_done = true
	}, io.File(file.Fd()), buffer, 0)
	if !operating_system_run_until(t, driver, func() (finished bool) { return read_done }) {
		t.Fatal("read did not complete")
	}

	if count != 5 {
		t.Fatalf("read %d bytes, want 5", count)
	}
	if string(buffer) != "hello" {
		t.Fatalf("read %q, want hello", buffer)
	}
}

// Test_Resolve_Passes_IP_Literal confirms an IP-literal host returns unchanged, so an
// already-resolved address skips the blocking DNS lookup and the loop's dial path only sees IPs.
func Test_Resolve_Passes_IP_Literal(t *testing.T) {
	address, err := system_io.Resolve("93.184.216.34")
	if err != nil {
		t.Fatalf("resolve ip literal: %v", err)
	}
	if address != "93.184.216.34" {
		t.Fatalf("resolve returned %q, want 93.184.216.34", address)
	}
}

// Test_Operating_System_IO_Run_Until_Deadlock verifies an unbounded Run_Until with no operation
// pending fails loud rather than blocking forever: a predicate no event can flip is a deadlock,
// so the pump panics instead of hanging the caller.
func Test_Operating_System_IO_Run_Until_Deadlock(t *testing.T) {
	clock, _ := timeos.New_Operating_System_Clock()
	_, driver := operating_system_loop(t, clock)
	defer func() {
		if recover() == nil {
			t.Fatal("an unbounded Run_Until with nothing pending must panic")
		}
	}()
	driver.Run_Until(func() (finished bool) { return false }, io.FOREVER)
}

// Test_Operating_System_IO_Timeout verifies a timeout fires once real time passes
// its deadline.
func Test_Operating_System_IO_Timeout(t *testing.T) {
	clock, _ := timeos.New_Operating_System_Clock()
	loop, driver := operating_system_loop(t, clock)
	fired := false
	var completion io.Completion
	loop.Timeout(&completion, func(_ *io.Completion, err error) {
		fired = true
	}, time.MILLISECOND)
	driver.Run_Until(func() (finished bool) { return fired }, REAL_DEADLINE)
	if !fired {
		t.Fatal("timeout did not fire")
	}
}

// Test_Operating_System_IO_Open_Socket_Profile verifies outbound sockets are non-blocking,
// close-on-exec, buffered for the client workload, keepalive-enabled, and caller-owned in Raw_Open.
func Test_Operating_System_IO_Open_Socket_Profile(t *testing.T) {
	clock, _ := timeos.New_Operating_System_Clock()
	loop, driver := operating_system_loop(t, clock)
	baseline := driver.Introspect().Raw_Open
	socket, open_err := test_open_socket(loop)
	if open_err != nil {
		t.Fatalf("open socket: %v", open_err)
	}
	if raw_open := driver.Introspect().Raw_Open; raw_open != baseline+1 {
		t.Fatalf("raw open after Open_Socket = %d, want %d", raw_open, baseline+1)
	}
	file_flags := socket_fcntl(t, socket, syscall.F_GETFD)
	if file_flags&syscall.FD_CLOEXEC == 0 {
		t.Fatal("Open_Socket descriptor is not close-on-exec")
	}
	status_flags := socket_fcntl(t, socket, syscall.F_GETFL)
	if status_flags&syscall.O_NONBLOCK == 0 {
		t.Fatal("Open_Socket descriptor is not non-blocking")
	}
	receive_buffer, receive_err := syscall.GetsockoptInt(
		int(socket), syscall.SOL_SOCKET, syscall.SO_RCVBUF)
	if receive_err != nil {
		t.Fatalf("get receive buffer: %v", receive_err)
	}
	if receive_buffer <= 0 {
		t.Fatalf("receive buffer = %d, want a configured positive size", receive_buffer)
	}
	send_buffer, send_err := syscall.GetsockoptInt(
		int(socket), syscall.SOL_SOCKET, syscall.SO_SNDBUF)
	if send_err != nil {
		t.Fatalf("get send buffer: %v", send_err)
	}
	if send_buffer <= 0 {
		t.Fatalf("send buffer = %d, want a configured positive size", send_buffer)
	}
	keepalive, keepalive_err := syscall.GetsockoptInt(
		int(socket), syscall.SOL_SOCKET, syscall.SO_KEEPALIVE)
	if keepalive_err != nil {
		t.Fatalf("get keepalive: %v", keepalive_err)
	}
	if keepalive == 0 {
		t.Fatal("keepalive is disabled")
	}
	self_exec_close(loop, driver, socket)
	if raw_open := driver.Introspect().Raw_Open; raw_open != baseline {
		t.Fatalf("raw open after caller Close = %d, want %d", raw_open, baseline)
	}
}

// Test_Operating_System_IO_Reuse verifies resubmitting a completion that is still in
// flight panics: one Completion backs one operation at a time, and the real backend must
// fail as loudly as the sim instead of silently double-arming it.
func Test_Operating_System_IO_Reuse(t *testing.T) {
	clock, _ := timeos.New_Operating_System_Clock()
	loop, _ := operating_system_loop(t, clock)
	var completion io.Completion
	loop.Timeout(&completion, func(_ *io.Completion, err error) {}, time.SECOND)
	defer func() {
		if recover() == nil {
			t.Fatal("resubmitting an in-flight completion must panic")
		}
	}()
	loop.Timeout(&completion, func(_ *io.Completion, err error) {}, time.SECOND)
}

// Test_Operating_System_IO_Reentrancy verifies driving the real loop from within a
// completion callback panics, so a re-entrant Run* fails loudly rather than corrupting it.
func Test_Operating_System_IO_Reentrancy(t *testing.T) {
	clock, _ := timeos.New_Operating_System_Clock()
	loop, driver := operating_system_loop(t, clock)
	var completion io.Completion
	loop.Timeout(&completion, func(_ *io.Completion, err error) {
		driver.Run()
	}, time.MILLISECOND)
	defer func() {
		if recover() == nil {
			t.Fatal("driving from within a callback must panic")
		}
	}()
	driver.Run_For(50 * time.MILLISECOND)
}

// Test_Operating_System_IO_Socket runs a TCP loopback round-trip through the real
// backend: a client connects to a listener, sends bytes, and the accepted server
// socket receives them — all driven by the single event loop.
func Test_Operating_System_IO_Socket(t *testing.T) {
	port := free_port(t)
	clock, _ := timeos.New_Operating_System_Clock()
	loop, driver := operating_system_loop(t, clock)
	baseline := driver.Introspect().Raw_Open

	listener, listen_err := test_listen(loop, "127.0.0.1", port)
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
	}, listener, REAL_DEADLINE)

	connected, open_err := test_open_socket(loop)
	if open_err != nil {
		t.Fatalf("open socket: %v", open_err)
	}
	connect_done := false
	var connect_completion io.Completion
	test_connect(loop,
		&connect_completion,
		func(_ *io.Completion, connect_err error) {
			if connect_err != nil {
				t.Errorf("connect: %v", connect_err)
			}
			connect_done = true
		},
		connected, "127.0.0.1", port,
	)

	driver.Run_Until(func() (finished bool) { return accepted > 0 }, REAL_DEADLINE)
	driver.Run_Until(func() (finished bool) { return connect_done }, REAL_DEADLINE)
	if accepted <= 0 {
		t.Fatalf("accept did not complete, got %d", accepted)
	}
	if connected <= 0 {
		t.Fatalf("connect did not complete, got %d", connected)
	}

	loopback_assert_roundtrip(&loopback_roundtrip_input{
		Test: t, Loop: loop, Driver: driver, Connected: connected, Accepted: accepted,
	})
	self_exec_close(loop, driver, connected)
	self_exec_close(loop, driver, accepted)
	self_exec_close(loop, driver, listener)
	if raw_open := driver.Introspect().Raw_Open; raw_open != baseline {
		t.Fatalf("raw open after loopback teardown = %d, want %d", raw_open, baseline)
	}
}

// Test_Operating_System_IO_Accept_Deadline proves a listener with no inbound connection retires
// its accept exactly once, after which the listener and backend may be released safely.
func Test_Operating_System_IO_Accept_Deadline(t *testing.T) {
	clock, _ := timeos.New_Operating_System_Clock()
	loop, driver := operating_system_loop(t, clock)
	listener, listen_err := test_listen(loop, "127.0.0.1", 0)
	if listen_err != nil {
		t.Fatalf("listen: %v", listen_err)
	}
	callback_count := 0
	accepted := io.File(-1)
	var operation_err error
	var completion io.Completion
	loop.Accept(&completion, func(_ *io.Completion, socket io.File, err error) {
		callback_count++
		accepted = socket
		operation_err = err
	}, listener, REAL_OPERATION_DEADLINE)
	if !operating_system_run_until(
		t, driver, func() (finished bool) { return callback_count > 0 },
	) {
		t.Fatal("accept deadline did not resolve")
	}
	if callback_count != 1 {
		t.Fatalf("accept callback count = %d, want 1", callback_count)
	}
	if operation_err != io.Deadline_Exceeded {
		t.Fatalf("accept error = %v, want %v", operation_err, io.Deadline_Exceeded)
	}
	if accepted != -1 {
		t.Fatalf("deadline yielded accepted descriptor %d", accepted)
	}
	loop.Close_Socket(listener)
	driver.Deinit()
}

// Test_Operating_System_IO_Connect_Error_Preserves_Socket verifies refusal leaves the caller-owned
// descriptor open until the caller explicitly closes it.
func Test_Operating_System_IO_Connect_Error_Preserves_Socket(t *testing.T) {
	port := free_port(t)
	clock, _ := timeos.New_Operating_System_Clock()
	loop, driver := operating_system_loop(t, clock)
	raw_open_before := driver.Introspect().Raw_Open

	socket, open_err := test_open_socket(loop)
	if open_err != nil {
		t.Fatalf("open socket: %v", open_err)
	}
	called := false
	var connect_err error
	var completion io.Completion
	test_connect(loop, &completion, func(_ *io.Completion, err error) {
		called = true
		connect_err = err
	}, socket, "127.0.0.1", port)

	driver.Run_Until(func() (finished bool) { return called }, REAL_DEADLINE)
	if connect_err == nil {
		t.Fatal("connect to an unbound port succeeded, want an error")
	}
	if connect_err != io.Connection_Refused {
		t.Fatalf("connect error %v, want %v", connect_err, io.Connection_Refused)
	}
	if raw_open := driver.Introspect().Raw_Open; raw_open != raw_open_before+1 {
		t.Fatalf(
			"failed connect left %d raw descriptors open, want caller-owned %d",
			raw_open,
			raw_open_before+1,
		)
	}
	self_exec_close(loop, driver, socket)
	if raw_open := driver.Introspect().Raw_Open; raw_open != raw_open_before {
		t.Fatalf("raw open after caller Close = %d, want %d", raw_open, raw_open_before)
	}
}

// Test_Operating_System_IO_Send_In_Connect_Completion arms a send from inside the connect
// completion — the send shares the connected descriptor's write-waiter slot with the connect
// it is armed within. The loop must retire the connect waiter before delivering its callback,
// or the send is deleted the instant it is armed and never fires (the ClickHouse-daemon bug).
func Test_Operating_System_IO_Send_In_Connect_Completion(t *testing.T) {
	port := free_port(t)
	clock, _ := timeos.New_Operating_System_Clock()
	loop, driver := operating_system_loop(t, clock)

	listener, listen_err := test_listen(loop, "127.0.0.1", port)
	if listen_err != nil {
		t.Fatalf("listen: %v", listen_err)
	}
	accepted := io.File(-1)
	var accept_completion io.Completion
	loop.Accept(&accept_completion, func(_ *io.Completion, socket io.File, err error) {
		if err != nil {
			t.Errorf("accept: %v", err)
		}
		accepted = socket
	}, listener, REAL_DEADLINE)

	sent := -1
	var send_completion io.Completion
	var connect_completion io.Completion
	socket, open_err := test_open_socket(loop)
	if open_err != nil {
		t.Fatalf("open socket: %v", open_err)
	}
	test_connect(loop, &connect_completion, func(_ *io.Completion, err error) {
		if err != nil {
			t.Errorf("connect: %v", err)
			return
		}
		// Arm the send inside the connect completion: same descriptor, same write slot.
		loop.Send(&send_completion, func(_ *io.Completion, count int, send_err error) {
			if send_err != nil {
				t.Errorf("send: %v", send_err)
			}
			sent = count
		}, socket, []byte("ping"))
	}, socket, "127.0.0.1", port)

	driver.Run_Until(func() (finished bool) { return sent >= 0 }, REAL_DEADLINE)
	if sent != 4 {
		t.Fatalf("send armed in the connect completion delivered %d bytes, want 4", sent)
	}

	driver.Run_Until(func() (finished bool) { return accepted > 0 }, REAL_DEADLINE)
	buffer := make([]byte, 16)
	received := -1
	var receive_completion io.Completion
	loop.Receive(&receive_completion, func(_ *io.Completion, count int, receive_err error) {
		if receive_err != nil {
			t.Errorf("receive: %v", receive_err)
		}
		received = count
	}, accepted, buffer)
	driver.Run_Until(func() (finished bool) { return received >= 0 }, REAL_DEADLINE)
	if received != 4 {
		t.Fatalf("peer received %d bytes, want 4", received)
	}
	if string(buffer[:4]) != "ping" {
		t.Fatalf("peer received %q, want ping", buffer[:4])
	}
}

// Test_Operating_System_IO_Drain_Then_Recycle verifies Shutdown resolves an armed receive before
// Close, after which a later connection still receives readiness normally.
func Test_Operating_System_IO_Drain_Then_Recycle(t *testing.T) {
	port := free_port(t)
	clock, _ := timeos.New_Operating_System_Clock()
	loop, driver := operating_system_loop(t, clock)
	listener, listen_err := test_listen(loop, "127.0.0.1", port)
	if listen_err != nil {
		t.Fatalf("listen: %v", listen_err)
	}

	first, first_connected := loopback_pair(t, loop, driver, listener, port)
	buffer := make([]byte, 16)
	var receive_completion io.Completion
	fired := 0
	loop.Receive(&receive_completion,
		func(_ *io.Completion, _ int, _ error) { fired++ }, first, buffer)
	if shutdown_err := loop.Shutdown(first, io.SHUTDOWN_RECEIVE); shutdown_err != nil {
		t.Fatalf("shutdown receive: %v", shutdown_err)
	}
	if !operating_system_run_until(t, driver, func() (finished bool) { return fired > 0 }) {
		t.Fatal("shutdown did not drain the armed receive")
	}
	closed := false
	var close_completion io.Completion
	loop.Close(&close_completion, func(_ *io.Completion, _ error) { closed = true }, first)
	if !operating_system_run_until(t, driver, func() (finished bool) { return closed }) {
		t.Fatal("joined close did not complete")
	}
	self_exec_close(loop, driver, first_connected)

	// A later socket may reuse the descriptor and must still deliver readiness.
	recycled, second := loopback_pair(t, loop, driver, listener, port)
	var send_completion io.Completion
	loop.Send(&send_completion, func(_ *io.Completion, _ int, send_err error) {
		if send_err != nil {
			t.Errorf("send: %v", send_err)
		}
	}, second, []byte("pong"))
	received := -1
	var second_receive io.Completion
	loop.Receive(&second_receive, func(_ *io.Completion, count int, err error) {
		if err != nil {
			t.Errorf("recycled receive: %v", err)
		}
		received = count
	}, recycled, buffer)
	if !operating_system_run_until(t, driver, func() (finished bool) { return received >= 0 }) {
		t.Fatal("timed out waiting for the recycled descriptor to deliver its bytes")
	}
	if received != 4 {
		t.Fatalf("recycled descriptor received %d bytes, want 4", received)
	}
	if string(buffer[:4]) != "pong" {
		t.Fatalf("recycled descriptor received %q, want pong", buffer[:4])
	}
}

// Builds one accepted/connected loopback socket pair through the loop, for tests that need a live
// server-side socket with a client on the other end.
func loopback_pair(
	t *testing.T, loop io.IO, driver io.Driver, listener io.File, port int,
) (accepted io.File, connected io.File) {
	accepted = io.File(-1)
	connected, open_err := test_open_socket(loop)
	if open_err != nil {
		t.Fatalf("open socket: %v", open_err)
	}
	connect_done := false
	var accept_completion io.Completion
	loop.Accept(&accept_completion, func(_ *io.Completion, socket io.File, accept_err error) {
		if accept_err != nil {
			t.Errorf("accept: %v", accept_err)
		}
		accepted = socket
	}, listener, REAL_DEADLINE)
	var connect_completion io.Completion
	test_connect(loop,
		&connect_completion,
		func(_ *io.Completion, connect_err error) {
			if connect_err != nil {
				t.Errorf("connect: %v", connect_err)
			}
			connect_done = true
		},
		connected, "127.0.0.1", port,
	)
	driver.Run_Until(func() (finished bool) { return accepted > 0 && connect_done },
		REAL_DEADLINE)
	if accepted <= 0 {
		t.Fatalf("accept did not complete, got %d", accepted)
	}
	return accepted, connected
}

// Test_Operating_System_IO_Close_With_Armed_Receive verifies Close rejects a descriptor still
// borrowed by a submitted receive. The owner must shutdown, drain the receive callback, and only
// then close, matching third-party/tigerbeetle/src/message_bus.zig:1104-1145.
func Test_Operating_System_IO_Close_With_Armed_Receive(t *testing.T) {
	port := free_port(t)
	clock, _ := timeos.New_Operating_System_Clock()
	loop, driver := operating_system_loop(t, clock)
	listener, listen_err := test_listen(loop, "127.0.0.1", port)
	if listen_err != nil {
		t.Fatalf("listen: %v", listen_err)
	}
	accepted, connected := loopback_pair(t, loop, driver, listener, port)

	received := false
	var receive_completion io.Completion
	loop.Receive(&receive_completion, func(_ *io.Completion, _ int, _ error) {
		received = true
	}, accepted, make([]byte, 8))

	close_panicked := false
	var close_completion io.Completion
	func() {
		defer func() { close_panicked = recover() != nil }()
		loop.Close(&close_completion, func(_ *io.Completion, _ error) {}, accepted)
	}()
	if !close_panicked {
		t.Fatal("close with an armed receive must panic")
	}
	if shutdown_err := loop.Shutdown(accepted, io.SHUTDOWN_BOTH); shutdown_err != nil {
		t.Fatalf("shutdown: %v", shutdown_err)
	}
	if !operating_system_run_until(t, driver, func() (finished bool) { return received }) {
		t.Fatal("shutdown did not drain the armed receive")
	}

	self_exec_close(loop, driver, accepted)
	self_exec_close(loop, driver, connected)
	self_exec_close(loop, driver, listener)
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
	loop, driver := operating_system_loop(t, clock)
	file, open_err := loop.Open(source.Name())
	if open_err != nil {
		t.Fatalf("open: %v", open_err)
	}
	buffer := make([]byte, 5)
	count := -1
	read_done := false
	var completion io.Completion
	loop.Read(&completion, func(_ *io.Completion, bytes int, read_err error) {
		if read_err != nil {
			t.Errorf("read: %v", read_err)
		}
		count = bytes
		read_done = true
	}, file, buffer, 0)
	if !operating_system_run_until(t, driver, func() (finished bool) { return read_done }) {
		t.Fatal("read did not complete")
	}

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
	loop, driver := operating_system_loop(t, clock)
	file, create_err := loop.Create(source.Name())
	if create_err != nil {
		t.Fatalf("create: %v", create_err)
	}
	count := -1
	write_done := false
	var completion io.Completion
	loop.Write(&completion, func(_ *io.Completion, bytes int, write_err error) {
		if write_err != nil {
			t.Errorf("write: %v", write_err)
		}
		count = bytes
		write_done = true
	}, file, []byte("world"), 0)
	if !operating_system_run_until(t, driver, func() (finished bool) { return write_done }) {
		t.Fatal("write did not complete")
	}

	if count != 5 {
		t.Fatalf("wrote %d bytes, want 5", count)
	}
	verify, open_err := loop.Open(source.Name())
	if open_err != nil {
		t.Fatalf("open: %v", open_err)
	}
	buffer := make([]byte, 5)
	read := -1
	read_done := false
	var read_completion io.Completion
	loop.Read(&read_completion, func(_ *io.Completion, bytes int, read_err error) {
		if read_err != nil {
			t.Errorf("read back: %v", read_err)
		}
		read = bytes
		read_done = true
	}, verify, buffer, 0)
	if !operating_system_run_until(t, driver, func() (finished bool) { return read_done }) {
		t.Fatal("read back did not complete")
	}
	if read != 5 {
		t.Fatalf("read back %d bytes, want 5", read)
	}
	if string(buffer) != "world" {
		t.Fatalf("file holds %q, want world", buffer)
	}
}

// Test_Operating_System_IO_Tiger_Beetle_File_Parity ports the file-operation chain from
// third-party/tigerbeetle/src/io/test.zig:25-155: openat, write, fsync, read, and close all reuse
// caller-owned completions and preserve the written bytes.
func Test_Operating_System_IO_Tiger_Beetle_File_Parity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tigerbeetle-io")
	clock, _ := timeos.New_Operating_System_Clock()
	loop, driver := operating_system_loop(t, clock)
	opened := io.File(-1)
	var open_completion io.Completion
	loop.Open_At(&open_completion, func(_ *io.Completion, file io.File, err error) {
		if err != nil {
			t.Errorf("open at: %v", err)
			return
		}
		opened = file
	}, io.DIRECTORY_CURRENT, path, io.Open_At_Options{
		Access: io.OPEN_READ_WRITE, Create: true, Truncate: true, Mode: 0o600,
	})
	if !operating_system_run_until(t, driver, func() (finished bool) { return opened >= 0 }) {
		t.Fatal("open at did not complete")
	}

	written := false
	var write_completion io.Completion
	loop.Write(&write_completion, func(_ *io.Completion, count int, err error) {
		if err != nil {
			t.Errorf("write: %v", err)
		}
		written = count == 5
	}, opened, []byte("hello"), 10)
	if !operating_system_run_until(t, driver, func() (finished bool) { return written }) {
		t.Fatal("write did not complete")
	}

	synced := false
	var fsync_completion io.Completion
	loop.Fsync(&fsync_completion, func(_ *io.Completion, err error) {
		if err != nil {
			t.Errorf("fsync: %v", err)
		}
		synced = true
	}, opened)
	if !operating_system_run_until(t, driver, func() (finished bool) { return synced }) {
		t.Fatal("fsync did not complete")
	}

	buffer := make([]byte, 5)
	read := false
	var read_completion io.Completion
	loop.Read(&read_completion, func(_ *io.Completion, count int, err error) {
		if err != nil {
			t.Errorf("read: %v", err)
		}
		read = count == len(buffer)
	}, opened, buffer, 10)
	if !operating_system_run_until(t, driver, func() (finished bool) { return read }) {
		t.Fatal("read did not complete")
	}
	if string(buffer) != "hello" {
		t.Fatalf("read %q, want hello", buffer)
	}
	self_exec_close(loop, driver, opened)
}

// Test_Operating_System_IO_Open_At_No_Follow verifies that OPEN_AT_NO_FOLLOW rejects a symbolic
// link in the final path part and does not reject an ordinary file.
func Test_Operating_System_IO_Open_At_No_Follow(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	link := filepath.Join(root, "link")
	if write_err := os.WriteFile(target, []byte("secret"), 0o600); write_err != nil {
		t.Fatalf("write target: %v", write_err)
	}
	if link_err := os.Symlink(target, link); link_err != nil {
		t.Fatalf("make symbolic link: %v", link_err)
	}

	clock, _ := timeos.New_Operating_System_Clock()
	loop, driver := operating_system_loop(t, clock)
	assert_open_at_result(t, loop, driver, target, nil)
	assert_open_at_result(t, loop, driver, link, errors.New("symbolic link must fail"))
}

// Opens one path with no-follow and compares the result class with expected_error.
func assert_open_at_result(
	t *testing.T,
	loop io.IO,
	driver io.Driver,
	path string,
	expected_error error,
) {
	t.Helper()
	opened := io.File(-1)
	completed := false
	var open_err error
	var completion io.Completion
	loop.Open_At(&completion, func(_ *io.Completion, file io.File, err error) {
		opened = file
		open_err = err
		completed = true
	}, io.DIRECTORY_CURRENT, path, io.Open_At_Options{
		Access: io.OPEN_READ_ONLY,
		Flags:  io.OPEN_AT_NO_FOLLOW,
	})
	if !operating_system_run_until(t, driver, func() (finished bool) { return completed }) {
		t.Fatal("open at did not complete")
	}
	if expected_error == nil {
		if open_err != nil {
			t.Fatalf("ordinary open: %v", open_err)
		}
		self_exec_close(loop, driver, opened)
		return
	}
	if open_err == nil {
		self_exec_close(loop, driver, opened)
		t.Fatal(expected_error)
	}
}

// Test_Operating_System_IO_Event ports TigerBeetle's Event reattachment contract: one trigger
// retires one listener on the loop thread, after which the same completion may be armed again.
func Test_Operating_System_IO_Event(t *testing.T) {
	clock, _ := timeos.New_Operating_System_Clock()
	loop, driver := operating_system_loop(t, clock)
	event, open_err := loop.Open_Event()
	if open_err != nil {
		t.Fatalf("open event: %v", open_err)
	}
	fired := 0
	var completion io.Completion
	callback := func(_ *io.Completion) { fired++ }
	loop.Event_Listen(event, &completion, callback)
	loop.Event_Trigger(event, &completion)
	if !operating_system_run_until(t, driver, func() (finished bool) { return fired == 1 }) {
		t.Fatal("first event did not fire")
	}
	loop.Event_Listen(event, &completion, callback)
	loop.Event_Trigger(event, &completion)
	if !operating_system_run_until(t, driver, func() (finished bool) { return fired == 2 }) {
		t.Fatal("second event did not fire")
	}
	loop.Close_Event(event)
}

// Test_Operating_System_IO_Peer_Address reports the remote address of an accepted
// loopback connection.
func Test_Operating_System_IO_Peer_Address(t *testing.T) {
	port := free_port(t)
	clock, _ := timeos.New_Operating_System_Clock()
	loop, driver := operating_system_loop(t, clock)
	listener, listen_err := test_listen(loop, "127.0.0.1", port)
	if listen_err != nil {
		t.Fatalf("listen: %v", listen_err)
	}

	accepted := io.File(-1)
	var accept_completion io.Completion
	loop.Accept(&accept_completion, func(_ *io.Completion, socket io.File, err error) {
		accepted = socket
	}, listener, REAL_DEADLINE)
	var connect_completion io.Completion
	connected, open_err := test_open_socket(loop)
	if open_err != nil {
		t.Fatalf("open socket: %v", open_err)
	}
	test_connect(loop,
		&connect_completion,
		func(_ *io.Completion, err error) {},
		connected, "127.0.0.1", port,
	)
	driver.Run_Until(func() (finished bool) { return accepted > 0 }, REAL_DEADLINE)

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

// Test_Operating_System_IO_Deinit_Rejects_Undrained_Extension verifies Deinit cannot close the
// Event/backend while a repository-extension completion is still owned by Spawn. Spawn is the
// vehicle because it retires through the same off-loop post path the Event bridges.
func Test_Operating_System_IO_Deinit_Rejects_Undrained_Extension(t *testing.T) {
	clock, _ := timeos.New_Operating_System_Clock()
	loop, driver := operating_system_loop(t, clock)
	drained := false
	var completion io.Completion
	loop.Spawn(&completion, func(
		_ *io.Completion, _ io.Process_Result, _ error,
	) {
		drained = true
	}, io.Process_Request{Path: "true"}, REAL_DEADLINE)
	deinit_panicked := false
	func() {
		defer func() { deinit_panicked = recover() != nil }()
		driver.Deinit()
	}()
	if !deinit_panicked {
		t.Fatal("deinit with an undrained extension completion must panic")
	}
	if !operating_system_run_until(t, driver, func() (finished bool) { return drained }) {
		t.Fatal("spawn did not drain after rejected deinit")
	}
	driver.Deinit()
}

// Test_Operating_System_IO_Watch_Signal delivers a real SIGTERM onto the loop.
func Test_Operating_System_IO_Watch_Signal(t *testing.T) {
	clock, _ := timeos.New_Operating_System_Clock()
	loop, driver := operating_system_loop(t, clock)
	got := io.Signal(-1)
	fired := 0
	var completion io.Completion
	loop.Watch_Signal(&completion, func(
		_ *io.Completion, signal io.Signal, err error,
	) {
		if err != nil {
			t.Errorf("watch signal: %v", err)
		}
		fired++
		got = signal
	}, io.SIGNAL_TERMINATE, REAL_DEADLINE)
	if kill_err := syscall.Kill(os.Getpid(), syscall.SIGTERM); kill_err != nil {
		t.Fatalf("kill: %v", kill_err)
	}
	driver.Run_Until(func() (finished bool) { return fired > 0 }, REAL_DEADLINE)

	if fired != 1 {
		t.Fatalf("signal callback fired %d times, want 1", fired)
	}
	if got != io.SIGNAL_TERMINATE {
		t.Fatalf("signal = %d, want SIGNAL_TERMINATE", got)
	}
}

// Test_Operating_System_IO_Watch_Signal_Deadline proves a signal that never arrives retires the
// extension completion exactly once and permits backend deinitialization.
func Test_Operating_System_IO_Watch_Signal_Deadline(t *testing.T) {
	clock, _ := timeos.New_Operating_System_Clock()
	loop, driver := operating_system_loop(t, clock)
	callback_count := 0
	got := io.Signal(-1)
	var operation_err error
	var completion io.Completion
	loop.Watch_Signal(&completion, func(
		_ *io.Completion, signal io.Signal, err error,
	) {
		callback_count++
		got = signal
		operation_err = err
	}, io.SIGNAL_TERMINATE, REAL_OPERATION_DEADLINE)
	if !operating_system_run_until(
		t, driver, func() (finished bool) { return callback_count > 0 },
	) {
		t.Fatal("signal deadline did not resolve")
	}
	if callback_count != 1 {
		t.Fatalf("signal callback count = %d, want 1", callback_count)
	}
	if operation_err != io.Deadline_Exceeded {
		t.Fatalf("signal error = %v, want %v", operation_err, io.Deadline_Exceeded)
	}
	if got != -1 {
		t.Fatalf("deadline yielded signal %d", got)
	}
	driver.Deinit()
}

// Test_Operating_System_IO_Spawn runs real commands through the loop: a success with
// captured output, and a non-zero exit reported without a start error.
func Test_Operating_System_IO_Spawn(t *testing.T) {
	clock, _ := timeos.New_Operating_System_Clock()
	loop, driver := operating_system_loop(t, clock)

	echo := io.Process_Result{}
	echoed := false
	var echo_completion io.Completion
	loop.Spawn(&echo_completion, func(_ *io.Completion, result io.Process_Result, err error) {
		if err != nil {
			t.Errorf("echo spawn: %v", err)
		}
		echo = result
		echoed = true
	}, io.Process_Request{Path: "/bin/echo", Arguments: []string{"hi"}}, REAL_DEADLINE)
	operating_system_run_until(t, driver, func() (finished bool) { return echoed })

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
	}, io.Process_Request{Path: "/bin/sh", Arguments: []string{"-c", "exit 1"}}, REAL_DEADLINE)
	operating_system_run_until(t, driver, func() (finished bool) { return failed })

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
	loop, driver := operating_system_loop(t, clock)

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
	}, io.Process_Request{Path: "/bin/echo", Arguments: []string{"hi"}, Stdout: &streamed},
		REAL_DEADLINE)
	operating_system_run_until(t, driver, func() (finished bool) { return done })

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

// Test_Operating_System_IO_Spawn_Deadline proves timeout kills the entire subprocess group,
// preserves output captured before expiry, and delivers one terminal callback.
func Test_Operating_System_IO_Spawn_Deadline(t *testing.T) {
	clock, _ := timeos.New_Operating_System_Clock()
	loop, driver := operating_system_loop(t, clock)
	process_path := filepath.Join(t.TempDir(), "process")
	request := io.Process_Request{
		Path: "/bin/sh",
		Arguments: []string{
			"-c", "printf '%d' $$ > \"$1\"; printf partial; sleep 30 & wait",
			"bounded-spawn", process_path,
		},
	}
	callback_count := 0
	result := io.Process_Result{}
	var operation_err error
	var completion io.Completion
	loop.Spawn(&completion, func(
		_ *io.Completion, spawned io.Process_Result, err error,
	) {
		callback_count++
		result = spawned
		operation_err = err
	}, request, 100*time.MILLISECOND)
	if !operating_system_run_until(
		t, driver, func() (finished bool) { return callback_count > 0 },
	) {
		t.Fatal("bounded spawn did not complete")
	}
	if callback_count != 1 {
		t.Fatalf("spawn callback count = %d, want 1", callback_count)
	}
	if operation_err != io.Deadline_Exceeded {
		t.Fatalf("spawn error = %v, want %v", operation_err, io.Deadline_Exceeded)
	}
	if string(result.Output) != "partial" {
		t.Fatalf("partial output = %q, want partial", result.Output)
	}
	process_file, open_err := os.Open(process_path)
	if open_err != nil {
		t.Fatalf("open process identifier: %v", open_err)
	}
	process_buffer := make([]byte, 64)
	process_count, read_err := stdio.ReadFull(
		stdio.LimitReader(process_file, int64(len(process_buffer))), process_buffer,
	)
	if read_err != nil {
		if read_err != stdio.ErrUnexpectedEOF {
			t.Fatalf("read process identifier: %v", read_err)
		}
	}
	if close_err := process_file.Close(); close_err != nil {
		t.Fatalf("close process identifier: %v", close_err)
	}
	process_bytes := process_buffer[:process_count]
	process_identifier, parse_err := strconv.Atoi(strings.TrimSpace(string(process_bytes)))
	if parse_err != nil {
		t.Fatalf("parse process identifier: %v", parse_err)
	}
	group_exited, group_err := process_group_wait_for_exit(driver, process_identifier)
	if group_err != nil {
		t.Fatalf("wait for subprocess group %d: %v", process_identifier, group_err)
	}
	if !group_exited {
		t.Fatalf("subprocess group %d remains after deadline", process_identifier)
	}
	driver.Run_For(2 * REAL_OPERATION_DEADLINE)
	if callback_count != 1 {
		t.Fatalf("late spawn callback count = %d, want 1", callback_count)
	}
	driver.Deinit()
}

// Waits for host init to reap killed grandchildren without accepting a live bounded process.
func process_group_wait_for_exit(
	driver io.Driver, process_identifier int,
) (exited bool, err error) {
	for attempt_index := 0; attempt_index < 80; attempt_index++ {
		kill_err := syscall.Kill(-process_identifier, 0)
		if errors.Is(kill_err, syscall.ESRCH) {
			return true, nil
		}
		if kill_err != nil {
			return false, kill_err
		}
		drive_err := driver.Run_For(REAL_OPERATION_DEADLINE)
		if drive_err != nil {
			return false, drive_err
		}
	}
	return false, nil
}

// Test_Operating_System_IO_Directory exercises the filesystem-traversal ops on a real temp
// tree: Make_Directory builds a nested path (into which the fixture file is seeded), and
// Status and Read_Directory then report the tree's shape, including an absent path.
func Test_Operating_System_IO_Directory(t *testing.T) {
	clock, _ := timeos.New_Operating_System_Clock()
	loop, _ := operating_system_loop(t, clock)

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
	if directory_status.Is_Regular {
		t.Fatalf("nested status = %+v, want a non-regular file", directory_status)
	}
	regular_status, _ := loop.Status(file_path)
	if !regular_status.Exists {
		t.Fatalf("file status = %+v, want an existing path", regular_status)
	}
	if regular_status.Is_Directory {
		t.Fatalf("file status = %+v, want a non-directory", regular_status)
	}
	if !regular_status.Is_Regular {
		t.Fatalf("file status = %+v, want a regular file", regular_status)
	}
	absent_status, _ := loop.Status(filepath.Join(root, "nope"))
	if absent_status.Exists {
		t.Fatalf("absent status = %+v, want not exists", absent_status)
	}
	if absent_status.Is_Regular {
		t.Fatalf("absent status = %+v, want a non-regular file", absent_status)
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

type loopback_roundtrip_input struct {
	Test      *testing.T
	Loop      io.IO
	Driver    io.Driver
	Connected io.File
	Accepted  io.File
}

func loopback_assert_roundtrip(input *loopback_roundtrip_input) {
	input.Test.Helper()
	var send_completion io.Completion
	input.Loop.Send(&send_completion, func(_ *io.Completion, _ int, send_err error) {
		if send_err != nil {
			input.Test.Errorf("send: %v", send_err)
		}
	}, input.Connected, []byte("ping"))
	buffer := make([]byte, 16)
	received := -1
	var receive_completion io.Completion
	input.Loop.Receive(&receive_completion, func(
		_ *io.Completion, count int, receive_err error,
	) {
		if receive_err != nil {
			input.Test.Errorf("receive: %v", receive_err)
		}
		received = count
	}, input.Accepted, buffer)
	input.Driver.Run_Until(func() (finished bool) { return received >= 0 }, REAL_DEADLINE)
	if received != 4 {
		input.Test.Fatalf("received %d bytes, want 4", received)
	}
	if string(buffer[:4]) != "ping" {
		input.Test.Fatalf("received %q, want ping", buffer[:4])
	}
}

func socket_fcntl(t *testing.T, socket io.File, command int) (flags int) {
	t.Helper()
	value, _, errno := syscall.Syscall(
		syscall.SYS_FCNTL, uintptr(socket), uintptr(command), 0)
	if errno != 0 {
		t.Fatalf("fcntl %d: %v", command, errno)
	}
	return int(value)
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

// Closes a socket asynchronously and drives its completion.
func self_exec_close(loop io.IO, driver io.Driver, socket io.File) {
	closed := false
	var completion io.Completion
	loop.Close(&completion, func(_ *io.Completion, _ error) { closed = true }, socket)
	driver.Run_Until(func() (finished bool) { return closed }, REAL_DEADLINE)
}
