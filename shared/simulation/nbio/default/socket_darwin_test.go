//go:build darwin

package nbio

import (
	"syscall"
	"testing"

	"local/james-orcales/shared/simulation/nbio"
	"local/james-orcales/shared/simulation/time"
	"local/james-orcales/shared/testify"
)

// Darwin reports the applied byte quantity without the Linux accounting multiplier.
func socket_test_default_buffers(
	t *testing.T, descriptor int, receive_buffer_bytes uint32, send_buffer_bytes uint32,
) {
	t.Helper()
	socket_test_option_equal(
		t, descriptor, syscall.SOL_SOCKET, syscall.SO_RCVBUF, int(receive_buffer_bytes))
	socket_test_option_equal(
		t, descriptor, syscall.SOL_SOCKET, syscall.SO_SNDBUF, int(send_buffer_bytes))
}

// Test_Socket_Open_Darwin_No_Sigpipe verify Darwin-only portable client option suppress SIGPIPE.
// It complement common black-box buffer, keepalive, nonblocking, and CLOEXEC checks.
func Test_Socket_Open_Darwin_No_Sigpipe(t *testing.T) {
	descriptor, open_err := socket_open_tcp(nbio.FAMILY_IPV4, socket_test_tcp_options())
	if !testify.No_Error(t, open_err) {
		return
	}
	defer syscall.Close(descriptor)
	value, get_err := syscall.GetsockoptInt(
		descriptor, syscall.SOL_SOCKET, SOCKET_NO_SIGPIPE)
	testify.No_Error(t, get_err)
	testify.Not_Zero(t, value)
	receive_buffer, receive_err := syscall.GetsockoptInt(
		descriptor, syscall.SOL_SOCKET, syscall.SO_RCVBUF)
	testify.No_Error(t, receive_err)
	testify.Greater_Or_Equal(t, &testify.Greater_Or_Equal_Input[int]{
		First: receive_buffer, Second: SOCKET_RECEIVE_BUFFER_SIZE,
	})
	send_buffer, send_err := syscall.GetsockoptInt(
		descriptor, syscall.SOL_SOCKET, syscall.SO_SNDBUF)
	testify.No_Error(t, send_err)
	testify.Greater_Or_Equal(t, &testify.Greater_Or_Equal_Input[int]{
		First: send_buffer, Second: SOCKET_SEND_BUFFER_SIZE,
	})
}

// State counters alone cannot prove the kernel filter was deleted, so wake the old descriptor.
func Test_Platform_Expire_Operation_Deletes_Kernel_Registration(t *testing.T) {
	platform, platform_err := platform_initialize(1, 0)
	if !testify.No_Error(t, platform_err) {
		return
	}
	state := &Operating_System{Platform: platform}
	defer platform_deinitialize(state)
	descriptors := []int{0, 0}
	if !testify.No_Error(t, syscall.Pipe(descriptors)) {
		return
	}
	defer syscall.Close(descriptors[0])
	defer syscall.Close(descriptors[1])
	operation := &Operating_System_Operation{
		Kind:       OPERATING_SYSTEM_OPERATION_RECEIVE,
		Descriptor: descriptors[0],
		Identifier: 1,
	}
	operation.Kernel_Submitted = true
	change := platform_change(operation)
	_, add_err := kernel_kevent(&Kernel_Kevent_Input{
		Descriptor: state.Platform.Descriptor,
		Changes:    []Kernel_Event{change},
		Wait:       0,
	})
	if !testify.No_Error(t, add_err) {
		return
	}
	state.Platform.IO_Inflight = 1
	if !testify.No_Error(t, platform_expire_operation(state, operation)) {
		return
	}
	_, write_err := syscall.Write(descriptors[1], []byte{1})
	if !testify.No_Error(t, write_err) {
		return
	}
	events := make([]Kernel_Event, 1)
	count, wait_err := kernel_kevent(&Kernel_Kevent_Input{
		Descriptor: state.Platform.Descriptor,
		Events:     events,
		Wait:       0,
	})
	testify.No_Error(t, wait_err)
	testify.Zero(t, count)
}

// Darwin must omit the deadline, or common expiration can report a timeout that did not retire
// the eager file syscall.
func Test_Platform_Storage_Deadline_Darwin_Is_Disabled(t *testing.T) {
	state := &Operating_System{
		Host: time.Clock{
			Now_Monotonic: func() (moment time.Monotonic_Moment) { return 7 },
		},
	}
	deadline := platform_storage_deadline(state, time.NANOSECOND)
	testify.Zero(t, deadline)
}
