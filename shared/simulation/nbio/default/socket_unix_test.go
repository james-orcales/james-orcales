//go:build darwin || linux

package nbio

import (
	"syscall"
	"testing"
	"unsafe"

	sharedio "local/james-orcales/shared/simulation/nbio"
	"local/james-orcales/shared/simulation/time"
	"local/james-orcales/shared/testify"
)

// One profile keeps platform checks comparable instead of letting each kernel test weaker values.
func socket_test_tcp_options() (options sharedio.TCP_Options) {
	return sharedio.TCP_Options{
		Receive_Buffer_Bytes:     SOCKET_RECEIVE_BUFFER_SIZE,
		Send_Buffer_Bytes:        SOCKET_SEND_BUFFER_SIZE,
		Receive_Low_Water_Bytes:  1,
		Linger_Timeout:           1 * time.SECOND,
		Maximum_Segment_Bytes:    512,
		Not_Sent_Low_Water_Bytes: 1024,
		Keepalive: sharedio.TCP_Keepalive{
			Idle: 5 * time.SECOND, Interval: 4 * time.SECOND, Probe_Count: 3,
		},
		No_Delay: true,
	}
}

// UDP has no portable transport-specific override, so its profile contains the shared set.
func socket_test_udp_options() (options sharedio.UDP_Options) {
	return sharedio.UDP_Options{
		Receive_Buffer_Bytes:    SOCKET_RECEIVE_BUFFER_SIZE,
		Send_Buffer_Bytes:       SOCKET_SEND_BUFFER_SIZE,
		Receive_Low_Water_Bytes: 1,
		Linger_Timeout:          1 * time.SECOND,
	}
}

// Build the pure TCP profile explicitly because zero is a disabled limit, not an omission.
func socket_test_tcp_default_options() (options sharedio.TCP_Options) {
	return sharedio.TCP_Options{
		Receive_Buffer_Bytes:     sharedio.TCP_RECEIVE_BUFFER_BYTES_DEFAULT,
		Send_Buffer_Bytes:        sharedio.TCP_SEND_BUFFER_BYTES_DEFAULT,
		Receive_Low_Water_Bytes:  sharedio.SOCKET_RECEIVE_LOW_WATER_BYTES_DEFAULT,
		Linger_Timeout:           sharedio.SOCKET_LINGER_TIMEOUT_DEFAULT,
		Maximum_Segment_Bytes:    sharedio.TCP_MAXIMUM_SEGMENT_BYTES_DEFAULT,
		Not_Sent_Low_Water_Bytes: sharedio.TCP_NOT_SENT_LOW_WATER_BYTES_DEFAULT,
		Keepalive: sharedio.TCP_Keepalive{
			Idle:        sharedio.TCP_KEEPALIVE_IDLE_DEFAULT,
			Interval:    sharedio.TCP_KEEPALIVE_INTERVAL_DEFAULT,
			Probe_Count: sharedio.TCP_KEEPALIVE_PROBE_COUNT_DEFAULT,
		},
		No_Delay: sharedio.TCP_NO_DELAY_DEFAULT,
	}
}

// Build the pure UDP profile explicitly because zero is a disabled limit, not an omission.
func socket_test_udp_default_options() (options sharedio.UDP_Options) {
	return sharedio.UDP_Options{
		Receive_Buffer_Bytes:    sharedio.UDP_RECEIVE_BUFFER_BYTES_DEFAULT,
		Send_Buffer_Bytes:       sharedio.UDP_SEND_BUFFER_BYTES_DEFAULT,
		Receive_Low_Water_Bytes: sharedio.SOCKET_RECEIVE_LOW_WATER_BYTES_DEFAULT,
		Linger_Timeout:          sharedio.SOCKET_LINGER_TIMEOUT_DEFAULT,
	}
}

// Inspect both descriptors because a TCP-only check cannot prove UDP also receives generic limits.
func Test_Socket_Open_Portable_Profiles(t *testing.T) {
	tcp_options := socket_test_tcp_options()
	tcp, tcp_err := socket_open_tcp(sharedio.FAMILY_IPV4, tcp_options)
	if !testify.No_Error(t, tcp_err) {
		return
	}
	defer syscall.Close(tcp)
	socket_test_common_options(
		t, tcp, tcp_options.Receive_Low_Water_Bytes, tcp_options.Linger_Timeout)
	socket_test_option_equal(
		t, tcp, syscall.IPPROTO_TCP, syscall.TCP_MAXSEG,
		int(tcp_options.Maximum_Segment_Bytes))
	socket_test_option_equal(
		t, tcp, syscall.IPPROTO_TCP, SOCKET_TCP_NOT_SENT_LOW_WATER,
		int(tcp_options.Not_Sent_Low_Water_Bytes))
	socket_test_option_positive(t, tcp, syscall.SOL_SOCKET, syscall.SO_KEEPALIVE)
	socket_test_option_equal(
		t, tcp, syscall.IPPROTO_TCP, SOCKET_TCP_KEEPALIVE_IDLE,
		int(tcp_options.Keepalive.Idle/time.SECOND))
	socket_test_option_equal(
		t, tcp, syscall.IPPROTO_TCP, syscall.TCP_KEEPINTVL,
		int(tcp_options.Keepalive.Interval/time.SECOND))
	socket_test_option_equal(
		t, tcp, syscall.IPPROTO_TCP, syscall.TCP_KEEPCNT,
		int(tcp_options.Keepalive.Probe_Count))
	socket_test_option_positive(t, tcp, syscall.IPPROTO_TCP, syscall.TCP_NODELAY)

	udp_options := socket_test_udp_options()
	udp, udp_err := socket_open_udp(sharedio.FAMILY_IPV4, udp_options)
	if !testify.No_Error(t, udp_err) {
		return
	}
	defer syscall.Close(udp)
	socket_test_common_options(
		t, udp, udp_options.Receive_Low_Water_Bytes, udp_options.Linger_Timeout)
}

// Test each limit separately because one rejected empty profile cannot prove all backend checks.
func Test_Socket_Open_Rejects_Disabled_Limits(t *testing.T) {
	tcp := socket_test_tcp_options()
	tcp.Receive_Buffer_Bytes = 0
	socket_test_tcp_limit_rejected(t, "receive buffer", tcp)
	tcp = socket_test_tcp_options()
	tcp.Send_Buffer_Bytes = 0
	socket_test_tcp_limit_rejected(t, "send buffer", tcp)
	tcp = socket_test_tcp_options()
	tcp.Receive_Low_Water_Bytes = 0
	socket_test_tcp_limit_rejected(t, "receive low water", tcp)
	tcp.Linger_Timeout = 0
	socket_test_tcp_limit_rejected(t, "linger timeout", tcp)
	tcp = socket_test_tcp_options()
	tcp.Maximum_Segment_Bytes = 0
	socket_test_tcp_limit_rejected(t, "maximum segment", tcp)
	tcp = socket_test_tcp_options()
	tcp.Not_Sent_Low_Water_Bytes = 0
	socket_test_tcp_limit_rejected(t, "not-sent low water", tcp)
	tcp = socket_test_tcp_options()
	tcp.Keepalive.Idle = 0
	socket_test_tcp_limit_rejected(t, "keepalive idle", tcp)
	tcp = socket_test_tcp_options()
	tcp.Keepalive.Interval = 0
	socket_test_tcp_limit_rejected(t, "keepalive interval", tcp)
	tcp = socket_test_tcp_options()
	tcp.Keepalive.Probe_Count = 0
	socket_test_tcp_limit_rejected(t, "keepalive probe count", tcp)

	udp := socket_test_udp_options()
	udp.Receive_Buffer_Bytes = 0
	socket_test_udp_limit_rejected(t, "receive buffer", udp)
	udp = socket_test_udp_options()
	udp.Send_Buffer_Bytes = 0
	socket_test_udp_limit_rejected(t, "send buffer", udp)
	udp = socket_test_udp_options()
	udp.Receive_Low_Water_Bytes = 0
	socket_test_udp_limit_rejected(t, "receive low water", udp)
	udp.Linger_Timeout = 0
	socket_test_udp_limit_rejected(t, "linger timeout", udp)
}

func socket_test_tcp_limit_rejected(t *testing.T, name string, options sharedio.TCP_Options) {
	t.Helper()
	descriptor, err := socket_open_tcp(sharedio.FAMILY_IPV4, options)
	if testify.Error(t, err, name) {
		return
	}
	syscall.Close(descriptor)
}

func socket_test_udp_limit_rejected(t *testing.T, name string, options sharedio.UDP_Options) {
	t.Helper()
	descriptor, err := socket_open_udp(sharedio.FAMILY_IPV4, options)
	if testify.Error(t, err, name) {
		return
	}
	syscall.Close(descriptor)
}

// A backend can reduce a pure default when its per-socket limit is lower.
func Test_Socket_Open_Default_Profiles(t *testing.T) {
	tcp, tcp_err := socket_open_tcp(sharedio.FAMILY_IPV4, socket_test_tcp_default_options())
	if !testify.No_Error(t, tcp_err) {
		return
	}
	defer syscall.Close(tcp)
	socket_test_common_defaults(
		t, tcp,
		sharedio.TCP_RECEIVE_BUFFER_BYTES_DEFAULT,
		sharedio.TCP_SEND_BUFFER_BYTES_DEFAULT)
	socket_test_option_positive(t, tcp, syscall.SOL_SOCKET, syscall.SO_KEEPALIVE)
	socket_test_option_equal(
		t, tcp, syscall.IPPROTO_TCP, SOCKET_TCP_KEEPALIVE_IDLE,
		int(sharedio.TCP_KEEPALIVE_IDLE_DEFAULT/time.SECOND))
	socket_test_option_equal(
		t, tcp, syscall.IPPROTO_TCP, syscall.TCP_KEEPINTVL,
		int(sharedio.TCP_KEEPALIVE_INTERVAL_DEFAULT/time.SECOND))
	socket_test_option_equal(
		t, tcp, syscall.IPPROTO_TCP, syscall.TCP_KEEPCNT,
		int(sharedio.TCP_KEEPALIVE_PROBE_COUNT_DEFAULT))
	socket_test_option_positive(t, tcp, syscall.IPPROTO_TCP, syscall.TCP_NODELAY)
	socket_test_option_equal(
		t, tcp, syscall.IPPROTO_TCP, syscall.TCP_MAXSEG,
		int(platform_tcp_maximum_segment_clamp(
			sharedio.TCP_MAXIMUM_SEGMENT_BYTES_DEFAULT)))
	socket_test_option_equal(
		t, tcp, syscall.IPPROTO_TCP, SOCKET_TCP_NOT_SENT_LOW_WATER,
		int(sharedio.TCP_NOT_SENT_LOW_WATER_BYTES_DEFAULT))

	udp, udp_err := socket_open_udp(sharedio.FAMILY_IPV4, socket_test_udp_default_options())
	if !testify.No_Error(t, udp_err) {
		return
	}
	defer syscall.Close(udp)
	socket_test_common_defaults(
		t, udp,
		sharedio.UDP_RECEIVE_BUFFER_BYTES_DEFAULT,
		sharedio.UDP_SEND_BUFFER_BYTES_DEFAULT)
}

// Every backend owns the comparison because the platform can cap the pure buffer profile.
func socket_test_common_defaults(
	t *testing.T, descriptor int, receive_buffer_bytes uint32, send_buffer_bytes uint32,
) {
	t.Helper()
	socket_test_default_buffers(t, descriptor, receive_buffer_bytes, send_buffer_bytes)
	socket_test_option_equal(
		t, descriptor, syscall.SOL_SOCKET, syscall.SO_RCVLOWAT,
		int(sharedio.SOCKET_RECEIVE_LOW_WATER_BYTES_DEFAULT))
	socket_test_linger_default(t, descriptor)
}

// Raw linger inspection proves the option cannot silently fall back to disabled.
func socket_test_common_options(
	t *testing.T, descriptor int, receive_low_water uint32, linger_timeout time.Duration,
) {
	t.Helper()
	socket_test_option_positive(t, descriptor, syscall.SOL_SOCKET, syscall.SO_RCVBUF)
	socket_test_option_positive(t, descriptor, syscall.SOL_SOCKET, syscall.SO_SNDBUF)
	socket_test_option_equal(
		t, descriptor, syscall.SOL_SOCKET, syscall.SO_RCVLOWAT, int(receive_low_water))
	socket_test_linger_equal(t, descriptor, linger_timeout)
}

func socket_test_option_equal(
	t *testing.T, descriptor int, level int, option int, expected int,
) {
	t.Helper()
	value, err := syscall.GetsockoptInt(descriptor, level, option)
	testify.No_Error(t, err, option)
	testify.Equal(t, expected, value, option)
}

func socket_test_option_positive(
	t *testing.T, descriptor int, level int, option int,
) {
	t.Helper()
	value, err := syscall.GetsockoptInt(descriptor, level, option)
	testify.No_Error(t, err, option)
	testify.Positive(t, value, option)
}

func socket_test_linger_equal(t *testing.T, descriptor int, expected time.Duration) {
	t.Helper()
	value := syscall.Linger{}
	socket_test_option_get(
		t, descriptor, syscall.SOL_SOCKET, syscall.SO_LINGER,
		unsafe.Pointer(&value), unsafe.Sizeof(value))
	testify.Equal(t, int32(1), value.Onoff)
	testify.Equal(t, int32(expected/time.SECOND), value.Linger)
}

func socket_test_linger_default(t *testing.T, descriptor int) {
	t.Helper()
	value := syscall.Linger{}
	socket_test_option_get(
		t, descriptor, syscall.SOL_SOCKET, syscall.SO_LINGER,
		unsafe.Pointer(&value), unsafe.Sizeof(value))
	testify.Equal(t, int32(1), value.Onoff)
	testify.Equal(t,
		int32(sharedio.SOCKET_LINGER_TIMEOUT_DEFAULT/time.SECOND), value.Linger)
}

func socket_test_option_get(
	t *testing.T, descriptor int, level int, option int, value unsafe.Pointer, size uintptr,
) {
	t.Helper()
	size_returned := uint32(size)
	_, _, errno := syscall.Syscall6(
		syscall.SYS_GETSOCKOPT,
		uintptr(descriptor), uintptr(level), uintptr(option),
		uintptr(value), uintptr(unsafe.Pointer(&size_returned)), 0)
	testify.Zero(t, errno, option)
	testify.Equal(t, size, uintptr(size_returned), option)
}
