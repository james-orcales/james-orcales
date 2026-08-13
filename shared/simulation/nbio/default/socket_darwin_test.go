//go:build darwin

package nbio

import (
	"syscall"
	"testing"

	sharedio "local/james-orcales/shared/simulation/nbio"
)

// Test_Socket_Open_Darwin_No_Sigpipe verifies the Darwin-only portable client option suppresses
// SIGPIPE, complementing the common black-box buffer, keepalive, nonblocking, and CLOEXEC checks.
func Test_Socket_Open_Darwin_No_Sigpipe(t *testing.T) {
	descriptor, open_err := socket_open(
		sharedio.FAMILY_IPV4, sharedio.SOCKET_TRANSPORT_TCP,
	)
	if open_err != nil {
		t.Fatalf("socket open: %v", open_err)
	}
	defer syscall.Close(descriptor)
	// The buffer sizes are caller options now, so the test applies them through the
	// setsockopt primitive. NOSIGPIPE stays platform-mandatory inside socket_open.
	buffers := []struct {
		Option sharedio.Socket_Option
		Value  int
	}{
		{sharedio.SOCKET_OPTION_RECEIVE_BUFFER, SOCKET_RECEIVE_BUFFER_SIZE},
		{sharedio.SOCKET_OPTION_SEND_BUFFER, SOCKET_SEND_BUFFER_SIZE},
	}
	for _, buffer := range buffers {
		set_err := socket_option_set(descriptor, buffer.Option, buffer.Value)
		if set_err != nil {
			t.Fatalf("set socket option: %v", set_err)
		}
	}
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
