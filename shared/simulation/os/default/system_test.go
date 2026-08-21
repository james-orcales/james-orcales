package os_test

import (
	"strings"
	"testing"

	invariant "local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/simulation/os"
	system_os "local/james-orcales/shared/simulation/os/default"
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
	host := system_os.New_Operating_System()
	testify.Not_Empty(t, host.Arguments())
	testify.Positive(t, host.Process_Identifier())
	directory, directory_err := host.Working_Directory()
	testify.No_Error(t, directory_err)
	testify.True(t, strings.HasPrefix(directory, "/"), directory)
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
	host := system_os.New_Operating_System()
	variables := host.Environment()
	if !testify.Not_Empty(t, variables) {
		return
	}
	for _, variable := range variables {
		name, _, split := strings.Cut(variable, "=")
		if !testify.True(t, split, variable) {
			continue
		}
		want, found := os.Environment_Lookup(variables, name)
		if !testify.True(t, found, name) {
			continue
		}
		value, single := host.Variable(name)
		if !testify.True(t, single, name) {
			continue
		}
		testify.Equal(t, want, value, name)
	}
}

// Test_Operating_System_Arguments_Copy verifies a caller editing the returned argv cannot reach
// the runtime's own slice, which every later reader shares.
func Test_Operating_System_Arguments_Copy(t *testing.T) {
	host := system_os.New_Operating_System()
	first := host.Arguments()
	original := first[0]
	first[0] = "edited"
	second := host.Arguments()
	testify.Equal(t, original, second[0])
}
