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
		Read_Stdin:         func() (data []byte, err error) { return stdin, nil },
		Read_File:          read_file,
		Stdout_Is_Terminal: false,
		Link:               func(destination string) (err error) { return nil },
	})
	return output_buffer.String(), status
}

// Test_Filter_And_First checks a filter with a word operator, forced to JSON.
func Test_Filter_And_First(t *testing.T) {
	stdin := []byte(`[{"name":"ada","age":36},{"name":"bob","age":19},{"name":"cy","age":51}]`)
	output, code := drive_verb("filter", []string{"age", "gt", "30", "-json"}, stdin)
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
	output, _ := drive_verb("get", []string{"users", "-json"}, stdin)
	want := `[{"name":"ada","age":36},{"name":"bob","age":19}]`
	if output != want {
		t.Fatalf("got %s", output)
	}
}

// Test_Load_Reads_File checks load parsing the injected file.
func Test_Load_Reads_File(t *testing.T) {
	output, _ := drive_verb("load", []string{"users.json", "-json"}, nil)
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
	output, _ := drive_verb("first", []string{"-count=1", "-table"}, stdin)
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

// Test_From_Csv checks parsing CSV stdin into records with inferred numbers.
func Test_From_Csv(t *testing.T) {
	stdin := []byte("name,age\nada,36\nbob,19\n")
	output, _ := drive_verb("from", []string{"csv", "-json"}, stdin)
	want := `[{"name":"ada","age":36},{"name":"bob","age":19}]`
	if output != want {
		t.Fatalf("got %s", output)
	}
}

// Test_To_Csv checks a table decoded from stdin serializing out to CSV text.
func Test_To_Csv(t *testing.T) {
	stdin := []byte(`[{"name":"ada","age":36},{"name":"bob","age":19}]`)
	output, _ := drive_verb("to", []string{"csv"}, stdin)
	want := "name,age\nada,36\nbob,19\n"
	if output != want {
		t.Fatalf("got %q", output)
	}
}

// Test_Csv_Quotes_Commas checks a cell with a comma round-trips through quoting.
func Test_Csv_Quotes_Commas(t *testing.T) {
	stdin := []byte(`[{"city":"Paris, FR"}]`)
	output, _ := drive_verb("to", []string{"csv"}, stdin)
	want := "city\n\"Paris, FR\"\n"
	if output != want {
		t.Fatalf("got %q", output)
	}
}

// Test_Reject_Drops_Field checks reject removing a named field from records.
func Test_Reject_Drops_Field(t *testing.T) {
	stdin := []byte(`[{"name":"ada","age":36}]`)
	output, _ := drive_verb("reject", []string{"age", "-json"}, stdin)
	if output != `[{"name":"ada"}]` {
		t.Fatalf("got %s", output)
	}
}

// Test_Relabel_Field checks relabel renaming a field across records.
func Test_Relabel_Field(t *testing.T) {
	stdin := []byte(`[{"name":"ada"}]`)
	output, _ := drive_verb("relabel", []string{"name", "who", "-json"}, stdin)
	if output != `[{"who":"ada"}]` {
		t.Fatalf("got %s", output)
	}
}

// Test_Reverse_List checks reverse flipping list order.
func Test_Reverse_List(t *testing.T) {
	output, _ := drive_verb("reverse", []string{"-json"}, []byte(`[1,2,3]`))
	if output != `[3,2,1]` {
		t.Fatalf("got %s", output)
	}
}

// Test_Final_Tail checks final taking the last n as a list.
func Test_Final_Tail(t *testing.T) {
	output, _ := drive_verb("final", []string{"-count=2", "-json"}, []byte(`[1,2,3,4]`))
	if output != `[3,4]` {
		t.Fatalf("got %s", output)
	}
}

// Test_Columns_Keys checks columns listing a table's field names.
func Test_Columns_Keys(t *testing.T) {
	stdin := []byte(`[{"name":"ada","age":36}]`)
	output, _ := drive_verb("columns", []string{"-json"}, stdin)
	if output != `["name","age"]` {
		t.Fatalf("got %s", output)
	}
}

// Test_Wrap_And_Flatten checks wrap boxing a value and flatten unnesting a level.
func Test_Wrap_And_Flatten(t *testing.T) {
	wrapped, _ := drive_verb("wrap", []string{"items", "-json"}, []byte(`[1,2]`))
	if wrapped != `{"items":[1,2]}` {
		t.Fatalf("wrap got %s", wrapped)
	}
	flat, _ := drive_verb("flatten", []string{"-json"}, []byte(`[[1,2],[3]]`))
	if flat != `[1,2,3]` {
		t.Fatalf("flatten got %s", flat)
	}
}

// Test_Unknown_Flag_Suggests checks a mistyped flag is a usage error that suggests
// the intended flag, the behavior shell_sdk inherits from cli.
func Test_Unknown_Flag_Suggests(t *testing.T) {
	output := strings.Builder{}
	problems := strings.Builder{}
	code := shell_sdk.Main(&shell_sdk.Main_Input{
		Arguments:    []string{"filter", "age", "gt", "30", "-jsonn"},
		Output:       &output,
		Error_Output: &problems,
		Read_Stdin:   func() (data []byte, err error) { return nil, nil },
		Read_File:    func(name string) (data []byte, err error) { return nil, nil },
	})
	if code != 2 {
		t.Fatalf("an unknown flag should exit 2, got %d", code)
	}
	if !strings.Contains(problems.String(), "did you mean -json") {
		t.Fatalf("expected a suggestion, got %q", problems.String())
	}
}

// Test_Install_Links verifies the install verb, self-dispatched on the bare binary,
// symlinks every data verb into the destination — but not install itself, which would
// shadow the system install(1).
func Test_Install_Links(t *testing.T) {
	output := strings.Builder{}
	problems := strings.Builder{}
	links := []string{}
	code := shell_sdk.Main(&shell_sdk.Main_Input{
		Arguments:    []string{"shell_sdk", "install", "/opt/bin"},
		Output:       &output,
		Error_Output: &problems,
		Read_Stdin:   func() (data []byte, err error) { return nil, nil },
		Read_File:    func(name string) (data []byte, err error) { return nil, nil },
		Link: func(destination string) (err error) {
			links = append(links, destination)
			return nil
		},
	})
	if code != 0 {
		t.Fatalf("install should exit 0, got %d (%s)", code, problems.String())
	}
	found_data_verb := false
	for _, link := range links {
		if strings.HasSuffix(link, "/install") {
			t.Errorf("install must not link itself: %q", link)
		}
		if strings.HasSuffix(link, "/from") {
			found_data_verb = true
		}
	}
	if !found_data_verb {
		t.Errorf("expected a data verb like /opt/bin/from among %v", links)
	}
}

// Test_From_Rejects_Unknown_Format checks that an out-of-set format is a usage error
// (exit 2) rejected at parse time, not a runtime failure (exit 1).
func Test_From_Rejects_Unknown_Format(t *testing.T) {
	_, code := drive_verb("from", []string{"xml"}, []byte("{}"))
	if code != 2 {
		t.Fatalf("an unknown format should exit 2, got %d", code)
	}
}

// Test_Filter_Rejects_Unknown_Operator checks that an out-of-set operator is a usage
// error (exit 2) rejected at parse time, not a runtime failure (exit 1).
func Test_Filter_Rejects_Unknown_Operator(t *testing.T) {
	stdin := []byte(`[{"age":36}]`)
	_, code := drive_verb("filter", []string{"age", "bogus", "30"}, stdin)
	if code != 2 {
		t.Fatalf("an unknown operator should exit 2, got %d", code)
	}
}

// Test_From_Format_Enum_Message checks the enum error lists the permitted formats.
func Test_From_Format_Enum_Message(t *testing.T) {
	output := strings.Builder{}
	problems := strings.Builder{}
	code := shell_sdk.Main(&shell_sdk.Main_Input{
		Arguments:    []string{"from", "xml"},
		Output:       &output,
		Error_Output: &problems,
		Read_Stdin:   func() (data []byte, err error) { return nil, nil },
		Read_File:    func(name string) (data []byte, err error) { return nil, nil },
	})
	if code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
	if !strings.Contains(problems.String(), "allowed: json, csv") {
		t.Fatalf("expected the allowed formats, got %q", problems.String())
	}
}
