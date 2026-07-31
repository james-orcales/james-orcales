// Package zlib provides bounded zlib decompression. The constructor requires
// an output cap, which prevents compressed input from allocating unbounded
// output memory.
package zlib

import (
	stdzlib "compress/zlib"
	"fmt"
	"io"
)

// Reader reads one zlib stream and rejects decompressed output above its cap.
type Reader struct {
	// Decompressed is the checksum-validating standard-library decoder.
	Decompressed io.ReadCloser
	// Bytes_Max is the output cap that the caller supplied.
	Bytes_Max int64
	// Bytes_Read_Count is the output size that Read returned.
	Bytes_Read_Count int64
}

// Output_Overflow_Error reports that decompression produced more bytes than
// the cap that the caller supplied to New_Reader.
type Output_Overflow_Error struct {
	// Bytes_Max is the caller's decompressed output cap.
	Bytes_Max int64
}

// Error formats the bounded-decompression failure.
func (err *Output_Overflow_Error) Error() (message string) {
	return fmt.Sprintf("zlib output exceeds %d bytes", err.Bytes_Max)
}

// New_Reader validates the zlib header and returns a reader that reports an
// overflow when decompressed output exceeds decompressed_bytes_max.
func New_Reader(
	compressed io.Reader,
	decompressed_bytes_max int64,
) (reader *Reader, err error) {
	if decompressed_bytes_max < 0 {
		return nil, fmt.Errorf(
			"zlib output cap is negative: %d",
			decompressed_bytes_max,
		)
	}
	decompressed, new_err := stdzlib.NewReader(compressed)
	if new_err != nil {
		return nil, new_err
	}
	return &Reader{
		Decompressed: decompressed,
		Bytes_Max:    decompressed_bytes_max,
	}, nil
}

// Read returns decompressed bytes until the cap. It then probes one byte to
// distinguish an exact-size stream from an overflowing stream.
func (reader *Reader) Read(buffer []byte) (count int, err error) {
	if reader.Bytes_Read_Count == reader.Bytes_Max {
		var probe [1]byte
		probe_count, probe_err := reader.Decompressed.Read(probe[:])
		if probe_count != 0 {
			return 0, &Output_Overflow_Error{Bytes_Max: reader.Bytes_Max}
		}
		return 0, probe_err
	}
	read_buffer := buffer
	bytes_available := reader.Bytes_Max - reader.Bytes_Read_Count
	if int64(len(read_buffer)) > bytes_available {
		read_buffer = read_buffer[:bytes_available]
	}
	count, err = reader.Decompressed.Read(read_buffer)
	reader.Bytes_Read_Count += int64(count)
	return count, err
}

// Close releases the checksum state owned by the standard-library decoder.
func (reader *Reader) Close() (err error) {
	return reader.Decompressed.Close()
}
