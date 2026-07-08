package bzip2_test

import (
	"bytes"
	"io"
	"testing"

	"local/james-orcales/shared/bzip2"
)

// Test_Bounded_Decompression verifies New_Reader decompresses a bzip2 stream and
// caps the decompressed output at the caller's limit.
func Test_Bounded_Decompression(t *testing.T) {
	t.Parallel()
	compressed := []byte{
		66, 90, 104, 57, 49, 65, 89, 38, 83, 89, 172, 2, 115, 97, 0, 0,
		6, 89, 128, 0, 16, 64, 0, 16, 0, 22, 101, 210, 144, 32, 0, 34,
		38, 141, 52, 61, 6, 161, 76, 0, 19, 70, 10, 209, 180, 29, 135, 30,
		52, 252, 181, 54, 74, 86, 190, 46, 228, 138, 112, 161, 33, 88, 4, 230,
		194,
	}
	const WANT = "hello bounded bzip2 world\n"
	full := bzip2.New_Reader(bytes.NewReader(compressed), 1024)
	decoded := make([]byte, len(WANT))
	if _, read_err := io.ReadFull(full, decoded); read_err != nil {
		t.Fatalf("ReadFull: %v", read_err)
	}
	if string(decoded) != WANT {
		t.Fatalf("decoded = %q, want %q", decoded, WANT)
	}
	capped := bzip2.New_Reader(bytes.NewReader(compressed), 5)
	overflow := make([]byte, len(WANT))
	count, _ := io.ReadFull(capped, overflow)
	if count != 5 {
		t.Fatalf("capped read = %d bytes, want 5", count)
	}
}
