package aver

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"local/james-orcales/shared/sim/aver"
)

// OVERFLOW_FIXTURE_BYTES_MAX bounds the fixture read. Two records fit far inside it, thus a short
// read is what proves the whole artifact was taken.
const OVERFLOW_FIXTURE_BYTES_MAX = 4096

// Test_Overflow_File keeps the terminal line pointing at a real artifact: the wired seam writes
// every record it was handed, as JSON, under a name a reader can find.
func Test_Overflow_File(t *testing.T) {
	property := "The value equals the minimum."
	link := uint8(2)
	gaps := []aver.Coverage_Gap{
		{
			Section: "branch", Assertion: "spilled", Package: "pkg", Type: "Value",
			Link: &link, Absent: "true", Property: &property, Source: "int(value)",
		},
		{
			Section: "reachability", Assertion: "A guard is reached.",
			Absent: "reachability", Source: "ready",
		},
	}
	path, write_error := coverage_gap_overflow_write(gaps)
	if write_error != nil {
		t.Fatal(write_error)
	}
	defer os.Remove(path)
	if !strings.HasSuffix(path, ".json") {
		t.Fatalf("path = %q, want a .json artifact", path)
	}
	opened, open_error := os.Open(path)
	if open_error != nil {
		t.Fatal(open_error)
	}
	defer opened.Close()
	content := make([]byte, OVERFLOW_FIXTURE_BYTES_MAX)
	read, read_error := io.ReadFull(
		io.LimitReader(opened, OVERFLOW_FIXTURE_BYTES_MAX), content)
	if !errors.Is(read_error, io.ErrUnexpectedEOF) {
		t.Fatalf("read error = %v, want the fixture to fit its bound", read_error)
	}
	var restored []aver.Coverage_Gap
	if decode_error := json.Unmarshal(content[:read], &restored); decode_error != nil {
		t.Fatal(decode_error)
	}
	if len(restored) != len(gaps) {
		t.Fatalf("restored %d records, want %d", len(restored), len(gaps))
	}
	if restored[0].Assertion != "spilled" {
		t.Fatalf("restored[0] = %q, want the record it was handed",
			restored[0].Assertion)
	}
}

// Test_JSON_Output_Records keeps branch and reachability gaps in one stable flat schema.
func Test_JSON_Output_Records(t *testing.T) {
	link := uint8(2)
	property := "The value equals the minimum."
	gaps := []aver.Coverage_Gap{
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
		`"link":2,"missing":"true","reached":false,` +
		`"property":"The value equals the minimum.",` +
		`"source":"len(file_path)","declared":"","observed":"",` +
		`"observations":null},{"section":"reachability",` +
		`"assertion":"A guard is reached.","package":"","type":"",` +
		`"link":null,"missing":"reachability","reached":false,` +
		`"property":null,"source":"ready","declared":"","observed":"",` +
		`"observations":null}]` + "\n"
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
		recorder := &aver.Recorder{}
		recorder_output_configure(recorder, os.Getenv(OUTPUT_ENVIRONMENT))
		if recorder.Output_Configuration_Diagnostic != value.Diagnostic {
			t.Fatalf("mode %q diagnostic = %q, want %q", value.Environment,
				recorder.Output_Configuration_Diagnostic, value.Diagnostic)
		}
		if value.Diagnostic != "" {
			continue
		}
		output := &bytes.Buffer{}
		gap := aver.Coverage_Gap{
			Section: "reachability", Assertion: "guard",
			Absent: "reachability", Source: "ready",
		}
		gaps := []aver.Coverage_Gap{gap}
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
	gap := aver.Coverage_Gap{
		Section: "reachability", Assertion: "guard",
		Absent: "reachability", Source: "ready",
	}
	if err := Coverage_Gap_Json_Write(
		failure_writer{}, []aver.Coverage_Gap{gap},
	); err == nil {
		t.Fatal("encoder write failure was ignored")
	}
	writer := &newline_failure_writer{}
	if err := Coverage_Gap_Json_Write(writer, []aver.Coverage_Gap{gap}); err == nil {
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
