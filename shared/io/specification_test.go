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
