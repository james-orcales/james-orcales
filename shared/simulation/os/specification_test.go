package os_test

import (
	"testing"

	"local/james-orcales/shared/simulation/os"
	"local/james-orcales/shared/simulation/time"
)

// Test_Virtual_OS_Arguments verifies the stated argv reads back whole, and that a caller
// editing the result cannot change what a later read sees.
func Test_Virtual_OS_Arguments(t *testing.T) {
	system := os.Virtual_OS_To_OS(test_virtual())
	first := system.Arguments()
	if len(first) != 2 {
		t.Fatalf("arguments = %d entries, want 2", len(first))
	}
	if first[0] != "setup" {
		t.Fatalf("arguments[0] = %q, want %q", first[0], "setup")
	}
	first[0] = "edited"
	if second := system.Arguments(); second[0] != "setup" {
		t.Fatalf("a later read = %q, want the edit not to reach it", second[0])
	}
}

// Test_Virtual_OS_Environment verifies the stated variables read back whole, and that a
// caller editing the result cannot change what a later read sees.
func Test_Virtual_OS_Environment(t *testing.T) {
	system := os.Virtual_OS_To_OS(test_virtual())
	first := system.Environment()
	if len(first) != 3 {
		t.Fatalf("environment = %d entries, want 3", len(first))
	}
	first[0] = "EDITED=1"
	if second := system.Environment(); second[0] != "HOME=/root" {
		t.Fatalf("a later read = %q, want the edit not to reach it", second[0])
	}
}

// Test_Virtual_OS_Variable verifies one lookup: a set name is found, an unset name is not,
// a name set to the empty value is still found, and a later duplicate wins.
func Test_Virtual_OS_Variable(t *testing.T) {
	system := os.Virtual_OS_To_OS(test_virtual())
	value, found := system.Variable("HOME")
	if !found {
		t.Fatal("HOME is set, want found")
	}
	if value != "/root" {
		t.Fatalf("HOME = %q, want %q", value, "/root")
	}
	if _, absent := system.Variable("ABSENT"); absent {
		t.Fatal("ABSENT is unset, want not found")
	}
	empty, empty_found := system.Variable("EMPTY")
	if !empty_found {
		t.Fatal("EMPTY is set to the empty value, want found")
	}
	if empty != "" {
		t.Fatalf("EMPTY = %q, want the empty value", empty)
	}
	last, _ := os.Environment_Lookup([]string{"SAME=first", "SAME=last"}, "SAME")
	if last != "last" {
		t.Fatalf("a duplicated name = %q, want the later entry %q", last, "last")
	}
}

// Test_Virtual_OS_Executable verifies the stated image path reads back with no error.
func Test_Virtual_OS_Executable(t *testing.T) {
	system := os.Virtual_OS_To_OS(test_virtual())
	path, err := system.Executable()
	if err != nil {
		t.Fatalf("executable: %v", err)
	}
	if path != "/usr/local/bin/setup" {
		t.Fatalf("executable = %q, want %q", path, "/usr/local/bin/setup")
	}
}

// Test_Virtual_OS_Working_Directory verifies the stated directory reads back with no error.
func Test_Virtual_OS_Working_Directory(t *testing.T) {
	system := os.Virtual_OS_To_OS(test_virtual())
	path, err := system.Working_Directory()
	if err != nil {
		t.Fatalf("working directory: %v", err)
	}
	if path != "/home/simulation" {
		t.Fatalf("working directory = %q, want %q", path, "/home/simulation")
	}
}

// Test_Virtual_OS_Hostname verifies the stated machine name reads back with no error.
func Test_Virtual_OS_Hostname(t *testing.T) {
	system := os.Virtual_OS_To_OS(test_virtual())
	name, err := system.Hostname()
	if err != nil {
		t.Fatalf("hostname: %v", err)
	}
	if name != "simulation" {
		t.Fatalf("hostname = %q, want %q", name, "simulation")
	}
}

// Test_Virtual_OS_Identifier verifies the stated process id reads back.
func Test_Virtual_OS_Identifier(t *testing.T) {
	system := os.Virtual_OS_To_OS(test_virtual())
	if identifier := system.Identifier(); identifier != 4242 {
		t.Fatalf("identifier = %d, want 4242", identifier)
	}
}

// Test_Virtual_OS_Effective_User_Identifier verifies the stated effective user id reads
// back, including zero, which is root rather than an unset field.
func Test_Virtual_OS_Effective_User_Identifier(t *testing.T) {
	system := os.Virtual_OS_To_OS(test_virtual())
	if identifier := system.Effective_User_Identifier(); identifier != 501 {
		t.Fatalf("effective user identifier = %d, want 501", identifier)
	}
	root := test_virtual()
	root.Effective_User_Identifier = 0
	elevated := os.Virtual_OS_To_OS(root)
	if identifier := elevated.Effective_User_Identifier(); identifier != 0 {
		t.Fatalf("effective user identifier = %d, want 0 for root", identifier)
	}
}

// Test_Virtual_OS_Self_Exec verifies the simulation refuses to replace its own process and
// reports the refusal rather than pretending to succeed.
func Test_Virtual_OS_Self_Exec(t *testing.T) {
	system := os.Virtual_OS_To_OS(test_virtual())
	err := system.Self_Exec("/bin/true", []string{"true"}, []string{})
	if err != os.Self_Exec_Unsupported {
		t.Fatalf("self exec = %v, want %v", err, os.Self_Exec_Unsupported)
	}
}

// Test_OS_Watch_Signal verifies a signal watch resolves exactly once with either its signal or
// Deadline_Exceeded.
func Test_OS_Watch_Signal(t *testing.T) {
	signal_count := 0
	deadline_count := 0
	for seed := uint64(0); seed < 64; seed++ {
		system, driver := sim_system(seed)
		got := os.Signal(-1)
		callback_count := 0
		var operation_err error
		var completion time.Completion
		system.Watch_Signal(&completion, func(
			_ *time.Completion, signal os.Signal, err error,
		) {
			callback_count++
			got = signal
			operation_err = err
		}, os.SIGNAL_TERMINATE, time.NANOSECOND)
		completed, drive_err := driver.Run_Until(
			func() (finished bool) { return callback_count > 0 }, 16*time.NANOSECOND)
		if drive_err != nil {
			t.Fatalf("seed %d: drive: %v", seed, drive_err)
		}
		if !completed {
			t.Fatalf("seed %d: signal watch did not resolve", seed)
		}
		if callback_count != 1 {
			t.Fatalf("seed %d: callback count = %d, want 1", seed, callback_count)
		}
		if operation_err == time.Deadline_Exceeded {
			deadline_count++
			if got != -1 {
				t.Fatalf("seed %d: deadline yielded signal %d", seed, got)
			}
		} else {
			if operation_err != nil {
				t.Fatalf("seed %d: signal watch: %v", seed, operation_err)
			}
			signal_count++
			if got != os.SIGNAL_TERMINATE {
				t.Fatalf("seed %d: signal = %d, want terminate", seed, got)
			}
		}
		driver.Run_For(16 * time.NANOSECOND)
		if callback_count != 1 {
			t.Fatalf("seed %d: callback repeated %d times", seed, callback_count)
		}
	}
	if signal_count == 0 {
		t.Fatal("seed sweep witnessed no signal before the deadline")
	}
	if deadline_count == 0 {
		t.Fatal("seed sweep witnessed no signal deadline")
	}
}

// Test_OS_Spawn verifies a spawn delivers a result on the loop.
func Test_OS_Spawn(t *testing.T) {
	system, driver := sim_system(0)
	fired := 0
	var completion time.Completion
	system.Spawn(&completion, func(_ *time.Completion, result os.Process_Result, err error) {
		fired++
	}, os.Process_Request{Path: "echo"}, SIM_DEADLINE)
	driver.Run_For(16 * time.NANOSECOND)
	if fired != 1 {
		t.Fatalf("spawn callback fired %d times, want 1", fired)
	}
}

// Runs one bounded simulated spawn past its modeled completion time and reports its sole result.
func sim_spawn_with_deadline(
	t *testing.T, seed uint64, deadline time.Duration,
) (spawn_err error) {
	t.Helper()
	system, driver := sim_system(seed)
	callback_count := 0
	var completion time.Completion
	system.Spawn(&completion, func(
		_ *time.Completion, _ os.Process_Result, err error,
	) {
		callback_count++
		spawn_err = err
	}, os.Process_Request{Path: "true"}, deadline)
	driver.Run_Until(func() (finished bool) { return callback_count > 0 }, SIM_DEADLINE)
	driver.Run_For(16 * time.NANOSECOND)
	if callback_count != 1 {
		t.Fatalf("seed %d: spawn callback count = %d, want 1", seed, callback_count)
	}
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
	if !saw_deadline {
		t.Fatal("seed sweep witnessed no spawn deadline")
	}
	if !saw_tie {
		t.Fatal("seed sweep witnessed no spawn latency tie lost to the deadline")
	}
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
		Arguments:         []string{"setup", "--dry-run"},
		Environment:       []string{"HOME=/root", "EMPTY=", "PATH=/usr/bin"},
		Executable:        "/usr/local/bin/setup",
		Working_Directory: "/home/simulation",
		Hostname:          "simulation",
		Identifier:        4242,

		Effective_User_Identifier: 501,
	}
}
