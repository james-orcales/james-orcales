// Package zip decodes bounded classic ZIP entries into caller-owned storage.
package zip

import (
	"unsafe"

	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/compress/flate"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/hash/crc32"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/aver/default"
	"local/james-orcales/shared/sim/nbio"
	"local/james-orcales/shared/sim/time"
	"local/james-orcales/shared/slices"
	"local/james-orcales/shared/strconv"
	"local/james-orcales/shared/unicode/utf8"
)

// ZIP_SIGNATURE_PREFIX derives the shared PK signature prefix from wire bytes.
const ZIP_SIGNATURE_PREFIX = binary.Word_32('P') |
	binary.Word_32('K')<<bits.BIT_COUNT_8_MAXIMUM

// ZIP_SIGNATURE_KIND_SHIFT locates the record-kind byte after the PK prefix.
const ZIP_SIGNATURE_KIND_SHIFT = bits.BIT_COUNT_16_MAXIMUM

// ZIP_SIGNATURE_SUBKIND_SHIFT locates the final record identity byte.
const ZIP_SIGNATURE_SUBKIND_SHIFT = ZIP_SIGNATURE_KIND_SHIFT + bits.BIT_COUNT_8_MAXIMUM

// ZIP_LOCAL_SIGNATURE_KIND is the local-header record identity byte.
const ZIP_LOCAL_SIGNATURE_KIND = 3

// ZIP_LOCAL_SIGNATURE_SUBKIND is the local-header record identity byte.
const ZIP_LOCAL_SIGNATURE_SUBKIND = 4

// ZIP_CENTRAL_SIGNATURE_KIND is the central-header record identity byte.
const ZIP_CENTRAL_SIGNATURE_KIND = 1

// ZIP_CENTRAL_SIGNATURE_SUBKIND is the central-header record identity byte.
const ZIP_CENTRAL_SIGNATURE_SUBKIND = 2

// ZIP_DESCRIPTOR_SIGNATURE_KIND is the descriptor record identity byte.
const ZIP_DESCRIPTOR_SIGNATURE_KIND = 7

// ZIP_DESCRIPTOR_SIGNATURE_SUBKIND is the descriptor record identity byte.
const ZIP_DESCRIPTOR_SIGNATURE_SUBKIND = 8

// ZIP_DIRECTORY_END_SIGNATURE_KIND is the directory-end record identity byte.
const ZIP_DIRECTORY_END_SIGNATURE_KIND = 5

// ZIP_DIRECTORY_END_SIGNATURE_SUBKIND is the directory-end record identity byte.
const ZIP_DIRECTORY_END_SIGNATURE_SUBKIND = 6

// ZIP_VERSION_2_0 is the classic Store and Deflate extraction version.
const ZIP_VERSION_2_0 = 2 * strconv.DECIMAL_BASE

// LOCAL_EXTRACTOR_POSITION follows the local signature.
const LOCAL_EXTRACTOR_POSITION = binary.UINT_32_SIZE

// WRITER_LOCAL_FLAGS_POSITION follows the local extractor version.
const WRITER_LOCAL_FLAGS_POSITION = LOCAL_EXTRACTOR_POSITION + binary.UINT_16_SIZE

// WRITER_LOCAL_METHOD_POSITION follows local flags.
const WRITER_LOCAL_METHOD_POSITION = WRITER_LOCAL_FLAGS_POSITION + binary.UINT_16_SIZE

// LOCAL_MODIFIED_TIME_POSITION follows local compression method.
const LOCAL_MODIFIED_TIME_POSITION = WRITER_LOCAL_METHOD_POSITION + binary.UINT_16_SIZE

// LOCAL_MODIFIED_DATE_POSITION follows local modification time.
const LOCAL_MODIFIED_DATE_POSITION = LOCAL_MODIFIED_TIME_POSITION + binary.UINT_16_SIZE

// LOCAL_CHECKSUM_POSITION follows local modification date.
const LOCAL_CHECKSUM_POSITION = LOCAL_MODIFIED_DATE_POSITION + binary.UINT_16_SIZE

// LOCAL_COMPRESSED_SIZE_POSITION follows local checksum.
const LOCAL_COMPRESSED_SIZE_POSITION = LOCAL_CHECKSUM_POSITION + binary.UINT_32_SIZE

// LOCAL_UNCOMPRESSED_SIZE_POSITION follows local compressed size.
const LOCAL_UNCOMPRESSED_SIZE_POSITION = LOCAL_COMPRESSED_SIZE_POSITION + binary.UINT_32_SIZE

// WRITER_LOCAL_NAME_SIZE_POSITION follows local uncompressed size.
const WRITER_LOCAL_NAME_SIZE_POSITION = LOCAL_UNCOMPRESSED_SIZE_POSITION + binary.UINT_32_SIZE

// WRITER_LOCAL_EXTRA_SIZE_POSITION follows local name size.
const WRITER_LOCAL_EXTRA_SIZE_POSITION = WRITER_LOCAL_NAME_SIZE_POSITION + binary.UINT_16_SIZE

// LOCAL_HEADER_SIZE follows the final fixed local field.
const LOCAL_HEADER_SIZE = WRITER_LOCAL_EXTRA_SIZE_POSITION + binary.UINT_16_SIZE

// WRITER_CENTRAL_CREATOR_POSITION follows the central signature.
const WRITER_CENTRAL_CREATOR_POSITION = binary.UINT_32_SIZE

// WRITER_CENTRAL_EXTRACTOR_POSITION follows the central creator version.
const WRITER_CENTRAL_EXTRACTOR_POSITION = WRITER_CENTRAL_CREATOR_POSITION + binary.UINT_16_SIZE

// CENTRAL_FLAGS_POSITION follows the central extractor version.
const CENTRAL_FLAGS_POSITION = WRITER_CENTRAL_EXTRACTOR_POSITION + binary.UINT_16_SIZE

// CENTRAL_METHOD_POSITION follows central flags.
const CENTRAL_METHOD_POSITION = CENTRAL_FLAGS_POSITION + binary.UINT_16_SIZE

// CENTRAL_MODIFIED_TIME_POSITION follows central compression method.
const CENTRAL_MODIFIED_TIME_POSITION = CENTRAL_METHOD_POSITION + binary.UINT_16_SIZE

// CENTRAL_MODIFIED_DATE_POSITION follows central modification time.
const CENTRAL_MODIFIED_DATE_POSITION = CENTRAL_MODIFIED_TIME_POSITION + binary.UINT_16_SIZE

// CENTRAL_CHECKSUM_POSITION follows central modification date.
const CENTRAL_CHECKSUM_POSITION = CENTRAL_MODIFIED_DATE_POSITION + binary.UINT_16_SIZE

// CENTRAL_COMPRESSED_SIZE_POSITION follows central checksum.
const CENTRAL_COMPRESSED_SIZE_POSITION = CENTRAL_CHECKSUM_POSITION + binary.UINT_32_SIZE

// CENTRAL_UNCOMPRESSED_SIZE_POSITION follows central compressed size.
const CENTRAL_UNCOMPRESSED_SIZE_POSITION = CENTRAL_COMPRESSED_SIZE_POSITION + binary.UINT_32_SIZE

// WRITER_CENTRAL_NAME_SIZE_POSITION follows central uncompressed size.
const WRITER_CENTRAL_NAME_SIZE_POSITION = CENTRAL_UNCOMPRESSED_SIZE_POSITION + binary.UINT_32_SIZE

// WRITER_CENTRAL_EXTRA_SIZE_POSITION follows central name size.
const WRITER_CENTRAL_EXTRA_SIZE_POSITION = WRITER_CENTRAL_NAME_SIZE_POSITION +
	binary.UINT_16_SIZE

// WRITER_CENTRAL_COMMENT_SIZE_POSITION follows central extra size.
const WRITER_CENTRAL_COMMENT_SIZE_POSITION = WRITER_CENTRAL_EXTRA_SIZE_POSITION +
	binary.UINT_16_SIZE

// CENTRAL_DISK_POSITION follows central comment size.
const CENTRAL_DISK_POSITION = WRITER_CENTRAL_COMMENT_SIZE_POSITION + binary.UINT_16_SIZE

// CENTRAL_INTERNAL_ATTRIBUTES_POSITION follows central disk number.
const CENTRAL_INTERNAL_ATTRIBUTES_POSITION = CENTRAL_DISK_POSITION + binary.UINT_16_SIZE

// WRITER_CENTRAL_EXTERNAL_ATTRIBUTES_POSITION follows central internal attributes.
const WRITER_CENTRAL_EXTERNAL_ATTRIBUTES_POSITION = CENTRAL_INTERNAL_ATTRIBUTES_POSITION +
	binary.UINT_16_SIZE

// WRITER_CENTRAL_LOCAL_OFFSET_POSITION follows central external attributes.
const WRITER_CENTRAL_LOCAL_OFFSET_POSITION = WRITER_CENTRAL_EXTERNAL_ATTRIBUTES_POSITION +
	binary.UINT_32_SIZE

// CENTRAL_LOCAL_OFFSET_POSITION shares the writer-computed wire coordinate.
const CENTRAL_LOCAL_OFFSET_POSITION = WRITER_CENTRAL_LOCAL_OFFSET_POSITION

// CENTRAL_HEADER_SIZE follows the final fixed central field.
const CENTRAL_HEADER_SIZE = CENTRAL_LOCAL_OFFSET_POSITION + binary.UINT_32_SIZE

// DIRECTORY_END_DISK_POSITION follows the directory-end signature.
const DIRECTORY_END_DISK_POSITION = binary.UINT_32_SIZE

// DIRECTORY_END_CENTRAL_DISK_POSITION follows the current disk number.
const DIRECTORY_END_CENTRAL_DISK_POSITION = DIRECTORY_END_DISK_POSITION + binary.UINT_16_SIZE

// DIRECTORY_END_DISK_ENTRY_COUNT_POSITION follows the central disk number.
const DIRECTORY_END_DISK_ENTRY_COUNT_POSITION = DIRECTORY_END_CENTRAL_DISK_POSITION +
	binary.UINT_16_SIZE

// DIRECTORY_END_ENTRY_COUNT_POSITION follows the per-disk entry count.
const DIRECTORY_END_ENTRY_COUNT_POSITION = DIRECTORY_END_DISK_ENTRY_COUNT_POSITION +
	binary.UINT_16_SIZE

// DIRECTORY_END_CENTRAL_SIZE_POSITION follows the total entry count.
const DIRECTORY_END_CENTRAL_SIZE_POSITION = DIRECTORY_END_ENTRY_COUNT_POSITION +
	binary.UINT_16_SIZE

// DIRECTORY_END_CENTRAL_OFFSET_POSITION follows the central size.
const DIRECTORY_END_CENTRAL_OFFSET_POSITION = DIRECTORY_END_CENTRAL_SIZE_POSITION +
	binary.UINT_32_SIZE

// DIRECTORY_END_COMMENT_SIZE_POSITION follows the central offset.
const DIRECTORY_END_COMMENT_SIZE_POSITION = DIRECTORY_END_CENTRAL_OFFSET_POSITION +
	binary.UINT_32_SIZE

// WRITER_DIRECTORY_END_SIZE follows the final fixed directory-end field.
const WRITER_DIRECTORY_END_SIZE = DIRECTORY_END_COMMENT_SIZE_POSITION + binary.UINT_16_SIZE

// DATA_DESCRIPTOR_CHECKSUM_POSITION follows the optional signature.
const DATA_DESCRIPTOR_CHECKSUM_POSITION = binary.UINT_32_SIZE

// DATA_DESCRIPTOR_COMPRESSED_SIZE_POSITION follows the signed checksum.
const DATA_DESCRIPTOR_COMPRESSED_SIZE_POSITION = DATA_DESCRIPTOR_CHECKSUM_POSITION +
	binary.UINT_32_SIZE

// DATA_DESCRIPTOR_UNCOMPRESSED_SIZE_POSITION follows the signed compressed size.
const DATA_DESCRIPTOR_UNCOMPRESSED_SIZE_POSITION = DATA_DESCRIPTOR_COMPRESSED_SIZE_POSITION +
	binary.UINT_32_SIZE

// DATA_DESCRIPTOR_SIZE is checksum plus two classic size fields.
const DATA_DESCRIPTOR_SIZE = binary.UINT_32_SIZE * 3

// WRITER_DESCRIPTOR_SIGNATURE_SIZE is the optional descriptor signature.
const WRITER_DESCRIPTOR_SIGNATURE_SIZE = binary.UINT_32_SIZE

// WRITER_DESCRIPTOR_SIZE adds the optional signature to the descriptor body.
const WRITER_DESCRIPTOR_SIZE = WRITER_DESCRIPTOR_SIGNATURE_SIZE + DATA_DESCRIPTOR_SIZE

// EXTENDED_TIMESTAMP_DATA_SIZE_POSITION follows its identifier.
const EXTENDED_TIMESTAMP_DATA_SIZE_POSITION = binary.UINT_16_SIZE

// EXTENDED_TIMESTAMP_FLAGS_POSITION follows its data-size field.
const EXTENDED_TIMESTAMP_FLAGS_POSITION = EXTENDED_TIMESTAMP_DATA_SIZE_POSITION +
	binary.UINT_16_SIZE

// EXTENDED_TIMESTAMP_SECONDS_POSITION follows its flag byte.
const EXTENDED_TIMESTAMP_SECONDS_POSITION = EXTENDED_TIMESTAMP_FLAGS_POSITION + binary.UINT_8_SIZE

// EXTENDED_TIMESTAMP_DATA_SIZE is flags plus modification seconds.
const EXTENDED_TIMESTAMP_DATA_SIZE = binary.UINT_8_SIZE + binary.UINT_32_SIZE

// EXTENDED_TIMESTAMP_HEADER_SIZE is identifier plus data-size field.
const EXTENDED_TIMESTAMP_HEADER_SIZE = binary.UINT_16_SIZE + binary.UINT_16_SIZE

// EXTENDED_TIMESTAMP_DATA_SECONDS_POSITION follows the data flag byte.
const EXTENDED_TIMESTAMP_DATA_SECONDS_POSITION = binary.UINT_8_SIZE

// EXTENDED_TIMESTAMP_SIZE includes identifier, data size, and data.
const EXTENDED_TIMESTAMP_SIZE = EXTENDED_TIMESTAMP_HEADER_SIZE + EXTENDED_TIMESTAMP_DATA_SIZE

// EXTENDED_TIMESTAMP_MODIFIED_FLAG marks modification seconds as present.
const EXTENDED_TIMESTAMP_MODIFIED_FLAG = 1

// WIRE_TIMESTAMP_PRESENT_OFFSET reserves zero for absent metadata.
const WIRE_TIMESTAMP_PRESENT_OFFSET = binary.UINT_8_SIZE

// STATUS_MINIMUM anchors output coverage at successful decode.
const STATUS_MINIMUM uint8 = bits.WORD_8_MINIMUM

// STATUS_MAXIMUM closes output coverage at unsupported codec.
const STATUS_MAXIMUM uint8 = uint8(STATUS_METHOD_UNSUPPORTED)

// Status keeps failure state scalar and caller-owned.
type Status uint8

// Status_Invariants closes result domain over every rejection class.
func Status_Invariants(value Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), STATUS_MINIMUM, STATUS_MAXIMUM).
		Ensure()
}

// Validation_Status excludes failures unavailable before archive processing.
type Validation_Status Status

// Validation_Status_Invariants keeps pure validation outcomes exact.
func Validation_Status_Invariants(
	value Validation_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(STATUS_OK), uint8(STATUS_INPUT_INVALID)).
		Ensure()
}

// Found_Status separates malformed input from absent selection.
type Found_Status Status

// Found_Status_Invariants keeps lookup outcomes exact.
func Found_Status_Invariants(value Found_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_INPUT_INVALID),
			uint8(STATUS_ENTRY_NOT_FOUND),
		).
		Ensure()
}

// Bounded_Status includes malformed input and exhausted output only.
type Bounded_Status Status

// Bounded_Status_Invariants keeps bounded transformation outcomes exact.
func Bounded_Status_Invariants(
	value Bounded_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_INPUT_INVALID),
			uint8(STATUS_OUTPUT_TOO_SMALL),
		).
		Ensure()
}

// Directory_Status includes lookup and caller-capacity failures.
type Directory_Status Status

// Directory_Status_Invariants keeps filesystem traversal outcomes exact.
func Directory_Status_Invariants(
	value Directory_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_INPUT_INVALID),
			uint8(STATUS_ENTRY_NOT_FOUND), uint8(STATUS_OUTPUT_TOO_SMALL),
		).
		Ensure()
}

// Creation_Status excludes lookup failure from Writer mutations.
type Creation_Status Status

// Creation_Status_Invariants keeps Writer creation outcomes exact.
func Creation_Status_Invariants(
	value Creation_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_INPUT_INVALID),
			uint8(STATUS_OUTPUT_TOO_SMALL), uint8(STATUS_METHOD_UNSUPPORTED),
		).
		Ensure()
}

// Preparation_Status excludes storage exhaustion handled after header staging.
type Preparation_Status Status

// Preparation_Status_Invariants keeps header staging outcomes exact.
func Preparation_Status_Invariants(
	value Preparation_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_INPUT_INVALID),
			uint8(STATUS_METHOD_UNSUPPORTED),
		).
		Ensure()
}

// Capacity_Status reports success or one exhausted caller buffer.
type Capacity_Status Status

// Capacity_Status_Invariants keeps two noncontiguous public values exact.
func Capacity_Status_Invariants(
	value Capacity_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_OUTPUT_TOO_SMALL),
		).
		Ensure()
}

// Writer_Header_Status reports supported or unsupported compression only.
type Writer_Header_Status Status

// Writer_Header_Status_Invariants keeps method validation exact.
func Writer_Header_Status_Invariants(
	value Writer_Header_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_METHOD_UNSUPPORTED),
		).
		Ensure()
}

// STATUS_OK separates successful zero-byte members from failure.
const STATUS_OK Status = Status(STATUS_MINIMUM)

// STATUS_INPUT_INVALID stops hostile metadata before unsafe slicing.
const STATUS_INPUT_INVALID Status = STATUS_OK + 1

// STATUS_ENTRY_NOT_FOUND keeps absence distinct from corruption.
const STATUS_ENTRY_NOT_FOUND Status = STATUS_INPUT_INVALID + 1

// STATUS_OUTPUT_TOO_SMALL protects caller output boundary.
const STATUS_OUTPUT_TOO_SMALL Status = STATUS_ENTRY_NOT_FOUND + 1

// STATUS_METHOD_UNSUPPORTED rejects codecs outside bounded scope.
const STATUS_METHOD_UNSUPPORTED Status = STATUS_OUTPUT_TOO_SMALL + 1

// ENTRY_COUNT_MAXIMUM follows smallest possible classic central header.
const ENTRY_COUNT_MAXIMUM = (bytes.SLICE_SIZE_MAXIMUM - WRITER_DIRECTORY_END_SIZE) /
	CENTRAL_HEADER_SIZE

// FILE_SYSTEM_CENTRAL_BYTES_MAXIMUM leaves the directory end intact.
const FILE_SYSTEM_CENTRAL_BYTES_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM - ARCHIVE_SIZE_MINIMUM

// FILE_SYSTEM_ENTRY_COUNT_MAXIMUM requires one byte for every valid member name.
const FILE_SYSTEM_ENTRY_COUNT_MAXIMUM = FILE_SYSTEM_CENTRAL_BYTES_MAXIMUM /
	(CENTRAL_HEADER_SIZE + SELECTED_NAME_SIZE_MINIMUM)

// FILE_SYSTEM_HEADER_INDEX_MAXIMUM is final valid named member position.
const FILE_SYSTEM_HEADER_INDEX_MAXIMUM = FILE_SYSTEM_ENTRY_COUNT_MAXIMUM -
	SELECTED_NAME_SIZE_MINIMUM

// FILE_SYSTEM_WALK_COUNT_MAXIMUM is root plus maximum one-byte path components.
const FILE_SYSTEM_WALK_COUNT_MAXIMUM = (HEADER_NAME_SIZE_MAXIMUM+len("/"))/
	(len("/")+SELECTED_NAME_SIZE_MINIMUM) + len(".")

// FILE_SYSTEM_DIRECTORY_PREVIOUS_COUNT_MAXIMUM leaves one unseen archive member.
const FILE_SYSTEM_DIRECTORY_PREVIOUS_COUNT_MAXIMUM = FILE_SYSTEM_ENTRY_COUNT_MAXIMUM -
	SELECTED_NAME_SIZE_MINIMUM

// FILE_SYSTEM_WALK_PREVIOUS_COUNT_MAXIMUM leaves one path component to add.
const FILE_SYSTEM_WALK_PREVIOUS_COUNT_MAXIMUM = FILE_SYSTEM_WALK_COUNT_MAXIMUM -
	len(".")

// FILE_SYSTEM_PARENT_PATH_SIZE_MAXIMUM leaves slash and one child byte.
const FILE_SYSTEM_PARENT_PATH_SIZE_MAXIMUM = HEADER_NAME_SIZE_MAXIMUM -
	len("/") - SELECTED_NAME_SIZE_MINIMUM

// ARCHIVE_SIZE_MINIMUM keeps one complete directory end record.
const ARCHIVE_SIZE_MINIMUM = WRITER_DIRECTORY_END_SIZE

// ARCHIVE_SIZE_MAXIMUM follows shared caller-storage boundary.
const ARCHIVE_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// DESTINATION_SIZE_MINIMUM admits empty members.
const DESTINATION_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// DESTINATION_SIZE_MAXIMUM follows shared caller-storage boundary.
const DESTINATION_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// ENTRY_SUFFIX_SIZE_MINIMUM rejects empty lookup ambiguity.
const ENTRY_SUFFIX_SIZE_MINIMUM = bytes.TEXT_SIZE_MINIMUM + binary.UINT_8_SIZE

// ENTRY_SUFFIX_SIZE_MAXIMUM follows shared text boundary.
const ENTRY_SUFFIX_SIZE_MAXIMUM = bytes.TEXT_SIZE_MAXIMUM

// MEMBER_NAME_SIZE_MINIMUM admits nameless hostile central metadata.
const MEMBER_NAME_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// MEMBER_NAME_SIZE_MAXIMUM reserves one central prefix plus directory end.
const MEMBER_NAME_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM -
	CENTRAL_HEADER_SIZE - ARCHIVE_SIZE_MINIMUM

// POSITION_MINIMUM begins caller archive storage.
const POSITION_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// DIRECTORY_POSITION_MAXIMUM leaves complete directory end after its start.
const DIRECTORY_POSITION_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM - ARCHIVE_SIZE_MINIMUM

// CENTRAL_SIZE_MAXIMUM leaves complete directory end after central bytes.
const CENTRAL_SIZE_MAXIMUM = DIRECTORY_POSITION_MAXIMUM

// CENTRAL_OFFSET_MAXIMUM covers complete bounded ZIP segment coordinate.
const CENTRAL_OFFSET_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// CENTRAL_TAIL_SIZE_MAXIMUM leaves fixed central prefix and directory end.
const CENTRAL_TAIL_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM -
	CENTRAL_HEADER_SIZE - ARCHIVE_SIZE_MINIMUM

// CENTRAL_END_MINIMUM keeps one fixed central header.
const CENTRAL_END_MINIMUM = CENTRAL_HEADER_SIZE

// CENTRAL_END_MAXIMUM leaves complete directory end.
const CENTRAL_END_MAXIMUM = DIRECTORY_POSITION_MAXIMUM

// CENTRAL_POSITION_MAXIMUM leaves one fixed central header before end.
const CENTRAL_POSITION_MAXIMUM = CENTRAL_END_MAXIMUM - CENTRAL_HEADER_SIZE

// SELECTED_CENTRAL_START_MAXIMUM leaves one nonempty-name central entry.
const SELECTED_CENTRAL_START_MAXIMUM = CENTRAL_END_MAXIMUM -
	CENTRAL_HEADER_SIZE - SELECTED_NAME_SIZE_MINIMUM

// PAYLOAD_POSITION_MINIMUM follows one local prefix and nonempty name.
const PAYLOAD_POSITION_MINIMUM = LOCAL_HEADER_SIZE + SELECTED_NAME_SIZE_MINIMUM

// PAYLOAD_POSITION_MAXIMUM ends before selected central entry.
const PAYLOAD_POSITION_MAXIMUM = SELECTED_CENTRAL_START_MAXIMUM

// CENTRAL_ARCHIVE_SIZE_MINIMUM keeps one central header and directory end.
const CENTRAL_ARCHIVE_SIZE_MINIMUM = CENTRAL_HEADER_SIZE + ARCHIVE_SIZE_MINIMUM

// SELECTED_ARCHIVE_SIZE_MINIMUM keeps one nonempty-name central entry.
const SELECTED_ARCHIVE_SIZE_MINIMUM = CENTRAL_HEADER_SIZE +
	SELECTED_NAME_SIZE_MINIMUM + ARCHIVE_SIZE_MINIMUM

// PAYLOAD_ARCHIVE_SIZE_MINIMUM keeps empty local and selected central entries.
const PAYLOAD_ARCHIVE_SIZE_MINIMUM = LOCAL_HEADER_SIZE + CENTRAL_HEADER_SIZE +
	2*SELECTED_NAME_SIZE_MINIMUM + ARCHIVE_SIZE_MINIMUM

// RAW_CONTENT_SIZE_MAXIMUM leaves local, central, names, and EOCD metadata.
const RAW_CONTENT_SIZE_MAXIMUM = ARCHIVE_SIZE_MAXIMUM - PAYLOAD_ARCHIVE_SIZE_MINIMUM

// DESCRIPTOR_ARCHIVE_SIZE_MINIMUM adds one unsigned classic descriptor.
const DESCRIPTOR_ARCHIVE_SIZE_MINIMUM = PAYLOAD_ARCHIVE_SIZE_MINIMUM + DATA_DESCRIPTOR_SIZE

// DESCRIPTOR_POSITION_MAXIMUM leaves unsigned descriptor before central entry.
const DESCRIPTOR_POSITION_MAXIMUM = PAYLOAD_POSITION_MAXIMUM - DATA_DESCRIPTOR_SIZE

// DESCRIPTOR_CENTRAL_START_MINIMUM follows minimum payload plus descriptor.
const DESCRIPTOR_CENTRAL_START_MINIMUM = PAYLOAD_POSITION_MINIMUM + DATA_DESCRIPTOR_SIZE

// ENTRY_COUNT_MINIMUM admits empty archives.
const ENTRY_COUNT_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// OUTPUT_COUNT_MINIMUM admits empty members.
const OUTPUT_COUNT_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// OUTPUT_COUNT_MAXIMUM follows shared caller-storage boundary.
const OUTPUT_COUNT_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// FLAG_UTF8 permits standard Unicode name declaration.
const FLAG_UTF8 = 1 << 11

// FLAG_DEFLATE_OPTION_BIT_COUNT covers both standard codec hint bits.
const FLAG_DEFLATE_OPTION_BIT_COUNT = 2

// FLAG_DEFLATE_OPTION_SHIFT locates codec hints after encryption state.
const FLAG_DEFLATE_OPTION_SHIFT = 1

// FLAG_DEFLATE_OPTIONS permits standard DEFLATE effort hints.
const FLAG_DEFLATE_OPTIONS = ((1 << FLAG_DEFLATE_OPTION_BIT_COUNT) - 1) <<
	FLAG_DEFLATE_OPTION_SHIFT

// FLAG_DATA_DESCRIPTOR permits streaming local headers.
const FLAG_DATA_DESCRIPTOR = 1 << 3

// FLAGS_COMMON rejects every flag without bounded local meaning.
const FLAGS_COMMON = FLAG_UTF8 | FLAG_DATA_DESCRIPTOR

// FLAGS_DEFLATE adds codec hints only for DEFLATE members.
const FLAGS_DEFLATE = FLAGS_COMMON | FLAG_DEFLATE_OPTIONS

// METHOD_STORE keeps uncompressed payload handling explicit.
const METHOD_STORE = 0

// METHOD_DEFLATE selects shared bounded RFC 1951 decoder.
const METHOD_DEFLATE = 1 << 3

// WRITER_LOCAL_HEADER_SIGNATURE marks one local member record.
const WRITER_LOCAL_HEADER_SIGNATURE binary.Word_32 = ZIP_SIGNATURE_PREFIX |
	binary.Word_32(ZIP_LOCAL_SIGNATURE_KIND)<<ZIP_SIGNATURE_KIND_SHIFT |
	binary.Word_32(ZIP_LOCAL_SIGNATURE_SUBKIND)<<ZIP_SIGNATURE_SUBKIND_SHIFT

// WRITER_CENTRAL_HEADER_SIGNATURE marks one central member record.
const WRITER_CENTRAL_HEADER_SIGNATURE binary.Word_32 = ZIP_SIGNATURE_PREFIX |
	binary.Word_32(ZIP_CENTRAL_SIGNATURE_KIND)<<ZIP_SIGNATURE_KIND_SHIFT |
	binary.Word_32(ZIP_CENTRAL_SIGNATURE_SUBKIND)<<ZIP_SIGNATURE_SUBKIND_SHIFT

// WRITER_DATA_DESCRIPTOR_SIGNATURE marks signed streaming size metadata.
const WRITER_DATA_DESCRIPTOR_SIGNATURE binary.Word_32 = ZIP_SIGNATURE_PREFIX |
	binary.Word_32(ZIP_DESCRIPTOR_SIGNATURE_KIND)<<ZIP_SIGNATURE_KIND_SHIFT |
	binary.Word_32(ZIP_DESCRIPTOR_SIGNATURE_SUBKIND)<<ZIP_SIGNATURE_SUBKIND_SHIFT

// WRITER_DIRECTORY_END_SIGNATURE marks classic central-directory end.
const WRITER_DIRECTORY_END_SIGNATURE binary.Word_32 = ZIP_SIGNATURE_PREFIX |
	binary.Word_32(ZIP_DIRECTORY_END_SIGNATURE_KIND)<<ZIP_SIGNATURE_KIND_SHIFT |
	binary.Word_32(ZIP_DIRECTORY_END_SIGNATURE_SUBKIND)<<ZIP_SIGNATURE_SUBKIND_SHIFT

// HEADER_NAME_SIZE_MAXIMUM leaves central prefix and directory end in bounded input.
const HEADER_NAME_SIZE_MAXIMUM = MEMBER_NAME_SIZE_MAXIMUM

// HEADER_COMMENT_SIZE_MAXIMUM leaves central prefix and directory end in bounded input.
const HEADER_COMMENT_SIZE_MAXIMUM = CENTRAL_TAIL_SIZE_MAXIMUM

// HEADER_EXTRA_SIZE_MAXIMUM leaves central prefix and directory end in bounded input.
const HEADER_EXTRA_SIZE_MAXIMUM = CENTRAL_TAIL_SIZE_MAXIMUM

// HEADER_FIELD_SIZE_UNVALIDATED_MAXIMUM admits the first rejected metadata byte.
const HEADER_FIELD_SIZE_UNVALIDATED_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM + 1

// HEADER_SIZE_MAXIMUM preserves the complete classic 32-bit wire domain.
const HEADER_SIZE_MAXIMUM uint64 = uint64(bits.WORD_32_MAXIMUM)

// HEADER_SIZE_UNVALIDATED_MAXIMUM admits the first rejected ZIP64 byte count.
const HEADER_SIZE_UNVALIDATED_MAXIMUM = HEADER_SIZE_MAXIMUM + binary.UINT_8_SIZE

// WRITER_RECORD_NAME_SIZE_MAXIMUM preserves the validated header name domain.
const WRITER_RECORD_NAME_SIZE_MAXIMUM = HEADER_NAME_SIZE_MAXIMUM

// WRITER_RECORD_TAIL_SIZE_MAXIMUM preserves the validated header tail domain.
const WRITER_RECORD_TAIL_SIZE_MAXIMUM = HEADER_EXTRA_SIZE_MAXIMUM

// WRITER_RECORD_EXTRA_SIZE_MAXIMUM includes optional timestamp bytes.
const WRITER_RECORD_EXTRA_SIZE_MAXIMUM = WRITER_RECORD_TAIL_SIZE_MAXIMUM +
	EXTENDED_TIMESTAMP_SIZE

// WRITER_LOCAL_STORAGE_SIZE_MINIMUM keeps a prefix and one name byte.
const WRITER_LOCAL_STORAGE_SIZE_MINIMUM = LOCAL_HEADER_SIZE + SELECTED_NAME_SIZE_MINIMUM

// WRITER_CENTRAL_STORAGE_SIZE_MINIMUM keeps a prefix and one name byte.
const WRITER_CENTRAL_STORAGE_SIZE_MINIMUM = CENTRAL_HEADER_SIZE +
	SELECTED_NAME_SIZE_MINIMUM

// WRITER_WORD_16_DESTINATION_SIZE_MINIMUM is one complete encoded word.
const WRITER_WORD_16_DESTINATION_SIZE_MINIMUM = binary.UINT_16_SIZE

// WRITER_WORD_16_DESTINATION_SIZE_MAXIMUM is the largest central-header tail used.
const WRITER_WORD_16_DESTINATION_SIZE_MAXIMUM = CENTRAL_HEADER_SIZE -
	WRITER_WORD_32_DESTINATION_SIZE_MINIMUM

// WRITER_WORD_32_DESTINATION_SIZE_MINIMUM is one complete encoded word.
const WRITER_WORD_32_DESTINATION_SIZE_MINIMUM = binary.UINT_32_SIZE

// WRITER_WORD_32_DESTINATION_SIZE_MAXIMUM is one complete central header.
const WRITER_WORD_32_DESTINATION_SIZE_MAXIMUM = CENTRAL_HEADER_SIZE

// WRITER_LOCAL_START_MAXIMUM leaves one minimum local record.
const WRITER_LOCAL_START_MAXIMUM = ARCHIVE_SIZE_MAXIMUM -
	WRITER_LOCAL_STORAGE_SIZE_MINIMUM

// ARCHIVE_FILE_MODE_MAXIMUM combines highest kind and reconstructable flags.
const ARCHIVE_FILE_MODE_MAXIMUM uint32 = uint32(
	nbio.FILE_MODE_DIRECTORY | nbio.FILE_MODE_SET_USER_IDENTIFIER |
		nbio.FILE_MODE_SET_GROUP_IDENTIFIER | nbio.FILE_MODE_STICKY |
		nbio.FILE_MODE_PERMISSIONS,
)

// DOS_FILE_ATTRIBUTES_MINIMUM begins the hostile attribute byte.
const DOS_FILE_ATTRIBUTES_MINIMUM uint8 = bits.WORD_8_MINIMUM

// DOS_FILE_ATTRIBUTES_MAXIMUM closes the hostile attribute byte.
const DOS_FILE_ATTRIBUTES_MAXIMUM uint8 = bits.WORD_8_MAXIMUM

// DOS_FILE_ATTRIBUTE_READ_ONLY marks legacy write protection.
const DOS_FILE_ATTRIBUTE_READ_ONLY = 1 << 0

// DOS_FILE_ATTRIBUTE_DIRECTORY marks a legacy directory.
const DOS_FILE_ATTRIBUTE_DIRECTORY = 1 << 4

// DOS_ARCHIVE_FILE_MODE_MINIMUM begins legacy regular permissions.
const DOS_ARCHIVE_FILE_MODE_MINIMUM uint32 = uint32(nbio.FILE_MODE_READ_PERMISSIONS)

// DOS_ARCHIVE_FILE_MODE_MAXIMUM includes directory and every permission bit.
const DOS_ARCHIVE_FILE_MODE_MAXIMUM uint32 = uint32(
	nbio.FILE_MODE_DIRECTORY | nbio.FILE_MODE_PERMISSIONS,
)

// TIMESTAMP_SECONDS_MINIMUM begins the extended timestamp wire domain.
const TIMESTAMP_SECONDS_MINIMUM int64 = int64(bits.WORD_32_MINIMUM)

// TIMESTAMP_SECONDS_MAXIMUM closes the 32-bit extended timestamp wire domain.
const TIMESTAMP_SECONDS_MAXIMUM int64 = int64(bits.WORD_32_MAXIMUM)

// DOS_CIVIL_YEAR_MINIMUM is the first legacy year.
const DOS_CIVIL_YEAR_MINIMUM = 1980

// DOS_DATE_DAY_BIT_COUNT is the packed legacy day field width.
const DOS_DATE_DAY_BIT_COUNT = 5

// DOS_DATE_MONTH_BIT_COUNT is the packed legacy month field width.
const DOS_DATE_MONTH_BIT_COUNT = 4

// DOS_DATE_YEAR_BIT_COUNT consumes the remainder of the packed date word.
const DOS_DATE_YEAR_BIT_COUNT = bits.BIT_COUNT_16_MAXIMUM - DOS_DATE_DAY_BIT_COUNT -
	DOS_DATE_MONTH_BIT_COUNT

// DOS_CIVIL_YEAR_COUNT follows the packed legacy year field.
const DOS_CIVIL_YEAR_COUNT = 1 << DOS_DATE_YEAR_BIT_COUNT

// DOS_CIVIL_YEAR_MAXIMUM is the final legacy year.
const DOS_CIVIL_YEAR_MAXIMUM = DOS_CIVIL_YEAR_MINIMUM + DOS_CIVIL_YEAR_COUNT - 1

// DOS_EPOCH_YEAR_COUNT spans Unix epoch through the first legacy year.
const DOS_EPOCH_YEAR_COUNT = DOS_CIVIL_YEAR_MINIMUM - time.CIVIL_UNIX_EPOCH_YEAR

// DOS_CIVIL_YEAR_PREVIOUS closes leap counting before the legacy epoch.
const DOS_CIVIL_YEAR_PREVIOUS = DOS_CIVIL_YEAR_MINIMUM - 1

// DOS_EPOCH_LEAP_YEAR_COUNT derives leap days between both epochs.
const DOS_EPOCH_LEAP_YEAR_COUNT = DOS_CIVIL_YEAR_PREVIOUS/time.CIVIL_LEAP_YEAR_INTERVAL -
	time.CIVIL_UNIX_EPOCH_PREVIOUS_YEAR/time.CIVIL_LEAP_YEAR_INTERVAL -
	(DOS_CIVIL_YEAR_PREVIOUS/time.CIVIL_CENTURY_YEAR_INTERVAL -
		time.CIVIL_UNIX_EPOCH_PREVIOUS_YEAR/time.CIVIL_CENTURY_YEAR_INTERVAL) +
	(DOS_CIVIL_YEAR_PREVIOUS/time.CIVIL_ERA_YEAR_INTERVAL -
		time.CIVIL_UNIX_EPOCH_PREVIOUS_YEAR/time.CIVIL_ERA_YEAR_INTERVAL)

// DOS_CALENDAR_DAY_MINIMUM derives 1980-01-01 relative to Unix epoch.
const DOS_CALENDAR_DAY_MINIMUM = DOS_EPOCH_YEAR_COUNT*time.CIVIL_COMMON_YEAR_DAY_COUNT +
	DOS_EPOCH_LEAP_YEAR_COUNT

// DOS_TIMESTAMP_SECONDS_MINIMUM is 1980-01-01 UTC.
const DOS_TIMESTAMP_SECONDS_MINIMUM int64 = DOS_CALENDAR_DAY_MINIMUM *
	time.SECOND_COUNT_PER_DAY

// DOS_TIMESTAMP_SECONDS_MAXIMUM is final even second inside extended range.
const DOS_TIMESTAMP_SECONDS_MAXIMUM int64 = TIMESTAMP_SECONDS_MAXIMUM - 1

// DOS_TIMESTAMP_SECOND_SPAN is the complete legacy civil interval.
const DOS_TIMESTAMP_SECOND_SPAN = DOS_TIMESTAMP_SECONDS_MAXIMUM -
	DOS_TIMESTAMP_SECONDS_MINIMUM

// DOS_TIMESTAMP_HALF_SECOND_OFFSET_MAXIMUM includes absent zero and final instant.
const DOS_TIMESTAMP_HALF_SECOND_OFFSET_MAXIMUM = DOS_TIMESTAMP_SECOND_SPAN/
	DOS_TIMESTAMP_SECOND_PRECISION + 1

// DOS_TIMESTAMP_HALF_SECOND_OFFSET_MINIMUM is absent metadata.
const DOS_TIMESTAMP_HALF_SECOND_OFFSET_MINIMUM = TIMESTAMP_SECONDS_MINIMUM

// DOS_ENCODING_HALF_SECOND_OFFSET_MAXIMUM includes the final local civil instant.
const DOS_ENCODING_HALF_SECOND_OFFSET_MAXIMUM = DOS_ENCODING_SECOND_SPAN/
	DOS_TIMESTAMP_SECOND_PRECISION + 1

// DOS_ENCODING_DATE_MINIMUM reserves zero for absent or rejected metadata.
const DOS_ENCODING_DATE_MINIMUM uint16 = bits.WORD_16_MINIMUM

// DOS_ENCODING_DATE_HOLE_FIRST begins invalid nonzero packed dates.
const DOS_ENCODING_DATE_HOLE_FIRST = DOS_ENCODING_DATE_MINIMUM + 1

// DOS_ENCODING_DATE_HOLE_SECOND follows first sampled invalid packed date.
const DOS_ENCODING_DATE_HOLE_SECOND = DOS_ENCODING_DATE_HOLE_FIRST + 1

// DOS_ENCODING_DATE_HOLE_THIRD follows second sampled invalid packed date.
const DOS_ENCODING_DATE_HOLE_THIRD = DOS_ENCODING_DATE_HOLE_SECOND + 1

// DOS_DATE_MONTH_SHIFT follows the packed day field.
const DOS_DATE_MONTH_SHIFT = DOS_DATE_DAY_BIT_COUNT

// DOS_DATE_YEAR_SHIFT follows packed day and month fields.
const DOS_DATE_YEAR_SHIFT = DOS_DATE_DAY_BIT_COUNT + DOS_DATE_MONTH_BIT_COUNT

// DOS_DATE_DAY_MASK selects packed legacy day.
const DOS_DATE_DAY_MASK = (1 << DOS_DATE_DAY_BIT_COUNT) - 1

// DOS_DATE_MONTH_MASK selects packed legacy month.
const DOS_DATE_MONTH_MASK = (1 << DOS_DATE_MONTH_BIT_COUNT) - 1

// DOS_ENCODING_DATE_PRESENT_MINIMUM is 1980-01-01 in packed form.
const DOS_ENCODING_DATE_PRESENT_MINIMUM uint16 = uint16(
	time.CIVIL_MONTH_MINIMUM<<DOS_DATE_MONTH_SHIFT | time.CIVIL_DAY_MINIMUM,
)

// DOS_ENCODING_DATE_MAXIMUM is the final date reachable by extended timestamps.
const DOS_ENCODING_DATE_MAXIMUM uint16 = uint16(
	(ZIP_TIMESTAMP_CIVIL_YEAR_MAXIMUM-DOS_CIVIL_YEAR_MINIMUM)<<DOS_DATE_YEAR_SHIFT |
		ZIP_TIMESTAMP_CIVIL_YEAR_MAXIMUM_MONTH<<DOS_DATE_MONTH_SHIFT |
		ZIP_TIMESTAMP_CIVIL_YEAR_MAXIMUM_DAY,
)

// DOS_ENCODING_TIME_MINIMUM reserves zero for midnight and absent metadata.
const DOS_ENCODING_TIME_MINIMUM uint16 = bits.WORD_16_MINIMUM

// DOS_TIME_SECOND_BIT_COUNT is the packed half-second field width.
const DOS_TIME_SECOND_BIT_COUNT = 5

// DOS_TIME_MINUTE_BIT_COUNT is the packed minute field width.
const DOS_TIME_MINUTE_BIT_COUNT = 6

// DOS_TIME_MINUTE_SHIFT follows packed half-seconds.
const DOS_TIME_MINUTE_SHIFT = DOS_TIME_SECOND_BIT_COUNT

// DOS_TIME_HOUR_SHIFT follows packed half-seconds and minutes.
const DOS_TIME_HOUR_SHIFT = DOS_TIME_SECOND_BIT_COUNT + DOS_TIME_MINUTE_BIT_COUNT

// DOS_TIME_SECOND_MASK selects packed legacy half-seconds.
const DOS_TIME_SECOND_MASK = (1 << DOS_TIME_SECOND_BIT_COUNT) - 1

// DOS_TIME_MINUTE_MASK selects packed legacy minutes.
const DOS_TIME_MINUTE_MASK = (1 << DOS_TIME_MINUTE_BIT_COUNT) - 1

// DOS_TIMESTAMP_SECOND_PRECISION is the legacy two-second wire precision.
const DOS_TIMESTAMP_SECOND_PRECISION = 2

// DOS_TIME_HOUR_MAXIMUM is the last hour inside one day.
const DOS_TIME_HOUR_MAXIMUM = time.SECOND_COUNT_PER_DAY/time.SECOND_COUNT_PER_HOUR - 1

// DOS_TIME_MINUTE_MAXIMUM is the last minute inside one hour.
const DOS_TIME_MINUTE_MAXIMUM = time.SECOND_COUNT_PER_HOUR/time.SECOND_COUNT_PER_MINUTE - 1

// DOS_TIME_SECOND_MAXIMUM is the last second inside one minute.
const DOS_TIME_SECOND_MAXIMUM = time.SECOND_COUNT_PER_MINUTE - 1

// DOS_ENCODING_TIME_MAXIMUM is 23:59:58 at legacy two-second precision.
const DOS_ENCODING_TIME_MAXIMUM = uint16(DOS_TIME_HOUR_MAXIMUM<<DOS_TIME_HOUR_SHIFT |
	DOS_TIME_MINUTE_MAXIMUM<<DOS_TIME_MINUTE_SHIFT |
	DOS_TIME_SECOND_MAXIMUM/DOS_TIMESTAMP_SECOND_PRECISION)

// WIRE_TIMESTAMP_SECOND_OFFSET_MAXIMUM includes absent zero and uint32 maximum.
const WIRE_TIMESTAMP_SECOND_OFFSET_MAXIMUM = uint64(bits.WORD_32_MAXIMUM) +
	WIRE_TIMESTAMP_PRESENT_OFFSET

// WIRE_TIMESTAMP_SECOND_OFFSET_MINIMUM is absent metadata.
const WIRE_TIMESTAMP_SECOND_OFFSET_MINIMUM uint64 = uint64(bits.WORD_32_MINIMUM)

// ZIP_LOCAL_CALENDAR_DAY_MAXIMUM includes final extended second at eastern offset.
const ZIP_LOCAL_CALENDAR_DAY_MAXIMUM int64 = LOCAL_TIMESTAMP_SECONDS_MAXIMUM /
	time.SECOND_COUNT_PER_DAY

// DOS_CIVIL_LEAP_YEAR_COUNT derives leap days in the packed year domain.
const DOS_CIVIL_LEAP_YEAR_COUNT = DOS_CIVIL_YEAR_MAXIMUM/time.CIVIL_LEAP_YEAR_INTERVAL -
	DOS_CIVIL_YEAR_PREVIOUS/time.CIVIL_LEAP_YEAR_INTERVAL -
	(DOS_CIVIL_YEAR_MAXIMUM/time.CIVIL_CENTURY_YEAR_INTERVAL -
		DOS_CIVIL_YEAR_PREVIOUS/time.CIVIL_CENTURY_YEAR_INTERVAL) +
	(DOS_CIVIL_YEAR_MAXIMUM/time.CIVIL_ERA_YEAR_INTERVAL -
		DOS_CIVIL_YEAR_PREVIOUS/time.CIVIL_ERA_YEAR_INTERVAL)

// DOS_CIVIL_DAY_COUNT derives all days representable by the packed year field.
const DOS_CIVIL_DAY_COUNT = DOS_CIVIL_YEAR_COUNT*time.CIVIL_COMMON_YEAR_DAY_COUNT +
	DOS_CIVIL_LEAP_YEAR_COUNT

// DOS_CALENDAR_DAY_MAXIMUM is 2107-12-31 relative to Unix epoch.
const DOS_CALENDAR_DAY_MAXIMUM = DOS_CALENDAR_DAY_MINIMUM + DOS_CIVIL_DAY_COUNT - 1

// ZIP_TIMESTAMP_CIVIL_YEAR_MAXIMUM includes the final extended timestamp year.
const ZIP_TIMESTAMP_CIVIL_YEAR_MAXIMUM = time.CIVIL_UNIX_EPOCH_YEAR +
	TIMESTAMP_SECONDS_MAXIMUM/(time.CIVIL_COMMON_YEAR_DAY_COUNT*time.SECOND_COUNT_PER_DAY)

// ZIP_COMPLETE_YEAR_MAXIMUM closes leap counting before final partial year.
const ZIP_COMPLETE_YEAR_MAXIMUM = ZIP_TIMESTAMP_CIVIL_YEAR_MAXIMUM - 1

// ZIP_EPOCH_PREVIOUS_YEAR closes leap counting before Unix epoch.
const ZIP_EPOCH_PREVIOUS_YEAR = time.CIVIL_UNIX_EPOCH_PREVIOUS_YEAR

// ZIP_FINAL_QUADRENNIAL_COUNT counts four-year boundaries before final year.
const ZIP_FINAL_QUADRENNIAL_COUNT = ZIP_COMPLETE_YEAR_MAXIMUM / time.CIVIL_LEAP_YEAR_INTERVAL

// ZIP_EPOCH_QUADRENNIAL_COUNT counts four-year boundaries before Unix epoch.
const ZIP_EPOCH_QUADRENNIAL_COUNT = ZIP_EPOCH_PREVIOUS_YEAR / time.CIVIL_LEAP_YEAR_INTERVAL

// ZIP_TIMESTAMP_CIVIL_REGULAR_LEAP_COUNT removes pre-epoch boundaries.
const ZIP_TIMESTAMP_CIVIL_REGULAR_LEAP_COUNT = ZIP_FINAL_QUADRENNIAL_COUNT -
	ZIP_EPOCH_QUADRENNIAL_COUNT

// ZIP_FINAL_CENTURY_COUNT counts excluded boundaries before final year.
const ZIP_FINAL_CENTURY_COUNT = ZIP_COMPLETE_YEAR_MAXIMUM / time.CIVIL_CENTURY_YEAR_INTERVAL

// ZIP_EPOCH_CENTURY_COUNT counts excluded boundaries before Unix epoch.
const ZIP_EPOCH_CENTURY_COUNT = ZIP_EPOCH_PREVIOUS_YEAR / time.CIVIL_CENTURY_YEAR_INTERVAL

// ZIP_TIMESTAMP_CIVIL_CENTURY_COUNT removes pre-epoch boundaries.
const ZIP_TIMESTAMP_CIVIL_CENTURY_COUNT = ZIP_FINAL_CENTURY_COUNT -
	ZIP_EPOCH_CENTURY_COUNT

// ZIP_FINAL_ERA_COUNT counts restored boundaries before final year.
const ZIP_FINAL_ERA_COUNT = ZIP_COMPLETE_YEAR_MAXIMUM / time.CIVIL_ERA_YEAR_INTERVAL

// ZIP_EPOCH_ERA_COUNT counts restored boundaries before Unix epoch.
const ZIP_EPOCH_ERA_COUNT = ZIP_EPOCH_PREVIOUS_YEAR / time.CIVIL_ERA_YEAR_INTERVAL

// ZIP_TIMESTAMP_CIVIL_ERA_COUNT removes pre-epoch boundaries.
const ZIP_TIMESTAMP_CIVIL_ERA_COUNT = ZIP_FINAL_ERA_COUNT - ZIP_EPOCH_ERA_COUNT

// ZIP_TIMESTAMP_CIVIL_YEAR_LEAP_COUNT derives leap days before final year.
const ZIP_TIMESTAMP_CIVIL_YEAR_LEAP_COUNT = ZIP_TIMESTAMP_CIVIL_REGULAR_LEAP_COUNT -
	ZIP_TIMESTAMP_CIVIL_CENTURY_COUNT + ZIP_TIMESTAMP_CIVIL_ERA_COUNT

// ZIP_TIMESTAMP_CIVIL_YEAR_COUNT is complete years before final extended year.
const ZIP_TIMESTAMP_CIVIL_YEAR_COUNT = ZIP_TIMESTAMP_CIVIL_YEAR_MAXIMUM -
	time.CIVIL_UNIX_EPOCH_YEAR

// ZIP_TIMESTAMP_CIVIL_YEAR_START_DAY derives the final year's Unix-relative start.
const ZIP_TIMESTAMP_CIVIL_YEAR_START_DAY = ZIP_TIMESTAMP_CIVIL_YEAR_COUNT*
	time.CIVIL_COMMON_YEAR_DAY_COUNT + ZIP_TIMESTAMP_CIVIL_YEAR_LEAP_COUNT

// ZIP_TIMESTAMP_CIVIL_YEAR_DAY_OFFSET derives the final local day inside its year.
const ZIP_TIMESTAMP_CIVIL_YEAR_DAY_OFFSET = ZIP_LOCAL_CALENDAR_DAY_MAXIMUM -
	ZIP_TIMESTAMP_CIVIL_YEAR_START_DAY

// ZIP_TIMESTAMP_CIVIL_YEAR_MAXIMUM_MONTH derives February from the final local day offset.
const ZIP_TIMESTAMP_CIVIL_YEAR_MAXIMUM_MONTH = time.CIVIL_MONTH_MINIMUM +
	ZIP_TIMESTAMP_CIVIL_YEAR_DAY_OFFSET/time.CIVIL_DAY_MAXIMUM

// ZIP_TIMESTAMP_CIVIL_MONTH_DAY_OFFSET locates final day inside its month.
const ZIP_TIMESTAMP_CIVIL_MONTH_DAY_OFFSET = ZIP_TIMESTAMP_CIVIL_YEAR_DAY_OFFSET %
	time.CIVIL_DAY_MAXIMUM

// ZIP_TIMESTAMP_CIVIL_YEAR_MAXIMUM_DAY derives the final day inside its month.
const ZIP_TIMESTAMP_CIVIL_YEAR_MAXIMUM_DAY = ZIP_TIMESTAMP_CIVIL_MONTH_DAY_OFFSET +
	time.CIVIL_DAY_MINIMUM

// LOCAL_TIMESTAMP_SECONDS_MINIMUM includes Unix epoch at western offset.
const LOCAL_TIMESTAMP_SECONDS_MINIMUM = int64(time.ZONE_OFFSET_SECONDS_MINIMUM)

// LOCAL_TIMESTAMP_SECONDS_MAXIMUM includes final extended second at eastern offset.
const LOCAL_TIMESTAMP_SECONDS_MAXIMUM = TIMESTAMP_SECONDS_MAXIMUM +
	int64(time.ZONE_OFFSET_SECONDS_MAXIMUM)

// DOS_ENCODING_SECOND_SPAN reaches the final caller-selected local instant.
const DOS_ENCODING_SECOND_SPAN = LOCAL_TIMESTAMP_SECONDS_MAXIMUM -
	DOS_TIMESTAMP_SECONDS_MINIMUM

// EXTENDED_TIMESTAMP_IDENTIFIER is Info-ZIP modification timestamp tag.
const EXTENDED_TIMESTAMP_IDENTIFIER binary.Word_16 = binary.Word_16('U') |
	binary.Word_16('T')<<bits.BIT_COUNT_8_MAXIMUM

// TIMESTAMP_ZONE_QUARTER_HOUR_SECONDS derives one encoded offset unit.
const TIMESTAMP_ZONE_QUARTER_HOUR_SECONDS = time.SECOND_COUNT_PER_HOUR / 4

// TIMESTAMP_ZONE_QUARTER_HOUR_ROUNDING_SECONDS derives nearest-unit rounding.
const TIMESTAMP_ZONE_QUARTER_HOUR_ROUNDING_SECONDS = TIMESTAMP_ZONE_QUARTER_HOUR_SECONDS / 2

// TIMESTAMP_ZONE_QUARTER_HOUR_MINIMUM is the encoded sensible offset floor.
const TIMESTAMP_ZONE_QUARTER_HOUR_MINIMUM = int8(
	int64(time.ZONE_OFFSET_SECONDS_MINIMUM) / TIMESTAMP_ZONE_QUARTER_HOUR_SECONDS,
)

// TIMESTAMP_ZONE_QUARTER_HOUR_MAXIMUM is the encoded sensible offset ceiling.
const TIMESTAMP_ZONE_QUARTER_HOUR_MAXIMUM = int8(
	int64(time.ZONE_OFFSET_SECONDS_MAXIMUM) / TIMESTAMP_ZONE_QUARTER_HOUR_SECONDS,
)

// TIMESTAMP_ZONE_DIFFERENCE_MINIMUM spans first DOS instant to final extended instant.
const TIMESTAMP_ZONE_DIFFERENCE_MINIMUM = DOS_TIMESTAMP_SECONDS_MINIMUM -
	TIMESTAMP_SECONDS_MAXIMUM

// TIMESTAMP_ZONE_DIFFERENCE_MAXIMUM spans final DOS instant to Unix epoch.
const TIMESTAMP_ZONE_DIFFERENCE_MAXIMUM = DOS_TIMESTAMP_SECONDS_MAXIMUM

// TIMESTAMP_ZONE_DIFFERENCE_FLOOR applies nearest-quarter-hour rounding.
const TIMESTAMP_ZONE_DIFFERENCE_FLOOR = TIMESTAMP_ZONE_DIFFERENCE_MINIMUM -
	TIMESTAMP_ZONE_QUARTER_HOUR_ROUNDING_SECONDS

// TIMESTAMP_ZONE_DIFFERENCE_CEILING applies nearest-quarter-hour rounding.
const TIMESTAMP_ZONE_DIFFERENCE_CEILING = TIMESTAMP_ZONE_DIFFERENCE_MAXIMUM +
	TIMESTAMP_ZONE_QUARTER_HOUR_ROUNDING_SECONDS

// TIMESTAMP_ZONE_QUARTER_HOURS_UNBOUNDED_MINIMUM rounds the difference floor.
const TIMESTAMP_ZONE_QUARTER_HOURS_UNBOUNDED_MINIMUM = TIMESTAMP_ZONE_DIFFERENCE_FLOOR /
	TIMESTAMP_ZONE_QUARTER_HOUR_SECONDS

// TIMESTAMP_ZONE_QUARTER_HOURS_UNBOUNDED_MAXIMUM rounds the difference ceiling.
const TIMESTAMP_ZONE_QUARTER_HOURS_UNBOUNDED_MAXIMUM = TIMESTAMP_ZONE_DIFFERENCE_CEILING /
	TIMESTAMP_ZONE_QUARTER_HOUR_SECONDS

// DOS_FILE_MODE_READ_EXECUTE combines legacy read and traversal permissions.
const DOS_FILE_MODE_READ_EXECUTE = nbio.FILE_MODE_READ_PERMISSIONS |
	nbio.FILE_MODE_EXECUTE_PERMISSIONS

// DOS_FILE_MODE_READ_WRITE combines legacy read and mutation permissions.
const DOS_FILE_MODE_READ_WRITE = nbio.FILE_MODE_READ_PERMISSIONS |
	nbio.FILE_MODE_WRITE_PERMISSIONS

// CREATOR_UNIX identifies POSIX external attributes.
const CREATOR_UNIX = 3

// CREATOR_MAC_OS_X identifies POSIX-compatible macOS external attributes.
const CREATOR_MAC_OS_X = 19

// CREATOR_SYSTEM_MASK selects the low creator-system byte.
const CREATOR_SYSTEM_MASK Header_Creator_Version = Header_Creator_Version(bits.WORD_8_MAXIMUM)

// CREATOR_SYSTEM_SHIFT locates creator system above the version byte.
const CREATOR_SYSTEM_SHIFT = bits.BIT_COUNT_8_MAXIMUM

// EXTERNAL_ATTRIBUTES_UNIX_SHIFT locates Unix mode above DOS attributes.
const EXTERNAL_ATTRIBUTES_UNIX_SHIFT = bits.BIT_COUNT_16_MAXIMUM

// CENTRAL_HEADER_OPTIONAL_SIZE_EMPTY keeps invalid parser output representable.
const CENTRAL_HEADER_OPTIONAL_SIZE_EMPTY = bytes.SLICE_SIZE_MINIMUM

// SELECTED_NAME_SIZE_MINIMUM follows nonempty lookup suffix.
const SELECTED_NAME_SIZE_MINIMUM = MEMBER_NAME_SIZE_MINIMUM + binary.UINT_8_SIZE

// SELECTED_NAME_SIZE_MAXIMUM keeps central metadata untrusted until local validation.
const SELECTED_NAME_SIZE_MAXIMUM = MEMBER_NAME_SIZE_MAXIMUM

// Archive keeps validated source boundary distinct from hostile input.
type Archive []byte

// Archive_Invariants keeps every internal slice inside caller archive.
func Archive_Invariants(value Archive, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), ARCHIVE_SIZE_MINIMUM, ARCHIVE_SIZE_MAXIMUM).
		Ensure()
}

// Destination keeps validated output separate from hostile caller storage.
type Destination []byte

// Destination_Invariants keeps internal output inside shared byte boundary.
func Destination_Invariants(
	value Destination, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), DESTINATION_SIZE_MINIMUM, DESTINATION_SIZE_MAXIMUM,
		).
		Ensure()
}

// Entry_Suffix keeps validated lookup grammar after malicious-input checks.
type Entry_Suffix string

// Entry_Suffix_Invariants protects nonempty bounded lookup storage.
func Entry_Suffix_Invariants(
	value Entry_Suffix, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), ENTRY_SUFFIX_SIZE_MINIMUM, ENTRY_SUFFIX_SIZE_MAXIMUM,
		).
		Ensure()
}

// Member_Name borrows one validated central-directory name.
type Member_Name []byte

// Member_Name_Invariants protects borrowed source boundary.
func Member_Name_Invariants(
	value Member_Name, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), MEMBER_NAME_SIZE_MINIMUM, MEMBER_NAME_SIZE_MAXIMUM,
		).
		Ensure()
}

// Directory_Position is one bounded directory end coordinate.
type Directory_Position int

// Directory_Position_Invariants protects directory end search boundary.
func Directory_Position_Invariants(
	value Directory_Position, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), POSITION_MINIMUM, DIRECTORY_POSITION_MAXIMUM,
		).
		Ensure()
}

// Central_Size is bounded central-directory byte count.
type Central_Size int

// Central_Size_Invariants protects central start subtraction.
func Central_Size_Invariants(
	value Central_Size, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), POSITION_MINIMUM, CENTRAL_SIZE_MAXIMUM).
		Ensure()
}

// Central_Offset is bounded ZIP-relative central coordinate.
type Central_Offset int

// Central_Offset_Invariants protects base-offset subtraction.
func Central_Offset_Invariants(
	value Central_Offset, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), POSITION_MINIMUM, CENTRAL_OFFSET_MAXIMUM).
		Ensure()
}

// Central_Tail_Size bounds variable bytes after one central prefix.
type Central_Tail_Size int

// Central_Tail_Size_Invariants protects central variable-field slicing.
func Central_Tail_Size_Invariants(
	value Central_Tail_Size, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), POSITION_MINIMUM, CENTRAL_TAIL_SIZE_MAXIMUM,
		).
		Ensure()
}

// Central_End is bounded end of central directory entries.
type Central_End int

// Central_End_Invariants protects one complete central prefix before end.
func Central_End_Invariants(value Central_End, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), CENTRAL_END_MINIMUM, CENTRAL_END_MAXIMUM).
		Ensure()
}

// Central_Position is bounded start of one central entry.
type Central_Position int

// Central_Position_Invariants leaves one fixed prefix before Central_End.
func Central_Position_Invariants(
	value Central_Position, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), POSITION_MINIMUM, CENTRAL_POSITION_MAXIMUM,
		).
		Ensure()
}

// Central_Boundary is parser cursor after optional central entry.
type Central_Boundary int

// Central_Boundary_Invariants protects parser cursor conversion.
func Central_Boundary_Invariants(
	value Central_Boundary, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), POSITION_MINIMUM, CENTRAL_END_MAXIMUM).
		Ensure()
}

// Base_Offset is one bounded self-extracting prefix size.
type Base_Offset int

// Base_Offset_Invariants protects local-pointer translation.
func Base_Offset_Invariants(value Base_Offset, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), POSITION_MINIMUM, SELECTED_CENTRAL_START_MAXIMUM,
		).
		Ensure()
}

// Selected_Central_Start is central start after one lookup match.
type Selected_Central_Start int

// Selected_Central_Start_Invariants leaves one selected central entry.
func Selected_Central_Start_Invariants(
	value Selected_Central_Start, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), POSITION_MINIMUM, SELECTED_CENTRAL_START_MAXIMUM,
		).
		Ensure()
}

// Local_Position is optional local-header or data boundary.
type Local_Position int

// Local_Position_Invariants protects local parser cursor conversion.
func Local_Position_Invariants(
	value Local_Position, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), POSITION_MINIMUM, SELECTED_CENTRAL_START_MAXIMUM,
		).
		Ensure()
}

// Payload_Position is validated payload or descriptor boundary.
type Payload_Position int

// Payload_Position_Invariants protects selected member slicing.
func Payload_Position_Invariants(
	value Payload_Position, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), PAYLOAD_POSITION_MINIMUM, PAYLOAD_POSITION_MAXIMUM,
		).
		Ensure()
}

// Central_Archive keeps central parser minimum distinct from empty archive.
type Central_Archive []byte

// Central_Archive_Invariants protects central fixed-prefix reads.
func Central_Archive_Invariants(
	value Central_Archive, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), CENTRAL_ARCHIVE_SIZE_MINIMUM, ARCHIVE_SIZE_MAXIMUM,
		).
		Ensure()
}

// Selected_Archive keeps lookup-match minimum distinct from central scan.
type Selected_Archive []byte

// Selected_Archive_Invariants protects selected central metadata access.
func Selected_Archive_Invariants(
	value Selected_Archive, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), SELECTED_ARCHIVE_SIZE_MINIMUM, ARCHIVE_SIZE_MAXIMUM,
		).
		Ensure()
}

// Payload_Archive keeps valid local-header minimum distinct from selection.
type Payload_Archive []byte

// Payload_Archive_Invariants protects payload and descriptor access.
func Payload_Archive_Invariants(
	value Payload_Archive, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), PAYLOAD_ARCHIVE_SIZE_MINIMUM, ARCHIVE_SIZE_MAXIMUM,
		).
		Ensure()
}

// Descriptor_Archive keeps descriptor minimum distinct from payload archive.
type Descriptor_Archive []byte

// Descriptor_Archive_Invariants protects descriptor fixed-field reads.
func Descriptor_Archive_Invariants(
	value Descriptor_Archive, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), DESCRIPTOR_ARCHIVE_SIZE_MINIMUM, ARCHIVE_SIZE_MAXIMUM,
		).
		Ensure()
}

// Descriptor_Position is validated opening boundary of classic descriptor.
type Descriptor_Position int

// Descriptor_Position_Invariants leaves one unsigned descriptor before central entry.
func Descriptor_Position_Invariants(
	value Descriptor_Position, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), PAYLOAD_POSITION_MINIMUM, DESCRIPTOR_POSITION_MAXIMUM,
		).
		Ensure()
}

// Descriptor_Central_Start closes validated classic descriptor region.
type Descriptor_Central_Start int

// Descriptor_Central_Start_Invariants protects descriptor upper boundary.
func Descriptor_Central_Start_Invariants(
	value Descriptor_Central_Start, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), DESCRIPTOR_CENTRAL_START_MINIMUM,
			SELECTED_CENTRAL_START_MAXIMUM,
		).
		Ensure()
}

// Entry_Count bounds directory traversal work before its loop starts.
type Entry_Count int

// Entry_Count_Invariants protects bounded central traversal.
func Entry_Count_Invariants(
	value Entry_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), ENTRY_COUNT_MINIMUM, ENTRY_COUNT_MAXIMUM).
		Ensure()
}

// Output_Count is complete bytes written by one selected member.
type Output_Count int

// Output_Count_Invariants protects conversion to shared boundary.
func Output_Count_Invariants(
	value Output_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), OUTPUT_COUNT_MINIMUM, OUTPUT_COUNT_MAXIMUM).
		Ensure()
}

// Entry_Status excludes lookup absence after member selection.
type Entry_Status uint8

// Entry_Status_Invariants protects conversion to public Status.
func Entry_Status_Invariants(
	value Entry_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_INPUT_INVALID),
			uint8(STATUS_OUTPUT_TOO_SMALL), uint8(STATUS_METHOD_UNSUPPORTED),
		).
		Ensure()
}

// Central_Header_Optional keeps failed parser output allocation-free.
type Central_Header_Optional []byte

// Central_Header_Optional_Invariants separates empty failure from complete header.
func Central_Header_Optional_Invariants(
	value Central_Header_Optional, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Int(
			len(value), CENTRAL_HEADER_OPTIONAL_SIZE_EMPTY, CENTRAL_HEADER_SIZE,
		).
		Ensure()
}

// Central_Header borrows one complete classic central prefix.
type Central_Header []byte

// Central_Header_Invariants protects fixed-offset reads.
func Central_Header_Invariants(
	value Central_Header, namespace aver.Namespace,
) {
	aver.Always(
		len(value) == CENTRAL_HEADER_SIZE,
		"Selected central header has complete fixed prefix.",
	)
}

// Local_Header borrows one complete classic local prefix.
type Local_Header []byte

// Local_Header_Invariants protects fixed-offset reads.
func Local_Header_Invariants(value Local_Header, namespace aver.Namespace) {
	aver.Always(
		len(value) == LOCAL_HEADER_SIZE,
		"Local header has complete fixed prefix.",
	)
}

// Selected_Name keeps lookup match distinct from optional central name.
type Selected_Name []byte

// Selected_Name_Invariants protects nonempty borrowed lookup result.
func Selected_Name_Invariants(
	value Selected_Name, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), SELECTED_NAME_SIZE_MINIMUM, SELECTED_NAME_SIZE_MAXIMUM,
		).
		Ensure()
}

// Central_Entry_Optional keeps parser failure representable without pointers.
type Central_Entry_Optional struct {
	// Header is empty only when parser rejects entry.
	Header Central_Header_Optional
	// Name borrows central storage only after bounds checks.
	Name Member_Name
}

// Central_Entry_Optional_Invariants composes parser result storage.
func Central_Entry_Optional_Invariants(
	value Central_Entry_Optional, namespace aver.Namespace,
) {
	Central_Header_Optional_Invariants(value.Header, namespace)
	Member_Name_Invariants(value.Name, namespace)
}

// Central_Entry_Parsed retains one complete header with its possibly empty name.
type Central_Entry_Parsed struct {
	// Header passed central fixed-prefix validation.
	Header Central_Header
	// Name preserves the complete hostile wire domain.
	Name Member_Name
}

// Central_Entry_Parsed_Invariants composes one successful parser result.
func Central_Entry_Parsed_Invariants(
	value Central_Entry_Parsed, namespace aver.Namespace,
) {
	Central_Header_Invariants(value.Header, namespace)
	Member_Name_Invariants(value.Name, namespace)
}

// Central_Entry retains selected borrowed central metadata.
type Central_Entry struct {
	// Header keeps fixed metadata in caller source.
	Header Central_Header
	// Name keeps local-header comparison allocation-free.
	Name Selected_Name
}

// Central_Entry_Invariants composes selected borrowed metadata.
func Central_Entry_Invariants(
	value Central_Entry, namespace aver.Namespace,
) {
	Central_Header_Invariants(value.Header, namespace)
	Selected_Name_Invariants(value.Name, namespace)
}

// Archive_File_Mode is one mode reconstructable from ZIP external attributes.
type Archive_File_Mode uint32

// Archive_File_Mode_Invariants states each independently reconstructable mode bit.
func Archive_File_Mode_Invariants(
	value Archive_File_Mode, namespace aver.Namespace,
) {
	mode := nbio.File_Mode(value)
	aver.Tree(value, namespace).
		Range_Uint32(
			uint32(value), nbio.FILE_MODE_MINIMUM, ARCHIVE_FILE_MODE_MAXIMUM,
		).
		Range_Uint32(
			uint32(mode&nbio.FILE_MODE_PERMISSIONS), nbio.FILE_MODE_PERMISSION_MINIMUM,
			nbio.FILE_MODE_PERMISSION_MAXIMUM,
		).
		Sometimes(nbio.File_Mode_Is_Directory(mode), "ZIP mode is a directory.").
		Sometimes(mode&nbio.FILE_MODE_SYMBOLIC_LINK != 0, "ZIP mode is a symbolic link.").
		Sometimes(mode&nbio.FILE_MODE_NAMED_PIPE != 0, "ZIP mode is a named pipe.").
		Sometimes(mode&nbio.FILE_MODE_SOCKET != 0, "ZIP mode is a socket.").
		Sometimes(mode&nbio.FILE_MODE_DEVICE != 0, "ZIP mode is a device.").
		Sometimes(
			mode&nbio.FILE_MODE_CHARACTER_DEVICE != 0,
			"ZIP mode is a character device.",
		).
		Sometimes(
			mode&nbio.FILE_MODE_SET_USER_IDENTIFIER != 0,
			"ZIP mode has set-user-ID.",
		).
		Sometimes(
			mode&nbio.FILE_MODE_SET_GROUP_IDENTIFIER != 0,
			"ZIP mode has set-group-ID.",
		).
		Sometimes(mode&nbio.FILE_MODE_STICKY != 0, "ZIP mode has sticky bit.").
		Ensure()
}

// DOS_File_Attributes accepts every low legacy attribute byte.
type DOS_File_Attributes uint8

// DOS_File_Attributes_Invariants keeps hostile legacy attributes complete.
func DOS_File_Attributes_Invariants(
	value DOS_File_Attributes, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value), DOS_FILE_ATTRIBUTES_MINIMUM,
			DOS_FILE_ATTRIBUTES_MAXIMUM,
		).
		Ensure()
}

// DOS_Archive_File_Mode is one legacy regular or directory mode.
type DOS_Archive_File_Mode uint32

// DOS_Archive_File_Mode_Invariants keeps four permission outcomes and kind exact.
func DOS_Archive_File_Mode_Invariants(
	value DOS_Archive_File_Mode, namespace aver.Namespace,
) {
	mode := nbio.File_Mode(value)
	aver.Tree(value, namespace).
		Range_Uint32(
			uint32(value), DOS_ARCHIVE_FILE_MODE_MINIMUM,
			DOS_ARCHIVE_FILE_MODE_MAXIMUM,
		).
		Enum_4_Uint16(
			uint16(mode&nbio.FILE_MODE_PERMISSIONS),
			uint16(nbio.FILE_MODE_READ_PERMISSIONS), uint16(DOS_FILE_MODE_READ_EXECUTE),
			uint16(DOS_FILE_MODE_READ_WRITE), uint16(nbio.FILE_MODE_PERMISSIONS),
		).
		Sometimes(nbio.File_Mode_Is_Directory(mode), "DOS mode is a directory.").
		Ensure()
}

// Local_Timestamp_Seconds is caller seconds after civil offset.
type Local_Timestamp_Seconds int64

// Local_Timestamp_Seconds_Invariants bounds civil conversion input.
func Local_Timestamp_Seconds_Invariants(
	value Local_Timestamp_Seconds, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int64(
			int64(value), LOCAL_TIMESTAMP_SECONDS_MINIMUM,
			LOCAL_TIMESTAMP_SECONDS_MAXIMUM,
		).
		Ensure()
}

// DOS_Calendar_Day_Count is one decoded legacy civil day.
type DOS_Calendar_Day_Count int64

// DOS_Calendar_Day_Count_Invariants bounds legacy calendar conversion.
func DOS_Calendar_Day_Count_Invariants(
	value DOS_Calendar_Day_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int64(
			int64(value), DOS_CALENDAR_DAY_MINIMUM, DOS_CALENDAR_DAY_MAXIMUM,
		).
		Ensure()
}

// DOS_Civil_Year is one decoded legacy Gregorian year.
type DOS_Civil_Year int

// DOS_Civil_Year_Invariants bounds every legacy year field.
func DOS_Civil_Year_Invariants(
	value DOS_Civil_Year, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), DOS_CIVIL_YEAR_MINIMUM, DOS_CIVIL_YEAR_MAXIMUM).
		Ensure()
}

// Timestamp_Set distinguishes Unix epoch from absent metadata.
type Timestamp_Set bool

// Timestamp_Set_Invariants covers absent and present instants.
func Timestamp_Set_Invariants(value Timestamp_Set, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "ZIP timestamp is present.").
		Ensure()
}

// Timestamp_Seconds is one Info-ZIP extended timestamp value.
type Timestamp_Seconds int64

// Timestamp_Seconds_Invariants bounds the 32-bit Unix wire value.
func Timestamp_Seconds_Invariants(
	value Timestamp_Seconds, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int64(
			int64(value), TIMESTAMP_SECONDS_MINIMUM, TIMESTAMP_SECONDS_MAXIMUM,
		).
		Ensure()
}

// Timestamp preserves instant and caller-selected civil offset without ambient clock.
type Timestamp struct {
	// Seconds holds Unix epoch offset.
	Seconds Timestamp_Seconds
	// Nanoseconds holds normalized fractional second.
	Nanoseconds time.Nanosecond_Count
	// Zone_Offset_Seconds preserves civil representation used by legacy DOS fields.
	Zone_Offset_Seconds time.Zone_Offset_Seconds
	// Set distinguishes Unix epoch from absent metadata.
	Set Timestamp_Set
}

// Timestamp_Invariants bounds fraction and civil offset.
func Timestamp_Invariants(value Timestamp, namespace aver.Namespace) {
	Timestamp_Seconds_Invariants(value.Seconds, namespace)
	time.Nanosecond_Count_Invariants(value.Nanoseconds, namespace)
	time.Zone_Offset_Seconds_Invariants(value.Zone_Offset_Seconds, namespace)
	Timestamp_Set_Invariants(value.Set, namespace)
}

// Timestamp_Second_Precision is an instant representable by ZIP metadata.
type Timestamp_Second_Precision struct {
	// Seconds retains the complete extended timestamp wire value.
	Seconds Timestamp_Seconds
	// Zone_Offset_Seconds preserves a compatible DOS civil representation.
	Zone_Offset_Seconds time.Zone_Offset_Seconds
	// Set keeps an absent timestamp distinct from Unix epoch.
	Set Timestamp_Set
}

// Timestamp_Second_Precision_Invariants excludes fractional state ZIP cannot encode.
func Timestamp_Second_Precision_Invariants(
	value Timestamp_Second_Precision, namespace aver.Namespace,
) {
	Timestamp_Seconds_Invariants(value.Seconds, namespace)
	time.Zone_Offset_Seconds_Invariants(value.Zone_Offset_Seconds, namespace)
	Timestamp_Set_Invariants(value.Set, namespace)
}

// DOS_Timestamp_Half_Second_Offset encodes absent zero and DOS two-second steps.
type DOS_Timestamp_Half_Second_Offset int64

// DOS_Timestamp_Half_Second_Offset_Invariants keeps optional legacy time exact.
func DOS_Timestamp_Half_Second_Offset_Invariants(
	value DOS_Timestamp_Half_Second_Offset, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int64(
			int64(value), DOS_TIMESTAMP_HALF_SECOND_OFFSET_MINIMUM,
			DOS_TIMESTAMP_HALF_SECOND_OFFSET_MAXIMUM,
		).
		Ensure()
}

// DOS_Timestamp is an optional bounded legacy timestamp.
type DOS_Timestamp struct {
	// Half_Second_Offset reserves zero for absent legacy fields.
	Half_Second_Offset DOS_Timestamp_Half_Second_Offset
}

// DOS_Timestamp_Invariants separates absent metadata from its civil range.
func DOS_Timestamp_Invariants(value DOS_Timestamp, namespace aver.Namespace) {
	DOS_Timestamp_Half_Second_Offset_Invariants(
		value.Half_Second_Offset, namespace,
	)
}

// DOS_Encoding_Half_Second_Offset witnesses every emitted legacy instant.
type DOS_Encoding_Half_Second_Offset int64

// DOS_Encoding_Half_Second_Offset_Invariants bounds caller timestamp encoding.
func DOS_Encoding_Half_Second_Offset_Invariants(
	value DOS_Encoding_Half_Second_Offset, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int64(
			int64(value), DOS_TIMESTAMP_HALF_SECOND_OFFSET_MINIMUM,
			DOS_ENCODING_HALF_SECOND_OFFSET_MAXIMUM,
		).
		Ensure()
}

// DOS_Encoding_Date is one packed date emitted from an extended timestamp.
type DOS_Encoding_Date uint16

// DOS_Encoding_Date_Invariants bounds emitted packed calendar words.
func DOS_Encoding_Date_Invariants(
	value DOS_Encoding_Date, namespace aver.Namespace,
) {
	aver.Always(
		uint16(value)-1 >= DOS_ENCODING_DATE_PRESENT_MINIMUM-1,
		"Encoded DOS date is absent or a representable calendar date.",
	)
	aver.Tree(value, namespace).
		Range_Holed_Uint16(
			uint16(value), DOS_ENCODING_DATE_MINIMUM, DOS_ENCODING_DATE_MAXIMUM,
			DOS_ENCODING_DATE_HOLE_FIRST, DOS_ENCODING_DATE_HOLE_SECOND,
			DOS_ENCODING_DATE_HOLE_THIRD,
		).
		Ensure()
}

// DOS_Encoding_Time is one packed clock emitted at two-second precision.
type DOS_Encoding_Time uint16

// DOS_Encoding_Time_Invariants bounds emitted packed clock words.
func DOS_Encoding_Time_Invariants(
	value DOS_Encoding_Time, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), DOS_ENCODING_TIME_MINIMUM, DOS_ENCODING_TIME_MAXIMUM,
		).
		Ensure()
}

// DOS_Encoding carries raw fields beside their exact semantic coordinate.
type DOS_Encoding struct {
	// Date is the packed legacy civil date.
	Date DOS_Encoding_Date
	// Clock is the packed legacy civil time.
	Clock DOS_Encoding_Time
	// Half_Second_Offset makes the sparse packed-word domain continuous.
	Half_Second_Offset DOS_Encoding_Half_Second_Offset
}

// DOS_Encoding_Invariants proves the exact emitted legacy-time domain.
func DOS_Encoding_Invariants(value DOS_Encoding, namespace aver.Namespace) {
	DOS_Encoding_Date_Invariants(value.Date, namespace)
	DOS_Encoding_Time_Invariants(value.Clock, namespace)
	DOS_Encoding_Half_Second_Offset_Invariants(
		value.Half_Second_Offset, namespace,
	)
}

// Wire_Timestamp_Second_Offset encodes absent zero and Unix seconds plus one.
type Wire_Timestamp_Second_Offset uint64

// Wire_Timestamp_Second_Offset_Invariants keeps optional wire seconds exact.
func Wire_Timestamp_Second_Offset_Invariants(
	value Wire_Timestamp_Second_Offset, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint64(
			uint64(value), WIRE_TIMESTAMP_SECOND_OFFSET_MINIMUM,
			WIRE_TIMESTAMP_SECOND_OFFSET_MAXIMUM,
		).
		Ensure()
}

// Timestamp_Zone_Difference is legacy seconds minus extended seconds.
type Timestamp_Zone_Difference int64

// Timestamp_Zone_Difference_Invariants bounds inference before sensible-offset filtering.
func Timestamp_Zone_Difference_Invariants(
	value Timestamp_Zone_Difference, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int64(
			int64(value), TIMESTAMP_ZONE_DIFFERENCE_MINIMUM,
			TIMESTAMP_ZONE_DIFFERENCE_MAXIMUM,
		).
		Ensure()
}

// Timestamp_Zone_Quarter_Hours_Unbounded is one rounded inference before filtering.
type Timestamp_Zone_Quarter_Hours_Unbounded int64

// Timestamp_Zone_Quarter_Hours_Unbounded_Invariants bounds every rounded inference.
func Timestamp_Zone_Quarter_Hours_Unbounded_Invariants(
	value Timestamp_Zone_Quarter_Hours_Unbounded, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int64(
			int64(value), TIMESTAMP_ZONE_QUARTER_HOURS_UNBOUNDED_MINIMUM,
			TIMESTAMP_ZONE_QUARTER_HOURS_UNBOUNDED_MAXIMUM,
		).
		Ensure()
}

// Timestamp_Zone_Quarter_Hours is one inferred ZIP civil offset.
type Timestamp_Zone_Quarter_Hours int8

// Timestamp_Zone_Quarter_Hours_Invariants keeps inferred offset steps exact.
func Timestamp_Zone_Quarter_Hours_Invariants(
	value Timestamp_Zone_Quarter_Hours, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int8(
			int8(value), TIMESTAMP_ZONE_QUARTER_HOUR_MINIMUM,
			TIMESTAMP_ZONE_QUARTER_HOUR_MAXIMUM,
		).
		Ensure()
}

// Wire_Timestamp is an optional extended timestamp with inferred civil offset.
type Wire_Timestamp struct {
	// Second_Offset reserves zero for absent metadata.
	Second_Offset Wire_Timestamp_Second_Offset
	// Zone_Quarter_Hours is derived from compatible DOS metadata.
	Zone_Quarter_Hours Timestamp_Zone_Quarter_Hours
}

// Wire_Timestamp_Invariants keeps absent and present wire metadata separate.
func Wire_Timestamp_Invariants(value Wire_Timestamp, namespace aver.Namespace) {
	Wire_Timestamp_Second_Offset_Invariants(value.Second_Offset, namespace)
	Timestamp_Zone_Quarter_Hours_Invariants(value.Zone_Quarter_Hours, namespace)
}

// Header_Name is one validated member path.
type Header_Name []byte

// Header_Name_Invariants bounds one validated member path.
func Header_Name_Invariants(value Header_Name, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), MEMBER_NAME_SIZE_MINIMUM, HEADER_NAME_SIZE_MAXIMUM).
		Ensure()
}

// Header_Comment is one validated central member comment.
type Header_Comment []byte

// Header_Comment_Invariants bounds one validated member comment.
func Header_Comment_Invariants(value Header_Comment, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), MEMBER_NAME_SIZE_MINIMUM, HEADER_COMMENT_SIZE_MAXIMUM).
		Ensure()
}

// Header_Extra is one validated extension sequence.
type Header_Extra []byte

// Header_Extra_Invariants bounds one validated extension sequence.
func Header_Extra_Invariants(value Header_Extra, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), MEMBER_NAME_SIZE_MINIMUM, HEADER_EXTRA_SIZE_MAXIMUM).
		Ensure()
}

// Header_Method preserves complete wire codec identifiers.
type Header_Method uint16

// Header_Method_Invariants preserves complete wire codec identifiers.
func Header_Method_Invariants(value Header_Method, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value), bits.WORD_16_MINIMUM, bits.WORD_16_MAXIMUM).
		Ensure()
}

// Header_Flags preserves complete general-purpose wire flags.
type Header_Flags uint16

// Header_Flags_Invariants preserves complete general-purpose flags.
func Header_Flags_Invariants(value Header_Flags, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value), bits.WORD_16_MINIMUM, bits.WORD_16_MAXIMUM).
		Ensure()
}

// Header_Checksum is one complete CRC-32 value.
type Header_Checksum uint32

// Header_Checksum_Invariants preserves the complete CRC-32 domain.
func Header_Checksum_Invariants(value Header_Checksum, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), bits.WORD_32_MINIMUM, bits.WORD_32_MAXIMUM).
		Ensure()
}

// Header_Compressed_Size preserves the complete classic compressed-size word.
type Header_Compressed_Size uint64

// Header_Compressed_Size_Invariants preserves the complete classic size domain.
func Header_Compressed_Size_Invariants(
	value Header_Compressed_Size, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, HEADER_SIZE_MAXIMUM).
		Ensure()
}

// Header_Uncompressed_Size preserves the complete classic logical-size word.
type Header_Uncompressed_Size uint64

// Header_Uncompressed_Size_Invariants preserves the complete classic size domain.
func Header_Uncompressed_Size_Invariants(
	value Header_Uncompressed_Size, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, HEADER_SIZE_MAXIMUM).
		Ensure()
}

// Header_External_Attributes is one complete central attribute word.
type Header_External_Attributes uint32

// Header_External_Attributes_Invariants preserves complete attribute words.
func Header_External_Attributes_Invariants(
	value Header_External_Attributes, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), bits.WORD_32_MINIMUM, bits.WORD_32_MAXIMUM).
		Ensure()
}

// Header_Creator_Version is one complete central creator word.
type Header_Creator_Version uint16

// Header_Creator_Version_Invariants preserves complete creator words.
func Header_Creator_Version_Invariants(
	value Header_Creator_Version, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value), bits.WORD_16_MINIMUM, bits.WORD_16_MAXIMUM).
		Ensure()
}

// Header_Extractor_Version is one complete minimum extractor word.
type Header_Extractor_Version uint16

// Header_Extractor_Version_Invariants preserves complete extractor words.
func Header_Extractor_Version_Invariants(
	value Header_Extractor_Version, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value), bits.WORD_16_MINIMUM, bits.WORD_16_MAXIMUM).
		Ensure()
}

// Header_Modified_Time is one complete MS-DOS clock word.
type Header_Modified_Time uint16

// Header_Modified_Time_Invariants preserves complete DOS clock words.
func Header_Modified_Time_Invariants(
	value Header_Modified_Time, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value), bits.WORD_16_MINIMUM, bits.WORD_16_MAXIMUM).
		Ensure()
}

// Header_Modified_Date is one complete MS-DOS calendar word.
type Header_Modified_Date uint16

// Header_Modified_Date_Invariants preserves complete DOS calendar words.
func Header_Modified_Date_Invariants(
	value Header_Modified_Date, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value), bits.WORD_16_MINIMUM, bits.WORD_16_MAXIMUM).
		Ensure()
}

// Header_Internal_Attributes is one complete central hint word.
type Header_Internal_Attributes uint16

// Header_Internal_Attributes_Invariants preserves complete hint words.
func Header_Internal_Attributes_Invariants(
	value Header_Internal_Attributes, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value), bits.WORD_16_MINIMUM, bits.WORD_16_MAXIMUM).
		Ensure()
}

// Header_Non_UTF8 preserves deliberate legacy text encoding.
type Header_Non_UTF8 bool

// Header_Non_UTF8_Invariants covers Unicode and legacy text policy.
func Header_Non_UTF8_Invariants(
	value Header_Non_UTF8, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "ZIP header keeps legacy text encoding.").
		Ensure()
}

// Header carries one ZIP member without owning variable wire fields.
type Header struct {
	// Name borrows the validated member path.
	Name Header_Name
	// Comment borrows the validated member comment.
	Comment Header_Comment
	// Extra borrows validated extension records.
	Extra Header_Extra
	// Method selects stored or DEFLATE content.
	Method Header_Method
	// Flags preserve general-purpose wire policy.
	Flags Header_Flags
	// Checksum protects uncompressed content.
	Checksum Header_Checksum
	// Compressed_Size reports stored payload bytes.
	Compressed_Size Header_Compressed_Size
	// Uncompressed_Size reports logical payload bytes.
	Uncompressed_Size Header_Uncompressed_Size
	// External_Attributes preserve portable mode metadata.
	External_Attributes Header_External_Attributes
	// Creator_Version identifies archive host policy.
	Creator_Version Header_Creator_Version
	// Extractor_Version identifies the minimum reader version.
	Extractor_Version Header_Extractor_Version
	// Modified_Time preserves legacy DOS clock bits.
	Modified_Time Header_Modified_Time
	// Modified_Date preserves legacy DOS calendar bits.
	Modified_Date Header_Modified_Date
	// Internal_Attributes preserve central text hints.
	Internal_Attributes Header_Internal_Attributes
	// Modified preserves an extended Unix instant.
	Modified Timestamp
	// Non_UTF8 retains deliberate legacy name encoding.
	Non_UTF8 Header_Non_UTF8
}

// Header_Invariants bounds each borrowed field and classic wire quantity.
func Header_Invariants(value Header, namespace aver.Namespace) {
	Header_Name_Invariants(value.Name, namespace)
	Header_Comment_Invariants(value.Comment, namespace)
	Header_Extra_Invariants(value.Extra, namespace)
	Header_Method_Invariants(value.Method, namespace)
	Header_Flags_Invariants(value.Flags, namespace)
	Header_Checksum_Invariants(value.Checksum, namespace)
	Header_Compressed_Size_Invariants(value.Compressed_Size, namespace)
	Header_Uncompressed_Size_Invariants(value.Uncompressed_Size, namespace)
	Header_External_Attributes_Invariants(value.External_Attributes, namespace)
	Header_Creator_Version_Invariants(value.Creator_Version, namespace)
	Header_Extractor_Version_Invariants(value.Extractor_Version, namespace)
	Header_Modified_Time_Invariants(value.Modified_Time, namespace)
	Header_Modified_Date_Invariants(value.Modified_Date, namespace)
	Header_Internal_Attributes_Invariants(value.Internal_Attributes, namespace)
	Timestamp_Invariants(value.Modified, namespace)
	Header_Non_UTF8_Invariants(value.Non_UTF8, namespace)
}

// Header_Handle keeps caller-owned header storage explicit.
type Header_Handle *Header

// Header_Handle_Invariants composes a present header value.
func Header_Handle_Invariants(value Header_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Header_Invariants(*value, namespace)
}

// Header_Name_Unvalidated admits the first rejected path byte.
type Header_Name_Unvalidated []byte

// Header_Name_Unvalidated_Invariants admits the first rejected path byte.
func Header_Name_Unvalidated_Invariants(
	value Header_Name_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), MEMBER_NAME_SIZE_MINIMUM,
			HEADER_FIELD_SIZE_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Header_Comment_Unvalidated admits the first rejected comment byte.
type Header_Comment_Unvalidated []byte

// Header_Comment_Unvalidated_Invariants admits the first rejected comment byte.
func Header_Comment_Unvalidated_Invariants(
	value Header_Comment_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), MEMBER_NAME_SIZE_MINIMUM,
			HEADER_FIELD_SIZE_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Header_Extra_Unvalidated admits the first rejected extension byte.
type Header_Extra_Unvalidated []byte

// Header_Extra_Unvalidated_Invariants admits the first rejected extension byte.
func Header_Extra_Unvalidated_Invariants(
	value Header_Extra_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), MEMBER_NAME_SIZE_MINIMUM,
			HEADER_FIELD_SIZE_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Header_Compressed_Size_Unvalidated admits the first rejected ZIP64 size.
type Header_Compressed_Size_Unvalidated uint64

// Header_Compressed_Size_Unvalidated_Invariants bounds hostile size input.
func Header_Compressed_Size_Unvalidated_Invariants(
	value Header_Compressed_Size_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint64(
			uint64(value), bits.WORD_64_MINIMUM, HEADER_SIZE_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Header_Uncompressed_Size_Unvalidated admits the first rejected ZIP64 size.
type Header_Uncompressed_Size_Unvalidated uint64

// Header_Uncompressed_Size_Unvalidated_Invariants bounds hostile size input.
func Header_Uncompressed_Size_Unvalidated_Invariants(
	value Header_Uncompressed_Size_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint64(
			uint64(value), bits.WORD_64_MINIMUM, HEADER_SIZE_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Header_Unvalidated keeps hostile field sizes outside writer state.
type Header_Unvalidated struct {
	// Name remains hostile until Header_Validate.
	Name Header_Name_Unvalidated
	// Comment remains hostile until Header_Validate.
	Comment Header_Comment_Unvalidated
	// Extra remains hostile until Header_Validate.
	Extra Header_Extra_Unvalidated
	// Method preserves caller codec input.
	Method Header_Method
	// Flags preserve caller raw policy.
	Flags Header_Flags
	// Checksum preserves caller raw CRC-32.
	Checksum Header_Checksum
	// Compressed_Size preserves caller ZIP64 input.
	Compressed_Size Header_Compressed_Size_Unvalidated
	// Uncompressed_Size preserves caller ZIP64 input.
	Uncompressed_Size Header_Uncompressed_Size_Unvalidated
	// External_Attributes preserve caller mode metadata.
	External_Attributes Header_External_Attributes
	// Creator_Version preserves caller host policy.
	Creator_Version Header_Creator_Version
	// Extractor_Version preserves caller reader version.
	Extractor_Version Header_Extractor_Version
	// Modified_Time preserves caller DOS clock bits.
	Modified_Time Header_Modified_Time
	// Modified_Date preserves caller DOS calendar bits.
	Modified_Date Header_Modified_Date
	// Internal_Attributes preserve caller central hints.
	Internal_Attributes Header_Internal_Attributes
	// Modified preserves caller extended instant.
	Modified Timestamp
	// Non_UTF8 preserves caller legacy text policy.
	Non_UTF8 Header_Non_UTF8
}

// Header_Unvalidated_Invariants admits exactly one rejected metadata boundary.
func Header_Unvalidated_Invariants(
	value Header_Unvalidated, namespace aver.Namespace,
) {
	Header_Name_Unvalidated_Invariants(value.Name, namespace)
	Header_Comment_Unvalidated_Invariants(value.Comment, namespace)
	Header_Extra_Unvalidated_Invariants(value.Extra, namespace)
	Header_Method_Invariants(value.Method, namespace)
	Header_Flags_Invariants(value.Flags, namespace)
	Header_Checksum_Invariants(value.Checksum, namespace)
	Header_Compressed_Size_Unvalidated_Invariants(value.Compressed_Size, namespace)
	Header_Uncompressed_Size_Unvalidated_Invariants(value.Uncompressed_Size, namespace)
	Header_External_Attributes_Invariants(value.External_Attributes, namespace)
	Header_Creator_Version_Invariants(value.Creator_Version, namespace)
	Header_Extractor_Version_Invariants(value.Extractor_Version, namespace)
	Header_Modified_Time_Invariants(value.Modified_Time, namespace)
	Header_Modified_Date_Invariants(value.Modified_Date, namespace)
	Header_Internal_Attributes_Invariants(value.Internal_Attributes, namespace)
	Timestamp_Invariants(value.Modified, namespace)
	Header_Non_UTF8_Invariants(value.Non_UTF8, namespace)
}

// Header_Unvalidated_Handle keeps hostile caller fields in caller-owned storage.
type Header_Unvalidated_Handle *Header_Unvalidated

// Header_Unvalidated_Handle_Invariants composes a present hostile header value.
func Header_Unvalidated_Handle_Invariants(
	value Header_Unvalidated_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Header_Unvalidated_Invariants(*value, namespace)
}

// Header_Validate is the one boundary where hostile lengths enter ZIP state.
func Header_Validate(
	value Header_Unvalidated_Handle,
) (header Header, status Validation_Status) {
	defer func() {
		Header_Invariants(header, "Header_Validate.header")
		Validation_Status_Invariants(
			status, "Header_Validate.status",
		)
	}()
	Header_Unvalidated_Handle_Invariants(value, "Header_Validate.value")
	if len(value.Name) == 0 {
		return Header{}, Validation_Status(STATUS_INPUT_INVALID)
	}
	if len(value.Name) > HEADER_NAME_SIZE_MAXIMUM {
		return Header{}, Validation_Status(STATUS_INPUT_INVALID)
	}
	if len(value.Comment) > HEADER_COMMENT_SIZE_MAXIMUM {
		return Header{}, Validation_Status(STATUS_INPUT_INVALID)
	}
	if len(value.Extra) > HEADER_EXTRA_SIZE_MAXIMUM {
		return Header{}, Validation_Status(STATUS_INPUT_INVALID)
	}
	if uint64(value.Compressed_Size) > HEADER_SIZE_MAXIMUM {
		return Header{}, Validation_Status(STATUS_INPUT_INVALID)
	}
	if uint64(value.Uncompressed_Size) > HEADER_SIZE_MAXIMUM {
		return Header{}, Validation_Status(STATUS_INPUT_INVALID)
	}
	return Header{
		Name:                Header_Name(value.Name),
		Comment:             Header_Comment(value.Comment),
		Extra:               Header_Extra(value.Extra),
		Method:              value.Method,
		Flags:               value.Flags,
		Checksum:            value.Checksum,
		Compressed_Size:     Header_Compressed_Size(value.Compressed_Size),
		Uncompressed_Size:   Header_Uncompressed_Size(value.Uncompressed_Size),
		External_Attributes: value.External_Attributes,
		Creator_Version:     value.Creator_Version,
		Extractor_Version:   value.Extractor_Version,
		Modified_Time:       value.Modified_Time,
		Modified_Date:       value.Modified_Date,
		Internal_Attributes: value.Internal_Attributes,
		Modified:            value.Modified,
		Non_UTF8:            value.Non_UTF8,
	}, Validation_Status(STATUS_OK)
}

// Header_Set_Modification_Time stores extended instant and compatible DOS fields.
func Header_Set_Modification_Time(
	header Header_Unvalidated_Handle, modified Timestamp,
) (status Validation_Status) {
	defer func() {
		Validation_Status_Invariants(
			status, "Header_Set_Modification_Time.status",
		)
	}()
	Header_Unvalidated_Handle_Invariants(header, "Header_Set_Modification_Time.header")
	Timestamp_Invariants(modified, "Header_Set_Modification_Time.modified")
	encoding, valid := timestamp_to_dos(modified)
	if !valid {
		return Validation_Status(STATUS_INPUT_INVALID)
	}
	header.Modified = modified
	header.Modified_Date = Header_Modified_Date(encoding.Date)
	header.Modified_Time = Header_Modified_Time(encoding.Clock)
	return Validation_Status(STATUS_OK)
}

// Header_Modification_Time returns extended instant or legacy DOS value in UTC.
func Header_Modification_Time(header Header_Handle) (modified Timestamp) {
	defer func() {
		Timestamp_Invariants(modified, "Header_Modification_Time.modified")
	}()
	Header_Handle_Invariants(header, "Header_Modification_Time.header")
	if header.Modified.Set {
		return header.Modified
	}
	legacy := timestamp_from_dos(
		binary.Word_16(header.Modified_Date), binary.Word_16(header.Modified_Time),
	)
	if legacy.Half_Second_Offset == 0 {
		return Timestamp{}
	}
	seconds := DOS_TIMESTAMP_SECONDS_MINIMUM +
		(int64(legacy.Half_Second_Offset)-WIRE_TIMESTAMP_PRESENT_OFFSET)*
			DOS_TIMESTAMP_SECOND_PRECISION
	return Timestamp{
		Seconds: Timestamp_Seconds(seconds), Set: true,
	}
}

// Header_Set_Mode stores portable mode through standard Unix ZIP attributes.
func Header_Set_Mode(header Header_Unvalidated_Handle, mode nbio.File_Mode) {
	Header_Unvalidated_Handle_Invariants(header, "Header_Set_Mode.header")
	nbio.File_Mode_Invariants(mode, "Header_Set_Mode.mode")
	header.Creator_Version = header.Creator_Version&CREATOR_SYSTEM_MASK |
		CREATOR_UNIX<<CREATOR_SYSTEM_SHIFT
	header.External_Attributes = Header_External_Attributes(
		binary.Word_32(nbio.File_Mode_To_POSIX(mode)) << EXTERNAL_ATTRIBUTES_UNIX_SHIFT,
	)
	if nbio.File_Mode_Is_Directory(mode) {
		header.External_Attributes |= DOS_FILE_ATTRIBUTE_DIRECTORY
	}
	if mode&nbio.FILE_MODE_OWNER_WRITE == 0 {
		header.External_Attributes |= DOS_FILE_ATTRIBUTE_READ_ONLY
	}
}

// Header_Mode restores portable mode from central external attributes.
func Header_Mode(header Header_Handle) (mode Archive_File_Mode) {
	defer func() { Archive_File_Mode_Invariants(mode, "Header_Mode.mode") }()
	Header_Handle_Invariants(header, "Header_Mode.header")
	creator := header.Creator_Version >> CREATOR_SYSTEM_SHIFT
	unix_creator := creator == CREATOR_UNIX
	if creator == CREATOR_MAC_OS_X {
		unix_creator = true
	}
	if unix_creator {
		mode = Archive_File_Mode(nbio.File_Mode_From_POSIX(uint16(
			header.External_Attributes >> EXTERNAL_ATTRIBUTES_UNIX_SHIFT,
		)))
	} else {
		mode = Archive_File_Mode(dos_to_file_mode(
			DOS_File_Attributes(header.External_Attributes),
		))
	}
	if len(header.Name) != 0 {
		if header.Name[len(header.Name)-1] == '/' {
			mode |= Archive_File_Mode(nbio.FILE_MODE_DIRECTORY)
		}
	}
	return mode
}

func dos_to_file_mode(attributes DOS_File_Attributes) (mode DOS_Archive_File_Mode) {
	defer func() {
		DOS_Archive_File_Mode_Invariants(mode, "dos_to_file_mode.mode")
	}()
	DOS_File_Attributes_Invariants(attributes, "dos_to_file_mode.attributes")
	if attributes&DOS_FILE_ATTRIBUTE_DIRECTORY != 0 {
		mode = DOS_Archive_File_Mode(nbio.FILE_MODE_DIRECTORY | nbio.FILE_MODE_PERMISSIONS)
	} else {
		mode = DOS_Archive_File_Mode(DOS_FILE_MODE_READ_WRITE)
	}
	if attributes&DOS_FILE_ATTRIBUTE_READ_ONLY != 0 {
		mode &^= DOS_Archive_File_Mode(nbio.FILE_MODE_WRITE_PERMISSIONS)
	}
	return mode
}

func timestamp_to_dos(
	modified Timestamp,
) (encoding DOS_Encoding, valid binary.Boolean) {
	defer func() {
		DOS_Encoding_Invariants(encoding, "timestamp_to_dos.encoding")
		binary.Boolean_Invariants(valid, "timestamp_to_dos.valid")
	}()
	Timestamp_Invariants(modified, "timestamp_to_dos.modified")
	if !modified.Set {
		return DOS_Encoding{}, true
	}
	local_seconds := Local_Timestamp_Seconds(
		binary.Integer_64(modified.Seconds) +
			binary.Integer_64(modified.Zone_Offset_Seconds),
	)
	Local_Timestamp_Seconds_Invariants(local_seconds, "timestamp_to_dos.local_seconds")
	split := time.Unix_Second_Split(time.Unix_Second_Count(local_seconds))
	date := time.Civil_From_Days(split.Days)
	year, month, day := date.Year, date.Month, date.Day
	if year < DOS_CIVIL_YEAR_MINIMUM {
		return DOS_Encoding{}, false
	}
	if year > DOS_CIVIL_YEAR_MAXIMUM {
		return DOS_Encoding{}, false
	}
	day_second_count := int64(split.Day_Seconds)
	hour := day_second_count / time.SECOND_COUNT_PER_HOUR
	minute := day_second_count % time.SECOND_COUNT_PER_HOUR /
		time.SECOND_COUNT_PER_MINUTE
	second := day_second_count % time.SECOND_COUNT_PER_MINUTE
	return DOS_Encoding{
		Date: DOS_Encoding_Date(
			int(day) | int(month)<<DOS_DATE_MONTH_SHIFT |
				(int(year)-DOS_CIVIL_YEAR_MINIMUM)<<DOS_DATE_YEAR_SHIFT,
		),
		Clock: DOS_Encoding_Time(
			second/DOS_TIMESTAMP_SECOND_PRECISION |
				minute<<DOS_TIME_MINUTE_SHIFT | hour<<DOS_TIME_HOUR_SHIFT,
		),
		Half_Second_Offset: DOS_Encoding_Half_Second_Offset(
			(int64(local_seconds)-DOS_TIMESTAMP_SECONDS_MINIMUM)/
				DOS_TIMESTAMP_SECOND_PRECISION + 1,
		),
	}, true
}

func timestamp_from_dos(
	date binary.Word_16, time_of_day binary.Word_16,
) (modified DOS_Timestamp) {
	defer func() {
		DOS_Timestamp_Invariants(modified, "timestamp_from_dos.modified")
	}()
	binary.Word_16_Invariants(date, "timestamp_from_dos.date")
	binary.Word_16_Invariants(time_of_day, "timestamp_from_dos.time_of_day")
	if date == 0 {
		if time_of_day == 0 {
			return DOS_Timestamp{}
		}
	}
	year := DOS_Civil_Year(date>>DOS_DATE_YEAR_SHIFT) + DOS_CIVIL_YEAR_MINIMUM
	month := time.Civil_Month(date>>DOS_DATE_MONTH_SHIFT) & DOS_DATE_MONTH_MASK
	day := time.Civil_Day(date & DOS_DATE_DAY_MASK)
	hour := int(time_of_day >> DOS_TIME_HOUR_SHIFT)
	minute := int(time_of_day>>DOS_TIME_MINUTE_SHIFT) & DOS_TIME_MINUTE_MASK
	second := int(time_of_day&DOS_TIME_SECOND_MASK) * DOS_TIMESTAMP_SECOND_PRECISION
	if month < time.CIVIL_MONTH_MINIMUM {
		return DOS_Timestamp{}
	}
	if month > time.CIVIL_MONTH_MAXIMUM {
		return DOS_Timestamp{}
	}
	if day < time.CIVIL_DAY_MINIMUM {
		return DOS_Timestamp{}
	}
	if day > time.CIVIL_DAY_MAXIMUM {
		return DOS_Timestamp{}
	}
	DOS_Civil_Year_Invariants(year, "timestamp_from_dos.year")
	days := DOS_Calendar_Day_Count(time.Days_From_Civil(time.Civil_Date{
		Year: time.Civil_Year(year), Month: month, Day: day,
	}))
	DOS_Calendar_Day_Count_Invariants(days, "timestamp_from_dos.days")
	seconds := int64(days)*time.SECOND_COUNT_PER_DAY +
		int64(hour)*time.SECOND_COUNT_PER_HOUR +
		int64(minute)*time.SECOND_COUNT_PER_MINUTE + int64(second)
	if seconds < DOS_TIMESTAMP_SECONDS_MINIMUM {
		return DOS_Timestamp{}
	}
	if seconds > DOS_TIMESTAMP_SECONDS_MAXIMUM {
		return DOS_Timestamp{}
	}
	return DOS_Timestamp{Half_Second_Offset: DOS_Timestamp_Half_Second_Offset(
		(seconds-DOS_TIMESTAMP_SECONDS_MINIMUM)/
			DOS_TIMESTAMP_SECOND_PRECISION + 1,
	)}
}

func header_modified(extra Header_Extra, date binary.Word_16, clock binary.Word_16,
) (modified Wire_Timestamp) {
	defer func() { Wire_Timestamp_Invariants(modified, "header_modified.modified") }()
	Header_Extra_Invariants(extra, "header_modified.extra")
	binary.Word_16_Invariants(date, "header_modified.date")
	binary.Word_16_Invariants(clock, "header_modified.clock")
	legacy := timestamp_from_dos(date, clock)
	position := 0
	for position <= len(extra)-EXTENDED_TIMESTAMP_HEADER_SIZE {
		identifier := binary.Uint_16(binary.Bytes(extra[position:]), binary.LITTLE_ENDIAN)
		size := int(binary.Uint_16(
			binary.Bytes(extra[position+EXTENDED_TIMESTAMP_DATA_SIZE_POSITION:]),
			binary.LITTLE_ENDIAN,
		))
		position += EXTENDED_TIMESTAMP_HEADER_SIZE
		if size > len(extra)-position {
			break
		}
		if identifier == EXTENDED_TIMESTAMP_IDENTIFIER {
			if size < EXTENDED_TIMESTAMP_DATA_SIZE {
				position += size
				continue
			}
			if extra[position]&EXTENDED_TIMESTAMP_MODIFIED_FLAG == 0 {
				position += size
				continue
			}
			seconds_position := position + EXTENDED_TIMESTAMP_DATA_SECONDS_POSITION
			seconds := binary.Integer_64(binary.Uint_32(
				binary.Bytes(extra[seconds_position:]),
				binary.LITTLE_ENDIAN,
			))
			zone := Timestamp_Zone_Quarter_Hours_Unbounded(0)
			if legacy.Half_Second_Offset != 0 {
				legacy_offset := int64(legacy.Half_Second_Offset) -
					WIRE_TIMESTAMP_PRESENT_OFFSET
				legacy_seconds := DOS_TIMESTAMP_SECONDS_MINIMUM +
					legacy_offset*DOS_TIMESTAMP_SECOND_PRECISION
				zone = quarter_hour_round(Timestamp_Zone_Difference(
					binary.Integer_64(legacy_seconds) - seconds,
				))
			}
			zone_minimum := Timestamp_Zone_Quarter_Hours_Unbounded(
				TIMESTAMP_ZONE_QUARTER_HOUR_MINIMUM)
			zone_maximum := Timestamp_Zone_Quarter_Hours_Unbounded(
				TIMESTAMP_ZONE_QUARTER_HOUR_MAXIMUM)
			if zone < zone_minimum {
				zone = 0
			}
			if zone > zone_maximum {
				zone = 0
			}
			second_offset := Wire_Timestamp_Second_Offset(
				uint64(seconds) + WIRE_TIMESTAMP_PRESENT_OFFSET)
			return Wire_Timestamp{
				Second_Offset:      second_offset,
				Zone_Quarter_Hours: Timestamp_Zone_Quarter_Hours(zone),
			}
		}
		position += size
	}
	if legacy.Half_Second_Offset == 0 {
		return Wire_Timestamp{}
	}
	legacy_seconds := DOS_TIMESTAMP_SECONDS_MINIMUM +
		(int64(legacy.Half_Second_Offset)-WIRE_TIMESTAMP_PRESENT_OFFSET)*
			DOS_TIMESTAMP_SECOND_PRECISION
	return Wire_Timestamp{Second_Offset: Wire_Timestamp_Second_Offset(
		uint64(legacy_seconds) + WIRE_TIMESTAMP_PRESENT_OFFSET,
	)}
}

func quarter_hour_round(
	seconds Timestamp_Zone_Difference,
) (rounded Timestamp_Zone_Quarter_Hours_Unbounded) {
	defer func() {
		Timestamp_Zone_Quarter_Hours_Unbounded_Invariants(
			rounded, "quarter_hour_round.rounded",
		)
	}()
	Timestamp_Zone_Difference_Invariants(seconds, "quarter_hour_round.seconds")
	if seconds >= 0 {
		return Timestamp_Zone_Quarter_Hours_Unbounded(
			(int64(seconds) + TIMESTAMP_ZONE_QUARTER_HOUR_ROUNDING_SECONDS) /
				TIMESTAMP_ZONE_QUARTER_HOUR_SECONDS,
		)
	}
	return Timestamp_Zone_Quarter_Hours_Unbounded(
		(int64(seconds) - TIMESTAMP_ZONE_QUARTER_HOUR_ROUNDING_SECONDS) /
			TIMESTAMP_ZONE_QUARTER_HOUR_SECONDS,
	)
}

// Reader_Storage_Archive is caller memory available for one complete archive.
type Reader_Storage_Archive []byte

// Reader_Storage_Archive_Invariants bounds caller archive storage.
func Reader_Storage_Archive_Invariants(
	value Reader_Storage_Archive, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), POSITION_MINIMUM, ARCHIVE_SIZE_MAXIMUM).
		Ensure()
}

// Reader_Archive is the complete initialized archive view.
type Reader_Archive []byte

// Reader_Archive_Invariants bounds one initialized archive view.
func Reader_Archive_Invariants(value Reader_Archive, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), POSITION_MINIMUM, ARCHIVE_SIZE_MAXIMUM).
		Ensure()
}

// Reader_Comment is the validated directory comment view.
type Reader_Comment []byte

// Reader_Comment_Invariants bounds one validated directory comment.
func Reader_Comment_Invariants(value Reader_Comment, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), POSITION_MINIMUM, HEADER_COMMENT_SIZE_MAXIMUM).
		Ensure()
}

// Reader_Raw_Content borrows one compressed payload inside a bounded archive.
type Reader_Raw_Content []byte

// Reader_Raw_Content_Invariants leaves mandatory ZIP metadata in the archive.
func Reader_Raw_Content_Invariants(
	value Reader_Raw_Content, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), POSITION_MINIMUM, RAW_CONTENT_SIZE_MAXIMUM).
		Ensure()
}

// Reader_Active protects one retained initialization callback.
type Reader_Active bool

// Reader_Active_Invariants covers idle and active initialization.
func Reader_Active_Invariants(value Reader_Active, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "ZIP Reader initialization is active.").
		Ensure()
}

// Reader_Transfer_Buffer retains one pending Stream borrow.
type Reader_Transfer_Buffer []byte

// Reader_Transfer_Buffer_Invariants bounds one pending Stream borrow.
func Reader_Transfer_Buffer_Invariants(
	value Reader_Transfer_Buffer, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), POSITION_MINIMUM, ARCHIVE_SIZE_MAXIMUM).
		Ensure()
}

// Reader_Stream_Offset is one bounded archive coordinate.
type Reader_Stream_Offset int64

// Reader_Stream_Offset_Invariants bounds one explicit Stream coordinate.
func Reader_Stream_Offset_Invariants(
	value Reader_Stream_Offset, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int64(
			int64(value), int64(POSITION_MINIMUM), int64(ARCHIVE_SIZE_MAXIMUM),
		).
		Ensure()
}

// Reader_Submission_Active prevents inline callback recursion.
type Reader_Submission_Active bool

// Reader_Submission_Active_Invariants covers outer and nested Stream retirement.
func Reader_Submission_Active_Invariants(
	value Reader_Submission_Active, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "ZIP Reader is inside Stream Procedure.").
		Ensure()
}

// Reader_Continue asks the submitting frame to process inline retirement.
type Reader_Continue bool

// Reader_Continue_Invariants covers deferred and inline Stream retirement.
func Reader_Continue_Invariants(
	value Reader_Continue, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "ZIP Reader has inline Stream work.").
		Ensure()
}

// Reader_Storage owns bytes retained after injected Stream retires.
type Reader_Storage struct {
	// Archive is complete bounded ZIP storage.
	Archive Reader_Storage_Archive
}

// Reader_Storage_Invariants bounds retained archive storage.
func Reader_Storage_Invariants(value Reader_Storage, namespace aver.Namespace) {
	Reader_Storage_Archive_Invariants(value.Archive, namespace)
}

// Reader_Stage identifies injected Stream continuation.
type Reader_Stage uint8

// Reader_Stage_Invariants closes initialization continuation domain.
func Reader_Stage_Invariants(value Reader_Stage, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(READER_STAGE_NONE), uint8(READER_STAGE_SIZE),
			uint8(READER_STAGE_READ),
		).
		Ensure()
}

// READER_STAGE_NONE means no initialization operation waits.
const READER_STAGE_NONE Reader_Stage = 0

// READER_STAGE_SIZE waits for Stream size.
const READER_STAGE_SIZE Reader_Stage = READER_STAGE_NONE + 1

// READER_STAGE_READ waits for complete archive read.
const READER_STAGE_READ Reader_Stage = READER_STAGE_SIZE + 1

// Reader_Stream_Stage is one pending initialization operation.
type Reader_Stream_Stage uint8

// Reader_Stream_Stage_Invariants excludes the idle public state.
func Reader_Stream_Stage_Invariants(
	value Reader_Stream_Stage, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(READER_STAGE_SIZE), uint8(READER_STAGE_READ),
		).
		Ensure()
}

// Reader retains injected Stream and caller storage, never event-loop ownership.
type Reader struct {
	// Stream owns completion timing.
	Stream nbio.Stream
	// Storage remains caller-owned for Reader lifetime.
	Storage Reader_Storage
	// Archive is populated storage after successful Stream read.
	Archive Reader_Archive
	// Callback retires complete Reader_Init, not internal Stream steps.
	Callback nbio.Callback
	// Status reports ZIP validation separately from Completion.Error.
	Status Status
	// Entry_Count records validated central member count.
	Entry_Count Entry_Count
	// Comment borrows validated EOCD comment bytes from Archive.
	Comment Reader_Comment
	// Stage prevents one callback from applying wrong transition.
	Stage Reader_Stage
	// Active rejects overlapping initialization.
	Active Reader_Active
	// Transfer_Buffer remains borrowed until its Stream operation retires.
	Transfer_Buffer Reader_Transfer_Buffer
	// Stream_Mode selects the pending injected operation.
	Stream_Mode nbio.Stream_Mode
	// Stream_Offset preserves the pending explicit archive coordinate.
	Stream_Offset Reader_Stream_Offset
	// Submitting converts inline callback recursion into iteration.
	Submission_Active Reader_Submission_Active
	// Continue reports inline retirement to the submitting frame.
	Continue Reader_Continue
}

// Reader_Invariants keeps retained archive inside caller storage.
func Reader_Invariants(value Reader, namespace aver.Namespace) {
	Reader_Storage_Invariants(value.Storage, namespace)
	Reader_Archive_Invariants(value.Archive, namespace)
	Status_Invariants(value.Status, namespace)
	Entry_Count_Invariants(value.Entry_Count, namespace)
	Reader_Comment_Invariants(value.Comment, namespace)
	Reader_Stage_Invariants(value.Stage, namespace)
	Reader_Active_Invariants(value.Active, namespace)
	Reader_Transfer_Buffer_Invariants(value.Transfer_Buffer, namespace)
	Reader_Stream_Offset_Invariants(value.Stream_Offset, namespace)
	Reader_Submission_Active_Invariants(value.Submission_Active, namespace)
	Reader_Continue_Invariants(value.Continue, namespace)
	aver.Always(
		len(value.Archive) <= len(value.Storage.Archive),
		"ZIP Reader archive stays inside caller storage.",
	)
}

// Reader_Handle keeps injected Stream state in caller-owned storage.
type Reader_Handle *Reader

// Reader_Handle_Invariants composes a present Reader value.
func Reader_Handle_Invariants(value Reader_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Reader_Invariants(*value, namespace)
}

// File_Info_Name is one normalized filesystem path view.
type File_Info_Name []byte

// File_Info_Name_Invariants bounds one normalized filesystem path.
func File_Info_Name_Invariants(value File_Info_Name, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), POSITION_MINIMUM, HEADER_NAME_SIZE_MAXIMUM).
		Ensure()
}

// File_Info_Size is one logical member size.
type File_Info_Size uint64

// File_Info_Size_Invariants preserves complete classic member sizes.
func File_Info_Size_Invariants(value File_Info_Size, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, HEADER_SIZE_MAXIMUM).
		Ensure()
}

// File_Info_Is_Directory distinguishes traversable nodes.
type File_Info_Is_Directory bool

// File_Info_Is_Directory_Invariants covers file and directory nodes.
func File_Info_Is_Directory_Invariants(
	value File_Info_Is_Directory, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "ZIP filesystem node is a directory.").
		Ensure()
}

// File_Info_Explicit distinguishes archive members from synthesized parents.
type File_Info_Explicit bool

// File_Info_Explicit_Invariants covers explicit and synthesized nodes.
func File_Info_Explicit_Invariants(
	value File_Info_Explicit, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "ZIP filesystem node has an archive member.").
		Ensure()
}

// FILE_INFO_EXPLICIT_PRESENT is the only state for a parsed archive member.
const FILE_INFO_EXPLICIT_PRESENT File_Info_Explicit = true

// File_Info_Header_Index is one valid named member position.
type File_Info_Header_Index int

// File_Info_Header_Index_Invariants excludes impossible nameless positions.
func File_Info_Header_Index_Invariants(
	value File_Info_Header_Index, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), ENTRY_COUNT_MINIMUM, FILE_SYSTEM_HEADER_INDEX_MAXIMUM,
		).
		Ensure()
}

// File_Info is one bounded ZIP filesystem node borrowing Reader archive bytes.
type File_Info struct {
	// Name is full slash-separated path without directory suffix slash.
	Name File_Info_Name
	// Size is logical file content size.
	Size File_Info_Size
	// Mode preserves permission and kind bits.
	Mode nbio.File_Mode
	// Modified preserves decoded modification time.
	Modified Timestamp_Second_Precision
	// Header_Index identifies explicit archive member when present.
	Header_Index File_Info_Header_Index
	// Is_Directory identifies explicit or synthesized directory.
	Is_Directory File_Info_Is_Directory
	// Explicit distinguishes synthesized parent from archive member.
	Explicit File_Info_Explicit
}

// File_Info_Invariants bounds borrowed path and metadata.
func File_Info_Invariants(value File_Info, namespace aver.Namespace) {
	File_Info_Name_Invariants(value.Name, namespace)
	File_Info_Size_Invariants(value.Size, namespace)
	nbio.File_Mode_Invariants(value.Mode, namespace)
	Timestamp_Second_Precision_Invariants(value.Modified, namespace)
	File_Info_Header_Index_Invariants(value.Header_Index, namespace)
	File_Info_Is_Directory_Invariants(value.Is_Directory, namespace)
	File_Info_Explicit_Invariants(value.Explicit, namespace)
}

// File_Info_Handle keeps metadata output in caller-owned storage.
type File_Info_Handle *File_Info

// File_Info_Handle_Invariants composes a present metadata value.
func File_Info_Handle_Invariants(value File_Info_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	File_Info_Invariants(*value, namespace)
}

// File_Info_Addition_Name is one nonempty normalized archive path.
type File_Info_Addition_Name []byte

// File_Info_Addition_Name_Invariants excludes roots handled before node addition.
func File_Info_Addition_Name_Invariants(
	value File_Info_Addition_Name, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), SELECTED_NAME_SIZE_MINIMUM, HEADER_NAME_SIZE_MAXIMUM,
		).
		Ensure()
}

// File_Info_Addition is one parsed or synthesized non-root node.
type File_Info_Addition struct {
	// Name is nonempty because root is installed separately.
	Name File_Info_Addition_Name
	// Size is the classic member logical size.
	Size File_Info_Size
	// Mode contains only reconstructable archive bits.
	Mode Archive_File_Mode
	// Modified retains parsed wire precision.
	Modified Wire_Timestamp
	// Header_Index identifies explicit source member.
	Header_Index File_Info_Header_Index
	// Is_Directory distinguishes traversable nodes.
	Is_Directory File_Info_Is_Directory
	// Explicit distinguishes source members from synthesized parents.
	Explicit File_Info_Explicit
}

// File_Info_Addition_Invariants keeps non-root parsed metadata exact.
func File_Info_Addition_Invariants(
	value File_Info_Addition, namespace aver.Namespace,
) {
	File_Info_Addition_Name_Invariants(value.Name, namespace)
	File_Info_Size_Invariants(value.Size, namespace)
	Archive_File_Mode_Invariants(value.Mode, namespace)
	Wire_Timestamp_Invariants(value.Modified, namespace)
	File_Info_Header_Index_Invariants(value.Header_Index, namespace)
	File_Info_Is_Directory_Invariants(value.Is_Directory, namespace)
	File_Info_Explicit_Invariants(value.Explicit, namespace)
}

// File_Info_Explicit_Addition is one node proven to come from an archive member.
type File_Info_Explicit_Addition File_Info_Addition

// File_Info_Explicit_Addition_Invariants excludes synthesized-node state.
func File_Info_Explicit_Addition_Invariants(
	value File_Info_Explicit_Addition, namespace aver.Namespace,
) {
	mode := nbio.File_Mode(value.Mode)
	Wire_Timestamp_Invariants(value.Modified, namespace)
	aver.Always(
		value.Explicit == FILE_INFO_EXPLICIT_PRESENT,
		"Explicit ZIP File_Info addition comes from an archive member.",
	)
	aver.Tree(value, namespace).
		Range_Int(
			len(value.Name), SELECTED_NAME_SIZE_MINIMUM,
			HEADER_NAME_SIZE_MAXIMUM,
		).
		Range_Uint64(
			uint64(value.Size), bits.WORD_64_MINIMUM, HEADER_SIZE_MAXIMUM,
		).
		Range_Uint32(
			uint32(value.Mode), nbio.FILE_MODE_MINIMUM,
			ARCHIVE_FILE_MODE_MAXIMUM,
		).
		Range_Uint32(
			uint32(mode&nbio.FILE_MODE_PERMISSIONS), nbio.FILE_MODE_PERMISSION_MINIMUM,
			nbio.FILE_MODE_PERMISSION_MAXIMUM,
		).
		Sometimes(nbio.File_Mode_Is_Directory(mode), "ZIP mode is a directory.").
		Sometimes(mode&nbio.FILE_MODE_SYMBOLIC_LINK != 0, "ZIP mode is a symbolic link.").
		Sometimes(mode&nbio.FILE_MODE_NAMED_PIPE != 0, "ZIP mode is a named pipe.").
		Sometimes(mode&nbio.FILE_MODE_SOCKET != 0, "ZIP mode is a socket.").
		Sometimes(mode&nbio.FILE_MODE_DEVICE != 0, "ZIP mode is a device.").
		Sometimes(
			mode&nbio.FILE_MODE_CHARACTER_DEVICE != 0,
			"ZIP mode is a character device.",
		).
		Sometimes(
			mode&nbio.FILE_MODE_SET_USER_IDENTIFIER != 0,
			"ZIP mode has set-user-ID.",
		).
		Sometimes(
			mode&nbio.FILE_MODE_SET_GROUP_IDENTIFIER != 0,
			"ZIP mode has set-group-ID.",
		).
		Sometimes(mode&nbio.FILE_MODE_STICKY != 0, "ZIP mode has sticky bit.").
		Range_Int(
			int(value.Header_Index), ENTRY_COUNT_MINIMUM,
			FILE_SYSTEM_HEADER_INDEX_MAXIMUM,
		).
		Sometimes(
			bool(value.Is_Directory),
			"Explicit ZIP File_Info addition identifies a directory.",
		).
		Ensure()
}

// Directory_Entry is one immediate child borrowing Reader archive bytes.
type Directory_Entry struct {
	// Name is child component, not full path.
	Name bytes.Slice
	// Is_Directory controls traversal.
	Is_Directory binary.Boolean
}

// Directory_Entry_Invariants bounds one borrowed component.
func Directory_Entry_Invariants(value Directory_Entry, namespace aver.Namespace) {
	bytes.Slice_Invariants(value.Name, namespace)
	binary.Boolean_Invariants(value.Is_Directory, namespace)
}

// Directory_Entries is caller-owned bounded directory output.
type Directory_Entries []Directory_Entry

// Directory_Entries_Invariants bounds child count by classic member count.
func Directory_Entries_Invariants(
	value Directory_Entries, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), ENTRY_COUNT_MINIMUM, ENTRY_COUNT_MAXIMUM).
		Ensure()
}

// File_Infos is caller-owned bounded filesystem node output.
type File_Infos []File_Info

// File_Infos_Invariants bounds node count by archive byte budget.
func File_Infos_Invariants(value File_Infos, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), ENTRY_COUNT_MINIMUM, ARCHIVE_SIZE_MAXIMUM).
		Ensure()
}

// File_System_Valid distinguishes accepted archive paths from unbound state.
type File_System_Valid bool

// File_System_Valid_Invariants covers unbound and validated path state.
func File_System_Valid_Invariants(
	value File_System_Valid, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "ZIP filesystem paths are validated.").
		Ensure()
}

// File_System_Reader retains initialized Reader state without pointer lifetime coupling.
type File_System_Reader Reader

// File_System_Reader_Invariants composes one retained initialized Reader.
func File_System_Reader_Invariants(
	value File_System_Reader, namespace aver.Namespace,
) {
	Reader_Storage_Invariants(value.Storage, namespace)
	aver.Tree(value, namespace).
		Range_Int(len(value.Archive), POSITION_MINIMUM, ARCHIVE_SIZE_MAXIMUM).
		Range_Uint8(uint8(value.Status), STATUS_MINIMUM, STATUS_MAXIMUM).
		Range_Int(
			int(value.Entry_Count), ENTRY_COUNT_MINIMUM, ENTRY_COUNT_MAXIMUM,
		).
		Range_Int(
			len(value.Comment), POSITION_MINIMUM, HEADER_COMMENT_SIZE_MAXIMUM,
		).
		Enum_3_Uint8(
			uint8(value.Stage), uint8(READER_STAGE_NONE),
			uint8(READER_STAGE_SIZE), uint8(READER_STAGE_READ),
		).
		Sometimes(bool(value.Active), "Filesystem Reader initialization is active.").
		Range_Int(
			len(value.Transfer_Buffer), POSITION_MINIMUM, ARCHIVE_SIZE_MAXIMUM,
		).
		Range_Int64(
			int64(value.Stream_Offset), int64(POSITION_MINIMUM),
			int64(ARCHIVE_SIZE_MAXIMUM),
		).
		Sometimes(
			bool(value.Submission_Active),
			"Filesystem Reader is inside Stream Procedure.",
		).
		Sometimes(bool(value.Continue), "Filesystem Reader has inline Stream work.").
		Ensure()
	aver.Always(
		len(value.Archive) <= len(value.Storage.Archive),
		"ZIP filesystem Reader archive stays inside retained storage.",
	)
}

// File_System binds path operations to one initialized Reader.
type File_System struct {
	// Reader retains the initialized archive view and caller storage ownership.
	Reader File_System_Reader
	// Valid records complete member-path validation.
	Valid File_System_Valid
}

// File_System_Invariants requires bound Reader after successful initialization.
func File_System_Invariants(value File_System, namespace aver.Namespace) {
	File_System_Reader_Invariants(value.Reader, namespace)
	File_System_Valid_Invariants(value.Valid, namespace)
}

// File_System_Handle keeps path state in caller-owned storage.
type File_System_Handle *File_System

// File_System_Handle_Invariants composes a present filesystem value.
func File_System_Handle_Invariants(
	value File_System_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	File_System_Invariants(*value, namespace)
}

// File_System_Archive is one archive after filesystem readiness validation.
type File_System_Archive []byte

// File_System_Archive_Invariants keeps one complete bounded directory end.
func File_System_Archive_Invariants(
	value File_System_Archive, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), ARCHIVE_SIZE_MINIMUM, ARCHIVE_SIZE_MAXIMUM).
		Ensure()
}

// File_System_Entry_Count is valid named members that fit bounded input.
type File_System_Entry_Count int

// File_System_Entry_Count_Invariants reserves one name byte for every member.
func File_System_Entry_Count_Invariants(
	value File_System_Entry_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), ENTRY_COUNT_MINIMUM, FILE_SYSTEM_ENTRY_COUNT_MAXIMUM,
		).
		Ensure()
}

// Directory_Output_Count is the number of immediate children returned.
type Directory_Output_Count int

// Directory_Output_Count_Invariants bounds output by archive member count.
func Directory_Output_Count_Invariants(
	value Directory_Output_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), ENTRY_COUNT_MINIMUM, FILE_SYSTEM_ENTRY_COUNT_MAXIMUM,
		).
		Ensure()
}

// File_System_Walk_Count includes root beside bounded archive members.
type File_System_Walk_Count int

// File_System_Walk_Count_Invariants bounds collected traversal nodes.
func File_System_Walk_Count_Invariants(
	value File_System_Walk_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), ENTRY_COUNT_MINIMUM, FILE_SYSTEM_WALK_COUNT_MAXIMUM,
		).
		Ensure()
}

// Collected_Directory_Entries contains only archive-derived immediate children.
type Collected_Directory_Entries []Directory_Entry

// Collected_Directory_Entries_Invariants bounds sorting and duplicate scans.
func Collected_Directory_Entries_Invariants(
	value Collected_Directory_Entries, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), ENTRY_COUNT_MINIMUM, FILE_SYSTEM_ENTRY_COUNT_MAXIMUM,
		).
		Ensure()
}

// Previous_Directory_Entries excludes the child currently being considered.
type Previous_Directory_Entries []Directory_Entry

// Previous_Directory_Entries_Invariants leaves one archive member unseen.
func Previous_Directory_Entries_Invariants(
	value Previous_Directory_Entries, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), ENTRY_COUNT_MINIMUM,
			FILE_SYSTEM_DIRECTORY_PREVIOUS_COUNT_MAXIMUM,
		).
		Ensure()
}

// Collected_File_Infos includes synthesized root and implicit directories.
type Collected_File_Infos []File_Info

// Collected_File_Infos_Invariants bounds completed traversal sorting.
func Collected_File_Infos_Invariants(
	value Collected_File_Infos, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), SELECTED_NAME_SIZE_MINIMUM,
			FILE_SYSTEM_WALK_COUNT_MAXIMUM,
		).
		Ensure()
}

// File_Info_Addition_Storage is nonempty caller scratch after root insertion.
type File_Info_Addition_Storage []File_Info

// File_Info_Addition_Storage_Invariants preserves complete caller capacity.
func File_Info_Addition_Storage_Invariants(
	value File_Info_Addition_Storage, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), SELECTED_NAME_SIZE_MINIMUM, bytes.SLICE_SIZE_MAXIMUM,
		).
		Ensure()
}

// File_Info_Previous_Count is collected state before one candidate insertion.
type File_Info_Previous_Count int

// File_Info_Previous_Count_Invariants leaves one component available to add.
func File_Info_Previous_Count_Invariants(
	value File_Info_Previous_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), SELECTED_NAME_SIZE_MINIMUM,
			FILE_SYSTEM_WALK_PREVIOUS_COUNT_MAXIMUM,
		).
		Ensure()
}

// File_Info_Result_Count includes the retained root after candidate handling.
type File_Info_Result_Count int

// File_Info_Result_Count_Invariants excludes zero after root insertion.
func File_Info_Result_Count_Invariants(
	value File_Info_Result_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), SELECTED_NAME_SIZE_MINIMUM, FILE_SYSTEM_WALK_COUNT_MAXIMUM,
		).
		Ensure()
}

// File_System_Source is the state collection needs after public readiness checks.
type File_System_Source struct {
	// Archive excludes incomplete state before directory traversal.
	Archive File_System_Archive
	// Entry_Count excludes counts that cannot carry valid names.
	Entry_Count File_System_Entry_Count
}

// File_System_Source_Invariants composes traversal-only Reader state.
func File_System_Source_Invariants(
	value File_System_Source, namespace aver.Namespace,
) {
	File_System_Archive_Invariants(value.Archive, namespace)
	File_System_Entry_Count_Invariants(value.Entry_Count, namespace)
}

// File_System_Path is one validated root or member path.
type File_System_Path []byte

// File_System_Path_Invariants excludes empty and oversized public paths.
func File_System_Path_Invariants(
	value File_System_Path, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), SELECTED_NAME_SIZE_MINIMUM, HEADER_NAME_SIZE_MAXIMUM).
		Ensure()
}

// File_System_Parent_Path leaves slash and child room in one member name.
type File_System_Parent_Path []byte

// File_System_Parent_Path_Invariants bounds paths that can own descendants.
func File_System_Parent_Path_Invariants(
	value File_System_Parent_Path, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), SELECTED_NAME_SIZE_MINIMUM,
			FILE_SYSTEM_PARENT_PATH_SIZE_MAXIMUM,
		).
		Ensure()
}

// File_System_Member_Name is one validated archive member name.
type File_System_Member_Name []byte

// File_System_Member_Name_Invariants excludes names rejected during binding.
func File_System_Member_Name_Invariants(
	value File_System_Member_Name, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), SELECTED_NAME_SIZE_MINIMUM, HEADER_NAME_SIZE_MAXIMUM).
		Ensure()
}

// File_System_Child_Name is one optional immediate component.
type File_System_Child_Name []byte

// File_System_Child_Name_Invariants admits empty failure and bounded components.
func File_System_Child_Name_Invariants(
	value File_System_Child_Name, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), POSITION_MINIMUM, HEADER_NAME_SIZE_MAXIMUM).
		Ensure()
}

// File_System_Entry_Name is one present immediate component.
type File_System_Entry_Name []byte

// File_System_Entry_Name_Invariants excludes absent child names.
func File_System_Entry_Name_Invariants(
	value File_System_Entry_Name, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), SELECTED_NAME_SIZE_MINIMUM, HEADER_NAME_SIZE_MAXIMUM).
		Ensure()
}

// File_System_Normalized_Name is one member name without directory suffix.
type File_System_Normalized_Name []byte

// File_System_Normalized_Name_Invariants keeps validated member bounds.
func File_System_Normalized_Name_Invariants(
	value File_System_Normalized_Name, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), SELECTED_NAME_SIZE_MINIMUM, HEADER_NAME_SIZE_MAXIMUM).
		Ensure()
}

// File_System_Component is one possibly empty component from a bounded path.
type File_System_Component []byte

// File_System_Component_Invariants bounds hostile path validation scans.
func File_System_Component_Invariants(
	value File_System_Component, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), POSITION_MINIMUM, HEADER_NAME_SIZE_MAXIMUM).
		Ensure()
}

// File_System_Walk_Callback receives sorted root and descendant metadata.
type File_System_Walk_Callback func(info File_Info) (keep_going binary.Boolean)

// Writer_Record_Name is a member name after complete-record fit validation.
type Writer_Record_Name []byte

// Writer_Record_Name_Invariants bounds a name that fits one central record.
func Writer_Record_Name_Invariants(
	value Writer_Record_Name, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), SELECTED_NAME_SIZE_MINIMUM, WRITER_RECORD_NAME_SIZE_MAXIMUM).
		Ensure()
}

// Writer_Record_Comment is a member comment after complete-record fit validation.
type Writer_Record_Comment []byte

// Writer_Record_Comment_Invariants bounds a comment that fits beside one name byte.
func Writer_Record_Comment_Invariants(
	value Writer_Record_Comment, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), POSITION_MINIMUM, WRITER_RECORD_TAIL_SIZE_MAXIMUM).
		Ensure()
}

// Writer_Record_Extra is extension data after complete-record fit validation.
type Writer_Record_Extra []byte

// Writer_Record_Extra_Invariants bounds extensions beside one name byte.
func Writer_Record_Extra_Invariants(
	value Writer_Record_Extra, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), POSITION_MINIMUM, WRITER_RECORD_TAIL_SIZE_MAXIMUM).
		Ensure()
}

// Writer_Record_Extra_Size includes the optional extended timestamp record.
type Writer_Record_Extra_Size int

// Writer_Record_Extra_Size_Invariants bounds complete extension bytes.
func Writer_Record_Extra_Size_Invariants(
	value Writer_Record_Extra_Size, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), POSITION_MINIMUM, WRITER_RECORD_EXTRA_SIZE_MAXIMUM,
		).
		Ensure()
}

// Writer_Record_Method is one method accepted by bounded Writer output.
type Writer_Record_Method uint16

// Writer_Record_Method_Invariants closes stored and DEFLATE output methods.
func Writer_Record_Method_Invariants(
	value Writer_Record_Method, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Uint16(uint16(value), METHOD_STORE, METHOD_DEFLATE).
		Ensure()
}

// Writer_Local_Record_Storage is local-record tail after fit validation.
type Writer_Local_Record_Storage []byte

// Writer_Local_Record_Storage_Invariants keeps one complete local record.
func Writer_Local_Record_Storage_Invariants(
	value Writer_Local_Record_Storage, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), WRITER_LOCAL_STORAGE_SIZE_MINIMUM, ARCHIVE_SIZE_MAXIMUM,
		).
		Ensure()
}

// Writer_Central_Record_Storage is central-record tail after fit validation.
type Writer_Central_Record_Storage []byte

// Writer_Central_Record_Storage_Invariants keeps one complete central record.
func Writer_Central_Record_Storage_Invariants(
	value Writer_Central_Record_Storage, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), WRITER_CENTRAL_STORAGE_SIZE_MINIMUM, ARCHIVE_SIZE_MAXIMUM,
		).
		Ensure()
}

// Writer_Word_16_Destination is one fixed-record tail that receives a word.
type Writer_Word_16_Destination []byte

// Writer_Word_16_Destination_Invariants protects the complete encoded word.
func Writer_Word_16_Destination_Invariants(
	value Writer_Word_16_Destination, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), WRITER_WORD_16_DESTINATION_SIZE_MINIMUM,
			WRITER_WORD_16_DESTINATION_SIZE_MAXIMUM,
		).
		Ensure()
}

// Writer_Word_32_Destination is one fixed-record tail that receives a word.
type Writer_Word_32_Destination []byte

// Writer_Word_32_Destination_Invariants protects the complete encoded word.
func Writer_Word_32_Destination_Invariants(
	value Writer_Word_32_Destination, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), WRITER_WORD_32_DESTINATION_SIZE_MINIMUM,
			WRITER_WORD_32_DESTINATION_SIZE_MAXIMUM,
		).
		Ensure()
}

// Writer_Finalized_Compressed_Size is payload bytes that fit after a local prefix.
type Writer_Finalized_Compressed_Size uint64

// Writer_Finalized_Compressed_Size_Invariants bounds retained payload output.
func Writer_Finalized_Compressed_Size_Invariants(
	value Writer_Finalized_Compressed_Size, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint64(
			uint64(value), bits.WORD_64_MINIMUM, WRITER_LOCAL_START_MAXIMUM,
		).
		Ensure()
}

// Writer_Finalized_Uncompressed_Size preserves complete classic raw metadata.
type Writer_Finalized_Uncompressed_Size uint64

// Writer_Finalized_Uncompressed_Size_Invariants protects central conversion.
func Writer_Finalized_Uncompressed_Size_Invariants(
	value Writer_Finalized_Uncompressed_Size, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, HEADER_SIZE_MAXIMUM).
		Ensure()
}

// Writer_Local_Start locates one local record after fit validation.
type Writer_Local_Start int

// Writer_Local_Start_Invariants leaves a complete minimum local record.
func Writer_Local_Start_Invariants(
	value Writer_Local_Start, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), POSITION_MINIMUM, WRITER_LOCAL_START_MAXIMUM).
		Ensure()
}

// Writer_Record is metadata after method, name, and storage validation.
type Writer_Record struct {
	// Name excludes paths that cannot leave a complete fixed prefix.
	Name Writer_Record_Name
	// Comment excludes tails that cannot fit central staging.
	Comment Writer_Record_Comment
	// Extra excludes extensions that cannot fit central staging.
	Extra Writer_Record_Extra
	// Extra_Size includes generated bytes before any slicing occurs.
	Extra_Size Writer_Record_Extra_Size
	// Flags are normalized once so local and central records agree.
	Flags Header_Flags
	// Method is normalized once so directories remain stored.
	Method Writer_Record_Method
	// Creator_Version preserves explicit caller compatibility policy.
	Creator_Version Header_Creator_Version
	// Extractor_Version preserves explicit caller compatibility policy.
	Extractor_Version Header_Extractor_Version
	// Modified_Time keeps both record copies byte-identical.
	Modified_Time Header_Modified_Time
	// Modified_Date keeps both record copies byte-identical.
	Modified_Date Header_Modified_Date
	// External_Attributes keep portable modes in the central record.
	External_Attributes Header_External_Attributes
	// Internal_Attributes keep caller text hints in the central record.
	Internal_Attributes Header_Internal_Attributes
	// Modified_Seconds supplies the bounded extended timestamp payload.
	Modified_Seconds Timestamp_Seconds
	// Modified_Set prevents an absent timestamp from growing each record.
	Modified_Set Timestamp_Set
	// Local_Start keeps the central pointer inside the same ZIP segment.
	Local_Start Writer_Local_Start
}

// Writer_Record_Invariants composes one record that already fits caller storage.
func Writer_Record_Invariants(value Writer_Record, namespace aver.Namespace) {
	Writer_Record_Name_Invariants(value.Name, namespace)
	Writer_Record_Comment_Invariants(value.Comment, namespace)
	Writer_Record_Extra_Invariants(value.Extra, namespace)
	Writer_Record_Extra_Size_Invariants(value.Extra_Size, namespace)
	Header_Flags_Invariants(value.Flags, namespace)
	Writer_Record_Method_Invariants(value.Method, namespace)
	Header_Creator_Version_Invariants(value.Creator_Version, namespace)
	Header_Extractor_Version_Invariants(value.Extractor_Version, namespace)
	Header_Modified_Time_Invariants(value.Modified_Time, namespace)
	Header_Modified_Date_Invariants(value.Modified_Date, namespace)
	Header_External_Attributes_Invariants(value.External_Attributes, namespace)
	Header_Internal_Attributes_Invariants(value.Internal_Attributes, namespace)
	Timestamp_Seconds_Invariants(value.Modified_Seconds, namespace)
	Timestamp_Set_Invariants(value.Modified_Set, namespace)
	Writer_Local_Start_Invariants(value.Local_Start, namespace)
}

// Writer_Archive_Storage stages local records and final directory bytes.
type Writer_Archive_Storage []byte

// Writer_Archive_Storage_Invariants bounds local and final archive storage.
func Writer_Archive_Storage_Invariants(
	value Writer_Archive_Storage, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), POSITION_MINIMUM, ARCHIVE_SIZE_MAXIMUM).
		Ensure()
}

// Writer_Central_Storage stages central records until close.
type Writer_Central_Storage []byte

// Writer_Central_Storage_Invariants bounds central staging storage.
func Writer_Central_Storage_Invariants(
	value Writer_Central_Storage, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), POSITION_MINIMUM, ARCHIVE_SIZE_MAXIMUM).
		Ensure()
}

// Writer_Content_Storage retains one logical or raw payload.
type Writer_Content_Storage []byte

// Writer_Content_Storage_Invariants bounds retained logical content.
func Writer_Content_Storage_Invariants(
	value Writer_Content_Storage, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), POSITION_MINIMUM, ARCHIVE_SIZE_MAXIMUM).
		Ensure()
}

// Writer_Compressed_Storage receives bounded DEFLATE output.
type Writer_Compressed_Storage []byte

// Writer_Compressed_Storage_Invariants bounds DEFLATE output storage.
func Writer_Compressed_Storage_Invariants(
	value Writer_Compressed_Storage, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), POSITION_MINIMUM, ARCHIVE_SIZE_MAXIMUM).
		Ensure()
}

// Writer_Comment_Storage retains the directory comment.
type Writer_Comment_Storage []byte

// Writer_Comment_Storage_Invariants bounds directory comment storage.
func Writer_Comment_Storage_Invariants(
	value Writer_Comment_Storage, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), POSITION_MINIMUM, ARCHIVE_SIZE_MAXIMUM).
		Ensure()
}

// Writer_Name_Storage stages one synthesized directory name.
type Writer_Name_Storage []byte

// Writer_Name_Storage_Invariants bounds synthesized directory names.
func Writer_Name_Storage_Invariants(
	value Writer_Name_Storage, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), POSITION_MINIMUM, ARCHIVE_SIZE_MAXIMUM).
		Ensure()
}

// Writer_Storage owns every byte retained across Writer calls.
type Writer_Storage struct {
	// Archive stages local records and final central bytes.
	Archive Writer_Archive_Storage
	// Central stages central records until close.
	Central Writer_Central_Storage
	// Content retains current logical or raw payload.
	Content Writer_Content_Storage
	// Compressed receives bounded DEFLATE output.
	Compressed Writer_Compressed_Storage
	// Comment retains directory comment bytes.
	Comment Writer_Comment_Storage
	// Name stages one synthesized directory path.
	Name Writer_Name_Storage
	// Heads is caller-owned DEFLATE hash storage.
	Heads flate.Hash_Positions_Unvalidated
	// Previous is caller-owned DEFLATE history storage.
	Previous flate.History_Positions_Unvalidated
}

// Writer_Storage_Invariants bounds every retained byte and workspace index.
func Writer_Storage_Invariants(value Writer_Storage, namespace aver.Namespace) {
	Writer_Archive_Storage_Invariants(value.Archive, namespace)
	Writer_Central_Storage_Invariants(value.Central, namespace)
	Writer_Content_Storage_Invariants(value.Content, namespace)
	Writer_Compressed_Storage_Invariants(value.Compressed, namespace)
	Writer_Comment_Storage_Invariants(value.Comment, namespace)
	Writer_Name_Storage_Invariants(value.Name, namespace)
	flate.Hash_Positions_Unvalidated_Invariants(value.Heads, namespace)
	flate.History_Positions_Unvalidated_Invariants(value.Previous, namespace)
}

// Writer_Count is complete archive bytes after flush or close.
type Writer_Count int

// Writer_Count_Invariants bounds complete emitted archive bytes.
func Writer_Count_Invariants(value Writer_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), POSITION_MINIMUM, ARCHIVE_SIZE_MAXIMUM).
		Ensure()
}

// Writer_Archive_Position is first unused local archive byte.
type Writer_Archive_Position int

// Writer_Archive_Position_Invariants bounds local archive progress.
func Writer_Archive_Position_Invariants(
	value Writer_Archive_Position, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), POSITION_MINIMUM, ARCHIVE_SIZE_MAXIMUM).
		Ensure()
}

// Writer_Central_Position is first unused central staging byte.
type Writer_Central_Position int

// Writer_Central_Position_Invariants bounds central staging progress.
func Writer_Central_Position_Invariants(
	value Writer_Central_Position, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), POSITION_MINIMUM, ARCHIVE_SIZE_MAXIMUM).
		Ensure()
}

// Writer_Content_Position is first unused current payload byte.
type Writer_Content_Position int

// Writer_Content_Position_Invariants bounds current payload progress.
func Writer_Content_Position_Invariants(
	value Writer_Content_Position, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), POSITION_MINIMUM, ARCHIVE_SIZE_MAXIMUM).
		Ensure()
}

// Writer_Current_Central_Position locates the member awaiting size fields.
type Writer_Current_Central_Position int

// Writer_Current_Central_Position_Invariants bounds pending central location.
func Writer_Current_Central_Position_Invariants(
	value Writer_Current_Central_Position, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), POSITION_MINIMUM, ARCHIVE_SIZE_MAXIMUM).
		Ensure()
}

// Writer_Offset places one ZIP segment after caller stream prefix.
type Writer_Offset int

// Writer_Offset_Invariants bounds the caller stream prefix.
func Writer_Offset_Invariants(value Writer_Offset, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), POSITION_MINIMUM, ARCHIVE_SIZE_MAXIMUM).
		Ensure()
}

// Writer_Comment_Count is copied directory comment bytes.
type Writer_Comment_Count int

// Writer_Comment_Count_Invariants bounds copied comment bytes.
func Writer_Comment_Count_Invariants(
	value Writer_Comment_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), POSITION_MINIMUM, HEADER_COMMENT_SIZE_MAXIMUM).
		Ensure()
}

// Writer_Directory_State is only state required to emit a central directory.
type Writer_Directory_State struct {
	// Archive receives staged central records and directory end.
	Archive Writer_Archive_Storage
	// Central holds complete central records.
	Central Writer_Central_Storage
	// Comment holds the selected directory comment prefix.
	Comment Writer_Comment_Storage
	// Archive_Position begins central output.
	Archive_Position Writer_Archive_Position
	// Central_Position bounds staged central bytes.
	Central_Position Writer_Central_Position
	// Entry_Count is the classic central member count.
	Entry_Count Entry_Count
	// Comment_Count bounds copied comment bytes.
	Comment_Count Writer_Comment_Count
}

// Writer_Directory_State_Invariants excludes unrelated mutable Writer state.
func Writer_Directory_State_Invariants(
	value Writer_Directory_State, namespace aver.Namespace,
) {
	Writer_Archive_Storage_Invariants(value.Archive, namespace)
	Writer_Central_Storage_Invariants(value.Central, namespace)
	Writer_Comment_Storage_Invariants(value.Comment, namespace)
	Writer_Archive_Position_Invariants(value.Archive_Position, namespace)
	Writer_Central_Position_Invariants(value.Central_Position, namespace)
	Entry_Count_Invariants(value.Entry_Count, namespace)
	Writer_Comment_Count_Invariants(value.Comment_Count, namespace)
}

// Writer_Directory_State_Handle keeps final directory output caller-owned.
type Writer_Directory_State_Handle *Writer_Directory_State

// Writer_Directory_State_Handle_Invariants composes present directory state.
func Writer_Directory_State_Handle_Invariants(
	value Writer_Directory_State_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Writer_Directory_State_Invariants(*value, namespace)
}

// Writer_Active distinguishes an open member from idle state.
type Writer_Active bool

// Writer_Active_Invariants covers idle and active member state.
func Writer_Active_Invariants(value Writer_Active, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "ZIP Writer has an active member.").
		Ensure()
}

// Writer_Raw distinguishes compressed caller bytes from logical content.
type Writer_Raw bool

// Writer_Raw_Invariants covers logical and compressed caller content.
func Writer_Raw_Invariants(value Writer_Raw, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "ZIP Writer accepts raw compressed content.").
		Ensure()
}

// Writer_Directory rejects content for directory members.
type Writer_Directory bool

// Writer_Directory_Invariants covers regular and directory members.
func Writer_Directory_Invariants(
	value Writer_Directory, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "ZIP Writer member is a directory.").
		Ensure()
}

// Writer_Closed rejects mutation after directory emission.
type Writer_Closed bool

// Writer_Closed_Invariants covers mutable and finalized archives.
func Writer_Closed_Invariants(value Writer_Closed, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "ZIP Writer emitted the directory.").
		Ensure()
}

// Writer_In_Flight protects one retained Stream callback.
type Writer_In_Flight bool

// Writer_In_Flight_Invariants covers idle and retained Stream callbacks.
func Writer_In_Flight_Invariants(
	value Writer_In_Flight, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "ZIP Writer has a Stream write in flight.").
		Ensure()
}

// Writer retains one bounded archive assembly and injected output Stream.
type Writer struct {
	// Stream owns output completion timing.
	Stream nbio.Stream
	// Storage remains caller-owned for Writer lifetime.
	Storage Writer_Storage
	// Header keeps fixed current member metadata after variable fields are copied.
	Header Header
	// Callback retires complete flush or close operation.
	Callback nbio.Callback
	// Status reports ZIP validation separately from Completion.Error.
	Status Status
	// Count reports complete archive bytes after close.
	Count Writer_Count
	// Archive_Position is first unused local or final archive byte.
	Archive_Position Writer_Archive_Position
	// Central_Position is first unused central staging byte.
	Central_Position Writer_Central_Position
	// Content_Position is first unused current payload byte.
	Content_Position Writer_Content_Position
	// Current_Central_Position locates central record awaiting size fields.
	Current_Central_Position Writer_Current_Central_Position
	// Offset places ZIP segment after caller-owned stream prefix.
	Offset Writer_Offset
	// Entry_Count tracks finalized classic members.
	Entry_Count Entry_Count
	// Comment_Count bounds bytes copied into caller comment storage.
	Comment_Count Writer_Comment_Count
	// Active identifies current member.
	Active Writer_Active
	// Raw distinguishes precompressed content from logical content.
	Raw Writer_Raw
	// Directory rejects nonempty content and descriptor emission.
	Directory Writer_Directory
	// Closed rejects mutation after central directory emission.
	Closed Writer_Closed
	// In_Flight rejects overlapping Stream writes.
	In_Flight Writer_In_Flight
}

// Writer_Invariants keeps every cursor inside caller storage.
func Writer_Invariants(value Writer, namespace aver.Namespace) {
	Writer_Storage_Invariants(value.Storage, namespace)
	Header_Invariants(value.Header, namespace)
	Status_Invariants(value.Status, namespace)
	Writer_Count_Invariants(value.Count, namespace)
	Writer_Archive_Position_Invariants(value.Archive_Position, namespace)
	Writer_Central_Position_Invariants(value.Central_Position, namespace)
	Writer_Content_Position_Invariants(value.Content_Position, namespace)
	Writer_Current_Central_Position_Invariants(
		value.Current_Central_Position, namespace,
	)
	Writer_Offset_Invariants(value.Offset, namespace)
	Entry_Count_Invariants(value.Entry_Count, namespace)
	Writer_Comment_Count_Invariants(value.Comment_Count, namespace)
	Writer_Active_Invariants(value.Active, namespace)
	Writer_Raw_Invariants(value.Raw, namespace)
	Writer_Directory_Invariants(value.Directory, namespace)
	Writer_Closed_Invariants(value.Closed, namespace)
	Writer_In_Flight_Invariants(value.In_Flight, namespace)
	aver.Always(
		int(value.Archive_Position) <= len(value.Storage.Archive),
		"ZIP Writer archive cursor stays inside caller storage.",
	)
	aver.Always(
		int(value.Central_Position) <= len(value.Storage.Central),
		"ZIP Writer central cursor stays inside caller storage.",
	)
	aver.Always(
		int(value.Content_Position) <= len(value.Storage.Content),
		"ZIP Writer content cursor stays inside caller storage.",
	)
}

// Writer_Handle keeps archive assembly in caller-owned storage.
type Writer_Handle *Writer

// Writer_Handle_Invariants composes a present Writer value.
func Writer_Handle_Invariants(value Writer_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Writer_Invariants(*value, namespace)
}

// Reader_Init loads one bounded archive through injected Stream.
func Reader_Init(
	reader Reader_Handle, stream nbio.Stream, storage Reader_Storage,
	completion nbio.Completion_Handle, callback nbio.Callback,
) {
	Reader_Handle_Invariants(reader, "Reader_Init.reader")
	Reader_Storage_Invariants(storage, "Reader_Init.storage")
	aver.Always(completion != nil, "ZIP Reader initialization has completion storage.")
	aver.Always(callback != nil, "ZIP Reader initialization has callback.")
	*reader = Reader{
		Stream: stream, Storage: storage, Callback: callback,
		Status: STATUS_INPUT_INVALID, Stage: READER_STAGE_SIZE, Active: true,
	}
	reader_stream_submit(
		unsafe.Pointer(reader), completion, nbio.STREAM_MODE_SIZE,
	)
}

// Reader_Decode decodes one validated central member into caller storage.
func Reader_Decode(
	reader Reader_Handle, index Entry_Count, destination bytes.Slice,
) (count bytes.Boundary, status Status) {
	defer func() {
		bytes.Boundary_Invariants(count, "Reader_Decode.count")
		Status_Invariants(status, "Reader_Decode.status")
	}()
	Reader_Handle_Invariants(reader, "Reader_Decode.reader")
	Entry_Count_Invariants(index, "Reader_Decode.index")
	bytes.Slice_Invariants(destination, "Reader_Decode.destination")
	if reader.Active {
		return 0, STATUS_INPUT_INVALID
	}
	if reader.Status != STATUS_OK {
		return 0, STATUS_INPUT_INVALID
	}
	if len(reader.Archive) < ARCHIVE_SIZE_MINIMUM {
		return 0, STATUS_INPUT_INVALID
	}
	selected_optional, base_offset, central_start, selected_status :=
		reader_central_entry(Archive(reader.Archive), index)
	if selected_status != Found_Status(STATUS_OK) {
		return 0, Status(selected_status)
	}
	if len(selected_optional.Name) == 0 {
		return 0, STATUS_INPUT_INVALID
	}
	selected := Central_Entry{
		Header: Central_Header(selected_optional.Header),
		Name:   Selected_Name(selected_optional.Name),
	}
	entry_count, entry_status := decode_entry(
		Selected_Archive(reader.Archive), Destination(destination), base_offset,
		central_start, selected,
	)
	return bytes.Boundary(entry_count), Status(entry_status)
}

// Reader_Header returns one validated member header borrowing Reader archive bytes.
func Reader_Header(
	reader Reader_Handle, index Entry_Count, header Header_Handle,
) (status Found_Status) {
	defer func() {
		Found_Status_Invariants(status, "Reader_Header.status")
	}()
	Reader_Handle_Invariants(reader, "Reader_Header.reader")
	Entry_Count_Invariants(index, "Reader_Header.index")
	Header_Handle_Invariants(header, "Reader_Header.header")
	if reader.Active {
		return Found_Status(STATUS_INPUT_INVALID)
	}
	if reader.Status != STATUS_OK {
		return Found_Status(STATUS_INPUT_INVALID)
	}
	if len(reader.Archive) < ARCHIVE_SIZE_MINIMUM {
		return Found_Status(STATUS_INPUT_INVALID)
	}
	return reader_header(Archive(reader.Archive), index, header)
}

// Reader_Raw returns one member compressed payload borrowing Reader archive bytes.
func Reader_Raw(
	reader Reader_Handle, index Entry_Count, header Header_Handle,
) (raw Reader_Raw_Content, status Found_Status) {
	defer func() {
		Reader_Raw_Content_Invariants(raw, "Reader_Raw.raw")
		Found_Status_Invariants(status, "Reader_Raw.status")
	}()
	Reader_Handle_Invariants(reader, "Reader_Raw.reader")
	Entry_Count_Invariants(index, "Reader_Raw.index")
	Header_Handle_Invariants(header, "Reader_Raw.header")
	if reader.Active {
		return nil, Found_Status(STATUS_INPUT_INVALID)
	}
	if reader.Status != STATUS_OK {
		return nil, Found_Status(STATUS_INPUT_INVALID)
	}
	if len(reader.Archive) < ARCHIVE_SIZE_MINIMUM {
		return nil, Found_Status(STATUS_INPUT_INVALID)
	}
	selected_optional, base_offset, central_start, selected_status :=
		reader_central_entry(Archive(reader.Archive), index)
	if selected_status != Found_Status(STATUS_OK) {
		return nil, selected_status
	}
	if len(selected_optional.Name) == 0 {
		return nil, Found_Status(STATUS_INPUT_INVALID)
	}
	selected := Central_Entry{
		Header: Central_Header(selected_optional.Header),
		Name:   Selected_Name(selected_optional.Name),
	}
	data_position, valid := local_data_position(
		Selected_Archive(reader.Archive), base_offset, central_start, selected,
	)
	if !bool(valid) {
		return nil, Found_Status(STATUS_INPUT_INVALID)
	}
	compressed_size := int(binary.Uint_32(
		binary.Bytes(selected.Header[CENTRAL_COMPRESSED_SIZE_POSITION:]),
		binary.LITTLE_ENDIAN,
	))
	if compressed_size > int(central_start)-int(data_position) {
		return nil, Found_Status(STATUS_INPUT_INVALID)
	}
	status = reader_header(Archive(reader.Archive), index, header)
	if status != Found_Status(STATUS_OK) {
		return nil, status
	}
	return Reader_Raw_Content(
		reader.Archive[data_position : int(data_position)+compressed_size],
	), Found_Status(STATUS_OK)
}

// File_System_Init binds initialized Reader and reports insecure member paths.
func File_System_Init(
	file_system File_System_Handle, reader Reader_Handle,
) (status Validation_Status) {
	defer func() {
		Validation_Status_Invariants(
			status, "File_System_Init.status",
		)
	}()
	File_System_Handle_Invariants(file_system, "File_System_Init.file_system")
	Reader_Handle_Invariants(reader, "File_System_Init.reader")
	if reader.Active {
		return Validation_Status(STATUS_INPUT_INVALID)
	}
	if reader.Status != STATUS_OK {
		return Validation_Status(STATUS_INPUT_INVALID)
	}
	if len(reader.Archive) < ARCHIVE_SIZE_MINIMUM {
		return Validation_Status(STATUS_INPUT_INVALID)
	}
	*file_system = File_System{Reader: File_System_Reader(*reader)}
	for index := Entry_Count(0); index < reader.Entry_Count; index++ {
		var header Header
		header_status := reader_header(Archive(reader.Archive), index, &header)
		if header_status != Found_Status(STATUS_OK) {
			file_system.Valid = false
			return Validation_Status(STATUS_INPUT_INVALID)
		}
		if !file_system_member_name_valid(bytes.Slice(header.Name)) {
			file_system.Valid = false
			return Validation_Status(STATUS_INPUT_INVALID)
		}
	}
	file_system.Valid = true
	return Validation_Status(STATUS_OK)
}

// File_System_Status returns exact member or synthesized directory metadata.
func File_System_Status(
	file_system File_System_Handle, path bytes.Slice, info File_Info_Handle,
) (status Found_Status) {
	defer func() { Found_Status_Invariants(status, "File_System_Status.status") }()
	File_System_Handle_Invariants(file_system, "File_System_Status.file_system")
	bytes.Slice_Invariants(path, "File_System_Status.path")
	File_Info_Handle_Invariants(info, "File_System_Status.info")
	if !file_system_ready(file_system) {
		return Found_Status(STATUS_INPUT_INVALID)
	}
	if !file_system_path_valid(path) {
		return Found_Status(STATUS_INPUT_INVALID)
	}
	if file_system_root(path) {
		*info = File_Info{
			Name: File_Info_Name(path), Mode: nbio.FILE_MODE_DIRECTORY,
			Is_Directory: true,
		}
		return Found_Status(STATUS_OK)
	}
	for index := Entry_Count(0); index < file_system.Reader.Entry_Count; index++ {
		var header Header
		header_status := reader_header(
			Archive(file_system.Reader.Archive), index, &header,
		)
		if header_status != Found_Status(STATUS_OK) {
			return header_status
		}
		name := file_system_name_without_directory_suffix(
			File_System_Member_Name(header.Name),
		)
		if bytes.Equal(bytes.Slice(name), bytes.Slice(path)) {
			mode := Header_Mode(&header)
			modified := Header_Modification_Time(&header)
			*info = File_Info{
				Name: File_Info_Name(name),
				Size: File_Info_Size(header.Uncompressed_Size),
				Mode: nbio.File_Mode(mode),
				Modified: Timestamp_Second_Precision{
					Seconds:             modified.Seconds,
					Zone_Offset_Seconds: modified.Zone_Offset_Seconds,
					Set:                 modified.Set,
				},
				Header_Index: File_Info_Header_Index(index),
				Is_Directory: File_Info_Is_Directory(
					nbio.File_Mode_Is_Directory(nbio.File_Mode(mode)),
				),
				Explicit: true,
			}
			return Found_Status(STATUS_OK)
		}
	}
	for index := Entry_Count(0); index < file_system.Reader.Entry_Count; index++ {
		var header Header
		header_status := reader_header(
			Archive(file_system.Reader.Archive), index, &header,
		)
		if header_status != Found_Status(STATUS_OK) {
			return header_status
		}
		if name_below_path(
			File_System_Member_Name(header.Name), File_System_Path(path),
		) {
			*info = File_Info{
				Name: File_Info_Name(path), Mode: nbio.FILE_MODE_DIRECTORY,
				Is_Directory: true,
			}
			return Found_Status(STATUS_OK)
		}
	}
	return Found_Status(STATUS_ENTRY_NOT_FOUND)
}

// File_System_Open decodes one exact regular member through bound Reader.
func File_System_Open(
	file_system File_System_Handle, path bytes.Slice, destination bytes.Slice,
) (count bytes.Boundary, status Status) {
	defer func() {
		bytes.Boundary_Invariants(count, "File_System_Open.count")
		Status_Invariants(status, "File_System_Open.status")
	}()
	File_System_Handle_Invariants(file_system, "File_System_Open.file_system")
	bytes.Slice_Invariants(path, "File_System_Open.path")
	bytes.Slice_Invariants(destination, "File_System_Open.destination")
	if !file_system_ready(file_system) {
		return 0, STATUS_INPUT_INVALID
	}
	if !file_system_path_valid(path) {
		return 0, STATUS_INPUT_INVALID
	}
	if file_system_root(path) {
		return 0, STATUS_INPUT_INVALID
	}
	for index := Entry_Count(0); index < file_system.Reader.Entry_Count; index++ {
		var header Header
		header_status := reader_header(
			Archive(file_system.Reader.Archive), index, &header,
		)
		if header_status != Found_Status(STATUS_OK) {
			return 0, Status(header_status)
		}
		if bytes.Equal(bytes.Slice(header.Name), bytes.Slice(path)) {
			if nbio.File_Mode_Is_Directory(nbio.File_Mode(Header_Mode(&header))) {
				return 0, STATUS_INPUT_INVALID
			}
			return Reader_Decode((*Reader)(&file_system.Reader), index, destination)
		}
	}
	return 0, STATUS_ENTRY_NOT_FOUND
}

// File_System_Read_Directory returns sorted unique immediate children.
func File_System_Read_Directory(
	file_system File_System_Handle, path bytes.Slice, entries Directory_Entries,
) (count Directory_Output_Count, status Directory_Status) {
	defer func() {
		Directory_Output_Count_Invariants(
			count, "File_System_Read_Directory.count",
		)
		Directory_Status_Invariants(
			status, "File_System_Read_Directory.status",
		)
	}()
	File_System_Handle_Invariants(file_system, "File_System_Read_Directory.file_system")
	bytes.Slice_Invariants(path, "File_System_Read_Directory.path")
	Directory_Entries_Invariants(entries, "File_System_Read_Directory.entries")
	if !file_system_ready(file_system) {
		return 0, Directory_Status(STATUS_INPUT_INVALID)
	}
	if !file_system_path_valid(path) {
		return 0, Directory_Status(STATUS_INPUT_INVALID)
	}
	var info File_Info
	info_status := File_System_Status(file_system, path, &info)
	if info_status != Found_Status(STATUS_OK) {
		return 0, Directory_Status(info_status)
	}
	if !info.Is_Directory {
		return 0, Directory_Status(STATUS_INPUT_INVALID)
	}
	entry_count := 0
	for index := Entry_Count(0); index < file_system.Reader.Entry_Count; index++ {
		var header Header
		header_status := reader_header(
			Archive(file_system.Reader.Archive), index, &header,
		)
		if header_status != Found_Status(STATUS_OK) {
			return 0, Directory_Status(header_status)
		}
		name, directory, present := immediate_child(
			File_System_Member_Name(header.Name), File_System_Parent_Path(path),
		)
		if !present {
			continue
		}
		if directory_entry_present(
			Previous_Directory_Entries(entries[:entry_count]),
			File_System_Entry_Name(name),
		) {
			continue
		}
		if entry_count == len(entries) {
			return 0, Directory_Status(STATUS_OUTPUT_TOO_SMALL)
		}
		entries[entry_count] = Directory_Entry{
			Name: bytes.Slice(name), Is_Directory: directory,
		}
		entry_count++
	}
	directory_entries_sort(Collected_Directory_Entries(entries[:entry_count]))
	return Directory_Output_Count(entry_count), Directory_Status(STATUS_OK)
}

// File_System_Walk visits root and descendants in lexical order using caller node storage.
func File_System_Walk(
	file_system File_System_Handle, root bytes.Slice, nodes File_Infos,
	callback File_System_Walk_Callback,
) (count File_System_Walk_Count, status Directory_Status) {
	defer func() {
		File_System_Walk_Count_Invariants(count, "File_System_Walk.count")
		Directory_Status_Invariants(
			status, "File_System_Walk.status",
		)
	}()
	File_System_Handle_Invariants(file_system, "File_System_Walk.file_system")
	bytes.Slice_Invariants(root, "File_System_Walk.root")
	File_Infos_Invariants(nodes, "File_System_Walk.nodes")
	aver.Always(callback != nil, "ZIP File_System walk has callback.")
	if !file_system_ready(file_system) {
		return 0, Directory_Status(STATUS_INPUT_INVALID)
	}
	if !file_system_path_valid(root) {
		return 0, Directory_Status(STATUS_INPUT_INVALID)
	}
	source := File_System_Source{
		Archive:     File_System_Archive(file_system.Reader.Archive),
		Entry_Count: File_System_Entry_Count(file_system.Reader.Entry_Count),
	}
	node_count, collect_status := file_system_collect(
		source, File_System_Path(root), nodes,
	)
	if collect_status != Directory_Status(STATUS_OK) {
		return 0, collect_status
	}
	file_info_sort(Collected_File_Infos(nodes[:node_count]))
	for index := 0; index < int(node_count); index++ {
		if !callback(nodes[index]) {
			return File_System_Walk_Count(index + 1), Directory_Status(STATUS_OK)
		}
	}
	return node_count, Directory_Status(STATUS_OK)
}

// Writer_Init binds bounded assembly storage and injected output Stream.
func Writer_Init(
	writer Writer_Handle, stream nbio.Stream, storage Writer_Storage,
) (status Capacity_Status) {
	defer func() {
		Capacity_Status_Invariants(status, "Writer_Init.status")
	}()
	Writer_Handle_Invariants(writer, "Writer_Init.writer")
	Writer_Storage_Invariants(storage, "Writer_Init.storage")
	if !writer_storage_valid(storage) {
		return Capacity_Status(STATUS_OUTPUT_TOO_SMALL)
	}
	*writer = Writer{Stream: stream, Storage: storage, Status: STATUS_OK}
	return Capacity_Status(STATUS_OK)
}

// Writer_Create starts one logical member after finalizing previous content.
func Writer_Create(
	writer Writer_Handle, header Header_Unvalidated_Handle,
) (status Creation_Status) {
	defer func() {
		Creation_Status_Invariants(status, "Writer_Create.status")
	}()
	Writer_Handle_Invariants(writer, "Writer_Create.writer")
	Header_Unvalidated_Handle_Invariants(header, "Writer_Create.header")
	return writer_create(writer, header, false)
}

// Writer_Create_Raw starts one precompressed member with caller checksum and sizes.
func Writer_Create_Raw(
	writer Writer_Handle, header Header_Unvalidated_Handle,
) (status Creation_Status) {
	defer func() {
		Creation_Status_Invariants(
			status, "Writer_Create_Raw.status",
		)
	}()
	Writer_Handle_Invariants(writer, "Writer_Create_Raw.writer")
	Header_Unvalidated_Handle_Invariants(header, "Writer_Create_Raw.header")
	return writer_create(writer, header, true)
}

// Writer_Copy copies one validated member without decompressing payload.
func Writer_Copy(writer Writer_Handle, reader Reader_Handle, index Entry_Count) (status Status) {
	defer func() { Status_Invariants(status, "Writer_Copy.status") }()
	Writer_Handle_Invariants(writer, "Writer_Copy.writer")
	Reader_Handle_Invariants(reader, "Writer_Copy.reader")
	Entry_Count_Invariants(index, "Writer_Copy.index")
	var header Header
	raw, raw_status := Reader_Raw(reader, index, &header)
	if raw_status != Found_Status(STATUS_OK) {
		return Status(raw_status)
	}
	unvalidated := Header_Unvalidated{
		Name:     Header_Name_Unvalidated(header.Name),
		Comment:  Header_Comment_Unvalidated(header.Comment),
		Extra:    Header_Extra_Unvalidated(header.Extra),
		Method:   header.Method,
		Flags:    header.Flags,
		Checksum: header.Checksum,
		Compressed_Size: Header_Compressed_Size_Unvalidated(
			header.Compressed_Size,
		),
		Uncompressed_Size: Header_Uncompressed_Size_Unvalidated(
			header.Uncompressed_Size,
		),
		External_Attributes: header.External_Attributes,
		Creator_Version:     header.Creator_Version,
		Extractor_Version:   header.Extractor_Version,
		Modified_Time:       header.Modified_Time,
		Modified_Date:       header.Modified_Date,
		Internal_Attributes: header.Internal_Attributes,
		Modified:            header.Modified,
		Non_UTF8:            header.Non_UTF8,
	}
	create_status := Writer_Create_Raw(writer, &unvalidated)
	if create_status != Creation_Status(STATUS_OK) {
		return Status(create_status)
	}
	count, write_status := Writer_Write(writer, bytes.Slice(raw))
	if write_status != Bounded_Status(STATUS_OK) {
		return Status(write_status)
	}
	if int(count) != len(raw) {
		return STATUS_INPUT_INVALID
	}
	return STATUS_OK
}

// Writer_Add_File_System writes sorted regular files and synthesized directories.
func Writer_Add_File_System(
	writer Writer_Handle, file_system File_System_Handle, nodes File_Infos,
) (status Creation_Status) {
	defer func() { Creation_Status_Invariants(status, "Writer_Add_File_System.status") }()
	Writer_Handle_Invariants(writer, "Writer_Add_File_System.writer")
	File_System_Handle_Invariants(file_system, "Writer_Add_File_System.file_system")
	File_Infos_Invariants(nodes, "Writer_Add_File_System.nodes")
	if !file_system_ready(file_system) {
		return Creation_Status(STATUS_INPUT_INVALID)
	}
	if len(writer.Storage.Name) == 0 {
		return Creation_Status(STATUS_OUTPUT_TOO_SMALL)
	}
	writer.Storage.Name[0] = '.'
	node_count, collect_status := file_system_collect(File_System_Source{
		Archive:     File_System_Archive(file_system.Reader.Archive),
		Entry_Count: File_System_Entry_Count(file_system.Reader.Entry_Count),
	}, File_System_Path(writer.Storage.Name[:1]), nodes)
	if collect_status != Directory_Status(STATUS_OK) {
		return Creation_Status(collect_status)
	}
	file_info_sort(Collected_File_Infos(nodes[:node_count]))
	for index := 0; index < int(node_count); index++ {
		info := nodes[index]
		if file_system_root(bytes.Slice(info.Name)) {
			continue
		}
		if !nbio.File_Mode_Is_Regular(info.Mode) {
			if !info.Is_Directory {
				return Creation_Status(STATUS_INPUT_INVALID)
			}
		}
		header := Header_Unvalidated{
			Name: Header_Name_Unvalidated(info.Name), Method: METHOD_DEFLATE,
			Modified: Timestamp{
				Seconds: info.Modified.Seconds, Set: info.Modified.Set,
				Zone_Offset_Seconds: info.Modified.Zone_Offset_Seconds,
			},
		}
		Header_Set_Mode(&header, info.Mode)
		if info.Is_Directory {
			if len(info.Name) >= len(writer.Storage.Name) {
				return Creation_Status(STATUS_OUTPUT_TOO_SMALL)
			}
			name_count := copy(writer.Storage.Name, info.Name)
			writer.Storage.Name[name_count] = '/'
			header.Name = Header_Name_Unvalidated(writer.Storage.Name[:name_count+1])
			header.Method = METHOD_STORE
		}
		create_status := writer_create(writer, &header, false)
		if create_status != Creation_Status(STATUS_OK) {
			return create_status
		}
		if info.Is_Directory {
			continue
		}
		content := bytes.Slice(writer.Storage.Content)
		count, open_status := File_System_Open(file_system, bytes.Slice(info.Name), content)
		if open_status != STATUS_OK {
			return Creation_Status(open_status)
		}
		written, write_status := Writer_Write(writer, content[:count])
		if write_status != Bounded_Status(STATUS_OK) {
			return Creation_Status(write_status)
		}
		if written != count {
			return Creation_Status(STATUS_INPUT_INVALID)
		}
	}
	return Creation_Status(STATUS_OK)
}

// Writer_Set_Comment copies bounded EOCD comment into caller storage.
func Writer_Set_Comment(
	writer Writer_Handle, comment bytes.Slice,
) (status Bounded_Status) {
	defer func() {
		Bounded_Status_Invariants(status, "Writer_Set_Comment.status")
	}()
	Writer_Handle_Invariants(writer, "Writer_Set_Comment.writer")
	bytes.Slice_Invariants(comment, "Writer_Set_Comment.comment")
	if writer.Closed {
		return Bounded_Status(STATUS_INPUT_INVALID)
	}
	if writer.In_Flight {
		return Bounded_Status(STATUS_INPUT_INVALID)
	}
	if len(comment) > len(writer.Storage.Comment) {
		return Bounded_Status(STATUS_OUTPUT_TOO_SMALL)
	}
	copy(writer.Storage.Comment, comment)
	writer.Comment_Count = Writer_Comment_Count(len(comment))
	return Bounded_Status(STATUS_OK)
}

// Writer_Set_Offset starts ZIP segment after caller-owned Stream prefix.
func Writer_Set_Offset(
	writer Writer_Handle, offset bytes.Boundary,
) (status Validation_Status) {
	defer func() {
		Validation_Status_Invariants(
			status, "Writer_Set_Offset.status",
		)
	}()
	Writer_Handle_Invariants(writer, "Writer_Set_Offset.writer")
	bytes.Boundary_Invariants(offset, "Writer_Set_Offset.offset")
	if writer.Closed {
		return Validation_Status(STATUS_INPUT_INVALID)
	}
	if writer.In_Flight {
		return Validation_Status(STATUS_INPUT_INVALID)
	}
	if writer.Archive_Position != 0 {
		return Validation_Status(STATUS_INPUT_INVALID)
	}
	writer.Offset = Writer_Offset(offset)
	return Validation_Status(STATUS_OK)
}

// Writer_Write retains current payload until bounded compression and record finalization.
func Writer_Write(
	writer Writer_Handle, source bytes.Slice,
) (count bytes.Boundary, status Bounded_Status) {
	defer func() {
		bytes.Boundary_Invariants(count, "Writer_Write.count")
		Bounded_Status_Invariants(status, "Writer_Write.status")
	}()
	Writer_Handle_Invariants(writer, "Writer_Write.writer")
	bytes.Slice_Invariants(source, "Writer_Write.source")
	if writer.Closed {
		return 0, Bounded_Status(STATUS_INPUT_INVALID)
	}
	if writer.In_Flight {
		return 0, Bounded_Status(STATUS_INPUT_INVALID)
	}
	if !writer.Active {
		return 0, Bounded_Status(STATUS_INPUT_INVALID)
	}
	if writer.Directory {
		if len(source) != 0 {
			return 0, Bounded_Status(STATUS_INPUT_INVALID)
		}
	}
	position := int(writer.Content_Position)
	if len(source) > len(writer.Storage.Content)-position {
		return 0, Bounded_Status(STATUS_OUTPUT_TOO_SMALL)
	}
	copy(writer.Storage.Content[position:], source)
	writer.Content_Position += Writer_Content_Position(len(source))
	return bytes.Boundary(len(source)), Bounded_Status(STATUS_OK)
}

// Writer_Close emits central directory, then submits complete archive through Stream.
func Writer_Close(
	writer Writer_Handle, completion nbio.Completion_Handle, callback nbio.Callback,
) {
	Writer_Handle_Invariants(writer, "Writer_Close.writer")
	aver.Always(completion != nil, "ZIP Writer close has completion storage.")
	aver.Always(callback != nil, "ZIP Writer close has callback.")
	writer.Status = Status(writer_finalize(writer))
	if writer.Status == STATUS_OK {
		directory := Writer_Directory_State{
			Archive: writer.Storage.Archive, Central: writer.Storage.Central,
			Comment:          writer.Storage.Comment,
			Archive_Position: writer.Archive_Position,
			Central_Position: writer.Central_Position,
			Entry_Count:      writer.Entry_Count, Comment_Count: writer.Comment_Count,
		}
		writer.Status = Status(writer_directory_end(&directory))
		writer.Archive_Position = directory.Archive_Position
		writer.Count = Writer_Count(directory.Archive_Position)
	}
	if writer.Status != STATUS_OK {
		writer_callback(completion, callback)
		return
	}
	writer.Closed = true
	writer.In_Flight = true
	writer.Callback = callback
	writer_stream_submit(unsafe.Pointer(writer), completion)
}

// Writer_Flush finalizes current member and submits current local records.
func Writer_Flush(
	writer Writer_Handle, completion nbio.Completion_Handle, callback nbio.Callback,
) {
	Writer_Handle_Invariants(writer, "Writer_Flush.writer")
	aver.Always(completion != nil, "ZIP Writer flush has completion storage.")
	aver.Always(callback != nil, "ZIP Writer flush has callback.")
	writer.Status = Status(writer_finalize(writer))
	if writer.Status != STATUS_OK {
		writer_callback(completion, callback)
		return
	}
	writer.Count = Writer_Count(writer.Archive_Position)
	writer.In_Flight = true
	writer.Callback = callback
	writer_stream_submit(unsafe.Pointer(writer), completion)
}

func reader_stream_submit(
	state unsafe.Pointer, completion nbio.Completion_Handle, mode nbio.Stream_Mode,
) {
	reader := (*Reader)(state)
	reader.Transfer_Buffer = nil
	if mode == nbio.STREAM_MODE_READ_AT {
		reader.Transfer_Buffer = Reader_Transfer_Buffer(reader.Archive)
	}
	reader.Stream_Mode = mode
	reader.Stream_Offset = 0
	if reader.Submission_Active {
		reader.Continue = true
		return
	}
	reader.Submission_Active = true
	reader.Continue = true
	for reader.Continue {
		reader.Continue = false
		callback := nbio.Stream_Callback{
			State: reader, Data: int(reader.Stage),
			Procedure: reader_stream_callback,
		}
		if reader.Stream.Procedure == nil {
			completion.Data = 0
			completion.Error = nbio.Stream_Empty
			nbio.Stream_Callback_Call(callback, completion)
		} else {
			reader.Stream.Procedure(
				reader.Stream.State, completion, reader.Stream_Mode,
				reader.Transfer_Buffer, int64(reader.Stream_Offset),
				nbio.SEEK_FROM_START, callback,
			)
		}
	}
	reader.Submission_Active = false
}

func reader_stream_callback(
	state_value nbio.State, data nbio.Stream_Callback_Data, callback nbio.Callback,
	completion nbio.Completion_Handle,
) {
	reader_stream_complete(
		unsafe.Pointer(state_value.(*Reader)), Reader_Stream_Stage(data),
		callback, completion,
	)
}

func reader_stream_complete(
	state unsafe.Pointer, stage Reader_Stream_Stage, _ nbio.Callback,
	completion nbio.Completion_Handle,
) {
	Reader_Stream_Stage_Invariants(stage, "reader_stream_complete.stage")
	reader := (*Reader)(state)
	if Reader_Stage(stage) != reader.Stage {
		reader.Status = STATUS_INPUT_INVALID
		reader_finish(state, completion)
		return
	}
	if Reader_Stage(stage) == READER_STAGE_SIZE {
		reader_size_complete(state, completion)
		return
	}
	if Reader_Stage(stage) == READER_STAGE_READ {
		reader_read_complete(state, completion)
		return
	}
	reader.Status = STATUS_INPUT_INVALID
	reader_finish(state, completion)
}

func reader_size_complete(state unsafe.Pointer, completion nbio.Completion_Handle) {
	reader := (*Reader)(state)
	if completion.Error != nil {
		reader_finish(state, completion)
		return
	}
	size := completion.Data
	if size < ARCHIVE_SIZE_MINIMUM {
		reader.Status = STATUS_INPUT_INVALID
		reader_finish(state, completion)
		return
	}
	if size > ARCHIVE_SIZE_MAXIMUM {
		reader.Status = STATUS_INPUT_INVALID
		reader_finish(state, completion)
		return
	}
	if size > len(reader.Storage.Archive) {
		reader.Status = STATUS_OUTPUT_TOO_SMALL
		reader_finish(state, completion)
		return
	}
	reader.Archive = Reader_Archive(reader.Storage.Archive[:size])
	reader.Stage = READER_STAGE_READ
	reader_stream_submit(
		state, completion, nbio.STREAM_MODE_READ_AT,
	)
}

func reader_read_complete(state unsafe.Pointer, completion nbio.Completion_Handle) {
	reader := (*Reader)(state)
	if completion.Error != nil {
		reader_finish(state, completion)
		return
	}
	if completion.Data != len(reader.Archive) {
		completion.Error = nbio.Stream_Unexpected_EOF
		reader_finish(state, completion)
		return
	}
	end_position, entry_count, central_size, central_offset, found :=
		directory_end(Archive(reader.Archive))
	if !bool(found) {
		reader.Status = STATUS_INPUT_INVALID
		reader_finish(state, completion)
		return
	}
	if !reader_directory_valid(
		Archive(reader.Archive), entry_count, central_size, central_offset,
	) {
		reader.Status = STATUS_INPUT_INVALID
		reader_finish(state, completion)
		return
	}
	reader.Entry_Count = entry_count
	comment_size := int(binary.Uint_16(
		binary.Bytes(
			reader.Archive[int(end_position)+DIRECTORY_END_COMMENT_SIZE_POSITION:],
		),
		binary.LITTLE_ENDIAN,
	))
	comment_start := int(end_position) + WRITER_DIRECTORY_END_SIZE
	reader.Comment = Reader_Comment(
		reader.Archive[comment_start : comment_start+comment_size],
	)
	reader.Status = STATUS_OK
	reader_finish(state, completion)
}

func reader_finish(state unsafe.Pointer, completion nbio.Completion_Handle) {
	reader := (*Reader)(state)
	callback := reader.Callback
	reader.Callback = nil
	reader.Stage = READER_STAGE_NONE
	reader.Active = false
	callback(completion)
}

func reader_directory_valid(
	archive Archive, entry_count Entry_Count, central_size Central_Size,
	central_offset Central_Offset,
) (valid binary.Boolean) {
	defer func() {
		binary.Boolean_Invariants(valid, "reader_directory_valid.valid")
	}()
	Archive_Invariants(archive, "reader_directory_valid.archive")
	Entry_Count_Invariants(entry_count, "reader_directory_valid.entry_count")
	Central_Size_Invariants(central_size, "reader_directory_valid.central_size")
	Central_Offset_Invariants(
		central_offset, "reader_directory_valid.central_offset",
	)
	end_position, _, _, _, found := directory_end(Archive(archive))
	if !bool(found) {
		return false
	}
	if int(central_size) > int(end_position) {
		return false
	}
	central_start := Central_Position(int(end_position) - int(central_size))
	if int(central_offset) > int(central_start) {
		return false
	}
	position := central_start
	for entry_index := Entry_Count(0); entry_index < entry_count; entry_index++ {
		entry, next, entry_valid := read_central_entry(
			Central_Archive(archive), position, Central_End(end_position),
		)
		if !bool(entry_valid) {
			return false
		}
		if len(entry.Header) == 0 {
			return false
		}
		position = Central_Position(next)
	}
	return int(position) == int(end_position)
}

func reader_central_entry(
	archive Archive, selected_index Entry_Count,
) (
	selected Central_Entry_Optional, base_offset Base_Offset,
	central_start Selected_Central_Start, status Found_Status,
) {
	defer func() {
		Central_Entry_Optional_Invariants(
			selected, "reader_central_entry.selected",
		)
		Base_Offset_Invariants(base_offset, "reader_central_entry.base_offset")
		Selected_Central_Start_Invariants(
			central_start, "reader_central_entry.central_start",
		)
		Found_Status_Invariants(
			status, "reader_central_entry.status",
		)
	}()
	Archive_Invariants(archive, "reader_central_entry.archive")
	Entry_Count_Invariants(selected_index, "reader_central_entry.selected_index")
	end_position, entry_count, central_size, central_offset, found :=
		directory_end(Archive(archive))
	if !bool(found) {
		return Central_Entry_Optional{}, 0, 0, Found_Status(STATUS_ENTRY_NOT_FOUND)
	}
	if selected_index >= entry_count {
		return Central_Entry_Optional{}, 0, 0, Found_Status(STATUS_ENTRY_NOT_FOUND)
	}
	start := Central_Position(int(end_position) - int(central_size))
	base_offset = Base_Offset(int(start) - int(central_offset))
	position := start
	for entry_index := Entry_Count(0); entry_index < entry_count; entry_index++ {
		entry, next, valid := read_central_entry(
			Central_Archive(archive), position, Central_End(end_position),
		)
		if !bool(valid) {
			return Central_Entry_Optional{}, 0, 0, Found_Status(STATUS_INPUT_INVALID)
		}
		if entry_index == selected_index {
			return Central_Entry_Optional{
				Header: Central_Header_Optional(entry.Header),
				Name:   Member_Name(entry.Name),
			}, base_offset, Selected_Central_Start(start), Found_Status(STATUS_OK)
		}
		position = Central_Position(next)
	}
	return Central_Entry_Optional{}, 0, 0, Found_Status(STATUS_ENTRY_NOT_FOUND)
}

func reader_header(
	archive Archive, selected_index Entry_Count, header Header_Handle,
) (status Found_Status) {
	defer func() {
		Found_Status_Invariants(status, "reader_header.status")
	}()
	Archive_Invariants(archive, "reader_header.archive")
	Entry_Count_Invariants(selected_index, "reader_header.selected_index")
	Header_Handle_Invariants(header, "reader_header.header")
	end_position, entry_count, central_size, _, found :=
		directory_end(Archive(archive))
	if !bool(found) {
		return Found_Status(STATUS_ENTRY_NOT_FOUND)
	}
	if selected_index >= entry_count {
		return Found_Status(STATUS_ENTRY_NOT_FOUND)
	}
	position := Central_Position(int(end_position) - int(central_size))
	for entry_index := Entry_Count(0); entry_index < entry_count; entry_index++ {
		entry, next, valid := read_central_entry(
			Central_Archive(archive), position, Central_End(end_position),
		)
		if !bool(valid) {
			return Found_Status(STATUS_INPUT_INVALID)
		}
		if entry_index == selected_index {
			reader_header_value(
				Central_Archive(archive),
				Central_Entry_Parsed{
					Header: Central_Header(entry.Header),
					Name:   Member_Name(entry.Name),
				},
				position, header,
			)
			return Found_Status(STATUS_OK)
		}
		position = Central_Position(next)
	}
	return Found_Status(STATUS_ENTRY_NOT_FOUND)
}

func reader_header_value(
	archive Central_Archive, entry Central_Entry_Parsed, position Central_Position,
	header Header_Handle,
) {
	Central_Archive_Invariants(archive, "reader_header_value.archive")
	Central_Entry_Parsed_Invariants(entry, "reader_header_value.entry")
	Central_Position_Invariants(position, "reader_header_value.position")
	Header_Handle_Invariants(header, "reader_header_value.header")
	fixed := binary.Bytes(entry.Header)
	order := binary.LITTLE_ENDIAN
	extra_size := int(binary.Uint_16(fixed[WRITER_CENTRAL_EXTRA_SIZE_POSITION:], order))
	comment_size := int(binary.Uint_16(fixed[WRITER_CENTRAL_COMMENT_SIZE_POSITION:], order))
	tail_start := int(position) + CENTRAL_HEADER_SIZE
	extra_start := tail_start + len(entry.Name)
	comment_start := extra_start + extra_size
	extra := archive[extra_start:comment_start]
	modified_time := binary.Uint_16(fixed[CENTRAL_MODIFIED_TIME_POSITION:], order)
	modified_date := binary.Uint_16(fixed[CENTRAL_MODIFIED_DATE_POSITION:], order)
	*header = Header{
		Name: Header_Name(entry.Name), Extra: Header_Extra(extra),
		Comment: Header_Comment(
			archive[comment_start : comment_start+comment_size],
		),
		Method: Header_Method(binary.Uint_16(fixed[CENTRAL_METHOD_POSITION:], order)),
		Flags:  Header_Flags(binary.Uint_16(fixed[CENTRAL_FLAGS_POSITION:], order)),
		Checksum: Header_Checksum(
			binary.Uint_32(fixed[CENTRAL_CHECKSUM_POSITION:], order),
		),
		Compressed_Size: Header_Compressed_Size(
			binary.Uint_32(fixed[CENTRAL_COMPRESSED_SIZE_POSITION:], order),
		),
		Uncompressed_Size: Header_Uncompressed_Size(
			binary.Uint_32(fixed[CENTRAL_UNCOMPRESSED_SIZE_POSITION:], order),
		),
		External_Attributes: Header_External_Attributes(
			binary.Uint_32(fixed[WRITER_CENTRAL_EXTERNAL_ATTRIBUTES_POSITION:], order),
		),
		Creator_Version: Header_Creator_Version(
			binary.Uint_16(fixed[WRITER_CENTRAL_CREATOR_POSITION:], order),
		),
		Extractor_Version: Header_Extractor_Version(
			binary.Uint_16(fixed[WRITER_CENTRAL_EXTRACTOR_POSITION:], order),
		),
		Modified_Time: Header_Modified_Time(modified_time),
		Modified_Date: Header_Modified_Date(modified_date),
		Internal_Attributes: Header_Internal_Attributes(
			binary.Uint_16(fixed[CENTRAL_INTERNAL_ATTRIBUTES_POSITION:], order),
		),
		Non_UTF8: Header_Non_UTF8(
			binary.Uint_16(fixed[CENTRAL_FLAGS_POSITION:], order)&FLAG_UTF8 == 0,
		),
	}
	wire_modified := header_modified(
		Header_Extra(extra), modified_date, modified_time,
	)
	header.Modified = Timestamp{}
	if wire_modified.Second_Offset != 0 {
		header.Modified = Timestamp{
			Seconds: Timestamp_Seconds(
				wire_modified.Second_Offset - WIRE_TIMESTAMP_PRESENT_OFFSET,
			),
			Zone_Offset_Seconds: time.Zone_Offset_Seconds(
				int64(wire_modified.Zone_Quarter_Hours) *
					TIMESTAMP_ZONE_QUARTER_HOUR_SECONDS,
			),
			Set: true,
		}
	}
}

func file_system_ready(file_system File_System_Handle) (ready binary.Boolean) {
	defer func() { binary.Boolean_Invariants(ready, "file_system_ready.ready") }()
	File_System_Handle_Invariants(file_system, "file_system_ready.file_system")
	if !file_system.Valid {
		return false
	}
	if len(file_system.Reader.Archive) < ARCHIVE_SIZE_MINIMUM {
		return false
	}
	if file_system.Reader.Entry_Count > FILE_SYSTEM_ENTRY_COUNT_MAXIMUM {
		return false
	}
	if file_system.Reader.Active {
		return false
	}
	return file_system.Reader.Status == STATUS_OK
}

func file_system_root(path bytes.Slice) (root binary.Boolean) {
	defer func() { binary.Boolean_Invariants(root, "file_system_root.root") }()
	bytes.Slice_Invariants(path, "file_system_root.path")
	if len(path) != len(".") {
		return false
	}
	return path[0] == '.'
}

func file_system_path_valid(path bytes.Slice) (valid binary.Boolean) {
	defer func() {
		binary.Boolean_Invariants(valid, "file_system_path_valid.valid")
	}()
	bytes.Slice_Invariants(path, "file_system_path_valid.path")
	if file_system_root(path) {
		return true
	}
	if !file_system_member_name_valid(path) {
		return false
	}
	return path[len(path)-1] != '/'
}

func file_system_member_name_valid(name bytes.Slice) (valid binary.Boolean) {
	defer func() {
		binary.Boolean_Invariants(valid, "file_system_member_name_valid.valid")
	}()
	bytes.Slice_Invariants(name, "file_system_member_name_valid.name")
	if len(name) == 0 {
		return false
	}
	if len(name) > HEADER_NAME_SIZE_MAXIMUM {
		return false
	}
	if name[0] == '/' {
		return false
	}
	component_start := 0
	for position, character := range name {
		if character == 0 {
			return false
		}
		if character == '\\' {
			return false
		}
		if character != '/' {
			continue
		}
		if position == component_start {
			return false
		}
		if path_component_special(
			File_System_Component(name[component_start:position]),
		) {
			return false
		}
		component_start = position + 1
	}
	return !path_component_special(File_System_Component(name[component_start:]))
}

func path_component_special(
	component File_System_Component,
) (special binary.Boolean) {
	defer func() {
		binary.Boolean_Invariants(special, "path_component_special.special")
	}()
	File_System_Component_Invariants(
		component, "path_component_special.component",
	)
	if len(component) == 1 {
		return component[0] == '.'
	}
	return len(component) == len("..") &&
		component[POSITION_MINIMUM] == '.' &&
		component[SELECTED_NAME_SIZE_MINIMUM] == '.'
}

func file_system_name_without_directory_suffix(
	name File_System_Member_Name,
) (result File_System_Normalized_Name) {
	defer func() {
		File_System_Normalized_Name_Invariants(
			result, "file_system_name_without_directory_suffix.result",
		)
	}()
	File_System_Member_Name_Invariants(
		name, "file_system_name_without_directory_suffix.name",
	)
	if len(name) != 0 {
		if name[len(name)-1] == '/' {
			return File_System_Normalized_Name(name[:len(name)-1])
		}
	}
	return File_System_Normalized_Name(name)
}

func name_below_path(
	name File_System_Member_Name, path File_System_Path,
) (below binary.Boolean) {
	defer func() { binary.Boolean_Invariants(below, "name_below_path.below") }()
	File_System_Member_Name_Invariants(name, "name_below_path.name")
	File_System_Path_Invariants(path, "name_below_path.path")
	if len(name) <= len(path) {
		return false
	}
	if name[len(path)] != '/' {
		return false
	}
	return binary.Boolean(bytes.Equal(bytes.Slice(name[:len(path)]), bytes.Slice(path)))
}

func immediate_child(
	name File_System_Member_Name, path File_System_Parent_Path,
) (
	child File_System_Child_Name, directory binary.Boolean,
	present binary.Boolean,
) {
	defer func() {
		File_System_Child_Name_Invariants(child, "immediate_child.child")
		binary.Boolean_Invariants(directory, "immediate_child.directory")
		binary.Boolean_Invariants(present, "immediate_child.present")
	}()
	File_System_Member_Name_Invariants(name, "immediate_child.name")
	File_System_Parent_Path_Invariants(path, "immediate_child.path")
	start := 0
	if !file_system_root(bytes.Slice(path)) {
		if !name_below_path(name, File_System_Path(path)) {
			return nil, false, false
		}
		start = len(path) + 1
	}
	if start >= len(name) {
		return nil, false, false
	}
	for position := start; position < len(name); position++ {
		if name[position] == '/' {
			if position == start {
				return nil, false, false
			}
			return File_System_Child_Name(name[start:position]), true, true
		}
	}
	return File_System_Child_Name(name[start:]), false, true
}

func directory_entry_present(
	entries Previous_Directory_Entries, name File_System_Entry_Name,
) (present binary.Boolean) {
	defer func() {
		binary.Boolean_Invariants(present, "directory_entry_present.present")
	}()
	Previous_Directory_Entries_Invariants(
		entries, "directory_entry_present.entries",
	)
	File_System_Entry_Name_Invariants(name, "directory_entry_present.name")
	for _, entry := range entries {
		if bytes.Equal(bytes.Slice(entry.Name), bytes.Slice(name)) {
			return true
		}
	}
	return false
}

func directory_entries_sort(entries Collected_Directory_Entries) {
	Collected_Directory_Entries_Invariants(
		entries, "directory_entries_sort.entries",
	)
	slices.Sort_Function(entries, func(
		first Directory_Entry, second Directory_Entry,
	) (comparison slices.Comparison) {
		return slices.Comparison(bytes.Compare(
			bytes.Slice(first.Name), bytes.Slice(second.Name),
		))
	})
}

func file_system_collect(source File_System_Source, root File_System_Path, nodes File_Infos,
) (count File_System_Walk_Count, status Directory_Status) {
	defer func() {
		File_System_Walk_Count_Invariants(count, "file_system_collect.count")
		Directory_Status_Invariants(status, "file_system_collect.status")
	}()
	File_System_Source_Invariants(source, "file_system_collect.source")
	File_System_Path_Invariants(root, "file_system_collect.root")
	File_Infos_Invariants(nodes, "file_system_collect.nodes")
	reader := Reader{Storage: Reader_Storage{
		Archive: Reader_Storage_Archive(source.Archive)},
		Archive: Reader_Archive(source.Archive), Status: STATUS_OK,
		Entry_Count: Entry_Count(source.Entry_Count),
	}
	file_system := File_System{Reader: File_System_Reader(reader), Valid: true}
	var root_info File_Info
	root_status := File_System_Status(&file_system, bytes.Slice(root), &root_info)
	if root_status != Found_Status(STATUS_OK) {
		return 0, Directory_Status(root_status)
	}
	if len(nodes) == 0 {
		return 0, Directory_Status(STATUS_OUTPUT_TOO_SMALL)
	}
	nodes[POSITION_MINIMUM], count = root_info, File_System_Walk_Count(len("."))
	if !root_info.Is_Directory {
		return count, Directory_Status(STATUS_OK)
	}
	for index := Entry_Count(0); index < Entry_Count(source.Entry_Count); index++ {
		var header Header
		header_status := reader_header(Archive(source.Archive), index, &header)
		if header_status != Found_Status(STATUS_OK) {
			return 0, Directory_Status(header_status)
		}
		member := File_System_Member_Name(header.Name)
		name := file_system_name_without_directory_suffix(member)
		for position, character := range name {
			if character != '/' {
				continue
			}
			candidate_name := File_Info_Addition_Name(name[:position])
			candidate := File_Info_Addition{Name: candidate_name, Is_Directory: true}
			candidate.Mode = Archive_File_Mode(nbio.FILE_MODE_DIRECTORY)
			result, add_status := file_info_add(
				File_Info_Addition_Storage(nodes), File_Info_Previous_Count(count),
				File_System_Parent_Path(root), candidate,
			)
			count = File_System_Walk_Count(result)
			if add_status != Capacity_Status(STATUS_OK) {
				return 0, Directory_Status(add_status)
			}
		}
		modified := Header_Modification_Time(&header)
		zone_seconds := int64(modified.Zone_Offset_Seconds)
		zone_quarters := zone_seconds / TIMESTAMP_ZONE_QUARTER_HOUR_SECONDS
		zone := Timestamp_Zone_Quarter_Hours(zone_quarters)
		candidate := File_Info_Addition(file_info_from_header(
			File_Info_Addition_Name(name), File_Info_Size(header.Uncompressed_Size),
			Header_Mode(&header), modified.Seconds,
			zone, modified.Set, File_Info_Header_Index(index),
		))
		result, add_status := file_info_add(
			File_Info_Addition_Storage(nodes), File_Info_Previous_Count(count),
			File_System_Parent_Path(root), candidate,
		)
		count = File_System_Walk_Count(result)
		if add_status != Capacity_Status(STATUS_OK) {
			return 0, Directory_Status(add_status)
		}
	}
	return count, Directory_Status(STATUS_OK)
}

func file_info_from_header(
	name File_Info_Addition_Name, size File_Info_Size, mode Archive_File_Mode,
	seconds Timestamp_Seconds, zone Timestamp_Zone_Quarter_Hours, set Timestamp_Set,
	index File_Info_Header_Index,
) (candidate File_Info_Explicit_Addition) {
	defer func() {
		File_Info_Explicit_Addition_Invariants(
			candidate, "file_info_from_header.candidate",
		)
	}()
	File_Info_Addition_Name_Invariants(name, "file_info_from_header.name")
	File_Info_Size_Invariants(size, "file_info_from_header.size")
	Archive_File_Mode_Invariants(mode, "file_info_from_header.mode")
	Timestamp_Seconds_Invariants(seconds, "file_info_from_header.seconds")
	Timestamp_Zone_Quarter_Hours_Invariants(zone, "file_info_from_header.zone")
	Timestamp_Set_Invariants(set, "file_info_from_header.set")
	File_Info_Header_Index_Invariants(index, "file_info_from_header.index")
	wire_modified := Wire_Timestamp{}
	if set {
		wire_modified = Wire_Timestamp{
			Second_Offset: Wire_Timestamp_Second_Offset(
				uint64(seconds) + WIRE_TIMESTAMP_PRESENT_OFFSET,
			),
			Zone_Quarter_Hours: zone,
		}
	}
	return File_Info_Explicit_Addition{
		Name: name, Size: size, Mode: mode,
		Modified:     wire_modified,
		Header_Index: index,
		Is_Directory: File_Info_Is_Directory(
			nbio.File_Mode_Is_Directory(nbio.File_Mode(mode)),
		),
		Explicit: true,
	}
}

func file_info_add(
	nodes File_Info_Addition_Storage, count File_Info_Previous_Count,
	root File_System_Parent_Path, candidate File_Info_Addition,
) (result File_Info_Result_Count, status Capacity_Status) {
	defer func() {
		File_Info_Result_Count_Invariants(result, "file_info_add.result")
		Capacity_Status_Invariants(status, "file_info_add.status")
	}()
	File_Info_Addition_Storage_Invariants(nodes, "file_info_add.nodes")
	File_Info_Previous_Count_Invariants(count, "file_info_add.count")
	File_System_Parent_Path_Invariants(root, "file_info_add.root")
	File_Info_Addition_Invariants(candidate, "file_info_add.candidate")
	modified := Timestamp_Second_Precision{}
	if candidate.Modified.Second_Offset != 0 {
		modified = Timestamp_Second_Precision{
			Seconds: Timestamp_Seconds(
				candidate.Modified.Second_Offset - WIRE_TIMESTAMP_PRESENT_OFFSET,
			),
			Zone_Offset_Seconds: time.Zone_Offset_Seconds(
				int64(candidate.Modified.Zone_Quarter_Hours) *
					TIMESTAMP_ZONE_QUARTER_HOUR_SECONDS,
			),
			Set: true,
		}
	}
	file_info := File_Info{
		Name: File_Info_Name(candidate.Name), Size: candidate.Size,
		Mode: nbio.File_Mode(candidate.Mode), Modified: modified,
		Header_Index: candidate.Header_Index,
		Is_Directory: candidate.Is_Directory, Explicit: candidate.Explicit,
	}
	if !path_at_or_below(candidate.Name, root) {
		return File_Info_Result_Count(count), Capacity_Status(STATUS_OK)
	}
	for index := 0; index < int(count); index++ {
		if !bytes.Equal(bytes.Slice(nodes[index].Name), bytes.Slice(candidate.Name)) {
			continue
		}
		if candidate.Explicit {
			nodes[index] = file_info
		}
		return File_Info_Result_Count(count), Capacity_Status(STATUS_OK)
	}
	if int(count) == len(nodes) {
		return File_Info_Result_Count(count), Capacity_Status(STATUS_OUTPUT_TOO_SMALL)
	}
	nodes[count] = file_info
	return File_Info_Result_Count(count + 1), Capacity_Status(STATUS_OK)
}

func path_at_or_below(
	path File_Info_Addition_Name, root File_System_Parent_Path,
) (included binary.Boolean) {
	defer func() { binary.Boolean_Invariants(included, "path_at_or_below.included") }()
	File_Info_Addition_Name_Invariants(path, "path_at_or_below.path")
	File_System_Parent_Path_Invariants(root, "path_at_or_below.root")
	if file_system_root(bytes.Slice(root)) {
		return true
	}
	equal := bool(bytes.Equal(bytes.Slice(path), bytes.Slice(root)))
	below := bool(name_below_path(File_System_Member_Name(path), File_System_Path(root)))
	return binary.Boolean(equal || below)
}

func file_info_sort(nodes Collected_File_Infos) {
	Collected_File_Infos_Invariants(nodes, "file_info_sort.nodes")
	slices.Sort_Function(nodes, func(
		first File_Info, second File_Info,
	) (comparison slices.Comparison) {
		return slices.Comparison(bytes.Compare(
			bytes.Slice(first.Name), bytes.Slice(second.Name),
		))
	})
}

func writer_storage_valid(storage Writer_Storage) (valid binary.Boolean) {
	defer func() {
		binary.Boolean_Invariants(valid, "writer_storage_valid.valid")
	}()
	Writer_Storage_Invariants(storage, "writer_storage_valid.storage")
	if len(storage.Archive) < WRITER_DIRECTORY_END_SIZE {
		return false
	}
	if len(storage.Archive) > ARCHIVE_SIZE_MAXIMUM {
		return false
	}
	if len(storage.Central) < CENTRAL_HEADER_SIZE {
		return false
	}
	if len(storage.Central) > ARCHIVE_SIZE_MAXIMUM {
		return false
	}
	if len(storage.Content) > ARCHIVE_SIZE_MAXIMUM {
		return false
	}
	if len(storage.Compressed) > ARCHIVE_SIZE_MAXIMUM {
		return false
	}
	if len(storage.Comment) > ARCHIVE_SIZE_MAXIMUM {
		return false
	}
	if len(storage.Name) > ARCHIVE_SIZE_MAXIMUM {
		return false
	}
	return len(storage.Heads) == flate.HASH_COUNT &&
		len(storage.Previous) == flate.WINDOW_SIZE
}

func writer_create(
	writer Writer_Handle, header_unvalidated Header_Unvalidated_Handle, raw Writer_Raw,
) (status Creation_Status) {
	defer func() { Creation_Status_Invariants(status, "writer_create.status") }()
	Writer_Handle_Invariants(writer, "writer_create.writer")
	Header_Unvalidated_Handle_Invariants(header_unvalidated, "writer_create.header_unvalidated")
	Writer_Raw_Invariants(raw, "writer_create.raw")
	if bool(writer.Closed) {
		return Creation_Status(STATUS_INPUT_INVALID)
	}
	if bool(writer.In_Flight) {
		return Creation_Status(STATUS_INPUT_INVALID)
	}
	if writer.Active {
		finalize_status := writer_finalize(writer)
		if finalize_status != Bounded_Status(STATUS_OK) {
			return Creation_Status(finalize_status)
		}
	}
	header, extra_size, prepare_status := writer_header_prepare(header_unvalidated, raw)
	if prepare_status != Preparation_Status(STATUS_OK) {
		return Creation_Status(prepare_status)
	}
	local_size := LOCAL_HEADER_SIZE + len(header.Name) + int(extra_size)
	central_size := CENTRAL_HEADER_SIZE + len(header.Name) +
		int(extra_size) + len(header.Comment)
	if local_size > len(writer.Storage.Archive)-int(writer.Archive_Position) {
		return Creation_Status(STATUS_OUTPUT_TOO_SMALL)
	}
	if central_size > len(writer.Storage.Central)-int(writer.Central_Position) {
		return Creation_Status(STATUS_OUTPUT_TOO_SMALL)
	}
	writer.Header = header
	writer.Raw = raw
	writer.Directory = Writer_Directory(header.Name[len(header.Name)-1] == '/')
	name := Writer_Record_Name(header.Name)
	comment := Writer_Record_Comment(header.Comment)
	flags := writer_record_flags(
		name, comment, header.Flags, header.Non_UTF8, raw, writer.Directory,
	)
	method := Writer_Record_Method(header.Method)
	if writer.Directory {
		method = METHOD_STORE
	}
	local_start := int(writer.Archive_Position)
	record := Writer_Record{
		Name: name, Comment: comment, Extra: Writer_Record_Extra(header.Extra),
		Extra_Size: extra_size, Flags: flags, Method: method,
		Creator_Version:   header.Creator_Version,
		Extractor_Version: header.Extractor_Version,
		Modified_Time:     header.Modified_Time, Modified_Date: header.Modified_Date,
		External_Attributes: header.External_Attributes,
		Internal_Attributes: header.Internal_Attributes,
		Modified_Seconds:    header.Modified.Seconds, Modified_Set: header.Modified.Set,
		Local_Start: Writer_Local_Start(local_start),
	}
	writer.Header.Flags = flags
	writer.Header.Method = Header_Method(method)
	writer.Content_Position = Writer_Content_Position(POSITION_MINIMUM)
	writer.Current_Central_Position = Writer_Current_Central_Position(writer.Central_Position)
	writer_local_record_write(
		Writer_Local_Record_Storage(writer.Storage.Archive[local_start:]), record,
	)
	writer.Archive_Position += Writer_Archive_Position(local_size)
	central_start := int(writer.Central_Position)
	writer_central_record_write(
		Writer_Central_Record_Storage(writer.Storage.Central[central_start:]), record,
	)
	writer.Central_Position += Writer_Central_Position(central_size)
	writer.Active = true
	return Creation_Status(STATUS_OK)
}

func writer_header_prepare(
	value Header_Unvalidated_Handle, raw Writer_Raw,
) (header Header, extra_size Writer_Record_Extra_Size, status Preparation_Status) {
	defer func() {
		Header_Invariants(header, "writer_header_prepare.header")
		Writer_Record_Extra_Size_Invariants(
			extra_size, "writer_header_prepare.extra_size",
		)
		Preparation_Status_Invariants(
			status, "writer_header_prepare.status",
		)
	}()
	Header_Unvalidated_Handle_Invariants(value, "writer_header_prepare.value")
	Writer_Raw_Invariants(raw, "writer_header_prepare.raw")
	header, validation_status := Header_Validate(value)
	if validation_status != Validation_Status(STATUS_OK) {
		return header, 0, Preparation_Status(validation_status)
	}
	if header.Modified.Set {
		encoding, valid := timestamp_to_dos(header.Modified)
		if !valid {
			return Header{}, 0, Preparation_Status(STATUS_INPUT_INVALID)
		}
		header.Modified_Date = Header_Modified_Date(encoding.Date)
		header.Modified_Time = Header_Modified_Time(encoding.Clock)
	}
	header_status := writer_header_status(header.Method, raw)
	if header_status != Writer_Header_Status(STATUS_OK) {
		return header, 0, Preparation_Status(header_status)
	}
	extra_size = Writer_Record_Extra_Size(len(header.Extra))
	if header.Modified.Set {
		extra_size += EXTENDED_TIMESTAMP_SIZE
	}
	if int(extra_size) > WRITER_RECORD_EXTRA_SIZE_MAXIMUM {
		return Header{}, 0, Preparation_Status(STATUS_INPUT_INVALID)
	}
	return header, extra_size, Preparation_Status(STATUS_OK)
}

func writer_header_status(
	method Header_Method, raw Writer_Raw,
) (status Writer_Header_Status) {
	defer func() {
		Writer_Header_Status_Invariants(status, "writer_header_status.status")
	}()
	Header_Method_Invariants(method, "writer_header_status.method")
	Writer_Raw_Invariants(raw, "writer_header_status.raw")
	if method != METHOD_STORE {
		if method != METHOD_DEFLATE {
			return Writer_Header_Status(STATUS_METHOD_UNSUPPORTED)
		}
	}
	if !raw {
		return Writer_Header_Status(STATUS_OK)
	}
	return Writer_Header_Status(STATUS_OK)
}

func writer_record_flags(
	name Writer_Record_Name, comment Writer_Record_Comment,
	supplied Header_Flags, non_utf8 Header_Non_UTF8,
	raw Writer_Raw, directory Writer_Directory,
) (flags Header_Flags) {
	defer func() { Header_Flags_Invariants(flags, "writer_record_flags.flags") }()
	Writer_Record_Name_Invariants(name, "writer_record_flags.name")
	Writer_Record_Comment_Invariants(comment, "writer_record_flags.comment")
	Header_Flags_Invariants(supplied, "writer_record_flags.supplied")
	Header_Non_UTF8_Invariants(non_utf8, "writer_record_flags.non_utf8")
	Writer_Raw_Invariants(raw, "writer_record_flags.raw")
	Writer_Directory_Invariants(directory, "writer_record_flags.directory")
	if raw {
		return supplied
	}
	if !directory {
		flags |= FLAG_DATA_DESCRIPTOR
	}
	if !non_utf8 {
		for _, value := range []bytes.Slice{bytes.Slice(name), bytes.Slice(comment)} {
			for _, character := range value {
				if utf8.Byte(character) >= utf8.CHARACTER_SELF {
					return flags | FLAG_UTF8
				}
			}
		}
	}
	return flags
}

func writer_local_record_write(
	destination Writer_Local_Record_Storage, record Writer_Record,
) {
	Writer_Local_Record_Storage_Invariants(
		destination, "writer_local_record_write.destination",
	)
	Writer_Record_Invariants(record, "writer_local_record_write.record")
	header := destination[:LOCAL_HEADER_SIZE]
	clear(header)
	writer_put_word_32(Writer_Word_32_Destination(header), WRITER_LOCAL_HEADER_SIGNATURE)
	writer_put_word_16(
		Writer_Word_16_Destination(header[LOCAL_EXTRACTOR_POSITION:]),
		ZIP_VERSION_2_0,
	)
	writer_put_word_16(
		Writer_Word_16_Destination(header[WRITER_LOCAL_FLAGS_POSITION:]),
		binary.Word_16(record.Flags),
	)
	writer_put_word_16(
		Writer_Word_16_Destination(header[WRITER_LOCAL_METHOD_POSITION:]),
		binary.Word_16(record.Method),
	)
	writer_put_word_16(
		Writer_Word_16_Destination(header[LOCAL_MODIFIED_TIME_POSITION:]),
		binary.Word_16(record.Modified_Time),
	)
	writer_put_word_16(
		Writer_Word_16_Destination(header[LOCAL_MODIFIED_DATE_POSITION:]),
		binary.Word_16(record.Modified_Date),
	)
	writer_put_word_16(
		Writer_Word_16_Destination(header[WRITER_LOCAL_NAME_SIZE_POSITION:]),
		binary.Word_16(len(record.Name)),
	)
	writer_put_word_16(
		Writer_Word_16_Destination(header[WRITER_LOCAL_EXTRA_SIZE_POSITION:]),
		binary.Word_16(record.Extra_Size),
	)
	position := LOCAL_HEADER_SIZE
	position += copy(destination[position:], record.Name)
	position += copy(destination[position:], record.Extra)
	if record.Modified_Set {
		extended := destination[position : position+EXTENDED_TIMESTAMP_SIZE]
		writer_put_word_16(
			Writer_Word_16_Destination(extended), EXTENDED_TIMESTAMP_IDENTIFIER,
		)
		writer_put_word_16(
			Writer_Word_16_Destination(
				extended[EXTENDED_TIMESTAMP_DATA_SIZE_POSITION:],
			),
			EXTENDED_TIMESTAMP_DATA_SIZE,
		)
		extended[EXTENDED_TIMESTAMP_FLAGS_POSITION] = EXTENDED_TIMESTAMP_MODIFIED_FLAG
		writer_put_word_32(
			Writer_Word_32_Destination(extended[EXTENDED_TIMESTAMP_SECONDS_POSITION:]),
			binary.Word_32(record.Modified_Seconds),
		)
	}
}

func writer_central_record_write(
	destination Writer_Central_Record_Storage, record Writer_Record,
) {
	Writer_Central_Record_Storage_Invariants(
		destination, "writer_central_record_write.destination",
	)
	Writer_Record_Invariants(record, "writer_central_record_write.record")
	writer_central_record_header_write(destination, record)
	writer_central_record_tail_write(destination, record)
}

func writer_central_record_header_write(
	destination Writer_Central_Record_Storage, record Writer_Record,
) {
	Writer_Central_Record_Storage_Invariants(
		destination, "writer_central_record_header_write.destination",
	)
	Writer_Record_Invariants(record, "writer_central_record_header_write.record")
	header := destination[:CENTRAL_HEADER_SIZE]
	clear(header)
	creator := record.Creator_Version
	if creator == 0 {
		creator = ZIP_VERSION_2_0
	}
	extractor := record.Extractor_Version
	if extractor == 0 {
		extractor = ZIP_VERSION_2_0
	}
	writer_put_word_32(
		Writer_Word_32_Destination(header), WRITER_CENTRAL_HEADER_SIGNATURE,
	)
	writer_put_word_16(
		Writer_Word_16_Destination(header[WRITER_CENTRAL_CREATOR_POSITION:]),
		binary.Word_16(creator),
	)
	writer_put_word_16(
		Writer_Word_16_Destination(header[WRITER_CENTRAL_EXTRACTOR_POSITION:]),
		binary.Word_16(extractor),
	)
	writer_put_word_16(
		Writer_Word_16_Destination(header[CENTRAL_FLAGS_POSITION:]),
		binary.Word_16(record.Flags),
	)
	writer_put_word_16(
		Writer_Word_16_Destination(header[CENTRAL_METHOD_POSITION:]),
		binary.Word_16(record.Method),
	)
	writer_put_word_16(
		Writer_Word_16_Destination(header[CENTRAL_MODIFIED_TIME_POSITION:]),
		binary.Word_16(record.Modified_Time),
	)
	writer_put_word_16(
		Writer_Word_16_Destination(header[CENTRAL_MODIFIED_DATE_POSITION:]),
		binary.Word_16(record.Modified_Date),
	)
	writer_put_word_16(
		Writer_Word_16_Destination(header[WRITER_CENTRAL_NAME_SIZE_POSITION:]),
		binary.Word_16(len(record.Name)),
	)
	writer_put_word_16(
		Writer_Word_16_Destination(header[WRITER_CENTRAL_EXTRA_SIZE_POSITION:]),
		binary.Word_16(record.Extra_Size),
	)
	writer_put_word_16(
		Writer_Word_16_Destination(header[WRITER_CENTRAL_COMMENT_SIZE_POSITION:]),
		binary.Word_16(len(record.Comment)),
	)
	writer_put_word_32(
		Writer_Word_32_Destination(header[WRITER_CENTRAL_EXTERNAL_ATTRIBUTES_POSITION:]),
		binary.Word_32(record.External_Attributes),
	)
	writer_put_word_16(
		Writer_Word_16_Destination(header[CENTRAL_INTERNAL_ATTRIBUTES_POSITION:]),
		binary.Word_16(record.Internal_Attributes),
	)
	writer_put_word_32(
		Writer_Word_32_Destination(header[WRITER_CENTRAL_LOCAL_OFFSET_POSITION:]),
		binary.Word_32(record.Local_Start),
	)
}

func writer_central_record_tail_write(
	destination Writer_Central_Record_Storage, record Writer_Record,
) {
	Writer_Central_Record_Storage_Invariants(
		destination, "writer_central_record_tail_write.destination",
	)
	Writer_Record_Invariants(record, "writer_central_record_tail_write.record")
	position := CENTRAL_HEADER_SIZE
	position += copy(destination[position:], record.Name)
	position += copy(destination[position:], record.Extra)
	if record.Modified_Set {
		extended := destination[position : position+EXTENDED_TIMESTAMP_SIZE]
		writer_put_word_16(
			Writer_Word_16_Destination(extended), EXTENDED_TIMESTAMP_IDENTIFIER,
		)
		writer_put_word_16(
			Writer_Word_16_Destination(
				extended[EXTENDED_TIMESTAMP_DATA_SIZE_POSITION:],
			),
			EXTENDED_TIMESTAMP_DATA_SIZE,
		)
		extended[EXTENDED_TIMESTAMP_FLAGS_POSITION] = EXTENDED_TIMESTAMP_MODIFIED_FLAG
		writer_put_word_32(
			Writer_Word_32_Destination(extended[EXTENDED_TIMESTAMP_SECONDS_POSITION:]),
			binary.Word_32(record.Modified_Seconds),
		)
		position += EXTENDED_TIMESTAMP_SIZE
	}
	copy(destination[position:], record.Comment)
}

func writer_finalize(writer Writer_Handle) (status Bounded_Status) {
	defer func() { Bounded_Status_Invariants(status, "writer_finalize.status") }()
	Writer_Handle_Invariants(writer, "writer_finalize.writer")
	switch {
	case bool(writer.Closed):
		return Bounded_Status(STATUS_INPUT_INVALID)
	case bool(writer.In_Flight):
		return Bounded_Status(STATUS_INPUT_INVALID)
	case !bool(writer.Active):
		return Bounded_Status(STATUS_OK)
	}
	content := writer.Storage.Content[:writer.Content_Position]
	compressed := bytes.Slice(content)
	checksum := binary.Word_32(crc32.Checksum_IEEE(crc32.Source(content)))
	uncompressed_size := binary.Word_64(len(content))
	compressed_size := uncompressed_size
	if writer.Raw {
		checksum = binary.Word_32(writer.Header.Checksum)
		uncompressed_size = binary.Word_64(writer.Header.Uncompressed_Size)
		compressed_size = binary.Word_64(writer.Header.Compressed_Size)
		if binary.Word_64(len(content)) != compressed_size {
			return Bounded_Status(STATUS_INPUT_INVALID)
		}
	} else if writer.Header.Method == METHOD_DEFLATE {
		destination := flate.Destination_Unvalidated(writer.Storage.Compressed)
		workspace := flate.Workspace_Unvalidated{
			Heads:    flate.Hash_Positions_Unvalidated(writer.Storage.Heads),
			Previous: flate.History_Positions_Unvalidated(writer.Storage.Previous),
		}
		source := flate.Source_Unvalidated(content)
		level := flate.Level_Unvalidated(flate.DEFAULT_COMPRESSION)
		count, encode_status := flate.Encode_Into(destination, workspace, source, level)
		if encode_status != flate.Encode_Status(flate.STATUS_OK) {
			return Bounded_Status(STATUS_OUTPUT_TOO_SMALL)
		}
		compressed = bytes.Slice(writer.Storage.Compressed[:count])
		compressed_size = binary.Word_64(count)
	}
	descriptor_size := 0
	if binary.Word_16(writer.Header.Flags)&FLAG_DATA_DESCRIPTOR != 0 {
		descriptor_size = WRITER_DESCRIPTOR_SIZE
	}
	required := len(compressed) + descriptor_size
	if required > len(writer.Storage.Archive)-int(writer.Archive_Position) {
		return Bounded_Status(STATUS_OUTPUT_TOO_SMALL)
	}
	position := int(writer.Archive_Position)
	position += copy(writer.Storage.Archive[position:], compressed)
	if descriptor_size != 0 {
		descriptor := Writer_Word_32_Destination(
			writer.Storage.Archive[position : position+descriptor_size],
		)
		writer_put_word_32(descriptor, WRITER_DATA_DESCRIPTOR_SIGNATURE)
		checksum_tail := descriptor[DATA_DESCRIPTOR_CHECKSUM_POSITION:]
		writer_put_word_32(checksum_tail, checksum)
		compressed_tail := descriptor[DATA_DESCRIPTOR_COMPRESSED_SIZE_POSITION:]
		writer_put_word_32(compressed_tail, binary.Word_32(compressed_size))
		uncompressed_tail := descriptor[DATA_DESCRIPTOR_UNCOMPRESSED_SIZE_POSITION:]
		writer_put_word_32(uncompressed_tail, binary.Word_32(uncompressed_size))
		position += descriptor_size
	}
	central := writer.Storage.Central[writer.Current_Central_Position:]
	writer_central_sizes_write(Central_Header(central[:CENTRAL_HEADER_SIZE]), checksum,
		Writer_Finalized_Compressed_Size(compressed_size),
		Writer_Finalized_Uncompressed_Size(uncompressed_size))
	writer.Archive_Position = Writer_Archive_Position(position)
	writer.Content_Position, writer.Active = 0, false
	writer.Entry_Count++
	return Bounded_Status(STATUS_OK)
}

func writer_central_sizes_write(
	header Central_Header, checksum binary.Word_32,
	compressed_size Writer_Finalized_Compressed_Size,
	uncompressed_size Writer_Finalized_Uncompressed_Size,
) {
	Central_Header_Invariants(header, "writer_central_sizes_write.header")
	binary.Word_32_Invariants(checksum, "writer_central_sizes_write.checksum")
	Writer_Finalized_Compressed_Size_Invariants(
		compressed_size, "writer_central_sizes_write.compressed_size",
	)
	Writer_Finalized_Uncompressed_Size_Invariants(
		uncompressed_size, "writer_central_sizes_write.uncompressed_size",
	)
	writer_put_word_32(
		Writer_Word_32_Destination(header[CENTRAL_CHECKSUM_POSITION:]), checksum,
	)
	writer_put_word_32(
		Writer_Word_32_Destination(header[CENTRAL_COMPRESSED_SIZE_POSITION:]),
		binary.Word_32(compressed_size),
	)
	writer_put_word_32(
		Writer_Word_32_Destination(header[CENTRAL_UNCOMPRESSED_SIZE_POSITION:]),
		binary.Word_32(uncompressed_size),
	)
}

func writer_directory_end(
	writer Writer_Directory_State_Handle,
) (status Capacity_Status) {
	defer func() {
		Capacity_Status_Invariants(status, "writer_directory_end.status")
	}()
	Writer_Directory_State_Handle_Invariants(writer, "writer_directory_end.writer")
	central_size := int(writer.Central_Position)
	comment_size := int(writer.Comment_Count)
	required := central_size + WRITER_DIRECTORY_END_SIZE + comment_size
	if required > len(writer.Archive)-int(writer.Archive_Position) {
		return Capacity_Status(STATUS_OUTPUT_TOO_SMALL)
	}
	central_offset := int(writer.Archive_Position)
	position := central_offset
	position += copy(writer.Archive[position:], writer.Central[:central_size])
	end := writer.Archive[position : position+WRITER_DIRECTORY_END_SIZE]
	clear(end)
	writer_put_word_32(
		Writer_Word_32_Destination(end), WRITER_DIRECTORY_END_SIGNATURE,
	)
	writer_put_word_16(
		Writer_Word_16_Destination(end[DIRECTORY_END_DISK_ENTRY_COUNT_POSITION:]),
		binary.Word_16(writer.Entry_Count),
	)
	writer_put_word_16(
		Writer_Word_16_Destination(end[DIRECTORY_END_ENTRY_COUNT_POSITION:]),
		binary.Word_16(writer.Entry_Count),
	)
	writer_put_word_32(
		Writer_Word_32_Destination(end[DIRECTORY_END_CENTRAL_SIZE_POSITION:]),
		binary.Word_32(central_size),
	)
	writer_put_word_32(
		Writer_Word_32_Destination(end[DIRECTORY_END_CENTRAL_OFFSET_POSITION:]),
		binary.Word_32(central_offset),
	)
	position += WRITER_DIRECTORY_END_SIZE
	writer_put_word_16(
		Writer_Word_16_Destination(end[DIRECTORY_END_COMMENT_SIZE_POSITION:]),
		binary.Word_16(comment_size),
	)
	position += copy(
		writer.Archive[position:], writer.Comment[:comment_size],
	)
	writer.Archive_Position = Writer_Archive_Position(position)
	return Capacity_Status(STATUS_OK)
}

func writer_stream_submit(state unsafe.Pointer, completion nbio.Completion_Handle) {
	writer := (*Writer)(state)
	callback := nbio.Stream_Callback{
		State: writer, Data: int(writer.Count),
		Procedure: writer_stream_callback,
	}
	if writer.Stream.Procedure == nil {
		completion.Data = 0
		completion.Error = nbio.Stream_Empty
		nbio.Stream_Callback_Call(callback, completion)
		return
	}
	writer.Stream.Procedure(
		writer.Stream.State, completion, nbio.STREAM_MODE_WRITE_AT,
		bytes.Slice(writer.Storage.Archive[:writer.Count]), int64(writer.Offset),
		nbio.SEEK_FROM_START, callback,
	)
}

func writer_stream_callback(
	state_value nbio.State, data nbio.Stream_Callback_Data, callback nbio.Callback,
	completion nbio.Completion_Handle,
) {
	writer_stream_complete(
		unsafe.Pointer(state_value.(*Writer)), Writer_Count(data), callback, completion,
	)
}

func writer_stream_complete(
	state unsafe.Pointer, requested_count Writer_Count, _ nbio.Callback,
	completion nbio.Completion_Handle,
) {
	Writer_Count_Invariants(requested_count, "writer_stream_complete.requested_count")
	writer := (*Writer)(state)
	if requested_count != writer.Count {
		completion.Error = nbio.Stream_Short_Write
	}
	if completion.Error == nil {
		if completion.Data != int(requested_count) {
			completion.Error = nbio.Stream_Short_Write
		}
	}
	writer.In_Flight = false
	callback := writer.Callback
	writer.Callback = nil
	callback(completion)
}

func writer_callback(completion nbio.Completion_Handle, callback nbio.Callback) {
	completion.Data = 0
	callback(completion)
}

func writer_put_word_16(
	destination Writer_Word_16_Destination, value binary.Word_16,
) {
	Writer_Word_16_Destination_Invariants(
		destination, "writer_put_word_16.destination",
	)
	binary.Word_16_Invariants(value, "writer_put_word_16.value")
	binary.Put_Uint_16(binary.Bytes(destination), value, binary.LITTLE_ENDIAN)
}

func writer_put_word_32(
	destination Writer_Word_32_Destination, value binary.Word_32,
) {
	Writer_Word_32_Destination_Invariants(
		destination, "writer_put_word_32.destination",
	)
	binary.Word_32_Invariants(value, "writer_put_word_32.value")
	binary.Put_Uint_32(binary.Bytes(destination), value, binary.LITTLE_ENDIAN)
}

// Decode_Into finds first matching base-name suffix only after full directory validation.
func Decode_Into(
	source bytes.Slice, entry_suffix bytes.Text, destination bytes.Slice,
) (count bytes.Boundary, status Status) {
	defer func() {
		bytes.Boundary_Invariants(count, "Decode_Into.count")
		Status_Invariants(status, "Decode_Into.status")
	}()
	bytes.Slice_Invariants(source, "Decode_Into.source")
	bytes.Text_Invariants(entry_suffix, "Decode_Into.entry_suffix")
	bytes.Slice_Invariants(destination, "Decode_Into.destination")
	if !input_valid(source, entry_suffix, destination) {
		return 0, STATUS_INPUT_INVALID
	}
	archive := Archive(source)
	entry_suffix_validated := Entry_Suffix(entry_suffix)
	destination_validated := Destination(destination)
	end_position, entry_count, central_size, central_offset, found :=
		directory_end(archive)
	if !bool(found) {
		return 0, STATUS_INPUT_INVALID
	}
	if int(central_size) > int(end_position) {
		return 0, STATUS_INPUT_INVALID
	}
	central_start := Central_Position(int(end_position) - int(central_size))
	if int(central_offset) > int(central_start) {
		return 0, STATUS_INPUT_INVALID
	}
	base_offset := Base_Offset(int(central_start) - int(central_offset))
	position := central_start
	var selected Central_Entry
	selected_found := false
	for entry_index := Entry_Count(0); entry_index < entry_count; entry_index++ {
		if int(position) > int(end_position)-CENTRAL_HEADER_SIZE {
			return 0, STATUS_INPUT_INVALID
		}
		entry, next, valid := read_central_entry(
			Central_Archive(archive), position, Central_End(end_position),
		)
		if !bool(valid) {
			return 0, STATUS_INPUT_INVALID
		}
		position = Central_Position(next)
		name := entry.Name
		if !selected_found {
			selected_found = bool(name_has_suffix(name, entry_suffix_validated))
			if selected_found {
				selected = Central_Entry{
					Header: Central_Header(entry.Header),
					Name:   Selected_Name(entry.Name),
				}
			}
		}
	}
	if int(position) != int(end_position) {
		return 0, STATUS_INPUT_INVALID
	}
	if !selected_found {
		return 0, STATUS_ENTRY_NOT_FOUND
	}
	entry_output_count, entry_status := decode_entry(
		Selected_Archive(archive), destination_validated, base_offset,
		Selected_Central_Start(central_start), selected,
	)
	return bytes.Boundary(entry_output_count), Status(entry_status)
}

func input_valid(
	source bytes.Slice, entry_suffix bytes.Text, destination bytes.Slice,
) (valid binary.Boolean) {
	defer func() {
		binary.Boolean_Invariants(valid, "input_valid.valid")
	}()
	bytes.Slice_Invariants(source, "input_valid.source")
	bytes.Text_Invariants(entry_suffix, "input_valid.entry_suffix")
	bytes.Slice_Invariants(destination, "input_valid.destination")
	if len(source) < ARCHIVE_SIZE_MINIMUM {
		return false
	}
	if len(source) > ARCHIVE_SIZE_MAXIMUM {
		return false
	}
	if len(entry_suffix) < ENTRY_SUFFIX_SIZE_MINIMUM {
		return false
	}
	if len(entry_suffix) > ENTRY_SUFFIX_SIZE_MAXIMUM {
		return false
	}
	for position := range entry_suffix {
		if entry_suffix[position] == 0 {
			return false
		}
		if entry_suffix[position] == '/' {
			return false
		}
	}
	if len(destination) > DESTINATION_SIZE_MAXIMUM {
		return false
	}
	if len(destination) == 0 {
		return true
	}
	return binary.Boolean(!bytes.Overlap(source, destination))
}

func directory_end(
	source Archive,
) (
	position Directory_Position,
	entry_count Entry_Count,
	central_size Central_Size,
	central_offset Central_Offset,
	found binary.Boolean,
) {
	defer func() {
		Directory_Position_Invariants(position, "directory_end.position")
		Entry_Count_Invariants(entry_count, "directory_end.entry_count")
		Central_Size_Invariants(central_size, "directory_end.central_size")
		Central_Offset_Invariants(central_offset, "directory_end.central_offset")
		binary.Boolean_Invariants(found, "directory_end.found")
	}()
	Archive_Invariants(source, "directory_end.source")
	const DIRECTORY_END_SEARCH_SIZE_MAXIMUM = WRITER_DIRECTORY_END_SIZE +
		bytes.SLICE_SIZE_MAXIMUM
	minimum := len(source) - DIRECTORY_END_SEARCH_SIZE_MAXIMUM
	if minimum < 0 {
		minimum = 0
	}
	candidate_start := len(source) - WRITER_DIRECTORY_END_SIZE
	for candidate := candidate_start; candidate >= minimum; candidate-- {
		if binary.Uint_32(
			binary.Bytes(source[candidate:]), binary.LITTLE_ENDIAN,
		) != WRITER_DIRECTORY_END_SIGNATURE {
			continue
		}
		comment_size := int(binary.Uint_16(
			binary.Bytes(source[candidate+DIRECTORY_END_COMMENT_SIZE_POSITION:]),
			binary.LITTLE_ENDIAN,
		))
		if candidate+WRITER_DIRECTORY_END_SIZE+comment_size > len(source) {
			// Latest truncated record must hide earlier signatures or issue 66869
			// returns.
			return 0, 0, 0, 0, false
		}
		candidate_entry_count, candidate_central_size,
			candidate_central_offset, valid :=
			directory_end_candidate(source, Directory_Position(candidate))
		if !bool(valid) {
			return 0, 0, 0, 0, false
		}
		return Directory_Position(candidate), candidate_entry_count,
			candidate_central_size, candidate_central_offset, true
	}
	return 0, 0, 0, 0, false
}

func directory_end_candidate(
	source Archive, candidate Directory_Position,
) (
	entry_count Entry_Count,
	central_size Central_Size,
	central_offset Central_Offset,
	valid binary.Boolean,
) {
	defer func() {
		Entry_Count_Invariants(entry_count, "directory_end_candidate.entry_count")
		Central_Size_Invariants(
			central_size, "directory_end_candidate.central_size",
		)
		Central_Offset_Invariants(
			central_offset, "directory_end_candidate.central_offset",
		)
		binary.Boolean_Invariants(valid, "directory_end_candidate.valid")
	}()
	Archive_Invariants(source, "directory_end_candidate.source")
	Directory_Position_Invariants(
		candidate, "directory_end_candidate.candidate",
	)
	if binary.Uint_16(
		binary.Bytes(source[candidate+DIRECTORY_END_DISK_POSITION:]),
		binary.LITTLE_ENDIAN,
	) != 0 {
		return 0, 0, 0, false
	}
	if binary.Uint_16(
		binary.Bytes(source[candidate+DIRECTORY_END_CENTRAL_DISK_POSITION:]),
		binary.LITTLE_ENDIAN,
	) != 0 {
		return 0, 0, 0, false
	}
	raw_entry_count := binary.Uint_16(
		binary.Bytes(source[candidate+DIRECTORY_END_ENTRY_COUNT_POSITION:]),
		binary.LITTLE_ENDIAN,
	)
	if binary.Uint_16(
		binary.Bytes(source[candidate+DIRECTORY_END_DISK_ENTRY_COUNT_POSITION:]),
		binary.LITTLE_ENDIAN,
	) != raw_entry_count {
		return 0, 0, 0, false
	}
	raw_central_size := binary.Uint_32(
		binary.Bytes(source[candidate+DIRECTORY_END_CENTRAL_SIZE_POSITION:]),
		binary.LITTLE_ENDIAN,
	)
	if uint64(raw_central_size) > uint64(candidate) {
		return 0, 0, 0, false
	}
	if uint64(raw_entry_count) > uint64(raw_central_size)/CENTRAL_HEADER_SIZE {
		return 0, 0, 0, false
	}
	raw_central_offset := binary.Uint_32(
		binary.Bytes(source[candidate+DIRECTORY_END_CENTRAL_OFFSET_POSITION:]),
		binary.LITTLE_ENDIAN,
	)
	if uint64(raw_central_offset) > bytes.SLICE_SIZE_MAXIMUM {
		return 0, 0, 0, false
	}
	return Entry_Count(raw_entry_count), Central_Size(raw_central_size),
		Central_Offset(raw_central_offset), true
}

func read_central_entry(
	source Central_Archive, position Central_Position, central_end Central_End,
) (entry Central_Entry_Optional, next Central_Boundary, valid binary.Boolean) {
	defer func() {
		Central_Entry_Optional_Invariants(entry, "read_central_entry.entry")
		Central_Boundary_Invariants(next, "read_central_entry.next")
		binary.Boolean_Invariants(valid, "read_central_entry.valid")
	}()
	Central_Archive_Invariants(source, "read_central_entry.source")
	Central_Position_Invariants(position, "read_central_entry.position")
	Central_End_Invariants(central_end, "read_central_entry.central_end")
	available_size := int(central_end) - int(position)
	if available_size < CENTRAL_HEADER_SIZE {
		return Central_Entry_Optional{}, Central_Boundary(position), false
	}
	header := source[position : position+CENTRAL_HEADER_SIZE]
	if binary.Uint_32(binary.Bytes(header), binary.LITTLE_ENDIAN) !=
		WRITER_CENTRAL_HEADER_SIGNATURE {
		return Central_Entry_Optional{}, Central_Boundary(position), false
	}
	if binary.Uint_16(
		binary.Bytes(header[CENTRAL_DISK_POSITION:]), binary.LITTLE_ENDIAN,
	) != 0 {
		return Central_Entry_Optional{}, Central_Boundary(position), false
	}
	name_size, extra_size, comment_size, sizes_valid := central_sizes(
		Central_Header(header),
		Central_Tail_Size(available_size-CENTRAL_HEADER_SIZE),
	)
	if !bool(sizes_valid) {
		return Central_Entry_Optional{}, Central_Boundary(position), false
	}
	entry_size := CENTRAL_HEADER_SIZE + int(name_size) +
		int(extra_size) + int(comment_size)
	name_start := Central_Boundary(int(position) + CENTRAL_HEADER_SIZE)
	entry = Central_Entry_Optional{
		Header: Central_Header_Optional(header),
		Name: Member_Name(
			source[name_start : name_start+Central_Boundary(name_size)],
		),
	}
	return entry, Central_Boundary(int(position) + entry_size), true
}

func central_sizes(
	header Central_Header, available_count Central_Tail_Size,
) (
	name_size Central_Tail_Size,
	extra_size Central_Tail_Size,
	comment_size Central_Tail_Size,
	valid binary.Boolean,
) {
	defer func() {
		Central_Tail_Size_Invariants(name_size, "central_sizes.name_size")
		Central_Tail_Size_Invariants(extra_size, "central_sizes.extra_size")
		Central_Tail_Size_Invariants(comment_size, "central_sizes.comment_size")
		binary.Boolean_Invariants(valid, "central_sizes.valid")
	}()
	Central_Header_Invariants(header, "central_sizes.header")
	Central_Tail_Size_Invariants(
		available_count, "central_sizes.available_count",
	)
	name_size = Central_Tail_Size(binary.Uint_16(
		binary.Bytes(header[WRITER_CENTRAL_NAME_SIZE_POSITION:]), binary.LITTLE_ENDIAN,
	))
	extra_size = Central_Tail_Size(binary.Uint_16(
		binary.Bytes(header[WRITER_CENTRAL_EXTRA_SIZE_POSITION:]), binary.LITTLE_ENDIAN,
	))
	comment_size = Central_Tail_Size(binary.Uint_16(
		binary.Bytes(header[WRITER_CENTRAL_COMMENT_SIZE_POSITION:]), binary.LITTLE_ENDIAN,
	))
	if name_size > available_count {
		return 0, 0, 0, false
	}
	available_count -= name_size
	if extra_size > available_count {
		return 0, 0, 0, false
	}
	available_count -= extra_size
	if comment_size > available_count {
		return 0, 0, 0, false
	}
	return name_size, extra_size, comment_size, true
}

func name_has_suffix(
	name Member_Name, suffix Entry_Suffix,
) (matches binary.Boolean) {
	defer func() {
		binary.Boolean_Invariants(matches, "name_has_suffix.matches")
	}()
	Member_Name_Invariants(name, "name_has_suffix.name")
	Entry_Suffix_Invariants(suffix, "name_has_suffix.suffix")
	base_start := int(bytes.Last_Index_Byte(bytes.Slice(name), '/')) + 1
	base := bytes.Slice(name[base_start:])
	return binary.Boolean(bytes.Has_Text_Suffix(base, bytes.Text(suffix)))
}

func decode_entry(
	source Selected_Archive,
	destination Destination,
	base_offset Base_Offset,
	central_start Selected_Central_Start,
	entry Central_Entry,
) (count Output_Count, status Entry_Status) {
	defer func() {
		Output_Count_Invariants(count, "decode_entry.count")
		Entry_Status_Invariants(status, "decode_entry.status")
	}()
	Selected_Archive_Invariants(source, "decode_entry.source")
	Destination_Invariants(destination, "decode_entry.destination")
	Base_Offset_Invariants(base_offset, "decode_entry.base_offset")
	Selected_Central_Start_Invariants(central_start, "decode_entry.central_start")
	Central_Entry_Invariants(entry, "decode_entry.entry")
	header := entry.Header
	central_bytes := binary.Bytes(header)
	flags := binary.Uint_16(central_bytes[CENTRAL_FLAGS_POSITION:], binary.LITTLE_ENDIAN)
	method := binary.Uint_16(central_bytes[CENTRAL_METHOD_POSITION:], binary.LITTLE_ENDIAN)
	compressed_size := binary.Uint_32(
		central_bytes[CENTRAL_COMPRESSED_SIZE_POSITION:], binary.LITTLE_ENDIAN,
	)
	uncompressed_size := binary.Uint_32(
		central_bytes[CENTRAL_UNCOMPRESSED_SIZE_POSITION:], binary.LITTLE_ENDIAN,
	)
	if method != METHOD_STORE {
		if method != METHOD_DEFLATE {
			return 0, Entry_Status(STATUS_METHOD_UNSUPPORTED)
		}
	}
	allowed_flags := binary.Word_16(FLAGS_COMMON)
	if method == METHOD_DEFLATE {
		allowed_flags = FLAGS_DEFLATE
	}
	if flags&^allowed_flags != 0 {
		return 0, Entry_Status(STATUS_INPUT_INVALID)
	}
	if uint64(uncompressed_size) > uint64(len(destination)) {
		return 0, Entry_Status(STATUS_OUTPUT_TOO_SMALL)
	}
	data_position, valid := local_data_position(
		source, base_offset, central_start, entry,
	)
	if !bool(valid) {
		return 0, Entry_Status(STATUS_INPUT_INVALID)
	}
	if uint64(compressed_size) >
		uint64(int(central_start)-int(data_position)) {
		return 0, Entry_Status(STATUS_INPUT_INVALID)
	}
	compressed_end := Payload_Position(
		int(data_position) + int(compressed_size),
	)
	if flags&FLAG_DATA_DESCRIPTOR != 0 {
		if len(source) < DESCRIPTOR_ARCHIVE_SIZE_MINIMUM {
			return 0, Entry_Status(STATUS_INPUT_INVALID)
		}
		if int(central_start)-int(compressed_end) < DATA_DESCRIPTOR_SIZE {
			return 0, Entry_Status(STATUS_INPUT_INVALID)
		}
		if !bool(data_descriptor_matches(
			Descriptor_Archive(source), Descriptor_Position(compressed_end),
			Descriptor_Central_Start(central_start), header,
		)) {
			return 0, Entry_Status(STATUS_INPUT_INVALID)
		}
	}
	if !decode_payload(
		Payload_Archive(source), destination, Payload_Position(data_position),
		compressed_end, header,
	) {
		return 0, Entry_Status(STATUS_INPUT_INVALID)
	}
	return Output_Count(uncompressed_size), Entry_Status(STATUS_OK)
}

func decode_payload(
	source Payload_Archive,
	destination Destination,
	data_position Payload_Position,
	compressed_end Payload_Position,
	header Central_Header,
) (valid binary.Boolean) {
	defer func() {
		binary.Boolean_Invariants(valid, "decode_payload.valid")
	}()
	Payload_Archive_Invariants(source, "decode_payload.source")
	Destination_Invariants(destination, "decode_payload.destination")
	Payload_Position_Invariants(data_position, "decode_payload.data_position")
	Payload_Position_Invariants(compressed_end, "decode_payload.compressed_end")
	Central_Header_Invariants(header, "decode_payload.header")
	method := binary.Uint_16(
		binary.Bytes(header[CENTRAL_METHOD_POSITION:]), binary.LITTLE_ENDIAN,
	)
	checksum := binary.Uint_32(
		binary.Bytes(header[CENTRAL_CHECKSUM_POSITION:]), binary.LITTLE_ENDIAN,
	)
	compressed_size := binary.Uint_32(
		binary.Bytes(header[CENTRAL_COMPRESSED_SIZE_POSITION:]),
		binary.LITTLE_ENDIAN,
	)
	uncompressed_size := binary.Uint_32(
		binary.Bytes(header[CENTRAL_UNCOMPRESSED_SIZE_POSITION:]),
		binary.LITTLE_ENDIAN,
	)
	compressed := source[data_position:compressed_end]
	output := destination[:int(uncompressed_size)]
	switch method {
	case METHOD_STORE:
		if compressed_size != uncompressed_size {
			return false
		}
		copy(output, compressed)
	case METHOD_DEFLATE:
		decoded_count, decoded_status := flate.Decode_Into(
			flate.Destination_Unvalidated(output),
			flate.Compressed_Unvalidated(compressed),
		)
		if decoded_status != flate.STATUS_OK {
			return false
		}
		if uint64(decoded_count) != uint64(uncompressed_size) {
			return false
		}
	}
	return binary.Word_32(crc32.Checksum_IEEE(crc32.Source(output))) == checksum
}

func local_data_position(
	source Selected_Archive,
	base_offset Base_Offset,
	central_start Selected_Central_Start,
	entry Central_Entry,
) (data_position Local_Position, valid binary.Boolean) {
	defer func() {
		Local_Position_Invariants(
			data_position, "local_data_position.data_position",
		)
		binary.Boolean_Invariants(valid, "local_data_position.valid")
	}()
	Selected_Archive_Invariants(source, "local_data_position.source")
	Base_Offset_Invariants(base_offset, "local_data_position.base_offset")
	Selected_Central_Start_Invariants(
		central_start, "local_data_position.central_start",
	)
	Central_Entry_Invariants(entry, "local_data_position.entry")
	local_offset := binary.Uint_32(
		binary.Bytes(entry.Header[CENTRAL_LOCAL_OFFSET_POSITION:]),
		binary.LITTLE_ENDIAN,
	)
	if uint64(local_offset) >
		uint64(int(central_start)-int(base_offset)) {
		return Local_Position(base_offset), false
	}
	local_position := Local_Position(
		int(base_offset) + int(local_offset),
	)
	available_count := int(central_start) - int(local_position)
	if available_count < LOCAL_HEADER_SIZE {
		return local_position, false
	}
	header := source[local_position : local_position+LOCAL_HEADER_SIZE]
	if !bool(local_header_matches(
		Local_Header(header), entry.Header,
	)) {
		return local_position, false
	}
	name_size := int(binary.Uint_16(
		binary.Bytes(header[WRITER_LOCAL_NAME_SIZE_POSITION:]), binary.LITTLE_ENDIAN,
	))
	extra_size := int(binary.Uint_16(
		binary.Bytes(header[WRITER_LOCAL_EXTRA_SIZE_POSITION:]), binary.LITTLE_ENDIAN,
	))
	available_count -= LOCAL_HEADER_SIZE
	if name_size > available_count {
		return local_position, false
	}
	available_count -= name_size
	if extra_size > available_count {
		return local_position, false
	}
	data_position = Local_Position(
		int(local_position) + LOCAL_HEADER_SIZE + name_size + extra_size,
	)
	local_name_start := Local_Position(int(local_position) + LOCAL_HEADER_SIZE)
	local_name_end := local_name_start + Local_Position(name_size)
	local_name := source[local_name_start:local_name_end]
	central_name := entry.Name
	if !bytes.Equal(bytes.Slice(local_name), bytes.Slice(central_name)) {
		return data_position, false
	}
	return data_position, true
}

func local_header_matches(
	header Local_Header, central_header Central_Header,
) (matches binary.Boolean) {
	defer func() {
		binary.Boolean_Invariants(matches, "local_header_matches.matches")
	}()
	Local_Header_Invariants(header, "local_header_matches.header")
	Central_Header_Invariants(
		central_header, "local_header_matches.central_header",
	)
	central_flags := binary.Uint_16(
		binary.Bytes(central_header[CENTRAL_FLAGS_POSITION:]), binary.LITTLE_ENDIAN,
	)
	central_method := binary.Uint_16(
		binary.Bytes(central_header[CENTRAL_METHOD_POSITION:]), binary.LITTLE_ENDIAN,
	)
	central_checksum := binary.Uint_32(
		binary.Bytes(central_header[CENTRAL_CHECKSUM_POSITION:]), binary.LITTLE_ENDIAN,
	)
	central_compressed_size := binary.Uint_32(
		binary.Bytes(central_header[CENTRAL_COMPRESSED_SIZE_POSITION:]),
		binary.LITTLE_ENDIAN,
	)
	central_uncompressed_size := binary.Uint_32(
		binary.Bytes(central_header[CENTRAL_UNCOMPRESSED_SIZE_POSITION:]),
		binary.LITTLE_ENDIAN,
	)
	if binary.Uint_32(binary.Bytes(header), binary.LITTLE_ENDIAN) !=
		WRITER_LOCAL_HEADER_SIGNATURE {
		return false
	}
	if binary.Uint_16(
		binary.Bytes(header[WRITER_LOCAL_FLAGS_POSITION:]), binary.LITTLE_ENDIAN,
	) != central_flags {
		return false
	}
	if binary.Uint_16(
		binary.Bytes(header[WRITER_LOCAL_METHOD_POSITION:]), binary.LITTLE_ENDIAN,
	) != central_method {
		return false
	}
	if central_flags&FLAG_DATA_DESCRIPTOR != 0 {
		return true
	}
	if binary.Uint_32(
		binary.Bytes(header[LOCAL_CHECKSUM_POSITION:]), binary.LITTLE_ENDIAN,
	) != central_checksum {
		return false
	}
	if binary.Uint_32(
		binary.Bytes(header[LOCAL_COMPRESSED_SIZE_POSITION:]),
		binary.LITTLE_ENDIAN,
	) != central_compressed_size {
		return false
	}
	return binary.Uint_32(
		binary.Bytes(header[LOCAL_UNCOMPRESSED_SIZE_POSITION:]),
		binary.LITTLE_ENDIAN,
	) == central_uncompressed_size
}

func data_descriptor_matches(
	source Descriptor_Archive,
	position Descriptor_Position,
	central_start Descriptor_Central_Start,
	header Central_Header,
) (matches binary.Boolean) {
	defer func() {
		binary.Boolean_Invariants(matches, "data_descriptor_matches.matches")
	}()
	Descriptor_Archive_Invariants(source, "data_descriptor_matches.source")
	Descriptor_Position_Invariants(position, "data_descriptor_matches.position")
	Descriptor_Central_Start_Invariants(
		central_start, "data_descriptor_matches.central_start",
	)
	Central_Header_Invariants(header, "data_descriptor_matches.header")
	checksum := binary.Uint_32(
		binary.Bytes(header[CENTRAL_CHECKSUM_POSITION:]), binary.LITTLE_ENDIAN,
	)
	compressed_size := binary.Uint_32(
		binary.Bytes(header[CENTRAL_COMPRESSED_SIZE_POSITION:]),
		binary.LITTLE_ENDIAN,
	)
	uncompressed_size := binary.Uint_32(
		binary.Bytes(header[CENTRAL_UNCOMPRESSED_SIZE_POSITION:]),
		binary.LITTLE_ENDIAN,
	)
	available_count := int(central_start) - int(position)
	if available_count < DATA_DESCRIPTOR_SIZE {
		return false
	}
	first := binary.Uint_32(
		binary.Bytes(source[position:]), binary.LITTLE_ENDIAN,
	)
	if first == WRITER_DATA_DESCRIPTOR_SIGNATURE {
		if available_count >= WRITER_DESCRIPTOR_SIZE {
			checksum_position := position + DATA_DESCRIPTOR_CHECKSUM_POSITION
			if binary.Uint_32(
				binary.Bytes(source[checksum_position:]),
				binary.LITTLE_ENDIAN,
			) == checksum {
				compressed_position := position +
					DATA_DESCRIPTOR_COMPRESSED_SIZE_POSITION
				if binary.Uint_32(
					binary.Bytes(source[compressed_position:]),
					binary.LITTLE_ENDIAN,
				) == compressed_size {
					uncompressed_position := position +
						DATA_DESCRIPTOR_UNCOMPRESSED_SIZE_POSITION
					if binary.Uint_32(
						binary.Bytes(source[uncompressed_position:]),
						binary.LITTLE_ENDIAN,
					) == uncompressed_size {
						return true
					}
				}
			}
		}
	}
	if first != checksum {
		return false
	}
	if binary.Uint_32(
		binary.Bytes(source[position+binary.UINT_32_SIZE:]), binary.LITTLE_ENDIAN,
	) != compressed_size {
		return false
	}
	return binary.Uint_32(
		binary.Bytes(source[position+2*binary.UINT_32_SIZE:]), binary.LITTLE_ENDIAN,
	) == uncompressed_size
}
