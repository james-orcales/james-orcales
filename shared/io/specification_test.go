package io_test

import (
	"testing"

	"github.com/james-orcales/james-orcales/shared/io"
	"github.com/james-orcales/james-orcales/shared/time"
)

// Test_Sim_Timeout verifies a timeout fires exactly when the virtual clock reaches
// its deadline — no real waiting, fully deterministic.
func Test_Sim_Timeout(t *testing.T) {
	virtual := time.Virtual_Clock_To_Clock(time.Virtual_Clock{Resolution: time.Nanosecond})
	sim := &io.Sim{Clock: virtual, Latency: func() (duration time.Duration) { return 0 }}
	loop := io.Sim_To_IO(sim)

	fired_at := time.Moment(-1)
	var completion io.Completion
	loop.Timeout(&completion, func(_ *io.Completion, err error) {
		if err != nil {
			t.Fatalf("timeout error: %v", err)
		}
		fired_at = virtual.Now_Monotonic()
	}, 5*time.Nanosecond)

	loop.Run_For(10 * time.Nanosecond)

	if fired_at != 5 {
		t.Fatalf("timeout fired at %d, want 5", fired_at)
	}
}

// Test_Sim_Read verifies a read completes after the modeled latency and reports the
// buffer length.
func Test_Sim_Read(t *testing.T) {
	virtual := time.Virtual_Clock_To_Clock(time.Virtual_Clock{Resolution: time.Nanosecond})
	sim := &io.Sim{Clock: virtual, Latency: func() (duration time.Duration) {
		return 3 * time.Nanosecond
	}}
	loop := io.Sim_To_IO(sim)

	count := -1
	var completion io.Completion
	loop.Read(&completion, func(_ *io.Completion, bytes int, err error) {
		count = bytes
	}, io.File(0), make([]byte, 64), 0)

	loop.Run_For(10 * time.Nanosecond)

	if count != 64 {
		t.Fatalf("read reported %d bytes, want 64", count)
	}
}

// Test_Sim_Listen verifies Listen returns a fresh descriptor synchronously.
func Test_Sim_Listen(t *testing.T) {
	virtual := time.Virtual_Clock_To_Clock(time.Virtual_Clock{Resolution: time.Nanosecond})
	sim := &io.Sim{Clock: virtual, Latency: func() (duration time.Duration) { return 0 }}
	loop := io.Sim_To_IO(sim)

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
	virtual := time.Virtual_Clock_To_Clock(time.Virtual_Clock{Resolution: time.Nanosecond})
	sim := &io.Sim{Clock: virtual, Latency: func() (duration time.Duration) {
		return 2 * time.Nanosecond
	}}
	loop := io.Sim_To_IO(sim)

	listener, _ := loop.Listen("127.0.0.1", 0)
	accepted := io.File(-1)
	var completion io.Completion
	loop.Accept(&completion, func(_ *io.Completion, socket io.File, err error) {
		accepted = socket
	}, listener)

	loop.Run_For(10 * time.Nanosecond)

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
	virtual := time.Virtual_Clock_To_Clock(time.Virtual_Clock{Resolution: time.Nanosecond})
	sim := &io.Sim{Clock: virtual, Latency: func() (duration time.Duration) {
		return 4 * time.Nanosecond
	}}
	loop := io.Sim_To_IO(sim)

	connected := io.File(-1)
	var completion io.Completion
	loop.Connect(&completion, func(_ *io.Completion, socket io.File, err error) {
		connected = socket
	}, "127.0.0.1", 8123)

	loop.Run_For(10 * time.Nanosecond)

	if connected <= 0 {
		t.Fatalf("connect yielded %d, want a positive descriptor", connected)
	}
}

// Test_Sim_Receive verifies a receive completes after the modeled latency and
// reports the buffer length.
func Test_Sim_Receive(t *testing.T) {
	virtual := time.Virtual_Clock_To_Clock(time.Virtual_Clock{Resolution: time.Nanosecond})
	sim := &io.Sim{Clock: virtual, Latency: func() (duration time.Duration) {
		return 3 * time.Nanosecond
	}}
	loop := io.Sim_To_IO(sim)

	count := -1
	var completion io.Completion
	loop.Receive(&completion, func(_ *io.Completion, bytes int, err error) {
		count = bytes
	}, io.File(1), make([]byte, 64))

	loop.Run_For(10 * time.Nanosecond)

	if count != 64 {
		t.Fatalf("receive reported %d bytes, want 64", count)
	}
}

// Test_Sim_Send verifies a send completes after the modeled latency and reports
// the buffer length.
func Test_Sim_Send(t *testing.T) {
	virtual := time.Virtual_Clock_To_Clock(time.Virtual_Clock{Resolution: time.Nanosecond})
	sim := &io.Sim{Clock: virtual, Latency: func() (duration time.Duration) {
		return 3 * time.Nanosecond
	}}
	loop := io.Sim_To_IO(sim)

	count := -1
	var completion io.Completion
	loop.Send(&completion, func(_ *io.Completion, bytes int, err error) {
		count = bytes
	}, io.File(1), make([]byte, 32))

	loop.Run_For(10 * time.Nanosecond)

	if count != 32 {
		t.Fatalf("send reported %d bytes, want 32", count)
	}
}

// Test_Sim_Close verifies a close completes after the modeled latency and reports
// no error.
func Test_Sim_Close(t *testing.T) {
	virtual := time.Virtual_Clock_To_Clock(time.Virtual_Clock{Resolution: time.Nanosecond})
	sim := &io.Sim{Clock: virtual, Latency: func() (duration time.Duration) {
		return time.Nanosecond
	}}
	loop := io.Sim_To_IO(sim)

	closed := false
	failed := error(nil)
	var completion io.Completion
	loop.Close(&completion, func(_ *io.Completion, err error) {
		closed = true
		failed = err
	}, io.File(1))

	loop.Run_For(10 * time.Nanosecond)

	if !closed {
		t.Fatal("close did not complete")
	}
	if failed != nil {
		t.Fatalf("close error: %v", failed)
	}
}
