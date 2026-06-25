// Package bzip2 is a bounded bzip2 decompressor: the only constructor requires
// an explicit output cap, so a decompression bomb cannot exhaust memory. It is
// the one package permitted to call compress/bzip2's NewReader (lintv2 bans it
// everywhere else).
package bzip2

import (
	stdbzip2 "compress/bzip2"
	"io"
)

// Reader decompresses a bzip2 stream, yielding at most the caller-set cap.
type Reader struct {
	// Bounded is the decompressed stream capped at the caller's limit.
	Bounded io.Reader
}

// New_Reader wraps compressed in a bzip2 decompressor that yields at most
// decompressed_bytes_max bytes. The cap is mandatory; no constructor omits it.
// Unlike gzip, compress/bzip2's reader has no header to validate, so there is no
// error to return.
func New_Reader(compressed io.Reader, decompressed_bytes_max int64) (reader *Reader) {
	bounded := io.LimitReader(stdbzip2.NewReader(compressed), decompressed_bytes_max)
	return &Reader{Bounded: bounded}
}

// Read fills p from the bounded decompressed stream.
func (reader *Reader) Read(p []byte) (n int, err error) {
	return reader.Bounded.Read(p)
}
