package nbio

import (
	"errors"
	"net"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"unsafe"

	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/simulation/nbio"
	"local/james-orcales/shared/simulation/os"
	"local/james-orcales/shared/simulation/time"
	"local/james-orcales/shared/testify"
)

// Run_Until cap of real-backend tests: generous, since completion return pump instant it fire —
// this bound only bite genuine hang, and fail test instead of block until package timeout.
const REAL_DEADLINE = 5 * time.SECOND

// Short finite operation deadline used to prove dormant kernel wait retire prompt.
const REAL_OPERATION_DEADLINE = 25 * time.MILLISECOND

// Bound buffer holding process identifier read back from fixture file.
const PROCESS_IDENTIFIER_BYTES = 64

// OPERATING_SYSTEM_TEST_TIMEOUT_CAPACITY leaves one slot per byte value for stress tests.
const OPERATING_SYSTEM_TEST_TIMEOUT_CAPACITY = int(bits.WORD_8_MAXIMUM) + 1

// OPERATING_SYSTEM_TEST_COMPLETION_CAPACITY matches timeout burst capacity.
const OPERATING_SYSTEM_TEST_COMPLETION_CAPACITY = OPERATING_SYSTEM_TEST_TIMEOUT_CAPACITY

// OPERATING_SYSTEM_TEST_OPERATION_CAPACITY matches completion burst capacity.
const OPERATING_SYSTEM_TEST_OPERATION_CAPACITY = OPERATING_SYSTEM_TEST_COMPLETION_CAPACITY

// OPERATING_SYSTEM_TEST_DESCRIPTOR_CAPACITY uses one machine-bit census slot per descriptor.
const OPERATING_SYSTEM_TEST_DESCRIPTOR_CAPACITY = bits.WORD_SIZE

// OPERATING_SYSTEM_TEST_SIGNAL_CAPACITY keeps a quarter-word of simultaneous watches.
const OPERATING_SYSTEM_TEST_SIGNAL_CAPACITY = bits.WORD_SIZE / 4

// OPERATING_SYSTEM_TEST_SPAWN_CAPACITY matches simultaneous signal-watch capacity.
const OPERATING_SYSTEM_TEST_SPAWN_CAPACITY = OPERATING_SYSTEM_TEST_SIGNAL_CAPACITY

// OPERATING_SYSTEM_TEST_PLATFORM_CAPACITY matches operation retry capacity.
const OPERATING_SYSTEM_TEST_PLATFORM_CAPACITY = OPERATING_SYSTEM_TEST_OPERATION_CAPACITY

// OPERATING_SYSTEM_ALLOCATION_FILE_COUNT covers both AllocsPerRun invocations.
const OPERATING_SYSTEM_ALLOCATION_FILE_COUNT = PIPE_ENDS

// OPERATING_SYSTEM_ALLOCATION_BUFFER_BYTES lets both measured directory passes return names.
const OPERATING_SYSTEM_ALLOCATION_BUFFER_BYTES = DIRECTORY_READ_BYTES

// OPERATING_SYSTEM_ALLOCATION_ENTRY_CAPACITY matches one IPv4-width directory batch.
const OPERATING_SYSTEM_ALLOCATION_ENTRY_CAPACITY = nbio.IPV4_ADDRESS_BYTES

// OPERATING_SYSTEM_ALLOCATION_NETWORK_FILES holds one listener and two socket pairs.
const OPERATING_SYSTEM_ALLOCATION_NETWORK_FILES = 1 + PIPE_ENDS*PIPE_ENDS

// OPERATING_SYSTEM_ALLOCATION_NETWORK_BUFFER_BYTES holds two IPv4 addresses.
const OPERATING_SYSTEM_ALLOCATION_NETWORK_BUFFER_BYTES = PIPE_ENDS * nbio.IPV4_ADDRESS_BYTES

type operating_system_test_memory struct {
	State               Operating_System
	Timeouts            [OPERATING_SYSTEM_TEST_TIMEOUT_CAPACITY]*time.Completion
	Completed           [OPERATING_SYSTEM_TEST_COMPLETION_CAPACITY]*time.Completion
	Operations          [OPERATING_SYSTEM_TEST_OPERATION_CAPACITY]Operating_System_Operation
	Operation_Registry  [OPERATING_SYSTEM_TEST_OPERATION_CAPACITY]*Operating_System_Operation
	Descriptors         [OPERATING_SYSTEM_TEST_DESCRIPTOR_CAPACITY]Operating_System_Descriptor
	Signal_Waiters      [OPERATING_SYSTEM_TEST_SIGNAL_CAPACITY]Signal_Waiter
	Spawns              [OPERATING_SYSTEM_TEST_SPAWN_CAPACITY]Spawn
	Spawn_Registry      [OPERATING_SYSTEM_TEST_SPAWN_CAPACITY]*Spawn
	Platform_Operations [OPERATING_SYSTEM_TEST_PLATFORM_CAPACITY]*Operating_System_Operation
}

func operating_system_memory_view(
	memory *operating_system_test_memory,
) (view Operating_System_Memory) {
	return Operating_System_Memory{
		Timeouts:            memory.Timeouts[:],
		Completed:           memory.Completed[:],
		Operations:          memory.Operations[:],
		Operation_Registry:  memory.Operation_Registry[:],
		Descriptors:         memory.Descriptors[:],
		Signal_Waiters:      memory.Signal_Waiters[:],
		Spawns:              memory.Spawns[:],
		Spawn_Registry:      memory.Spawn_Registry[:],
		Platform_Operations: memory.Platform_Operations[:],
	}
}

type operating_system_clock_state struct {
	Maximum atomic.Int64
}

type clock_value struct {
	Moment time.Monotonic_Moment
	Step   time.Duration
}

// A local host clock keeps these backend tests independent of another backend package.
func new_operating_system_clock() (clock time.Clock) {
	state := &operating_system_clock_state{}
	return time.Clock{
		State:         unsafe.Pointer(state),
		Now_Monotonic: operating_system_clock_now_monotonic,
		Now_Realtime:  operating_system_clock_now_realtime,
	}
}

func operating_system_clock_read() (nanoseconds int64) {
	value := syscall.Timeval{}
	if err := syscall.Gettimeofday(&value); err != nil {
		panic(err)
	}
	return int64(value.Sec)*int64(time.SECOND) +
		int64(value.Usec)*int64(time.MICROSECOND)
}

func operating_system_clock_now_monotonic(state unsafe.Pointer) (moment time.Monotonic_Moment) {
	maximum := &(*operating_system_clock_state)(state).Maximum
	current := operating_system_clock_read()
	previous := maximum.Load()
	if current <= previous {
		return time.Monotonic_Moment(previous)
	}
	if maximum.CompareAndSwap(previous, current) {
		return time.Monotonic_Moment(current)
	}
	return time.Monotonic_Moment(maximum.Load())
}

func operating_system_clock_now_realtime(_ unsafe.Pointer) (moment time.Moment) {
	return time.Moment(operating_system_clock_read())
}

func clock_value_to_clock(value *clock_value) (clock time.Clock) {
	return time.Clock{
		State:         unsafe.Pointer(value),
		Now_Monotonic: clock_value_now_monotonic,
		Now_Realtime:  clock_value_now_realtime,
	}
}

func clock_value_now_monotonic(state unsafe.Pointer) (moment time.Monotonic_Moment) {
	value := (*clock_value)(state)
	value.Moment += time.Monotonic_Moment(value.Step)
	return value.Moment
}

func clock_value_now_realtime(state unsafe.Pointer) (moment time.Moment) {
	return time.Moment((*clock_value)(state).Moment)
}

func test_decimal(content []byte) (value int, valid bool) {
	start := 0
	for start < len(content) {
		if !test_space(content[start]) {
			break
		}
		start++
	}
	end_count := len(content)
	for end_count > start {
		if !test_space(content[end_count-1]) {
			break
		}
		end_count--
	}
	if start == end_count {
		return 0, false
	}
	for _, character := range content[start:end_count] {
		if character < '0' {
			return 0, false
		}
		if character > '9' {
			return 0, false
		}
		digit := int(character - '0')
		if value > (bits.INTEGER_MAXIMUM-digit)/nbio.DECIMAL_RADIX {
			return 0, false
		}
		value = value*nbio.DECIMAL_RADIX + digit
	}
	return value, true
}

func test_decimal_text(value int) (text string) {
	if value < 0 {
		return ""
	}
	buffer := [bits.WORD_SIZE]byte{}
	position_count := len(buffer)
	for digit_index := 0; digit_index < len(buffer); digit_index++ {
		position_count--
		buffer[position_count] = byte(value%nbio.DECIMAL_RADIX) + '0'
		value /= nbio.DECIMAL_RADIX
		if value == 0 {
			return string(buffer[position_count:])
		}
	}
	return ""
}

func test_trimmed_text(content []byte) (text string) {
	start := 0
	for start < len(content) {
		if !test_space(content[start]) {
			break
		}
		start++
	}
	end_count := len(content)
	for end_count > start {
		if !test_space(content[end_count-1]) {
			break
		}
		end_count--
	}
	return string(content[start:end_count])
}

func test_space(character byte) (space bool) {
	return character == ' ' || character == '\t' || character == '\n' ||
		character == '\r' || character == '\v' || character == '\f'
}

// A required bound must fail before the backend can borrow a caller-owned descriptor.
func Test_Operating_System_Storage_Rejects_Disabled_Timeouts(t *testing.T) {
	for _, operation := range []string{"read", "write", "fsync"} {
		for _, timeout := range []time.Duration{0, -time.NANOSECOND} {
			loop, _, driver := operating_system_loop(
				t, new_operating_system_clock(),
			)
			testify.Panics(t, func() {
				var completion time.Completion
				if operation == "read" {
					nbio.Storage_Read(
						loop.Storage, &completion, nbio.File(-1), nil, 0,
						timeout,
						func(_ *time.Completion) {})
					return
				}
				if operation == "write" {
					nbio.Storage_Write(
						loop.Storage, &completion, nbio.File(-1), nil, 0,
						timeout,
						func(_ *time.Completion) {})
					return
				}
				nbio.Storage_Fsync(
					loop.Storage, &completion, nbio.File(-1), timeout,
					func(_ *time.Completion) {})
			}, operation, timeout)
			time.Driver_Deinit(driver)
		}
	}
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
	nbio.Storage_Write(loop.Storage, &completion, file, content, 0, REAL_DEADLINE, func(
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
		nbio.Storage_Mkdir_At(
			loop.Storage, &completion, nbio.DIRECTORY_CURRENT, component, 0o755, func(
				completed *time.Completion,
			) {
				// Existing component converge because a parent walk must accept the
				// Path_Exists that mkdirat reports for a directory already there.
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
	nbio.Storage_Open_At(loop.Storage, &completion, nbio.DIRECTORY_CURRENT, path, options, func(
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
	pass_storage := make([]nbio.Directory_Entry, DIRECTORY_PASS_BYTES)
	for pass_number_index := 0; pass_number_index < DIRECTORY_PASSES_MAX; pass_number_index++ {
		done := false
		var pass []nbio.Directory_Entry
		var pass_err error
		var completion time.Completion
		nbio.Storage_Get_Directory_Entries(
			loop.Storage, &completion, directory, buffer, pass_storage,
			func(completed *time.Completion) {
				pass = pass_storage[:completed.Data]
				pass_err = completed.Error
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
	nbio.IO_Close(loop, &close_completion, directory, func(_ *time.Completion) {
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
	nbio.Storage_Read(loop.Storage, &completion, file, buffer, 0, REAL_DEADLINE, func(
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
	nbio.IO_Close(loop, &completion, file, func(completed *time.Completion) {
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
) (loop nbio.IO, pump time.Timeline, driver time.Driver, system os.OS) {
	t.Helper()
	memory := &operating_system_test_memory{}
	loop, pump, driver, system, err := New_Operating_System_IO(
		&memory.State, operating_system_memory_view(memory),
		clock, 32, 0, operating_system_ambient())
	if !testify.No_Error(t, err) {
		return nbio.IO{}, time.Timeline{}, time.Driver{}, os.OS{}
	}
	return loop, pump, driver, system
}

// Test_Operating_System_Constructor_Heap_Allocation guards backend ownership before submission.
func Test_Operating_System_Constructor_Heap_Allocation(t *testing.T) {
	clock_state := operating_system_clock_state{}
	clock := time.Clock{
		State:         unsafe.Pointer(&clock_state),
		Now_Monotonic: operating_system_clock_now_monotonic,
		Now_Realtime:  operating_system_clock_now_realtime,
	}
	ambient := operating_system_ambient()
	memory := operating_system_test_memory{}
	var loop nbio.IO
	var driver time.Driver
	var construct_err error
	testify.Zero_Allocation(t, func() {
		loop, _, driver, _, construct_err = New_Operating_System_IO(
			&memory.State, operating_system_memory_view(&memory),
			clock, 32, 0, ambient,
		)
		if construct_err == nil {
			nbio.IO_Deinit(loop)
			time.Driver_Deinit(driver)
		}
	})
	testify.No_Error(t, construct_err)
}

// Invalid input belongs in allocation contract because validation runs before any kernel state.
func Test_Operating_System_Constructor_Error_Heap_Allocation(t *testing.T) {
	clock := new_operating_system_clock()
	ambient := operating_system_ambient()
	memory := operating_system_test_memory{}
	var construct_err error
	testify.Zero_Allocation(t, func() {
		_, _, _, _, construct_err = New_Operating_System_IO(
			&memory.State, operating_system_memory_view(&memory),
			clock, 0, 0, ambient,
		)
	})
	testify.Error(t, construct_err)
}

// Kernel option failure cannot allocate merely to decorate errno.
func Test_Socket_Option_Error_Heap_Allocation(t *testing.T) {
	var option_err error
	testify.Zero_Allocation(t, func() {
		option_err = socket_tcp_options_set(-1, test_tcp_options())
	})
	testify.Error(t, option_err)
}

// Public constructors must retire rejected descriptors without allocation.
func Test_Operating_System_Socket_Error_Heap_Allocation(t *testing.T) {
	clock := new_operating_system_clock()
	loop, _, driver := operating_system_loop(t, clock)
	var socket nbio.File
	var socket_err error
	t.Run("TCP", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			socket, socket_err = nbio.Network_Socket_TCP(
				loop.Network, nbio.FAMILY_IPV4, nbio.TCP_Options{},
			)
		})
		testify.Error(t, socket_err)
		testify.Equal(t, nbio.File(-1), socket)
	})
	t.Run("UDP", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			socket, socket_err = nbio.Network_Socket_UDP(
				loop.Network, nbio.FAMILY_IPV4, nbio.UDP_Options{},
			)
		})
		testify.Error(t, socket_err)
		testify.Equal(t, nbio.File(-1), socket)
	})
	nbio.IO_Deinit(loop)
	time.Driver_Deinit(driver)
}

// Test_Operating_System_Read_Heap_Allocation guards submission and retirement together.
func Test_Operating_System_Read_Heap_Allocation(t *testing.T) {
	clock := new_operating_system_clock()
	loop, _, driver := operating_system_loop(t, clock)
	path := filepath.Join(t.TempDir(), "allocation-read")
	write_file(t, loop, driver, path, []byte("data"))
	file, open_err := open_file(t, loop, driver, path)
	if !testify.No_Error(t, open_err) {
		return
	}
	harness := operating_system_allocation_harness{Driver: driver}
	harness.Completion.Backend = unsafe.Pointer(&harness)
	testify.Zero_Allocation(t, func() {
		harness.Called = false
		nbio.Storage_Read(
			loop.Storage, &harness.Completion, file, harness.Buffer[:], 0,
			REAL_DEADLINE, operating_system_allocation_callback,
		)
		harness.Driver_Error = time.Driver_Run(driver)
	})
	testify.No_Error(t, harness.Driver_Error)
	testify.True(t, harness.Called)
	close_file(t, loop, driver, path, file)
}

type operating_system_allocation_harness struct {
	Driver       time.Driver
	Completion   time.Completion
	Buffer       [nbio.IPV4_ADDRESS_BYTES]byte
	Called       bool
	Driver_Error error
}

func operating_system_allocation_callback(completion *time.Completion) {
	harness := (*operating_system_allocation_harness)(completion.Backend)
	harness.Called = true
}

// Test_Operating_System_Storage_API_Heap_Allocation guards every real storage boundary.
func Test_Operating_System_Storage_API_Heap_Allocation(t *testing.T) {
	tests := []struct {
		Name      string
		Operation storage_allocation_operation
	}{
		{Name: "Read", Operation: OPERATING_SYSTEM_STORAGE_ALLOCATION_READ},
		{Name: "Write", Operation: OPERATING_SYSTEM_STORAGE_ALLOCATION_WRITE},
		{Name: "Fsync", Operation: OPERATING_SYSTEM_STORAGE_ALLOCATION_FSYNC},
		{Name: "Open_At", Operation: OPERATING_SYSTEM_STORAGE_ALLOCATION_OPEN_AT},
		{Name: "Mkdir_At", Operation: OPERATING_SYSTEM_STORAGE_ALLOCATION_MKDIR_AT},
		{Name: "Directory", Operation: OPERATING_SYSTEM_STORAGE_ALLOCATION_DIRECTORY},
		{Name: "Status", Operation: OPERATING_SYSTEM_STORAGE_ALLOCATION_STATUS},
		{Name: "Close", Operation: OPERATING_SYSTEM_STORAGE_ALLOCATION_CLOSE},
		{Name: "Read_Empty", Operation: OPERATING_SYSTEM_STORAGE_ALLOCATION_READ_EMPTY},
		{Name: "Write_Empty", Operation: OPERATING_SYSTEM_STORAGE_ALLOCATION_WRITE_EMPTY},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			operating_system_storage_allocation_case(t, test.Operation)
		})
	}
}

// Rejected paths retire through same completion queue without error construction.
func Test_Operating_System_Path_Error_Heap_Allocation(t *testing.T) {
	tests := []struct {
		Name      string
		Operation path_error_allocation_operation
	}{
		{Name: "Open_At", Operation: PATH_ERROR_ALLOCATION_OPEN_AT},
		{Name: "Mkdir_At", Operation: PATH_ERROR_ALLOCATION_MKDIR_AT},
		{Name: "Status", Operation: PATH_ERROR_ALLOCATION_STATUS},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			path_error_allocation_case(t, test.Operation)
		})
	}
}

type path_error_allocation_operation uint8

// PATH_ERROR_ALLOCATION_OPEN_AT keeps operation selection outside measured closure.
const PATH_ERROR_ALLOCATION_OPEN_AT path_error_allocation_operation = 0

// PATH_ERROR_ALLOCATION_MKDIR_AT keeps operation selection outside measured closure.
const PATH_ERROR_ALLOCATION_MKDIR_AT path_error_allocation_operation = 1

// PATH_ERROR_ALLOCATION_STATUS keeps operation selection outside measured closure.
const PATH_ERROR_ALLOCATION_STATUS path_error_allocation_operation = 2

type path_error_allocation_harness struct {
	Loop         nbio.IO
	Driver       time.Driver
	Operation    path_error_allocation_operation
	Completion   time.Completion
	Error        error
	Driver_Error error
	Called       bool
}

func path_error_allocation_case(t *testing.T, operation path_error_allocation_operation) {
	t.Helper()
	clock := new_operating_system_clock()
	loop, _, driver := operating_system_loop(t, clock)
	harness := path_error_allocation_harness{
		Loop: loop, Driver: driver, Operation: operation,
	}
	harness.Completion.Backend = unsafe.Pointer(&harness)
	testify.Zero_Allocation(t, func() {
		path_error_allocation_run(&harness)
	})
	testify.No_Error(t, harness.Driver_Error)
	testify.Error(t, harness.Error)
	if operation != PATH_ERROR_ALLOCATION_STATUS {
		testify.True(t, harness.Called)
	}
	nbio.IO_Deinit(loop)
	time.Driver_Deinit(driver)
}

func path_error_allocation_run(harness *path_error_allocation_harness) {
	harness.Called = false
	if harness.Operation == PATH_ERROR_ALLOCATION_OPEN_AT {
		nbio.Storage_Open_At(
			harness.Loop.Storage,
			&harness.Completion,
			nbio.DIRECTORY_CURRENT,
			"\x00",
			nbio.Open_At_Options{Access: nbio.OPEN_READ_ONLY},
			path_error_allocation_callback,
		)
		harness.Driver_Error = time.Driver_Run(harness.Driver)
		return
	}
	if harness.Operation == PATH_ERROR_ALLOCATION_MKDIR_AT {
		nbio.Storage_Mkdir_At(
			harness.Loop.Storage,
			&harness.Completion,
			nbio.DIRECTORY_CURRENT,
			"\x00",
			0o700,
			path_error_allocation_callback,
		)
		harness.Driver_Error = time.Driver_Run(harness.Driver)
		return
	}
	_, harness.Error = nbio.Storage_Status(harness.Loop.Storage, "\x00")
}

func path_error_allocation_callback(completion *time.Completion) {
	harness := (*path_error_allocation_harness)(completion.Backend)
	harness.Error = completion.Error
	harness.Called = true
}

type storage_allocation_operation uint8

// OPERATING_SYSTEM_STORAGE_ALLOCATION_READ selects positioned read.
const OPERATING_SYSTEM_STORAGE_ALLOCATION_READ storage_allocation_operation = 0

// OPERATING_SYSTEM_STORAGE_ALLOCATION_WRITE selects positioned write.
const OPERATING_SYSTEM_STORAGE_ALLOCATION_WRITE storage_allocation_operation = 1

// OPERATING_SYSTEM_STORAGE_ALLOCATION_FSYNC selects file synchronization.
const OPERATING_SYSTEM_STORAGE_ALLOCATION_FSYNC storage_allocation_operation = 2

// OPERATING_SYSTEM_STORAGE_ALLOCATION_OPEN_AT selects descriptor creation.
const OPERATING_SYSTEM_STORAGE_ALLOCATION_OPEN_AT storage_allocation_operation = 3

// OPERATING_SYSTEM_STORAGE_ALLOCATION_MKDIR_AT selects directory creation.
const OPERATING_SYSTEM_STORAGE_ALLOCATION_MKDIR_AT storage_allocation_operation = 4

// OPERATING_SYSTEM_STORAGE_ALLOCATION_DIRECTORY selects one directory pass.
const OPERATING_SYSTEM_STORAGE_ALLOCATION_DIRECTORY storage_allocation_operation = 5

// OPERATING_SYSTEM_STORAGE_ALLOCATION_STATUS selects synchronous metadata read.
const OPERATING_SYSTEM_STORAGE_ALLOCATION_STATUS storage_allocation_operation = 6

// OPERATING_SYSTEM_STORAGE_ALLOCATION_CLOSE selects descriptor retirement.
const OPERATING_SYSTEM_STORAGE_ALLOCATION_CLOSE storage_allocation_operation = 7

// OPERATING_SYSTEM_STORAGE_ALLOCATION_READ_EMPTY selects immediate empty read retirement.
const OPERATING_SYSTEM_STORAGE_ALLOCATION_READ_EMPTY storage_allocation_operation = 8

// OPERATING_SYSTEM_STORAGE_ALLOCATION_WRITE_EMPTY selects immediate empty write retirement.
const OPERATING_SYSTEM_STORAGE_ALLOCATION_WRITE_EMPTY storage_allocation_operation = 9

type operating_system_storage_allocation_harness struct {
	Loop         nbio.IO
	Driver       time.Driver
	Operation    storage_allocation_operation
	Completion   time.Completion
	File         nbio.File
	Files        [OPERATING_SYSTEM_ALLOCATION_FILE_COUNT]nbio.File
	Paths        [OPERATING_SYSTEM_ALLOCATION_FILE_COUNT]string
	Buffer       [OPERATING_SYSTEM_ALLOCATION_BUFFER_BYTES]byte
	Entries      [OPERATING_SYSTEM_ALLOCATION_ENTRY_CAPACITY]nbio.Directory_Entry
	Run          int
	Driver_Error error
	Result_Error error
	Data         int
}

func operating_system_storage_allocation_case(
	t *testing.T, operation storage_allocation_operation,
) {
	t.Helper()
	clock := new_operating_system_clock()
	loop, _, driver := operating_system_loop(t, clock)
	root := t.TempDir()
	path := filepath.Join(root, "file")
	write_file(t, loop, driver, path, []byte("data"))
	harness := operating_system_storage_allocation_harness{
		Loop: loop, Driver: driver, Operation: operation, File: nbio.File(-1),
		Files: [OPERATING_SYSTEM_ALLOCATION_FILE_COUNT]nbio.File{
			nbio.File(-1), nbio.File(-1),
		}, Paths: [OPERATING_SYSTEM_ALLOCATION_FILE_COUNT]string{
			filepath.Join(root, "first"), filepath.Join(root, "second"),
		},
	}
	if operation == OPERATING_SYSTEM_STORAGE_ALLOCATION_OPEN_AT {
		harness.Paths = [OPERATING_SYSTEM_ALLOCATION_FILE_COUNT]string{path, path}
	}
	if operation == OPERATING_SYSTEM_STORAGE_ALLOCATION_STATUS {
		harness.Paths[0] = path
	}
	harness.Completion.Backend = unsafe.Pointer(&harness)
	operating_system_storage_allocation_prepare(t, &harness, path, root)
	testify.Zero_Allocation(t, func() {
		operating_system_storage_allocation_run(&harness)
	})
	testify.No_Error(t, harness.Driver_Error)
	if operation != OPERATING_SYSTEM_STORAGE_ALLOCATION_STATUS {
		testify.No_Error(t, harness.Result_Error)
	}
	if operation == OPERATING_SYSTEM_STORAGE_ALLOCATION_DIRECTORY {
		testify.True(t, harness.Data > 0)
		name_pointer := uintptr(unsafe.Pointer(unsafe.StringData(harness.Entries[0].Name)))
		buffer_start := uintptr(unsafe.Pointer(&harness.Buffer[0]))
		buffer_end := buffer_start + uintptr(len(harness.Buffer))
		testify.True(t, name_pointer >= buffer_start)
		testify.True(t, name_pointer < buffer_end)
	}
	operating_system_storage_allocation_finish(t, &harness, path)
}

func operating_system_storage_allocation_prepare(
	t *testing.T, harness *operating_system_storage_allocation_harness,
	path string, root string,
) {
	t.Helper()
	if harness.Operation == OPERATING_SYSTEM_STORAGE_ALLOCATION_MKDIR_AT {
		return
	}
	if harness.Operation == OPERATING_SYSTEM_STORAGE_ALLOCATION_STATUS {
		return
	}
	if harness.Operation == OPERATING_SYSTEM_STORAGE_ALLOCATION_DIRECTORY {
		for index := range harness.Files {
			harness.Files[index], _ = open_file(t, harness.Loop, harness.Driver, root)
		}
		return
	}
	if harness.Operation == OPERATING_SYSTEM_STORAGE_ALLOCATION_CLOSE {
		harness.Files[0], _ = open_file(t, harness.Loop, harness.Driver, path)
		harness.Files[1], _ = open_file(t, harness.Loop, harness.Driver, path)
		return
	}
	switch harness.Operation {
	case OPERATING_SYSTEM_STORAGE_ALLOCATION_WRITE,
		OPERATING_SYSTEM_STORAGE_ALLOCATION_WRITE_EMPTY:
		harness.File, _ = open_file_options(
			t, harness.Loop, harness.Driver, path,
			nbio.Open_At_Options{Access: nbio.OPEN_READ_WRITE},
		)
	default:
		harness.File, _ = open_file(t, harness.Loop, harness.Driver, path)
	}
}

func operating_system_storage_allocation_run(
	harness *operating_system_storage_allocation_harness,
) {
	harness.Completion.Backend = unsafe.Pointer(harness)
	harness.Result_Error = nil
	harness.Data = 0
	switch harness.Operation {
	case OPERATING_SYSTEM_STORAGE_ALLOCATION_READ:
		nbio.Storage_Read(
			harness.Loop.Storage, &harness.Completion, harness.File,
			harness.Buffer[:], 0,
			REAL_DEADLINE, operating_system_storage_allocation_callback,
		)
	case OPERATING_SYSTEM_STORAGE_ALLOCATION_WRITE:
		nbio.Storage_Write(
			harness.Loop.Storage, &harness.Completion, harness.File,
			harness.Buffer[:nbio.IPV4_ADDRESS_BYTES], 0,
			REAL_DEADLINE, operating_system_storage_allocation_callback,
		)
	case OPERATING_SYSTEM_STORAGE_ALLOCATION_FSYNC:
		nbio.Storage_Fsync(
			harness.Loop.Storage, &harness.Completion, harness.File, REAL_DEADLINE,
			operating_system_storage_allocation_callback,
		)
	case OPERATING_SYSTEM_STORAGE_ALLOCATION_OPEN_AT:
		nbio.Storage_Open_At(
			harness.Loop.Storage, &harness.Completion, nbio.DIRECTORY_CURRENT,
			harness.Paths[harness.Run],
			nbio.Open_At_Options{Access: nbio.OPEN_READ_ONLY},
			operating_system_storage_allocation_callback,
		)
	case OPERATING_SYSTEM_STORAGE_ALLOCATION_MKDIR_AT:
		nbio.Storage_Mkdir_At(
			harness.Loop.Storage, &harness.Completion, nbio.DIRECTORY_CURRENT,
			harness.Paths[harness.Run], 0o700,
			operating_system_storage_allocation_callback,
		)
	case OPERATING_SYSTEM_STORAGE_ALLOCATION_DIRECTORY:
		nbio.Storage_Get_Directory_Entries(
			harness.Loop.Storage, &harness.Completion, harness.Files[harness.Run],
			harness.Buffer[:],
			harness.Entries[:], operating_system_storage_allocation_callback,
		)
	case OPERATING_SYSTEM_STORAGE_ALLOCATION_STATUS:
		_, harness.Driver_Error = nbio.Storage_Status(
			harness.Loop.Storage, harness.Paths[0],
		)
	case OPERATING_SYSTEM_STORAGE_ALLOCATION_CLOSE:
		nbio.IO_Close(
			harness.Loop, &harness.Completion, harness.Files[harness.Run],
			operating_system_storage_allocation_callback,
		)
	case OPERATING_SYSTEM_STORAGE_ALLOCATION_READ_EMPTY:
		nbio.Storage_Read(
			harness.Loop.Storage, &harness.Completion, harness.File,
			harness.Buffer[:0], 0,
			REAL_DEADLINE, operating_system_storage_allocation_callback,
		)
	case OPERATING_SYSTEM_STORAGE_ALLOCATION_WRITE_EMPTY:
		nbio.Storage_Write(
			harness.Loop.Storage, &harness.Completion, harness.File,
			harness.Buffer[:0], 0,
			REAL_DEADLINE, operating_system_storage_allocation_callback,
		)
	}
	if harness.Operation != OPERATING_SYSTEM_STORAGE_ALLOCATION_STATUS {
		harness.Driver_Error = time.Driver_Run(harness.Driver)
	}
	harness.Run++
}

func operating_system_storage_allocation_callback(completion *time.Completion) {
	harness := (*operating_system_storage_allocation_harness)(completion.Backend)
	harness.Result_Error = completion.Error
	harness.Data = completion.Data
	if harness.Operation == OPERATING_SYSTEM_STORAGE_ALLOCATION_OPEN_AT {
		harness.Files[harness.Run] = nbio.File(completion.Data)
	}
}

func operating_system_storage_allocation_finish(
	t *testing.T, harness *operating_system_storage_allocation_harness, path string,
) {
	t.Helper()
	if harness.Operation == OPERATING_SYSTEM_STORAGE_ALLOCATION_OPEN_AT {
		close_file(t, harness.Loop, harness.Driver, path, harness.Files[0])
		close_file(t, harness.Loop, harness.Driver, path, harness.Files[1])
	}
	if harness.Operation == OPERATING_SYSTEM_STORAGE_ALLOCATION_DIRECTORY {
		close_file(t, harness.Loop, harness.Driver, path, harness.Files[0])
		close_file(t, harness.Loop, harness.Driver, path, harness.Files[1])
	}
	if harness.File >= 0 {
		close_file(t, harness.Loop, harness.Driver, path, harness.File)
	}
	nbio.IO_Deinit(harness.Loop)
	time.Driver_Deinit(harness.Driver)
}

// Test_Operating_System_Network_Synchronous_API_Heap_Allocation guards socket control paths.
func Test_Operating_System_Network_Synchronous_API_Heap_Allocation(t *testing.T) {
	tests := []struct {
		Name      string
		Operation network_allocation_operation
	}{
		{Name: "Socket_TCP", Operation: OPERATING_SYSTEM_NETWORK_ALLOCATION_SOCKET_TCP},
		{Name: "Socket_UDP", Operation: OPERATING_SYSTEM_NETWORK_ALLOCATION_SOCKET_UDP},
		{Name: "Bind", Operation: OPERATING_SYSTEM_NETWORK_ALLOCATION_BIND},
		{Name: "Listen_Socket", Operation: OPERATING_SYSTEM_NETWORK_ALLOCATION_LISTEN},
		{Name: "Get_Socket_Name", Operation: OPERATING_SYSTEM_NETWORK_ALLOCATION_NAME},
		{Name: "Shutdown", Operation: OPERATING_SYSTEM_NETWORK_ALLOCATION_SHUTDOWN},
		{Name: "Peer_Address", Operation: OPERATING_SYSTEM_NETWORK_ALLOCATION_PEER},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			operating_system_network_allocation_case(t, test.Operation)
		})
	}
}

// Test_Operating_System_Network_Asynchronous_API_Heap_Allocation guards kernel retirement.
func Test_Operating_System_Network_Asynchronous_API_Heap_Allocation(t *testing.T) {
	tests := []struct {
		Name      string
		Operation network_allocation_operation
	}{
		{Name: "Accept", Operation: OPERATING_SYSTEM_NETWORK_ALLOCATION_ACCEPT},
		{Name: "Connect", Operation: OPERATING_SYSTEM_NETWORK_ALLOCATION_CONNECT},
		{Name: "Receive", Operation: OPERATING_SYSTEM_NETWORK_ALLOCATION_RECEIVE},
		{Name: "Send", Operation: OPERATING_SYSTEM_NETWORK_ALLOCATION_SEND},
		{
			Name:      "Accept_Timeout",
			Operation: OPERATING_SYSTEM_NETWORK_ALLOCATION_ACCEPT_TIMEOUT,
		},
		{
			Name:      "Receive_Timeout",
			Operation: OPERATING_SYSTEM_NETWORK_ALLOCATION_RECEIVE_TIMEOUT,
		},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			operating_system_network_allocation_case(t, test.Operation)
		})
	}
}

type network_allocation_operation uint8

// OPERATING_SYSTEM_NETWORK_ALLOCATION_SOCKET_TCP selects TCP descriptor creation.
const OPERATING_SYSTEM_NETWORK_ALLOCATION_SOCKET_TCP network_allocation_operation = 0

// OPERATING_SYSTEM_NETWORK_ALLOCATION_SOCKET_UDP selects UDP descriptor creation.
const OPERATING_SYSTEM_NETWORK_ALLOCATION_SOCKET_UDP network_allocation_operation = 1

// OPERATING_SYSTEM_NETWORK_ALLOCATION_BIND selects local address ownership.
const OPERATING_SYSTEM_NETWORK_ALLOCATION_BIND network_allocation_operation = 2

// OPERATING_SYSTEM_NETWORK_ALLOCATION_LISTEN selects listener state.
const OPERATING_SYSTEM_NETWORK_ALLOCATION_LISTEN network_allocation_operation = 3

// OPERATING_SYSTEM_NETWORK_ALLOCATION_NAME selects local address readback.
const OPERATING_SYSTEM_NETWORK_ALLOCATION_NAME network_allocation_operation = 4

// OPERATING_SYSTEM_NETWORK_ALLOCATION_SHUTDOWN selects directional shutdown.
const OPERATING_SYSTEM_NETWORK_ALLOCATION_SHUTDOWN network_allocation_operation = 5

// OPERATING_SYSTEM_NETWORK_ALLOCATION_PEER selects remote address readback.
const OPERATING_SYSTEM_NETWORK_ALLOCATION_PEER network_allocation_operation = 6

// OPERATING_SYSTEM_NETWORK_ALLOCATION_ACCEPT selects inbound descriptor retirement.
const OPERATING_SYSTEM_NETWORK_ALLOCATION_ACCEPT network_allocation_operation = 7

// OPERATING_SYSTEM_NETWORK_ALLOCATION_CONNECT selects outbound handshake retirement.
const OPERATING_SYSTEM_NETWORK_ALLOCATION_CONNECT network_allocation_operation = 8

// OPERATING_SYSTEM_NETWORK_ALLOCATION_RECEIVE selects socket read retirement.
const OPERATING_SYSTEM_NETWORK_ALLOCATION_RECEIVE network_allocation_operation = 9

// OPERATING_SYSTEM_NETWORK_ALLOCATION_SEND selects socket write retirement.
const OPERATING_SYSTEM_NETWORK_ALLOCATION_SEND network_allocation_operation = 10

// OPERATING_SYSTEM_NETWORK_ALLOCATION_ACCEPT_TIMEOUT selects listener deadline retirement.
const OPERATING_SYSTEM_NETWORK_ALLOCATION_ACCEPT_TIMEOUT network_allocation_operation = 11

// OPERATING_SYSTEM_NETWORK_ALLOCATION_RECEIVE_TIMEOUT selects receive deadline retirement.
const OPERATING_SYSTEM_NETWORK_ALLOCATION_RECEIVE_TIMEOUT network_allocation_operation = 12

type operating_system_network_allocation_harness struct {
	Loop         nbio.IO
	Driver       time.Driver
	Operation    network_allocation_operation
	Files        [OPERATING_SYSTEM_ALLOCATION_NETWORK_FILES]nbio.File
	Count        int
	Run          int
	Address      nbio.Address
	Error        error
	Result_Error error
	Data         int
	Completion   time.Completion
	Buffer       [OPERATING_SYSTEM_ALLOCATION_NETWORK_BUFFER_BYTES]byte
	Called       bool
	Done         func() (finished bool)
}

func operating_system_network_allocation_case(
	t *testing.T, operation network_allocation_operation,
) {
	t.Helper()
	clock := new_operating_system_clock()
	loop, _, driver := operating_system_loop(t, clock)
	address := nbio.Address_IPV4([nbio.IPV4_ADDRESS_BYTES]byte{127, 0, 0, 1}, 0)
	harness := operating_system_network_allocation_harness{
		Loop: loop, Driver: driver, Operation: operation, Address: address,
	}
	harness.Completion.Backend = unsafe.Pointer(&harness)
	harness.Done = func() (finished bool) { return harness.Called }
	for index := range harness.Files {
		harness.Files[index] = nbio.File(-1)
	}
	operating_system_network_allocation_prepare(t, &harness)
	testify.Zero_Allocation(t, func() {
		operating_system_network_allocation_run(&harness)
	})
	testify.No_Error(t, harness.Error)
	operating_system_network_allocation_assert(t, &harness)
	for index := 0; index < harness.Count; index++ {
		self_exec_close(loop, driver, harness.Files[index])
	}
	nbio.IO_Deinit(loop)
	time.Driver_Deinit(driver)
}

func operating_system_network_allocation_assert(
	t *testing.T, harness *operating_system_network_allocation_harness,
) {
	t.Helper()
	if harness.Operation <= OPERATING_SYSTEM_NETWORK_ALLOCATION_PEER {
		return
	}
	testify.True(t, harness.Called)
	switch harness.Operation {
	case OPERATING_SYSTEM_NETWORK_ALLOCATION_ACCEPT_TIMEOUT,
		OPERATING_SYSTEM_NETWORK_ALLOCATION_RECEIVE_TIMEOUT:
		testify.Error_Is(t, harness.Result_Error, time.Deadline_Exceeded)
	default:
		testify.No_Error(t, harness.Result_Error)
	}
	if harness.Operation == OPERATING_SYSTEM_NETWORK_ALLOCATION_ACCEPT {
		testify.True(t, nbio.File(harness.Data) >= 0)
	}
	if harness.Operation == OPERATING_SYSTEM_NETWORK_ALLOCATION_RECEIVE {
		testify.Equal(t, nbio.IPV4_ADDRESS_BYTES, harness.Data)
	}
	if harness.Operation == OPERATING_SYSTEM_NETWORK_ALLOCATION_SEND {
		testify.Equal(t, nbio.IPV4_ADDRESS_BYTES, harness.Data)
	}
}

func operating_system_network_allocation_prepare(
	t *testing.T, harness *operating_system_network_allocation_harness,
) {
	t.Helper()
	if harness.Operation == OPERATING_SYSTEM_NETWORK_ALLOCATION_SOCKET_TCP {
		return
	}
	if harness.Operation == OPERATING_SYSTEM_NETWORK_ALLOCATION_SOCKET_UDP {
		return
	}
	if harness.Operation == OPERATING_SYSTEM_NETWORK_ALLOCATION_BIND {
		operating_system_network_allocation_open(t, harness, 2)
		return
	}
	if harness.Operation == OPERATING_SYSTEM_NETWORK_ALLOCATION_LISTEN {
		operating_system_network_allocation_open(t, harness, 2)
		for index := 0; index < harness.Count; index++ {
			testify.No_Error(t, nbio.Network_Bind(
				harness.Loop.Network, harness.Files[index], harness.Address,
			))
		}
		return
	}
	if harness.Operation == OPERATING_SYSTEM_NETWORK_ALLOCATION_NAME {
		operating_system_network_allocation_open(t, harness, 1)
		testify.No_Error(t, nbio.Network_Bind(
			harness.Loop.Network, harness.Files[0], harness.Address,
		))
		return
	}
	if harness.Operation >= OPERATING_SYSTEM_NETWORK_ALLOCATION_ACCEPT {
		operating_system_network_allocation_asynchronous_prepare(t, harness)
		return
	}
	operating_system_network_allocation_pairs(t, harness)
}

func operating_system_network_allocation_asynchronous_prepare(
	t *testing.T, harness *operating_system_network_allocation_harness,
) {
	t.Helper()
	port := free_port(t)
	listener, listen_err := test_listen(harness.Loop, harness.Driver, "127.0.0.1", port)
	if !testify.No_Error(t, listen_err) {
		return
	}
	harness.Files[harness.Count] = listener
	harness.Count++
	harness.Address = nbio.Address_IPV4(
		[nbio.IPV4_ADDRESS_BYTES]byte{127, 0, 0, 1}, uint16(port),
	)
	if harness.Operation == OPERATING_SYSTEM_NETWORK_ALLOCATION_ACCEPT {
		operating_system_network_allocation_queue_connections(t, harness)
		return
	}
	if harness.Operation == OPERATING_SYSTEM_NETWORK_ALLOCATION_ACCEPT_TIMEOUT {
		return
	}
	if harness.Operation == OPERATING_SYSTEM_NETWORK_ALLOCATION_CONNECT {
		operating_system_network_allocation_open(t, harness, 2)
		return
	}
	accepted, connected := loopback_pair(
		t, harness.Loop, harness.Driver, listener, port,
	)
	harness.Files[harness.Count] = accepted
	harness.Files[harness.Count+1] = connected
	harness.Count += 2
	if harness.Operation == OPERATING_SYSTEM_NETWORK_ALLOCATION_RECEIVE {
		operating_system_network_allocation_seed_receive(t, harness, connected)
	}
}

func operating_system_network_allocation_queue_connections(
	t *testing.T, harness *operating_system_network_allocation_harness,
) {
	t.Helper()
	for index := 0; index < 2; index++ {
		client, open_err := test_open_socket(harness.Loop)
		if !testify.No_Error(t, open_err) {
			return
		}
		harness.Files[harness.Count] = client
		harness.Count++
		connected := false
		var completion time.Completion
		nbio.Network_Connect(
			harness.Loop.Network, &completion, client, harness.Address, REAL_DEADLINE,
			func(completed *time.Completion) {
				testify.No_Error(t, completed.Error)
				connected = true
			},
		)
		operating_system_run_until(
			t, harness.Driver, func() (finished bool) { return connected },
		)
	}
}

func operating_system_network_allocation_seed_receive(
	t *testing.T, harness *operating_system_network_allocation_harness, connected nbio.File,
) {
	t.Helper()
	sent := false
	var completion time.Completion
	nbio.Network_Send(
		harness.Loop.Network, &completion, connected, []byte("12345678"), REAL_DEADLINE,
		func(completed *time.Completion) {
			testify.No_Error(t, completed.Error)
			sent = true
		},
	)
	operating_system_run_until(
		t, harness.Driver, func() (finished bool) { return sent },
	)
}

func operating_system_network_allocation_open(
	t *testing.T, harness *operating_system_network_allocation_harness, count int,
) {
	t.Helper()
	for index := 0; index < count; index++ {
		file, open_err := test_open_socket(harness.Loop)
		if !testify.No_Error(t, open_err) {
			return
		}
		harness.Files[harness.Count] = file
		harness.Count++
	}
}

func operating_system_network_allocation_pairs(
	t *testing.T, harness *operating_system_network_allocation_harness,
) {
	t.Helper()
	port := free_port(t)
	listener, listen_err := test_listen(harness.Loop, harness.Driver, "127.0.0.1", port)
	if !testify.No_Error(t, listen_err) {
		return
	}
	harness.Files[harness.Count] = listener
	harness.Count++
	for index := 0; index < 2; index++ {
		accepted, connected := loopback_pair(
			t, harness.Loop, harness.Driver, listener, port,
		)
		harness.Files[harness.Count] = accepted
		harness.Files[harness.Count+1] = connected
		harness.Count += 2
	}
}

func operating_system_network_allocation_run(
	harness *operating_system_network_allocation_harness,
) {
	if harness.Operation <= OPERATING_SYSTEM_NETWORK_ALLOCATION_PEER {
		operating_system_network_allocation_synchronous_run(harness)
	} else {
		operating_system_network_allocation_asynchronous_run(harness)
	}
	harness.Run++
}

func operating_system_network_allocation_synchronous_run(
	harness *operating_system_network_allocation_harness,
) {
	switch harness.Operation {
	case OPERATING_SYSTEM_NETWORK_ALLOCATION_SOCKET_TCP:
		file, err := nbio.Network_Socket_TCP(
			harness.Loop.Network, nbio.FAMILY_IPV4, test_tcp_options(),
		)
		harness.Files[harness.Count] = file
		harness.Count++
		harness.Error = err
	case OPERATING_SYSTEM_NETWORK_ALLOCATION_SOCKET_UDP:
		file, err := nbio.Network_Socket_UDP(
			harness.Loop.Network, nbio.FAMILY_IPV4, test_udp_options(),
		)
		harness.Files[harness.Count] = file
		harness.Count++
		harness.Error = err
	case OPERATING_SYSTEM_NETWORK_ALLOCATION_BIND:
		harness.Error = nbio.Network_Bind(
			harness.Loop.Network, harness.Files[harness.Run], harness.Address,
		)
	case OPERATING_SYSTEM_NETWORK_ALLOCATION_LISTEN:
		harness.Error = nbio.Network_Listen_Socket(
			harness.Loop.Network, harness.Files[harness.Run], 8,
		)
	case OPERATING_SYSTEM_NETWORK_ALLOCATION_NAME:
		_, harness.Error = nbio.Network_Get_Socket_Name(
			harness.Loop.Network, harness.Files[0],
		)
	case OPERATING_SYSTEM_NETWORK_ALLOCATION_SHUTDOWN:
		harness.Error = nbio.Network_Shutdown(
			harness.Loop.Network, harness.Files[1+harness.Run*2], nbio.SHUTDOWN_BOTH,
		)
	case OPERATING_SYSTEM_NETWORK_ALLOCATION_PEER:
		_, harness.Error = nbio.Network_Peer_Address(
			harness.Loop.Network, harness.Files[1],
		)
	}
}

func operating_system_network_allocation_asynchronous_run(
	harness *operating_system_network_allocation_harness,
) {
	harness.Result_Error = nil
	harness.Data = 0
	switch harness.Operation {
	case OPERATING_SYSTEM_NETWORK_ALLOCATION_ACCEPT:
		harness.Called = false
		nbio.Network_Accept(
			harness.Loop.Network, &harness.Completion, harness.Files[0], REAL_DEADLINE,
			operating_system_network_allocation_callback,
		)
		harness.Called, harness.Error = time.Driver_Run_Until(
			harness.Driver, REAL_DEADLINE, harness.Done,
		)
	case OPERATING_SYSTEM_NETWORK_ALLOCATION_CONNECT:
		harness.Called = false
		nbio.Network_Connect(
			harness.Loop.Network, &harness.Completion, harness.Files[1+harness.Run],
			harness.Address, REAL_DEADLINE,
			operating_system_network_allocation_callback,
		)
		harness.Called, harness.Error = time.Driver_Run_Until(
			harness.Driver, REAL_DEADLINE, harness.Done,
		)
	case OPERATING_SYSTEM_NETWORK_ALLOCATION_RECEIVE:
		harness.Called = false
		nbio.Network_Receive(
			harness.Loop.Network, &harness.Completion, harness.Files[1],
			harness.Buffer[:nbio.IPV4_ADDRESS_BYTES],
			REAL_DEADLINE, operating_system_network_allocation_callback,
		)
		harness.Called, harness.Error = time.Driver_Run_Until(
			harness.Driver, REAL_DEADLINE, harness.Done,
		)
	case OPERATING_SYSTEM_NETWORK_ALLOCATION_SEND:
		harness.Called = false
		nbio.Network_Send(
			harness.Loop.Network, &harness.Completion, harness.Files[2],
			harness.Buffer[:nbio.IPV4_ADDRESS_BYTES],
			REAL_DEADLINE, operating_system_network_allocation_callback,
		)
		harness.Called, harness.Error = time.Driver_Run_Until(
			harness.Driver, REAL_DEADLINE, harness.Done,
		)
	case OPERATING_SYSTEM_NETWORK_ALLOCATION_ACCEPT_TIMEOUT:
		harness.Called = false
		nbio.Network_Accept(
			harness.Loop.Network, &harness.Completion, harness.Files[0],
			REAL_OPERATION_DEADLINE, operating_system_network_allocation_callback,
		)
		harness.Called, harness.Error = time.Driver_Run_Until(
			harness.Driver, REAL_DEADLINE, harness.Done,
		)
	case OPERATING_SYSTEM_NETWORK_ALLOCATION_RECEIVE_TIMEOUT:
		harness.Called = false
		nbio.Network_Receive(
			harness.Loop.Network, &harness.Completion, harness.Files[1],
			harness.Buffer[:nbio.IPV4_ADDRESS_BYTES], REAL_OPERATION_DEADLINE,
			operating_system_network_allocation_callback,
		)
		harness.Called, harness.Error = time.Driver_Run_Until(
			harness.Driver, REAL_DEADLINE, harness.Done,
		)
	}
}

func operating_system_network_allocation_callback(completion *time.Completion) {
	harness := (*operating_system_network_allocation_harness)(completion.Backend)
	harness.Result_Error = completion.Error
	harness.Data = completion.Data
	if harness.Operation == OPERATING_SYSTEM_NETWORK_ALLOCATION_ACCEPT {
		harness.Files[harness.Count] = nbio.File(completion.Data)
		harness.Count++
	}
	harness.Called = true
}

// Tests need ambient state only to complete OS vtable owned by platform backend.
func operating_system_ambient() (system os.OS) {
	virtual := os.Virtual_OS{Process_Identifier: 1}
	return os.Virtual_OS_To_OS(&virtual)
}

// Drive real test predicate. Fail at once on backend scheduler error.
func operating_system_run_until(
	t *testing.T, driver time.Driver, done func() (finished bool),
) (completed bool) {
	t.Helper()
	completed, err := time.Driver_Run_Until(driver, REAL_DEADLINE, done)
	if !testify.No_Error(t, err) {
		return false
	}
	return completed
}

// Every entry rejects an absent bound before it can reach a platform scheduler.
func Test_Operating_System_IO_Network_Rejects_Disabled_Timeouts(t *testing.T) {
	clock := new_operating_system_clock()
	loop, _, driver := operating_system_loop(t, clock)
	for _, operation := range []string{"accept", "connect", "receive", "send"} {
		for _, timeout := range []time.Duration{0, -time.NANOSECOND} {
			operating_system_network_timeout_rejected(
				t, loop.Network, operation, timeout)
		}
	}
	time.Driver_Deinit(driver)
}

// Panic is synchronous because an invalid bound must not leave a kernel request to drain.
func operating_system_network_timeout_rejected(
	t *testing.T, network nbio.Network, operation string, timeout time.Duration,
) {
	t.Helper()
	testify.Panics(t, func() {
		var completion time.Completion
		if operation == "accept" {
			nbio.Network_Accept(network, &completion, nbio.File(-1), timeout,
				func(_ *time.Completion) {})
			return
		}
		if operation == "connect" {
			nbio.Network_Connect(
				network, &completion, nbio.File(-1), nbio.Address{}, timeout,
				func(_ *time.Completion) {})
			return
		}
		if operation == "receive" {
			nbio.Network_Receive(network, &completion, nbio.File(-1), nil, timeout,
				func(_ *time.Completion) {})
			return
		}
		nbio.Network_Send(network, &completion, nbio.File(-1), nil, timeout,
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
	nbio.IO_Deinit(loop)
	return false
}

// Open caller-owned IPv4 TCP socket with the profile backend tests verify.
func test_open_socket(loop nbio.IO) (socket nbio.File, err error) {
	return nbio.Network_Socket_TCP(loop.Network, nbio.FAMILY_IPV4, test_tcp_options())
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
	if bind_err := nbio.Network_Bind(loop.Network, listener, address); bind_err != nil {
		self_exec_close(loop, driver, listener)
		return nbio.File(-1), bind_err
	}
	if listen_err := nbio.Network_Listen_Socket(
		loop.Network, listener, 65535,
	); listen_err != nil {
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
	nbio.Network_Connect(loop.Network, completion, socket, address, REAL_DEADLINE, callback)
	return true
}

// Test_Operating_System_IO_Read write temp file and read it back through real backend. It
// confirm read run in loop and report bytes.
func Test_Operating_System_IO_Read(t *testing.T) {
	clock := new_operating_system_clock()
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
	nbio.Storage_Read(loop.Storage, &completion, file, buffer, 0, REAL_DEADLINE, func(
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

// Test_Operating_System_IO_Run_Until_Deadlock verify unbounded Run_Until with no operation
// pending fail loud, not block forever: predicate no event can flip is deadlock, thus pump panic
// instead of hang of caller.
func Test_Operating_System_IO_Run_Until_Deadlock(t *testing.T) {
	clock := new_operating_system_clock()
	_, _, driver := operating_system_loop(t, clock)
	testify.Panics(t, func() {
		time.Driver_Run_Until(driver, -1*time.NANOSECOND,
			func() (finished bool) { return false })
	})
}

// Test_Operating_System_IO_Timeout verify timeout fire once real time pass its deadline.
func Test_Operating_System_IO_Timeout(t *testing.T) {
	clock := new_operating_system_clock()
	_, pump, driver := operating_system_loop(t, clock)
	fired := false
	var completion time.Completion
	time.Timeline_Timeout(pump, &completion, time.MILLISECOND, func(_ *time.Completion) {
		fired = true
	})
	time.Driver_Run_Until(driver, REAL_DEADLINE, func() (finished bool) { return fired })
	testify.True(t, fired)
}

// Long-lived completion must release callback closure after real backend retire operation.
func Test_Operating_System_IO_Callback_Released(t *testing.T) {
	clock := new_operating_system_clock()
	_, pump, driver := operating_system_loop(t, clock)
	var completion time.Completion
	fired := false
	time.Timeline_Timeout(pump, &completion, time.MILLISECOND, func(_ *time.Completion) {
		fired = true
	})
	completed, drive_err := time.Driver_Run_Until(driver,
		REAL_DEADLINE, func() (finished bool) { return fired },
	)
	testify.No_Error(t, drive_err)
	testify.True(t, completed)
	testify.Nil(t, completion.Callback)
}

// Test_Operating_System_IO_Open_Socket_Profile verify outbound socket is non-blocking,
// close-on-exec, buffered for client workload, keepalive-enabled, and caller-owned in Raw_Open.
func Test_Operating_System_IO_Open_Socket_Profile(t *testing.T) {
	clock := new_operating_system_clock()
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
	nbio.IO_Deinit(loop)
}

// A public UDP constructor check prevents the typed transport split from existing only below the
// composition root.
func Test_Operating_System_IO_Open_UDP_Profile(t *testing.T) {
	clock := new_operating_system_clock()
	loop, _, driver := operating_system_loop(t, clock)
	socket, open_err := nbio.Network_Socket_UDP(
		loop.Network, nbio.FAMILY_IPV4, test_udp_options(),
	)
	if !testify.No_Error(t, open_err) {
		return
	}
	testify.True(t, descriptor_open(loop))
	receive_low_water, receive_low_water_err := syscall.GetsockoptInt(
		int(socket), syscall.SOL_SOCKET, syscall.SO_RCVLOWAT)
	testify.No_Error(t, receive_low_water_err)
	testify.Equal(t, int(test_udp_options().Receive_Low_Water_Bytes), receive_low_water)
	self_exec_close(loop, driver, socket)
	nbio.IO_Deinit(loop)
}

// Test_Operating_System_IO_Bind_Reuse_Address verify Bind enables address reuse before it gives
// the socket its local address.
func Test_Operating_System_IO_Bind_Reuse_Address(t *testing.T) {
	clock := new_operating_system_clock()
	loop, _, driver := operating_system_loop(t, clock)
	socket, open_err := nbio.Network_Socket_TCP(
		loop.Network, nbio.FAMILY_IPV4, test_tcp_options(),
	)
	if !testify.No_Error(t, open_err) {
		return
	}
	defer func() {
		self_exec_close(loop, driver, socket)
		nbio.IO_Deinit(loop)
	}()
	address := nbio.Address_IPV4([nbio.IPV4_ADDRESS_BYTES]byte{127, 0, 0, 1}, 0)
	testify.No_Error(t, nbio.Network_Bind(loop.Network, socket, address))
	reuse, reuse_err := syscall.GetsockoptInt(
		int(socket), syscall.SOL_SOCKET, syscall.SO_REUSEADDR)
	testify.No_Error(t, reuse_err)
	testify.Not_Zero(t, reuse)
}

// Test_Operating_System_IO_Reuse verify resubmit of completion still in flight panic: one
// Completion back one operation at a time, and real backend must fail as loud as sim, not
// silently double-arm it.
func Test_Operating_System_IO_Reuse(t *testing.T) {
	clock := new_operating_system_clock()
	_, pump, _ := operating_system_loop(t, clock)
	var completion time.Completion
	time.Timeline_Timeout(pump, &completion, time.SECOND, func(_ *time.Completion) {})
	testify.Panics(t, func() {
		time.Timeline_Timeout(pump, &completion, time.SECOND, func(_ *time.Completion) {})
	})
}

// Test_Operating_System_IO_Reentrancy verify drive of real loop from inside completion callback
// panic, thus re-entrant Run* fail loud, not corrupt it.
func Test_Operating_System_IO_Reentrancy(t *testing.T) {
	clock := new_operating_system_clock()
	_, pump, driver := operating_system_loop(t, clock)
	var completion time.Completion
	time.Timeline_Timeout(pump, &completion, time.MILLISECOND, func(_ *time.Completion) {
		time.Driver_Run(driver)
	})
	testify.Panics(t, func() {
		time.Driver_Run_For(driver, 50*time.MILLISECOND)
	})
}

// Test_Operating_System_IO_Socket run TCP loopback round-trip through real backend: client
// connect to listener, send bytes, and accepted server socket receive them — all driven by
// single event loop.
func Test_Operating_System_IO_Socket(t *testing.T) {
	port := free_port(t)
	clock := new_operating_system_clock()
	loop, _, driver := operating_system_loop(t, clock)

	listener, listen_err := test_listen(loop, driver, "127.0.0.1", port)
	if !testify.No_Error(t, listen_err) {
		return
	}

	accepted := nbio.File(-1)
	var accept_completion time.Completion
	nbio.Network_Accept(loop.Network, &accept_completion, listener, REAL_DEADLINE, func(
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

	time.Driver_Run_Until(driver, REAL_DEADLINE,
		func() (finished bool) { return accepted > 0 })
	time.Driver_Run_Until(driver, REAL_DEADLINE,
		func() (finished bool) { return connect_done })
	testify.Positive(t, accepted)
	testify.Positive(t, connected)

	loopback_assert_roundtrip(&loopback_roundtrip_input{
		Test: t, Timeline: loop, Driver: driver, Connected: connected, Accepted: accepted,
	})
	self_exec_close(loop, driver, connected)
	self_exec_close(loop, driver, accepted)
	self_exec_close(loop, driver, listener)
	nbio.IO_Deinit(loop)
}

// Test_Operating_System_IO_Accept_Deadline prove listener with no inbound connection retire its
// accept exactly once, after which listener and backend release safe.
func Test_Operating_System_IO_Accept_Deadline(t *testing.T) {
	clock := new_operating_system_clock()
	loop, _, driver := operating_system_loop(t, clock)
	listener, listen_err := test_listen(loop, driver, "127.0.0.1", 0)
	if !testify.No_Error(t, listen_err) {
		return
	}
	callback_count := 0
	accepted := nbio.File(-1)
	var operation_err error
	var completion time.Completion
	nbio.Network_Accept(
		loop.Network, &completion, listener, REAL_OPERATION_DEADLINE, func(
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
	time.Driver_Deinit(driver)
}

// A receive timeout must retire the kernel registration without taking socket ownership.
func Test_Operating_System_IO_Receive_Timeout_Preserves_Socket(t *testing.T) {
	port := free_port(t)
	clock := new_operating_system_clock()
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
	nbio.Network_Receive(
		loop.Network, &completion, accepted, make([]byte, 8), REAL_OPERATION_DEADLINE, func(
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
	time.Driver_Deinit(driver)
}

// Test_Operating_System_IO_Connect_Error_Preserves_Socket verify refusal leave caller-owned
// descriptor open until caller explicitly close it.
func Test_Operating_System_IO_Connect_Error_Preserves_Socket(t *testing.T) {
	port := free_port(t)
	clock := new_operating_system_clock()
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

	time.Driver_Run_Until(driver, REAL_DEADLINE, func() (finished bool) { return called })
	testify.Error_Is(t, connect_err, nbio.Connection_Refused)
	testify.True(t, descriptor_open(loop))
	self_exec_close(loop, driver, socket)
	nbio.IO_Deinit(loop)
}

// Test_Operating_System_IO_Send_In_Connect_Completion arm send from inside connect completion —
// send share write-waiter slot of connected descriptor with connect it is armed within. Loop must
// retire connect waiter before it deliver its callback, else send is deleted instant it is armed
// and never fire (ClickHouse-daemon bug).
func Test_Operating_System_IO_Send_In_Connect_Completion(t *testing.T) {
	port := free_port(t)
	clock := new_operating_system_clock()
	loop, _, driver := operating_system_loop(t, clock)

	listener, listen_err := test_listen(loop, driver, "127.0.0.1", port)
	if !testify.No_Error(t, listen_err) {
		return
	}
	accepted := nbio.File(-1)
	var accept_completion time.Completion
	nbio.Network_Accept(loop.Network, &accept_completion, listener, REAL_DEADLINE, func(
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
		nbio.Network_Send(
			loop.Network, &send_completion, socket, []byte("ping"), REAL_DEADLINE, func(
				completed *time.Completion,
			) {
				testify.No_Error(t, completed.Error)
				sent = completed.Data
			})
	}) {
		return
	}

	time.Driver_Run_Until(driver, REAL_DEADLINE, func() (finished bool) { return sent >= 0 })
	testify.Equal(t, 4, sent)

	time.Driver_Run_Until(driver, REAL_DEADLINE,
		func() (finished bool) { return accepted > 0 })
	buffer := make([]byte, 16)
	received := -1
	var receive_completion time.Completion
	nbio.Network_Receive(
		loop.Network, &receive_completion, accepted, buffer, REAL_DEADLINE, func(
			completed *time.Completion,
		) {
			testify.No_Error(t, completed.Error)
			received = completed.Data
		})
	time.Driver_Run_Until(driver, REAL_DEADLINE,
		func() (finished bool) { return received >= 0 })
	testify.Equal(t, 4, received)
	testify.Equal(t, "ping", string(buffer[:4]))
}

// Test_Operating_System_IO_Drain_Then_Recycle verify Shutdown resolve armed receive before Close,
// after which later connection still receive readiness normally.
func Test_Operating_System_IO_Drain_Then_Recycle(t *testing.T) {
	port := free_port(t)
	clock := new_operating_system_clock()
	loop, _, driver := operating_system_loop(t, clock)
	listener, listen_err := test_listen(loop, driver, "127.0.0.1", port)
	if !testify.No_Error(t, listen_err) {
		return
	}

	first, first_connected := loopback_pair(t, loop, driver, listener, port)
	buffer := make([]byte, 16)
	var receive_completion time.Completion
	fired := 0
	nbio.Network_Receive(loop.Network, &receive_completion, first, buffer, REAL_DEADLINE,
		func(_ *time.Completion) { fired++ })
	testify.No_Error(t, nbio.Network_Shutdown(loop.Network, first, nbio.SHUTDOWN_RECEIVE))
	testify.True(t,
		operating_system_run_until(t, driver, func() (finished bool) { return fired > 0 }))
	closed := false
	var close_completion time.Completion
	nbio.IO_Close(loop, &close_completion, first, func(_ *time.Completion) { closed = true })
	testify.True(t,
		operating_system_run_until(t, driver, func() (finished bool) { return closed }))
	self_exec_close(loop, driver, first_connected)

	// Later socket may reuse descriptor, and must still deliver readiness.
	recycled, second := loopback_pair(t, loop, driver, listener, port)
	var send_completion time.Completion
	nbio.Network_Send(
		loop.Network, &send_completion, second, []byte("pong"), REAL_DEADLINE, func(
			completed *time.Completion,
		) {
			testify.No_Error(t, completed.Error)
		})
	received := -1
	var second_receive time.Completion
	nbio.Network_Receive(
		loop.Network, &second_receive, recycled, buffer, REAL_DEADLINE, func(
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
	nbio.Network_Accept(loop.Network, &accept_completion, listener, REAL_DEADLINE, func(
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
	time.Driver_Run_Until(driver, REAL_DEADLINE,
		func() (finished bool) { return accepted > 0 && connect_done })
	testify.Positive(t, accepted)
	return accepted, connected
}

// Test_Operating_System_IO_Close_With_Armed_Receive verify Close reject descriptor still borrowed
// by submitted receive. Owner must shutdown, drain receive callback, and only then close.
func Test_Operating_System_IO_Close_With_Armed_Receive(t *testing.T) {
	port := free_port(t)
	clock := new_operating_system_clock()
	loop, _, driver := operating_system_loop(t, clock)
	listener, listen_err := test_listen(loop, driver, "127.0.0.1", port)
	if !testify.No_Error(t, listen_err) {
		return
	}
	accepted, connected := loopback_pair(t, loop, driver, listener, port)

	received := false
	var receive_completion time.Completion
	nbio.Network_Receive(
		loop.Network, &receive_completion, accepted, make([]byte, 8), REAL_DEADLINE, func(
			_ *time.Completion,
		) {
			received = true
		})

	var close_completion time.Completion
	testify.Panics(t, func() {
		nbio.IO_Close(loop, &close_completion, accepted, func(_ *time.Completion) {})
	})
	testify.No_Error(t,
		nbio.Network_Shutdown(loop.Network, accepted, nbio.SHUTDOWN_BOTH))
	testify.True(t,
		operating_system_run_until(t, driver, func() (finished bool) { return received }))

	self_exec_close(loop, driver, accepted)
	self_exec_close(loop, driver, connected)
	self_exec_close(loop, driver, listener)
}

// Test_Operating_System_IO_Open open file through loop and read it back.
func Test_Operating_System_IO_Open(t *testing.T) {
	clock := new_operating_system_clock()
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
	nbio.Storage_Read(loop.Storage, &completion, file, buffer, 0, REAL_DEADLINE, func(
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
	clock := new_operating_system_clock()
	loop, _, driver := operating_system_loop(t, clock)
	path := filepath.Join(t.TempDir(), "create")
	file, create_err := create_file(t, loop, driver, path)
	if !testify.No_Error(t, create_err) {
		return
	}
	count := -1
	write_done := false
	var completion time.Completion
	nbio.Storage_Write(
		loop.Storage, &completion, file, []byte("world"), 0, REAL_DEADLINE, func(
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
	clock := new_operating_system_clock()
	loop, _, driver := operating_system_loop(t, clock)
	opened := nbio.File(-1)
	var open_completion time.Completion
	nbio.Storage_Open_At(
		loop.Storage, &open_completion, nbio.DIRECTORY_CURRENT, path, nbio.Open_At_Options{
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
	nbio.Storage_Write(
		loop.Storage, &write_completion, opened, []byte("hello"), 10, REAL_DEADLINE, func(
			completed *time.Completion,
		) {
			testify.No_Error(t, completed.Error)
			written = completed.Data == 5
		})
	testify.True(t,
		operating_system_run_until(t, driver, func() (finished bool) { return written }))

	synced := false
	var fsync_completion time.Completion
	nbio.Storage_Fsync(
		loop.Storage, &fsync_completion, opened, REAL_DEADLINE,
		func(completed *time.Completion) {
			testify.No_Error(t, completed.Error)
			synced = true
		})
	testify.True(t,
		operating_system_run_until(t, driver, func() (finished bool) { return synced }))

	buffer := make([]byte, 5)
	read := false
	var read_completion time.Completion
	nbio.Storage_Read(loop.Storage, &read_completion, opened, buffer, 10, REAL_DEADLINE, func(
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
	clock_setup := new_operating_system_clock()
	loop_setup, _, driver_setup := operating_system_loop(t, clock_setup)
	root := t.TempDir()
	target := filepath.Join(root, "target")
	link := filepath.Join(root, "link")
	write_file(t, loop_setup, driver_setup, target, []byte("secret"))
	time.Driver_Deinit(driver_setup)
	// Symbolic link is one filesystem shape io.IO cannot make, thus link itself stay raw call.
	// That absence is what this test exist to guard against follow of.
	if !testify.No_Error(t, syscall.Symlink(target, link)) {
		return
	}

	clock := new_operating_system_clock()
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
	nbio.Storage_Open_At(
		loop.Storage, &completion, nbio.DIRECTORY_CURRENT, path, nbio.Open_At_Options{
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
	clock := new_operating_system_clock()
	_, pump, driver := operating_system_loop(t, clock)
	event, open_err := time.Timeline_Open_Event(pump)
	if !testify.No_Error(t, open_err) {
		return
	}
	fired := 0
	var completion time.Completion
	callback := func(_ *time.Completion) { fired++ }
	time.Timeline_Event_Listen(pump, event, &completion, callback)
	time.Timeline_Event_Trigger(pump, event, &completion)
	testify.True(t,
		operating_system_run_until(t, driver, func() (finished bool) { return fired == 1 }))
	time.Timeline_Event_Listen(pump, event, &completion, callback)
	time.Timeline_Event_Trigger(pump, event, &completion)
	testify.True(t,
		operating_system_run_until(t, driver, func() (finished bool) { return fired == 2 }))
	time.Timeline_Close_Event(pump, event)
}

// Trigger thread must never read completion metadata while loop rearm writes same storage.
func Test_Operating_System_IO_Event_Concurrent_Rearm(t *testing.T) {
	clock := new_operating_system_clock()
	_, pump, driver := operating_system_loop(t, clock)
	event, open_err := time.Timeline_Open_Event(pump)
	if !testify.No_Error(t, open_err) {
		return
	}
	const FIRES = 64
	fired := &atomic.Int64{}
	stopped := &atomic.Bool{}
	workers := &sync.WaitGroup{}
	var completion time.Completion
	var callback time.Callback
	callback = func(_ *time.Completion) {
		if fired.Add(1) < FIRES {
			time.Timeline_Event_Listen(pump, event, &completion, callback)
		}
	}
	time.Timeline_Event_Listen(pump, event, &completion, callback)
	workers.Add(1)
	go func() {
		defer workers.Done()
		for !stopped.Load() {
			time.Timeline_Event_Trigger(pump, event, &completion)
		}
	}()
	completed, drive_err := time.Driver_Run_Until(driver,
		REAL_DEADLINE, func() (finished bool) { return fired.Load() == FIRES },
	)
	stopped.Store(true)
	workers.Wait()
	testify.No_Error(t, drive_err)
	testify.True(t, completed)
	time.Timeline_Close_Event(pump, event)
}

// Test_Operating_System_IO_Peer_Address report remote address of accepted loopback connection.
func Test_Operating_System_IO_Peer_Address(t *testing.T) {
	port := free_port(t)
	clock := new_operating_system_clock()
	loop, _, driver := operating_system_loop(t, clock)
	listener, listen_err := test_listen(loop, driver, "127.0.0.1", port)
	if !testify.No_Error(t, listen_err) {
		return
	}

	accepted := nbio.File(-1)
	var accept_completion time.Completion
	nbio.Network_Accept(loop.Network, &accept_completion, listener, REAL_DEADLINE, func(
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
	time.Driver_Run_Until(driver, REAL_DEADLINE,
		func() (finished bool) { return accepted > 0 })

	testify.Positive(t, accepted)
	address, address_err := nbio.Network_Peer_Address(loop.Network, accepted)
	testify.No_Error(t, address_err)
	testify.Equal(t, nbio.FAMILY_IPV4, address.Family)
	testify.Equal(t, [nbio.IPV6_ADDRESS_BYTES]byte{127, 0, 0, 1}, address.IP)
}

// Test_Operating_System_IO_Deinit_Rejects_Undrained_Extension verify Deinit cannot close
// Event/backend while Spawn still own repository-extension completion. Spawn is vehicle because
// it retire through same off-loop post path Event bridge.
func Test_Operating_System_IO_Deinit_Rejects_Undrained_Extension(t *testing.T) {
	clock := new_operating_system_clock()
	_, _, driver, system := operating_system_all(t, clock)
	drained := false
	var completion time.Completion
	os.OS_Spawn(system, &completion, os.Process_Request{Path: "true"}, REAL_DEADLINE, func(
		_ *time.Completion, _ os.Process_Result, _ error,
	) {
		drained = true
	})
	testify.Panics(t, func() {
		time.Driver_Deinit(driver)
	})
	testify.True(t,
		operating_system_run_until(t, driver, func() (finished bool) { return drained }))
	time.Driver_Deinit(driver)
}

// Test_Operating_System_IO_Watch_Signal deliver real SIGTERM onto loop.
func Test_Operating_System_IO_Watch_Signal(t *testing.T) {
	clock := new_operating_system_clock()
	_, _, driver, system := operating_system_all(t, clock)
	got := os.SIGNAL_EXPIRED
	fired := 0
	var completion time.Completion
	os.OS_Watch_Signal(system, &completion, os.SIGNAL_TERMINATE, REAL_DEADLINE, func(
		_ *time.Completion, signal os.Signal, err error,
	) {
		testify.No_Error(t, err)
		fired++
		got = signal
	})
	testify.No_Error(t, syscall.Kill(syscall.Getpid(), syscall.SIGTERM))
	time.Driver_Run_Until(driver, REAL_DEADLINE, func() (finished bool) { return fired > 0 })

	testify.Equal(t, 1, fired)
	testify.Equal(t, os.SIGNAL_TERMINATE, got)
}

// Test_Operating_System_IO_Watch_Signal_Deadline prove signal that never arrive retire extension
// completion exactly once, and permit backend deinitialization.
func Test_Operating_System_IO_Watch_Signal_Deadline(t *testing.T) {
	clock := new_operating_system_clock()
	_, _, driver, system := operating_system_all(t, clock)
	callback_count := 0
	got := os.SIGNAL_EXPIRED
	var operation_err error
	var completion time.Completion
	os.OS_Watch_Signal(
		system,
		&completion, os.SIGNAL_TERMINATE, REAL_OPERATION_DEADLINE, func(
			_ *time.Completion, signal os.Signal, err error,
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
	testify.Equal(t, os.SIGNAL_EXPIRED, got)
	time.Driver_Deinit(driver)
}

// Test_Operating_System_IO_Spawn run real commands through loop: success with captured output,
// and non-zero exit reported without start error.
func Test_Operating_System_IO_Spawn(t *testing.T) {
	clock := new_operating_system_clock()
	_, _, driver, system := operating_system_all(t, clock)

	echo := os.Process_Result{}
	echoed := false
	var echo_completion time.Completion
	os.OS_Spawn(
		system,
		&echo_completion,
		os.Process_Request{Path: "/bin/echo", Arguments: []string{"hi"}},
		REAL_DEADLINE,
		func(
			_ *time.Completion, result os.Process_Result, err error,
		) {
			testify.No_Error(t, err)
			echo = result
			echoed = true
		})
	operating_system_run_until(t, driver, func() (finished bool) { return echoed })

	testify.True(t, echoed)
	testify.Zero(t, echo.Exit)
	testify.Equal(t, "hi\n", string(echo.Output))

	fail := os.Process_Result{}
	failed := false
	var fail_completion time.Completion
	os.OS_Spawn(system, &fail_completion, os.Process_Request{
		Path: "/bin/sh", Arguments: []string{"-c", "exit 1"},
	}, REAL_DEADLINE, func(
		_ *time.Completion, result os.Process_Result, err error,
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
	clock := new_operating_system_clock()
	_, _, driver, system := operating_system_all(t, clock)

	streamed := nbio.Stream_Memory{Memory: make([]byte, 64)}
	result := os.Process_Result{}
	done := false
	var completion time.Completion
	os.OS_Spawn(system, &completion, os.Process_Request{
		Path: "/bin/echo", Arguments: []string{"hi"},
		Stdout: nbio.Memory_To_Stream(&streamed),
	}, REAL_DEADLINE, func(
		_ *time.Completion, spawned os.Process_Result, err error,
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
	nbio.Storage_Read(
		loop.Storage, &read_completion, file, process_buffer, 0, REAL_DEADLINE, func(
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
	nbio.IO_Close(loop, &close_completion, file, func(completed *time.Completion) {
		testify.No_Error(t, completed.Error)
		close_done = true
	})
	testify.True(t,
		operating_system_run_until(t, driver, func() (finished bool) { return close_done }))
	identifier, parsed := test_decimal(process_buffer[:process_count])
	testify.True(t, parsed)
	return identifier
}

// Test_Operating_System_IO_Spawn_Deadline prove timeout kill whole subprocess group, keep output
// captured before expiry, and deliver one terminal callback.
func Test_Operating_System_IO_Spawn_Deadline(t *testing.T) {
	clock := new_operating_system_clock()
	loop, _, driver, system := operating_system_all(t, clock)
	process_path := filepath.Join(t.TempDir(), "process")
	request := os.Process_Request{
		Path: "/bin/sh",
		Arguments: []string{
			"-c", "printf '%d' $$ > \"$1\"; printf partial; sleep 30 & wait",
			"bounded-spawn", process_path,
		},
	}
	callback_count := 0
	result := os.Process_Result{}
	var operation_err error
	var completion time.Completion
	os.OS_Spawn(system, &completion, request, 100*time.MILLISECOND, func(
		_ *time.Completion, spawned os.Process_Result, err error,
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
	time.Driver_Run_For(driver, 2*REAL_OPERATION_DEADLINE)
	testify.Equal(t, 1, callback_count)
	time.Driver_Deinit(driver)
}

// Test_Operating_System_IO_Spawn_Reaps_After_Deadline verify deadline path still reap.
// Completion retire early on Deadline_Exceeded, thus exit event arrive for child nobody wait on,
// and only own tracking of backend keep reap on that path.
func Test_Operating_System_IO_Spawn_Reaps_After_Deadline(t *testing.T) {
	clock := new_operating_system_clock()
	_, _, driver, system := operating_system_all(t, clock)
	callback_count := 0
	var operation_err error
	var completion time.Completion
	os.OS_Spawn(system, &completion, os.Process_Request{
		Path: "/bin/sleep", Arguments: []string{"30"},
	}, 50*time.MILLISECOND, func(
		_ *time.Completion, _ os.Process_Result, err error,
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
	time.Driver_Deinit(driver)
}

// Test_Operating_System_IO_Spawn_Bounds_Lingering_Drain verify pipe-cleanup bound start when
// child exit, not when its deadline expire. Child that finish at once but leave grandchild
// holding standard output must retire about one second later, not wait out whole deadline. This
// is bound exec.Cmd.WaitDelay supplied before.
func Test_Operating_System_IO_Spawn_Bounds_Lingering_Drain(t *testing.T) {
	clock := new_operating_system_clock()
	_, _, driver, system := operating_system_all(t, clock)
	const SPAWN_DEADLINE = 4 * time.SECOND
	started := time.Clock_Now_Monotonic(clock)
	callback_count := 0
	result := os.Process_Result{}
	var operation_err error
	var completion time.Completion
	os.OS_Spawn(system, &completion, os.Process_Request{
		Path:      "/bin/sh",
		Arguments: []string{"-c", "printf quick; sleep 30 & exit 0"},
	}, SPAWN_DEADLINE, func(
		_ *time.Completion, spawned os.Process_Result, err error,
	) {
		callback_count++
		result = spawned
		operation_err = err
	})
	testify.True(t, operating_system_run_until(
		t, driver, func() (finished bool) { return callback_count > 0 },
	))
	elapsed := time.Duration(int64(time.Clock_Now_Monotonic(clock)) - int64(started))
	testify.True(t, elapsed < SPAWN_DEADLINE, elapsed)
	// Child exited clean, thus its code survive. Output is incomplete because drain was cut
	// short, and Deadline_Exceeded is how caller learn that.
	testify.Zero(t, result.Exit)
	testify.Equal(t, "quick", string(result.Output))
	testify.Error_Is(t, operation_err, time.Deadline_Exceeded)
	time.Driver_Deinit(driver)
}

// Test_Operating_System_IO_Spawn_Concurrent verify two children in flight at once keep their
// pipes separate. Pipe end that leak into fork of other child would hold standard input of that
// child open, thus this fail by deadlock, not by wrong result.
func Test_Operating_System_IO_Spawn_Concurrent(t *testing.T) {
	clock := new_operating_system_clock()
	_, _, driver, system := operating_system_all(t, clock)
	const SPAWNS_COUNT = 4
	finished := 0
	outputs := make([]string, SPAWNS_COUNT)
	completions := make([]*time.Completion, SPAWNS_COUNT)
	for index := 0; index < SPAWNS_COUNT; index++ {
		position := index
		completions[position] = &time.Completion{}
		os.OS_Spawn(system, completions[position], os.Process_Request{
			Path:  "/bin/cat",
			Input: []byte(test_decimal_text(position)),
		}, REAL_DEADLINE, func(
			_ *time.Completion, spawned os.Process_Result, err error,
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
		want := test_decimal_text(index)
		testify.Equal(t, want, outputs[index], index)
	}
	time.Driver_Deinit(driver)
}

// Test_Operating_System_IO_Spawn_Feeds_Input verify Process_Request.Input reach child standard
// input, and write end close, thus child observe end-of-file, not wait for more.
func Test_Operating_System_IO_Spawn_Feeds_Input(t *testing.T) {
	clock := new_operating_system_clock()
	_, _, driver, system := operating_system_all(t, clock)
	callback_count := 0
	result := os.Process_Result{}
	var operation_err error
	var completion time.Completion
	os.OS_Spawn(system, &completion, os.Process_Request{
		Path: "/bin/cat", Input: []byte("fed through stdin"),
	}, REAL_DEADLINE, func(
		_ *time.Completion, spawned os.Process_Result, err error,
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
	time.Driver_Deinit(driver)
}

// Test_Operating_System_IO_Spawn_Drains_Full_Pipe verify output larger than one pipe buffer
// still complete. Child that fill pipe block until loop read it, thus this fail by deadlock when
// reads are not armed for whole life of child.
func Test_Operating_System_IO_Spawn_Drains_Full_Pipe(t *testing.T) {
	clock := new_operating_system_clock()
	_, _, driver, system := operating_system_all(t, clock)
	const LINES = 20000
	callback_count := 0
	result := os.Process_Result{}
	var operation_err error
	var completion time.Completion
	os.OS_Spawn(system, &completion, os.Process_Request{
		Path: "/bin/sh",
		Arguments: []string{
			"-c", "i=0; while [ $i -lt 20000 ]; do echo line; i=$((i+1)); done",
		},
	}, REAL_DEADLINE, func(
		_ *time.Completion, spawned os.Process_Result, err error,
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
	time.Driver_Deinit(driver)
}

// Test_Operating_System_IO_Spawn_Resolves_Path verify bare command name still resolve through
// PATH. exec.Command supplied this before, and syscall.StartProcess does not.
func Test_Operating_System_IO_Spawn_Resolves_Path(t *testing.T) {
	clock := new_operating_system_clock()
	_, _, driver, system := operating_system_all(t, clock)
	callback_count := 0
	result := os.Process_Result{}
	var operation_err error
	var completion time.Completion
	os.OS_Spawn(
		system,
		&completion,
		os.Process_Request{Path: "echo", Arguments: []string{"resolved"}},
		REAL_DEADLINE,
		func(
			_ *time.Completion, spawned os.Process_Result, err error,
		) {
			callback_count++
			result = spawned
			operation_err = err
		})
	testify.True(t, operating_system_run_until(
		t, driver, func() (finished bool) { return callback_count > 0 },
	))
	testify.No_Error(t, operation_err)
	testify.Equal(t, "resolved", test_trimmed_text(result.Output))
	time.Driver_Deinit(driver)
}

// Test_Operating_System_IO_Spawn_Reports_Missing_Command verify unresolvable command name fail
// spawn, not start anything.
func Test_Operating_System_IO_Spawn_Reports_Missing_Command(t *testing.T) {
	clock := new_operating_system_clock()
	_, _, driver, system := operating_system_all(t, clock)
	callback_count := 0
	var operation_err error
	var completion time.Completion
	os.OS_Spawn(system, &completion, os.Process_Request{Path: "no-such-command-anywhere"},
		REAL_DEADLINE, func(
			_ *time.Completion, _ os.Process_Result, err error,
		) {
			callback_count++
			operation_err = err
		})
	testify.True(t, operating_system_run_until(
		t, driver, func() (finished bool) { return callback_count > 0 },
	))
	testify.Error(t, operation_err)
	time.Driver_Deinit(driver)
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
		drive_err := time.Driver_Run_For(driver, REAL_OPERATION_DEADLINE)
		if drive_err != nil {
			return false, drive_err
		}
	}
	return false, nil
}

// Test_Operating_System_IO_Make_Directory cover what converging mkdir can hide: repeat call,
// relative path, trailing slash, and final component that already exist as file.
func Test_Operating_System_IO_Make_Directory(t *testing.T) {
	clock := new_operating_system_clock()
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
		status, _ := nbio.Storage_Status(loop.Storage, path)
		testify.True(t, status.Is_Directory, path)
	}

	slashed := filepath.Join(root, "four", "five") + "/"
	testify.No_Error(t, make_directory(t, loop, driver, slashed))
	slashed_status, _ := nbio.Storage_Status(
		loop.Storage, filepath.Join(root, "four", "five"),
	)
	testify.True(t, slashed_status.Is_Directory)

	// Final component that already exist as file must report error, not converge, because
	// caller asked for directory and does not have one.
	occupied := filepath.Join(root, "occupied")
	write_file(t, loop, driver, occupied, []byte("not a directory"))
	// Mkdir_At report Path_Exists for file too, and Make_Directory converge on that. Caller
	// thus learn difference from Status, not from create.
	testify.No_Error(t, make_directory(t, loop, driver, occupied))
	status, _ := nbio.Storage_Status(loop.Storage, occupied)
	testify.False(t, status.Is_Directory)
	time.Driver_Deinit(driver)
}

// Test_Operating_System_IO_Directory exercise filesystem-traversal operations on real temp tree:
// Make_Directory build nested path, into which fixture file is seeded, and Status and
// Read_Directory then report tree shape, absent path included.
func Test_Operating_System_IO_Directory(t *testing.T) {
	clock := new_operating_system_clock()
	loop, _, driver := operating_system_loop(t, clock)

	root := t.TempDir()
	nested := filepath.Join(root, "a", "b")
	testify.No_Error(t, make_directory(t, loop, driver, nested))
	// Make of file through loop prove Make_Directory built parents: Create fail when
	// directory above path does not exist.
	file_path := filepath.Join(nested, "file.txt")
	write_file(t, loop, driver, file_path, []byte("hello"))

	directory_status, _ := nbio.Storage_Status(loop.Storage, nested)
	testify.True(t, directory_status.Exists)
	testify.True(t, directory_status.Is_Directory)
	testify.False(t, directory_status.Is_Regular)
	regular_status, _ := nbio.Storage_Status(loop.Storage, file_path)
	testify.True(t, regular_status.Exists)
	testify.False(t, regular_status.Is_Directory)
	testify.True(t, regular_status.Is_Regular)
	absent_status, _ := nbio.Storage_Status(loop.Storage, filepath.Join(root, "nope"))
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
	nbio.Network_Send(
		input.Timeline.Network, &send_completion, input.Connected, []byte("ping"),
		REAL_DEADLINE, func(
			completed *time.Completion,
		) {
			testify.No_Error(input.Test, completed.Error)
		})
	buffer := make([]byte, 16)
	received := -1
	var receive_completion time.Completion
	nbio.Network_Receive(
		input.Timeline.Network, &receive_completion, input.Accepted, buffer,
		REAL_DEADLINE, func(
			completed *time.Completion,
		) {
			testify.No_Error(input.Test, completed.Error)
			received = completed.Data
		})
	time.Driver_Run_Until(input.Driver, REAL_DEADLINE,
		func() (finished bool) { return received >= 0 })
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
	nbio.IO_Close(loop, &completion, socket, func(_ *time.Completion) { closed = true })
	time.Driver_Run_Until(driver, REAL_DEADLINE, func() (finished bool) { return closed })
}
