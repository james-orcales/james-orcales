package io_test

import (
	"fmt"
	"strings"
	"testing"
	"unsafe"

	"local/james-orcales/shared/io"
	snap "local/james-orcales/shared/snap/default"
	"local/james-orcales/shared/time"
)

// Test_Sim_Timeout verifies a timeout fires exactly when the virtual clock reaches
// its deadline — no real waiting, fully deterministic.
func Test_Sim_Timeout(t *testing.T) {
	loop, driver, clock := sim_loop(0)

	fired_at := time.Moment(-1)
	var completion io.Completion
	loop.Timeout(&completion, func(_ *io.Completion, err error) {
		if err != nil {
			t.Fatalf("timeout error: %v", err)
		}
		fired_at = clock.Now_Monotonic()
	}, 5*time.NANOSECOND)

	driver.Run_For(10 * time.NANOSECOND)

	if fired_at != 5 {
		t.Fatalf("timeout fired at %d, want 5", fired_at)
	}
}

// Test_Sim_Next_Tick verifies next-tick callbacks use the completed queue and reset removes every
// queued callback for a source without firing it.
func Test_Sim_Next_Tick(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	fired := 0
	var first io.Completion
	var second io.Completion
	loop.Next_Tick(&first, func(_ *io.Completion) { fired++ }, io.NEXT_TICK_VSR)
	loop.Next_Tick(&second, func(_ *io.Completion) { fired++ }, io.NEXT_TICK_VSR)
	loop.Reset_Next_Tick(io.NEXT_TICK_VSR)
	driver.Run()
	if fired != 0 {
		t.Fatalf("reset next tick fired %d callbacks, want 0", fired)
	}
	loop.Next_Tick(&first, func(_ *io.Completion) { fired++ }, io.NEXT_TICK_VSR)
	driver.Run()
	if fired != 1 {
		t.Fatalf("next tick fired %d callbacks, want 1", fired)
	}
}

// Test_Sim_Read verifies a read on an opened file returns the bytes an earlier write
// stored — the file descriptor's real-bytes path, distinct from a socket's byte count.
func Test_Sim_Read(t *testing.T) {
	loop, driver, _ := sim_loop(1)

	writer, create_err := loop.Create("file")
	if create_err != nil {
		t.Fatalf("create: %v", create_err)
	}
	wrote := false
	var write_completion io.Completion
	loop.Write(&write_completion, func(_ *io.Completion, _ int, err error) {
		if err != nil {
			t.Fatalf("write: %v", err)
		}
		wrote = true
	}, writer, []byte("hello"), 0)
	driver.Run_Until(func() (finished bool) { return wrote }, SIM_DEADLINE)

	reader, open_err := loop.Open("file")
	if open_err != nil {
		t.Fatalf("open: %v", open_err)
	}
	buffer := make([]byte, 64)
	count := -1
	var read_completion io.Completion
	loop.Read(&read_completion, func(_ *io.Completion, bytes int, _ error) {
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
	file, create_err := loop.Create("file")
	if create_err != nil {
		t.Fatalf("create: %v", create_err)
	}
	fired := false
	var completion io.Completion
	loop.Fsync(&completion, func(_ *io.Completion, err error) {
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
	var completion io.Completion
	loop.Open_At(&completion, func(_ *io.Completion, file io.File, err error) {
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
	var unknown_completion io.Completion
	unknown_loop.Open_At(
		&unknown_completion,
		func(_ *io.Completion, _ io.File, _ error) {},
		io.DIRECTORY_CURRENT,
		"file",
		io.Open_At_Options{Flags: io.Open_At_Flags(1 << 31)},
	)
}

// Test_Sim_Event verifies the TigerBeetle Event primitive retires its listener before invoking the
// callback, allowing the same completion to be re-armed for a later trigger.
func Test_Sim_Event(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	event, open_err := loop.Open_Event()
	if open_err != nil {
		t.Fatalf("open event: %v", open_err)
	}
	fired := 0
	var completion io.Completion
	callback := func(_ *io.Completion) { fired++ }
	loop.Event_Listen(event, &completion, callback)
	loop.Event_Trigger(event, &completion)
	driver.Run()
	if fired != 1 {
		t.Fatalf("event fired %d times, want 1", fired)
	}
	loop.Event_Listen(event, &completion, callback)
	loop.Event_Trigger(event, &completion)
	driver.Run()
	if fired != 2 {
		t.Fatalf("event fired %d times, want 2", fired)
	}
	loop.Close_Event(event)
}

// Test_Sim_Listen verifies Listen returns a fresh descriptor synchronously.
func Test_Sim_Listen(t *testing.T) {
	loop, _, _ := sim_loop(0)

	address := io.Address_I_Pv4([io.IPV4_ADDRESS_BYTES]byte{127, 0, 0, 1}, 0)
	listener, open_err := loop.Open_Socket_TCP(io.FAMILY_IPV4, io.TCP_Options{})
	if open_err != nil {
		t.Fatalf("open listener: %v", open_err)
	}
	resolved, err := loop.Listen(listener, address, io.Listen_Options{Backlog: 128})
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
		listener, _ := loop.Open_Socket_TCP(io.FAMILY_IPV4, io.TCP_Options{})
		_, listen_err := loop.Listen(
			listener, io.Address_I_Pv4([io.IPV4_ADDRESS_BYTES]byte{127, 0, 0, 1}, 0),
			io.Listen_Options{Backlog: 128},
		)
		if listen_err != nil {
			t.Fatalf("seed %d: listen: %v", seed, listen_err)
		}
		callback_count := 0
		accepted := io.File(-1)
		var operation_err error
		var completion io.Completion
		loop.Accept(&completion, func(_ *io.Completion, socket io.File, err error) {
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
		if operation_err == io.Deadline_Exceeded {
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
	socket, open_err := loop.Open_Socket_TCP(io.FAMILY_IPV4, io.TCP_Options{})
	if open_err != nil {
		t.Fatalf("open socket: %v", open_err)
	}
	opened := driver.Introspect().Raw_Open
	closed := false
	var completion io.Completion
	loop.Close(&completion, func(_ *io.Completion, err error) {
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
	socket, open_err := loop.Open_Socket_TCP(io.FAMILY_IPV4, io.TCP_Options{})
	if open_err != nil {
		t.Fatalf("open socket: %v", open_err)
	}

	count := -1
	var completion io.Completion
	loop.Receive(&completion, func(_ *io.Completion, bytes int, err error) {
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
	socket, open_err := loop.Open_Socket_TCP(io.FAMILY_IPV4, io.TCP_Options{})
	if open_err != nil {
		t.Fatalf("open socket: %v", open_err)
	}

	count := -1
	var completion io.Completion
	loop.Send(&completion, func(_ *io.Completion, bytes int, err error) {
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
	socket, open_err := loop.Open_Socket_UDP(io.FAMILY_IPV4)
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
	socket, open_err := loop.Open_Socket_TCP(io.FAMILY_IPV4, io.TCP_Options{})
	if open_err != nil {
		t.Fatalf("open socket: %v", open_err)
	}
	connected := false
	var connect_completion io.Completion
	loop.Connect(&connect_completion, func(_ *io.Completion, err error) {
		if err != nil {
			t.Fatalf("connect: %v", err)
		}
		connected = true
	}, socket, io.Address_I_Pv4([io.IPV4_ADDRESS_BYTES]byte{127, 0, 0, 1}, 8123), SIM_DEADLINE)
	driver.Run_Until(func() (finished bool) { return connected }, SIM_DEADLINE)
	var receive_completion io.Completion
	var send_completion io.Completion
	receive_count := -1
	var send_err error
	loop.Receive(&receive_completion, func(_ *io.Completion, count int, err error) {
		if err != nil {
			t.Fatalf("receive after shutdown: %v", err)
		}
		receive_count = count
	}, socket, make([]byte, 8))
	loop.Send(&send_completion, func(_ *io.Completion, _ int, err error) {
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

	socket, open_err := loop.Open_Socket_TCP(io.FAMILY_IPV4, io.TCP_Options{})
	if open_err != nil {
		t.Fatalf("open socket: %v", open_err)
	}
	connected := false
	var connect_completion io.Completion
	loop.Connect(&connect_completion, func(_ *io.Completion, err error) {
		if err != nil {
			t.Fatalf("connect: %v", err)
		}
		connected = true
	}, socket, io.Address_I_Pv4([io.IPV4_ADDRESS_BYTES]byte{127, 0, 0, 1}, 1), SIM_DEADLINE)
	driver.Run_Until(func() (finished bool) { return connected }, SIM_DEADLINE)
	received := false
	var receive_completion io.Completion
	loop.Receive(&receive_completion, func(_ *io.Completion, _ int, _ error) {
		received = true
	}, socket, make([]byte, 8))

	close_panicked := false
	var completion io.Completion
	func() {
		defer func() { close_panicked = recover() != nil }()
		loop.Close(&completion, func(_ *io.Completion, _ error) {}, socket)
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
	var close_completion io.Completion
	loop.Close(&close_completion, func(_ *io.Completion, err error) {
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
	socket, open_err := loop.Open_Socket_TCP(io.FAMILY_IPV4, io.TCP_Options{})
	if open_err != nil {
		t.Fatalf("open socket: %v", open_err)
	}
	loop.Close_Socket(socket)
	if driver.Introspect().Raw_Open != 0 {
		t.Fatal("close socket left the descriptor open")
	}
}

// Test_Sim_Run_Until verifies the driver pumps the loop until the predicate reports true,
// reporting completed, and — the point of the timeout — that a never-satisfied predicate
// returns false at the deadline instead of spinning the sim forever.
func Test_Sim_Run_Until(t *testing.T) {
	loop, driver, _ := sim_loop(1)

	done := false
	var completion io.Completion
	loop.Read(&completion, func(_ *io.Completion, bytes int, err error) {
		done = true
	}, io.File(0), make([]byte, 8), 0)

	completed, drive_err := driver.Run_Until(
		func() (finished bool) { return done }, SIM_DEADLINE,
	)
	if drive_err != nil {
		t.Fatalf("Run_Until error: %v", drive_err)
	}
	if !completed {
		t.Fatal("Run_Until reported the read did not complete")
	}
	if !done {
		t.Fatal("Run_Until returned before the read completed")
	}

	completed, drive_err = driver.Run_Until(
		func() (finished bool) { return false }, SIM_DEADLINE,
	)
	if drive_err != nil {
		t.Fatalf("Run_Until deadline error: %v", drive_err)
	}
	if completed {
		t.Fatal("Run_Until reported completion for a predicate that never trips")
	}
}

// Test_Sim_Reuse verifies submitting a completion that is still in flight panics, so a
// reused completion fails loudly instead of corrupting the queue.
func Test_Sim_Reuse(t *testing.T) {
	loop, _, _ := sim_loop(0)
	var completion io.Completion
	loop.Timeout(&completion, func(_ *io.Completion, err error) {}, 5*time.NANOSECOND)
	defer func() {
		if recover() == nil {
			t.Fatal("reusing an in-flight completion must panic")
		}
	}()
	loop.Timeout(&completion, func(_ *io.Completion, err error) {}, 5*time.NANOSECOND)
}

// Test_Sim_Copy verifies submitting a by-value copy of a completion panics, so a copied
// completion fails loudly instead of splitting the loop's view from the caller's. It fires
// the original first so the copy is unarmed — isolating the copy guard from the reuse one.
func Test_Sim_Copy(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	var completion io.Completion
	loop.Timeout(&completion, func(_ *io.Completion, err error) {}, 5*time.NANOSECOND)
	driver.Run_For(10 * time.NANOSECOND)
	duplicate := completion
	defer func() {
		if recover() == nil {
			t.Fatal("submitting a copied completion must panic")
		}
	}()
	loop.Timeout(&duplicate, func(_ *io.Completion, err error) {}, 5*time.NANOSECOND)
}

// Test_Sim_Reentrancy verifies driving the loop from within a completion callback panics,
// so a re-entrant Run* fails loudly instead of corrupting the queue mid-drain.
func Test_Sim_Reentrancy(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	var completion io.Completion
	loop.Timeout(&completion, func(_ *io.Completion, err error) {
		driver.Run()
	}, 5*time.NANOSECOND)
	defer func() {
		if recover() == nil {
			t.Fatal("driving from within a callback must panic")
		}
	}()
	driver.Run_For(10 * time.NANOSECOND)
}

// Test_Sim_Open verifies Open returns a fresh descriptor synchronously.
func Test_Sim_Open(t *testing.T) {
	loop, _, _ := sim_loop(0)
	if _, create_err := loop.Create("path"); create_err != nil {
		t.Fatalf("create: %v", create_err)
	}
	file, err := loop.Open("path")
	if err != nil {
		t.Fatalf("open error: %v", err)
	}
	if file <= 0 {
		t.Fatalf("open yielded %d, want a positive descriptor", file)
	}
	if _, absent_err := loop.Open("absent"); absent_err == nil {
		t.Fatal("open of an absent path should error")
	}
}

// Test_Sim_Create verifies Create returns a fresh writable descriptor synchronously.
func Test_Sim_Create(t *testing.T) {
	loop, _, _ := sim_loop(0)
	file, err := loop.Create("path")
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
	socket, _ := loop.Open_Socket_TCP(io.FAMILY_IPV4, io.TCP_Options{})
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

// Test_Sim_Watch_Signal verifies a signal watch resolves exactly once with either its signal or
// Deadline_Exceeded.
func Test_Sim_Watch_Signal(t *testing.T) {
	signal_count := 0
	deadline_count := 0
	for seed := uint64(0); seed < 64; seed++ {
		loop, driver, _ := sim_loop(seed)
		got := io.Signal(-1)
		callback_count := 0
		var operation_err error
		var completion io.Completion
		loop.Watch_Signal(&completion, func(
			_ *io.Completion, signal io.Signal, err error,
		) {
			callback_count++
			got = signal
			operation_err = err
		}, io.SIGNAL_TERMINATE, time.NANOSECOND)
		completed, drive_err := driver.Run_Until(
			func() (finished bool) { return callback_count > 0 }, 16*time.NANOSECOND)
		if drive_err != nil {
			t.Fatalf("seed %d: drive: %v", seed, drive_err)
		}
		if !completed {
			t.Fatalf("seed %d: signal watch did not resolve", seed)
		}
		if callback_count != 1 {
			t.Fatalf("seed %d: callback count = %d, want 1", seed, callback_count)
		}
		if operation_err == io.Deadline_Exceeded {
			deadline_count++
			if got != -1 {
				t.Fatalf("seed %d: deadline yielded signal %d", seed, got)
			}
		} else {
			if operation_err != nil {
				t.Fatalf("seed %d: signal watch: %v", seed, operation_err)
			}
			signal_count++
			if got != io.SIGNAL_TERMINATE {
				t.Fatalf("seed %d: signal = %d, want terminate", seed, got)
			}
		}
		driver.Run_For(16 * time.NANOSECOND)
		if callback_count != 1 {
			t.Fatalf("seed %d: callback repeated %d times", seed, callback_count)
		}
	}
	if signal_count == 0 {
		t.Fatal("seed sweep witnessed no signal before the deadline")
	}
	if deadline_count == 0 {
		t.Fatal("seed sweep witnessed no signal deadline")
	}
}

// Test_Sim_Spawn verifies a spawn delivers a result on the loop.
func Test_Sim_Spawn(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	fired := 0
	var completion io.Completion
	loop.Spawn(&completion, func(_ *io.Completion, result io.Process_Result, err error) {
		fired++
	}, io.Process_Request{Path: "echo"}, SIM_DEADLINE)
	driver.Run_For(16 * time.NANOSECOND)
	if fired != 1 {
		t.Fatalf("spawn callback fired %d times, want 1", fired)
	}
}

// Test_Sim_Self_Exec verifies the simulator refuses to replace its own process and reports
// the failure synchronously, so a caller's fallback path runs in every simulated run.
func Test_Sim_Self_Exec(t *testing.T) {
	loop, _, _ := sim_loop(0)
	err := loop.Self_Exec("/proc/self/exe", []string{"/proc/self/exe"}, []string{})
	if err == nil {
		t.Fatal("self-exec returned nil error; the simulator must always fail it")
	}
}

// Test_Sim_Read_Directory verifies Read_Directory lists a directory's immediate children,
// each named with whether it is itself a directory.
func Test_Sim_Read_Directory(t *testing.T) {
	loop, _, _ := sim_loop(0)
	if make_err := loop.Make_Directory("/a/b"); make_err != nil {
		t.Fatalf("make directory: %v", make_err)
	}
	if _, create_err := loop.Create("/a/b/file"); create_err != nil {
		t.Fatalf("create: %v", create_err)
	}
	entries, err := loop.Read_Directory("/a/b")
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
	parents, _ := loop.Read_Directory("/a")
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
	if make_err := loop.Make_Directory("/dir"); make_err != nil {
		t.Fatalf("make directory: %v", make_err)
	}
	file, create_err := loop.Create("/dir/file")
	if create_err != nil {
		t.Fatalf("create: %v", create_err)
	}
	// Write known bytes so the file's Size has a known expected value.
	content := []byte("hello world")
	written := false
	var write io.Completion
	loop.Write(&write, func(_ *io.Completion, _ int, _ error) {
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
	loop, _, _ := sim_loop(0)
	if make_err := loop.Make_Directory("/x/y/z"); make_err != nil {
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
	if repeat_err := loop.Make_Directory("/x/y/z"); repeat_err != nil {
		t.Fatalf("repeated make directory should converge, got %v", repeat_err)
	}
}

// Test_Sim_Introspect verifies every simulator operation class, lifecycle flag, and raw-open
// descriptor count is reported without exposing the simulator itself.
func Test_Sim_Introspect(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	file, create_err := loop.Create("/introspect")
	if create_err != nil {
		t.Fatalf("create: %v", create_err)
	}
	socket, open_err := loop.Open_Socket_TCP(io.FAMILY_IPV4, io.TCP_Options{})
	if open_err != nil {
		t.Fatalf("open socket: %v", open_err)
	}
	var completed, timeout, read, write, signal, posted io.Completion
	loop.Write(&completed, func(_ *io.Completion, _ int, _ error) {}, file, nil, 0)
	loop.Timeout(&timeout, func(_ *io.Completion, _ error) {}, time.MICROSECOND)
	loop.Receive(&read, func(_ *io.Completion, _ int, _ error) {}, socket, nil)
	loop.Send(&write, func(_ *io.Completion, _ int, _ error) {}, socket, nil)
	loop.Watch_Signal(&signal, func(
		_ *io.Completion, _ io.Signal, _ error,
	) {
	}, io.SIGNAL_TERMINATE, time.MICROSECOND)
	loop.Spawn(&posted, func(
		_ *io.Completion, _ io.Process_Result, _ error,
	) {
	}, io.Process_Request{Path: "true"}, SIM_DEADLINE)
	snap.Expect(t, snap.Init(`{Completed:1 Timeouts:1 IO_Backlog:2 IO_Inflight:0 IO_Queued:0 IO_In_Kernel:0 Signal_Waiters:1 Spawns:1 Raw_Open:2}`),
		fmt.Sprintf("%+v", driver.Introspect()))
}

// Test_Stream_Read verifies Read moves bytes from the cursor, advances the cursor, and
// reports Stream_EOF once the cursor has reached the end of the memory.
func Test_Stream_Read(t *testing.T) {
	stream := memory_stream([]byte("abcdef"))
	first := make([]byte, 3)
	count, err := io.Read(stream, first)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if count != 3 {
		t.Fatalf("read %d bytes, want 3", count)
	}
	if string(first) != "abc" {
		t.Fatalf("read %q, want abc", first)
	}
	second := make([]byte, 8)
	count, err = io.Read(stream, second)
	if err != nil {
		t.Fatalf("second read error: %v", err)
	}
	if count != 3 {
		t.Fatalf("second read %d bytes, want 3", count)
	}
	_, err = io.Read(stream, second)
	if err != io.Stream_EOF {
		t.Fatalf("exhausted read reported %v, want Stream_EOF", err)
	}
}

// Test_Stream_Write verifies Write stores bytes at the cursor, advances the cursor, and
// reports Stream_Short_Write when the remaining memory cannot hold the whole buffer.
func Test_Stream_Write(t *testing.T) {
	memory := make([]byte, 4)
	stream := memory_stream(memory)
	count, err := io.Write(stream, []byte("ab"))
	if err != nil {
		t.Fatalf("write error: %v", err)
	}
	if count != 2 {
		t.Fatalf("wrote %d bytes, want 2", count)
	}
	count, err = io.Write(stream, []byte("cdef"))
	if err != io.Stream_Short_Write {
		t.Fatalf("overflowing write reported %v, want Stream_Short_Write", err)
	}
	if count != 2 {
		t.Fatalf("overflowing write stored %d bytes, want 2", count)
	}
	if string(memory) != "abcd" {
		t.Fatalf("memory holds %q, want abcd", memory)
	}
}

// Test_Stream_Read_At verifies Read_At reads from an explicit offset and leaves the cursor
// where it was, the distinction Odin draws between Read and Read_At.
func Test_Stream_Read_At(t *testing.T) {
	stream := memory_stream([]byte("abcdef"))
	head := make([]byte, 2)
	_, read_err := io.Read(stream, head)
	if read_err != nil {
		t.Fatalf("read error: %v", read_err)
	}
	tail := make([]byte, 2)
	count, err := io.Read_At(stream, tail, 4)
	if err != nil {
		t.Fatalf("read at error: %v", err)
	}
	if count != 2 {
		t.Fatalf("read at moved %d bytes, want 2", count)
	}
	if string(tail) != "ef" {
		t.Fatalf("read at yielded %q, want ef", tail)
	}
	position, seek_err := io.Seek(stream, 0, io.SEEK_FROM_CURRENT)
	if seek_err != nil {
		t.Fatalf("seek error: %v", seek_err)
	}
	if position != 2 {
		t.Fatalf("read at moved the cursor to %d, want 2", position)
	}
}

// Test_Stream_Write_At verifies Write_At stores at an explicit offset and leaves the cursor
// where it was.
func Test_Stream_Write_At(t *testing.T) {
	memory := make([]byte, 6)
	stream := memory_stream(memory)
	count, err := io.Write_At(stream, []byte("xy"), 4)
	if err != nil {
		t.Fatalf("write at error: %v", err)
	}
	if count != 2 {
		t.Fatalf("write at stored %d bytes, want 2", count)
	}
	if memory[4] != 'x' {
		t.Fatalf("write at stored %q at 4, want x", memory[4])
	}
	position, seek_err := io.Seek(stream, 0, io.SEEK_FROM_CURRENT)
	if seek_err != nil {
		t.Fatalf("seek error: %v", seek_err)
	}
	if position != 0 {
		t.Fatalf("write at moved the cursor to %d, want 0", position)
	}
}

// Test_Stream_Seek verifies each Seek_From origin, and that an origin outside the three
// reports Stream_Invalid_Whence.
func Test_Stream_Seek(t *testing.T) {
	stream := memory_stream([]byte("abcdef"))
	position, err := io.Seek(stream, 2, io.SEEK_FROM_START)
	if err != nil {
		t.Fatalf("seek from start error: %v", err)
	}
	if position != 2 {
		t.Fatalf("seek from start reached %d, want 2", position)
	}
	position, err = io.Seek(stream, 1, io.SEEK_FROM_CURRENT)
	if err != nil {
		t.Fatalf("seek from current error: %v", err)
	}
	if position != 3 {
		t.Fatalf("seek from current reached %d, want 3", position)
	}
	position, err = io.Seek(stream, -1, io.SEEK_FROM_END)
	if err != nil {
		t.Fatalf("seek from end error: %v", err)
	}
	if position != 5 {
		t.Fatalf("seek from end reached %d, want 5", position)
	}
	_, err = io.Seek(stream, 0, io.Seek_From(9))
	if err != io.Stream_Invalid_Whence {
		t.Fatalf("unknown whence reported %v, want Stream_Invalid_Whence", err)
	}
	_, err = io.Seek(stream, -1, io.SEEK_FROM_START)
	if err != io.Stream_Invalid_Offset {
		t.Fatalf("negative position reported %v, want Stream_Invalid_Offset", err)
	}
}

// Test_Stream_Size verifies Size reports the whole memory, not the bytes remaining.
func Test_Stream_Size(t *testing.T) {
	stream := memory_stream([]byte("abcdef"))
	_, seek_err := io.Seek(stream, 4, io.SEEK_FROM_START)
	if seek_err != nil {
		t.Fatalf("seek error: %v", seek_err)
	}
	size, err := io.Size(stream)
	if err != nil {
		t.Fatalf("size error: %v", err)
	}
	if size != 6 {
		t.Fatalf("size reported %d, want 6", size)
	}
}

// Test_Stream_Query verifies Query names exactly the modes a stream answers, so a caller
// learns what a stream cannot do without provoking a failure.
func Test_Stream_Query(t *testing.T) {
	memory := io.Query(memory_stream(make([]byte, 4)))
	if !io.Mode_Set_Has(memory, io.STREAM_MODE_SEEK) {
		t.Fatal("a memory stream answers Seek")
	}
	if !io.Mode_Set_Has(memory, io.STREAM_MODE_SIZE) {
		t.Fatal("a memory stream answers Size")
	}
	discard := io.Query(discard_stream())
	if io.Mode_Set_Has(discard, io.STREAM_MODE_SEEK) {
		t.Fatal("a discard stream cannot seek")
	}
	if !io.Mode_Set_Has(discard, io.STREAM_MODE_WRITE) {
		t.Fatal("a discard stream answers Write")
	}
}

// Test_Stream_Flush verifies Flush succeeds on memory, which has nothing to flush, so a
// caller can flush any stream without asking what is behind it.
func Test_Stream_Flush(t *testing.T) {
	if err := io.Flush(memory_stream(make([]byte, 2))); err != nil {
		t.Fatalf("flush error: %v", err)
	}
	if err := io.Flush(discard_stream()); err != nil {
		t.Fatalf("discard flush error: %v", err)
	}
}

// Test_Stream_Close verifies Close is idempotent and that a closed stream answers no data
// mode.
func Test_Stream_Close(t *testing.T) {
	stream := memory_stream(make([]byte, 4))
	if err := io.Close(stream); err != nil {
		t.Fatalf("close error: %v", err)
	}
	if err := io.Close(stream); err != nil {
		t.Fatalf("second close error: %v", err)
	}
	_, err := io.Write(stream, []byte("a"))
	if err != io.Stream_Empty {
		t.Fatalf("write after close reported %v, want Stream_Empty", err)
	}
}

// Test_Stream_Destroy verifies Destroy closes the stream. Odin separates the two because a
// stream there can own an allocation; a memory stream owns nothing but its cursor.
func Test_Stream_Destroy(t *testing.T) {
	stream := memory_stream(make([]byte, 4))
	if err := io.Destroy(stream); err != nil {
		t.Fatalf("destroy error: %v", err)
	}
	_, err := io.Read(stream, make([]byte, 1))
	if err != io.Stream_Empty {
		t.Fatalf("read after destroy reported %v, want Stream_Empty", err)
	}
}

// Test_Stream_Errors verifies the dispatch checks Odin's write helper performs: a zero
// Stream reports Stream_Empty, and a mode a stream does not answer reports Stream_Empty.
func Test_Stream_Errors(t *testing.T) {
	var zero io.Stream
	_, err := io.Read(zero, make([]byte, 1))
	if err != io.Stream_Empty {
		t.Fatalf("zero stream read reported %v, want Stream_Empty", err)
	}
	if io.Query(zero) != 0 {
		t.Fatal("a zero stream answers no mode")
	}
	_, err = io.Seek(discard_stream(), 0, io.SEEK_FROM_START)
	if err != io.Stream_Empty {
		t.Fatalf("unsupported seek reported %v, want Stream_Empty", err)
	}
}

// Test_Stream_Memory verifies the memory stream never grows its slice: it is bounded by the
// slice it was built over, which is what makes it safe to hand to an unbounded encoder.
func Test_Stream_Memory(t *testing.T) {
	memory := make([]byte, 3)
	state := io.Stream_Memory{Memory: memory}
	stream := io.Memory_To_Stream(&state)
	count, err := io.Write(stream, []byte("abcdefgh"))
	if err != io.Stream_Short_Write {
		t.Fatalf("write past the end reported %v, want Stream_Short_Write", err)
	}
	if count != 3 {
		t.Fatalf("write past the end stored %d bytes, want 3", count)
	}
	if len(memory) != 3 {
		t.Fatalf("the memory grew to %d bytes", len(memory))
	}
	if string(memory) != "abc" {
		t.Fatalf("memory holds %q, want abc", memory)
	}
}

// Test_Stream_Discard verifies a discard stream absorbs every write, reports the whole buffer
// stored, and answers no read mode.
func Test_Stream_Discard(t *testing.T) {
	stream := discard_stream()
	count, err := io.Write(stream, []byte("abcdef"))
	if err != nil {
		t.Fatalf("discard write error: %v", err)
	}
	if count != 6 {
		t.Fatalf("discard reported %d bytes, want 6", count)
	}
	_, err = io.Read(stream, make([]byte, 4))
	if err != io.Stream_Empty {
		t.Fatalf("discard read reported %v, want Stream_Empty", err)
	}
}

// Test_Stream_Limit verifies a limit truncates at its budget and stops the bytes reaching the
// stream behind it, so the budget is a fact about the transport and not a caller convention.
func Test_Stream_Limit(t *testing.T) {
	memory := make([]byte, 8)
	inner := memory_stream(memory)
	state := io.Stream_Limit{Inner: inner, Budget: 3}
	stream := io.Limit_To_Stream(&state)
	count, err := io.Write(stream, []byte("abcde"))
	if err != io.Stream_Short_Write {
		t.Fatalf("write past the budget reported %v, want Stream_Short_Write", err)
	}
	if count != 3 {
		t.Fatalf("write past the budget passed %d bytes, want 3", count)
	}
	if string(memory[:4]) != "abc\x00" {
		t.Fatalf("the memory behind the limit holds %q, want abc and a zero", memory[:4])
	}
	_, err = io.Write(stream, []byte("f"))
	if err != io.Stream_Short_Write {
		t.Fatalf("write on a spent budget reported %v, want Stream_Short_Write", err)
	}
	if io.Mode_Set_Has(io.Query(stream), io.STREAM_MODE_SEEK) {
		t.Fatal("a limited stream cannot seek: a budget and a cursor disagree")
	}
}

// Test_Stream_Count verifies a count tallies every byte and changes nothing else, so the same
// encoder measures and stores without being told which it is doing.
func Test_Stream_Count(t *testing.T) {
	memory := make([]byte, 8)
	state := io.Stream_Count{Inner: memory_stream(memory)}
	stream := io.Count_To_Stream(&state)
	if _, err := io.Write(stream, []byte("ab")); err != nil {
		t.Fatalf("write error: %v", err)
	}
	if _, err := io.Write(stream, []byte("cde")); err != nil {
		t.Fatalf("second write error: %v", err)
	}
	if state.Tally != 5 {
		t.Fatalf("the tally is %d, want 5", state.Tally)
	}
	if string(memory[:5]) != "abcde" {
		t.Fatalf("the memory behind the count holds %q, want abcde", memory[:5])
	}
	measure := io.Stream_Count{Inner: discard_stream()}
	if _, err := io.Write(io.Count_To_Stream(&measure), []byte("abcd")); err != nil {
		t.Fatalf("measuring write error: %v", err)
	}
	if measure.Tally != 4 {
		t.Fatalf("the measuring tally is %d, want 4", measure.Tally)
	}
}

// Test_Stream_Tee verifies a tee writes each buffer to both streams and reports the smaller
// count, so a caller learns about the tighter of the two rather than the first.
func Test_Stream_Tee(t *testing.T) {
	wide := make([]byte, 8)
	narrow := make([]byte, 2)
	state := io.Stream_Tee{First: memory_stream(wide), Second: memory_stream(narrow)}
	stream := io.Tee_To_Stream(&state)
	count, err := io.Write(stream, []byte("abcd"))
	if err != io.Stream_Short_Write {
		t.Fatalf("tee onto a narrow stream reported %v, want Stream_Short_Write", err)
	}
	if count != 2 {
		t.Fatalf("the tee reported %d bytes, want the narrower 2", count)
	}
	if string(wide[:4]) != "abcd" {
		t.Fatalf("the wide stream holds %q, want abcd", wide[:4])
	}
	if string(narrow) != "ab" {
		t.Fatalf("the narrow stream holds %q, want ab", narrow)
	}
	if io.Mode_Set_Has(io.Query(stream), io.STREAM_MODE_READ) {
		t.Fatal("a tee cannot read: two streams give two answers")
	}
}

// Test_Stream_Composition verifies the transforms compose. One encoder writes through a tee,
// over a count, over a limit, over memory, and the tally, the truncation, and the stored bytes
// all agree — the property that makes the abstraction worth its indirection.
func Test_Stream_Composition(t *testing.T) {
	stored := make([]byte, 16)
	limit := io.Stream_Limit{Inner: memory_stream(stored), Budget: 6}
	count := io.Stream_Count{Inner: io.Limit_To_Stream(&limit)}
	audit := make([]byte, 16)
	tee := io.Stream_Tee{First: io.Count_To_Stream(&count), Second: memory_stream(audit)}
	stream := io.Tee_To_Stream(&tee)

	written, err := io.Write(stream, []byte("abcdefghij"))
	if err != io.Stream_Short_Write {
		t.Fatalf("composed write reported %v, want Stream_Short_Write", err)
	}
	if written != 6 {
		t.Fatalf("composed write reported %d bytes, want the limit's 6", written)
	}
	if count.Tally != 6 {
		t.Fatalf("the tally is %d, want 6", count.Tally)
	}
	if string(stored[:8]) != "abcdef\x00\x00" {
		t.Fatalf("the stored bytes are %q, want abcdef and two zeros", stored[:8])
	}
	if string(audit[:10]) != "abcdefghij" {
		t.Fatalf("the audit copy holds %q, want abcdefghij", audit[:10])
	}
}

// Test_Stream_Derived verifies the derived helpers Odin builds on top of the modes: the whole
// of core/io/util.odin, written here as the plain loops Odin writes, because a synchronous
// stream needs no continuation to read twice.
func Test_Stream_Derived(t *testing.T) {
	full := make([]byte, 6)
	count, err := io.Read_Full(memory_stream([]byte("abcdef")), full)
	if err != nil {
		t.Fatalf("read full error: %v", err)
	}
	if count != 6 {
		t.Fatalf("read full moved %d bytes, want 6", count)
	}
	short := make([]byte, 8)
	count, err = io.Read_Full(memory_stream([]byte("abcdef")), short)
	if err != io.Stream_Unexpected_EOF {
		t.Fatalf("read full past the end reported %v, want Stream_Unexpected_EOF", err)
	}
	if count != 6 {
		t.Fatalf("read full past the end moved %d bytes, want 6", count)
	}
	count, err = io.Read_At_Least(memory_stream([]byte("abcdef")), short, 3)
	if err != nil {
		t.Fatalf("read at least error: %v", err)
	}
	if count != 6 {
		t.Fatalf("read at least moved %d bytes, want 6", count)
	}

	memory := make([]byte, 8)
	stream := memory_stream(memory)
	if count, err = io.Write_String(stream, "ab"); err != nil {
		t.Fatalf("write string error: %v", err)
	}
	if count != 2 {
		t.Fatalf("write string stored %d bytes, want 2", count)
	}
	if err = io.Write_Byte(stream, 'c'); err != nil {
		t.Fatalf("write byte error: %v", err)
	}
	size, rune_err := io.Write_Rune(stream, 'é')
	if rune_err != nil {
		t.Fatalf("write rune error: %v", rune_err)
	}
	if size != 2 {
		t.Fatalf("write rune stored %d bytes, want 2", size)
	}
	if string(memory[:5]) != "abcé" {
		t.Fatalf("the memory holds %q, want abcé", memory[:5])
	}

	reader := memory_stream([]byte("abcé"))
	value, byte_err := io.Read_Byte(reader)
	if byte_err != nil {
		t.Fatalf("read byte error: %v", byte_err)
	}
	if value != 'a' {
		t.Fatalf("read byte yielded %q, want a", value)
	}
	if _, err = io.Seek(reader, 3, io.SEEK_FROM_START); err != nil {
		t.Fatalf("seek error: %v", err)
	}
	character, character_size, character_err := io.Read_Rune(reader)
	if character_err != nil {
		t.Fatalf("read rune error: %v", character_err)
	}
	if character != 'é' {
		t.Fatalf("read rune yielded %q, want é", character)
	}
	if character_size != 2 {
		t.Fatalf("read rune consumed %d bytes, want 2", character_size)
	}
}

// Test_Stream_Pointer verifies the pointer forms move raw memory, Odin's read_ptr and
// write_ptr. The caller states the size, so a negative one is rejected rather than trusted.
func Test_Stream_Pointer(t *testing.T) {
	source := [POINTER_TEST_SIZE]byte{'a', 'b', 'c', 'd'}
	memory := make([]byte, 8)
	stream := memory_stream(memory)
	count, err := io.Write_Pointer(stream, unsafe.Pointer(&source[0]), 4)
	if err != nil {
		t.Fatalf("write pointer error: %v", err)
	}
	if count != 4 {
		t.Fatalf("write pointer moved %d bytes, want 4", count)
	}
	if string(memory[:4]) != "abcd" {
		t.Fatalf("the memory holds %q, want abcd", memory[:4])
	}
	var target [POINTER_TEST_SIZE]byte
	reader := memory_stream([]byte("wxyz"))
	count, err = io.Read_Pointer(reader, unsafe.Pointer(&target[0]), 4)
	if err != nil {
		t.Fatalf("read pointer error: %v", err)
	}
	if count != 4 {
		t.Fatalf("read pointer moved %d bytes, want 4", count)
	}
	if string(target[:]) != "wxyz" {
		t.Fatalf("the target holds %q, want wxyz", target[:])
	}
	_, err = io.Read_Pointer(reader, unsafe.Pointer(&target[0]), -1)
	if err != io.Stream_Negative_Count {
		t.Fatalf("a negative size reported %v, want Stream_Negative_Count", err)
	}
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
	socket, open_err := loop.Open_Socket_TCP(io.FAMILY_IPV4, io.TCP_Options{})
	if open_err != nil {
		t.Fatalf("open socket: %v", open_err)
	}
	opened := driver.Introspect().Raw_Open
	called := false
	var connect_err error
	var completion io.Completion
	loop.Connect(&completion, func(_ *io.Completion, err error) {
		called = true
		connect_err = err
	}, socket, io.Address_I_Pv4([io.IPV4_ADDRESS_BYTES]byte{127, 0, 0, 1}, 8123), SIM_DEADLINE)
	driver.Run_Until(func() (finished bool) { return called }, SIM_DEADLINE)
	if !called {
		t.Fatal("connect callback did not fire")
	}
	connected := driver.Introspect().Raw_Open
	closed := false
	var close_completion io.Completion
	loop.Close(&close_completion, func(_ *io.Completion, err error) {
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
	socket, open_err := loop.Open_Socket_TCP(io.FAMILY_IPV4, io.TCP_Options{})
	if open_err != nil {
		t.Fatalf("open socket: %v", open_err)
	}
	callback_count := 0
	var completion io.Completion
	loop.Connect(&completion, func(_ *io.Completion, err error) {
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
	var close_completion io.Completion
	loop.Close(&close_completion, func(_ *io.Completion, err error) {
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

// Runs one bounded simulated spawn past its modeled completion time and reports its sole result.
func sim_spawn_with_deadline(
	t *testing.T, seed uint64, deadline time.Duration,
) (spawn_err error) {
	t.Helper()
	loop, driver, _ := sim_loop(seed)
	callback_count := 0
	var completion io.Completion
	loop.Spawn(&completion, func(
		_ *io.Completion, _ io.Process_Result, err error,
	) {
		callback_count++
		spawn_err = err
	}, io.Process_Request{Path: "true"}, deadline)
	driver.Run_Until(func() (finished bool) { return callback_count > 0 }, SIM_DEADLINE)
	driver.Run_For(16 * time.NANOSECOND)
	if callback_count != 1 {
		t.Fatalf("seed %d: spawn callback count = %d, want 1", seed, callback_count)
	}
	return spawn_err
}

// Builds a simulated loop, its driver, and the read-only clock, seeded by seed. A test
// holds only the IO, the driver, and the clock — never the sim, which New_Sim keeps to
// itself so the run stays a pure function of the seed.
func sim_loop(seed uint64) (loop io.IO, driver io.Driver, clock time.Clock) {
	return io.New_Sim(seed)
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
		if at_deadline == io.Deadline_Exceeded {
			saw_deadline = true
		}
		if at_deadline == io.Deadline_Exceeded {
			if after_deadline != io.Deadline_Exceeded {
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

// Test_Spawn_Deadline_Sim verifies a finite spawn deadline wins a latency tie and retires once.
func Test_Spawn_Deadline_Sim(t *testing.T) {
	saw_deadline := false
	saw_tie := false
	for seed := uint64(0); seed < 64; seed++ {
		at_deadline := sim_spawn_with_deadline(t, seed, 4*time.NANOSECOND)
		after_deadline := sim_spawn_with_deadline(t, seed, 5*time.NANOSECOND)
		if at_deadline == io.Deadline_Exceeded {
			saw_deadline = true
		}
		if at_deadline == io.Deadline_Exceeded {
			if after_deadline != io.Deadline_Exceeded {
				saw_tie = true
			}
		}
	}
	if !saw_deadline {
		t.Fatal("seed sweep witnessed no spawn deadline")
	}
	if !saw_tie {
		t.Fatal("seed sweep witnessed no spawn latency tie lost to the deadline")
	}
}

// Builds a memory stream over bytes the test owns. Test_Stream_Memory writes the constructor
// out in full; every other test uses this, because the state is not what it is testing.
func memory_stream(memory []byte) (stream io.Stream) {
	state := io.Stream_Memory{Memory: memory}
	return io.Memory_To_Stream(&state)
}

// Builds a discard stream for the tests that need a stream answering only write modes.
func discard_stream() (stream io.Stream) {
	state := io.Stream_Discard{}
	return io.Discard_To_Stream(&state)
}

// POINTER_TEST_SIZE is how many bytes the pointer forms move in Test_Stream_Pointer.
const POINTER_TEST_SIZE = 4
