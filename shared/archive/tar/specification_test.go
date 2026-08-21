package tar_test

import (
	"testing"
	"unsafe"

	"local/james-orcales/shared/archive/tar"
	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/nbio"
	"local/james-orcales/shared/sim/time"
	"local/james-orcales/shared/strconv"
)

// Test_Constants proves every constant formula.
func Test_Constants(t *testing.T) {
	test_constants(t)
}

// Test_Stream proves transport injection and retirement timing.
func Test_Stream(t *testing.T) {
	test_stream_boundary(t)
	test_deferred_stream(t)
}

// Test_Storage proves caller storage ownership and boundaries.
func Test_Storage(t *testing.T) {
	test_writer_state(t)
}

// Test_Formats proves supported wire families.
func Test_Formats(t *testing.T) {
	test_extended_formats(t)
	test_reader_star(t)
	test_reader_gnu_prefix(t)
	test_writer_extended_formats(t)
}

// Test_Bounds proves content and transport counts stay bounded.
func Test_Bounds(t *testing.T) {
	test_reader_next_discards(t)
	test_defective_stream_count(t)
	test_large_pax_payload(t)
	test_largest_pax_record(t)
}

// Test_Reader proves logical header and content decoding.
func Test_Reader(t *testing.T) {
	test_reader_standard_header(t)
}

// Test_Writer proves header, content, padding, and footer encoding.
func Test_Writer(t *testing.T) {
	test_writer_round_trip(t)
}

// Test_Allocation proves exported operations allocate no heap storage.
func Test_Allocation(t *testing.T) {
	test_allocation(t)
}

// Test_Untrusted_Input proves malformed bytes become Status.
func Test_Untrusted_Input(t *testing.T) {
	test_malformed_input(t)
	test_writer_header_validation(t)
}

// Test_Invariant_Domains reaches every public scalar boundary through production paths.
func Test_Invariant_Domains(t *testing.T) {
	test_invariant_domains(t)
}

const TEST_ARCHIVE_BLOCK_COUNT = 1 << 4
const TEST_ARCHIVE_SIZE = TEST_ARCHIVE_BLOCK_COUNT * tar.BLOCK_SIZE
const TEST_FIELD_SIZE = tar.BLOCK_SIZE
const TEST_LONG_NAME_SIZE = tar.NAME_FIELD_SIZE + tar.USER_NAME_FIELD_SIZE +
	tar.MODE_FIELD_SIZE
const TEST_FILE_MODE = 0o600 | 0o040
const TEST_USER_IDENTIFIER = tar.NAME_FIELD_SIZE
const TEST_GROUP_IDENTIFIER = 2 * TEST_USER_IDENTIFIER
const TEST_MODIFICATION_SECONDS = int64(time.SECOND / time.NANOSECOND)
const TEST_WRITER_WORKSPACE_BLOCK_COUNT = 1 << tar.OCTAL_BITS_PER_DIGIT
const TEST_PARTIAL_READ_SIZE = len("first")
const TEST_ALPHABET_SIZE = 'z' - 'a' + 1
const TEST_LARGE_PAX_TEXT_SIZE = bytes.SLICE_SIZE_MAXIMUM + tar.BLOCK_SIZE
const TEST_TIMESTAMP_TEXT_SIZE_MAXIMUM = strconv.INTEGER_TEXT_SIZE_MAXIMUM +
	tar.TIMESTAMP_FRACTION_SEPARATOR_SIZE + tar.TIMESTAMP_FRACTION_DIGIT_COUNT_MAXIMUM
const TEST_PADDING_MINIMUM = tar.PADDING_SIZE_MAXIMUM - tar.PADDING_SIZE_MAXIMUM

const TEST_DOMAIN_MINIMUM = 0
const TEST_DOMAIN_NEGATIVE_ONE = TEST_DOMAIN_MINIMUM + 1
const TEST_DOMAIN_ZERO = TEST_DOMAIN_NEGATIVE_ONE + 1
const TEST_DOMAIN_ONE = TEST_DOMAIN_ZERO + 1
const TEST_DOMAIN_TWO = TEST_DOMAIN_ONE + 1
const TEST_DOMAIN_MAXIMUM = TEST_DOMAIN_TWO + 1
const TEST_DOMAIN_COUNT = TEST_DOMAIN_MAXIMUM + 1

const TEST_HEADER_FORMAT = 0
const TEST_HEADER_TYPE_FLAG = TEST_HEADER_FORMAT + 1
const TEST_HEADER_NAME = TEST_HEADER_TYPE_FLAG + 1
const TEST_HEADER_LINK_NAME = TEST_HEADER_NAME + 1
const TEST_HEADER_SIZE = TEST_HEADER_LINK_NAME + 1
const TEST_HEADER_MODE = TEST_HEADER_SIZE + 1
const TEST_HEADER_USER_IDENTIFIER = TEST_HEADER_MODE + 1
const TEST_HEADER_GROUP_IDENTIFIER = TEST_HEADER_USER_IDENTIFIER + 1
const TEST_HEADER_USER_NAME = TEST_HEADER_GROUP_IDENTIFIER + 1
const TEST_HEADER_GROUP_NAME = TEST_HEADER_USER_NAME + 1
const TEST_HEADER_MODIFICATION_TIME = TEST_HEADER_GROUP_NAME + 1
const TEST_HEADER_ACCESS_TIME = TEST_HEADER_MODIFICATION_TIME + 1
const TEST_HEADER_CHANGE_TIME = TEST_HEADER_ACCESS_TIME + 1
const TEST_HEADER_DEVICE_MAJOR = TEST_HEADER_CHANGE_TIME + 1
const TEST_HEADER_DEVICE_MINOR = TEST_HEADER_DEVICE_MAJOR + 1
const TEST_HEADER_PAX_RECORDS = TEST_HEADER_DEVICE_MINOR + 1
const TEST_HEADER_FIELD_COUNT = TEST_HEADER_PAX_RECORDS + 1

// TEST_ALLOCATION_RUN_COUNT needs no averaging because any allocation fails.
const TEST_ALLOCATION_RUN_COUNT = 1

// TEST_ALLOCATION_CONTENT_SIZE keeps Reader and Writer proof on same complete entry.
const TEST_ALLOCATION_CONTENT_SIZE = 1

func test_invariant_domains(t *testing.T) {
	t.Helper()
	test_stream_boundary(t)
	test_deferred_stream(t)
	test_writer_state(t)
	test_extended_formats(t)
	test_reader_star(t)
	test_reader_gnu_prefix(t)
	test_writer_extended_formats(t)
	test_reader_next_discards(t)
	test_defective_stream_count(t)
	test_large_pax_payload(t)
	test_largest_pax_record(t)
	test_reader_standard_header(t)
	test_writer_round_trip(t)
	test_malformed_input(t)
	test_writer_header_validation(t)
	for marker := TEST_DOMAIN_MINIMUM; marker < TEST_DOMAIN_COUNT; marker++ {
		for field := TEST_HEADER_FORMAT; field < TEST_HEADER_FIELD_COUNT; field++ {
			test_header_domain(field, marker)
			test_writer_header_field_domain(field, marker, tar.FORMAT_UNKNOWN)
			test_writer_header_field_domain(field, marker, tar.FORMAT_V7)
			test_writer_header_field_domain(field, marker, tar.FORMAT_USTAR)
			test_writer_header_field_domain(field, marker, tar.FORMAT_GNU)
			test_writer_header_field_domain(field, marker, tar.FORMAT_PAX)
			test_writer_header_field_domain(field, marker, tar.FORMAT_STAR)
		}
		test_reader_domain(marker)
		test_reader_wire_domain(marker)
		if status := test_reader_storage_domain(marker); status != tar.STATUS_OK {
			t.Fatalf("reader storage domain %d: %d", marker, status)
		}
		test_reader_fixed_text_domain(marker)
		test_writer_domain(marker)
		test_writer_basic_domain(t, marker)
		test_writer_gnu_domain(t, marker)
		test_writer_pax_domain(t, marker)
	}
	test_reader_long_text_domain(t)
	test_reader_pax_text_domain(
		t, []byte(tar.PAX_USER_NAME_KEY), tar.HEADER_USER_NAME_SIZE_MAXIMUM,
	)
	test_reader_pax_text_domain(
		t, []byte(tar.PAX_GROUP_NAME_KEY), tar.HEADER_GROUP_NAME_SIZE_MAXIMUM,
	)
	test_reader_pax_text_boundaries(t)
	test_writer_position_boundaries(t)
	test_writer_gnu_long_boundaries(t)
	test_writer_pax_payload_boundaries(t)
	test_writer_pax_record_boundaries(t)
	test_writer_pax_candidate_maximum(t)
	test_writer_utf8_boundaries(t)
	test_reader_metadata_boundaries(t)
	test_reader_pax_record_domains(t)
	test_reader_fixed_header_failures(t)
	test_writer_remaining_domains(t)
	test_storage_overlap_domains(t)
	test_active_preconditions(t)
	test_checksum_domains()
}

func test_writer_utf8_boundaries(t *testing.T) {
	t.Helper()
	for _, name := range [][]byte{
		[]byte("¢"), []byte("a€"), []byte("ab😀"),
	} {
		var archive [TEST_ARCHIVE_SIZE]byte
		workspace := make([]byte, tar.WRITER_STORAGE_SIZE_MAXIMUM)
		memory := nbio.Stream_Memory{Memory: archive[:]}
		var writer tar.Writer
		status := tar.Writer_Init(
			&writer, nbio.Memory_To_Stream(&memory), workspace,
		)
		if status != tar.STATUS_OK {
			t.Fatalf("UTF-8 Writer_Init status = %v", status)
		}
		header := tar.Header_Unvalidated{
			Format:    tar.Format_Unvalidated(tar.FORMAT_PAX),
			Type_Flag: tar.TYPE_REGULAR, Name: name,
		}
		tar.Writer_Write_Header(
			&writer, &writer.Completion, &header, test_domain_callback,
		)
		if writer.Status != tar.STATUS_OK {
			t.Fatalf("UTF-8 name %q status = %v", name, writer.Status)
		}
	}
	test_writer_utf8_maximum(t)
}

func test_writer_utf8_maximum(t *testing.T) {
	t.Helper()
	maximum := make([]byte, tar.HEADER_TEXT_SIZE_MAXIMUM)
	test_fill_bytes(maximum, 'a')
	copy(maximum[len(maximum)-len("¢"):], []byte("¢"))
	archive := make([]byte, tar.ARCHIVE_SIZE_MAXIMUM)
	workspace := make([]byte, tar.WRITER_STORAGE_SIZE_MAXIMUM)
	memory := nbio.Stream_Memory{Memory: archive}
	var writer tar.Writer
	if status := tar.Writer_Init(
		&writer, nbio.Memory_To_Stream(&memory), workspace,
	); status != tar.STATUS_OK {
		t.Fatalf("maximum UTF-8 Writer_Init status = %v", status)
	}
	header := tar.Header_Unvalidated{
		Format:    tar.Format_Unvalidated(tar.FORMAT_PAX),
		Type_Flag: tar.TYPE_REGULAR, Name: maximum,
	}
	tar.Writer_Write_Header(
		&writer, &writer.Completion, &header, test_domain_callback,
	)
	if writer.Status != tar.STATUS_FIELD_TOO_LONG {
		t.Fatalf("maximum UTF-8 status = %v", writer.Status)
	}
	maximum_invalid := make([]byte, tar.HEADER_TEXT_SIZE_MAXIMUM)
	test_fill_bytes(maximum_invalid, 'a')
	maximum_invalid[len(maximum_invalid)-tar.TYPE_FLAG_FIELD_SIZE] =
		[]byte("¢")[TEST_PADDING_MINIMUM]
	header.Name = maximum_invalid
	memory = nbio.Stream_Memory{Memory: archive}
	if status := tar.Writer_Init(
		&writer, nbio.Memory_To_Stream(&memory), workspace,
	); status != tar.STATUS_OK {
		t.Fatalf("maximum truncated UTF-8 Writer_Init status = %v", status)
	}
	tar.Writer_Write_Header(
		&writer, &writer.Completion, &header, test_domain_callback,
	)
	if writer.Status != tar.STATUS_FIELD_TOO_LONG {
		t.Fatalf("maximum truncated UTF-8 status = %v", writer.Status)
	}
	invalid := []byte("¢")[:len("¢")-tar.TYPE_FLAG_FIELD_SIZE]
	header.Name = invalid
	memory = nbio.Stream_Memory{Memory: archive}
	if status := tar.Writer_Init(
		&writer, nbio.Memory_To_Stream(&memory), workspace,
	); status != tar.STATUS_OK {
		t.Fatalf("truncated UTF-8 Writer_Init status = %v", status)
	}
	tar.Writer_Write_Header(
		&writer, &writer.Completion, &header, test_domain_callback,
	)
	if writer.Status != tar.STATUS_FIELD_TOO_LONG {
		t.Fatalf("truncated UTF-8 status = %v", writer.Status)
	}
}

func test_reader_metadata_boundaries(t *testing.T) {
	t.Helper()
	for _, boundary_size := range [...]int{
		TEST_PADDING_MINIMUM, tar.TYPE_FLAG_FIELD_SIZE,
		tar.ARCHIVE_FOOTER_BLOCK_COUNT,
	} {
		content := make([]byte, boundary_size)
		var archive_storage [TEST_ARCHIVE_SIZE]byte
		archive := tar_entry(
			archive_storage[:], 0, tar.FORMAT_USTAR, tar.TYPE_GNU_LONG_NAME,
			[]byte(tar.PAX_HEADER_NAME), content,
		)
		archive = tar_entry(
			archive_storage[:], len(archive), tar.FORMAT_USTAR,
			tar.TYPE_REGULAR, []byte("entry"), nil,
		)
		archive = archive_footer(archive_storage[:], len(archive))
		status, _ := reader_archive_header(
			t, archive, make([]byte, boundary_size),
		)
		wanted := tar.Status(tar.STATUS_OK)
		if boundary_size != TEST_DOMAIN_MINIMUM {
			wanted = tar.Status(tar.STATUS_OUTPUT_TOO_SMALL)
		}
		if status != wanted {
			t.Fatalf("metadata boundary %d status = %v", boundary_size, status)
		}
	}

	var archive_storage [TEST_ARCHIVE_SIZE]byte
	archive := tar_entry(
		archive_storage[:], 0, tar.FORMAT_GNU, tar.TYPE_GNU_LONG_NAME,
		[]byte(tar.PAX_HEADER_NAME), nil,
	)
	archive = tar_entry(
		archive_storage[:], len(archive), tar.FORMAT_GNU,
		tar.TYPE_GNU_LONG_NAME, []byte(tar.PAX_HEADER_NAME), []byte("n"),
	)
	archive = tar_entry(
		archive_storage[:], len(archive), tar.FORMAT_GNU,
		tar.TYPE_GNU_LONG_LINK, []byte(tar.PAX_HEADER_NAME), []byte("ll"),
	)
	large := make([]byte, tar.BLOCK_SIZE+tar.TYPE_FLAG_FIELD_SIZE)
	test_fill_bytes(large, 'x')
	archive = tar_entry(
		archive_storage[:], len(archive), tar.FORMAT_GNU,
		tar.TYPE_GNU_LONG_NAME, []byte(tar.PAX_HEADER_NAME), large,
	)
	archive = tar_entry(
		archive_storage[:], len(archive), tar.FORMAT_GNU,
		tar.TYPE_REGULAR, []byte("entry"), nil,
	)
	archive = archive_footer(archive_storage[:], len(archive))
	status, _ := reader_archive_header(
		t, archive, make([]byte, tar.SPECIAL_FILE_SIZE_MAXIMUM),
	)
	if status != tar.STATUS_OK {
		t.Fatalf("metadata block positions status = %v", status)
	}
	test_reader_metadata_maximum(t)
}

func test_reader_metadata_maximum(t *testing.T) {
	t.Helper()
	maximum := make([]byte, tar.SPECIAL_FILE_SIZE_MAXIMUM)
	test_fill_bytes(maximum, 'm')
	maximum[len(maximum)-tar.GNU_LONG_FIELD_TERMINATOR_SIZE] =
		byte(TEST_PADDING_MINIMUM)
	maximum_archive_storage := make(
		[]byte, (tar.GNU_LONG_FIELD_COUNT_MAXIMUM+tar.TYPE_FLAG_FIELD_SIZE)*
			(tar.SPECIAL_FILE_SIZE_MAXIMUM+tar.BLOCK_SIZE)+
			tar.ARCHIVE_FOOTER_SIZE+tar.ARCHIVE_FOOTER_SIZE,
	)
	maximum_archive := maximum_archive_storage[:0]
	field_count_maximum := tar.GNU_LONG_FIELD_COUNT_MAXIMUM +
		tar.TYPE_FLAG_FIELD_SIZE
	for field := TEST_DOMAIN_MINIMUM; field < field_count_maximum; field++ {
		maximum_archive = tar_entry(
			maximum_archive_storage, len(maximum_archive), tar.FORMAT_GNU,
			tar.TYPE_GNU_LONG_NAME, []byte(tar.PAX_HEADER_NAME), maximum,
		)
	}
	maximum_archive = tar_entry(
		maximum_archive_storage, len(maximum_archive), tar.FORMAT_GNU,
		tar.TYPE_GNU_LONG_LINK, []byte(tar.PAX_HEADER_NAME), []byte("x"),
	)
	maximum_archive = archive_footer(
		maximum_archive_storage, len(maximum_archive),
	)
	status, _ := reader_archive_header(
		t, maximum_archive, make([]byte, tar.READER_METADATA_SIZE_MAXIMUM),
	)
	if status != tar.STATUS_OUTPUT_TOO_SMALL {
		t.Fatalf("maximum metadata position status = %v", status)
	}
}

func reader_archive_header(
	t *testing.T, archive []byte, metadata []byte,
) (status tar.Status, header tar.Header) {
	t.Helper()
	memory := nbio.Stream_Memory{Memory: archive}
	var block [tar.BLOCK_SIZE]byte
	name := make([]byte, tar.HEADER_TEXT_SIZE_MAXIMUM)
	link_name := make([]byte, tar.HEADER_TEXT_SIZE_MAXIMUM)
	var user_name [tar.HEADER_USER_NAME_SIZE_MAXIMUM]byte
	var group_name [tar.HEADER_GROUP_NAME_SIZE_MAXIMUM]byte
	var reader tar.Reader
	if initialization := tar.Reader_Init(
		&reader, nbio.Memory_To_Stream(&memory),
		tar.Reader_Storage{Block: block[:], Metadata: metadata},
	); initialization != tar.STATUS_OK {
		t.Fatalf("metadata Reader_Init status = %v", initialization)
	}
	called := false
	tar.Reader_Next(
		&reader, &reader.Completion, tar.Header_Storage{
			Name: name, Link_Name: link_name,
			User_Name: user_name[:], Group_Name: group_name[:],
		},
		func(_ nbio.Completion_Handle) {
			header = reader.Archive.Header
			called = true
		},
	)
	if !called {
		t.Fatal("metadata Reader_Next did not retire")
	}
	return reader.Status, header
}

func test_reader_pax_record_domains(t *testing.T) {
	t.Helper()
	test_reader_pax_record_syntax(t)
	test_reader_pax_size_domains(t)
	test_reader_pax_value_domains(t)
}

func test_reader_pax_record_syntax(t *testing.T) {
	t.Helper()
	if status := reader_pax_records_status(
		t, nil, tar.Header_Storage{},
	); status != tar.STATUS_OK {
		t.Fatalf("empty PAX records status = %v", status)
	}
	var shortest [tar.PAX_RECORD_SIZE_MINIMUM]byte
	shortest_count := pax_record_into(shortest[:], []byte("a"), nil)
	if shortest_count != len(shortest) {
		t.Fatalf("shortest PAX record = %d", shortest_count)
	}
	if status := reader_pax_records_status(
		t, shortest[:shortest_count], tar.Header_Storage{},
	); status != tar.STATUS_OK {
		t.Fatalf("shortest PAX record status = %v", status)
	}
	var two_key [tar.PAX_RECORD_SIZE_MINIMUM + tar.TYPE_FLAG_FIELD_SIZE]byte
	two_key_count := pax_record_into(two_key[:], []byte("ab"), nil)
	if status := reader_pax_records_status(
		t, two_key[:two_key_count], tar.Header_Storage{},
	); status != tar.STATUS_OK {
		t.Fatalf("two-byte PAX key status = %v", status)
	}
	var empty_basic [TEST_FIELD_SIZE]byte
	empty_basic_count := pax_record_into(
		empty_basic[:], []byte(tar.PAX_PATH_KEY), nil,
	)
	if status := reader_pax_records_status(
		t, empty_basic[:empty_basic_count], tar.Header_Storage{},
	); status != tar.STATUS_OK {
		t.Fatalf("empty basic PAX value status = %v", status)
	}
	var invalid_numeric [TEST_FIELD_SIZE]byte
	invalid_numeric_count := pax_record_into(
		invalid_numeric[:], []byte(tar.PAX_USER_IDENTIFIER_KEY), []byte("-"),
	)
	if status := reader_pax_records_status(
		t, invalid_numeric[:invalid_numeric_count], tar.Header_Storage{},
	); status != tar.STATUS_INPUT_INVALID {
		t.Fatalf("invalid PAX numeric status = %v", status)
	}
	var invalid_timestamp [TEST_FIELD_SIZE]byte
	invalid_timestamp_count := pax_record_into(
		invalid_timestamp[:], []byte(tar.PAX_MODIFICATION_TIME_KEY), []byte("."),
	)
	if status := reader_pax_records_status(
		t, invalid_timestamp[:invalid_timestamp_count], tar.Header_Storage{},
	); status != tar.STATUS_INPUT_INVALID {
		t.Fatalf("invalid PAX timestamp status = %v", status)
	}
	var sparse [TEST_FIELD_SIZE]byte
	sparse_count := pax_record_into(
		sparse[:], []byte("GNU.sparse.x"), []byte("value"),
	)
	if status := reader_pax_records_status(
		t, sparse[:sparse_count], tar.Header_Storage{},
	); status != tar.STATUS_FORMAT_UNSUPPORTED {
		t.Fatalf("sparse PAX status = %v", status)
	}
}

func test_reader_pax_size_domains(t *testing.T) {
	t.Helper()
	var oversized_size_text [strconv.INTEGER_TEXT_SIZE_MAXIMUM]byte
	oversized_size := test_integer_text(
		oversized_size_text[:],
		int64(tar.ARCHIVE_SIZE_MAXIMUM+tar.TYPE_FLAG_FIELD_SIZE),
	)
	var oversized_size_record [TEST_FIELD_SIZE]byte
	oversized_size_count := pax_record_into(
		oversized_size_record[:], []byte(tar.PAX_SIZE_KEY), oversized_size,
	)
	if status := reader_pax_records_status(
		t, oversized_size_record[:oversized_size_count], tar.Header_Storage{},
	); status != tar.STATUS_FIELD_TOO_LONG {
		t.Fatalf("oversized PAX size status = %v", status)
	}
	for _, size := range [...]int64{
		bits.INTEGER_64_MINIMUM, -tar.TYPE_FLAG_FIELD_SIZE,
		bits.INTEGER_64_MAXIMUM,
	} {
		var size_text [strconv.INTEGER_TEXT_SIZE_MAXIMUM]byte
		value := test_integer_text(size_text[:], size)
		var record [TEST_FIELD_SIZE]byte
		count := pax_record_into(record[:], []byte(tar.PAX_SIZE_KEY), value)
		if size == bits.INTEGER_64_MAXIMUM {
			count += pax_record_into(record[count:], []byte("a"), nil)
		}
		wanted := tar.Status(tar.STATUS_FIELD_TOO_LONG)
		if size < TEST_PADDING_MINIMUM {
			wanted = tar.Status(tar.STATUS_INPUT_INVALID)
		}
		if status := reader_pax_records_status(
			t, record[:count], tar.Header_Storage{},
		); status != wanted {
			t.Fatalf("PAX size %d status = %v", size, status)
		}
	}
}

func test_reader_pax_value_domains(t *testing.T) {
	t.Helper()
	maximum_value := make([]byte, tar.PAX_VALUE_SIZE_MAXIMUM)
	test_fill_bytes(maximum_value, 'v')
	maximum_record := make([]byte, tar.SPECIAL_FILE_SIZE_MAXIMUM)
	maximum_count := pax_record_into(maximum_record, []byte("a"), maximum_value)
	if maximum_count != len(maximum_record) {
		t.Fatalf("maximum PAX value record = %d", maximum_count)
	}
	if status := reader_pax_records_status(
		t, maximum_record, tar.Header_Storage{},
	); status != tar.STATUS_OK {
		t.Fatalf("maximum PAX value status = %v", status)
	}
	numeric_value := make([]byte, tar.PAX_NUMERIC_VALUE_SIZE_MAXIMUM)
	test_fill_bytes(numeric_value, '9')
	numeric_record := make([]byte, tar.SPECIAL_FILE_SIZE_MAXIMUM)
	numeric_count := pax_record_into(
		numeric_record, []byte(tar.PAX_USER_IDENTIFIER_KEY), numeric_value,
	)
	if status := reader_pax_records_status(
		t, numeric_record[:numeric_count], tar.Header_Storage{},
	); status != tar.STATUS_INPUT_INVALID {
		t.Fatalf("maximum PAX numeric status = %v", status)
	}
	timestamp_value := make([]byte, tar.PAX_TIMESTAMP_VALUE_SIZE_MAXIMUM)
	test_fill_bytes(timestamp_value, '9')
	timestamp_record := make([]byte, tar.SPECIAL_FILE_SIZE_MAXIMUM)
	timestamp_count := pax_record_into(
		timestamp_record, []byte(tar.PAX_MODIFICATION_TIME_KEY), timestamp_value,
	)
	if status := reader_pax_records_status(
		t, timestamp_record[:timestamp_count], tar.Header_Storage{},
	); status != tar.STATUS_INPUT_INVALID {
		t.Fatalf("maximum PAX timestamp status = %v", status)
	}

	test_reader_pax_apply_text_maximum(
		t, []byte(tar.PAX_USER_NAME_KEY), tar.PAX_APPLY_USER_NAME_SIZE_MAXIMUM,
		tar.Header_Storage{
			User_Name: make([]byte, tar.HEADER_USER_NAME_SIZE_MAXIMUM),
		},
	)
	test_reader_pax_apply_text_maximum(
		t, []byte(tar.PAX_GROUP_NAME_KEY), tar.PAX_APPLY_GROUP_NAME_SIZE_MAXIMUM,
		tar.Header_Storage{
			Group_Name: make([]byte, tar.HEADER_GROUP_NAME_SIZE_MAXIMUM),
		},
	)
	var nul_record [TEST_FIELD_SIZE]byte
	nul_count := pax_record_into(
		nul_record[:], []byte(tar.PAX_PATH_KEY), []byte("\x00"),
	)
	if status := reader_pax_records_status(
		t, nul_record[:nul_count], tar.Header_Storage{},
	); status != tar.STATUS_INPUT_INVALID {
		t.Fatalf("NUL PAX value status = %v", status)
	}
}

func test_reader_pax_apply_text_maximum(
	t *testing.T, key []byte, value_size int, storage tar.Header_Storage,
) {
	t.Helper()
	value := make([]byte, value_size)
	test_fill_bytes(value, 'v')
	records := make([]byte, tar.SPECIAL_FILE_SIZE_MAXIMUM)
	count := pax_record_into(records, key, value)
	count += pax_record_into(records[count:], []byte("a"), nil)
	if count != len(records) {
		t.Fatalf("PAX apply text payload %q = %d", key, count)
	}
	if status := reader_pax_records_status(t, records, storage); status != tar.STATUS_OK {
		t.Fatalf("PAX apply text %q status = %v", key, status)
	}
}

func reader_pax_records_status(
	t *testing.T, records []byte, storage tar.Header_Storage,
) (status tar.Status) {
	t.Helper()
	archive_storage := make(
		[]byte, padded_size(len(records))+
			tar.ARCHIVE_FOOTER_SIZE+tar.ARCHIVE_FOOTER_SIZE,
	)
	archive := tar_entry(
		archive_storage, 0, tar.FORMAT_USTAR, tar.TYPE_PAX_LOCAL,
		[]byte(tar.PAX_HEADER_NAME), records,
	)
	archive = tar_entry(
		archive_storage, len(archive), tar.FORMAT_USTAR,
		tar.TYPE_REGULAR, []byte("entry"), nil,
	)
	archive = archive_footer(archive_storage, len(archive))
	if len(storage.Name) == 0 {
		storage.Name = make([]byte, TEST_FIELD_SIZE)
	}
	if len(storage.Link_Name) == 0 {
		storage.Link_Name = make([]byte, TEST_FIELD_SIZE)
	}
	if len(storage.User_Name) == 0 {
		storage.User_Name = make([]byte, TEST_FIELD_SIZE)
	}
	if len(storage.Group_Name) == 0 {
		storage.Group_Name = make([]byte, TEST_FIELD_SIZE)
	}
	return test_reader_archive(
		archive, make([]byte, tar.SPECIAL_FILE_SIZE_MAXIMUM), storage,
	)
}

func test_reader_fixed_header_failures(t *testing.T) {
	t.Helper()
	var invalid_mode_storage [TEST_ARCHIVE_SIZE]byte
	invalid_mode := ustar_archive(
		invalid_mode_storage[:], []byte("entry"), nil,
	)
	invalid_mode[tar.MODE_FIELD_OFFSET] = '8'
	tar_header_checksum(invalid_mode[:tar.BLOCK_SIZE])
	test_reader_fixed_status(t, "mode", invalid_mode, tar.STATUS_INPUT_INVALID)

	var oversized_storage [TEST_ARCHIVE_SIZE]byte
	oversized := ustar_archive(oversized_storage[:], []byte("entry"), nil)
	test_fill_bytes(
		oversized[tar.ENTRY_SIZE_FIELD_OFFSET:][:tar.ENTRY_SIZE_FIELD_SIZE], '7',
	)
	tar_header_checksum(oversized[:tar.BLOCK_SIZE])
	test_reader_fixed_status(t, "size", oversized, tar.STATUS_FIELD_TOO_LONG)

	var invalid_device_storage [TEST_ARCHIVE_SIZE]byte
	invalid_device := ustar_archive(
		invalid_device_storage[:], []byte("entry"), nil,
	)
	invalid_device[tar.DEVICE_MAJOR_FIELD_OFFSET] = '8'
	tar_header_checksum(invalid_device[:tar.BLOCK_SIZE])
	test_reader_fixed_status(t, "device", invalid_device, tar.STATUS_INPUT_INVALID)

	var invalid_star_storage [TEST_ARCHIVE_SIZE]byte
	invalid_star := ustar_archive(
		invalid_star_storage[:], []byte("entry"), nil,
	)
	invalid_star_header := invalid_star[:tar.BLOCK_SIZE]
	copy(
		invalid_star_header[tar.STAR_TRAILER_FIELD_OFFSET:][:tar.STAR_TRAILER_FIELD_SIZE],
		tar.STAR_TRAILER,
	)
	invalid_star_header[tar.STAR_ACCESS_TIMESTAMP_FIELD_OFFSET] = '8'
	tar_header_checksum(invalid_star_header)
	test_reader_fixed_status(t, "STAR", invalid_star, tar.STATUS_INPUT_INVALID)
	test_reader_gnu_header_domains(t)
}

func test_reader_gnu_header_domains(t *testing.T) {
	t.Helper()
	var invalid_gnu_storage [TEST_ARCHIVE_SIZE]byte
	invalid_gnu := tar_entry(
		invalid_gnu_storage[:], 0, tar.FORMAT_GNU, tar.TYPE_REGULAR,
		[]byte("entry"), nil,
	)
	invalid_gnu_header := invalid_gnu[:tar.BLOCK_SIZE]
	test_fill_bytes(
		invalid_gnu_header[tar.PREFIX_FIELD_OFFSET:][:tar.PREFIX_FIELD_SIZE], '8',
	)
	invalid_gnu_header[tar.PREFIX_FIELD_OFFSET+tar.TIMESTAMP_FIELD_SIZE] =
		[]byte("¢")[TEST_PADDING_MINIMUM]
	tar_header_checksum(invalid_gnu_header)
	invalid_gnu = archive_footer(invalid_gnu_storage[:], len(invalid_gnu))
	test_reader_fixed_status(t, "GNU", invalid_gnu, tar.STATUS_INPUT_INVALID)

	var empty_gnu_storage [TEST_ARCHIVE_SIZE]byte
	empty_gnu := tar_entry(
		empty_gnu_storage[:], 0, tar.FORMAT_GNU, tar.TYPE_REGULAR, nil, nil,
	)
	empty_gnu_header := empty_gnu[:tar.BLOCK_SIZE]
	empty_gnu_header[tar.GNU_CHANGE_TIMESTAMP_FIELD_OFFSET] = '8'
	tar_header_checksum(empty_gnu_header)
	empty_gnu = archive_footer(empty_gnu_storage[:], len(empty_gnu))
	test_reader_fixed_status(t, "empty GNU", empty_gnu, tar.STATUS_OK)

	var maximum_gnu_storage [TEST_ARCHIVE_SIZE]byte
	maximum_gnu := tar_entry(
		maximum_gnu_storage[:], 0, tar.FORMAT_GNU, tar.TYPE_REGULAR,
		[]byte("entry"), nil,
	)
	maximum_gnu_header := maximum_gnu[:tar.BLOCK_SIZE]
	test_fill_bytes(
		maximum_gnu_header[tar.PREFIX_FIELD_OFFSET:][:tar.PREFIX_FIELD_SIZE], 'p',
	)
	maximum_gnu_header[tar.GNU_ACCESS_TIMESTAMP_FIELD_OFFSET] = '8'
	tar_header_checksum(maximum_gnu_header)
	maximum_gnu = archive_footer(maximum_gnu_storage[:], len(maximum_gnu))
	test_reader_fixed_status(t, "maximum GNU", maximum_gnu, tar.STATUS_OK)
}

func test_reader_fixed_status(
	t *testing.T, name string, archive []byte, wanted tar.Status,
) {
	t.Helper()
	status := test_reader_archive(
		archive, make([]byte, tar.SPECIAL_FILE_SIZE_MAXIMUM), tar.Header_Storage{
			Name:       make([]byte, tar.HEADER_TEXT_SIZE_MAXIMUM),
			Link_Name:  make([]byte, tar.HEADER_TEXT_SIZE_MAXIMUM),
			User_Name:  make([]byte, tar.HEADER_USER_NAME_SIZE_MAXIMUM),
			Group_Name: make([]byte, tar.HEADER_GROUP_NAME_SIZE_MAXIMUM),
		},
	)
	if status != wanted {
		t.Fatalf("%s fixed header status = %v; want %v", name, status, wanted)
	}
}

func test_writer_remaining_domains(t *testing.T) {
	t.Helper()
	test_writer_ustar_name_domains(t)
	test_writer_timestamp_domains(t)
	test_writer_workspace_domains(t)
	test_writer_padding_domains(t)
}

func test_writer_ustar_name_domains(t *testing.T) {
	t.Helper()
	for _, prefix_size := range [...]int{
		tar.TYPE_FLAG_FIELD_SIZE, tar.ARCHIVE_FOOTER_BLOCK_COUNT,
		tar.PREFIX_FIELD_SIZE,
	} {
		name := make(
			[]byte, prefix_size+tar.TYPE_FLAG_FIELD_SIZE+tar.NAME_FIELD_SIZE,
		)
		test_fill_bytes(name[:prefix_size], 'p')
		name[prefix_size] = '/'
		test_fill_bytes(name[prefix_size+tar.TYPE_FLAG_FIELD_SIZE:], 'n')
		header := tar.Header_Unvalidated{
			Format:    tar.Format_Unvalidated(tar.FORMAT_USTAR),
			Type_Flag: tar.TYPE_REGULAR, Name: name,
		}
		test_writer_header_after_padding(
			t, header, TEST_PADDING_MINIMUM, tar.WRITER_STORAGE_SIZE_MAXIMUM,
		)
	}
	rejected_split := make(
		[]byte, tar.TYPE_FLAG_FIELD_SIZE+tar.NAME_FIELD_SIZE,
	)
	rejected_split[TEST_PADDING_MINIMUM] = '/'
	test_fill_bytes(rejected_split[tar.TYPE_FLAG_FIELD_SIZE:], 'n')
	test_writer_status(
		t, tar.Header_Unvalidated{
			Format:    tar.Format_Unvalidated(tar.FORMAT_USTAR),
			Type_Flag: tar.TYPE_REGULAR, Name: rejected_split,
		}, tar.WRITER_STORAGE_SIZE_MAXIMUM, tar.STATUS_FIELD_TOO_LONG,
	)
}

func test_writer_timestamp_domains(t *testing.T) {
	t.Helper()
	test_writer_status(
		t, tar.Header_Unvalidated{
			Format:    tar.Format_Unvalidated(tar.FORMAT_USTAR),
			Type_Flag: tar.TYPE_REGULAR, Name: []byte("entry"),
			Modification_Time: tar.Timestamp_Unvalidated{
				Seconds: tar.Integer(tar.LARGE_OCTAL_MAXIMUM), Set: true,
			},
		}, tar.WRITER_STORAGE_SIZE_MAXIMUM, tar.STATUS_OK,
	)
	test_writer_status(
		t, tar.Header_Unvalidated{
			Format:    tar.Format_Unvalidated(tar.FORMAT_GNU),
			Type_Flag: tar.TYPE_REGULAR, Name: []byte("entry"),
			Modification_Time: tar.Timestamp_Unvalidated{
				Seconds: tar.Integer(bits.INTEGER_64_MINIMUM), Set: true,
			},
		}, tar.WRITER_STORAGE_SIZE_MAXIMUM, tar.STATUS_OK,
	)
	for _, divisor := range [...]int32{
		tar.DECIMAL_BASE, tar.DECIMAL_BASE * tar.DECIMAL_BASE,
	} {
		nanoseconds := tar.TIMESTAMP_NANOSECOND_COUNT / time.Duration(divisor)
		test_writer_status(
			t, tar.Header_Unvalidated{
				Format:    tar.Format_Unvalidated(tar.FORMAT_PAX),
				Type_Flag: tar.TYPE_REGULAR, Name: []byte("entry"),
				Modification_Time: tar.Timestamp_Unvalidated{
					Nanoseconds: tar.Nanosecond_Count_Unvalidated(nanoseconds),
					Set:         true,
				},
			}, tar.WRITER_STORAGE_SIZE_MAXIMUM, tar.STATUS_OK,
		)
	}
}

func test_writer_workspace_domains(t *testing.T) {
	t.Helper()
	test_writer_status(
		t, tar.Header_Unvalidated{
			Format:    tar.Format_Unvalidated(tar.FORMAT_PAX),
			Type_Flag: tar.TYPE_REGULAR, Name: []byte("entry"),
		}, tar.WRITER_STORAGE_SIZE_MINIMUM, tar.STATUS_OUTPUT_TOO_SMALL,
	)
	var records [tar.PAX_RECORD_SIZE_MINIMUM]byte
	record_count := pax_record_into(records[:], []byte("a"), nil)
	test_writer_status(
		t, tar.Header_Unvalidated{
			Format:    tar.Format_Unvalidated(tar.FORMAT_PAX),
			Type_Flag: tar.TYPE_REGULAR, Name: []byte("entry"),
			PAX_Records: records[:record_count],
		}, tar.WRITER_STORAGE_SIZE_MINIMUM, tar.STATUS_OUTPUT_TOO_SMALL,
	)

	maximum_nul_name := make([]byte, tar.HEADER_TEXT_SIZE_MAXIMUM)
	test_fill_bytes(maximum_nul_name, 'n')
	maximum_nul_name[len(maximum_nul_name)-tar.TYPE_FLAG_FIELD_SIZE] =
		byte(TEST_PADDING_MINIMUM)
	test_writer_status(
		t, tar.Header_Unvalidated{
			Format:    tar.Format_Unvalidated(tar.FORMAT_PAX),
			Type_Flag: tar.TYPE_REGULAR, Name: maximum_nul_name,
		}, tar.WRITER_STORAGE_SIZE_MINIMUM, tar.STATUS_INPUT_INVALID,
	)
}

func test_writer_padding_domains(t *testing.T) {
	t.Helper()
	var archive [TEST_ARCHIVE_SIZE]byte
	var fixture writer_fixture
	if status := writer_fixture_init(&fixture, archive[:]); status != tar.STATUS_OK {
		t.Fatalf("maximum padding Writer_Init status = %v", status)
	}
	header := tar.Header_Unvalidated{
		Format:    tar.Format_Unvalidated(tar.FORMAT_USTAR),
		Type_Flag: tar.TYPE_REGULAR, Name: []byte("entry"),
		Size: tar.Entry_Size_Unvalidated(tar.TYPE_FLAG_FIELD_SIZE),
	}
	writer_fixture_write_header(&fixture, &header)
	writer_fixture_write(&fixture, []byte("x"))
	writer_fixture_close(&fixture)
	if fixture.Writer.Status != tar.STATUS_OK {
		t.Fatalf("maximum padding close status = %v", fixture.Writer.Status)
	}

	var directory_archive [TEST_ARCHIVE_SIZE]byte
	var directory writer_fixture
	if status := writer_fixture_init(
		&directory, directory_archive[:],
	); status != tar.STATUS_OK {
		t.Fatalf("directory Writer_Init status = %v", status)
	}
	directory_header := tar.Header_Unvalidated{
		Format:    tar.Format_Unvalidated(tar.FORMAT_USTAR),
		Type_Flag: tar.TYPE_DIRECTORY, Name: []byte("directory"),
		Size: tar.Entry_Size_Unvalidated(tar.TYPE_FLAG_FIELD_SIZE),
	}
	writer_fixture_write_header(&directory, &directory_header)
	if directory.Writer.Status != tar.STATUS_OK {
		t.Fatalf("directory header status = %v", directory.Writer.Status)
	}
}

func test_writer_status(
	t *testing.T, header tar.Header_Unvalidated, workspace_size int,
	wanted tar.Status,
) {
	t.Helper()
	archive := make([]byte, TEST_ARCHIVE_SIZE)
	workspace := make([]byte, workspace_size)
	memory := nbio.Stream_Memory{Memory: archive}
	var writer tar.Writer
	if status := tar.Writer_Init(
		&writer, nbio.Memory_To_Stream(&memory), workspace,
	); status != tar.STATUS_OK {
		t.Fatalf("domain Writer_Init status = %v", status)
	}
	tar.Writer_Write_Header(
		&writer, &writer.Completion, &header, test_domain_callback,
	)
	if writer.Status != wanted {
		t.Fatalf("domain Writer status = %v; want %v", writer.Status, wanted)
	}
}

func test_storage_overlap_domains(t *testing.T) {
	t.Helper()
	var overlap_storage [tar.BLOCK_SIZE + tar.TYPE_FLAG_FIELD_SIZE]byte
	var reader tar.Reader
	memory := nbio.Stream_Memory{Memory: overlap_storage[:]}
	status := tar.Reader_Init(
		&reader, nbio.Memory_To_Stream(&memory), tar.Reader_Storage{
			Block:    overlap_storage[:tar.BLOCK_SIZE],
			Metadata: overlap_storage[tar.TYPE_FLAG_FIELD_SIZE:],
		},
	)
	if status != tar.STATUS_STORAGE_INVALID {
		t.Fatalf("overlapping Reader_Init status = %v", status)
	}

	var archive [tar.ARCHIVE_FOOTER_SIZE]byte
	var block [tar.BLOCK_SIZE]byte
	metadata := make([]byte, tar.READER_METADATA_SIZE_MAXIMUM)
	memory = nbio.Stream_Memory{Memory: archive[:]}
	if status = tar.Reader_Init(
		&reader, nbio.Memory_To_Stream(&memory),
		tar.Reader_Storage{Block: block[:], Metadata: metadata},
	); status != tar.STATUS_OK {
		t.Fatalf("overlap Reader_Init status = %v", status)
	}
	called := false
	tar.Reader_Next(
		&reader, &reader.Completion,
		tar.Header_Storage{Name: metadata[:tar.HEADER_TEXT_SIZE_MAXIMUM]},
		func(_ nbio.Completion_Handle) { called = true },
	)
	if !called {
		t.Fatal("overlap Reader_Next did not retire")
	}
	if reader.Status != tar.STATUS_STORAGE_INVALID {
		t.Fatalf("overlap Reader_Next status = %v", reader.Status)
	}
	shared := make([]byte, TEST_FIELD_SIZE)
	called = false
	tar.Reader_Next(
		&reader, &reader.Completion,
		tar.Header_Storage{Name: shared, Link_Name: shared},
		func(_ nbio.Completion_Handle) { called = true },
	)
	if !called {
		t.Fatal("field overlap Reader_Next did not retire")
	}
	if reader.Status != tar.STATUS_STORAGE_INVALID {
		t.Fatalf("field overlap Reader_Next status = %v", reader.Status)
	}
}

func test_active_preconditions(t *testing.T) {
	t.Helper()
	reader := test_domain_reader(TEST_DOMAIN_ONE)
	reader.Active = true
	test_invariant_panic(t, "Reader_Next active", func() {
		tar.Reader_Next(
			&reader, &reader.Completion, tar.Header_Storage{}, test_domain_callback,
		)
	})
	reader = test_domain_reader(TEST_DOMAIN_ONE)
	reader.Active = true
	test_invariant_panic(t, "Reader_Read active", func() {
		tar.Reader_Read(&reader, &reader.Completion, nil, test_domain_callback)
	})
	header := tar.Header_Unvalidated{Name: []byte{'n'}}
	writer := test_domain_writer(TEST_DOMAIN_ONE)
	writer.Active = true
	test_invariant_panic(t, "Writer_Write_Header active", func() {
		tar.Writer_Write_Header(
			&writer, &writer.Completion, &header, test_domain_callback,
		)
	})
	writer = test_domain_writer(TEST_DOMAIN_ONE)
	writer.Active = true
	test_invariant_panic(t, "Writer_Write active", func() {
		tar.Writer_Write(&writer, &writer.Completion, nil, test_domain_callback)
	})
	writer = test_domain_writer(TEST_DOMAIN_ONE)
	writer.Active = true
	test_invariant_panic(t, "Writer_Close active", func() {
		tar.Writer_Close(&writer, &writer.Completion, test_domain_callback)
	})
}

func test_invariant_panic(t *testing.T, name string, operation func()) {
	t.Helper()
	panicked := false
	func() {
		defer func() {
			if recover() != nil {
				panicked = true
			}
		}()
		operation()
	}()
	if !panicked {
		t.Fatalf("%s did not assert", name)
	}
}

func test_writer_position_boundaries(t *testing.T) {
	t.Helper()
	for _, marker := range []int{
		TEST_DOMAIN_ZERO, TEST_DOMAIN_ONE, TEST_DOMAIN_TWO, TEST_DOMAIN_MAXIMUM,
	} {
		padding := test_domain_size(marker, tar.PADDING_SIZE_MAXIMUM)
		test_writer_position(
			t, tar.FORMAT_GNU, nil, padding, tar.WRITER_STORAGE_SIZE_MAXIMUM,
		)
		test_writer_position(
			t, tar.FORMAT_PAX, nil, padding, tar.WRITER_STORAGE_SIZE_MAXIMUM,
		)
		test_writer_position(
			t, tar.FORMAT_PAX, test_writer_pax_records(TEST_DOMAIN_ONE), padding,
			tar.WRITER_STORAGE_SIZE_MAXIMUM,
		)
	}
	test_writer_position(
		t, tar.FORMAT_GNU, nil, TEST_PADDING_MINIMUM, tar.WRITER_STORAGE_SIZE_MINIMUM,
	)
	test_writer_position(
		t, tar.FORMAT_PAX, nil, TEST_PADDING_MINIMUM, tar.PAX_HEADER_WIRE_SIZE_MINIMUM,
	)
	test_writer_position(
		t, tar.FORMAT_PAX, test_writer_pax_records(TEST_DOMAIN_ONE),
		TEST_PADDING_MINIMUM, tar.PAX_HEADER_WIRE_SIZE_MINIMUM,
	)
}

func test_writer_position(
	t *testing.T, format tar.Format, records []byte, padding int, storage_size int,
) {
	t.Helper()
	second := tar.Header_Unvalidated{
		Format: tar.Format_Unvalidated(format), Type_Flag: tar.TYPE_REGULAR,
		Name: []byte("second"), PAX_Records: records,
	}
	test_writer_header_after_padding(t, second, padding, storage_size)
}

func test_writer_gnu_long_boundaries(t *testing.T) {
	t.Helper()
	for _, field := range []int{TEST_HEADER_NAME, TEST_HEADER_LINK_NAME} {
		header := tar.Header_Unvalidated{
			Format:    tar.Format_Unvalidated(tar.FORMAT_GNU),
			Type_Flag: tar.TYPE_REGULAR, Name: []byte("second"),
		}
		value := make([]byte, tar.GNU_EXTENDED_FIELD_SIZE_MINIMUM)
		test_fill_bytes(value, 'g')
		if field == TEST_HEADER_NAME {
			header.Name = value
			for marker := TEST_DOMAIN_ONE; marker <= TEST_DOMAIN_TWO; marker++ {
				test_writer_header_after_padding(
					t, header,
					test_domain_size(marker, tar.PADDING_SIZE_MAXIMUM),
					tar.WRITER_STORAGE_SIZE_MAXIMUM,
				)
			}
		} else {
			header.Link_Name = value
		}
		test_writer_header_after_padding(
			t, header, TEST_PADDING_MINIMUM, tar.GNU_LONG_DESTINATION_SIZE_MINIMUM,
		)
	}
	maximum := make([]byte, tar.HEADER_TEXT_SIZE_MAXIMUM)
	test_fill_bytes(maximum, 'm')
	header := tar.Header_Unvalidated{
		Format:    tar.Format_Unvalidated(tar.FORMAT_GNU),
		Type_Flag: tar.TYPE_REGULAR, Name: maximum, Link_Name: maximum,
	}
	test_writer_header_after_padding(
		t, header, tar.PADDING_SIZE_MAXIMUM, tar.WRITER_STORAGE_SIZE_MAXIMUM,
	)
}

func test_writer_pax_payload_boundaries(t *testing.T) {
	t.Helper()
	maximum_name := make([]byte, tar.PAX_NAME_SIZE_MAXIMUM)
	test_fill_bytes(maximum_name, 'p')
	maximum_payload := tar.Header_Unvalidated{
		Format:    tar.Format_Unvalidated(tar.FORMAT_PAX),
		Type_Flag: tar.TYPE_REGULAR, Name: maximum_name,
		User_Identifier: tar.User_Identifier(test_domain_size(
			TEST_DOMAIN_ONE, tar.ARCHIVE_SIZE_MAXIMUM,
		)),
		Group_Identifier: tar.Group_Identifier(test_domain_size(
			TEST_DOMAIN_ONE, tar.ARCHIVE_SIZE_MAXIMUM,
		)),
	}
	test_writer_header_after_padding(
		t, maximum_payload, tar.PADDING_SIZE_MAXIMUM,
		tar.WRITER_STORAGE_SIZE_MAXIMUM,
	)
	maximum_entry := tar.Header_Unvalidated{
		Format:    tar.Format_Unvalidated(tar.FORMAT_PAX),
		Type_Flag: tar.TYPE_REGULAR, Name: []byte{'n'},
		Size: tar.Entry_Size_Unvalidated(tar.ARCHIVE_SIZE_MAXIMUM),
	}
	test_writer_header_after_padding(
		t, maximum_entry, TEST_PADDING_MINIMUM, tar.WRITER_STORAGE_SIZE_MAXIMUM,
	)
	minimum_records := tar.Header_Unvalidated{
		Format:    tar.Format_Unvalidated(tar.FORMAT_PAX),
		Type_Flag: tar.TYPE_REGULAR, Name: []byte{'n'},
		PAX_Records: test_writer_pax_records(TEST_DOMAIN_ONE),
	}
	test_writer_header_after_padding(
		t, minimum_records, TEST_PADDING_MINIMUM, tar.PAX_HEADER_WIRE_SIZE_MINIMUM,
	)
}

func test_writer_pax_record_boundaries(t *testing.T) {
	t.Helper()
	maximum_timestamp := tar.Timestamp_Unvalidated{
		Seconds: tar.Integer(bits.INTEGER_64_MINIMUM),
		Nanoseconds: tar.Nanosecond_Count_Unvalidated(
			tar.TIMESTAMP_NANOSECOND_MAXIMUM,
		), Set: true,
	}
	all_times := tar.Header_Unvalidated{
		Format:    tar.Format_Unvalidated(tar.FORMAT_PAX),
		Type_Flag: tar.TYPE_REGULAR, Name: []byte{'n'},
		Modification_Time: maximum_timestamp,
		Access_Time: tar.Access_Timestamp_Unvalidated{
			Seconds: tar.Access_Seconds(maximum_timestamp.Seconds),
			Nanoseconds: tar.Access_Nanosecond_Count_Unvalidated(
				maximum_timestamp.Nanoseconds,
			), Set: true,
		},
		Change_Time: tar.Change_Timestamp_Unvalidated{
			Seconds: tar.Change_Seconds(maximum_timestamp.Seconds),
			Nanoseconds: tar.Change_Nanosecond_Count_Unvalidated(
				maximum_timestamp.Nanoseconds,
			), Set: true,
		},
	}
	test_writer_header_after_padding(
		t, all_times, TEST_PADDING_MINIMUM, tar.WRITER_STORAGE_SIZE_MAXIMUM,
	)
	minimum_text_destination := tar.Header_Unvalidated{
		Format:    tar.Format_Unvalidated(tar.FORMAT_PAX),
		Type_Flag: tar.TYPE_REGULAR, Name: []byte{'n'}, Group_Name: []byte{'g'},
	}
	test_writer_header_after_padding(
		t, minimum_text_destination, TEST_PADDING_MINIMUM,
		tar.PAX_HEADER_WIRE_SIZE_MINIMUM,
	)
	maximum_decimal := tar.Header_Unvalidated{
		Format:    tar.Format_Unvalidated(tar.FORMAT_PAX),
		Type_Flag: tar.TYPE_REGULAR, Name: []byte{'n'},
		User_Identifier: tar.User_Identifier(bits.INTEGER_64_MINIMUM),
	}
	test_writer_header_after_padding(
		t, maximum_decimal, TEST_PADDING_MINIMUM, tar.PAX_HEADER_WIRE_SIZE_MINIMUM,
	)
	test_writer_pax_maximum_decimal_destination(t)
	test_writer_pax_maximum_records(t)
}

func test_writer_pax_maximum_decimal_destination(t *testing.T) {
	t.Helper()
	link_name := make([]byte, tar.PAX_LINK_NAME_SIZE_MAXIMUM)
	test_fill_bytes(link_name, 'l')
	one := test_domain_size(TEST_DOMAIN_ONE, tar.ARCHIVE_SIZE_MAXIMUM)
	header := tar.Header_Unvalidated{
		Format:    tar.Format_Unvalidated(tar.FORMAT_PAX),
		Type_Flag: tar.TYPE_REGULAR, Name: []byte{'n'}, Link_Name: link_name,
		User_Identifier:  tar.User_Identifier(one),
		Group_Identifier: tar.Group_Identifier(one),
	}
	test_writer_header_after_padding(
		t, header, TEST_PADDING_MINIMUM, tar.WRITER_STORAGE_SIZE_MAXIMUM,
	)
}

func test_writer_pax_maximum_records(t *testing.T) {
	t.Helper()
	header := tar.Header_Unvalidated{
		Format:    tar.Format_Unvalidated(tar.FORMAT_PAX),
		Type_Flag: tar.TYPE_REGULAR, Name: []byte{'n'},
		PAX_Records: test_writer_pax_records(TEST_DOMAIN_MAXIMUM),
	}
	test_writer_header_after_padding(
		t, header, tar.PADDING_SIZE_MAXIMUM, tar.WRITER_STORAGE_SIZE_MAXIMUM,
	)
}

func test_writer_pax_candidate_maximum(t *testing.T) {
	t.Helper()
	header := tar.Header_Unvalidated{
		Format:           tar.Format_Unvalidated(tar.FORMAT_PAX),
		Type_Flag:        tar.TYPE_REGULAR,
		Name:             make([]byte, tar.HEADER_TEXT_SIZE_MAXIMUM),
		Link_Name:        make([]byte, tar.HEADER_TEXT_SIZE_MAXIMUM),
		Size:             tar.Entry_Size_Unvalidated(tar.ARCHIVE_SIZE_MAXIMUM),
		User_Identifier:  tar.User_Identifier(bits.INTEGER_64_MINIMUM),
		Group_Identifier: tar.Group_Identifier(bits.INTEGER_64_MINIMUM),
		User_Name:        make([]byte, tar.HEADER_USER_NAME_SIZE_MAXIMUM),
		Group_Name:       make([]byte, tar.HEADER_GROUP_NAME_SIZE_MAXIMUM),
		Modification_Time: tar.Timestamp_Unvalidated{
			Seconds: tar.Integer(bits.INTEGER_64_MINIMUM),
			Nanoseconds: tar.Nanosecond_Count_Unvalidated(
				tar.TIMESTAMP_NANOSECOND_MAXIMUM,
			), Set: true,
		},
		Access_Time: tar.Access_Timestamp_Unvalidated{
			Seconds: tar.Access_Seconds(bits.INTEGER_64_MINIMUM),
			Nanoseconds: tar.Access_Nanosecond_Count_Unvalidated(
				tar.TIMESTAMP_NANOSECOND_MAXIMUM,
			), Set: true,
		},
		Change_Time: tar.Change_Timestamp_Unvalidated{
			Seconds: tar.Change_Seconds(bits.INTEGER_64_MINIMUM),
			Nanoseconds: tar.Change_Nanosecond_Count_Unvalidated(
				tar.TIMESTAMP_NANOSECOND_MAXIMUM,
			), Set: true,
		},
	}
	test_fill_bytes(header.Name, 'n')
	test_fill_bytes(header.Link_Name, 'l')
	test_fill_bytes(header.User_Name, 'u')
	test_fill_bytes(header.Group_Name, 'g')
	var archive [TEST_ARCHIVE_SIZE]byte
	memory := nbio.Stream_Memory{Memory: archive[:]}
	var writer tar.Writer
	tar.Writer_Init(
		&writer, nbio.Memory_To_Stream(&memory),
		make([]byte, tar.WRITER_STORAGE_SIZE_MAXIMUM),
	)
	tar.Writer_Write_Header(
		&writer, &writer.Completion, &header, test_domain_callback,
	)
	if writer.Status != tar.STATUS_FIELD_TOO_LONG {
		t.Fatalf("PAX maximum candidate status = %d", writer.Status)
	}
}

func test_writer_header_after_padding(
	t *testing.T, second tar.Header_Unvalidated, padding int, storage_size int,
) {
	t.Helper()
	content_size := tar.BLOCK_SIZE - padding
	if padding == TEST_PADDING_MINIMUM {
		content_size = tar.BLOCK_SIZE
	}
	archive := make([]byte, storage_size+TEST_ARCHIVE_SIZE)
	memory := nbio.Stream_Memory{Memory: archive}
	var writer tar.Writer
	status := tar.Writer_Init(
		&writer, nbio.Memory_To_Stream(&memory), make([]byte, storage_size),
	)
	if status != tar.STATUS_OK {
		t.Fatalf("position Writer_Init size %d: status %d", storage_size, status)
	}
	first := tar.Header_Unvalidated{
		Format: tar.Format_Unvalidated(tar.FORMAT_USTAR), Type_Flag: tar.TYPE_REGULAR,
		Name: []byte("first"), Size: tar.Entry_Size_Unvalidated(content_size),
	}
	tar.Writer_Write_Header(
		&writer, &writer.Completion, &first, test_domain_callback,
	)
	tar.Writer_Write(
		&writer, &writer.Completion, make([]byte, content_size), test_domain_callback,
	)
	tar.Writer_Write_Header(
		&writer, &writer.Completion, &second, test_domain_callback,
	)
	if writer.Status != tar.STATUS_OK {
		t.Fatalf(
			"position format %d padding %d storage %d: status %d",
			second.Format, padding, storage_size, writer.Status,
		)
	}
}

func test_checksum_domains() {
	var block [tar.BLOCK_SIZE]byte
	signed_minimum := byte(1 << (bits.BIT_COUNT_8_MAXIMUM - 1))
	block[tar.NAME_FIELD_OFFSET] = tar.TYPE_FLAG_FIELD_SIZE
	test_checksum_block(block[:])
	test_fill_bytes(block[:], bits.WORD_8_MAXIMUM)
	test_checksum_block(block[:])
	test_fill_bytes(block[:], signed_minimum)
	test_checksum_block(block[:])
	test_fill_bytes(block[:], signed_minimum-tar.TYPE_FLAG_FIELD_SIZE)
	test_checksum_block(block[:])
	for _, values := range [][tar.OCTAL_BITS_PER_DIGIT]byte{
		{signed_minimum, signed_minimum, bits.WORD_8_MAXIMUM},
		{signed_minimum, signed_minimum},
		{signed_minimum, signed_minimum + tar.TYPE_FLAG_FIELD_SIZE},
		{signed_minimum, signed_minimum + tar.VERSION_FIELD_SIZE},
	} {
		test_fill_bytes(block[:], 0)
		copy(block[:], values[:])
		test_checksum_block(block[:])
	}
}

func test_checksum_block(block []byte) {
	format_octal(
		block[tar.CHECKSUM_FIELD_OFFSET:][:tar.CHECKSUM_FIELD_SIZE],
		0,
	)
	test_reader_archive(block, nil, tar.Header_Storage{})
}

func test_reader_storage_domain(marker int) (status tar.Status) {
	var archive_storage [TEST_ARCHIVE_SIZE]byte
	archive := tar_entry(
		archive_storage[:], 0, tar.FORMAT_USTAR, tar.TYPE_REGULAR, nil, nil,
	)
	header := archive[:tar.BLOCK_SIZE]
	test_fill_field(
		header[tar.USER_NAME_FIELD_OFFSET:][:tar.USER_NAME_FIELD_SIZE],
		TEST_DOMAIN_ZERO, 'u',
	)
	test_fill_field(
		header[tar.GROUP_NAME_FIELD_OFFSET:][:tar.GROUP_NAME_FIELD_SIZE],
		TEST_DOMAIN_ZERO, 'g',
	)
	tar_header_checksum(header)
	archive = archive_footer(archive_storage[:], len(archive))
	name_size := test_domain_size(marker, tar.HEADER_TEXT_SIZE_MAXIMUM)
	user_size := test_domain_size(marker, tar.HEADER_USER_NAME_SIZE_MAXIMUM)
	group_size := test_domain_size(marker, tar.HEADER_GROUP_NAME_SIZE_MAXIMUM)
	return test_reader_archive(archive, nil, tar.Header_Storage{
		Name: make([]byte, name_size), Link_Name: make([]byte, name_size),
		User_Name:  make([]byte, user_size),
		Group_Name: make([]byte, group_size),
	})
}

func test_reader_fixed_text_domain(marker int) {
	var archive_storage [TEST_ARCHIVE_SIZE]byte
	archive := tar_entry(
		archive_storage[:], 0, tar.FORMAT_USTAR, tar.TYPE_REGULAR, nil, nil,
	)
	header := archive[:tar.BLOCK_SIZE]
	test_fill_field(header[tar.NAME_FIELD_OFFSET:][:tar.NAME_FIELD_SIZE], marker, 'n')
	test_fill_field(
		header[tar.LINK_NAME_FIELD_OFFSET:][:tar.LINK_NAME_FIELD_SIZE], marker, 'l',
	)
	test_fill_field(
		header[tar.USER_NAME_FIELD_OFFSET:][:tar.USER_NAME_FIELD_SIZE], marker, 'u',
	)
	test_fill_field(
		header[tar.GROUP_NAME_FIELD_OFFSET:][:tar.GROUP_NAME_FIELD_SIZE], marker, 'g',
	)
	tar_header_checksum(header)
	archive = archive_footer(archive_storage[:], len(archive))
	test_reader_archive(archive, nil, tar.Header_Storage{
		Name:       make([]byte, tar.NAME_FIELD_SIZE),
		Link_Name:  make([]byte, tar.LINK_NAME_FIELD_SIZE),
		User_Name:  make([]byte, tar.USER_NAME_FIELD_SIZE),
		Group_Name: make([]byte, tar.GROUP_NAME_FIELD_SIZE),
	})
}

func test_reader_long_text_domain(t *testing.T) {
	t.Helper()
	value := make([]byte, tar.HEADER_TEXT_SIZE_MAXIMUM)
	test_fill_bytes(value, 'n')
	payload := make([]byte, tar.SPECIAL_FILE_SIZE_MAXIMUM)
	copy(payload, value)
	archive_block_count := tar.GNU_LONG_FIELD_COUNT_MAXIMUM +
		tar.TYPE_FLAG_FIELD_SIZE + tar.ARCHIVE_FOOTER_BLOCK_COUNT
	archive_storage := make(
		[]byte, tar.READER_METADATA_SIZE_MAXIMUM+archive_block_count*tar.BLOCK_SIZE,
	)
	archive := tar_entry(
		archive_storage, 0, tar.FORMAT_GNU, tar.TYPE_GNU_LONG_NAME,
		[]byte(tar.GNU_LONG_HEADER_NAME), payload,
	)
	test_fill_bytes(value, 'l')
	copy(payload, value)
	archive = tar_entry(
		archive_storage, len(archive), tar.FORMAT_GNU, tar.TYPE_GNU_LONG_LINK,
		[]byte(tar.GNU_LONG_HEADER_NAME), payload,
	)
	archive = tar_entry(
		archive_storage, len(archive), tar.FORMAT_GNU, tar.TYPE_REGULAR,
		[]byte{'n'}, nil,
	)
	archive = archive_footer(archive_storage, len(archive))
	test_reader_archive(
		archive, make([]byte, tar.READER_METADATA_SIZE_MAXIMUM), tar.Header_Storage{
			Name:      make([]byte, tar.HEADER_TEXT_SIZE_MAXIMUM),
			Link_Name: make([]byte, tar.HEADER_TEXT_SIZE_MAXIMUM),
		},
	)
	for _, value_size := range [...]int{
		tar.TYPE_FLAG_FIELD_SIZE, tar.ARCHIVE_FOOTER_BLOCK_COUNT,
	} {
		test_reader_gnu_long_value(t, tar.TYPE_GNU_LONG_NAME, value_size, tar.STATUS_OK)
		test_reader_gnu_long_value(t, tar.TYPE_GNU_LONG_LINK, value_size, tar.STATUS_OK)
	}
	test_reader_gnu_long_value(
		t, tar.TYPE_GNU_LONG_NAME, tar.SPECIAL_FILE_SIZE_MAXIMUM,
		tar.STATUS_FIELD_TOO_LONG,
	)
	test_reader_gnu_long_value(
		t, tar.TYPE_GNU_LONG_LINK, tar.SPECIAL_FILE_SIZE_MAXIMUM,
		tar.STATUS_FIELD_TOO_LONG,
	)
}

func test_reader_gnu_long_value(
	t *testing.T, type_flag tar.Type_Flag, value_size int, wanted tar.Status,
) {
	t.Helper()
	value := make([]byte, value_size)
	test_fill_bytes(value, 'v')
	archive_size := tar.BLOCK_SIZE + padded_size(value_size) +
		tar.BLOCK_SIZE + tar.ARCHIVE_FOOTER_SIZE
	archive_storage := make([]byte, archive_size)
	archive := tar_entry(
		archive_storage, 0, tar.FORMAT_GNU, type_flag,
		[]byte(tar.GNU_LONG_HEADER_NAME), value,
	)
	archive = tar_entry(
		archive_storage, len(archive), tar.FORMAT_GNU, tar.TYPE_REGULAR,
		[]byte("entry"), nil,
	)
	archive = archive_footer(archive_storage, len(archive))
	status := test_reader_archive(
		archive, make([]byte, tar.READER_METADATA_SIZE_MAXIMUM),
		tar.Header_Storage{
			Name:      make([]byte, tar.HEADER_TEXT_SIZE_MAXIMUM),
			Link_Name: make([]byte, tar.HEADER_TEXT_SIZE_MAXIMUM),
		},
	)
	if status != wanted {
		t.Fatalf(
			"GNU long type %q size %d status = %v; want %v",
			type_flag, value_size, status, wanted,
		)
	}
}

func test_reader_pax_text_domain(t *testing.T, key []byte, value_size int) {
	t.Helper()
	value := make([]byte, value_size)
	test_fill_bytes(value, 'v')
	records := make([]byte, tar.SPECIAL_FILE_SIZE_MAXIMUM)
	record_count := pax_record_into(records, key, value)
	archive_storage := make([]byte, tar.SPECIAL_FILE_SIZE_MAXIMUM+4*tar.BLOCK_SIZE)
	archive := tar_entry(
		archive_storage, 0, tar.FORMAT_USTAR, tar.TYPE_PAX_LOCAL,
		[]byte(tar.PAX_HEADER_NAME), records[:record_count],
	)
	archive = tar_entry(
		archive_storage, len(archive), tar.FORMAT_USTAR, tar.TYPE_REGULAR,
		[]byte{'n'}, nil,
	)
	archive = archive_footer(archive_storage, len(archive))
	storage := tar.Header_Storage{
		Name:       make([]byte, tar.HEADER_TEXT_SIZE_MAXIMUM),
		Link_Name:  make([]byte, tar.HEADER_TEXT_SIZE_MAXIMUM),
		User_Name:  make([]byte, tar.HEADER_USER_NAME_SIZE_MAXIMUM),
		Group_Name: make([]byte, tar.HEADER_GROUP_NAME_SIZE_MAXIMUM),
	}
	status := test_reader_archive(
		archive, make([]byte, tar.SPECIAL_FILE_SIZE_MAXIMUM), storage,
	)
	if status != tar.STATUS_OK {
		t.Fatalf("PAX text key %q size %d: status %d", key, value_size, status)
	}
}

func test_reader_pax_text_boundaries(t *testing.T) {
	t.Helper()
	for marker := TEST_DOMAIN_ONE; marker <= TEST_DOMAIN_TWO; marker++ {
		value_size := test_domain_size(marker, tar.HEADER_TEXT_SIZE_MAXIMUM)
		for _, key := range [][]byte{
			[]byte(tar.PAX_PATH_KEY), []byte(tar.PAX_LINK_PATH_KEY),
			[]byte(tar.PAX_USER_NAME_KEY), []byte(tar.PAX_GROUP_NAME_KEY),
		} {
			test_reader_pax_text_domain(t, key, value_size)
		}
	}
	for marker := TEST_DOMAIN_ONE; marker <= TEST_DOMAIN_TWO; marker++ {
		test_reader_pax_prefixed_name(t, marker)
	}
	test_reader_pax_prefixed_name(t, TEST_DOMAIN_MAXIMUM)
	test_reader_pax_after_gnu_long(t)
}

func test_reader_pax_prefixed_name(t *testing.T, marker int) {
	t.Helper()
	var records [TEST_FIELD_SIZE]byte
	record_count := pax_record_into(records[:], []byte("unknown"), []byte("value"))
	var archive_storage [TEST_ARCHIVE_SIZE]byte
	archive := tar_entry(
		archive_storage[:], 0, tar.FORMAT_USTAR, tar.TYPE_PAX_LOCAL,
		[]byte(tar.PAX_HEADER_NAME), records[:record_count],
	)
	header_start_count := len(archive)
	archive = tar_entry(
		archive_storage[:], header_start_count, tar.FORMAT_USTAR, tar.TYPE_REGULAR,
		[]byte{'n'}, nil,
	)
	header := archive[header_start_count:][:tar.BLOCK_SIZE]
	test_fill_field(header[tar.PREFIX_FIELD_OFFSET:][:tar.PREFIX_FIELD_SIZE], marker, 'p')
	tar_header_checksum(header)
	archive = archive_footer(archive_storage[:], len(archive))
	status := test_reader_archive(
		archive, make([]byte, TEST_FIELD_SIZE),
		tar.Header_Storage{
			Name:       make([]byte, tar.HEADER_TEXT_SIZE_MAXIMUM),
			User_Name:  make([]byte, tar.USER_NAME_FIELD_SIZE),
			Group_Name: make([]byte, tar.GROUP_NAME_FIELD_SIZE),
		},
	)
	if status != tar.STATUS_OK {
		t.Fatalf("PAX prefixed marker %d: status %d", marker, status)
	}
}

func test_reader_pax_after_gnu_long(t *testing.T) {
	t.Helper()
	long_name := make([]byte, tar.HEADER_TEXT_SIZE_MAXIMUM)
	long_link := make([]byte, tar.HEADER_TEXT_SIZE_MAXIMUM)
	test_fill_bytes(long_name, 'n')
	test_fill_bytes(long_link, 'l')
	metadata := make([]byte, tar.SPECIAL_FILE_SIZE_MAXIMUM)
	archive_storage := make([]byte, tar.READER_METADATA_SIZE_MAXIMUM+TEST_ARCHIVE_SIZE)
	archive := tar_entry(
		archive_storage, 0, tar.FORMAT_GNU, tar.TYPE_GNU_LONG_NAME,
		[]byte(tar.GNU_LONG_HEADER_NAME), append_nul(metadata, long_name),
	)
	archive = tar_entry(
		archive_storage, len(archive), tar.FORMAT_GNU, tar.TYPE_GNU_LONG_LINK,
		[]byte(tar.GNU_LONG_HEADER_NAME), append_nul(metadata, long_link),
	)
	record_count := pax_record_into(metadata, []byte("unknown"), []byte("value"))
	archive = tar_entry(
		archive_storage, len(archive), tar.FORMAT_USTAR, tar.TYPE_PAX_LOCAL,
		[]byte(tar.PAX_HEADER_NAME), metadata[:record_count],
	)
	archive = tar_entry(
		archive_storage, len(archive), tar.FORMAT_USTAR, tar.TYPE_REGULAR,
		[]byte{'n'}, nil,
	)
	archive = archive_footer(archive_storage, len(archive))
	status := test_reader_archive(
		archive, make([]byte, tar.READER_METADATA_SIZE_MAXIMUM), tar.Header_Storage{
			Name: make([]byte, len(long_name)), Link_Name: make([]byte, len(long_link)),
			User_Name:  make([]byte, tar.USER_NAME_FIELD_SIZE),
			Group_Name: make([]byte, tar.GROUP_NAME_FIELD_SIZE),
		},
	)
	if status != tar.STATUS_OK {
		t.Fatalf("PAX after GNU long fields: status %d", status)
	}
}

func test_reader_archive(
	archive []byte, metadata []byte, storage tar.Header_Storage,
) (status tar.Status) {
	var block [tar.BLOCK_SIZE]byte
	memory := nbio.Stream_Memory{Memory: archive}
	var reader tar.Reader
	tar.Reader_Init(
		&reader, nbio.Memory_To_Stream(&memory),
		tar.Reader_Storage{Block: block[:], Metadata: metadata},
	)
	tar.Reader_Next(
		&reader, &reader.Completion, storage, test_domain_callback,
	)
	return reader.Status
}

func test_writer_basic_domain(t *testing.T, marker int) {
	t.Helper()
	for field := TEST_HEADER_TYPE_FLAG; field <= TEST_HEADER_DEVICE_MINOR; field++ {
		broad_header := test_valid_header_field_domain(field, marker, tar.FORMAT_USTAR)
		broad_workspace := make([]byte, tar.WRITER_STORAGE_SIZE_MAXIMUM)
		var broad_archive [TEST_ARCHIVE_SIZE]byte
		broad_memory := nbio.Stream_Memory{Memory: broad_archive[:]}
		var broad_writer tar.Writer
		tar.Writer_Init(
			&broad_writer, nbio.Memory_To_Stream(&broad_memory), broad_workspace,
		)
		tar.Writer_Write_Header(
			&broad_writer, &broad_writer.Completion, &broad_header,
			test_domain_callback,
		)
		header, present := test_writer_basic_header(field, marker)
		if !present {
			continue
		}
		var archive [TEST_ARCHIVE_SIZE]byte
		var workspace [TEST_WRITER_WORKSPACE_BLOCK_COUNT * tar.BLOCK_SIZE]byte
		memory := nbio.Stream_Memory{Memory: archive[:]}
		var writer tar.Writer
		tar.Writer_Init(&writer, nbio.Memory_To_Stream(&memory), workspace[:])
		tar.Writer_Write_Header(
			&writer, &writer.Completion, &header, test_domain_callback,
		)
		if writer.Status != tar.STATUS_OK {
			t.Fatalf(
				"basic domain field %d marker %d: status %d", field, marker,
				writer.Status,
			)
		}
	}
}

func test_writer_basic_header(
	field int, marker int,
) (header tar.Header_Unvalidated, present bool) {
	header = tar.Header_Unvalidated{
		Format: tar.Format_Unvalidated(tar.FORMAT_USTAR), Name: []byte{'n'},
	}
	small := int64(test_domain_size(marker, int(tar.SMALL_OCTAL_MAXIMUM)))
	switch field {
	case TEST_HEADER_TYPE_FLAG:
		header.Type_Flag = tar.Type_Flag(test_domain_word_8(marker))
	case TEST_HEADER_NAME:
		size := test_domain_size(marker, tar.USTAR_PATH_SIZE_MAXIMUM)
		if size == 0 {
			size = tar.USTAR_PATH_SIZE_MINIMUM
		}
		header.Name = make([]byte, size)
		test_fill_bytes(header.Name, 'n')
		if size == tar.USTAR_PATH_SIZE_MAXIMUM {
			header.Name[tar.PREFIX_FIELD_SIZE] = '/'
		}
	case TEST_HEADER_LINK_NAME:
		header.Link_Name = make(
			[]byte, test_domain_size(marker, tar.LINK_NAME_FIELD_SIZE),
		)
		test_fill_bytes(header.Link_Name, 'l')
	case TEST_HEADER_SIZE:
		header.Size = tar.Entry_Size_Unvalidated(
			test_domain_size(marker, tar.ARCHIVE_SIZE_MAXIMUM),
		)
	case TEST_HEADER_MODE:
		header.Mode = tar.File_Mode(small)
	case TEST_HEADER_USER_IDENTIFIER:
		header.User_Identifier = tar.User_Identifier(small)
	case TEST_HEADER_GROUP_IDENTIFIER:
		header.Group_Identifier = tar.Group_Identifier(small)
	case TEST_HEADER_USER_NAME:
		header.User_Name = make(
			[]byte, test_domain_size(marker, tar.USER_NAME_FIELD_SIZE),
		)
		test_fill_bytes(header.User_Name, 'u')
	case TEST_HEADER_GROUP_NAME:
		header.Group_Name = make(
			[]byte, test_domain_size(marker, tar.GROUP_NAME_FIELD_SIZE),
		)
		test_fill_bytes(header.Group_Name, 'g')
	case TEST_HEADER_MODIFICATION_TIME:
		header.Modification_Time = tar.Timestamp_Unvalidated{
			Seconds: tar.Integer(test_domain_size(
				marker, int(tar.LARGE_OCTAL_MAXIMUM),
			)), Set: true,
		}
	case TEST_HEADER_DEVICE_MAJOR:
		header.Device_Major = tar.Device_Major(small)
	case TEST_HEADER_DEVICE_MINOR:
		header.Device_Minor = tar.Device_Minor(small)
	default:
		return header, false
	}
	return header, true
}

func test_writer_gnu_domain(t *testing.T, marker int) {
	t.Helper()
	for field := TEST_HEADER_MODE; field <= TEST_HEADER_DEVICE_MINOR; field++ {
		header := tar.Header_Unvalidated{
			Format: tar.Format_Unvalidated(tar.FORMAT_GNU), Name: []byte{'n'},
		}
		integer := test_domain_small_integer(marker)
		switch field {
		case TEST_HEADER_MODE:
			header.Mode = tar.File_Mode(integer)
		case TEST_HEADER_USER_IDENTIFIER:
			header.User_Identifier = tar.User_Identifier(integer)
		case TEST_HEADER_GROUP_IDENTIFIER:
			header.Group_Identifier = tar.Group_Identifier(integer)
		case TEST_HEADER_USER_NAME:
			header.User_Name = make(
				[]byte, test_domain_size(marker, tar.USER_NAME_FIELD_SIZE),
			)
			test_fill_bytes(header.User_Name, 'u')
		case TEST_HEADER_GROUP_NAME:
			header.Group_Name = make(
				[]byte, test_domain_size(marker, tar.GROUP_NAME_FIELD_SIZE),
			)
			test_fill_bytes(header.Group_Name, 'g')
		case TEST_HEADER_DEVICE_MAJOR:
			header.Device_Major = tar.Device_Major(integer)
		case TEST_HEADER_DEVICE_MINOR:
			header.Device_Minor = tar.Device_Minor(integer)
		default:
			continue
		}
		workspace := make([]byte, tar.WRITER_STORAGE_SIZE_MAXIMUM)
		var archive [TEST_ARCHIVE_SIZE]byte
		memory := nbio.Stream_Memory{Memory: archive[:]}
		var writer tar.Writer
		tar.Writer_Init(&writer, nbio.Memory_To_Stream(&memory), workspace)
		tar.Writer_Write_Header(
			&writer, &writer.Completion, &header, test_domain_callback,
		)
		if writer.Status != tar.STATUS_OK {
			t.Fatalf(
				"GNU domain field %d marker %d: status %d", field, marker,
				writer.Status,
			)
		}
	}
}

func test_writer_pax_domain(t *testing.T, marker int) {
	t.Helper()
	for field := TEST_HEADER_NAME; field <= TEST_HEADER_PAX_RECORDS; field++ {
		header, present := test_writer_pax_header(field, marker)
		if !present {
			continue
		}
		if field != TEST_HEADER_MODIFICATION_TIME {
			if field != TEST_HEADER_PAX_RECORDS {
				header.Modification_Time.Set = true
			}
		}
		if field != TEST_HEADER_PAX_RECORDS {
			workspace := make([]byte, tar.WRITER_STORAGE_SIZE_MAXIMUM)
			archive := make([]byte, tar.WRITER_STORAGE_SIZE_MAXIMUM)
			memory := nbio.Stream_Memory{Memory: archive}
			var writer tar.Writer
			tar.Writer_Init(&writer, nbio.Memory_To_Stream(&memory), workspace)
			tar.Writer_Write_Header(
				&writer, &writer.Completion, &header, test_domain_callback,
			)
			if writer.Status != tar.STATUS_OK {
				t.Fatalf(
					"PAX plain domain field %d marker %d: status %d",
					field, marker, writer.Status,
				)
			}
			header.PAX_Records = test_writer_pax_records(TEST_DOMAIN_ONE)
		}
		workspace := make([]byte, tar.WRITER_STORAGE_SIZE_MAXIMUM)
		archive := make([]byte, tar.WRITER_STORAGE_SIZE_MAXIMUM)
		memory := nbio.Stream_Memory{Memory: archive}
		var writer tar.Writer
		tar.Writer_Init(&writer, nbio.Memory_To_Stream(&memory), workspace)
		tar.Writer_Write_Header(
			&writer, &writer.Completion, &header, test_domain_callback,
		)
		if writer.Status != tar.STATUS_OK {
			t.Fatalf(
				"PAX domain field %d marker %d: status %d", field, marker,
				writer.Status,
			)
		}
	}
}

func test_writer_pax_header(
	field int, marker int,
) (header tar.Header_Unvalidated, present bool) {
	header = tar.Header_Unvalidated{
		Format: tar.Format_Unvalidated(tar.FORMAT_PAX), Name: []byte{'n'},
	}
	switch field {
	case TEST_HEADER_NAME:
		size := test_domain_size(marker, tar.PAX_NAME_SIZE_MAXIMUM)
		if size == 0 {
			size = tar.USTAR_PATH_SIZE_MINIMUM
		}
		header.Name = make([]byte, size)
		test_fill_bytes(header.Name, 'n')
	case TEST_HEADER_LINK_NAME:
		header.Link_Name = make(
			[]byte, test_domain_size(marker, tar.PAX_LINK_NAME_SIZE_MAXIMUM),
		)
		test_fill_bytes(header.Link_Name, 'l')
	case TEST_HEADER_MODE:
		header.Mode = tar.File_Mode(test_domain_size(
			marker, int(tar.SMALL_OCTAL_MAXIMUM),
		))
	case TEST_HEADER_USER_NAME:
		header.User_Name = make(
			[]byte, test_domain_size(marker, tar.PAX_USER_NAME_SIZE_MAXIMUM),
		)
		test_fill_bytes(header.User_Name, 'u')
	case TEST_HEADER_GROUP_NAME:
		header.Group_Name = make(
			[]byte, test_domain_size(marker, tar.PAX_GROUP_NAME_SIZE_MAXIMUM),
		)
		test_fill_bytes(header.Group_Name, 'g')
	case TEST_HEADER_MODIFICATION_TIME:
		header.Modification_Time = tar.Timestamp_Unvalidated{
			Seconds: tar.Integer(test_domain_integer_64(marker)),
			Nanoseconds: tar.Nanosecond_Count_Unvalidated(test_domain_size(
				marker, int(tar.TIMESTAMP_NANOSECOND_MAXIMUM),
			)),
			Set: true,
		}
	case TEST_HEADER_ACCESS_TIME:
		header.Access_Time = tar.Access_Timestamp_Unvalidated{
			Seconds: tar.Access_Seconds(test_domain_integer_64(marker)),
			Nanoseconds: tar.Access_Nanosecond_Count_Unvalidated(test_domain_size(
				marker, int(tar.TIMESTAMP_NANOSECOND_MAXIMUM),
			)),
			Set: true,
		}
	case TEST_HEADER_CHANGE_TIME:
		header.Change_Time = tar.Change_Timestamp_Unvalidated{
			Seconds: tar.Change_Seconds(test_domain_integer_64(marker)),
			Nanoseconds: tar.Change_Nanosecond_Count_Unvalidated(test_domain_size(
				marker, int(tar.TIMESTAMP_NANOSECOND_MAXIMUM),
			)),
			Set: true,
		}
	case TEST_HEADER_DEVICE_MAJOR:
		header.Device_Major = tar.Device_Major(test_domain_size(
			marker, int(tar.SMALL_OCTAL_MAXIMUM),
		))
	case TEST_HEADER_DEVICE_MINOR:
		header.Device_Minor = tar.Device_Minor(test_domain_size(
			marker, int(tar.SMALL_OCTAL_MAXIMUM),
		))
	case TEST_HEADER_PAX_RECORDS:
		header.PAX_Records = test_writer_pax_records(marker)
	default:
		return header, false
	}
	return header, true
}

func test_writer_pax_records(marker int) (records []byte) {
	return test_pax_records_domain(marker, tar.PAX_HEADER_RECORDS_SIZE_MAXIMUM)
}

func test_pax_records_domain(marker int, maximum int) (records []byte) {
	size := 0
	switch marker {
	case TEST_DOMAIN_ONE:
		size = tar.PAX_RECORD_SIZE_MINIMUM
	case TEST_DOMAIN_TWO:
		size = tar.PAX_RECORD_SIZE_MINIMUM + tar.TYPE_FLAG_FIELD_SIZE
	case TEST_DOMAIN_MAXIMUM:
		size = maximum
	}
	if size == 0 {
		return nil
	}
	records = make([]byte, size)
	value_size := size - tar.PAX_RECORD_FIXED_SIZE_MAXIMUM - tar.TYPE_FLAG_FIELD_SIZE
	if size < tar.PAX_RECORD_COUNT_TEXT_SIZE_MAXIMUM+tar.PAX_RECORD_FIXED_SIZE_MINIMUM {
		value_size = size - tar.PAX_RECORD_SIZE_MINIMUM
	}
	if pax_record_into(records, []byte{'k'}, make([]byte, value_size)) != len(records) {
		return nil
	}
	return records
}

func test_reader_wire_domain(marker int) {
	var pax_storage [TEST_FIELD_SIZE]byte
	pax_count := test_pax_domain_records(pax_storage[:], marker)
	var archive_storage [TEST_ARCHIVE_SIZE]byte
	archive := tar_entry(
		archive_storage[:], 0, tar.FORMAT_USTAR, tar.TYPE_PAX_LOCAL,
		[]byte(tar.PAX_HEADER_NAME), pax_storage[:pax_count],
	)
	header_start_count := len(archive)
	archive = tar_entry(
		archive_storage[:], header_start_count, tar.FORMAT_USTAR,
		tar.Type_Flag(test_domain_word_8(marker)), nil, nil,
	)
	header := archive[header_start_count:][:tar.BLOCK_SIZE]
	test_fill_field(
		header[tar.NAME_FIELD_OFFSET:][:tar.NAME_FIELD_SIZE], marker, 'n',
	)
	test_fill_field(
		header[tar.LINK_NAME_FIELD_OFFSET:][:tar.LINK_NAME_FIELD_SIZE], marker, 'l',
	)
	test_fill_field(
		header[tar.USER_NAME_FIELD_OFFSET:][:tar.USER_NAME_FIELD_SIZE], marker, 'u',
	)
	test_fill_field(
		header[tar.GROUP_NAME_FIELD_OFFSET:][:tar.GROUP_NAME_FIELD_SIZE], marker, 'g',
	)
	test_fill_field(
		header[tar.PREFIX_FIELD_OFFSET:][:tar.PREFIX_FIELD_SIZE], marker, 'p',
	)
	small := test_domain_small_integer(marker)
	full := test_domain_integer_64(marker)
	test_format_base_256(
		header[tar.MODE_FIELD_OFFSET:][:tar.MODE_FIELD_SIZE], small,
	)
	test_format_base_256(
		header[tar.USER_IDENTIFIER_FIELD_OFFSET:][:tar.USER_IDENTIFIER_FIELD_SIZE],
		small,
	)
	test_format_base_256(
		header[tar.GROUP_IDENTIFIER_FIELD_OFFSET:][:tar.GROUP_IDENTIFIER_FIELD_SIZE],
		small,
	)
	test_format_base_256(
		header[tar.ENTRY_SIZE_FIELD_OFFSET:][:tar.ENTRY_SIZE_FIELD_SIZE],
		int64(test_domain_size(marker, tar.ARCHIVE_SIZE_MAXIMUM)),
	)
	test_format_base_256(
		header[tar.TIMESTAMP_FIELD_OFFSET:][:tar.TIMESTAMP_FIELD_SIZE], full,
	)
	test_format_base_256(
		header[tar.DEVICE_MAJOR_FIELD_OFFSET:][:tar.DEVICE_MAJOR_FIELD_SIZE], small,
	)
	test_format_base_256(
		header[tar.DEVICE_MINOR_FIELD_OFFSET:][:tar.DEVICE_MINOR_FIELD_SIZE], small,
	)
	tar_header_checksum(header)
	archive = archive_footer(archive_storage[:], len(archive))
	var fixture reader_fixture
	reader_fixture_init(&fixture, archive)
	reader_fixture_next(&fixture)
	test_reader_gnu_wire_domain(marker)
	test_reader_gnu_prefix_wire_domain(marker)
	test_reader_star_wire_domain(marker)
}

func test_reader_gnu_wire_domain(marker int) {
	var archive_storage [TEST_ARCHIVE_SIZE]byte
	var pax_storage [TEST_FIELD_SIZE]byte
	pax_count := pax_record_into(pax_storage[:], []byte("unknown"), []byte("value"))
	archive := tar_entry(
		archive_storage[:], 0, tar.FORMAT_USTAR, tar.TYPE_PAX_LOCAL,
		[]byte(tar.PAX_HEADER_NAME), pax_storage[:pax_count],
	)
	header_start_count := len(archive)
	archive = tar_entry(
		archive_storage[:], header_start_count, tar.FORMAT_GNU, tar.TYPE_REGULAR,
		[]byte("gnu"), nil,
	)
	header := archive[header_start_count:][:tar.BLOCK_SIZE]
	test_fill_field(
		header[tar.NAME_FIELD_OFFSET:][:tar.NAME_FIELD_SIZE], marker, 'n',
	)
	seconds := test_domain_integer_64(marker)
	test_format_base_256(
		header[tar.GNU_ACCESS_TIMESTAMP_FIELD_OFFSET:][:tar.TIMESTAMP_FIELD_SIZE],
		seconds,
	)
	test_format_base_256(
		header[tar.GNU_CHANGE_TIMESTAMP_FIELD_OFFSET:][:tar.TIMESTAMP_FIELD_SIZE],
		seconds,
	)
	tar_header_checksum(header)
	archive = archive_footer(archive_storage[:], len(archive))
	var fixture reader_fixture
	reader_fixture_init(&fixture, archive)
	reader_fixture_next(&fixture)
}

func test_reader_gnu_prefix_wire_domain(marker int) {
	var archive_storage [TEST_ARCHIVE_SIZE]byte
	archive := tar_entry(
		archive_storage[:], 0, tar.FORMAT_GNU, tar.TYPE_REGULAR,
		[]byte("gnu-prefix"), nil,
	)
	header := archive[:tar.BLOCK_SIZE]
	test_fill_field(
		header[tar.NAME_FIELD_OFFSET:][:tar.NAME_FIELD_SIZE], marker, 'n',
	)
	test_fill_field(
		header[tar.PREFIX_FIELD_OFFSET:][:tar.PREFIX_FIELD_SIZE], marker, 'p',
	)
	if marker == TEST_DOMAIN_ZERO {
		header[tar.GNU_ACCESS_TIMESTAMP_FIELD_OFFSET] = 'x'
	}
	tar_header_checksum(header)
	archive = archive_footer(archive_storage[:], len(archive))
	var fixture reader_fixture
	reader_fixture_init(&fixture, archive)
	reader_fixture_next(&fixture)
}

func test_reader_star_wire_domain(marker int) {
	var archive_storage [TEST_ARCHIVE_SIZE]byte
	archive := tar_entry(
		archive_storage[:], 0, tar.FORMAT_USTAR, tar.TYPE_REGULAR,
		[]byte("star"), nil,
	)
	header := archive[:tar.BLOCK_SIZE]
	test_fill_field(
		header[tar.NAME_FIELD_OFFSET:][:tar.NAME_FIELD_SIZE], marker, 'n',
	)
	test_fill_field(
		header[tar.PREFIX_FIELD_OFFSET:][:tar.STAR_PREFIX_FIELD_SIZE], marker, 'p',
	)
	seconds := test_domain_integer_64(marker)
	test_format_base_256(
		header[tar.STAR_ACCESS_TIMESTAMP_FIELD_OFFSET:][:tar.TIMESTAMP_FIELD_SIZE],
		seconds,
	)
	test_format_base_256(
		header[tar.STAR_CHANGE_TIMESTAMP_FIELD_OFFSET:][:tar.TIMESTAMP_FIELD_SIZE],
		seconds,
	)
	copy(
		header[tar.STAR_TRAILER_FIELD_OFFSET:][:tar.STAR_TRAILER_FIELD_SIZE],
		tar.STAR_TRAILER,
	)
	tar_header_checksum(header)
	archive = archive_footer(archive_storage[:], len(archive))
	var fixture reader_fixture
	reader_fixture_init(&fixture, archive)
	reader_fixture_next(&fixture)
}

func test_pax_domain_records(storage []byte, marker int) (count int) {
	var integer_text [strconv.INTEGER_TEXT_SIZE_MAXIMUM]byte
	var timestamp_text [TEST_TIMESTAMP_TEXT_SIZE_MAXIMUM]byte
	count += pax_record_into(
		storage[count:], []byte(tar.PAX_PATH_KEY), []byte("path"),
	)
	count += pax_record_into(
		storage[count:], []byte(tar.PAX_LINK_PATH_KEY), []byte("link"),
	)
	count += pax_record_into(
		storage[count:], []byte(tar.PAX_USER_NAME_KEY), []byte("user"),
	)
	count += pax_record_into(
		storage[count:], []byte(tar.PAX_GROUP_NAME_KEY), []byte("group"),
	)
	count += pax_record_into(
		storage[count:], []byte(tar.PAX_SIZE_KEY), test_integer_text(
			integer_text[:], int64(test_domain_size(marker, tar.ARCHIVE_SIZE_MAXIMUM)),
		),
	)
	count += pax_record_into(
		storage[count:], []byte(tar.PAX_USER_IDENTIFIER_KEY), test_integer_text(
			integer_text[:], test_domain_integer_64(marker),
		),
	)
	count += pax_record_into(
		storage[count:], []byte(tar.PAX_GROUP_IDENTIFIER_KEY), test_integer_text(
			integer_text[:], test_domain_integer_64(marker),
		),
	)
	count += pax_record_into(
		storage[count:], []byte(tar.PAX_MODIFICATION_TIME_KEY),
		test_timestamp_text(timestamp_text[:], marker),
	)
	count += pax_record_into(
		storage[count:], []byte(tar.PAX_ACCESS_TIME_KEY),
		test_timestamp_text(timestamp_text[:], marker),
	)
	count += pax_record_into(
		storage[count:], []byte(tar.PAX_CHANGE_TIME_KEY),
		test_timestamp_text(timestamp_text[:], marker),
	)
	count += pax_record_into(
		storage[count:], []byte("unknown"), []byte("value"),
	)
	return count
}

func test_integer_text(storage []byte, value int64) (text []byte) {
	count := strconv.Format_Integer_Into(
		strconv.Buffer(storage), strconv.Signed_Integer(value),
		strconv.DECIMAL_BASE,
	)
	return storage[:count]
}

func test_timestamp_text(storage []byte, marker int) (text []byte) {
	seconds := test_domain_integer_64(marker)
	text = test_integer_text(storage, seconds)
	nanoseconds := test_domain_size(marker, int(tar.TIMESTAMP_NANOSECOND_MAXIMUM))
	if seconds < 0 {
		nanoseconds = 0
	}
	if nanoseconds == 0 {
		return text
	}
	count := len(text)
	storage[count] = '.'
	count += tar.TIMESTAMP_FRACTION_SEPARATOR_SIZE
	fraction_end := count + tar.TIMESTAMP_FRACTION_DIGIT_COUNT_MAXIMUM
	for position := fraction_end - 1; position >= count; position-- {
		storage[position] = byte(nanoseconds%tar.DECIMAL_BASE) + '0'
		nanoseconds /= tar.DECIMAL_BASE
	}
	return storage[:fraction_end]
}

func test_fill_field(field []byte, marker int, value byte) {
	count := test_domain_size(marker, len(field))
	for position_index := range field {
		field[position_index] = 0
	}
	for position_index := 0; position_index < count; position_index++ {
		field[position_index] = value
	}
}

func test_format_base_256(field []byte, value int64) {
	for position := len(field) - 1; position >= 0; position-- {
		field[position] = byte(value)
		value >>= bits.BIT_COUNT_8_MAXIMUM
	}
	field[0] |= tar.BASE_256_MARKER_MASK
}

func test_header_domain(field int, marker int) {
	header := test_header_unvalidated_domain(field, marker)
	tar.Header_Validate(&header)
	var writer tar.Writer
	tar.Writer_Write_Header(
		&writer, &writer.Completion, &header, test_domain_callback,
	)
	var archive [TEST_ARCHIVE_SIZE]byte
	var workspace [TEST_WRITER_WORKSPACE_BLOCK_COUNT * tar.BLOCK_SIZE]byte
	memory := nbio.Stream_Memory{Memory: archive[:]}
	tar.Writer_Init(
		&writer, nbio.Memory_To_Stream(&memory),
		workspace[:],
	)
	tar.Writer_Write_Header(
		&writer, &writer.Completion, &header, test_domain_callback,
	)
}

func test_writer_header_field_domain(field int, marker int, format tar.Format) {
	header := test_valid_header_field_domain(field, marker, format)
	if format == tar.FORMAT_PAX {
		if field != TEST_HEADER_PAX_RECORDS {
			header.PAX_Records = test_writer_pax_records(TEST_DOMAIN_ONE)
		}
		if field != TEST_HEADER_MODIFICATION_TIME {
			if field != TEST_HEADER_PAX_RECORDS {
				header.Modification_Time.Set = true
			}
		}
	}
	workspace := make([]byte, tar.WRITER_STORAGE_SIZE_MAXIMUM)
	archive := make([]byte, TEST_ARCHIVE_SIZE)
	memory := nbio.Stream_Memory{Memory: archive}
	var writer tar.Writer
	tar.Writer_Init(
		&writer, nbio.Memory_To_Stream(&memory),
		workspace,
	)
	tar.Writer_Write_Header(
		&writer, &writer.Completion, &header, test_domain_callback,
	)
}

func test_valid_header_field_domain(
	field int, marker int, format tar.Format,
) (header tar.Header_Unvalidated) {
	header = tar.Header_Unvalidated{
		Format: tar.Format_Unvalidated(format), Type_Flag: tar.TYPE_REGULAR,
		Name: []byte("x"),
	}
	text_size := test_domain_size(marker, tar.HEADER_TEXT_SIZE_MAXIMUM)
	integer := test_domain_integer_64(marker)
	switch field {
	case TEST_HEADER_TYPE_FLAG:
		header.Type_Flag = tar.Type_Flag(test_domain_word_8(marker))
	case TEST_HEADER_NAME:
		header.Name = make([]byte, text_size)
		test_fill_bytes(header.Name, 'n')
	case TEST_HEADER_LINK_NAME:
		header.Link_Name = make([]byte, text_size)
		test_fill_bytes(header.Link_Name, 'l')
	case TEST_HEADER_SIZE:
		header.Size = tar.Entry_Size_Unvalidated(
			test_domain_size(marker, tar.ARCHIVE_SIZE_MAXIMUM),
		)
	case TEST_HEADER_MODE:
		header.Mode = tar.File_Mode(integer)
	case TEST_HEADER_USER_IDENTIFIER:
		header.User_Identifier = tar.User_Identifier(integer)
	case TEST_HEADER_GROUP_IDENTIFIER:
		header.Group_Identifier = tar.Group_Identifier(integer)
	case TEST_HEADER_USER_NAME:
		header.User_Name = make(
			[]byte, test_domain_size(marker, tar.HEADER_USER_NAME_SIZE_MAXIMUM),
		)
		test_fill_bytes(header.User_Name, 'u')
	case TEST_HEADER_GROUP_NAME:
		header.Group_Name = make(
			[]byte, test_domain_size(marker, tar.HEADER_GROUP_NAME_SIZE_MAXIMUM),
		)
		test_fill_bytes(header.Group_Name, 'g')
	case TEST_HEADER_MODIFICATION_TIME:
		header.Modification_Time = test_domain_timestamp_valid(marker)
	case TEST_HEADER_ACCESS_TIME:
		header.Access_Time = test_domain_access_time_valid(marker)
	case TEST_HEADER_CHANGE_TIME:
		header.Change_Time = test_domain_change_time_valid(marker)
	case TEST_HEADER_DEVICE_MAJOR:
		header.Device_Major = tar.Device_Major(integer)
	case TEST_HEADER_DEVICE_MINOR:
		header.Device_Minor = tar.Device_Minor(integer)
	case TEST_HEADER_PAX_RECORDS:
		header.PAX_Records = test_pax_records_domain(
			marker, tar.SPECIAL_FILE_SIZE_MAXIMUM,
		)
	}
	return header
}

func test_domain_timestamp_valid(marker int) (timestamp tar.Timestamp_Unvalidated) {
	return tar.Timestamp_Unvalidated{
		Seconds: tar.Integer(test_domain_integer_64(marker)),
		Nanoseconds: tar.Nanosecond_Count_Unvalidated(test_domain_size(
			marker, int(tar.TIMESTAMP_NANOSECOND_MAXIMUM),
		)),
		Set: bytes.Boolean(test_domain_boolean(marker)),
	}
}

func test_domain_access_time_valid(
	marker int,
) (timestamp tar.Access_Timestamp_Unvalidated) {
	return tar.Access_Timestamp_Unvalidated{
		Seconds: tar.Access_Seconds(test_domain_integer_64(marker)),
		Nanoseconds: tar.Access_Nanosecond_Count_Unvalidated(test_domain_size(
			marker, int(tar.TIMESTAMP_NANOSECOND_MAXIMUM),
		)),
		Set: tar.Access_Time_Set(test_domain_boolean(marker)),
	}
}

func test_domain_change_time_valid(
	marker int,
) (timestamp tar.Change_Timestamp_Unvalidated) {
	return tar.Change_Timestamp_Unvalidated{
		Seconds: tar.Change_Seconds(test_domain_integer_64(marker)),
		Nanoseconds: tar.Change_Nanosecond_Count_Unvalidated(test_domain_size(
			marker, int(tar.TIMESTAMP_NANOSECOND_MAXIMUM),
		)),
		Set: tar.Change_Time_Set(test_domain_boolean(marker)),
	}
}

func test_header_unvalidated_domain(
	field int, marker int,
) (header tar.Header_Unvalidated) {
	header = tar.Header_Unvalidated{
		Format:    tar.Format_Unvalidated(tar.FORMAT_PAX),
		Type_Flag: tar.TYPE_REGULAR,
		Name:      []byte("x"),
	}
	size := test_domain_size(marker, tar.SPECIAL_FILE_SIZE_UNVALIDATED_MAXIMUM)
	text_size := test_domain_size(marker, tar.HEADER_TEXT_SIZE_UNVALIDATED_MAXIMUM)
	switch field {
	case TEST_HEADER_FORMAT:
		header.Format = tar.Format_Unvalidated(test_domain_word_8(marker))
	case TEST_HEADER_TYPE_FLAG:
		header.Type_Flag = tar.Type_Flag(test_domain_word_8(marker))
	case TEST_HEADER_NAME:
		header.Name = make([]byte, text_size)
		test_fill_bytes(header.Name, 'n')
	case TEST_HEADER_LINK_NAME:
		header.Link_Name = make([]byte, text_size)
		test_fill_bytes(header.Link_Name, 'l')
	case TEST_HEADER_SIZE:
		header.Size = tar.Entry_Size_Unvalidated(test_domain_integer_64(marker))
	case TEST_HEADER_MODE:
		header.Mode = tar.File_Mode(test_domain_integer_64(marker))
	case TEST_HEADER_USER_IDENTIFIER:
		header.User_Identifier = tar.User_Identifier(test_domain_integer_64(marker))
	case TEST_HEADER_GROUP_IDENTIFIER:
		header.Group_Identifier = tar.Group_Identifier(test_domain_integer_64(marker))
	case TEST_HEADER_USER_NAME:
		header.User_Name = make(
			[]byte, test_domain_size(
				marker, tar.HEADER_USER_NAME_SIZE_UNVALIDATED_MAXIMUM,
			),
		)
		test_fill_bytes(header.User_Name, 'u')
	case TEST_HEADER_GROUP_NAME:
		header.Group_Name = make(
			[]byte, test_domain_size(
				marker, tar.HEADER_GROUP_NAME_SIZE_UNVALIDATED_MAXIMUM,
			),
		)
		test_fill_bytes(header.Group_Name, 'g')
	case TEST_HEADER_MODIFICATION_TIME:
		header.Modification_Time = test_domain_timestamp_unvalidated(marker)
	case TEST_HEADER_ACCESS_TIME:
		header.Access_Time = test_domain_access_time_unvalidated(marker)
	case TEST_HEADER_CHANGE_TIME:
		header.Change_Time = test_domain_change_time_unvalidated(marker)
	case TEST_HEADER_DEVICE_MAJOR:
		header.Device_Major = tar.Device_Major(test_domain_integer_64(marker))
	case TEST_HEADER_DEVICE_MINOR:
		header.Device_Minor = tar.Device_Minor(test_domain_integer_64(marker))
	case TEST_HEADER_PAX_RECORDS:
		header.PAX_Records = make([]byte, size)
	}
	return header
}

func test_fill_bytes(destination []byte, value byte) {
	for position_index := range destination {
		destination[position_index] = value
	}
}

func test_domain_timestamp_unvalidated(
	marker int,
) (timestamp tar.Timestamp_Unvalidated) {
	return tar.Timestamp_Unvalidated{
		Seconds: tar.Integer(test_domain_integer_64(marker)),
		Nanoseconds: tar.Nanosecond_Count_Unvalidated(
			test_domain_integer_32(marker),
		),
		Set: bytes.Boolean(test_domain_boolean(marker)),
	}
}

func test_domain_access_time_unvalidated(
	marker int,
) (timestamp tar.Access_Timestamp_Unvalidated) {
	return tar.Access_Timestamp_Unvalidated{
		Seconds: tar.Access_Seconds(test_domain_integer_64(marker)),
		Nanoseconds: tar.Access_Nanosecond_Count_Unvalidated(
			test_domain_integer_32(marker),
		),
		Set: tar.Access_Time_Set(test_domain_boolean(marker)),
	}
}

func test_domain_change_time_unvalidated(
	marker int,
) (timestamp tar.Change_Timestamp_Unvalidated) {
	return tar.Change_Timestamp_Unvalidated{
		Seconds: tar.Change_Seconds(test_domain_integer_64(marker)),
		Nanoseconds: tar.Change_Nanosecond_Count_Unvalidated(
			test_domain_integer_32(marker),
		),
		Set: tar.Change_Time_Set(test_domain_boolean(marker)),
	}
}

func test_domain_callback(_ nbio.Completion_Handle) { return }

func test_domain_header(marker int) (header tar.Header) {
	field_size := test_domain_size(marker, tar.HEADER_TEXT_SIZE_MAXIMUM)
	user_size := test_domain_size(marker, tar.HEADER_USER_NAME_SIZE_MAXIMUM)
	group_size := test_domain_size(marker, tar.HEADER_GROUP_NAME_SIZE_MAXIMUM)
	seconds := test_domain_integer_64(marker)
	nanoseconds := test_domain_size(marker, int(tar.TIMESTAMP_NANOSECOND_MAXIMUM))
	header = tar.Header{
		Format:    tar.Format(test_domain_size(marker, int(tar.FORMAT_MAXIMUM))),
		Type_Flag: tar.Type_Flag(test_domain_word_8(marker)),
		Name:      make([]byte, field_size),
		Link_Name: make([]byte, field_size),
		Size: tar.Entry_Size(
			test_domain_size(marker, tar.ARCHIVE_SIZE_MAXIMUM),
		),
		Mode:             tar.File_Mode(seconds),
		User_Identifier:  tar.User_Identifier(seconds),
		Group_Identifier: tar.Group_Identifier(seconds),
		User_Name:        make([]byte, user_size),
		Group_Name:       make([]byte, group_size),
		Modification_Time: tar.Timestamp{
			Seconds:     tar.Integer(seconds),
			Nanoseconds: tar.Nanosecond_Count(nanoseconds),
			Set:         bytes.Boolean(test_domain_boolean(marker)),
		},
		Access_Time: tar.Access_Timestamp{
			Seconds:     tar.Access_Seconds(seconds),
			Nanoseconds: tar.Access_Nanosecond_Count(nanoseconds),
			Set:         tar.Access_Time_Set(test_domain_boolean(marker)),
		},
		Change_Time: tar.Change_Timestamp{
			Seconds:     tar.Change_Seconds(seconds),
			Nanoseconds: tar.Change_Nanosecond_Count(nanoseconds),
			Set:         tar.Change_Time_Set(test_domain_boolean(marker)),
		},
		Device_Major: tar.Device_Major(seconds),
		Device_Minor: tar.Device_Minor(seconds),
	}
	return header
}

func test_domain_header_storage(marker int) (storage tar.Header_Storage) {
	size := test_domain_size(marker, tar.HEADER_TEXT_SIZE_MAXIMUM)
	user_size := test_domain_size(marker, tar.HEADER_USER_NAME_SIZE_MAXIMUM)
	group_size := test_domain_size(marker, tar.HEADER_GROUP_NAME_SIZE_MAXIMUM)
	return tar.Header_Storage{
		Name: make([]byte, size), Link_Name: make([]byte, size),
		User_Name: make([]byte, user_size), Group_Name: make([]byte, group_size),
	}
}

func test_reader_domain(marker int) {
	reader := test_domain_reader(marker)
	storage := tar.Reader_Storage{
		Block: make([]byte, test_domain_size(marker, tar.BLOCK_SIZE)),
		Metadata: make(
			[]byte, test_domain_size(marker, tar.READER_METADATA_SIZE_MAXIMUM),
		),
	}
	tar.Reader_Init(&reader, nbio.Stream{}, storage)
	reader = test_domain_reader(marker)
	reader.Active = false
	reader.Initialized = false
	tar.Reader_Next(
		&reader, &reader.Completion, test_domain_header_storage(marker),
		test_domain_callback,
	)
	reader = test_domain_reader(marker)
	reader.Active = false
	reader.Initialized = false
	destination := make(
		[]byte, test_domain_size(marker, tar.ARCHIVE_SIZE_MAXIMUM),
	)
	tar.Reader_Read(
		&reader, &reader.Completion, destination, test_domain_callback,
	)
	test_reader_retained_header_domain(marker)
	test_reader_content_domain(marker)
	test_reader_failure_domain(marker)
}

func test_reader_retained_header_domain(marker int) {
	var pax_storage [TEST_FIELD_SIZE]byte
	pax_count := pax_record_into(
		pax_storage[:], []byte(tar.PAX_PATH_KEY), []byte("retained.txt"),
	)
	var archive_storage [TEST_ARCHIVE_SIZE]byte
	archive := tar_entry(
		archive_storage[:], 0, tar.FORMAT_USTAR, tar.TYPE_PAX_LOCAL,
		[]byte(tar.PAX_HEADER_NAME), pax_storage[:pax_count],
	)
	archive = tar_entry(
		archive_storage[:], len(archive), tar.FORMAT_USTAR, tar.TYPE_REGULAR,
		[]byte("placeholder"), nil,
	)
	archive = archive_footer(archive_storage[:], len(archive))
	memory := nbio.Stream_Memory{Memory: archive}
	var block [tar.BLOCK_SIZE]byte
	var metadata [TEST_FIELD_SIZE]byte
	reader := tar.Reader{
		Stream: nbio.Memory_To_Stream(&memory),
		Archive: tar.Reader_Archive{
			Block: block[:], Metadata: metadata[:], Header: test_domain_header(marker),
		},
		Initialized: true,
	}
	tar.Reader_Next(
		&reader, &reader.Completion, test_domain_header_storage(marker),
		test_domain_callback,
	)
}

func test_reader_content_domain(marker int) {
	size := test_domain_size(marker, tar.ARCHIVE_SIZE_MAXIMUM)
	if size == 0 {
		size = test_domain_size(TEST_DOMAIN_ONE, tar.ARCHIVE_SIZE_MAXIMUM)
	}
	content := make([]byte, size)
	memory := nbio.Stream_Memory{Memory: content}
	reader := tar.Reader{
		Stream: nbio.Memory_To_Stream(&memory),
		Archive: tar.Reader_Archive{
			Header: test_domain_header(marker), Physical_Debt: tar.Physical_Debt(size),
			Logical_Size: tar.Logical_Count(size), Entry_Active: true,
		},
		Initialized: true,
	}
	tar.Reader_Read(
		&reader, &reader.Completion, make([]byte, size), test_domain_callback,
	)
}

func test_reader_failure_domain(marker int) {
	size := test_domain_size(marker, tar.ARCHIVE_SIZE_MAXIMUM)
	if size == 0 {
		size = test_domain_size(TEST_DOMAIN_ONE, tar.ARCHIVE_SIZE_MAXIMUM)
	}
	reader := tar.Reader{
		Stream: nbio.Stream{Procedure: defective_read_procedure},
		Archive: tar.Reader_Archive{
			Header: test_domain_header(marker), Physical_Debt: tar.Physical_Debt(size),
			Logical_Size: tar.Logical_Count(size), Entry_Active: true,
		},
		Initialized: true,
	}
	tar.Reader_Read(
		&reader, &reader.Completion, make([]byte, size), test_domain_callback,
	)
}

func test_domain_reader(marker int) (reader tar.Reader) {
	archive_size := test_domain_size(marker, tar.ARCHIVE_SIZE_MAXIMUM)
	metadata_size := test_domain_size(marker, tar.READER_METADATA_SIZE_MAXIMUM)
	special_size := test_domain_size(marker, tar.SPECIAL_FILE_SIZE_MAXIMUM)
	long_size := test_domain_size(marker, tar.HEADER_TEXT_SIZE_MAXIMUM)
	padding := test_domain_size(marker, tar.PADDING_SIZE_MAXIMUM)
	logical := test_domain_size(marker, tar.ARCHIVE_SIZE_MAXIMUM)
	return tar.Reader{
		Archive: tar.Reader_Archive{
			Block: make(
				[]byte, test_domain_size(marker, tar.BLOCK_SIZE),
			),
			Metadata:             make([]byte, metadata_size),
			Metadata_Position:    tar.Metadata_Position(metadata_size),
			Header:               test_domain_header(marker),
			Header_Storage:       test_domain_header_storage(marker),
			Pending_PAX_Position: tar.Pending_PAX_Position(metadata_size),
			Pending_PAX_Size:     tar.Pending_PAX_Size(special_size),
			Long_Name:            make([]byte, long_size),
			Long_Link:            make([]byte, long_size),
			Physical_Debt:        tar.Physical_Debt(archive_size),
			Padding:              tar.Padding_Count(padding),
			Extension_Type:       tar.Reader_Extension_Type(test_domain_word_8(marker)),
			Extension_Size:       tar.Reader_Extension_Size(special_size),
			Logical_Size:         tar.Logical_Count(logical),
			Logical_Position:     tar.Logical_Position(logical),
			Entry_Active:         tar.Entry_Active(test_domain_boolean(marker)),
		},
		Transfer: tar.Reader_Transfer{
			Buffer: make([]byte, archive_size),
			Offset: tar.Reader_Transfer_Offset(archive_size),
			Needed: tar.Reader_Transfer_Needed(archive_size),
			Stage:  test_domain_reader_stage(marker),
			Submission_Active: tar.Reader_Submission_Active(
				test_domain_boolean(marker),
			),
			Wait_Active: tar.Reader_Wait_Active(test_domain_boolean(marker)),
			Continue:    tar.Reader_Continue(test_domain_boolean(marker)),
		},
		Skip_Debt:   tar.Reader_Skip_Debt(archive_size),
		Status:      test_domain_status(marker),
		Count:       tar.Count(archive_size),
		Active:      tar.Reader_Active(test_domain_boolean(marker)),
		Initialized: tar.Reader_Initialized(test_domain_boolean(marker)),
	}
}

func test_writer_domain(marker int) {
	writer := test_domain_writer(marker)
	storage := make(
		[]byte, test_domain_size(marker, tar.WRITER_STORAGE_SIZE_MAXIMUM),
	)
	tar.Writer_Init(&writer, nbio.Stream{}, storage)
	header := test_header_unvalidated_domain(TEST_HEADER_FORMAT, TEST_DOMAIN_ZERO)
	writer = test_domain_writer(marker)
	writer.Active = false
	writer.Initialized = false
	tar.Writer_Write_Header(
		&writer, &writer.Completion, &header, test_domain_callback,
	)
	writer = test_domain_writer(marker)
	writer.Active = false
	writer.Initialized = false
	source := make([]byte, test_domain_size(marker, tar.ARCHIVE_SIZE_MAXIMUM))
	tar.Writer_Write(
		&writer, &writer.Completion, source, test_domain_callback,
	)
	writer = test_domain_writer(marker)
	writer.Active = false
	writer.Initialized = false
	tar.Writer_Close(&writer, &writer.Completion, test_domain_callback)
}

func test_domain_writer(marker int) (writer tar.Writer) {
	archive_size := test_domain_size(marker, tar.ARCHIVE_SIZE_MAXIMUM)
	storage_size := test_domain_size(marker, tar.WRITER_STORAGE_SIZE_MAXIMUM)
	padding := test_domain_size(marker, tar.PADDING_SIZE_MAXIMUM)
	return tar.Writer{
		Source:            make([]byte, archive_size),
		Pending_Content:   tar.Pending_Content_Count(archive_size),
		Pending_Padding:   tar.Pending_Padding_Count(padding),
		Archive_Count:     tar.Archive_Count(archive_size),
		Operation:         test_domain_writer_operation(marker),
		Status:            test_domain_status(marker),
		Count:             tar.Count(archive_size),
		Active:            tar.Writer_Active(test_domain_boolean(marker)),
		Submission_Active: tar.Writer_Submission_Active(test_domain_boolean(marker)),
		Wait_Active:       tar.Writer_Wait_Active(test_domain_boolean(marker)),
		Continue:          tar.Writer_Continue(test_domain_boolean(marker)),
		Destination:       make([]byte, storage_size),
		Position:          tar.Destination_Position(storage_size),
		Content_Count:     tar.Content_Count(archive_size),
		Padding:           tar.Padding_Count(padding),
		Initialized:       tar.Writer_Initialized(test_domain_boolean(marker)),
		Closed:            tar.Writer_Closed(test_domain_boolean(marker)),
	}
}

func test_domain_size(marker int, maximum int) (size int) {
	switch marker {
	case TEST_DOMAIN_ONE:
		return 1
	case TEST_DOMAIN_TWO:
		return 2
	case TEST_DOMAIN_MAXIMUM:
		return maximum
	default:
		return 0
	}
}

func test_domain_integer_64(marker int) (value int64) {
	switch marker {
	case TEST_DOMAIN_MINIMUM:
		return bits.INTEGER_64_MINIMUM
	case TEST_DOMAIN_NEGATIVE_ONE:
		return -1
	case TEST_DOMAIN_ONE:
		return 1
	case TEST_DOMAIN_TWO:
		return 2
	case TEST_DOMAIN_MAXIMUM:
		return bits.INTEGER_64_MAXIMUM
	default:
		return 0
	}
}

func test_domain_small_integer(marker int) (value int64) {
	switch marker {
	case TEST_DOMAIN_MINIMUM:
		return tar.SMALL_NUMERIC_MINIMUM
	case TEST_DOMAIN_NEGATIVE_ONE:
		return -1
	case TEST_DOMAIN_ONE:
		return 1
	case TEST_DOMAIN_TWO:
		return 2
	case TEST_DOMAIN_MAXIMUM:
		return tar.SMALL_NUMERIC_MAXIMUM
	default:
		return 0
	}
}

func test_domain_integer_32(marker int) (value int32) {
	switch marker {
	case TEST_DOMAIN_MINIMUM:
		return bits.INTEGER_32_MINIMUM
	case TEST_DOMAIN_NEGATIVE_ONE:
		return -1
	case TEST_DOMAIN_ONE:
		return 1
	case TEST_DOMAIN_TWO:
		return 2
	case TEST_DOMAIN_MAXIMUM:
		return bits.INTEGER_32_MAXIMUM
	default:
		return 0
	}
}

func test_domain_word_8(marker int) (value uint8) {
	if marker == TEST_DOMAIN_MAXIMUM {
		return bits.WORD_8_MAXIMUM
	}
	return uint8(test_domain_size(marker, int(bits.WORD_8_MAXIMUM)))
}

func test_domain_boolean(marker int) (value bool) {
	return marker == TEST_DOMAIN_ONE || marker == TEST_DOMAIN_MAXIMUM
}

func test_domain_status(marker int) (status tar.Status) {
	return tar.Status(test_domain_size(marker, int(tar.STATUS_MAXIMUM)))
}

func test_domain_reader_stage(marker int) (stage tar.Reader_Stage) {
	return tar.Reader_Stage(
		test_domain_size(marker, int(tar.READER_STAGE_CONTENT)),
	)
}

func test_domain_writer_operation(marker int) (operation tar.Writer_Operation) {
	return tar.Writer_Operation(
		test_domain_size(marker, int(tar.WRITER_OPERATION_CLOSE)),
	)
}

func test_constants(t *testing.T) {
	t.Helper()
	test_layout_constants(t)
	test_pax_constants(t)
	test_numeric_constants(t)
	test_storage_constants(t)
}

func test_layout_constants(t *testing.T) {
	t.Helper()
	header_size := tar.NAME_FIELD_SIZE + tar.MODE_FIELD_SIZE +
		tar.USER_IDENTIFIER_FIELD_SIZE + tar.GROUP_IDENTIFIER_FIELD_SIZE +
		tar.ENTRY_SIZE_FIELD_SIZE + tar.TIMESTAMP_FIELD_SIZE +
		tar.CHECKSUM_FIELD_SIZE + tar.TYPE_FLAG_FIELD_SIZE +
		tar.LINK_NAME_FIELD_SIZE + tar.MAGIC_FIELD_SIZE +
		tar.VERSION_FIELD_SIZE + tar.USER_NAME_FIELD_SIZE +
		tar.GROUP_NAME_FIELD_SIZE + tar.DEVICE_MAJOR_FIELD_SIZE +
		tar.DEVICE_MINOR_FIELD_SIZE + tar.PREFIX_FIELD_SIZE +
		tar.HEADER_PADDING_FIELD_SIZE
	if tar.BLOCK_SIZE != header_size {
		t.Fatalf("BLOCK_SIZE = %d; field sum = %d", tar.BLOCK_SIZE, header_size)
	}
	test_constant_int(t, "MODE_FIELD_OFFSET", tar.MODE_FIELD_OFFSET,
		tar.NAME_FIELD_OFFSET+tar.NAME_FIELD_SIZE)
	test_constant_int(t, "USER_IDENTIFIER_FIELD_OFFSET",
		tar.USER_IDENTIFIER_FIELD_OFFSET, tar.MODE_FIELD_OFFSET+tar.MODE_FIELD_SIZE)
	test_constant_int(t, "GROUP_IDENTIFIER_FIELD_OFFSET",
		tar.GROUP_IDENTIFIER_FIELD_OFFSET,
		tar.USER_IDENTIFIER_FIELD_OFFSET+tar.USER_IDENTIFIER_FIELD_SIZE)
	test_constant_int(t, "ENTRY_SIZE_FIELD_OFFSET", tar.ENTRY_SIZE_FIELD_OFFSET,
		tar.GROUP_IDENTIFIER_FIELD_OFFSET+tar.GROUP_IDENTIFIER_FIELD_SIZE)
	test_constant_int(t, "TIMESTAMP_FIELD_OFFSET", tar.TIMESTAMP_FIELD_OFFSET,
		tar.ENTRY_SIZE_FIELD_OFFSET+tar.ENTRY_SIZE_FIELD_SIZE)
	test_constant_int(t, "CHECKSUM_FIELD_OFFSET", tar.CHECKSUM_FIELD_OFFSET,
		tar.TIMESTAMP_FIELD_OFFSET+tar.TIMESTAMP_FIELD_SIZE)
	test_constant_int(t, "TYPE_FLAG_FIELD_OFFSET", tar.TYPE_FLAG_FIELD_OFFSET,
		tar.CHECKSUM_FIELD_OFFSET+tar.CHECKSUM_FIELD_SIZE)
	test_constant_int(t, "LINK_NAME_FIELD_OFFSET", tar.LINK_NAME_FIELD_OFFSET,
		tar.TYPE_FLAG_FIELD_OFFSET+tar.TYPE_FLAG_FIELD_SIZE)
	test_constant_int(t, "MAGIC_FIELD_OFFSET", tar.MAGIC_FIELD_OFFSET,
		tar.LINK_NAME_FIELD_OFFSET+tar.LINK_NAME_FIELD_SIZE)
	test_constant_int(t, "VERSION_FIELD_OFFSET", tar.VERSION_FIELD_OFFSET,
		tar.MAGIC_FIELD_OFFSET+tar.MAGIC_FIELD_SIZE)
	test_constant_int(t, "USER_NAME_FIELD_OFFSET", tar.USER_NAME_FIELD_OFFSET,
		tar.VERSION_FIELD_OFFSET+tar.VERSION_FIELD_SIZE)
	test_constant_int(t, "GROUP_NAME_FIELD_OFFSET", tar.GROUP_NAME_FIELD_OFFSET,
		tar.USER_NAME_FIELD_OFFSET+tar.USER_NAME_FIELD_SIZE)
	test_constant_int(t, "DEVICE_MAJOR_FIELD_OFFSET", tar.DEVICE_MAJOR_FIELD_OFFSET,
		tar.GROUP_NAME_FIELD_OFFSET+tar.GROUP_NAME_FIELD_SIZE)
	test_constant_int(t, "DEVICE_MINOR_FIELD_OFFSET", tar.DEVICE_MINOR_FIELD_OFFSET,
		tar.DEVICE_MAJOR_FIELD_OFFSET+tar.DEVICE_MAJOR_FIELD_SIZE)
	test_constant_int(t, "PREFIX_FIELD_OFFSET", tar.PREFIX_FIELD_OFFSET,
		tar.DEVICE_MINOR_FIELD_OFFSET+tar.DEVICE_MINOR_FIELD_SIZE)
	test_constant_int(t, "HEADER_PADDING_FIELD_OFFSET",
		tar.HEADER_PADDING_FIELD_OFFSET, tar.PREFIX_FIELD_OFFSET+tar.PREFIX_FIELD_SIZE)
	test_constant_int(t, "header end",
		tar.HEADER_PADDING_FIELD_OFFSET+tar.HEADER_PADDING_FIELD_SIZE, tar.BLOCK_SIZE)
	test_constant_int(t, "STAR_PREFIX_FIELD_SIZE", tar.STAR_PREFIX_FIELD_SIZE,
		tar.PREFIX_FIELD_SIZE-tar.STAR_TIMESTAMP_FIELD_COUNT*tar.TIMESTAMP_FIELD_SIZE)
	test_constant_int(t, "STAR_TRAILER_FIELD_SIZE", tar.STAR_TRAILER_FIELD_SIZE,
		tar.STAR_TRAILER_VERSION_FIELD_COUNT*tar.VERSION_FIELD_SIZE)
}

func test_pax_constants(t *testing.T) {
	t.Helper()
	test_constant_int(t, "PAX_COUNT_SEPARATOR_SIZE", tar.PAX_COUNT_SEPARATOR_SIZE,
		len(" "))
	test_constant_int(t, "PAX_KEY_SEPARATOR_SIZE", tar.PAX_KEY_SEPARATOR_SIZE,
		len("="))
	test_constant_int(t, "PAX_RECORD_TERMINATOR_SIZE",
		tar.PAX_RECORD_TERMINATOR_SIZE, len("\n"))
	test_constant_int(t, "GNU_LONG_FIELD_TERMINATOR_SIZE",
		tar.GNU_LONG_FIELD_TERMINATOR_SIZE, len("\x00"))
	test_constant_int(t, "PAX_RECORD_COUNT_TEXT_SIZE_MAXIMUM",
		tar.PAX_RECORD_COUNT_TEXT_SIZE_MAXIMUM,
		decimal_size(tar.SPECIAL_FILE_SIZE_MAXIMUM))
	test_constant_int(t, "PAX_RECORD_COUNT_TEXT_SIZE_MINIMUM",
		tar.PAX_RECORD_COUNT_TEXT_SIZE_MINIMUM, tar.TYPE_FLAG_FIELD_SIZE)
	test_constant_int(t, "PAX_SMALL_RECORD_COUNT_TEXT_SIZE",
		tar.PAX_SMALL_RECORD_COUNT_TEXT_SIZE,
		tar.PAX_RECORD_COUNT_TEXT_SIZE_MINIMUM+tar.TYPE_FLAG_FIELD_SIZE)
	test_constant_int(t, "PAX_PAYLOAD_SIZE_MINIMUM", tar.PAX_PAYLOAD_SIZE_MINIMUM,
		tar.PAX_MANDATORY_RECORD_COUNT*tar.PAX_RECORD_FIXED_SIZE_MINIMUM+
			len(tar.PAX_PATH_KEY)+tar.TYPE_FLAG_FIELD_SIZE+
			len(tar.PAX_SIZE_KEY)+tar.TYPE_FLAG_FIELD_SIZE)
	test_constant_int(t, "ENTRY_SIZE_DECIMAL_SIZE_MAXIMUM",
		tar.ENTRY_SIZE_DECIMAL_SIZE_MAXIMUM,
		decimal_size(tar.ARCHIVE_SIZE_MAXIMUM))
	test_constant_int(t, "INTEGER_DECIMAL_SIZE_MAXIMUM",
		tar.INTEGER_DECIMAL_SIZE_MAXIMUM, strconv.DECIMAL_TEXT_SIZE_MAXIMUM)
	test_constant_int(t, "PAX_TIMESTAMP_RECORD_COUNT",
		tar.PAX_TIMESTAMP_RECORD_COUNT,
		(len(tar.PAX_MODIFICATION_TIME_KEY)+len(tar.PAX_ACCESS_TIME_KEY)+
			len(tar.PAX_CHANGE_TIME_KEY))/tar.PAX_TIME_KEY_SIZE)
	test_constant_int(t, "HEADER_TEXT_SIZE_MAXIMUM", tar.HEADER_TEXT_SIZE_MAXIMUM,
		tar.SPECIAL_FILE_SIZE_MAXIMUM-tar.GNU_LONG_FIELD_TERMINATOR_SIZE)
}

func test_numeric_constants(t *testing.T) {
	t.Helper()
	test_constant_int(t, "INTEGER_BYTE_SHIFT", tar.INTEGER_BYTE_SHIFT,
		bits.BIT_COUNT_64_MAXIMUM-bits.BIT_COUNT_8_MAXIMUM)
	test_constant_int(t, "SMALL_NUMERIC_PAYLOAD_BIT_COUNT",
		tar.SMALL_NUMERIC_PAYLOAD_BIT_COUNT,
		(tar.MODE_FIELD_SIZE-tar.TYPE_FLAG_FIELD_SIZE)*bits.BIT_COUNT_8_MAXIMUM)
	test_constant_int64(t, "SMALL_NUMERIC_MINIMUM", tar.SMALL_NUMERIC_MINIMUM,
		-(1 << tar.SMALL_NUMERIC_PAYLOAD_BIT_COUNT))
	test_constant_int64(t, "SMALL_NUMERIC_MAXIMUM", tar.SMALL_NUMERIC_MAXIMUM,
		1<<tar.SMALL_NUMERIC_PAYLOAD_BIT_COUNT-tar.TYPE_FLAG_FIELD_SIZE)
	test_constant_int64(t, "SMALL_OCTAL_MAXIMUM", tar.SMALL_OCTAL_MAXIMUM,
		1<<tar.SMALL_OCTAL_PAYLOAD_BIT_COUNT-tar.TYPE_FLAG_FIELD_SIZE)
	test_constant_int64(t, "LARGE_OCTAL_MAXIMUM", tar.LARGE_OCTAL_MAXIMUM,
		1<<tar.LARGE_OCTAL_PAYLOAD_BIT_COUNT-tar.TYPE_FLAG_FIELD_SIZE)
	test_constant_int64(t, "OCTAL_PARSE_MAXIMUM", tar.OCTAL_PARSE_MAXIMUM,
		1<<tar.OCTAL_PARSE_BIT_COUNT_MAXIMUM-tar.TYPE_FLAG_FIELD_SIZE)
}

func test_storage_constants(t *testing.T) {
	t.Helper()
	test_constant_int(t, "ARCHIVE_MEBIBYTE_COUNT_MAXIMUM",
		tar.ARCHIVE_MEBIBYTE_COUNT_MAXIMUM, 1<<6)
	archive_size := tar.ARCHIVE_MEBIBYTE_COUNT_MAXIMUM * bits.MEBIBYTE_BYTES
	if tar.ARCHIVE_SIZE_MAXIMUM != archive_size {
		t.Fatalf(
			"ARCHIVE_SIZE_MAXIMUM = %d; formula = %d",
			tar.ARCHIVE_SIZE_MAXIMUM, archive_size,
		)
	}
	if tar.SPECIAL_FILE_SIZE_MAXIMUM != bits.MEBIBYTE_BYTES {
		t.Fatalf(
			"SPECIAL_FILE_SIZE_MAXIMUM = %d; formula = %d",
			tar.SPECIAL_FILE_SIZE_MAXIMUM, bits.MEBIBYTE_BYTES,
		)
	}
	footer_size := tar.ARCHIVE_FOOTER_BLOCK_COUNT * tar.BLOCK_SIZE
	if tar.ARCHIVE_FOOTER_SIZE != footer_size {
		t.Fatalf(
			"ARCHIVE_FOOTER_SIZE = %d; formula = %d",
			tar.ARCHIVE_FOOTER_SIZE, footer_size,
		)
	}
	writer_minimum := tar.PADDING_SIZE_MAXIMUM + tar.ARCHIVE_FOOTER_SIZE
	if tar.WRITER_STORAGE_SIZE_MINIMUM != writer_minimum {
		t.Fatalf(
			"WRITER_STORAGE_SIZE_MINIMUM = %d; formula = %d",
			tar.WRITER_STORAGE_SIZE_MINIMUM, writer_minimum,
		)
	}
	gnu_field_size := tar.BLOCK_SIZE + tar.SPECIAL_FILE_SIZE_MAXIMUM
	writer_maximum := tar.PADDING_SIZE_MAXIMUM +
		tar.GNU_LONG_FIELD_COUNT_MAXIMUM*gnu_field_size + tar.BLOCK_SIZE
	if tar.WRITER_STORAGE_SIZE_MAXIMUM != writer_maximum {
		t.Fatalf(
			"WRITER_STORAGE_SIZE_MAXIMUM = %d; formula = %d",
			tar.WRITER_STORAGE_SIZE_MAXIMUM, writer_maximum,
		)
	}
	test_constant_int(t, "READER_METADATA_SIZE_MAXIMUM",
		tar.READER_METADATA_SIZE_MAXIMUM,
		(tar.GNU_LONG_FIELD_COUNT_MAXIMUM+tar.TYPE_FLAG_FIELD_SIZE)*
			tar.SPECIAL_FILE_SIZE_MAXIMUM)
	test_constant_int(t, "READER_METADATA_BLOCK_COUNT_MAXIMUM",
		tar.READER_METADATA_BLOCK_COUNT_MAXIMUM,
		tar.READER_METADATA_SIZE_MAXIMUM/tar.BLOCK_SIZE)
	test_constant_int(t, "ENCODED_HEADER_SIZE_MAXIMUM",
		tar.ENCODED_HEADER_SIZE_MAXIMUM,
		tar.WRITER_STORAGE_SIZE_MAXIMUM-tar.PADDING_SIZE_MAXIMUM)
}

func test_constant_int(t *testing.T, name string, value int, formula int) {
	t.Helper()
	if value != formula {
		t.Fatalf("%s = %d; formula = %d", name, value, formula)
	}
}

func test_constant_int64(t *testing.T, name string, value int64, formula int64) {
	t.Helper()
	if value != formula {
		t.Fatalf("%s = %d; formula = %d", name, value, formula)
	}
}

func test_stream_boundary(t *testing.T) {
	var archive [tar.ARCHIVE_FOOTER_BLOCK_COUNT * tar.ARCHIVE_FOOTER_SIZE]byte
	var reader_block [tar.BLOCK_SIZE]byte
	var reader_metadata [tar.SPECIAL_FILE_SIZE_MAXIMUM]byte
	var writer_workspace [tar.WRITER_STORAGE_SIZE_MINIMUM]byte
	reader_memory := nbio.Stream_Memory{Memory: archive[:]}
	writer_memory := nbio.Stream_Memory{Memory: archive[:]}
	var reader tar.Reader
	reader_status := tar.Reader_Init(
		&reader,
		nbio.Memory_To_Stream(&reader_memory),
		tar.Reader_Storage{
			Block:    reader_block[:],
			Metadata: reader_metadata[:],
		},
	)
	if reader_status != tar.STATUS_OK {
		t.Fatalf("Reader_Init status = %v", reader_status)
	}
	var writer tar.Writer
	writer_status := tar.Writer_Init(
		&writer,
		nbio.Memory_To_Stream(&writer_memory),
		writer_workspace[:],
	)
	if writer_status != tar.STATUS_OK {
		t.Fatalf("Writer_Init status = %v", writer_status)
	}
	callback_called := false
	tar.Writer_Close(&writer, &writer.Completion, func(_ nbio.Completion_Handle) {
		callback_called = true
	})
	if !callback_called {
		t.Fatal("Writer_Close did not retire")
	}
}

type reader_fixture struct {
	Reader      tar.Reader
	Memory      nbio.Stream_Memory
	Header      tar.Header
	Block       []byte
	Metadata    []byte
	Name        []byte
	Link_Name   []byte
	User_Name   []byte
	Group_Name  []byte
	Destination []byte
	Called      bool
}

func reader_fixture_init(
	fixture *reader_fixture, archive []byte,
) (status tar.Initialization_Status) {
	if fixture.Block == nil {
		fixture.Block = make([]byte, tar.BLOCK_SIZE)
		fixture.Metadata = make(
			[]byte, tar.SPECIAL_FILE_SIZE_MAXIMUM+tar.BLOCK_SIZE-1,
		)
		fixture.Name = make([]byte, TEST_FIELD_SIZE)
		fixture.Link_Name = make([]byte, TEST_FIELD_SIZE)
		fixture.User_Name = make([]byte, TEST_FIELD_SIZE)
		fixture.Group_Name = make([]byte, TEST_FIELD_SIZE)
		fixture.Destination = make([]byte, TEST_FIELD_SIZE)
	}
	fixture.Memory = nbio.Stream_Memory{Memory: archive}
	status = tar.Reader_Init(
		&fixture.Reader,
		nbio.Memory_To_Stream(&fixture.Memory),
		tar.Reader_Storage{
			Block: fixture.Block[:], Metadata: fixture.Metadata[:],
		},
	)
	return status
}

func reader_fixture_storage(fixture *reader_fixture) (storage tar.Header_Storage) {
	storage = tar.Header_Storage{
		Name: fixture.Name[:], Link_Name: fixture.Link_Name[:],
		User_Name: fixture.User_Name[:], Group_Name: fixture.Group_Name[:],
	}
	return storage
}

func reader_fixture_next(fixture *reader_fixture) {
	fixture.Called = false
	tar.Reader_Next(
		&fixture.Reader, &fixture.Reader.Completion, reader_fixture_storage(fixture),
		func(_ nbio.Completion_Handle) {
			fixture.Header = fixture.Reader.Archive.Header
			fixture.Called = true
		},
	)
}

func reader_fixture_read(fixture *reader_fixture, destination []byte) {
	fixture.Called = false
	tar.Reader_Read(
		&fixture.Reader, &fixture.Reader.Completion, destination,
		func(_ nbio.Completion_Handle) { fixture.Called = true },
	)
}

type writer_fixture struct {
	Writer    tar.Writer
	Memory    nbio.Stream_Memory
	Workspace []byte
	Called    bool
}

func writer_fixture_init(
	fixture *writer_fixture, archive []byte,
) (status tar.Initialization_Status) {
	if fixture.Workspace == nil {
		fixture.Workspace = make(
			[]byte, TEST_WRITER_WORKSPACE_BLOCK_COUNT*tar.BLOCK_SIZE,
		)
	}
	fixture.Memory = nbio.Stream_Memory{Memory: archive}
	status = tar.Writer_Init(
		&fixture.Writer,
		nbio.Memory_To_Stream(&fixture.Memory),
		fixture.Workspace[:],
	)
	return status
}

func writer_fixture_write_header(
	fixture *writer_fixture, header *tar.Header_Unvalidated,
) {
	fixture.Called = false
	tar.Writer_Write_Header(
		&fixture.Writer, &fixture.Writer.Completion, header,
		func(_ nbio.Completion_Handle) { fixture.Called = true },
	)
}

func writer_fixture_write(fixture *writer_fixture, source []byte) {
	fixture.Called = false
	tar.Writer_Write(
		&fixture.Writer, &fixture.Writer.Completion, source,
		func(_ nbio.Completion_Handle) { fixture.Called = true },
	)
}

func writer_fixture_close(fixture *writer_fixture) {
	fixture.Called = false
	tar.Writer_Close(
		&fixture.Writer, &fixture.Writer.Completion,
		func(_ nbio.Completion_Handle) { fixture.Called = true },
	)
}

// Test_Reader_Standard_Header proves Stream cursor, USTAR prefix, and content remain aligned.
func test_reader_standard_header(t *testing.T) {
	var archive_storage [TEST_ARCHIVE_SIZE]byte
	content := []byte("bounded tar content")
	archive := ustar_archive(
		archive_storage[:], []byte("directory/entry.txt"), content,
	)
	var fixture reader_fixture
	if status := reader_fixture_init(&fixture, archive); status != tar.STATUS_OK {
		t.Fatalf("Reader_Init status = %v", status)
	}
	reader_fixture_next(&fixture)
	if !fixture.Called {
		t.Fatal("Reader_Next did not retire")
	}
	if fixture.Reader.Status != tar.STATUS_OK {
		t.Fatalf(
			"Reader_Next status = %v", fixture.Reader.Status,
		)
	}
	if fixture.Reader.Completion.Error != nil {
		t.Fatalf("Reader_Next error = %v", fixture.Reader.Completion.Error)
	}
	if !bytes.Equal(
		bytes.Slice(fixture.Header.Name), []byte("directory/entry.txt"),
	) {
		t.Fatalf("Header.Name = %q", fixture.Header.Name)
	}
	if fixture.Header.Format != tar.FORMAT_USTAR {
		t.Fatalf("Header.Format = %v", fixture.Header.Format)
	}
	reader_fixture_read(&fixture, fixture.Destination[:TEST_PARTIAL_READ_SIZE])
	first := fixture.Reader.Count
	reader_fixture_read(&fixture, fixture.Destination[first:])
	count := first + fixture.Reader.Count
	if !bytes.Equal(fixture.Destination[:count], content) {
		t.Fatalf("Reader_Read content = %q", fixture.Destination[:count])
	}
	reader_fixture_read(&fixture, fixture.Destination[:])
	if fixture.Reader.Status != tar.STATUS_END {
		t.Fatalf("exhausted Reader_Read status = %v", fixture.Reader.Status)
	}
	if fixture.Reader.Count != 0 {
		t.Fatalf("exhausted Reader_Read count = %d", fixture.Reader.Count)
	}
	reader_fixture_next(&fixture)
	if fixture.Reader.Status != tar.STATUS_END {
		t.Fatalf("final Reader_Next status = %v", fixture.Reader.Status)
	}
	if fixture.Reader.Completion.Error != nil {
		t.Fatalf("final Reader_Next error = %v", fixture.Reader.Completion.Error)
	}
}

// Test_Reader_Next_Discards proves unread entry bytes never become the following header.
func test_reader_next_discards(t *testing.T) {
	var archive_storage [TEST_ARCHIVE_SIZE]byte
	archive := ustar_entry(
		archive_storage[:], 0, []byte("one.txt"), []byte("first"),
	)
	archive = ustar_entry(
		archive_storage[:], len(archive), []byte("two.txt"), []byte("second"),
	)
	archive = archive_footer(archive_storage[:], len(archive))
	var fixture reader_fixture
	if status := reader_fixture_init(&fixture, archive); status != tar.STATUS_OK {
		t.Fatalf("Reader_Init status = %v", status)
	}
	reader_fixture_next(&fixture)
	reader_fixture_next(&fixture)
	if fixture.Reader.Status != tar.STATUS_OK {
		t.Fatalf("second header status = %v", fixture.Reader.Status)
	}
	if !bytes.Equal(bytes.Slice(fixture.Header.Name), []byte("two.txt")) {
		t.Fatalf("second header name = %q", fixture.Header.Name)
	}
}

// Test_Extended_Formats proves metadata survives its separate Stream transfer.
func test_extended_formats(t *testing.T) {
	var long_name [TEST_LONG_NAME_SIZE]byte
	for index := range long_name {
		long_name[index] = byte('a' + index%TEST_ALPHABET_SIZE)
	}
	long_name[tar.MODE_FIELD_SIZE*tar.DECIMAL_BASE] = '/'
	test_reader_pax(t, long_name[:])
	test_reader_gnu(t, long_name[:])
}

func test_reader_pax(t *testing.T, long_name []byte) {
	var pax_storage [TEST_FIELD_SIZE]byte
	pax_count := pax_record_into(
		pax_storage[:], []byte(tar.PAX_PATH_KEY), long_name[:],
	)
	pax_count += pax_record_into(
		pax_storage[pax_count:], []byte(tar.PAX_LINK_PATH_KEY), []byte("target"),
	)
	pax_count += pax_record_into(
		pax_storage[pax_count:], []byte(tar.PAX_SIZE_KEY), []byte("3"),
	)
	pax_count += pax_record_into(
		pax_storage[pax_count:], []byte(tar.PAX_USER_IDENTIFIER_KEY), []byte("-1"),
	)
	pax_count += pax_record_into(
		pax_storage[pax_count:], []byte(tar.PAX_GROUP_IDENTIFIER_KEY), []byte("2"),
	)
	pax_count += pax_record_into(
		pax_storage[pax_count:], []byte(tar.PAX_USER_NAME_KEY), []byte("user"),
	)
	pax_count += pax_record_into(
		pax_storage[pax_count:], []byte(tar.PAX_GROUP_NAME_KEY), []byte("group"),
	)
	pax_count += pax_record_into(
		pax_storage[pax_count:], []byte(tar.PAX_MODIFICATION_TIME_KEY), []byte("-1.5"),
	)
	pax_count += pax_record_into(
		pax_storage[pax_count:], []byte(tar.PAX_ACCESS_TIME_KEY), []byte("2.25"),
	)
	pax_count += pax_record_into(
		pax_storage[pax_count:], []byte(tar.PAX_CHANGE_TIME_KEY), []byte("3"),
	)
	var archive_storage [TEST_ARCHIVE_SIZE]byte
	archive := tar_entry(
		archive_storage[:], 0, tar.FORMAT_USTAR, tar.TYPE_PAX_LOCAL,
		[]byte("PaxHeaders.0"), pax_storage[:pax_count],
	)
	archive = tar_entry(
		archive_storage[:], len(archive), tar.FORMAT_USTAR, tar.TYPE_REGULAR,
		[]byte("placeholder"), []byte("pax"),
	)
	archive = archive_footer(archive_storage[:], len(archive))
	var fixture reader_fixture
	if status := reader_fixture_init(&fixture, archive); status != tar.STATUS_OK {
		t.Fatalf("Reader_Init status = %v", status)
	}
	reader_fixture_next(&fixture)
	if fixture.Reader.Status != tar.STATUS_OK {
		t.Fatalf("PAX header status = %v", fixture.Reader.Status)
	}
	if fixture.Header.Format != tar.FORMAT_PAX {
		t.Fatalf("PAX header format = %v", fixture.Header.Format)
	}
	if !bytes.Equal(bytes.Slice(fixture.Header.Name), long_name[:]) {
		t.Fatalf("PAX header name = %q", fixture.Header.Name)
	}
}

func test_reader_gnu(t *testing.T, long_name []byte) {
	var pax_storage [TEST_FIELD_SIZE]byte
	var archive_storage [TEST_ARCHIVE_SIZE]byte
	gnu_payload := append_nul(pax_storage[:], long_name[:])
	archive := tar_entry(
		archive_storage[:], 0, tar.FORMAT_GNU, tar.TYPE_GNU_LONG_NAME,
		[]byte("././@LongLink"), gnu_payload,
	)
	archive = tar_entry(
		archive_storage[:], len(archive), tar.FORMAT_GNU, tar.TYPE_REGULAR,
		[]byte("placeholder"), []byte("gnu"),
	)
	archive = archive_footer(archive_storage[:], len(archive))
	var fixture reader_fixture
	if status := reader_fixture_init(&fixture, archive); status != tar.STATUS_OK {
		t.Fatalf("GNU Reader_Init status = %v", status)
	}
	reader_fixture_next(&fixture)
	if fixture.Reader.Status != tar.STATUS_OK {
		t.Fatalf("GNU header status = %v", fixture.Reader.Status)
	}
	if fixture.Header.Format != tar.FORMAT_GNU {
		t.Fatalf("GNU header format = %v", fixture.Header.Format)
	}
	if !bytes.Equal(bytes.Slice(fixture.Header.Name), long_name[:]) {
		t.Fatalf("GNU header name = %q", fixture.Header.Name)
	}
}

func test_writer_extended_formats(t *testing.T) {
	for _, format := range []tar.Format_Unvalidated{
		tar.Format_Unvalidated(tar.FORMAT_PAX),
		tar.Format_Unvalidated(tar.FORMAT_GNU),
	} {
		var archive [TEST_ARCHIVE_SIZE]byte
		var fixture writer_fixture
		if status := writer_fixture_init(&fixture, archive[:]); status != tar.STATUS_OK {
			t.Fatalf("Writer_Init status = %v", status)
		}
		name := make([]byte, TEST_LONG_NAME_SIZE)
		link_name := make([]byte, TEST_LONG_NAME_SIZE)
		for index := range name {
			name[index] = 'n'
			link_name[index] = 'l'
		}
		header := tar.Header_Unvalidated{
			Format: format, Type_Flag: tar.TYPE_REGULAR,
			Name: name, Link_Name: link_name,
			User_Identifier:  TEST_USER_IDENTIFIER,
			Group_Identifier: TEST_GROUP_IDENTIFIER,
			User_Name:        []byte("user"), Group_Name: []byte("group"),
			Modification_Time: tar.Timestamp_Unvalidated{
				Seconds:     tar.Integer(TEST_MODIFICATION_SECONDS),
				Nanoseconds: 1, Set: true,
			},
			Access_Time: tar.Access_Timestamp_Unvalidated{
				Seconds: 1, Nanoseconds: 2, Set: true,
			},
			Change_Time: tar.Change_Timestamp_Unvalidated{
				Seconds: 2, Nanoseconds: 1, Set: true,
			},
		}
		writer_fixture_write_header(&fixture, &header)
		if fixture.Writer.Status != tar.STATUS_OK {
			t.Fatalf("format %v status = %v", format, fixture.Writer.Status)
		}
		writer_fixture_close(&fixture)
		if fixture.Writer.Status != tar.STATUS_OK {
			t.Fatalf("format %v close = %v", format, fixture.Writer.Status)
		}
	}
}

func test_reader_star(t *testing.T) {
	var archive_storage [TEST_ARCHIVE_SIZE]byte
	archive := tar_entry(
		archive_storage[:], 0, tar.FORMAT_USTAR, tar.TYPE_REGULAR,
		[]byte("star.txt"), []byte("star"),
	)
	header := archive[:tar.BLOCK_SIZE]
	copy(
		header[tar.STAR_TRAILER_FIELD_OFFSET:][:tar.STAR_TRAILER_FIELD_SIZE],
		tar.STAR_TRAILER,
	)
	format_octal(
		header[tar.STAR_ACCESS_TIMESTAMP_FIELD_OFFSET:][:tar.TIMESTAMP_FIELD_SIZE],
		TEST_MODIFICATION_SECONDS,
	)
	format_octal(
		header[tar.STAR_CHANGE_TIMESTAMP_FIELD_OFFSET:][:tar.TIMESTAMP_FIELD_SIZE],
		TEST_MODIFICATION_SECONDS,
	)
	tar_header_checksum(header)
	archive = archive_footer(archive_storage[:], len(archive))
	var fixture reader_fixture
	if status := reader_fixture_init(&fixture, archive); status != tar.STATUS_OK {
		t.Fatalf("STAR Reader_Init status = %v", status)
	}
	reader_fixture_next(&fixture)
	if fixture.Reader.Status != tar.STATUS_OK {
		t.Fatalf("STAR status = %v", fixture.Reader.Status)
	}
	if fixture.Header.Format != tar.FORMAT_STAR {
		t.Fatalf("STAR format = %v", fixture.Header.Format)
	}
}

func test_reader_gnu_prefix(t *testing.T) {
	var archive_storage [TEST_ARCHIVE_SIZE]byte
	archive := tar_entry(
		archive_storage[:], 0, tar.FORMAT_GNU, tar.TYPE_REGULAR,
		[]byte("entry.txt"), []byte("gnu"),
	)
	header := archive[:tar.BLOCK_SIZE]
	copy(
		header[tar.PREFIX_FIELD_OFFSET:][:tar.PREFIX_FIELD_SIZE],
		[]byte("prefix"),
	)
	header[tar.GNU_ACCESS_TIMESTAMP_FIELD_OFFSET] = 'x'
	tar_header_checksum(header)
	archive = archive_footer(archive_storage[:], len(archive))
	var fixture reader_fixture
	if status := reader_fixture_init(&fixture, archive); status != tar.STATUS_OK {
		t.Fatalf("GNU prefix Reader_Init status = %v", status)
	}
	reader_fixture_next(&fixture)
	if fixture.Reader.Status != tar.STATUS_OK {
		t.Fatalf("GNU prefix status = %v", fixture.Reader.Status)
	}
	if fixture.Header.Format != tar.FORMAT_GNU {
		t.Fatalf("GNU prefix format = %v", fixture.Header.Format)
	}
	if !bytes.Equal(bytes.Slice(fixture.Header.Name), []byte("xrefix/entry.txt")) {
		t.Fatalf("GNU prefix name = %q", fixture.Header.Name)
	}
}

// Test_Writer_Round_Trip proves every encoded byte crosses Stream before Reader sees it.
func test_writer_round_trip(t *testing.T) {
	var archive [TEST_ARCHIVE_SIZE]byte
	content := []byte("writer content")
	header := tar.Header_Unvalidated{
		Format:    tar.Format_Unvalidated(tar.FORMAT_USTAR),
		Type_Flag: tar.TYPE_REGULAR,
		Name:      []byte("nested/written.txt"),
		Size:      tar.Entry_Size_Unvalidated(len(content)),
		Mode:      TEST_FILE_MODE, User_Identifier: TEST_USER_IDENTIFIER,
		Group_Identifier: TEST_GROUP_IDENTIFIER,
		Modification_Time: tar.Timestamp_Unvalidated{Seconds: tar.Integer(
			TEST_MODIFICATION_SECONDS,
		), Set: true},
	}
	var writer writer_fixture
	if status := writer_fixture_init(&writer, archive[:]); status != tar.STATUS_OK {
		t.Fatalf("Writer_Init status = %v", status)
	}
	writer_fixture_write_header(&writer, &header)
	if !writer.Called {
		t.Fatal("Writer_Write_Header did not retire")
	}
	if writer.Writer.Status != tar.STATUS_OK {
		t.Fatalf("Writer_Write_Header status = %v", writer.Writer.Status)
	}
	if writer.Writer.Completion.Error != nil {
		t.Fatalf("Writer_Write_Header error = %v", writer.Writer.Completion.Error)
	}
	writer_fixture_write(&writer, content)
	if writer.Writer.Status != tar.STATUS_OK {
		t.Fatalf("Writer_Write status = %v", writer.Writer.Status)
	}
	if writer.Writer.Count != tar.Count(len(content)) {
		t.Fatalf("Writer_Write count = %d", writer.Writer.Count)
	}
	writer_fixture_close(&writer)
	if writer.Writer.Status != tar.STATUS_OK {
		t.Fatalf("Writer_Close status = %v", writer.Writer.Status)
	}
	if writer.Writer.Completion.Error != nil {
		t.Fatalf("Writer_Close error = %v", writer.Writer.Completion.Error)
	}
	archive_count := writer.Writer.Count

	var reader reader_fixture
	status := reader_fixture_init(&reader, archive[:archive_count])
	if status != tar.STATUS_OK {
		t.Fatalf("round-trip Reader_Init status = %v", status)
	}
	reader_fixture_next(&reader)
	reader_fixture_read(&reader, reader.Destination[:])
	if reader.Reader.Status != tar.STATUS_OK {
		t.Fatalf("round-trip status = %v", reader.Reader.Status)
	}
	if !bytes.Equal(bytes.Slice(reader.Header.Name), bytes.Slice(header.Name)) {
		t.Fatalf("round-trip name = %q", reader.Header.Name)
	}
	if !bytes.Equal(reader.Destination[:reader.Reader.Count], content) {
		t.Fatalf("round-trip content = %q", reader.Destination[:reader.Reader.Count])
	}
}

// Test_Writer_State proves declared content remains a hard boundary.
func test_writer_state(t *testing.T) {
	var archive [TEST_ARCHIVE_SIZE]byte
	header := tar.Header_Unvalidated{
		Format:    tar.Format_Unvalidated(tar.FORMAT_USTAR),
		Type_Flag: tar.TYPE_REGULAR,
		Name:      []byte("entry.txt"), Size: tar.Entry_Size_Unvalidated(len("ok")),
	}
	var fixture writer_fixture
	if status := writer_fixture_init(&fixture, archive[:]); status != tar.STATUS_OK {
		t.Fatalf("Writer_Init status = %v", status)
	}
	writer_fixture_write_header(&fixture, &header)
	writer_fixture_write(&fixture, []byte("too long"))
	if fixture.Writer.Status != tar.STATUS_WRITE_TOO_LONG {
		t.Fatalf("oversized write status = %v", fixture.Writer.Status)
	}
	if fixture.Writer.Count != 0 {
		t.Fatalf("oversized write count = %d", fixture.Writer.Count)
	}
	writer_fixture_close(&fixture)
	if fixture.Writer.Status != tar.STATUS_CONTENT_INCOMPLETE {
		t.Fatalf("incomplete close = %v", fixture.Writer.Status)
	}
}

// Test_Malformed_Input proves a bad checksum becomes parser status, not transport failure.
func test_malformed_input(t *testing.T) {
	var archive_storage [TEST_ARCHIVE_SIZE]byte
	archive := ustar_archive(archive_storage[:], []byte("entry.txt"), []byte("x"))
	archive[tar.NAME_FIELD_OFFSET] ^= byte(bits.WORD_8_MAXIMUM)
	var fixture reader_fixture
	if status := reader_fixture_init(&fixture, archive); status != tar.STATUS_OK {
		t.Fatalf("Reader_Init status = %v", status)
	}
	reader_fixture_next(&fixture)
	if fixture.Reader.Status != tar.STATUS_INPUT_INVALID {
		t.Fatalf("malformed input status = %v", fixture.Reader.Status)
	}
	if fixture.Reader.Completion.Error != nil {
		t.Fatalf("malformed input error = %v", fixture.Reader.Completion.Error)
	}
}

func test_writer_header_validation(t *testing.T) {
	var archive [TEST_ARCHIVE_SIZE]byte
	var fixture writer_fixture
	if status := writer_fixture_init(&fixture, archive[:]); status != tar.STATUS_OK {
		t.Fatalf("Writer_Init status = %v", status)
	}
	oversized_name := make([]byte, tar.HEADER_TEXT_SIZE_UNVALIDATED_MAXIMUM)
	header := tar.Header_Unvalidated{
		Format:    tar.Format_Unvalidated(tar.FORMAT_PAX),
		Type_Flag: tar.TYPE_REGULAR,
		Name:      oversized_name,
	}
	writer_fixture_write_header(&fixture, &header)
	if fixture.Writer.Status != tar.STATUS_FIELD_TOO_LONG {
		t.Fatalf("oversized header status = %v", fixture.Writer.Status)
	}
}

type deferred_stream struct {
	Memory     []byte
	Cursor     int
	Completion *nbio.Completion
	Buffer     []byte
	Mode       nbio.Stream_Mode
	Callback   nbio.Stream_Callback
	Retirement bool
}

func deferred_to_stream(state *deferred_stream) (stream nbio.Stream) {
	stream = nbio.Stream{State: unsafe.Pointer(state), Procedure: deferred_procedure}
	return stream
}

func deferred_procedure(
	state_pointer unsafe.Pointer, completion *nbio.Completion, mode nbio.Stream_Mode,
	buffer []byte, _ int64, _ nbio.Seek_From, callback nbio.Stream_Callback,
) {
	state := (*deferred_stream)(state_pointer)
	state.Completion = completion
	state.Buffer = buffer
	state.Mode = mode
	state.Callback = callback
	state.Retirement = true
}

func deferred_stream_retire(state *deferred_stream) {
	count := 0
	var err error
	if state.Mode == nbio.STREAM_MODE_READ {
		if state.Cursor == len(state.Memory) {
			err = nbio.Stream_EOF
		} else {
			count = copy(state.Buffer, state.Memory[state.Cursor:])
			state.Cursor += count
		}
	} else if state.Mode == nbio.STREAM_MODE_WRITE {
		count = copy(state.Memory[state.Cursor:], state.Buffer)
		state.Cursor += count
		if count != len(state.Buffer) {
			err = nbio.Stream_Short_Write
		}
	} else {
		err = nbio.Stream_Empty
	}
	completion := state.Completion
	callback := state.Callback
	state.Retirement = false
	state.Buffer = nil
	state.Callback = nbio.Stream_Callback{}
	completion.Data = count
	completion.Error = err
	nbio.Stream_Callback_Call(callback, completion)
}

// Test_Deferred_Stream proves no TAR call assumes callback retirement is inline.
func test_deferred_stream(t *testing.T) {
	var archive_storage [TEST_ARCHIVE_SIZE]byte
	archive := ustar_archive(archive_storage[:], []byte("entry.txt"), []byte("x"))
	stream_state := deferred_stream{Memory: archive}
	var block [tar.BLOCK_SIZE]byte
	var metadata [TEST_FIELD_SIZE]byte
	var reader tar.Reader
	status := tar.Reader_Init(
		&reader, deferred_to_stream(&stream_state),
		tar.Reader_Storage{Block: block[:], Metadata: metadata[:]},
	)
	if status != tar.STATUS_OK {
		t.Fatalf("Reader_Init status = %v", status)
	}
	var name [TEST_FIELD_SIZE]byte
	var user_name [TEST_FIELD_SIZE]byte
	var group_name [TEST_FIELD_SIZE]byte
	called := false
	tar.Reader_Next(
		&reader, &reader.Completion, tar.Header_Storage{
			Name: name[:], User_Name: user_name[:], Group_Name: group_name[:],
		},
		func(_ nbio.Completion_Handle) { called = true },
	)
	if called {
		t.Fatal("Reader_Next retired before deferred Stream")
	}
	if !stream_state.Retirement {
		t.Fatal("Reader_Next submitted no deferred Stream operation")
	}
	deferred_stream_retire(&stream_state)
	if !called {
		t.Fatal("deferred Reader_Next did not retire")
	}
	if reader.Status != tar.STATUS_OK {
		t.Fatalf("deferred Reader_Next status = %v", reader.Status)
	}
}

// Test_Defective_Stream_Count proves malicious transport cannot overrun caller storage.
func test_defective_stream_count(t *testing.T) {
	stream := nbio.Stream{Procedure: defective_read_procedure}
	var block [tar.BLOCK_SIZE]byte
	var reader tar.Reader
	status := tar.Reader_Init(
		&reader, stream, tar.Reader_Storage{Block: block[:]},
	)
	if status != tar.STATUS_OK {
		t.Fatalf("Reader_Init status = %v", status)
	}
	called := false
	tar.Reader_Next(
		&reader, &reader.Completion, tar.Header_Storage{},
		func(_ nbio.Completion_Handle) { called = true },
	)
	if !called {
		t.Fatal("defective Reader_Next did not retire")
	}
	if reader.Status != tar.STATUS_TRANSPORT_FAILED {
		t.Fatalf("defective count status = %v", reader.Status)
	}
	if reader.Completion.Error != nbio.Stream_Short_Buffer {
		t.Fatalf("defective count error = %v", reader.Completion.Error)
	}
	var writer tar.Writer
	var workspace [TEST_WRITER_WORKSPACE_BLOCK_COUNT * tar.BLOCK_SIZE]byte
	status = tar.Writer_Init(
		&writer, stream, workspace[:],
	)
	if status != tar.STATUS_OK {
		t.Fatalf("Writer_Init status = %v", status)
	}
	header := tar.Header_Unvalidated{
		Format:    tar.Format_Unvalidated(tar.FORMAT_USTAR),
		Type_Flag: tar.TYPE_REGULAR, Name: []byte("entry.txt"),
	}
	tar.Writer_Write_Header(
		&writer, &writer.Completion, &header, test_domain_callback,
	)
	if writer.Status != tar.STATUS_TRANSPORT_FAILED {
		t.Fatalf("defective writer status = %v", writer.Status)
	}
	if writer.Completion.Error != nbio.Stream_Invalid_Write {
		t.Fatalf("defective writer error = %v", writer.Completion.Error)
	}
}

// The shared byte helper bound is smaller than a valid PAX payload, so this guards the TAR-owned
// metadata domain rather than one convenient small fixture.
func test_large_pax_payload(t *testing.T) {
	name := make([]byte, TEST_LARGE_PAX_TEXT_SIZE)
	test_fill_bytes(name, 'n')
	archive := make([]byte, TEST_LARGE_PAX_TEXT_SIZE+3*tar.BLOCK_SIZE)
	workspace := make([]byte, tar.WRITER_STORAGE_SIZE_MAXIMUM)
	memory := nbio.Stream_Memory{Memory: archive}
	var writer tar.Writer
	status := tar.Writer_Init(
		&writer, nbio.Memory_To_Stream(&memory), workspace,
	)
	if status != tar.STATUS_OK {
		t.Fatalf("large PAX Writer_Init status = %v", status)
	}
	header := tar.Header_Unvalidated{
		Format:    tar.Format_Unvalidated(tar.FORMAT_PAX),
		Type_Flag: tar.TYPE_REGULAR, Name: name,
	}
	called := false
	tar.Writer_Write_Header(
		&writer, &writer.Completion, &header,
		func(_ nbio.Completion_Handle) { called = true },
	)
	if !called {
		t.Fatal("large PAX Writer_Write_Header did not retire")
	}
	if writer.Status != tar.STATUS_OK {
		t.Fatalf("large PAX Writer_Write_Header status = %v", writer.Status)
	}
}

// One complete maximum record proves the parser owns the advertised metadata bound, including
// the key width left after the count and separators.
func test_largest_pax_record(t *testing.T) {
	key := make([]byte, tar.PAX_KEY_SIZE_MAXIMUM)
	test_fill_bytes(key, 'k')
	records := make([]byte, tar.SPECIAL_FILE_SIZE_MAXIMUM)
	record_count := pax_record_into(records, key, nil)
	if record_count != len(records) {
		t.Fatalf("largest PAX record = %d; bound = %d", record_count, len(records))
	}
	archive_storage := make(
		[]byte, tar.SPECIAL_FILE_SIZE_MAXIMUM+4*tar.BLOCK_SIZE,
	)
	archive := tar_entry(
		archive_storage, 0, tar.FORMAT_USTAR, tar.TYPE_PAX_LOCAL,
		[]byte(tar.PAX_HEADER_NAME), records,
	)
	archive = tar_entry(
		archive_storage, len(archive), tar.FORMAT_USTAR,
		tar.TYPE_REGULAR, []byte("entry"), nil,
	)
	archive = archive_footer(archive_storage, len(archive))
	memory := nbio.Stream_Memory{Memory: archive}
	metadata := make([]byte, tar.READER_METADATA_SIZE_MAXIMUM)
	var block [tar.BLOCK_SIZE]byte
	var name [tar.NAME_FIELD_SIZE]byte
	var link_name [tar.LINK_NAME_FIELD_SIZE]byte
	var user_name [tar.USER_NAME_FIELD_SIZE]byte
	var group_name [tar.GROUP_NAME_FIELD_SIZE]byte
	var reader tar.Reader
	status := tar.Reader_Init(
		&reader, nbio.Memory_To_Stream(&memory),
		tar.Reader_Storage{Block: block[:], Metadata: metadata},
	)
	if status != tar.STATUS_OK {
		t.Fatalf("largest PAX Reader_Init status = %v", status)
	}
	called := false
	tar.Reader_Next(
		&reader, &reader.Completion, tar.Header_Storage{
			Name: name[:], Link_Name: link_name[:],
			User_Name: user_name[:], Group_Name: group_name[:],
		},
		func(_ nbio.Completion_Handle) { called = true },
	)
	if !called {
		t.Fatal("largest PAX Reader_Next did not retire")
	}
	if reader.Status != tar.STATUS_OK {
		t.Fatalf("largest PAX Reader_Next status = %v", reader.Status)
	}
}

func defective_read_procedure(
	_ unsafe.Pointer, completion *nbio.Completion, _ nbio.Stream_Mode, buffer []byte,
	_ int64, _ nbio.Seek_From, callback nbio.Stream_Callback,
) {
	completion.Data = len(buffer) + 1
	completion.Error = nil
	nbio.Stream_Callback_Call(callback, completion)
}

// Test_Allocation measures the injected boundaries in steady state.
func test_allocation(t *testing.T) {
	test_header_allocation(t)
	test_format_allocation(
		t, tar.Format_Unvalidated(tar.FORMAT_USTAR), []byte("entry.txt"), nil,
	)
	var name [TEST_LONG_NAME_SIZE]byte
	var link_name [TEST_LONG_NAME_SIZE]byte
	for index := range name {
		name[index] = 'n'
		link_name[index] = 'l'
	}
	test_format_allocation(
		t, tar.Format_Unvalidated(tar.FORMAT_PAX), name[:], link_name[:],
	)
	test_format_allocation(
		t, tar.Format_Unvalidated(tar.FORMAT_GNU), name[:], link_name[:],
	)
	test_gnu_reader_allocation(t, name[:], link_name[:])
}

func test_header_allocation(t *testing.T) {
	t.Helper()
	header := tar.Header_Unvalidated{
		Format:    tar.Format_Unvalidated(tar.FORMAT_USTAR),
		Type_Flag: tar.TYPE_REGULAR,
		Name:      []byte("entry.txt"), Size: TEST_ALLOCATION_CONTENT_SIZE,
	}
	var validated tar.Header
	var validation_status tar.Header_Validation_Status
	header_allocations := testing.AllocsPerRun(TEST_ALLOCATION_RUN_COUNT, func() {
		validated, validation_status = tar.Header_Validate(&header)
	})
	if header_allocations != 0 {
		t.Fatalf("Header_Validate allocated %v times; want 0", header_allocations)
	}
	if validation_status != tar.STATUS_OK {
		t.Fatalf("allocation Header_Validate status = %v", validation_status)
	}
	if len(validated.Name) != len(header.Name) {
		t.Fatalf("allocation Header_Validate name size = %d", len(validated.Name))
	}
}

func test_format_allocation(
	t *testing.T, format tar.Format_Unvalidated, name []byte, link_name []byte,
) {
	t.Helper()
	var archive [TEST_ARCHIVE_SIZE]byte
	archive_count := test_writer_allocation(t, archive[:], format, name, link_name)
	if format == tar.Format_Unvalidated(tar.FORMAT_GNU) {
		return
	}
	test_reader_allocation(t, archive[:archive_count], tar.Format(format))
}

func test_gnu_reader_allocation(t *testing.T, name []byte, link_name []byte) {
	t.Helper()
	var payload [TEST_FIELD_SIZE]byte
	var archive_storage [TEST_ARCHIVE_SIZE]byte
	archive := tar_entry(
		archive_storage[:], 0, tar.FORMAT_GNU, tar.TYPE_GNU_LONG_NAME,
		[]byte("././@LongLink"), append_nul(payload[:], name),
	)
	archive = tar_entry(
		archive_storage[:], len(archive), tar.FORMAT_GNU, tar.TYPE_GNU_LONG_LINK,
		[]byte("././@LongLink"), append_nul(payload[:], link_name),
	)
	archive = tar_entry(
		archive_storage[:], len(archive), tar.FORMAT_GNU, tar.TYPE_REGULAR,
		[]byte("placeholder"), []byte{'x'},
	)
	archive = archive_footer(archive_storage[:], len(archive))
	test_reader_allocation(t, archive, tar.FORMAT_GNU)
}

func test_reader_allocation(t *testing.T, archive []byte, format tar.Format) {
	t.Helper()
	var block [tar.BLOCK_SIZE]byte
	metadata := make([]byte, tar.READER_METADATA_SIZE_MAXIMUM)
	var name [TEST_FIELD_SIZE]byte
	var link_name [TEST_FIELD_SIZE]byte
	var destination [TEST_ALLOCATION_CONTENT_SIZE]byte
	var reader tar.Reader
	var reader_memory nbio.Stream_Memory
	var reader_status tar.Initialization_Status
	reader_allocations := testing.AllocsPerRun(TEST_ALLOCATION_RUN_COUNT, func() {
		reader_memory = nbio.Stream_Memory{Memory: archive}
		reader_status = tar.Reader_Init(
			&reader, nbio.Memory_To_Stream(&reader_memory),
			tar.Reader_Storage{Block: block[:], Metadata: metadata},
		)
		tar.Reader_Next(
			&reader, &reader.Completion, tar.Header_Storage{
				Name: name[:], Link_Name: link_name[:],
			},
			allocation_callback,
		)
		tar.Reader_Read(
			&reader, &reader.Completion, destination[:], allocation_callback,
		)
	})
	if reader_allocations != 0 {
		t.Fatalf("Reader allocated %v times; want 0", reader_allocations)
	}
	if reader_status != tar.STATUS_OK {
		t.Fatalf("allocation Reader_Init status = %v", reader_status)
	}
	if reader.Status != tar.STATUS_OK {
		t.Fatalf("allocation Reader status = %v", reader.Status)
	}
	if reader.Archive.Header.Format != format {
		t.Fatalf("allocation Reader format = %v", reader.Archive.Header.Format)
	}
}

func test_writer_allocation(
	t *testing.T, archive []byte, format tar.Format_Unvalidated,
	name []byte, link_name []byte,
) (archive_count int) {
	t.Helper()
	var workspace [TEST_WRITER_WORKSPACE_BLOCK_COUNT * tar.BLOCK_SIZE]byte
	var writer tar.Writer
	var writer_memory nbio.Stream_Memory
	source := []byte{'x'}
	header := tar.Header_Unvalidated{
		Format: format, Type_Flag: tar.TYPE_REGULAR,
		Name: name, Link_Name: link_name, Size: TEST_ALLOCATION_CONTENT_SIZE,
	}
	var writer_status tar.Initialization_Status
	writer_allocations := testing.AllocsPerRun(TEST_ALLOCATION_RUN_COUNT, func() {
		writer_memory = nbio.Stream_Memory{Memory: archive}
		writer_status = tar.Writer_Init(
			&writer, nbio.Memory_To_Stream(&writer_memory),
			workspace[:],
		)
		tar.Writer_Write_Header(
			&writer, &writer.Completion, &header, allocation_callback,
		)
		tar.Writer_Write(
			&writer, &writer.Completion, source, allocation_callback,
		)
		tar.Writer_Close(&writer, &writer.Completion, allocation_callback)
	})
	if writer_allocations != 0 {
		t.Fatalf("Writer allocated %v times; want 0", writer_allocations)
	}
	if writer_status != tar.STATUS_OK {
		t.Fatalf("allocation Writer_Init status = %v", writer_status)
	}
	if writer.Status != tar.STATUS_OK {
		t.Fatalf("allocation Writer status = %v", writer.Status)
	}
	return int(writer.Count)
}

func allocation_callback(_ nbio.Completion_Handle) { return }

func ustar_archive(storage []byte, name []byte, content []byte) (archive []byte) {
	archive = ustar_entry(storage, 0, name, content)
	return archive_footer(storage, len(archive))
}

func ustar_entry(
	storage []byte, position int, name []byte, content []byte,
) (archive []byte) {
	return tar_entry(
		storage, position, tar.FORMAT_USTAR, tar.TYPE_REGULAR, name, content,
	)
}

func tar_entry(
	storage []byte, position int, format tar.Format, type_flag tar.Type_Flag,
	name []byte, content []byte,
) (archive []byte) {
	header := storage[position : position+tar.BLOCK_SIZE]
	for index := range header {
		header[index] = 0
	}
	tar_header_name(header, name)
	tar_header_numeric(header, len(content))
	header[tar.TYPE_FLAG_FIELD_OFFSET] = byte(type_flag)
	tar_header_format(header, format)
	tar_header_checksum(header)
	data_start := position + tar.BLOCK_SIZE
	copy(storage[data_start:], content)
	end := data_start + padded_size(len(content))
	for index := data_start + len(content); index < end; index++ {
		storage[index] = 0
	}
	return storage[:end]
}

func tar_header_name(header []byte, name []byte) {
	prefix_count, name_start := split_name(name)
	copy(
		header[tar.NAME_FIELD_OFFSET:tar.NAME_FIELD_OFFSET+tar.NAME_FIELD_SIZE],
		name[name_start:],
	)
	copy(
		header[tar.PREFIX_FIELD_OFFSET:tar.PREFIX_FIELD_OFFSET+tar.PREFIX_FIELD_SIZE],
		name[:prefix_count],
	)
}

func tar_header_numeric(header []byte, content_size int) {
	user_identifier_end := tar.USER_IDENTIFIER_FIELD_OFFSET +
		tar.USER_IDENTIFIER_FIELD_SIZE
	group_identifier_end := tar.GROUP_IDENTIFIER_FIELD_OFFSET +
		tar.GROUP_IDENTIFIER_FIELD_SIZE
	entry_size_end := tar.ENTRY_SIZE_FIELD_OFFSET + tar.ENTRY_SIZE_FIELD_SIZE
	timestamp_end := tar.TIMESTAMP_FIELD_OFFSET + tar.TIMESTAMP_FIELD_SIZE
	format_octal(
		header[tar.MODE_FIELD_OFFSET:tar.MODE_FIELD_OFFSET+tar.MODE_FIELD_SIZE],
		TEST_FILE_MODE,
	)
	format_octal(
		header[tar.USER_IDENTIFIER_FIELD_OFFSET:user_identifier_end],
		TEST_USER_IDENTIFIER,
	)
	format_octal(
		header[tar.GROUP_IDENTIFIER_FIELD_OFFSET:group_identifier_end],
		TEST_GROUP_IDENTIFIER,
	)
	format_octal(
		header[tar.ENTRY_SIZE_FIELD_OFFSET:entry_size_end],
		int64(content_size),
	)
	format_octal(
		header[tar.TIMESTAMP_FIELD_OFFSET:timestamp_end],
		TEST_MODIFICATION_SECONDS,
	)
}

func tar_header_format(header []byte, format tar.Format) {
	magic_end := tar.MAGIC_FIELD_OFFSET + tar.MAGIC_FIELD_SIZE
	version_end := tar.VERSION_FIELD_OFFSET + tar.VERSION_FIELD_SIZE
	user_name_end := tar.USER_NAME_FIELD_OFFSET + tar.USER_NAME_FIELD_SIZE
	group_name_end := tar.GROUP_NAME_FIELD_OFFSET + tar.GROUP_NAME_FIELD_SIZE
	switch format {
	case tar.FORMAT_USTAR, tar.FORMAT_PAX:
		copy(
			header[tar.MAGIC_FIELD_OFFSET:magic_end],
			tar.USTAR_MAGIC,
		)
		copy(
			header[tar.VERSION_FIELD_OFFSET:version_end],
			tar.USTAR_VERSION,
		)
		copy(
			header[tar.USER_NAME_FIELD_OFFSET:user_name_end],
			[]byte("user"),
		)
		copy(
			header[tar.GROUP_NAME_FIELD_OFFSET:group_name_end],
			[]byte("group"),
		)
	case tar.FORMAT_GNU:
		copy(
			header[tar.MAGIC_FIELD_OFFSET:magic_end],
			tar.GNU_MAGIC,
		)
		copy(
			header[tar.VERSION_FIELD_OFFSET:version_end],
			tar.GNU_VERSION,
		)
	}
}

func tar_header_checksum(header []byte) {
	checksum_end := tar.CHECKSUM_FIELD_OFFSET + tar.CHECKSUM_FIELD_SIZE
	for index := tar.CHECKSUM_FIELD_OFFSET; index < checksum_end; index++ {
		header[index] = ' '
	}
	format_checksum(
		header[tar.CHECKSUM_FIELD_OFFSET:tar.CHECKSUM_FIELD_OFFSET+tar.CHECKSUM_FIELD_SIZE],
		checksum(header),
	)
}

func pax_record_into(storage []byte, key []byte, value []byte) (count int) {
	body_size := tar.PAX_COUNT_SEPARATOR_SIZE + len(key) +
		tar.PAX_KEY_SEPARATOR_SIZE + len(value) + tar.PAX_RECORD_TERMINATOR_SIZE
	count = body_size + decimal_size(body_size+tar.PAX_COUNT_SEPARATOR_SIZE)
	for decimal_size(count)+body_size != count {
		count = decimal_size(count) + body_size
	}
	position := format_decimal(storage, int64(count))
	storage[position] = ' '
	position += tar.PAX_COUNT_SEPARATOR_SIZE
	position += copy(storage[position:], key)
	storage[position] = '='
	position += tar.PAX_KEY_SEPARATOR_SIZE
	position += copy(storage[position:], value)
	storage[position] = '\n'
	return count
}

func append_nul(storage []byte, value []byte) (result []byte) {
	count := copy(storage, value)
	storage[count] = 0
	return storage[:count+tar.GNU_LONG_FIELD_TERMINATOR_SIZE]
}

func archive_footer(storage []byte, position int) (archive []byte) {
	end := position + tar.ARCHIVE_FOOTER_SIZE
	for index := position; index < end; index++ {
		storage[index] = 0
	}
	return storage[:end]
}

func split_name(name []byte) (prefix_count int, name_start int) {
	if len(name) <= tar.NAME_FIELD_SIZE {
		return 0, 0
	}
	for index := len(name) - tar.TYPE_FLAG_FIELD_SIZE; index >= 0; index-- {
		if name[index] == '/' {
			return index, index + tar.TYPE_FLAG_FIELD_SIZE
		}
	}
	return 0, 0
}

func padded_size(size int) (padded int) {
	return (size + tar.PADDING_SIZE_MAXIMUM) &^ tar.PADDING_SIZE_MAXIMUM
}

func decimal_size(value int) (size int) {
	size = 1
	for value >= tar.DECIMAL_BASE {
		value /= tar.DECIMAL_BASE
		size++
	}
	return size
}

func format_decimal(storage []byte, value int64) (count int) {
	divisor := int64(1)
	for divisor <= value/tar.DECIMAL_BASE {
		divisor *= tar.DECIMAL_BASE
	}
	for divisor != 0 {
		storage[count] = byte(value/divisor) + '0'
		value %= divisor
		divisor /= tar.DECIMAL_BASE
		count++
	}
	return count
}

func format_octal(field []byte, value int64) {
	for index := range field {
		field[index] = 0
	}
	for position := len(field) - tar.VERSION_FIELD_SIZE; position >= 0; position-- {
		field[position] = byte(value&(tar.OCTAL_BASE-1)) + '0'
		value >>= tar.OCTAL_BITS_PER_DIGIT
	}
}

func format_checksum(field []byte, value int64) {
	format_octal(field, value)
	field[len(field)-1] = ' '
}

func checksum(header []byte) (sum int64) {
	for _, value := range header {
		sum += int64(value)
	}
	return sum
}
