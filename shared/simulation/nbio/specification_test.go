package nbio_test

import (
	"fmt"
	"strings"
	"testing"

	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/simulation/nbio"
	"local/james-orcales/shared/simulation/time"
	snap "local/james-orcales/shared/snap/default"
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
	var write_completion time.Completion
	loop.Storage.Write(&write_completion, writer, []byte("hello"), 0, SIM_DEADLINE,
		func(completed *time.Completion) {
			testify.No_Error(t, completed.Error)
			wrote = true
		})
	driver.Run_Until(SIM_DEADLINE, func() (finished bool) { return wrote })

	reader, open_err := sim_open(t, loop, driver, "file")
	if !testify.No_Error(t, open_err) {
		return
	}
	buffer := make([]byte, 64)
	count := -1
	var read_completion time.Completion
	loop.Storage.Read(&read_completion, reader, buffer, 0, SIM_DEADLINE,
		func(completed *time.Completion) {
			count = completed.Data
		})
	driver.Run_Until(SIM_DEADLINE, func() (finished bool) { return count >= 0 })

	testify.Equal(t, 5, count)
	testify.Equal(t, "hello", string(buffer[:count]))
	sim_close(t, loop, driver, writer)
	sim_close(t, loop, driver, reader)
	loop.Deinit()

	socket_loop, _, _ := sim_loop(1)
	socket, socket_err := socket_loop.Network.Socket_TCP(
		nbio.FAMILY_IPV4, sim_tcp_options())
	if !testify.No_Error(t, socket_err) {
		return
	}
	var socket_completion time.Completion
	testify.Panics(t, func() {
		socket_loop.Storage.Read(
			&socket_completion, socket, make([]byte, 8), 0, SIM_DEADLINE,
			func(_ *time.Completion) {})
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
	var completion time.Completion
	loop.Storage.Write(&completion, file, []byte("hello"), 0, SIM_DEADLINE,
		func(completed *time.Completion) {
			testify.No_Error(t, completed.Error)
			count = completed.Data
		})
	driver.Run_Until(SIM_DEADLINE, func() (finished bool) { return count >= 0 })
	testify.Equal(t, len("hello"), count)
	sim_close(t, loop, driver, file)
	loop.Deinit()
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
	var completion time.Completion
	loop.Storage.Fsync(&completion, file, SIM_DEADLINE, func(completed *time.Completion) {
		testify.No_Error(t, completed.Error)
		fired = true
	})
	driver.Run_Until(SIM_DEADLINE, func() (finished bool) { return fired })
	testify.True(t, sim_descriptor_open(loop))
	sim_close(t, loop, driver, file)
	loop.Deinit()
}

// Test_Sim_Open_At verify asynchronous openat surface make caller-owned file ordinary read and
// write operations can use.
func Test_Sim_Open_At(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	opened := nbio.File(-1)
	var completion time.Completion
	loop.Storage.Open_At(&completion, nbio.DIRECTORY_CURRENT, "file", nbio.Open_At_Options{
		Access: nbio.OPEN_READ_WRITE, Create: true, Truncate: true, Mode: 0o600,
	}, func(completed *time.Completion) {
		testify.No_Error(t, completed.Error)
		opened = nbio.File(completed.Data)
	})
	driver.Run_Until(SIM_DEADLINE, func() (finished bool) { return opened >= 0 })
	testify.Positive(t, opened)
	testify.True(t, sim_descriptor_open(loop))
	sim_close(t, loop, driver, opened)
	loop.Deinit()

	unknown_loop, _, _ := sim_loop(0)
	var unknown_completion time.Completion
	testify.Panics(t, func() {
		unknown_loop.Storage.Open_At(
			&unknown_completion,
			nbio.DIRECTORY_CURRENT,
			"file",
			nbio.Open_At_Options{Flags: nbio.Open_At_Flags(1 << 31)},
			func(_ *time.Completion) {},
		)
	})
}

// Test_Sim_Socket verify each transport requires all limits and transfers descriptor ownership
// only through its result.
func Test_Sim_Socket(t *testing.T) {
	socket_default_profile_assert(t)
	loop, driver, _ := sim_loop(0)
	socket, open_err := loop.Network.Socket_TCP(nbio.FAMILY_IPV4, sim_tcp_options())
	if !testify.No_Error(t, open_err) {
		return
	}
	testify.True(t, sim_descriptor_open(loop))
	sim_close(t, loop, driver, socket)

	udp, udp_err := loop.Network.Socket_UDP(nbio.FAMILY_IPV4, sim_udp_options())
	if !testify.No_Error(t, udp_err) {
		return
	}
	sim_close(t, loop, driver, udp)

	sim_socket_limits_rejected(t, loop, driver)
	loop.Deinit()
}

// Test_Sim_Bind verify Bind gives a socket its requested local address and assigns a port when
// the request uses port zero.
func Test_Sim_Bind(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	socket, open_err := loop.Network.Socket_TCP(nbio.FAMILY_IPV4, sim_tcp_options())
	if !testify.No_Error(t, open_err) {
		return
	}
	requested := nbio.Address_IPV4([nbio.IPV4_ADDRESS_BYTES]byte{127, 0, 0, 1}, 0)
	testify.No_Error(t, loop.Network.Bind(socket, requested))
	bound, name_err := loop.Network.Get_Socket_Name(socket)
	testify.No_Error(t, name_err)
	testify.Equal(t, requested.IP, bound.IP)
	testify.Not_Zero(t, bound.Port)
	sim_close(t, loop, driver, socket)
	loop.Deinit()
}

// Test_Sim_Accept verify accept resolve exactly once, with distinct accepted socket, or with
// Deadline_Exceeded.
func Test_Sim_Accept(t *testing.T) {
	accepted_count := 0
	deadline_count := 0
	for seed := uint64(0); seed < 64; seed++ {
		accepted, outcome := sim_accept_once(t, seed, time.NANOSECOND)
		if outcome == time.Deadline_Exceeded {
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
		if strings.HasSuffix(outcome, "outcome=success") {
			saw_success = true
		}
		if strings.HasSuffix(outcome, "outcome=refused") {
			saw_refusal = true
		}
	}
	testify.True(t, saw_success)
	testify.True(t, saw_refusal)
}

// Test_Sim_Receive verify receive complete after modeled latency and report buffer length.
func Test_Sim_Receive(t *testing.T) {
	loop, driver, _ := sim_loop(1)
	socket, open_err := loop.Network.Socket_TCP(nbio.FAMILY_IPV4, sim_tcp_options())
	if !testify.No_Error(t, open_err) {
		return
	}

	count := -1
	var completion time.Completion
	loop.Network.Receive(&completion, socket, make([]byte, 64), SIM_DEADLINE,
		func(completed *time.Completion) {
			count = completed.Data
		})

	driver.Run_For(10 * time.NANOSECOND)

	testify.Equal(t, 64, count)
	sim_close(t, loop, driver, socket)
	loop.Deinit()
}

// Test_Sim_Send verify send complete after modeled latency and report buffer length.
func Test_Sim_Send(t *testing.T) {
	loop, driver, _ := sim_loop(1)
	socket, open_err := loop.Network.Socket_TCP(nbio.FAMILY_IPV4, sim_tcp_options())
	if !testify.No_Error(t, open_err) {
		return
	}

	count := -1
	var completion time.Completion
	loop.Network.Send(&completion, socket, make([]byte, 32), SIM_DEADLINE,
		func(completed *time.Completion) {
			count = completed.Data
		})

	driver.Run_For(10 * time.NANOSECOND)

	testify.Equal(t, 32, count)
	sim_close(t, loop, driver, socket)
	loop.Deinit()
}

// Test_Sim_Shutdown verify shutdown resolve armed and later socket operations through their
// normal callbacks, and leave descriptor ownership with caller.
func Test_Sim_Shutdown(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	socket, open_err := loop.Network.Socket_TCP(nbio.FAMILY_IPV4, sim_tcp_options())
	if !testify.No_Error(t, open_err) {
		return
	}
	connected := false
	var connect_completion time.Completion
	loop.Network.Connect(&connect_completion, socket,
		nbio.Address_IPV4([nbio.IPV4_ADDRESS_BYTES]byte{127, 0, 0, 1}, 8123), SIM_DEADLINE,
		func(completed *time.Completion) {
			testify.No_Error(t, completed.Error)
			connected = true
		})
	driver.Run_Until(SIM_DEADLINE, func() (finished bool) { return connected })
	var receive_completion time.Completion
	var send_completion time.Completion
	receive_count := -1
	var send_err error
	loop.Network.Receive(&receive_completion, socket, make([]byte, 8), SIM_DEADLINE,
		func(completed *time.Completion) {
			testify.No_Error(t, completed.Error)
			receive_count = completed.Data
		})
	loop.Network.Send(&send_completion, socket, []byte("hello"), SIM_DEADLINE,
		func(completed *time.Completion) {
			send_err = completed.Error
		})
	testify.No_Error(t, loop.Network.Shutdown(socket, nbio.SHUTDOWN_BOTH))
	driver.Run_For(10 * time.NANOSECOND)
	testify.Zero(t, receive_count)
	testify.Error_Is(t, send_err, nbio.Broken_Pipe)
	testify.True(t, sim_descriptor_open(loop))
	sim_close(t, loop, driver, socket)
	loop.Deinit()
}

// Test_Sim_Close verify close reject descriptor an armed operation borrow, then
// shutdown-drain-close sequence complete and remove caller-owned descriptor.
func Test_Sim_Close(t *testing.T) {
	loop, driver, _ := sim_loop(0)

	socket, open_err := loop.Network.Socket_TCP(nbio.FAMILY_IPV4, sim_tcp_options())
	if !testify.No_Error(t, open_err) {
		return
	}
	connected := false
	var connect_completion time.Completion
	loop.Network.Connect(&connect_completion, socket,
		nbio.Address_IPV4([nbio.IPV4_ADDRESS_BYTES]byte{127, 0, 0, 1}, 1), SIM_DEADLINE,
		func(completed *time.Completion) {
			testify.No_Error(t, completed.Error)
			connected = true
		})
	driver.Run_Until(SIM_DEADLINE, func() (finished bool) { return connected })
	received := false
	var receive_completion time.Completion
	loop.Network.Receive(&receive_completion, socket, make([]byte, 8), SIM_DEADLINE,
		func(_ *time.Completion) {
			received = true
		})

	var completion time.Completion
	testify.Panics(t, func() {
		loop.Close(&completion, socket, func(_ *time.Completion) {})
	})
	testify.No_Error(t, loop.Network.Shutdown(socket, nbio.SHUTDOWN_BOTH))
	driver.Run_For(10 * time.NANOSECOND)
	testify.True(t, received)

	sim_close(t, loop, driver, socket)
	loop.Deinit()
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
	loop.Deinit()
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
	loop.Deinit()
}

// Test_Sim_Peer_Address verify Peer_Address report address of live descriptor, and empty address
// of unknown one.
func Test_Sim_Peer_Address(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	socket, _ := loop.Network.Socket_TCP(nbio.FAMILY_IPV4, sim_tcp_options())
	address, err := loop.Network.Peer_Address(socket)
	testify.No_Error(t, err)
	testify.Not_Empty(t, address)
	unknown, _ := loop.Network.Peer_Address(0)
	testify.Empty(t, unknown)
	sim_close(t, loop, driver, socket)
	loop.Deinit()
}

// Test_Sim_Status verify Status report existence, kind, and size, and report absent path as
// not-exists with nil error, not as failure.
func Test_Sim_Status(t *testing.T) {
	loop, driver, _ := sim_loop(0)
	made := false
	var mkdir time.Completion
	loop.Storage.Mkdir_At(&mkdir, nbio.DIRECTORY_CURRENT, "/dir", 0o755,
		func(completed *time.Completion) {
			testify.No_Error(t, completed.Error)
			made = true
		})
	driver.Run_Until(SIM_DEADLINE, func() (finished bool) { return made })
	file, create_err := sim_create(t, loop, driver, "/dir/file")
	if !testify.No_Error(t, create_err) {
		return
	}
	// Write known bytes, thus file Size has known expected value.
	content := []byte("hello world")
	written := false
	var write time.Completion
	loop.Storage.Write(&write, file, content, 0, SIM_DEADLINE, func(_ *time.Completion) {
		written = true
	})
	driver.Run_Until(SIM_DEADLINE, func() (finished bool) { return written })
	directory, _ := loop.Storage.Status("/dir")
	testify.True(t, directory.Exists)
	testify.True(t, directory.Is_Directory)
	testify.False(t, directory.Is_Regular)
	testify.Zero(t, directory.Size)
	sim_status_regular(t, loop, content)
	sim_close(t, loop, driver, file)
	loop.Deinit()
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
	loop.Deinit()
}

// Test_Address_Parse verify hand-written literal parse: IPv4, IPv6 in every accepted shape, and
// every rejection the net/netip call used to give for free.
func Test_Address_Parse(t *testing.T) {
	for _, accepted := range address_parse_accepted() {
		address, err := nbio.Address_Parse(accepted.Host, 8123)
		if !testify.No_Error(t, err, accepted.Host) {
			continue
		}
		testify.Equal(t, accepted.Family, address.Family, accepted.Host)
		testify.Equal(t, uint16(8123), address.Port, accepted.Host)
		testify.Equal(t, accepted.IP, address.IP, accepted.Host)
		if accepted.Family == nbio.FAMILY_IPV6 {
			expected := nbio.Address_IPV6(accepted.IP, 8123)
			testify.Equal(t, expected, address, accepted.Host)
		}
	}
	for _, rejected := range address_parse_rejected() {
		_, err := nbio.Address_Parse(rejected, 8123)
		testify.Error(t, err, rejected)
	}
	_, negative_err := nbio.Address_Parse("127.0.0.1", -1)
	testify.Error(t, negative_err)
	_, overflow_err := nbio.Address_Parse("127.0.0.1", 65536)
	testify.Error(t, overflow_err)
}

// The Stream function owns callback time because only it knows if the concrete operation is
// immediate, simulated, or kernel-backed.
func Test_Stream_Callback(t *testing.T) {
	timeline, driver, _ := time.New_Virtual_Timeline(
		time.Virtual_Clock{Resolution: time.NANOSECOND},
	)
	buffer := make([]byte, 3)
	called := false
	stream := nbio.Stream(
		func(
			completion *time.Completion, _ nbio.Stream_Mode, _ []byte, _ int64,
			_ nbio.Seek_From, callback time.Callback,
		) {
			timeline.Submit(
				completion, time.NANOSECOND, func(completed *time.Completion) {
					completed.Data = len(buffer)
					callback(completed)
				},
			)
		},
	)
	var completion time.Completion
	nbio.Read(stream, &completion, buffer, func(completed *time.Completion) {
		called = true
		testify.Equal(t, len(buffer), completed.Data)
		testify.No_Error(t, completed.Error)
	})
	testify.False(t, called)
	completed, drive_err := driver.Run_Until(
		10*time.NANOSECOND, func() (finished bool) { return called },
	)
	testify.No_Error(t, drive_err)
	testify.True(t, completed)
	testify.True(t, called)
	inline_called := false
	stream = func(
		completion *time.Completion, _ nbio.Stream_Mode, _ []byte, _ int64,
		_ nbio.Seek_From, callback time.Callback,
	) {
		completion.Data = len(buffer)
		callback(completion)
	}
	nbio.Read(stream, &completion, buffer, func(_ *time.Completion) {
		inline_called = true
	})
	testify.True(t, inline_called)
}

// Test_Stream_Read verifies Read moves bytes from the cursor, advances the cursor, and
// reports Stream_EOF once the cursor has reached the end of the memory.
func Test_Stream_Read(t *testing.T) {
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
func Test_Stream_Write(t *testing.T) {
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
func Test_Stream_Read_At(t *testing.T) {
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
func Test_Stream_Write_At(t *testing.T) {
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
func Test_Stream_Seek(t *testing.T) {
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
func Test_Stream_Size(t *testing.T) {
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
func Test_Stream_Query(t *testing.T) {
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
func Test_Stream_Flush(t *testing.T) {
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
func Test_Stream_Close(t *testing.T) {
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
func Test_Stream_Destroy(t *testing.T) {
	harness := stream_harness(t)
	stream := memory_stream(harness, make([]byte, 4))
	_, err := stream_destroy(t, harness, stream)
	testify.No_Error(t, err)
	_, err = stream_read(t, harness, stream, make([]byte, 1))
	testify.Error_Is(t, err, nbio.Stream_Empty)
}

// Test_Stream_Errors verifies the dispatch checks Odin's write helper performs: a zero
// Stream reports Stream_Empty, and a mode a stream does not answer reports Stream_Empty.
func Test_Stream_Errors(t *testing.T) {
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
func Test_Stream_Memory(t *testing.T) {
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
func Test_Stream_Discard(t *testing.T) {
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
func Test_Stream_Limit(t *testing.T) {
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
func Test_Stream_Count(t *testing.T) {
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
func Test_Stream_Tee(t *testing.T) {
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
func Test_Stream_Composition(t *testing.T) {
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

// One literal the parse must accept, with the address it must yield.
type Address_Case struct {
	// Host is the literal under test.
	Host string
	// Family is the family the parse must report.
	Family nbio.Address_Family
	// IP is the byte layout the parse must produce.
	IP [nbio.IPV6_ADDRESS_BYTES]byte
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
func address_bytes(values ...byte) (layout [nbio.IPV6_ADDRESS_BYTES]byte) {
	copy(layout[:], values)
	return layout
}

// Lay values at back of address, zero-filled from the front.
func address_suffix(values ...byte) (layout [nbio.IPV6_ADDRESS_BYTES]byte) {
	copy(layout[nbio.IPV6_ADDRESS_BYTES-len(values):], values)
	return layout
}

// Put one byte in the last position, the shape a "::" run with one trailing group yield.
func address_low(value byte) (layout [nbio.IPV6_ADDRESS_BYTES]byte) {
	layout[nbio.IPV6_ADDRESS_BYTES-1] = value
	return layout
}

// Combine two layouts, taking every non-zero byte of each.
func address_merge(
	front [nbio.IPV6_ADDRESS_BYTES]byte, back [nbio.IPV6_ADDRESS_BYTES]byte,
) (layout [nbio.IPV6_ADDRESS_BYTES]byte) {
	layout = front
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
func sim_socket_limits_rejected(t *testing.T, loop nbio.IO, driver time.Driver) {
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
	t *testing.T, loop nbio.IO, driver time.Driver, name string, options nbio.TCP_Options,
) {
	t.Helper()
	socket, err := loop.Network.Socket_TCP(nbio.FAMILY_IPV4, options)
	if testify.Error(t, err, name) {
		return
	}
	sim_close(t, loop, driver, socket)
}

func sim_udp_limit_rejected(
	t *testing.T, loop nbio.IO, driver time.Driver, name string, options nbio.UDP_Options,
) {
	t.Helper()
	socket, err := loop.Network.Socket_UDP(nbio.FAMILY_IPV4, options)
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
	regular, _ := loop.Storage.Status("/dir/file")
	testify.False(t, regular.Is_Directory)
	testify.True(t, regular.Is_Regular)
	testify.Equal(t, int64(len(content)), regular.Size)
	absent, _ := loop.Storage.Status("/nope")
	testify.False(t, absent.Exists)
	testify.False(t, absent.Is_Regular)
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
	listener, _ := loop.Network.Socket_TCP(nbio.FAMILY_IPV4, sim_tcp_options())
	address := nbio.Address_IPV4([nbio.IPV4_ADDRESS_BYTES]byte{127, 0, 0, 1}, 0)
	testify.No_Error(t, loop.Network.Bind(listener, address), seed)
	testify.No_Error(t, loop.Network.Listen_Socket(listener, 128), seed)
	callback_count := 0
	accepted = nbio.File(-1)
	var completion time.Completion
	loop.Network.Accept(&completion, listener, timeout,
		func(completed *time.Completion) {
			callback_count++
			accepted = nbio.File(completed.Data)
			operation_err = completed.Error
		})
	completed, drive_err := driver.Run_Until(
		16*time.NANOSECOND, func() (finished bool) { return callback_count > 0 })
	testify.No_Error(t, drive_err, seed)
	testify.True(t, completed, seed)
	driver.Run_For(16 * time.NANOSECOND)
	testify.Equal(t, 1, callback_count, seed)
	testify.Not_Equal(t, listener, accepted, seed)
	if accepted > 0 {
		sim_close(t, loop, driver, accepted)
	}
	sim_close(t, loop, driver, listener)
	loop.Deinit()
	return accepted, operation_err
}

func sim_connect_lifecycle(t *testing.T, seed uint64) (snapshot string) {
	t.Helper()
	loop, driver, _ := sim_loop(seed)
	socket, open_err := loop.Network.Socket_TCP(nbio.FAMILY_IPV4, sim_tcp_options())
	if !testify.No_Error(t, open_err) {
		return ""
	}
	called := false
	var connect_err error
	var completion time.Completion
	loop.Network.Connect(&completion, socket,
		nbio.Address_IPV4([nbio.IPV4_ADDRESS_BYTES]byte{127, 0, 0, 1}, 8123), SIM_DEADLINE,
		func(completed *time.Completion) {
			called = true
			connect_err = completed.Error
		})
	driver.Run_Until(SIM_DEADLINE, func() (finished bool) { return called })
	testify.True(t, called)
	open_before_close := sim_descriptor_open(loop)
	sim_close(t, loop, driver, socket)
	loop.Deinit()
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
	socket, open_err := loop.Network.Socket_TCP(nbio.FAMILY_IPV4, sim_tcp_options())
	if !testify.No_Error(t, open_err) {
		return open_err
	}
	callback_count := 0
	var completion time.Completion
	loop.Network.Connect(&completion, socket,
		nbio.Address_IPV4([nbio.IPV4_ADDRESS_BYTES]byte{127, 0, 0, 1}, 8123), timeout,
		func(completed *time.Completion) {
			callback_count++
			connect_err = completed.Error
		})
	driver.Run_Until(SIM_DEADLINE, func() (finished bool) { return callback_count > 0 })
	driver.Run_For(16 * time.NANOSECOND)
	testify.Equal(t, 1, callback_count, seed)
	testify.True(t, sim_descriptor_open(loop), seed)
	sim_close(t, loop, driver, socket)
	loop.Deinit()
	return connect_err
}

// Report whether run still hold open descriptor. Deinit is only census surface expose, thus test
// state "still open" by watch of Deinit reject run.
func sim_descriptor_open(loop nbio.IO) (open bool) {
	defer func() { open = recover() != nil }()
	loop.Deinit()
	return false
}

// Fresh loops make latency and equal-time ordering pure functions of the supplied seed.
func sim_transfer_once(
	t *testing.T, seed uint64, datagram bool, send bool, timeout time.Duration,
) (count int, operation_err error, completed_at time.Monotonic_Moment) {
	t.Helper()
	loop, driver, clock := sim_loop(seed)
	var socket nbio.File
	var open_err error
	if datagram {
		socket, open_err = loop.Network.Socket_UDP(nbio.FAMILY_IPV4, sim_udp_options())
	} else {
		socket, open_err = loop.Network.Socket_TCP(nbio.FAMILY_IPV4, sim_tcp_options())
	}
	if !testify.No_Error(t, open_err) {
		return 0, open_err, 0
	}
	callback_count := 0
	var completion time.Completion
	callback := func(completed *time.Completion) {
		callback_count++
		count = completed.Data
		operation_err = completed.Error
		completed_at = clock.Now_Monotonic()
	}
	if send {
		loop.Network.Send(&completion, socket, []byte("data"), timeout, callback)
	} else {
		loop.Network.Receive(&completion, socket, make([]byte, 4), timeout, callback)
	}
	driver.Run_Until(SIM_DEADLINE, func() (finished bool) { return callback_count > 0 })
	driver.Run_For(16 * time.NANOSECOND)
	testify.Equal(t, 1, callback_count, seed)
	testify.True(t, sim_descriptor_open(loop), seed)
	sim_close(t, loop, driver, socket)
	loop.Deinit()
	return count, operation_err, completed_at
}

// Panic is the synchronous rejection surface, so no callback or driver pass may be necessary.
func sim_network_timeout_rejected(
	t *testing.T, network nbio.Network, operation string, timeout time.Duration,
) {
	t.Helper()
	testify.Panics(t, func() {
		var completion time.Completion
		if operation == "accept" {
			network.Accept(&completion, nbio.File(-1), timeout,
				func(_ *time.Completion) {})
			return
		}
		if operation == "connect" {
			network.Connect(&completion, nbio.File(-1), nbio.Address{}, timeout,
				func(_ *time.Completion) {})
			return
		}
		if operation == "receive" {
			network.Receive(&completion, nbio.File(-1), nil, timeout,
				func(_ *time.Completion) {})
			return
		}
		network.Send(&completion, nbio.File(-1), nil, timeout,
			func(_ *time.Completion) {})
	}, operation, timeout)
}

// Run one storage operation through its late-delivery window and verify canceled effects.
func sim_storage_once(
	t *testing.T, seed uint64, operation string, timeout time.Duration,
) (data int, operation_err error, elapsed time.Monotonic_Moment) {
	t.Helper()
	loop, driver, clock := sim_loop(seed)
	file, create_err := sim_create(t, loop, driver, operation)
	if !testify.No_Error(t, create_err) {
		return 0, create_err, 0
	}
	sim_storage_write(t, loop, driver, file, []byte("base"))
	read_buffer := []byte("keep")
	start := clock.Now_Monotonic()
	called := false
	var completion time.Completion
	callback := func(completed *time.Completion) {
		data = completed.Data
		operation_err = completed.Error
		elapsed = clock.Now_Monotonic() - start
		called = true
	}
	if operation == "read" {
		loop.Storage.Read(&completion, file, read_buffer, 0, timeout, callback)
	} else if operation == "write" {
		loop.Storage.Write(&completion, file, []byte("next"), 0, timeout, callback)
	} else {
		loop.Storage.Fsync(&completion, file, timeout, callback)
	}
	driver.Run_Until(SIM_DEADLINE, func() (finished bool) { return called })
	testify.True(t, called, seed, operation)
	driver.Run_For(16 * time.NANOSECOND)
	if operation_err == time.Deadline_Exceeded {
		if operation == "read" {
			testify.Equal(t, "keep", string(read_buffer), seed)
		}
		if operation == "write" {
			contents := sim_storage_read(t, loop, driver, file, len("base"))
			testify.Equal(t, "base", contents, seed)
		}
	}
	sim_close(t, loop, driver, file)
	loop.Deinit()
	return data, operation_err, elapsed
}

// Use a long bound for fixture writes, so setup cannot consume the operation under test.
func sim_storage_write(
	t *testing.T, loop nbio.IO, driver time.Driver, file nbio.File, buffer []byte,
) {
	t.Helper()
	done := false
	var completion time.Completion
	loop.Storage.Write(&completion, file, buffer, 0, SIM_DEADLINE,
		func(completed *time.Completion) {
			testify.No_Error(t, completed.Error)
			done = true
		})
	driver.Run_Until(SIM_DEADLINE, func() (finished bool) { return done })
	testify.True(t, done)
}

// Read fixture state through Storage, so the side-effect check uses the public surface.
func sim_storage_read(
	t *testing.T, loop nbio.IO, driver time.Driver, file nbio.File, size int,
) (contents string) {
	t.Helper()
	buffer := make([]byte, size)
	done := false
	var completion time.Completion
	loop.Storage.Read(&completion, file, buffer, 0, SIM_DEADLINE,
		func(completed *time.Completion) {
			testify.No_Error(t, completed.Error)
			buffer = buffer[:completed.Data]
			done = true
		})
	driver.Run_Until(SIM_DEADLINE, func() (finished bool) { return done })
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
		var completion time.Completion
		if operation == "read" {
			loop.Storage.Read(&completion, file, nil, 0, timeout,
				func(_ *time.Completion) {})
			return
		}
		if operation == "write" {
			loop.Storage.Write(&completion, file, nil, 0, timeout,
				func(_ *time.Completion) {})
			return
		}
		loop.Storage.Fsync(&completion, file, timeout, func(_ *time.Completion) {})
	}, operation, timeout)
	sim_close(t, loop, driver, file)
	loop.Deinit()
}

// Build simulated loop, its driver, and read-only clock, seeded by seed. Test hold only IO,
// driver, and clock — never sim, which New_Simulated_IO keep to itself, thus run stay pure
// function of seed.
func sim_loop(seed uint64) (loop nbio.IO, driver time.Driver, clock time.Clock) {
	pump, driver, clock := time.New_Virtual_Timeline(
		time.Virtual_Clock{Resolution: time.NANOSECOND})
	return nbio.New_Simulated_IO(seed, pump), driver, clock
}

// Open path for read through Open_At. Drive loop until descriptor arrive.
func sim_open(
	t *testing.T, loop nbio.IO, driver time.Driver, path string,
) (file nbio.File, err error) {
	t.Helper()
	return sim_open_options(t, loop, driver, path, nbio.Open_At_Options{
		Access: nbio.OPEN_READ_ONLY,
	})
}

// Make or truncate path for write through Open_At.
func sim_create(
	t *testing.T, loop nbio.IO, driver time.Driver, path string,
) (file nbio.File, err error) {
	t.Helper()
	return sim_open_options(t, loop, driver, path, nbio.Open_At_Options{
		Access: nbio.OPEN_WRITE_ONLY, Create: true, Truncate: true, Mode: 0o644,
	})
}

// Submit one Open_At with caller options and drive loop until it retire.
func sim_open_options(
	t *testing.T, loop nbio.IO, driver time.Driver, path string, options nbio.Open_At_Options,
) (file nbio.File, err error) {
	t.Helper()
	done := false
	var completion time.Completion
	loop.Storage.Open_At(&completion, nbio.DIRECTORY_CURRENT, path, options,
		func(completed *time.Completion) {
			file = nbio.File(completed.Data)
			err = completed.Error
			done = true
		})
	driver.Run_Until(SIM_DEADLINE, func() (finished bool) { return done })
	testify.True(t, done, path)
	return file, err
}

// Release file through asynchronous Close and drive loop until close retire, thus caller reach
// Deinit with nothing still open.
func sim_close(t *testing.T, loop nbio.IO, driver time.Driver, file nbio.File) {
	t.Helper()
	closed := false
	var completion time.Completion
	loop.Close(&completion, file, func(completed *time.Completion) {
		testify.No_Error(t, completed.Error)
		closed = true
	})
	driver.Run_Until(SIM_DEADLINE, func() (finished bool) { return closed })
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
		if short == time.Deadline_Exceeded {
			saw_timeout = true
		}
		long := sim_connect_with_timeout(t, seed, SIM_DEADLINE)
		if long != time.Deadline_Exceeded {
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
			if operation_err == time.Deadline_Exceeded {
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
		if operation_err == time.Deadline_Exceeded {
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
			loop.Deinit()
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
			if operation_err == time.Deadline_Exceeded {
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
		if operation_err == time.Deadline_Exceeded {
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
		loop, driver, clock := sim_loop(0)
		file, create_err := sim_create(t, loop, driver, operation)
		if !testify.No_Error(t, create_err, operation) {
			continue
		}
		start := clock.Now_Monotonic()
		called := false
		var completion time.Completion
		callback := func(completed *time.Completion) {
			testify.Zero(t, completed.Data, operation)
			testify.No_Error(t, completed.Error, operation)
			called = true
		}
		if operation == "read" {
			loop.Storage.Read(&completion, file, nil, 0, time.NANOSECOND, callback)
		} else {
			loop.Storage.Write(&completion, file, nil, 0, time.NANOSECOND, callback)
		}
		testify.False(t, called, operation)
		driver.Run_Until(time.NANOSECOND, func() (finished bool) { return called })
		testify.True(t, called, operation)
		want := start + time.Monotonic_Moment(time.NANOSECOND)
		testify.Equal(t, want, clock.Now_Monotonic(), operation)
		sim_close(t, loop, driver, file)
		loop.Deinit()
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
	Timeline time.Timeline
	Driver   time.Driver
}

// One virtual Timeline tests delayed Stream functions without a production Driver.
func stream_harness(t *testing.T) (harness *stream_harness_state) {
	t.Helper()
	timeline, driver, _ := time.New_Virtual_Timeline(
		time.Virtual_Clock{Resolution: time.NANOSECOND},
	)
	harness = &stream_harness_state{Timeline: timeline, Driver: driver}
	t.Cleanup(harness.Driver.Deinit)
	return harness
}

// This join accepts the callback time that each concrete Stream selects.
func stream_result(
	t *testing.T, harness *stream_harness_state,
	submit func(completion *time.Completion, callback time.Callback),
) (data int, err error) {
	t.Helper()
	called := false
	var completion time.Completion
	submit(&completion, func(completed *time.Completion) {
		data = completed.Data
		err = completed.Error
		called = true
	})
	if called {
		return data, err
	}
	completed, drive_err := harness.Driver.Run_Until(
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
		completion *time.Completion, callback time.Callback,
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
		completion *time.Completion, callback time.Callback,
	) {
		nbio.Read_At(stream, completion, buffer, offset, callback)
	})
}

func stream_write(
	t *testing.T, harness *stream_harness_state, stream nbio.Stream, buffer []byte,
) (count int, err error) {
	t.Helper()
	return stream_result(t, harness, func(
		completion *time.Completion, callback time.Callback,
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
		completion *time.Completion, callback time.Callback,
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
		completion *time.Completion, callback time.Callback,
	) {
		nbio.Seek(stream, completion, offset, whence, callback)
	})
}

func stream_size(
	t *testing.T, harness *stream_harness_state, stream nbio.Stream,
) (size int, err error) {
	t.Helper()
	return stream_result(t, harness, func(
		completion *time.Completion, callback time.Callback,
	) {
		nbio.Size(stream, completion, callback)
	})
}

func stream_query(
	t *testing.T, harness *stream_harness_state, stream nbio.Stream,
) (modes nbio.Stream_Mode_Set, err error) {
	t.Helper()
	data, query_err := stream_result(t, harness, func(
		completion *time.Completion, callback time.Callback,
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
		completion *time.Completion, callback time.Callback,
	) {
		nbio.Flush(stream, completion, callback)
	})
}

func stream_close(
	t *testing.T, harness *stream_harness_state, stream nbio.Stream,
) (data int, err error) {
	t.Helper()
	return stream_result(t, harness, func(
		completion *time.Completion, callback time.Callback,
	) {
		nbio.Close(stream, completion, callback)
	})
}

func stream_destroy(
	t *testing.T, harness *stream_harness_state, stream nbio.Stream,
) (data int, err error) {
	t.Helper()
	return stream_result(t, harness, func(
		completion *time.Completion, callback time.Callback,
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
