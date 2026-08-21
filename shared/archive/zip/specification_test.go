package zip_test

import (
	"compress/flate"
	"encoding/binary"
	"hash/crc32"
	"testing"

	"local/james-orcales/shared/archive/zip"
	"local/james-orcales/shared/bytes"
)

// Test_Bounded_Entry verifies stored and deflated entries stay within caller storage.
func Test_Bounded_Entry(t *testing.T) {
	t.Parallel()
	const ARCHIVE_STORAGE_SIZE = 512
	const OUTPUT_STORAGE_SIZE = 64
	content := []byte("hello bounded zip world")
	for _, one := range []struct {
		Name       string
		Method     uint16
		Compressed []byte
	}{
		{Name: "stored", Method: 0, Compressed: content},
		{Name: "deflated", Method: 8, Compressed: deflated(t, content)},
	} {
		t.Run(one.Name, func(t *testing.T) {
			t.Parallel()
			var archive_storage [ARCHIVE_STORAGE_SIZE]byte
			archive := archive_data(
				archive_storage[:], []byte("dated/data.txt"), one.Method,
				one.Compressed, content,
			)
			var decoded [OUTPUT_STORAGE_SIZE]byte
			count, status := zip.Decode_Into(archive, "data.txt", decoded[:])
			if status != zip.STATUS_OK {
				t.Fatalf("Decode_Into status = %v, want STATUS_OK", status)
			}
			if !bytes.Equal(decoded[:count], content) {
				t.Fatalf("decoded = %q, want %q", decoded[:count], content)
			}
		})
	}
}

// Test_Allocation proves every decompression path leaves heap untouched.
func Test_Allocation(t *testing.T) {
	const ARCHIVE_STORAGE_SIZE = 512
	const OUTPUT_STORAGE_SIZE = 64
	content := []byte("hello bounded zip world")
	compressed := deflated(t, content)
	var stored_storage [ARCHIVE_STORAGE_SIZE]byte
	stored := archive_data(
		stored_storage[:], []byte("data.txt"), 0, content, content,
	)
	var deflated_storage [ARCHIVE_STORAGE_SIZE]byte
	deflated_archive := archive_data(
		deflated_storage[:], []byte("data.txt"), 8, compressed, content,
	)
	var descriptor_storage [ARCHIVE_STORAGE_SIZE]byte
	descriptor_archive := descriptor_archive_data(
		descriptor_storage[:], []byte("data.txt"), compressed, content,
	)
	var decoded [OUTPUT_STORAGE_SIZE]byte
	var count bytes.Boundary
	var status zip.Status
	allocations := testing.AllocsPerRun(100, func() {
		count, status = zip.Decode_Into(stored, "data.txt", decoded[:])
		count, status = zip.Decode_Into(descriptor_archive, "data.txt", decoded[:])
		count, status = zip.Decode_Into(deflated_archive, "data.txt", decoded[:])
	})
	if status != zip.STATUS_OK {
		t.Fatalf("Decode_Into status = %v", status)
	}
	if count != bytes.Boundary(len(content)) {
		t.Fatalf("Decode_Into count = %d", count)
	}
	if allocations != 0 {
		t.Errorf("Decode_Into allocated %v times; want 0", allocations)
	}
}

// Test_Deflate_Forms covers stored, Huffman-only, and repeated-span DEFLATE blocks.
func Test_Deflate_Forms(t *testing.T) {
	t.Parallel()
	const ARCHIVE_STORAGE_SIZE = 16384
	const OUTPUT_STORAGE_SIZE = bytes.SLICE_SIZE_MAXIMUM
	var content_storage [OUTPUT_STORAGE_SIZE]byte
	content_count := bytes.Repeat_Into(
		content_storage[:],
		[]byte("bounded DEFLATE data with repeated words and skewed symbols\n"),
		48,
	)
	content := content_storage[:content_count]
	for _, one := range []struct {
		Name  string
		Level int
	}{
		{Name: "stored blocks", Level: flate.NoCompression},
		{Name: "Huffman only", Level: flate.HuffmanOnly},
		{Name: "repeated spans", Level: flate.BestCompression},
	} {
		t.Run(one.Name, func(t *testing.T) {
			t.Parallel()
			compressed := deflated_level(t, content, one.Level)
			var archive_storage [ARCHIVE_STORAGE_SIZE]byte
			archive := archive_data(
				archive_storage[:], []byte("data.txt"), 8, compressed, content,
			)
			var decoded [OUTPUT_STORAGE_SIZE]byte
			count, status := zip.Decode_Into(archive, "data.txt", decoded[:])
			if status != zip.STATUS_OK {
				t.Fatalf("Decode_Into status = %v", status)
			}
			if !bytes.Equal(decoded[:count], content) {
				t.Fatalf("decoded %d bytes incorrectly", count)
			}
		})
	}
}

// Test_Data_Descriptor preserves streaming-writer metadata layout.
func Test_Data_Descriptor(t *testing.T) {
	t.Parallel()
	const ARCHIVE_STORAGE_SIZE = 512
	const OUTPUT_STORAGE_SIZE = 64
	content := []byte("hello bounded zip world")
	compressed := deflated(t, content)
	var archive_storage [ARCHIVE_STORAGE_SIZE]byte
	archive := descriptor_archive_data(
		archive_storage[:], []byte("data.txt"), compressed, content,
	)
	var decoded [OUTPUT_STORAGE_SIZE]byte
	count, status := zip.Decode_Into(archive, "data.txt", decoded[:])
	if status != zip.STATUS_OK {
		t.Fatalf("Decode_Into status = %v", status)
	}
	if !bytes.Equal(decoded[:count], content) {
		t.Fatalf("decoded = %q", decoded[:count])
	}
}

// Test_Deflate_Corpus covers codec choices across distinct byte distributions.
func Test_Deflate_Corpus(t *testing.T) {
	t.Parallel()
	const DATA_SIZE = 1024
	const CORPUS_COUNT = 4
	const ARCHIVE_STORAGE_SIZE = 4096
	var corpus [CORPUS_COUNT][DATA_SIZE]byte
	for index := 0; index < DATA_SIZE; index++ {
		corpus[0][index] = 'A'
		corpus[1][index] = byte(index)
		corpus[2][index] = byte(index*73 + 19)
		corpus[3][index] = byte((index/17 + index/113) % 11)
	}
	levels := [...]int{
		flate.NoCompression, flate.HuffmanOnly, flate.BestSpeed,
		flate.DefaultCompression, flate.BestCompression,
	}
	for corpus_index := range corpus {
		for _, level := range levels {
			content := corpus[corpus_index][:]
			compressed := deflated_level(t, content, level)
			var archive_storage [ARCHIVE_STORAGE_SIZE]byte
			archive := archive_data(
				archive_storage[:], []byte("data.txt"), 8, compressed, content,
			)
			var decoded [DATA_SIZE]byte
			count, status := zip.Decode_Into(archive, "data.txt", decoded[:])
			if status != zip.STATUS_OK {
				t.Fatalf("corpus %d/%d status = %v", corpus_index, level, status)
			}
			if !bytes.Equal(decoded[:count], content) {
				t.Fatalf("corpus %d level %d mismatch", corpus_index, level)
			}
		}
	}
}

// Test_Deflate_Rejection proves corrupt bitstreams remain scalar failures.
func Test_Deflate_Rejection(t *testing.T) {
	t.Parallel()
	const ARCHIVE_STORAGE_SIZE = 2048
	const OUTPUT_STORAGE_SIZE = 512
	var content_storage [OUTPUT_STORAGE_SIZE]byte
	content_count := bytes.Repeat_Into(
		content_storage[:], []byte("bounded corrupt stream "), 16,
	)
	content := content_storage[:content_count]
	compressed := deflated_level(t, content, flate.BestCompression)
	var archive_storage [ARCHIVE_STORAGE_SIZE]byte
	name := []byte("data.txt")
	archive := archive_data(
		archive_storage[:], name, 8, compressed, content,
	)
	data_position := 30 + len(name)
	var decoded [OUTPUT_STORAGE_SIZE]byte
	for compressed_index := range compressed {
		archive[data_position+compressed_index] ^= 1
		count, status := zip.Decode_Into(archive, "data.txt", decoded[:])
		if status == zip.STATUS_OK {
			if !bytes.Equal(decoded[:count], content) {
				t.Fatalf("mutation %d escaped checksum", compressed_index)
			}
		} else if status != zip.STATUS_INPUT_INVALID {
			t.Fatalf("mutation %d status = %v", compressed_index, status)
		}
		archive[data_position+compressed_index] ^= 1
	}
}

// Test_Malformed_Metadata verifies each single-byte mutation remains bounded.
func Test_Malformed_Metadata(t *testing.T) {
	t.Parallel()
	const ARCHIVE_STORAGE_SIZE = 512
	const OUTPUT_STORAGE_SIZE = 64
	content := []byte("hello bounded zip world")
	var archive_storage [ARCHIVE_STORAGE_SIZE]byte
	archive := archive_data(
		archive_storage[:], []byte("data.txt"), 0, content, content,
	)
	var decoded [OUTPUT_STORAGE_SIZE]byte
	for mutation_index := range archive {
		archive[mutation_index] ^= 0xff
		count, status := zip.Decode_Into(archive, "data.txt", decoded[:])
		switch status {
		case zip.STATUS_OK:
			if !bytes.Equal(decoded[:count], content) {
				t.Fatalf("mutation %d changed accepted output", mutation_index)
			}
		case zip.STATUS_INPUT_INVALID,
			zip.STATUS_ENTRY_NOT_FOUND,
			zip.STATUS_OUTPUT_TOO_SMALL,
			zip.STATUS_METHOD_UNSUPPORTED:
			if count != 0 {
				t.Fatalf("mutation %d count = %d", mutation_index, count)
			}
		default:
			t.Fatalf("mutation %d returned status %v", mutation_index, status)
		}
		archive[mutation_index] ^= 0xff
	}
}

// Test_Empty_Entry keeps zero-byte output distinct from missing output storage.
func Test_Empty_Entry(t *testing.T) {
	t.Parallel()
	const ARCHIVE_STORAGE_SIZE = 256
	var archive_storage [ARCHIVE_STORAGE_SIZE]byte
	archive := archive_data(
		archive_storage[:], []byte("empty.txt"), 0, nil, nil,
	)
	count, status := zip.Decode_Into(archive, "empty.txt", nil)
	if status != zip.STATUS_OK {
		t.Fatalf("Decode_Into status = %v", status)
	}
	if count != 0 {
		t.Fatalf("Decode_Into count = %d", count)
	}
}

// Test_Rejection verifies hostile metadata cannot escape source or destination bounds.
func Test_Rejection(t *testing.T) {
	t.Parallel()
	const ARCHIVE_STORAGE_SIZE = 512
	const OUTPUT_STORAGE_SIZE = 64
	content := []byte("hello bounded zip world")
	var archive_storage [ARCHIVE_STORAGE_SIZE]byte
	archive := archive_data(
		archive_storage[:], []byte("data.txt"), 0, content, content,
	)
	var decoded [OUTPUT_STORAGE_SIZE]byte
	for end_index := 0; end_index < len(archive); end_index++ {
		count, status := zip.Decode_Into(
			archive[:end_index], "data.txt", decoded[:],
		)
		if count != 0 {
			t.Fatalf("truncation %d count = %d", end_index, count)
		}
		if status != zip.STATUS_INPUT_INVALID {
			t.Fatalf("truncation %d status = %v", end_index, status)
		}
	}
	if count, status := zip.Decode_Into(
		archive, "absent.txt", decoded[:],
	); count != 0 {
		t.Fatalf("missing entry count = %d", count)
	} else if status != zip.STATUS_ENTRY_NOT_FOUND {
		t.Fatalf("missing entry status = %v", status)
	}
	if count, status := zip.Decode_Into(
		archive, "data.txt", decoded[:4],
	); count != 0 {
		t.Fatalf("short output count = %d", count)
	} else if status != zip.STATUS_OUTPUT_TOO_SMALL {
		t.Fatalf("short output status = %v", status)
	}
	if count, status := zip.Decode_Into(
		archive[:len(archive)-1], "data.txt", decoded[:],
	); count != 0 {
		t.Fatalf("truncated input count = %d", count)
	} else if status != zip.STATUS_INPUT_INVALID {
		t.Fatalf("truncated input status = %v", status)
	}
	corrupted := archive_data(
		archive_storage[:], []byte("data.txt"), 0, content, content,
	)
	corrupted[30+len("data.txt")] ^= 1
	if count, status := zip.Decode_Into(
		corrupted, "data.txt", decoded[:],
	); count != 0 {
		t.Fatalf("checksum mismatch count = %d", count)
	} else if status != zip.STATUS_INPUT_INVALID {
		t.Fatalf("checksum mismatch status = %v", status)
	}
	unsupported := archive_data(
		archive_storage[:], []byte("data.txt"), 99, content, content,
	)
	if count, status := zip.Decode_Into(
		unsupported, "data.txt", decoded[:],
	); count != 0 {
		t.Fatalf("unsupported method count = %d", count)
	} else if status != zip.STATUS_METHOD_UNSUPPORTED {
		t.Fatalf("unsupported method status = %v", status)
	}
	assert_invalid_suffixes(t, archive, decoded[:])
}

// Compression_Writer keeps generated fixture output within shared byte bounds.
type Compression_Writer struct {
	// Buffer makes fixed caller storage visible to compressor's writer interface.
	Buffer bytes.Buffer
}

// Write preserves compressor interface without storage growth.
func (writer *Compression_Writer) Write(source []byte) (count int, err error) {
	return int(bytes.Buffer_Write(&writer.Buffer, bytes.Slice(source))), nil
}

func assert_invalid_suffixes(t *testing.T, archive []byte, decoded []byte) {
	t.Helper()
	for _, suffix := range []bytes.Text{"", "bad/name", "bad\x00name"} {
		count, status := zip.Decode_Into(archive, suffix, decoded[:])
		if count != 0 {
			t.Fatalf("suffix %q count = %d", suffix, count)
		}
		if status != zip.STATUS_INPUT_INVALID {
			t.Fatalf("suffix %q status = %v", suffix, status)
		}
	}
}

func deflated(t *testing.T, content []byte) (compressed []byte) {
	return deflated_level(t, content, flate.DefaultCompression)
}

func deflated_level(
	t *testing.T, content []byte, level int,
) (compressed []byte) {
	t.Helper()
	var compressed_storage [bytes.SLICE_SIZE_MAXIMUM]byte
	var target Compression_Writer
	bytes.Buffer_Init(&target.Buffer, compressed_storage[:], nil)
	writer, new_err := flate.NewWriter(&target, level)
	if new_err != nil {
		t.Fatalf("NewWriter: %v", new_err)
	}
	if _, write_err := writer.Write(content); write_err != nil {
		t.Fatalf("Write: %v", write_err)
	}
	if close_err := writer.Close(); close_err != nil {
		t.Fatalf("Close: %v", close_err)
	}
	return bytes.Buffer_Bytes(&target.Buffer)
}

func archive_data(
	storage []byte,
	name []byte,
	method uint16,
	compressed []byte,
	uncompressed []byte,
) (archive []byte) {
	const LOCAL_HEADER_SIZE = 30
	const CENTRAL_HEADER_SIZE = 46
	const END_SIZE = 22
	checksum := crc32.ChecksumIEEE(uncompressed)
	local := storage[:LOCAL_HEADER_SIZE]
	binary.LittleEndian.PutUint32(local[0:], 0x04034b50)
	binary.LittleEndian.PutUint16(local[4:], 20)
	binary.LittleEndian.PutUint16(local[8:], method)
	binary.LittleEndian.PutUint32(local[14:], checksum)
	binary.LittleEndian.PutUint32(local[18:], uint32(len(compressed)))
	binary.LittleEndian.PutUint32(local[22:], uint32(len(uncompressed)))
	binary.LittleEndian.PutUint16(local[26:], uint16(len(name)))
	position := LOCAL_HEADER_SIZE
	position += copy(storage[position:], name)
	position += copy(storage[position:], compressed)
	central_position := position
	central := storage[position : position+CENTRAL_HEADER_SIZE]
	binary.LittleEndian.PutUint32(central[0:], 0x02014b50)
	binary.LittleEndian.PutUint16(central[4:], 20)
	binary.LittleEndian.PutUint16(central[6:], 20)
	binary.LittleEndian.PutUint16(central[10:], method)
	binary.LittleEndian.PutUint32(central[16:], checksum)
	binary.LittleEndian.PutUint32(central[20:], uint32(len(compressed)))
	binary.LittleEndian.PutUint32(central[24:], uint32(len(uncompressed)))
	binary.LittleEndian.PutUint16(central[28:], uint16(len(name)))
	position += CENTRAL_HEADER_SIZE
	position += copy(storage[position:], name)
	central_size := position - central_position
	end := storage[position : position+END_SIZE]
	binary.LittleEndian.PutUint32(end[0:], 0x06054b50)
	binary.LittleEndian.PutUint16(end[8:], 1)
	binary.LittleEndian.PutUint16(end[10:], 1)
	binary.LittleEndian.PutUint32(end[12:], uint32(central_size))
	binary.LittleEndian.PutUint32(end[16:], uint32(central_position))
	position += END_SIZE
	return storage[:position]
}

func descriptor_archive_data(
	storage []byte, name []byte, compressed []byte, uncompressed []byte,
) (archive []byte) {
	const LOCAL_HEADER_SIZE = 30
	const END_SIZE = 22
	const DESCRIPTOR_SIZE = 16
	const FLAG_DATA_DESCRIPTOR = 1 << 3
	archive = archive_data(storage, name, 8, compressed, uncompressed)
	central_position := LOCAL_HEADER_SIZE + len(name) + len(compressed)
	end_position := len(archive) - END_SIZE
	tail_size := len(archive) - central_position
	tail_destination := central_position + DESCRIPTOR_SIZE
	bytes.Clone_Into(
		storage[tail_destination:tail_destination+tail_size],
		storage[central_position:len(archive)],
	)
	binary.LittleEndian.PutUint16(storage[6:], FLAG_DATA_DESCRIPTOR)
	binary.LittleEndian.PutUint32(storage[14:], 0)
	binary.LittleEndian.PutUint32(storage[18:], 0)
	binary.LittleEndian.PutUint32(storage[22:], 0)
	descriptor := storage[central_position : central_position+DESCRIPTOR_SIZE]
	binary.LittleEndian.PutUint32(descriptor[0:], 0x08074b50)
	binary.LittleEndian.PutUint32(descriptor[4:], crc32.ChecksumIEEE(uncompressed))
	binary.LittleEndian.PutUint32(descriptor[8:], uint32(len(compressed)))
	binary.LittleEndian.PutUint32(descriptor[12:], uint32(len(uncompressed)))
	central_position += DESCRIPTOR_SIZE
	binary.LittleEndian.PutUint16(
		storage[central_position+8:], FLAG_DATA_DESCRIPTOR,
	)
	end_position += DESCRIPTOR_SIZE
	binary.LittleEndian.PutUint32(
		storage[end_position+16:], uint32(central_position),
	)
	return storage[:len(archive)+DESCRIPTOR_SIZE]
}
