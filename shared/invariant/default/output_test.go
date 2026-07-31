package invariant

import (
	"bytes"
	"errors"
	"os"
	"testing"

	core "local/james-orcales/shared/invariant"
)

// Test_JSON_Output_Records keeps branch and reachability gaps in one stable flat schema.
func Test_JSON_Output_Records(t *testing.T) {
	link := uint8(2)
	property := "The value equals the minimum."
	gaps := []core.Coverage_Gap{
		{
			Section: "branch", Assertion: "Classify_File_Input.Path",
			Package: "local/james-orcales/sloc/internal", Type: "File_Path",
			Link:   &link,
			Absent: "true", Property: &property, Source: "len(file_path)",
		},
		// An eager guard owns a unique message, thus it carries no subject identity.
		{
			Section: "reachability", Assertion: "A guard is reached.",
			Absent: "reachability", Source: "ready",
		},
	}
	output := &bytes.Buffer{}
	if err := Coverage_Gap_Json_Write(output, gaps); err != nil {
		t.Fatal(err)
	}
	want := `[{"section":"branch","assertion":"Classify_File_Input.Path",` +
		`"package":"local/james-orcales/sloc/internal","type":"File_Path",` +
		`"link":2,"missing":"true","property":"The value equals the minimum.",` +
		`"source":"len(file_path)"},{"section":"reachability",` +
		`"assertion":"A guard is reached.","package":"","type":"",` +
		`"link":null,"missing":"reachability",` +
		`"property":null,"source":"ready"}]` + "\n"
	if output.String() != want {
		t.Fatalf("output = %q, want %q", output.String(), want)
	}
}

// Test_JSON_Output_Selection keeps unset, table, json, and invalid environment values distinct.
func Test_JSON_Output_Selection(t *testing.T) {
	values := []struct {
		Environment string
		Prefix      string
		Diagnostic  string
	}{
		{"", "🚨 1 coverage gaps 🚨\n", ""},
		{"table", "🚨 1 coverage gaps 🚨\n", ""},
		{"json", "[{\"section\":\"reachability\"", ""},
		{
			"dense", "", "INVARIANT_OUTPUT has unknown value \"dense\"; " +
				"expected \"table\" or \"json\"",
		},
	}
	for _, value := range values {
		t.Setenv(OUTPUT_ENVIRONMENT, value.Environment)
		recorder := &core.Recorder{}
		recorder_output_configure(recorder, os.Getenv(OUTPUT_ENVIRONMENT))
		if recorder.Output_Configuration_Diagnostic != value.Diagnostic {
			t.Fatalf("mode %q diagnostic = %q, want %q", value.Environment,
				recorder.Output_Configuration_Diagnostic, value.Diagnostic)
		}
		if value.Diagnostic != "" {
			continue
		}
		output := &bytes.Buffer{}
		gap := core.Coverage_Gap{
			Section: "reachability", Assertion: "guard",
			Absent: "reachability", Source: "ready",
		}
		gaps := []core.Coverage_Gap{gap}
		if err := recorder.Report_Coverage_Gaps(output, gaps); err != nil {
			t.Fatal(err)
		}
		if !bytes.HasPrefix(output.Bytes(), []byte(value.Prefix)) {
			t.Fatalf("mode %q output = %q, want prefix %q",
				value.Environment, output.String(), value.Prefix)
		}
	}
}

// Test_JSON_Output_Failure keeps both encoder and terminating-newline writes checked.
func Test_JSON_Output_Failure(t *testing.T) {
	gap := core.Coverage_Gap{
		Section: "reachability", Assertion: "guard",
		Absent: "reachability", Source: "ready",
	}
	if err := Coverage_Gap_Json_Write(failure_writer{}, []core.Coverage_Gap{gap}); err == nil {
		t.Fatal("encoder write failure was ignored")
	}
	writer := &newline_failure_writer{}
	if err := Coverage_Gap_Json_Write(writer, []core.Coverage_Gap{gap}); err == nil {
		t.Fatal("terminating newline failure was ignored")
	}
}

type failure_writer struct{}

// Write supplies the injected io.Writer failure the JSON reporter must propagate.
func (failure_writer) Write(data []byte) (written int, err error) {
	return 0, errors.New("write failed")
}

type newline_failure_writer struct {
	Write_Count int
}

// Write accepts the encoded array and rejects only the required newline write.
func (writer *newline_failure_writer) Write(data []byte) (written int, err error) {
	writer.Write_Count++
	if writer.Write_Count == 1 {
		return len(data), nil
	}
	return 0, errors.New("newline failed")
}
