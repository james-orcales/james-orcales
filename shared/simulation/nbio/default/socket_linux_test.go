//go:build linux

package nbio

import (
	"syscall"
	"testing"

	"local/james-orcales/shared/simulation/nbio"
	"local/james-orcales/shared/simulation/time"
	"local/james-orcales/shared/testify"
)

// Cap wait of sysctl read, thus hang fail test, not block package.
const SYSCTL_READ_DEADLINE = 5 * time.SECOND

// Linux can cap an unprivileged request and can report twice the requested accounting quantity.
func socket_test_default_buffers(
	t *testing.T, descriptor int, receive_buffer_bytes uint32, send_buffer_bytes uint32,
) {
	t.Helper()
	socket_option_at_least(&socket_option_input{
		Test: t, Descriptor: descriptor, Level: syscall.SOL_SOCKET,
		Option: syscall.SO_RCVBUF,
		Expected: socket_buffer_minimum(
			t, "/proc/sys/net/core/rmem_max", int(receive_buffer_bytes)),
	})
	socket_option_at_least(&socket_option_input{
		Test: t, Descriptor: descriptor, Level: syscall.SOL_SOCKET,
		Option: syscall.SO_SNDBUF,
		Expected: socket_buffer_minimum(
			t, "/proc/sys/net/core/wmem_max", int(send_buffer_bytes)),
	})
}

// Buffer get may report kernel doubled accounting value, thus requested size is a lower bound.
func Test_Socket_Open_Linux_Profile(t *testing.T) {
	options := socket_test_tcp_options()
	descriptor, open_err := socket_open_tcp(nbio.FAMILY_IPV4, options)
	if !testify.No_Error(t, open_err) {
		return
	}
	defer syscall.Close(descriptor)
	socket_option_at_least(&socket_option_input{
		Test: t, Descriptor: descriptor, Level: syscall.SOL_SOCKET,
		Option: syscall.SO_RCVBUF,
		Expected: socket_buffer_minimum(
			t, "/proc/sys/net/core/rmem_max", SOCKET_RECEIVE_BUFFER_SIZE),
	})
	socket_option_at_least(&socket_option_input{
		Test: t, Descriptor: descriptor, Level: syscall.SOL_SOCKET,
		Option: syscall.SO_SNDBUF,
		Expected: socket_buffer_minimum(
			t, "/proc/sys/net/core/wmem_max", SOCKET_SEND_BUFFER_SIZE),
	})
	socket_option_equal(&socket_option_input{
		Test: t, Descriptor: descriptor, Level: syscall.SOL_SOCKET,
		Option: syscall.SO_RCVLOWAT, Expected: int(options.Receive_Low_Water_Bytes),
	})
	socket_option_equal(&socket_option_input{
		Test: t, Descriptor: descriptor, Level: syscall.SOL_SOCKET,
		Option: syscall.SO_KEEPALIVE, Expected: 1,
	})
	socket_option_equal(&socket_option_input{
		Test: t, Descriptor: descriptor, Level: syscall.IPPROTO_TCP,
		Option: syscall.TCP_KEEPIDLE, Expected: int(options.Keepalive.Idle / time.SECOND),
	})
	socket_option_equal(&socket_option_input{
		Test: t, Descriptor: descriptor, Level: syscall.IPPROTO_TCP,
		Option:   syscall.TCP_KEEPINTVL,
		Expected: int(options.Keepalive.Interval / time.SECOND),
	})
	socket_option_equal(&socket_option_input{
		Test: t, Descriptor: descriptor, Level: syscall.IPPROTO_TCP,
		Option: syscall.TCP_KEEPCNT, Expected: int(options.Keepalive.Probe_Count),
	})
	socket_option_equal(&socket_option_input{
		Test: t, Descriptor: descriptor, Level: syscall.IPPROTO_TCP,
		Option: syscall.TCP_MAXSEG, Expected: int(options.Maximum_Segment_Bytes),
	})
	socket_option_equal(&socket_option_input{
		Test: t, Descriptor: descriptor, Level: syscall.IPPROTO_TCP,
		Option:   SOCKET_TCP_NOT_SENT_LOW_WATER,
		Expected: int(options.Not_Sent_Low_Water_Bytes),
	})
	socket_option_equal(&socket_option_input{
		Test: t, Descriptor: descriptor, Level: syscall.IPPROTO_TCP,
		Option: syscall.TCP_NODELAY, Expected: 1,
	})
}

type socket_option_input struct {
	Test       *testing.T
	Descriptor int
	Level      int
	Option     int
	Expected   int
}

// Bound buffer holding one sysctl value. Kernel buffer limit is short decimal.
const SYSCTL_READ_BYTES = 64

// Read sysctl file through loop. Return byte count. This package is io gateway, thus even its
// internal tests read files through io.IO, not around it.
func socket_sysctl_read(t *testing.T, path string, content []byte) (count int) {
	t.Helper()
	clock := new_operating_system_clock()
	memory := &operating_system_test_memory{}
	loop, _, driver, _, loop_err := New_Operating_System_IO(
		&memory.State, operating_system_memory_view(memory),
		clock, 32, 0, operating_system_ambient())
	if !testify.No_Error(t, loop_err) {
		return 0
	}
	file := nbio.File(-1)
	open_done := false
	var open_completion time.Completion
	nbio.Storage_Open_At(
		loop.Storage, &open_completion, nbio.DIRECTORY_CURRENT, path, nbio.Open_At_Options{
			Access: nbio.OPEN_READ_ONLY,
		}, func(
			completed *time.Completion,
		) {
			testify.No_Error(t, completed.Error, path)
			file = nbio.File(completed.Data)
			open_done = true
		})
	time.Driver_Run_Until(driver, SYSCTL_READ_DEADLINE,
		func() (finished bool) { return open_done })
	read_done := false
	var read_completion time.Completion
	nbio.Storage_Read(loop.Storage, &read_completion, file, content, 0, SYSCTL_READ_DEADLINE,
		func(completed *time.Completion) {
			testify.No_Error(t, completed.Error, path)
			count = completed.Data
			read_done = true
		})
	time.Driver_Run_Until(driver, SYSCTL_READ_DEADLINE,
		func() (finished bool) { return read_done })
	testify.True(t, read_done, path)
	close_done := false
	var close_completion time.Completion
	nbio.IO_Close(loop, &close_completion, file, func(completed *time.Completion) {
		testify.No_Error(t, completed.Error, path)
		close_done = true
	})
	time.Driver_Run_Until(driver, SYSCTL_READ_DEADLINE,
		func() (finished bool) { return close_done })
	time.Driver_Deinit(driver)
	return count
}

// Return requested buffer size capped at unprivileged kernel maximum. Forced sizing reach
// requested. This sysctl cap its permission-denied fallback.
func socket_buffer_minimum(t *testing.T, path string, requested int) (minimum int) {
	t.Helper()
	content := make([]byte, SYSCTL_READ_BYTES)
	count := socket_sysctl_read(t, path, content)
	maximum, parsed := test_decimal(content[:count])
	if !testify.True(t, parsed, path) {
		return 0
	}
	if maximum < requested {
		return maximum
	}
	return requested
}

func socket_option_equal(input *socket_option_input) {
	input.Test.Helper()
	value, err := syscall.GetsockoptInt(input.Descriptor, input.Level, input.Option)
	if !testify.No_Error(input.Test, err, input.Option) {
		return
	}
	testify.Equal(input.Test, input.Expected, value, input.Option)
}

func socket_option_at_least(input *socket_option_input) {
	input.Test.Helper()
	value, err := syscall.GetsockoptInt(input.Descriptor, input.Level, input.Option)
	if !testify.No_Error(input.Test, err, input.Option) {
		return
	}
	testify.Greater_Or_Equal(input.Test, &testify.Greater_Or_Equal_Input[int]{
		First: value, Second: input.Expected,
	}, input.Option)
}
