package io_test

import (
	"fmt"
	"strings"
	"testing"

	io "local/james-orcales/shared/simulation/nbio"
	"local/james-orcales/shared/simulation/time"
	snap "local/james-orcales/shared/snap/default"
)

// Test_Sim_Read verifies a read on an opened file returns the bytes an earlier write
// stored — the file descriptor's real-bytes path, distinct from a socket's byte count.
func Test_Sim_Read(t *testing.T) {
	loop, driver, _ := sim_loop(1)

	writer, create_err := sim_create(t, loop, driver, "file")
	if create_err != nil {
		t.Fatalf("create: %v", create_err)
	}
	wrote := false
	var write_completion time.Completion
	loop.Write(&write_completion, func(_ *time.Completion, _ int, err error) {
		if err != nil {
			t.Fatalf("write: %v", err)
		}
		wrote = true
	}, writer, []byte("hello"), 0)
	driver.Run_Until(func() (finished bool) { return wrote }, SIM_DEADLINE)

	reader, open_err := sim_open(t, loop, driver, "file")
	if open_err != nil {
		t.Fatalf("open: %v", open_err)
	}
	buffer := make([]byte, 64)
	count := -1
	var read_completion time.Completion
	loop.Read(&read_completion, func(_ *time.Completion, bytes int, _ error) {
		count = bytes
	}, reader, buffer, 0)
	driver.Run_Until(func() (finished bool) { return count >= 0 }, SIM_DEADLINE)

	if count != 5 {
		t.Fatalf("read reported %d bytes, want 5", count)
	}
	if string(buffer[:count]) != "hello" {
		t.Fatalf("read %q, want hello", buffer[:count])
	}
}

// Test_Sim_Fsync verifies the TigerBeetle fsync operation completes without changing descriptor
// ownership after a prior write has retired.
func Test_Sim_Fsync(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	file, create_err := sim_create(t, loop, driver, "file")
	if create_err != nil {
		t.Fatalf("create: %v", create_err)
	}
	fired := false
	var completion time.Completion
	loop.Fsync(&completion, func(_ *time.Completion, err error) {
		if err != nil {
			t.Fatalf("fsync: %v", err)
		}
		fired = true
	}, file)
	driver.Run_Until(func() (finished bool) { return fired }, SIM_DEADLINE)
	if driver.Introspect().Raw_Open != 1 {
		t.Fatal("fsync changed descriptor ownership")
	}
}

// Test_Sim_Open_At verifies the TigerBeetle asynchronous openat surface creates a caller-owned
// file which the ordinary read and write operations can use.
func Test_Sim_Open_At(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	opened := io.File(-1)
	var completion time.Completion
	loop.Open_At(&completion, func(_ *time.Completion, file io.File, err error) {
		if err != nil {
			t.Fatalf("open at: %v", err)
		}
		opened = file
	}, io.DIRECTORY_CURRENT, "file", io.Open_At_Options{
		Access: io.OPEN_READ_WRITE, Create: true, Truncate: true, Mode: 0o600,
	})
	driver.Run_Until(func() (finished bool) { return opened >= 0 }, SIM_DEADLINE)
	if opened < 0 {
		t.Fatal("open at did not return a descriptor")
	}
	if driver.Introspect().Raw_Open != 1 {
		t.Fatal("open at did not transfer caller ownership")
	}

	unknown_loop, _, _ := sim_loop(0)
	defer func() {
		if recover() == nil {
			t.Fatal("Open_At accepted an unknown flag")
		}
	}()
	var unknown_completion time.Completion
	unknown_loop.Open_At(
		&unknown_completion,
		func(_ *time.Completion, _ io.File, _ error) {},
		io.DIRECTORY_CURRENT,
		"file",
		io.Open_At_Options{Flags: io.Open_At_Flags(1 << 31)},
	)
}

// Test_Sim_Listen verifies Listen returns a fresh descriptor synchronously.
func Test_Sim_Listen(t *testing.T) {
	loop, _, _ := sim_loop(0)

	address := io.Address_I_Pv4([io.IPV4_ADDRESS_BYTES]byte{127, 0, 0, 1}, 0)
	listener, open_err := io.Open_Socket_TCP(loop, io.FAMILY_IPV4, io.TCP_Options{})
	if open_err != nil {
		t.Fatalf("open listener: %v", open_err)
	}
	resolved, err := io.Listen(loop, listener, address, io.Listen_Options{Backlog: 128})
	if err != nil {
		t.Fatalf("listen error: %v", err)
	}
	if resolved.Port == 0 {
		t.Fatal("listen did not resolve port zero")
	}
}

// Test_Sim_Accept verifies accept resolves exactly once with either a distinct accepted socket or
// Deadline_Exceeded, with a tie belonging to the finite deadline.
func Test_Sim_Accept(t *testing.T) {
	accepted_count := 0
	deadline_count := 0
	for seed := uint64(0); seed < 64; seed++ {
		loop, driver, _ := sim_loop(seed)
		listener, _ := io.Open_Socket_TCP(loop, io.FAMILY_IPV4, io.TCP_Options{})
		_, listen_err := io.Listen(loop,
			listener, io.Address_I_Pv4([io.IPV4_ADDRESS_BYTES]byte{127, 0, 0, 1}, 0),
			io.Listen_Options{Backlog: 128},
		)
		if listen_err != nil {
			t.Fatalf("seed %d: listen: %v", seed, listen_err)
		}
		callback_count := 0
		accepted := io.File(-1)
		var operation_err error
		var completion time.Completion
		loop.Accept(&completion, func(_ *time.Completion, socket io.File, err error) {
			callback_count++
			accepted = socket
			operation_err = err
		}, listener, time.NANOSECOND)
		completed, drive_err := driver.Run_Until(
			func() (finished bool) { return callback_count > 0 }, 16*time.NANOSECOND)
		if drive_err != nil {
			t.Fatalf("seed %d: drive: %v", seed, drive_err)
		}
		if !completed {
			t.Fatalf("seed %d: accept did not resolve", seed)
		}
		if callback_count != 1 {
			t.Fatalf("seed %d: callback count = %d, want 1", seed, callback_count)
		}
		if operation_err == time.Deadline_Exceeded {
			deadline_count++
			if accepted != -1 {
				t.Fatalf("seed %d: deadline yielded descriptor %d", seed, accepted)
			}
			if driver.Introspect().Raw_Open != 1 {
				t.Fatalf("seed %d: deadline leaked an accepted descriptor", seed)
			}
		} else {
			if operation_err != nil {
				t.Fatalf("seed %d: accept: %v", seed, operation_err)
			}
			accepted_count++
			if accepted <= 0 {
				t.Fatalf("seed %d: accept yielded descriptor %d", seed, accepted)
			}
			if accepted == listener {
				t.Fatalf("seed %d: accept yielded listener %d", seed, accepted)
			}
		}
		driver.Run_For(16 * time.NANOSECOND)
		if callback_count != 1 {
			t.Fatalf("seed %d: callback repeated %d times", seed, callback_count)
		}
	}
	if accepted_count == 0 {
		t.Fatal("seed sweep witnessed no accepted connection before the deadline")
	}
	if deadline_count == 0 {
		t.Fatal("seed sweep witnessed no accept deadline")
	}
}

// Test_Sim_Open_Socket verifies opening a synthetic outbound socket records caller ownership
// immediately and only an explicit Close releases it.
func Test_Sim_Open_Socket(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	baseline := driver.Introspect().Raw_Open
	socket, open_err := io.Open_Socket_TCP(loop, io.FAMILY_IPV4, io.TCP_Options{})
	if open_err != nil {
		t.Fatalf("open socket: %v", open_err)
	}
	opened := driver.Introspect().Raw_Open
	closed := false
	var completion time.Completion
	loop.Close(&completion, func(_ *time.Completion, err error) {
		if err != nil {
			t.Fatalf("close socket: %v", err)
		}
		closed = true
	}, socket)
	driver.Run_Until(func() (finished bool) { return closed }, SIM_DEADLINE)
	snap.Expect(t, snap.Init(`baseline=0 opened=1 closed=0`), fmt.Sprintf(
		"baseline=%d opened=%d closed=%d",
		baseline, opened, driver.Introspect().Raw_Open,
	))
}

// Test_Sim_Connect verifies success and refusal preserve the caller-owned socket until explicit
// Close, and a seed sweep reaches both network outcomes.
func Test_Sim_Connect(t *testing.T) {
	snap.Expect(t,
		snap.Init(`baseline=0 opened=1 connected=1 closed=0 outcome=success`),
		sim_connect_lifecycle(t, 0),
	)
	snap.Expect(t,
		snap.Init(`baseline=0 opened=1 connected=1 closed=0 outcome=refused`),
		sim_connect_lifecycle(t, 3),
	)

	saw_success := false
	saw_refusal := false
	for seed := uint64(0); seed < 64; seed++ {
		outcome := sim_connect_lifecycle(t, seed)
		if strings.HasSuffix(outcome, "outcome=success") {
			saw_success = true
		}
		if strings.HasSuffix(outcome, "outcome=refused") {
			saw_refusal = true
		}
	}
	if !saw_success {
		t.Fatal("seed sweep never reached connect success")
	}
	if !saw_refusal {
		t.Fatal("seed sweep never reached connect refusal")
	}
}

// Test_Sim_Receive verifies a receive completes after the modeled latency and
// reports the buffer length.
func Test_Sim_Receive(t *testing.T) {
	loop, driver, _ := sim_loop(1)
	socket, open_err := io.Open_Socket_TCP(loop, io.FAMILY_IPV4, io.TCP_Options{})
	if open_err != nil {
		t.Fatalf("open socket: %v", open_err)
	}

	count := -1
	var completion time.Completion
	loop.Receive(&completion, func(_ *time.Completion, bytes int, err error) {
		count = bytes
	}, socket, make([]byte, 64))

	driver.Run_For(10 * time.NANOSECOND)

	if count != 64 {
		t.Fatalf("receive reported %d bytes, want 64", count)
	}
}

// Test_Sim_Send verifies a send completes after the modeled latency and reports
// the buffer length.
func Test_Sim_Send(t *testing.T) {
	loop, driver, _ := sim_loop(1)
	socket, open_err := io.Open_Socket_TCP(loop, io.FAMILY_IPV4, io.TCP_Options{})
	if open_err != nil {
		t.Fatalf("open socket: %v", open_err)
	}

	count := -1
	var completion time.Completion
	loop.Send(&completion, func(_ *time.Completion, bytes int, err error) {
		count = bytes
	}, socket, make([]byte, 32))

	driver.Run_For(10 * time.NANOSECOND)

	if count != 32 {
		t.Fatalf("send reported %d bytes, want 32", count)
	}
}

// Test_Sim_Send_Now verifies the synchronous datagram send reports whether the bytes were accepted.
func Test_Sim_Send_Now(t *testing.T) {
	loop, _, _ := sim_loop(0)
	socket, open_err := io.Open_Socket_UDP(loop, io.FAMILY_IPV4)
	if open_err != nil {
		t.Fatalf("open UDP socket: %v", open_err)
	}
	count, sent := loop.Send_Now(socket, []byte("hello"))
	if !sent {
		t.Fatalf("send now = (%d, %t), want sent", count, sent)
	}
	if count != 5 {
		t.Fatalf("send now = (%d, %t), want (5, true)", count, sent)
	}
}

// Test_Sim_Shutdown verifies shutdown resolves armed and later socket operations through their
// normal callbacks while leaving descriptor ownership with the caller.
func Test_Sim_Shutdown(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	socket, open_err := io.Open_Socket_TCP(loop, io.FAMILY_IPV4, io.TCP_Options{})
	if open_err != nil {
		t.Fatalf("open socket: %v", open_err)
	}
	connected := false
	var connect_completion time.Completion
	loop.Connect(&connect_completion, func(_ *time.Completion, err error) {
		if err != nil {
			t.Fatalf("connect: %v", err)
		}
		connected = true
	}, socket, io.Address_I_Pv4([io.IPV4_ADDRESS_BYTES]byte{127, 0, 0, 1}, 8123), SIM_DEADLINE)
	driver.Run_Until(func() (finished bool) { return connected }, SIM_DEADLINE)
	var receive_completion time.Completion
	var send_completion time.Completion
	receive_count := -1
	var send_err error
	loop.Receive(&receive_completion, func(_ *time.Completion, count int, err error) {
		if err != nil {
			t.Fatalf("receive after shutdown: %v", err)
		}
		receive_count = count
	}, socket, make([]byte, 8))
	loop.Send(&send_completion, func(_ *time.Completion, _ int, err error) {
		send_err = err
	}, socket, []byte("hello"))
	if shutdown_err := loop.Shutdown(socket, io.SHUTDOWN_BOTH); shutdown_err != nil {
		t.Fatalf("shutdown: %v", shutdown_err)
	}
	driver.Run_For(10 * time.NANOSECOND)
	if receive_count != 0 {
		t.Fatalf("receive count = %d, want EOF", receive_count)
	}
	if send_err != io.Broken_Pipe {
		t.Fatalf("send error = %v, want %v", send_err, io.Broken_Pipe)
	}
	if driver.Introspect().Raw_Open != 1 {
		t.Fatal("shutdown released a caller-owned descriptor")
	}
}

// Test_Sim_Close verifies close rejects a descriptor borrowed by an armed operation, then a
// shutdown-drain-close sequence completes and removes the caller-owned descriptor.
func Test_Sim_Close(t *testing.T) {
	loop, driver, _ := sim_loop(0)

	socket, open_err := io.Open_Socket_TCP(loop, io.FAMILY_IPV4, io.TCP_Options{})
	if open_err != nil {
		t.Fatalf("open socket: %v", open_err)
	}
	connected := false
	var connect_completion time.Completion
	loop.Connect(&connect_completion, func(_ *time.Completion, err error) {
		if err != nil {
			t.Fatalf("connect: %v", err)
		}
		connected = true
	}, socket, io.Address_I_Pv4([io.IPV4_ADDRESS_BYTES]byte{127, 0, 0, 1}, 1), SIM_DEADLINE)
	driver.Run_Until(func() (finished bool) { return connected }, SIM_DEADLINE)
	received := false
	var receive_completion time.Completion
	loop.Receive(&receive_completion, func(_ *time.Completion, _ int, _ error) {
		received = true
	}, socket, make([]byte, 8))

	close_panicked := false
	var completion time.Completion
	func() {
		defer func() { close_panicked = recover() != nil }()
		loop.Close(&completion, func(_ *time.Completion, _ error) {}, socket)
	}()
	if !close_panicked {
		t.Fatal("close with an armed receive must panic")
	}
	if shutdown_err := loop.Shutdown(socket, io.SHUTDOWN_BOTH); shutdown_err != nil {
		t.Fatalf("shutdown: %v", shutdown_err)
	}
	driver.Run_For(10 * time.NANOSECOND)
	if !received {
		t.Fatal("shutdown did not drain the armed receive")
	}

	closed := false
	var close_completion time.Completion
	loop.Close(&close_completion, func(_ *time.Completion, err error) {
		closed = true
		if err != nil {
			t.Errorf("close: %v", err)
		}
	}, socket)

	driver.Run_For(10 * time.NANOSECOND)

	if !closed {
		t.Fatal("close did not complete")
	}
	if driver.Introspect().Raw_Open != 0 {
		t.Fatal("close left the synthetic descriptor in Raw_Open")
	}
}

// Test_Sim_Close_Socket verifies setup cleanup releases a socket synchronously.
func Test_Sim_Close_Socket(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	socket, open_err := io.Open_Socket_TCP(loop, io.FAMILY_IPV4, io.TCP_Options{})
	if open_err != nil {
		t.Fatalf("open socket: %v", open_err)
	}
	loop.Close_Socket(socket)
	if driver.Introspect().Raw_Open != 0 {
		t.Fatal("close socket left the descriptor open")
	}
}

// Test_Sim_Open verifies Open returns a fresh descriptor synchronously.
func Test_Sim_Open(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	if _, create_err := sim_create(t, loop, driver, "path"); create_err != nil {
		t.Fatalf("create: %v", create_err)
	}
	file, err := sim_open(t, loop, driver, "path")
	if err != nil {
		t.Fatalf("open error: %v", err)
	}
	if file <= 0 {
		t.Fatalf("open yielded %d, want a positive descriptor", file)
	}
	if _, absent_err := sim_open(t, loop, driver, "absent"); absent_err == nil {
		t.Fatal("open of an absent path should error")
	}
}

// Test_Sim_Create verifies Create returns a fresh writable descriptor synchronously.
func Test_Sim_Create(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	file, err := sim_create(t, loop, driver, "path")
	if err != nil {
		t.Fatalf("create error: %v", err)
	}
	if file <= 0 {
		t.Fatalf("create yielded %d, want a positive descriptor", file)
	}
}

// Test_Sim_Peer_Address verifies Peer_Address reports an address for a live descriptor
// and the empty address for an unknown one.
func Test_Sim_Peer_Address(t *testing.T) {
	loop, _, _ := sim_loop(0)
	socket, _ := io.Open_Socket_TCP(loop, io.FAMILY_IPV4, io.TCP_Options{})
	address, err := loop.Peer_Address(socket)
	if err != nil {
		t.Fatalf("peer address error: %v", err)
	}
	if address == "" {
		t.Fatal("peer address of a live descriptor must not be empty")
	}
	if unknown, _ := loop.Peer_Address(0); unknown != "" {
		t.Fatalf("peer address of an unknown descriptor = %q, want empty", unknown)
	}
}

// Test_Sim_Read_Directory verifies Read_Directory lists a directory's immediate children,
// each named with whether it is itself a directory.
func Test_Sim_Read_Directory(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	if make_err := sim_directory(t, loop, driver, "/a/b"); make_err != nil {
		t.Fatalf("make directory: %v", make_err)
	}
	if _, create_err := sim_create(t, loop, driver, "/a/b/file"); create_err != nil {
		t.Fatalf("create: %v", create_err)
	}
	entries, err := sim_read_directory(t, loop, driver, "/a/b")
	if err != nil {
		t.Fatalf("read directory: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries = %v, want exactly one", entries)
	}
	if entries[0].Name != "file" {
		t.Fatalf("entry name = %q, want file", entries[0].Name)
	}
	if entries[0].Is_Directory {
		t.Fatal("file entry reported as a directory")
	}
	parents, _ := sim_read_directory(t, loop, driver, "/a")
	if len(parents) != 1 {
		t.Fatalf("parent entries = %v, want exactly one", parents)
	}
	if !parents[0].Is_Directory {
		t.Fatal("nested entry b should be a directory")
	}
}

// Test_Sim_Status verifies Status reports a directory, a file with its byte Size, and an
// absent path.
func Test_Sim_Status(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	if make_err := sim_directory(t, loop, driver, "/dir"); make_err != nil {
		t.Fatalf("make directory: %v", make_err)
	}
	file, create_err := sim_create(t, loop, driver, "/dir/file")
	if create_err != nil {
		t.Fatalf("create: %v", create_err)
	}
	// Write known bytes so the file's Size has a known expected value.
	content := []byte("hello world")
	written := false
	var write time.Completion
	loop.Write(&write, func(_ *time.Completion, _ int, _ error) {
		written = true
	}, file, content, 0)
	driver.Run_Until(func() (finished bool) { return written }, SIM_DEADLINE)
	directory, _ := loop.Status("/dir")
	if !directory.Exists {
		t.Fatalf("dir status = %+v, want exists", directory)
	}
	if !directory.Is_Directory {
		t.Fatalf("dir status = %+v, want a directory", directory)
	}
	if directory.Is_Regular {
		t.Fatalf("dir status = %+v, want a non-regular file", directory)
	}
	if directory.Size != 0 {
		t.Fatalf("dir status = %+v, want a zero Size", directory)
	}
	regular, _ := loop.Status("/dir/file")
	if regular.Is_Directory {
		t.Fatalf("file status = %+v, want a non-directory", regular)
	}
	if !regular.Is_Regular {
		t.Fatalf("file status = %+v, want a regular file", regular)
	}
	if regular.Size != int64(len(content)) {
		t.Fatalf("file status = %+v, want Size %d", regular, len(content))
	}
	absent, _ := loop.Status("/nope")
	if absent.Exists {
		t.Fatalf("absent status = %+v, want not exists", absent)
	}
	if absent.Is_Regular {
		t.Fatalf("absent status = %+v, want a non-regular file", absent)
	}
}

// Test_Sim_Make_Directory verifies Make_Directory creates a nested path and its parents,
// and that a repeated call converges.
func Test_Sim_Make_Directory(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	if make_err := sim_directory(t, loop, driver, "/x/y/z"); make_err != nil {
		t.Fatalf("make directory: %v", make_err)
	}
	leaf, _ := loop.Status("/x/y/z")
	if !leaf.Is_Directory {
		t.Fatalf("leaf status = %+v, want a directory", leaf)
	}
	parent, _ := loop.Status("/x")
	if !parent.Is_Directory {
		t.Fatalf("parent status = %+v, want a directory created by mkdir -p", parent)
	}
	if repeat_err := sim_directory(t, loop, driver, "/x/y/z"); repeat_err != nil {
		t.Fatalf("a repeated make directory converges, got %v", repeat_err)
	}
}

// Test_Sim_Introspect verifies every simulator operation class, lifecycle flag, and raw-open
// descriptor count is reported without exposing the simulator itself.
func Test_Sim_Introspect(t *testing.T) {
	loop, pump, driver, _ := sim_loop_pump(0)
	file, create_err := sim_create(t, loop, driver, "/introspect")
	if create_err != nil {
		t.Fatalf("create: %v", create_err)
	}
	socket, open_err := io.Open_Socket_TCP(loop, io.FAMILY_IPV4, io.TCP_Options{})
	if open_err != nil {
		t.Fatalf("open socket: %v", open_err)
	}
	var completed, timeout, read, write time.Completion
	loop.Write(&completed, func(_ *time.Completion, _ int, _ error) {}, file, nil, 0)
	pump.Timeout(&timeout, func(_ *time.Completion, _ error) {}, time.MICROSECOND)
	loop.Receive(&read, func(_ *time.Completion, _ int, _ error) {}, socket, nil)
	loop.Send(&write, func(_ *time.Completion, _ int, _ error) {}, socket, nil)
	// The signal-watch and spawn classes belong to the OS surface, and shared/os states them
	// over the same census. What this test owns is the IO half of it.
	snap.Expect(t, snap.Init(`{Completed:1 Timeouts:1 IO_Backlog:2 IO_Inflight:0 IO_Queued:0 IO_In_Kernel:0 Signal_Waiters:0 Spawns:0 Raw_Open:2}`),
		fmt.Sprintf("%+v", driver.Introspect()))
}

func connect_outcome(err error) (outcome string) {
	if err == nil {
		return "success"
	}
	if err == io.Connection_Refused {
		return "refused"
	}
	return err.Error()
}

func sim_connect_lifecycle(t *testing.T, seed uint64) (snapshot string) {
	t.Helper()
	loop, driver, _ := sim_loop(seed)
	baseline := driver.Introspect().Raw_Open
	socket, open_err := io.Open_Socket_TCP(loop, io.FAMILY_IPV4, io.TCP_Options{})
	if open_err != nil {
		t.Fatalf("open socket: %v", open_err)
	}
	opened := driver.Introspect().Raw_Open
	called := false
	var connect_err error
	var completion time.Completion
	loop.Connect(&completion, func(_ *time.Completion, err error) {
		called = true
		connect_err = err
	}, socket, io.Address_I_Pv4([io.IPV4_ADDRESS_BYTES]byte{127, 0, 0, 1}, 8123), SIM_DEADLINE)
	driver.Run_Until(func() (finished bool) { return called }, SIM_DEADLINE)
	if !called {
		t.Fatal("connect callback did not fire")
	}
	connected := driver.Introspect().Raw_Open
	closed := false
	var close_completion time.Completion
	loop.Close(&close_completion, func(_ *time.Completion, err error) {
		if err != nil {
			t.Fatalf("close socket: %v", err)
		}
		closed = true
	}, socket)
	driver.Run_Until(func() (finished bool) { return closed }, SIM_DEADLINE)
	return fmt.Sprintf(
		"baseline=%d opened=%d connected=%d closed=%d outcome=%s",
		baseline, opened, connected, driver.Introspect().Raw_Open,
		connect_outcome(connect_err),
	)
}

// Runs one bounded simulated connect through its late-callback window and caller-owned close.
func sim_connect_with_deadline(
	t *testing.T, seed uint64, deadline time.Duration,
) (connect_err error) {
	t.Helper()
	loop, driver, _ := sim_loop(seed)
	socket, open_err := io.Open_Socket_TCP(loop, io.FAMILY_IPV4, io.TCP_Options{})
	if open_err != nil {
		t.Fatalf("open socket: %v", open_err)
	}
	callback_count := 0
	var completion time.Completion
	loop.Connect(&completion, func(_ *time.Completion, err error) {
		callback_count++
		connect_err = err
	}, socket, io.Address_I_Pv4([io.IPV4_ADDRESS_BYTES]byte{127, 0, 0, 1}, 8123), deadline)
	driver.Run_Until(func() (finished bool) { return callback_count > 0 }, SIM_DEADLINE)
	driver.Run_For(16 * time.NANOSECOND)
	if callback_count != 1 {
		t.Fatalf("seed %d: connect callback count = %d, want 1", seed, callback_count)
	}
	if driver.Introspect().Raw_Open != 1 {
		t.Fatalf("seed %d: connect changed caller-owned descriptor count", seed)
	}
	closed := false
	var close_completion time.Completion
	loop.Close(&close_completion, func(_ *time.Completion, err error) {
		if err != nil {
			t.Fatalf("seed %d: close socket: %v", seed, err)
		}
		closed = true
	}, socket)
	driver.Run_Until(func() (finished bool) { return closed }, SIM_DEADLINE)
	if driver.Introspect().Raw_Open != 0 {
		t.Fatalf("seed %d: caller close leaked the connect socket", seed)
	}
	return connect_err
}

// Builds a simulated loop, its driver, and the read-only clock, seeded by seed. A test
// holds only the IO, the driver, and the clock — never the sim, which New_Simulated_IO keeps to
// itself so the run stays a pure function of the seed.
func sim_loop(seed uint64) (loop io.IO, driver time.Driver, clock time.Any_Clock) {
	pump, driver, clock := time.New_Virtual_Timeline(
		time.Virtual_Clock{Resolution: time.NANOSECOND})
	return io.New_Simulated_IO(seed, pump, clock), driver, clock
}

// Opens path for reading through Open_At, driving the loop until the descriptor arrives.
func sim_open(t *testing.T, loop io.IO, driver time.Driver, path string) (file io.File, err error) {
	t.Helper()
	return sim_open_options(t, loop, driver, path, io.Open_At_Options{
		Access: io.OPEN_READ_ONLY,
	})
}

// Creates or truncates path for writing through Open_At.
func sim_create(
	t *testing.T, loop io.IO, driver time.Driver, path string,
) (file io.File, err error) {
	t.Helper()
	return sim_open_options(t, loop, driver, path, io.Open_At_Options{
		Access: io.OPEN_WRITE_ONLY, Create: true, Truncate: true, Mode: 0o644,
	})
}

// Submits one Open_At with the caller's options and drives the loop until it retires.
func sim_open_options(
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
	driver.Run_Until(func() (finished bool) { return done }, SIM_DEADLINE)
	if !done {
		t.Fatalf("the open of %s did not complete", path)
	}
	return file, err
}

// Lists path's children through the derived Read_Directory, which composes Open_At, repeated
// Get_Directory_Entries passes, and Close.
func sim_read_directory(
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
	driver.Run_Until(func() (finished bool) { return done }, SIM_DEADLINE)
	if !done {
		t.Fatalf("the listing of %s did not complete", path)
	}
	return entries, err
}

// Creates path and every missing parent through the derived Make_Directory, which composes the
// Mkdir_At primitive. The simulated backend and the operating-system backend run this same walk.
func sim_directory(
	t *testing.T, loop io.IO, driver time.Driver, path string,
) (err error) {
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
	driver.Run_Until(func() (finished bool) { return done }, SIM_DEADLINE)
	if !done {
		t.Fatalf("the create of %s did not complete", path)
	}
	return err
}

// The Run_Until cap for the sim tests, in virtual time: ample for ops that finish in a
// handful of grains, while a never-satisfied predicate fails after this many cheap grains
// instead of spinning the sim forever.
const SIM_DEADLINE = time.MICROSECOND

// Test_Connect_Deadline_Sim verifies a finite connect deadline wins a latency tie, delivers once,
// and leaves the borrowed descriptor open until the caller closes it.
func Test_Connect_Deadline_Sim(t *testing.T) {
	saw_deadline := false
	saw_tie := false
	for seed := uint64(0); seed < 64; seed++ {
		at_deadline := sim_connect_with_deadline(t, seed, 4*time.NANOSECOND)
		after_deadline := sim_connect_with_deadline(t, seed, 5*time.NANOSECOND)
		if at_deadline == time.Deadline_Exceeded {
			saw_deadline = true
		}
		if at_deadline == time.Deadline_Exceeded {
			if after_deadline != time.Deadline_Exceeded {
				saw_tie = true
			}
		}
	}
	if !saw_deadline {
		t.Fatal("seed sweep witnessed no connect deadline")
	}
	if !saw_tie {
		t.Fatal("seed sweep witnessed no connect latency tie lost to the deadline")
	}
}

// Builds a simulator beside the loop that owns its order, for a test that needs the loop's own
// control plane as well as the IO surface.
func sim_loop_pump(
	seed uint64,
) (loop io.IO, pump time.Timeline, driver time.Driver, clock time.Any_Clock) {
	pump, driver, clock = time.New_Virtual_Timeline(
		time.Virtual_Clock{Resolution: time.NANOSECOND})
	return io.New_Simulated_IO(seed, pump, clock), pump, driver, clock
}
