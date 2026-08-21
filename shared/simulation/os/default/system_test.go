package os_test

import (
	"os/exec"
	"syscall"
	"testing"

	"local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/simulation/os/default"
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
	testify.Not_Empty(t, host.Arguments())
	testify.Positive(t, host.Process_Identifier())
	directory, directory_err := host.Working_Directory()
	testify.No_Error(t, directory_err)
	testify.True(t, len(directory) > 0 && directory[0] == '/', directory)
	executable, executable_err := host.Executable()
	testify.No_Error(t, executable_err)
	testify.Not_Empty(t, executable)
	name, name_err := host.Hostname()
	testify.No_Error(t, name_err)
	testify.Not_Empty(t, name)
}

// Test_Operating_System_Environment verifies Environment and Variable agree, because they reach
// the kernel by different calls: Environment reads the whole block and Variable reads one name.
func Test_Operating_System_Environment(t *testing.T) {
	host := os.New_Operating_System()
	variables := host.Environment()
	if !testify.Not_Empty(t, variables) {
		return
	}
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
		value, single := host.Variable(name)
		if !testify.True(t, single, name) {
			continue
		}
		testify.Equal(t, expected, value, name)
	}
}

// Test_Operating_System_Arguments_Copy verifies a caller editing the returned argv cannot reach
// the runtime's own slice, which every later reader shares.
func Test_Operating_System_Arguments_Copy(t *testing.T) {
	host := os.New_Operating_System()
	first := host.Arguments()
	original := first[0]
	first[0] = "edited"
	second := host.Arguments()
	testify.Equal(t, original, second[0])
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
		err := host.Self_Exec("/bin/sh", []string{"sh", "-c", check}, environment)
		t.Fatal(err)
	}

	host := os.New_Operating_System()
	executable, executable_err := host.Executable()
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
