package io_test

import (
	"fmt"
	"strings"
	"testing"

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

// Test_Sim_Compute verifies offloaded work runs and its callback fires on the loop.
func Test_Sim_Compute(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	ran := false
	fired := false
	var completion io.Completion
	loop.Compute(&completion, func(_ *io.Completion) {
		fired = true
	}, func() { ran = true })
	driver.Run_For(16 * time.NANOSECOND)
	if !ran {
		t.Fatal("compute work did not run")
	}
	if !fired {
		t.Fatal("compute callback did not fire")
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
	var completed, timeout, read, write, signal, posted, result io.Completion
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
	loop.Compute(&result, func(_ *io.Completion) {}, func() {})
	snap.Expect(t, snap.Init(`{Completed:1 Timeouts:1 IO_Backlog:2 IO_Inflight:0 IO_Queued:0 IO_In_Kernel:0 Signal_Waiters:1 Posted:1 Results:1 Raw_Open:2 Wake_Active:true Compute_Active:true}`),
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
