// Package zip is a bounded zip entry reader: the only constructor checks the
// entry's declared uncompressed size against an explicit cap and then bounds the
// reader, so a zip bomb cannot exhaust memory. It is the one package permitted to
// call archive/zip's NewReader (lintv2 bans it everywhere else).
package zip

import (
	stdzip "archive/zip"
	"fmt"
	"io"
	"path"
	"strings"
)

// Reader decompresses one zip entry, yielding at most the caller-set cap.
type Reader struct {
	// Bounded is the entry stream capped at the caller's limit.
	Bounded io.Reader
	// Inner is the underlying entry reader, retained so Close can release it.
	Inner io.Closer
}

// New_Reader_Input names the archive, the entry suffix, and the output cap. A
// suffix is matched rather than a full name because some archives prefix members
// with a date-stamped directory whose name is not known ahead of time.
type New_Reader_Input struct {
	// Source is the random-access archive content.
	Source io.ReaderAt
	// Size is the archive's byte length.
	Size int64
	// Entry_Suffix is the suffix of the archive member to extract.
	Entry_Suffix string
	// Bytes_Max caps the entry's decompressed size.
	Bytes_Max int64
}

// New_Reader opens the first entry whose name ends with the suffix, rejecting an
// entry whose declared uncompressed size exceeds the cap and bounding the reader
// at it.
func New_Reader(input *New_Reader_Input) (reader *Reader, err error) {
	if input == nil {
		return nil, fmt.Errorf("zip: nil input")
	}
	archive, archive_err := stdzip.NewReader(input.Source, input.Size)
	if archive_err != nil {
		return nil, archive_err
	}
	for _, file := range archive.File {
		// Match the member's base name, not any trailing substring of the full
		// path, so a partial filename suffix cannot select the wrong entry.
		if !strings.HasSuffix(path.Base(file.Name), input.Entry_Suffix) {
			continue
		}
		if file.UncompressedSize64 > uint64(input.Bytes_Max) {
			return nil, fmt.Errorf("zip: entry %q exceeds %d-byte cap",
				input.Entry_Suffix, input.Bytes_Max)
		}
		opened, open_err := file.Open()
		if open_err != nil {
			return nil, open_err
		}
		return &Reader{Bounded: io.LimitReader(opened, input.Bytes_Max), Inner: opened}, nil
	}
	return nil, fmt.Errorf("zip: no entry with suffix %q", input.Entry_Suffix)
}

// Read fills p from the bounded entry stream.
func (reader *Reader) Read(p []byte) (n int, err error) {
	return reader.Bounded.Read(p)
}

// Close releases the underlying entry reader.
func (reader *Reader) Close() (err error) {
	return reader.Inner.Close()
}
