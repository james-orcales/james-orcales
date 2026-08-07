package os_test

import (
	"testing"

	"local/james-orcales/shared/os"
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
