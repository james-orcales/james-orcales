package os_test

import (
	"os/exec"
	"syscall"
	"testing"

	"local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/simulation/os/default"
	"local/james-orcales/shared/slices"
	"local/james-orcales/shared/testify"
)

// TestMain registers the package invariant roots before the smoke test runs.
func TestMain(m *testing.M) {
	invariant.Run_Test_Main(m)
}

// Test_Operating_System_Smoke verifies the host answers every ambient reader. The values a
// machine supplies are not deterministic, so this checks their shape, not their content. The
// constructor asserts every slot is filled, so reaching the end proves the vtable is whole.
func Test_Operating_System_Smoke(t *testing.T) {
	host := os.New_Operating_System()
	arguments := [slices.SLICE_COUNT_MAXIMUM]string{}
	testify.Positive(t, os.OS_Arguments(host, arguments[:]))
	testify.Positive(t, os.OS_Process_Identifier(host))
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
	host := os.New_Operating_System()
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
		value, single := os.OS_Variable(host, name)
		if !testify.True(t, single, name) {
			continue
		}
		testify.Equal(t, expected, value, name)
	}
}

// Test_Operating_System_Arguments_Copy verifies caller storage cannot reach runtime-owned argv.
func Test_Operating_System_Arguments_Copy(t *testing.T) {
	host := os.New_Operating_System()
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
	host := os.New_Operating_System()
	testify.Panics(t, func() {
		os.OS_Arguments(host, nil)
	})
	testify.Panics(t, func() {
		os.OS_Environment(host, nil)
	})
}

// Nil preserves root-provided ambient environment; explicit empty slice prevents secret leak.
func Test_Operating_System_Self_Exec_Environment(t *testing.T) {
	const MARKER = "SIMULATION_OS_SELF_EXEC_ENVIRONMENT"
	mode, helper := syscall.Getenv(MARKER)
	if helper {
		host := os.New_Operating_System()
		environment := []string(nil)
		check := "test \"$" + MARKER + "\" = inherit"
		if mode == "empty" {
			environment = []string{}
			check = "test -z \"${" + MARKER + "+x}\""
		}
		err := os.OS_Self_Exec(
			host, "/bin/sh", []string{"sh", "-c", check}, environment,
		)
		t.Fatal(err)
	}

	host := os.New_Operating_System()
	executable, executable_err := os.OS_Executable(host)
	if !testify.No_Error(t, executable_err) {
		return
	}
	for _, case_mode := range []string{"inherit", "empty"} {
		command := exec.Command(
			executable, "-test.run=^Test_Operating_System_Self_Exec_Environment$",
		)
		command.Env = []string{MARKER + "=" + case_mode}
		testify.No_Error(t, command.Run(), case_mode)
	}
}
