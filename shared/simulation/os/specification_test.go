package os_test

import (
	"testing"

	"local/james-orcales/shared/simulation/os"
	"local/james-orcales/shared/simulation/time"
	"local/james-orcales/shared/testify"
)

// Test_Virtual_OS_Arguments verifies caller storage cannot reach backend state.
func Test_Virtual_OS_Arguments(t *testing.T) {
	virtual := test_virtual()
	system := os.Virtual_OS_To_OS(&virtual)
	storage := [TEST_ARGUMENT_COUNT + 1]string{}
	first_count := os.OS_Arguments(system, storage[:])
	testify.Equal(t, TEST_ARGUMENT_COUNT, first_count)
	testify.Equal(t, "setup", storage[0])
	storage[0] = "edited"
	second_count := os.OS_Arguments(system, storage[:])
	testify.Equal(t, TEST_ARGUMENT_COUNT, second_count)
	testify.Equal(t, "setup", storage[0])
}

// Test_Virtual_OS_Environment verifies caller storage cannot reach backend state.
func Test_Virtual_OS_Environment(t *testing.T) {
	virtual := test_virtual()
	system := os.Virtual_OS_To_OS(&virtual)
	storage := [TEST_ENVIRONMENT_COUNT + 1]string{}
	first_count := os.OS_Environment(system, storage[:])
	testify.Equal(t, TEST_ENVIRONMENT_COUNT, first_count)
	storage[0] = "EDITED=1"
	second_count := os.OS_Environment(system, storage[:])
	testify.Equal(t, TEST_ENVIRONMENT_COUNT, second_count)
	testify.Equal(t, "HOME=/root", storage[0])
}

// Test_Virtual_OS_Variable verifies one lookup: a set name is found, an unset name is not,
// a name set to the empty value is still found, and a later duplicate wins.
func Test_Virtual_OS_Variable(t *testing.T) {
	virtual := test_virtual()
	system := os.Virtual_OS_To_OS(&virtual)
	value, found := os.OS_Variable(system, "HOME")
	testify.True(t, found)
	testify.Equal(t, "/root", value)
	_, absent := os.OS_Variable(system, "ABSENT")
	testify.False(t, absent)
	empty, empty_found := os.OS_Variable(system, "EMPTY")
	testify.True(t, empty_found)
	testify.Empty(t, empty)
	last, _ := os.Environment_Lookup([]string{"SAME=first", "SAME=last"}, "SAME")
	testify.Equal(t, "last", last)
}

// Test_Virtual_OS_Executable verifies the stated image path reads back with no error.
func Test_Virtual_OS_Executable(t *testing.T) {
	virtual := test_virtual()
	system := os.Virtual_OS_To_OS(&virtual)
	path, err := os.OS_Executable(system)
	testify.No_Error(t, err)
	testify.Equal(t, "/usr/local/bin/setup", path)
}

// Test_Virtual_OS_Working_Directory verifies the stated directory reads back with no error.
func Test_Virtual_OS_Working_Directory(t *testing.T) {
	virtual := test_virtual()
	system := os.Virtual_OS_To_OS(&virtual)
	path, err := os.OS_Working_Directory(system)
	testify.No_Error(t, err)
	testify.Equal(t, "/home/simulation", path)
}

// Test_Virtual_OS_Hostname verifies the stated machine name reads back with no error.
func Test_Virtual_OS_Hostname(t *testing.T) {
	virtual := test_virtual()
	system := os.Virtual_OS_To_OS(&virtual)
	name, err := os.OS_Hostname(system)
	testify.No_Error(t, err)
	testify.Equal(t, "simulation", name)
}

// Test_Virtual_OS_Process_Identifier verifies the stated process id reads back.
func Test_Virtual_OS_Process_Identifier(t *testing.T) {
	virtual := test_virtual()
	system := os.Virtual_OS_To_OS(&virtual)
	testify.Equal(t, 4242, os.OS_Process_Identifier(system))
}

// Test_Virtual_OS_Effective_User_Identifier verifies the stated effective user id reads
// back, including zero, which is root rather than an unset field.
func Test_Virtual_OS_Effective_User_Identifier(t *testing.T) {
	virtual := test_virtual()
	system := os.Virtual_OS_To_OS(&virtual)
	testify.Equal(t, 501, os.OS_Effective_User_Identifier(system))
	root := test_virtual()
	root.Effective_User_Identifier = 0
	elevated := os.Virtual_OS_To_OS(&root)
	testify.Zero(t, os.OS_Effective_User_Identifier(elevated))
}

// Test_Virtual_OS_Self_Exec verifies the simulation refuses to replace its own process and
// reports the refusal rather than pretending to succeed.
func Test_Virtual_OS_Self_Exec(t *testing.T) {
	virtual := test_virtual()
	system := os.Virtual_OS_To_OS(&virtual)
	err := os.OS_Self_Exec(system, "/bin/true", []string{"true"}, []string{})
	testify.Error_Is(t, err, os.Self_Exec_Unsupported)
}

// Test_OS_Self_Exec verifies seeded backend cannot replace test process for either environment
// ownership mode.
func Test_OS_Self_Exec(t *testing.T) {
	for _, environment := range [][]string{nil, {}} {
		system, _ := sim_system(0)
		err := os.OS_Self_Exec(system, "/bin/true", []string{"true"}, environment)
		testify.Error_Is(t, err, os.Self_Exec_Unsupported)
	}
}

// Test_OS_Watch_Signal verifies a signal watch resolves exactly once with either its signal or
// Deadline_Exceeded.
func Test_OS_Watch_Signal(t *testing.T) {
	signal_count := 0
	deadline_count := 0
	for seed := uint64(0); seed < 64; seed++ {
		system, driver := sim_system(seed)
		got := os.SIGNAL_EXPIRED
		callback_count := 0
		var operation_err error
		var completion time.Completion
		os.OS_Watch_Signal(system, &completion, os.SIGNAL_TERMINATE, time.NANOSECOND, func(
			_ *time.Completion, signal os.Signal, err error,
		) {
			callback_count++
			got = signal
			operation_err = err
		})
		completed, drive_err := time.Driver_Run_Until(driver,
			16*time.NANOSECOND, func() (finished bool) { return callback_count > 0 })
		testify.No_Error(t, drive_err, seed)
		testify.True(t, completed, seed)
		testify.Equal(t, 1, callback_count, seed)
		if operation_err == time.Deadline_Exceeded {
			deadline_count++
			testify.Equal(t, os.SIGNAL_EXPIRED, got, seed)
		} else {
			testify.No_Error(t, operation_err, seed)
			signal_count++
			testify.Equal(t, os.SIGNAL_TERMINATE, got, seed)
		}
		time.Driver_Run_For(driver, 16*time.NANOSECOND)
		testify.Equal(t, 1, callback_count, seed)
	}
	testify.Positive(t, signal_count)
	testify.Positive(t, deadline_count)
}

// Test_OS_Spawn verifies a spawn delivers a result on the loop.
func Test_OS_Spawn(t *testing.T) {
	system, driver := sim_system(0)
	fired := 0
	var completion time.Completion
	os.OS_Spawn(system, &completion, os.Process_Request{Path: "echo"}, SIM_DEADLINE,
		func(_ *time.Completion, result os.Process_Result, err error) {
			fired++
		})
	time.Driver_Run_For(driver, 16*time.NANOSECOND)
	testify.Equal(t, 1, fired)
}

// Caller-owned operation capacity rejects second ownership before arming its completion.
func Test_OS_Operation_Capacity(t *testing.T) {
	system, driver := sim_system(0)
	var first time.Completion
	os.OS_Watch_Signal(
		system, &first, os.SIGNAL_TERMINATE, SIM_DEADLINE,
		allocation_signal_callback,
	)
	var rejected time.Completion
	testify.Panics(t, func() {
		os.OS_Spawn(
			system, &rejected, os.Process_Request{Path: "true"}, SIM_DEADLINE,
			allocation_process_callback,
		)
	})
	testify.False(t, rejected.Armed)
	time.Driver_Run_For(driver, SIM_DEADLINE)
	testify.False(t, first.Armed)
	testify.Nil(t, first.Backend)
}

// Retirement releases sole storage entry before callback, so callback can arm next operation.
func Test_OS_Callback_Can_Submit(t *testing.T) {
	system, driver := sim_system(0)
	callback_count := 0
	var signal_completion time.Completion
	var spawn_completion time.Completion
	os.OS_Watch_Signal(
		system, &signal_completion, os.SIGNAL_TERMINATE, SIM_DEADLINE,
		func(_ *time.Completion, _ os.Signal, signal_err error) {
			testify.No_Error(t, signal_err)
			callback_count++
			os.OS_Spawn(
				system, &spawn_completion,
				os.Process_Request{Path: "true"}, SIM_DEADLINE,
				func(_ *time.Completion, _ os.Process_Result, spawn_err error) {
					testify.No_Error(t, spawn_err)
					callback_count++
				},
			)
		},
	)
	time.Driver_Run_For(driver, SIM_DEADLINE)
	testify.Equal(t, 2, callback_count)
	testify.Nil(t, signal_completion.Backend)
	testify.Nil(t, spawn_completion.Backend)
}

// Runs one bounded simulated spawn past its modeled completion time and reports its sole result.
func sim_spawn_with_deadline(
	t *testing.T, seed uint64, deadline time.Duration,
) (spawn_err error) {
	t.Helper()
	system, driver := sim_system(seed)
	callback_count := 0
	var completion time.Completion
	os.OS_Spawn(system, &completion, os.Process_Request{Path: "true"}, deadline, func(
		_ *time.Completion, _ os.Process_Result, err error,
	) {
		callback_count++
		spawn_err = err
	})
	time.Driver_Run_Until(driver, SIM_DEADLINE,
		func() (finished bool) { return callback_count > 0 })
	time.Driver_Run_For(driver, 16*time.NANOSECOND)
	testify.Equal(t, 1, callback_count, seed)
	return spawn_err
}

// Test_Spawn_Deadline_Sim verifies a finite spawn deadline wins a latency tie and retires once.
func Test_Spawn_Deadline_Sim(t *testing.T) {
	saw_deadline := false
	saw_tie := false
	for seed := uint64(0); seed < 64; seed++ {
		at_deadline := sim_spawn_with_deadline(t, seed, 4*time.NANOSECOND)
		after_deadline := sim_spawn_with_deadline(t, seed, 5*time.NANOSECOND)
		if at_deadline == time.Deadline_Exceeded {
			saw_deadline = true
		}
		if at_deadline == time.Deadline_Exceeded {
			if after_deadline != time.Deadline_Exceeded {
				saw_tie = true
			}
		}
	}
	testify.True(t, saw_deadline)
	testify.True(t, saw_tie)
}

// Fuzz_OS keeps seed whole so every modeled latency and exit outcome remains reachable.
func Fuzz_OS(f *testing.F) {
	f.Add(uint64(0))
	f.Add(uint64(1))
	f.Add(^uint64(0))
	f.Fuzz(func(t *testing.T, seed uint64) {
		system, driver := sim_system(seed)
		fuzz_signal(t, system, driver)
		fuzz_spawn(t, system, driver)
	})
}

func fuzz_signal(t *testing.T, system os.OS, driver time.Driver) {
	t.Helper()
	callback_count := 0
	delivered := os.SIGNAL_EXPIRED
	var operation_err error
	var completion time.Completion
	os.OS_Watch_Signal(
		system, &completion, os.SIGNAL_INTERRUPT, SIM_FUZZ_OPERATION_DEADLINE,
		func(_ *time.Completion, signal os.Signal, err error) {
			callback_count++
			delivered = signal
			operation_err = err
		},
	)
	time.Driver_Run_For(driver, SIM_DEADLINE)
	testify.Equal(t, 1, callback_count, operation_err)
	testify.Nil(t, completion.Backend)
	if operation_err == nil {
		testify.Equal(t, os.SIGNAL_INTERRUPT, delivered)
		return
	}
	testify.Error_Is(t, operation_err, time.Deadline_Exceeded)
	testify.Equal(t, os.SIGNAL_EXPIRED, delivered)
}

func fuzz_spawn(t *testing.T, system os.OS, driver time.Driver) {
	t.Helper()
	callback_count := 0
	result := os.Process_Result{}
	var operation_err error
	var completion time.Completion
	os.OS_Spawn(
		system, &completion, os.Process_Request{Path: "true"}, SIM_FUZZ_OPERATION_DEADLINE,
		func(_ *time.Completion, spawned os.Process_Result, err error) {
			callback_count++
			result = spawned
			operation_err = err
		},
	)
	time.Driver_Run_For(driver, SIM_DEADLINE)
	testify.Equal(t, 1, callback_count, operation_err)
	testify.Nil(t, completion.Backend)
	if operation_err == nil {
		testify.True(t, result.Exit == 0 || result.Exit == 1, result.Exit)
		return
	}
	testify.Error_Is(t, operation_err, time.Deadline_Exceeded)
	testify.Zero(t, result.Exit)
}

// Every OS operation needs direct heap evidence; functional tests cannot establish it.
func Test_OS_API_Heap_Allocation(t *testing.T) {
	verify_virtual_os_allocations(t)
	verify_simulated_os_allocations(t)
}

func verify_virtual_os_allocations(t *testing.T) {
	virtual := test_virtual()
	var system os.OS
	t.Run("Virtual_OS_To_OS", func(t *testing.T) {
		testify.Zero_Allocation(t, func() { system = os.Virtual_OS_To_OS(&virtual) })
	})
	verify_os_ambient_allocations(t, system, virtual)
	var text string
	var found bool
	t.Run("Environment_Lookup", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			text, found = os.Environment_Lookup(virtual.Environment, "HOME")
		})
		testify.True(t, found)
		testify.Equal(t, "/root", text)
	})
}

func verify_os_ambient_allocations(t *testing.T, system os.OS, virtual os.Virtual_OS) {
	t.Helper()
	argument_storage := [TEST_ARGUMENT_COUNT]string{}
	environment_storage := [TEST_ENVIRONMENT_COUNT]string{}
	var count int
	t.Run("Arguments", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			count = os.OS_Arguments(system, argument_storage[:])
		})
		testify.Equal(t, len(virtual.Arguments), count)
	})
	t.Run("Environment", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			count = os.OS_Environment(system, environment_storage[:])
		})
		testify.Equal(t, len(virtual.Environment), count)
	})
	var text string
	var found bool
	t.Run("Variable", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			text, found = os.OS_Variable(system, "HOME")
		})
		testify.True(t, found)
	})
	verify_virtual_os_text_allocations(t, system, &text)
	var identifier int
	t.Run("Process_Identifier", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			identifier = os.OS_Process_Identifier(system)
		})
		testify.Positive(t, identifier)
	})
	t.Run("Effective_User_Identifier", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			identifier = os.OS_Effective_User_Identifier(system)
		})
	})
	var operation_err error
	t.Run("Self_Exec", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			operation_err = os.OS_Self_Exec(system, "", nil, nil)
		})
		testify.Error_Is(t, operation_err, os.Self_Exec_Unsupported)
	})
}

func verify_virtual_os_text_allocations(t *testing.T, system os.OS, text *string) {
	t.Helper()
	var read_err error
	t.Run("Executable", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			*text, read_err = os.OS_Executable(system)
		})
		testify.No_Error(t, read_err)
	})
	t.Run("Working_Directory", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			*text, read_err = os.OS_Working_Directory(system)
		})
		testify.No_Error(t, read_err)
	})
	t.Run("Hostname", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			*text, read_err = os.OS_Hostname(system)
		})
		testify.No_Error(t, read_err)
	})
}

func verify_simulated_os_allocations(t *testing.T) {
	state := time.Virtual_Timeline{}
	queue := [SIM_TIMELINE_CAPACITY]*time.Completion{}
	events := [SIM_TIMELINE_CAPACITY]time.Virtual_Event{}
	pump, driver, _ := time.New_Virtual_Timeline(&state,
		time.Virtual_Clock{Resolution: time.NANOSECOND}, time.Virtual_Timeline_Memory{
			Queue: queue[:], Events: events[:],
		})
	virtual := test_virtual()
	var simulated os.Sim
	operations := [SIM_OPERATION_CAPACITY]os.Sim_Operation{}
	var system os.OS
	t.Run("New_Simulated_OS", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			system = os.New_Simulated_OS(
				&simulated, 0, virtual, pump,
				os.Sim_Memory{Operations: operations[:]},
			)
		})
	})
	t.Run("Ambient", func(t *testing.T) {
		verify_os_ambient_allocations(t, system, virtual)
	})
	var signal_completion time.Completion
	t.Run("Watch_Signal", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			os.OS_Watch_Signal(
				system,
				&signal_completion, os.SIGNAL_TERMINATE, SIM_DEADLINE,
				allocation_signal_callback,
			)
			time.Driver_Run_For(driver, SIM_DEADLINE)
		})
	})
	var spawn_completion time.Completion
	t.Run("Spawn", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			os.OS_Spawn(
				system,
				&spawn_completion, os.Process_Request{Path: "true"}, SIM_DEADLINE,
				allocation_process_callback,
			)
			time.Driver_Run_For(driver, SIM_DEADLINE)
		})
	})
	verify_simulated_deadline_allocations(
		t, &simulated, operations[:], virtual, pump, driver,
	)
}

func verify_simulated_deadline_allocations(
	t *testing.T, simulated *os.Sim, operations []os.Sim_Operation,
	virtual os.Virtual_OS, pump time.Timeline, driver time.Driver,
) {
	t.Helper()
	var system os.OS
	var signal_completion time.Completion
	t.Run("Watch_Signal_Deadline", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			system = os.New_Simulated_OS(
				simulated, 0, virtual, pump, os.Sim_Memory{Operations: operations},
			)
			os.OS_Watch_Signal(
				system, &signal_completion, os.SIGNAL_TERMINATE, time.NANOSECOND,
				allocation_signal_callback,
			)
			time.Driver_Run_For(driver, 2*time.NANOSECOND)
		})
		testify.Error_Is(t, signal_completion.Error, time.Deadline_Exceeded)
	})
	var spawn_completion time.Completion
	t.Run("Spawn_Deadline", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			system = os.New_Simulated_OS(
				simulated, 0, virtual, pump, os.Sim_Memory{Operations: operations},
			)
			os.OS_Spawn(
				system, &spawn_completion, os.Process_Request{Path: "true"},
				time.NANOSECOND, allocation_process_callback,
			)
			time.Driver_Run_For(driver, 2*time.NANOSECOND)
		})
		testify.Error_Is(t, spawn_completion.Error, time.Deadline_Exceeded)
	})
}

func allocation_signal_callback(
	completion *time.Completion, signal os.Signal, err error,
) {
	completion.Data = int(signal)
	completion.Error = err
}

func allocation_process_callback(
	completion *time.Completion, result os.Process_Result, err error,
) {
	completion.Data = result.Exit
	completion.Error = err
}

// Test root keeps caller-owned state alive beside vtables that point into it.
func sim_system(seed uint64) (system os.OS, driver time.Driver) {
	state := time.Virtual_Timeline{}
	queue := [SIM_TIMELINE_CAPACITY]*time.Completion{}
	events := [SIM_TIMELINE_CAPACITY]time.Virtual_Event{}
	pump, driver, _ := time.New_Virtual_Timeline(&state,
		time.Virtual_Clock{Resolution: time.NANOSECOND}, time.Virtual_Timeline_Memory{
			Queue: queue[:], Events: events[:],
		})
	simulated := os.Sim{}
	operations := [SIM_OPERATION_CAPACITY]os.Sim_Operation{}
	return os.New_Simulated_OS(
		&simulated, seed, test_virtual(), pump,
		os.Sim_Memory{Operations: operations[:]},
	), driver
}

// The Run_Until cap for the simulated tests: generous, since a completion returns the pump the
// instant it fires — this bound only bites a genuine stall.
const SIM_DEADLINE = 4096 * time.NANOSECOND

// Mid-range deadline makes seed sweep reach both modeled event and deadline winner.
const SIM_FUZZ_OPERATION_DEADLINE = 4 * time.NANOSECOND

// A bounded test loop needs room for one operation and its competing timeout.
const SIM_TIMELINE_CAPACITY = 2

// One sequential OS operation stays armed in these tests.
const SIM_OPERATION_CAPACITY = 1

// Fixed fixture capacity makes caller-owned copies exact.
const TEST_ARGUMENT_COUNT = 2

// Fixed fixture capacity makes caller-owned copies exact.
const TEST_ENVIRONMENT_COUNT = 3

// Returns one fully stated simulated process, so each test reads a value it did not set up
// itself and a missing field shows as a wrong answer rather than as a zero.
func test_virtual() (virtual os.Virtual_OS) {
	return os.Virtual_OS{
		Arguments:          []string{"setup", "--dry-run"},
		Environment:        []string{"HOME=/root", "EMPTY=", "PATH=/usr/bin"},
		Executable:         "/usr/local/bin/setup",
		Working_Directory:  "/home/simulation",
		Hostname:           "simulation",
		Process_Identifier: 4242,

		Effective_User_Identifier: 501,
	}
}
