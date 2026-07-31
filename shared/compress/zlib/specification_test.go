package zlib_test

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"local/james-orcales/shared/compress/zlib"
)

// Test_Bounded_Decompression verifies that New_Reader decodes a zlib stream
// and reports output that is larger than the caller's cap.
func Test_Bounded_Decompression(t *testing.T) {
	t.Parallel()
	compressed := []byte{
		120, 156, 203, 72, 205, 201, 201, 87, 72, 202, 47, 205, 75, 73, 77,
		81, 168, 202, 201, 76, 82, 40, 207, 47, 202, 73, 225, 2, 0, 123, 233,
		9, 57,
	}
	const WANT = "hello bounded zlib world\n"
	reader, new_err := zlib.New_Reader(bytes.NewReader(compressed), 1024)
	if new_err != nil {
		t.Fatalf("New_Reader: %v", new_err)
	}
	decoded := make([]byte, len(WANT))
	_, read_err := io.ReadFull(reader, decoded)
	if read_err != nil {
		t.Fatalf("ReadFull: %v", read_err)
	}
	if string(decoded) != WANT {
		t.Fatalf("decoded = %q, want %q", decoded, WANT)
	}
	exact, exact_new_err := zlib.New_Reader(
		bytes.NewReader(compressed), int64(len(WANT)),
	)
	if exact_new_err != nil {
		t.Fatalf("New_Reader exact case: %v", exact_new_err)
	}
	exact_decoded := make([]byte, len(WANT))
	_, exact_read_err := io.ReadFull(exact, exact_decoded)
	if exact_read_err != nil {
		t.Fatalf("exact-cap read: %v", exact_read_err)
	}
	if string(exact_decoded) != WANT {
		t.Fatalf("exact-cap decoded = %q, want %q", exact_decoded, WANT)
	}
	const PROBE_BYTE_COUNT = 1
	var exact_probe [PROBE_BYTE_COUNT]byte
	exact_probe_count, exact_probe_err := exact.Read(exact_probe[:])
	if exact_probe_count != 0 {
		t.Fatalf("exact-cap probe read %d bytes", exact_probe_count)
	}
	if exact_probe_err != io.EOF {
		t.Fatalf("exact-cap probe error = %v, want EOF", exact_probe_err)
	}
	overflow, overflow_new_err := zlib.New_Reader(bytes.NewReader(compressed), 5)
	if overflow_new_err != nil {
		t.Fatalf("New_Reader overflow case: %v", overflow_new_err)
	}
	overflow_buffer := make([]byte, 6)
	_, overflow_err := io.ReadFull(overflow, overflow_buffer)
	var overflow_target *zlib.Output_Overflow_Error
	if !errors.As(overflow_err, &overflow_target) {
		t.Fatalf("overflow error = %v, want Output_Overflow_Error", overflow_err)
	}
	_, negative_err := zlib.New_Reader(bytes.NewReader(compressed), -1)
	if negative_err == nil {
		t.Fatal("negative output cap succeeded")
	}
}
