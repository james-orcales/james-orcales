package os_test

import process "os"
import "os/exec"
import "syscall"
import "testing"

import "local/james-orcales/shared/math/bits"
import shared_os "local/james-orcales/shared/os"
import "local/james-orcales/shared/os/default"
import "local/james-orcales/shared/slices"
import "local/james-orcales/shared/testify"

// Test_Operating_System_Smoke verifies the host answers every ambient reader. The values a
// machine supplies are not deterministic, so this checks their shape, not their content. The
// constructor asserts every slot is filled, so reaching the end proves the vtable is whole.
func test_operating_system_smoke(t *testing.T) {
  state := test_host()
  host := os.New_Operating_System(&state)
  arguments := [slices.SLICE_COUNT_MAXIMUM]string{}
  testify.Positive(t, os.OS_Arguments(host, arguments[:]))
  testify.Positive(t, os.OS_Process_Identifier(host))
  testify.True(t, !(os.OS_Effective_User_Identifier(host) < 0))
  directory, directory_err := os.OS_Working_Directory(host)
  testify.No_Error(t, directory_err)
  if testify.True(t, len(directory) > 0, directory) {
    testify.True(t, directory[0] == '/', directory)
  }
  executable, executable_err := os.OS_Executable(host)
  testify.No_Error(t, executable_err)
  testify.Not_Empty(t, executable)
  name, name_err := os.OS_Hostname(host)
  testify.No_Error(t, name_err)
  testify.Not_Empty(t, name)
}

// Test_Operating_System_Environment verifies Environment and Variable agree, because they reach
// the kernel by different calls: Environment reads the whole block and Variable reads one name.
func test_operating_system_environment(t *testing.T) {
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
    if !testify.True(t, !(separator < 0), variable) {
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

// Test_Operating_System_Arguments_Live verifies argv comes from the process itself, so package
// main needs no raw stdlib import to read it.
func test_operating_system_arguments_live(t *testing.T) {
  state := test_host()
  host := os.New_Operating_System(&state)
  storage := [slices.SLICE_COUNT_MAXIMUM]string{}
  count := os.OS_Arguments(host, storage[:])
  testify.Equal(t, process.Args, []string(storage[:count]))
}

// Test_Operating_System_Environment_Live verifies a variable set after construction is visible,
// which a construction-time capture would miss.
func test_operating_system_environment_live(t *testing.T) {
  state := test_host()
  host := os.New_Operating_System(&state)
  t.Setenv("OS_DEFAULT_ENVIRONMENT_LIVE", "set")
  storage := [slices.SLICE_COUNT_MAXIMUM]string{}
  count := os.OS_Environment(host, storage[:])
  testify.Contains(t, []string(storage[:count]), "OS_DEFAULT_ENVIRONMENT_LIVE=set")
}

// TEST_EXIT_CODE is a status no test binary exits with on its own.
const TEST_EXIT_CODE = 37

// TEST_EXIT_VARIABLE marks the child run that exits instead of testing.
const TEST_EXIT_VARIABLE = "OS_DEFAULT_TEST_EXIT"

// Test_Operating_System_Exit verifies the process ends with the given status. Exit ends the
// test binary, so a child run of this test performs it and the parent reads the status.
func test_operating_system_exit(t *testing.T) {
  state := test_host()
  host := os.New_Operating_System(&state)
  if _, child := syscall.Getenv(TEST_EXIT_VARIABLE); child {
    os.OS_Exit(host, TEST_EXIT_CODE)
    return
  }
  command := exec.Command(process.Args[0], "-test.run=^Test_Exit$")
  command.Env = append(syscall.Environ(), TEST_EXIT_VARIABLE+"=1")
  run_err := command.Run()
  testify.Error(t, run_err)
  testify.Equal(t, TEST_EXIT_CODE, command.ProcessState.ExitCode())
}

// Test_Operating_System_Arguments_Copy verifies caller storage cannot reach runtime-owned argv.
func test_operating_system_arguments_copy(t *testing.T) {
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
func test_operating_system_destination_capacity(t *testing.T) {
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
func test_operating_system_invariant_boundaries(t *testing.T) {
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
func test_operating_system_heap_allocation(t *testing.T) {
  var host shared_os.OS
  state := test_host()
  t.Run("New_Operating_System", func(t *testing.T) {
    testify.Zero_Allocation(t, func() {
      allocation_host(&host, &state)
    })
  })
  arguments := [slices.SLICE_COUNT_MAXIMUM]string{}
  environment := [slices.SLICE_COUNT_MAXIMUM]string{}
  var count shared_os.Entry_Count
  t.Run("Arguments", func(t *testing.T) {
    testify.Zero_Allocation(t, func() {
      allocation_arguments(&count, host, arguments[:])
    })
    testify.Positive(t, count)
  })
  // Environment is exempt: the runtime exposes the live block only as an allocated copy.
  count = os.OS_Environment(host, environment[:])
  testify.Positive(t, count)
  var text shared_os.Variable_Value
  var found shared_os.Variable_Found
  t.Run("Variable", func(t *testing.T) {
    testify.Zero_Allocation(t, func() {
      allocation_variable(&text, &found, host)
    })
    testify.True(t, bool(found))
    testify.Not_Empty(t, text)
  })
  var path shared_os.Executable_Path
  var operation_err error
  t.Run("Executable", func(t *testing.T) {
    testify.Zero_Allocation(t, func() {
      allocation_executable(&path, &operation_err, host)
    })
    testify.No_Error(t, operation_err)
    testify.Not_Empty(t, path)
  })
  var directory shared_os.Working_Directory_Path
  t.Run("Working_Directory", func(t *testing.T) {
    testify.Zero_Allocation(t, func() {
      allocation_directory(&directory, &operation_err, host)
    })
    testify.No_Error(t, operation_err)
    testify.Not_Empty(t, directory)
  })
  var hostname shared_os.Hostname
  t.Run("Hostname", func(t *testing.T) {
    testify.Zero_Allocation(t, func() {
      allocation_hostname(&hostname, &operation_err, host)
    })
    testify.No_Error(t, operation_err)
    testify.Not_Empty(t, hostname)
  })
  verify_host_identity_allocations(t, host)
  verify_host_self_exec_allocations(t, host, &operation_err)
}

func verify_host_self_exec_allocations(t *testing.T, host shared_os.OS, operation_err *error) {
  t.Helper()
  // A nil environment is exempt: it reads the live block, which the runtime only copies out.
  *operation_err = os.OS_Self_Exec(host, "", nil, nil)
  testify.Error(t, *operation_err)
  t.Run("Self_Exec_Empty", func(t *testing.T) {
    testify.Zero_Allocation(t, func() {
      allocation_self_exec(operation_err, host)
    })
    testify.Error(t, *operation_err)
  })
}

func verify_host_identity_allocations(t *testing.T, host shared_os.OS) {
  t.Helper()
  var process_identifier shared_os.Process_Identifier
  t.Run("Process_Identifier", func(t *testing.T) {
    testify.Zero_Allocation(t, func() {
      allocation_process_identifier(&process_identifier, host)
    })
    testify.Positive(t, process_identifier)
  })
  var user_identifier shared_os.Effective_User_Identifier
  t.Run("Effective_User_Identifier", func(t *testing.T) {
    testify.Zero_Allocation(t, func() {
      allocation_user_identifier(&user_identifier, host)
    })
    testify.True(t, !(user_identifier < 0))
  })
}

// Nil preserves root-provided ambient environment; explicit empty slice prevents secret leak.
func test_operating_system_self_exec_environment(t *testing.T) {
  state := test_host()
  host := os.New_Operating_System(&state)
  err := os.OS_Self_Exec(host, "/missing", nil, nil)
  testify.Error(t, err)
  testify.True(t, (*state.Self_Exec.Environment_Pointers)[0] != nil)
  err = os.OS_Self_Exec(host, "/missing", nil, shared_os.Environment{})
  testify.Error(t, err)
  testify.False(t, (*state.Self_Exec.Environment_Pointers)[0] != nil)
}

// Test_Operating_System_Memory verifies kernel region is zeroed, writable to its last byte, and
// unmaps without error, and empty region unmaps as no-op.
func test_operating_system_memory(t *testing.T) {
  state := test_host()
  host := os.New_Operating_System(&state)
  testify.No_Error(t, os.OS_Memory_Unmap(host, nil))
  for _, memory_size := range [...]shared_os.Memory_Size{
    shared_os.MEMORY_SIZE_MINIMUM,
    shared_os.MEMORY_SIZE_MAXIMUM,
  } {
    region, map_err := os.OS_Memory_Map(host, memory_size)
    if !testify.No_Error(t, map_err) {
      continue
    }
    testify.Equal(t, int(memory_size), len(region))
    testify.Equal(t, int(memory_size), cap(region))
    testify.Zero(t, region[len(region)-1])
    region[len(region)-1] = 1
    testify.No_Error(t, os.OS_Memory_Unmap(host, region))
  }
}

// Kernel map and unmap leave Go heap untouched.
func test_operating_system_memory_heap_allocation(t *testing.T) {
  state := test_host()
  host := os.New_Operating_System(&state)
  var region shared_os.Memory
  var operation_err error
  testify.Zero_Allocation(t, func() {
    allocation_memory(&region, &operation_err, host)
  })
  testify.No_Error(t, operation_err)
}

func test_host() (host os.Host) {
  return os.Host{
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
    os.Self_Exec_Environment_Pointer_Vector,
    os.SELF_EXEC_VECTOR_POINTER_COUNT,
  )
  return &os.Self_Exec_Workspace{
    Path:                 os.Self_Exec_Path_Storage(&path),
    Arguments:            os.Self_Exec_Argument_Storage(&arguments),
    Environment:          os.Self_Exec_Environment_Storage(&environment),
    Argument_Pointers:    os.Self_Exec_Argument_Pointers(&argument_pointers),
    Environment_Pointers: os.Self_Exec_Environment_Pointers(&environment_pointers),
  }
}

// FILE_TEST_PERMISSION is owner read and write, the mode a rewritten source file keeps.
const FILE_TEST_PERMISSION os.File_Permission = 0o600

// Test_File_Round_Trip covers File_Write creating a file with the given content and
// File_Read_Into returning exactly those bytes, then a rewrite replacing rather than appending.
func test_file_round_trip(t *testing.T) {
  path := os.File_Path(t.TempDir() + "/round_trip.txt")
  const STORAGE_BYTES = 32
  var storage [STORAGE_BYTES]byte
  first := os.File_Content("first content\n")
  testify.No_Error(t, os.File_Write(path, first, FILE_TEST_PERMISSION))
  read, read_err := os.File_Read_Into(path, storage[:])
  testify.No_Error(t, read_err)
  testify.Equal(t, string(first), string(read))

  second := os.File_Content("second\n")
  testify.No_Error(t, os.File_Write(path, second, FILE_TEST_PERMISSION))
  read, read_err = os.File_Read_Into(path, storage[:])
  testify.No_Error(t, read_err)
  testify.Equal(t, string(second), string(read))
}

// Test_File_Read_Into covers reading into caller storage: content lands in destination, and a
// file larger than destination reports File_Too_Large instead of a partial read.
func test_file_read_into(t *testing.T) {
  path := os.File_Path(t.TempDir() + "/into.txt")
  content := os.File_Content("into content\n")
  testify.No_Error(t, os.File_Write(path, content, FILE_TEST_PERMISSION))
  const STORAGE_BYTES = 32
  const SMALL_BYTES = 4
  var storage [STORAGE_BYTES]byte
  read, read_err := os.File_Read_Into(path, storage[:])
  testify.No_Error(t, read_err)
  testify.Equal(t, string(content), string(read))
  testify.Equal(t, &storage[0], &read[0])
  var small [SMALL_BYTES]byte
  read, read_err = os.File_Read_Into(path, small[:])
  testify.Error_Is(t, read_err, os.File_Too_Large)
  testify.Equal(t, 0, len(read))
}

// Test_File_Read_Absent covers a missing path reporting the kernel's error and no content.
func test_file_read_absent(t *testing.T) {
  const STORAGE_BYTES = 32
  var storage [STORAGE_BYTES]byte
  read, read_err := os.File_Read_Into(os.File_Path(t.TempDir()+"/absent.txt"), storage[:])
  if read_err == nil {
    t.Fatal("reading an absent file must fail")
  }
  testify.Equal(t, 0, len(read))
}

// Test_Standard_Error_Write covers the diagnostics path accepting a line; its bytes land on the
// test binary's own stderr.
func test_standard_error_write(t *testing.T) {
  testify.No_Error(t, os.Standard_Error_Write(os.File_Content("os/default: stderr write\n")))
}

// Test_Machine_Read verifies the host describes itself. Values differ per machine, thus this checks
// that every field a supported host exposes is present, not what it holds.
func test_machine_read(t *testing.T) {
  machine := os.Machine_Read()
  testify.Not_Empty(t, machine.Processor_Architecture)
  testify.Positive(t, machine.Physical_Core_Count)
  testify.Positive(t, machine.Logical_Core_Count)
  testify.True(t, !(int(machine.Physical_Core_Count) > int(machine.Logical_Core_Count)))
  testify.Positive(t, machine.Physical_Memory_Size)
  testify.Positive(t, machine.Storage_Size)
  testify.Not_Empty(t, machine.Operating_System_Name)
  testify.Not_Empty(t, machine.Kernel_Version)
  if machine.Operating_System_Name == "macOS" {
    testify.Not_Empty(t, machine.Processor_Model)
    testify.Not_Empty(t, machine.Operating_System_Version)
    testify.Positive(t, machine.Level_1_Cache_Size)
  }
}

// PIPE_END_COUNT is a pipe's read end and write end.
const PIPE_END_COUNT = 2

// Test_Descriptor_Terminal verifies a pipe end is not a terminal.
func test_descriptor_terminal(t *testing.T) {
  var ends [PIPE_END_COUNT]int
  testify.No_Error(t, syscall.Pipe(ends[:]))
  defer syscall.Close(ends[0])
  defer syscall.Close(ends[1])
  testify.False(t, bool(os.Descriptor_Terminal(os.Descriptor(ends[1]))))
}

func allocation_host(host *shared_os.OS, state *os.Host) {
  *host = os.New_Operating_System(state)
}

func allocation_arguments(count *shared_os.Entry_Count, host shared_os.OS, storage []string) {
  *count = os.OS_Arguments(host, storage)
}

func allocation_self_exec(operation_err *error, host shared_os.OS) {
  *operation_err = os.OS_Self_Exec(host, "", nil, shared_os.Environment{})
}

func allocation_process_identifier(identifier *shared_os.Process_Identifier, host shared_os.OS) {
  *identifier = os.OS_Process_Identifier(host)
}

func allocation_user_identifier(
  identifier *shared_os.Effective_User_Identifier,
  host shared_os.OS,
) {
  *identifier = os.OS_Effective_User_Identifier(host)
}

func allocation_variable(
  text *shared_os.Variable_Value,
  found *shared_os.Variable_Found,
  host shared_os.OS,
) {
  *text, *found = os.OS_Variable(host, "PATH")
}

func allocation_executable(
  path *shared_os.Executable_Path,
  operation_err *error,
  host shared_os.OS,
) {
  *path, *operation_err = os.OS_Executable(host)
}

func allocation_directory(
  path *shared_os.Working_Directory_Path,
  operation_err *error,
  host shared_os.OS,
) {
  *path, *operation_err = os.OS_Working_Directory(host)
}

func allocation_hostname(name *shared_os.Hostname, operation_err *error, host shared_os.OS) {
  *name, *operation_err = os.OS_Hostname(host)
}

func allocation_memory(region *shared_os.Memory, operation_err *error, host shared_os.OS) {
  *region, *operation_err = os.OS_Memory_Map(host, shared_os.MEMORY_SIZE_MINIMUM)
  if *operation_err == nil {
    *operation_err = os.OS_Memory_Unmap(host, *region)
  }
}
