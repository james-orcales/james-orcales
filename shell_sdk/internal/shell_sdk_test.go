package shell_sdk_test

import (
	"strings"
	"testing"

	shell_sdk "local/james-orcales/shell_sdk/internal"
)

// The JSON the fake filesystem serves for any load path.
func sample_file() (data []byte) {
	return []byte(`{"users":[{"name":"ada","age":36},{"name":"bob","age":19}]}`)
}

// Drives Main with argv[0] set to the verb name, a fixed stdin, and a non-terminal
// stdout, returning the written output and the exit code.
func drive_verb(name string, arguments []string, stdin []byte) (output string, code int) {
	output_buffer := strings.Builder{}
	problems := strings.Builder{}
	argv := append([]string{name}, arguments...)
	read_file := func(file_name string) (data []byte, err error) {
		return sample_file(), nil
	}
	status := shell_sdk.Main(&shell_sdk.Main_Input{
		Arguments:          argv,
		Output:             &output_buffer,
		Error_Output:       &problems,
		Read_Stdin:         func() (data []byte) { return stdin },
		Read_File:          read_file,
		Stdout_Is_Terminal: false,
		Link:               func(destination string) (err error) { return nil },
	})
	return output_buffer.String(), status
}

// Test_Filter_And_First checks a filter with a word operator, forced to JSON.
func Test_Filter_And_First(t *testing.T) {
	stdin := []byte(`[{"name":"ada","age":36},{"name":"bob","age":19},{"name":"cy","age":51}]`)
	output, code := drive_verb("filter", []string{"age", "gt", "30", "--json"}, stdin)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	want := `[{"name":"ada","age":36},{"name":"cy","age":51}]`
	if output != want {
		t.Fatalf("got %s want %s", output, want)
	}
}

// Test_Get_Column checks navigating a dotted path to a column.
func Test_Get_Column(t *testing.T) {
	stdin := []byte(`{"users":[{"name":"ada","age":36},{"name":"bob","age":19}]}`)
	output, _ := drive_verb("get", []string{"users", "--json"}, stdin)
	want := `[{"name":"ada","age":36},{"name":"bob","age":19}]`
	if output != want {
		t.Fatalf("got %s", output)
	}
}

// Test_Load_Reads_File checks load parsing the injected file.
func Test_Load_Reads_File(t *testing.T) {
	output, _ := drive_verb("load", []string{"users.json", "--json"}, nil)
	if !strings.Contains(output, `"users"`) {
		t.Fatalf("got %s", output)
	}
}

// Test_Piped_Output_Is_Wire checks a piped tail emits the binary wire format.
func Test_Piped_Output_Is_Wire(t *testing.T) {
	output, _ := drive_verb("get", []string{"users"}, sample_file())
	if !strings.HasPrefix(output, "SSDK") {
		t.Fatalf("piped output should be wire, got %q", output)
	}
}

// Test_Table_Mode checks the --table mode renders an aligned header.
func Test_Table_Mode(t *testing.T) {
	stdin := []byte(`[{"name":"ada","age":36}]`)
	output, _ := drive_verb("first", []string{"1", "--table"}, stdin)
	if !strings.Contains(output, "name | age") {
		t.Fatalf("expected a table header, got %s", output)
	}
}

// Test_Unknown_Verb_Fails checks an unrecognized name is a usage error.
func Test_Unknown_Verb_Fails(t *testing.T) {
	_, code := drive_verb("bogus", nil, nil)
	if code != 2 {
		t.Fatalf("unknown verb should exit 2, got %d", code)
	}
}
