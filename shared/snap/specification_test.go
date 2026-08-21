package snap_test

import (
	"errors"
	"testing"
	"unsafe"

	"local/james-orcales/shared/sim/aver"
	"local/james-orcales/shared/snap"
)

// Test_Equal_Match verifies a matching value returns true and emits no output.
func Test_Equal_Match(t *testing.T) {
	s, output_buffer, _ := test_snapper(nil)
	snapshot := snap.New_Snapshot(&snap.New_Snapshot_Input{
		Snapper:   s,
		File_Path: "/fake/test.go",
		Line:      5,
		Expected:  "hello",
	})
	if !snap.Snapshot_Is_Equal(snapshot, "hello") {
		t.Fatal("expected Snapshot_Is_Equal to return true for matching strings")
	}
	if snap.Buffer_Size(output_buffer) != 0 {
		t.Fatalf(
			"expected no diagnostic output, got: %s", snap.Buffer_String(output_buffer),
		)
	}
}

// Test_Equal_Mismatch verifies a differing value returns false and prints a diff.
func Test_Equal_Mismatch(t *testing.T) {
	s, output_buffer, _ := test_snapper(nil)
	snapshot := snap.New_Snapshot(&snap.New_Snapshot_Input{
		Snapper:   s,
		File_Path: "/fake/test.go",
		Line:      5,
		Expected:  "expected",
	})
	if snap.Snapshot_Is_Equal(snapshot, "actual") {
		t.Fatal("expected Snapshot_Is_Equal to return false for mismatching strings")
	}
	if !contains(snap.Buffer_String(output_buffer), "Snapshot mismatch") {
		t.Fatalf(
			"expected mismatch header in output, got: %s",
			snap.Buffer_String(output_buffer),
		)
	}
}

// Test_Equal_Legend verifies the mismatch header carries a colored
// expected-vs-actual legend so readers can map - / + to red / green.
func Test_Equal_Legend(t *testing.T) {
	s, output_buffer, _ := test_snapper(nil)
	snapshot := snap.New_Snapshot(&snap.New_Snapshot_Input{
		Snapper:   s,
		File_Path: "/fake/test.go",
		Line:      5,
		Expected:  "expected",
	})
	snap.Snapshot_Is_Equal(snapshot, "actual")

	output := snap.Buffer_String(output_buffer)
	if !contains(output, "\033[31mexpected\033[0m") {
		t.Fatalf("expected red 'expected' in legend, got:\n%s", output)
	}
	if !contains(output, "\033[32mactual\033[0m") {
		t.Fatalf("expected green 'actual' in legend, got:\n%s", output)
	}
}

// Test_Edit_Rewrite verifies Should_Edit writes the updated source content to W
// and prints an UPDATED SNAPSHOT notice to Output.
func Test_Edit_Rewrite(t *testing.T) {
	// Line 5 contains a snap.Edit call — the line-finder scans newlines to locate it.
	source := "package foo\n\nfunc F() {}\n\nvar _ = snap.Edit(`old`)\n"
	s, output_buffer, w_buffer := test_snapper(map[string]string{
		"fake/test.go": source,
	})
	snapshot := snap.New_Snapshot(&snap.New_Snapshot_Input{
		Snapper:     s,
		File_Path:   "/fake/test.go",
		Line:        5,
		Expected:    "old",
		Should_Edit: true,
	})
	if !snap.Snapshot_Is_Equal(snapshot, "new") {
		t.Fatal("expected Snapshot_Is_Equal with Should_Edit=true to return true")
	}
	if !contains(snap.Buffer_String(output_buffer), "UPDATED SNAPSHOT") {
		t.Fatalf(
			"expected UPDATED SNAPSHOT notice, got: %s",
			snap.Buffer_String(output_buffer),
		)
	}
	if !contains(snap.Buffer_String(w_buffer), "snap.Init(`new`)") {
		t.Fatalf(
			"expected W to contain snap.Init(`new`), got: %s",
			snap.Buffer_String(w_buffer),
		)
	}
}

// Test_Edit_Line_Delta verifies a multi-line replacement records the correct
// line delta in Snapper.Edits for use by subsequent edits in the same file.
func Test_Edit_Line_Delta(t *testing.T) {
	source := "package foo\n\n\nvar a = snap.Edit(`x`)\nvar b = snap.Edit(`y`)\n"
	s, _, _ := test_snapper(map[string]string{
		"fake/test.go": source,
	})
	snapshot := snap.New_Snapshot(&snap.New_Snapshot_Input{
		Snapper:     s,
		File_Path:   "/fake/test.go",
		Line:        4,
		Expected:    "x",
		Should_Edit: true,
	})
	// Replace "x" with "x\nexpanded": new content has one more newline, delta=1.
	snap.Snapshot_Is_Equal(snapshot, "x\nexpanded")
	edits := s.Edits["/fake/test.go"]
	if len(edits) != 1 {
		t.Fatalf("expected 1 edit recorded, got %d", len(edits))
	}
	if edits[0].Delta != 1 {
		t.Fatalf("expected delta=1, got %d", edits[0].Delta)
	}
}

// Test_Panic_Match verifies Expect_Panic passes when the panic message matches.
func Test_Panic_Match(t *testing.T) {
	s, _, _ := test_snapper(nil)
	snapshot := snap.New_Snapshot(&snap.New_Snapshot_Input{
		Snapper:   s,
		File_Path: "/fake/test.go",
		Line:      5,
		Expected: aver.ASSERTION_FAILURE_MESSAGE_PREFIX +
			"Snapshot_Is_Equal snapshot is bound to a Snapper" +
			"  Always — condition was false: false",
	})
	snap.Expect_Panic(t, snapshot, func() {
		snap.Snapshot_Is_Equal(snap.Snapshot{}, "")
	})
}

// Test_Panic_Mismatch verifies a panic message differing from the snapshot is
// reported as a failed comparison.
func Test_Panic_Mismatch(t *testing.T) {
	s, _, _ := test_snapper(nil)
	snapshot := snap.New_Snapshot(&snap.New_Snapshot_Input{
		Snapper:   s,
		File_Path: "/fake/test.go",
		Line:      5,
		Expected:  "boom",
	})
	if snap.Snapshot_Is_Equal(snapshot, "not boom") {
		t.Fatal("expected Snapshot_Is_Equal to return false for mismatching panic messages")
	}
}

// Test_Batch_Expect verifies Batch_Expect runs every entry as its own subtest.
func Test_Batch_Expect(t *testing.T) {
	s, _, _ := test_snapper(nil)
	entries := []snap.Entry{
		{
			Name:  "upper",
			Input: "hello",
			Snapshot: snap.New_Snapshot(&snap.New_Snapshot_Input{
				Snapper:   s,
				File_Path: "/fake/test.go",
				Line:      5,
				Expected:  "HELLO",
			}),
		},
	}
	snap.Batch_Expect(t, func(in string) (result any) {
		return upper(in)
	}, entries)
}

// Test_Run_Capture verifies Run captures the callback's output and asserts it
// against the snapshot.
func Test_Run_Capture(t *testing.T) {
	s, _, _ := test_snapper(nil)
	// Fprintln writes "hello\n"; Run wraps it as "\nSTDOUT:\nhello\n\n".
	snapshot := snap.New_Snapshot(&snap.New_Snapshot_Input{
		Snapper:   s,
		File_Path: "/fake/test.go",
		Line:      5,
		Expected:  "\nSTDOUT:\nhello\n\n",
	})
	snap.Run(t, func() {
		snap.Buffer_Write(s.Stdout, snap.Data("hello\n"))
	}, snapshot)
}

// Builds a Snapper backed by an in-memory file system and capture buffers.
func test_snapper(source map[string]string) (
	s *snap.Snapper, output_buffer *snap.Buffer, w_buffer *snap.Buffer,
) {
	memory_file_system := memory_files(source)
	output_buffer = &snap.Buffer{}
	w_buffer = &snap.Buffer{}
	s = &snap.Snapper{
		File_System_State: unsafe.Pointer(&memory_file_system),
		Read_File:         memory_read,
		Writer_State:      unsafe.Pointer(w_buffer),
		Write:             buffer_write,
		Output_State:      unsafe.Pointer(output_buffer),
		Output_Write:      buffer_write,
		Write_File: func(
			state unsafe.Pointer, path string, data snap.Data,
			permission snap.Permission,
		) (err error) {
			return nil
		},
		Get_Caller: func(skip int) (frame_information snap.Frame_Information, err error) {
			return snap.Frame_Information{}, nil
		},
		Stdout: &snap.Buffer{},
		Stderr: &snap.Buffer{},
		Edits:  make(map[string][]snap.File_Edit),
	}
	return s, output_buffer, w_buffer
}

type memory_files map[string]string

func memory_read(
	state unsafe.Pointer, path string,
) (data snap.Data, err error) {
	content, found := (*(*memory_files)(state))[path]
	if !found {
		return nil, errors.New("missing file")
	}
	return snap.Data(content), nil
}

func buffer_write(
	state unsafe.Pointer, data snap.Data,
) (written snap.Data_Size, err error) {
	return snap.Buffer_Write((*snap.Buffer)(state), data)
}

func contains(value string, sought string) (present bool) {
	if len(sought) == 0 {
		return true
	}
	if len(sought) > len(value) {
		return false
	}
	last_start := len(value) - len(sought)
	for start_index := 0; start_index <= last_start; start_index++ {
		matched := true
		value_index := start_index
		for sought_index := 0; sought_index < len(sought); sought_index++ {
			if value[value_index] != sought[sought_index] {
				matched = false
				break
			}
			value_index++
		}
		if matched {
			return true
		}
	}
	return false
}

func upper(value string) (result string) {
	data := []byte(value)
	for index := 0; index < len(data); index++ {
		if data[index] >= 'a' {
			if data[index] <= 'z' {
				data[index] -= 'a' - 'A'
			}
		}
	}
	return string(data)
}
