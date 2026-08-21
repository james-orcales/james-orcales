package os_test

import (
	"syscall"
	"testing"

	"local/james-orcales/shared/math/bits"
	shared_os "local/james-orcales/shared/os"
	"local/james-orcales/shared/os/default"
	"local/james-orcales/shared/slices"
	"local/james-orcales/shared/testify"
)

// Test_Operating_System_Smoke verifies the host answers every ambient reader. The values a
// machine supplies are not deterministic, so this checks their shape, not their content. The
// constructor asserts every slot is filled, so reaching the end proves the vtable is whole.
func Test_Operating_System_Smoke(t *testing.T) {
	state := test_host()
	host := os.New_Operating_System(&state)
	arguments := [slices.SLICE_COUNT_MAXIMUM]string{}
	testify.Positive(t, os.OS_Arguments(host, arguments[:]))
	testify.Positive(t, os.OS_Process_Identifier(host))
	testify.True(t, os.OS_Effective_User_Identifier(host) >= 0)
	directory, directory_err := os.OS_Working_Directory(host)
	testify.No_Error(t, directory_err)
	testify.True(t, len(directory) > 0 && directory[0] == '/', directory)
	executable, executable_err := os.OS_Executable(host)
	testify.No_Error(t, executable_err)
	testify.Not_Empty(t, executable)
	name, name_err := os.OS_Hostname(host)
	testify.No_Error(t, name_err)
	testify.Not_Empty(t, name)
}

// Test_Operating_System_Environment verifies Environment and Variable agree, because they reach
// the kernel by different calls: Environment reads the whole block and Variable reads one name.
func Test_Operating_System_Environment(t *testing.T) {
	state := test_host()
	host := os.New_Operating_System(&state)
	_, invalid_found := os.OS_Variable(host, "=")
	testify.False(t, bool(invalid_found))
	storage := [slices.SLICE_COUNT_MAXIMUM]string{}
	count := os.OS_Environment(host, storage[:])
	if !testify.Positive(t, count) {
		return
	}
	variables := storage[:count]
	for _, variable := range variables {
		separator := -1
		for index := range variable {
			if variable[index] == '=' {
				separator = index
				break
			}
		}
		if !testify.True(t, separator >= 0, variable) {
			continue
		}
		name := variable[:separator]
		expected := variable[separator+1:]
		value, single := os.OS_Variable(host, shared_os.Variable_Name(name))
		if !testify.True(t, bool(single), name) {
			continue
		}
		testify.Equal(t, shared_os.Variable_Value(expected), value, name)
	}
}

// Test_Operating_System_Arguments_Copy verifies caller storage cannot reach runtime-owned argv.
func Test_Operating_System_Arguments_Copy(t *testing.T) {
	state := test_host()
	host := os.New_Operating_System(&state)
	storage := [slices.SLICE_COUNT_MAXIMUM]string{}
	first_count := os.OS_Arguments(host, storage[:])
	original := storage[0]
	storage[0] = "edited"
	second_count := os.OS_Arguments(host, storage[:])
	testify.Equal(t, first_count, second_count)
	testify.Equal(t, original, storage[0])
}

// Host readers reject partial caller storage because truncating ambient state hides input.
func Test_Operating_System_Destination_Capacity(t *testing.T) {
	state := test_host()
	host := os.New_Operating_System(&state)
	testify.Panics(t, func() {
		os.OS_Arguments(host, nil)
	})
	testify.Panics(t, func() {
		os.OS_Environment(host, nil)
	})
}

// Composition wrappers preserve every bounded value accepted by injected OS.
func Test_Operating_System_Invariant_Boundaries(t *testing.T) {
	lengths := [...]int{
		slices.COUNT_MINIMUM,
		slices.COUNT_MINIMUM + 1,
		slices.COUNT_MINIMUM + 2,
		slices.COUNT_MAXIMUM,
	}
	process_identifiers := [...]shared_os.Process_Identifier{
		shared_os.Process_Identifier(shared_os.PROCESS_IDENTIFIER_MINIMUM),
		shared_os.Process_Identifier(shared_os.PROCESS_IDENTIFIER_MINIMUM + 1),
		shared_os.Process_Identifier(shared_os.PROCESS_IDENTIFIER_MINIMUM + 2),
		shared_os.Process_Identifier(bits.INTEGER_MAXIMUM),
	}
	effective_user_identifiers := [...]shared_os.Effective_User_Identifier{
		shared_os.Effective_User_Identifier(slices.COUNT_MINIMUM),
		shared_os.Effective_User_Identifier(slices.COUNT_MINIMUM + 1),
		shared_os.Effective_User_Identifier(slices.COUNT_MINIMUM + 2),
		shared_os.Effective_User_Identifier(bits.INTEGER_MAXIMUM),
	}
	for index, byte_size := range lengths {
		text := string(make([]byte, byte_size))
		arguments := make(shared_os.Arguments, byte_size)
		environment := make(shared_os.Environment, byte_size)
		if byte_size > slices.COUNT_MINIMUM {
			environment[0] = text + "=" + text
		}
		state := os.Host{
			Arguments:         arguments,
			Environment:       environment,
			Executable:        shared_os.Executable_Path(text),
			Working_Directory: shared_os.Working_Directory_Path(text),
			Hostname:          shared_os.Hostname(text),
			Self_Exec:         test_self_exec_workspace(),
		}
		actual := os.New_Operating_System(&state)
		os.OS_Self_Exec(
			actual, shared_os.Executable_Path(text), arguments, environment,
		)
		virtual := shared_os.Virtual_OS{
			Arguments:                 arguments,
			Environment:               environment,
			Executable:                shared_os.Executable_Path(text),
			Working_Directory:         shared_os.Working_Directory_Path(text),
			Hostname:                  shared_os.Hostname(text),
			Process_Identifier:        process_identifiers[index],
			Effective_User_Identifier: effective_user_identifiers[index],
		}
		host := shared_os.Virtual_OS_To_OS(&virtual)
		argument_destination := make(shared_os.Arguments, byte_size)
		environment_destination := make(shared_os.Environment, byte_size)
		os.OS_Arguments(host, argument_destination)
		os.OS_Environment(host, environment_destination)
		os.OS_Variable(host, shared_os.Variable_Name(text))
		os.OS_Executable(host)
		os.OS_Working_Directory(host)
		os.OS_Hostname(host)
		os.OS_Process_Identifier(host)
		os.OS_Effective_User_Identifier(host)
		os.OS_Self_Exec(
			host, shared_os.Executable_Path(text), arguments, environment,
		)
	}
}

// Host composition and every returned operation must leave heap untouched.
func Test_Operating_System_Heap_Allocation(t *testing.T) {
	var host shared_os.OS
	state := test_host()
	t.Run("New_Operating_System", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			host = os.New_Operating_System(&state)
		})
	})
	arguments := [slices.SLICE_COUNT_MAXIMUM]string{}
	environment := [slices.SLICE_COUNT_MAXIMUM]string{}
	var count shared_os.Entry_Count
	t.Run("Arguments", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			count = os.OS_Arguments(host, arguments[:])
		})
		testify.Positive(t, count)
	})
	t.Run("Environment", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			count = os.OS_Environment(host, environment[:])
		})
		testify.Positive(t, count)
	})
	var text shared_os.Variable_Value
	var found shared_os.Variable_Found
	t.Run("Variable", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			text, found = os.OS_Variable(host, "PATH")
		})
		testify.True(t, bool(found))
		testify.Not_Empty(t, text)
	})
	var path shared_os.Executable_Path
	var operation_err error
	t.Run("Executable", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			path, operation_err = os.OS_Executable(host)
		})
		testify.No_Error(t, operation_err)
		testify.Not_Empty(t, path)
	})
	var directory shared_os.Working_Directory_Path
	t.Run("Working_Directory", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			directory, operation_err = os.OS_Working_Directory(host)
		})
		testify.No_Error(t, operation_err)
		testify.Not_Empty(t, directory)
	})
	var hostname shared_os.Hostname
	t.Run("Hostname", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			hostname, operation_err = os.OS_Hostname(host)
		})
		testify.No_Error(t, operation_err)
		testify.Not_Empty(t, hostname)
	})
	verify_host_identity_allocations(t, host)
	verify_host_self_exec_allocations(t, host, &operation_err)
}

func verify_host_self_exec_allocations(t *testing.T, host shared_os.OS, operation_err *error) {
	t.Helper()
	t.Run("Self_Exec_Ambient", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			*operation_err = os.OS_Self_Exec(host, "", nil, nil)
		})
		testify.Error(t, *operation_err)
	})
	t.Run("Self_Exec_Empty", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			*operation_err = os.OS_Self_Exec(host, "", nil, shared_os.Environment{})
		})
		testify.Error(t, *operation_err)
	})
}

func verify_host_identity_allocations(t *testing.T, host shared_os.OS) {
	t.Helper()
	var process_identifier shared_os.Process_Identifier
	t.Run("Process_Identifier", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			process_identifier = os.OS_Process_Identifier(host)
		})
		testify.Positive(t, process_identifier)
	})
	var user_identifier shared_os.Effective_User_Identifier
	t.Run("Effective_User_Identifier", func(t *testing.T) {
		testify.Zero_Allocation(t, func() {
			user_identifier = os.OS_Effective_User_Identifier(host)
		})
		testify.True(t, user_identifier >= 0)
	})
}

// Nil preserves root-provided ambient environment; explicit empty slice prevents secret leak.
func Test_Operating_System_Self_Exec_Environment(t *testing.T) {
	state := test_host()
	host := os.New_Operating_System(&state)
	err := os.OS_Self_Exec(host, "/missing", nil, nil)
	testify.Error(t, err)
	testify.True(t, (*state.Self_Exec.Environment_Pointers)[0] != nil)
	err = os.OS_Self_Exec(host, "/missing", nil, shared_os.Environment{})
	testify.Error(t, err)
	testify.False(t, (*state.Self_Exec.Environment_Pointers)[0] != nil)
}

func test_host() (host os.Host) {
	return os.Host{
		Arguments:         []string{"os.test"},
		Environment:       syscall.Environ(),
		Executable:        "/bin/true",
		Working_Directory: "/tmp",
		Hostname:          "test",
		Self_Exec:         test_self_exec_workspace(),
	}
}

func test_self_exec_workspace() (workspace *os.Self_Exec_Workspace) {
	path := make(os.Self_Exec_Path_Bytes, os.SELF_EXEC_PATH_STORAGE_BYTES)
	arguments := make(os.Self_Exec_Argument_Bytes, os.SELF_EXEC_VECTOR_STORAGE_BYTES)
	environment := make(os.Self_Exec_Environment_Bytes, os.SELF_EXEC_VECTOR_STORAGE_BYTES)
	argument_pointers := make(
		os.Self_Exec_Argument_Pointer_Vector, os.SELF_EXEC_VECTOR_POINTER_COUNT,
	)
	environment_pointers := make(
		os.Self_Exec_Environment_Pointer_Vector, os.SELF_EXEC_VECTOR_POINTER_COUNT,
	)
	return &os.Self_Exec_Workspace{
		Path:                 os.Self_Exec_Path_Storage(&path),
		Arguments:            os.Self_Exec_Argument_Storage(&arguments),
		Environment:          os.Self_Exec_Environment_Storage(&environment),
		Argument_Pointers:    os.Self_Exec_Argument_Pointers(&argument_pointers),
		Environment_Pointers: os.Self_Exec_Environment_Pointers(&environment_pointers),
	}
}
