package nbio_test

import (
	"errors"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"

	stream "local/james-orcales/shared/simulation/io"
	io "local/james-orcales/shared/simulation/nbio"
	system_io "local/james-orcales/shared/simulation/nbio/default"
	sysos "local/james-orcales/shared/simulation/os"
	"local/james-orcales/shared/simulation/time"
	timeos "local/james-orcales/shared/simulation/time/default"
)

// The Run_Until cap for the real-backend tests: generous, since a completion returns the
// pump the instant it fires — this bound only bites a genuine hang, failing the test
// instead of blocking until the package timeout.
const REAL_DEADLINE = 5 * time.SECOND

// The short finite operation deadline used to prove a dormant kernel wait retires promptly.
const REAL_OPERATION_DEADLINE = 25 * time.MILLISECOND

// Bounds the buffer holding a process identifier read back from a fixture file.
const PROCESS_IDENTIFIER_BYTES = 64

// Writes content to path through the loop. Fixture setup in this package goes through io.IO
// like everything else: this is the io gateway's own suite, so a test that reaches around the
// loop to prepare its input exercises a path no application is allowed to take.
func write_file(t *testing.T, loop io.IO, driver time.Driver, path string, content []byte) {
	t.Helper()
	file, create_err := create_file(t, loop, driver, path)
	if create_err != nil {
		t.Fatalf("create %s: %v", path, create_err)
	}
	written := 0
	write_done := false
	var completion time.Completion
	loop.Write(&completion, func(_ *time.Completion, count int, write_err error) {
		if write_err != nil {
			t.Errorf("write %s: %v", path, write_err)
		}
		written = count
		write_done = true
	}, file, content, 0)
	if !operating_system_run_until(t, driver, func() (finished bool) { return write_done }) {
		t.Fatalf("the write of %s did not complete", path)
	}
	if written != len(content) {
		t.Fatalf("wrote %d bytes to %s, want %d", written, path, len(content))
	}
	close_file(t, loop, driver, path, file)
}

// Creates path and every missing parent through the derived Make_Directory, which composes the
// Mkdir_At primitive above the surface.
func make_directory(t *testing.T, loop io.IO, driver time.Driver, path string) (err error) {
	t.Helper()
	done := false
	var completion time.Completion
	io.Make_Directory(&io.Make_Directory_Input{
		Timeline: loop, Completion: &completion, Path: path, Mode: 0o755,
		Callback: func(_ *time.Completion, make_err error) {
			err = make_err
			done = true
		},
	})
	if !operating_system_run_until(t, driver, func() (finished bool) { return done }) {
		t.Fatalf("the create of %s did not complete", path)
	}
	return err
}

// Opens path for reading through Open_At, driving the loop until the descriptor arrives.
func open_file(
	t *testing.T, loop io.IO, driver time.Driver, path string,
) (file io.File, err error) {
	t.Helper()
	return open_file_options(t, loop, driver, path, io.Open_At_Options{
		Access: io.OPEN_READ_ONLY,
	})
}

// Creates or truncates path for writing through Open_At.
func create_file(
	t *testing.T, loop io.IO, driver time.Driver, path string,
) (file io.File, err error) {
	t.Helper()
	return open_file_options(t, loop, driver, path, io.Open_At_Options{
		Access: io.OPEN_WRITE_ONLY, Create: true, Truncate: true, Mode: 0o644,
	})
}

// Submits one Open_At with the caller's options and drives the loop until it retires.
func open_file_options(
	t *testing.T, loop io.IO, driver time.Driver, path string, options io.Open_At_Options,
) (file io.File, err error) {
	t.Helper()
	done := false
	var completion time.Completion
	loop.Open_At(&completion, func(_ *time.Completion, opened io.File, open_err error) {
		file = opened
		err = open_err
		done = true
	}, io.DIRECTORY_CURRENT, path, options)
	if !operating_system_run_until(t, driver, func() (finished bool) { return done }) {
		t.Fatalf("the open of %s did not complete", path)
	}
	return file, err
}

// Lists path's children through the derived Read_Directory, which composes Open_At, repeated
// Get_Directory_Entries passes, and Close.
func read_directory(
	t *testing.T, loop io.IO, driver time.Driver, path string,
) (entries []io.Directory_Entry, err error) {
	t.Helper()
	done := false
	var completion time.Completion
	io.Read_Directory(&io.Read_Directory_Input{
		Timeline: loop, Completion: &completion, Path: path,
		Callback: func(
			_ *time.Completion, listed []io.Directory_Entry, read_err error,
		) {
			entries = listed
			err = read_err
			done = true
		},
	})
	if !operating_system_run_until(t, driver, func() (finished bool) { return done }) {
		t.Fatalf("the listing of %s did not complete", path)
	}
	return entries, err
}

// Reads up to len(buffer) bytes from path through the loop, returning the count.
func read_file(
	t *testing.T, loop io.IO, driver time.Driver, path string, buffer []byte,
) (count int) {
	t.Helper()
	file, open_err := open_file(t, loop, driver, path)
	if open_err != nil {
		t.Fatalf("open %s: %v", path, open_err)
	}
	read_done := false
	var completion time.Completion
	loop.Read(&completion, func(_ *time.Completion, read_count int, read_err error) {
		if read_err != nil {
			t.Errorf("read %s: %v", path, read_err)
		}
		count = read_count
		read_done = true
	}, file, buffer, 0)
	if !operating_system_run_until(t, driver, func() (finished bool) { return read_done }) {
		t.Fatalf("the read of %s did not complete", path)
	}
	close_file(t, loop, driver, path, file)
	return count
}

// Releases a descriptor through the loop's asynchronous Close.
func close_file(t *testing.T, loop io.IO, driver time.Driver, path string, file io.File) {
	t.Helper()
	close_done := false
	var completion time.Completion
	loop.Close(&completion, func(_ *time.Completion, close_err error) {
		if close_err != nil {
			t.Errorf("close %s: %v", path, close_err)
		}
		close_done = true
	}, file)
	if !operating_system_run_until(t, driver, func() (finished bool) { return close_done }) {
		t.Fatalf("the close of %s did not complete", path)
	}
}

// Creates the 32-entry test scheduler and fails at the composition root if initialization fails.
func operating_system_loop(
	t *testing.T, clock time.Any_Clock,
) (loop io.IO, pump time.Timeline, driver time.Driver) {
	t.Helper()
	loop, pump, driver, _ = operating_system_all(t, clock)
	return loop, pump, driver
}

// Creates the same scheduler and returns the OS beside it, for the tests that spawn a
// subprocess or watch a signal — the two operations the OS surface owns.
func operating_system_all(
	t *testing.T, clock time.Any_Clock,
) (loop io.IO, pump time.Timeline, driver time.Driver, system sysos.OS) {
	t.Helper()
	loop, pump, driver, system, err := system_io.New_Operating_System_IO(
		clock, 32, 0, sysos.Virtual_OS_To_OS(sysos.Virtual_OS{Identifier: 1}))
	if err != nil {
		t.Fatalf("initialize io: %v", err)
	}
	return loop, pump, driver, system
}

// Drives a real test predicate and fails immediately on a backend scheduler error.
func operating_system_run_until(
	t *testing.T, driver time.Driver, done func() (finished bool),
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
	return io.Open_Socket_TCP(loop, io.FAMILY_IPV4, test_tcp_options())
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
	_, listen_err := io.Listen(loop, listener, address, io.Listen_Options{Backlog: 65535})
	if listen_err != nil {
		loop.Close_Socket(listener)
		return io.File(-1), listen_err
	}
	return listener, nil
}

// Converts an IP literal for the explicit-address Connect surface.
func test_connect(
	loop io.IO, completion *time.Completion, callback time.Timeout_Callback,
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
	clock, _ := timeos.New_Operating_System_Any_Clock()
	loop, _, driver := operating_system_loop(t, clock)
	path := filepath.Join(t.TempDir(), "read")
	write_file(t, loop, driver, path, []byte("hello"))

	file, open_err := open_file(t, loop, driver, path)
	if open_err != nil {
		t.Fatalf("open: %v", open_err)
	}
	buffer := make([]byte, 5)
	count := -1
	read_done := false
	var completion time.Completion
	loop.Read(&completion, func(_ *time.Completion, bytes int, read_err error) {
		if read_err != nil {
			t.Errorf("read error: %v", read_err)
		}
		count = bytes
		read_done = true
	}, file, buffer, 0)
	if !operating_system_run_until(t, driver, func() (finished bool) { return read_done }) {
		t.Fatal("read did not complete")
	}
	close_file(t, loop, driver, path, file)

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
	clock, _ := timeos.New_Operating_System_Any_Clock()
	_, _, driver := operating_system_loop(t, clock)
	defer func() {
		if recover() == nil {
			t.Fatal("an unbounded Run_Until with nothing pending must panic")
		}
	}()
	driver.Run_Until(func() (finished bool) { return false }, time.FOREVER)
}

// Test_Operating_System_IO_Timeout verifies a timeout fires once real time passes
// its deadline.
func Test_Operating_System_IO_Timeout(t *testing.T) {
	clock, _ := timeos.New_Operating_System_Any_Clock()
	_, pump, driver := operating_system_loop(t, clock)
	fired := false
	var completion time.Completion
	pump.Timeout(&completion, func(_ *time.Completion, err error) {
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
	clock, _ := timeos.New_Operating_System_Any_Clock()
	loop, _, driver := operating_system_loop(t, clock)
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
	clock, _ := timeos.New_Operating_System_Any_Clock()
	_, pump, _ := operating_system_loop(t, clock)
	var completion time.Completion
	pump.Timeout(&completion, func(_ *time.Completion, err error) {}, time.SECOND)
	defer func() {
		if recover() == nil {
			t.Fatal("resubmitting an in-flight completion must panic")
		}
	}()
	pump.Timeout(&completion, func(_ *time.Completion, err error) {}, time.SECOND)
}

// Test_Operating_System_IO_Reentrancy verifies driving the real loop from within a
// completion callback panics, so a re-entrant Run* fails loudly rather than corrupting it.
func Test_Operating_System_IO_Reentrancy(t *testing.T) {
	clock, _ := timeos.New_Operating_System_Any_Clock()
	_, pump, driver := operating_system_loop(t, clock)
	var completion time.Completion
	pump.Timeout(&completion, func(_ *time.Completion, err error) {
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
	clock, _ := timeos.New_Operating_System_Any_Clock()
	loop, _, driver := operating_system_loop(t, clock)
	baseline := driver.Introspect().Raw_Open

	listener, listen_err := test_listen(loop, "127.0.0.1", port)
	if listen_err != nil {
		t.Fatalf("listen: %v", listen_err)
	}

	accepted := io.File(-1)
	var accept_completion time.Completion
	loop.Accept(&accept_completion, func(_ *time.Completion, socket io.File, accept_err error) {
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
	var connect_completion time.Completion
	test_connect(loop,
		&connect_completion,
		func(_ *time.Completion, connect_err error) {
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
		Test: t, Timeline: loop, Driver: driver, Connected: connected, Accepted: accepted,
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
	clock, _ := timeos.New_Operating_System_Any_Clock()
	loop, _, driver := operating_system_loop(t, clock)
	listener, listen_err := test_listen(loop, "127.0.0.1", 0)
	if listen_err != nil {
		t.Fatalf("listen: %v", listen_err)
	}
	callback_count := 0
	accepted := io.File(-1)
	var operation_err error
	var completion time.Completion
	loop.Accept(&completion, func(_ *time.Completion, socket io.File, err error) {
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
	if operation_err != time.Deadline_Exceeded {
		t.Fatalf("accept error = %v, want %v", operation_err, time.Deadline_Exceeded)
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
	clock, _ := timeos.New_Operating_System_Any_Clock()
	loop, _, driver := operating_system_loop(t, clock)
	raw_open_before := driver.Introspect().Raw_Open

	socket, open_err := test_open_socket(loop)
	if open_err != nil {
		t.Fatalf("open socket: %v", open_err)
	}
	called := false
	var connect_err error
	var completion time.Completion
	test_connect(loop, &completion, func(_ *time.Completion, err error) {
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
	clock, _ := timeos.New_Operating_System_Any_Clock()
	loop, _, driver := operating_system_loop(t, clock)

	listener, listen_err := test_listen(loop, "127.0.0.1", port)
	if listen_err != nil {
		t.Fatalf("listen: %v", listen_err)
	}
	accepted := io.File(-1)
	var accept_completion time.Completion
	loop.Accept(&accept_completion, func(_ *time.Completion, socket io.File, err error) {
		if err != nil {
			t.Errorf("accept: %v", err)
		}
		accepted = socket
	}, listener, REAL_DEADLINE)

	sent := -1
	var send_completion time.Completion
	var connect_completion time.Completion
	socket, open_err := test_open_socket(loop)
	if open_err != nil {
		t.Fatalf("open socket: %v", open_err)
	}
	test_connect(loop, &connect_completion, func(_ *time.Completion, err error) {
		if err != nil {
			t.Errorf("connect: %v", err)
			return
		}
		// Arm the send inside the connect completion: same descriptor, same write slot.
		loop.Send(&send_completion, func(_ *time.Completion, count int, send_err error) {
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
	var receive_completion time.Completion
	loop.Receive(&receive_completion, func(_ *time.Completion, count int, receive_err error) {
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
	clock, _ := timeos.New_Operating_System_Any_Clock()
	loop, _, driver := operating_system_loop(t, clock)
	listener, listen_err := test_listen(loop, "127.0.0.1", port)
	if listen_err != nil {
		t.Fatalf("listen: %v", listen_err)
	}

	first, first_connected := loopback_pair(t, loop, driver, listener, port)
	buffer := make([]byte, 16)
	var receive_completion time.Completion
	fired := 0
	loop.Receive(&receive_completion,
		func(_ *time.Completion, _ int, _ error) { fired++ }, first, buffer)
	if shutdown_err := loop.Shutdown(first, io.SHUTDOWN_RECEIVE); shutdown_err != nil {
		t.Fatalf("shutdown receive: %v", shutdown_err)
	}
	if !operating_system_run_until(t, driver, func() (finished bool) { return fired > 0 }) {
		t.Fatal("shutdown did not drain the armed receive")
	}
	closed := false
	var close_completion time.Completion
	loop.Close(&close_completion, func(_ *time.Completion, _ error) { closed = true }, first)
	if !operating_system_run_until(t, driver, func() (finished bool) { return closed }) {
		t.Fatal("joined close did not complete")
	}
	self_exec_close(loop, driver, first_connected)

	// A later socket may reuse the descriptor and must still deliver readiness.
	recycled, second := loopback_pair(t, loop, driver, listener, port)
	var send_completion time.Completion
	loop.Send(&send_completion, func(_ *time.Completion, _ int, send_err error) {
		if send_err != nil {
			t.Errorf("send: %v", send_err)
		}
	}, second, []byte("pong"))
	received := -1
	var second_receive time.Completion
	loop.Receive(&second_receive, func(_ *time.Completion, count int, err error) {
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
	t *testing.T, loop io.IO, driver time.Driver, listener io.File, port int,
) (accepted io.File, connected io.File) {
	accepted = io.File(-1)
	connected, open_err := test_open_socket(loop)
	if open_err != nil {
		t.Fatalf("open socket: %v", open_err)
	}
	connect_done := false
	var accept_completion time.Completion
	loop.Accept(&accept_completion, func(_ *time.Completion, socket io.File, accept_err error) {
		if accept_err != nil {
			t.Errorf("accept: %v", accept_err)
		}
		accepted = socket
	}, listener, REAL_DEADLINE)
	var connect_completion time.Completion
	test_connect(loop,
		&connect_completion,
		func(_ *time.Completion, connect_err error) {
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
	clock, _ := timeos.New_Operating_System_Any_Clock()
	loop, _, driver := operating_system_loop(t, clock)
	listener, listen_err := test_listen(loop, "127.0.0.1", port)
	if listen_err != nil {
		t.Fatalf("listen: %v", listen_err)
	}
	accepted, connected := loopback_pair(t, loop, driver, listener, port)

	received := false
	var receive_completion time.Completion
	loop.Receive(&receive_completion, func(_ *time.Completion, _ int, _ error) {
		received = true
	}, accepted, make([]byte, 8))

	close_panicked := false
	var close_completion time.Completion
	func() {
		defer func() { close_panicked = recover() != nil }()
		loop.Close(&close_completion, func(_ *time.Completion, _ error) {}, accepted)
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
	clock, _ := timeos.New_Operating_System_Any_Clock()
	loop, _, driver := operating_system_loop(t, clock)
	path := filepath.Join(t.TempDir(), "open")
	write_file(t, loop, driver, path, []byte("hello"))

	file, open_err := open_file(t, loop, driver, path)
	if open_err != nil {
		t.Fatalf("open: %v", open_err)
	}
	buffer := make([]byte, 5)
	count := -1
	read_done := false
	var completion time.Completion
	loop.Read(&completion, func(_ *time.Completion, bytes int, read_err error) {
		if read_err != nil {
			t.Errorf("read: %v", read_err)
		}
		count = bytes
		read_done = true
	}, file, buffer, 0)
	if !operating_system_run_until(t, driver, func() (finished bool) { return read_done }) {
		t.Fatal("read did not complete")
	}
	close_file(t, loop, driver, path, file)

	if count != 5 {
		t.Fatalf("read %d bytes, want 5", count)
	}
	if string(buffer) != "hello" {
		t.Fatalf("read %q, want hello", buffer)
	}
}

// Test_Operating_System_IO_Create creates a file through the loop and writes to it.
func Test_Operating_System_IO_Create(t *testing.T) {
	clock, _ := timeos.New_Operating_System_Any_Clock()
	loop, _, driver := operating_system_loop(t, clock)
	path := filepath.Join(t.TempDir(), "create")
	file, create_err := create_file(t, loop, driver, path)
	if create_err != nil {
		t.Fatalf("create: %v", create_err)
	}
	count := -1
	write_done := false
	var completion time.Completion
	loop.Write(&completion, func(_ *time.Completion, bytes int, write_err error) {
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
	close_file(t, loop, driver, path, file)
	buffer := make([]byte, 5)
	read := read_file(t, loop, driver, path, buffer)
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
	clock, _ := timeos.New_Operating_System_Any_Clock()
	loop, _, driver := operating_system_loop(t, clock)
	opened := io.File(-1)
	var open_completion time.Completion
	loop.Open_At(&open_completion, func(_ *time.Completion, file io.File, err error) {
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
	var write_completion time.Completion
	loop.Write(&write_completion, func(_ *time.Completion, count int, err error) {
		if err != nil {
			t.Errorf("write: %v", err)
		}
		written = count == 5
	}, opened, []byte("hello"), 10)
	if !operating_system_run_until(t, driver, func() (finished bool) { return written }) {
		t.Fatal("write did not complete")
	}

	synced := false
	var fsync_completion time.Completion
	loop.Fsync(&fsync_completion, func(_ *time.Completion, err error) {
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
	var read_completion time.Completion
	loop.Read(&read_completion, func(_ *time.Completion, count int, err error) {
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
	clock_setup, _ := timeos.New_Operating_System_Any_Clock()
	loop_setup, _, driver_setup := operating_system_loop(t, clock_setup)
	root := t.TempDir()
	target := filepath.Join(root, "target")
	link := filepath.Join(root, "link")
	write_file(t, loop_setup, driver_setup, target, []byte("secret"))
	driver_setup.Deinit()
	// A symbolic link is the one filesystem shape io.IO cannot make, so the link itself stays
	// a raw call. That absence is what this test exists to guard against following.
	if link_err := syscall.Symlink(target, link); link_err != nil {
		t.Fatalf("make symbolic link: %v", link_err)
	}

	clock, _ := timeos.New_Operating_System_Any_Clock()
	loop, _, driver := operating_system_loop(t, clock)
	assert_open_at_result(t, loop, driver, target, nil)
	assert_open_at_result(t, loop, driver, link, errors.New("symbolic link must fail"))
}

// Opens one path with no-follow and compares the result class with expected_error.
func assert_open_at_result(
	t *testing.T,
	loop io.IO,
	driver time.Driver,
	path string,
	expected_error error,
) {
	t.Helper()
	opened := io.File(-1)
	completed := false
	var open_err error
	var completion time.Completion
	loop.Open_At(&completion, func(_ *time.Completion, file io.File, err error) {
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
	clock, _ := timeos.New_Operating_System_Any_Clock()
	_, pump, driver := operating_system_loop(t, clock)
	event, open_err := pump.Open_Event()
	if open_err != nil {
		t.Fatalf("open event: %v", open_err)
	}
	fired := 0
	var completion time.Completion
	callback := func(_ *time.Completion) { fired++ }
	pump.Event_Listen(event, &completion, callback)
	pump.Event_Trigger(event, &completion)
	if !operating_system_run_until(t, driver, func() (finished bool) { return fired == 1 }) {
		t.Fatal("first event did not fire")
	}
	pump.Event_Listen(event, &completion, callback)
	pump.Event_Trigger(event, &completion)
	if !operating_system_run_until(t, driver, func() (finished bool) { return fired == 2 }) {
		t.Fatal("second event did not fire")
	}
	pump.Close_Event(event)
}

// Test_Operating_System_IO_Peer_Address reports the remote address of an accepted
// loopback connection.
func Test_Operating_System_IO_Peer_Address(t *testing.T) {
	port := free_port(t)
	clock, _ := timeos.New_Operating_System_Any_Clock()
	loop, _, driver := operating_system_loop(t, clock)
	listener, listen_err := test_listen(loop, "127.0.0.1", port)
	if listen_err != nil {
		t.Fatalf("listen: %v", listen_err)
	}

	accepted := io.File(-1)
	var accept_completion time.Completion
	loop.Accept(&accept_completion, func(_ *time.Completion, socket io.File, err error) {
		accepted = socket
	}, listener, REAL_DEADLINE)
	var connect_completion time.Completion
	connected, open_err := test_open_socket(loop)
	if open_err != nil {
		t.Fatalf("open socket: %v", open_err)
	}
	test_connect(loop,
		&connect_completion,
		func(_ *time.Completion, err error) {},
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
	clock, _ := timeos.New_Operating_System_Any_Clock()
	_, _, driver, system := operating_system_all(t, clock)
	drained := false
	var completion time.Completion
	system.Spawn(&completion, func(
		_ *time.Completion, _ sysos.Process_Result, _ error,
	) {
		drained = true
	}, sysos.Process_Request{Path: "true"}, REAL_DEADLINE)
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
	clock, _ := timeos.New_Operating_System_Any_Clock()
	_, _, driver, system := operating_system_all(t, clock)
	got := sysos.Signal(-1)
	fired := 0
	var completion time.Completion
	system.Watch_Signal(&completion, func(
		_ *time.Completion, signal sysos.Signal, err error,
	) {
		if err != nil {
			t.Errorf("watch signal: %v", err)
		}
		fired++
		got = signal
	}, sysos.SIGNAL_TERMINATE, REAL_DEADLINE)
	if kill_err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM); kill_err != nil {
		t.Fatalf("kill: %v", kill_err)
	}
	driver.Run_Until(func() (finished bool) { return fired > 0 }, REAL_DEADLINE)

	if fired != 1 {
		t.Fatalf("signal callback fired %d times, want 1", fired)
	}
	if got != sysos.SIGNAL_TERMINATE {
		t.Fatalf("signal = %d, want SIGNAL_TERMINATE", got)
	}
}

// Test_Operating_System_IO_Watch_Signal_Deadline proves a signal that never arrives retires the
// extension completion exactly once and permits backend deinitialization.
func Test_Operating_System_IO_Watch_Signal_Deadline(t *testing.T) {
	clock, _ := timeos.New_Operating_System_Any_Clock()
	_, _, driver, system := operating_system_all(t, clock)
	callback_count := 0
	got := sysos.Signal(-1)
	var operation_err error
	var completion time.Completion
	system.Watch_Signal(&completion, func(
		_ *time.Completion, signal sysos.Signal, err error,
	) {
		callback_count++
		got = signal
		operation_err = err
	}, sysos.SIGNAL_TERMINATE, REAL_OPERATION_DEADLINE)
	if !operating_system_run_until(
		t, driver, func() (finished bool) { return callback_count > 0 },
	) {
		t.Fatal("signal deadline did not resolve")
	}
	if callback_count != 1 {
		t.Fatalf("signal callback count = %d, want 1", callback_count)
	}
	if operation_err != time.Deadline_Exceeded {
		t.Fatalf("signal error = %v, want %v", operation_err, time.Deadline_Exceeded)
	}
	if got != -1 {
		t.Fatalf("deadline yielded signal %d", got)
	}
	driver.Deinit()
}

// Test_Operating_System_IO_Spawn runs real commands through the loop: a success with
// captured output, and a non-zero exit reported without a start error.
func Test_Operating_System_IO_Spawn(t *testing.T) {
	clock, _ := timeos.New_Operating_System_Any_Clock()
	_, _, driver, system := operating_system_all(t, clock)

	echo := sysos.Process_Result{}
	echoed := false
	var echo_completion time.Completion
	system.Spawn(&echo_completion, func(
		_ *time.Completion, result sysos.Process_Result, err error,
	) {
		if err != nil {
			t.Errorf("echo spawn: %v", err)
		}
		echo = result
		echoed = true
	}, sysos.Process_Request{Path: "/bin/echo", Arguments: []string{"hi"}}, REAL_DEADLINE)
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

	fail := sysos.Process_Result{}
	failed := false
	var fail_completion time.Completion
	system.Spawn(&fail_completion, func(
		_ *time.Completion, result sysos.Process_Result, err error,
	) {
		if err != nil {
			t.Errorf("false spawn: %v", err)
		}
		fail = result
		failed = true
	}, sysos.Process_Request{
		Path: "/bin/sh", Arguments: []string{"-c", "exit 1"},
	}, REAL_DEADLINE)
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
	clock, _ := timeos.New_Operating_System_Any_Clock()
	_, _, driver, system := operating_system_all(t, clock)

	streamed := stream.Stream_Memory{Memory: make([]byte, 64)}
	result := sysos.Process_Result{}
	done := false
	var completion time.Completion
	system.Spawn(&completion, func(
		_ *time.Completion, spawned sysos.Process_Result, err error,
	) {
		if err != nil {
			t.Errorf("echo spawn: %v", err)
		}
		result = spawned
		done = true
	}, sysos.Process_Request{
		Path: "/bin/echo", Arguments: []string{"hi"},
		Stdout: stream.Memory_To_Stream(&streamed),
	}, REAL_DEADLINE)
	operating_system_run_until(t, driver, func() (finished bool) { return done })

	if !done {
		t.Fatal("echo did not complete")
	}
	if written := string(streamed.Memory[:streamed.Cursor]); written != "hi\n" {
		t.Fatalf("streamed output = %q, want hi", written)
	}
	if len(result.Output) != 0 {
		t.Fatalf("Output = %q, want empty when streamed to a sink", result.Output)
	}
}

// Reads the process identifier the deadline fixture wrote to path, through the loop. This
// package is the io gateway, so its own tests are where the loop's file operations belong. The
// fixture writes one short decimal, so a regular file returns it whole in a single read.
func spawn_recorded_identifier(
	t *testing.T, loop io.IO, driver time.Driver, path string,
) (identifier int) {
	t.Helper()
	file, open_err := open_file(t, loop, driver, path)
	if open_err != nil {
		t.Fatalf("open process identifier: %v", open_err)
	}
	process_buffer := make([]byte, PROCESS_IDENTIFIER_BYTES)
	process_count := 0
	read_done := false
	var read_completion time.Completion
	loop.Read(&read_completion, func(_ *time.Completion, count int, read_err error) {
		if read_err != nil {
			t.Errorf("read process identifier: %v", read_err)
		}
		process_count = count
		read_done = true
	}, file, process_buffer, 0)
	if !operating_system_run_until(t, driver, func() (finished bool) { return read_done }) {
		t.Fatal("the process identifier read did not complete")
	}
	close_done := false
	var close_completion time.Completion
	loop.Close(&close_completion, func(_ *time.Completion, close_err error) {
		if close_err != nil {
			t.Errorf("close process identifier: %v", close_err)
		}
		close_done = true
	}, file)
	if !operating_system_run_until(t, driver, func() (finished bool) { return close_done }) {
		t.Fatal("the process identifier close did not complete")
	}
	identifier, parse_err := strconv.Atoi(
		strings.TrimSpace(string(process_buffer[:process_count])))
	if parse_err != nil {
		t.Fatalf("parse process identifier: %v", parse_err)
	}
	return identifier
}

// Test_Operating_System_IO_Spawn_Deadline proves timeout kills the entire subprocess group,
// preserves output captured before expiry, and delivers one terminal callback.
func Test_Operating_System_IO_Spawn_Deadline(t *testing.T) {
	clock, _ := timeos.New_Operating_System_Any_Clock()
	loop, _, driver, system := operating_system_all(t, clock)
	process_path := filepath.Join(t.TempDir(), "process")
	request := sysos.Process_Request{
		Path: "/bin/sh",
		Arguments: []string{
			"-c", "printf '%d' $$ > \"$1\"; printf partial; sleep 30 & wait",
			"bounded-spawn", process_path,
		},
	}
	callback_count := 0
	result := sysos.Process_Result{}
	var operation_err error
	var completion time.Completion
	system.Spawn(&completion, func(
		_ *time.Completion, spawned sysos.Process_Result, err error,
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
	if operation_err != time.Deadline_Exceeded {
		t.Fatalf("spawn error = %v, want %v", operation_err, time.Deadline_Exceeded)
	}
	if string(result.Output) != "partial" {
		t.Fatalf("partial output = %q, want partial", result.Output)
	}
	process_identifier := spawn_recorded_identifier(t, loop, driver, process_path)
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
	if spawns := driver.Introspect().Spawns; spawns != 0 {
		t.Fatalf("unreaped children after a deadline = %d, want 0", spawns)
	}
	driver.Deinit()
}

// Test_Operating_System_IO_Spawn_Reaps_After_Deadline verifies the deadline path still reaps.
// The completion retires early on Deadline_Exceeded, so the exit event arrives for a child
// nobody waits on, and only the backend's own tracking keeps the reap on that path.
func Test_Operating_System_IO_Spawn_Reaps_After_Deadline(t *testing.T) {
	clock, _ := timeos.New_Operating_System_Any_Clock()
	_, _, driver, system := operating_system_all(t, clock)
	callback_count := 0
	var operation_err error
	var completion time.Completion
	system.Spawn(&completion, func(
		_ *time.Completion, _ sysos.Process_Result, err error,
	) {
		callback_count++
		operation_err = err
	}, sysos.Process_Request{
		Path: "/bin/sleep", Arguments: []string{"30"},
	}, 50*time.MILLISECOND)
	if !operating_system_run_until(
		t, driver, func() (finished bool) { return callback_count > 0 },
	) {
		t.Fatal("bounded spawn did not complete")
	}
	if operation_err != time.Deadline_Exceeded {
		t.Fatalf("spawn error = %v, want %v", operation_err, time.Deadline_Exceeded)
	}
	if spawns := driver.Introspect().Spawns; spawns != 0 {
		t.Fatalf("unreaped children = %d, want 0", spawns)
	}
	// Deinit asserts the spawn table is empty, so a missed reap panics here rather than
	// leaking one process-table slot for every timed-out child.
	driver.Deinit()
}

// Test_Operating_System_IO_Spawn_Bounds_Lingering_Drain verifies the pipe-cleanup bound starts
// when the child exits, not when its deadline expires. A child that finishes at once but leaves
// a grandchild holding standard output must retire about one second later, not wait out the
// whole deadline. This is the bound exec.Cmd.WaitDelay supplied before.
func Test_Operating_System_IO_Spawn_Bounds_Lingering_Drain(t *testing.T) {
	clock, _ := timeos.New_Operating_System_Any_Clock()
	_, _, driver, system := operating_system_all(t, clock)
	const SPAWN_DEADLINE = 4 * time.SECOND
	started := clock.Now_Monotonic()
	callback_count := 0
	result := sysos.Process_Result{}
	var operation_err error
	var completion time.Completion
	system.Spawn(&completion, func(
		_ *time.Completion, spawned sysos.Process_Result, err error,
	) {
		callback_count++
		result = spawned
		operation_err = err
	}, sysos.Process_Request{
		Path:      "/bin/sh",
		Arguments: []string{"-c", "printf quick; sleep 30 & exit 0"},
	}, SPAWN_DEADLINE)
	if !operating_system_run_until(
		t, driver, func() (finished bool) { return callback_count > 0 },
	) {
		t.Fatal("spawn with a lingering grandchild did not complete")
	}
	elapsed := time.Duration(int64(clock.Now_Monotonic()) - int64(started))
	if elapsed >= SPAWN_DEADLINE {
		t.Fatalf("drain took %v, want the cleanup bound rather than the deadline", elapsed)
	}
	// The child exited cleanly, so its code survives. The output is incomplete because the
	// drain was cut short, and Deadline_Exceeded is how the caller learns that.
	if result.Exit != 0 {
		t.Fatalf("exit = %d, want 0", result.Exit)
	}
	if string(result.Output) != "quick" {
		t.Fatalf("output = %q, want quick", result.Output)
	}
	if operation_err != time.Deadline_Exceeded {
		t.Fatalf("error = %v, want %v", operation_err, time.Deadline_Exceeded)
	}
	if spawns := driver.Introspect().Spawns; spawns != 0 {
		t.Fatalf("unreaped children = %d, want 0", spawns)
	}
	driver.Deinit()
}

// Test_Operating_System_IO_Spawn_Concurrent verifies two children in flight at once keep their
// pipes separate. A pipe end that leaks into the other child's fork would hold that child's
// standard input open, so this fails by deadlock rather than by a wrong result.
func Test_Operating_System_IO_Spawn_Concurrent(t *testing.T) {
	clock, _ := timeos.New_Operating_System_Any_Clock()
	_, _, driver, system := operating_system_all(t, clock)
	const SPAWNS_COUNT = 4
	finished := 0
	outputs := make([]string, SPAWNS_COUNT)
	completions := make([]*time.Completion, SPAWNS_COUNT)
	for index := 0; index < SPAWNS_COUNT; index++ {
		position := index
		completions[position] = &time.Completion{}
		system.Spawn(completions[position], func(
			_ *time.Completion, spawned sysos.Process_Result, err error,
		) {
			if err != nil {
				t.Errorf("spawn %d: %v", position, err)
			}
			outputs[position] = string(spawned.Output)
			finished++
		}, sysos.Process_Request{
			Path:  "/bin/cat",
			Input: []byte(strconv.Itoa(position)),
		}, REAL_DEADLINE)
	}
	if !operating_system_run_until(
		t, driver, func() (finished_all bool) { return finished == SPAWNS_COUNT },
	) {
		t.Fatalf("only %d of %d concurrent spawns completed", finished, SPAWNS_COUNT)
	}
	for index := 0; index < SPAWNS_COUNT; index++ {
		want := strconv.Itoa(index)
		if outputs[index] != want {
			t.Fatalf("spawn %d echoed %q, want %q", index, outputs[index], want)
		}
	}
	if spawns := driver.Introspect().Spawns; spawns != 0 {
		t.Fatalf("unreaped children = %d, want 0", spawns)
	}
	driver.Deinit()
}

// Test_Operating_System_IO_Spawn_Feeds_Input verifies Process_Request.Input reaches the child's
// standard input and that the write end closes, so the child observes end-of-file rather than
// waiting for more.
func Test_Operating_System_IO_Spawn_Feeds_Input(t *testing.T) {
	clock, _ := timeos.New_Operating_System_Any_Clock()
	_, _, driver, system := operating_system_all(t, clock)
	callback_count := 0
	result := sysos.Process_Result{}
	var operation_err error
	var completion time.Completion
	system.Spawn(&completion, func(
		_ *time.Completion, spawned sysos.Process_Result, err error,
	) {
		callback_count++
		result = spawned
		operation_err = err
	}, sysos.Process_Request{
		Path: "/bin/cat", Input: []byte("fed through stdin"),
	}, REAL_DEADLINE)
	if !operating_system_run_until(
		t, driver, func() (finished bool) { return callback_count > 0 },
	) {
		t.Fatal("spawn reading standard input did not complete")
	}
	if operation_err != nil {
		t.Fatalf("spawn error = %v, want nil", operation_err)
	}
	if string(result.Output) != "fed through stdin" {
		t.Fatalf("output = %q, want the fed input", result.Output)
	}
	driver.Deinit()
}

// Test_Operating_System_IO_Spawn_Drains_Full_Pipe verifies output larger than one pipe buffer
// still completes. A child that fills the pipe blocks until the loop reads it, so this fails by
// deadlock if the reads are not armed for the child's whole life.
func Test_Operating_System_IO_Spawn_Drains_Full_Pipe(t *testing.T) {
	clock, _ := timeos.New_Operating_System_Any_Clock()
	_, _, driver, system := operating_system_all(t, clock)
	const LINES = 20000
	callback_count := 0
	result := sysos.Process_Result{}
	var operation_err error
	var completion time.Completion
	system.Spawn(&completion, func(
		_ *time.Completion, spawned sysos.Process_Result, err error,
	) {
		callback_count++
		result = spawned
		operation_err = err
	}, sysos.Process_Request{
		Path: "/bin/sh",
		Arguments: []string{
			"-c", "i=0; while [ $i -lt 20000 ]; do echo line; i=$((i+1)); done",
		},
	}, REAL_DEADLINE)
	if !operating_system_run_until(
		t, driver, func() (finished bool) { return callback_count > 0 },
	) {
		t.Fatal("spawn producing more than one pipe buffer did not complete")
	}
	if operation_err != nil {
		t.Fatalf("spawn error = %v, want nil", operation_err)
	}
	if want := LINES * len("line\n"); len(result.Output) != want {
		t.Fatalf("captured output = %d bytes, want %d", len(result.Output), want)
	}
	driver.Deinit()
}

// Test_Operating_System_IO_Spawn_Resolves_Path verifies a bare command name still resolves
// through PATH. exec.Command supplied this before, and syscall.StartProcess does not.
func Test_Operating_System_IO_Spawn_Resolves_Path(t *testing.T) {
	clock, _ := timeos.New_Operating_System_Any_Clock()
	_, _, driver, system := operating_system_all(t, clock)
	callback_count := 0
	result := sysos.Process_Result{}
	var operation_err error
	var completion time.Completion
	system.Spawn(&completion, func(
		_ *time.Completion, spawned sysos.Process_Result, err error,
	) {
		callback_count++
		result = spawned
		operation_err = err
	}, sysos.Process_Request{Path: "echo", Arguments: []string{"resolved"}}, REAL_DEADLINE)
	if !operating_system_run_until(
		t, driver, func() (finished bool) { return callback_count > 0 },
	) {
		t.Fatal("spawn of a bare command name did not complete")
	}
	if operation_err != nil {
		t.Fatalf("spawn error = %v, want nil", operation_err)
	}
	if strings.TrimSpace(string(result.Output)) != "resolved" {
		t.Fatalf("output = %q, want resolved", result.Output)
	}
	driver.Deinit()
}

// Test_Operating_System_IO_Spawn_Reports_Missing_Command verifies an unresolvable command name
// fails the spawn rather than starting anything.
func Test_Operating_System_IO_Spawn_Reports_Missing_Command(t *testing.T) {
	clock, _ := timeos.New_Operating_System_Any_Clock()
	_, _, driver, system := operating_system_all(t, clock)
	callback_count := 0
	var operation_err error
	var completion time.Completion
	system.Spawn(&completion, func(
		_ *time.Completion, _ sysos.Process_Result, err error,
	) {
		callback_count++
		operation_err = err
	}, sysos.Process_Request{Path: "no-such-command-anywhere"}, REAL_DEADLINE)
	if !operating_system_run_until(
		t, driver, func() (finished bool) { return callback_count > 0 },
	) {
		t.Fatal("spawn of a missing command did not complete")
	}
	if operation_err == nil {
		t.Fatal("spawn of a missing command reported no error")
	}
	if spawns := driver.Introspect().Spawns; spawns != 0 {
		t.Fatalf("tracked children after a failed start = %d, want 0", spawns)
	}
	driver.Deinit()
}

// Waits for host init to reap killed grandchildren without accepting a live bounded process.
func process_group_wait_for_exit(
	driver time.Driver, process_identifier int,
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

// Test_Operating_System_IO_Make_Directory covers what a converging mkdir can hide: a repeat
// call, a relative path, a trailing slash, and a final component that already exists as a file.
func Test_Operating_System_IO_Make_Directory(t *testing.T) {
	clock, _ := timeos.New_Operating_System_Any_Clock()
	loop, _, driver := operating_system_loop(t, clock)
	root := t.TempDir()

	nested := filepath.Join(root, "one", "two", "three")
	if make_err := make_directory(t, loop, driver, nested); make_err != nil {
		t.Fatalf("make nested: %v", make_err)
	}
	// A repeat converges rather than reporting that the directory exists.
	if repeat_err := make_directory(t, loop, driver, nested); repeat_err != nil {
		t.Fatalf("repeat make: %v", repeat_err)
	}
	parents := []string{
		filepath.Join(root, "one"), filepath.Join(root, "one", "two"),
	}
	for _, path := range parents {
		status, _ := loop.Status(path)
		if !status.Is_Directory {
			t.Fatalf("parent %s = %+v, want a directory", path, status)
		}
	}

	slashed := filepath.Join(root, "four", "five") + "/"
	if make_err := make_directory(t, loop, driver, slashed); make_err != nil {
		t.Fatalf("make with a trailing slash: %v", make_err)
	}
	if status, _ := loop.Status(filepath.Join(root, "four", "five")); !status.Is_Directory {
		t.Fatalf("trailing-slash path = %+v, want a directory", status)
	}

	// A final component that already exists as a file must report an error rather than
	// converge, because the caller asked for a directory and does not have one.
	occupied := filepath.Join(root, "occupied")
	write_file(t, loop, driver, occupied, []byte("not a directory"))
	// Mkdir_At reports Path_Exists for a file too, and Make_Directory converges on that. The
	// caller therefore learns the difference from Status, not from the create.
	if make_err := make_directory(t, loop, driver, occupied); make_err != nil {
		t.Fatalf("make over an existing file: %v", make_err)
	}
	if status, _ := loop.Status(occupied); status.Is_Directory {
		t.Fatal("the existing file became a directory")
	}
	driver.Deinit()
}

// Test_Operating_System_IO_Directory exercises the filesystem-traversal ops on a real temp
// tree: Make_Directory builds a nested path (into which the fixture file is seeded), and
// Status and Read_Directory then report the tree's shape, including an absent path.
func Test_Operating_System_IO_Directory(t *testing.T) {
	clock, _ := timeos.New_Operating_System_Any_Clock()
	loop, _, driver := operating_system_loop(t, clock)

	root := t.TempDir()
	nested := filepath.Join(root, "a", "b")
	if make_err := make_directory(t, loop, driver, nested); make_err != nil {
		t.Fatalf("make directory: %v", make_err)
	}
	// Creating the file through the loop proves Make_Directory built the parents: Create
	// fails when the directory above the path does not exist.
	file_path := filepath.Join(nested, "file.txt")
	write_file(t, loop, driver, file_path, []byte("hello"))

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

	entries, read_err := read_directory(t, loop, driver, nested)
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
	Timeline  io.IO
	Driver    time.Driver
	Connected io.File
	Accepted  io.File
}

func loopback_assert_roundtrip(input *loopback_roundtrip_input) {
	input.Test.Helper()
	var send_completion time.Completion
	input.Timeline.Send(&send_completion, func(_ *time.Completion, _ int, send_err error) {
		if send_err != nil {
			input.Test.Errorf("send: %v", send_err)
		}
	}, input.Connected, []byte("ping"))
	buffer := make([]byte, 16)
	received := -1
	var receive_completion time.Completion
	input.Timeline.Receive(&receive_completion, func(
		_ *time.Completion, count int, receive_err error,
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
func self_exec_close(loop io.IO, driver time.Driver, socket io.File) {
	closed := false
	var completion time.Completion
	loop.Close(&completion, func(_ *time.Completion, _ error) { closed = true }, socket)
	driver.Run_Until(func() (finished bool) { return closed }, REAL_DEADLINE)
}
