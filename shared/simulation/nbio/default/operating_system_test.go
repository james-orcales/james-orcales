package nbio_test

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"local/james-orcales/shared/simulation/nbio"
	system_io "local/james-orcales/shared/simulation/nbio/default"
	sysos "local/james-orcales/shared/simulation/os"
	"local/james-orcales/shared/simulation/time"
	timeos "local/james-orcales/shared/simulation/time/default"
	"local/james-orcales/shared/testify"
)

// Run_Until cap of real-backend tests: generous, since completion return pump instant it fire —
// this bound only bite genuine hang, and fail test instead of block until package timeout.
const REAL_DEADLINE = 5 * time.SECOND

// Short finite operation deadline used to prove dormant kernel wait retire prompt.
const REAL_OPERATION_DEADLINE = 25 * time.MILLISECOND

// Bound buffer holding process identifier read back from fixture file.
const PROCESS_IDENTIFIER_BYTES = 64

// Inspect the full simulation tree because one inconsistent callback order makes every caller
// remember a different rule.
func Test_Callback_Parameter_Is_Last(t *testing.T) {
	files := token.NewFileSet()
	err := filepath.WalkDir("../..", func(
		path string, entry fs.DirEntry, walk_err error,
	) (err error) {
		if walk_err != nil {
			return walk_err
		}
		if entry.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		parsed, parse_err := parser.ParseFile(
			files, path, nil, parser.SkipObjectResolution,
		)
		if parse_err != nil {
			return parse_err
		}
		ast.Inspect(parsed, func(node ast.Node) (visit_children bool) {
			function, is_function := node.(*ast.FuncType)
			if !is_function {
				return true
			}
			callback_parameter_last(t, files, function)
			return true
		})
		return nil
	})
	testify.No_Error(t, err)
}

// A required bound must fail before the backend can borrow a caller-owned descriptor.
func Test_Operating_System_Storage_Rejects_Disabled_Timeouts(t *testing.T) {
	for _, operation := range []string{"read", "write", "fsync"} {
		for _, timeout := range []time.Duration{0, -time.NANOSECOND} {
			loop, _, driver := operating_system_loop(
				t, timeos.New_Operating_System_Clock(),
			)
			testify.Panics(t, func() {
				var completion time.Completion
				if operation == "read" {
					loop.Storage.Read(
						&completion, nbio.File(-1), nil, 0, timeout,
						func(_ *time.Completion) {})
					return
				}
				if operation == "write" {
					loop.Storage.Write(
						&completion, nbio.File(-1), nil, 0, timeout,
						func(_ *time.Completion) {})
					return
				}
				loop.Storage.Fsync(&completion, nbio.File(-1), timeout,
					func(_ *time.Completion) {})
			}, operation, timeout)
			driver.Deinit()
		}
	}
}

func callback_parameter_last(t *testing.T, files *token.FileSet, function *ast.FuncType) {
	t.Helper()
	if function.Params == nil {
		return
	}
	for index, parameter := range function.Params.List {
		if !callback_parameter_type(parameter.Type) {
			continue
		}
		position := files.Position(parameter.Pos())
		testify.Equal(t, len(function.Params.List)-1, index, position)
		testify.True(t, len(parameter.Names) <= 1, position)
	}
}

func callback_parameter_type(expression ast.Expr) (callback bool) {
	switch value := expression.(type) {
	case *ast.FuncType:
		return true
	case *ast.Ident:
		return strings.HasSuffix(value.Name, "Callback")
	case *ast.SelectorExpr:
		return strings.HasSuffix(value.Sel.Name, "Callback")
	}
	return false
}

// Write content to path through loop. Fixture setup in this package go through io.IO same as
// everything else: this is own suite of io gateway, thus test that reach around loop to prepare
// its input exercise path no application may take.
func write_file(t *testing.T, loop nbio.IO, driver time.Driver, path string, content []byte) {
	t.Helper()
	file, create_err := create_file(t, loop, driver, path)
	if !testify.No_Error(t, create_err, path) {
		return
	}
	written := 0
	write_done := false
	var completion time.Completion
	loop.Storage.Write(&completion, file, content, 0, REAL_DEADLINE, func(
		completed *time.Completion,
	) {
		testify.No_Error(t, completed.Error, path)
		written = completed.Data
		write_done = true
	})
	testify.True(t,
		operating_system_run_until(t, driver, func() (finished bool) { return write_done }),
		path)
	testify.Equal(t, len(content), written, path)
	close_file(t, loop, driver, path, file)
}

// Make path and every missing parent through derived Make_Directory, which compose Mkdir_At
// primitive above surface.
func make_directory(t *testing.T, loop nbio.IO, driver time.Driver, path string) (err error) {
	t.Helper()
	// Walk is caller business now: Mkdir_At is mkdirat primitive, thus each component is its
	// own submission, and existing one converge, not fail.
	for index, bound := range directory_bounds(path) {
		done := false
		var completion time.Completion
		component := path[:bound]
		loop.Storage.Mkdir_At(&completion, nbio.DIRECTORY_CURRENT, component, 0o755, func(
			completed *time.Completion,
		) {
			// Existing component converge: Path_Exists is what mkdirat report for
			// directory already there, which walk that make parents must
			// accept.
			if completed.Error != nil {
				if completed.Error != nbio.Path_Exists {
					err = completed.Error
				}
			}
			done = true
		})
		testify.True(t,
			operating_system_run_until(
				t, driver, func() (finished bool) { return done },
			),
			path, index)
		if err != nil {
			return err
		}
	}
	return nil
}

// End offsets of each path component, shortest first, thus walk make every missing parent before
// leaf.
func directory_bounds(path string) (bounds []int) {
	for index := 1; index < len(path); index++ {
		if path[index] != '/' {
			continue
		}
		if index == 0 {
			continue
		}
		bounds = append(bounds, index)
	}
	if len(path) > 0 {
		bounds = append(bounds, len(path))
	}
	return bounds
}

// Open path for read through Open_At. Drive loop until descriptor arrive.
func open_file(
	t *testing.T, loop nbio.IO, driver time.Driver, path string,
) (file nbio.File, err error) {
	t.Helper()
	return open_file_options(t, loop, driver, path, nbio.Open_At_Options{
		Access: nbio.OPEN_READ_ONLY,
	})
}

// Make or truncate path for write through Open_At.
func create_file(
	t *testing.T, loop nbio.IO, driver time.Driver, path string,
) (file nbio.File, err error) {
	t.Helper()
	return open_file_options(t, loop, driver, path, nbio.Open_At_Options{
		Access: nbio.OPEN_WRITE_ONLY, Create: true, Truncate: true, Mode: 0o644,
	})
}

// Submit one Open_At with caller options and drive loop until it retire.
func open_file_options(
	t *testing.T, loop nbio.IO, driver time.Driver, path string, options nbio.Open_At_Options,
) (file nbio.File, err error) {
	t.Helper()
	done := false
	var completion time.Completion
	loop.Storage.Open_At(&completion, nbio.DIRECTORY_CURRENT, path, options, func(
		completed *time.Completion,
	) {
		file = nbio.File(completed.Data)
		err = completed.Error
		done = true
	})
	testify.True(t,
		operating_system_run_until(t, driver, func() (finished bool) { return done }), path)
	return file, err
}

// Bound one listing pass, thus wide directory is drained in repeated passes, not into one
// unbounded allocation.
const DIRECTORY_PASS_BYTES = 8192

// Cap passes one listing take, thus backend that never report empty pass fail test, not spin it.
const DIRECTORY_PASSES_MAX = 4096

// List path children by composition of primitives caller now hold: Open_At, repeated
// Get_Directory_Entries passes until one report none, and Close.
func read_directory(
	t *testing.T, loop nbio.IO, driver time.Driver, path string,
) (entries []nbio.Directory_Entry, err error) {
	t.Helper()
	directory, open_err := open_file(t, loop, driver, path)
	if open_err != nil {
		return nil, open_err
	}
	buffer := make([]byte, DIRECTORY_PASS_BYTES)
	for pass_number_index := 0; pass_number_index < DIRECTORY_PASSES_MAX; pass_number_index++ {
		done := false
		var pass []nbio.Directory_Entry
		var pass_err error
		var completion time.Completion
		loop.Storage.Get_Directory_Entries(&completion, directory, buffer, func(
			_ *time.Completion, listed []nbio.Directory_Entry, read_err error,
		) {
			pass = listed
			pass_err = read_err
			done = true
		})
		testify.True(t,
			operating_system_run_until(
				t, driver, func() (finished bool) { return done },
			),
			path, pass_number_index)
		if pass_err != nil {
			err = pass_err
			break
		}
		if len(pass) == 0 {
			break
		}
		entries = append(entries, pass...)
	}
	closed := false
	var close_completion time.Completion
	loop.Close(&close_completion, directory, func(_ *time.Completion) {
		closed = true
	})
	testify.True(t,
		operating_system_run_until(
			t, driver, func() (finished bool) { return closed },
		), path)
	return entries, err
}

// Read up to len(buffer) bytes from path through loop. Return count.
func read_file(
	t *testing.T, loop nbio.IO, driver time.Driver, path string, buffer []byte,
) (count int) {
	t.Helper()
	file, open_err := open_file(t, loop, driver, path)
	if !testify.No_Error(t, open_err, path) {
		return 0
	}
	read_done := false
	var completion time.Completion
	loop.Storage.Read(&completion, file, buffer, 0, REAL_DEADLINE, func(
		completed *time.Completion,
	) {
		testify.No_Error(t, completed.Error, path)
		count = completed.Data
		read_done = true
	})
	testify.True(t,
		operating_system_run_until(t, driver, func() (finished bool) { return read_done }),
		path)
	close_file(t, loop, driver, path, file)
	return count
}

// Release descriptor through asynchronous Close of loop.
func close_file(t *testing.T, loop nbio.IO, driver time.Driver, path string, file nbio.File) {
	t.Helper()
	close_done := false
	var completion time.Completion
	loop.Close(&completion, file, func(completed *time.Completion) {
		testify.No_Error(t, completed.Error, path)
		close_done = true
	})
	testify.True(t,
		operating_system_run_until(t, driver, func() (finished bool) { return close_done }),
		path)
}

// Make 32-entry test scheduler. Fail at composition root when initialization fail.
func operating_system_loop(
	t *testing.T, clock time.Clock,
) (loop nbio.IO, pump time.Timeline, driver time.Driver) {
	t.Helper()
	loop, pump, driver, _ = operating_system_all(t, clock)
	return loop, pump, driver
}

// Make same scheduler and return OS beside it, for tests that spawn subprocess, or watch signal
// — two operations OS surface own.
func operating_system_all(
	t *testing.T, clock time.Clock,
) (loop nbio.IO, pump time.Timeline, driver time.Driver, system sysos.OS) {
	t.Helper()
	loop, pump, driver, system, err := system_io.New_Operating_System_IO(
		clock, 32, 0, sysos.Virtual_OS_To_OS(sysos.Virtual_OS{Process_Identifier: 1}))
	if !testify.No_Error(t, err) {
		return nbio.IO{}, time.Timeline{}, time.Driver{}, sysos.OS{}
	}
	return loop, pump, driver, system
}

// Drive real test predicate. Fail at once on backend scheduler error.
func operating_system_run_until(
	t *testing.T, driver time.Driver, done func() (finished bool),
) (completed bool) {
	t.Helper()
	completed, err := driver.Run_Until(REAL_DEADLINE, done)
	if !testify.No_Error(t, err) {
		return false
	}
	return completed
}

// Every entry rejects an absent bound before it can reach a platform scheduler.
func Test_Operating_System_IO_Network_Rejects_Disabled_Timeouts(t *testing.T) {
	clock := timeos.New_Operating_System_Clock()
	loop, _, driver := operating_system_loop(t, clock)
	for _, operation := range []string{"accept", "connect", "receive", "send"} {
		for _, timeout := range []time.Duration{0, -time.NANOSECOND} {
			operating_system_network_timeout_rejected(
				t, loop.Network, operation, timeout)
		}
	}
	driver.Deinit()
}

// Panic is synchronous because an invalid bound must not leave a kernel request to drain.
func operating_system_network_timeout_rejected(
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

// Return explicit profile real-backend socket tests use.
func test_tcp_options() (options nbio.TCP_Options) {
	return nbio.TCP_Options{
		Receive_Buffer_Bytes:     4 * 1024 * 1024,
		Send_Buffer_Bytes:        2 * 1024 * 1024,
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

// Return explicit overrides for each setting common to UDP sockets.
func test_udp_options() (options nbio.UDP_Options) {
	return nbio.UDP_Options{
		Receive_Buffer_Bytes:    4 * 1024 * 1024,
		Send_Buffer_Bytes:       2 * 1024 * 1024,
		Receive_Low_Water_Bytes: 1,
		Linger_Timeout:          1 * time.SECOND,
	}
}

// Report whether backend still hold open descriptor. Deinit is only census surface expose, thus
// test state "still open" by watch of Deinit reject run.
func descriptor_open(loop nbio.IO) (open bool) {
	defer func() { open = recover() != nil }()
	loop.Deinit()
	return false
}

// Open caller-owned IPv4 TCP socket with the profile backend tests verify.
func test_open_socket(loop nbio.IO) (socket nbio.File, err error) {
	return loop.Network.Socket_TCP(nbio.FAMILY_IPV4, test_tcp_options())
}

// Open and bind one caller-owned IPv4 TCP listener.
func test_listen(
	loop nbio.IO, driver time.Driver, host string, port int,
) (listener nbio.File, err error) {
	address, address_err := nbio.Address_Parse(host, port)
	if address_err != nil {
		return nbio.File(-1), address_err
	}
	listener, open_err := test_open_socket(loop)
	if open_err != nil {
		return nbio.File(-1), open_err
	}
	if bind_err := loop.Network.Bind(listener, address); bind_err != nil {
		self_exec_close(loop, driver, listener)
		return nbio.File(-1), bind_err
	}
	if listen_err := loop.Network.Listen_Socket(listener, 65535); listen_err != nil {
		self_exec_close(loop, driver, listener)
		return nbio.File(-1), listen_err
	}
	return listener, nil
}

// Convert IP literal for explicit-address Connect surface.
func test_connect(
	t *testing.T, loop nbio.IO, completion *time.Completion, socket nbio.File, host string,
	port int, callback time.Callback,
) (submitted bool) {
	t.Helper()
	address, err := nbio.Address_Parse(host, port)
	if !testify.No_Error(t, err) {
		return false
	}
	loop.Network.Connect(completion, socket, address, REAL_DEADLINE, callback)
	return true
}

// Test_Operating_System_IO_Read write temp file and read it back through real backend. It
// confirm read run in loop and report bytes.
func Test_Operating_System_IO_Read(t *testing.T) {
	clock := timeos.New_Operating_System_Clock()
	loop, _, driver := operating_system_loop(t, clock)
	path := filepath.Join(t.TempDir(), "read")
	write_file(t, loop, driver, path, []byte("hello"))

	file, open_err := open_file(t, loop, driver, path)
	if !testify.No_Error(t, open_err) {
		return
	}
	buffer := make([]byte, 5)
	count := -1
	read_done := false
	var completion time.Completion
	loop.Storage.Read(&completion, file, buffer, 0, REAL_DEADLINE, func(
		completed *time.Completion,
	) {
		testify.No_Error(t, completed.Error)
		count = completed.Data
		read_done = true
	})
	testify.True(t,
		operating_system_run_until(t, driver, func() (finished bool) { return read_done }))
	close_file(t, loop, driver, path, file)

	testify.Equal(t, 5, count)
	testify.Equal(t, "hello", string(buffer))
}

// Test_Resolve_Passes_IP_Literal confirm IP-literal host return unchanged, thus already-resolved
// address skip blocking DNS lookup, and dial path of loop see only IPs.
func Test_Resolve_Passes_IP_Literal(t *testing.T) {
	address, err := system_io.Resolve("93.184.216.34")
	testify.No_Error(t, err)
	testify.Equal(t, "93.184.216.34", address)
}

// Test_Operating_System_IO_Run_Until_Deadlock verify unbounded Run_Until with no operation
// pending fail loud, not block forever: predicate no event can flip is deadlock, thus pump panic
// instead of hang of caller.
func Test_Operating_System_IO_Run_Until_Deadlock(t *testing.T) {
	clock := timeos.New_Operating_System_Clock()
	_, _, driver := operating_system_loop(t, clock)
	testify.Panics(t, func() {
		driver.Run_Until(-1*time.NANOSECOND, func() (finished bool) { return false })
	})
}

// Test_Operating_System_IO_Timeout verify timeout fire once real time pass its deadline.
func Test_Operating_System_IO_Timeout(t *testing.T) {
	clock := timeos.New_Operating_System_Clock()
	_, pump, driver := operating_system_loop(t, clock)
	fired := false
	var completion time.Completion
	pump.Timeout(&completion, time.MILLISECOND, func(_ *time.Completion) {
		fired = true
	})
	driver.Run_Until(REAL_DEADLINE, func() (finished bool) { return fired })
	testify.True(t, fired)
}

// Test_Operating_System_IO_Open_Socket_Profile verify outbound socket is non-blocking,
// close-on-exec, buffered for client workload, keepalive-enabled, and caller-owned in Raw_Open.
func Test_Operating_System_IO_Open_Socket_Profile(t *testing.T) {
	clock := timeos.New_Operating_System_Clock()
	loop, _, driver := operating_system_loop(t, clock)
	socket, open_err := test_open_socket(loop)
	if !testify.No_Error(t, open_err) {
		return
	}
	testify.True(t, descriptor_open(loop))
	file_flags := socket_fcntl(t, socket, syscall.F_GETFD)
	testify.Not_Zero(t, file_flags&syscall.FD_CLOEXEC)
	status_flags := socket_fcntl(t, socket, syscall.F_GETFL)
	testify.Not_Zero(t, status_flags&syscall.O_NONBLOCK)
	receive_buffer, receive_err := syscall.GetsockoptInt(
		int(socket), syscall.SOL_SOCKET, syscall.SO_RCVBUF)
	testify.No_Error(t, receive_err)
	testify.Positive(t, receive_buffer)
	send_buffer, send_err := syscall.GetsockoptInt(
		int(socket), syscall.SOL_SOCKET, syscall.SO_SNDBUF)
	testify.No_Error(t, send_err)
	testify.Positive(t, send_buffer)
	keepalive, keepalive_err := syscall.GetsockoptInt(
		int(socket), syscall.SOL_SOCKET, syscall.SO_KEEPALIVE)
	testify.No_Error(t, keepalive_err)
	testify.Not_Zero(t, keepalive)
	self_exec_close(loop, driver, socket)
	loop.Deinit()
}

// A public UDP constructor check prevents the typed transport split from existing only below the
// composition root.
func Test_Operating_System_IO_Open_UDP_Profile(t *testing.T) {
	clock := timeos.New_Operating_System_Clock()
	loop, _, driver := operating_system_loop(t, clock)
	socket, open_err := loop.Network.Socket_UDP(nbio.FAMILY_IPV4, test_udp_options())
	if !testify.No_Error(t, open_err) {
		return
	}
	testify.True(t, descriptor_open(loop))
	receive_low_water, receive_low_water_err := syscall.GetsockoptInt(
		int(socket), syscall.SOL_SOCKET, syscall.SO_RCVLOWAT)
	testify.No_Error(t, receive_low_water_err)
	testify.Equal(t, int(test_udp_options().Receive_Low_Water_Bytes), receive_low_water)
	self_exec_close(loop, driver, socket)
	loop.Deinit()
}

// Test_Operating_System_IO_Bind_Reuse_Address verify Bind enables address reuse before it gives
// the socket its local address.
func Test_Operating_System_IO_Bind_Reuse_Address(t *testing.T) {
	clock := timeos.New_Operating_System_Clock()
	loop, _, driver := operating_system_loop(t, clock)
	socket, open_err := loop.Network.Socket_TCP(nbio.FAMILY_IPV4, test_tcp_options())
	if !testify.No_Error(t, open_err) {
		return
	}
	defer func() {
		self_exec_close(loop, driver, socket)
		loop.Deinit()
	}()
	address := nbio.Address_IPV4([nbio.IPV4_ADDRESS_BYTES]byte{127, 0, 0, 1}, 0)
	testify.No_Error(t, loop.Network.Bind(socket, address))
	reuse, reuse_err := syscall.GetsockoptInt(
		int(socket), syscall.SOL_SOCKET, syscall.SO_REUSEADDR)
	testify.No_Error(t, reuse_err)
	testify.Not_Zero(t, reuse)
}

// Test_Operating_System_IO_Reuse verify resubmit of completion still in flight panic: one
// Completion back one operation at a time, and real backend must fail as loud as sim, not
// silently double-arm it.
func Test_Operating_System_IO_Reuse(t *testing.T) {
	clock := timeos.New_Operating_System_Clock()
	_, pump, _ := operating_system_loop(t, clock)
	var completion time.Completion
	pump.Timeout(&completion, time.SECOND, func(_ *time.Completion) {})
	testify.Panics(t, func() {
		pump.Timeout(&completion, time.SECOND, func(_ *time.Completion) {})
	})
}

// Test_Operating_System_IO_Reentrancy verify drive of real loop from inside completion callback
// panic, thus re-entrant Run* fail loud, not corrupt it.
func Test_Operating_System_IO_Reentrancy(t *testing.T) {
	clock := timeos.New_Operating_System_Clock()
	_, pump, driver := operating_system_loop(t, clock)
	var completion time.Completion
	pump.Timeout(&completion, time.MILLISECOND, func(_ *time.Completion) {
		driver.Run()
	})
	testify.Panics(t, func() {
		driver.Run_For(50 * time.MILLISECOND)
	})
}

// Test_Operating_System_IO_Socket run TCP loopback round-trip through real backend: client
// connect to listener, send bytes, and accepted server socket receive them — all driven by
// single event loop.
func Test_Operating_System_IO_Socket(t *testing.T) {
	port := free_port(t)
	clock := timeos.New_Operating_System_Clock()
	loop, _, driver := operating_system_loop(t, clock)

	listener, listen_err := test_listen(loop, driver, "127.0.0.1", port)
	if !testify.No_Error(t, listen_err) {
		return
	}

	accepted := nbio.File(-1)
	var accept_completion time.Completion
	loop.Network.Accept(&accept_completion, listener, REAL_DEADLINE, func(
		completed *time.Completion,
	) {
		testify.No_Error(t, completed.Error)
		accepted = nbio.File(completed.Data)
	})

	connected, open_err := test_open_socket(loop)
	if !testify.No_Error(t, open_err) {
		return
	}
	connect_done := false
	var connect_completion time.Completion
	if !test_connect(t, loop, &connect_completion, connected, "127.0.0.1", port,
		func(completed *time.Completion) {
			testify.No_Error(t, completed.Error)
			connect_done = true
		},
	) {
		return
	}

	driver.Run_Until(REAL_DEADLINE, func() (finished bool) { return accepted > 0 })
	driver.Run_Until(REAL_DEADLINE, func() (finished bool) { return connect_done })
	testify.Positive(t, accepted)
	testify.Positive(t, connected)

	loopback_assert_roundtrip(&loopback_roundtrip_input{
		Test: t, Timeline: loop, Driver: driver, Connected: connected, Accepted: accepted,
	})
	self_exec_close(loop, driver, connected)
	self_exec_close(loop, driver, accepted)
	self_exec_close(loop, driver, listener)
	loop.Deinit()
}

// Test_Operating_System_IO_Accept_Deadline prove listener with no inbound connection retire its
// accept exactly once, after which listener and backend release safe.
func Test_Operating_System_IO_Accept_Deadline(t *testing.T) {
	clock := timeos.New_Operating_System_Clock()
	loop, _, driver := operating_system_loop(t, clock)
	listener, listen_err := test_listen(loop, driver, "127.0.0.1", 0)
	if !testify.No_Error(t, listen_err) {
		return
	}
	callback_count := 0
	accepted := nbio.File(-1)
	var operation_err error
	var completion time.Completion
	loop.Network.Accept(&completion, listener, REAL_OPERATION_DEADLINE, func(
		completed *time.Completion,
	) {
		callback_count++
		accepted = nbio.File(completed.Data)
		operation_err = completed.Error
	})
	testify.True(t, operating_system_run_until(
		t, driver, func() (finished bool) { return callback_count > 0 },
	))
	testify.Equal(t, 1, callback_count)
	testify.Error_Is(t, operation_err, time.Deadline_Exceeded)
	testify.Equal(t, nbio.File(-1), accepted)
	self_exec_close(loop, driver, listener)
	driver.Deinit()
}

// A receive timeout must retire the kernel registration without taking socket ownership.
func Test_Operating_System_IO_Receive_Timeout_Preserves_Socket(t *testing.T) {
	port := free_port(t)
	clock := timeos.New_Operating_System_Clock()
	loop, _, driver := operating_system_loop(t, clock)
	listener, listen_err := test_listen(loop, driver, "127.0.0.1", port)
	if !testify.No_Error(t, listen_err) {
		return
	}
	accepted, connected := loopback_pair(t, loop, driver, listener, port)
	callback_count := 0
	count := -1
	var operation_err error
	var completion time.Completion
	loop.Network.Receive(
		&completion, accepted, make([]byte, 8), REAL_OPERATION_DEADLINE, func(
			completed *time.Completion,
		) {
			callback_count++
			count = completed.Data
			operation_err = completed.Error
		})
	testify.True(t, operating_system_run_until(
		t, driver, func() (finished bool) { return callback_count > 0 },
	))
	testify.Equal(t, 1, callback_count)
	testify.Zero(t, count)
	testify.Error_Is(t, operation_err, time.Deadline_Exceeded)
	socket_fcntl(t, accepted, syscall.F_GETFD)
	self_exec_close(loop, driver, accepted)
	self_exec_close(loop, driver, connected)
	self_exec_close(loop, driver, listener)
	driver.Deinit()
}

// Test_Operating_System_IO_Connect_Error_Preserves_Socket verify refusal leave caller-owned
// descriptor open until caller explicitly close it.
func Test_Operating_System_IO_Connect_Error_Preserves_Socket(t *testing.T) {
	port := free_port(t)
	clock := timeos.New_Operating_System_Clock()
	loop, _, driver := operating_system_loop(t, clock)

	socket, open_err := test_open_socket(loop)
	if !testify.No_Error(t, open_err) {
		return
	}
	called := false
	var connect_err error
	var completion time.Completion
	if !test_connect(t, loop, &completion, socket, "127.0.0.1", port, func(
		completed *time.Completion,
	) {
		called = true
		connect_err = completed.Error
	}) {
		return
	}

	driver.Run_Until(REAL_DEADLINE, func() (finished bool) { return called })
	testify.Error_Is(t, connect_err, nbio.Connection_Refused)
	testify.True(t, descriptor_open(loop))
	self_exec_close(loop, driver, socket)
	loop.Deinit()
}

// Test_Operating_System_IO_Send_In_Connect_Completion arm send from inside connect completion —
// send share write-waiter slot of connected descriptor with connect it is armed within. Loop must
// retire connect waiter before it deliver its callback, else send is deleted instant it is armed
// and never fire (ClickHouse-daemon bug).
func Test_Operating_System_IO_Send_In_Connect_Completion(t *testing.T) {
	port := free_port(t)
	clock := timeos.New_Operating_System_Clock()
	loop, _, driver := operating_system_loop(t, clock)

	listener, listen_err := test_listen(loop, driver, "127.0.0.1", port)
	if !testify.No_Error(t, listen_err) {
		return
	}
	accepted := nbio.File(-1)
	var accept_completion time.Completion
	loop.Network.Accept(&accept_completion, listener, REAL_DEADLINE, func(
		completed *time.Completion,
	) {
		testify.No_Error(t, completed.Error)
		accepted = nbio.File(completed.Data)
	})

	sent := -1
	var send_completion time.Completion
	var connect_completion time.Completion
	socket, open_err := test_open_socket(loop)
	if !testify.No_Error(t, open_err) {
		return
	}
	if !test_connect(t, loop, &connect_completion, socket, "127.0.0.1", port, func(
		completed *time.Completion,
	) {
		if !testify.No_Error(t, completed.Error) {
			return
		}
		// Arm send inside connect completion: same descriptor, same write slot.
		loop.Network.Send(&send_completion, socket, []byte("ping"), REAL_DEADLINE, func(
			completed *time.Completion,
		) {
			testify.No_Error(t, completed.Error)
			sent = completed.Data
		})
	}) {
		return
	}

	driver.Run_Until(REAL_DEADLINE, func() (finished bool) { return sent >= 0 })
	testify.Equal(t, 4, sent)

	driver.Run_Until(REAL_DEADLINE, func() (finished bool) { return accepted > 0 })
	buffer := make([]byte, 16)
	received := -1
	var receive_completion time.Completion
	loop.Network.Receive(&receive_completion, accepted, buffer, REAL_DEADLINE, func(
		completed *time.Completion,
	) {
		testify.No_Error(t, completed.Error)
		received = completed.Data
	})
	driver.Run_Until(REAL_DEADLINE, func() (finished bool) { return received >= 0 })
	testify.Equal(t, 4, received)
	testify.Equal(t, "ping", string(buffer[:4]))
}

// Test_Operating_System_IO_Drain_Then_Recycle verify Shutdown resolve armed receive before Close,
// after which later connection still receive readiness normally.
func Test_Operating_System_IO_Drain_Then_Recycle(t *testing.T) {
	port := free_port(t)
	clock := timeos.New_Operating_System_Clock()
	loop, _, driver := operating_system_loop(t, clock)
	listener, listen_err := test_listen(loop, driver, "127.0.0.1", port)
	if !testify.No_Error(t, listen_err) {
		return
	}

	first, first_connected := loopback_pair(t, loop, driver, listener, port)
	buffer := make([]byte, 16)
	var receive_completion time.Completion
	fired := 0
	loop.Network.Receive(&receive_completion, first, buffer, REAL_DEADLINE,
		func(_ *time.Completion) { fired++ })
	testify.No_Error(t, loop.Network.Shutdown(first, nbio.SHUTDOWN_RECEIVE))
	testify.True(t,
		operating_system_run_until(t, driver, func() (finished bool) { return fired > 0 }))
	closed := false
	var close_completion time.Completion
	loop.Close(&close_completion, first, func(_ *time.Completion) { closed = true })
	testify.True(t,
		operating_system_run_until(t, driver, func() (finished bool) { return closed }))
	self_exec_close(loop, driver, first_connected)

	// Later socket may reuse descriptor, and must still deliver readiness.
	recycled, second := loopback_pair(t, loop, driver, listener, port)
	var send_completion time.Completion
	loop.Network.Send(&send_completion, second, []byte("pong"), REAL_DEADLINE, func(
		completed *time.Completion,
	) {
		testify.No_Error(t, completed.Error)
	})
	received := -1
	var second_receive time.Completion
	loop.Network.Receive(&second_receive, recycled, buffer, REAL_DEADLINE, func(
		completed *time.Completion,
	) {
		testify.No_Error(t, completed.Error)
		received = completed.Data
	})
	testify.True(t,
		operating_system_run_until(
			t, driver, func() (finished bool) { return received >= 0 },
		))
	testify.Equal(t, 4, received)
	testify.Equal(t, "pong", string(buffer[:4]))
}

// Build one accepted/connected loopback socket pair through loop, for tests that need live
// server-side socket with client on other end.
func loopback_pair(
	t *testing.T, loop nbio.IO, driver time.Driver, listener nbio.File, port int,
) (accepted nbio.File, connected nbio.File) {
	accepted = nbio.File(-1)
	connected, open_err := test_open_socket(loop)
	if !testify.No_Error(t, open_err) {
		return nbio.File(-1), nbio.File(-1)
	}
	connect_done := false
	var accept_completion time.Completion
	loop.Network.Accept(&accept_completion, listener, REAL_DEADLINE, func(
		completed *time.Completion,
	) {
		testify.No_Error(t, completed.Error)
		accepted = nbio.File(completed.Data)
	})
	var connect_completion time.Completion
	if !test_connect(t, loop, &connect_completion, connected, "127.0.0.1", port,
		func(completed *time.Completion) {
			testify.No_Error(t, completed.Error)
			connect_done = true
		},
	) {
		return nbio.File(-1), nbio.File(-1)
	}
	driver.Run_Until(REAL_DEADLINE,
		func() (finished bool) { return accepted > 0 && connect_done })
	testify.Positive(t, accepted)
	return accepted, connected
}

// Test_Operating_System_IO_Close_With_Armed_Receive verify Close reject descriptor still borrowed
// by submitted receive. Owner must shutdown, drain receive callback, and only then close.
func Test_Operating_System_IO_Close_With_Armed_Receive(t *testing.T) {
	port := free_port(t)
	clock := timeos.New_Operating_System_Clock()
	loop, _, driver := operating_system_loop(t, clock)
	listener, listen_err := test_listen(loop, driver, "127.0.0.1", port)
	if !testify.No_Error(t, listen_err) {
		return
	}
	accepted, connected := loopback_pair(t, loop, driver, listener, port)

	received := false
	var receive_completion time.Completion
	loop.Network.Receive(&receive_completion, accepted, make([]byte, 8), REAL_DEADLINE, func(
		_ *time.Completion,
	) {
		received = true
	})

	var close_completion time.Completion
	testify.Panics(t, func() {
		loop.Close(&close_completion, accepted, func(_ *time.Completion) {})
	})
	testify.No_Error(t, loop.Network.Shutdown(accepted, nbio.SHUTDOWN_BOTH))
	testify.True(t,
		operating_system_run_until(t, driver, func() (finished bool) { return received }))

	self_exec_close(loop, driver, accepted)
	self_exec_close(loop, driver, connected)
	self_exec_close(loop, driver, listener)
}

// Test_Operating_System_IO_Open open file through loop and read it back.
func Test_Operating_System_IO_Open(t *testing.T) {
	clock := timeos.New_Operating_System_Clock()
	loop, _, driver := operating_system_loop(t, clock)
	path := filepath.Join(t.TempDir(), "open")
	write_file(t, loop, driver, path, []byte("hello"))

	file, open_err := open_file(t, loop, driver, path)
	if !testify.No_Error(t, open_err) {
		return
	}
	buffer := make([]byte, 5)
	count := -1
	read_done := false
	var completion time.Completion
	loop.Storage.Read(&completion, file, buffer, 0, REAL_DEADLINE, func(
		completed *time.Completion,
	) {
		testify.No_Error(t, completed.Error)
		count = completed.Data
		read_done = true
	})
	testify.True(t,
		operating_system_run_until(t, driver, func() (finished bool) { return read_done }))
	close_file(t, loop, driver, path, file)

	testify.Equal(t, 5, count)
	testify.Equal(t, "hello", string(buffer))
}

// Test_Operating_System_IO_Create make file through loop and write to it.
func Test_Operating_System_IO_Create(t *testing.T) {
	clock := timeos.New_Operating_System_Clock()
	loop, _, driver := operating_system_loop(t, clock)
	path := filepath.Join(t.TempDir(), "create")
	file, create_err := create_file(t, loop, driver, path)
	if !testify.No_Error(t, create_err) {
		return
	}
	count := -1
	write_done := false
	var completion time.Completion
	loop.Storage.Write(&completion, file, []byte("world"), 0, REAL_DEADLINE, func(
		completed *time.Completion,
	) {
		testify.No_Error(t, completed.Error)
		count = completed.Data
		write_done = true
	})
	testify.True(t,
		operating_system_run_until(t, driver, func() (finished bool) { return write_done }))

	testify.Equal(t, 5, count)
	close_file(t, loop, driver, path, file)
	buffer := make([]byte, 5)
	read := read_file(t, loop, driver, path, buffer)
	testify.Equal(t, 5, read)
	testify.Equal(t, "world", string(buffer))
}

// Test_Operating_System_IO_File_Chain run whole file-operation chain: openat, write, fsync, read,
// and close all reuse caller-owned completions, and keep written bytes.
func Test_Operating_System_IO_File_Chain(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file-chain")
	clock := timeos.New_Operating_System_Clock()
	loop, _, driver := operating_system_loop(t, clock)
	opened := nbio.File(-1)
	var open_completion time.Completion
	loop.Storage.Open_At(&open_completion, nbio.DIRECTORY_CURRENT, path, nbio.Open_At_Options{
		Access: nbio.OPEN_READ_WRITE, Create: true, Truncate: true, Mode: 0o600,
	}, func(completed *time.Completion) {
		if !testify.No_Error(t, completed.Error) {
			return
		}
		opened = nbio.File(completed.Data)
	})
	testify.True(t,
		operating_system_run_until(
			t, driver, func() (finished bool) { return opened >= 0 },
		))

	written := false
	var write_completion time.Completion
	loop.Storage.Write(
		&write_completion, opened, []byte("hello"), 10, REAL_DEADLINE, func(
			completed *time.Completion,
		) {
			testify.No_Error(t, completed.Error)
			written = completed.Data == 5
		})
	testify.True(t,
		operating_system_run_until(t, driver, func() (finished bool) { return written }))

	synced := false
	var fsync_completion time.Completion
	loop.Storage.Fsync(
		&fsync_completion, opened, REAL_DEADLINE, func(completed *time.Completion) {
			testify.No_Error(t, completed.Error)
			synced = true
		})
	testify.True(t,
		operating_system_run_until(t, driver, func() (finished bool) { return synced }))

	buffer := make([]byte, 5)
	read := false
	var read_completion time.Completion
	loop.Storage.Read(&read_completion, opened, buffer, 10, REAL_DEADLINE, func(
		completed *time.Completion,
	) {
		testify.No_Error(t, completed.Error)
		read = completed.Data == len(buffer)
	})
	testify.True(t,
		operating_system_run_until(t, driver, func() (finished bool) { return read }))
	testify.Equal(t, "hello", string(buffer))
	self_exec_close(loop, driver, opened)
}

// Test_Operating_System_IO_Open_At_No_Follow verify OPEN_AT_NO_FOLLOW reject symbolic link in
// final path part, and does not reject ordinary file.
func Test_Operating_System_IO_Open_At_No_Follow(t *testing.T) {
	clock_setup := timeos.New_Operating_System_Clock()
	loop_setup, _, driver_setup := operating_system_loop(t, clock_setup)
	root := t.TempDir()
	target := filepath.Join(root, "target")
	link := filepath.Join(root, "link")
	write_file(t, loop_setup, driver_setup, target, []byte("secret"))
	driver_setup.Deinit()
	// Symbolic link is one filesystem shape io.IO cannot make, thus link itself stay raw call.
	// That absence is what this test exist to guard against follow of.
	if !testify.No_Error(t, syscall.Symlink(target, link)) {
		return
	}

	clock := timeos.New_Operating_System_Clock()
	loop, _, driver := operating_system_loop(t, clock)
	assert_open_at_result(t, loop, driver, target, nil)
	assert_open_at_result(t, loop, driver, link, errors.New("symbolic link must fail"))
}

// Open one path with no-follow and compare result class with expected_error.
func assert_open_at_result(
	t *testing.T,
	loop nbio.IO,
	driver time.Driver,
	path string,
	expected_error error,
) {
	t.Helper()
	opened := nbio.File(-1)
	completed := false
	var open_err error
	var completion time.Completion
	loop.Storage.Open_At(&completion, nbio.DIRECTORY_CURRENT, path, nbio.Open_At_Options{
		Access: nbio.OPEN_READ_ONLY,
		Flags:  nbio.OPEN_AT_NO_FOLLOW,
	}, func(result *time.Completion) {
		opened = nbio.File(result.Data)
		open_err = result.Error
		completed = true
	})
	testify.True(t,
		operating_system_run_until(t, driver, func() (finished bool) { return completed }))
	if expected_error == nil {
		if !testify.No_Error(t, open_err) {
			return
		}
		self_exec_close(loop, driver, opened)
		return
	}
	if !testify.Error(t, open_err, expected_error) {
		self_exec_close(loop, driver, opened)
	}
}

// Test_Operating_System_IO_Event verify Event reattachment contract: one trigger retire one
// listener on loop thread, after which same completion may arm again.
func Test_Operating_System_IO_Event(t *testing.T) {
	clock := timeos.New_Operating_System_Clock()
	_, pump, driver := operating_system_loop(t, clock)
	event, open_err := pump.Open_Event()
	if !testify.No_Error(t, open_err) {
		return
	}
	fired := 0
	var completion time.Completion
	callback := func(_ *time.Completion) { fired++ }
	pump.Event_Listen(event, &completion, callback)
	pump.Event_Trigger(event, &completion)
	testify.True(t,
		operating_system_run_until(t, driver, func() (finished bool) { return fired == 1 }))
	pump.Event_Listen(event, &completion, callback)
	pump.Event_Trigger(event, &completion)
	testify.True(t,
		operating_system_run_until(t, driver, func() (finished bool) { return fired == 2 }))
	pump.Close_Event(event)
}

// Test_Operating_System_IO_Peer_Address report remote address of accepted loopback connection.
func Test_Operating_System_IO_Peer_Address(t *testing.T) {
	port := free_port(t)
	clock := timeos.New_Operating_System_Clock()
	loop, _, driver := operating_system_loop(t, clock)
	listener, listen_err := test_listen(loop, driver, "127.0.0.1", port)
	if !testify.No_Error(t, listen_err) {
		return
	}

	accepted := nbio.File(-1)
	var accept_completion time.Completion
	loop.Network.Accept(&accept_completion, listener, REAL_DEADLINE, func(
		completed *time.Completion,
	) {
		accepted = nbio.File(completed.Data)
	})
	var connect_completion time.Completion
	connected, open_err := test_open_socket(loop)
	if !testify.No_Error(t, open_err) {
		return
	}
	if !test_connect(t, loop, &connect_completion, connected, "127.0.0.1", port,
		func(_ *time.Completion) {},
	) {
		return
	}
	driver.Run_Until(REAL_DEADLINE, func() (finished bool) { return accepted > 0 })

	testify.Positive(t, accepted)
	address, address_err := loop.Network.Peer_Address(accepted)
	testify.No_Error(t, address_err)
	testify.Equal(t, "127.0.0.1", address)
}

// Test_Operating_System_IO_Deinit_Rejects_Undrained_Extension verify Deinit cannot close
// Event/backend while Spawn still own repository-extension completion. Spawn is vehicle because
// it retire through same off-loop post path Event bridge.
func Test_Operating_System_IO_Deinit_Rejects_Undrained_Extension(t *testing.T) {
	clock := timeos.New_Operating_System_Clock()
	_, _, driver, system := operating_system_all(t, clock)
	drained := false
	var completion time.Completion
	system.Spawn(&completion, sysos.Process_Request{Path: "true"}, REAL_DEADLINE, func(
		_ *time.Completion, _ sysos.Process_Result, _ error,
	) {
		drained = true
	})
	testify.Panics(t, func() {
		driver.Deinit()
	})
	testify.True(t,
		operating_system_run_until(t, driver, func() (finished bool) { return drained }))
	driver.Deinit()
}

// Test_Operating_System_IO_Watch_Signal deliver real SIGTERM onto loop.
func Test_Operating_System_IO_Watch_Signal(t *testing.T) {
	clock := timeos.New_Operating_System_Clock()
	_, _, driver, system := operating_system_all(t, clock)
	got := sysos.Signal(-1)
	fired := 0
	var completion time.Completion
	system.Watch_Signal(&completion, sysos.SIGNAL_TERMINATE, REAL_DEADLINE, func(
		_ *time.Completion, signal sysos.Signal, err error,
	) {
		testify.No_Error(t, err)
		fired++
		got = signal
	})
	testify.No_Error(t, syscall.Kill(syscall.Getpid(), syscall.SIGTERM))
	driver.Run_Until(REAL_DEADLINE, func() (finished bool) { return fired > 0 })

	testify.Equal(t, 1, fired)
	testify.Equal(t, sysos.SIGNAL_TERMINATE, got)
}

// Test_Operating_System_IO_Watch_Signal_Deadline prove signal that never arrive retire extension
// completion exactly once, and permit backend deinitialization.
func Test_Operating_System_IO_Watch_Signal_Deadline(t *testing.T) {
	clock := timeos.New_Operating_System_Clock()
	_, _, driver, system := operating_system_all(t, clock)
	callback_count := 0
	got := sysos.Signal(-1)
	var operation_err error
	var completion time.Completion
	system.Watch_Signal(
		&completion, sysos.SIGNAL_TERMINATE, REAL_OPERATION_DEADLINE, func(
			_ *time.Completion, signal sysos.Signal, err error,
		) {
			callback_count++
			got = signal
			operation_err = err
		})
	testify.True(t, operating_system_run_until(
		t, driver, func() (finished bool) { return callback_count > 0 },
	))
	testify.Equal(t, 1, callback_count)
	testify.Error_Is(t, operation_err, time.Deadline_Exceeded)
	testify.Equal(t, sysos.Signal(-1), got)
	driver.Deinit()
}

// Test_Operating_System_IO_Spawn run real commands through loop: success with captured output,
// and non-zero exit reported without start error.
func Test_Operating_System_IO_Spawn(t *testing.T) {
	clock := timeos.New_Operating_System_Clock()
	_, _, driver, system := operating_system_all(t, clock)

	echo := sysos.Process_Result{}
	echoed := false
	var echo_completion time.Completion
	system.Spawn(
		&echo_completion,
		sysos.Process_Request{Path: "/bin/echo", Arguments: []string{"hi"}},
		REAL_DEADLINE,
		func(
			_ *time.Completion, result sysos.Process_Result, err error,
		) {
			testify.No_Error(t, err)
			echo = result
			echoed = true
		})
	operating_system_run_until(t, driver, func() (finished bool) { return echoed })

	testify.True(t, echoed)
	testify.Zero(t, echo.Exit)
	testify.Equal(t, "hi\n", string(echo.Output))

	fail := sysos.Process_Result{}
	failed := false
	var fail_completion time.Completion
	system.Spawn(&fail_completion, sysos.Process_Request{
		Path: "/bin/sh", Arguments: []string{"-c", "exit 1"},
	}, REAL_DEADLINE, func(
		_ *time.Completion, result sysos.Process_Result, err error,
	) {
		testify.No_Error(t, err)
		fail = result
		failed = true
	})
	operating_system_run_until(t, driver, func() (finished bool) { return failed })

	testify.True(t, failed)
	testify.Equal(t, 1, fail.Exit)
}

// Test_Operating_System_IO_Spawn_Streams_To_Sink run command with live stdout sink. It confirm
// backend stream child output to writer as it run, instead of capture of it — affordance long
// build need — and leave Output empty.
func Test_Operating_System_IO_Spawn_Streams_To_Sink(t *testing.T) {
	clock := timeos.New_Operating_System_Clock()
	_, _, driver, system := operating_system_all(t, clock)

	streamed := nbio.Stream_Memory{Memory: make([]byte, 64)}
	result := sysos.Process_Result{}
	done := false
	var completion time.Completion
	system.Spawn(&completion, sysos.Process_Request{
		Path: "/bin/echo", Arguments: []string{"hi"},
		Stdout: nbio.Memory_To_Stream(&streamed),
	}, REAL_DEADLINE, func(
		_ *time.Completion, spawned sysos.Process_Result, err error,
	) {
		testify.No_Error(t, err)
		result = spawned
		done = true
	})
	operating_system_run_until(t, driver, func() (finished bool) { return done })

	testify.True(t, done)
	testify.Equal(t, "hi\n", string(streamed.Memory[:streamed.Cursor]))
	testify.Empty(t, result.Output)
}

// Read process identifier deadline fixture wrote to path, through loop. This package is io
// gateway, thus its own tests are where loop file operations belong. Fixture write one short
// decimal, thus regular file return it whole in single read.
func spawn_recorded_identifier(
	t *testing.T, loop nbio.IO, driver time.Driver, path string,
) (identifier int) {
	t.Helper()
	file, open_err := open_file(t, loop, driver, path)
	if !testify.No_Error(t, open_err) {
		return 0
	}
	process_buffer := make([]byte, PROCESS_IDENTIFIER_BYTES)
	process_count := 0
	read_done := false
	var read_completion time.Completion
	loop.Storage.Read(
		&read_completion, file, process_buffer, 0, REAL_DEADLINE, func(
			completed *time.Completion,
		) {
			testify.No_Error(t, completed.Error)
			process_count = completed.Data
			read_done = true
		})
	testify.True(t,
		operating_system_run_until(t, driver, func() (finished bool) { return read_done }))
	close_done := false
	var close_completion time.Completion
	loop.Close(&close_completion, file, func(completed *time.Completion) {
		testify.No_Error(t, completed.Error)
		close_done = true
	})
	testify.True(t,
		operating_system_run_until(t, driver, func() (finished bool) { return close_done }))
	identifier, parse_err := strconv.Atoi(
		strings.TrimSpace(string(process_buffer[:process_count])))
	testify.No_Error(t, parse_err)
	return identifier
}

// Test_Operating_System_IO_Spawn_Deadline prove timeout kill whole subprocess group, keep output
// captured before expiry, and deliver one terminal callback.
func Test_Operating_System_IO_Spawn_Deadline(t *testing.T) {
	clock := timeos.New_Operating_System_Clock()
	loop, _, driver, system := operating_system_all(t, clock)
	process_path := filepath.Join(t.TempDir(), "process")
	request := sysos.Process_Request{
		Path: "/bin/sh",
		Arguments: []string{
			"-c", "printf '%d' $$ > \"$1\"; printf partial; sleep 30 & wait",
			"bounded-spawn", process_path,
		},
	}
	callback_count := 0
	result := sysos.Process_Result{}
	var operation_err error
	var completion time.Completion
	system.Spawn(&completion, request, 100*time.MILLISECOND, func(
		_ *time.Completion, spawned sysos.Process_Result, err error,
	) {
		callback_count++
		result = spawned
		operation_err = err
	})
	testify.True(t, operating_system_run_until(
		t, driver, func() (finished bool) { return callback_count > 0 },
	))
	testify.Equal(t, 1, callback_count)
	testify.Error_Is(t, operation_err, time.Deadline_Exceeded)
	testify.Equal(t, "partial", string(result.Output))
	process_identifier := spawn_recorded_identifier(t, loop, driver, process_path)
	group_exited, group_err := process_group_wait_for_exit(driver, process_identifier)
	testify.No_Error(t, group_err, process_identifier)
	testify.True(t, group_exited, process_identifier)
	driver.Run_For(2 * REAL_OPERATION_DEADLINE)
	testify.Equal(t, 1, callback_count)
	driver.Deinit()
}

// Test_Operating_System_IO_Spawn_Reaps_After_Deadline verify deadline path still reap.
// Completion retire early on Deadline_Exceeded, thus exit event arrive for child nobody wait on,
// and only own tracking of backend keep reap on that path.
func Test_Operating_System_IO_Spawn_Reaps_After_Deadline(t *testing.T) {
	clock := timeos.New_Operating_System_Clock()
	_, _, driver, system := operating_system_all(t, clock)
	callback_count := 0
	var operation_err error
	var completion time.Completion
	system.Spawn(&completion, sysos.Process_Request{
		Path: "/bin/sleep", Arguments: []string{"30"},
	}, 50*time.MILLISECOND, func(
		_ *time.Completion, _ sysos.Process_Result, err error,
	) {
		callback_count++
		operation_err = err
	})
	testify.True(t, operating_system_run_until(
		t, driver, func() (finished bool) { return callback_count > 0 },
	))
	testify.Error_Is(t, operation_err, time.Deadline_Exceeded)
	// Deinit assert spawn table is empty, thus missed reap panic here, not leak one
	// process-table slot for every timed-out child.
	driver.Deinit()
}

// Test_Operating_System_IO_Spawn_Bounds_Lingering_Drain verify pipe-cleanup bound start when
// child exit, not when its deadline expire. Child that finish at once but leave grandchild
// holding standard output must retire about one second later, not wait out whole deadline. This
// is bound exec.Cmd.WaitDelay supplied before.
func Test_Operating_System_IO_Spawn_Bounds_Lingering_Drain(t *testing.T) {
	clock := timeos.New_Operating_System_Clock()
	_, _, driver, system := operating_system_all(t, clock)
	const SPAWN_DEADLINE = 4 * time.SECOND
	started := clock.Now_Monotonic()
	callback_count := 0
	result := sysos.Process_Result{}
	var operation_err error
	var completion time.Completion
	system.Spawn(&completion, sysos.Process_Request{
		Path:      "/bin/sh",
		Arguments: []string{"-c", "printf quick; sleep 30 & exit 0"},
	}, SPAWN_DEADLINE, func(
		_ *time.Completion, spawned sysos.Process_Result, err error,
	) {
		callback_count++
		result = spawned
		operation_err = err
	})
	testify.True(t, operating_system_run_until(
		t, driver, func() (finished bool) { return callback_count > 0 },
	))
	elapsed := time.Duration(int64(clock.Now_Monotonic()) - int64(started))
	testify.True(t, elapsed < SPAWN_DEADLINE, elapsed)
	// Child exited clean, thus its code survive. Output is incomplete because drain was cut
	// short, and Deadline_Exceeded is how caller learn that.
	testify.Zero(t, result.Exit)
	testify.Equal(t, "quick", string(result.Output))
	testify.Error_Is(t, operation_err, time.Deadline_Exceeded)
	driver.Deinit()
}

// Test_Operating_System_IO_Spawn_Concurrent verify two children in flight at once keep their
// pipes separate. Pipe end that leak into fork of other child would hold standard input of that
// child open, thus this fail by deadlock, not by wrong result.
func Test_Operating_System_IO_Spawn_Concurrent(t *testing.T) {
	clock := timeos.New_Operating_System_Clock()
	_, _, driver, system := operating_system_all(t, clock)
	const SPAWNS_COUNT = 4
	finished := 0
	outputs := make([]string, SPAWNS_COUNT)
	completions := make([]*time.Completion, SPAWNS_COUNT)
	for index := 0; index < SPAWNS_COUNT; index++ {
		position := index
		completions[position] = &time.Completion{}
		system.Spawn(completions[position], sysos.Process_Request{
			Path:  "/bin/cat",
			Input: []byte(strconv.Itoa(position)),
		}, REAL_DEADLINE, func(
			_ *time.Completion, spawned sysos.Process_Result, err error,
		) {
			testify.No_Error(t, err, position)
			outputs[position] = string(spawned.Output)
			finished++
		})
	}
	testify.True(t, operating_system_run_until(
		t, driver, func() (finished_all bool) { return finished == SPAWNS_COUNT },
	))
	for index := 0; index < SPAWNS_COUNT; index++ {
		want := strconv.Itoa(index)
		testify.Equal(t, want, outputs[index], index)
	}
	driver.Deinit()
}

// Test_Operating_System_IO_Spawn_Feeds_Input verify Process_Request.Input reach child standard
// input, and write end close, thus child observe end-of-file, not wait for more.
func Test_Operating_System_IO_Spawn_Feeds_Input(t *testing.T) {
	clock := timeos.New_Operating_System_Clock()
	_, _, driver, system := operating_system_all(t, clock)
	callback_count := 0
	result := sysos.Process_Result{}
	var operation_err error
	var completion time.Completion
	system.Spawn(&completion, sysos.Process_Request{
		Path: "/bin/cat", Input: []byte("fed through stdin"),
	}, REAL_DEADLINE, func(
		_ *time.Completion, spawned sysos.Process_Result, err error,
	) {
		callback_count++
		result = spawned
		operation_err = err
	})
	testify.True(t, operating_system_run_until(
		t, driver, func() (finished bool) { return callback_count > 0 },
	))
	testify.No_Error(t, operation_err)
	testify.Equal(t, "fed through stdin", string(result.Output))
	driver.Deinit()
}

// Test_Operating_System_IO_Spawn_Drains_Full_Pipe verify output larger than one pipe buffer
// still complete. Child that fill pipe block until loop read it, thus this fail by deadlock when
// reads are not armed for whole life of child.
func Test_Operating_System_IO_Spawn_Drains_Full_Pipe(t *testing.T) {
	clock := timeos.New_Operating_System_Clock()
	_, _, driver, system := operating_system_all(t, clock)
	const LINES = 20000
	callback_count := 0
	result := sysos.Process_Result{}
	var operation_err error
	var completion time.Completion
	system.Spawn(&completion, sysos.Process_Request{
		Path: "/bin/sh",
		Arguments: []string{
			"-c", "i=0; while [ $i -lt 20000 ]; do echo line; i=$((i+1)); done",
		},
	}, REAL_DEADLINE, func(
		_ *time.Completion, spawned sysos.Process_Result, err error,
	) {
		callback_count++
		result = spawned
		operation_err = err
	})
	testify.True(t, operating_system_run_until(
		t, driver, func() (finished bool) { return callback_count > 0 },
	))
	testify.No_Error(t, operation_err)
	testify.Count(t, result.Output, LINES*len("line\n"))
	driver.Deinit()
}

// Test_Operating_System_IO_Spawn_Resolves_Path verify bare command name still resolve through
// PATH. exec.Command supplied this before, and syscall.StartProcess does not.
func Test_Operating_System_IO_Spawn_Resolves_Path(t *testing.T) {
	clock := timeos.New_Operating_System_Clock()
	_, _, driver, system := operating_system_all(t, clock)
	callback_count := 0
	result := sysos.Process_Result{}
	var operation_err error
	var completion time.Completion
	system.Spawn(
		&completion,
		sysos.Process_Request{Path: "echo", Arguments: []string{"resolved"}},
		REAL_DEADLINE,
		func(
			_ *time.Completion, spawned sysos.Process_Result, err error,
		) {
			callback_count++
			result = spawned
			operation_err = err
		})
	testify.True(t, operating_system_run_until(
		t, driver, func() (finished bool) { return callback_count > 0 },
	))
	testify.No_Error(t, operation_err)
	testify.Equal(t, "resolved", strings.TrimSpace(string(result.Output)))
	driver.Deinit()
}

// Test_Operating_System_IO_Spawn_Reports_Missing_Command verify unresolvable command name fail
// spawn, not start anything.
func Test_Operating_System_IO_Spawn_Reports_Missing_Command(t *testing.T) {
	clock := timeos.New_Operating_System_Clock()
	_, _, driver, system := operating_system_all(t, clock)
	callback_count := 0
	var operation_err error
	var completion time.Completion
	system.Spawn(&completion, sysos.Process_Request{Path: "no-such-command-anywhere"},
		REAL_DEADLINE, func(
			_ *time.Completion, _ sysos.Process_Result, err error,
		) {
			callback_count++
			operation_err = err
		})
	testify.True(t, operating_system_run_until(
		t, driver, func() (finished bool) { return callback_count > 0 },
	))
	testify.Error(t, operation_err)
	driver.Deinit()
}

// Wait for host init to reap killed grandchildren, without accept of live bounded process.
func process_group_wait_for_exit(
	driver time.Driver, process_identifier int,
) (exited bool, err error) {
	for attempt_index := 0; attempt_index < 80; attempt_index++ {
		kill_err := syscall.Kill(-process_identifier, 0)
		if errors.Is(kill_err, syscall.ESRCH) {
			return true, nil
		}
		if kill_err != nil {
			return false, kill_err
		}
		drive_err := driver.Run_For(REAL_OPERATION_DEADLINE)
		if drive_err != nil {
			return false, drive_err
		}
	}
	return false, nil
}

// Test_Operating_System_IO_Make_Directory cover what converging mkdir can hide: repeat call,
// relative path, trailing slash, and final component that already exist as file.
func Test_Operating_System_IO_Make_Directory(t *testing.T) {
	clock := timeos.New_Operating_System_Clock()
	loop, _, driver := operating_system_loop(t, clock)
	root := t.TempDir()

	nested := filepath.Join(root, "one", "two", "three")
	testify.No_Error(t, make_directory(t, loop, driver, nested))
	// Repeat converge, not report that directory exist.
	testify.No_Error(t, make_directory(t, loop, driver, nested))
	parents := []string{
		filepath.Join(root, "one"), filepath.Join(root, "one", "two"),
	}
	for _, path := range parents {
		status, _ := loop.Storage.Status(path)
		testify.True(t, status.Is_Directory, path)
	}

	slashed := filepath.Join(root, "four", "five") + "/"
	testify.No_Error(t, make_directory(t, loop, driver, slashed))
	slashed_status, _ := loop.Storage.Status(filepath.Join(root, "four", "five"))
	testify.True(t, slashed_status.Is_Directory)

	// Final component that already exist as file must report error, not converge, because
	// caller asked for directory and does not have one.
	occupied := filepath.Join(root, "occupied")
	write_file(t, loop, driver, occupied, []byte("not a directory"))
	// Mkdir_At report Path_Exists for file too, and Make_Directory converge on that. Caller
	// thus learn difference from Status, not from create.
	testify.No_Error(t, make_directory(t, loop, driver, occupied))
	status, _ := loop.Storage.Status(occupied)
	testify.False(t, status.Is_Directory)
	driver.Deinit()
}

// Test_Operating_System_IO_Directory exercise filesystem-traversal operations on real temp tree:
// Make_Directory build nested path, into which fixture file is seeded, and Status and
// Read_Directory then report tree shape, absent path included.
func Test_Operating_System_IO_Directory(t *testing.T) {
	clock := timeos.New_Operating_System_Clock()
	loop, _, driver := operating_system_loop(t, clock)

	root := t.TempDir()
	nested := filepath.Join(root, "a", "b")
	testify.No_Error(t, make_directory(t, loop, driver, nested))
	// Make of file through loop prove Make_Directory built parents: Create fail when
	// directory above path does not exist.
	file_path := filepath.Join(nested, "file.txt")
	write_file(t, loop, driver, file_path, []byte("hello"))

	directory_status, _ := loop.Storage.Status(nested)
	testify.True(t, directory_status.Exists)
	testify.True(t, directory_status.Is_Directory)
	testify.False(t, directory_status.Is_Regular)
	regular_status, _ := loop.Storage.Status(file_path)
	testify.True(t, regular_status.Exists)
	testify.False(t, regular_status.Is_Directory)
	testify.True(t, regular_status.Is_Regular)
	absent_status, _ := loop.Storage.Status(filepath.Join(root, "nope"))
	testify.False(t, absent_status.Exists)
	testify.False(t, absent_status.Is_Regular)

	entries, read_err := read_directory(t, loop, driver, nested)
	testify.No_Error(t, read_err)
	found := false
	for _, entry := range entries {
		if entry.Name != "file.txt" {
			continue
		}
		found = true
		testify.False(t, entry.Is_Directory)
	}
	testify.True(t, found, nested, entries)
}

type loopback_roundtrip_input struct {
	Test      *testing.T
	Timeline  nbio.IO
	Driver    time.Driver
	Connected nbio.File
	Accepted  nbio.File
}

func loopback_assert_roundtrip(input *loopback_roundtrip_input) {
	input.Test.Helper()
	var send_completion time.Completion
	input.Timeline.Network.Send(
		&send_completion, input.Connected, []byte("ping"), REAL_DEADLINE, func(
			completed *time.Completion,
		) {
			testify.No_Error(input.Test, completed.Error)
		})
	buffer := make([]byte, 16)
	received := -1
	var receive_completion time.Completion
	input.Timeline.Network.Receive(
		&receive_completion, input.Accepted, buffer, REAL_DEADLINE, func(
			completed *time.Completion,
		) {
			testify.No_Error(input.Test, completed.Error)
			received = completed.Data
		})
	input.Driver.Run_Until(REAL_DEADLINE, func() (finished bool) { return received >= 0 })
	testify.Equal(input.Test, 4, received)
	testify.Equal(input.Test, "ping", string(buffer[:4]))
}

func socket_fcntl(t *testing.T, socket nbio.File, command int) (flags int) {
	t.Helper()
	value, _, errno := syscall.Syscall(
		syscall.SYS_FCNTL, uintptr(socket), uintptr(command), 0)
	testify.Zero(t, errno, command)
	return int(value)
}

// Return probably-free TCP port by bind and release of one through standard library. Used only
// to pick target for backend under test.
func free_port(t *testing.T) (port int) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if !testify.No_Error(t, err) {
		return 0
	}
	port = listener.Addr().(*net.TCPAddr).Port
	testify.No_Error(t, listener.Close())
	return port
}

// Close socket asynchronously and drive its completion.
func self_exec_close(loop nbio.IO, driver time.Driver, socket nbio.File) {
	closed := false
	var completion time.Completion
	loop.Close(&completion, socket, func(_ *time.Completion) { closed = true })
	driver.Run_Until(REAL_DEADLINE, func() (finished bool) { return closed })
}
