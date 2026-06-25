package io_test

import (
	"testing"

	"github.com/james-orcales/james-orcales/shared/io"
	"github.com/james-orcales/james-orcales/shared/time"
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
	}, 5*time.Nanosecond)

	driver.Run_For(10 * time.Nanosecond)

	if fired_at != 5 {
		t.Fatalf("timeout fired at %d, want 5", fired_at)
	}
}

// Test_Sim_Read verifies a read completes after the modeled latency and reports the
// buffer length.
func Test_Sim_Read(t *testing.T) {
	loop, driver, _ := sim_loop(1)

	count := -1
	var completion io.Completion
	loop.Read(&completion, func(_ *io.Completion, bytes int, err error) {
		count = bytes
	}, io.File(0), make([]byte, 64), 0)

	driver.Run_For(10 * time.Nanosecond)

	if count != 64 {
		t.Fatalf("read reported %d bytes, want 64", count)
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

	driver.Run_For(10 * time.Nanosecond)

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

	driver.Run_For(10 * time.Nanosecond)

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

	driver.Run_For(10 * time.Nanosecond)

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

	driver.Run_For(10 * time.Nanosecond)

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

	driver.Run_For(10 * time.Nanosecond)

	if !closed {
		t.Fatal("close did not complete")
	}
	if failed != nil {
		t.Fatalf("close error: %v", failed)
	}
}

// Test_Sim_Run_Until verifies the driver pumps the loop until the predicate reports
// true, delivering the op's completion inline for a straight-line caller.
func Test_Sim_Run_Until(t *testing.T) {
	loop, driver, _ := sim_loop(1)

	done := false
	var completion io.Completion
	loop.Read(&completion, func(_ *io.Completion, bytes int, err error) {
		done = true
	}, io.File(0), make([]byte, 8), 0)

	driver.Run_Until(func() (finished bool) { return done })

	if !done {
		t.Fatal("Run_Until returned before the read completed")
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
	}, 5*time.Nanosecond)

	loop.Cancel(&completion)
	driver.Run_For(10 * time.Nanosecond)

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
	loop.Timeout(&completion, func(_ *io.Completion, err error) {}, 5*time.Nanosecond)
	defer func() {
		if recover() == nil {
			t.Fatal("reusing an in-flight completion must panic")
		}
	}()
	loop.Timeout(&completion, func(_ *io.Completion, err error) {}, 5*time.Nanosecond)
}

// Test_Sim_Open verifies Open returns a fresh descriptor synchronously.
func Test_Sim_Open(t *testing.T) {
	loop, _, _ := sim_loop(0)
	file, err := loop.Open("path")
	if err != nil {
		t.Fatalf("open error: %v", err)
	}
	if file <= 0 {
		t.Fatalf("open yielded %d, want a positive descriptor", file)
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
	driver.Run_For(16 * time.Nanosecond)
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
	driver.Run_For(16 * time.Nanosecond)
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
	driver.Run_For(16 * time.Nanosecond)
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
	}, io.Signal_Terminate)
	driver.Run_For(16 * time.Nanosecond)
	if fired != 1 {
		t.Fatalf("signal callback fired %d times, want 1", fired)
	}
	if got != io.Signal_Terminate {
		t.Fatalf("signal = %d, want Signal_Terminate", got)
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
	driver.Run_For(16 * time.Nanosecond)
	if !ran {
		t.Fatal("compute work did not run")
	}
	if !fired {
		t.Fatal("compute callback did not fire")
	}
}

// Builds a simulated loop, its driver, and the read-only clock, seeded by seed. A test
// holds only the IO, the driver, and the clock — never the sim, which New_Sim keeps to
// itself so the run stays a pure function of the seed.
func sim_loop(seed uint64) (loop io.IO, driver io.Driver, clock time.Clock) {
	return io.New_Sim(seed)
}
