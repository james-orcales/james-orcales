package zip_test

import (
	"hash/crc32"
	"testing"

	"local/james-orcales/shared/archive/zip"
	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/compress/flate"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/simulation/nbio"
	"local/james-orcales/shared/simulation/time"
	"local/james-orcales/shared/testify"
)

// Test_Bounded_Archive verifies stored and deflated bounded members.
func Test_Bounded_Archive(t *testing.T) {
	bounded_entry(t)
}

// Test_Allocation proves package operations leave heap ownership with callers.
func Test_Allocation(t *testing.T) {
	allocation(t)
}

const ZIP_BYTE_BIT_COUNT = 8
const ZIP_WORD_GROWTH_FACTOR = 2
const ZIP_WORD_16_SIZE = ZIP_WORD_GROWTH_FACTOR
const ZIP_WORD_32_SIZE = ZIP_WORD_16_SIZE * ZIP_WORD_GROWTH_FACTOR
const ZIP_SIGNATURE_SIZE = ZIP_WORD_32_SIZE
const ZIP_NAME_SIZE_MINIMUM = ZIP_WORD_16_SIZE / ZIP_WORD_GROWTH_FACTOR
const ZIP_LOCAL_EXTRACTOR_POSITION = ZIP_SIGNATURE_SIZE
const ZIP_LOCAL_FLAGS_POSITION = ZIP_LOCAL_EXTRACTOR_POSITION + ZIP_WORD_16_SIZE
const ZIP_LOCAL_METHOD_POSITION = ZIP_LOCAL_FLAGS_POSITION + ZIP_WORD_16_SIZE
const ZIP_LOCAL_TIME_POSITION = ZIP_LOCAL_METHOD_POSITION + ZIP_WORD_16_SIZE
const ZIP_LOCAL_DATE_POSITION = ZIP_LOCAL_TIME_POSITION + ZIP_WORD_16_SIZE
const ZIP_LOCAL_CHECKSUM_POSITION = ZIP_LOCAL_DATE_POSITION + ZIP_WORD_16_SIZE
const ZIP_LOCAL_COMPRESSED_SIZE_POSITION = ZIP_LOCAL_CHECKSUM_POSITION + ZIP_WORD_32_SIZE
const ZIP_LOCAL_UNCOMPRESSED_SIZE_POSITION = ZIP_LOCAL_COMPRESSED_SIZE_POSITION +
	ZIP_WORD_32_SIZE
const ZIP_LOCAL_NAME_SIZE_POSITION = ZIP_LOCAL_UNCOMPRESSED_SIZE_POSITION + ZIP_WORD_32_SIZE
const ZIP_LOCAL_EXTRA_SIZE_POSITION = ZIP_LOCAL_NAME_SIZE_POSITION + ZIP_WORD_16_SIZE
const ZIP_LOCAL_HEADER_SIZE = ZIP_LOCAL_EXTRA_SIZE_POSITION + ZIP_WORD_16_SIZE
const ZIP_CENTRAL_CREATOR_POSITION = ZIP_SIGNATURE_SIZE
const ZIP_CENTRAL_EXTRACTOR_POSITION = ZIP_CENTRAL_CREATOR_POSITION + ZIP_WORD_16_SIZE
const ZIP_CENTRAL_FLAGS_POSITION = ZIP_CENTRAL_EXTRACTOR_POSITION + ZIP_WORD_16_SIZE
const ZIP_CENTRAL_METHOD_POSITION = ZIP_CENTRAL_FLAGS_POSITION + ZIP_WORD_16_SIZE
const ZIP_CENTRAL_TIME_POSITION = ZIP_CENTRAL_METHOD_POSITION + ZIP_WORD_16_SIZE
const ZIP_CENTRAL_DATE_POSITION = ZIP_CENTRAL_TIME_POSITION + ZIP_WORD_16_SIZE
const ZIP_CENTRAL_CHECKSUM_POSITION = ZIP_CENTRAL_DATE_POSITION + ZIP_WORD_16_SIZE
const ZIP_CENTRAL_COMPRESSED_SIZE_POSITION = ZIP_CENTRAL_CHECKSUM_POSITION + ZIP_WORD_32_SIZE
const ZIP_CENTRAL_UNCOMPRESSED_SIZE_POSITION = ZIP_CENTRAL_COMPRESSED_SIZE_POSITION +
	ZIP_WORD_32_SIZE
const ZIP_CENTRAL_NAME_SIZE_POSITION = ZIP_CENTRAL_UNCOMPRESSED_SIZE_POSITION +
	ZIP_WORD_32_SIZE
const ZIP_CENTRAL_EXTRA_SIZE_POSITION = ZIP_CENTRAL_NAME_SIZE_POSITION + ZIP_WORD_16_SIZE
const ZIP_CENTRAL_COMMENT_SIZE_POSITION = ZIP_CENTRAL_EXTRA_SIZE_POSITION + ZIP_WORD_16_SIZE
const ZIP_CENTRAL_DISK_POSITION = ZIP_CENTRAL_COMMENT_SIZE_POSITION + ZIP_WORD_16_SIZE
const ZIP_CENTRAL_INTERNAL_ATTRIBUTES_POSITION = ZIP_CENTRAL_DISK_POSITION +
	ZIP_WORD_16_SIZE
const ZIP_CENTRAL_EXTERNAL_ATTRIBUTES_POSITION = ZIP_CENTRAL_INTERNAL_ATTRIBUTES_POSITION +
	ZIP_WORD_16_SIZE
const ZIP_CENTRAL_LOCAL_OFFSET_POSITION = ZIP_CENTRAL_EXTERNAL_ATTRIBUTES_POSITION +
	ZIP_WORD_32_SIZE
const ZIP_CENTRAL_HEADER_SIZE = ZIP_CENTRAL_LOCAL_OFFSET_POSITION + ZIP_WORD_32_SIZE
const ZIP_DIRECTORY_DISK_POSITION = ZIP_SIGNATURE_SIZE
const ZIP_DIRECTORY_CENTRAL_DISK_POSITION = ZIP_DIRECTORY_DISK_POSITION + ZIP_WORD_16_SIZE
const ZIP_DIRECTORY_DISK_ENTRY_COUNT_POSITION = ZIP_DIRECTORY_CENTRAL_DISK_POSITION +
	ZIP_WORD_16_SIZE
const ZIP_DIRECTORY_ENTRY_COUNT_POSITION = ZIP_DIRECTORY_DISK_ENTRY_COUNT_POSITION +
	ZIP_WORD_16_SIZE
const ZIP_DIRECTORY_CENTRAL_SIZE_POSITION = ZIP_DIRECTORY_ENTRY_COUNT_POSITION +
	ZIP_WORD_16_SIZE
const ZIP_DIRECTORY_CENTRAL_OFFSET_POSITION = ZIP_DIRECTORY_CENTRAL_SIZE_POSITION +
	ZIP_WORD_32_SIZE
const ZIP_DIRECTORY_COMMENT_SIZE_POSITION = ZIP_DIRECTORY_CENTRAL_OFFSET_POSITION +
	ZIP_WORD_32_SIZE
const ZIP_DIRECTORY_END_SIZE = ZIP_DIRECTORY_COMMENT_SIZE_POSITION + ZIP_WORD_16_SIZE
const ZIP_DATA_DESCRIPTOR_SIZE = 3 * ZIP_WORD_32_SIZE
const ZIP_SIGNED_DESCRIPTOR_SIZE = ZIP_SIGNATURE_SIZE + ZIP_DATA_DESCRIPTOR_SIZE
const ZIP_DESCRIPTOR_CHECKSUM_POSITION = ZIP_SIGNATURE_SIZE
const ZIP_MINIMUM_ARCHIVE_SIZE = ZIP_LOCAL_HEADER_SIZE + ZIP_CENTRAL_HEADER_SIZE +
	2*ZIP_NAME_SIZE_MINIMUM + ZIP_DIRECTORY_END_SIZE
const ZIP_MINIMUM_DESCRIPTOR_ARCHIVE_SIZE = ZIP_MINIMUM_ARCHIVE_SIZE +
	ZIP_DATA_DESCRIPTOR_SIZE
const STANDARD_LIBRARY_WORD_GROWTH_FACTOR = ZIP_WORD_GROWTH_FACTOR
const STANDARD_LIBRARY_WORD_16_SIZE = ZIP_WORD_16_SIZE
const STANDARD_LIBRARY_WORD_32_SIZE = ZIP_WORD_32_SIZE
const STANDARD_LIBRARY_SIGNATURE_SIZE = ZIP_SIGNATURE_SIZE
const STANDARD_LIBRARY_LOCAL_HEADER_SIZE = ZIP_LOCAL_HEADER_SIZE
const STANDARD_LIBRARY_LOCAL_UNCOMPRESSED_SIZE_POSITION = ZIP_LOCAL_UNCOMPRESSED_SIZE_POSITION
const STANDARD_LIBRARY_CENTRAL_HEADER_SIZE = ZIP_CENTRAL_HEADER_SIZE
const STANDARD_LIBRARY_CENTRAL_NAME_SIZE_POSITION = ZIP_CENTRAL_NAME_SIZE_POSITION
const STANDARD_LIBRARY_CENTRAL_EXTRA_SIZE_POSITION = ZIP_CENTRAL_EXTRA_SIZE_POSITION
const STANDARD_LIBRARY_CENTRAL_COMMENT_SIZE_POSITION = ZIP_CENTRAL_COMMENT_SIZE_POSITION
const STANDARD_LIBRARY_CENTRAL_UNCOMPRESSED_SIZE_POSITION = ZIP_CENTRAL_UNCOMPRESSED_SIZE_POSITION
const STANDARD_LIBRARY_DIRECTORY_END_SIZE = ZIP_DIRECTORY_END_SIZE
const STANDARD_LIBRARY_DIRECTORY_DISK_ENTRY_COUNT_POSITION = ZIP_DIRECTORY_DISK_ENTRY_COUNT_POSITION
const STANDARD_LIBRARY_DIRECTORY_ENTRY_COUNT_POSITION = ZIP_DIRECTORY_ENTRY_COUNT_POSITION
const STANDARD_LIBRARY_DIRECTORY_CENTRAL_SIZE_POSITION = ZIP_DIRECTORY_CENTRAL_SIZE_POSITION
const STANDARD_LIBRARY_DIRECTORY_CENTRAL_OFFSET_POSITION = ZIP_DIRECTORY_CENTRAL_OFFSET_POSITION
const STANDARD_LIBRARY_DATA_DESCRIPTOR_SIZE = ZIP_DATA_DESCRIPTOR_SIZE
const STANDARD_LIBRARY_SIGNED_DESCRIPTOR_SIZE = ZIP_SIGNED_DESCRIPTOR_SIZE
const STANDARD_LIBRARY_DESCRIPTOR_CHECKSUM_POSITION = ZIP_DESCRIPTOR_CHECKSUM_POSITION
const STANDARD_LIBRARY_NAME_SIZE_MINIMUM = ZIP_NAME_SIZE_MINIMUM
const STANDARD_LIBRARY_MINIMUM_ARCHIVE_SIZE = ZIP_MINIMUM_ARCHIVE_SIZE
const STANDARD_LIBRARY_MINIMUM_DESCRIPTOR_ARCHIVE_SIZE = ZIP_MINIMUM_DESCRIPTOR_ARCHIVE_SIZE
const STANDARD_LIBRARY_SINGLE_NODE_CAPACITY = ZIP_NAME_SIZE_MINIMUM
const STANDARD_LIBRARY_SPECIAL_NODE_CAPACITY = ZIP_WORD_16_SIZE
const STANDARD_LIBRARY_ALLOCATION_NODE_CAPACITY = ZIP_WORD_32_SIZE
const STANDARD_LIBRARY_FILE_SYSTEM_CAPACITY = STANDARD_LIBRARY_ALLOCATION_NODE_CAPACITY *
	ZIP_WORD_GROWTH_FACTOR
const STANDARD_LIBRARY_ARCHIVE_CAPACITY = 1 << ZIP_BYTE_BIT_COUNT
const STANDARD_LIBRARY_ALLOCATION_OUTPUT_CAPACITY = STANDARD_LIBRARY_ARCHIVE_CAPACITY /
	STANDARD_LIBRARY_ALLOCATION_NODE_CAPACITY
const INVARIANT_DOMAIN_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM
const STANDARD_LIBRARY_WRITER_RECORD_NAME_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM -
	ZIP_CENTRAL_HEADER_SIZE - ZIP_DIRECTORY_END_SIZE
const STANDARD_LIBRARY_WRITER_RECORD_TAIL_MAXIMUM = STANDARD_LIBRARY_WRITER_RECORD_NAME_MAXIMUM
const STANDARD_LIBRARY_WRITER_LOCAL_STORAGE_MINIMUM = ZIP_LOCAL_HEADER_SIZE +
	ZIP_NAME_SIZE_MINIMUM
const STANDARD_LIBRARY_WRITER_CENTRAL_STORAGE_MINIMUM = ZIP_CENTRAL_HEADER_SIZE +
	ZIP_NAME_SIZE_MINIMUM
const STANDARD_LIBRARY_WRITER_LOCAL_START_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM -
	STANDARD_LIBRARY_WRITER_LOCAL_STORAGE_MINIMUM
const STANDARD_LIBRARY_FILE_SYSTEM_HEADER_INDEX_MAXIMUM = ((bytes.SLICE_SIZE_MAXIMUM-
	ZIP_DIRECTORY_END_SIZE)/
	STANDARD_LIBRARY_WRITER_CENTRAL_STORAGE_MINIMUM -
	STANDARD_LIBRARY_SINGLE_NODE_CAPACITY)

// Test_Standard_Library_Stream_Round_Trip ports reader and writer transport through nbio.
func standard_library_stream_round_trip(t *testing.T) {
	const STORAGE_SIZE = bytes.SLICE_SIZE_MAXIMUM
	var harness standard_library_stream_harness
	standard_library_stream_harness_init(t, &harness)
	testify.Equal_Values(
		t, zip.STATUS_OK,
		zip.Writer_Set_Comment(&harness.Writer, []byte("comment")),
	)
	first := zip.Header_Unvalidated{Name: []byte("foo"), Method: zip.METHOD_STORE}
	testify.Equal_Values(t, zip.STATUS_OK, zip.Writer_Create(&harness.Writer, &first))
	first_content := []byte("Rabbits, guinea pigs, gophers, marsupial rats, and quolls.")
	count, status := zip.Writer_Write(&harness.Writer, first_content)
	testify.Equal_Values(t, zip.STATUS_OK, status)
	testify.Equal(t, bytes.Boundary(len(first_content)), count)
	second := zip.Header_Unvalidated{Name: []byte("bar"), Method: zip.METHOD_DEFLATE}
	testify.Equal_Values(t, zip.STATUS_OK, zip.Writer_Create(&harness.Writer, &second))
	second_content := []byte("bounded deflated content")
	count, status = zip.Writer_Write(&harness.Writer, second_content)
	testify.Equal_Values(t, zip.STATUS_OK, status)
	testify.Equal(t, bytes.Boundary(len(second_content)), count)
	reader := standard_library_stream_harness_close_and_read(t, &harness)
	testify.Equal(t, zip.Entry_Count(2), reader.Entry_Count)
	testify.Equal(t, bytes.Slice("comment"), bytes.Slice(reader.Comment))
	var decoded [STORAGE_SIZE]byte
	for index, expected := range [][]byte{first_content, second_content} {
		var header zip.Header
		testify.Equal_Values(
			t, zip.STATUS_OK,
			zip.Reader_Header(&reader, zip.Entry_Count(index), &header), index,
		)
		testify.Equal(
			t,
			[]zip.Header_Name{
				zip.Header_Name(first.Name), zip.Header_Name(second.Name),
			}[index],
			header.Name, index,
		)
		decoded_count, decoded_status := zip.Reader_Decode(
			&reader, zip.Entry_Count(index), decoded[:],
		)
		testify.Equal(t, zip.STATUS_OK, decoded_status, index)
		testify.Equal(t, expected, decoded[:decoded_count], index)
	}
}

// Test_Standard_Library_Writer_Comment_UTF8_Offset ports metadata and prefix behavior.
func standard_library_writer_comment_utf8_offset(t *testing.T) {
	var harness standard_library_stream_harness
	standard_library_stream_harness_init(t, &harness)
	prefix := []byte{1, 2, 3, 1, 2, 3, 1, 2, 3}
	copy(harness.Output[:], prefix)
	testify.Equal_Values(
		t, zip.STATUS_OK,
		zip.Writer_Set_Offset(&harness.Writer, bytes.Boundary(len(prefix))),
	)
	comment := bytes.Slice("hi, こんにちわ")
	testify.Equal_Values(t, zip.STATUS_OK, zip.Writer_Set_Comment(&harness.Writer, comment))
	for _, header := range []zip.Header_Unvalidated{
		{Name: []byte("hi, hello"), Method: zip.METHOD_DEFLATE},
		{Name: []byte("hi, こんにちわ"), Method: zip.METHOD_DEFLATE},
		{
			Name: []byte("\x93\xfa\x96{\x8c\xea.txt"), Method: zip.METHOD_DEFLATE,
			Non_UTF8: true,
		},
	} {
		testify.Equal_Values(t, zip.STATUS_OK, zip.Writer_Create(&harness.Writer, &header))
		count, status := zip.Writer_Write(&harness.Writer, nil)
		testify.Equal_Values(t, zip.STATUS_OK, status)
		testify.Zero(t, count)
	}
	reader := standard_library_stream_harness_close_and_read(t, &harness)
	testify.Equal(t, prefix, harness.Output[:len(prefix)])
	testify.Equal(t, comment, bytes.Slice(reader.Comment))
	for index, flags := range []binary.Word_16{0x8, 0x808, 0x8} {
		var header zip.Header
		testify.Equal_Values(
			t, zip.STATUS_OK,
			zip.Reader_Header(&reader, zip.Entry_Count(index), &header),
		)
		testify.Equal(t, zip.Header_Flags(flags), header.Flags, index)
	}
}

// Test_Standard_Library_Writer_Flush_Directory ports flush and directory contracts.
func standard_library_writer_flush_directory(t *testing.T) {
	var harness standard_library_stream_harness
	standard_library_stream_harness_init(t, &harness)
	file := zip.Header_Unvalidated{Name: []byte("foo"), Method: zip.METHOD_STORE}
	testify.Equal_Values(t, zip.STATUS_OK, zip.Writer_Create(&harness.Writer, &file))
	var completion time.Completion
	flushed := false
	zip.Writer_Flush(
		&harness.Writer, &completion, func(_ *time.Completion) { flushed = true },
	)
	testify.True(t, flushed)
	testify.No_Error(t, completion.Error)
	testify.Not_Zero(t, harness.Writer.Count)
	testify.Not_Equal(
		t, [STANDARD_LIBRARY_SIGNATURE_SIZE]byte{},
		[STANDARD_LIBRARY_SIGNATURE_SIZE]byte(
			harness.Output[:STANDARD_LIBRARY_SIGNATURE_SIZE],
		),
	)
	directory := zip.Header_Unvalidated{
		Name: []byte("dir/"), Method: zip.METHOD_DEFLATE,
		Compressed_Size: 1234, Uncompressed_Size: 5678,
	}
	testify.Equal_Values(t, zip.STATUS_OK, zip.Writer_Create(&harness.Writer, &directory))
	count, status := zip.Writer_Write(&harness.Writer, nil)
	testify.Zero(t, count)
	testify.Equal_Values(t, zip.STATUS_OK, status)
	count, status = zip.Writer_Write(&harness.Writer, []byte("hello"))
	testify.Zero(t, count)
	testify.Equal_Values(t, zip.STATUS_INPUT_INVALID, status)
	reader := standard_library_stream_harness_close_and_read(t, &harness)
	var header zip.Header
	testify.Equal_Values(t, zip.STATUS_OK, zip.Reader_Header(&reader, 1, &header))
	testify.Equal(t, zip.Header_Method(zip.METHOD_STORE), header.Method)
	testify.Zero(t, header.Flags)
	testify.Zero(t, header.Compressed_Size)
	testify.Zero(t, header.Uncompressed_Size)
}

// Test_Standard_Library_Writer_Raw ports precompressed member behavior.
func standard_library_writer_raw(t *testing.T) {
	var harness standard_library_stream_harness
	standard_library_stream_harness_init(t, &harness)
	content := []byte("abcdefgabcdefgabcdefg")
	compressed := deflated_level(t, content, flate.BEST_SPEED)
	header := zip.Header_Unvalidated{
		Name: []byte("raw"), Method: zip.METHOD_DEFLATE,
		Flags:    zip.FLAG_DATA_DESCRIPTOR,
		Checksum: zip.Header_Checksum(crc32.ChecksumIEEE(content)),
		Compressed_Size: zip.Header_Compressed_Size_Unvalidated(
			len(compressed),
		),
		Uncompressed_Size: zip.Header_Uncompressed_Size_Unvalidated(
			len(content),
		),
	}
	testify.Equal_Values(t, zip.STATUS_OK, zip.Writer_Create_Raw(&harness.Writer, &header))
	count, status := zip.Writer_Write(&harness.Writer, compressed)
	testify.Equal_Values(t, zip.STATUS_OK, status)
	testify.Equal(t, bytes.Boundary(len(compressed)), count)
	reader := standard_library_stream_harness_close_and_read(t, &harness)
	var decoded [bytes.SLICE_SIZE_MAXIMUM]byte
	decoded_count, decoded_status := zip.Reader_Decode(&reader, 0, decoded[:])
	testify.Equal(t, zip.STATUS_OK, decoded_status)
	testify.Equal(t, content, decoded[:decoded_count])
	var decoded_header zip.Header
	testify.Equal_Values(t, zip.STATUS_OK, zip.Reader_Header(&reader, 0, &decoded_header))
	testify.Equal(t, header.Flags, decoded_header.Flags)
	testify.Equal(t, header.Checksum, decoded_header.Checksum)
	testify.Equal(
		t, zip.Header_Compressed_Size(header.Compressed_Size),
		decoded_header.Compressed_Size,
	)
	testify.Equal(
		t, zip.Header_Uncompressed_Size(header.Uncompressed_Size),
		decoded_header.Uncompressed_Size,
	)
}

// Test_Standard_Library_Writer_Copy ports compressed member copy behavior.
func standard_library_writer_copy(t *testing.T) {
	var source standard_library_stream_harness
	standard_library_stream_harness_init(t, &source)
	contents := [][]byte{[]byte("stored"), []byte("deflated deflated deflated")}
	for index, content := range contents {
		header := zip.Header_Unvalidated{
			Name: []byte{byte('a' + index)}, Method: zip.METHOD_STORE,
		}
		if index == 1 {
			header.Method = zip.METHOD_DEFLATE
		}
		testify.Equal_Values(t, zip.STATUS_OK, zip.Writer_Create(&source.Writer, &header))
		count, status := zip.Writer_Write(&source.Writer, content)
		testify.Equal_Values(t, zip.STATUS_OK, status)
		testify.Equal(t, bytes.Boundary(len(content)), count)
	}
	source_reader := standard_library_stream_harness_close_and_read(t, &source)
	var destination standard_library_stream_harness
	standard_library_stream_harness_init(t, &destination)
	for index := range contents {
		testify.Equal(
			t, zip.STATUS_OK, zip.Writer_Copy(
				&destination.Writer, &source_reader, zip.Entry_Count(index),
			),
		)
	}
	destination_reader := standard_library_stream_harness_close_and_read(t, &destination)
	var decoded [bytes.SLICE_SIZE_MAXIMUM]byte
	for index, content := range contents {
		count, status := zip.Reader_Decode(
			&destination_reader, zip.Entry_Count(index), decoded[:],
		)
		testify.Equal_Values(t, zip.STATUS_OK, status)
		testify.Equal(t, content, decoded[:count])
	}
}

// Test_Standard_Library_Header_Mode_Time ports FileHeader mode and time contracts.
func standard_library_header_mode_time(t *testing.T) {
	for _, mode := range []nbio.File_Mode{
		0,
		1,
		2,
		0o777,
		0o666,
		0o755 | nbio.FILE_MODE_SET_USER_IDENTIFIER,
		0o755 | nbio.FILE_MODE_SET_GROUP_IDENTIFIER,
		0o755 | nbio.FILE_MODE_SYMBOLIC_LINK,
		0o755 | nbio.FILE_MODE_DEVICE,
		0o755 | nbio.FILE_MODE_DEVICE | nbio.FILE_MODE_CHARACTER_DEVICE,
		0o755 | nbio.FILE_MODE_DIRECTORY,
		0o755 | nbio.FILE_MODE_NAMED_PIPE,
		0o755 | nbio.FILE_MODE_SOCKET,
		0o755 | nbio.FILE_MODE_STICKY,
	} {
		header := zip.Header_Unvalidated{Name: []byte("mode")}
		zip.Header_Set_Mode(&header, mode)
		validated, status := zip.Header_Validate(&header)
		testify.Equal_Values(t, zip.STATUS_OK, status)
		testify.Equal_Values(t, mode, zip.Header_Mode(&validated), mode)
	}
	modified := zip.Timestamp{Seconds: 1257896758, Set: true}
	header := zip.Header_Unvalidated{Name: []byte("time")}
	testify.Equal_Values(
		t, zip.STATUS_OK, zip.Header_Set_Modification_Time(&header, modified),
	)
	validated, status := zip.Header_Validate(&header)
	testify.Equal_Values(t, zip.STATUS_OK, status)
	testify.Equal(t, modified, zip.Header_Modification_Time(&validated))

	var harness standard_library_stream_harness
	standard_library_stream_harness_init(t, &harness)
	modified = zip.Timestamp{
		Seconds: 1509509517, Zone_Offset_Seconds: -7 * 60 * 60, Set: true,
	}
	header = zip.Header_Unvalidated{
		Name: []byte("test.txt"), Method: zip.METHOD_STORE, Modified: modified,
	}
	testify.Equal_Values(t, zip.STATUS_OK, zip.Writer_Create(&harness.Writer, &header))
	reader := standard_library_stream_harness_close_and_read(t, &harness)
	testify.Equal_Values(t, zip.STATUS_OK, zip.Reader_Header(&reader, 0, &validated))
	testify.Equal(t, modified, zip.Header_Modification_Time(&validated))
}

// Test_Standard_Library_Header_Extra_Bounds ports malformed-extra tolerance and field bounds.
func standard_library_header_extra_bounds(t *testing.T) {
	var harness standard_library_stream_harness
	standard_library_stream_harness_init(t, &harness)
	for _, extra := range [][]byte{
		{1},
		{1, 0, 24, 0, 1, 2, 3},
		{85, 84, 5, 0, 3, 154, 144, 195, 77, 85, 120, 0, 0},
	} {
		header := zip.Header_Unvalidated{
			Name: []byte("extra"), Extra: extra, Method: zip.METHOD_STORE,
		}
		testify.Equal_Values(t, zip.STATUS_OK, zip.Writer_Create(&harness.Writer, &header))
	}
	long_name := zip.Header_Unvalidated{
		Name: make([]byte, bytes.SLICE_SIZE_MAXIMUM+1), Method: zip.METHOD_STORE,
	}
	testify.Equal_Values(
		t, zip.STATUS_INPUT_INVALID, zip.Writer_Create(&harness.Writer, &long_name),
	)
	long_extra := zip.Header_Unvalidated{
		Name: []byte("extra"), Extra: make([]byte, bytes.SLICE_SIZE_MAXIMUM+1),
		Method: zip.METHOD_STORE,
	}
	testify.Equal_Values(
		t, zip.STATUS_INPUT_INVALID, zip.Writer_Create(&harness.Writer, &long_extra),
	)
	reader := standard_library_stream_harness_close_and_read(t, &harness)
	testify.Equal(t, zip.Entry_Count(3), reader.Entry_Count)
}

// Test_Standard_Library_File_System ports Open, ReadDir, Stat, and Walk behavior.
func standard_library_file_system(t *testing.T) {
	var harness standard_library_stream_harness
	standard_library_stream_harness_init(t, &harness)
	standard_library_file_system_archive(t, &harness)
	reader := standard_library_stream_harness_close_and_read(t, &harness)
	var file_system zip.File_System
	testify.Equal_Values(t, zip.STATUS_OK, zip.File_System_Init(&file_system, &reader))
	var root zip.File_Info
	status := zip.File_System_Status(&file_system, []byte("."), &root)
	testify.Equal_Values(t, zip.STATUS_OK, status)
	testify.True(t, bool(root.Is_Directory))
	var implicit zip.File_Info
	status = zip.File_System_Status(&file_system, []byte("a"), &implicit)
	testify.Equal_Values(t, zip.STATUS_OK, status)
	testify.True(t, bool(implicit.Is_Directory))
	var file zip.File_Info
	status = zip.File_System_Status(&file_system, []byte("a/b/c"), &file)
	testify.Equal_Values(t, zip.STATUS_OK, status)
	testify.Equal(t, zip.File_Info_Size(len("content")), file.Size)

	var decoded [bytes.SLICE_SIZE_MAXIMUM]byte
	count, open_status := zip.File_System_Open(
		&file_system, []byte("a/b/c"), decoded[:],
	)
	testify.Equal(t, zip.STATUS_OK, open_status)
	testify.Equal(t, []byte("content"), decoded[:count])
	var entries [STANDARD_LIBRARY_FILE_SYSTEM_CAPACITY]zip.Directory_Entry
	directory_count, directory_status := zip.File_System_Read_Directory(
		&file_system, []byte("a"), entries[:],
	)
	testify.Equal_Values(t, zip.STATUS_OK, directory_status)
	testify.Equal_Values(t, bytes.Boundary(2), directory_count)
	testify.Equal(t, bytes.Slice("b"), entries[0].Name)
	testify.True(t, bool(entries[0].Is_Directory))
	testify.Equal(t, bytes.Slice("d"), entries[1].Name)
	testify.False(t, bool(entries[1].Is_Directory))
	standard_library_file_system_walk(t, &file_system)
	var nodes [STANDARD_LIBRARY_FILE_SYSTEM_CAPACITY]zip.File_Info
	zip.File_System_Walk(
		&file_system, []byte("a"), nodes[:],
		func(_ zip.File_Info) (keep_going binary.Boolean) { return true },
	)
}

// Test_Standard_Library_File_System_Insecure_Path ports insecure-path rejection.
func standard_library_file_system_insecure_path(t *testing.T) {
	var harness standard_library_stream_harness
	standard_library_stream_harness_init(t, &harness)
	header := zip.Header_Unvalidated{
		Name: []byte("../test.txt"), Method: zip.METHOD_STORE,
	}
	testify.Equal_Values(t, zip.STATUS_OK, zip.Writer_Create(&harness.Writer, &header))
	reader := standard_library_stream_harness_close_and_read(t, &harness)
	var file_system zip.File_System
	testify.Equal_Values(
		t, zip.STATUS_INPUT_INVALID, zip.File_System_Init(&file_system, &reader),
	)
}

// Test_Standard_Library_Writer_Add_File_System ports AddFS and implicit directories.
func standard_library_writer_add_file_system(t *testing.T) {
	var source standard_library_stream_harness
	standard_library_stream_harness_init(t, &source)
	standard_library_file_system_archive(t, &source)
	source_reader := standard_library_stream_harness_close_and_read(t, &source)
	var source_file_system zip.File_System
	testify.Equal_Values(
		t, zip.STATUS_OK,
		zip.File_System_Init(&source_file_system, &source_reader),
	)
	var destination standard_library_stream_harness
	standard_library_stream_harness_init(t, &destination)
	var nodes [STANDARD_LIBRARY_FILE_SYSTEM_CAPACITY]zip.File_Info
	testify.Equal_Values(
		t, zip.STATUS_OK,
		zip.Writer_Add_File_System(&destination.Writer, &source_file_system, nodes[:]),
	)
	destination_reader := standard_library_stream_harness_close_and_read(t, &destination)
	testify.Equal(t, zip.Entry_Count(5), destination_reader.Entry_Count)
	var destination_file_system zip.File_System
	testify.Equal_Values(
		t, zip.STATUS_OK,
		zip.File_System_Init(&destination_file_system, &destination_reader),
	)
	var decoded [bytes.SLICE_SIZE_MAXIMUM]byte
	count, status := zip.File_System_Open(
		&destination_file_system, []byte("a/b/c"), decoded[:],
	)
	testify.Equal_Values(t, zip.STATUS_OK, status)
	testify.Equal(t, []byte("content"), decoded[:count])
}

// Test_Standard_Library_Writer_Add_File_System_Rejects_Special ports issue 61875.
func standard_library_writer_add_file_system_rejects_special(t *testing.T) {
	var source standard_library_stream_harness
	standard_library_stream_harness_init(t, &source)
	header := zip.Header_Unvalidated{
		Name: []byte("symlink"), Method: zip.METHOD_STORE,
	}
	zip.Header_Set_Mode(&header, 0o755|nbio.FILE_MODE_SYMBOLIC_LINK)
	testify.Equal_Values(t, zip.STATUS_OK, zip.Writer_Create(&source.Writer, &header))
	_, status := zip.Writer_Write(&source.Writer, []byte("../target"))
	testify.Equal_Values(t, zip.STATUS_OK, status)
	source_reader := standard_library_stream_harness_close_and_read(t, &source)
	var file_system zip.File_System
	testify.Equal_Values(t, zip.STATUS_OK, zip.File_System_Init(&file_system, &source_reader))
	var destination standard_library_stream_harness
	standard_library_stream_harness_init(t, &destination)
	var nodes [STANDARD_LIBRARY_SPECIAL_NODE_CAPACITY]zip.File_Info
	testify.Equal_Values(
		t, zip.STATUS_INPUT_INVALID,
		zip.Writer_Add_File_System(&destination.Writer, &file_system, nodes[:]),
	)
}

// Test_Standard_Library_ZIP64_Bounds ports large size and record tests to bounded rejection.
func standard_library_zip64_bounds(t *testing.T) {
	var harness standard_library_stream_harness
	standard_library_stream_harness_init(t, &harness)
	large := zip.Header_Unvalidated{
		Name: []byte("huge"), Method: zip.METHOD_STORE,
		Compressed_Size: 1 << 32, Uncompressed_Size: 1 << 32,
	}
	testify.Equal_Values(
		t, zip.STATUS_INPUT_INVALID,
		zip.Writer_Create_Raw(&harness.Writer, &large),
	)
	header := zip.Header_Unvalidated{Name: []byte("a"), Method: zip.METHOD_STORE}
	status := zip.Creation_Status(zip.STATUS_OK)
	testify.Not_Panics(t, func() {
		for status == zip.Creation_Status(zip.STATUS_OK) {
			status = zip.Writer_Create(&harness.Writer, &header)
		}
	})
	testify.Equal_Values(t, zip.STATUS_OUTPUT_TOO_SMALL, status)
	testify.Less_Or_Equal(t, &testify.Less_Or_Equal_Input[zip.Entry_Count]{
		First: harness.Writer.Entry_Count, Second: zip.Entry_Count(zip.ENTRY_COUNT_MAXIMUM),
	})
}

// Bounded entry tests stored and deflated entries inside caller storage.
func bounded_entry(t *testing.T) {
	t.Parallel()
	const ARCHIVE_STORAGE_SIZE = STANDARD_LIBRARY_ARCHIVE_CAPACITY *
		STANDARD_LIBRARY_WORD_GROWTH_FACTOR
	const OUTPUT_STORAGE_SIZE = STANDARD_LIBRARY_ALLOCATION_OUTPUT_CAPACITY
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
			testify.Equal(t, zip.STATUS_OK, status, one.Name)
			testify.Equal(t, content, decoded[:count], one.Name)
		})
	}
}

// Allocation test proves every result and decompression path leaves heap untouched.
func allocation(t *testing.T) {
	const ARCHIVE_STORAGE_SIZE = STANDARD_LIBRARY_ARCHIVE_CAPACITY *
		STANDARD_LIBRARY_WORD_GROWTH_FACTOR
	const OUTPUT_STORAGE_SIZE = STANDARD_LIBRARY_ALLOCATION_OUTPUT_CAPACITY
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
	var unsupported_storage [ARCHIVE_STORAGE_SIZE]byte
	unsupported := archive_data(
		unsupported_storage[:], []byte("data.txt"), 99, content, content,
	)
	var decoded [OUTPUT_STORAGE_SIZE]byte
	var count bytes.Boundary
	var status zip.Status
	testify.Zero_Allocation(t, func() {
		count, status = zip.Decode_Into(nil, "data.txt", decoded[:])
	})
	testify.Zero_Allocation(t, func() {
		count, status = zip.Decode_Into(stored, "absent.txt", decoded[:])
	})
	testify.Zero_Allocation(t, func() {
		count, status = zip.Decode_Into(stored, "data.txt", decoded[:4])
	})
	testify.Zero_Allocation(t, func() {
		count, status = zip.Decode_Into(unsupported, "data.txt", decoded[:])
	})
	testify.Zero_Allocation(t, func() {
		count, status = zip.Decode_Into(stored, "data.txt", decoded[:])
	})
	testify.Zero_Allocation(t, func() {
		count, status = zip.Decode_Into(descriptor_archive, "data.txt", decoded[:])
	})
	testify.Zero_Allocation(t, func() {
		count, status = zip.Decode_Into(deflated_archive, "data.txt", decoded[:])
	})
	testify.Equal(t, zip.STATUS_OK, status)
	testify.Equal(t, bytes.Boundary(len(content)), count)
	allocation_header(t)
	allocation_reader(t, stored)
	allocation_file_system(t, stored)
	allocation_writer(t, stored)
}

func allocation_header(t *testing.T) {
	t.Helper()
	unvalidated := zip.Header_Unvalidated{
		Name: []byte("x"), Method: zip.METHOD_STORE,
	}
	validated := zip.Header{Name: []byte("x"), Method: zip.METHOD_STORE}
	modified := zip.Timestamp{Seconds: 315_532_800, Set: true}
	var header zip.Header
	var status zip.Validation_Status
	var mode zip.Archive_File_Mode
	var timestamp zip.Timestamp
	testify.Zero_Allocation(t, func() { header, status = zip.Header_Validate(&unvalidated) })
	testify.Zero_Allocation(t, func() {
		status = zip.Header_Set_Modification_Time(&unvalidated, modified)
	})
	testify.Zero_Allocation(t, func() { zip.Header_Set_Mode(&unvalidated, 0o644) })
	testify.Zero_Allocation(t, func() { mode = zip.Header_Mode(&validated) })
	testify.Zero_Allocation(t, func() {
		timestamp = zip.Header_Modification_Time(&validated)
	})
	testify.Equal_Values(t, zip.STATUS_OK, status)
	testify.Equal(t, zip.Header_Name([]byte("x")), header.Name)
	testify.Equal_Values(t, 0o666, mode)
	testify.False(t, bool(timestamp.Set))
}

func allocation_reader(t *testing.T, archive []byte) {
	t.Helper()
	memory := nbio.Stream_Memory{Memory: archive}
	var storage [bytes.SLICE_SIZE_MAXIMUM]byte
	var reader zip.Reader
	var completion time.Completion
	callback := func(_ *time.Completion) {}
	testify.Zero_Allocation(t, func() {
		reader = zip.Reader{}
		completion = time.Completion{}
		zip.Reader_Init(
			&reader, nbio.Memory_To_Stream(&memory),
			zip.Reader_Storage{Archive: storage[:]}, &completion, callback,
		)
	})
	reader = standard_library_reader_domain_archive(archive)
	var destination [STANDARD_LIBRARY_ALLOCATION_OUTPUT_CAPACITY]byte
	var header zip.Header
	var count bytes.Boundary
	var status zip.Status
	var found zip.Found_Status
	var raw zip.Reader_Raw_Content
	testify.Zero_Allocation(t, func() {
		count, status = zip.Reader_Decode(&reader, 0, destination[:])
	})
	testify.Zero_Allocation(t, func() { found = zip.Reader_Header(&reader, 0, &header) })
	testify.Zero_Allocation(t, func() { raw, found = zip.Reader_Raw(&reader, 0, &header) })
	testify.Equal(t, bytes.Boundary(len(raw)), count)
	testify.Equal(t, zip.STATUS_OK, status)
	testify.Equal_Values(t, zip.STATUS_OK, found)
	testify.Not_Empty(t, raw)
}

func allocation_file_system(t *testing.T, archive []byte) {
	t.Helper()
	reader := standard_library_reader_domain_archive(archive)
	var file_system zip.File_System
	var validation zip.Validation_Status
	testify.Zero_Allocation(t, func() {
		file_system = zip.File_System{}
		validation = zip.File_System_Init(&file_system, &reader)
	}, "File_System_Init")
	zip.File_System_Init(&file_system, &reader)
	var info zip.File_Info
	var destination [STANDARD_LIBRARY_ALLOCATION_OUTPUT_CAPACITY]byte
	var entries [STANDARD_LIBRARY_ALLOCATION_NODE_CAPACITY]zip.Directory_Entry
	var nodes [STANDARD_LIBRARY_ALLOCATION_NODE_CAPACITY]zip.File_Info
	var count bytes.Boundary
	var found zip.Found_Status
	var directory_count zip.Directory_Output_Count
	var walk_count zip.File_System_Walk_Count
	var status zip.Status
	var directory_status zip.Directory_Status
	path := []byte("data.txt")
	root := []byte(".")
	callback := func(_ zip.File_Info) (keep_going binary.Boolean) { return true }
	testify.Zero_Allocation(t, func() {
		found = zip.File_System_Status(&file_system, path, &info)
	}, "File_System_Status")
	testify.Zero_Allocation(t, func() {
		count, status = zip.File_System_Open(
			&file_system, path, destination[:],
		)
	}, "File_System_Open")
	testify.Zero_Allocation(t, func() {
		directory_count, directory_status = zip.File_System_Read_Directory(
			&file_system, root, entries[:],
		)
	}, "File_System_Read_Directory")
	testify.Zero_Allocation(t, func() {
		walk_count, directory_status = zip.File_System_Walk(
			&file_system, root, nodes[:], callback,
		)
	}, "File_System_Walk")
	testify.Equal_Values(t, zip.STATUS_OK, validation)
	testify.Equal_Values(t, zip.STATUS_OK, found)
	testify.Not_Zero(t, count)
	testify.Equal(t, zip.STATUS_OK, status)
	testify.Not_Zero(t, directory_count)
	testify.Not_Zero(t, walk_count)
	testify.Equal_Values(t, zip.STATUS_OK, directory_status)
}

func allocation_writer(t *testing.T, archive []byte) {
	t.Helper()
	reader := standard_library_reader_domain_archive(archive)
	var source_file_system zip.File_System
	zip.File_System_Init(&source_file_system, &reader)
	var harness standard_library_stream_harness
	standard_library_stream_harness_init(t, &harness)
	base := harness.Writer
	header := zip.Header_Unvalidated{Name: []byte("x"), Method: zip.METHOD_STORE}
	active := base
	zip.Writer_Create(&active, &header)
	var writer zip.Writer
	var capacity_status zip.Capacity_Status
	testify.Zero_Allocation(t, func() {
		writer = zip.Writer{}
		capacity_status = zip.Writer_Init(
			&writer, nbio.Memory_To_Stream(&harness.Memory), base.Storage,
		)
	}, "Writer_Init")
	allocation_writer_creation(t, base, active, header, reader)
	allocation_writer_metadata(t, base, active)
	allocation_writer_file_system(t, base, source_file_system)
	testify.Equal_Values(t, zip.STATUS_OK, capacity_status)
}

func allocation_writer_creation(
	t *testing.T, base zip.Writer, active zip.Writer,
	header zip.Header_Unvalidated, reader zip.Reader,
) {
	t.Helper()
	var writer zip.Writer
	var creation zip.Creation_Status
	var status zip.Status
	var count bytes.Boundary
	var bounded zip.Bounded_Status
	source := []byte("x")
	testify.Zero_Allocation(t, func() {
		writer = base
		creation = zip.Writer_Create(&writer, &header)
	}, "Writer_Create")
	testify.Zero_Allocation(t, func() {
		writer = base
		creation = zip.Writer_Create_Raw(&writer, &header)
	}, "Writer_Create_Raw")
	testify.Zero_Allocation(t, func() {
		writer = base
		status = zip.Writer_Copy(&writer, &reader, 0)
	}, "Writer_Copy")
	testify.Zero_Allocation(t, func() {
		writer = active
		count, bounded = zip.Writer_Write(&writer, source)
	}, "Writer_Write")
	testify.Equal_Values(t, zip.STATUS_OK, creation)
	testify.Equal(t, zip.STATUS_OK, status)
	testify.Equal(t, bytes.Boundary(1), count)
	testify.Equal_Values(t, zip.STATUS_OK, bounded)
}

func allocation_writer_metadata(t *testing.T, base zip.Writer, active zip.Writer) {
	t.Helper()
	var writer zip.Writer
	var status zip.Bounded_Status
	var validation zip.Validation_Status
	var completion time.Completion
	comment := []byte("x")
	callback := func(_ *time.Completion) {}
	testify.Zero_Allocation(t, func() {
		writer = base
		status = zip.Writer_Set_Comment(&writer, comment)
	}, "Writer_Set_Comment")
	testify.Zero_Allocation(t, func() {
		writer = base
		validation = zip.Writer_Set_Offset(&writer, 1)
	}, "Writer_Set_Offset")
	testify.Zero_Allocation(t, func() {
		writer = active
		completion = time.Completion{}
		zip.Writer_Close(&writer, &completion, callback)
	}, "Writer_Close")
	testify.Zero_Allocation(t, func() {
		writer = active
		completion = time.Completion{}
		zip.Writer_Flush(&writer, &completion, callback)
	}, "Writer_Flush")
	testify.Equal_Values(t, zip.STATUS_OK, status)
	testify.Equal_Values(t, zip.STATUS_OK, validation)
}

func allocation_writer_file_system(
	t *testing.T, base zip.Writer, file_system zip.File_System,
) {
	t.Helper()
	var writer zip.Writer
	var nodes [STANDARD_LIBRARY_ALLOCATION_NODE_CAPACITY]zip.File_Info
	var status zip.Creation_Status
	testify.Zero_Allocation(t, func() {
		writer = base
		status = zip.Writer_Add_File_System(&writer, &file_system, nodes[:])
	}, "Writer_Add_File_System")
	testify.Equal_Values(t, zip.STATUS_OK, status)
}

// Test_Standard_Library_Stream_APIs ports nbio reader, writer, header, and filesystem tests.
func Test_Standard_Library_Stream_APIs(t *testing.T) {
	standard_library_stream_round_trip(t)
	standard_library_writer_comment_utf8_offset(t)
	standard_library_writer_flush_directory(t)
	standard_library_writer_raw(t)
	standard_library_writer_copy(t)
	standard_library_header_mode_time(t)
	standard_library_header_extra_bounds(t)
	standard_library_file_system(t)
	standard_library_file_system_insecure_path(t)
	standard_library_writer_add_file_system(t)
	standard_library_writer_add_file_system_rejects_special(t)
	standard_library_zip64_bounds(t)
	standard_library_reader_output_domains(t)
	standard_library_writer_output_domains(t)
	standard_library_copy_add_domains(t)
	standard_library_finalize_domains(t)
}

func standard_library_reader_output_domains(t *testing.T) {
	t.Helper()
	for _, content_size := range []int{2, bytes.SLICE_SIZE_MAXIMUM - 100} {
		content := make([]byte, content_size)
		var archive_storage [bytes.SLICE_SIZE_MAXIMUM]byte
		archive := archive_data(
			archive_storage[:], []byte("x"), zip.METHOD_STORE, content, content,
		)
		reader := standard_library_reader_domain_archive(archive)
		var header zip.Header
		raw, status := zip.Reader_Raw(&reader, 0, &header)
		testify.Equal_Values(t, zip.STATUS_OK, status)
		testify.Equal(t, content_size, len(raw))
		if content_size == 2 {
			var destination [STANDARD_LIBRARY_SPECIAL_NODE_CAPACITY]byte
			count, decode_status := zip.Reader_Decode(&reader, 0, destination[:])
			testify.Equal(t, bytes.Boundary(2), count)
			testify.Equal(t, zip.STATUS_OK, decode_status)
			var file_system zip.File_System
			testify.Equal_Values(
				t, zip.STATUS_OK, zip.File_System_Init(&file_system, &reader),
			)
			count, decode_status = zip.File_System_Open(
				&file_system, []byte("x"), destination[:],
			)
			testify.Equal(t, bytes.Boundary(2), count)
			testify.Equal(t, zip.STATUS_OK, decode_status)
		}
	}
	content := make([]byte, bytes.SLICE_SIZE_MAXIMUM)
	compressed := deflated(t, content)
	var archive_storage [bytes.SLICE_SIZE_MAXIMUM]byte
	archive := archive_data(
		archive_storage[:], []byte("x"), zip.METHOD_DEFLATE,
		compressed, content,
	)
	reader := standard_library_reader_domain_archive(archive)
	destination := make([]byte, bytes.SLICE_SIZE_MAXIMUM)
	count, status := zip.Reader_Decode(&reader, 0, destination)
	testify.Equal(t, bytes.Boundary(bytes.SLICE_SIZE_MAXIMUM), count)
	testify.Equal(t, zip.STATUS_OK, status)
	var file_system zip.File_System
	testify.Equal_Values(
		t, zip.STATUS_OK, zip.File_System_Init(&file_system, &reader),
	)
	count, status = zip.File_System_Open(
		&file_system, []byte("x"), destination,
	)
	testify.Equal(t, bytes.Boundary(bytes.SLICE_SIZE_MAXIMUM), count)
	testify.Equal(t, zip.STATUS_OK, status)
}

func standard_library_writer_output_domains(t *testing.T) {
	t.Helper()
	for _, content_size := range []int{2, bytes.SLICE_SIZE_MAXIMUM} {
		var harness standard_library_stream_harness
		standard_library_stream_harness_init(t, &harness)
		header := zip.Header_Unvalidated{Name: []byte("x"), Method: zip.METHOD_STORE}
		testify.Equal_Values(t, zip.STATUS_OK, zip.Writer_Create(&harness.Writer, &header))
		count, status := zip.Writer_Write(
			&harness.Writer, make([]byte, content_size),
		)
		testify.Equal(t, bytes.Boundary(content_size), count)
		testify.Equal_Values(t, zip.STATUS_OK, status)
	}
	var harness standard_library_stream_harness
	standard_library_stream_harness_init(t, &harness)
	header := zip.Header_Unvalidated{Name: []byte("x"), Method: zip.METHOD_STORE}
	testify.Equal_Values(t, zip.STATUS_OK, zip.Writer_Create(&harness.Writer, &header))
	harness.Writer.Storage.Content = harness.Content[:1]
	_, write_status := zip.Writer_Write(&harness.Writer, []byte("xx"))
	testify.Equal_Values(t, zip.STATUS_OUTPUT_TOO_SMALL, write_status)
	harness.Writer.Storage.Comment = harness.Comment[:1]
	comment_status := zip.Writer_Set_Comment(&harness.Writer, []byte("xx"))
	testify.Equal_Values(t, zip.STATUS_OUTPUT_TOO_SMALL, comment_status)
}

func standard_library_copy_add_domains(t *testing.T) {
	t.Helper()
	var source standard_library_stream_harness
	standard_library_stream_harness_init(t, &source)
	header := zip.Header_Unvalidated{Name: []byte("x"), Method: zip.METHOD_STORE}
	testify.Equal_Values(t, zip.STATUS_OK, zip.Writer_Create(&source.Writer, &header))
	_, status := zip.Writer_Write(&source.Writer, []byte("x"))
	testify.Equal_Values(t, zip.STATUS_OK, status)
	reader := standard_library_stream_harness_close_and_read(t, &source)
	var file_system zip.File_System
	testify.Equal_Values(
		t, zip.STATUS_OK, zip.File_System_Init(&file_system, &reader),
	)
	var target standard_library_stream_harness
	standard_library_stream_harness_init(t, &target)
	testify.Equal(
		t, zip.STATUS_ENTRY_NOT_FOUND, zip.Writer_Copy(&target.Writer, &reader, 1),
	)
	var one [STANDARD_LIBRARY_SINGLE_NODE_CAPACITY]zip.File_Info
	testify.Equal_Values(
		t, zip.STATUS_OUTPUT_TOO_SMALL,
		zip.Writer_Add_File_System(&target.Writer, &file_system, one[:]),
	)
	standard_library_stream_harness_init(t, &target)
	var nodes [bytes.SLICE_SIZE_MAXIMUM]zip.File_Info
	testify.Equal_Values(
		t, zip.STATUS_OK,
		zip.Writer_Add_File_System(&target.Writer, &file_system, nodes[:]),
	)
	var unsupported_storage [STANDARD_LIBRARY_ARCHIVE_CAPACITY]byte
	unsupported_archive := archive_data(
		unsupported_storage[:], []byte("x"), 99, []byte("x"), []byte("x"),
	)
	unsupported_reader := standard_library_reader_domain_archive(unsupported_archive)
	standard_library_stream_harness_init(t, &target)
	testify.Equal(
		t, zip.STATUS_METHOD_UNSUPPORTED,
		zip.Writer_Copy(&target.Writer, &unsupported_reader, 0),
	)
	var unsupported_file_system zip.File_System
	testify.Equal_Values(
		t, zip.STATUS_OK,
		zip.File_System_Init(&unsupported_file_system, &unsupported_reader),
	)
	standard_library_stream_harness_init(t, &target)
	testify.Equal_Values(
		t, zip.STATUS_METHOD_UNSUPPORTED,
		zip.Writer_Add_File_System(
			&target.Writer, &unsupported_file_system, nodes[:],
		),
	)
	var destination [STANDARD_LIBRARY_SINGLE_NODE_CAPACITY]byte
	_, decode_status := zip.Reader_Decode(
		&unsupported_reader, 0, destination[:],
	)
	testify.Equal(t, zip.STATUS_METHOD_UNSUPPORTED, decode_status)
	_, decode_status = zip.File_System_Open(
		&unsupported_file_system, []byte("x"), destination[:],
	)
	testify.Equal(t, zip.STATUS_METHOD_UNSUPPORTED, decode_status)
}

func standard_library_finalize_domains(t *testing.T) {
	t.Helper()
	var harness standard_library_stream_harness
	standard_library_stream_harness_init(t, &harness)
	header := zip.Header_Unvalidated{
		Name: []byte("x"), Method: zip.METHOD_STORE,
		Compressed_Size: 2, Uncompressed_Size: 2,
	}
	testify.Equal_Values(t, zip.STATUS_OK, zip.Writer_Create_Raw(&harness.Writer, &header))
	_, status := zip.Writer_Write(&harness.Writer, []byte("x"))
	testify.Equal_Values(t, zip.STATUS_OK, status)
	var completion time.Completion
	zip.Writer_Flush(&harness.Writer, &completion, func(_ *time.Completion) {})
	testify.Equal(t, zip.STATUS_INPUT_INVALID, harness.Writer.Status)
	standard_library_stream_harness_init(t, &harness)
	harness.Writer.Storage.Archive = harness.Archive[:31]
	header = zip.Header_Unvalidated{Name: []byte("x"), Method: zip.METHOD_STORE}
	testify.Equal_Values(t, zip.STATUS_OK, zip.Writer_Create(&harness.Writer, &header))
	_, status = zip.Writer_Write(&harness.Writer, []byte("x"))
	testify.Equal_Values(t, zip.STATUS_OK, status)
	zip.Writer_Flush(&harness.Writer, &completion, func(_ *time.Completion) {})
	testify.Equal(t, zip.STATUS_OUTPUT_TOO_SMALL, harness.Writer.Status)
}

// Test_Invariant_Domains exercises public boundaries at every scalar witness.
func Test_Invariant_Domains(t *testing.T) {
	for _, marker := range []int{0, 1, 2, INVARIANT_DOMAIN_MAXIMUM} {
		testify.Not_Panics(t, func() {
			standard_library_writer_domain(marker)
			standard_library_writer_field_domains(marker)
			standard_library_header_field_domains(marker)
			standard_library_reader_domain(marker)
			standard_library_live_domains(marker)
			standard_library_reader_header_field_domains(marker)
			standard_library_file_system_domain(marker)
			standard_library_header_domain(marker)
		}, marker)
	}
	testify.Not_Panics(t, standard_library_writer_record_domains)
	testify.Not_Panics(t, standard_library_writer_central_size_domains)
	testify.Not_Panics(t, standard_library_reader_header_tail_domains)
	testify.Not_Panics(t, standard_library_rejected_header_domain)
	testify.Not_Panics(t, standard_library_negative_one_timestamp_domains)
	testify.Not_Panics(t, standard_library_negative_one_writer_domains)
	testify.Not_Panics(t, standard_library_timestamp_domains)
	testify.Not_Panics(t, standard_library_civil_domains)
	testify.Not_Panics(t, standard_library_mode_domains)
	testify.Not_Panics(t, standard_library_writer_timestamp_domains)
	testify.Not_Panics(t, standard_library_file_system_path_domains)
	testify.Not_Panics(t, standard_library_file_system_count_domains)
}

func standard_library_live_domains(marker int) {
	var archive_storage [bytes.SLICE_SIZE_MAXIMUM]byte
	archive := archive_data(
		archive_storage[:], []byte("x"), zip.METHOD_STORE, []byte("x"), []byte("x"),
	)
	reader := standard_library_reader_domain_archive(archive)
	destination := make([]byte, standard_library_domain_size(marker))
	zip.Reader_Decode(
		&reader, standard_library_domain_entry_count(marker), destination,
	)
	var header zip.Header
	zip.Reader_Header(&reader, standard_library_domain_entry_count(marker), &header)
	zip.Reader_Raw(&reader, standard_library_domain_entry_count(marker), &header)
	var file_system zip.File_System
	zip.File_System_Init(&file_system, &reader)
	path := make([]byte, standard_library_domain_size(marker))
	info := standard_library_file_info_domain_value(marker)
	zip.File_System_Status(&file_system, path, &info)
	zip.File_System_Status(&file_system, []byte("x"), &info)
	zip.File_System_Open(&file_system, []byte("x"), destination)
	entry_count := standard_library_domain_size(marker)
	if marker == INVARIANT_DOMAIN_MAXIMUM {
		entry_count = zip.ENTRY_COUNT_MAXIMUM
	}
	entries := make([]zip.Directory_Entry, entry_count)
	zip.File_System_Read_Directory(&file_system, []byte("."), entries)
	nodes := make([]zip.File_Info, standard_library_domain_size(marker))
	zip.File_System_Walk(
		&file_system, []byte("."), nodes,
		func(_ zip.File_Info) (keep_going binary.Boolean) { return marker != 0 },
	)
}

// Test_Deflate_Forms covers stored, Huffman-only, and repeated-span DEFLATE blocks.
func Test_Deflate_Forms(t *testing.T) {
	t.Parallel()
	const ARCHIVE_STORAGE_SIZE = bytes.SLICE_SIZE_MAXIMUM * STANDARD_LIBRARY_WORD_32_SIZE
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
		{Name: "stored blocks", Level: flate.NO_COMPRESSION},
		{Name: "Huffman only", Level: flate.HUFFMAN_ONLY},
		{Name: "repeated spans", Level: flate.BEST_COMPRESSION},
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
			testify.Equal(t, zip.STATUS_OK, status, one.Name)
			testify.Equal(t, content, decoded[:count], one.Name)
		})
	}
}

// Test_Data_Descriptor preserves streaming-writer metadata layout.
func Test_Data_Descriptor(t *testing.T) {
	t.Parallel()
	const ARCHIVE_STORAGE_SIZE = STANDARD_LIBRARY_ARCHIVE_CAPACITY *
		STANDARD_LIBRARY_WORD_GROWTH_FACTOR
	const OUTPUT_STORAGE_SIZE = STANDARD_LIBRARY_ALLOCATION_OUTPUT_CAPACITY
	content := []byte("hello bounded zip world")
	compressed := deflated(t, content)
	var archive_storage [ARCHIVE_STORAGE_SIZE]byte
	archive := descriptor_archive_data(
		archive_storage[:], []byte("data.txt"), compressed, content,
	)
	var decoded [OUTPUT_STORAGE_SIZE]byte
	count, status := zip.Decode_Into(archive, "data.txt", decoded[:])
	testify.Equal(t, zip.STATUS_OK, status)
	testify.Equal(t, content, decoded[:count])
}

// Test_Data_Descriptor_Rejection prevents central metadata masking corrupt stream trailer.
func Test_Data_Descriptor_Rejection(t *testing.T) {
	t.Parallel()
	const ARCHIVE_STORAGE_SIZE = STANDARD_LIBRARY_ARCHIVE_CAPACITY *
		STANDARD_LIBRARY_WORD_GROWTH_FACTOR
	const OUTPUT_STORAGE_SIZE = STANDARD_LIBRARY_ALLOCATION_OUTPUT_CAPACITY
	const LOCAL_HEADER_SIZE = STANDARD_LIBRARY_LOCAL_HEADER_SIZE
	const DESCRIPTOR_CHECKSUM_POSITION = STANDARD_LIBRARY_DESCRIPTOR_CHECKSUM_POSITION
	content := []byte("hello bounded zip world")
	compressed := deflated(t, content)
	name := []byte("data.txt")
	var archive_storage [ARCHIVE_STORAGE_SIZE]byte
	archive := descriptor_archive_data(
		archive_storage[:], name, compressed, content,
	)
	descriptor_position := LOCAL_HEADER_SIZE + len(name) + len(compressed)
	archive[descriptor_position+DESCRIPTOR_CHECKSUM_POSITION] ^= 1
	var decoded [OUTPUT_STORAGE_SIZE]byte
	count, status := zip.Decode_Into(archive, "data.txt", decoded[:])
	testify.Zero(t, count)
	testify.Equal(t, zip.STATUS_INPUT_INVALID, status)
}

// Test_Deflate_Corpus covers codec choices across distinct byte distributions.
func Test_Deflate_Corpus(t *testing.T) {
	t.Parallel()
	const DATA_SIZE = STANDARD_LIBRARY_ARCHIVE_CAPACITY * STANDARD_LIBRARY_WORD_32_SIZE
	const CORPUS_COUNT = STANDARD_LIBRARY_WORD_32_SIZE
	const ARCHIVE_STORAGE_SIZE = bytes.SLICE_SIZE_MAXIMUM
	var corpus [CORPUS_COUNT][DATA_SIZE]byte
	for index := 0; index < DATA_SIZE; index++ {
		corpus[0][index] = 'A'
		corpus[1][index] = byte(index)
		corpus[2][index] = byte(index*73 + 19)
		corpus[3][index] = byte((index/17 + index/113) % 11)
	}
	levels := [...]int{
		flate.NO_COMPRESSION, flate.BEST_SPEED, flate.DEFAULT_COMPRESSION,
		flate.BEST_COMPRESSION,
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
			testify.Equal(t, zip.STATUS_OK, status, corpus_index, level)
			testify.Equal(t, content, decoded[:count], corpus_index, level)
		}
	}
}

// Test_Deflate_Rejection proves corrupt bitstreams remain scalar failures.
func Test_Deflate_Rejection(t *testing.T) {
	t.Parallel()
	const ARCHIVE_STORAGE_SIZE = bytes.SLICE_SIZE_MAXIMUM /
		STANDARD_LIBRARY_WORD_GROWTH_FACTOR
	const OUTPUT_STORAGE_SIZE = STANDARD_LIBRARY_ARCHIVE_CAPACITY *
		STANDARD_LIBRARY_WORD_GROWTH_FACTOR
	var content_storage [OUTPUT_STORAGE_SIZE]byte
	content_count := bytes.Repeat_Into(
		content_storage[:], []byte("bounded corrupt stream "), 16,
	)
	content := content_storage[:content_count]
	compressed := deflated_level(t, content, flate.BEST_COMPRESSION)
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
			testify.Equal(t, content, decoded[:count], compressed_index)
		} else {
			testify.Equal(t, zip.STATUS_INPUT_INVALID, status, compressed_index)
		}
		archive[data_position+compressed_index] ^= 1
	}
}

// Test_Malformed_Metadata verifies each single-byte mutation remains bounded.
func Test_Malformed_Metadata(t *testing.T) {
	t.Parallel()
	const ARCHIVE_STORAGE_SIZE = STANDARD_LIBRARY_ARCHIVE_CAPACITY *
		STANDARD_LIBRARY_WORD_GROWTH_FACTOR
	const OUTPUT_STORAGE_SIZE = STANDARD_LIBRARY_ALLOCATION_OUTPUT_CAPACITY
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
			testify.Equal(t, content, decoded[:count], mutation_index)
		case zip.STATUS_INPUT_INVALID,
			zip.STATUS_ENTRY_NOT_FOUND,
			zip.STATUS_OUTPUT_TOO_SMALL,
			zip.STATUS_METHOD_UNSUPPORTED:
			testify.Zero(t, count, mutation_index)
		default:
			testify.Fail_Now(
				t, "mutation returned unknown status", mutation_index, status,
			)
		}
		archive[mutation_index] ^= 0xff
	}
}

// Test_Central_Directory_Completion prevents selected member hiding malformed followers.
func Test_Central_Directory_Completion(t *testing.T) {
	t.Parallel()
	const ARCHIVE_STORAGE_SIZE = STANDARD_LIBRARY_ARCHIVE_CAPACITY *
		STANDARD_LIBRARY_WORD_GROWTH_FACTOR
	const OUTPUT_STORAGE_SIZE = STANDARD_LIBRARY_ALLOCATION_OUTPUT_CAPACITY
	const DIRECTORY_END_SIZE = STANDARD_LIBRARY_DIRECTORY_END_SIZE
	const DIRECTORY_DISK_ENTRY_COUNT_POSITION = ZIP_DIRECTORY_DISK_ENTRY_COUNT_POSITION
	const DIRECTORY_ENTRY_COUNT_POSITION = STANDARD_LIBRARY_DIRECTORY_ENTRY_COUNT_POSITION
	content := []byte("hello bounded zip world")
	var archive_storage [ARCHIVE_STORAGE_SIZE]byte
	archive := archive_data(
		archive_storage[:], []byte("data.txt"), 0, content, content,
	)
	directory_end := archive[len(archive)-DIRECTORY_END_SIZE:]
	put_word_16(directory_end[DIRECTORY_DISK_ENTRY_COUNT_POSITION:], 2)
	put_word_16(directory_end[DIRECTORY_ENTRY_COUNT_POSITION:], 2)
	var decoded [OUTPUT_STORAGE_SIZE]byte
	count, status := zip.Decode_Into(archive, "data.txt", decoded[:])
	testify.Zero(t, count)
	testify.Equal(t, zip.STATUS_INPUT_INVALID, status)
}

// Test_Central_Directory_Bounds keeps hostile metadata work inside one archive boundary.
func Test_Central_Directory_Bounds(t *testing.T) {
	t.Parallel()
	const CENTRAL_HEADER_SIZE = STANDARD_LIBRARY_CENTRAL_HEADER_SIZE
	const DIRECTORY_END_SIZE = STANDARD_LIBRARY_DIRECTORY_END_SIZE
	const CENTRAL_TAIL_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM -
		CENTRAL_HEADER_SIZE - DIRECTORY_END_SIZE
	for _, sizes := range []struct {
		Name    int
		Extra   int
		Comment int
	}{
		{},
		{Name: 1},
		{Name: 2},
		{Extra: 1},
		{Extra: 2},
		{Comment: 1},
		{Comment: 2},
		{Name: CENTRAL_TAIL_SIZE_MAXIMUM},
		{Extra: CENTRAL_TAIL_SIZE_MAXIMUM},
		{Comment: CENTRAL_TAIL_SIZE_MAXIMUM},
	} {
		var archive_storage [bytes.SLICE_SIZE_MAXIMUM]byte
		archive := central_archive_data(
			archive_storage[:], sizes.Name, sizes.Extra, sizes.Comment,
		)
		count, status := zip.Decode_Into(archive, "x", nil)
		testify.Zero(t, count, sizes)
		testify.Equal(t, zip.STATUS_ENTRY_NOT_FOUND, status, sizes)
	}
	const ENTRY_COUNT_MAXIMUM = (bytes.SLICE_SIZE_MAXIMUM -
		DIRECTORY_END_SIZE) / CENTRAL_HEADER_SIZE
	for _, entry_count := range []int{2, ENTRY_COUNT_MAXIMUM} {
		var archive_storage [bytes.SLICE_SIZE_MAXIMUM]byte
		archive := central_entries_archive_data(
			archive_storage[:], entry_count,
		)
		count, status := zip.Decode_Into(archive, "x", nil)
		testify.Zero(t, count, entry_count)
		testify.Equal(t, zip.STATUS_ENTRY_NOT_FOUND, status, entry_count)
	}
}

// Test_Central_Field_Size_Rejection keeps raw 16-bit lengths outside trusted bounds.
func Test_Central_Field_Size_Rejection(t *testing.T) {
	t.Parallel()
	const ARCHIVE_STORAGE_SIZE = STANDARD_LIBRARY_ARCHIVE_CAPACITY
	const LOCAL_HEADER_SIZE = STANDARD_LIBRARY_LOCAL_HEADER_SIZE
	const CENTRAL_NAME_SIZE_POSITION = STANDARD_LIBRARY_CENTRAL_NAME_SIZE_POSITION
	const CENTRAL_EXTRA_SIZE_POSITION = STANDARD_LIBRARY_CENTRAL_EXTRA_SIZE_POSITION
	const CENTRAL_COMMENT_SIZE_POSITION = STANDARD_LIBRARY_CENTRAL_COMMENT_SIZE_POSITION
	name := []byte("x")
	for _, field_position := range []int{
		CENTRAL_NAME_SIZE_POSITION,
		CENTRAL_EXTRA_SIZE_POSITION,
		CENTRAL_COMMENT_SIZE_POSITION,
	} {
		var archive_storage [ARCHIVE_STORAGE_SIZE]byte
		archive := archive_data(
			archive_storage[:], name, 0, nil, nil,
		)
		central_position := LOCAL_HEADER_SIZE + len(name)
		put_word_16(archive[central_position+field_position:], 1<<16-1)
		count, status := zip.Decode_Into(archive, "x", nil)
		testify.Zero(t, count, field_position)
		testify.Equal(t, zip.STATUS_INPUT_INVALID, status, field_position)
	}
}

// Test_Oversized_Selected_Name_Rejection rejects central-only names before local trust.
func Test_Oversized_Selected_Name_Rejection(t *testing.T) {
	t.Parallel()
	const CENTRAL_HEADER_SIZE = STANDARD_LIBRARY_CENTRAL_HEADER_SIZE
	const DIRECTORY_END_SIZE = STANDARD_LIBRARY_DIRECTORY_END_SIZE
	const NAME_SIZE = bytes.SLICE_SIZE_MAXIMUM -
		CENTRAL_HEADER_SIZE - DIRECTORY_END_SIZE
	var archive_storage [bytes.SLICE_SIZE_MAXIMUM]byte
	archive := central_archive_data(archive_storage[:], NAME_SIZE, 0, 0)
	archive[CENTRAL_HEADER_SIZE+NAME_SIZE-1] = 'x'
	count, status := zip.Decode_Into(archive, "x", nil)
	testify.Zero(t, count)
	testify.Equal(t, zip.STATUS_INPUT_INVALID, status)
}

// Test_Parser_Bounds keeps every staged cursor inside its tighter structural domain.
func Test_Parser_Bounds(t *testing.T) {
	t.Parallel()
	assert_directory_end_bounds(t)
	assert_selected_entry_bounds(t)
	assert_payload_bounds(t)
	assert_descriptor_bounds(t)
	assert_reader_mutation_rejected(t)
}

func assert_directory_end_bounds(t *testing.T) {
	t.Helper()
	const DIRECTORY_POSITION_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM -
		STANDARD_LIBRARY_DIRECTORY_END_SIZE
	for _, one := range []struct {
		Prefix         int
		Central_Size   uint32
		Central_Offset uint32
	}{
		{},
		{Prefix: 1, Central_Size: 1, Central_Offset: 1},
		{Prefix: 2, Central_Size: 2, Central_Offset: 2},
		{Prefix: DIRECTORY_POSITION_MAXIMUM, Central_Offset: bytes.SLICE_SIZE_MAXIMUM},
	} {
		var archive_storage [bytes.SLICE_SIZE_MAXIMUM]byte
		archive := directory_end_archive_data(
			archive_storage[:], one.Prefix,
			one.Central_Size, one.Central_Offset,
		)
		count, status := zip.Decode_Into(archive, "x", nil)
		testify.Zero(t, count, one)
		testify.Not_Equal(t, zip.STATUS_OK, status, one)
		reader := standard_library_reader_domain_archive(archive)
		var header zip.Header
		zip.Reader_Header(&reader, 0, &header)
		zip.Reader_Raw(&reader, 0, &header)
	}
}

func assert_selected_entry_bounds(t *testing.T) {
	t.Helper()
	const SELECTED_CENTRAL_START_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM -
		STANDARD_LIBRARY_CENTRAL_HEADER_SIZE - STANDARD_LIBRARY_NAME_SIZE_MINIMUM -
		STANDARD_LIBRARY_DIRECTORY_END_SIZE
	for _, prefix_size := range []int{0, 1, 2, SELECTED_CENTRAL_START_MAXIMUM} {
		var archive_storage [bytes.SLICE_SIZE_MAXIMUM]byte
		archive := selected_central_archive_data(
			archive_storage[:], prefix_size, true,
		)
		count, status := zip.Decode_Into(archive, "x", nil)
		testify.Zero(t, count, prefix_size)
		testify.Equal(t, zip.STATUS_INPUT_INVALID, status, prefix_size)
		reader := standard_library_reader_domain_archive(archive)
		var header zip.Header
		zip.Reader_Header(&reader, 0, &header)
		zip.Reader_Raw(&reader, 0, &header)
	}
	for _, prefix_size := range []int{0, 1, 2} {
		var archive_storage [bytes.SLICE_SIZE_MAXIMUM]byte
		archive := selected_central_archive_data(
			archive_storage[:], prefix_size, false,
		)
		zip.Decode_Into(archive, "x", nil)
	}
	var archive_storage [bytes.SLICE_SIZE_MAXIMUM]byte
	archive := maximum_central_cursor_archive_data(archive_storage[:])
	zip.Decode_Into(archive, "x", nil)
	reader := standard_library_reader_domain_archive(archive)
	var header zip.Header
	zip.Reader_Header(&reader, 1, &header)
	zip.Reader_Raw(&reader, 1, &header)
}

func assert_payload_bounds(t *testing.T) {
	t.Helper()
	const MINIMUM_ARCHIVE_SIZE = STANDARD_LIBRARY_MINIMUM_ARCHIVE_SIZE
	const PREFIX_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM - MINIMUM_ARCHIVE_SIZE
	var archive_storage [bytes.SLICE_SIZE_MAXIMUM]byte
	archive := prefixed_empty_archive_data(
		archive_storage[:], PREFIX_SIZE_MAXIMUM,
	)
	count, status := zip.Decode_Into(archive, "x", nil)
	testify.Equal(t, zip.STATUS_OK, status)
	testify.Zero(t, count)
}

func assert_descriptor_bounds(t *testing.T) {
	t.Helper()
	const MINIMUM_ARCHIVE_SIZE = STANDARD_LIBRARY_MINIMUM_DESCRIPTOR_ARCHIVE_SIZE
	for _, prefix_size := range []int{
		0, bytes.SLICE_SIZE_MAXIMUM - MINIMUM_ARCHIVE_SIZE,
	} {
		var archive_storage [bytes.SLICE_SIZE_MAXIMUM]byte
		archive := prefixed_stored_descriptor_archive_data(
			archive_storage[:], prefix_size,
		)
		count, status := zip.Decode_Into(archive, "x", nil)
		testify.Equal(t, zip.STATUS_OK, status, prefix_size)
		testify.Zero(t, count, prefix_size)
	}
}

func assert_reader_mutation_rejected(t *testing.T) {
	t.Helper()
	var archive_storage [STANDARD_LIBRARY_ARCHIVE_CAPACITY]byte
	archive := archive_data(
		archive_storage[:], []byte("x"), zip.METHOD_STORE, nil, nil,
	)
	reader := standard_library_reader_domain_archive(archive)
	var file_system zip.File_System
	testify.Equal_Values(
		t, zip.STATUS_OK, zip.File_System_Init(&file_system, &reader),
	)
	reader.Archive[31] = 0
	var header zip.Header
	testify.Equal_Values(
		t, zip.STATUS_INPUT_INVALID, zip.Reader_Header(&reader, 0, &header),
	)
	_, status := zip.Reader_Raw(&reader, 0, &header)
	testify.Equal_Values(t, zip.STATUS_INPUT_INVALID, status)
	var nodes [STANDARD_LIBRARY_SPECIAL_NODE_CAPACITY]zip.File_Info
	_, walk_status := zip.File_System_Walk(
		&file_system, []byte("."), nodes[:],
		func(_ zip.File_Info) (keep_going binary.Boolean) { return true },
	)
	testify.Equal_Values(t, zip.STATUS_INPUT_INVALID, walk_status)
}

// Test_Empty_Entry keeps zero-byte output distinct from missing output storage.
func Test_Empty_Entry(t *testing.T) {
	t.Parallel()
	const ARCHIVE_STORAGE_SIZE = STANDARD_LIBRARY_ARCHIVE_CAPACITY
	var archive_storage [ARCHIVE_STORAGE_SIZE]byte
	archive := archive_data(
		archive_storage[:], []byte("empty.txt"), 0, nil, nil,
	)
	count, status := zip.Decode_Into(archive, "empty.txt", nil)
	testify.Equal(t, zip.STATUS_OK, status)
	testify.Zero(t, count)
}

// Test_Rejection verifies hostile metadata cannot escape source or destination bounds.
func Test_Rejection(t *testing.T) {
	t.Parallel()
	const ARCHIVE_STORAGE_SIZE = STANDARD_LIBRARY_ARCHIVE_CAPACITY *
		STANDARD_LIBRARY_WORD_GROWTH_FACTOR
	const OUTPUT_STORAGE_SIZE = STANDARD_LIBRARY_ALLOCATION_OUTPUT_CAPACITY
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
		testify.Zero(t, count, end_index)
		testify.Equal(t, zip.STATUS_INPUT_INVALID, status, end_index)
	}
	count, status := zip.Decode_Into(
		archive, "absent.txt", decoded[:],
	)
	testify.Zero(t, count)
	testify.Equal(t, zip.STATUS_ENTRY_NOT_FOUND, status)
	count, status = zip.Decode_Into(
		archive, "data.txt", decoded[:4],
	)
	testify.Zero(t, count)
	testify.Equal(t, zip.STATUS_OUTPUT_TOO_SMALL, status)
	count, status = zip.Decode_Into(
		archive[:len(archive)-1], "data.txt", decoded[:],
	)
	testify.Zero(t, count)
	testify.Equal(t, zip.STATUS_INPUT_INVALID, status)
	corrupted := archive_data(
		archive_storage[:], []byte("data.txt"), 0, content, content,
	)
	corrupted[30+len("data.txt")] ^= 1
	count, status = zip.Decode_Into(
		corrupted, "data.txt", decoded[:],
	)
	testify.Zero(t, count)
	testify.Equal(t, zip.STATUS_INPUT_INVALID, status)
	unsupported := archive_data(
		archive_storage[:], []byte("data.txt"), 99, content, content,
	)
	count, status = zip.Decode_Into(
		unsupported, "data.txt", decoded[:],
	)
	testify.Zero(t, count)
	testify.Equal(t, zip.STATUS_METHOD_UNSUPPORTED, status)
	assert_invalid_suffixes(t, archive, decoded[:])
}

// Test_Format_Rejection keeps encrypted, multi-disk, and ZIP64 state outside classic trust.
func Test_Format_Rejection(t *testing.T) {
	t.Parallel()
	const ARCHIVE_STORAGE_SIZE = STANDARD_LIBRARY_ARCHIVE_CAPACITY
	const LOCAL_HEADER_SIZE = STANDARD_LIBRARY_LOCAL_HEADER_SIZE
	const CENTRAL_HEADER_SIZE = STANDARD_LIBRARY_CENTRAL_HEADER_SIZE
	const NAME_SIZE = STANDARD_LIBRARY_NAME_SIZE_MINIMUM
	const CENTRAL_POSITION = LOCAL_HEADER_SIZE + NAME_SIZE
	const DIRECTORY_END_POSITION = CENTRAL_POSITION + CENTRAL_HEADER_SIZE + NAME_SIZE
	for _, one := range []struct {
		Name       string
		Position   int
		Field_Size int
		Value      uint32
	}{
		{Name: "encrypted", Position: CENTRAL_POSITION + 8, Field_Size: 2, Value: 1},
		{Name: "multi-disk", Position: DIRECTORY_END_POSITION + 4, Field_Size: 2, Value: 1},
		{Name: "ZIP64", Position: CENTRAL_POSITION + 20, Field_Size: 4, Value: 1<<32 - 1},
	} {
		t.Run(one.Name, func(t *testing.T) {
			t.Parallel()
			var archive_storage [ARCHIVE_STORAGE_SIZE]byte
			archive := archive_data(
				archive_storage[:], []byte("x"), 0, nil, nil,
			)
			if one.Field_Size == 2 {
				put_word_16(archive[one.Position:], uint16(one.Value))
			} else {
				put_word_32(archive[one.Position:], one.Value)
			}
			count, status := zip.Decode_Into(archive, "x", nil)
			testify.Zero(t, count)
			testify.Equal(t, zip.STATUS_INPUT_INVALID, status)
		})
	}
}

// Test_Aliased_Storage prevents output mutation from changing untrusted archive bytes.
func Test_Aliased_Storage(t *testing.T) {
	t.Parallel()
	const ARCHIVE_STORAGE_SIZE = STANDARD_LIBRARY_ARCHIVE_CAPACITY *
		STANDARD_LIBRARY_WORD_GROWTH_FACTOR
	const LOCAL_HEADER_SIZE = STANDARD_LIBRARY_LOCAL_HEADER_SIZE
	content := []byte("hello bounded zip world")
	name := []byte("data.txt")
	var archive_storage [ARCHIVE_STORAGE_SIZE]byte
	archive := archive_data(
		archive_storage[:], name, 0, content, content,
	)
	data_position := LOCAL_HEADER_SIZE + len(name)
	count, status := zip.Decode_Into(
		archive, "data.txt", archive[data_position:data_position+len(content)],
	)
	testify.Zero(t, count)
	testify.Equal(t, zip.STATUS_INPUT_INVALID, status)
}

// Test_Public_Domain_Bounds keeps every shared byte boundary observed through public API.
func Test_Public_Domain_Bounds(t *testing.T) {
	t.Parallel()
	assert_public_content_bounds(t)
	assert_public_name_bounds(t)
	var hostile_source [bytes.SLICE_SIZE_MAXIMUM]byte
	var maximum_destination [bytes.SLICE_SIZE_MAXIMUM]byte
	zip.Decode_Into(hostile_source[:], "x", maximum_destination[:1])
	zip.Decode_Into(hostile_source[:], "xy", maximum_destination[:2])
}

func assert_public_content_bounds(t *testing.T) {
	t.Helper()
	const SMALL_ARCHIVE_STORAGE_SIZE = STANDARD_LIBRARY_ARCHIVE_CAPACITY
	const SMALL_CONTENT_SIZE_MAXIMUM = STANDARD_LIBRARY_WORD_16_SIZE
	for _, content := range []bytes.Slice{{'a'}, {'a', 'b'}} {
		var archive_storage [SMALL_ARCHIVE_STORAGE_SIZE]byte
		archive := archive_data(
			archive_storage[:], []byte("data.txt"), 0, content, content,
		)
		var destination [SMALL_CONTENT_SIZE_MAXIMUM]byte
		count, status := zip.Decode_Into(
			archive, "data.txt", destination[:len(content)],
		)
		testify.Equal(t, zip.STATUS_OK, status, content)
		testify.Equal(t, bytes.Boundary(len(content)), count, content)
	}
	var maximum_content [bytes.SLICE_SIZE_MAXIMUM]byte
	for position := range maximum_content {
		maximum_content[position] = 'a'
	}
	compressed := deflated_level(
		t, maximum_content[:], flate.BEST_COMPRESSION,
	)
	var maximum_archive_storage [bytes.SLICE_SIZE_MAXIMUM]byte
	maximum_archive := archive_data(
		maximum_archive_storage[:], []byte("data.txt"), 8,
		compressed, maximum_content[:],
	)
	var maximum_destination [bytes.SLICE_SIZE_MAXIMUM]byte
	count, status := zip.Decode_Into(
		maximum_archive, "data.txt", maximum_destination[:],
	)
	testify.Equal(t, zip.STATUS_OK, status)
	testify.Equal(t, bytes.Boundary(bytes.SLICE_SIZE_MAXIMUM), count)
	zip.Decode_Into(maximum_archive, "x", maximum_destination[:1])
	zip.Decode_Into(maximum_archive, "xy", maximum_destination[:2])
}

func assert_public_name_bounds(t *testing.T) {
	t.Helper()
	const SMALL_ARCHIVE_STORAGE_SIZE = STANDARD_LIBRARY_ARCHIVE_CAPACITY
	for _, name := range []bytes.Slice{{'x'}, {'x', 'y'}} {
		var archive_storage [SMALL_ARCHIVE_STORAGE_SIZE]byte
		archive := archive_data(
			archive_storage[:], name, 0, nil, nil,
		)
		count, status := zip.Decode_Into(
			archive, bytes.Text(string(name)), nil,
		)
		testify.Equal(t, zip.STATUS_OK, status, name)
		testify.Zero(t, count, name)
	}
	const MEMBER_NAME_SIZE_MAXIMUM = (bytes.SLICE_SIZE_MAXIMUM -
		STANDARD_LIBRARY_LOCAL_HEADER_SIZE - STANDARD_LIBRARY_CENTRAL_HEADER_SIZE -
		STANDARD_LIBRARY_DIRECTORY_END_SIZE) / STANDARD_LIBRARY_WORD_16_SIZE
	var maximum_name_storage [MEMBER_NAME_SIZE_MAXIMUM]byte
	for position := range maximum_name_storage {
		maximum_name_storage[position] = 'x'
	}
	var maximum_name_archive_storage [bytes.SLICE_SIZE_MAXIMUM]byte
	maximum_name_archive := archive_data(
		maximum_name_archive_storage[:], maximum_name_storage[:], 0, nil, nil,
	)
	count, status := zip.Decode_Into(maximum_name_archive, "x", nil)
	testify.Equal(t, zip.STATUS_OK, status)
	testify.Zero(t, count)
	var maximum_suffix_storage [bytes.TEXT_SIZE_MAXIMUM]byte
	for position := range maximum_suffix_storage {
		maximum_suffix_storage[position] = 'x'
	}
	maximum_suffix := bytes.Text(string(maximum_suffix_storage[:]))
	zip.Decode_Into(maximum_name_archive, maximum_suffix, nil)
}

type compression_workspace struct {
	Heads    [flate.HASH_COUNT]int32
	Previous [flate.WINDOW_SIZE]int32
}

// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by BSD-style license in Go source tree LICENSE file.

// STANDARD_LIBRARY_DATA_DESCRIPTOR preserves upstream dd.zip bytes.
const STANDARD_LIBRARY_DATA_DESCRIPTOR = ("504b0304140008000800ca68423e0000" +
	"00000000000000000000080000006669" +
	"6c656e616d650bc9c82c5600a2448592d4e21220515192969993aac70500504b" +
	"0708d3d6e3a21800000019000000504b01021400140008000800ca68423ed3d6" +
	"e3a2180000001900000008000000000000000000000000000000000066696c65" +
	"6e616d65504b05060000000001000100360000004e0000000000")

// STANDARD_LIBRARY_SIGNED_DESCRIPTOR preserves upstream signed descriptor bytes.
const STANDARD_LIBRARY_SIGNED_DESCRIPTOR = ("504b0304140008000000000000000000" +
	"0000000000000000000007000000666f" +
	"6f2e747874666f6f0a504b0708a865327e0400000004000000504b0304140008" +
	"00000000000000000000000000000000000000070000006261722e7478746261" +
	"720a504b0708e9b3a2040400000004000000504b010214001400080000000000" +
	"0000a865327e0400000004000000070000000000000000000000000000000000" +
	"666f6f2e747874504b0102140014000800000000000000e9b3a2040400000004" +
	"0000000700000000000000000000000000390000006261722e747874504b0506" +
	"00000000020002006a000000720000000000")

// STANDARD_LIBRARY_HEADER_CHECKSUM preserves upstream non-streamed CRC bytes.
const STANDARD_LIBRARY_HEADER_CHECKSUM = ("504b03040a000000000065876840a86532" +
	"7e040000000400000007001c00666f" +
	"6f2e7478745554090003de55594fde55594f75780b000104f501000004140000" +
	"00666f6f0a504b03040a000000000066876840e9b3a204040000000400000007" +
	"001c006261722e7478745554090003e055594fe055594f75780b000104f50100" +
	"0004140000006261720a504b01021e030a000000000065876840a865327e0400" +
	"000004000000070018000000000001000000a48100000000666f6f2e74787455" +
	"54050003de55594f75780b000104f50100000414000000504b01021e030a0000" +
	"00000066876840e9b3a2040400000004000000070018000000000001000000a4" +
	"81450000006261722e7478745554050003e055594f75780b000104f501000004" +
	"14000000504b050600000000020002009a0000008a0000000000")

// STANDARD_LIBRARY_TRUNCATED_COMMENT preserves upstream issue 66869 bytes.
const STANDARD_LIBRARY_TRUNCATED_COMMENT = ("504b03040000000000000000000079be" +
	"69b90100000001000000040000004649" +
	"4c45504b03040000000000000000000063f3f3ad040000000400000004000000" +
	"66696c6564617461504b010200000000000000000000000063f3f3ad04000000" +
	"0400000004000000000000000000000000000000000066696c65504b05060000" +
	"00000100010032000000260000000000504b0102000000000000000000000000" +
	"79be69b901000000010000000400000000000000000000000000000000004649" +
	"4c45504b0506000000000100010032000000900000000100")

// STANDARD_LIBRARY_ZIP64 preserves upstream bounded unsupported-format fixture.
const STANDARD_LIBRARY_ZIP64 = ("504b030414000000080030740a417ee7ff692400000024000000060000005245" +
	"41444d450bc9c82c5628ce4dccc95148cbcc495500f232f314a23c03cc4c14d2" +
	"f28b72134bf4b800504b01022d032d000000080030740a417ee7ff69ffffffff" +
	"ffffffff060014000000000000000000a48100000000524541444d4501001000" +
	"24000000000000002400000000000000504b06062c000000000000002d002d00" +
	"0000000000000000010000000000000001000000000000004800000000000000" +
	"4800000000000000504b060700000000900000000000000001000000504b0506" +
	"00000000ffffffffffffffffffffffff0000")

// STANDARD_LIBRARY_UTF8 preserves upstream cross-tool Unicode-name fixture.
const STANDARD_LIBRARY_UTF8 = ("504b03040a00000800002e69664b00000000000000000000000006000000e4b8" +
	"96e7958c504b01023f000a00000800002e69664b000000000000000000000000" +
	"060024000000000000008000000000000000e4b896e7958c0a00200000000000" +
	"0100180061caeb874357d3016fbbf0de4357d30161caeb874357d301504b0506" +
	"000000000100010058000000240000000000")

// STANDARD_LIBRARY_SYMLINK preserves upstream Unix-mode content fixture.
const STANDARD_LIBRARY_SYMLINK = ("504b03040a0000000000189f4340d1fa" +
	"9e8e090000000900000007001c007379" +
	"6d6c696e6b555409000320582c4f20582c4f75780b000104e803000004e80300" +
	"002e2e2f746172676574504b01021e030a0000000000189f4340d1fa9e8e0900" +
	"000009000000070018000000000000000000ffa10000000073796d6c696e6b55" +
	"5405000320582c4f75780b000104e803000004e8030000504b05060000000001" +
	"0001004d0000004a0000000000")

// Test_Standard_Library_Reader_Fixtures ports upstream TestReader wire fixtures that exercise
// classic entries. ZIP64 fixture belongs to bounded rejection tests below.
func Test_Standard_Library_Reader_Fixtures(t *testing.T) {
	t.Parallel()
	cases := [...]struct {
		Name    string
		Hex     string
		Suffix  bytes.Text
		Content []byte
		Status  zip.Status
		Size    int
	}{
		{Name: "data descriptor", Hex: STANDARD_LIBRARY_DATA_DESCRIPTOR, Suffix: "filename",
			Content: []byte("This is a test textfile.\n"),
			Status:  zip.STATUS_OK, Size: 154},
		{Name: "signed descriptor first", Hex: STANDARD_LIBRARY_SIGNED_DESCRIPTOR,
			Suffix:  "foo.txt",
			Content: []byte("foo\n"), Status: zip.STATUS_OK, Size: 242},
		{Name: "signed descriptor second", Hex: STANDARD_LIBRARY_SIGNED_DESCRIPTOR,
			Suffix:  "bar.txt",
			Content: []byte("bar\n"), Status: zip.STATUS_OK, Size: 242},
		{Name: "header checksum first", Hex: STANDARD_LIBRARY_HEADER_CHECKSUM,
			Suffix:  "foo.txt",
			Content: []byte("foo\n"), Status: zip.STATUS_OK, Size: 314},
		{Name: "header checksum second", Hex: STANDARD_LIBRARY_HEADER_CHECKSUM,
			Suffix:  "bar.txt",
			Content: []byte("bar\n"), Status: zip.STATUS_OK, Size: 314},
		{Name: "truncated comment", Hex: STANDARD_LIBRARY_TRUNCATED_COMMENT, Suffix: "file",
			Status: zip.STATUS_INPUT_INVALID, Size: 216},
		{Name: "ZIP64", Hex: STANDARD_LIBRARY_ZIP64, Suffix: "README",
			Status: zip.STATUS_INPUT_INVALID, Size: 242},
		{Name: "UTF-8", Hex: STANDARD_LIBRARY_UTF8, Suffix: "世界",
			Content: []byte{}, Status: zip.STATUS_OK, Size: 146},
		{Name: "symlink content", Hex: STANDARD_LIBRARY_SYMLINK, Suffix: "symlink",
			Content: []byte("../target"), Status: zip.STATUS_OK, Size: 173},
	}
	for _, one := range cases {
		one := one
		t.Run(one.Name, func(t *testing.T) {
			t.Parallel()
			var archive_storage [bytes.SLICE_SIZE_MAXIMUM]byte
			archive := decode_standard_library_hexadecimal(
				t, archive_storage[:], one.Hex,
			)
			testify.Count(t, archive, one.Size)
			var destination [bytes.SLICE_SIZE_MAXIMUM]byte
			count, status := zip.Decode_Into(archive, one.Suffix, destination[:])
			testify.Equal(t, one.Status, status)
			if one.Status == zip.STATUS_OK {
				testify.Equal(t, one.Content, destination[:count])
			} else {
				testify.Zero(t, count)
			}
		})
	}
}

// Test_Standard_Library_Trailing_Junk ports upstream test-trailing-junk.zip and
// test-prefix.zip. EOCD bounds archive grammar; bytes after its complete comment stay ignored.
func Test_Standard_Library_Trailing_Junk(t *testing.T) {
	t.Parallel()
	const PREFIX_SIZE = STANDARD_LIBRARY_CENTRAL_HEADER_SIZE -
		STANDARD_LIBRARY_SIGNATURE_SIZE + STANDARD_LIBRARY_NAME_SIZE_MINIMUM
	const ARCHIVE_STORAGE_SIZE = STANDARD_LIBRARY_ARCHIVE_CAPACITY *
		STANDARD_LIBRARY_WORD_GROWTH_FACTOR
	content := []byte("This is a test text file.\n")
	for _, prefix_size := range []int{0, PREFIX_SIZE} {
		var archive_storage [ARCHIVE_STORAGE_SIZE]byte
		segment := archive_data(
			archive_storage[prefix_size:], []byte("test.txt"), 0, content, content,
		)
		archive_size := prefix_size + len(segment)
		archive_size += copy(archive_storage[archive_size:], "some nonsense\n")
		archive := archive_storage[:archive_size]
		var destination [bytes.SLICE_SIZE_MAXIMUM]byte
		count, status := zip.Decode_Into(archive, "test.txt", destination[:])
		testify.Equal(t, zip.STATUS_OK, status, prefix_size)
		testify.Equal(t, content, destination[:count], prefix_size)
	}
}

// Test_Standard_Library_Invalid_Files ports upstream TestInvalidFiles inside 4096-byte input
// bound. Size parameter rejection has no counterpart because slice carries its own size.
func Test_Standard_Library_Invalid_Files(t *testing.T) {
	t.Parallel()
	var zeroes [bytes.SLICE_SIZE_MAXIMUM]byte
	count, status := zip.Decode_Into(zeroes[:], "x", nil)
	testify.Zero(t, count)
	testify.Equal(t, zip.STATUS_INPUT_INVALID, status)
	var signatures [bytes.SLICE_SIZE_MAXIMUM]byte
	for position := 0; position <= len(signatures)-4; position += 4 {
		put_word_32(signatures[position:], 0x06054b50)
	}
	count, status = zip.Decode_Into(signatures[:], "x", nil)
	testify.Zero(t, count)
	testify.Equal(t, zip.STATUS_INPUT_INVALID, status)
	for _, source_size := range []int{0, 1, 2, 21} {
		count, status = zip.Decode_Into(zeroes[:source_size], "x", nil)
		testify.Zero(t, count, source_size)
		testify.Equal(t, zip.STATUS_INPUT_INVALID, status, source_size)
	}
}

// Test_Standard_Library_Descriptor_Truncation ports upstream issue 11146 across every cut.
func Test_Standard_Library_Descriptor_Truncation(t *testing.T) {
	t.Parallel()
	var archive_storage [bytes.SLICE_SIZE_MAXIMUM]byte
	archive := decode_standard_library_hexadecimal(
		t, archive_storage[:], STANDARD_LIBRARY_SIGNED_DESCRIPTOR,
	)
	var destination [bytes.SLICE_SIZE_MAXIMUM]byte
	for source_index := 0; source_index < len(archive); source_index++ {
		count, status := zip.Decode_Into(
			archive[:source_index], "foo.txt", destination[:],
		)
		testify.Zero(t, count, source_index)
		testify.Equal(t, zip.STATUS_INPUT_INVALID, status, source_index)
	}
}

// Test_Standard_Library_Under_Size ports upstream TestUnderSize without mutable reader metadata.
func Test_Standard_Library_Under_Size(t *testing.T) {
	t.Parallel()
	const ARCHIVE_STORAGE_SIZE = STANDARD_LIBRARY_ARCHIVE_CAPACITY
	const LOCAL_UNCOMPRESSED_SIZE_POSITION = STANDARD_LIBRARY_LOCAL_UNCOMPRESSED_SIZE_POSITION
	const CENTRAL_UNCOMPRESSED_SIZE_POSITION = ZIP_CENTRAL_UNCOMPRESSED_SIZE_POSITION
	content := []byte("declared size mismatch")
	name := []byte("data.txt")
	var archive_storage [ARCHIVE_STORAGE_SIZE]byte
	archive := archive_data(
		archive_storage[:], name, 0, content, content,
	)
	central_position := 30 + len(name) + len(content)
	put_word_32(archive[LOCAL_UNCOMPRESSED_SIZE_POSITION:], 1)
	put_word_32(archive[central_position+CENTRAL_UNCOMPRESSED_SIZE_POSITION:], 1)
	var destination [bytes.SLICE_SIZE_MAXIMUM]byte
	count, status := zip.Decode_Into(archive, "data.txt", destination[:])
	testify.Zero(t, count)
	testify.Equal(t, zip.STATUS_INPUT_INVALID, status)
}

// Test_Standard_Library_Compressed_Empty ports upstream TestCompressedDirectory payload case.
// Lookup grammar omits trailing slash because Decode_Into selects basename suffixes, not FS nodes.
func Test_Standard_Library_Compressed_Empty(t *testing.T) {
	t.Parallel()
	const ARCHIVE_STORAGE_SIZE = STANDARD_LIBRARY_ARCHIVE_CAPACITY
	compressed := deflated(t, nil)
	var archive_storage [ARCHIVE_STORAGE_SIZE]byte
	archive := archive_data(
		archive_storage[:], []byte("META-INF"), 8, compressed, nil,
	)
	count, status := zip.Decode_Into(archive, "META-INF", nil)
	testify.Zero(t, count)
	testify.Equal(t, zip.STATUS_OK, status)
}

// Test_Standard_Library_Insecure_Paths ports upstream insecure-path and CVE-2021-27919 cases.
// Caller receives bytes only; no filesystem path reaches an OS boundary.
func Test_Standard_Library_Insecure_Paths(t *testing.T) {
	t.Parallel()
	const ARCHIVE_STORAGE_SIZE = STANDARD_LIBRARY_ARCHIVE_CAPACITY
	content := []byte("safe caller-owned bytes")
	for _, one := range []struct {
		Name   string
		Suffix bytes.Text
	}{
		{Name: "../test.txt", Suffix: "test.txt"},
		{Name: "/test.txt", Suffix: "test.txt"},
		{Name: "a/b/../../../test.txt", Suffix: "test.txt"},
		{Name: `a\b`, Suffix: `a\b`},
	} {
		one := one
		t.Run(one.Name, func(t *testing.T) {
			t.Parallel()
			var archive_storage [ARCHIVE_STORAGE_SIZE]byte
			archive := archive_data(
				archive_storage[:], []byte(one.Name), 0, content, content,
			)
			var destination [bytes.SLICE_SIZE_MAXIMUM]byte
			count, status := zip.Decode_Into(archive, one.Suffix, destination[:])
			testify.Equal(t, zip.STATUS_OK, status)
			testify.Equal(t, content, destination[:count])
		})
	}
	var archive_storage [ARCHIVE_STORAGE_SIZE]byte
	archive := archive_data(archive_storage[:], []byte("/"), 0, nil, nil)
	count, status := zip.Decode_Into(archive, "/", nil)
	testify.Zero(t, count)
	testify.Equal(t, zip.STATUS_INPUT_INVALID, status)
}

// Test_Standard_Library_Overflow_Rejection ports CVE-2021-33196, CVE-2021-39293, and
// TestBaseOffsetPlusOverflow through classic fixed-width sentinels.
func Test_Standard_Library_Overflow_Rejection(t *testing.T) {
	t.Parallel()
	const ARCHIVE_STORAGE_SIZE = STANDARD_LIBRARY_ARCHIVE_CAPACITY
	const DIRECTORY_END_SIZE = STANDARD_LIBRARY_DIRECTORY_END_SIZE
	const DIRECTORY_DISK_ENTRY_COUNT_POSITION = ZIP_DIRECTORY_DISK_ENTRY_COUNT_POSITION
	const DIRECTORY_ENTRY_COUNT_POSITION = STANDARD_LIBRARY_DIRECTORY_ENTRY_COUNT_POSITION
	const DIRECTORY_CENTRAL_SIZE_POSITION = STANDARD_LIBRARY_DIRECTORY_CENTRAL_SIZE_POSITION
	const DIRECTORY_CENTRAL_OFFSET_POSITION = STANDARD_LIBRARY_DIRECTORY_CENTRAL_OFFSET_POSITION
	for _, one := range []struct {
		Name     string
		Position int
		Width    int
	}{
		{Name: "entry count", Position: DIRECTORY_ENTRY_COUNT_POSITION, Width: 2},
		{Name: "central size", Position: DIRECTORY_CENTRAL_SIZE_POSITION, Width: 4},
		{Name: "central offset", Position: DIRECTORY_CENTRAL_OFFSET_POSITION, Width: 4},
	} {
		one := one
		t.Run(one.Name, func(t *testing.T) {
			t.Parallel()
			var archive_storage [ARCHIVE_STORAGE_SIZE]byte
			archive := archive_data(
				archive_storage[:], []byte("x"), 0, nil, nil,
			)
			end := archive[len(archive)-DIRECTORY_END_SIZE:]
			if one.Width == 2 {
				put_word_16(end[DIRECTORY_DISK_ENTRY_COUNT_POSITION:], 1<<16-1)
				put_word_16(end[one.Position:], 1<<16-1)
			} else {
				put_word_32(end[one.Position:], 1<<32-1)
			}
			var count bytes.Boundary
			var status zip.Status
			testify.Not_Panics(t, func() {
				count, status = zip.Decode_Into(archive, "x", nil)
			})
			testify.Zero(t, count)
			testify.Equal(t, zip.STATUS_INPUT_INVALID, status)
		})
	}
}

// Test_Standard_Library_Issue_10956 ports upstream malformed ZIP64 locator regression bytes.
func Test_Standard_Library_Issue_10956(t *testing.T) {
	t.Parallel()
	source := []byte("PK\x06\x06PK\x06\a0000\x00\x00\x00\x00\x00\x00\x00\x00" +
		"0000PK\x05\x0600000000000000\v\x00000\x00\x00\x00\x00\x00\x00\x000")
	count, status := zip.Decode_Into(source, "x", nil)
	testify.Zero(t, count)
	testify.Equal(t, zip.STATUS_INPUT_INVALID, status)
}

func decode_standard_library_hexadecimal(
	t *testing.T, destination []byte, source string,
) (decoded []byte) {
	t.Helper()
	testify.Zero(t, len(source)%2)
	decoded_size := len(source) / 2
	testify.Less_Or_Equal(t, &testify.Less_Or_Equal_Input[int]{
		First: decoded_size, Second: len(destination),
	})
	for source_position := 0; source_position < len(source); source_position += 2 {
		high, high_valid := standard_library_hexadecimal_nibble(source[source_position])
		low, low_valid := standard_library_hexadecimal_nibble(source[source_position+1])
		testify.True(t, high_valid)
		testify.True(t, low_valid)
		destination[source_position/2] = high<<4 | low
	}
	return destination[:decoded_size]
}

func standard_library_hexadecimal_nibble(source byte) (nibble byte, valid bool) {
	if source >= '0' {
		if source <= '9' {
			return source - '0', true
		}
	}
	if source >= 'a' {
		if source <= 'f' {
			return source - 'a' + 10, true
		}
	}
	return 0, false
}

func standard_library_reader_header_field_domains(marker int) {
	for field_index := 0; field_index < 11; field_index++ {
		archive := standard_library_reader_header_archive(1, 0, 0)
		standard_library_reader_header_field_domain(archive, field_index, marker)
		reader := standard_library_reader_domain_archive(archive)
		var header zip.Header
		zip.Reader_Header(&reader, 0, &header)
		zip.Header_Mode(&header)
		zip.Header_Modification_Time(&header)
		reader = standard_library_reader_domain_archive(archive)
		zip.Reader_Header(&reader, 0, &header)
		reader = standard_library_reader_domain_archive(archive)
		zip.Reader_Raw(&reader, 0, &header)
		standard_library_writer_validated_header_domain(header)
		standard_library_file_system_collect_archive(archive)
	}
}

func standard_library_reader_header_field_domain(
	archive []byte, field_index int, marker int,
) {
	word_16 := standard_library_domain_word_16(marker)
	word_32 := standard_library_domain_word_32(marker)
	switch field_index {
	case 0:
		standard_library_put_word_16(archive[4:], word_16)
	case 1:
		standard_library_put_word_16(archive[6:], word_16)
	case 2:
		standard_library_put_word_16(archive[8:], word_16)
	case 3:
		standard_library_put_word_16(archive[10:], word_16)
	case 4:
		standard_library_put_word_16(archive[12:], word_16)
	case 5:
		standard_library_put_word_16(archive[14:], word_16)
	case 6:
		standard_library_put_word_32(archive[16:], word_32)
	case 7:
		standard_library_put_word_32(archive[20:], word_32)
	case 8:
		standard_library_put_word_32(archive[24:], word_32)
	case 9:
		standard_library_put_word_16(archive[36:], word_16)
	case 10:
		standard_library_put_word_32(archive[38:], word_32)
	}
}

func standard_library_reader_header_tail_domains() {
	for _, one := range []struct {
		Name_Count    int
		Extra_Count   int
		Comment_Count int
	}{
		{},
		{Name_Count: 1},
		{Name_Count: 2},
		{Extra_Count: 1},
		{Extra_Count: 2},
		{Comment_Count: 1},
		{Comment_Count: 2},
		{Name_Count: STANDARD_LIBRARY_WRITER_RECORD_NAME_MAXIMUM},
		{Extra_Count: STANDARD_LIBRARY_WRITER_RECORD_TAIL_MAXIMUM},
		{Comment_Count: STANDARD_LIBRARY_WRITER_RECORD_TAIL_MAXIMUM},
	} {
		archive := standard_library_reader_header_archive(
			one.Name_Count, one.Extra_Count, one.Comment_Count,
		)
		reader := standard_library_reader_domain_archive(archive)
		var header zip.Header
		zip.Reader_Header(&reader, 0, &header)
		zip.Header_Mode(&header)
		zip.Header_Modification_Time(&header)
		reader = standard_library_reader_domain_archive(archive)
		zip.Reader_Header(&reader, 0, &header)
		reader = standard_library_reader_domain_archive(archive)
		zip.Reader_Raw(&reader, 0, &header)
		standard_library_writer_validated_header_domain(header)
		standard_library_file_system_collect_archive(archive)
	}
	standard_library_file_system_collect_archive(
		standard_library_reader_headers_archive(0),
	)
	standard_library_file_system_collect_archive(
		standard_library_reader_headers_archive(2),
	)
	standard_library_file_system_collect_archive(
		standard_library_reader_headers_archive(86),
	)
	standard_library_reader_domain_archive(
		standard_library_reader_empty_headers_archive(88),
	)
}

func standard_library_reader_headers_archive(entry_count int) (archive []byte) {
	central_count := 47 * entry_count
	archive = make([]byte, central_count+22)
	for index := 0; index < entry_count; index++ {
		position := index * 47
		copy(archive[position:], []byte("PK\x01\x02"))
		standard_library_put_word_16(archive[position+28:], 1)
		archive[position+46] = 'x'
	}
	end := archive[central_count:]
	copy(end, []byte("PK\x05\x06"))
	standard_library_put_word_16(end[8:], uint16(entry_count))
	standard_library_put_word_16(end[10:], uint16(entry_count))
	standard_library_put_word_32(end[12:], uint32(central_count))
	return archive
}

func standard_library_reader_empty_headers_archive(
	entry_count int,
) (archive []byte) {
	central_count := 46 * entry_count
	archive = make([]byte, central_count+22)
	for index := 0; index < entry_count; index++ {
		copy(archive[index*46:], []byte("PK\x01\x02"))
	}
	end := archive[central_count:]
	copy(end, []byte("PK\x05\x06"))
	standard_library_put_word_16(end[8:], uint16(entry_count))
	standard_library_put_word_16(end[10:], uint16(entry_count))
	standard_library_put_word_32(end[12:], uint32(central_count))
	return archive
}

func standard_library_reader_header_archive(
	name_count int, extra_count int, comment_count int,
) (archive []byte) {
	central_count := 46 + name_count + extra_count + comment_count
	archive = make([]byte, central_count+22)
	copy(archive, []byte("PK\x01\x02"))
	standard_library_put_word_16(archive[28:], uint16(name_count))
	standard_library_put_word_16(archive[30:], uint16(extra_count))
	standard_library_put_word_16(archive[32:], uint16(comment_count))
	for position_index := 0; position_index < name_count; position_index++ {
		archive[46+position_index] = 'x'
	}
	end := archive[central_count:]
	copy(end, []byte("PK\x05\x06"))
	standard_library_put_word_16(end[8:], 1)
	standard_library_put_word_16(end[10:], 1)
	standard_library_put_word_32(end[12:], uint32(central_count))
	return archive
}

func standard_library_reader_domain_archive(archive []byte) (reader zip.Reader) {
	memory := nbio.Stream_Memory{Memory: archive}
	storage := make([]byte, bytes.SLICE_SIZE_MAXIMUM)
	var completion time.Completion
	zip.Reader_Init(
		&reader, nbio.Memory_To_Stream(&memory),
		zip.Reader_Storage{Archive: storage}, &completion,
		func(_ *time.Completion) {},
	)
	return reader
}

func standard_library_file_system_collect_archive(archive []byte) {
	reader := standard_library_reader_domain_archive(archive)
	var file_system zip.File_System
	zip.File_System_Init(&file_system, &reader)
	var nodes [zip.ENTRY_COUNT_MAXIMUM + 1]zip.File_Info
	zip.File_System_Walk(
		&file_system, []byte("."), nodes[:],
		func(_ zip.File_Info) (keep_going binary.Boolean) { return true },
	)
}

func standard_library_writer_validated_header_domain(header zip.Header) {
	value := zip.Header_Unvalidated{
		Name:    zip.Header_Name_Unvalidated(header.Name),
		Comment: zip.Header_Comment_Unvalidated(header.Comment),
		Extra:   zip.Header_Extra_Unvalidated(header.Extra),
		Method:  header.Method, Flags: header.Flags, Checksum: header.Checksum,
		Compressed_Size: zip.Header_Compressed_Size_Unvalidated(header.Compressed_Size),
		Uncompressed_Size: zip.Header_Uncompressed_Size_Unvalidated(
			header.Uncompressed_Size,
		),
		External_Attributes: header.External_Attributes,
		Creator_Version:     header.Creator_Version,
		Extractor_Version:   header.Extractor_Version,
		Modified_Time:       header.Modified_Time, Modified_Date: header.Modified_Date,
		Internal_Attributes: header.Internal_Attributes,
		Modified:            header.Modified, Non_UTF8: header.Non_UTF8,
	}
	writer := standard_library_writer_valid_domain_value()
	zip.Writer_Create(&writer, &value)
	writer = standard_library_writer_valid_domain_value()
	zip.Writer_Create_Raw(&writer, &value)
}

func standard_library_put_word_16(destination []byte, value uint16) {
	destination[0] = byte(value)
	destination[1] = byte(value >> 8)
}

func standard_library_put_word_32(destination []byte, value uint32) {
	destination[0] = byte(value)
	destination[1] = byte(value >> 8)
	destination[2] = byte(value >> 16)
	destination[3] = byte(value >> 24)
}

func standard_library_writer_field_domains(marker int) {
	base := standard_library_writer_valid_domain_value()
	for field_index := 0; field_index < 38; field_index++ {
		standard_library_writer_field_operations(base, field_index, marker)
	}
}

func standard_library_writer_field_operations(
	base zip.Writer, field_index int, marker int,
) {
	header := zip.Header_Unvalidated{Name: []byte("x"), Method: zip.METHOD_STORE}
	reader := standard_library_reader_domain_value(0)
	file_system := standard_library_file_system_domain_value(0)
	var completion time.Completion
	callback := func(_ *time.Completion) {}
	writer := base
	standard_library_writer_field_domain(&writer, field_index, marker)
	zip.Writer_Init(&writer, nbio.Stream{}, base.Storage)
	writer = base
	standard_library_writer_field_domain(&writer, field_index, marker)
	zip.Writer_Create(&writer, &header)
	writer = base
	standard_library_writer_field_domain(&writer, field_index, marker)
	zip.Writer_Create_Raw(&writer, &header)
	writer = base
	standard_library_writer_field_domain(&writer, field_index, marker)
	zip.Writer_Copy(&writer, &reader, 0)
	writer = base
	standard_library_writer_field_domain(&writer, field_index, marker)
	zip.Writer_Add_File_System(&writer, &file_system, nil)
	writer = base
	standard_library_writer_field_domain(&writer, field_index, marker)
	zip.Writer_Set_Comment(&writer, nil)
	writer = base
	standard_library_writer_field_domain(&writer, field_index, marker)
	zip.Writer_Set_Offset(&writer, 0)
	writer = base
	standard_library_writer_field_domain(&writer, field_index, marker)
	zip.Writer_Write(&writer, nil)
	writer = base
	standard_library_writer_field_domain(&writer, field_index, marker)
	zip.Writer_Close(&writer, &completion, callback)
	writer = base
	standard_library_writer_field_domain(&writer, field_index, marker)
	zip.Writer_Flush(&writer, &completion, callback)
}

func standard_library_writer_field_domain(
	writer *zip.Writer, field_index int, marker int,
) {
	size := standard_library_domain_size(marker)
	if field_index >= 8 {
		if field_index < 24 {
			standard_library_writer_header_field_domain(writer, field_index-8, marker)
			return
		}
	}
	switch field_index {
	case 0:
		writer.Storage.Archive = make([]byte, size)
	case 1:
		writer.Storage.Central = make([]byte, size)
	case 2:
		writer.Storage.Content = make([]byte, size)
	case 3:
		writer.Storage.Compressed = make([]byte, size)
	case 4:
		writer.Storage.Comment = make([]byte, size)
	case 5:
		writer.Storage.Name = make([]byte, size)
	case 6:
		writer.Storage.Heads = make([]int32, standard_library_workspace_count(marker))
	case 7:
		writer.Storage.Previous = make([]int32, standard_library_history_count(marker))
	case 24:
		writer.Status = standard_library_domain_status(marker)
	case 25:
		writer.Count = zip.Writer_Count(size)
	case 26:
		writer.Archive_Position = zip.Writer_Archive_Position(size)
	case 27:
		writer.Central_Position = zip.Writer_Central_Position(size)
	case 28:
		writer.Content_Position = zip.Writer_Content_Position(size)
	case 29:
		writer.Current_Central_Position = zip.Writer_Current_Central_Position(size)
	case 30:
		writer.Offset = zip.Writer_Offset(size)
	case 31:
		writer.Entry_Count = standard_library_domain_entry_count(marker)
	case 32:
		writer.Comment_Count = zip.Writer_Comment_Count(
			standard_library_domain_header_tail_size(marker),
		)
	case 33:
		writer.Active = zip.Writer_Active(marker != 0)
	case 34:
		writer.Raw = zip.Writer_Raw(marker != 0)
	case 35:
		writer.Directory = zip.Writer_Directory(marker != 0)
	case 36:
		writer.Closed = zip.Writer_Closed(marker != 0)
	case 37:
		writer.In_Flight = zip.Writer_In_Flight(marker != 0)
	}
}

func standard_library_writer_header_field_domain(
	writer *zip.Writer, field_index int, marker int,
) {
	tail_size := standard_library_domain_header_tail_size(marker)
	switch field_index {
	case 0:
		writer.Header.Name = make([]byte, tail_size)
	case 1:
		writer.Header.Comment = make([]byte, tail_size)
	case 2:
		writer.Header.Extra = make([]byte, tail_size)
	case 3:
		writer.Header.Method = zip.Header_Method(standard_library_domain_word_16(marker))
	case 4:
		writer.Header.Flags = zip.Header_Flags(standard_library_domain_word_16(marker))
	case 5:
		writer.Header.Checksum = zip.Header_Checksum(
			standard_library_domain_word_32(marker),
		)
	case 6:
		writer.Header.Compressed_Size = zip.Header_Compressed_Size(
			standard_library_domain_header_size(marker),
		)
	case 7:
		writer.Header.Uncompressed_Size = zip.Header_Uncompressed_Size(
			standard_library_domain_header_size(marker),
		)
	case 8:
		writer.Header.External_Attributes = zip.Header_External_Attributes(
			standard_library_domain_word_32(marker),
		)
	case 9:
		writer.Header.Creator_Version = zip.Header_Creator_Version(
			standard_library_domain_word_16(marker),
		)
	case 10:
		writer.Header.Extractor_Version = zip.Header_Extractor_Version(
			standard_library_domain_word_16(marker),
		)
	case 11:
		writer.Header.Modified_Time = zip.Header_Modified_Time(
			standard_library_domain_word_16(marker),
		)
	case 12:
		writer.Header.Modified_Date = zip.Header_Modified_Date(
			standard_library_domain_word_16(marker),
		)
	case 13:
		writer.Header.Internal_Attributes = zip.Header_Internal_Attributes(
			standard_library_domain_word_16(marker),
		)
	case 14:
		writer.Header.Modified = standard_library_domain_timestamp(marker)
	case 15:
		writer.Header.Non_UTF8 = zip.Header_Non_UTF8(marker != 0)
	}
}

func standard_library_header_field_domains(marker int) {
	for field_index := 0; field_index < 17; field_index++ {
		header := standard_library_header_field_domain_value(field_index, marker)
		validated, _ := zip.Header_Validate(&header)
		setter_header := header
		zip.Header_Set_Modification_Time(
			&setter_header, standard_library_domain_timestamp(marker),
		)
		zip.Header_Set_Mode(&setter_header, standard_library_domain_file_mode(marker))
		zip.Header_Mode(&validated)
		zip.Header_Modification_Time(&validated)
		archive := standard_library_reader_header_archive(1, 0, 0)
		reader := standard_library_reader_domain_archive(archive)
		zip.Reader_Raw(&reader, 0, &validated)
		reader = standard_library_reader_domain_archive(archive)
		zip.Reader_Header(&reader, 0, &validated)
		writer := standard_library_writer_valid_domain_value()
		zip.Writer_Create(&writer, &header)
		writer = standard_library_writer_valid_domain_value()
		zip.Writer_Create_Raw(&writer, &header)
		reader = standard_library_reader_domain_value(0)
		zip.Reader_Header(&reader, 0, &validated)
		reader = standard_library_reader_domain_value(0)
		zip.Reader_Raw(&reader, 0, &validated)
	}
}

func standard_library_header_field_domain_value(
	field_index int, marker int,
) (header zip.Header_Unvalidated) {
	header = zip.Header_Unvalidated{Name: []byte("x"), Method: zip.METHOD_STORE}
	size := standard_library_domain_size(marker)
	switch field_index {
	case 0:
		header.Name = make([]byte, size)
	case 1:
		header.Comment = make([]byte, size)
	case 2:
		header.Extra = make([]byte, size)
	case 3:
		header.Method = zip.Header_Method(standard_library_domain_word_16(marker))
	case 4:
		header.Flags = zip.Header_Flags(standard_library_domain_word_16(marker))
	case 5:
		header.Checksum = zip.Header_Checksum(standard_library_domain_word_32(marker))
	case 6:
		header.Compressed_Size = zip.Header_Compressed_Size_Unvalidated(
			standard_library_domain_header_size_unvalidated(marker),
		)
	case 7:
		header.Uncompressed_Size = zip.Header_Uncompressed_Size_Unvalidated(
			standard_library_domain_header_size_unvalidated(marker),
		)
	case 8:
		header.External_Attributes = zip.Header_External_Attributes(
			standard_library_domain_word_32(marker),
		)
	case 9:
		header.Creator_Version = zip.Header_Creator_Version(
			standard_library_domain_word_16(marker),
		)
	case 10:
		header.Extractor_Version = zip.Header_Extractor_Version(
			standard_library_domain_word_16(marker),
		)
	case 11:
		header.Modified_Time = zip.Header_Modified_Time(
			standard_library_domain_word_16(marker),
		)
	case 12:
		header.Modified_Date = zip.Header_Modified_Date(
			standard_library_domain_word_16(marker),
		)
	case 13:
		header.Internal_Attributes = zip.Header_Internal_Attributes(
			standard_library_domain_word_16(marker),
		)
	case 14:
		header.Modified = standard_library_domain_timestamp(marker)
	case 15:
		header.Non_UTF8 = zip.Header_Non_UTF8(marker != 0)
	case 16:
		header.Name = []byte("x")
	}
	return header
}

func standard_library_writer_record_domains() {
	for _, one := range []zip.Header_Unvalidated{
		{
			Name:   make([]byte, STANDARD_LIBRARY_WRITER_RECORD_NAME_MAXIMUM),
			Method: zip.METHOD_STORE,
		},
		{
			Name:    []byte("x"),
			Comment: make([]byte, STANDARD_LIBRARY_WRITER_RECORD_TAIL_MAXIMUM),
			Method:  zip.METHOD_STORE,
		},
		{
			Name:   []byte("x"),
			Extra:  make([]byte, STANDARD_LIBRARY_WRITER_RECORD_TAIL_MAXIMUM),
			Method: zip.METHOD_STORE,
		},
		{
			Name:     []byte("x"),
			Extra:    make([]byte, STANDARD_LIBRARY_WRITER_RECORD_TAIL_MAXIMUM),
			Method:   zip.METHOD_STORE,
			Modified: zip.Timestamp{Seconds: 315_532_800, Set: true},
		},
	} {
		writer := standard_library_writer_valid_domain_value()
		zip.Writer_Create(&writer, &one)
	}
	writer := standard_library_writer_valid_domain_value()
	writer.Storage.Archive = make([]byte, STANDARD_LIBRARY_WRITER_LOCAL_STORAGE_MINIMUM)
	writer.Storage.Central = make([]byte, STANDARD_LIBRARY_WRITER_CENTRAL_STORAGE_MINIMUM)
	header := zip.Header_Unvalidated{
		Name: []byte("x"), Method: zip.METHOD_STORE, Flags: zip.FLAG_UTF8,
	}
	zip.Writer_Create(&writer, &header)
	writer = standard_library_writer_valid_domain_value()
	writer.Archive_Position = STANDARD_LIBRARY_WRITER_LOCAL_START_MAXIMUM
	zip.Writer_Create(&writer, &header)
}

func standard_library_writer_central_size_domains() {
	for _, one := range []struct {
		Checksum          uint32
		Uncompressed_Size uint64
	}{
		{Checksum: 1, Uncompressed_Size: 2},
		{Checksum: 2, Uncompressed_Size: 1<<32 - 1},
		{Checksum: 1<<32 - 1},
	} {
		writer := standard_library_writer_valid_domain_value()
		header := zip.Header_Unvalidated{
			Name: []byte("x"), Method: zip.METHOD_STORE,
			Checksum: zip.Header_Checksum(one.Checksum),
			Uncompressed_Size: zip.Header_Uncompressed_Size_Unvalidated(
				one.Uncompressed_Size,
			),
		}
		zip.Writer_Create_Raw(&writer, &header)
		zip.Writer_Create(&writer, &zip.Header_Unvalidated{
			Name: []byte("y"), Method: zip.METHOD_STORE,
		})
	}
	writer := standard_library_writer_valid_domain_value()
	header := zip.Header_Unvalidated{
		Name: []byte("x"), Method: zip.METHOD_STORE,
		Compressed_Size: STANDARD_LIBRARY_WRITER_LOCAL_START_MAXIMUM,
	}
	zip.Writer_Create_Raw(&writer, &header)
	zip.Writer_Write(&writer, make([]byte, STANDARD_LIBRARY_WRITER_LOCAL_START_MAXIMUM))
	zip.Writer_Create(&writer, &zip.Header_Unvalidated{
		Name: []byte("y"), Method: zip.METHOD_STORE,
	})
}

func standard_library_writer_valid_domain_value() (writer zip.Writer) {
	size := bytes.SLICE_SIZE_MAXIMUM
	return zip.Writer{
		Storage: zip.Writer_Storage{
			Archive: make([]byte, size), Central: make([]byte, size),
			Content: make([]byte, size), Compressed: make([]byte, size),
			Comment: make([]byte, size), Name: make([]byte, size),
			Heads:    make([]int32, flate.HASH_COUNT),
			Previous: make([]int32, flate.WINDOW_SIZE),
		},
		Status: zip.STATUS_OK,
	}
}

func standard_library_workspace_count(marker int) (count int) {
	if marker == INVARIANT_DOMAIN_MAXIMUM {
		return flate.HASH_COUNT + 1
	}
	return marker
}

func standard_library_history_count(marker int) (count int) {
	if marker == INVARIANT_DOMAIN_MAXIMUM {
		return flate.WINDOW_SIZE + 1
	}
	return marker
}

func standard_library_writer_domain(marker int) {
	base := standard_library_writer_domain_value(marker)
	header := standard_library_header_domain_value(marker, marker)
	reader := standard_library_reader_domain_value(marker)
	file_system := standard_library_file_system_domain_value(marker)
	buffer := make([]byte, standard_library_domain_size(marker))
	var completion time.Completion
	callback := func(_ *time.Completion) {}
	writer := base
	zip.Writer_Init(&writer, nbio.Stream{}, writer.Storage)
	writer = base
	zip.Writer_Create(&writer, &header)
	writer = base
	zip.Writer_Create_Raw(&writer, &header)
	writer = base
	zip.Writer_Copy(&writer, &reader, standard_library_domain_entry_count(marker))
	writer = base
	zip.Writer_Add_File_System(&writer, &file_system, nil)
	writer = base
	zip.Writer_Set_Comment(&writer, buffer)
	writer = base
	zip.Writer_Set_Offset(&writer, bytes.Boundary(standard_library_domain_size(marker)))
	writer = base
	zip.Writer_Write(&writer, buffer)
	writer = base
	zip.Writer_Close(&writer, &completion, callback)
	writer = base
	zip.Writer_Flush(&writer, &completion, callback)
}

func standard_library_reader_domain(marker int) {
	base := standard_library_reader_domain_value(marker)
	var completion time.Completion
	reader := base
	zip.Reader_Init(
		&reader, nbio.Stream{}, reader.Storage, &completion,
		func(_ *time.Completion) {},
	)
	reader = base
	zip.Reader_Decode(
		&reader, standard_library_domain_entry_count(marker), nil,
	)
	reader = base
	var header zip.Header
	zip.Reader_Header(
		&reader, standard_library_domain_entry_count(marker), &header,
	)
	reader = base
	zip.Reader_Raw(
		&reader, standard_library_domain_entry_count(marker), &header,
	)
}

func standard_library_file_system_domain(marker int) {
	base := standard_library_file_system_domain_value(marker)
	reader := standard_library_reader_domain_value(marker)
	path := make([]byte, standard_library_domain_size(marker))
	file_system := base
	zip.File_System_Init(&file_system, &reader)
	file_system = base
	info := standard_library_file_info_domain_value(marker)
	zip.File_System_Status(&file_system, path, &info)
	file_system = base
	zip.File_System_Open(&file_system, path, nil)
	file_system = base
	zip.File_System_Read_Directory(&file_system, path, nil)
	file_system = base
	zip.File_System_Walk(
		&file_system, path, nil,
		func(_ zip.File_Info) (keep_going binary.Boolean) { return false },
	)
}

func standard_library_file_info_domain_value(marker int) (info zip.File_Info) {
	return zip.File_Info{
		Name: zip.File_Info_Name(make(
			[]byte, standard_library_domain_header_tail_size(marker),
		)),
		Size:         zip.File_Info_Size(standard_library_domain_header_size(marker)),
		Mode:         standard_library_domain_file_mode(marker),
		Modified:     standard_library_domain_second_timestamp(marker),
		Header_Index: standard_library_domain_file_info_header_index(marker),
		Is_Directory: zip.File_Info_Is_Directory(marker != 0),
		Explicit:     zip.File_Info_Explicit(marker != 0),
	}
}

func standard_library_header_domain(marker int) {
	header := standard_library_header_domain_value(marker, marker)
	validated, _ := zip.Header_Validate(&header)
	zip.Header_Set_Mode(&header, standard_library_domain_file_mode(marker))
	zip.Header_Set_Modification_Time(
		&header, standard_library_domain_timestamp(marker),
	)
	zip.Header_Mode(&validated)
	zip.Header_Modification_Time(&validated)
}

func standard_library_rejected_header_domain() {
	marker := INVARIANT_DOMAIN_MAXIMUM
	header := standard_library_header_domain_value(marker, marker+1)
	zip.Header_Validate(&header)
	zip.Header_Set_Modification_Time(
		&header, standard_library_domain_timestamp(marker),
	)
	zip.Header_Set_Mode(&header, standard_library_domain_file_mode(marker))
	writer := standard_library_writer_valid_domain_value()
	zip.Writer_Create(&writer, &header)
	writer = standard_library_writer_valid_domain_value()
	zip.Writer_Create_Raw(&writer, &header)
}

func standard_library_negative_one_timestamp_domains() {
	modified := zip.Timestamp{Zone_Offset_Seconds: -1}
	header := zip.Header_Unvalidated{
		Name: []byte("x"), Method: zip.METHOD_STORE, Modified: modified,
	}
	zip.Header_Set_Modification_Time(&header, modified)
	zip.Header_Set_Mode(&header, 0)
	validated, _ := zip.Header_Validate(&header)
	zip.Header_Mode(&validated)
	zip.Header_Modification_Time(&validated)
	extended := zip.Header{
		Modified: zip.Timestamp{Zone_Offset_Seconds: -1, Set: true},
	}
	zip.Header_Modification_Time(&extended)
	archive := standard_library_reader_header_archive(1, 0, 0)
	reader := standard_library_reader_domain_archive(archive)
	zip.Reader_Header(&reader, 0, &validated)
	validated.Modified.Zone_Offset_Seconds = -1
	zip.Reader_Raw(&reader, 0, &validated)
	standard_library_writer_validated_header_domain(validated)
	file_system := standard_library_file_system_domain_value(0)
	info := zip.File_Info{
		Modified: zip.Timestamp_Second_Precision{Zone_Offset_Seconds: -1},
	}
	zip.File_System_Status(&file_system, nil, &info)
}

func standard_library_negative_one_writer_domains() {
	base := standard_library_writer_valid_domain_value()
	base.Header.Modified.Zone_Offset_Seconds = -1
	header := zip.Header_Unvalidated{
		Name: []byte("x"), Method: zip.METHOD_STORE,
		Modified: zip.Timestamp{Zone_Offset_Seconds: -1},
	}
	reader := standard_library_reader_domain_value(0)
	file_system := standard_library_file_system_domain_value(0)
	var completion time.Completion
	callback := func(_ *time.Completion) {}
	writer := base
	zip.Writer_Init(&writer, nbio.Stream{}, writer.Storage)
	writer = base
	zip.Writer_Create(&writer, &header)
	writer = base
	zip.Writer_Create_Raw(&writer, &header)
	writer = base
	zip.Writer_Copy(&writer, &reader, 0)
	writer = base
	zip.Writer_Add_File_System(&writer, &file_system, nil)
	writer = base
	zip.Writer_Set_Comment(&writer, nil)
	writer = base
	zip.Writer_Set_Offset(&writer, 0)
	writer = base
	zip.Writer_Write(&writer, nil)
	writer = base
	zip.Writer_Close(&writer, &completion, callback)
	writer = base
	zip.Writer_Flush(&writer, &completion, callback)
}

func standard_library_writer_timestamp_domains() {
	for _, marker := range []int{1, 2} {
		header := zip.Header_Unvalidated{
			Name: []byte("x"), Method: zip.METHOD_STORE,
			Modified: standard_library_domain_timestamp(marker),
		}
		header.Modified.Set = false
		writer := standard_library_writer_valid_domain_value()
		zip.Writer_Create(&writer, &header)
		writer = standard_library_writer_valid_domain_value()
		zip.Writer_Create_Raw(&writer, &header)
	}
}

func standard_library_timestamp_domains() {
	for _, one := range []struct {
		Date  uint16
		Clock uint16
	}{
		{Date: 0x21},
		{Date: 0x21, Clock: 1},
		{Date: 0x42},
		{Date: 0xfc47, Clock: 0x3387},
		{Date: 0xff9f, Clock: 0xbf7d},
	} {
		archive := standard_library_reader_header_archive(1, 0, 0)
		standard_library_put_word_16(archive[12:], one.Clock)
		standard_library_put_word_16(archive[14:], one.Date)
		standard_library_read_timestamp_archive(archive)
	}
	for _, one := range []struct {
		Date    uint16
		Clock   uint16
		Seconds uint32
	}{
		{Seconds: 0},
		{Seconds: 1},
		{Seconds: 2},
		{Seconds: 1<<32 - 1},
		{Date: 0x21, Seconds: 315_576_000},
		{Date: 0x21, Seconds: 315_533_700},
		{Date: 0x21, Seconds: 315_532_801},
		{Date: 0x21, Seconds: 315_532_800},
		{Date: 0x21, Seconds: 315_532_799},
		{Date: 0x21, Seconds: 315_532_798},
		{Date: 0x21, Seconds: 315_531_900},
		{Date: 0x21, Seconds: 315_531_000},
		{Date: 0x21, Seconds: 315_482_400},
		{Date: 0x21, Seconds: 1<<32 - 1},
		{Date: 0xfc47, Clock: 0x3387, Seconds: 0},
	} {
		archive := standard_library_reader_header_archive(1, 9, 0)
		standard_library_put_word_16(archive[12:], one.Clock)
		standard_library_put_word_16(archive[14:], one.Date)
		extra := archive[47:56]
		copy(extra, []byte{0x55, 0x54, 5, 0, 1})
		standard_library_put_word_32(extra[5:], one.Seconds)
		standard_library_read_timestamp_archive(archive)
	}
}

func standard_library_civil_domains() {
	for _, modified := range []zip.Timestamp{
		{Seconds: 0, Zone_Offset_Seconds: time.Zone_Offset_Seconds(
			time.ZONE_OFFSET_SECONDS_MINIMUM,
		), Set: true},
		{Seconds: 0, Zone_Offset_Seconds: -1, Set: true},
		{Seconds: 0, Set: true},
		{Seconds: 1, Set: true},
		{Seconds: 2, Set: true},
		{Seconds: 315_532_800, Set: true},
		{Seconds: 315_532_802, Set: true},
		{Seconds: 315_532_804, Set: true},
		{Seconds: 315_619_199, Set: true},
		{Seconds: 86_399, Set: true},
		{Seconds: 86_400, Set: true},
		{Seconds: 172_800, Set: true},
		{Seconds: 2_764_800, Set: true},
		{
			Seconds: zip.Timestamp_Seconds(zip.TIMESTAMP_SECONDS_MAXIMUM),
			Zone_Offset_Seconds: time.Zone_Offset_Seconds(
				time.ZONE_OFFSET_SECONDS_MAXIMUM,
			),
			Set: true,
		},
	} {
		header := zip.Header_Unvalidated{Name: []byte("x")}
		zip.Header_Set_Modification_Time(&header, modified)
	}
}

func standard_library_mode_domains() {
	for _, mode := range []nbio.File_Mode{
		nbio.FILE_MODE_NAMED_PIPE,
		nbio.FILE_MODE_SOCKET | nbio.FILE_MODE_SET_USER_IDENTIFIER |
			nbio.FILE_MODE_SET_GROUP_IDENTIFIER | nbio.FILE_MODE_STICKY | 0o777,
	} {
		header := zip.Header_Unvalidated{Name: []byte("x")}
		zip.Header_Set_Mode(&header, mode)
	}
	for _, attributes := range []zip.Header_External_Attributes{0, 1, 0x10, 0x11} {
		header := zip.Header{External_Attributes: attributes}
		zip.Header_Mode(&header)
	}
	for _, unix := range []uint32{0, 1, 2, 0x4fff, 1<<16 - 1} {
		header := zip.Header{
			Creator_Version: zip.Header_Creator_Version(3 << 8),
			External_Attributes: zip.Header_External_Attributes(
				unix << 16,
			),
		}
		zip.Header_Mode(&header)
	}
	for _, unix := range []uint32{
		0, 1, 2, 0o777, 0x1000, 0x2000, 0x4fff, 0x6000, 0xc000,
		0x0800, 0x0400, 0x0200,
	} {
		archive := standard_library_reader_header_archive(1, 0, 0)
		standard_library_put_word_16(archive[4:], 3<<8)
		standard_library_put_word_32(archive[38:], unix<<16)
		standard_library_file_system_collect_archive(archive)
	}
}

func standard_library_file_system_path_domains() {
	for _, size := range []int{
		1, 2, STANDARD_LIBRARY_WRITER_RECORD_NAME_MAXIMUM,
	} {
		name := make([]byte, size)
		for position_index := range name {
			name[position_index] = 'a'
		}
		standard_library_file_system_path_archive(name, name)
	}
	for _, size := range []int{
		1, 2, STANDARD_LIBRARY_WRITER_RECORD_NAME_MAXIMUM - 2,
	} {
		root := make([]byte, size)
		for position_index := range root {
			root[position_index] = 'a'
		}
		name := make([]byte, size+2)
		copy(name, root)
		name[size] = '/'
		name[size+1] = 'x'
		standard_library_file_system_path_archive(name, root)
	}
	name := make([]byte, STANDARD_LIBRARY_WRITER_RECORD_NAME_MAXIMUM)
	for position_index := range name {
		if position_index%2 == 0 {
			name[position_index] = 'a'
		} else {
			name[position_index] = '/'
		}
	}
	standard_library_file_system_path_archive(name, []byte("."))
}

func standard_library_file_system_count_domains() {
	empty_reader := standard_library_reader_domain_archive(
		standard_library_reader_headers_archive(0),
	)
	var empty_file_system zip.File_System
	zip.File_System_Init(&empty_file_system, &empty_reader)
	zip.File_System_Read_Directory(&empty_file_system, []byte("."), nil)
	archive := standard_library_reader_distinct_headers_archive(86)
	reader := standard_library_reader_domain_archive(archive)
	var file_system zip.File_System
	zip.File_System_Init(&file_system, &reader)
	var entries [zip.ENTRY_COUNT_MAXIMUM]zip.Directory_Entry
	zip.File_System_Read_Directory(&file_system, []byte("."), entries[:])
	for _, names := range [][STANDARD_LIBRARY_SPECIAL_NODE_CAPACITY]byte{
		{'a', 'z'}, {'z', 'a'},
	} {
		archive = standard_library_reader_distinct_headers_archive(2)
		archive[46] = names[0]
		archive[93] = names[1]
		reader = standard_library_reader_domain_archive(archive)
		zip.File_System_Init(&file_system, &reader)
		zip.File_System_Read_Directory(&file_system, []byte("."), entries[:])
	}
}

func standard_library_reader_distinct_headers_archive(
	entry_count int,
) (archive []byte) {
	archive = standard_library_reader_headers_archive(entry_count)
	for index := 0; index < entry_count; index++ {
		name := byte(index + 'A')
		if name == '\\' {
			name = 200
		}
		archive[index*47+46] = name
	}
	return archive
}

func standard_library_file_system_path_archive(name []byte, path []byte) {
	archive := standard_library_reader_header_archive(len(name), 0, 0)
	copy(archive[46:], name)
	reader := standard_library_reader_domain_archive(archive)
	var file_system zip.File_System
	zip.File_System_Init(&file_system, &reader)
	var info zip.File_Info
	zip.File_System_Status(&file_system, path, &info)
	zip.File_System_Status(&file_system, []byte("missing"), &info)
	zip.File_System_Open(&file_system, []byte("missing"), nil)
	zip.File_System_Read_Directory(&file_system, []byte("missing"), nil)
	zip.File_System_Walk(
		&file_system, []byte("missing"), nil,
		func(_ zip.File_Info) (keep_going binary.Boolean) { return true },
	)
	if len(path) == STANDARD_LIBRARY_WRITER_RECORD_NAME_MAXIMUM {
		absent_path := make([]byte, len(path))
		copy(absent_path, path)
		absent_path[len(absent_path)-1]++
		zip.File_System_Status(&file_system, absent_path, &info)
	}
	var entries [zip.ENTRY_COUNT_MAXIMUM]zip.Directory_Entry
	zip.File_System_Read_Directory(&file_system, path, entries[:])
	var nodes [bytes.SLICE_SIZE_MAXIMUM]zip.File_Info
	zip.File_System_Walk(
		&file_system, path, nodes[:],
		func(_ zip.File_Info) (keep_going binary.Boolean) { return true },
	)
	zip.File_System_Walk(
		&file_system, []byte("."), nodes[:],
		func(_ zip.File_Info) (keep_going binary.Boolean) { return true },
	)
	zip.File_System_Read_Directory(&file_system, []byte("."), entries[:])
}

func standard_library_read_timestamp_archive(archive []byte) {
	reader := standard_library_reader_domain_archive(archive)
	var header zip.Header
	zip.Reader_Header(&reader, 0, &header)
	zip.Header_Modification_Time(&header)
	standard_library_file_system_collect_archive(archive)
}

func standard_library_writer_domain_value(marker int) (writer zip.Writer) {
	size := standard_library_domain_size(marker)
	workspace_count := size
	history_count := size
	if marker == INVARIANT_DOMAIN_MAXIMUM {
		workspace_count = flate.HASH_COUNT + 1
		history_count = flate.WINDOW_SIZE + 1
	}
	return zip.Writer{
		Storage: zip.Writer_Storage{
			Archive: make([]byte, size), Central: make([]byte, size),
			Content: make([]byte, size), Compressed: make([]byte, size),
			Comment: make([]byte, size), Name: make([]byte, size),
			Heads:    make([]int32, workspace_count),
			Previous: make([]int32, history_count),
		},
		Header:                   standard_library_validated_header_domain_value(marker),
		Status:                   standard_library_domain_status(marker),
		Count:                    zip.Writer_Count(size),
		Archive_Position:         zip.Writer_Archive_Position(size),
		Central_Position:         zip.Writer_Central_Position(size),
		Content_Position:         zip.Writer_Content_Position(size),
		Current_Central_Position: zip.Writer_Current_Central_Position(size),
		Offset:                   zip.Writer_Offset(size),
		Entry_Count:              standard_library_domain_entry_count(marker),
		Comment_Count: zip.Writer_Comment_Count(
			standard_library_domain_header_tail_size(marker),
		),
		Active:    zip.Writer_Active(marker != 0),
		Raw:       zip.Writer_Raw(marker != 0),
		Directory: zip.Writer_Directory(marker != 0),
		Closed:    zip.Writer_Closed(marker != 0),
		In_Flight: zip.Writer_In_Flight(marker != 0),
	}
}

func standard_library_reader_domain_value(marker int) (reader zip.Reader) {
	size := standard_library_domain_size(marker)
	stage := zip.Reader_Stage(marker)
	if marker == INVARIANT_DOMAIN_MAXIMUM {
		stage = zip.READER_STAGE_READ
	}
	storage := make([]byte, size)
	return zip.Reader{
		Storage:     zip.Reader_Storage{Archive: storage},
		Archive:     zip.Reader_Archive(storage),
		Status:      standard_library_domain_status(marker),
		Entry_Count: standard_library_domain_entry_count(marker),
		Comment: zip.Reader_Comment(make(
			[]byte, standard_library_domain_header_tail_size(marker),
		)),
		Stage:             stage,
		Active:            zip.Reader_Active(marker != 1),
		Transfer_Buffer:   zip.Reader_Transfer_Buffer(storage),
		Stream_Offset:     zip.Reader_Stream_Offset(size),
		Submission_Active: zip.Reader_Submission_Active(marker != 0),
		Continue:          zip.Reader_Continue(marker != 0),
	}
}

func standard_library_file_system_domain_value(marker int) (file_system zip.File_System) {
	reader := standard_library_reader_domain_value(marker)
	return zip.File_System{
		Reader: zip.File_System_Reader(reader),
		Valid:  zip.File_System_Valid(marker == 1 || marker == INVARIANT_DOMAIN_MAXIMUM),
	}
}

func standard_library_header_domain_value(
	marker int, field_size int,
) (header zip.Header_Unvalidated) {
	return zip.Header_Unvalidated{
		Name: make([]byte, field_size), Comment: make([]byte, field_size),
		Extra:    make([]byte, field_size),
		Method:   zip.Header_Method(standard_library_domain_word_16(marker)),
		Flags:    zip.Header_Flags(standard_library_domain_word_16(marker)),
		Checksum: zip.Header_Checksum(standard_library_domain_word_32(marker)),
		Compressed_Size: zip.Header_Compressed_Size_Unvalidated(
			standard_library_domain_header_size_unvalidated(marker),
		),
		Uncompressed_Size: zip.Header_Uncompressed_Size_Unvalidated(
			standard_library_domain_header_size_unvalidated(marker),
		),
		External_Attributes: zip.Header_External_Attributes(
			standard_library_domain_word_32(marker),
		),
		Creator_Version: zip.Header_Creator_Version(
			standard_library_domain_word_16(marker),
		),
		Extractor_Version: zip.Header_Extractor_Version(
			standard_library_domain_word_16(marker),
		),
		Modified_Time: zip.Header_Modified_Time(
			standard_library_domain_word_16(marker),
		),
		Modified_Date: zip.Header_Modified_Date(
			standard_library_domain_word_16(marker),
		),
		Internal_Attributes: zip.Header_Internal_Attributes(
			standard_library_domain_word_16(marker),
		),
		Modified: standard_library_domain_timestamp(marker),
		Non_UTF8: zip.Header_Non_UTF8(marker != 0),
	}
}

func standard_library_validated_header_domain_value(marker int) (header zip.Header) {
	unvalidated := standard_library_header_domain_value(marker, marker)
	validated, _ := zip.Header_Validate(&unvalidated)
	return validated
}

func standard_library_domain_timestamp(marker int) (timestamp zip.Timestamp) {
	seconds := zip.Timestamp_Seconds(marker)
	nanoseconds := time.Nanosecond_Count(marker)
	zone := time.Zone_Offset_Seconds(marker)
	if marker == 0 {
		zone = time.Zone_Offset_Seconds(time.ZONE_OFFSET_SECONDS_MINIMUM)
	}
	if marker == INVARIANT_DOMAIN_MAXIMUM {
		seconds = zip.Timestamp_Seconds(zip.TIMESTAMP_SECONDS_MAXIMUM)
		nanoseconds = time.Nanosecond_Count(time.NANOSECOND_COUNT_MAXIMUM)
		zone = time.Zone_Offset_Seconds(time.ZONE_OFFSET_SECONDS_MAXIMUM)
	}
	return zip.Timestamp{
		Seconds: seconds, Nanoseconds: nanoseconds,
		Zone_Offset_Seconds: zone, Set: zip.Timestamp_Set(marker != 0),
	}
}

func standard_library_domain_second_timestamp(
	marker int,
) (timestamp zip.Timestamp_Second_Precision) {
	value := standard_library_domain_timestamp(marker)
	return zip.Timestamp_Second_Precision{
		Seconds: value.Seconds, Zone_Offset_Seconds: value.Zone_Offset_Seconds,
		Set: value.Set,
	}
}

func standard_library_domain_size(marker int) (size int) {
	return marker
}

func standard_library_domain_header_tail_size(marker int) (size int) {
	if marker == INVARIANT_DOMAIN_MAXIMUM {
		return STANDARD_LIBRARY_WRITER_RECORD_TAIL_MAXIMUM
	}
	return marker
}

func standard_library_domain_word_16(marker int) (value uint16) {
	if marker == INVARIANT_DOMAIN_MAXIMUM {
		return 1<<16 - 1
	}
	return uint16(marker)
}

func standard_library_domain_word_32(marker int) (value uint32) {
	if marker == INVARIANT_DOMAIN_MAXIMUM {
		return 1<<32 - 1
	}
	return uint32(marker)
}

func standard_library_domain_word_64(marker int) (value uint64) {
	if marker == INVARIANT_DOMAIN_MAXIMUM {
		return 1<<64 - 1
	}
	return uint64(marker)
}

func standard_library_domain_header_size(marker int) (value uint64) {
	if marker == INVARIANT_DOMAIN_MAXIMUM {
		return 1<<32 - 1
	}
	return uint64(marker)
}

func standard_library_domain_header_size_unvalidated(marker int) (value uint64) {
	if marker == INVARIANT_DOMAIN_MAXIMUM {
		return 1 << 32
	}
	return uint64(marker)
}

func standard_library_domain_status(marker int) (status zip.Status) {
	if marker == INVARIANT_DOMAIN_MAXIMUM {
		return zip.STATUS_METHOD_UNSUPPORTED
	}
	return zip.Status(marker)
}

func standard_library_domain_entry_count(marker int) (count zip.Entry_Count) {
	if marker == INVARIANT_DOMAIN_MAXIMUM {
		return zip.Entry_Count(zip.ENTRY_COUNT_MAXIMUM)
	}
	return zip.Entry_Count(marker)
}

func standard_library_domain_file_info_header_index(
	marker int,
) (index zip.File_Info_Header_Index) {
	if marker == INVARIANT_DOMAIN_MAXIMUM {
		return STANDARD_LIBRARY_FILE_SYSTEM_HEADER_INDEX_MAXIMUM
	}
	return zip.File_Info_Header_Index(marker)
}

func standard_library_domain_file_mode(marker int) (mode nbio.File_Mode) {
	if marker == INVARIANT_DOMAIN_MAXIMUM {
		return nbio.FILE_MODE_VALID
	}
	return nbio.File_Mode(marker)
}

type standard_library_stream_harness struct {
	Output         [bytes.SLICE_SIZE_MAXIMUM]byte
	Archive        [bytes.SLICE_SIZE_MAXIMUM]byte
	Reader_Archive [bytes.SLICE_SIZE_MAXIMUM]byte
	Central        [bytes.SLICE_SIZE_MAXIMUM]byte
	Content        [bytes.SLICE_SIZE_MAXIMUM]byte
	Compressed     [bytes.SLICE_SIZE_MAXIMUM]byte
	Comment        [bytes.SLICE_SIZE_MAXIMUM]byte
	Name           [bytes.SLICE_SIZE_MAXIMUM]byte
	Heads          [flate.HASH_COUNT]int32
	Previous       [flate.WINDOW_SIZE]int32
	Memory         nbio.Stream_Memory
	Writer         zip.Writer
}

func standard_library_stream_harness_init(
	t *testing.T, harness *standard_library_stream_harness,
) {
	t.Helper()
	harness.Memory = nbio.Stream_Memory{Memory: harness.Output[:]}
	status := zip.Writer_Init(
		&harness.Writer,
		nbio.Memory_To_Stream(&harness.Memory),
		zip.Writer_Storage{
			Archive:    harness.Archive[:],
			Central:    harness.Central[:],
			Content:    harness.Content[:],
			Compressed: harness.Compressed[:],
			Comment:    harness.Comment[:],
			Name:       harness.Name[:],
			Heads:      harness.Heads[:],
			Previous:   harness.Previous[:],
		},
	)
	testify.Equal_Values(t, zip.STATUS_OK, status)
}

func standard_library_stream_harness_close_and_read(
	t *testing.T, harness *standard_library_stream_harness,
) (reader zip.Reader) {
	t.Helper()
	var completion time.Completion
	closed := false
	zip.Writer_Close(
		&harness.Writer, &completion, func(_ *time.Completion) { closed = true },
	)
	testify.True(t, closed)
	testify.No_Error(t, completion.Error)
	testify.Equal(t, zip.STATUS_OK, harness.Writer.Status)
	input_size := int(harness.Writer.Offset) + int(harness.Writer.Count)
	input := nbio.Stream_Memory{Memory: harness.Output[:input_size]}
	initialized := false
	zip.Reader_Init(
		&reader, nbio.Memory_To_Stream(&input),
		zip.Reader_Storage{Archive: harness.Reader_Archive[:]},
		&completion, func(_ *time.Completion) { initialized = true },
	)
	testify.True(t, initialized)
	testify.No_Error(t, completion.Error)
	testify.Equal(t, zip.STATUS_OK, reader.Status)
	return reader
}

func standard_library_file_system_archive(
	t *testing.T, harness *standard_library_stream_harness,
) {
	t.Helper()
	for _, one := range []struct {
		Name      string
		Content   string
		Directory bool
	}{
		{Name: "a/b/", Directory: true},
		{Name: "a/b/c", Content: "content"},
		{Name: "a/d", Content: "d"},
		{Name: "root.txt", Content: "root"},
	} {
		header := zip.Header_Unvalidated{
			Name: []byte(one.Name), Method: zip.METHOD_STORE,
		}
		if one.Directory {
			zip.Header_Set_Mode(&header, 0o755|nbio.FILE_MODE_DIRECTORY)
		}
		testify.Equal_Values(t, zip.STATUS_OK, zip.Writer_Create(&harness.Writer, &header))
		count, status := zip.Writer_Write(&harness.Writer, []byte(one.Content))
		testify.Equal_Values(t, zip.STATUS_OK, status)
		testify.Equal(t, bytes.Boundary(len(one.Content)), count)
	}
}

func standard_library_file_system_walk(t *testing.T, file_system *zip.File_System) {
	t.Helper()
	var nodes [STANDARD_LIBRARY_FILE_SYSTEM_CAPACITY]zip.File_Info
	var visited [STANDARD_LIBRARY_FILE_SYSTEM_CAPACITY]bytes.Slice
	visited_count := 0
	count, status := zip.File_System_Walk(
		file_system, []byte("."), nodes[:], func(
			info zip.File_Info,
		) (keep_going binary.Boolean) {
			visited[visited_count] = bytes.Slice(info.Name)
			visited_count++
			return true
		},
	)
	testify.Equal_Values(t, zip.STATUS_OK, status)
	expected := []bytes.Slice{
		bytes.Slice("."), bytes.Slice("a"), bytes.Slice("a/b"),
		bytes.Slice("a/b/c"), bytes.Slice("a/d"), bytes.Slice("root.txt"),
	}
	testify.Equal_Values(t, bytes.Boundary(len(expected)), count)
	testify.Equal(t, expected, visited[:visited_count])
}

func assert_invalid_suffixes(t *testing.T, archive []byte, decoded []byte) {
	t.Helper()
	for _, suffix := range []bytes.Text{"", "bad/name", "bad\x00name"} {
		count, status := zip.Decode_Into(archive, suffix, decoded[:])
		testify.Zero(t, count, suffix)
		testify.Equal(t, zip.STATUS_INPUT_INVALID, status, suffix)
	}
}

func deflated(t *testing.T, content []byte) (compressed []byte) {
	return deflated_level(t, content, flate.DEFAULT_COMPRESSION)
}

func deflated_level(
	t *testing.T, content []byte, level int,
) (compressed []byte) {
	t.Helper()
	var compressed_storage [bytes.SLICE_SIZE_MAXIMUM]byte
	var workspace_storage compression_workspace
	workspace := flate.Workspace_Unvalidated{
		Heads:    workspace_storage.Heads[:],
		Previous: workspace_storage.Previous[:],
	}
	count, status := flate.Encode_Into(
		compressed_storage[:], workspace, content,
		flate.Level_Unvalidated(level),
	)
	testify.Equal(t, flate.Encode_Status(flate.STATUS_OK), status)
	return compressed_storage[:count]
}

func archive_data(
	storage []byte,
	name []byte,
	method uint16,
	compressed []byte,
	uncompressed []byte,
) (archive []byte) {
	const LOCAL_HEADER_SIZE = STANDARD_LIBRARY_LOCAL_HEADER_SIZE
	const CENTRAL_HEADER_SIZE = STANDARD_LIBRARY_CENTRAL_HEADER_SIZE
	const END_SIZE = STANDARD_LIBRARY_DIRECTORY_END_SIZE
	checksum := crc32.ChecksumIEEE(uncompressed)
	local := storage[:LOCAL_HEADER_SIZE]
	put_word_32(local[0:], 0x04034b50)
	put_word_16(local[4:], 20)
	put_word_16(local[8:], method)
	put_word_32(local[14:], checksum)
	put_word_32(local[18:], uint32(len(compressed)))
	put_word_32(local[22:], uint32(len(uncompressed)))
	put_word_16(local[26:], uint16(len(name)))
	position := LOCAL_HEADER_SIZE
	position += copy(storage[position:], name)
	position += copy(storage[position:], compressed)
	central_position := position
	central := storage[position : position+CENTRAL_HEADER_SIZE]
	put_word_32(central[0:], 0x02014b50)
	put_word_16(central[4:], 20)
	put_word_16(central[6:], 20)
	put_word_16(central[10:], method)
	put_word_32(central[16:], checksum)
	put_word_32(central[20:], uint32(len(compressed)))
	put_word_32(central[24:], uint32(len(uncompressed)))
	put_word_16(central[28:], uint16(len(name)))
	position += CENTRAL_HEADER_SIZE
	position += copy(storage[position:], name)
	central_size := position - central_position
	end := storage[position : position+END_SIZE]
	put_word_32(end[0:], 0x06054b50)
	put_word_16(end[8:], 1)
	put_word_16(end[10:], 1)
	put_word_32(end[12:], uint32(central_size))
	put_word_32(end[16:], uint32(central_position))
	position += END_SIZE
	return storage[:position]
}

func central_archive_data(
	storage []byte, name_size int, extra_size int, comment_size int,
) (archive []byte) {
	const CENTRAL_HEADER_SIZE = STANDARD_LIBRARY_CENTRAL_HEADER_SIZE
	const DIRECTORY_END_SIZE = STANDARD_LIBRARY_DIRECTORY_END_SIZE
	central := storage[:CENTRAL_HEADER_SIZE]
	put_word_32(central[0:], 0x02014b50)
	put_word_16(central[4:], 20)
	put_word_16(central[6:], 20)
	put_word_16(central[28:], uint16(name_size))
	put_word_16(central[30:], uint16(extra_size))
	put_word_16(central[32:], uint16(comment_size))
	position := CENTRAL_HEADER_SIZE
	for name_index := 0; name_index < name_size; name_index++ {
		storage[position+name_index] = 'a'
	}
	position += name_size + extra_size + comment_size
	end := storage[position : position+DIRECTORY_END_SIZE]
	put_word_32(end[0:], 0x06054b50)
	put_word_16(end[8:], 1)
	put_word_16(end[10:], 1)
	put_word_32(end[12:], uint32(position))
	position += DIRECTORY_END_SIZE
	return storage[:position]
}

func central_entries_archive_data(
	storage []byte, entry_count int,
) (archive []byte) {
	const CENTRAL_HEADER_SIZE = STANDARD_LIBRARY_CENTRAL_HEADER_SIZE
	const DIRECTORY_END_SIZE = STANDARD_LIBRARY_DIRECTORY_END_SIZE
	position := 0
	for entry_index := 0; entry_index < entry_count; entry_index++ {
		header := storage[position : position+CENTRAL_HEADER_SIZE]
		put_word_32(header[0:], 0x02014b50)
		put_word_16(header[4:], 20)
		put_word_16(header[6:], 20)
		position += CENTRAL_HEADER_SIZE
	}
	end := storage[position : position+DIRECTORY_END_SIZE]
	put_word_32(end[0:], 0x06054b50)
	put_word_16(end[8:], uint16(entry_count))
	put_word_16(end[10:], uint16(entry_count))
	put_word_32(end[12:], uint32(position))
	position += DIRECTORY_END_SIZE
	return storage[:position]
}

func directory_end_archive_data(
	storage []byte, prefix_size int,
	central_size uint32, central_offset uint32,
) (archive []byte) {
	const DIRECTORY_END_SIZE = STANDARD_LIBRARY_DIRECTORY_END_SIZE
	end := storage[prefix_size : prefix_size+DIRECTORY_END_SIZE]
	put_word_32(end[0:], 0x06054b50)
	put_word_32(end[12:], central_size)
	put_word_32(end[16:], central_offset)
	return storage[:prefix_size+DIRECTORY_END_SIZE]
}

func selected_central_archive_data(
	storage []byte, prefix_size int, valid_signature bool,
) (archive []byte) {
	const CENTRAL_HEADER_SIZE = STANDARD_LIBRARY_CENTRAL_HEADER_SIZE
	const DIRECTORY_END_SIZE = STANDARD_LIBRARY_DIRECTORY_END_SIZE
	const NAME_SIZE = STANDARD_LIBRARY_NAME_SIZE_MINIMUM
	header := storage[prefix_size : prefix_size+CENTRAL_HEADER_SIZE]
	if valid_signature {
		put_word_32(header[0:], 0x02014b50)
	}
	put_word_16(header[4:], 20)
	put_word_16(header[6:], 20)
	put_word_16(header[28:], NAME_SIZE)
	archive_size := prefix_size + CENTRAL_HEADER_SIZE
	storage[archive_size] = 'x'
	archive_size += NAME_SIZE
	end := storage[archive_size : archive_size+DIRECTORY_END_SIZE]
	put_word_32(end[0:], 0x06054b50)
	put_word_16(end[8:], 1)
	put_word_16(end[10:], 1)
	put_word_32(end[12:], CENTRAL_HEADER_SIZE+NAME_SIZE)
	archive_size += DIRECTORY_END_SIZE
	return storage[:archive_size]
}

func maximum_central_cursor_archive_data(storage []byte) (archive []byte) {
	const CENTRAL_HEADER_SIZE = STANDARD_LIBRARY_CENTRAL_HEADER_SIZE
	const DIRECTORY_END_SIZE = STANDARD_LIBRARY_DIRECTORY_END_SIZE
	const FIRST_NAME_SIZE = bytes.SLICE_SIZE_MAXIMUM -
		2*CENTRAL_HEADER_SIZE - DIRECTORY_END_SIZE
	first := storage[:CENTRAL_HEADER_SIZE]
	put_word_32(first[0:], 0x02014b50)
	put_word_16(first[4:], 20)
	put_word_16(first[6:], 20)
	put_word_16(first[28:], FIRST_NAME_SIZE)
	archive_size := CENTRAL_HEADER_SIZE
	name := storage[archive_size : archive_size+FIRST_NAME_SIZE]
	for name_index := range name {
		name[name_index] = 'a'
	}
	archive_size += FIRST_NAME_SIZE
	second := storage[archive_size : archive_size+CENTRAL_HEADER_SIZE]
	put_word_32(second[0:], 0x02014b50)
	put_word_16(second[4:], 20)
	put_word_16(second[6:], 20)
	archive_size += CENTRAL_HEADER_SIZE
	end := storage[archive_size : archive_size+DIRECTORY_END_SIZE]
	put_word_32(end[0:], 0x06054b50)
	put_word_16(end[8:], 2)
	put_word_16(end[10:], 2)
	put_word_32(end[12:], uint32(archive_size))
	archive_size += DIRECTORY_END_SIZE
	return storage[:archive_size]
}

func prefixed_empty_archive_data(
	storage []byte, prefix_size int,
) (archive []byte) {
	segment := archive_data(
		storage[prefix_size:], []byte("x"), 0, nil, nil,
	)
	return storage[:prefix_size+len(segment)]
}

func prefixed_stored_descriptor_archive_data(
	storage []byte, prefix_size int,
) (archive []byte) {
	const LOCAL_HEADER_SIZE = STANDARD_LIBRARY_LOCAL_HEADER_SIZE
	const DIRECTORY_END_SIZE = STANDARD_LIBRARY_DIRECTORY_END_SIZE
	const DESCRIPTOR_SIZE = STANDARD_LIBRARY_DATA_DESCRIPTOR_SIZE
	const FLAG_DATA_DESCRIPTOR = zip.FLAG_DATA_DESCRIPTOR
	segment_storage := storage[prefix_size:]
	segment := archive_data(
		segment_storage, []byte("x"), 0, nil, nil,
	)
	central_position := LOCAL_HEADER_SIZE + 1
	tail_size := len(segment) - central_position
	bytes.Clone_Into(
		segment_storage[central_position+DESCRIPTOR_SIZE:],
		segment_storage[central_position:central_position+tail_size],
	)
	put_word_32(segment_storage[central_position:], 0)
	put_word_32(segment_storage[central_position+4:], 0)
	put_word_32(segment_storage[central_position+8:], 0)
	put_word_16(segment_storage[6:], FLAG_DATA_DESCRIPTOR)
	central_position += DESCRIPTOR_SIZE
	put_word_16(segment_storage[central_position+8:], FLAG_DATA_DESCRIPTOR)
	end_position := len(segment) - DIRECTORY_END_SIZE + DESCRIPTOR_SIZE
	put_word_32(segment_storage[end_position+16:], uint32(central_position))
	segment_size := len(segment) + DESCRIPTOR_SIZE
	return storage[:prefix_size+segment_size]
}

func descriptor_archive_data(
	storage []byte, name []byte, compressed []byte, uncompressed []byte,
) (archive []byte) {
	const LOCAL_HEADER_SIZE = STANDARD_LIBRARY_LOCAL_HEADER_SIZE
	const END_SIZE = STANDARD_LIBRARY_DIRECTORY_END_SIZE
	const DESCRIPTOR_SIZE = STANDARD_LIBRARY_SIGNED_DESCRIPTOR_SIZE
	const FLAG_DATA_DESCRIPTOR = zip.FLAG_DATA_DESCRIPTOR
	archive = archive_data(storage, name, 8, compressed, uncompressed)
	central_position := LOCAL_HEADER_SIZE + len(name) + len(compressed)
	end_position := len(archive) - END_SIZE
	tail_size := len(archive) - central_position
	tail_destination := central_position + DESCRIPTOR_SIZE
	bytes.Clone_Into(
		storage[tail_destination:tail_destination+tail_size],
		storage[central_position:len(archive)],
	)
	put_word_16(storage[6:], FLAG_DATA_DESCRIPTOR)
	put_word_32(storage[14:], 0)
	put_word_32(storage[18:], 0)
	put_word_32(storage[22:], 0)
	descriptor := storage[central_position : central_position+DESCRIPTOR_SIZE]
	put_word_32(descriptor[0:], 0x08074b50)
	put_word_32(descriptor[4:], crc32.ChecksumIEEE(uncompressed))
	put_word_32(descriptor[8:], uint32(len(compressed)))
	put_word_32(descriptor[12:], uint32(len(uncompressed)))
	central_position += DESCRIPTOR_SIZE
	put_word_16(storage[central_position+8:], FLAG_DATA_DESCRIPTOR)
	end_position += DESCRIPTOR_SIZE
	put_word_32(storage[end_position+16:], uint32(central_position))
	return storage[:len(archive)+DESCRIPTOR_SIZE]
}

func put_word_16(destination []byte, value uint16) {
	binary.Put_Uint_16(
		binary.Bytes(destination), binary.Word_16(value), binary.LITTLE_ENDIAN,
	)
}

func put_word_32(destination []byte, value uint32) {
	binary.Put_Uint_32(
		binary.Bytes(destination), binary.Word_32(value), binary.LITTLE_ENDIAN,
	)
}
