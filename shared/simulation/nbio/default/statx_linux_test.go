//go:build linux && amd64

package nbio_test

import (
	"path/filepath"
	"testing"

	"local/james-orcales/shared/simulation/nbio"
	"local/james-orcales/shared/simulation/time"
	timeos "local/james-orcales/shared/simulation/time/default"
	"local/james-orcales/shared/testify"
)

// Test_Operating_System_IO_Statx cover Linux-only tail of
// open/write/read/close/statx chain, and verify written size through IORING_OP_STATX.
func Test_Operating_System_IO_Statx(t *testing.T) {
	path := filepath.Join(t.TempDir(), "statx")
	clock := timeos.New_Operating_System_Clock()
	loop, _, driver := operating_system_loop(t, clock)
	opened := nbio.File(-1)
	var open_completion time.Completion
	loop.Storage.Open_At(&open_completion, nbio.DIRECTORY_CURRENT, path, nbio.Open_At_Options{
		Access: nbio.OPEN_READ_WRITE, Create: true, Truncate: true, Mode: 0o600,
	}, func(completed *time.Completion) {
		if !testify.No_Error(t, completed.Error) {
			return
		}
		opened = nbio.File(completed.Data)
	})
	testify.True(t,
		operating_system_run_until(
			t, driver, func() (finished bool) { return opened >= 0 },
		))
	written := false
	var write_completion time.Completion
	loop.Storage.Write(&write_completion, opened, []byte("hello"), 10, REAL_DEADLINE,
		func(completed *time.Completion) {
			testify.No_Error(t, completed.Error)
			written = completed.Data == 5
		})
	testify.True(t,
		operating_system_run_until(t, driver, func() (finished bool) { return written }))
	self_exec_close(loop, driver, opened)

	status := nbio.Statx{}
	statted := false
	var statx_completion time.Completion
	loop.Statx(
		&statx_completion, nbio.DIRECTORY_CURRENT, path, 0, nbio.STATX_BASIC_STATS, &status,
		func(completed *time.Completion) {
			testify.No_Error(t, completed.Error)
			statted = true
		},
	)
	testify.True(t,
		operating_system_run_until(t, driver, func() (finished bool) { return statted }))
	testify.Equal(t, uint64(15), status.Size)
}
