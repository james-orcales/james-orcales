package snap_test

import (
	"bytes"
	"fmt"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	snap "github.com/james-orcales/james-orcales/golang_snacks/snap/v2"
)

func test_snapper(source map[string]string) (s *snap.Snapper, output_buffer *bytes.Buffer, w_buffer *bytes.Buffer) {
	memory_file_system := fstest.MapFS{}
	for path, content := range source {
		memory_file_system[path] = &fstest.MapFile{Data: []byte(content)}
	}
	output_buffer = &bytes.Buffer{}
	w_buffer = &bytes.Buffer{}
	s = &snap.Snapper{
		File_System: memory_file_system,
		W:           w_buffer,
		Output:      output_buffer,
		Write_File: func(path string, data []byte, perm fs.FileMode) (err error) {
			return nil
		},
		Get_Caller: func(skip int) (frame_information snap.Frame_Information, err error) {
			return snap.Frame_Information{}, nil
		},
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		Edits:  make(map[string][]snap.File_Edit),
	}
	return s, output_buffer, w_buffer
}

// Test_Is_Equal_Match verifies that Is_Equal returns true and emits no output when strings match.
func Test_Is_Equal_Match(t *testing.T) {
	s, output_buffer, _ := test_snapper(nil)
	snapshot := snap.New_Snapshot(&snap.New_Snapshot_Input{
		Snapper:   s,
		File_Path: "/fake/test.go",
		Line:      5,
		Expected:  "hello",
	})
	if !snap.Snapshot_Is_Equal(snapshot, "hello") {
		t.Fatal("expected Is_Equal to return true for matching strings")
	}
	if output_buffer.Len() != 0 {
		t.Fatalf("expected no diagnostic output, got: %s", output_buffer.String())
	}
}

// Test_Is_Equal_Mismatch verifies that Is_Equal returns false and prints a diff on mismatch.
func Test_Is_Equal_Mismatch(t *testing.T) {
	s, output_buffer, _ := test_snapper(nil)
	snapshot := snap.New_Snapshot(&snap.New_Snapshot_Input{
		Snapper:   s,
		File_Path: "/fake/test.go",
		Line:      5,
		Expected:  "expected",
	})
	if snap.Snapshot_Is_Equal(snapshot, "actual") {
		t.Fatal("expected Is_Equal to return false for mismatching strings")
	}
	if !strings.Contains(output_buffer.String(), "Snapshot mismatch") {
		t.Fatalf("expected mismatch header in output, got: %s", output_buffer.String())
	}
}

// Test_Is_Equal_Edit verifies that Is_Equal with Should_Edit=true writes the updated
// source content to W and prints an UPDATED SNAPSHOT notice to Output.
func Test_Is_Equal_Edit(t *testing.T) {
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
		t.Fatal("expected Is_Equal with Should_Edit=true to return true")
	}
	if !strings.Contains(output_buffer.String(), "UPDATED SNAPSHOT") {
		t.Fatalf("expected UPDATED SNAPSHOT notice, got: %s", output_buffer.String())
	}
	if !strings.Contains(w_buffer.String(), "snap.Init(`new`)") {
		t.Fatalf("expected W to contain snap.Init(`new`), got: %s", w_buffer.String())
	}
}

// Test_Is_Equal_Edit_Line_Delta verifies that a multi-line replacement records the
// correct line delta in Snapper.Edits for use by subsequent edits in the same file.
func Test_Is_Equal_Edit_Line_Delta(t *testing.T) {
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

// Test_Expect_Panic_Match verifies that Expect_Panic passes when the panic message matches.
func Test_Expect_Panic_Match(t *testing.T) {
	s, _, _ := test_snapper(nil)
	snapshot := snap.New_Snapshot(&snap.New_Snapshot_Input{
		Snapper:   s,
		File_Path: "/fake/test.go",
		Line:      5,
		Expected:  "boom",
	})
	snap.Expect_Panic(t, snapshot, func() { panic("boom") })
}

// Test_Expect_Panic_No_Panic verifies that Is_Equal correctly identifies a mismatch
// when the actual panic message differs from the expected snapshot.
func Test_Expect_Panic_No_Panic(t *testing.T) {
	s, _, _ := test_snapper(nil)
	snapshot := snap.New_Snapshot(&snap.New_Snapshot_Input{
		Snapper:   s,
		File_Path: "/fake/test.go",
		Line:      5,
		Expected:  "boom",
	})
	if snap.Snapshot_Is_Equal(snapshot, "not boom") {
		t.Fatal("expected Is_Equal to return false for mismatching panic messages")
	}
}

// Test_Batch_Expect verifies that Batch_Expect runs all entries as subtests.
func Test_Batch_Expect(t *testing.T) {
	s, _, _ := test_snapper(nil)
	entries := []snap.Entry[string]{
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
		return strings.ToUpper(in)
	}, entries)
}

// Test_Run verifies that Run captures fn's output and asserts it against the snapshot.
func Test_Run(t *testing.T) {
	s, _, _ := test_snapper(nil)
	// Fprintln writes "hello\n"; Run wraps it as "\nSTDOUT:\nhello\n\n".
	snapshot := snap.New_Snapshot(&snap.New_Snapshot_Input{
		Snapper:   s,
		File_Path: "/fake/test.go",
		Line:      5,
		Expected:  "\nSTDOUT:\nhello\n\n",
	})
	snap.Run(t, func() {
		fmt.Fprintln(s.Stdout, "hello")
	}, snapshot)
}
