//go:build linux

package io

import (
	"os"
	"strconv"
	"strings"
	"syscall"
	"testing"

	sharedio "local/james-orcales/g/shared/io"
)

// Test_Socket_Open_Linux_Profile verifies every Linux-only TigerBeetle client option on the
// configured descriptor. Buffer gets may report the kernel's doubled accounting value, so the
// requested size is a lower bound.
func Test_Socket_Open_Linux_Profile(t *testing.T) {
	descriptor, open_err := socket_open_tcp(sharedio.FAMILY_IPV4, sharedio.TCP_Options{
		Receive_Buffer: socket_receive_buffer_size,
		Send_Buffer:    socket_send_buffer_size,
		Keepalive: &sharedio.TCP_Keepalive{
			Idle_Seconds:     socket_keepalive_idle_seconds,
			Interval_Seconds: socket_keepalive_interval_seconds,
			Count:            socket_keepalive_count,
		},
		User_Timeout_Milliseconds: socket_user_timeout_milliseconds,
		No_Delay:                  true,
	})
	if open_err != nil {
		t.Fatalf("socket open: %v", open_err)
	}
	defer syscall.Close(descriptor)
	socket_option_at_least(&socket_option_input{
		Test: t, Descriptor: descriptor, Level: syscall.SOL_SOCKET,
		Option: syscall.SO_RCVBUF,
		Expected: socket_buffer_minimum(
			t, "/proc/sys/net/core/rmem_max", socket_receive_buffer_size),
	})
	socket_option_at_least(&socket_option_input{
		Test: t, Descriptor: descriptor, Level: syscall.SOL_SOCKET,
		Option: syscall.SO_SNDBUF,
		Expected: socket_buffer_minimum(
			t, "/proc/sys/net/core/wmem_max", socket_send_buffer_size),
	})
	socket_option_equal(&socket_option_input{
		Test: t, Descriptor: descriptor, Level: syscall.SOL_SOCKET,
		Option: syscall.SO_KEEPALIVE, Expected: 1,
	})
	socket_option_equal(&socket_option_input{
		Test: t, Descriptor: descriptor, Level: syscall.IPPROTO_TCP,
		Option: syscall.TCP_KEEPIDLE, Expected: socket_keepalive_idle_seconds,
	})
	socket_option_equal(&socket_option_input{
		Test: t, Descriptor: descriptor, Level: syscall.IPPROTO_TCP,
		Option: syscall.TCP_KEEPINTVL, Expected: socket_keepalive_interval_seconds,
	})
	socket_option_equal(&socket_option_input{
		Test: t, Descriptor: descriptor, Level: syscall.IPPROTO_TCP,
		Option: syscall.TCP_KEEPCNT, Expected: socket_keepalive_count,
	})
	socket_option_equal(&socket_option_input{
		Test: t, Descriptor: descriptor, Level: syscall.IPPROTO_TCP,
		Option: socket_user_timeout, Expected: socket_user_timeout_milliseconds,
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

// Returns the requested buffer size capped at the unprivileged kernel maximum. Forced sizing
// reaches requested; its permission-denied fallback is capped by this sysctl.
func socket_buffer_minimum(t *testing.T, path string, requested int) (minimum int) {
	t.Helper()
	file, open_err := os.Open(path)
	if open_err != nil {
		t.Fatalf("open %s: %v", path, open_err)
	}
	defer file.Close()
	content := make([]byte, 64)
	count, read_err := file.Read(content)
	if read_err != nil {
		t.Fatalf("read %s: %v", path, read_err)
	}
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
