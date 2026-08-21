package os_test

import (
	"testing"

	"local/james-orcales/shared/simulation/os"
	"local/james-orcales/shared/simulation/time"
	"local/james-orcales/shared/testify"
)

// Test_Virtual_OS_Arguments verifies the stated argv reads back whole, and that a caller
// editing the result cannot change what a later read sees.
func Test_Virtual_OS_Arguments(t *testing.T) {
	system := os.Virtual_OS_To_OS(test_virtual())
	first := system.Arguments()
	testify.Count(t, first, 2)
	testify.Equal(t, "setup", first[0])
	first[0] = "edited"
	second := system.Arguments()
	testify.Equal(t, "setup", second[0])
}

// Test_Virtual_OS_Environment verifies the stated variables read back whole, and that a
// caller editing the result cannot change what a later read sees.
func Test_Virtual_OS_Environment(t *testing.T) {
	system := os.Virtual_OS_To_OS(test_virtual())
	first := system.Environment()
	testify.Count(t, first, 3)
	first[0] = "EDITED=1"
	second := system.Environment()
	testify.Equal(t, "HOME=/root", second[0])
}

// Test_Virtual_OS_Variable verifies one lookup: a set name is found, an unset name is not,
// a name set to the empty value is still found, and a later duplicate wins.
func Test_Virtual_OS_Variable(t *testing.T) {
	system := os.Virtual_OS_To_OS(test_virtual())
	value, found := system.Variable("HOME")
	testify.True(t, found)
	testify.Equal(t, "/root", value)
	_, absent := system.Variable("ABSENT")
	testify.False(t, absent)
	empty, empty_found := system.Variable("EMPTY")
	testify.True(t, empty_found)
	testify.Empty(t, empty)
	last, _ := os.Environment_Lookup([]string{"SAME=first", "SAME=last"}, "SAME")
	testify.Equal(t, "last", last)
}

// Test_Virtual_OS_Executable verifies the stated image path reads back with no error.
func Test_Virtual_OS_Executable(t *testing.T) {
	system := os.Virtual_OS_To_OS(test_virtual())
	path, err := system.Executable()
	testify.No_Error(t, err)
	testify.Equal(t, "/usr/local/bin/setup", path)
}

// Test_Virtual_OS_Working_Directory verifies the stated directory reads back with no error.
func Test_Virtual_OS_Working_Directory(t *testing.T) {
	system := os.Virtual_OS_To_OS(test_virtual())
	path, err := system.Working_Directory()
	testify.No_Error(t, err)
	testify.Equal(t, "/home/simulation", path)
}

// Test_Virtual_OS_Hostname verifies the stated machine name reads back with no error.
func Test_Virtual_OS_Hostname(t *testing.T) {
	system := os.Virtual_OS_To_OS(test_virtual())
	name, err := system.Hostname()
	testify.No_Error(t, err)
	testify.Equal(t, "simulation", name)
}

// Test_Virtual_OS_Process_Identifier verifies the stated process id reads back.
func Test_Virtual_OS_Process_Identifier(t *testing.T) {
	system := os.Virtual_OS_To_OS(test_virtual())
	testify.Equal(t, 4242, system.Process_Identifier())
}

// Test_Virtual_OS_Effective_User_Identifier verifies the stated effective user id reads
// back, including zero, which is root rather than an unset field.
func Test_Virtual_OS_Effective_User_Identifier(t *testing.T) {
	system := os.Virtual_OS_To_OS(test_virtual())
	testify.Equal(t, 501, system.Effective_User_Identifier())
	root := test_virtual()
	root.Effective_User_Identifier = 0
	elevated := os.Virtual_OS_To_OS(root)
	testify.Zero(t, elevated.Effective_User_Identifier())
}

// Test_Virtual_OS_Self_Exec verifies the simulation refuses to replace its own process and
// reports the refusal rather than pretending to succeed.
func Test_Virtual_OS_Self_Exec(t *testing.T) {
	system := os.Virtual_OS_To_OS(test_virtual())
	err := system.Self_Exec("/bin/true", []string{"true"}, []string{})
	testify.Error_Is(t, err, os.Self_Exec_Unsupported)
}

// Test_OS_Self_Exec verifies seeded backend cannot replace test process for either environment
// ownership mode.
func Test_OS_Self_Exec(t *testing.T) {
	for _, environment := range [][]string{nil, {}} {
		system, _ := sim_system(0)
		err := system.Self_Exec("/bin/true", []string{"true"}, environment)
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
		system.Watch_Signal(&completion, os.SIGNAL_TERMINATE, time.NANOSECOND, func(
			_ *time.Completion, signal os.Signal, err error,
		) {
			callback_count++
			got = signal
			operation_err = err
		})
		completed, drive_err := driver.Run_Until(
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
		driver.Run_For(16 * time.NANOSECOND)
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
	system.Spawn(&completion, os.Process_Request{Path: "echo"}, SIM_DEADLINE,
		func(_ *time.Completion, result os.Process_Result, err error) {
			fired++
		})
	driver.Run_For(16 * time.NANOSECOND)
	testify.Equal(t, 1, fired)
}

// Runs one bounded simulated spawn past its modeled completion time and reports its sole result.
func sim_spawn_with_deadline(
	t *testing.T, seed uint64, deadline time.Duration,
) (spawn_err error) {
	t.Helper()
	system, driver := sim_system(seed)
	callback_count := 0
	var completion time.Completion
	system.Spawn(&completion, os.Process_Request{Path: "true"}, deadline, func(
		_ *time.Completion, _ os.Process_Result, err error,
	) {
		callback_count++
		spawn_err = err
	})
	driver.Run_Until(SIM_DEADLINE, func() (finished bool) { return callback_count > 0 })
	driver.Run_For(16 * time.NANOSECOND)
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

// Builds the simulated operating system beside the loop that owns its order, seeded by seed. A
// test holds the OS and the driver — never the backend, which New_Simulated_OS keeps to
// itself so the
// run stays a pure function of the seed.
func sim_system(seed uint64) (system os.OS, driver time.Driver) {
	pump, driver, _ := time.New_Virtual_Timeline(
		time.Virtual_Clock{Resolution: time.NANOSECOND})
	return os.New_Simulated_OS(seed, test_virtual(), pump), driver
}

// The Run_Until cap for the simulated tests: generous, since a completion returns the pump the
// instant it fires — this bound only bites a genuine stall.
const SIM_DEADLINE = 4096 * time.NANOSECOND

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
