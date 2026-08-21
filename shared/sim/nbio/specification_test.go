package nbio_test

import (
	"fmt"
	"testing"
	"unsafe"

	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/nbio"
	"local/james-orcales/shared/sim/time"
	"local/james-orcales/shared/snap/default"
	"local/james-orcales/shared/testify"
)

// Test_Sim_Read verify read on opened file return bytes earlier write store — real-bytes path of
// file descriptor, distinct from socket byte count.
func Test_Sim_Read(t *testing.T) {
	loop, driver, _ := sim_loop(1)

	writer, create_err := sim_create(t, loop, driver, "file")
	if !testify.No_Error(t, create_err) {
		return
	}
	wrote := false
	var write_completion nbio.Completion
	nbio.Storage_Write(
		loop.Storage, &write_completion, writer, []byte("hello"), 0, SIM_DEADLINE,
		func(completed nbio.Completion_Handle) {
			testify.No_Error(t, completed.Error)
			wrote = true
		})
	nbio.Driver_Run_Until(driver, SIM_DEADLINE, func() (finished bool) { return wrote })

	reader, open_err := sim_open(t, loop, driver, "file")
	if !testify.No_Error(t, open_err) {
		return
	}
	buffer := make([]byte, 64)
	count := -1
	var read_completion nbio.Completion
	nbio.Storage_Read(loop.Storage, &read_completion, reader, buffer, 0, SIM_DEADLINE,
		func(completed nbio.Completion_Handle) {
			count = completed.Data
		})
	nbio.Driver_Run_Until(driver, SIM_DEADLINE, func() (finished bool) { return count >= 0 })

	testify.Equal(t, 5, count)
	testify.Equal(t, "hello", string(buffer[:count]))
	sim_close(t, loop, driver, writer)
	sim_close(t, loop, driver, reader)
	nbio.IO_Deinit(loop)

	socket_loop, _, _ := sim_loop(1)
	socket, socket_err := nbio.Network_Socket_TCP(socket_loop.Network,
		nbio.FAMILY_IPV4, sim_tcp_options())
	if !testify.No_Error(t, socket_err) {
		return
	}
	var socket_completion nbio.Completion
	testify.Panics(t, func() {
		nbio.Storage_Read(
			socket_loop.Storage, &socket_completion, socket, make([]byte, 8), 0,
			SIM_DEADLINE,
			func(_ nbio.Completion_Handle) {})
	})
}

// The write result must report accepted bytes before the caller can close its descriptor.
func Test_Sim_Write(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	file, create_err := sim_create(t, loop, driver, "file")
	if !testify.No_Error(t, create_err) {
		return
	}
	count := -1
	var completion nbio.Completion
	nbio.Storage_Write(loop.Storage, &completion, file, []byte("hello"), 0, SIM_DEADLINE,
		func(completed nbio.Completion_Handle) {
			testify.No_Error(t, completed.Error)
			count = completed.Data
		})
	nbio.Driver_Run_Until(driver, SIM_DEADLINE, func() (finished bool) { return count >= 0 })
	testify.Equal(t, len("hello"), count)
	sim_close(t, loop, driver, file)
	nbio.IO_Deinit(loop)
}

// Test_Sim_Fsync verify fsync operation complete without change of descriptor
// ownership, after prior write retired.
func Test_Sim_Fsync(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	file, create_err := sim_create(t, loop, driver, "file")
	if !testify.No_Error(t, create_err) {
		return
	}
	fired := false
	var completion nbio.Completion
	nbio.Storage_Fsync(loop.Storage, &completion, file, SIM_DEADLINE,
		func(completed nbio.Completion_Handle) {
			testify.No_Error(t, completed.Error)
			fired = true
		})
	nbio.Driver_Run_Until(driver, SIM_DEADLINE, func() (finished bool) { return fired })
	testify.True(t, sim_descriptor_open(loop))
	sim_close(t, loop, driver, file)
	nbio.IO_Deinit(loop)
}

// Test_Sim_Open_At verify asynchronous openat surface make caller-owned file ordinary read and
// write operations can use.
func Test_Sim_Open_At(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	opened := nbio.File(-1)
	var completion nbio.Completion
	nbio.Storage_Open_At(loop.Storage, &completion, nbio.DIRECTORY_CURRENT, "file",
		nbio.Open_At_Options{
			Access: nbio.OPEN_READ_WRITE, Create: true, Truncate: true,
			Permissions: 0o600,
		}, func(completed nbio.Completion_Handle) {
			testify.No_Error(t, completed.Error)
			opened = nbio.File(completed.Data)
		})
	nbio.Driver_Run_Until(driver, SIM_DEADLINE, func() (finished bool) { return opened >= 0 })
	testify.Positive(t, opened)
	testify.True(t, sim_descriptor_open(loop))
	sim_close(t, loop, driver, opened)
	nbio.IO_Deinit(loop)

	unknown_loop, _, _ := sim_loop(0)
	var unknown_completion nbio.Completion
	testify.Panics(t, func() {
		nbio.Storage_Open_At(
			unknown_loop.Storage, &unknown_completion,
			nbio.DIRECTORY_CURRENT,
			"file",
			nbio.Open_At_Options{Flags: nbio.Open_At_Flags(1 << 31)},
			func(_ nbio.Completion_Handle) {},
		)
	})
}

// Create and truncate are independent kernel flags; reopening must preserve that distinction.
func Test_Sim_Open_Options(t *testing.T) {
	for _, create := range []bool{false, true} {
		for _, truncate := range []bool{false, true} {
			loop, driver, _ := sim_loop(0)
			file, err := sim_create(t, loop, driver, "file")
			if !testify.No_Error(t, err) {
				return
			}
			sim_storage_write(t, loop, driver, file, []byte("secret"))
			sim_close(t, loop, driver, file)
			file, err = sim_open_options(t, loop, driver, "file", nbio.Open_At_Options{
				Access: nbio.OPEN_READ_WRITE, Create: create, Truncate: truncate,
			})
			if !testify.No_Error(t, err) {
				return
			}
			want := "secret"
			if truncate {
				want = ""
			}
			contents := sim_storage_read(t, loop, driver, file, 6)
			testify.Equal(t, want, contents, create, truncate)
			sim_close(t, loop, driver, file)
			nbio.IO_Deinit(loop)
		}
	}
}

// Access belongs to each descriptor, even when several descriptors name the same node.
func Test_Sim_Access(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	writer, err := sim_create(t, loop, driver, "file")
	if !testify.No_Error(t, err) {
		return
	}
	sim_storage_write(t, loop, driver, writer, []byte("base"))
	reader, err := sim_open(t, loop, driver, "file")
	if !testify.No_Error(t, err) {
		return
	}
	data, write_err := sim_write_at(t, loop, driver, reader, []byte("next"), 0)
	testify.Error(t, write_err)
	testify.Zero(t, data)
	sim_read_rejected(t, loop, driver, writer, []byte("keep"))
	data, write_err = sim_write_at(t, loop, driver, reader, nil, 5)
	testify.No_Error(t, write_err)
	testify.Zero(t, data)
	testify.Equal(t, "", sim_storage_read(t, loop, driver, writer, 0))
	testify.Equal(t, "base", sim_storage_read(t, loop, driver, reader, 4))
	sim_close(t, loop, driver, reader)
	sim_close(t, loop, driver, writer)
	nbio.IO_Deinit(loop)
}

// Truncated backing bytes must never reappear when a later write leaves a hole.
func Test_Sim_Holes(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	file, err := sim_create(t, loop, driver, "file")
	if !testify.No_Error(t, err) {
		return
	}
	sim_storage_write(t, loop, driver, file, []byte("secret"))
	sim_close(t, loop, driver, file)
	file, err = sim_open_options(t, loop, driver, "file", nbio.Open_At_Options{
		Access: nbio.OPEN_READ_WRITE, Truncate: true, Create: true,
	})
	if !testify.No_Error(t, err) {
		return
	}
	data, write_err := sim_write_at(t, loop, driver, file, nil, 5)
	testify.No_Error(t, write_err)
	testify.Zero(t, data)
	status, status_err := nbio.Storage_Status(loop.Storage, "file")
	testify.No_Error(t, status_err)
	testify.Zero(t, status.Size)
	data, write_err = sim_write_at(t, loop, driver, file, []byte("x"), 5)
	testify.No_Error(t, write_err)
	testify.Equal(t, 1, data)
	testify.Equal(t, "\x00\x00\x00\x00\x00x", sim_storage_read(t, loop, driver, file, 6))
	sim_close(t, loop, driver, file)
	nbio.IO_Deinit(loop)
}

// Test_Sim_Socket verify each transport requires all limits and transfers descriptor ownership
// only through its result.
func Test_Sim_Socket(t *testing.T) {
	socket_default_profile_assert(t)
	loop, driver, _ := sim_loop(0)
	socket, open_err := nbio.Network_Socket_TCP(
		loop.Network, nbio.FAMILY_IPV4, sim_tcp_options(),
	)
	if !testify.No_Error(t, open_err) {
		return
	}
	testify.True(t, sim_descriptor_open(loop))
	sim_close(t, loop, driver, socket)

	udp, udp_err := nbio.Network_Socket_UDP(
		loop.Network, nbio.FAMILY_IPV4, sim_udp_options(),
	)
	if !testify.No_Error(t, udp_err) {
		return
	}
	sim_close(t, loop, driver, udp)

	sim_socket_limits_rejected(t, loop, driver)
	nbio.IO_Deinit(loop)
}

// Test_Sim_Bind verify Bind gives a socket its requested local address and assigns a port when
// the request uses port zero.
func Test_Sim_Bind(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	socket, open_err := nbio.Network_Socket_TCP(
		loop.Network, nbio.FAMILY_IPV4, sim_tcp_options(),
	)
	if !testify.No_Error(t, open_err) {
		return
	}
	requested := nbio.Address_IPV4([]byte{127, 0, 0, 1}, 0)
	testify.No_Error(t, nbio.Network_Bind(loop.Network, socket, requested))
	bound := nbio.Address{IP: make([]byte, nbio.IPV6_ADDRESS_BYTES)}
	name_err := nbio.Network_Get_Socket_Name(loop.Network, socket, &bound)
	testify.No_Error(t, name_err)
	testify.Equal(t, requested.IP, bound.IP)
	testify.Not_Zero(t, bound.Port)
	sim_close(t, loop, driver, socket)
	nbio.IO_Deinit(loop)
}

// Test_Sim_Accept verify accept resolve exactly once, with distinct accepted socket, or with
// Deadline_Exceeded.
func Test_Sim_Accept(t *testing.T) {
	accepted_count := 0
	deadline_count := 0
	for seed := uint64(0); seed < 64; seed++ {
		accepted, outcome := sim_accept_once(t, seed, time.NANOSECOND)
		if outcome == nbio.Deadline_Exceeded {
			deadline_count++
			continue
		}
		testify.No_Error(t, outcome, seed)
		accepted_count++
		testify.Positive(t, accepted, seed)
	}
	testify.Positive(t, accepted_count)
	testify.Positive(t, deadline_count)
}

// Test_Sim_Connect verify success and refusal both keep caller-owned socket until explicit
// Close, and seed sweep reach both network outcomes.
func Test_Sim_Connect(t *testing.T) {
	snap.Expect(t,
		snap.Init(`open_before_close=true outcome=success`),
		sim_connect_lifecycle(t, 0),
	)
	snap.Expect(t,
		snap.Init(`open_before_close=true outcome=refused`),
		sim_connect_lifecycle(t, 4),
	)

	saw_success := false
	saw_refusal := false
	for seed := uint64(0); seed < 64; seed++ {
		outcome := sim_connect_lifecycle(t, seed)
		if test_text_has_suffix(outcome, "outcome=success") {
			saw_success = true
		}
		if test_text_has_suffix(outcome, "outcome=refused") {
			saw_refusal = true
		}
	}
	testify.True(t, saw_success)
	testify.True(t, saw_refusal)
}

// Test_Sim_Receive verify receive complete after modeled latency and report buffer length.
func Test_Sim_Receive(t *testing.T) {
	loop, driver, _ := sim_loop(1)
	socket, open_err := nbio.Network_Socket_TCP(
		loop.Network, nbio.FAMILY_IPV4, sim_tcp_options(),
	)
	if !testify.No_Error(t, open_err) {
		return
	}

	count := -1
	var completion nbio.Completion
	nbio.Network_Receive(loop.Network, &completion, socket, make([]byte, 64), SIM_DEADLINE,
		func(completed nbio.Completion_Handle) {
			count = completed.Data
		})

	nbio.Driver_Run_For(driver, 10*time.NANOSECOND)

	testify.Equal(t, 64, count)
	sim_close(t, loop, driver, socket)
	nbio.IO_Deinit(loop)
}

// Test_Sim_Send verify send complete after modeled latency and report buffer length.
func Test_Sim_Send(t *testing.T) {
	loop, driver, _ := sim_loop(1)
	socket, open_err := nbio.Network_Socket_TCP(
		loop.Network, nbio.FAMILY_IPV4, sim_tcp_options(),
	)
	if !testify.No_Error(t, open_err) {
		return
	}

	count := -1
	var completion nbio.Completion
	nbio.Network_Send(loop.Network, &completion, socket, make([]byte, 32), SIM_DEADLINE,
		func(completed nbio.Completion_Handle) {
			count = completed.Data
		})

	nbio.Driver_Run_For(driver, 10*time.NANOSECOND)

	testify.Equal(t, 32, count)
	sim_close(t, loop, driver, socket)
	nbio.IO_Deinit(loop)
}

// Test_Sim_Shutdown verify shutdown resolve armed and later socket operations through their
// normal callbacks, and leave descriptor ownership with caller.
func Test_Sim_Shutdown(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	socket, open_err := nbio.Network_Socket_TCP(
		loop.Network, nbio.FAMILY_IPV4, sim_tcp_options(),
	)
	if !testify.No_Error(t, open_err) {
		return
	}
	connected := false
	var connect_completion nbio.Completion
	nbio.Network_Connect(loop.Network, &connect_completion, socket,
		nbio.Address_IPV4([]byte{127, 0, 0, 1}, 8123), SIM_DEADLINE,
		func(completed nbio.Completion_Handle) {
			testify.No_Error(t, completed.Error)
			connected = true
		})
	nbio.Driver_Run_Until(driver, SIM_DEADLINE, func() (finished bool) { return connected })
	var receive_completion nbio.Completion
	var send_completion nbio.Completion
	receive_count := -1
	var send_err error
	nbio.Network_Receive(
		loop.Network, &receive_completion, socket, make([]byte, 8), SIM_DEADLINE,
		func(completed nbio.Completion_Handle) {
			testify.No_Error(t, completed.Error)
			receive_count = completed.Data
		})
	nbio.Network_Send(loop.Network, &send_completion, socket, []byte("hello"), SIM_DEADLINE,
		func(completed nbio.Completion_Handle) {
			send_err = completed.Error
		})
	testify.No_Error(t, nbio.Network_Shutdown(loop.Network, socket, nbio.SHUTDOWN_BOTH))
	nbio.Driver_Run_For(driver, 10*time.NANOSECOND)
	testify.Zero(t, receive_count)
	testify.Error_Is(t, send_err, nbio.Broken_Pipe)
	testify.True(t, sim_descriptor_open(loop))
	sim_close(t, loop, driver, socket)
	nbio.IO_Deinit(loop)
}

// Test_Sim_Close verify close reject descriptor an armed operation borrow, then
// shutdown-drain-close sequence complete and remove caller-owned descriptor.
func Test_Sim_Close(t *testing.T) {
	loop, driver, _ := sim_loop(0)

	socket, open_err := nbio.Network_Socket_TCP(
		loop.Network, nbio.FAMILY_IPV4, sim_tcp_options(),
	)
	if !testify.No_Error(t, open_err) {
		return
	}
	connected := false
	var connect_completion nbio.Completion
	nbio.Network_Connect(loop.Network, &connect_completion, socket,
		nbio.Address_IPV4([]byte{127, 0, 0, 1}, 1), SIM_DEADLINE,
		func(completed nbio.Completion_Handle) {
			testify.No_Error(t, completed.Error)
			connected = true
		})
	nbio.Driver_Run_Until(driver, SIM_DEADLINE, func() (finished bool) { return connected })
	received := false
	var receive_completion nbio.Completion
	nbio.Network_Receive(
		loop.Network, &receive_completion, socket, make([]byte, 8), SIM_DEADLINE,
		func(_ nbio.Completion_Handle) {
			received = true
		})

	var completion nbio.Completion
	testify.Panics(t, func() {
		nbio.IO_Close(loop, &completion, socket, func(_ nbio.Completion_Handle) {})
	})
	testify.No_Error(t, nbio.Network_Shutdown(loop.Network, socket, nbio.SHUTDOWN_BOTH))
	nbio.Driver_Run_For(driver, 10*time.NANOSECOND)
	testify.True(t, received)

	sim_close(t, loop, driver, socket)
	nbio.IO_Deinit(loop)
}

// Test_Sim_Open verify Open return fresh descriptor, synchronously.
func Test_Sim_Open(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	created, create_err := sim_create(t, loop, driver, "path")
	if !testify.No_Error(t, create_err) {
		return
	}
	file, err := sim_open(t, loop, driver, "path")
	if !testify.No_Error(t, err) {
		return
	}
	testify.Positive(t, file)
	_, absent_err := sim_open(t, loop, driver, "absent")
	testify.Error(t, absent_err)
	sim_close(t, loop, driver, created)
	sim_close(t, loop, driver, file)
	nbio.IO_Deinit(loop)
}

// Test_Sim_Create verify Create return fresh writable descriptor, synchronously.
func Test_Sim_Create(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	file, err := sim_create(t, loop, driver, "path")
	if !testify.No_Error(t, err) {
		return
	}
	testify.Positive(t, file)
	sim_close(t, loop, driver, file)
	nbio.IO_Deinit(loop)
}

// Test_Sim_Peer_Address verify only live connected socket has peer. File numbers and released
// sockets must never fabricate client identity.
func Test_Sim_Peer_Address(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	socket, _ := nbio.Network_Socket_TCP(loop.Network, nbio.FAMILY_IPV4, sim_tcp_options())
	unconnected := nbio.Address{IP: make([]byte, nbio.IPV6_ADDRESS_BYTES)}
	err := nbio.Network_Peer_Address(loop.Network, socket, &unconnected)
	testify.No_Error(t, err)
	testify.Zero(t, unconnected.Family)
	testify.Zero(t, len(unconnected.IP))
	testify.Zero(t, unconnected.Port)
	connected := false
	var completion nbio.Completion
	nbio.Network_Connect(
		loop.Network, &completion, socket,
		nbio.Address_IPV4([]byte{127, 0, 0, 1}, 8123),
		SIM_DEADLINE, func(completed nbio.Completion_Handle) {
			testify.No_Error(t, completed.Error)
			connected = true
		},
	)
	nbio.Driver_Run_Until(driver, SIM_DEADLINE, func() (finished bool) { return connected })
	address := nbio.Address{IP: make([]byte, nbio.IPV6_ADDRESS_BYTES)}
	err = nbio.Network_Peer_Address(loop.Network, socket, &address)
	testify.No_Error(t, err)
	testify.Equal(t,
		nbio.Address_IPV4([]byte{127, 0, 0, 1}, 8123), address)
	unknown := nbio.Address{IP: make([]byte, nbio.IPV6_ADDRESS_BYTES)}
	unknown_err := nbio.Network_Peer_Address(loop.Network, nbio.File(65535), &unknown)
	testify.No_Error(t, unknown_err)
	testify.Zero(t, unknown.Family)
	testify.Zero(t, len(unknown.IP))
	testify.Zero(t, unknown.Port)
	sim_close(t, loop, driver, socket)
	released := nbio.Address{IP: make([]byte, nbio.IPV6_ADDRESS_BYTES)}
	released_err := nbio.Network_Peer_Address(loop.Network, socket, &released)
	testify.No_Error(t, released_err)
	testify.Zero(t, released.Family)
	testify.Zero(t, len(released.IP))
	testify.Zero(t, released.Port)
	nbio.IO_Deinit(loop)
}

// Test_Sim_Status verify Status report existence, kind, and size, and report absent path as
// not-exists with nil error, not as failure.
func Test_Sim_Status(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	made := false
	var mkdir nbio.Completion
	nbio.Storage_Mkdir_At(loop.Storage, &mkdir, nbio.DIRECTORY_CURRENT, "/dir", 0o755,
		func(completed nbio.Completion_Handle) {
			testify.No_Error(t, completed.Error)
			made = true
		})
	nbio.Driver_Run_Until(driver, SIM_DEADLINE, func() (finished bool) { return made })
	file, create_err := sim_create(t, loop, driver, "/dir/file")
	if !testify.No_Error(t, create_err) {
		return
	}
	// Write known bytes, thus file Size has known expected value.
	content := []byte("hello world")
	written := false
	var write nbio.Completion
	nbio.Storage_Write(loop.Storage, &write, file, content, 0, SIM_DEADLINE,
		func(_ nbio.Completion_Handle) {
			written = true
		})
	nbio.Driver_Run_Until(driver, SIM_DEADLINE, func() (finished bool) { return written })
	directory, _ := nbio.Storage_Status(loop.Storage, "/dir")
	testify.True(t, directory.Exists)
	testify.True(t, nbio.File_Mode_Is_Directory(directory.Mode))
	testify.False(t, nbio.File_Mode_Is_Regular(directory.Mode))
	// Stated 0o755 minus the modeled umask, thus Status reports stored mode, not requested one.
	testify.Equal(t,
		nbio.File_Mode(0o755&^nbio.SIM_UMASK), directory.Mode&nbio.FILE_MODE_PERMISSIONS)
	testify.Zero(t, directory.Size)
	sim_status_regular(t, loop, content)
	var link_target [nbio.SIM_PATH_TEXT_BYTES_MAXIMUM]byte
	_, read_link_err := nbio.Storage_Read_Link(
		loop.Storage, "/dir/file", link_target[:],
	)
	testify.Error_Is(t, read_link_err, nbio.Not_Symbolic_Link)
	sim_close(t, loop, driver, file)
	nbio.IO_Deinit(loop)
}

// Test_Sim_Watch_Signal verify one-shot signal watch: seed decide whether signal or deadline win,
// callback fire exactly one time either way, and expired watch report SIGNAL_EXPIRED.
func Test_Sim_Watch_Signal(t *testing.T) {
	signal_count := 0
	deadline_count := 0
	for seed := uint64(0); seed < 64; seed++ {
		loop, driver, _ := sim_loop(seed)
		got := nbio.SIGNAL_EXPIRED
		callback_count := 0
		var operation_err error
		var completion nbio.Completion
		nbio.IO_Watch_Signal(
			loop, &completion, nbio.SIGNAL_TERMINATE, time.NANOSECOND, func(
				_ *nbio.Completion, signal nbio.Signal, err error,
			) {
				callback_count++
				got = signal
				operation_err = err
			})
		completed, drive_err := nbio.Driver_Run_Until(driver,
			16*time.NANOSECOND, func() (finished bool) { return callback_count > 0 })
		testify.No_Error(t, drive_err, seed)
		testify.True(t, completed, seed)
		testify.Equal(t, 1, callback_count, seed)
		if operation_err == nbio.Deadline_Exceeded {
			deadline_count++
			testify.Equal(t, nbio.SIGNAL_EXPIRED, got, seed)
		} else {
			testify.No_Error(t, operation_err, seed)
			signal_count++
			testify.Equal(t, nbio.SIGNAL_TERMINATE, got, seed)
		}
		nbio.Driver_Run_For(driver, 16*time.NANOSECOND)
		testify.Equal(t, 1, callback_count, seed)
		testify.Nil(t, completion.Backend, seed)
		nbio.IO_Deinit(loop)
	}
	testify.Positive(t, signal_count)
	testify.Positive(t, deadline_count)
}

// Test_Sim_Spawn verify spawn deliver one seed-drawn result on loop and capture no output.
func Test_Sim_Spawn(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	callback_count := 0
	result := nbio.Process_Result{}
	var completion nbio.Completion
	nbio.IO_Spawn(loop, &completion, nbio.Process_Request{Path: "echo"}, SIM_DEADLINE,
		func(_ *nbio.Completion, spawned nbio.Process_Result, _ error) {
			callback_count++
			result = spawned
		})
	nbio.Driver_Run_For(driver, 16*time.NANOSECOND)
	testify.Equal(t, 1, callback_count)
	testify.Nil(t, completion.Backend)
	testify.Nil(t, result.Output)
	testify.Nil(t, result.Error_Output)
	nbio.IO_Deinit(loop)
}

// Test_Sim_Deinit verify surface own leak check: Deinit reject run that still hold descriptor,
// and accept same run once caller closed it.
func Test_Sim_Deinit(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	file, create_err := sim_create(t, loop, driver, "/deinit")
	if !testify.No_Error(t, create_err) {
		return
	}
	testify.True(t, sim_descriptor_open(loop))
	sim_close(t, loop, driver, file)
	nbio.IO_Deinit(loop)
}

// Test_Sim_Halves verify IO carry one backend pointer per half and nothing above them: flat
// operations read the storage half, thus the constructor hand out halves that agree, and
// Close on a sim descriptor land on the same backend that opened it.
func Test_Sim_Halves(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	testify.Not_Nil(t, loop.Storage.State)
	testify.Equal(t, loop.Network.State, loop.Storage.State)
	file, create_err := sim_create(t, loop, driver, "/halves")
	if !testify.No_Error(t, create_err) {
		return
	}
	sim_close(t, loop, driver, file)
	testify.False(t, sim_descriptor_open(loop))
}

// Test_File_Mode_Portable verify platform-word conversion round trip every representable kind
// across every non-kind bit, and verify unsupported kind nibble decode to no kind at all.
func Test_File_Mode_Portable(t *testing.T) {
	for _, kind := range posix_file_mode_kinds() {
		for high := uint16(0); high <= POSIX_FILE_MODE_HIGH_MAXIMUM; high++ {
			raw := kind | high
			decoded := nbio.File_Mode(nbio.File_Mode_From_POSIX(raw))
			if !testify.Equal(t, raw, nbio.File_Mode_To_POSIX(decoded)) {
				return
			}
		}
	}
	// Kind nibble outside the supported set carries no kind through, thus a hostile stat word
	// cannot smuggle one in, and re-encoding names a regular file.
	unsupported := nbio.File_Mode(nbio.File_Mode_From_POSIX(
		3<<nbio.POSIX_FILE_TYPE_SHIFT | 0o644,
	))
	testify.True(t, nbio.File_Mode_Is_Regular(unsupported))
	testify.Equal(t, nbio.POSIX_FILE_MODE_REGULAR|0o644,
		nbio.File_Mode_To_POSIX(unsupported))
	// Every portable bit at once names no single kind, thus encode falls back to regular while
	// keeping each permission bit the platform word can hold.
	testify.Equal(t, nbio.POSIX_FILE_MODE_REGULAR|POSIX_FILE_MODE_HIGH_MAXIMUM,
		nbio.File_Mode_To_POSIX(nbio.FILE_MODE_VALID))
}

// Test_File_Mode_Permissions verify creation operand carry no kind nibble, and verify open and
// directory make accept every value of the permission domain and store it less the umask.
func Test_File_Mode_Permissions(t *testing.T) {
	// Encoded operand must never name a kind: openat and mkdirat derive kind themselves.
	for _, permissions := range file_permissions_domain() {
		raw := nbio.File_Permissions_To_POSIX(permissions)
		testify.Zero(t, raw&nbio.POSIX_FILE_TYPE_MASK)
	}
	loop, driver, _ := sim_loop(0)
	for index, permissions := range file_permissions_domain() {
		path := "/permission" + string(rune('0'+index))
		file, create_err := sim_open_options(t, loop, driver, path, nbio.Open_At_Options{
			Access: nbio.OPEN_WRITE_ONLY, Create: true, Permissions: permissions,
		})
		if !testify.No_Error(t, create_err) {
			return
		}
		sim_close(t, loop, driver, file)
		status, _ := nbio.Storage_Status(loop.Storage, path)
		testify.True(t, status.Exists)
		testify.True(t, nbio.File_Mode_Is_Regular(status.Mode))
		testify.Equal(t, nbio.File_Mode(permissions&^nbio.SIM_UMASK), status.Mode)
	}
	sim_mkdir_permissions(t, loop, driver)
	nbio.IO_Deinit(loop)
}

// Test_File_Mode_Status verify Status report stored mode for every seed, and report absent path
// as the zero status rather than a kind a caller could read out of a zero mode.
func Test_File_Mode_Status(t *testing.T) {
	for seed := uint64(0); seed < 16; seed++ {
		loop, driver, _ := sim_loop(seed)
		file, create_err := sim_open_options(t, loop, driver, "/stored",
			nbio.Open_At_Options{
				Access: nbio.OPEN_WRITE_ONLY, Create: true, Permissions: 0o777,
			})
		if !testify.No_Error(t, create_err) {
			return
		}
		sim_close(t, loop, driver, file)
		status, status_err := nbio.Storage_Status(loop.Storage, "/stored")
		testify.No_Error(t, status_err)
		testify.Equal(t, nbio.File_Mode(0o777&^nbio.SIM_UMASK), status.Mode)
		absent, absent_err := nbio.Storage_Status(loop.Storage, "/absent")
		testify.No_Error(t, absent_err)
		testify.Equal(t, nbio.File_Status{}, absent)
		nbio.IO_Deinit(loop)
	}
}

// Test_File_Mode_Symbolic_Link verify seeded generation make links, Status report link kind
// without following it, Read_Link return the stored target, and no-follow open refuse one.
func Test_File_Mode_Symbolic_Link(t *testing.T) {
	seen := false
	for seed := uint64(0); seed < 64; seed++ {
		loop, driver, nodes := sim_loop_nodes(seed)
		path, found := sim_first_link_path(nodes)
		if found {
			seen = true
			t.Run(fmt.Sprintf("seed_%d", seed), func(t *testing.T) {
				sim_assert_link(t, loop, driver, nodes, path)
			})
		}
		nbio.IO_Deinit(loop)
	}
	// A sweep that produced no link would prove nothing, thus absence is itself a failure.
	testify.True(t, seen)
}

// Test_Address_Parse verify hand-written literal parse: IPv4, IPv6 in every accepted shape, and
// every rejection the net/netip call used to give for free.
func Test_Address_Parse(t *testing.T) {
	for _, accepted := range address_parse_accepted() {
		storage := make([]byte, nbio.IPV6_ADDRESS_BYTES)
		address := nbio.Address{IP: storage}
		err := nbio.Address_Parse(&address, accepted.Host, 8123)
		if !testify.No_Error(t, err, accepted.Host) {
			continue
		}
		testify.Equal(t, accepted.Family, address.Family, accepted.Host)
		testify.Equal(t, uint16(8123), address.Port, accepted.Host)
		expected_ip := accepted.IP
		if accepted.Family == nbio.FAMILY_IPV4 {
			expected_ip = expected_ip[:nbio.IPV4_ADDRESS_BYTES]
		}
		testify.Equal(t, expected_ip, address.IP, accepted.Host)
		if accepted.Family == nbio.FAMILY_IPV6 {
			expected := nbio.Address_IPV6(accepted.IP, 8123)
			testify.Equal(t, expected, address, accepted.Host)
		}
	}
	for _, rejected := range address_parse_rejected() {
		storage := make([]byte, nbio.IPV6_ADDRESS_BYTES)
		address := nbio.Address{IP: storage}
		err := nbio.Address_Parse(&address, rejected, 8123)
		testify.Error(t, err, rejected)
	}
	storage := make([]byte, nbio.IPV6_ADDRESS_BYTES)
	address := nbio.Address{IP: storage}
	negative_err := nbio.Address_Parse(&address, "127.0.0.1", -1)
	testify.Error(t, negative_err)
	overflow_err := nbio.Address_Parse(&address, "127.0.0.1", 65536)
	testify.Error(t, overflow_err)
	oversized := string(make([]byte, nbio.ADDRESS_TEXT_BYTES_MAXIMUM+1))
	oversized_err := nbio.Address_Parse(&address, oversized, 8123)
	testify.Error(t, oversized_err)
	short := nbio.Address{IP: storage[: nbio.IPV6_ADDRESS_BYTES-1 : nbio.IPV6_ADDRESS_BYTES-1]}
	storage_err := nbio.Address_Parse(&short, "::1", 8123)
	testify.Error(t, storage_err)
	testify.Error(t, nbio.Address_Parse(nil, "::1", 8123))
}

// Test_Stream_Callback verifies callback ownership remains with each concrete stream.
func Test_Stream_Callback(t *testing.T) { test_stream_callback(t) }

// Test_Stream_Read verifies cursor reads preserve the stream contract.
func Test_Stream_Read(t *testing.T) { test_stream_read(t) }

// Test_Stream_Write verifies bounded cursor writes preserve the stream contract.
func Test_Stream_Write(t *testing.T) { test_stream_write(t) }

// Test_Stream_Read_At verifies offset reads do not move the cursor.
func Test_Stream_Read_At(t *testing.T) { test_stream_read_at(t) }

// Test_Stream_Write_At verifies offset writes do not move the cursor.
func Test_Stream_Write_At(t *testing.T) { test_stream_write_at(t) }

// Test_Stream_Seek verifies every supported origin and invalid bound.
func Test_Stream_Seek(t *testing.T) { test_stream_seek(t) }

// Test_Stream_Size verifies size does not depend on cursor position.
func Test_Stream_Size(t *testing.T) { test_stream_size(t) }

// Test_Stream_Query verifies capability inspection uses completion data.
func Test_Stream_Query(t *testing.T) { test_stream_query(t) }

// Test_Stream_Flush verifies immediate streams retire flush inline.
func Test_Stream_Flush(t *testing.T) { test_stream_flush(t) }

// Test_Stream_Close verifies close is idempotent and terminal.
func Test_Stream_Close(t *testing.T) { test_stream_close(t) }

// Test_Stream_Destroy verifies destroy leaves memory stream terminal.
func Test_Stream_Destroy(t *testing.T) { test_stream_destroy(t) }

// Test_Stream_Errors verifies invalid operations do not fabricate bytes.
func Test_Stream_Errors(t *testing.T) { test_stream_errors(t) }

// Test_Stream_Memory verifies caller storage is the complete memory budget.
func Test_Stream_Memory(t *testing.T) { test_stream_memory(t) }

// Test_Stream_Discard verifies discarded bytes consume no storage.
func Test_Stream_Discard(t *testing.T) { test_stream_discard(t) }

// Test_Stream_Limit verifies forwarding cannot exceed its remaining budget.
func Test_Stream_Limit(t *testing.T) { test_stream_limit(t) }

// Test_Stream_Count verifies tally changes only after inner completion.
func Test_Stream_Count(t *testing.T) { test_stream_count(t) }

// Test_Stream_Tee verifies the smaller destination count wins.
func Test_Stream_Tee(t *testing.T) { test_stream_tee(t) }

// Test_Stream_Composition verifies dependent callbacks preserve composition order.
func Test_Stream_Composition(t *testing.T) { test_stream_composition(t) }

// Test_Timeline_Timeout check timeout fire exactly when virtual clock reach its deadline. No
// real wait. Fully deterministic. Zero duration is a caller mistake and panic at the guard.
func Test_Timeline_Timeout(t *testing.T) {
	loop, driver, host := sim_timeline()

	fired_at := time.Monotonic_Moment(-1)
	var completion nbio.Completion
	nbio.Timeline_Timeout(
		loop, &completion, 5*time.NANOSECOND, func(completed nbio.Completion_Handle) {
			testify.No_Error(t, completed.Error)
			fired_at = time.Clock_Now_Monotonic(host)
		},
	)

	nbio.Driver_Run_For(driver, 10*time.NANOSECOND)

	testify.Equal(t, time.Monotonic_Moment(5), fired_at)
	testify.Panics(t, func() {
		nbio.Timeline_Timeout(loop, &completion, 0, allocation_callback)
	})
}

// Long-lived completion must not keep callback closure, or closure keep operation buffer live.
// Clear before call so callback can arm same completion again without new callback being erased.
func Test_Timeline_Callback_Released(t *testing.T) {
	loop, driver, _ := sim_timeline()

	var once nbio.Completion
	nbio.Timeline_Timeout(loop, &once, time.NANOSECOND, func(_ nbio.Completion_Handle) {})
	nbio.Driver_Run_For(driver, 2*time.NANOSECOND)
	testify.Nil(t, once.Callback)

	var repeated nbio.Completion
	fired := 0
	var arm func()
	arm = func() {
		nbio.Timeline_Timeout(loop, &repeated, time.NANOSECOND, func(
			_ nbio.Completion_Handle,
		) {
			fired++
			if fired < 3 {
				arm()
			}
		})
	}
	arm()
	nbio.Driver_Run_For(driver, 4*time.NANOSECOND)
	testify.Equal(t, 3, fired)
	testify.Nil(t, repeated.Callback)
}

// Test_Timeline_Event check Event primitive retire its listener before callback run. Same
// completion can then arm again for later trigger. Every misuse of a handle panic.
func Test_Timeline_Event(t *testing.T) {
	loop, driver, _ := sim_timeline()
	event, open_err := nbio.Timeline_Open_Event(loop)
	if !testify.No_Error(t, open_err) {
		return
	}
	fired := 0
	var completion nbio.Completion
	callback := func(_ nbio.Completion_Handle) { fired++ }
	nbio.Timeline_Event_Listen(loop, event, &completion, callback)
	nbio.Timeline_Event_Trigger(loop, event, &completion)
	nbio.Driver_Run(driver)
	testify.Equal(t, 1, fired)
	nbio.Timeline_Event_Listen(loop, event, &completion, callback)
	nbio.Timeline_Event_Trigger(loop, event, &completion)
	nbio.Driver_Run(driver)
	testify.Equal(t, 2, fired)
	nbio.Timeline_Close_Event(loop, event)
	verify_event_misuse_panics(t)
}

// Test_Timeline_Capacity checks a full caller store rejects new work before it changes
// completion or event lifecycle state.
func Test_Timeline_Capacity(t *testing.T) {
	loop, driver, _ := sim_timeline()
	slots := [SIM_TIMELINE_CAPACITY]nbio.Completion{}
	sim_timeline_fill(loop, slots[:])
	var overflow nbio.Completion
	testify.Panics(t, func() {
		nbio.Timeline_Submit(loop, &overflow, 2*time.NANOSECOND, allocation_callback)
	})
	testify.False(t, overflow.Armed)

	event, open_err := nbio.Timeline_Open_Event(loop)
	if !testify.No_Error(t, open_err) {
		return
	}
	_, capacity_err := nbio.Timeline_Open_Event(loop)
	testify.Error_Is(t, capacity_err, nbio.Virtual_Event_Capacity_Exceeded)
	fired := false
	var listener nbio.Completion
	nbio.Driver_Run_For(driver, 3*time.NANOSECOND)
	sim_timeline_fill(loop, slots[:])
	nbio.Timeline_Event_Listen(loop, event, &listener, func(_ nbio.Completion_Handle) {
		fired = true
	})
	testify.Panics(t, func() {
		nbio.Timeline_Event_Trigger(loop, event, &listener)
	})
	nbio.Driver_Run_For(driver, 3*time.NANOSECOND)
	nbio.Timeline_Event_Trigger(loop, event, &listener)
	nbio.Driver_Run(driver)
	if !testify.True(t, fired) {
		return
	}
	nbio.Timeline_Close_Event(loop, event)
	verify_pending_trigger_capacity(t)
}

// Test_Timeline_Run_Until check driver pump loop until predicate report true, and report
// completed. Also check predicate that never trip return false at deadline, and negative
// timeout panic: unbounded pump have no cap, thus stalled operation spin as long as process
// live.
func Test_Timeline_Run_Until(t *testing.T) {
	loop, driver, _ := sim_timeline()

	done := false
	var completion nbio.Completion
	nbio.Timeline_Timeout(loop, &completion, 3*time.NANOSECOND, func(_ nbio.Completion_Handle) {
		done = true
	})

	completed, drive_err := nbio.Driver_Run_Until(driver,
		SIM_DEADLINE, func() (finished bool) { return done },
	)
	testify.No_Error(t, drive_err)
	testify.True(t, completed)
	testify.True(t, done)

	completed, drive_err = nbio.Driver_Run_Until(driver,
		SIM_DEADLINE, func() (finished bool) { return false },
	)
	testify.No_Error(t, drive_err)
	testify.False(t, completed)
	testify.Panics(t, func() {
		nbio.Driver_Run_Until(driver, -1*time.NANOSECOND,
			func() (finished bool) { return false })
	})
}

// Test_Timeline_Reuse check submit of completion still in flight panic. Reused completion thus
// fail loud, never corrupt queue.
func Test_Timeline_Reuse(t *testing.T) {
	loop, _, _ := sim_timeline()
	var completion nbio.Completion
	nbio.Timeline_Timeout(
		loop, &completion, 5*time.NANOSECOND, func(_ nbio.Completion_Handle) {},
	)
	testify.Panics(t, func() {
		nbio.Timeline_Timeout(loop, &completion, 5*time.NANOSECOND,
			func(_ nbio.Completion_Handle) {})
	})
}

// Test_Timeline_Copy check submit of by-value copy panic. Copied completion thus fail loud,
// never split view of loop from view of caller. Fire original first, thus copy is unarmed.
// That isolate copy guard from reuse guard.
func Test_Timeline_Copy(t *testing.T) {
	loop, driver, _ := sim_timeline()
	var completion nbio.Completion
	nbio.Timeline_Timeout(
		loop, &completion, 5*time.NANOSECOND, func(_ nbio.Completion_Handle) {},
	)
	nbio.Driver_Run_For(driver, 10*time.NANOSECOND)
	duplicate := completion
	testify.Panics(t, func() {
		nbio.Timeline_Timeout(loop, &duplicate, 5*time.NANOSECOND,
			func(_ nbio.Completion_Handle) {})
	})
}

// Test_Timeline_Reentrancy check drive of loop from inside completion callback panic.
// Re-entrant Run* thus fail loud, never corrupt queue mid-drain.
func Test_Timeline_Reentrancy(t *testing.T) {
	loop, driver, _ := sim_timeline()
	var completion nbio.Completion
	nbio.Timeline_Timeout(loop, &completion, 5*time.NANOSECOND, func(_ nbio.Completion_Handle) {
		nbio.Driver_Run(driver)
	})
	testify.Panics(t, func() {
		nbio.Driver_Run_For(driver, 10*time.NANOSECOND)
	})
}

// Test_Timeline_Clocks check every view read the one counter, differ in realtime by skew
// alone, and that the epoch and each skew come from the seed and nothing else. Resolution
// below one grain panic, and a one-year grain reach the uptime bound on the first tick.
func Test_Timeline_Clocks(t *testing.T) {
	memory := &sim_memory{}
	_, driver := nbio.New_Simulated_IO(
		&memory.Sim, 7, time.NANOSECOND, sim_memory_view(memory),
	)
	first := nbio.Sim_Clock_To_Clock(&memory.Sim.Clocks[0])
	second := nbio.Sim_Clock_To_Clock(&memory.Sim.Clocks[1])
	nbio.Driver_Run_For(driver, 16*time.NANOSECOND)
	testify.Equal(t, time.Clock_Now_Monotonic(first), time.Clock_Now_Monotonic(second))
	testify.Equal(t, time.Monotonic_Moment(16), time.Clock_Now_Monotonic(first))
	epoch := memory.Sim.Timeline.Epoch
	skew := time.Offset_Read(memory.Sim.Clocks[0].Skew, 16)
	testify.Equal(t, epoch+16-time.Moment(skew), time.Clock_Now_Realtime(first))
	testify.True(t, epoch >= 0)
	testify.True(t, epoch < time.Moment(nbio.SIM_EPOCH_SECONDS)*time.Moment(time.SECOND))

	replay := &sim_memory{}
	nbio.New_Simulated_IO(&replay.Sim, 7, time.NANOSECOND, sim_memory_view(replay))
	testify.Equal(t, memory.Sim.Timeline.Epoch, replay.Sim.Timeline.Epoch)
	testify.Equal(t, memory.Sim.Clocks[0].Skew, replay.Sim.Clocks[0].Skew)
	testify.Equal(t, memory.Sim.Clocks[1].Skew, replay.Sim.Clocks[1].Skew)

	other := &sim_memory{}
	nbio.New_Simulated_IO(&other.Sim, 8, time.NANOSECOND, sim_memory_view(other))
	testify.True(t, memory.Sim.Timeline.Epoch != other.Sim.Timeline.Epoch)

	verify_resolution_domains(t)
}

// Test_Allocation proves constructor, Stream, and simulated IO boundaries stay off heap.
func Test_Allocation(t *testing.T) {
	nbio_value_api_heap_allocation(t)
	nbio_constructors_heap_allocation(t)
	nbio_stream_api_heap_allocation(t)
	nbio_stream_error_heap_allocation(t)
	sim_api_heap_allocation(t)
	sim_operation_exhaustion_heap_allocation(t)
	verify_timeline_submission_allocations(t)
	verify_timeline_event_allocations(t)
	verify_driver_allocations(t)
}

// Exact IP storage prevents malformed slice bounds from escaping into socket encoding.
func Test_Address_Constructors_Reject_Invalid_Storage(t *testing.T) {
	testify.Panics(t, func() {
		nbio.Address_IPV4(make([]byte, nbio.IPV4_ADDRESS_BYTES-1), 0)
	})
	testify.Panics(t, func() {
		nbio.Address_IPV6(make([]byte, nbio.IPV6_ADDRESS_BYTES-1), 0)
	})
}

func verify_timeline_submission_allocations(t *testing.T) {
	t.Run("Callback", func(t *testing.T) {
		completion := nbio.Completion{}
		callback := nbio.Callback(allocation_callback)
		testify.Zero_Allocation(t, func() { callback(&completion) })
		testify.Positive(t, completion.Data)
	})
	t.Run("Submit", func(t *testing.T) {
		loop, driver, _ := sim_timeline()
		completion := nbio.Completion{}
		var run_err error
		testify.Zero_Allocation(t, func() {
			nbio.Timeline_Submit(loop, &completion, 0, allocation_callback)
			run_err = nbio.Driver_Run(driver)
		})
		testify.No_Error(t, run_err)
	})
	t.Run("Timeout", func(t *testing.T) {
		loop, driver, _ := sim_timeline()
		completion := nbio.Completion{}
		var run_err error
		testify.Zero_Allocation(t, func() {
			nbio.Timeline_Timeout(loop, &completion, 1, allocation_callback)
			run_err = nbio.Driver_Run_For(driver, 2)
		})
		testify.No_Error(t, run_err)
	})
}

func verify_timeline_event_allocations(t *testing.T) {
	t.Run("Listen_Trigger", func(t *testing.T) {
		loop, driver, _ := sim_timeline()
		event, err := nbio.Timeline_Open_Event(loop)
		testify.No_Error(t, err)
		completion := nbio.Completion{}
		var run_err error
		testify.Zero_Allocation(t, func() {
			nbio.Timeline_Event_Listen(loop, event, &completion, allocation_callback)
			nbio.Timeline_Event_Trigger(loop, event, &completion)
			run_err = nbio.Driver_Run(driver)
		})
		testify.No_Error(t, run_err)
		nbio.Timeline_Close_Event(loop, event)
	})
	t.Run("Open_Close", func(t *testing.T) {
		loop, _, _ := sim_timeline()
		var opened nbio.Event
		var open_err error
		testify.Zero_Allocation(t, func() {
			opened, open_err = nbio.Timeline_Open_Event(loop)
			nbio.Timeline_Close_Event(loop, opened)
		})
		testify.No_Error(t, open_err)
	})
}

func verify_driver_allocations(t *testing.T) {
	t.Run("Run", func(t *testing.T) {
		_, driver, _ := sim_timeline()
		var run_err error
		testify.Zero_Allocation(t, func() { run_err = nbio.Driver_Run(driver) })
		testify.No_Error(t, run_err)
	})
	t.Run("Run_For", func(t *testing.T) {
		_, driver, _ := sim_timeline()
		var run_err error
		testify.Zero_Allocation(t, func() { run_err = nbio.Driver_Run_For(driver, 1) })
		testify.No_Error(t, run_err)
	})
	t.Run("Run_Until", func(t *testing.T) {
		loop, driver, _ := sim_timeline()
		completion := nbio.Completion{}
		done := func() (finished bool) { return !completion.Armed }
		var completed bool
		var run_err error
		testify.Zero_Allocation(t, func() {
			nbio.Timeline_Timeout(loop, &completion, 1, allocation_callback)
			completed, run_err = nbio.Driver_Run_Until(driver, 2, done)
		})
		testify.True(t, completed)
		testify.No_Error(t, run_err)
	})
	t.Run("Deinit", func(t *testing.T) {
		_, driver, _ := sim_timeline()
		testify.Zero_Allocation(t, func() { nbio.Driver_Deinit(driver) })
	})
}

func allocation_callback(completion nbio.Completion_Handle) {
	completion.Data++
}

// Each handle misuse fail at its own guard, before any listener or queue state change.
func verify_event_misuse_panics(t *testing.T) {
	t.Helper()
	loop, _, _ := sim_timeline()
	event, open_err := nbio.Timeline_Open_Event(loop)
	if !testify.No_Error(t, open_err) {
		return
	}
	var listener nbio.Completion
	var stranger nbio.Completion
	callback := func(_ nbio.Completion_Handle) {}
	testify.Panics(t, func() { nbio.Timeline_Event_Trigger(loop, 0, &listener) })
	testify.Panics(t, func() {
		nbio.Timeline_Event_Trigger(loop, nbio.Event(SIM_EVENT_CAPACITY+1), &listener)
	})
	nbio.Timeline_Event_Listen(loop, event, &listener, callback)
	testify.Panics(t, func() { nbio.Timeline_Event_Listen(loop, event, &stranger, callback) })
	testify.Panics(t, func() { nbio.Timeline_Event_Trigger(loop, event, &stranger) })
	testify.Panics(t, func() { nbio.Timeline_Close_Event(loop, event) })
	closed_loop, _, _ := sim_timeline()
	closed, closed_err := nbio.Timeline_Open_Event(closed_loop)
	if !testify.No_Error(t, closed_err) {
		return
	}
	nbio.Timeline_Close_Event(closed_loop, closed)
	testify.Panics(t, func() { nbio.Timeline_Event_Trigger(closed_loop, closed, &stranger) })
}

// Fill every queue slot with a two-grain timer, thus the next enqueue trip the capacity guard.

func sim_timeline_fill(loop nbio.Timeline, slots []nbio.Completion) {
	for index := range slots {
		nbio.Timeline_Submit(loop, &slots[index], 2*time.NANOSECOND, allocation_callback)
	}
}

// A trigger can arrive before its listener, so that deferred enqueue needs the same bound.
func verify_pending_trigger_capacity(t *testing.T) {
	t.Helper()
	loop, driver, _ := sim_timeline()
	slots := [SIM_TIMELINE_CAPACITY]nbio.Completion{}
	sim_timeline_fill(loop, slots[:])
	event, open_err := nbio.Timeline_Open_Event(loop)
	if !testify.No_Error(t, open_err) {
		return
	}
	var listener nbio.Completion
	nbio.Timeline_Event_Trigger(loop, event, &listener)
	fired := false
	callback := func(_ nbio.Completion_Handle) { fired = true }
	testify.Panics(t, func() {
		nbio.Timeline_Event_Listen(loop, event, &listener, callback)
	})
	testify.False(t, listener.Armed)
	nbio.Driver_Run_For(driver, 3*time.NANOSECOND)
	nbio.Timeline_Event_Listen(loop, event, &listener, callback)
	nbio.Driver_Run(driver)
	if !testify.True(t, fired) {
		return
	}
	nbio.Timeline_Close_Event(loop, event)
}

// Hit the resolution domain at each sentinel the invariant framework expand. One-year grain
// reach the uptime bound on the first tick, thus the sweep cost one tick, not a year of them.
func verify_resolution_domains(t *testing.T) {
	t.Helper()
	for _, value := range [...]int64{
		bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM, 0, 1, 2, -1,
	} {
		memory := &sim_memory{}
		construct := func() {
			nbio.New_Simulated_IO(
				&memory.Sim, 0, time.Duration(value), sim_memory_view(memory),
			)
		}
		if value > 0 {
			testify.Not_Panics(t, construct)
			continue
		}
		testify.Panics(t, construct)
	}
	bound := &sim_memory{}
	one_year_grain := time.Duration(time.MONOTONIC_MOMENT_MAXIMUM)
	_, driver := nbio.New_Simulated_IO(&bound.Sim, 0, one_year_grain, sim_memory_view(bound))
	nbio.Driver_Run(driver)
	one_year := nbio.Sim_Clock_To_Clock(&bound.Sim.Clocks[0])
	testify.Equal(t, time.MONOTONIC_MOMENT_MAXIMUM, time.Clock_Now_Monotonic(one_year))
}

// Test_Callback_Result keeps operation meaning outside the shared completion: Data is opaque to
// the timeline, while each operation interprets it as its descriptor, count, or status.
func Test_Callback_Result(t *testing.T) {
	called := false
	completion := nbio.Completion{Data: 42, Error: nbio.Deadline_Exceeded}
	callback := nbio.Callback(func(got nbio.Completion_Handle) {
		testify.True(t, got == &completion)
		testify.Equal(t, 42, got.Data)
		testify.Error_Is(t, got.Error, nbio.Deadline_Exceeded)
		called = true
	})
	callback(&completion)
	testify.True(t, called)
}

func nbio_value_api_heap_allocation(t *testing.T) {
	var modes nbio.Stream_Mode_Set
	var present bool
	completion := nbio.Completion{}
	callback := nbio.Stream_Callback{
		Callback:  stream_allocation_callback,
		Procedure: stream_callback_allocation_procedure,
	}
	t.Run("Mode_Set_Add", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			modes = nbio.Mode_Set_Add(modes, nbio.STREAM_MODE_READ)
		})
	})
	t.Run("Mode_Set_Has", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			present = nbio.Mode_Set_Has(modes, nbio.STREAM_MODE_READ)
		})
		testify.True(t, present)
	})
	t.Run("Stream_Callback_Call", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			nbio.Stream_Callback_Call(callback, &completion)
		})
	})
}

func stream_callback_allocation_procedure(
	_ nbio.State, _ nbio.Stream_Callback_Data, callback nbio.Callback,
	completion nbio.Completion_Handle,
) {
	callback(completion)
}

// Every constructor belongs in allocation proof because one hidden closure or map makes every
// operation after it inherit heap ownership.
func nbio_constructors_heap_allocation(t *testing.T) {
	nbio_address_constructors_heap_allocation(t)

	var loop nbio.IO
	memory := &sim_memory{}
	t.Run("New_Simulated_IO", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			loop, _ = nbio.New_Simulated_IO(
				&memory.Sim, 0, time.NANOSECOND, sim_memory_view(memory),
			)
		})
		nbio.IO_Deinit(loop)
	})
	t.Run("Sim_Clock_To_Clock", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			nbio.Sim_Clock_To_Clock(&memory.Sim.Clocks[0])
		})
	})
	nbio_stream_constructors_heap_allocation(t)
}

func nbio_address_constructors_heap_allocation(t *testing.T) {
	ipv4 := [nbio.IPV4_ADDRESS_BYTES]byte{127, 0, 0, 1}
	ipv6 := [nbio.IPV6_ADDRESS_BYTES]byte{15: 1}
	parse_storage := [nbio.IPV6_ADDRESS_BYTES]byte{}
	address := nbio.Address{IP: parse_storage[:]}
	var parse_err error
	t.Run("Address_IPV4", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			address = nbio.Address_IPV4(ipv4[:], 8123)
		})
		testify.Equal(t, nbio.FAMILY_IPV4, address.Family)
	})
	t.Run("Address_IPV6", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			address = nbio.Address_IPV6(ipv6[:], 8123)
		})
		testify.Equal(t, nbio.FAMILY_IPV6, address.Family)
	})
	t.Run("Address_Parse", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			parse_err = nbio.Address_Parse(&address, "2001:db8::1", 8123)
		})
		testify.No_Error(t, parse_err)
		testify.Equal(t, nbio.FAMILY_IPV6, address.Family)
	})
	t.Run("Address_Parse_Error", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			parse_err = nbio.Address_Parse(&address, "localhost", 8123)
		})
		testify.Error(t, parse_err)
	})
}

func nbio_stream_constructors_heap_allocation(t *testing.T) {
	memory_state := nbio.Stream_Memory{Memory: []byte("memory")}
	discard_state := nbio.Stream_Discard{}
	var stream nbio.Stream
	t.Run("Memory_To_Stream", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			stream = nbio.Memory_To_Stream(&memory_state)
		})
	})
	t.Run("Discard_To_Stream", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			stream = nbio.Discard_To_Stream(&discard_state)
		})
	})
	inner := nbio.Memory_To_Stream(&memory_state)
	limit_state := nbio.Stream_Limit{Inner: inner, Budget: 1}
	count_state := nbio.Stream_Count{Inner: inner}
	tee_state := nbio.Stream_Tee{First: inner, Second: inner}
	t.Run("Limit_To_Stream", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			stream = nbio.Limit_To_Stream(&limit_state)
		})
	})
	t.Run("Count_To_Stream", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			stream = nbio.Count_To_Stream(&count_state)
		})
	})
	t.Run("Tee_To_Stream", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			stream = nbio.Tee_To_Stream(&tee_state)
		})
	})
	testify.Not_Nil(t, stream.Procedure)
}

// Each submitted Stream operation needs direct evidence because constructor proof cannot see
// callback adapters made after construction.
func nbio_stream_api_heap_allocation(t *testing.T) {
	storage := [STREAM_ALLOCATION_STORAGE_BYTES]byte{}
	state := nbio.Stream_Memory{Memory: storage[:]}
	stream := nbio.Memory_To_Stream(&state)
	buffer := [STREAM_ALLOCATION_BUFFER_BYTES]byte{1, 2}
	var completion nbio.Completion
	t.Run("Read", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			nbio.Read(stream, &completion, buffer[:], stream_allocation_callback)
		})
	})
	t.Run("Read_At", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			nbio.Read_At(stream, &completion, buffer[:], 0, stream_allocation_callback)
		})
	})
	t.Run("Write", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			nbio.Write(stream, &completion, buffer[:], stream_allocation_callback)
		})
	})
	t.Run("Write_At", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			nbio.Write_At(stream, &completion, buffer[:], 0, stream_allocation_callback)
		})
	})
	t.Run("Seek", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			nbio.Seek(stream, &completion, 0, nbio.SEEK_FROM_START,
				stream_allocation_callback)
		})
	})
	t.Run("Size", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			nbio.Size(stream, &completion, stream_allocation_callback)
		})
	})
	t.Run("Flush", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			nbio.Flush(stream, &completion, stream_allocation_callback)
		})
	})
	t.Run("Query", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			nbio.Query(stream, &completion, stream_allocation_callback)
		})
	})
	t.Run("Close", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			nbio.Close(stream, &completion, stream_allocation_callback)
		})
	})
	t.Run("Destroy", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			nbio.Destroy(stream, &completion, stream_allocation_callback)
		})
	})
	stream_composition_heap_allocation(t)
}

func stream_composition_heap_allocation(t *testing.T) {
	buffer := [STREAM_ALLOCATION_BUFFER_BYTES]byte{1, 2}
	completion := nbio.Completion{}
	first_storage := [STREAM_ALLOCATION_STORAGE_BYTES]byte{}
	second_storage := [STREAM_ALLOCATION_STORAGE_BYTES]byte{}
	first_memory := nbio.Stream_Memory{Memory: first_storage[:]}
	second_memory := nbio.Stream_Memory{Memory: second_storage[:]}
	first := nbio.Memory_To_Stream(&first_memory)
	second := nbio.Memory_To_Stream(&second_memory)
	limit_state := nbio.Stream_Limit{Inner: first, Budget: STREAM_ALLOCATION_STORAGE_BYTES}
	limit := nbio.Limit_To_Stream(&limit_state)
	t.Run("Limit_Write", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			nbio.Write(limit, &completion, buffer[:], stream_allocation_callback)
		})
	})
	count_state := nbio.Stream_Count{Inner: first}
	count := nbio.Count_To_Stream(&count_state)
	t.Run("Count_Write", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			nbio.Write(count, &completion, buffer[:], stream_allocation_callback)
		})
	})
	tee_state := nbio.Stream_Tee{First: first, Second: second}
	tee := nbio.Tee_To_Stream(&tee_state)
	t.Run("Tee_Write", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			nbio.Write(tee, &completion, buffer[:], stream_allocation_callback)
		})
	})
}

func nbio_stream_error_heap_allocation(t *testing.T) {
	buffer := [STREAM_ALLOCATION_BUFFER_BYTES]byte{}
	completion := nbio.Completion{}
	t.Run("Empty", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			nbio.Read(
				nbio.Stream{}, &completion, buffer[:], stream_allocation_callback,
			)
		})
		testify.Error_Is(t, completion.Error, nbio.Stream_Empty)
	})
	storage := [STREAM_ALLOCATION_STORAGE_BYTES]byte{}
	state := nbio.Stream_Memory{Memory: storage[:]}
	stream := nbio.Memory_To_Stream(&state)
	t.Run("Invalid_Seek", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			nbio.Seek(
				stream, &completion, -1, nbio.SEEK_FROM_START,
				stream_allocation_callback,
			)
		})
		testify.Error_Is(t, completion.Error, nbio.Stream_Invalid_Offset)
	})
	t.Run("Negative_Read", func(t *testing.T) {
		defective := nbio.Stream{Procedure: stream_allocation_negative_read}
		testify.Zero_Allocation(t, func() {
			nbio.Read(defective, &completion, buffer[:], stream_allocation_callback)
		})
		testify.Error_Is(t, completion.Error, nbio.Stream_Negative_Read)
	})
	t.Run("Invalid_Write", func(t *testing.T) {
		defective := nbio.Stream{Procedure: stream_allocation_invalid_write}
		testify.Zero_Allocation(t, func() {
			nbio.Write(defective, &completion, buffer[:], stream_allocation_callback)
		})
		testify.Error_Is(t, completion.Error, nbio.Stream_Invalid_Write)
	})
}

func stream_allocation_negative_read(
	_ nbio.State, completion *nbio.Completion, _ nbio.Stream_Mode, _ []byte,
	_ int64, _ nbio.Seek_From, callback nbio.Stream_Callback,
) {
	completion.Data = -1
	nbio.Stream_Callback_Call(callback, completion)
}

func stream_allocation_invalid_write(
	_ nbio.State, completion *nbio.Completion, _ nbio.Stream_Mode, buffer []byte,
	_ int64, _ nbio.Seek_From, callback nbio.Stream_Callback,
) {
	completion.Data = len(buffer) + 1
	nbio.Stream_Callback_Call(callback, completion)
}

const STREAM_ALLOCATION_STORAGE_BYTES = 8
const STREAM_ALLOCATION_BUFFER_BYTES = 2

func stream_allocation_callback(completion nbio.Completion_Handle) {
	if completion == nil {
		return
	}
}

// Every simulated surface operation gets direct allocation evidence because static constructor
// wiring does not prove retirement state stays off heap.
func sim_api_heap_allocation(t *testing.T) {
	tests := []struct {
		Name      string
		Operation sim_allocation_operation
	}{
		{Name: "Socket_TCP", Operation: SIM_ALLOCATION_SOCKET_TCP},
		{Name: "Socket_UDP", Operation: SIM_ALLOCATION_SOCKET_UDP},
		{Name: "Bind", Operation: SIM_ALLOCATION_BIND},
		{Name: "Listen_Socket", Operation: SIM_ALLOCATION_LISTEN},
		{Name: "Get_Socket_Name", Operation: SIM_ALLOCATION_SOCKET_NAME},
		{Name: "Accept", Operation: SIM_ALLOCATION_ACCEPT},
		{Name: "Connect", Operation: SIM_ALLOCATION_CONNECT},
		{Name: "Receive", Operation: SIM_ALLOCATION_RECEIVE},
		{Name: "Send", Operation: SIM_ALLOCATION_SEND},
		{Name: "Shutdown", Operation: SIM_ALLOCATION_SHUTDOWN},
		{Name: "Peer_Address", Operation: SIM_ALLOCATION_PEER_ADDRESS},
		{Name: "Read", Operation: SIM_ALLOCATION_READ},
		{Name: "Write", Operation: SIM_ALLOCATION_WRITE},
		{Name: "Fsync", Operation: SIM_ALLOCATION_FSYNC},
		{Name: "Open_At", Operation: SIM_ALLOCATION_OPEN_AT},
		{Name: "Mkdir_At", Operation: SIM_ALLOCATION_MKDIR_AT},
		{Name: "Get_Directory_Entries", Operation: SIM_ALLOCATION_DIRECTORY},
		{Name: "Status", Operation: SIM_ALLOCATION_STATUS},
		{Name: "Open_Link", Operation: SIM_ALLOCATION_OPEN_LINK},
		{Name: "Close", Operation: SIM_ALLOCATION_CLOSE},
		{Name: "Deinit", Operation: SIM_ALLOCATION_DEINIT},
		{Name: "Watch_Signal", Operation: SIM_ALLOCATION_WATCH_SIGNAL},
		{Name: "Spawn", Operation: SIM_ALLOCATION_SPAWN},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			harness := sim_allocation_harness{}
			if test.Operation == SIM_ALLOCATION_OPEN_LINK {
				// A seed without a link measures a plain open, proving nothing.
				testify.True(t, sim_allocation_find_link(&harness))
			}
			testify.Zero_Allocation(t, func() {
				sim_allocation_run(&harness, test.Operation)
			})
			testify.No_Error(t, harness.Error)
			sim_allocation_assert_retired(t, &harness, test.Operation)
		})
	}
	sim_socket_error_heap_allocation(t)
}

func sim_allocation_assert_retired(
	t *testing.T, harness *sim_allocation_harness, operation sim_allocation_operation,
) {
	t.Helper()
	switch operation {
	case SIM_ALLOCATION_ACCEPT,
		SIM_ALLOCATION_CONNECT,
		SIM_ALLOCATION_RECEIVE,
		SIM_ALLOCATION_SEND,
		SIM_ALLOCATION_SHUTDOWN,
		SIM_ALLOCATION_PEER_ADDRESS,
		SIM_ALLOCATION_READ,
		SIM_ALLOCATION_WRITE,
		SIM_ALLOCATION_FSYNC,
		SIM_ALLOCATION_OPEN_AT,
		SIM_ALLOCATION_MKDIR_AT,
		SIM_ALLOCATION_DIRECTORY,
		SIM_ALLOCATION_OPEN_LINK,
		SIM_ALLOCATION_CLOSE:
		testify.Not_Nil(t, harness.Completion.Self)
		testify.False(t, harness.Completion.Armed)
	}
	if operation == SIM_ALLOCATION_DIRECTORY {
		testify.True(t, harness.Completion.Data > 0)
		testify.Equal(t, "allocation", harness.Entries[0].Name)
		name_pointer := uintptr(unsafe.Pointer(unsafe.StringData(harness.Entries[0].Name)))
		names_start := uintptr(unsafe.Pointer(&harness.Memory.Node_Names[0]))
		names_end := names_start + uintptr(len(harness.Memory.Node_Names))
		testify.True(t, name_pointer >= names_start)
		testify.True(t, name_pointer < names_end)
	}
}

func sim_socket_error_heap_allocation(t *testing.T) {
	harness := sim_allocation_harness{}
	var socket nbio.File
	var socket_err error
	tcp := sim_tcp_options()
	tcp.Receive_Buffer_Bytes = 0
	t.Run("Socket_TCP_Error", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			sim_allocation_reset(&harness)
			socket, socket_err = nbio.Network_Socket_TCP(
				harness.Loop.Network, nbio.FAMILY_IPV4, tcp,
			)
		})
		testify.Error(t, socket_err)
		testify.Equal(t, nbio.File(-1), socket)
	})
	udp := sim_udp_options()
	udp.Receive_Buffer_Bytes = 0
	t.Run("Socket_UDP_Error", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			sim_allocation_reset(&harness)
			socket, socket_err = nbio.Network_Socket_UDP(
				harness.Loop.Network, nbio.FAMILY_IPV4, udp,
			)
		})
		testify.Error(t, socket_err)
		testify.Equal(t, nbio.File(-1), socket)
	})
}

type sim_allocation_operation uint8

const SIM_ALLOCATION_SOCKET_TCP sim_allocation_operation = 0
const SIM_ALLOCATION_SOCKET_UDP sim_allocation_operation = 1
const SIM_ALLOCATION_BIND sim_allocation_operation = 2
const SIM_ALLOCATION_LISTEN sim_allocation_operation = 3
const SIM_ALLOCATION_SOCKET_NAME sim_allocation_operation = 4
const SIM_ALLOCATION_ACCEPT sim_allocation_operation = 5
const SIM_ALLOCATION_CONNECT sim_allocation_operation = 6
const SIM_ALLOCATION_RECEIVE sim_allocation_operation = 7
const SIM_ALLOCATION_SEND sim_allocation_operation = 8
const SIM_ALLOCATION_SHUTDOWN sim_allocation_operation = 9
const SIM_ALLOCATION_PEER_ADDRESS sim_allocation_operation = 10
const SIM_ALLOCATION_READ sim_allocation_operation = 11
const SIM_ALLOCATION_WRITE sim_allocation_operation = 12
const SIM_ALLOCATION_FSYNC sim_allocation_operation = 13
const SIM_ALLOCATION_OPEN_AT sim_allocation_operation = 14
const SIM_ALLOCATION_MKDIR_AT sim_allocation_operation = 15
const SIM_ALLOCATION_DIRECTORY sim_allocation_operation = 16
const SIM_ALLOCATION_STATUS sim_allocation_operation = 17
const SIM_ALLOCATION_OPEN_LINK sim_allocation_operation = 18
const SIM_ALLOCATION_CLOSE sim_allocation_operation = 19
const SIM_ALLOCATION_DEINIT sim_allocation_operation = 20
const SIM_ALLOCATION_WATCH_SIGNAL sim_allocation_operation = 21
const SIM_ALLOCATION_SPAWN sim_allocation_operation = 22

type sim_allocation_harness struct {
	Memory            sim_memory
	Completion        nbio.Completion
	Second_Completion nbio.Completion
	Buffer            []byte
	Entries           []nbio.Directory_Entry
	Address_Storage   []byte
	Loop              nbio.IO
	Driver            nbio.Driver
	File              nbio.File
	Error             error
	Address           nbio.Address
	Status            nbio.File_Status
	// Seed picks the generated tree; a case that needs a link sweeps for one.
	Seed uint64
	// Link_Path is built before measurement because path assembly allocates and one seed
	// regenerates the same tree on every reset.
	Link_Path string
}

func sim_allocation_reset(harness *sim_allocation_harness) {
	if harness.Buffer == nil {
		harness.Buffer = make([]byte, SIM_ALLOCATION_BUFFER_BYTES)
	}
	if harness.Entries == nil {
		harness.Entries = make([]nbio.Directory_Entry, SIM_NODE_CAPACITY)
	}
	if harness.Address_Storage == nil {
		harness.Address_Storage = make([]byte, nbio.IPV6_ADDRESS_BYTES)
	}
	harness.Address = nbio.Address{IP: harness.Address_Storage}
	harness.Loop, harness.Driver = nbio.New_Simulated_IO(
		&harness.Memory.Sim, harness.Seed, time.NANOSECOND,
		sim_memory_view(&harness.Memory),
	)
	harness.Completion = nbio.Completion{}
	harness.File = 0
	harness.Error = nil
}

// Same sweep width as Test_File_Mode_Symbolic_Link, which already proves a link occurs inside it.
const SIM_ALLOCATION_LINK_SEED_SWEEP uint64 = 64

func sim_allocation_find_link(harness *sim_allocation_harness) (found bool) {
	for seed := uint64(0); seed < SIM_ALLOCATION_LINK_SEED_SWEEP; seed++ {
		harness.Seed = seed
		sim_allocation_reset(harness)
		harness.Link_Path, found = sim_first_link_path(harness.Memory.Nodes)
		if found {
			return true
		}
	}
	return false
}

func sim_allocation_run(
	harness *sim_allocation_harness, operation sim_allocation_operation,
) {
	sim_allocation_reset(harness)
	if operation <= SIM_ALLOCATION_PEER_ADDRESS {
		sim_allocation_network_run(harness, operation)
		return
	}
	if operation >= SIM_ALLOCATION_WATCH_SIGNAL {
		sim_allocation_effects_run(harness, operation)
		return
	}
	sim_allocation_storage_run(harness, operation)
}

func sim_allocation_network_run(
	harness *sim_allocation_harness, operation sim_allocation_operation,
) {
	switch operation {
	case SIM_ALLOCATION_SOCKET_TCP:
		harness.File, harness.Error = nbio.Network_Socket_TCP(
			harness.Loop.Network, nbio.FAMILY_IPV4, sim_tcp_options(),
		)
	case SIM_ALLOCATION_SOCKET_UDP:
		harness.File, harness.Error = nbio.Network_Socket_UDP(
			harness.Loop.Network, nbio.FAMILY_IPV4, sim_udp_options(),
		)
	case SIM_ALLOCATION_BIND:
		sim_allocation_open_tcp(harness)
		harness.Error = nbio.Network_Bind(
			harness.Loop.Network, harness.File,
			nbio.Address_IPV4(harness.Address_Storage[:nbio.IPV4_ADDRESS_BYTES], 0),
		)
	case SIM_ALLOCATION_LISTEN:
		sim_allocation_open_tcp(harness)
		harness.Error = nbio.Network_Listen_Socket(harness.Loop.Network, harness.File, 1)
	case SIM_ALLOCATION_SOCKET_NAME:
		sim_allocation_open_tcp(harness)
		harness.Error = nbio.Network_Get_Socket_Name(
			harness.Loop.Network, harness.File, &harness.Address,
		)
	case SIM_ALLOCATION_ACCEPT:
		sim_allocation_open_listener(harness)
		nbio.Network_Accept(
			harness.Loop.Network, &harness.Completion, harness.File, SIM_DEADLINE,
			sim_allocation_callback,
		)
		sim_allocation_drive(harness)
	case SIM_ALLOCATION_CONNECT:
		sim_allocation_open_tcp(harness)
		nbio.Network_Connect(
			harness.Loop.Network, &harness.Completion, harness.File,
			nbio.Address_IPV4(harness.Address_Storage[:nbio.IPV4_ADDRESS_BYTES], 1),
			SIM_DEADLINE, sim_allocation_callback,
		)
		sim_allocation_drive(harness)
	case SIM_ALLOCATION_RECEIVE:
		sim_allocation_open_tcp(harness)
		nbio.Network_Receive(
			harness.Loop.Network, &harness.Completion, harness.File,
			harness.Buffer[:], SIM_DEADLINE,
			sim_allocation_callback,
		)
		sim_allocation_drive(harness)
	case SIM_ALLOCATION_SEND:
		sim_allocation_open_tcp(harness)
		nbio.Network_Send(
			harness.Loop.Network, &harness.Completion, harness.File,
			harness.Buffer[:], SIM_DEADLINE,
			sim_allocation_callback,
		)
		sim_allocation_drive(harness)
	case SIM_ALLOCATION_SHUTDOWN:
		sim_allocation_accept(harness)
		harness.Error = nbio.Network_Shutdown(
			harness.Loop.Network, harness.File, nbio.SHUTDOWN_BOTH,
		)
	case SIM_ALLOCATION_PEER_ADDRESS:
		sim_allocation_accept(harness)
		harness.Error = nbio.Network_Peer_Address(
			harness.Loop.Network, harness.File, &harness.Address,
		)
	}
}

func sim_allocation_storage_run(
	harness *sim_allocation_harness, operation sim_allocation_operation,
) {
	switch operation {
	case SIM_ALLOCATION_READ:
		sim_allocation_open_file(harness)
		nbio.Storage_Read(
			harness.Loop.Storage, &harness.Completion, harness.File,
			harness.Buffer[:], 0, SIM_DEADLINE,
			sim_allocation_callback,
		)
		sim_allocation_drive(harness)
	case SIM_ALLOCATION_WRITE:
		sim_allocation_open_file(harness)
		nbio.Storage_Write(
			harness.Loop.Storage, &harness.Completion, harness.File,
			harness.Buffer[:], 0, SIM_DEADLINE,
			sim_allocation_callback,
		)
		sim_allocation_drive(harness)
	case SIM_ALLOCATION_FSYNC:
		sim_allocation_open_file(harness)
		nbio.Storage_Fsync(
			harness.Loop.Storage, &harness.Completion, harness.File, SIM_DEADLINE,
			sim_allocation_callback,
		)
		sim_allocation_drive(harness)
	case SIM_ALLOCATION_OPEN_AT:
		nbio.Storage_Open_At(
			harness.Loop.Storage, &harness.Completion, nbio.DIRECTORY_CURRENT,
			"allocation",
			nbio.Open_At_Options{
				Access: nbio.OPEN_READ_WRITE, Create: true, Permissions: 0o600,
			},
			sim_allocation_callback,
		)
		sim_allocation_drive(harness)
	case SIM_ALLOCATION_MKDIR_AT:
		nbio.Storage_Mkdir_At(
			harness.Loop.Storage, &harness.Completion, nbio.DIRECTORY_CURRENT,
			"allocation", 0o700,
			sim_allocation_callback,
		)
		sim_allocation_drive(harness)
	case SIM_ALLOCATION_DIRECTORY:
		sim_allocation_open_file(harness)
		sim_allocation_open_directory(harness)
		nbio.Storage_Get_Directory_Entries(
			harness.Loop.Storage, &harness.Completion, harness.File,
			harness.Buffer[:], harness.Entries[:],
			sim_allocation_callback,
		)
		sim_allocation_drive(harness)
	case SIM_ALLOCATION_STATUS:
		harness.Status, harness.Error = nbio.Storage_Status(harness.Loop.Storage, "/")
	case SIM_ALLOCATION_OPEN_LINK:
		sim_allocation_open_link(harness)
	case SIM_ALLOCATION_CLOSE:
		sim_allocation_open_tcp(harness)
		nbio.IO_Close(
			harness.Loop, &harness.Completion, harness.File, sim_allocation_callback,
		)
		sim_allocation_drive(harness)
	case SIM_ALLOCATION_DEINIT:
		nbio.IO_Deinit(harness.Loop)
	}
}

func sim_allocation_effects_run(
	harness *sim_allocation_harness, operation sim_allocation_operation,
) {
	if operation == SIM_ALLOCATION_WATCH_SIGNAL {
		nbio.IO_Watch_Signal(
			harness.Loop, &harness.Completion, nbio.SIGNAL_TERMINATE, SIM_DEADLINE,
			sim_allocation_signal_callback,
		)
		sim_allocation_drive(harness)
		return
	}
	nbio.IO_Spawn(
		harness.Loop, &harness.Completion, nbio.Process_Request{Path: "true"},
		SIM_DEADLINE, sim_allocation_process_callback,
	)
	sim_allocation_drive(harness)
}

func sim_allocation_open_tcp(harness *sim_allocation_harness) {
	harness.File, harness.Error = nbio.Network_Socket_TCP(
		harness.Loop.Network, nbio.FAMILY_IPV4, sim_tcp_options(),
	)
}

func sim_allocation_open_listener(harness *sim_allocation_harness) {
	sim_allocation_open_tcp(harness)
	harness.Error = nbio.Network_Listen_Socket(harness.Loop.Network, harness.File, 1)
}

func sim_allocation_accept(harness *sim_allocation_harness) {
	sim_allocation_open_listener(harness)
	nbio.Network_Accept(
		harness.Loop.Network, &harness.Completion, harness.File, SIM_DEADLINE,
		sim_allocation_callback,
	)
	sim_allocation_drive(harness)
	harness.File = nbio.File(harness.Completion.Data)
}

func sim_allocation_open_file(harness *sim_allocation_harness) {
	nbio.Storage_Open_At(
		harness.Loop.Storage, &harness.Completion, nbio.DIRECTORY_CURRENT, "allocation",
		nbio.Open_At_Options{
			Access: nbio.OPEN_READ_WRITE, Create: true, Permissions: 0o600,
		},
		sim_allocation_callback,
	)
	sim_allocation_drive(harness)
	harness.File = nbio.File(harness.Completion.Data)
}

// Follow a generated link because target resolution views node bytes as a string.
func sim_allocation_open_link(harness *sim_allocation_harness) {
	nbio.Storage_Open_At(
		harness.Loop.Storage, &harness.Completion, nbio.DIRECTORY_CURRENT,
		harness.Link_Path, nbio.Open_At_Options{Access: nbio.OPEN_READ_ONLY},
		sim_allocation_callback,
	)
	sim_allocation_drive(harness)
	harness.File = nbio.File(harness.Completion.Data)
}

func sim_allocation_open_directory(harness *sim_allocation_harness) {
	nbio.Storage_Open_At(
		harness.Loop.Storage, &harness.Completion, nbio.DIRECTORY_CURRENT, "/",
		nbio.Open_At_Options{Access: nbio.OPEN_READ_ONLY}, sim_allocation_callback,
	)
	sim_allocation_drive(harness)
	harness.File = nbio.File(harness.Completion.Data)
}

func sim_allocation_drive(harness *sim_allocation_harness) {
	harness.Error = nbio.Driver_Run_For(harness.Driver, SIM_DEADLINE)
}

const SIM_ALLOCATION_BUFFER_BYTES = 8

// SIM_OPERATION_EXHAUSTION_CAPACITY leaves no second slot, forcing bounded failure.
const SIM_OPERATION_EXHAUSTION_CAPACITY = 1

func sim_operation_exhaustion_heap_allocation(t *testing.T) {
	harness := sim_allocation_harness{}
	testify.Zero_Allocation(t, func() {
		sim_operation_exhaustion_run(&harness)
	})
	testify.Error(t, harness.Second_Completion.Error)
}

func sim_operation_exhaustion_run(harness *sim_allocation_harness) {
	harness.Completion = nbio.Completion{}
	harness.Second_Completion = nbio.Completion{}
	memory := sim_memory_view(&harness.Memory)
	memory.Operations = memory.Operations[:SIM_OPERATION_EXHAUSTION_CAPACITY]
	harness.Loop, harness.Driver = nbio.New_Simulated_IO(
		&harness.Memory.Sim, 0, time.NANOSECOND, memory,
	)
	harness.File, harness.Error = nbio.Network_Socket_TCP(
		harness.Loop.Network, nbio.FAMILY_IPV4, sim_tcp_options(),
	)
	nbio.Network_Receive(
		harness.Loop.Network, &harness.Completion, harness.File,
		harness.Buffer[:], SIM_DEADLINE, sim_allocation_callback,
	)
	nbio.Network_Send(
		harness.Loop.Network, &harness.Second_Completion, harness.File,
		harness.Buffer[:], SIM_DEADLINE, sim_allocation_callback,
	)
	sim_allocation_drive(harness)
}

func sim_allocation_callback(completion nbio.Completion_Handle) {
	if completion == nil {
		return
	}
}

func sim_allocation_signal_callback(
	completion *nbio.Completion, signal nbio.Signal, err error,
) {
	completion.Data = int(signal)
	completion.Error = err
}

func sim_allocation_process_callback(
	completion *nbio.Completion, result nbio.Process_Result, err error,
) {
	completion.Data = result.Exit
	completion.Error = err
}

// The Stream function owns callback time because only it knows if the concrete operation is
// immediate, simulated, or kernel-backed.
func test_stream_callback(t *testing.T) {
	timeline, driver, _ := sim_timeline()
	buffer := make([]byte, 3)
	called := false
	state := stream_callback_state{
		Timeline: timeline, Count: len(buffer), Delayed: true,
	}
	stream := nbio.Stream{
		State: nbio.State(&state), Procedure: stream_callback_procedure,
	}
	var completion nbio.Completion
	nbio.Read(stream, &completion, buffer, func(completed nbio.Completion_Handle) {
		called = true
		testify.Equal(t, len(buffer), completed.Data)
		testify.No_Error(t, completed.Error)
	})
	testify.False(t, called)
	completed, drive_err := nbio.Driver_Run_Until(driver,
		10*time.NANOSECOND, func() (finished bool) { return called },
	)
	testify.No_Error(t, drive_err)
	testify.True(t, completed)
	testify.True(t, called)
	inline_called := false
	state.Delayed = false
	nbio.Read(stream, &completion, buffer, func(_ nbio.Completion_Handle) {
		inline_called = true
	})
	testify.True(t, inline_called)
}

type stream_callback_state struct {
	Timeline nbio.Timeline
	Count    int
	Delayed  bool
	Callback nbio.Stream_Callback
}

func stream_callback_procedure(
	state_pointer nbio.State, completion *nbio.Completion, _ nbio.Stream_Mode, _ []byte,
	_ int64, _ nbio.Seek_From, callback nbio.Stream_Callback,
) {
	state := state_pointer.(*stream_callback_state)
	completion.Data = state.Count
	if state.Delayed {
		state.Callback = callback
		completion.Backend = state_pointer
		nbio.Timeline_Submit(
			state.Timeline, completion, time.NANOSECOND, stream_callback_complete,
		)
		return
	}
	nbio.Stream_Callback_Call(callback, completion)
}

func stream_callback_complete(completion nbio.Completion_Handle) {
	state := completion.Backend.(*stream_callback_state)
	callback := state.Callback
	state.Callback = nbio.Stream_Callback{}
	completion.Backend = nil
	completion.Data = state.Count
	nbio.Stream_Callback_Call(callback, completion)
}

// Test_Stream_Read verifies Read moves bytes from the cursor, advances the cursor, and
// reports Stream_EOF once the cursor has reached the end of the memory.
func test_stream_read(t *testing.T) {
	harness := stream_harness(t)
	stream := memory_stream(harness, []byte("abcdef"))
	first := make([]byte, 3)
	count, err := stream_read(t, harness, stream, first)
	testify.No_Error(t, err)
	testify.Equal(t, 3, count)
	testify.Equal(t, "abc", string(first))
	second := make([]byte, 8)
	count, err = stream_read(t, harness, stream, second)
	testify.No_Error(t, err)
	testify.Equal(t, 3, count)
	_, err = stream_read(t, harness, stream, second)
	testify.Error_Is(t, err, nbio.Stream_EOF)
}

// Test_Stream_Write verifies Write stores bytes at the cursor, advances the cursor, and
// reports Stream_Short_Write when the remaining memory cannot hold the whole buffer.
func test_stream_write(t *testing.T) {
	harness := stream_harness(t)
	memory := make([]byte, 4)
	stream := memory_stream(harness, memory)
	count, err := stream_write(t, harness, stream, []byte("ab"))
	testify.No_Error(t, err)
	testify.Equal(t, 2, count)
	count, err = stream_write(t, harness, stream, []byte("cdef"))
	testify.Error_Is(t, err, nbio.Stream_Short_Write)
	testify.Equal(t, 2, count)
	testify.Equal(t, "abcd", string(memory))
}

// Test_Stream_Read_At verifies Read_At reads from an explicit offset and leaves the cursor
// where it was, the distinction Odin draws between Read and Read_At.
func test_stream_read_at(t *testing.T) {
	harness := stream_harness(t)
	stream := memory_stream(harness, []byte("abcdef"))
	head := make([]byte, 2)
	_, read_err := stream_read(t, harness, stream, head)
	testify.No_Error(t, read_err)
	tail := make([]byte, 2)
	count, err := stream_read_at(t, harness, stream, tail, 4)
	testify.No_Error(t, err)
	testify.Equal(t, 2, count)
	testify.Equal(t, "ef", string(tail))
	position, seek_err := stream_seek(t, harness, stream, 0, nbio.SEEK_FROM_CURRENT)
	testify.No_Error(t, seek_err)
	testify.Equal(t, 2, position)
}

// Test_Stream_Write_At verifies Write_At stores at an explicit offset and leaves the cursor
// where it was.
func test_stream_write_at(t *testing.T) {
	harness := stream_harness(t)
	memory := make([]byte, 6)
	stream := memory_stream(harness, memory)
	count, err := stream_write_at(t, harness, stream, []byte("xy"), 4)
	testify.No_Error(t, err)
	testify.Equal(t, 2, count)
	testify.Equal(t, byte('x'), memory[4])
	position, seek_err := stream_seek(t, harness, stream, 0, nbio.SEEK_FROM_CURRENT)
	testify.No_Error(t, seek_err)
	testify.Zero(t, position)
}

// Test_Stream_Seek verifies each Seek_From origin, and that an origin outside the three
// reports Stream_Invalid_Whence.
func test_stream_seek(t *testing.T) {
	harness := stream_harness(t)
	stream := memory_stream(harness, []byte("abcdef"))
	position, err := stream_seek(t, harness, stream, 2, nbio.SEEK_FROM_START)
	testify.No_Error(t, err)
	testify.Equal(t, 2, position)
	position, err = stream_seek(t, harness, stream, 1, nbio.SEEK_FROM_CURRENT)
	testify.No_Error(t, err)
	testify.Equal(t, 3, position)
	position, err = stream_seek(t, harness, stream, -1, nbio.SEEK_FROM_END)
	testify.No_Error(t, err)
	testify.Equal(t, 5, position)
	_, err = stream_seek(t, harness, stream, 0, nbio.Seek_From(9))
	testify.Error_Is(t, err, nbio.Stream_Invalid_Whence)
	_, err = stream_seek(t, harness, stream, -1, nbio.SEEK_FROM_START)
	testify.Error_Is(t, err, nbio.Stream_Invalid_Offset)
}

// Test_Stream_Size verifies Size reports the whole memory, not the bytes remaining.
func test_stream_size(t *testing.T) {
	harness := stream_harness(t)
	stream := memory_stream(harness, []byte("abcdef"))
	_, seek_err := stream_seek(t, harness, stream, 4, nbio.SEEK_FROM_START)
	testify.No_Error(t, seek_err)
	size, err := stream_size(t, harness, stream)
	testify.No_Error(t, err)
	testify.Equal(t, 6, size)
}

// Test_Stream_Query verifies Query names exactly the modes a stream answers, so a caller
// learns what a stream cannot do without provoking a failure.
func test_stream_query(t *testing.T) {
	harness := stream_harness(t)
	memory, memory_err := stream_query(t, harness, memory_stream(harness, make([]byte, 4)))
	testify.No_Error(t, memory_err)
	testify.True(t, nbio.Mode_Set_Has(memory, nbio.STREAM_MODE_SEEK))
	testify.True(t, nbio.Mode_Set_Has(memory, nbio.STREAM_MODE_SIZE))
	discard, discard_err := stream_query(t, harness, discard_stream(harness))
	testify.No_Error(t, discard_err)
	testify.False(t, nbio.Mode_Set_Has(discard, nbio.STREAM_MODE_SEEK))
	testify.True(t, nbio.Mode_Set_Has(discard, nbio.STREAM_MODE_WRITE))
}

// Test_Stream_Flush verifies Flush succeeds on memory, which has nothing to flush, so a
// caller can flush any stream without asking what is behind it.
func test_stream_flush(t *testing.T) {
	harness := stream_harness(t)
	_, err := stream_flush(
		t, harness, memory_stream(harness, make([]byte, 2)),
	)
	testify.No_Error(t, err)
	_, err = stream_flush(t, harness, discard_stream(harness))
	testify.No_Error(t, err)
}

// Test_Stream_Close verifies Close is idempotent and that a closed stream answers no data
// mode.
func test_stream_close(t *testing.T) {
	harness := stream_harness(t)
	stream := memory_stream(harness, make([]byte, 4))
	_, err := stream_close(t, harness, stream)
	testify.No_Error(t, err)
	_, err = stream_close(t, harness, stream)
	testify.No_Error(t, err)
	_, err = stream_write(t, harness, stream, []byte("a"))
	testify.Error_Is(t, err, nbio.Stream_Empty)
}

// Test_Stream_Destroy verifies Destroy closes the stream. Odin separates the two because a
// stream there can own an allocation; a memory stream owns nothing but its cursor.
func test_stream_destroy(t *testing.T) {
	harness := stream_harness(t)
	stream := memory_stream(harness, make([]byte, 4))
	_, err := stream_destroy(t, harness, stream)
	testify.No_Error(t, err)
	_, err = stream_read(t, harness, stream, make([]byte, 1))
	testify.Error_Is(t, err, nbio.Stream_Empty)
}

// Test_Stream_Errors verifies the dispatch checks Odin's write helper performs: a zero
// Stream reports Stream_Empty, and a mode a stream does not answer reports Stream_Empty.
func test_stream_errors(t *testing.T) {
	harness := stream_harness(t)
	var zero nbio.Stream
	_, err := stream_read(t, harness, zero, make([]byte, 1))
	testify.Error_Is(t, err, nbio.Stream_Empty)
	_, query_err := stream_query(t, harness, zero)
	testify.Error_Is(t, query_err, nbio.Stream_Empty)
	_, err = stream_seek(t, harness, discard_stream(harness), 0, nbio.SEEK_FROM_START)
	testify.Error_Is(t, err, nbio.Stream_Empty)
}

// Test_Stream_Memory verifies the memory stream never grows its slice: it is bounded by the
// slice it was built over, which is what makes it safe to hand to an unbounded encoder.
func test_stream_memory(t *testing.T) {
	harness := stream_harness(t)
	memory := make([]byte, 3)
	state := nbio.Stream_Memory{Memory: memory}
	stream := nbio.Memory_To_Stream(&state)
	count, err := stream_write(t, harness, stream, []byte("abcdefgh"))
	testify.Error_Is(t, err, nbio.Stream_Short_Write)
	testify.Equal(t, 3, count)
	testify.Count(t, memory, 3)
	testify.Equal(t, "abc", string(memory))
}

// Test_Stream_Discard verifies a discard stream absorbs every write, reports the whole buffer
// stored, and answers no read mode.
func test_stream_discard(t *testing.T) {
	harness := stream_harness(t)
	stream := discard_stream(harness)
	count, err := stream_write(t, harness, stream, []byte("abcdef"))
	testify.No_Error(t, err)
	testify.Equal(t, 6, count)
	_, err = stream_read(t, harness, stream, make([]byte, 4))
	testify.Error_Is(t, err, nbio.Stream_Empty)
}

// Test_Stream_Limit verifies a limit truncates at its budget and stops the bytes reaching the
// stream behind it, so the budget is a fact about the transport and not a caller convention.
func test_stream_limit(t *testing.T) {
	harness := stream_harness(t)
	memory := make([]byte, 8)
	inner := memory_stream(harness, memory)
	state := nbio.Stream_Limit{Inner: inner, Budget: 3}
	stream := nbio.Limit_To_Stream(&state)
	count, err := stream_write(t, harness, stream, []byte("abcde"))
	testify.Error_Is(t, err, nbio.Stream_Short_Write)
	testify.Equal(t, 3, count)
	testify.Equal(t, "abc\x00", string(memory[:4]))
	_, err = stream_write(t, harness, stream, []byte("f"))
	testify.Error_Is(t, err, nbio.Stream_Short_Write)
	modes, query_err := stream_query(t, harness, stream)
	testify.No_Error(t, query_err)
	testify.False(t, nbio.Mode_Set_Has(modes, nbio.STREAM_MODE_SEEK))
}

// Test_Stream_Count verifies a count tallies every byte and changes nothing else, so the same
// encoder measures and stores without being told which it is doing.
func test_stream_count(t *testing.T) {
	harness := stream_harness(t)
	memory := make([]byte, 8)
	state := nbio.Stream_Count{Inner: memory_stream(harness, memory)}
	stream := nbio.Count_To_Stream(&state)
	_, err := stream_write(t, harness, stream, []byte("ab"))
	testify.No_Error(t, err)
	_, err = stream_write(t, harness, stream, []byte("cde"))
	testify.No_Error(t, err)
	testify.Equal(t, int64(5), state.Tally)
	testify.Equal(t, "abcde", string(memory[:5]))
	measure := nbio.Stream_Count{Inner: discard_stream(harness)}
	measure_stream := nbio.Count_To_Stream(&measure)
	_, err = stream_write(t, harness, measure_stream, []byte("abcd"))
	testify.No_Error(t, err)
	testify.Equal(t, int64(4), measure.Tally)
}

// Test_Stream_Tee verifies a tee writes each buffer to both streams and reports the smaller
// count, so a caller learns about the tighter of the two rather than the first.
func test_stream_tee(t *testing.T) {
	harness := stream_harness(t)
	wide := make([]byte, 8)
	narrow := make([]byte, 2)
	state := nbio.Stream_Tee{
		First: memory_stream(harness, wide), Second: memory_stream(harness, narrow),
	}
	stream := nbio.Tee_To_Stream(&state)
	count, err := stream_write(t, harness, stream, []byte("abcd"))
	testify.Error_Is(t, err, nbio.Stream_Short_Write)
	testify.Equal(t, 2, count)
	testify.Equal(t, "abcd", string(wide[:4]))
	testify.Equal(t, "ab", string(narrow))
	modes, query_err := stream_query(t, harness, stream)
	testify.No_Error(t, query_err)
	testify.False(t, nbio.Mode_Set_Has(modes, nbio.STREAM_MODE_READ))
}

// Test_Stream_Composition verifies the transforms compose. One encoder writes through a tee,
// over a count, over a limit, over memory, and the tally, the truncation, and the stored bytes
// all agree — the property that makes the abstraction worth its indirection.
func test_stream_composition(t *testing.T) {
	harness := stream_harness(t)
	stored := make([]byte, 16)
	limit := nbio.Stream_Limit{Inner: memory_stream(harness, stored), Budget: 6}
	limited := nbio.Limit_To_Stream(&limit)
	count := nbio.Stream_Count{Inner: limited}
	audit := make([]byte, 16)
	counted := nbio.Count_To_Stream(&count)
	tee := nbio.Stream_Tee{
		First: counted, Second: memory_stream(harness, audit),
	}
	stream := nbio.Tee_To_Stream(&tee)

	written, err := stream_write(t, harness, stream, []byte("abcdefghij"))
	testify.Error_Is(t, err, nbio.Stream_Short_Write)
	testify.Equal(t, 6, written)
	testify.Equal(t, int64(6), count.Tally)
	testify.Equal(t, "abcdef\x00\x00", string(stored[:8]))
	testify.Equal(t, "abcdefghij", string(audit[:10]))
}

func test_text_has_suffix(source string, suffix string) (present bool) {
	if len(suffix) > len(source) {
		return false
	}
	return source[len(source)-len(suffix):] == suffix
}

// One literal the parse must accept, with the address it must yield.
type Address_Case struct {
	// Host is the literal under test.
	Host string
	// Family is the family the parse must report.
	Family nbio.Address_Family
	// IP is the byte layout the parse must produce.
	IP []byte
}

// State every literal shape the parse accept: dotted-quad, full IPv6, both ends of one "::" run,
// and embedded IPv4 in the low 32 bits.
func address_parse_accepted() (cases []Address_Case) {
	return []Address_Case{
		{Host: "127.0.0.1", Family: nbio.FAMILY_IPV4, IP: address_bytes(127, 0, 0, 1)},
		{Host: "0.0.0.0", Family: nbio.FAMILY_IPV4, IP: address_bytes()},
		{Host: "255.255.255.255", Family: nbio.FAMILY_IPV4,
			IP: address_bytes(255, 255, 255, 255)},
		{Host: "1:2:3:4:5:6:7:8", Family: nbio.FAMILY_IPV6, IP: address_bytes(
			0, 1, 0, 2, 0, 3, 0, 4, 0, 5, 0, 6, 0, 7, 0, 8)},
		{Host: "::", Family: nbio.FAMILY_IPV6, IP: address_bytes()},
		{Host: "::1", Family: nbio.FAMILY_IPV6, IP: address_low(1)},
		{Host: "fe80::", Family: nbio.FAMILY_IPV6, IP: address_bytes(0xfe, 0x80)},
		{Host: "2001:db8::1", Family: nbio.FAMILY_IPV6,
			IP: address_merge(address_bytes(0x20, 0x01, 0x0d, 0xb8), address_low(1))},
		{Host: "::ffff:127.0.0.1", Family: nbio.FAMILY_IPV6,
			IP: address_suffix(0xff, 0xff, 127, 0, 0, 1)},
		{Host: "FE80::1", Family: nbio.FAMILY_IPV6,
			IP: address_merge(address_bytes(0xfe, 0x80), address_low(1))},
	}
}

// Lay values at front of address, zero-filled to the end.
func address_bytes(values ...byte) (layout []byte) {
	layout = make([]byte, nbio.IPV6_ADDRESS_BYTES)
	copy(layout, values)
	return layout
}

// Lay values at back of address, zero-filled from the front.

func address_suffix(values ...byte) (layout []byte) {
	layout = make([]byte, nbio.IPV6_ADDRESS_BYTES)
	copy(layout[nbio.IPV6_ADDRESS_BYTES-len(values):], values)
	return layout
}

// Put one byte in the last position, the shape a "::" run with one trailing group yield.

func address_low(value byte) (layout []byte) {
	layout = make([]byte, nbio.IPV6_ADDRESS_BYTES)
	layout[nbio.IPV6_ADDRESS_BYTES-1] = value
	return layout
}

// Combine two layouts, taking every non-zero byte of each.
func address_merge(
	front []byte, back []byte,
) (layout []byte) {
	layout = make([]byte, nbio.IPV6_ADDRESS_BYTES)
	copy(layout, front)
	for index, value := range back {
		if value != 0 {
			layout[index] = value
		}
	}
	return layout
}

// State every literal the parse reject, one per rule the hand-written reader must hold.
func address_parse_rejected() (hosts []string) {
	return []string{
		"",
		"localhost",
		"127.0.0",
		"127.0.0.1.1",
		"127.0.0.256",
		"010.1.1.1",
		"127.0.0.-1",
		"127.0.0.0001",
		"1:2:3:4:5:6:7",
		"1:2:3:4:5:6:7:8:9",
		"1::2::3",
		"::12345",
		"::gggg",
		"fe80::1%eth0",
		":1",
		"1:",
		"1:2:3:4:5:6:7:8::",
		"::ffff:127.0.0.256",
	}
}

// One pure profile prevents the same simulation input from selecting different limits per host.
func socket_default_profile_assert(t *testing.T) {
	t.Helper()
	values := map[string]uint32{
		"TCP receive buffer":     nbio.TCP_RECEIVE_BUFFER_BYTES_DEFAULT,
		"TCP send buffer":        nbio.TCP_SEND_BUFFER_BYTES_DEFAULT,
		"UDP send buffer":        nbio.UDP_SEND_BUFFER_BYTES_DEFAULT,
		"receive low water":      nbio.SOCKET_RECEIVE_LOW_WATER_BYTES_DEFAULT,
		"TCP maximum segment":    nbio.TCP_MAXIMUM_SEGMENT_BYTES_DEFAULT,
		"TCP not-sent low water": nbio.TCP_NOT_SENT_LOW_WATER_BYTES_DEFAULT,
		"TCP keepalive probes":   nbio.TCP_KEEPALIVE_PROBE_COUNT_DEFAULT,
	}
	want := map[string]uint32{
		"TCP receive buffer":     128 * 1024,
		"TCP send buffer":        128 * 1024,
		"UDP send buffer":        208 * 1024,
		"receive low water":      1,
		"TCP maximum segment":    576 - 5*4 - 5*4,
		"TCP not-sent low water": 1 << 10,
		"TCP keepalive probes":   9,
	}
	for name, value := range values {
		testify.Equal(t, want[name], value, name)
	}
	durations := map[string]time.Duration{
		"linger timeout":         nbio.SOCKET_LINGER_TIMEOUT_DEFAULT,
		"TCP keepalive idle":     nbio.TCP_KEEPALIVE_IDLE_DEFAULT,
		"TCP keepalive interval": nbio.TCP_KEEPALIVE_INTERVAL_DEFAULT,
	}
	duration_want := map[string]time.Duration{
		"linger timeout":         1 * time.SECOND,
		"TCP keepalive idle":     2 * time.HOUR,
		"TCP keepalive interval": 75 * time.SECOND,
	}
	for name, value := range durations {
		testify.Equal(t, duration_want[name], value, name)
	}
	testify.True(t, nbio.TCP_NO_DELAY_DEFAULT)
}

// Test each limit separately because one rejected empty profile cannot prove all field checks.
func sim_socket_limits_rejected(t *testing.T, loop nbio.IO, driver nbio.Driver) {
	t.Helper()
	tcp := sim_tcp_options()
	tcp.Receive_Buffer_Bytes = 0
	sim_tcp_limit_rejected(t, loop, driver, "receive buffer", tcp)
	tcp = sim_tcp_options()
	tcp.Send_Buffer_Bytes = 0
	sim_tcp_limit_rejected(t, loop, driver, "send buffer", tcp)
	tcp = sim_tcp_options()
	tcp.Receive_Low_Water_Bytes = 0
	sim_tcp_limit_rejected(t, loop, driver, "receive low water", tcp)
	tcp.Linger_Timeout = 0
	sim_tcp_limit_rejected(t, loop, driver, "linger timeout", tcp)
	tcp = sim_tcp_options()
	tcp.Maximum_Segment_Bytes = 0
	sim_tcp_limit_rejected(t, loop, driver, "maximum segment", tcp)
	tcp = sim_tcp_options()
	tcp.Not_Sent_Low_Water_Bytes = 0
	sim_tcp_limit_rejected(t, loop, driver, "not-sent low water", tcp)
	tcp = sim_tcp_options()
	tcp.Keepalive.Idle = 0
	sim_tcp_limit_rejected(t, loop, driver, "keepalive idle", tcp)
	tcp = sim_tcp_options()
	tcp.Keepalive.Interval = 0
	sim_tcp_limit_rejected(t, loop, driver, "keepalive interval", tcp)
	tcp = sim_tcp_options()
	tcp.Keepalive.Probe_Count = 0
	sim_tcp_limit_rejected(t, loop, driver, "keepalive probe count", tcp)

	udp := sim_udp_options()
	udp.Receive_Buffer_Bytes = 0
	sim_udp_limit_rejected(t, loop, driver, "receive buffer", udp)
	udp = sim_udp_options()
	udp.Send_Buffer_Bytes = 0
	sim_udp_limit_rejected(t, loop, driver, "send buffer", udp)
	udp = sim_udp_options()
	udp.Receive_Low_Water_Bytes = 0
	sim_udp_limit_rejected(t, loop, driver, "receive low water", udp)
	udp.Linger_Timeout = 0
	sim_udp_limit_rejected(t, loop, driver, "linger timeout", udp)
}

func sim_tcp_limit_rejected(
	t *testing.T, loop nbio.IO, driver nbio.Driver, name string, options nbio.TCP_Options,
) {
	t.Helper()
	socket, err := nbio.Network_Socket_TCP(loop.Network, nbio.FAMILY_IPV4, options)
	if testify.Error(t, err, name) {
		return
	}
	sim_close(t, loop, driver, socket)
}

func sim_udp_limit_rejected(
	t *testing.T, loop nbio.IO, driver nbio.Driver, name string, options nbio.UDP_Options,
) {
	t.Helper()
	socket, err := nbio.Network_Socket_UDP(loop.Network, nbio.FAMILY_IPV4, options)
	if testify.Error(t, err, name) {
		return
	}
	sim_close(t, loop, driver, socket)
}

// Return the complete TCP profile every simulator caller must select.
func sim_tcp_options() (options nbio.TCP_Options) {
	return nbio.TCP_Options{
		Receive_Buffer_Bytes:     64 * 1024,
		Send_Buffer_Bytes:        64 * 1024,
		Receive_Low_Water_Bytes:  1,
		Linger_Timeout:           1 * time.SECOND,
		Maximum_Segment_Bytes:    512,
		Not_Sent_Low_Water_Bytes: 1024,
		Keepalive: nbio.TCP_Keepalive{
			Idle: 5 * time.SECOND, Interval: 4 * time.SECOND, Probe_Count: 3,
		},
		No_Delay: true,
	}
}

// Return the complete UDP profile every simulator caller must select.
func sim_udp_options() (options nbio.UDP_Options) {
	return nbio.UDP_Options{
		Receive_Buffer_Bytes:    64 * 1024,
		Send_Buffer_Bytes:       64 * 1024,
		Receive_Low_Water_Bytes: 1,
		Linger_Timeout:          1 * time.SECOND,
	}
}

// State regular-file half of Status, thus Status test stay inside line cap.
func sim_status_regular(t *testing.T, loop nbio.IO, content []byte) {
	t.Helper()
	regular, _ := nbio.Storage_Status(loop.Storage, "/dir/file")
	testify.False(t, nbio.File_Mode_Is_Directory(regular.Mode))
	testify.True(t, nbio.File_Mode_Is_Regular(regular.Mode))
	testify.Equal(t, int64(len(content)), regular.Size)
	absent, _ := nbio.Storage_Status(loop.Storage, "/nope")
	// Absent path is the zero status whole. A zero mode reads as a regular file with no
	// permission, thus Exists is the only guard a caller may branch on.
	testify.Equal(t, nbio.File_Status{}, absent)
}

func connect_outcome(err error) (outcome string) {
	if err == nil {
		return "success"
	}
	if err == nbio.Connection_Refused {
		return "refused"
	}
	return err.Error()
}

// Run one bounded accept against fresh listener and release both descriptors, thus seed sweep
// leave nothing open for Deinit to reject.
func sim_accept_once(
	t *testing.T, seed uint64, timeout time.Duration,
) (accepted nbio.File, operation_err error) {
	t.Helper()
	loop, driver, _ := sim_loop(seed)
	listener, _ := nbio.Network_Socket_TCP(
		loop.Network, nbio.FAMILY_IPV4, sim_tcp_options(),
	)
	address := nbio.Address_IPV4([]byte{127, 0, 0, 1}, 0)
	testify.No_Error(t, nbio.Network_Bind(loop.Network, listener, address), seed)
	testify.No_Error(t, nbio.Network_Listen_Socket(loop.Network, listener, 128), seed)
	callback_count := 0
	accepted = nbio.File(-1)
	var completion nbio.Completion
	nbio.Network_Accept(loop.Network, &completion, listener, timeout,
		func(completed nbio.Completion_Handle) {
			callback_count++
			accepted = nbio.File(completed.Data)
			operation_err = completed.Error
		})
	completed, drive_err := nbio.Driver_Run_Until(driver,
		16*time.NANOSECOND, func() (finished bool) { return callback_count > 0 })
	testify.No_Error(t, drive_err, seed)
	testify.True(t, completed, seed)
	nbio.Driver_Run_For(driver, 16*time.NANOSECOND)
	testify.Equal(t, 1, callback_count, seed)
	testify.Not_Equal(t, listener, accepted, seed)
	if accepted > 0 {
		sim_close(t, loop, driver, accepted)
	}
	sim_close(t, loop, driver, listener)
	nbio.IO_Deinit(loop)
	return accepted, operation_err
}

func sim_connect_lifecycle(t *testing.T, seed uint64) (snapshot string) {
	t.Helper()
	loop, driver, _ := sim_loop(seed)
	socket, open_err := nbio.Network_Socket_TCP(
		loop.Network, nbio.FAMILY_IPV4, sim_tcp_options(),
	)
	if !testify.No_Error(t, open_err) {
		return ""
	}
	called := false
	var connect_err error
	var completion nbio.Completion
	nbio.Network_Connect(loop.Network, &completion, socket,
		nbio.Address_IPV4([]byte{127, 0, 0, 1}, 8123), SIM_DEADLINE,
		func(completed nbio.Completion_Handle) {
			called = true
			connect_err = completed.Error
		})
	nbio.Driver_Run_Until(driver, SIM_DEADLINE, func() (finished bool) { return called })
	testify.True(t, called)
	open_before_close := sim_descriptor_open(loop)
	sim_close(t, loop, driver, socket)
	nbio.IO_Deinit(loop)
	return fmt.Sprintf(
		"open_before_close=%t outcome=%s",
		open_before_close, connect_outcome(connect_err),
	)
}

// Run one bounded simulated connect through its late-callback window and caller-owned close.
func sim_connect_with_timeout(
	t *testing.T, seed uint64, timeout time.Duration,
) (connect_err error) {
	t.Helper()
	loop, driver, _ := sim_loop(seed)
	socket, open_err := nbio.Network_Socket_TCP(
		loop.Network, nbio.FAMILY_IPV4, sim_tcp_options(),
	)
	if !testify.No_Error(t, open_err) {
		return open_err
	}
	callback_count := 0
	var completion nbio.Completion
	nbio.Network_Connect(loop.Network, &completion, socket,
		nbio.Address_IPV4([]byte{127, 0, 0, 1}, 8123), timeout,
		func(completed nbio.Completion_Handle) {
			callback_count++
			connect_err = completed.Error
		})
	nbio.Driver_Run_Until(driver, SIM_DEADLINE,
		func() (finished bool) { return callback_count > 0 })
	nbio.Driver_Run_For(driver, 16*time.NANOSECOND)
	testify.Equal(t, 1, callback_count, seed)
	testify.True(t, sim_descriptor_open(loop), seed)
	sim_close(t, loop, driver, socket)
	nbio.IO_Deinit(loop)
	return connect_err
}

// Report whether run still hold open descriptor. Deinit is only census surface expose, thus test
// state "still open" by watch of Deinit reject run.
func sim_descriptor_open(loop nbio.IO) (open bool) {
	defer func() { open = recover() != nil }()
	nbio.IO_Deinit(loop)
	return false
}

// Fresh loops make latency and equal-time ordering pure functions of the supplied seed.
func sim_transfer_once(
	t *testing.T, seed uint64, datagram bool, send bool, timeout time.Duration,
) (count int, operation_err error, completed_at time.Monotonic_Moment) {
	t.Helper()
	loop, driver, host := sim_loop(seed)
	var socket nbio.File
	var open_err error
	if datagram {
		socket, open_err = nbio.Network_Socket_UDP(
			loop.Network, nbio.FAMILY_IPV4, sim_udp_options(),
		)
	} else {
		socket, open_err = nbio.Network_Socket_TCP(
			loop.Network, nbio.FAMILY_IPV4, sim_tcp_options(),
		)
	}
	if !testify.No_Error(t, open_err) {
		return 0, open_err, 0
	}
	callback_count := 0
	var completion nbio.Completion
	callback := func(completed nbio.Completion_Handle) {
		callback_count++
		count = completed.Data
		operation_err = completed.Error
		completed_at = time.Clock_Now_Monotonic(host)
	}
	if send {
		nbio.Network_Send(
			loop.Network, &completion, socket, []byte("data"), timeout, callback,
		)
	} else {
		nbio.Network_Receive(
			loop.Network, &completion, socket, make([]byte, 4), timeout, callback,
		)
	}
	nbio.Driver_Run_Until(driver, SIM_DEADLINE,
		func() (finished bool) { return callback_count > 0 })
	nbio.Driver_Run_For(driver, 16*time.NANOSECOND)
	testify.Equal(t, 1, callback_count, seed)
	testify.True(t, sim_descriptor_open(loop), seed)
	sim_close(t, loop, driver, socket)
	nbio.IO_Deinit(loop)
	return count, operation_err, completed_at
}

// Panic is the synchronous rejection surface, so no callback or driver pass may be necessary.
func sim_network_timeout_rejected(
	t *testing.T, network nbio.Network, operation string, timeout time.Duration,
) {
	t.Helper()
	testify.Panics(t, func() {
		var completion nbio.Completion
		if operation == "accept" {
			nbio.Network_Accept(network, &completion, nbio.File(-1), timeout,
				func(_ nbio.Completion_Handle) {})
			return
		}
		if operation == "connect" {
			nbio.Network_Connect(
				network, &completion, nbio.File(-1), nbio.Address{}, timeout,
				func(_ nbio.Completion_Handle) {})
			return
		}
		if operation == "receive" {
			nbio.Network_Receive(network, &completion, nbio.File(-1), nil, timeout,
				func(_ nbio.Completion_Handle) {})
			return
		}
		nbio.Network_Send(network, &completion, nbio.File(-1), nil, timeout,
			func(_ nbio.Completion_Handle) {})
	}, operation, timeout)
}

// Run one storage operation through its late-delivery window and verify canceled effects.
func sim_storage_once(
	t *testing.T, seed uint64, operation string, timeout time.Duration,
) (data int, operation_err error, elapsed time.Monotonic_Moment) {
	t.Helper()
	loop, driver, host := sim_loop(seed)
	file, create_err := sim_open_options(t, loop, driver, operation, nbio.Open_At_Options{
		Access: nbio.OPEN_READ_WRITE, Create: true, Truncate: true, Permissions: 0o644,
	})
	if !testify.No_Error(t, create_err) {
		return 0, create_err, 0
	}
	sim_storage_write(t, loop, driver, file, []byte("base"))
	read_buffer := []byte("keep")
	start := time.Clock_Now_Monotonic(host)
	called := false
	var completion nbio.Completion
	callback := func(completed nbio.Completion_Handle) {
		data = completed.Data
		operation_err = completed.Error
		elapsed = time.Clock_Now_Monotonic(host) - start
		called = true
	}
	if operation == "read" {
		nbio.Storage_Read(
			loop.Storage, &completion, file, read_buffer, 0, timeout, callback,
		)
	} else if operation == "write" {
		nbio.Storage_Write(
			loop.Storage, &completion, file, []byte("next"), 0, timeout, callback,
		)
	} else {
		nbio.Storage_Fsync(loop.Storage, &completion, file, timeout, callback)
	}
	nbio.Driver_Run_Until(driver, SIM_DEADLINE, func() (finished bool) { return called })
	testify.True(t, called, seed, operation)
	nbio.Driver_Run_For(driver, 16*time.NANOSECOND)
	if operation_err == nbio.Deadline_Exceeded {
		if operation == "read" {
			testify.Equal(t, "keep", string(read_buffer), seed)
		}
		if operation == "write" {
			contents := sim_storage_read(t, loop, driver, file, len("base"))
			testify.Equal(t, "base", contents, seed)
		}
	}
	sim_close(t, loop, driver, file)
	nbio.IO_Deinit(loop)
	return data, operation_err, elapsed
}

// Use a long bound for fixture writes, so setup cannot consume the operation under test.
func sim_storage_write(
	t *testing.T, loop nbio.IO, driver nbio.Driver, file nbio.File, buffer []byte,
) {
	t.Helper()
	done := false
	var completion nbio.Completion
	nbio.Storage_Write(loop.Storage, &completion, file, buffer, 0, SIM_DEADLINE,
		func(completed nbio.Completion_Handle) {
			testify.No_Error(t, completed.Error)
			done = true
		})
	nbio.Driver_Run_Until(driver, SIM_DEADLINE, func() (finished bool) { return done })
	testify.True(t, done)
}

// Read fixture state through Storage, so the side-effect check uses the public surface.
func sim_storage_read(
	t *testing.T, loop nbio.IO, driver nbio.Driver, file nbio.File, size int,
) (contents string) {
	t.Helper()
	buffer := make([]byte, size)
	done := false
	var completion nbio.Completion
	nbio.Storage_Read(loop.Storage, &completion, file, buffer, 0, SIM_DEADLINE,
		func(completed nbio.Completion_Handle) {
			testify.No_Error(t, completed.Error)
			buffer = buffer[:completed.Data]
			done = true
		})
	nbio.Driver_Run_Until(driver, SIM_DEADLINE, func() (finished bool) { return done })
	testify.True(t, done)
	return string(buffer)
}

// Use an open file so only timeout validation can cause the synchronous rejection.
func sim_storage_timeout_rejected(
	t *testing.T, operation string, timeout time.Duration,
) {
	t.Helper()
	loop, driver, _ := sim_loop(0)
	file, create_err := sim_create(t, loop, driver, operation)
	if !testify.No_Error(t, create_err) {
		return
	}
	testify.Panics(t, func() {
		var completion nbio.Completion
		if operation == "read" {
			nbio.Storage_Read(loop.Storage, &completion, file, nil, 0, timeout,
				func(_ nbio.Completion_Handle) {})
			return
		}
		if operation == "write" {
			nbio.Storage_Write(loop.Storage, &completion, file, nil, 0, timeout,
				func(_ nbio.Completion_Handle) {})
			return
		}
		nbio.Storage_Fsync(
			loop.Storage, &completion, file, timeout, func(_ nbio.Completion_Handle) {},
		)
	}, operation, timeout)
	sim_close(t, loop, driver, file)
	nbio.IO_Deinit(loop)
}

// Build simulated loop, its driver, and read-only clock, seeded by seed. Test hold only IO,
// driver, and clock — never sim, which New_Simulated_IO keep to itself, thus run stay pure
// function of seed.
func sim_loop(seed uint64) (loop nbio.IO, driver nbio.Driver, host time.Clock) {
	memory := &sim_memory{}
	loop, driver = nbio.New_Simulated_IO(
		&memory.Sim, seed, time.NANOSECOND, sim_memory_view(memory),
	)
	return loop, driver, nbio.Sim_Clock_To_Clock(&memory.Sim.Clocks[0])
}

// Caller-owned capacity one simulated loop in this suite need, in one allocation.
type sim_memory struct {
	Sim                        nbio.Sim
	Nodes                      []nbio.Sim_Node
	Node_Names                 []byte
	Node_Contents              []byte
	Descriptors                []nbio.Sim_Descriptor
	Descriptor_Address_Storage []byte
	Operations                 []nbio.Sim_Operation
	Operation_Address_Storage  []byte
	Queue                      []*nbio.Completion
	Events                     []nbio.Virtual_Event
	Clocks                     []nbio.Sim_Clock
}

func sim_memory_view(memory *sim_memory) (view nbio.Sim_Memory) {
	sim_memory_initialize(memory)
	return nbio.Sim_Memory{
		Nodes:       memory.Nodes,
		Descriptors: memory.Descriptors,
		Operations:  memory.Operations,
		Queue:       memory.Queue,
		Events:      memory.Events,
		Clocks:      memory.Clocks,
	}
}

func sim_memory_initialize(memory *sim_memory) {
	if memory.Nodes != nil {
		return
	}
	memory.Nodes = make([]nbio.Sim_Node, SIM_NODE_CAPACITY)
	memory.Node_Names = make([]byte, SIM_NODE_CAPACITY*nbio.SIM_PATH_COMPONENT_BYTES_MAXIMUM)
	memory.Node_Contents = make([]byte, SIM_NODE_CAPACITY*nbio.SIM_FILE_BYTES_MAXIMUM)
	for index := range memory.Nodes {
		name_start := index * nbio.SIM_PATH_COMPONENT_BYTES_MAXIMUM
		name_end := name_start + nbio.SIM_PATH_COMPONENT_BYTES_MAXIMUM
		content_start := index * nbio.SIM_FILE_BYTES_MAXIMUM
		content_end := content_start + nbio.SIM_FILE_BYTES_MAXIMUM
		memory.Nodes[index].Name = memory.Node_Names[name_start:name_end:name_end]
		contents := memory.Node_Contents[content_start:content_end]
		memory.Nodes[index].Contents = contents[:len(contents):len(contents)]
	}
	memory.Descriptors = make([]nbio.Sim_Descriptor, SIM_DESCRIPTOR_CAPACITY)
	memory.Descriptor_Address_Storage = make(
		[]byte, SIM_DESCRIPTOR_CAPACITY*2*nbio.IPV6_ADDRESS_BYTES,
	)
	for index := range memory.Descriptors {
		local_start := index * 2 * nbio.IPV6_ADDRESS_BYTES
		local_end := local_start + nbio.IPV6_ADDRESS_BYTES
		peer_start := local_end
		peer_end := peer_start + nbio.IPV6_ADDRESS_BYTES
		local := memory.Descriptor_Address_Storage[local_start:local_end]
		peer := memory.Descriptor_Address_Storage[peer_start:peer_end]
		memory.Descriptors[index].Address_IP = local[:len(local):len(local)]
		memory.Descriptors[index].Peer_IP = peer[:len(peer):len(peer)]
	}
	memory.Operations = make([]nbio.Sim_Operation, SIM_CONCURRENT_OPERATION_CAPACITY)
	memory.Operation_Address_Storage = make(
		[]byte, SIM_CONCURRENT_OPERATION_CAPACITY*nbio.IPV6_ADDRESS_BYTES,
	)
	for index := range memory.Operations {
		start := index * nbio.IPV6_ADDRESS_BYTES
		end := start + nbio.IPV6_ADDRESS_BYTES
		address := memory.Operation_Address_Storage[start:end]
		memory.Operations[index].Address_IP = address[:len(address):len(address)]
	}
	memory.Queue = make([]*nbio.Completion, SIM_TIMELINE_CAPACITY)
	memory.Events = make([]nbio.Virtual_Event, SIM_EVENT_CAPACITY)
	memory.Clocks = make([]nbio.Sim_Clock, SIM_CLOCK_CAPACITY)
}

// Two concurrent network operations each arm work and a competing timeout.
const SIM_CONCURRENT_OPERATION_CAPACITY = 2
const SIM_COMPLETION_PER_OPERATION = 2
const SIM_TIMELINE_CAPACITY = SIM_CONCURRENT_OPERATION_CAPACITY * SIM_COMPLETION_PER_OPERATION

// Simulator memory leaves room for generated fixture and paths each test creates.
const SIM_NODE_CAPACITY = 32

// Tests hold listener, connector, accepted socket, and file descriptors concurrently.
const SIM_DESCRIPTOR_CAPACITY = 8

// Tests use no cross-thread event, but the timeline requires bounded event ownership.
const SIM_EVENT_CAPACITY = 1

// Two clock views per simulated loop: enough to show views share ticks and differ by skew.
const SIM_CLOCK_CAPACITY = 2

// Timeline half alone, for tests that arm timers and never touch an endpoint.
func sim_timeline() (loop nbio.Timeline, driver nbio.Driver, host time.Clock) {
	surface, driver, host := sim_loop(0)
	return surface.Timeline, driver, host
}

// Open path for read through Open_At. Drive loop until descriptor arrive.
func sim_open(
	t *testing.T, loop nbio.IO, driver nbio.Driver, path string,
) (file nbio.File, err error) {
	t.Helper()
	return sim_open_options(t, loop, driver, path, nbio.Open_At_Options{
		Access: nbio.OPEN_READ_ONLY,
	})
}

// Make or truncate path for write through Open_At.
func sim_create(
	t *testing.T, loop nbio.IO, driver nbio.Driver, path string,
) (file nbio.File, err error) {
	t.Helper()
	return sim_open_options(t, loop, driver, path, nbio.Open_At_Options{
		Access: nbio.OPEN_WRITE_ONLY, Create: true, Truncate: true, Permissions: 0o644,
	})
}

// Submit one Open_At with caller options and drive loop until it retire.
func sim_open_options(
	t *testing.T, loop nbio.IO, driver nbio.Driver, path string, options nbio.Open_At_Options,
) (file nbio.File, err error) {
	t.Helper()
	done := false
	var completion nbio.Completion
	nbio.Storage_Open_At(loop.Storage, &completion, nbio.DIRECTORY_CURRENT, path, options,
		func(completed nbio.Completion_Handle) {
			file = nbio.File(completed.Data)
			err = completed.Error
			done = true
		})
	nbio.Driver_Run_Until(driver, SIM_DEADLINE, func() (finished bool) { return done })
	testify.True(t, done, path)
	return file, err
}

// Release file through asynchronous Close and drive loop until close retire, thus caller reach
// Deinit with nothing still open.
func sim_close(t *testing.T, loop nbio.IO, driver nbio.Driver, file nbio.File) {
	t.Helper()
	closed := false
	var completion nbio.Completion
	nbio.IO_Close(loop, &completion, file, func(completed nbio.Completion_Handle) {
		testify.No_Error(t, completed.Error)
		closed = true
	})
	nbio.Driver_Run_Until(driver, SIM_DEADLINE, func() (finished bool) { return closed })
	testify.True(t, closed, file)
}

// Run_Until cap of sim tests, in virtual time: ample for operation that finish in handful of
// grains, while never-satisfied predicate fail after this many cheap grains instead of spin of
// sim forever.
const SIM_DEADLINE = time.MICROSECOND

// A finite connect timeout must coexist with terminal outcomes and caller-owned teardown.
func Test_Connect_Timeout_Sim(t *testing.T) {
	saw_operation := false
	saw_timeout := false
	for seed := uint64(0); seed < 64; seed++ {
		short := sim_connect_with_timeout(t, seed, time.NANOSECOND)
		if short == nbio.Deadline_Exceeded {
			saw_timeout = true
		}
		long := sim_connect_with_timeout(t, seed, SIM_DEADLINE)
		if long != nbio.Deadline_Exceeded {
			saw_operation = true
		}
	}
	testify.True(t, saw_operation)
	testify.True(t, saw_timeout)
}

// One sweep must reach both sides because a fixed success seed cannot prove the bound exists.
func Test_Network_Transfer_Timeouts_Sim(t *testing.T) {
	tests := []struct {
		Name     string
		Datagram bool
		Send     bool
	}{
		{Name: "TCP receive"},
		{Name: "TCP send", Send: true},
		{Name: "UDP receive", Datagram: true},
		{Name: "UDP send", Datagram: true, Send: true},
	}
	for _, test := range tests {
		saw_operation := false
		saw_timeout := false
		for seed := uint64(0); seed < 128; seed++ {
			count, operation_err, _ := sim_transfer_once(
				t, seed, test.Datagram, test.Send, time.NANOSECOND)
			if operation_err == nbio.Deadline_Exceeded {
				saw_timeout = true
				testify.Zero(t, count, test.Name)
				continue
			}
			testify.No_Error(t, operation_err, test.Name)
			saw_operation = true
			testify.Equal(t, 4, count, test.Name)
		}
		testify.True(t, saw_operation, test.Name)
		testify.True(t, saw_timeout, test.Name)
	}
}

// Equal simulated times need both kernel-valid orders, not one artificial global tie rule.
func Test_Network_Transfer_Equal_Time_Order_Sim(t *testing.T) {
	saw_operation := false
	saw_timeout := false
	for seed := uint64(0); seed < 256; seed++ {
		_, _, latency := sim_transfer_once(t, seed, false, false, SIM_DEADLINE)
		if latency <= 0 {
			continue
		}
		count, operation_err, _ := sim_transfer_once(
			t, seed, false, false, time.Duration(latency))
		if operation_err == nbio.Deadline_Exceeded {
			saw_timeout = true
			testify.Zero(t, count, seed)
		} else {
			saw_operation = true
			testify.No_Error(t, operation_err, seed)
			testify.Equal(t, 4, count, seed)
		}
	}
	testify.True(t, saw_operation)
	testify.True(t, saw_timeout)
}

// Every entry rejects an absent bound before it can borrow a descriptor.
func Test_Network_Rejects_Disabled_Timeouts_Sim(t *testing.T) {
	for _, operation := range []string{"accept", "connect", "receive", "send"} {
		for _, timeout := range []time.Duration{0, -time.NANOSECOND} {
			loop, _, _ := sim_loop(0)
			sim_network_timeout_rejected(t, loop.Network, operation, timeout)
			nbio.IO_Deinit(loop)
		}
	}
}

// A timeout must stop the modeled storage effect, or the callback lies about cancellation.
func Test_Storage_Timeouts_Sim(t *testing.T) {
	for _, operation := range []string{"read", "write", "fsync"} {
		saw_operation := false
		saw_timeout := false
		for seed := uint64(0); seed < 128; seed++ {
			data, operation_err, _ := sim_storage_once(
				t, seed, operation, time.NANOSECOND,
			)
			if operation_err == nbio.Deadline_Exceeded {
				saw_timeout = true
				testify.Zero(t, data, operation)
				continue
			}
			testify.No_Error(t, operation_err, operation, seed)
			saw_operation = true
		}
		testify.True(t, saw_operation, operation)
		testify.True(t, saw_timeout, operation)
	}
}

// A separate stream must make both equal-time storage orders reachable without moving network.
func Test_Storage_Equal_Time_Order_Sim(t *testing.T) {
	saw_operation := false
	saw_timeout := false
	for seed := uint64(0); seed < 256; seed++ {
		_, _, latency := sim_storage_once(t, seed, "read", SIM_DEADLINE)
		if latency <= 0 {
			continue
		}
		data, operation_err, _ := sim_storage_once(
			t, seed, "read", time.Duration(latency),
		)
		if operation_err == nbio.Deadline_Exceeded {
			saw_timeout = true
			testify.Zero(t, data, seed)
			continue
		}
		testify.No_Error(t, operation_err, seed)
		saw_operation = true
	}
	testify.True(t, saw_operation)
	testify.True(t, saw_timeout)
}

// An empty transfer has no storage effect that can consume a timeout or a latency draw.
func Test_Storage_Empty_Transfer_Completes_On_First_Grain_Sim(t *testing.T) {
	for _, operation := range []string{"read", "write"} {
		loop, driver, host := sim_loop(0)
		options := nbio.Open_At_Options{
			Access: nbio.OPEN_READ_WRITE, Create: true, Truncate: true,
			Permissions: 0o644,
		}
		file, create_err := sim_open_options(t, loop, driver, operation, options)
		if !testify.No_Error(t, create_err, operation) {
			continue
		}
		start := time.Clock_Now_Monotonic(host)
		called := false
		var completion nbio.Completion
		callback := func(completed nbio.Completion_Handle) {
			testify.Zero(t, completed.Data, operation)
			testify.No_Error(t, completed.Error, operation)
			called = true
		}
		if operation == "read" {
			nbio.Storage_Read(
				loop.Storage, &completion, file, nil, 0, time.NANOSECOND, callback,
			)
		} else {
			nbio.Storage_Write(
				loop.Storage, &completion, file, nil, 0, time.NANOSECOND, callback,
			)
		}
		testify.False(t, called, operation)
		nbio.Driver_Run_Until(driver, time.NANOSECOND,
			func() (finished bool) { return called })
		testify.True(t, called, operation)
		want := start + time.Monotonic_Moment(time.NANOSECOND)
		testify.Equal(t, want, time.Clock_Now_Monotonic(host), operation)
		sim_close(t, loop, driver, file)
		nbio.IO_Deinit(loop)
	}
}

// The bound is required on every backend, even where Darwin cannot enforce elapsed time.
func Test_Storage_Rejects_Disabled_Timeouts_Sim(t *testing.T) {
	for _, operation := range []string{"read", "write", "fsync"} {
		for _, timeout := range []time.Duration{0, -time.NANOSECOND} {
			sim_storage_timeout_rejected(t, operation, timeout)
		}
	}
}

// One slot leaves no second entry, thus a nested submit proves retirement frees before it runs.
const SIM_EFFECTS_OPERATION_CAPACITY = 1

// Test root keeps caller-owned state alive beside the vtable that points into it.
type sim_effects_harness struct {
	Memory                    sim_memory
	Operations                []nbio.Sim_Operation
	Operation_Address_Storage []byte
}

func sim_effects_loop(
	harness *sim_effects_harness,
) (loop nbio.IO, driver nbio.Driver) {
	memory := sim_memory_view(&harness.Memory)
	if harness.Operations == nil {
		harness.Operations = make([]nbio.Sim_Operation, SIM_EFFECTS_OPERATION_CAPACITY)
		harness.Operation_Address_Storage = make(
			[]byte, SIM_EFFECTS_OPERATION_CAPACITY*nbio.IPV6_ADDRESS_BYTES,
		)
		for index := range harness.Operations {
			start := index * nbio.IPV6_ADDRESS_BYTES
			end := start + nbio.IPV6_ADDRESS_BYTES
			address := harness.Operation_Address_Storage[start:end]
			harness.Operations[index].Address_IP = address[:len(address):len(address)]
		}
	}
	memory.Operations = harness.Operations
	return nbio.New_Simulated_IO(&harness.Memory.Sim, 0, time.NANOSECOND, memory)
}

// Test_Sim_Effects_Operation_Capacity verifies the sole slot rejects a second owner before it
// arms that caller's completion.
func Test_Sim_Effects_Operation_Capacity(t *testing.T) {
	harness := sim_effects_harness{}
	loop, driver := sim_effects_loop(&harness)
	var first nbio.Completion
	nbio.IO_Watch_Signal(loop, &first, nbio.SIGNAL_TERMINATE, SIM_DEADLINE,
		sim_allocation_signal_callback)
	var rejected nbio.Completion
	nbio.IO_Spawn(loop, &rejected, nbio.Process_Request{Path: "true"}, SIM_DEADLINE,
		sim_allocation_process_callback)
	testify.Error(t, rejected.Error)
	testify.Nil(t, rejected.Backend)
	nbio.Driver_Run_For(driver, SIM_DEADLINE)
	testify.Nil(t, first.Backend)
	nbio.IO_Deinit(loop)
}

// Test_Sim_Effects_Callback_Can_Submit verifies retirement releases the sole slot before the
// callback runs, so that callback can arm the next operation.
func Test_Sim_Effects_Callback_Can_Submit(t *testing.T) {
	harness := sim_effects_harness{}
	loop, driver := sim_effects_loop(&harness)
	callback_count := 0
	var signal_completion nbio.Completion
	var spawn_completion nbio.Completion
	nbio.IO_Watch_Signal(
		loop, &signal_completion, nbio.SIGNAL_TERMINATE, SIM_DEADLINE,
		func(_ *nbio.Completion, _ nbio.Signal, signal_err error) {
			testify.No_Error(t, signal_err)
			callback_count++
			nbio.IO_Spawn(
				loop, &spawn_completion, nbio.Process_Request{Path: "true"},
				SIM_DEADLINE,
				func(_ *nbio.Completion, _ nbio.Process_Result, spawn_err error) {
					testify.No_Error(t, spawn_err)
					callback_count++
				},
			)
		},
	)
	nbio.Driver_Run_For(driver, SIM_DEADLINE)
	testify.Equal(t, 2, callback_count)
	testify.Nil(t, signal_completion.Backend)
	testify.Nil(t, spawn_completion.Backend)
	nbio.IO_Deinit(loop)
}

// Runs one bounded simulated spawn past its modeled completion time and reports its sole result.
func sim_spawn_with_deadline(
	t *testing.T, seed uint64, deadline time.Duration,
) (spawn_err error) {
	t.Helper()
	loop, driver, _ := sim_loop(seed)
	callback_count := 0
	var completion nbio.Completion
	nbio.IO_Spawn(loop, &completion, nbio.Process_Request{Path: "true"}, deadline, func(
		_ *nbio.Completion, _ nbio.Process_Result, err error,
	) {
		callback_count++
		spawn_err = err
	})
	nbio.Driver_Run_Until(driver, SIM_DEADLINE,
		func() (finished bool) { return callback_count > 0 })
	nbio.Driver_Run_For(driver, 16*time.NANOSECOND)
	testify.Equal(t, 1, callback_count, seed)
	nbio.IO_Deinit(loop)
	return spawn_err
}

// Test_Spawn_Deadline_Sim verifies a finite spawn deadline wins a latency tie and retires once.
func Test_Spawn_Deadline_Sim(t *testing.T) {
	saw_deadline := false
	saw_tie := false
	for seed := uint64(0); seed < 64; seed++ {
		at_deadline := sim_spawn_with_deadline(t, seed, 4*time.NANOSECOND)
		after_deadline := sim_spawn_with_deadline(t, seed, 5*time.NANOSECOND)
		if at_deadline == nbio.Deadline_Exceeded {
			saw_deadline = true
		}
		if at_deadline == nbio.Deadline_Exceeded {
			if after_deadline != nbio.Deadline_Exceeded {
				saw_tie = true
			}
		}
	}
	testify.True(t, saw_deadline)
	testify.True(t, saw_tie)
}

// Fuzz_Sim_Effects keeps seed whole so every modeled signal grain and exit outcome stays
// reachable.
func Fuzz_Sim_Effects(f *testing.F) {
	f.Add(uint64(0))
	f.Add(uint64(1))
	f.Add(^uint64(0))
	f.Fuzz(func(t *testing.T, seed uint64) {
		loop, driver, _ := sim_loop(seed)
		fuzz_signal(t, loop, driver)
		fuzz_spawn(t, loop, driver)
		nbio.IO_Deinit(loop)
	})
}

func fuzz_signal(t *testing.T, loop nbio.IO, driver nbio.Driver) {
	t.Helper()
	callback_count := 0
	delivered := nbio.SIGNAL_EXPIRED
	var operation_err error
	var completion nbio.Completion
	nbio.IO_Watch_Signal(
		loop, &completion, nbio.SIGNAL_INTERRUPT, SIM_FUZZ_OPERATION_DEADLINE,
		func(_ *nbio.Completion, signal nbio.Signal, err error) {
			callback_count++
			delivered = signal
			operation_err = err
		},
	)
	nbio.Driver_Run_For(driver, SIM_DEADLINE)
	testify.Equal(t, 1, callback_count, operation_err)
	testify.Nil(t, completion.Backend)
	if operation_err == nil {
		testify.Equal(t, nbio.SIGNAL_INTERRUPT, delivered)
		return
	}
	testify.Error_Is(t, operation_err, nbio.Deadline_Exceeded)
	testify.Equal(t, nbio.SIGNAL_EXPIRED, delivered)
}

func fuzz_spawn(t *testing.T, loop nbio.IO, driver nbio.Driver) {
	t.Helper()
	callback_count := 0
	result := nbio.Process_Result{}
	var operation_err error
	var completion nbio.Completion
	nbio.IO_Spawn(
		loop, &completion, nbio.Process_Request{Path: "true"},
		SIM_FUZZ_OPERATION_DEADLINE,
		func(_ *nbio.Completion, spawned nbio.Process_Result, err error) {
			callback_count++
			result = spawned
			operation_err = err
		},
	)
	nbio.Driver_Run_For(driver, SIM_DEADLINE)
	testify.Equal(t, 1, callback_count, operation_err)
	testify.Nil(t, completion.Backend)
	if operation_err == nil {
		testify.True(t, result.Exit == 0 || result.Exit == 1, result.Exit)
		return
	}
	testify.Error_Is(t, operation_err, nbio.Deadline_Exceeded)
	testify.Zero(t, result.Exit)
}

// Mid-range deadline makes seed sweep reach both modeled event and deadline winner.
const SIM_FUZZ_OPERATION_DEADLINE = 4 * time.NANOSECOND

// Independent operation and timeout inputs keep one corpus case from suppressing another path.
func Fuzz_Sim_Network_Timeouts(f *testing.F) {
	f.Add(uint64(0), uint8(0), uint8(0), false)
	f.Add(uint64(1), uint8(1), uint8(1), false)
	f.Add(uint64(2), uint8(2), uint8(2), false)
	f.Add(uint64(3), uint8(3), uint8(7), true)
	f.Fuzz(func(
		t *testing.T, seed uint64, operation uint8, timeout_grain uint8, datagram bool,
	) {
		timeout := time.Duration(timeout_grain%uint8(nbio.SIM_LATENCY_GRAINS)+1) *
			time.NANOSECOND
		if operation%4 == 0 {
			sim_accept_once(t, seed, timeout)
			return
		}
		if operation%4 == 1 {
			sim_connect_with_timeout(t, seed, timeout)
			return
		}
		if operation%4 == 2 {
			sim_transfer_once(t, seed, datagram, false, timeout)
			return
		}
		sim_transfer_once(t, seed, datagram, true, timeout)
	})
}

// Independent storage kind and timeout inputs keep the fuzzer on the complete seed space.
func Fuzz_Sim_Storage_Timeouts(f *testing.F) {
	f.Add(uint64(0), uint8(0), uint8(0))
	f.Add(uint64(1), uint8(1), uint8(1))
	f.Add(uint64(2), uint8(2), uint8(7))
	f.Fuzz(func(t *testing.T, seed uint64, operation uint8, timeout_grain uint8) {
		timeout := time.Duration(timeout_grain%uint8(nbio.SIM_LATENCY_GRAINS)+1) *
			time.NANOSECOND
		names := [...]string{"read", "write", "fsync"}
		sim_storage_once(t, seed, names[operation%uint8(len(names))], timeout)
	})
}

// The test owns the Driver for Stream functions that select the virtual Timeline.
type stream_harness_state struct {
	Timeline nbio.Timeline
	Driver   nbio.Driver
}

// One virtual Timeline tests delayed Stream functions without a production Driver.
func stream_harness(t *testing.T) (harness *stream_harness_state) {
	t.Helper()
	timeline, driver, _ := sim_timeline()
	harness = &stream_harness_state{Timeline: timeline, Driver: driver}
	t.Cleanup(func() { nbio.Driver_Deinit(harness.Driver) })
	return harness
}

// This join accepts the callback time that each concrete Stream selects.
func stream_result(
	t *testing.T, harness *stream_harness_state,
	submit func(completion *nbio.Completion, callback nbio.Callback),
) (data int, err error) {
	t.Helper()
	called := false
	var completion nbio.Completion
	submit(&completion, func(completed nbio.Completion_Handle) {
		data = completed.Data
		err = completed.Error
		called = true
	})
	if called {
		return data, err
	}
	completed, drive_err := nbio.Driver_Run_Until(harness.Driver,
		10*time.NANOSECOND, func() (finished bool) { return called },
	)
	testify.No_Error(t, drive_err)
	testify.True(t, completed)
	return data, err
}

func stream_read(
	t *testing.T, harness *stream_harness_state, stream nbio.Stream, buffer []byte,
) (count int, err error) {
	t.Helper()
	return stream_result(t, harness, func(
		completion *nbio.Completion, callback nbio.Callback,
	) {
		nbio.Read(stream, completion, buffer, callback)
	})
}

func stream_read_at(
	t *testing.T, harness *stream_harness_state, stream nbio.Stream, buffer []byte,
	offset int64,
) (count int, err error) {
	t.Helper()
	return stream_result(t, harness, func(
		completion *nbio.Completion, callback nbio.Callback,
	) {
		nbio.Read_At(stream, completion, buffer, offset, callback)
	})
}

func stream_write(
	t *testing.T, harness *stream_harness_state, stream nbio.Stream, buffer []byte,
) (count int, err error) {
	t.Helper()
	return stream_result(t, harness, func(
		completion *nbio.Completion, callback nbio.Callback,
	) {
		nbio.Write(stream, completion, buffer, callback)
	})
}

func stream_write_at(
	t *testing.T, harness *stream_harness_state, stream nbio.Stream, buffer []byte,
	offset int64,
) (count int, err error) {
	t.Helper()
	return stream_result(t, harness, func(
		completion *nbio.Completion, callback nbio.Callback,
	) {
		nbio.Write_At(stream, completion, buffer, offset, callback)
	})
}

func stream_seek(
	t *testing.T, harness *stream_harness_state, stream nbio.Stream, offset int64,
	whence nbio.Seek_From,
) (position int, err error) {
	t.Helper()
	return stream_result(t, harness, func(
		completion *nbio.Completion, callback nbio.Callback,
	) {
		nbio.Seek(stream, completion, offset, whence, callback)
	})
}

func stream_size(
	t *testing.T, harness *stream_harness_state, stream nbio.Stream,
) (size int, err error) {
	t.Helper()
	return stream_result(t, harness, func(
		completion *nbio.Completion, callback nbio.Callback,
	) {
		nbio.Size(stream, completion, callback)
	})
}

func stream_query(
	t *testing.T, harness *stream_harness_state, stream nbio.Stream,
) (modes nbio.Stream_Mode_Set, err error) {
	t.Helper()
	data, query_err := stream_result(t, harness, func(
		completion *nbio.Completion, callback nbio.Callback,
	) {
		nbio.Query(stream, completion, callback)
	})
	return nbio.Stream_Mode_Set(data), query_err
}

func stream_flush(
	t *testing.T, harness *stream_harness_state, stream nbio.Stream,
) (data int, err error) {
	t.Helper()
	return stream_result(t, harness, func(
		completion *nbio.Completion, callback nbio.Callback,
	) {
		nbio.Flush(stream, completion, callback)
	})
}

func stream_close(
	t *testing.T, harness *stream_harness_state, stream nbio.Stream,
) (data int, err error) {
	t.Helper()
	return stream_result(t, harness, func(
		completion *nbio.Completion, callback nbio.Callback,
	) {
		nbio.Close(stream, completion, callback)
	})
}

func stream_destroy(
	t *testing.T, harness *stream_harness_state, stream nbio.Stream,
) (data int, err error) {
	t.Helper()
	return stream_result(t, harness, func(
		completion *nbio.Completion, callback nbio.Callback,
	) {
		nbio.Destroy(stream, completion, callback)
	})
}

// Builds a memory stream over bytes the test owns. Test_Stream_Memory writes the constructor
// out in full; every other test uses this, because the state is not what it is testing.
func memory_stream(
	harness *stream_harness_state, memory []byte,
) (stream nbio.Stream) {
	state := nbio.Stream_Memory{Memory: memory}
	return nbio.Memory_To_Stream(&state)
}

// Builds a discard stream for the tests that need a stream answering only write modes.
func discard_stream(harness *stream_harness_state) (stream nbio.Stream) {
	state := nbio.Stream_Discard{}
	return nbio.Discard_To_Stream(&state)
}

// Separate byte and offset inputs let the fuzzer cross every memory boundary without one input
// deciding whether another path exists.
func Fuzz_Stream_Memory(f *testing.F) {
	f.Add([]byte("abc"), []byte("xy"), int64(0), false)
	f.Add([]byte{}, []byte{}, int64(0), true)
	f.Add([]byte("a"), []byte("bc"), int64(1), true)
	f.Fuzz(func(
		t *testing.T, initial []byte, transfer []byte, offset int64, write bool,
	) {
		if len(initial) > bits.KIBIBYTE_BYTES {
			return
		}
		if len(transfer) > bits.KIBIBYTE_BYTES {
			return
		}
		memory := append([]byte(nil), initial...)
		before := append([]byte(nil), initial...)
		harness := stream_harness(t)
		stream := memory_stream(harness, memory)
		if write {
			stream_memory_fuzz_write(
				t, harness, stream, memory, before, transfer, offset,
			)
			return
		}
		stream_memory_fuzz_read(t, harness, stream, before, len(transfer), offset)
	})
}

// Positioned writes make the expected mutation independent of cursor history.
func stream_memory_fuzz_write(
	t *testing.T, harness *stream_harness_state, stream nbio.Stream, memory []byte,
	before []byte, transfer []byte, offset int64,
) {
	t.Helper()
	count, err := stream_write_at(t, harness, stream, transfer, offset)
	if offset < 0 {
		testify.Error_Is(t, err, nbio.Stream_Invalid_Offset)
		testify.Equal(t, before, memory)
		return
	}
	if offset > int64(len(memory)) {
		testify.Error_Is(t, err, nbio.Stream_Invalid_Offset)
		testify.Equal(t, before, memory)
		return
	}
	expected := append([]byte(nil), before...)
	expected_count := copy(expected[int(offset):], transfer)
	testify.Equal(t, expected_count, count)
	if expected_count < len(transfer) {
		testify.Error_Is(t, err, nbio.Stream_Short_Write)
	} else {
		testify.No_Error(t, err)
	}
	testify.Equal(t, expected, memory)
}

// Positioned reads expose invalid, terminal, empty, partial, and complete ranges directly.
func stream_memory_fuzz_read(
	t *testing.T, harness *stream_harness_state, stream nbio.Stream, memory []byte,
	buffer_size int, offset int64,
) {
	t.Helper()
	buffer := make([]byte, buffer_size)
	count, err := stream_read_at(t, harness, stream, buffer, offset)
	if offset < 0 {
		testify.Error_Is(t, err, nbio.Stream_Invalid_Offset)
		return
	}
	if offset > int64(len(memory)) {
		testify.Error_Is(t, err, nbio.Stream_Invalid_Offset)
		return
	}
	if offset == int64(len(memory)) {
		testify.Zero(t, count)
		testify.Error_Is(t, err, nbio.Stream_EOF)
		return
	}
	expected_count := copy(make([]byte, buffer_size), memory[int(offset):])
	testify.Equal(t, expected_count, count)
	testify.No_Error(t, err)
	testify.Equal(t, memory[int(offset):int(offset)+count], buffer[:count])
}

// POSIX_FILE_MODE_HIGH_MAXIMUM is every non-kind bit one platform word carries: setuid, setgid,
// sticky, and the nine permission bits.
const POSIX_FILE_MODE_HIGH_MAXIMUM uint16 = 0o7777

// Every POSIX kind nibble this repository decodes.
func posix_file_mode_kinds() (kinds []uint16) {
	return []uint16{
		nbio.POSIX_FILE_MODE_NAMED_PIPE, nbio.POSIX_FILE_MODE_CHARACTER_DEVICE,
		nbio.POSIX_FILE_MODE_DIRECTORY, nbio.POSIX_FILE_MODE_DEVICE,
		nbio.POSIX_FILE_MODE_REGULAR, nbio.POSIX_FILE_MODE_SYMBOLIC_LINK,
		nbio.POSIX_FILE_MODE_SOCKET,
	}
}

// Directory half of the permission proof, thus each test function stay inside the length cap.
func sim_mkdir_permissions(t *testing.T, loop nbio.IO, driver nbio.Driver) {
	t.Helper()
	for index, permissions := range file_permissions_domain() {
		path := "/directory" + string(rune('0'+index))
		made := false
		var completion nbio.Completion
		nbio.Storage_Mkdir_At(
			loop.Storage, &completion, nbio.DIRECTORY_CURRENT, path, permissions,
			func(completed nbio.Completion_Handle) {
				testify.No_Error(t, completed.Error)
				made = true
			})
		nbio.Driver_Run_Until(driver, SIM_DEADLINE, func() (finished bool) { return made })
		status, _ := nbio.Storage_Status(loop.Storage, path)
		testify.True(t, nbio.File_Mode_Is_Directory(status.Mode))
		testify.Equal(t, nbio.File_Mode(permissions&^nbio.SIM_UMASK),
			status.Mode&^nbio.FILE_MODE_DIRECTORY)
	}
}

// Boundary values of the creation permission domain: both ends and the two lowest bits.
func file_permissions_domain() (domain []nbio.File_Permissions) {
	return []nbio.File_Permissions{
		nbio.File_Permissions(nbio.FILE_PERMISSIONS_MINIMUM), 1, 2,
		nbio.FILE_PERMISSIONS_VALID,
	}
}

// Assert every link-facing contract for one generated link path.
func sim_assert_link(
	t *testing.T, loop nbio.IO, driver nbio.Driver, nodes []nbio.Sim_Node, path string,
) {
	t.Helper()
	status, status_err := nbio.Storage_Status(loop.Storage, path)
	testify.No_Error(t, status_err)
	testify.True(t, nbio.File_Mode_Is_Symbolic_Link(status.Mode))
	target := [nbio.SIM_PATH_COMPONENT_BYTES_MAXIMUM]byte{}
	count, read_err := nbio.Storage_Read_Link(loop.Storage, path, target[:])
	testify.No_Error(t, read_err)
	testify.True(t, count > 0)
	testify.Equal(t, int64(count), status.Size)
	// Following the link reaches the sibling it names, thus resolution is not a no-op.
	followed, followed_err := nbio.Storage_Status(
		loop.Storage, sim_node_directory(nodes, path)+string(target[:count]),
	)
	testify.No_Error(t, followed_err)
	testify.True(t, followed.Exists)
	_, follow_err := sim_open_options(t, loop, driver, path, nbio.Open_At_Options{
		Access: nbio.OPEN_READ_ONLY, Flags: nbio.OPEN_AT_NO_FOLLOW,
	})
	testify.Error_Is(t, follow_err, nbio.Symbolic_Link_Not_Followed)
	// Opening the link itself proves the resolver follows the stored target.
	opened, open_err := sim_open_options(t, loop, driver, path, nbio.Open_At_Options{
		Access: nbio.OPEN_READ_ONLY,
	})
	if testify.No_Error(t, open_err, path, string(target[:count])) {
		sim_close(t, loop, driver, opened)
	}
}

// First generated symbolic link in node storage, as an absolute path.
func sim_first_link_path(nodes []nbio.Sim_Node) (path string, found bool) {
	for index := 1; index < len(nodes); index++ {
		if !nodes[index].Used {
			continue
		}
		if !nbio.File_Mode_Is_Symbolic_Link(nodes[index].Mode) {
			continue
		}
		return sim_node_path(nodes, index), true
	}
	return "", false
}

// Absolute path of one node, built by walking its parent chain.
func sim_node_path(nodes []nbio.Sim_Node, index int) (path string) {
	for index > 0 {
		node := nodes[index]
		path = "/" + string(node.Name[:node.Name_Count]) + path
		index = node.Parent
	}
	return path
}

// Directory prefix of one absolute path, trailing separator kept.
func sim_node_directory(nodes []nbio.Sim_Node, path string) (directory string) {
	for position := len(path) - 1; position >= 0; position-- {
		if path[position] == '/' {
			return path[:position+1]
		}
	}
	return "/"
}

// Build one simulated loop and keep its node storage visible, thus a test can assert what the
// seeded generator actually made rather than guess at generated names.
func sim_loop_nodes(
	seed uint64,
) (loop nbio.IO, driver nbio.Driver, nodes []nbio.Sim_Node) {
	memory := &sim_memory{}
	loop, driver = nbio.New_Simulated_IO(
		&memory.Sim, seed, time.NANOSECOND, sim_memory_view(memory),
	)
	return loop, driver, memory.Nodes[:]
}

func sim_read_rejected(
	t *testing.T, loop nbio.IO, driver nbio.Driver, file nbio.File, buffer []byte,
) {
	t.Helper()
	before := string(buffer)
	var completion nbio.Completion
	done := false
	nbio.Storage_Read(loop.Storage, &completion, file, buffer, 0, SIM_DEADLINE,
		func(_ nbio.Completion_Handle) { done = true })
	testify.False(t, done)
	nbio.Driver_Run_Until(driver, SIM_DEADLINE, func() (finished bool) { return done })
	testify.True(t, done)
	testify.Error(t, completion.Error)
	testify.Zero(t, completion.Data)
	testify.Equal(t, before, string(buffer))
}

func sim_write_at(
	t *testing.T, loop nbio.IO, driver nbio.Driver, file nbio.File, buffer []byte, offset int64,
) (data int, err error) {
	t.Helper()
	var completion nbio.Completion
	done := false
	nbio.Storage_Write(loop.Storage, &completion, file, buffer, offset, SIM_DEADLINE,
		func(_ nbio.Completion_Handle) { done = true })
	testify.False(t, done)
	nbio.Driver_Run_Until(driver, SIM_DEADLINE, func() (finished bool) { return done })
	testify.True(t, done)
	return completion.Data, completion.Error
}

// No-follow must reject before creation can damage a link, with either truncate setting.
func Test_Sim_Create_No_Follow(t *testing.T) {
	seen := false
	for seed := uint64(0); seed < 64; seed++ {
		for _, truncate := range []bool{false, true} {
			loop, driver, nodes := sim_loop_nodes(seed)
			path, found := sim_first_link_path(nodes)
			if found {
				seen = true
				sim_create_no_follow(t, loop, driver, path, truncate)
			}
			nbio.IO_Deinit(loop)
		}
	}
	testify.True(t, seen)
}

func sim_create_no_follow(
	t *testing.T, loop nbio.IO, driver nbio.Driver, path string, truncate bool,
) {
	t.Helper()
	before := [nbio.SIM_FILE_BYTES_MAXIMUM]byte{}
	count, err := nbio.Storage_Read_Link(loop.Storage, path, before[:])
	testify.No_Error(t, err)
	file, err := sim_open_options(t, loop, driver, path, nbio.Open_At_Options{
		Access: nbio.OPEN_READ_WRITE, Create: true, Truncate: truncate,
		Flags: nbio.OPEN_AT_NO_FOLLOW,
	})
	testify.Error_Is(t, err, nbio.Symbolic_Link_Not_Followed)
	if err == nil {
		sim_close(t, loop, driver, file)
	}
	after := [nbio.SIM_FILE_BYTES_MAXIMUM]byte{}
	after_count, err := nbio.Storage_Read_Link(loop.Storage, path, after[:])
	testify.No_Error(t, err)
	testify.Equal(t, string(before[:count]), string(after[:after_count]))
}

// Create through a link must operate on its target while preserving link text.
func Test_Sim_Create_Follows_Link(t *testing.T) {
	seen := false
	for seed := uint64(0); seed < 64; seed++ {
		for _, truncate := range []bool{false, true} {
			loop, driver, nodes := sim_loop_nodes(seed)
			path, found := sim_first_link_path(nodes)
			if found {
				checked := sim_create_follows_link(
					t, loop, driver, nodes, path, truncate,
				)
				seen = checked || seen
			}
			nbio.IO_Deinit(loop)
		}
	}
	testify.True(t, seen)
}

func sim_create_follows_link(
	t *testing.T, loop nbio.IO, driver nbio.Driver, nodes []nbio.Sim_Node,
	path string, truncate bool,
) (checked bool) {
	t.Helper()
	text := [nbio.SIM_FILE_BYTES_MAXIMUM]byte{}
	count, err := nbio.Storage_Read_Link(loop.Storage, path, text[:])
	testify.No_Error(t, err)
	target := sim_node_directory(nodes, path) + string(text[:count])
	status, err := nbio.Storage_Status(loop.Storage, target)
	testify.No_Error(t, err)
	if !status.Exists {
		return false
	}
	if !nbio.File_Mode_Is_Regular(status.Mode) {
		return false
	}
	file, err := sim_create(t, loop, driver, target)
	if !testify.No_Error(t, err) {
		return false
	}
	sim_storage_write(t, loop, driver, file, []byte("base"))
	sim_close(t, loop, driver, file)
	file, err = sim_open_options(t, loop, driver, path, nbio.Open_At_Options{
		Access: nbio.OPEN_READ_WRITE, Create: true, Truncate: truncate,
	})
	if !testify.No_Error(t, err) {
		return false
	}
	want := "base"
	if truncate {
		want = ""
	}
	testify.Equal(t, want, sim_storage_read(t, loop, driver, file, 4))
	sim_storage_write(t, loop, driver, file, []byte("next"))
	sim_close(t, loop, driver, file)
	file, err = sim_open(t, loop, driver, target)
	if !testify.No_Error(t, err) {
		return false
	}
	testify.Equal(t, "next", sim_storage_read(t, loop, driver, file, 4))
	sim_close(t, loop, driver, file)
	after := [nbio.SIM_FILE_BYTES_MAXIMUM]byte{}
	after_count, err := nbio.Storage_Read_Link(loop.Storage, path, after[:])
	testify.No_Error(t, err)
	testify.Equal(t, string(text[:count]), string(after[:after_count]))
	return true
}
