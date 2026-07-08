package io_test

import (
	"testing"

	"local/james-orcales/shared/io"
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

// Test_Sim_Listen verifies Listen returns a fresh descriptor synchronously.
func Test_Sim_Listen(t *testing.T) {
	loop, _, _ := sim_loop(0)

	listener, err := loop.Listen("127.0.0.1", 0)
	if err != nil {
		t.Fatalf("listen error: %v", err)
	}
	if listener == 0 {
		t.Fatal("listen returned the zero descriptor")
	}
}

// Test_Sim_Accept verifies an accept completes after the modeled latency and
// yields a descriptor distinct from its listener.
func Test_Sim_Accept(t *testing.T) {
	loop, driver, _ := sim_loop(2)

	listener, _ := loop.Listen("127.0.0.1", 0)
	accepted := io.File(-1)
	var completion io.Completion
	loop.Accept(&completion, func(_ *io.Completion, socket io.File, err error) {
		accepted = socket
	}, listener)

	driver.Run_For(10 * time.NANOSECOND)

	if accepted == listener {
		t.Fatalf("accept yielded the listener descriptor %d", accepted)
	}
	if accepted <= 0 {
		t.Fatalf("accept yielded %d, want a positive descriptor", accepted)
	}
}

// Test_Sim_Connect verifies a connect completes after the modeled latency and
// yields a fresh connected descriptor.
func Test_Sim_Connect(t *testing.T) {
	loop, driver, _ := sim_loop(3)

	connected := io.File(-1)
	var completion io.Completion
	loop.Connect(&completion, func(_ *io.Completion, socket io.File, err error) {
		connected = socket
	}, "127.0.0.1", 8123)

	driver.Run_For(10 * time.NANOSECOND)

	if connected <= 0 {
		t.Fatalf("connect yielded %d, want a positive descriptor", connected)
	}
}

// Test_Sim_Receive verifies a receive completes after the modeled latency and
// reports the buffer length.
func Test_Sim_Receive(t *testing.T) {
	loop, driver, _ := sim_loop(1)

	count := -1
	var completion io.Completion
	loop.Receive(&completion, func(_ *io.Completion, bytes int, err error) {
		count = bytes
	}, io.File(1), make([]byte, 64))

	driver.Run_For(10 * time.NANOSECOND)

	if count != 64 {
		t.Fatalf("receive reported %d bytes, want 64", count)
	}
}

// Test_Sim_Send verifies a send completes after the modeled latency and reports
// the buffer length.
func Test_Sim_Send(t *testing.T) {
	loop, driver, _ := sim_loop(1)

	count := -1
	var completion io.Completion
	loop.Send(&completion, func(_ *io.Completion, bytes int, err error) {
		count = bytes
	}, io.File(1), make([]byte, 32))

	driver.Run_For(10 * time.NANOSECOND)

	if count != 32 {
		t.Fatalf("send reported %d bytes, want 32", count)
	}
}

// Test_Sim_Close verifies a close completes after the modeled latency and reports
// no error.
func Test_Sim_Close(t *testing.T) {
	loop, driver, _ := sim_loop(4)

	closed := false
	failed := error(nil)
	var completion io.Completion
	loop.Close(&completion, func(_ *io.Completion, err error) {
		closed = true
		failed = err
	}, io.File(1))

	driver.Run_For(10 * time.NANOSECOND)

	if !closed {
		t.Fatal("close did not complete")
	}
	if failed != nil {
		t.Fatalf("close error: %v", failed)
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

	if !driver.Run_Until(func() (finished bool) { return done }, SIM_DEADLINE) {
		t.Fatal("Run_Until reported the read did not complete")
	}
	if !done {
		t.Fatal("Run_Until returned before the read completed")
	}

	if driver.Run_Until(func() (finished bool) { return false }, SIM_DEADLINE) {
		t.Fatal("Run_Until reported completion for a predicate that never trips")
	}
}

// Test_Sim_Cancel verifies cancelling an in-flight op still fires its callback exactly
// once, with the Cancelled error, rather than dropping it.
func Test_Sim_Cancel(t *testing.T) {
	loop, driver, _ := sim_loop(0)

	got := error(nil)
	fired := 0
	var completion io.Completion
	loop.Timeout(&completion, func(_ *io.Completion, err error) {
		fired++
		got = err
	}, 5*time.NANOSECOND)

	loop.Cancel(&completion)
	driver.Run_For(10 * time.NANOSECOND)

	if fired != 1 {
		t.Fatalf("callback fired %d times, want exactly 1", fired)
	}
	if got != io.Cancelled {
		t.Fatalf("cancel error = %v, want io.Cancelled", got)
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
	listener, _ := loop.Listen("127.0.0.1", 0)
	address, err := loop.Peer_Address(listener)
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

// Test_Sim_Accept_Secure verifies a secure accept yields a distinct descriptor.
func Test_Sim_Accept_Secure(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	listener, _ := loop.Listen("127.0.0.1", 0)
	accepted := io.File(-1)
	var completion io.Completion
	loop.Accept_Secure(&completion, func(_ *io.Completion, socket io.File, err error) {
		accepted = socket
	}, listener, func() (value any) { return nil })
	driver.Run_For(16 * time.NANOSECOND)
	if accepted <= 0 {
		t.Fatalf("secure accept yielded %d, want a positive descriptor", accepted)
	}
}

// Test_Sim_Connect_Secure verifies a secure connect yields a connected descriptor.
func Test_Sim_Connect_Secure(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	connected := io.File(-1)
	var completion io.Completion
	loop.Connect_Secure(&completion, func(_ *io.Completion, socket io.File, err error) {
		connected = socket
	}, "127.0.0.1", 443, "host")
	driver.Run_For(16 * time.NANOSECOND)
	if connected <= 0 {
		t.Fatalf("secure connect yielded %d, want a positive descriptor", connected)
	}
}

// Test_Sim_Connect_Insecure verifies an insecure connect yields a connected descriptor.
func Test_Sim_Connect_Insecure(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	connected := io.File(-1)
	var completion io.Completion
	loop.Connect_Insecure(&completion, func(_ *io.Completion, socket io.File, err error) {
		connected = socket
	}, "127.0.0.1", 443, "host")
	driver.Run_For(16 * time.NANOSECOND)
	if connected <= 0 {
		t.Fatalf("insecure connect yielded %d, want a positive descriptor", connected)
	}
}

// Test_Sim_Watch_Signal verifies a watched signal fires its callback once with that
// signal.
func Test_Sim_Watch_Signal(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	got := io.Signal(-1)
	fired := 0
	var completion io.Completion
	loop.Watch_Signal(&completion, func(_ *io.Completion, signal io.Signal) {
		fired++
		got = signal
	}, io.SIGNAL_TERMINATE)
	driver.Run_For(16 * time.NANOSECOND)
	if fired != 1 {
		t.Fatalf("signal callback fired %d times, want 1", fired)
	}
	if got != io.SIGNAL_TERMINATE {
		t.Fatalf("signal = %d, want SIGNAL_TERMINATE", got)
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
	}, io.Process_Request{Path: "echo"})
	driver.Run_For(16 * time.NANOSECOND)
	if fired != 1 {
		t.Fatalf("spawn callback fired %d times, want 1", fired)
	}
}

// Test_Sim_Self_Exec verifies the simulator refuses to replace its own process and reports
// the failure synchronously, so a caller's fallback path runs in every simulated run.
func Test_Sim_Self_Exec(t *testing.T) {
	loop, _, _ := sim_loop(0)
	err := loop.Self_Exec("/proc/self/exe", []string{"/proc/self/exe"}, nil, nil)
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
	if directory.Size != 0 {
		t.Fatalf("dir status = %+v, want a zero Size", directory)
	}
	regular, _ := loop.Status("/dir/file")
	if regular.Is_Directory {
		t.Fatalf("file status = %+v, want a non-directory", regular)
	}
	if regular.Size != int64(len(content)) {
		t.Fatalf("file status = %+v, want Size %d", regular, len(content))
	}
	absent, _ := loop.Status("/nope")
	if absent.Exists {
		t.Fatalf("absent status = %+v, want not exists", absent)
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

// Test_Sim_Cancel_Window_Reuse verifies resubmitting a completion inside the cancel
// window — after Cancel accepted, before the cancelled delivery fired — panics: the
// pending delivery still owns the completion, so re-arming it is an edge the lifecycle
// machine does not have.
func Test_Sim_Cancel_Window_Reuse(t *testing.T) {
	loop, _, _ := sim_loop(0)
	var completion io.Completion
	loop.Timeout(&completion, func(_ *io.Completion, err error) {}, time.MICROSECOND)
	loop.Cancel(&completion)
	defer func() {
		if recover() == nil {
			t.Fatal("resubmitting inside the cancel window must panic")
		}
	}()
	loop.Timeout(&completion, func(_ *io.Completion, err error) {}, time.MICROSECOND)
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
