package os_test

import (
	"strings"
	"testing"

	invariant "local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/os"
	system_os "local/james-orcales/shared/os/default"
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
	if arguments := host.Arguments(); len(arguments) == 0 {
		t.Error("arguments are empty; the test binary is always argv[0]")
	}
	if identifier := host.Identifier(); identifier <= 0 {
		t.Errorf("identifier = %d, want a positive process id", identifier)
	}
	directory, directory_err := host.Working_Directory()
	if directory_err != nil {
		t.Errorf("working directory: %v", directory_err)
	}
	if !strings.HasPrefix(directory, "/") {
		t.Errorf("working directory = %q, want an absolute path", directory)
	}
	executable, executable_err := host.Executable()
	if executable_err != nil {
		t.Errorf("executable: %v", executable_err)
	}
	if executable == "" {
		t.Error("executable is empty")
	}
	name, name_err := host.Hostname()
	if name_err != nil {
		t.Errorf("hostname: %v", name_err)
	}
	if name == "" {
		t.Error("hostname is empty; both platforms name the machine")
	}
}

// Test_Operating_System_Environment verifies Environment and Variable agree, because they reach
// the kernel by different calls: Environment reads the whole block and Variable reads one name.
func Test_Operating_System_Environment(t *testing.T) {
	host := system_os.New_Operating_System()
	variables := host.Environment()
	if len(variables) == 0 {
		t.Fatal("the environment is empty; the test harness always sets PATH")
	}
	for _, variable := range variables {
		name, _, split := strings.Cut(variable, "=")
		if !split {
			t.Fatalf("the entry %q carries no separator", variable)
		}
		want, found := os.Environment_Lookup(variables, name)
		if !found {
			t.Fatalf("the name %q is absent from the block that listed it", name)
		}
		value, single := host.Variable(name)
		if !single {
			t.Fatalf("Variable did not find %q, which Environment listed", name)
		}
		if value != want {
			t.Fatalf("Variable(%q) = %q, want %q", name, value, want)
		}
	}
}

// Test_Operating_System_Arguments_Copy verifies a caller editing the returned argv cannot reach
// the runtime's own slice, which every later reader shares.
func Test_Operating_System_Arguments_Copy(t *testing.T) {
	host := system_os.New_Operating_System()
	first := host.Arguments()
	original := first[0]
	first[0] = "edited"
	if second := host.Arguments(); second[0] != original {
		t.Fatalf("a later read = %q, want the edit not to reach it", second[0])
	}
}
