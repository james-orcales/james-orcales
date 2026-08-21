package gzip_test

import (
	"hash/crc32"
	"testing"

	"local/james-orcales/shared/compress/gzip"
	"local/james-orcales/shared/testify"
)

// Test_Gzip_Members protects stdlib interoperability and iterative concatenation.
func Test_Gzip_Members(t *testing.T) {
	test_standard_member(t)
	test_member_round_trip(t)
	test_concatenated_members(t)
	test_empty_member_sequence(t)
}

// Test_Header_Metadata protects bounded Latin-1 conversion and optional header CRC.
func Test_Header_Metadata(t *testing.T) {
	test_metadata_round_trip(t)
	test_complete_header(t)
	test_header_rejections(t)
}

// Test_Compression_Levels keeps stdlib identities and bounded validation.
func Test_Compression_Levels(t *testing.T) {
	levels := []struct {
		Level gzip.Level_Unvalidated
		Want  int
	}{
		{gzip.NO_COMPRESSION, 0},
		{gzip.BEST_SPEED, 1},
		{gzip.BEST_COMPRESSION, 9},
		{gzip.DEFAULT_COMPRESSION, -1},
		{gzip.HUFFMAN_ONLY, -2},
	}
	var workspace_storage test_workspace
	workspace := test_workspace_value(&workspace_storage)
	var compressed [TEST_DESTINATION_SIZE]byte
	var destination [TEST_DESTINATION_SIZE]byte
	for _, level := range levels {
		testify.Equal(t, level.Want, int(level.Level), "compression level")
		count, status := gzip.Encode_Into(
			compressed[:], workspace, []byte(TEST_HELLO_CONTENT),
			gzip.Header_Unvalidated{}, level.Level,
		)
		testify.Equal_Values(t, gzip.STATUS_OK, status, "level %d encode", level.Level)
		decoded_count, _, decoded_status := gzip.Decode_Into(
			destination[:], gzip.Header_Storage_Unvalidated{}, compressed[:count],
		)
		testify.Equal_Values(
			t, gzip.STATUS_OK, decoded_status, "level %d decode", level.Level,
		)
		testify.Equal(
			t, TEST_HELLO_CONTENT, string(destination[:decoded_count]),
			"level %d output", level.Level,
		)
	}
	_, status := gzip.Encode_Into(
		compressed[:], workspace, nil, gzip.Header_Unvalidated{}, 10,
	)
	testify.Equal_Values(t, gzip.STATUS_LEVEL_INVALID, status, "invalid level")
}

// Test_Bounds protects caller-owned byte, metadata, and workspace limits.
func Test_Bounds(t *testing.T) {
	testify.Equal(t, 64*1024*1024, gzip.BYTE_COUNT_MAXIMUM, "byte boundary")
	testify.Equal(t, 1<<16-1, gzip.EXTRA_SIZE_MAXIMUM, "extra boundary")
	testify.Equal(t, 511, gzip.HEADER_STRING_BYTE_COUNT_MAXIMUM, "header string boundary")

	compressed := test_hello_compressed()
	var short [TEST_SHORT_DESTINATION_SIZE]byte
	var header_name [gzip.HEADER_TEXT_SIZE_MAXIMUM]byte
	count, _, decode_status := gzip.Decode_Into(
		short[:], gzip.Header_Storage_Unvalidated{Name: header_name[:]}, compressed[:],
	)
	testify.Equal_Values(t, gzip.STATUS_OUTPUT_TOO_SMALL, decode_status, "short decode")
	testify.Equal(t, len(short), int(count), "short decode count")

	var workspace_storage test_workspace
	workspace := test_workspace_value(&workspace_storage)
	encoded_count, encode_status := gzip.Encode_Into(
		short[:], workspace, []byte(TEST_HELLO_CONTENT),
		gzip.Header_Unvalidated{}, gzip.DEFAULT_COMPRESSION,
	)
	testify.Equal_Values(t, gzip.STATUS_OUTPUT_TOO_SMALL, encode_status, "short encode")
	testify.Less_Or_Equal(
		t, &testify.Less_Or_Equal_Input[int]{
			First: int(encoded_count), Second: len(short),
		}, "short encode count",
	)
	test_oversized_storage(t, workspace)
	test_workspace_bounds(t, workspace)
	test_small_public_storage(t, workspace, compressed[:])
	test_invariant_domains(t)

	var missing_workspace gzip.Workspace_Unvalidated
	_, encode_status = gzip.Encode_Into(
		short[:], missing_workspace, nil, gzip.Header_Unvalidated{},
		gzip.DEFAULT_COMPRESSION,
	)
	testify.Equal_Values(t, gzip.STATUS_STORAGE_INVALID, encode_status, "missing workspace")
}

// Test_Allocation proves every public result path owns no heap storage.
func Test_Allocation(t *testing.T) {
	t.Run("Encode", test_encode_allocation)
	t.Run("Decode", test_decode_allocation)
}

// Test_Untrusted_Input protects all parser stages and harmful aliases.
func Test_Untrusted_Input(t *testing.T) {
	test_truncated_input(t)
	test_corrupt_input(t)
	test_suffix_and_overlap(t)
	test_input_mutations(t)
}

func test_oversized_storage(
	t *testing.T, workspace gzip.Workspace_Unvalidated,
) {
	t.Helper()
	oversized := make([]byte, gzip.BYTE_COUNT_MAXIMUM+1)
	_, status := gzip.Encode_Into(
		oversized, workspace, nil, gzip.Header_Unvalidated{},
		gzip.DEFAULT_COMPRESSION,
	)
	testify.Equal_Values(t, gzip.STATUS_STORAGE_INVALID, status, "oversized encode output")
	_, status = gzip.Encode_Into(
		nil, workspace, oversized, gzip.Header_Unvalidated{},
		gzip.DEFAULT_COMPRESSION,
	)
	testify.Equal_Values(t, gzip.STATUS_STORAGE_INVALID, status, "oversized source")
	_, _, decode_status := gzip.Decode_Into(
		oversized, gzip.Header_Storage_Unvalidated{}, nil,
	)
	testify.Equal_Values(
		t, gzip.STATUS_STORAGE_INVALID, decode_status, "oversized decode output",
	)
	_, _, decode_status = gzip.Decode_Into(
		nil, gzip.Header_Storage_Unvalidated{}, oversized,
	)
	testify.Equal_Values(
		t, gzip.STATUS_STORAGE_INVALID, decode_status, "oversized compressed input",
	)
	_, _, decode_status = gzip.Decode_Into(
		nil, gzip.Header_Storage_Unvalidated{Extra: oversized}, nil,
	)
	testify.Equal_Values(
		t, gzip.STATUS_STORAGE_INVALID, decode_status, "oversized header storage",
	)
	test_oversized_metadata(t, workspace, oversized)
	test_oversized_member_storage(t, oversized)
}

func test_oversized_metadata(
	t *testing.T, workspace gzip.Workspace_Unvalidated, oversized []byte,
) {
	t.Helper()
	headers := [...]gzip.Header_Unvalidated{
		{Extra: oversized},
		{Name: oversized},
		{Comment: oversized},
	}
	for index, header := range headers {
		_, status := gzip.Encode_Into(
			nil, workspace, nil, header, gzip.DEFAULT_COMPRESSION,
		)
		testify.Equal_Values(
			t, gzip.STATUS_HEADER_INVALID, status, "oversized header %d", index,
		)
	}
	storages := [...]gzip.Header_Storage_Unvalidated{
		{Name: oversized},
		{Comment: oversized},
	}
	for index, storage := range storages {
		_, _, status := gzip.Decode_Into(nil, storage, nil)
		testify.Equal_Values(
			t, gzip.STATUS_STORAGE_INVALID, status,
			"oversized header storage %d", index,
		)
	}
}

func test_oversized_member_storage(t *testing.T, oversized []byte) {
	t.Helper()
	_, _, _, status := gzip.Decode_Member_Into(
		oversized, gzip.Header_Storage_Unvalidated{}, nil,
	)
	testify.Equal_Values(
		t, gzip.STATUS_STORAGE_INVALID, status, "oversized member output",
	)
	_, _, _, status = gzip.Decode_Member_Into(
		nil, gzip.Header_Storage_Unvalidated{}, oversized,
	)
	testify.Equal_Values(
		t, gzip.STATUS_STORAGE_INVALID, status, "oversized member input",
	)
	_, _, _, status = gzip.Decode_Member_Into(
		nil, gzip.Header_Storage_Unvalidated{Extra: oversized}, nil,
	)
	testify.Equal_Values(
		t, gzip.STATUS_STORAGE_INVALID, status, "oversized member header storage",
	)
	for index, storage := range [...]gzip.Header_Storage_Unvalidated{
		{Name: oversized}, {Comment: oversized},
	} {
		_, _, _, status = gzip.Decode_Member_Into(nil, storage, nil)
		testify.Equal_Values(
			t, gzip.STATUS_STORAGE_INVALID, status,
			"oversized member metadata %d", index,
		)
	}
}

func test_workspace_bounds(
	t *testing.T, workspace gzip.Workspace_Unvalidated,
) {
	t.Helper()
	workspaces := [...]gzip.Workspace_Unvalidated{
		{Heads: make([]int32, 1), Previous: workspace.Previous},
		{Heads: make([]int32, 2), Previous: workspace.Previous},
		{Heads: make([]int32, gzip.HASH_POSITIONS_COUNT+1), Previous: workspace.Previous},
		{Heads: workspace.Heads, Previous: make([]int32, 1)},
		{Heads: workspace.Heads, Previous: make([]int32, 2)},
		{Heads: workspace.Heads, Previous: make([]int32, gzip.HISTORY_POSITIONS_COUNT+1)},
	}
	for index, invalid := range workspaces {
		_, status := gzip.Encode_Into(
			nil, invalid, nil, gzip.Header_Unvalidated{}, gzip.DEFAULT_COMPRESSION,
		)
		testify.Equal_Values(
			t, gzip.STATUS_STORAGE_INVALID, status, "workspace %d", index,
		)
	}
}

func test_small_public_storage(
	t *testing.T, workspace gzip.Workspace_Unvalidated, compressed []byte,
) {
	t.Helper()
	for size := 1; size <= 2; size++ {
		destination := make([]byte, size)
		var name [gzip.HEADER_TEXT_SIZE_MAXIMUM]byte
		storage := gzip.Header_Storage_Unvalidated{Name: name[:]}
		count, status := gzip.Encode_Into(
			destination, workspace, nil, gzip.Header_Unvalidated{},
			gzip.DEFAULT_COMPRESSION,
		)
		testify.Equal_Values(
			t, gzip.STATUS_OUTPUT_TOO_SMALL, status, "small encode size %d", size,
		)
		testify.Equal(t, size, int(count), "small encode size %d count", size)
		decoded_count, _, decode_status := gzip.Decode_Into(
			destination, storage, compressed,
		)
		testify.Equal_Values(
			t, gzip.STATUS_OUTPUT_TOO_SMALL, decode_status,
			"small decode size %d", size,
		)
		testify.Less_Or_Equal(
			t, &testify.Less_Or_Equal_Input[int]{
				First: int(decoded_count), Second: size,
			}, "small decode size %d count", size,
		)
		gzip.Decode_Member_Into(
			destination, storage, compressed[:size],
		)
	}
}

func test_invariant_domains(t *testing.T) {
	t.Helper()
	test_metadata_domains(t)
	test_level_domains(t)
	test_maximum_encoded_domain(t)
	test_maximum_compressed_domain(t)
	test_parser_domains(t)
	test_maximum_header_checksum_domain(t)
	test_maximum_output_domain(t)
}

func test_parser_domains(t *testing.T) {
	t.Helper()
	minimal := [TEST_FIXED_HEADER_SIZE]byte{0x1f, 0x8b, 0x08}
	_, _, _, status := gzip.Decode_Member_Into(
		nil, gzip.Header_Storage_Unvalidated{}, minimal[:],
	)
	testify.Equal_Values(t, gzip.STATUS_INPUT_INVALID, status, "minimal member")
	checksum := minimal
	checksum[3] = TEST_FLAG_HEADER_CHECKSUM
	_, _, _, status = gzip.Decode_Member_Into(
		nil, gzip.Header_Storage_Unvalidated{}, checksum[:],
	)
	testify.Equal_Values(
		t, gzip.STATUS_HEADER_INVALID, status, "missing header checksum",
	)
	comment := minimal
	comment[3] = 1 << 4
	_, _, _, status = gzip.Decode_Member_Into(
		nil, gzip.Header_Storage_Unvalidated{}, comment[:],
	)
	testify.Equal_Values(t, gzip.STATUS_HEADER_INVALID, status, "unterminated comment")
	var overlong [TEST_FIXED_HEADER_SIZE + gzip.HEADER_STRING_BYTE_COUNT_MAXIMUM + 1]byte
	overlong[0] = 0x1f
	overlong[1] = 0x8b
	overlong[2] = 0x08
	overlong[3] = 1 << 4
	for position := TEST_FIXED_HEADER_SIZE; position < len(overlong); position++ {
		overlong[position] = 'a'
	}
	_, _, _, status = gzip.Decode_Member_Into(
		nil, gzip.Header_Storage_Unvalidated{}, overlong[:],
	)
	testify.Equal_Values(t, gzip.STATUS_HEADER_INVALID, status, "overlong comment")
	test_maximum_header_flags(t)
	test_minimum_text_header(t)
}

func test_maximum_header_flags(t *testing.T) {
	t.Helper()
	compressed := test_complete_header_compressed()
	compressed[3] = 1<<8 - 1
	checksum := crc32.ChecksumIEEE(compressed[:TEST_HEADER_CRC_POSITION])
	compressed[TEST_HEADER_CRC_POSITION] = byte(checksum)
	compressed[TEST_HEADER_CRC_POSITION+1] = byte(checksum >> 8)
	var extra [TEST_HEADER_EXTRA_SIZE]byte
	var name [gzip.HEADER_TEXT_SIZE_MAXIMUM]byte
	var comment [gzip.HEADER_TEXT_SIZE_MAXIMUM]byte
	storage := gzip.Header_Storage_Unvalidated{
		Extra: extra[:], Name: name[:], Comment: comment[:],
	}
	_, _, _, status := gzip.Decode_Member_Into(nil, storage, compressed)
	testify.Equal_Values(t, gzip.STATUS_OK, status, "maximum header flags")
}

func test_minimum_text_header(t *testing.T) {
	t.Helper()
	var workspace_storage test_workspace
	var compressed [TEST_DOMAIN_DESTINATION_SIZE]byte
	_, status := gzip.Encode_Into(
		compressed[:], test_workspace_value(&workspace_storage), nil,
		gzip.Header_Unvalidated{Name: []byte{'a'}}, gzip.DEFAULT_COMPRESSION,
	)
	testify.Equal_Values(t, gzip.STATUS_OK, status, "minimum text header")
}

func test_maximum_header_checksum_domain(t *testing.T) {
	t.Helper()
	extra := make([]byte, gzip.EXTRA_SIZE_MAXIMUM)
	name := test_domain_text(gzip.HEADER_TEXT_SIZE_MAXIMUM)
	comment := test_domain_text(gzip.HEADER_TEXT_SIZE_MAXIMUM)
	var compressed [gzip.HEADER_SIZE_MAXIMUM + TEST_DOMAIN_DESTINATION_SIZE]byte
	var workspace_storage test_workspace
	count, status := gzip.Encode_Into(
		compressed[:], test_workspace_value(&workspace_storage), nil,
		gzip.Header_Unvalidated{Extra: extra, Name: name, Comment: comment},
		gzip.DEFAULT_COMPRESSION,
	)
	testify.Equal_Values(t, gzip.STATUS_OK, status, "maximum checksum header encode")
	member := make([]byte, int(count)+gzip.HEADER_CHECKSUM_SIZE)
	copy(member[:gzip.HEADER_SIZE_MAXIMUM], compressed[:gzip.HEADER_SIZE_MAXIMUM])
	member[3] |= TEST_FLAG_HEADER_CHECKSUM
	checksum := crc32.ChecksumIEEE(member[:gzip.HEADER_SIZE_MAXIMUM])
	member[gzip.HEADER_SIZE_MAXIMUM] = byte(checksum)
	member[gzip.HEADER_SIZE_MAXIMUM+1] = byte(checksum >> 8)
	copy(
		member[gzip.HEADER_POSITION_MAXIMUM:],
		compressed[gzip.HEADER_SIZE_MAXIMUM:count],
	)
	decoded_extra := make([]byte, gzip.EXTRA_SIZE_MAXIMUM)
	decoded_name := make([]byte, gzip.HEADER_TEXT_SIZE_MAXIMUM)
	decoded_comment := make([]byte, gzip.HEADER_TEXT_SIZE_MAXIMUM)
	storage := gzip.Header_Storage_Unvalidated{
		Extra: decoded_extra, Name: decoded_name, Comment: decoded_comment,
	}
	_, consumed, _, decode_status := gzip.Decode_Member_Into(nil, storage, member)
	testify.Equal_Values(
		t, gzip.STATUS_OK, decode_status, "maximum checksum header decode",
	)
	testify.Equal(t, len(member), int(consumed), "maximum checksum header consumed")
}

func test_maximum_output_domain(t *testing.T) {
	t.Helper()
	destination := make([]byte, gzip.BYTE_COUNT_MAXIMUM)
	compressed := test_maximum_output_compressed(destination)
	member_count, _, _, member_status := gzip.Decode_Member_Into(
		destination, gzip.Header_Storage_Unvalidated{}, compressed,
	)
	testify.Equal_Values(t, gzip.STATUS_OK, member_status, "maximum member output")
	testify.Equal(t, len(destination), int(member_count), "maximum member count")
	count, _, decode_status := gzip.Decode_Into(
		destination, gzip.Header_Storage_Unvalidated{}, compressed,
	)
	testify.Equal_Values(t, gzip.STATUS_OK, decode_status, "maximum output")
	testify.Equal(t, len(destination), int(count), "maximum output count")
}

func test_maximum_output_compressed(decoded []byte) (compressed []byte) {
	compressed = make([]byte, 1<<20)
	compressed[0] = 0x1f
	compressed[1] = 0x8b
	compressed[2] = 8
	compressed[9] = gzip.OPERATING_SYSTEM_UNKNOWN
	writer := test_bit_writer{Destination: compressed[10 : len(compressed)-8]}
	test_bit_writer_write(&writer, 3, 3)
	test_fixed_symbol(&writer, 0)
	byte_count := len(decoded) - 1
	for byte_count >= 258 {
		test_fixed_symbol(&writer, 285)
		test_bit_writer_write(&writer, 0, 5)
		byte_count -= 258
	}
	for byte_count > 0 {
		test_fixed_symbol(&writer, 0)
		byte_count--
	}
	test_fixed_symbol(&writer, 256)
	test_bit_writer_finish(&writer)
	trailer_position := 10 + writer.Position
	checksum := crc32.ChecksumIEEE(decoded)
	test_put_word_32(compressed[trailer_position:], checksum)
	test_put_word_32(compressed[trailer_position+4:], uint32(len(decoded)))
	return compressed[:trailer_position+8]
}

type test_bit_writer struct {
	Destination []byte
	Bits        uint64
	Bit_Count   uint8
	Position    int
}

func test_bit_writer_write(writer *test_bit_writer, value uint32, count uint8) {
	writer.Bits |= uint64(value) << writer.Bit_Count
	writer.Bit_Count += count
	for writer.Bit_Count >= 8 {
		writer.Destination[writer.Position] = byte(writer.Bits)
		writer.Position++
		writer.Bits >>= 8
		writer.Bit_Count -= 8
	}
}

func test_bit_writer_finish(writer *test_bit_writer) {
	if writer.Bit_Count == 0 {
		return
	}
	writer.Destination[writer.Position] = byte(writer.Bits)
	writer.Position++
	writer.Bits = 0
	writer.Bit_Count = 0
}

func test_fixed_symbol(writer *test_bit_writer, symbol uint16) {
	var code uint32
	var count uint8
	switch {
	case symbol <= 143:
		code = 0x30 + uint32(symbol)
		count = 8
	case symbol <= 255:
		code = 0x190 + uint32(symbol-144)
		count = 9
	case symbol <= 279:
		code = uint32(symbol - 256)
		count = 7
	default:
		code = 0xc0 + uint32(symbol-280)
		count = 8
	}
	test_bit_writer_write(writer, test_reverse_bits(code, count), count)
}

func test_reverse_bits(value uint32, count uint8) (reversed uint32) {
	for ; count > 0; count-- {
		reversed = reversed<<1 | value&1
		value >>= 1
	}
	return reversed
}

func test_put_word_32(destination []byte, value uint32) {
	destination[0] = byte(value)
	destination[1] = byte(value >> 8)
	destination[2] = byte(value >> 16)
	destination[3] = byte(value >> 24)
}

func test_maximum_encoded_domain(t *testing.T) {
	t.Helper()
	destination := make([]byte, gzip.BYTE_COUNT_MAXIMUM)
	source_size := gzip.BYTE_COUNT_MAXIMUM - gzip.HEADER_SIZE -
		gzip.TRAILER_SIZE - 5*TEST_STORED_BLOCK_COUNT
	source := make([]byte, gzip.BYTE_COUNT_MAXIMUM)
	var workspace_storage test_workspace
	workspace := test_workspace_value(&workspace_storage)
	cases := [...]struct {
		Destination_Size int
		Source_Size      int
	}{
		{len(destination) - 1, source_size - 1},
		{len(destination), source_size},
	}
	for _, test_case := range cases {
		count, status := gzip.Encode_Into(
			destination[:test_case.Destination_Size], workspace,
			source[:test_case.Source_Size], gzip.Header_Unvalidated{},
			gzip.NO_COMPRESSION,
		)
		testify.Equal_Values(
			t, gzip.STATUS_OK, status, "maximum encoded size %d",
			test_case.Destination_Size,
		)
		testify.Equal(
			t, test_case.Destination_Size, int(count),
			"maximum encoded size %d count", test_case.Destination_Size,
		)
	}
	count, status := gzip.Encode_Into(
		destination, workspace, source, gzip.Header_Unvalidated{},
		gzip.NO_COMPRESSION,
	)
	testify.Equal_Values(t, gzip.STATUS_OUTPUT_TOO_SMALL, status, "maximum source")
	testify.Equal(
		t, len(destination)-gzip.TRAILER_SIZE, int(count), "maximum source count",
	)
}

func test_maximum_compressed_domain(t *testing.T) {
	t.Helper()
	member := test_hello_compressed()
	compressed := make([]byte, gzip.BYTE_COUNT_MAXIMUM)
	copy(compressed, member[:])
	var destination [TEST_DESTINATION_SIZE]byte
	var name [gzip.HEADER_TEXT_SIZE_MAXIMUM]byte
	storage := gzip.Header_Storage_Unvalidated{Name: name[:]}
	count, consumed, _, status := gzip.Decode_Member_Into(
		destination[:], storage, compressed,
	)
	testify.Equal_Values(t, gzip.STATUS_OK, status, "maximum compressed member")
	testify.Equal(
		t, len(TEST_HELLO_CONTENT), int(count), "maximum compressed member count",
	)
	testify.Equal(t, len(member), int(consumed), "maximum compressed consumed")
	compressed[0] = 0
	for offset := 1; offset >= 0; offset-- {
		_, rejected_count, _, rejected_status := gzip.Decode_Member_Into(
			destination[:], storage, compressed[:len(compressed)-offset],
		)
		testify.Equal_Values(
			t, gzip.STATUS_HEADER_INVALID, rejected_status,
			"maximum rejected offset %d", offset,
		)
		testify.Equal(
			t, len(compressed)-offset, int(rejected_count),
			"maximum rejected offset %d count", offset,
		)
	}
	var workspace_storage test_workspace
	var empty_member [TEST_DOMAIN_DESTINATION_SIZE]byte
	empty_count, encode_status := gzip.Encode_Into(
		empty_member[:], test_workspace_value(&workspace_storage), nil,
		gzip.Header_Unvalidated{}, gzip.DEFAULT_COMPRESSION,
	)
	testify.Equal_Values(t, gzip.STATUS_OK, encode_status, "empty member encode")
	copy(compressed, empty_member[:empty_count])
	_, _, _, status = gzip.Decode_Member_Into(
		nil, gzip.Header_Storage_Unvalidated{}, compressed,
	)
	testify.Equal_Values(t, gzip.STATUS_OK, status, "maximum trailer source")
	_, _, _, status = gzip.Decode_Member_Into(
		nil, gzip.Header_Storage_Unvalidated{}, nil,
	)
	testify.Equal_Values(t, gzip.STATUS_HEADER_INVALID, status, "empty member")
}

func test_level_domains(t *testing.T) {
	t.Helper()
	var workspace_storage test_workspace
	workspace := test_workspace_value(&workspace_storage)
	var destination [TEST_DOMAIN_DESTINATION_SIZE]byte
	for _, level := range []gzip.Level_Unvalidated{-1 << 7, 1<<7 - 1} {
		_, status := gzip.Encode_Into(
			destination[:], workspace, nil, gzip.Header_Unvalidated{}, level,
		)
		testify.Equal_Values(
			t, gzip.STATUS_LEVEL_INVALID, status, "domain level %d", level,
		)
	}
	_, status := gzip.Encode_Into(
		destination[:], workspace, nil, gzip.Header_Unvalidated{}, 2,
	)
	testify.Equal_Values(t, gzip.STATUS_OK, status, "domain level 2")
}

func test_metadata_domains(t *testing.T) {
	t.Helper()
	cases := [...]struct {
		Extra_Size       int
		Text_Size        int
		Modified_Seconds uint32
		Operating_System uint8
		Source_Size      int
	}{
		{1, 1, 1, 1, 1},
		{2, 2, 2, 2, 2},
		{gzip.EXTRA_SIZE_MAXIMUM, gzip.HEADER_TEXT_SIZE_MAXIMUM,
			1<<32 - 1, 1<<8 - 1, 0},
	}
	for _, test_case := range cases {
		test_metadata_domain(t, test_case.Extra_Size, test_case.Text_Size,
			test_case.Modified_Seconds,
			test_case.Operating_System, test_case.Source_Size)
	}
}

func test_metadata_domain(
	t *testing.T,
	extra_size int,
	text_size int,
	modified_seconds uint32,
	operating_system uint8,
	source_size int,
) {
	t.Helper()
	extra := make([]byte, extra_size)
	name := test_domain_text(text_size)
	comment := test_domain_text(text_size)
	source := make([]byte, source_size)
	header := gzip.Header_Unvalidated{
		Extra: extra, Name: name, Comment: comment,
		Modified_Seconds: gzip.Modified_Seconds(modified_seconds),
		Operating_System: gzip.Operating_System(operating_system),
	}
	var compressed [gzip.HEADER_SIZE_MAXIMUM + 64]byte
	var workspace_storage test_workspace
	compressed_count, status := gzip.Encode_Into(
		compressed[:], test_workspace_value(&workspace_storage), source, header,
		gzip.DEFAULT_COMPRESSION,
	)
	testify.Equal_Values(t, gzip.STATUS_OK, status, "domain encode size %d", text_size)
	test_metadata_domain_decode(
		t, compressed[:compressed_count], source_size, extra_size, text_size,
	)
}

func test_metadata_domain_decode(
	t *testing.T, compressed []byte, source_size int, extra_size int, text_size int,
) {
	t.Helper()
	destination := make([]byte, source_size)
	extra := make([]byte, extra_size)
	name := make([]byte, text_size)
	comment := make([]byte, text_size)
	storage := gzip.Header_Storage_Unvalidated{
		Extra: extra, Name: name, Comment: comment,
	}
	count, header, status := gzip.Decode_Into(destination, storage, compressed)
	testify.Equal_Values(t, gzip.STATUS_OK, status, "domain decode size %d", text_size)
	testify.Equal(t, source_size, int(count), "domain decode count %d", text_size)
	member_count, consumed, member_header, member_status :=
		gzip.Decode_Member_Into(destination, storage, compressed)
	testify.Equal_Values(
		t, gzip.STATUS_OK, member_status, "domain member size %d", text_size,
	)
	testify.Equal(
		t, source_size, int(member_count), "domain member count %d", text_size,
	)
	testify.Equal(t, len(compressed), int(consumed), "domain member consumed")
	testify.Equal(t, extra_size, len(header.Extra), "domain extra size")
	testify.Equal(t, text_size, len(member_header.Name), "domain header size")
}

func test_domain_text(size int) (text []byte) {
	text = make([]byte, size)
	if size == gzip.HEADER_TEXT_SIZE_MAXIMUM {
		for position := 0; position < size; position += 2 {
			text[position] = 0xc3
			text[position+1] = 0xbf
		}
		return text
	}
	for position := range text {
		text[position] = 'a'
	}
	return text
}

func test_standard_member(t *testing.T) {
	t.Helper()
	compressed := test_hello_compressed()
	var destination [TEST_DESTINATION_SIZE]byte
	var extra [TEST_HEADER_EXTRA_SIZE]byte
	var name [gzip.HEADER_TEXT_SIZE_MAXIMUM]byte
	var comment [gzip.HEADER_TEXT_SIZE_MAXIMUM]byte
	storage := gzip.Header_Storage_Unvalidated{
		Extra: extra[:], Name: name[:], Comment: comment[:],
	}
	count, header, status := gzip.Decode_Into(
		destination[:], storage, compressed[:],
	)
	testify.Equal_Values(t, gzip.STATUS_OK, status, "standard member decode")
	testify.Equal(t, TEST_HELLO_CONTENT, string(destination[:count]), "standard output")
	testify.Equal(t, "hello.txt", string(header.Name), "standard name")
	testify.Equal_Values(t, 0x4a1358c8, header.Modified_Seconds, "standard timestamp")
	testify.Equal_Values(t, 3, header.Operating_System, "standard operating system")
}

func test_member_round_trip(t *testing.T) {
	t.Helper()
	var workspace_storage test_workspace
	workspace := test_workspace_value(&workspace_storage)
	var compressed [TEST_DESTINATION_SIZE]byte
	count, status := gzip.Encode_Into(
		compressed[:], workspace, []byte("payload"),
		gzip.Header_Unvalidated{Operating_System: gzip.OPERATING_SYSTEM_UNKNOWN},
		gzip.DEFAULT_COMPRESSION,
	)
	testify.Equal_Values(t, gzip.STATUS_OK, status, "round trip encode")
	var destination [TEST_DESTINATION_SIZE]byte
	decoded_count, _, decoded_status := gzip.Decode_Into(
		destination[:], gzip.Header_Storage_Unvalidated{}, compressed[:count],
	)
	testify.Equal_Values(t, gzip.STATUS_OK, decoded_status, "round trip decode")
	testify.Equal(t, "payload", string(destination[:decoded_count]), "round trip output")
}

func test_concatenated_members(t *testing.T) {
	t.Helper()
	compressed := test_hello_compressed()
	var sequence [2 * TEST_HELLO_COMPRESSED_SIZE]byte
	copy(sequence[:], compressed[:])
	copy(sequence[len(compressed):], compressed[:])
	var destination [TEST_DESTINATION_SIZE]byte
	var name [gzip.HEADER_TEXT_SIZE_MAXIMUM]byte
	storage := gzip.Header_Storage_Unvalidated{Name: name[:]}
	count, header, status := gzip.Decode_Into(destination[:], storage, sequence[:])
	testify.Equal_Values(t, gzip.STATUS_OK, status, "concatenated decode")
	testify.Equal(
		t, TEST_HELLO_CONTENT+TEST_HELLO_CONTENT, string(destination[:count]),
		"concatenated output",
	)
	testify.Equal(t, "hello.txt", string(header.Name), "concatenated first name")
	member_count, compressed_count, member_header, member_status :=
		gzip.Decode_Member_Into(destination[:], storage, sequence[:])
	testify.Equal_Values(t, gzip.STATUS_OK, member_status, "single member decode")
	testify.Equal(t, len(compressed), int(compressed_count), "single member input count")
	testify.Equal(
		t, TEST_HELLO_CONTENT, string(destination[:member_count]), "single member output",
	)
	testify.Equal(t, "hello.txt", string(member_header.Name), "single member name")
}

func test_empty_member_sequence(t *testing.T) {
	t.Helper()
	count, header, status := gzip.Decode_Into(
		nil, gzip.Header_Storage_Unvalidated{}, nil,
	)
	testify.Zero(t, count, "empty stream count")
	testify.Equal_Values(t, gzip.STATUS_OK, status, "empty stream")
	testify.Empty(t, header.Extra, "empty stream extra")
	testify.Empty(t, header.Name, "empty stream name")
	testify.Empty(t, header.Comment, "empty stream comment")
}

func test_metadata_round_trip(t *testing.T) {
	t.Helper()
	var workspace_storage test_workspace
	header_input := gzip.Header_Unvalidated{
		Extra: []byte("extra"), Name: []byte("Äußerung"),
		Comment: []byte("Látin-1"), Modified_Seconds: 100_000_000,
		Operating_System: 1,
	}
	var compressed [TEST_DESTINATION_SIZE]byte
	compressed_count, status := gzip.Encode_Into(
		compressed[:], test_workspace_value(&workspace_storage),
		[]byte("metadata"), header_input, gzip.BEST_COMPRESSION,
	)
	testify.Equal_Values(t, gzip.STATUS_OK, status, "metadata encode")
	var destination [TEST_DESTINATION_SIZE]byte
	var extra [TEST_HEADER_EXTRA_SIZE]byte
	var name [gzip.HEADER_TEXT_SIZE_MAXIMUM]byte
	var comment [gzip.HEADER_TEXT_SIZE_MAXIMUM]byte
	storage := gzip.Header_Storage_Unvalidated{
		Extra: extra[:], Name: name[:], Comment: comment[:],
	}
	count, header, decode_status := gzip.Decode_Into(
		destination[:], storage, compressed[:compressed_count],
	)
	testify.Equal_Values(t, gzip.STATUS_OK, decode_status, "metadata decode")
	testify.Equal(t, "metadata", string(destination[:count]), "metadata output")
	assert_metadata_header(t, header)
}

func assert_metadata_header(t *testing.T, header gzip.Header) {
	t.Helper()
	testify.Equal(t, "extra", string(header.Extra), "metadata extra")
	testify.Equal(t, "Äußerung", string(header.Name), "metadata name")
	testify.Equal(t, "Látin-1", string(header.Comment), "metadata comment")
	testify.Equal_Values(t, 100_000_000, header.Modified_Seconds, "metadata timestamp")
	testify.Equal_Values(t, 1, header.Operating_System, "metadata operating system")
}

func test_complete_header(t *testing.T) {
	t.Helper()
	compressed := test_complete_header_compressed()
	var destination [TEST_DESTINATION_SIZE]byte
	var extra [TEST_HEADER_EXTRA_SIZE]byte
	var name [gzip.HEADER_TEXT_SIZE_MAXIMUM]byte
	var comment [gzip.HEADER_TEXT_SIZE_MAXIMUM]byte
	storage := gzip.Header_Storage_Unvalidated{
		Extra: extra[:], Name: name[:], Comment: comment[:],
	}
	count, header, status := gzip.Decode_Into(destination[:], storage, compressed)
	testify.Equal_Values(t, gzip.STATUS_OK, status, "complete header decode")
	testify.Zero(t, count, "complete header count")
	testify.Equal(t, "zz\x05\x00abcdef", string(header.Extra), "complete header extra")
	testify.Equal(t, "f1l3n4m3.tXt", string(header.Name), "complete header name")
	testify.Equal(
		t, TEST_COMPLETE_COMMENT_UTF8_SIZE, len(header.Comment),
		"complete header comment size",
	)
	testify.Equal_Values(t, 0x4af9f070, header.Modified_Seconds, "complete timestamp")
	testify.Equal_Values(t, 0xaa, header.Operating_System, "complete operating system")
	_, _, status = gzip.Decode_Into(
		destination[:], gzip.Header_Storage_Unvalidated{}, compressed,
	)
	testify.Equal_Values(
		t, gzip.STATUS_HEADER_STORAGE_TOO_SMALL, status, "short header storage",
	)
}

func test_header_rejections(t *testing.T) {
	t.Helper()
	var workspace_storage test_workspace
	workspace := test_workspace_value(&workspace_storage)
	var compressed [TEST_DESTINATION_SIZE]byte
	invalid_names := [][]byte{{0}, {0xc4, 0x80}, {0xff}}
	for _, invalid_name := range invalid_names {
		_, status := gzip.Encode_Into(
			compressed[:], workspace, nil,
			gzip.Header_Unvalidated{Name: invalid_name}, gzip.DEFAULT_COMPRESSION,
		)
		testify.Equal_Values(
			t, gzip.STATUS_INPUT_INVALID, status, "invalid name %x", invalid_name,
		)
	}
	var long_name [gzip.HEADER_STRING_BYTE_COUNT_MAXIMUM + 1]byte
	for index := range long_name {
		long_name[index] = 'a'
	}
	_, status := gzip.Encode_Into(
		compressed[:], workspace, nil,
		gzip.Header_Unvalidated{Name: long_name[:]}, gzip.DEFAULT_COMPRESSION,
	)
	testify.Equal_Values(t, gzip.STATUS_HEADER_INVALID, status, "long name")
	oversized_extra := make([]byte, gzip.EXTRA_SIZE_MAXIMUM+1)
	_, status = gzip.Encode_Into(
		compressed[:], workspace, nil,
		gzip.Header_Unvalidated{Extra: oversized_extra}, gzip.DEFAULT_COMPRESSION,
	)
	testify.Equal_Values(t, gzip.STATUS_HEADER_INVALID, status, "oversized extra")
}

func test_encode_allocation(t *testing.T) {
	t.Helper()
	var destination [TEST_DESTINATION_SIZE]byte
	var short [TEST_SHORT_DESTINATION_SIZE]byte
	var workspace_storage test_workspace
	workspace := test_workspace_value(&workspace_storage)
	source := []byte(TEST_HELLO_CONTENT)
	invalid_name := []byte{0}
	var long_name [gzip.HEADER_STRING_BYTE_COUNT_MAXIMUM + 1]byte
	for index := range long_name {
		long_name[index] = 'a'
	}
	var observed_count gzip.Encoded_Count
	var observed_status gzip.Encode_Status
	cases := [...]struct {
		Name        string
		Destination gzip.Destination_Unvalidated
		Workspace   gzip.Workspace_Unvalidated
		Source      gzip.Source_Unvalidated
		Header      gzip.Header_Unvalidated
		Level       gzip.Level_Unvalidated
	}{
		{"Success", destination[:], workspace, source, gzip.Header_Unvalidated{},
			gzip.DEFAULT_COMPRESSION},
		{"Output", short[:], workspace, source, gzip.Header_Unvalidated{},
			gzip.DEFAULT_COMPRESSION},
		{"Input", destination[:], workspace, nil,
			gzip.Header_Unvalidated{Name: invalid_name}, gzip.DEFAULT_COMPRESSION},
		{"Header", destination[:], workspace, nil,
			gzip.Header_Unvalidated{Name: long_name[:]}, gzip.DEFAULT_COMPRESSION},
		{"Level", destination[:], workspace, nil, gzip.Header_Unvalidated{}, 10},
		{"Workspace", destination[:], gzip.Workspace_Unvalidated{}, nil,
			gzip.Header_Unvalidated{}, gzip.DEFAULT_COMPRESSION},
		{"Overlap", destination[:], workspace, destination[:],
			gzip.Header_Unvalidated{}, gzip.DEFAULT_COMPRESSION},
	}
	for _, test_case := range cases {
		testify.Zero_Allocation(t, func() {
			observed_count, observed_status = gzip.Encode_Into(
				test_case.Destination, test_case.Workspace, test_case.Source,
				test_case.Header, test_case.Level,
			)
		}, test_case.Name+" Encode_Into")
	}
	testify.Greater_Or_Equal(
		t, &testify.Greater_Or_Equal_Input[gzip.Encoded_Count]{
			First: observed_count, Second: 0,
		}, "encode allocation count",
	)
	testify.Less_Or_Equal(
		t, &testify.Less_Or_Equal_Input[gzip.Encode_Status]{
			First: observed_status, Second: gzip.STATUS_STORAGE_INVALID,
		}, "encode allocation status",
	)
}

func test_decode_allocation(t *testing.T) {
	t.Helper()
	compressed := test_hello_compressed()
	bad_checksum := compressed
	bad_checksum[len(bad_checksum)-8] ^= 1
	bad_header := compressed
	bad_header[0] = 0
	minimal := [TEST_FIXED_HEADER_SIZE]byte{0x1f, 0x8b, 0x08}
	var aliased [TEST_DESTINATION_SIZE]byte
	copy(aliased[:], compressed[:])
	var destination [TEST_DESTINATION_SIZE]byte
	var short [TEST_SHORT_DESTINATION_SIZE]byte
	var name [gzip.HEADER_TEXT_SIZE_MAXIMUM]byte
	storage := gzip.Header_Storage_Unvalidated{Name: name[:]}
	var observed_count gzip.Count
	var observed_compressed_count gzip.Compressed_Count
	var observed_header gzip.Header
	var observed_status gzip.Decode_Status
	cases := [...]struct {
		Name        string
		Destination gzip.Destination_Unvalidated
		Storage     gzip.Header_Storage_Unvalidated
		Compressed  gzip.Compressed_Unvalidated
	}{
		{"Success", destination[:], storage, compressed[:]},
		{"Output", short[:], storage, compressed[:]},
		{"Header storage", destination[:], gzip.Header_Storage_Unvalidated{},
			compressed[:]},
		{"Checksum", destination[:], storage, bad_checksum[:]},
		{"Header", destination[:], storage, bad_header[:]},
		{"Input", destination[:], storage, minimal[:]},
		{"Storage", aliased[:], storage, aliased[:len(compressed)]},
	}
	for _, test_case := range cases {
		testify.Zero_Allocation(t, func() {
			observed_count, observed_header, observed_status = gzip.Decode_Into(
				test_case.Destination, test_case.Storage, test_case.Compressed,
			)
		}, test_case.Name+" Decode_Into")
		testify.Zero_Allocation(t, func() {
			observed_count, observed_compressed_count, observed_header,
				observed_status = gzip.Decode_Member_Into(
				test_case.Destination, test_case.Storage, test_case.Compressed,
			)
		}, test_case.Name+" Decode_Member_Into")
	}
	assert_decode_allocation(
		t, observed_count, observed_compressed_count, observed_header, observed_status,
	)
}

func assert_decode_allocation(
	t *testing.T,
	count gzip.Count,
	compressed_count gzip.Compressed_Count,
	header gzip.Header,
	status gzip.Decode_Status,
) {
	t.Helper()
	testify.Greater_Or_Equal(
		t, &testify.Greater_Or_Equal_Input[gzip.Count]{First: count, Second: 0},
		"decode allocation count",
	)
	testify.Greater_Or_Equal(
		t, &testify.Greater_Or_Equal_Input[gzip.Compressed_Count]{
			First: compressed_count, Second: 0,
		}, "member allocation input count",
	)
	testify.Less_Or_Equal(
		t, &testify.Less_Or_Equal_Input[int]{
			First: len(header.Name), Second: gzip.HEADER_TEXT_SIZE_MAXIMUM,
		}, "decode allocation header",
	)
	testify.Less_Or_Equal(
		t, &testify.Less_Or_Equal_Input[gzip.Decode_Status]{
			First: status, Second: gzip.STATUS_STORAGE_INVALID,
		}, "decode allocation status",
	)
}

func test_truncated_input(t *testing.T) {
	t.Helper()
	compressed := test_hello_compressed()
	var destination [TEST_DESTINATION_SIZE]byte
	var name [gzip.HEADER_TEXT_SIZE_MAXIMUM]byte
	storage := gzip.Header_Storage_Unvalidated{Name: name[:]}
	for index := 1; index < len(compressed); index++ {
		_, _, status := gzip.Decode_Into(
			destination[:], storage, compressed[:index],
		)
		testify.Not_Equal_Values(
			t, gzip.STATUS_OK, status, "truncated size %d", index,
		)
	}
}

func test_corrupt_input(t *testing.T) {
	t.Helper()
	compressed := test_hello_compressed()
	var destination [TEST_DESTINATION_SIZE]byte
	var name [gzip.HEADER_TEXT_SIZE_MAXIMUM]byte
	storage := gzip.Header_Storage_Unvalidated{Name: name[:]}
	bad_magic := compressed
	bad_magic[0] = 0
	_, _, status := gzip.Decode_Into(destination[:], storage, bad_magic[:])
	testify.Equal_Values(t, gzip.STATUS_HEADER_INVALID, status, "bad magic")
	bad_method := compressed
	bad_method[2] = 7
	_, _, status = gzip.Decode_Into(destination[:], storage, bad_method[:])
	testify.Equal_Values(t, gzip.STATUS_HEADER_INVALID, status, "bad method")
	bad_checksum := compressed
	bad_checksum[len(bad_checksum)-8] ^= 1
	_, _, status = gzip.Decode_Into(destination[:], storage, bad_checksum[:])
	testify.Equal_Values(t, gzip.STATUS_CHECKSUM_INVALID, status, "bad checksum")
	bad_size := compressed
	bad_size[len(bad_size)-4] ^= 1
	_, _, status = gzip.Decode_Into(destination[:], storage, bad_size[:])
	testify.Equal_Values(t, gzip.STATUS_CHECKSUM_INVALID, status, "bad size")
	header := test_complete_header_compressed()
	header[TEST_HEADER_CRC_POSITION] ^= 1
	var extra [TEST_HEADER_EXTRA_SIZE]byte
	var comment [TEST_COMPLETE_COMMENT_UTF8_SIZE]byte
	complete_storage := gzip.Header_Storage_Unvalidated{
		Extra: extra[:], Name: name[:], Comment: comment[:],
	}
	_, _, status = gzip.Decode_Into(destination[:], complete_storage, header)
	testify.Equal_Values(t, gzip.STATUS_HEADER_INVALID, status, "bad header checksum")
}

func test_suffix_and_overlap(t *testing.T) {
	t.Helper()
	compressed := test_hello_compressed()
	var destination [TEST_DESTINATION_SIZE]byte
	var name [gzip.HEADER_TEXT_SIZE_MAXIMUM]byte
	header_storage := gzip.Header_Storage_Unvalidated{Name: name[:]}
	var suffix_input [TEST_HELLO_COMPRESSED_SIZE + 1]byte
	copy(suffix_input[:], compressed[:])
	suffix_input[len(suffix_input)-1] = 1
	_, _, status := gzip.Decode_Into(
		destination[:], header_storage, suffix_input[:],
	)
	testify.Equal_Values(t, gzip.STATUS_HEADER_INVALID, status, "suffix stream")
	_, consumed, _, member_status := gzip.Decode_Member_Into(
		destination[:], header_storage, suffix_input[:],
	)
	testify.Equal_Values(t, gzip.STATUS_OK, member_status, "suffix member")
	testify.Equal(t, len(compressed), int(consumed), "suffix member count")
	storage := [TEST_DESTINATION_SIZE]byte{}
	copy(storage[:], compressed[:])
	_, _, status = gzip.Decode_Into(
		storage[:], header_storage, storage[:len(compressed)],
	)
	testify.Equal_Values(t, gzip.STATUS_STORAGE_INVALID, status, "overlap")
}

func test_input_mutations(t *testing.T) {
	t.Helper()
	compressed := test_hello_compressed()
	var destination [TEST_DESTINATION_SIZE]byte
	var name [gzip.HEADER_TEXT_SIZE_MAXIMUM]byte
	storage := gzip.Header_Storage_Unvalidated{Name: name[:]}
	for index := range compressed {
		candidate := compressed
		for value_index := 0; value_index < 256; value_index++ {
			candidate[index] = byte(value_index)
			count, _, status := gzip.Decode_Into(
				destination[:], storage, candidate[:],
			)
			testify.Greater_Or_Equal(
				t, &testify.Greater_Or_Equal_Input[int]{
					First: int(count), Second: 0,
				}, "mutation count",
			)
			testify.Less_Or_Equal(
				t, &testify.Less_Or_Equal_Input[int]{
					First: int(count), Second: len(destination),
				}, "mutation count",
			)
			testify.Less_Or_Equal(
				t, &testify.Less_Or_Equal_Input[gzip.Decode_Status]{
					First: status, Second: gzip.STATUS_STORAGE_INVALID,
				}, "mutation status",
			)
		}
	}
}

const TEST_HELLO_CONTENT = "hello world\n"
const TEST_HELLO_COMPRESSED_SIZE = 42
const TEST_DESTINATION_SIZE = 512
const TEST_DOMAIN_DESTINATION_SIZE = 64
const TEST_SHORT_DESTINATION_SIZE = 5
const TEST_HEADER_EXTRA_SIZE = 64
const TEST_COMPLETE_COMMENT_UTF8_SIZE = 383
const TEST_HEADER_CRC_POSITION = 291
const TEST_FIXED_HEADER_SIZE = 10
const TEST_STORED_BLOCK_COUNT = 1024
const TEST_FLAG_HEADER_CHECKSUM = 1 << 1

type test_workspace struct {
	Heads    [gzip.HASH_POSITIONS_COUNT]int32
	Previous [gzip.HISTORY_POSITIONS_COUNT]int32
}

func test_workspace_value(
	storage *test_workspace,
) (workspace gzip.Workspace_Unvalidated) {
	return gzip.Workspace_Unvalidated{
		Heads: storage.Heads[:], Previous: storage.Previous[:],
	}
}

func test_hello_compressed() (compressed [TEST_HELLO_COMPRESSED_SIZE]byte) {
	return [TEST_HELLO_COMPRESSED_SIZE]byte{
		0x1f, 0x8b, 0x08, 0x08, 0xc8, 0x58, 0x13, 0x4a,
		0x00, 0x03, 0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x2e,
		0x74, 0x78, 0x74, 0x00, 0xcb, 0x48, 0xcd, 0xc9,
		0xc9, 0x57, 0x28, 0xcf, 0x2f, 0xca, 0x49, 0xe1,
		0x02, 0x00, 0x2d, 0x3b, 0x08, 0xaf, 0x0c, 0x00,
		0x00, 0x00,
	}
}

func test_complete_header_compressed() (compressed []byte) {
	const FLAG_EXTRA_NAME_COMMENT_HEADER_CRC = 0x1e
	compressed = make([]byte, 0, 320)
	compressed = append(
		compressed,
		0x1f, 0x8b, 0x08, FLAG_EXTRA_NAME_COMMENT_HEADER_CRC,
		0x70, 0xf0, 0xf9, 0x4a, 0x00, 0xaa,
	)
	compressed = append(compressed, 10, 0)
	compressed = append(compressed, []byte("zz\x05\x00abcdef")...)
	compressed = append(compressed, []byte("f1l3n4m3.tXt")...)
	compressed = append(compressed, 0)
	for value := 1; value <= 255; value++ {
		compressed = append(compressed, byte(value))
	}
	compressed = append(compressed, 0)
	header_checksum := crc32.ChecksumIEEE(compressed)
	compressed = append(
		compressed, byte(header_checksum), byte(header_checksum>>8),
	)
	compressed = append(compressed, 3, 0)
	compressed = append(compressed, 0, 0, 0, 0, 0, 0, 0, 0)
	return compressed
}
