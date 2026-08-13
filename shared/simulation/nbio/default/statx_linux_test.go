//go:build linux && amd64

package nbio_test

import (
	"path/filepath"
	"testing"

	io "local/james-orcales/shared/simulation/nbio"
	"local/james-orcales/shared/simulation/time"
	timeos "local/james-orcales/shared/simulation/time/default"
)

// Test_Operating_System_IO_Statx ports the Linux-only tail of TigerBeetle's
// open/write/read/close/statx test and verifies the written size through IORING_OP_STATX.
func Test_Operating_System_IO_Statx(t *testing.T) {
	path := filepath.Join(t.TempDir(), "statx")
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
	self_exec_close(loop, driver, opened)

	status := io.Statx{}
	statted := false
	var statx_completion time.Completion
	loop.Statx(
		&statx_completion,
		func(_ *time.Completion, err error) {
			if err != nil {
				t.Errorf("statx: %v", err)
			}
			statted = true
		},
		io.DIRECTORY_CURRENT, path, 0, io.STATX_BASIC_STATS, &status,
	)
	if !operating_system_run_until(t, driver, func() (finished bool) { return statted }) {
		t.Fatal("statx did not complete")
	}
	if status.Size != 15 {
		t.Fatalf("statx size = %d, want 15", status.Size)
	}
}
