//go:build linux

package io

import (
	"os"
	"strconv"
	"strings"
	"syscall"
	"testing"

	sharedio "local/james-orcales/shared/io"
)

// Test_Socket_Open_Linux_Profile verifies every Linux-only TigerBeetle client option on the
// configured descriptor. Buffer gets may report the kernel's doubled accounting value, so the
// requested size is a lower bound.
func Test_Socket_Open_Linux_Profile(t *testing.T) {
	descriptor, open_err := socket_open_tcp(sharedio.FAMILY_IPV4, sharedio.TCP_Options{
		Receive_Buffer: SOCKET_RECEIVE_BUFFER_SIZE,
		Send_Buffer:    SOCKET_SEND_BUFFER_SIZE,
		Keepalive: &sharedio.TCP_Keepalive{
			Idle_Seconds:     SOCKET_KEEPALIVE_IDLE_SECONDS,
			Interval_Seconds: SOCKET_KEEPALIVE_INTERVAL_SECONDS,
			Count:            SOCKET_KEEPALIVE_COUNT,
		},
		User_Timeout_Milliseconds: SOCKET_USER_TIMEOUT_MILLISECONDS,
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
