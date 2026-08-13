//go:build linux

package nbio

import (
	"strconv"
	"strings"
	"syscall"
	"testing"

	sharedio "local/james-orcales/shared/simulation/nbio"
	sysos "local/james-orcales/shared/simulation/os"
	"local/james-orcales/shared/simulation/time"
	timeos "local/james-orcales/shared/simulation/time/default"
)

// Caps the wait for a sysctl read, so a hang fails the test rather than blocking the package.
const SYSCTL_READ_DEADLINE = 5 * time.SECOND

// Test_Socket_Open_Linux_Profile verifies every Linux-only TigerBeetle client option on the
// configured descriptor. Buffer gets may report the kernel's doubled accounting value, so the
// requested size is a lower bound.
func Test_Socket_Open_Linux_Profile(t *testing.T) {
	descriptor, open_err := socket_open(
		sharedio.FAMILY_IPV4, sharedio.SOCKET_TRANSPORT_TCP,
	)
	if open_err != nil {
		t.Fatalf("socket open: %v", open_err)
	}
	defer syscall.Close(descriptor)
	// Every caller-selected option arrives through the setsockopt primitive now, so the test
	// applies the profile itself and then reads each value back.
	options := []struct {
		Option sharedio.Socket_Option
		Value  int
	}{
		{sharedio.SOCKET_OPTION_RECEIVE_BUFFER, SOCKET_RECEIVE_BUFFER_SIZE},
		{sharedio.SOCKET_OPTION_SEND_BUFFER, SOCKET_SEND_BUFFER_SIZE},
		{sharedio.SOCKET_OPTION_KEEPALIVE, 1},
		{sharedio.SOCKET_OPTION_KEEPALIVE_IDLE, SOCKET_KEEPALIVE_IDLE_SECONDS},
		{sharedio.SOCKET_OPTION_KEEPALIVE_INTERVAL, SOCKET_KEEPALIVE_INTERVAL_SECONDS},
		{sharedio.SOCKET_OPTION_KEEPALIVE_COUNT, SOCKET_KEEPALIVE_COUNT},
		{sharedio.SOCKET_OPTION_USER_TIMEOUT, SOCKET_USER_TIMEOUT_MILLISECONDS},
		{sharedio.SOCKET_OPTION_NO_DELAY, 1},
	}
	for _, option := range options {
		set_err := socket_option_set(descriptor, option.Option, option.Value)
		if set_err != nil {
			t.Fatalf("set socket option: %v", set_err)
		}
	}
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
		Option: syscall.SO_KEEPALIVE, Expected: 1,
	})
	socket_option_equal(&socket_option_input{
		Test: t, Descriptor: descriptor, Level: syscall.IPPROTO_TCP,
		Option: syscall.TCP_KEEPIDLE, Expected: SOCKET_KEEPALIVE_IDLE_SECONDS,
	})
	socket_option_equal(&socket_option_input{
		Test: t, Descriptor: descriptor, Level: syscall.IPPROTO_TCP,
		Option: syscall.TCP_KEEPINTVL, Expected: SOCKET_KEEPALIVE_INTERVAL_SECONDS,
	})
	socket_option_equal(&socket_option_input{
		Test: t, Descriptor: descriptor, Level: syscall.IPPROTO_TCP,
		Option: syscall.TCP_KEEPCNT, Expected: SOCKET_KEEPALIVE_COUNT,
	})
	socket_option_equal(&socket_option_input{
		Test: t, Descriptor: descriptor, Level: syscall.IPPROTO_TCP,
		Option: SOCKET_USER_TIMEOUT, Expected: SOCKET_USER_TIMEOUT_MILLISECONDS,
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

// Bounds the buffer holding one sysctl value. A kernel buffer limit is a short decimal.
const SYSCTL_READ_BYTES = 64

// Reads a sysctl file through the loop, returning the byte count. This package is the io
// gateway, so even its internal tests read files through io.IO rather than around it.
func socket_sysctl_read(t *testing.T, path string, content []byte) (count int) {
	t.Helper()
	clock, _ := timeos.New_Operating_System_Any_Clock()
	loop, _, driver, _, loop_err := New_Operating_System_IO(
		clock, 32, 0, sysos.Virtual_OS_To_OS(sysos.Virtual_OS{Identifier: 1}))
	if loop_err != nil {
		t.Fatalf("scheduler: %v", loop_err)
	}
	file := sharedio.File(-1)
	open_done := false
	var open_completion time.Completion
	loop.Open_At(&open_completion, func(
		_ *time.Completion, opened sharedio.File, open_err error,
	) {
		if open_err != nil {
			t.Errorf("open %s: %v", path, open_err)
		}
		file = opened
		open_done = true
	}, sharedio.DIRECTORY_CURRENT, path, sharedio.Open_At_Options{
		Access: sharedio.OPEN_READ_ONLY,
	})
	driver.Run_Until(func() (finished bool) { return open_done }, SYSCTL_READ_DEADLINE)
	read_done := false
	var read_completion time.Completion
	loop.Read(&read_completion, func(_ *time.Completion, read int, read_err error) {
		if read_err != nil {
			t.Errorf("read %s: %v", path, read_err)
		}
		count = read
		read_done = true
	}, file, content, 0)
	driver.Run_Until(func() (finished bool) { return read_done }, SYSCTL_READ_DEADLINE)
	if !read_done {
		t.Fatalf("the read of %s did not complete", path)
	}
	close_done := false
	var close_completion time.Completion
	loop.Close(&close_completion, func(_ *time.Completion, close_err error) {
		if close_err != nil {
			t.Errorf("close %s: %v", path, close_err)
		}
		close_done = true
	}, file)
	driver.Run_Until(func() (finished bool) { return close_done }, SYSCTL_READ_DEADLINE)
	driver.Deinit()
	return count
}

// Returns the requested buffer size capped at the unprivileged kernel maximum. Forced sizing
// reaches requested; its permission-denied fallback is capped by this sysctl.
func socket_buffer_minimum(t *testing.T, path string, requested int) (minimum int) {
	t.Helper()
	content := make([]byte, SYSCTL_READ_BYTES)
	count := socket_sysctl_read(t, path, content)
	maximum, parse_err := strconv.Atoi(strings.TrimSpace(string(content[:count])))
	if parse_err != nil {
		t.Fatalf("parse %s: %v", path, parse_err)
	}
	if maximum < requested {
		return maximum
	}
	return requested
}

func socket_option_equal(input *socket_option_input) {
	input.Test.Helper()
	value, err := syscall.GetsockoptInt(input.Descriptor, input.Level, input.Option)
	if err != nil {
		input.Test.Fatalf("getsockopt %d: %v", input.Option, err)
	}
	if value != input.Expected {
		input.Test.Fatalf(
			"socket option %d = %d, want %d", input.Option, value, input.Expected)
	}
}

func socket_option_at_least(input *socket_option_input) {
	input.Test.Helper()
	value, err := syscall.GetsockoptInt(input.Descriptor, input.Level, input.Option)
	if err != nil {
		input.Test.Fatalf("getsockopt %d: %v", input.Option, err)
	}
	if value < input.Expected {
		input.Test.Fatalf(
			"socket option %d = %d, want at least %d",
			input.Option, value, input.Expected,
		)
	}
}
