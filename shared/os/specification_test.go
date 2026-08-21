package os_test

import (
	"testing"

	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/os"
	"local/james-orcales/shared/slices"
	"local/james-orcales/shared/testify"
)

// Test_Virtual_OS_Arguments verifies caller storage cannot reach backend state.
func Test_Virtual_OS_Arguments(t *testing.T) {
	virtual := test_virtual()
	system := os.Virtual_OS_To_OS(&virtual)
	storage := [TEST_ARGUMENT_COUNT + 1]string{}
	first_count := os.OS_Arguments(system, storage[:])
	testify.Equal(t, os.Entry_Count(TEST_ARGUMENT_COUNT), first_count)
	testify.Equal(t, "setup", storage[0])
	storage[0] = "edited"
	second_count := os.OS_Arguments(system, storage[:])
	testify.Equal(t, os.Entry_Count(TEST_ARGUMENT_COUNT), second_count)
	testify.Equal(t, "setup", storage[0])
}

// Test_Virtual_OS_Environment verifies caller storage cannot reach backend state.
func Test_Virtual_OS_Environment(t *testing.T) {
	virtual := test_virtual()
	system := os.Virtual_OS_To_OS(&virtual)
	storage := [TEST_ENVIRONMENT_COUNT + 1]string{}
	first_count := os.OS_Environment(system, storage[:])
	testify.Equal(t, os.Entry_Count(TEST_ENVIRONMENT_COUNT), first_count)
	storage[0] = "EDITED=1"
	second_count := os.OS_Environment(system, storage[:])
	testify.Equal(t, os.Entry_Count(TEST_ENVIRONMENT_COUNT), second_count)
	testify.Equal(t, "HOME=/root", storage[0])
}

// Test_Virtual_OS_Variable verifies one lookup: a set name is found, an unset name is not,
// a name set to the empty value is still found, and a later duplicate wins.
func Test_Virtual_OS_Variable(t *testing.T) {
	virtual := test_virtual()
	system := os.Virtual_OS_To_OS(&virtual)
	value, found := os.OS_Variable(system, "HOME")
	testify.True(t, bool(found))
	testify.Equal(t, os.Variable_Value("/root"), value)
	_, absent := os.OS_Variable(system, "ABSENT")
	testify.False(t, bool(absent))
	empty, empty_found := os.OS_Variable(system, "EMPTY")
	testify.True(t, bool(empty_found))
	testify.Empty(t, empty)
	last, _ := os.Environment_Lookup([]string{"SAME=first", "SAME=last"}, "SAME")
	testify.Equal(t, os.Variable_Value("last"), last)
}

// Test_Virtual_OS_Executable verifies the stated image path reads back with no error.
func Test_Virtual_OS_Executable(t *testing.T) {
	virtual := test_virtual()
	system := os.Virtual_OS_To_OS(&virtual)
	path, err := os.OS_Executable(system)
	testify.No_Error(t, err)
	testify.Equal(t, os.Executable_Path("/usr/local/bin/setup"), path)
}

// Test_Virtual_OS_Working_Directory verifies the stated directory reads back with no error.
func Test_Virtual_OS_Working_Directory(t *testing.T) {
	virtual := test_virtual()
	system := os.Virtual_OS_To_OS(&virtual)
	path, err := os.OS_Working_Directory(system)
	testify.No_Error(t, err)
	testify.Equal(t, os.Working_Directory_Path("/home/simulation"), path)
}

// Test_Virtual_OS_Hostname verifies the stated machine name reads back with no error.
func Test_Virtual_OS_Hostname(t *testing.T) {
	virtual := test_virtual()
	system := os.Virtual_OS_To_OS(&virtual)
	name, err := os.OS_Hostname(system)
	testify.No_Error(t, err)
	testify.Equal(t, os.Hostname("simulation"), name)
}

// Test_Virtual_OS_Process_Identifier verifies the stated process id reads back.
func Test_Virtual_OS_Process_Identifier(t *testing.T) {
	virtual := test_virtual()
	system := os.Virtual_OS_To_OS(&virtual)
	testify.Equal(t, os.Process_Identifier(4242), os.OS_Process_Identifier(system))
}

// Test_Virtual_OS_Effective_User_Identifier verifies the stated effective user id reads
// back, including zero, which is root rather than an unset field.
func Test_Virtual_OS_Effective_User_Identifier(t *testing.T) {
	virtual := test_virtual()
	system := os.Virtual_OS_To_OS(&virtual)
	testify.Equal(
		t, os.Effective_User_Identifier(501), os.OS_Effective_User_Identifier(system),
	)
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

// Test_OS_Self_Exec verifies simulated backend cannot replace test process for either
// environment ownership mode.
func Test_OS_Self_Exec(t *testing.T) {
	for _, environment := range [][]string{nil, {}} {
		virtual := test_virtual()
		system := os.Virtual_OS_To_OS(&virtual)
		err := os.OS_Self_Exec(system, "/bin/true", []string{"true"}, environment)
		testify.Error_Is(t, err, os.Self_Exec_Unsupported)
	}
}

// Every OS operation needs direct heap evidence; functional tests cannot establish it.
func Test_OS_API_Heap_Allocation(t *testing.T) {
	verify_virtual_os_allocations(t)
}

// Every bounded domain reaches both limits and small values through production entries.
func Test_OS_Invariant_Boundaries(t *testing.T) {
	lengths := [...]int{
		slices.COUNT_MINIMUM,
		slices.COUNT_MINIMUM + 1,
		slices.COUNT_MINIMUM + 2,
		slices.COUNT_MAXIMUM,
	}
	process_identifiers := [...]os.Process_Identifier{
		os.Process_Identifier(os.PROCESS_IDENTIFIER_MINIMUM),
		os.Process_Identifier(os.PROCESS_IDENTIFIER_MINIMUM + 1),
		os.Process_Identifier(os.PROCESS_IDENTIFIER_MINIMUM + 2),
		os.Process_Identifier(bits.INTEGER_MAXIMUM),
	}
	effective_user_identifiers := [...]os.Effective_User_Identifier{
		os.Effective_User_Identifier(slices.COUNT_MINIMUM),
		os.Effective_User_Identifier(slices.COUNT_MINIMUM + 1),
		os.Effective_User_Identifier(slices.COUNT_MINIMUM + 2),
		os.Effective_User_Identifier(bits.INTEGER_MAXIMUM),
	}
	for index, byte_size := range lengths {
		text := string(make([]byte, byte_size))
		arguments := make(os.Arguments, byte_size)
		environment := make(os.Environment, byte_size)
		if byte_size > slices.COUNT_MINIMUM {
			environment[0] = text + "=" + text
		}
		virtual := os.Virtual_OS{
			Arguments:                 arguments,
			Environment:               environment,
			Executable:                os.Executable_Path(text),
			Working_Directory:         os.Working_Directory_Path(text),
			Hostname:                  os.Hostname(text),
			Process_Identifier:        process_identifiers[index],
			Effective_User_Identifier: effective_user_identifiers[index],
		}
		system := os.Virtual_OS_To_OS(&virtual)
		argument_destination := make(os.Arguments, byte_size)
		environment_destination := make(os.Environment, byte_size)
		os.OS_Arguments(system, argument_destination)
		os.OS_Environment(system, environment_destination)
		os.OS_Variable(system, os.Variable_Name(text))
		os.Environment_Lookup(environment, os.Variable_Name(text))
		os.OS_Executable(system)
		os.OS_Working_Directory(system)
		os.OS_Hostname(system)
		os.OS_Process_Identifier(system)
		os.OS_Effective_User_Identifier(system)
		os.OS_Self_Exec(system, os.Executable_Path(text), arguments, environment)
	}
}

func verify_virtual_os_allocations(t *testing.T) {
	virtual := test_virtual()
	var system os.OS
	t.Run("Virtual_OS_To_OS", func(t *testing.T) {
		testify.Zero_Allocation(t, func() { system = os.Virtual_OS_To_OS(&virtual) })
	})
	verify_os_ambient_allocations(t, system, virtual)
	var text os.Variable_Value
	var found os.Variable_Found
	t.Run("Environment_Lookup", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			text, found = os.Environment_Lookup(virtual.Environment, "HOME")
		})
		testify.True(t, bool(found))
		testify.Equal(t, os.Variable_Value("/root"), text)
	})
}

func verify_os_ambient_allocations(t *testing.T, system os.OS, virtual os.Virtual_OS) {
	t.Helper()
	argument_storage := [TEST_ARGUMENT_COUNT]string{}
	environment_storage := [TEST_ENVIRONMENT_COUNT]string{}
	var count os.Entry_Count
	t.Run("Arguments", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			count = os.OS_Arguments(system, argument_storage[:])
		})
		testify.Equal(t, os.Entry_Count(len(virtual.Arguments)), count)
	})
	t.Run("Environment", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			count = os.OS_Environment(system, environment_storage[:])
		})
		testify.Equal(t, os.Entry_Count(len(virtual.Environment)), count)
	})
	var text os.Variable_Value
	var found os.Variable_Found
	t.Run("Variable", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			text, found = os.OS_Variable(system, "HOME")
		})
		testify.True(t, bool(found))
	})
	verify_virtual_os_text_allocations(t, system, &text)
	var identifier os.Process_Identifier
	t.Run("Process_Identifier", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			identifier = os.OS_Process_Identifier(system)
		})
		testify.Positive(t, identifier)
	})
	t.Run("Effective_User_Identifier", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			identifier = os.Process_Identifier(os.OS_Effective_User_Identifier(system))
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

func verify_virtual_os_text_allocations(t *testing.T, system os.OS, text *os.Variable_Value) {
	t.Helper()
	var read_err error
	t.Run("Executable", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			path, err := os.OS_Executable(system)
			*text, read_err = os.Variable_Value(path), err
		})
		testify.No_Error(t, read_err)
	})
	t.Run("Working_Directory", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			path, err := os.OS_Working_Directory(system)
			*text, read_err = os.Variable_Value(path), err
		})
		testify.No_Error(t, read_err)
	})
	t.Run("Hostname", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			name, err := os.OS_Hostname(system)
			*text, read_err = os.Variable_Value(name), err
		})
		testify.No_Error(t, read_err)
	})
}

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
