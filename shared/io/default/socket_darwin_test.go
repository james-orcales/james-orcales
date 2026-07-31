//go:build darwin

package io

import (
	"syscall"
	"testing"

	sharedio "local/james-orcales/shared/io"
)

// Test_Socket_Open_Darwin_No_Sigpipe verifies the Darwin-only portable client option suppresses
// SIGPIPE, complementing the common black-box buffer, keepalive, nonblocking, and CLOEXEC checks.
func Test_Socket_Open_Darwin_No_Sigpipe(t *testing.T) {
	descriptor, open_err := socket_open_tcp(sharedio.FAMILY_IPV4, sharedio.TCP_Options{
		Receive_Buffer: SOCKET_RECEIVE_BUFFER_SIZE,
		Send_Buffer:    SOCKET_SEND_BUFFER_SIZE,
		Keepalive: &sharedio.TCP_Keepalive{
			Idle_Seconds: 5, Interval_Seconds: 4, Count: 3,
		},
	})
	if open_err != nil {
		t.Fatalf("socket open: %v", open_err)
	}
	defer syscall.Close(descriptor)
	value, get_err := syscall.GetsockoptInt(
		descriptor, syscall.SOL_SOCKET, SOCKET_NO_SIGPIPE)
	if get_err != nil {
		t.Fatalf("get SO_NOSIGPIPE: %v", get_err)
	}
	if value == 0 {
		t.Fatal("SO_NOSIGPIPE is disabled")
	}
	receive_buffer, receive_err := syscall.GetsockoptInt(
		descriptor, syscall.SOL_SOCKET, syscall.SO_RCVBUF)
	if receive_err != nil {
		t.Fatalf("get receive buffer: %v", receive_err)
	}
	if receive_buffer < SOCKET_RECEIVE_BUFFER_SIZE {
		t.Fatalf("receive buffer = %d, want %d", receive_buffer, SOCKET_RECEIVE_BUFFER_SIZE)
	}
	send_buffer, send_err := syscall.GetsockoptInt(
		descriptor, syscall.SOL_SOCKET, syscall.SO_SNDBUF)
	if send_err != nil {
		t.Fatalf("get send buffer: %v", send_err)
	}
	if send_buffer < SOCKET_SEND_BUFFER_SIZE {
		t.Fatalf("send buffer = %d, want %d", send_buffer, SOCKET_SEND_BUFFER_SIZE)
	}
}
