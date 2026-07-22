package zip_test

import (
	stdzip "archive/zip"
	"io"
	"os"
	"testing"

	"local/james-orcales/shared/archive/zip"
)

// Test_Bounded_Entry verifies New_Reader extracts a named entry capped at the
// caller's limit and rejects a missing entry.
func Test_Bounded_Entry(t *testing.T) {
	t.Parallel()
	const WANT = "hello bounded zip world"
	archive_path := write_archive(t, &write_archive_input{
		Entry_Name: "data.txt", Content: WANT,
	})
	source, open_err := os.Open(archive_path)
	if open_err != nil {
		t.Fatalf("open archive: %v", open_err)
	}
	defer source.Close()
	info, stat_err := source.Stat()
	if stat_err != nil {
		t.Fatalf("stat archive: %v", stat_err)
	}
	reader, err := zip.New_Reader(&zip.New_Reader_Input{
		Source: source, Size: info.Size(), Entry_Suffix: "data.txt", Bytes_Max: 1024,
	})
	if err != nil {
		t.Fatalf("New_Reader: %v", err)
	}
	decoded := make([]byte, len(WANT))
	if _, read_err := io.ReadFull(reader, decoded); read_err != nil {
		t.Fatalf("ReadFull: %v", read_err)
	}
	if string(decoded) != WANT {
		t.Fatalf("decoded = %q, want %q", decoded, WANT)
	}
	if _, missing_err := zip.New_Reader(&zip.New_Reader_Input{
		Source: source, Size: info.Size(), Entry_Suffix: "absent.txt", Bytes_Max: 1024,
	}); missing_err == nil {
		t.Fatal("expected an error for a missing entry")
	}
	if _, nil_err := zip.New_Reader(nil); nil_err == nil {
		t.Fatal("expected an error for nil input rather than a panic")
	}
}

// Write_archive_input names one zip entry to write.
type write_archive_input struct {
	// Entry_Name is the archive member name.
	Entry_Name string
	// Content is the member's content.
	Content string
}

// Write_archive writes a one-entry zip archive to a temp file and returns its
// path.
func write_archive(t *testing.T, input *write_archive_input) (path string) {
	t.Helper()
	file, create_err := os.CreateTemp(t.TempDir(), "archive-*.zip")
	if create_err != nil {
		t.Fatalf("create temp: %v", create_err)
	}
	writer := stdzip.NewWriter(file)
	entry, entry_err := writer.Create(input.Entry_Name)
	if entry_err != nil {
		t.Fatalf("create entry: %v", entry_err)
	}
	if _, write_err := entry.Write([]byte(input.Content)); write_err != nil {
		t.Fatalf("write entry: %v", write_err)
	}
	if close_err := writer.Close(); close_err != nil {
		t.Fatalf("close writer: %v", close_err)
	}
	if close_err := file.Close(); close_err != nil {
		t.Fatalf("close file: %v", close_err)
	}
	return file.Name()
}
