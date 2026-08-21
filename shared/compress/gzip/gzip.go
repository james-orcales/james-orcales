// Package gzip bounds RFC 1952 members with caller-owned storage.
package gzip

import (
	"hash/crc32"

	"local/james-orcales/shared/compress/flate"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/sim/aver/default"
	"local/james-orcales/shared/unicode/utf8"
)

// BYTE_COUNT_MAXIMUM keeps wrapper storage inside raw DEFLATE boundary.
const BYTE_COUNT_MAXIMUM = flate.BYTE_COUNT_MAXIMUM

// BYTE_COUNT_MINIMUM admits empty source, destination, and member sequence.
const BYTE_COUNT_MINIMUM = 0

// BYTE_COUNT_UNVALIDATED_MAXIMUM admits first rejected byte.
const BYTE_COUNT_UNVALIDATED_MAXIMUM = BYTE_COUNT_MAXIMUM + 1

// EXTRA_SIZE_MAXIMUM follows RFC 1952 two-byte extra length.
const EXTRA_SIZE_MAXIMUM = 1<<16 - 1

// HEADER_STRING_BYTE_COUNT_MAXIMUM preserves stdlib fixed header reader bound.
const HEADER_STRING_BYTE_COUNT_MAXIMUM = 511

// HEADER_TEXT_SIZE_MAXIMUM reserves two UTF-8 bytes per Latin-1 wire byte.
const HEADER_TEXT_SIZE_MAXIMUM = HEADER_STRING_BYTE_COUNT_MAXIMUM * 2

// HEADER_TEXT_SIZE_MINIMUM excludes absent optional text from wire encoding.
const HEADER_TEXT_SIZE_MINIMUM = 1

// HASH_POSITIONS_COUNT exposes exact caller workspace required by DEFLATE.
const HASH_POSITIONS_COUNT = flate.HASH_POSITIONS_COUNT_MAXIMUM - 1

// HISTORY_POSITIONS_COUNT exposes exact caller workspace required by DEFLATE.
const HISTORY_POSITIONS_COUNT = flate.HISTORY_POSITIONS_COUNT_MAXIMUM - 1

// HASH_POSITIONS_COUNT_UNVALIDATED_MAXIMUM admits first rejected head slot.
const HASH_POSITIONS_COUNT_UNVALIDATED_MAXIMUM = HASH_POSITIONS_COUNT + 1

// HISTORY_POSITIONS_COUNT_UNVALIDATED_MAXIMUM admits first rejected history slot.
const HISTORY_POSITIONS_COUNT_UNVALIDATED_MAXIMUM = HISTORY_POSITIONS_COUNT + 1

// OPERATING_SYSTEM_UNKNOWN retains stdlib writer default.
const OPERATING_SYSTEM_UNKNOWN = 255

// NO_COMPRESSION retains stdlib compression-level identity.
const NO_COMPRESSION = flate.NO_COMPRESSION

// BEST_SPEED retains stdlib compression-level identity.
const BEST_SPEED = flate.BEST_SPEED

// BEST_COMPRESSION retains stdlib compression-level identity.
const BEST_COMPRESSION = flate.BEST_COMPRESSION

// DEFAULT_COMPRESSION retains stdlib compression-level identity.
const DEFAULT_COMPRESSION = flate.DEFAULT_COMPRESSION

// HUFFMAN_ONLY retains stdlib compression-level identity.
const HUFFMAN_ONLY = flate.HUFFMAN_ONLY

// HEADER_SIZE fixes member prefix before optional metadata.
const HEADER_SIZE = 10

// TRAILER_SIZE fixes checksum and uncompressed-size suffix.
const TRAILER_SIZE = 8

// EXTRA_SIZE_SIZE fixes extra-data length prefix.
const EXTRA_SIZE_SIZE = 2

// HEADER_CHECKSUM_SIZE fixes optional truncated CRC width.
const HEADER_CHECKSUM_SIZE = 2

// IDENTIFIER_ONE rejects non-gzip bytes before DEFLATE work.
const IDENTIFIER_ONE = 0x1f

// IDENTIFIER_TWO rejects non-gzip bytes before DEFLATE work.
const IDENTIFIER_TWO = 0x8b

// DEFLATE_METHOD rejects unsupported compression before DEFLATE work.
const DEFLATE_METHOD = 8

// FLAG_HEADER_CHECKSUM protects optional metadata integrity.
const FLAG_HEADER_CHECKSUM = 1 << 1

// FLAG_EXTRA guards two-byte extra length and payload.
const FLAG_EXTRA = 1 << 2

// FLAG_NAME guards NUL-terminated original name.
const FLAG_NAME = 1 << 3

// FLAG_COMMENT guards NUL-terminated comment.
const FLAG_COMMENT = 1 << 4

// EXTRA_FLAGS_DEFAULT leaves compression hint unset.
const EXTRA_FLAGS_DEFAULT Extra_Flags = 0

// EXTRA_FLAGS_BEST marks strongest compression.
const EXTRA_FLAGS_BEST Extra_Flags = 2

// EXTRA_FLAGS_FAST marks fastest compression.
const EXTRA_FLAGS_FAST Extra_Flags = 4

// STATUS_MINIMUM anchors complete scalar result domain.
const STATUS_MINIMUM = STATUS_OK

// STATUS_MAXIMUM closes complete scalar result domain.
const STATUS_MAXIMUM = STATUS_STORAGE_INVALID

// MODIFIED_SECONDS_MINIMUM is RFC zero timestamp sentinel.
const MODIFIED_SECONDS_MINIMUM uint32 = 0

// MODIFIED_SECONDS_MAXIMUM is largest RFC timestamp word.
const MODIFIED_SECONDS_MAXIMUM uint32 = 1<<32 - 1

// OPERATING_SYSTEM_MINIMUM is first RFC platform identifier.
const OPERATING_SYSTEM_MINIMUM uint8 = 0

// OPERATING_SYSTEM_MAXIMUM is last RFC platform identifier.
const OPERATING_SYSTEM_MAXIMUM uint8 = 1<<8 - 1

// HEADER_EXTRA_POSITION_MAXIMUM follows largest optional extra field.
const HEADER_EXTRA_POSITION_MAXIMUM = HEADER_SIZE + EXTRA_SIZE_SIZE + EXTRA_SIZE_MAXIMUM

// HEADER_NAME_POSITION_MAXIMUM follows largest optional extra and name fields.
const HEADER_NAME_POSITION_MAXIMUM = HEADER_EXTRA_POSITION_MAXIMUM +
	HEADER_STRING_BYTE_COUNT_MAXIMUM + 1

// HEADER_SIZE_MAXIMUM includes every largest optional encoder field.
const HEADER_SIZE_MAXIMUM = HEADER_NAME_POSITION_MAXIMUM +
	HEADER_STRING_BYTE_COUNT_MAXIMUM + 1

// HEADER_POSITION_MAXIMUM includes optional decoder header checksum.
const HEADER_POSITION_MAXIMUM = HEADER_SIZE_MAXIMUM + HEADER_CHECKSUM_SIZE

// HEADER_TEXT_WIRE_DESTINATION_SIZE_MINIMUM holds prefix, text, and terminator.
const HEADER_TEXT_WIRE_DESTINATION_SIZE_MINIMUM = HEADER_SIZE + 2

// TRAILER_SOURCE_SIZE_MAXIMUM follows shortest raw stream after fixed header.
const TRAILER_SOURCE_SIZE_MAXIMUM = BYTE_COUNT_MAXIMUM - HEADER_SIZE - 2

// Encode_Status excludes decoder-only failures.
type Encode_Status uint8

// Encode_Status_Invariants keeps every encoder result scalar.
func Encode_Status_Invariants(
	value Encode_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), STATUS_MINIMUM, STATUS_MAXIMUM).
		Ensure()
}

// Decode_Status excludes encoder-only invalid level.
type Decode_Status uint8

// Decode_Status_Invariants bounds all decoder results.
func Decode_Status_Invariants(
	value Decode_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), uint8(STATUS_MINIMUM), uint8(STATUS_MAXIMUM)).
		Ensure()
}

// Header_Status keeps metadata parser results.
type Header_Status uint8

// Header_Status_Invariants states metadata parser outcomes.
func Header_Status_Invariants(
	value Header_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), STATUS_OK, STATUS_HEADER_INVALID,
			STATUS_HEADER_STORAGE_TOO_SMALL,
		).
		Ensure()
}

// Header_Validation_Status excludes caller-storage failure.
type Header_Validation_Status uint8

// Header_Validation_Status_Invariants keeps pure parse result scalar.
func Header_Validation_Status_Invariants(
	value Header_Validation_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), STATUS_OK, STATUS_INPUT_INVALID,
			STATUS_HEADER_INVALID,
		).
		Ensure()
}

// Member_Status excludes public storage validation.
type Member_Status uint8

// Member_Status_Invariants bounds member parser results.
func Member_Status_Invariants(
	value Member_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value), STATUS_MINIMUM, STATUS_HEADER_STORAGE_TOO_SMALL,
		).
		Ensure()
}

// STATUS_OK proves complete requested operation.
const STATUS_OK = 0

// STATUS_INPUT_INVALID rejects malformed DEFLATE or incomplete trailer.
const STATUS_INPUT_INVALID = 1

// STATUS_HEADER_INVALID rejects malformed metadata or wire header.
const STATUS_HEADER_INVALID = 2

// STATUS_CHECKSUM_INVALID rejects tentative bytes with wrong trailer.
const STATUS_CHECKSUM_INVALID = 3

// STATUS_OUTPUT_TOO_SMALL protects caller decoded or encoded bound.
const STATUS_OUTPUT_TOO_SMALL = 4

// STATUS_HEADER_STORAGE_TOO_SMALL protects caller metadata bound.
const STATUS_HEADER_STORAGE_TOO_SMALL = 5

// STATUS_LEVEL_INVALID rejects compression work outside stdlib levels.
const STATUS_LEVEL_INVALID = 6

// STATUS_STORAGE_INVALID rejects missing workspace or harmful aliases.
const STATUS_STORAGE_INVALID = 7

// Count prevents output and input cursors from leaving package byte bound.
type Count int

// Count_Invariants keeps every public cursor inside caller storage domain.
func Count_Invariants(value Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), BYTE_COUNT_MINIMUM, BYTE_COUNT_MAXIMUM).
		Ensure()
}

// Encoded_Count reports bytes committed before success or bounded failure.
type Encoded_Count int

// Encoded_Count_Invariants keeps committed output inside caller byte bound.
func Encoded_Count_Invariants(
	value Encoded_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), BYTE_COUNT_MINIMUM, BYTE_COUNT_MAXIMUM).
		Ensure()
}

// Compressed_Count reports bytes accepted or rejected by one member parse.
type Compressed_Count int

// Compressed_Count_Invariants keeps input progress inside caller byte bound.
func Compressed_Count_Invariants(
	value Compressed_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), BYTE_COUNT_MINIMUM, BYTE_COUNT_MAXIMUM).
		Ensure()
}

// Boolean gives validation and alias decisions one bounded identity.
type Boolean bool

// Boolean_Invariants forces witnesses for both decisions.
func Boolean_Invariants(value Boolean, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Gzip decision is true.").
		Ensure()
}

// Capture_Header distinguishes first-member metadata from ignored later metadata.
type Capture_Header bool

// Capture_Header_Invariants forces first and later member witnesses.
func Capture_Header_Invariants(value Capture_Header, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Current member metadata reaches caller storage.").
		Ensure()
}

// Bytes gives alias checks bounded caller storage.
type Bytes []byte

// Bytes_Invariants prevents alias scans outside package byte bound.
func Bytes_Invariants(value Bytes, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), BYTE_COUNT_MINIMUM, BYTE_COUNT_MAXIMUM).
		Ensure()
}

// Header_Size bounds fixed prefix plus optional metadata.
type Header_Size int

// Header_Size_Invariants bounds one complete fixed and optional header.
func Header_Size_Invariants(value Header_Size, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), HEADER_SIZE, HEADER_SIZE_MAXIMUM).
		Ensure()
}

// Header_Position bounds one metadata parser or writer cursor.
type Header_Position int

// Header_Position_Invariants bounds cursors after one complete fixed prefix.
func Header_Position_Invariants(
	value Header_Position, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), HEADER_SIZE, HEADER_POSITION_MAXIMUM).
		Ensure()
}

// Header_Extra_Position bounds cursor after optional extra metadata.
type Header_Extra_Position int

// Header_Extra_Position_Invariants retains fixed prefix through largest extra.
func Header_Extra_Position_Invariants(
	value Header_Extra_Position, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), HEADER_SIZE, HEADER_EXTRA_POSITION_MAXIMUM).
		Ensure()
}

// Header_Name_Position bounds cursor after optional extra and name metadata.
type Header_Name_Position int

// Header_Name_Position_Invariants retains fixed prefix through largest name.
func Header_Name_Position_Invariants(
	value Header_Name_Position, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), HEADER_SIZE, HEADER_NAME_POSITION_MAXIMUM).
		Ensure()
}

// Header_Comment_Position bounds cursor before optional header checksum.
type Header_Comment_Position int

// Header_Comment_Position_Invariants retains fixed prefix through largest comment.
func Header_Comment_Position_Invariants(
	value Header_Comment_Position, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), HEADER_SIZE, HEADER_SIZE_MAXIMUM).
		Ensure()
}

// Header_Text_End_Position bounds one emitted non-empty text field.
type Header_Text_End_Position int

// Header_Text_End_Position_Invariants excludes absent emitted text.
func Header_Text_End_Position_Invariants(
	value Header_Text_End_Position, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), HEADER_TEXT_WIRE_DESTINATION_SIZE_MINIMUM,
			HEADER_SIZE_MAXIMUM,
		).
		Ensure()
}

// Header_Text_Size bounds UTF-8 expansion of one Latin-1 field.
type Header_Text_Size int

// Header_Text_Size_Invariants protects caller metadata destination.
func Header_Text_Size_Invariants(
	value Header_Text_Size, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), BYTE_COUNT_MINIMUM, HEADER_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Header_String_Byte_Count bounds one Latin-1 wire field.
type Header_String_Byte_Count int

// Header_String_Byte_Count_Invariants preserves stdlib fixed reader bound.
func Header_String_Byte_Count_Invariants(
	value Header_String_Byte_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), BYTE_COUNT_MINIMUM, HEADER_STRING_BYTE_COUNT_MAXIMUM,
		).
		Ensure()
}

// Header_Flags preserves every hostile wire flag combination.
type Header_Flags uint8

// Header_Flags_Invariants preserves complete wire byte domain.
func Header_Flags_Invariants(value Header_Flags, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value), OPERATING_SYSTEM_MINIMUM, OPERATING_SYSTEM_MAXIMUM,
		).
		Ensure()
}

// Extra_Flags keeps only stdlib compression hints.
type Extra_Flags uint8

// Extra_Flags_Invariants states default, strongest, and fastest hints.
func Extra_Flags_Invariants(value Extra_Flags, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(EXTRA_FLAGS_DEFAULT), uint8(EXTRA_FLAGS_BEST),
			uint8(EXTRA_FLAGS_FAST),
		).
		Ensure()
}

// Header_Destination keeps encoded metadata inside caller output.
type Header_Destination []byte

// Header_Destination_Invariants bounds complete encoded header.
func Header_Destination_Invariants(
	value Header_Destination, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), HEADER_SIZE, HEADER_SIZE_MAXIMUM).
		Ensure()
}

// Header_Text_Source keeps validated UTF-8 metadata bounded.
type Header_Text_Source []byte

// Header_Text_Source_Invariants bounds one validated API field.
func Header_Text_Source_Invariants(
	value Header_Text_Source, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), BYTE_COUNT_MINIMUM, HEADER_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Header_Text_Non_Empty_Source holds selected wire metadata.
type Header_Text_Non_Empty_Source []byte

// Header_Text_Non_Empty_Source_Invariants rejects absent optional text.
func Header_Text_Non_Empty_Source_Invariants(
	value Header_Text_Non_Empty_Source, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), HEADER_TEXT_SIZE_MINIMUM, HEADER_TEXT_SIZE_MAXIMUM,
		).
		Ensure()
}

// Header_Text_Destination keeps decoded UTF-8 metadata caller-owned.
type Header_Text_Destination []byte

// Header_Text_Destination_Invariants bounds one decoded API field.
func Header_Text_Destination_Invariants(
	value Header_Text_Destination, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), BYTE_COUNT_MINIMUM, HEADER_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Header_Text_Wire_Destination holds header containing optional text.
type Header_Text_Wire_Destination []byte

// Header_Text_Wire_Destination_Invariants retains text plus terminator.
func Header_Text_Wire_Destination_Invariants(
	value Header_Text_Wire_Destination, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), HEADER_TEXT_WIRE_DESTINATION_SIZE_MINIMUM,
			HEADER_SIZE_MAXIMUM,
		).
		Ensure()
}

// Header_String_Source holds one Latin-1 wire string.
type Header_String_Source []byte

// Header_String_Source_Invariants keeps stdlib wire bound.
func Header_String_Source_Invariants(
	value Header_String_Source, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), BYTE_COUNT_MINIMUM, HEADER_STRING_BYTE_COUNT_MAXIMUM,
		).
		Ensure()
}

// Level_Unvalidated retains hostile scalar without interface conversion.
type Level_Unvalidated int8

// Level_Unvalidated_Invariants preserves every representable caller value.
func Level_Unvalidated_Invariants(
	value Level_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int8(
			int8(value), flate.LEVEL_UNVALIDATED_MINIMUM,
			flate.LEVEL_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Level is one validated standard compression level.
type Level int8

// Level_Invariants bounds all accepted compression levels.
func Level_Invariants(value Level, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int8(int8(value), int8(HUFFMAN_ONLY), int8(BEST_COMPRESSION)).
		Ensure()
}

// Destination keeps decoded or encoded bytes caller-owned.
type Destination []byte

// Destination_Invariants prevents writes outside package byte bound.
func Destination_Invariants(value Destination, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), BYTE_COUNT_MINIMUM, BYTE_COUNT_MAXIMUM).
		Ensure()
}

// Destination_Unvalidated holds caller output before byte-bound check.
type Destination_Unvalidated []byte

// Destination_Unvalidated_Invariants admits first rejected byte.
func Destination_Unvalidated_Invariants(
	value Destination_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), BYTE_COUNT_MINIMUM, BYTE_COUNT_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Source_Unvalidated holds caller input before byte-bound check.
type Source_Unvalidated []byte

// Source_Unvalidated_Invariants admits first rejected byte.
func Source_Unvalidated_Invariants(
	value Source_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), BYTE_COUNT_MINIMUM, BYTE_COUNT_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Compressed keeps RFC 1952 bytes caller-owned.
type Compressed []byte

// Compressed_Invariants prevents parsing outside package byte bound.
func Compressed_Invariants(value Compressed, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), BYTE_COUNT_MINIMUM, BYTE_COUNT_MAXIMUM).
		Ensure()
}

// Member holds bytes after fixed header validation.
type Member []byte

// Member_Invariants retains one bounded fixed header.
func Member_Invariants(value Member, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), HEADER_SIZE, BYTE_COUNT_MAXIMUM).
		Ensure()
}

// Trailer_Source holds trailer plus possible following members.
type Trailer_Source []byte

// Trailer_Source_Invariants retains complete trailer.
func Trailer_Source_Invariants(
	value Trailer_Source, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), TRAILER_SIZE, TRAILER_SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Compressed_Unvalidated holds member bytes before byte-bound check.
type Compressed_Unvalidated []byte

// Compressed_Unvalidated_Invariants admits first rejected byte.
func Compressed_Unvalidated_Invariants(
	value Compressed_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), BYTE_COUNT_MINIMUM, BYTE_COUNT_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Extra_Unvalidated retains caller metadata until RFC length validation.
type Extra_Unvalidated []byte

// Extra_Unvalidated_Invariants keeps hostile metadata inside package byte bound.
func Extra_Unvalidated_Invariants(
	value Extra_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), BYTE_COUNT_MINIMUM, BYTE_COUNT_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Name_Unvalidated retains caller UTF-8 until Latin-1 validation.
type Name_Unvalidated []byte

// Name_Unvalidated_Invariants keeps hostile metadata inside package byte bound.
func Name_Unvalidated_Invariants(
	value Name_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), BYTE_COUNT_MINIMUM, BYTE_COUNT_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Comment_Unvalidated retains caller UTF-8 until Latin-1 validation.
type Comment_Unvalidated []byte

// Comment_Unvalidated_Invariants keeps hostile metadata inside package byte bound.
func Comment_Unvalidated_Invariants(
	value Comment_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), BYTE_COUNT_MINIMUM, BYTE_COUNT_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Extra_Destination_Unvalidated keeps decoded metadata caller-owned.
type Extra_Destination_Unvalidated []byte

// Extra_Destination_Unvalidated_Invariants bounds hostile destination storage.
func Extra_Destination_Unvalidated_Invariants(
	value Extra_Destination_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), BYTE_COUNT_MINIMUM, BYTE_COUNT_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Name_Destination_Unvalidated keeps decoded UTF-8 caller-owned.
type Name_Destination_Unvalidated []byte

// Name_Destination_Unvalidated_Invariants bounds hostile destination storage.
func Name_Destination_Unvalidated_Invariants(
	value Name_Destination_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), BYTE_COUNT_MINIMUM, BYTE_COUNT_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Comment_Destination_Unvalidated keeps decoded UTF-8 caller-owned.
type Comment_Destination_Unvalidated []byte

// Comment_Destination_Unvalidated_Invariants bounds hostile destination storage.
func Comment_Destination_Unvalidated_Invariants(
	value Comment_Destination_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), BYTE_COUNT_MINIMUM, BYTE_COUNT_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Extra_Destination bounds used extra storage.
type Extra_Destination []byte

// Extra_Destination_Invariants enforces RFC extra bound.
func Extra_Destination_Invariants(
	value Extra_Destination, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), BYTE_COUNT_MINIMUM, EXTRA_SIZE_MAXIMUM).
		Ensure()
}

// Name_Destination bounds used name storage.
type Name_Destination []byte

// Name_Destination_Invariants enforces Latin-1 expansion bound.
func Name_Destination_Invariants(
	value Name_Destination, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), BYTE_COUNT_MINIMUM, HEADER_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Comment_Destination bounds used comment storage.
type Comment_Destination []byte

// Comment_Destination_Invariants enforces Latin-1 expansion bound.
func Comment_Destination_Invariants(
	value Comment_Destination, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), BYTE_COUNT_MINIMUM, HEADER_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Extra is validated first-member metadata in caller storage.
type Extra []byte

// Extra_Invariants enforces RFC two-byte length.
func Extra_Invariants(value Extra, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), BYTE_COUNT_MINIMUM, EXTRA_SIZE_MAXIMUM).
		Ensure()
}

// Name is validated UTF-8 first-member metadata in caller storage.
type Name []byte

// Name_Invariants reserves worst-case Latin-1 expansion.
func Name_Invariants(value Name, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), BYTE_COUNT_MINIMUM, HEADER_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Comment is validated UTF-8 first-member metadata in caller storage.
type Comment []byte

// Comment_Invariants reserves worst-case Latin-1 expansion.
func Comment_Invariants(value Comment, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), BYTE_COUNT_MINIMUM, HEADER_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Modified_Seconds preserves complete RFC 1952 timestamp word.
type Modified_Seconds uint32

// Modified_Seconds_Invariants preserves complete wire domain.
func Modified_Seconds_Invariants(
	value Modified_Seconds, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint32(
			uint32(value), MODIFIED_SECONDS_MINIMUM, MODIFIED_SECONDS_MAXIMUM,
		).
		Ensure()
}

// Operating_System preserves complete RFC 1952 platform byte.
type Operating_System uint8

// Operating_System_Invariants preserves complete wire domain.
func Operating_System_Invariants(
	value Operating_System, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value), OPERATING_SYSTEM_MINIMUM, OPERATING_SYSTEM_MAXIMUM,
		).
		Ensure()
}

// Hash_Positions_Unvalidated keeps encoder hash heads caller-owned.
type Hash_Positions_Unvalidated []int32

// Hash_Positions_Unvalidated_Invariants admits missing or exact workspace.
func Hash_Positions_Unvalidated_Invariants(
	value Hash_Positions_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), BYTE_COUNT_MINIMUM,
			HASH_POSITIONS_COUNT_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// History_Positions_Unvalidated keeps encoder chains caller-owned.
type History_Positions_Unvalidated []int32

// History_Positions_Unvalidated_Invariants admits missing or exact workspace.
func History_Positions_Unvalidated_Invariants(
	value History_Positions_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), BYTE_COUNT_MINIMUM,
			HISTORY_POSITIONS_COUNT_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Header_Unvalidated separates hostile metadata from wire-safe Header.
type Header_Unvalidated struct {
	// Extra remains untrusted until two-byte wire length fits.
	Extra Extra_Unvalidated
	// Name remains untrusted until UTF-8 and Latin-1 checks pass.
	Name Name_Unvalidated
	// Comment remains untrusted until UTF-8 and Latin-1 checks pass.
	Comment Comment_Unvalidated
	// Modified_Seconds needs no validation because every uint32 is legal.
	Modified_Seconds Modified_Seconds
	// Operating_System needs no validation because every byte is legal.
	Operating_System Operating_System
}

// Header_Unvalidated_Invariants composes caller metadata domains.
func Header_Unvalidated_Invariants(
	value Header_Unvalidated, namespace aver.Namespace,
) {
	Extra_Unvalidated_Invariants(value.Extra, namespace)
	Name_Unvalidated_Invariants(value.Name, namespace)
	Comment_Unvalidated_Invariants(value.Comment, namespace)
	Modified_Seconds_Invariants(value.Modified_Seconds, namespace)
	Operating_System_Invariants(value.Operating_System, namespace)
}

// Header_Storage_Unvalidated separates caller capacity from decoded views.
type Header_Storage_Unvalidated struct {
	// Extra bounds metadata copy without ownership transfer.
	Extra Extra_Destination_Unvalidated
	// Name bounds Latin-1 to UTF-8 expansion.
	Name Name_Destination_Unvalidated
	// Comment bounds Latin-1 to UTF-8 expansion.
	Comment Comment_Destination_Unvalidated
}

// Header_Storage_Unvalidated_Invariants composes caller metadata storage.
func Header_Storage_Unvalidated_Invariants(
	value Header_Storage_Unvalidated, namespace aver.Namespace,
) {
	Extra_Destination_Unvalidated_Invariants(value.Extra, namespace)
	Name_Destination_Unvalidated_Invariants(value.Name, namespace)
	Comment_Destination_Unvalidated_Invariants(value.Comment, namespace)
}

// Header_Storage holds bounded metadata destinations.
type Header_Storage struct {
	// Extra holds the maximum RFC extra field prefix.
	Extra Extra_Destination
	// Name holds the maximum decoded first-member name.
	Name Name_Destination
	// Comment holds the maximum decoded first-member comment.
	Comment Comment_Destination
}

// Header_Storage_Invariants composes bounded metadata destinations.
func Header_Storage_Invariants(
	value Header_Storage, namespace aver.Namespace,
) {
	Extra_Destination_Invariants(value.Extra, namespace)
	Name_Destination_Invariants(value.Name, namespace)
	Comment_Destination_Invariants(value.Comment, namespace)
}

// Header borrows first-member metadata from caller storage.
type Header struct {
	// Extra aliases validated caller destination prefix.
	Extra Extra
	// Name aliases validated caller destination prefix.
	Name Name
	// Comment aliases validated caller destination prefix.
	Comment Comment
	// Modified_Seconds preserves first member timestamp.
	Modified_Seconds Modified_Seconds
	// Operating_System preserves first member platform identifier.
	Operating_System Operating_System
}

// Header_Invariants composes validated first-member metadata.
func Header_Invariants(value Header, namespace aver.Namespace) {
	Extra_Invariants(value.Extra, namespace)
	Name_Invariants(value.Name, namespace)
	Comment_Invariants(value.Comment, namespace)
	Modified_Seconds_Invariants(value.Modified_Seconds, namespace)
	Operating_System_Invariants(value.Operating_System, namespace)
}

// Workspace_Unvalidated prevents encoder scratch from escaping caller ownership.
type Workspace_Unvalidated struct {
	// Heads must supply one slot per DEFLATE hash.
	Heads Hash_Positions_Unvalidated
	// Previous must supply one slot per DEFLATE history position.
	Previous History_Positions_Unvalidated
}

// Workspace_Unvalidated_Invariants composes caller scratch dimensions.
func Workspace_Unvalidated_Invariants(
	value Workspace_Unvalidated, namespace aver.Namespace,
) {
	Hash_Positions_Unvalidated_Invariants(value.Heads, namespace)
	History_Positions_Unvalidated_Invariants(value.Previous, namespace)
}

// Encode_Into writes one member only after all caller state validates.
func Encode_Into(
	destination_unvalidated Destination_Unvalidated,
	workspace Workspace_Unvalidated,
	source_unvalidated Source_Unvalidated,
	header_unvalidated Header_Unvalidated,
	level_unvalidated Level_Unvalidated,
) (count Encoded_Count, status Encode_Status) {
	defer func() {
		Encoded_Count_Invariants(count, "Encode_Into.count")
		Encode_Status_Invariants(status, "Encode_Into.status")
	}()
	Destination_Unvalidated_Invariants(
		destination_unvalidated, "Encode_Into.destination_unvalidated",
	)
	Workspace_Unvalidated_Invariants(workspace, "Encode_Into.workspace")
	Source_Unvalidated_Invariants(
		source_unvalidated, "Encode_Into.source_unvalidated",
	)
	Header_Unvalidated_Invariants(header_unvalidated, "Encode_Into.header_unvalidated")
	Level_Unvalidated_Invariants(level_unvalidated, "Encode_Into.level_unvalidated")
	header_size, header, level, header_status := encode_validate(
		destination_unvalidated, workspace, source_unvalidated,
		header_unvalidated, level_unvalidated,
	)
	if header_status != STATUS_OK {
		return 0, header_status
	}
	destination := Destination(destination_unvalidated)
	if len(destination) == 1 {
		destination[0] = IDENTIFIER_ONE
		return 1, STATUS_OUTPUT_TOO_SMALL
	}
	if len(destination) == 2 {
		destination[0] = IDENTIFIER_ONE
		destination[1] = IDENTIFIER_TWO
		return 2, STATUS_OUTPUT_TOO_SMALL
	}
	if len(destination) < int(header_size)+TRAILER_SIZE {
		return 0, STATUS_OUTPUT_TOO_SMALL
	}
	header_write(
		Header_Destination(destination[:int(header_size)]),
		header, level,
	)
	payload := destination[int(header_size) : len(destination)-TRAILER_SIZE]
	payload_count, flate_status := flate.Encode_Into(
		flate.Destination_Unvalidated(payload),
		flate.Workspace_Unvalidated{
			Heads:    flate.Hash_Positions_Unvalidated(workspace.Heads),
			Previous: flate.History_Positions_Unvalidated(workspace.Previous),
		},
		flate.Source_Unvalidated(source_unvalidated),
		flate.Level_Unvalidated(level),
	)
	if flate_status != flate.STATUS_OK {
		return Encoded_Count(header_size) + Encoded_Count(payload_count),
			STATUS_OUTPUT_TOO_SMALL
	}
	trailer_position := int(header_size) + int(payload_count)
	binary.Put_Uint_32(
		binary.Bytes(destination[trailer_position:trailer_position+4]),
		binary.Word_32(crc32.ChecksumIEEE(source_unvalidated)),
		binary.LITTLE_ENDIAN,
	)
	binary.Put_Uint_32(
		binary.Bytes(destination[trailer_position+4:trailer_position+TRAILER_SIZE]),
		binary.Word_32(len(source_unvalidated)),
		binary.LITTLE_ENDIAN,
	)
	return Encoded_Count(trailer_position + TRAILER_SIZE), STATUS_OK
}

// Decode_Into iterates members so malicious concatenation cannot grow call stack.
func Decode_Into(
	destination_unvalidated Destination_Unvalidated,
	header_storage_unvalidated Header_Storage_Unvalidated,
	compressed_unvalidated Compressed_Unvalidated,
) (count Count, header Header, status Decode_Status) {
	defer func() {
		Count_Invariants(count, "Decode_Into.count")
		Header_Invariants(header, "Decode_Into.header")
		Decode_Status_Invariants(status, "Decode_Into.status")
	}()
	Destination_Unvalidated_Invariants(
		destination_unvalidated, "Decode_Into.destination_unvalidated",
	)
	Header_Storage_Unvalidated_Invariants(
		header_storage_unvalidated, "Decode_Into.header_storage_unvalidated",
	)
	Compressed_Unvalidated_Invariants(
		compressed_unvalidated, "Decode_Into.compressed_unvalidated",
	)
	header_storage, storage_valid := decode_storage_validate(
		destination_unvalidated, header_storage_unvalidated,
		compressed_unvalidated,
	)
	if !storage_valid {
		return 0, Header{}, STATUS_STORAGE_INVALID
	}
	destination := Destination(destination_unvalidated)
	compressed := Compressed(compressed_unvalidated)
	if len(compressed) == 0 {
		return 0, Header{}, STATUS_OK
	}
	position := 0
	for position < len(compressed) {
		capture_header := Capture_Header(position == 0)
		member_count, compressed_count, member_header, member_status :=
			decode_member_unchecked(
				destination[int(count):], header_storage, compressed[position:],
				capture_header,
			)
		count += member_count
		if capture_header {
			header = member_header
		}
		if member_status != STATUS_OK {
			return count, header, Decode_Status(member_status)
		}
		if compressed_count == 0 {
			return count, header, STATUS_INPUT_INVALID
		}
		position += int(compressed_count)
	}
	return count, header, STATUS_OK
}

// Decode_Member_Into exposes exact boundary needed by mixed-format callers.
func Decode_Member_Into(
	destination_unvalidated Destination_Unvalidated,
	header_storage_unvalidated Header_Storage_Unvalidated,
	compressed_unvalidated Compressed_Unvalidated,
) (
	count Count,
	compressed_count Compressed_Count,
	header Header,
	status Decode_Status,
) {
	defer func() {
		Count_Invariants(count, "Decode_Member_Into.count")
		Compressed_Count_Invariants(
			compressed_count, "Decode_Member_Into.compressed_count",
		)
		Header_Invariants(header, "Decode_Member_Into.header")
		Decode_Status_Invariants(status, "Decode_Member_Into.status")
	}()
	Destination_Unvalidated_Invariants(
		destination_unvalidated, "Decode_Member_Into.destination_unvalidated",
	)
	Header_Storage_Unvalidated_Invariants(
		header_storage_unvalidated,
		"Decode_Member_Into.header_storage_unvalidated",
	)
	Compressed_Unvalidated_Invariants(
		compressed_unvalidated, "Decode_Member_Into.compressed_unvalidated",
	)
	header_storage, storage_valid := decode_storage_validate(
		destination_unvalidated, header_storage_unvalidated,
		compressed_unvalidated,
	)
	if !storage_valid {
		return 0, 0, Header{}, STATUS_STORAGE_INVALID
	}
	count, compressed_count, header, member_status := decode_member_unchecked(
		Destination(destination_unvalidated), header_storage,
		Compressed(compressed_unvalidated), Capture_Header(true),
	)
	return count, compressed_count, header, Decode_Status(member_status)
}

func encode_validate(
	destination_unvalidated Destination_Unvalidated,
	workspace Workspace_Unvalidated,
	source_unvalidated Source_Unvalidated,
	header_unvalidated Header_Unvalidated,
	level_unvalidated Level_Unvalidated,
) (
	header_size Header_Size,
	header Header,
	level Level,
	status Encode_Status,
) {
	defer func() {
		Header_Size_Invariants(header_size, "encode_validate.header_size")
		Header_Invariants(header, "encode_validate.header")
		Level_Invariants(level, "encode_validate.level")
		Encode_Status_Invariants(status, "encode_validate.status")
	}()
	Destination_Unvalidated_Invariants(
		destination_unvalidated, "encode_validate.destination_unvalidated",
	)
	Workspace_Unvalidated_Invariants(workspace, "encode_validate.workspace")
	Source_Unvalidated_Invariants(
		source_unvalidated, "encode_validate.source_unvalidated",
	)
	Header_Unvalidated_Invariants(
		header_unvalidated, "encode_validate.header_unvalidated",
	)
	Level_Unvalidated_Invariants(
		level_unvalidated, "encode_validate.level_unvalidated",
	)
	level, level_valid := level_validate(level_unvalidated)
	if !level_valid {
		return HEADER_SIZE, Header{}, level, STATUS_LEVEL_INVALID
	}
	if len(workspace.Heads) != HASH_POSITIONS_COUNT {
		return HEADER_SIZE, Header{}, level, STATUS_STORAGE_INVALID
	}
	if len(workspace.Previous) != HISTORY_POSITIONS_COUNT {
		return HEADER_SIZE, Header{}, level, STATUS_STORAGE_INVALID
	}
	if len(destination_unvalidated) > BYTE_COUNT_MAXIMUM {
		return HEADER_SIZE, Header{}, level, STATUS_STORAGE_INVALID
	}
	if len(source_unvalidated) > BYTE_COUNT_MAXIMUM {
		return HEADER_SIZE, Header{}, level, STATUS_STORAGE_INVALID
	}
	header_size, header, header_status := header_validate(header_unvalidated)
	if header_status != STATUS_OK {
		return HEADER_SIZE, Header{}, level, Encode_Status(header_status)
	}
	destination := Destination(destination_unvalidated)
	if byte_slices_overlap(Bytes(destination), Bytes(source_unvalidated)) {
		return HEADER_SIZE, Header{}, level, STATUS_STORAGE_INVALID
	}
	if byte_slices_overlap(Bytes(destination), Bytes(header.Extra)) {
		return HEADER_SIZE, Header{}, level, STATUS_STORAGE_INVALID
	}
	if byte_slices_overlap(Bytes(destination), Bytes(header.Name)) {
		return HEADER_SIZE, Header{}, level, STATUS_STORAGE_INVALID
	}
	if byte_slices_overlap(Bytes(destination), Bytes(header.Comment)) {
		return HEADER_SIZE, Header{}, level, STATUS_STORAGE_INVALID
	}
	return header_size, header, level, STATUS_OK
}

func level_validate(
	value Level_Unvalidated,
) (level Level, valid Boolean) {
	defer func() {
		Level_Invariants(level, "level_validate.level")
		Boolean_Invariants(valid, "level_validate.valid")
	}()
	Level_Unvalidated_Invariants(value, "level_validate.value")
	if value == HUFFMAN_ONLY {
		return Level(value), true
	}
	if value == DEFAULT_COMPRESSION {
		return Level(value), true
	}
	if value < NO_COMPRESSION {
		return Level(NO_COMPRESSION), false
	}
	if value > BEST_COMPRESSION {
		return Level(NO_COMPRESSION), false
	}
	return Level(value), true
}

func decode_storage_validate(
	destination Destination_Unvalidated,
	storage_unvalidated Header_Storage_Unvalidated,
	compressed Compressed_Unvalidated,
) (storage Header_Storage, valid Boolean) {
	defer func() {
		Header_Storage_Invariants(storage, "decode_storage_validate.storage")
		Boolean_Invariants(valid, "decode_storage_validate.valid")
	}()
	Destination_Unvalidated_Invariants(
		destination, "decode_storage_validate.destination",
	)
	Header_Storage_Unvalidated_Invariants(
		storage_unvalidated, "decode_storage_validate.storage_unvalidated",
	)
	Compressed_Unvalidated_Invariants(
		compressed, "decode_storage_validate.compressed",
	)
	if len(destination) > BYTE_COUNT_MAXIMUM {
		return Header_Storage{}, false
	}
	if len(compressed) > BYTE_COUNT_MAXIMUM {
		return Header_Storage{}, false
	}
	if len(storage_unvalidated.Extra) > BYTE_COUNT_MAXIMUM {
		return Header_Storage{}, false
	}
	if len(storage_unvalidated.Name) > BYTE_COUNT_MAXIMUM {
		return Header_Storage{}, false
	}
	if len(storage_unvalidated.Comment) > BYTE_COUNT_MAXIMUM {
		return Header_Storage{}, false
	}
	extra_size := len(storage_unvalidated.Extra)
	if extra_size > EXTRA_SIZE_MAXIMUM {
		extra_size = EXTRA_SIZE_MAXIMUM
	}
	name_size := len(storage_unvalidated.Name)
	if name_size > HEADER_TEXT_SIZE_MAXIMUM {
		name_size = HEADER_TEXT_SIZE_MAXIMUM
	}
	comment_size := len(storage_unvalidated.Comment)
	if comment_size > HEADER_TEXT_SIZE_MAXIMUM {
		comment_size = HEADER_TEXT_SIZE_MAXIMUM
	}
	storage = Header_Storage{
		Extra: Extra_Destination(storage_unvalidated.Extra[:extra_size]),
		Name:  Name_Destination(storage_unvalidated.Name[:name_size]),
		Comment: Comment_Destination(
			storage_unvalidated.Comment[:comment_size],
		),
	}
	if !decode_storage_separate(Bytes(destination), storage, Bytes(compressed)) {
		return Header_Storage{}, false
	}
	return storage, true
}

func decode_storage_separate(
	destination Bytes,
	storage Header_Storage,
	compressed Bytes,
) (separate Boolean) {
	defer func() {
		Boolean_Invariants(separate, "decode_storage_separate.separate")
	}()
	Bytes_Invariants(destination, "decode_storage_separate.destination")
	Header_Storage_Invariants(storage, "decode_storage_separate.storage")
	Bytes_Invariants(compressed, "decode_storage_separate.compressed")
	if byte_slices_overlap(destination, compressed) {
		return false
	}
	if byte_slices_overlap(destination, Bytes(storage.Extra)) {
		return false
	}
	if byte_slices_overlap(destination, Bytes(storage.Name)) {
		return false
	}
	if byte_slices_overlap(destination, Bytes(storage.Comment)) {
		return false
	}
	if byte_slices_overlap(Bytes(storage.Extra), compressed) {
		return false
	}
	if byte_slices_overlap(Bytes(storage.Name), compressed) {
		return false
	}
	if byte_slices_overlap(Bytes(storage.Comment), compressed) {
		return false
	}
	if byte_slices_overlap(Bytes(storage.Extra), Bytes(storage.Name)) {
		return false
	}
	if byte_slices_overlap(Bytes(storage.Extra), Bytes(storage.Comment)) {
		return false
	}
	if byte_slices_overlap(Bytes(storage.Name), Bytes(storage.Comment)) {
		return false
	}
	return true
}

// Pointer equality rejects aliases without unsafe address arithmetic.
func byte_slices_overlap(first Bytes, second Bytes) (overlap Boolean) {
	defer func() { Boolean_Invariants(overlap, "byte_slices_overlap.overlap") }()
	Bytes_Invariants(first, "byte_slices_overlap.first")
	Bytes_Invariants(second, "byte_slices_overlap.second")
	if len(first) == 0 {
		return false
	}
	if len(second) == 0 {
		return false
	}
	for index := range first {
		if &first[index] == &second[0] {
			return true
		}
	}
	for index := range second {
		if &second[index] == &first[0] {
			return true
		}
	}
	return false
}

func header_validate(
	header_unvalidated Header_Unvalidated,
) (
	size Header_Size,
	header Header,
	status Header_Validation_Status,
) {
	defer func() {
		Header_Size_Invariants(size, "header_validate.size")
		Header_Invariants(header, "header_validate.header")
		Header_Validation_Status_Invariants(
			status, "header_validate.status",
		)
	}()
	Header_Unvalidated_Invariants(
		header_unvalidated, "header_validate.header_unvalidated",
	)
	if len(header_unvalidated.Extra) > EXTRA_SIZE_MAXIMUM {
		return HEADER_SIZE, Header{}, STATUS_HEADER_INVALID
	}
	if len(header_unvalidated.Name) > HEADER_TEXT_SIZE_MAXIMUM {
		return HEADER_SIZE, Header{}, STATUS_HEADER_INVALID
	}
	if len(header_unvalidated.Comment) > HEADER_TEXT_SIZE_MAXIMUM {
		return HEADER_SIZE, Header{}, STATUS_HEADER_INVALID
	}
	name_size, name_status := header_text_wire_size(
		Header_Text_Source(header_unvalidated.Name),
	)
	if name_status != STATUS_OK {
		return HEADER_SIZE, Header{}, name_status
	}
	comment_size, comment_status := header_text_wire_size(
		Header_Text_Source(header_unvalidated.Comment),
	)
	if comment_status != STATUS_OK {
		return HEADER_SIZE, Header{}, comment_status
	}
	header = Header{
		Extra:            Extra(header_unvalidated.Extra),
		Name:             Name(header_unvalidated.Name),
		Comment:          Comment(header_unvalidated.Comment),
		Modified_Seconds: header_unvalidated.Modified_Seconds,
		Operating_System: header_unvalidated.Operating_System,
	}
	size = HEADER_SIZE
	if header.Extra != nil {
		size += Header_Size(EXTRA_SIZE_SIZE + len(header.Extra))
	}
	if len(header.Name) != 0 {
		size += Header_Size(name_size + 1)
	}
	if len(header.Comment) != 0 {
		size += Header_Size(comment_size + 1)
	}
	return size, header, STATUS_OK
}

func header_text_wire_size(
	source Header_Text_Source,
) (
	wire_size Header_String_Byte_Count,
	status Header_Validation_Status,
) {
	defer func() {
		Header_String_Byte_Count_Invariants(
			wire_size, "header_text_wire_size.wire_size",
		)
		Header_Validation_Status_Invariants(
			status, "header_text_wire_size.status",
		)
	}()
	Header_Text_Source_Invariants(source, "header_text_wire_size.source")
	if len(source) > HEADER_TEXT_SIZE_MAXIMUM {
		return 0, STATUS_HEADER_INVALID
	}
	if !utf8.Valid(utf8.Bytes(source)) {
		return 0, STATUS_INPUT_INVALID
	}
	for position := 0; position < len(source); {
		character, size := utf8.Decode_Character(utf8.Bytes(source[position:]))
		if character == 0 {
			return 0, STATUS_INPUT_INVALID
		}
		if character > 0xff {
			return 0, STATUS_INPUT_INVALID
		}
		position += int(size)
		wire_size++
		if wire_size > HEADER_STRING_BYTE_COUNT_MAXIMUM {
			return 0, STATUS_HEADER_INVALID
		}
	}
	return wire_size, STATUS_OK
}

func header_write(
	destination Header_Destination,
	header Header,
	level Level,
) {
	Header_Destination_Invariants(destination, "header_write.destination")
	Header_Invariants(header, "header_write.header")
	Level_Invariants(level, "header_write.level")
	destination[0] = IDENTIFIER_ONE
	destination[1] = IDENTIFIER_TWO
	destination[2] = DEFLATE_METHOD
	flags := byte(0)
	if header.Extra != nil {
		flags |= FLAG_EXTRA
	}
	if len(header.Name) != 0 {
		flags |= FLAG_NAME
	}
	if len(header.Comment) != 0 {
		flags |= FLAG_COMMENT
	}
	destination[3] = flags
	binary.Put_Uint_32(
		binary.Bytes(destination[4:8]), binary.Word_32(header.Modified_Seconds),
		binary.LITTLE_ENDIAN,
	)
	destination[8] = byte(extra_flags(level))
	destination[9] = byte(header.Operating_System)
	position := Header_Name_Position(HEADER_SIZE)
	if header.Extra != nil {
		binary.Put_Uint_16(
			binary.Bytes(destination[int(position):int(position)+EXTRA_SIZE_SIZE]),
			binary.Word_16(len(header.Extra)), binary.LITTLE_ENDIAN,
		)
		position += EXTRA_SIZE_SIZE
		position += Header_Name_Position(
			copy(destination[int(position):], header.Extra),
		)
	}
	if len(header.Name) != 0 {
		position = Header_Name_Position(header_text_write(
			Header_Text_Wire_Destination(destination), position,
			Header_Text_Non_Empty_Source(header.Name),
		))
	}
	if len(header.Comment) != 0 {
		header_text_write(
			Header_Text_Wire_Destination(destination), position,
			Header_Text_Non_Empty_Source(header.Comment),
		)
	}
}

func extra_flags(level Level) (flags Extra_Flags) {
	defer func() { Extra_Flags_Invariants(flags, "extra_flags.flags") }()
	Level_Invariants(level, "extra_flags.level")
	if level == BEST_COMPRESSION {
		return EXTRA_FLAGS_BEST
	}
	if level == BEST_SPEED {
		return EXTRA_FLAGS_FAST
	}
	return EXTRA_FLAGS_DEFAULT
}

func header_text_write(
	destination Header_Text_Wire_Destination,
	position Header_Name_Position,
	source Header_Text_Non_Empty_Source,
) (next Header_Text_End_Position) {
	defer func() {
		Header_Text_End_Position_Invariants(next, "header_text_write.next")
	}()
	Header_Text_Wire_Destination_Invariants(
		destination, "header_text_write.destination",
	)
	Header_Name_Position_Invariants(position, "header_text_write.position")
	Header_Text_Non_Empty_Source_Invariants(
		source, "header_text_write.source",
	)
	next = Header_Text_End_Position(position)
	for source_position := 0; source_position < len(source); {
		character, size := utf8.Decode_Character(utf8.Bytes(source[source_position:]))
		destination[next] = byte(character)
		next++
		source_position += int(size)
	}
	destination[next] = 0
	return next + 1
}

func decode_member_unchecked(
	destination Destination,
	header_storage Header_Storage,
	compressed Compressed,
	capture_header Capture_Header,
) (
	count Count,
	compressed_count Compressed_Count,
	header Header,
	status Member_Status,
) {
	defer func() {
		Count_Invariants(count, "decode_member_unchecked.count")
		Compressed_Count_Invariants(
			compressed_count, "decode_member_unchecked.compressed_count",
		)
		Header_Invariants(header, "decode_member_unchecked.header")
		Member_Status_Invariants(status, "decode_member_unchecked.status")
	}()
	Destination_Invariants(destination, "decode_member_unchecked.destination")
	Header_Storage_Invariants(
		header_storage, "decode_member_unchecked.header_storage",
	)
	Compressed_Invariants(compressed, "decode_member_unchecked.compressed")
	Capture_Header_Invariants(
		capture_header, "decode_member_unchecked.capture_header",
	)
	header_end, header, header_status := header_parse(
		compressed, header_storage, capture_header,
	)
	if header_status != STATUS_OK {
		return 0, Compressed_Count(len(compressed)), header,
			Member_Status(header_status)
	}
	flate_count, flate_compressed_count, flate_status := flate.Decode_Prefix_Into(
		flate.Destination_Unvalidated(destination),
		flate.Compressed_Unvalidated(compressed[int(header_end):]),
	)
	count = Count(flate_count)
	payload_count := Count(flate_compressed_count)
	if flate_status != flate.STATUS_OK {
		status = STATUS_INPUT_INVALID
		if flate_status == flate.STATUS_OUTPUT_TOO_SMALL {
			status = STATUS_OUTPUT_TOO_SMALL
		}
		return Count(count),
			Compressed_Count(header_end) + Compressed_Count(payload_count),
			header, status
	}
	trailer_position := Count(header_end) + payload_count
	if len(compressed)-int(trailer_position) < TRAILER_SIZE {
		return Count(count), Compressed_Count(trailer_position),
			header, STATUS_INPUT_INVALID
	}
	compressed_count = Compressed_Count(trailer_position + TRAILER_SIZE)
	if !trailer_valid(
		Trailer_Source(compressed[int(trailer_position):]),
		Destination(destination[:count]),
	) {
		return Count(count), compressed_count, header, STATUS_CHECKSUM_INVALID
	}
	return Count(count), compressed_count, header, STATUS_OK
}

func header_parse(
	compressed Compressed,
	storage Header_Storage,
	capture Capture_Header,
) (position Header_Position, header Header, status Header_Status) {
	defer func() {
		Header_Position_Invariants(position, "header_parse.position")
		Header_Invariants(header, "header_parse.header")
		Header_Status_Invariants(status, "header_parse.status")
	}()
	Compressed_Invariants(compressed, "header_parse.compressed")
	Header_Storage_Invariants(storage, "header_parse.storage")
	Capture_Header_Invariants(capture, "header_parse.capture")
	if len(compressed) < HEADER_SIZE {
		return HEADER_SIZE, Header{}, STATUS_HEADER_INVALID
	}
	if compressed[0] != IDENTIFIER_ONE {
		return HEADER_SIZE, Header{}, STATUS_HEADER_INVALID
	}
	if compressed[1] != IDENTIFIER_TWO {
		return HEADER_SIZE, Header{}, STATUS_HEADER_INVALID
	}
	if compressed[2] != DEFLATE_METHOD {
		return HEADER_SIZE, Header{}, STATUS_HEADER_INVALID
	}
	member := Member(compressed)
	flags := Header_Flags(member[3])
	header.Modified_Seconds = Modified_Seconds(binary.Uint_32(
		binary.Bytes(member[4:8]), binary.LITTLE_ENDIAN,
	))
	header.Operating_System = Operating_System(member[9])
	extra_position, extra, status := header_extra_parse(
		member, storage.Extra, flags, capture,
	)
	header.Extra = extra
	if status != STATUS_OK {
		return Header_Position(extra_position), header, status
	}
	name_position, name, status := header_name_parse(
		member, storage.Name, extra_position, flags, capture,
	)
	header.Name = name
	if status != STATUS_OK {
		return Header_Position(name_position), header, status
	}
	comment_position, comment, status := header_comment_parse(
		member, storage.Comment, name_position, flags, capture,
	)
	header.Comment = comment
	if status != STATUS_OK {
		return Header_Position(comment_position), header, status
	}
	position, validation_status := header_checksum_parse(
		member, comment_position, flags,
	)
	if validation_status != STATUS_OK {
		return position, header, STATUS_HEADER_INVALID
	}
	return position, header, STATUS_OK
}

func header_extra_parse(
	compressed Member,
	destination Extra_Destination,
	flags Header_Flags,
	capture Capture_Header,
) (next Header_Extra_Position, extra Extra, status Header_Status) {
	defer func() {
		Header_Extra_Position_Invariants(next, "header_extra_parse.next")
		Extra_Invariants(extra, "header_extra_parse.extra")
		Header_Status_Invariants(status, "header_extra_parse.status")
	}()
	Member_Invariants(compressed, "header_extra_parse.compressed")
	Extra_Destination_Invariants(
		destination, "header_extra_parse.destination",
	)
	Header_Flags_Invariants(flags, "header_extra_parse.flags")
	Capture_Header_Invariants(capture, "header_extra_parse.capture")
	next = HEADER_SIZE
	if flags&FLAG_EXTRA == 0 {
		return next, nil, STATUS_OK
	}
	if len(compressed)-int(next) < EXTRA_SIZE_SIZE {
		return next, nil, STATUS_HEADER_INVALID
	}
	size := int(binary.Uint_16(
		binary.Bytes(compressed[int(next):int(next)+EXTRA_SIZE_SIZE]),
		binary.LITTLE_ENDIAN,
	))
	next += EXTRA_SIZE_SIZE
	if len(compressed)-int(next) < size {
		return next, nil, STATUS_HEADER_INVALID
	}
	if capture {
		if len(destination) < size {
			return next, nil, STATUS_HEADER_STORAGE_TOO_SMALL
		}
		copy(
			destination[:size],
			compressed[int(next):int(next)+size],
		)
		extra = Extra(destination[:size])
	}
	return next + Header_Extra_Position(size), extra, STATUS_OK
}

func header_name_parse(
	compressed Member,
	destination Name_Destination,
	position Header_Extra_Position,
	flags Header_Flags,
	capture Capture_Header,
) (next Header_Name_Position, name Name, status Header_Status) {
	defer func() {
		Header_Name_Position_Invariants(next, "header_name_parse.next")
		Name_Invariants(name, "header_name_parse.name")
		Header_Status_Invariants(status, "header_name_parse.status")
	}()
	Member_Invariants(compressed, "header_name_parse.compressed")
	Name_Destination_Invariants(
		destination, "header_name_parse.destination",
	)
	Header_Extra_Position_Invariants(position, "header_name_parse.position")
	Header_Flags_Invariants(flags, "header_name_parse.flags")
	Capture_Header_Invariants(capture, "header_name_parse.capture")
	if flags&FLAG_NAME == 0 {
		return Header_Name_Position(position), nil, STATUS_OK
	}
	text_end, text_size, text_status := header_text_bounds(
		compressed, Header_Name_Position(position),
	)
	if text_status != STATUS_OK {
		return Header_Name_Position(position), nil, STATUS_HEADER_INVALID
	}
	if !capture {
		return Header_Name_Position(text_end), nil, STATUS_OK
	}
	if len(destination) < int(text_size) {
		return Header_Name_Position(position), nil, STATUS_HEADER_STORAGE_TOO_SMALL
	}
	header_text_decode(
		Header_Text_Destination(destination[:int(text_size)]),
		Header_String_Source(compressed[int(position):int(text_end)-1]),
	)
	return Header_Name_Position(text_end), Name(destination[:int(text_size)]), STATUS_OK
}

func header_comment_parse(
	compressed Member,
	destination Comment_Destination,
	position Header_Name_Position,
	flags Header_Flags,
	capture Capture_Header,
) (next Header_Comment_Position, comment Comment, status Header_Status) {
	defer func() {
		Header_Comment_Position_Invariants(next, "header_comment_parse.next")
		Comment_Invariants(comment, "header_comment_parse.comment")
		Header_Status_Invariants(status, "header_comment_parse.status")
	}()
	Member_Invariants(compressed, "header_comment_parse.compressed")
	Comment_Destination_Invariants(
		destination, "header_comment_parse.destination",
	)
	Header_Name_Position_Invariants(position, "header_comment_parse.position")
	Header_Flags_Invariants(flags, "header_comment_parse.flags")
	Capture_Header_Invariants(capture, "header_comment_parse.capture")
	if flags&FLAG_COMMENT == 0 {
		return Header_Comment_Position(position), nil, STATUS_OK
	}
	next, text_size, text_status := header_text_bounds(compressed, position)
	if text_status != STATUS_OK {
		return Header_Comment_Position(position), nil, STATUS_HEADER_INVALID
	}
	if !capture {
		return next, nil, STATUS_OK
	}
	if len(destination) < int(text_size) {
		return Header_Comment_Position(position), nil, STATUS_HEADER_STORAGE_TOO_SMALL
	}
	header_text_decode(
		Header_Text_Destination(destination[:int(text_size)]),
		Header_String_Source(compressed[int(position):int(next)-1]),
	)
	return next, Comment(destination[:int(text_size)]), STATUS_OK
}

func header_text_bounds(
	compressed Member, position Header_Name_Position,
) (
	next Header_Comment_Position,
	text_size Header_Text_Size,
	status Header_Validation_Status,
) {
	defer func() {
		Header_Comment_Position_Invariants(next, "header_text_bounds.next")
		Header_Text_Size_Invariants(text_size, "header_text_bounds.text_size")
		Header_Validation_Status_Invariants(
			status, "header_text_bounds.status",
		)
	}()
	Member_Invariants(compressed, "header_text_bounds.compressed")
	Header_Name_Position_Invariants(position, "header_text_bounds.position")
	maximum := Header_String_Byte_Count(HEADER_STRING_BYTE_COUNT_MAXIMUM)
	for wire_size := Header_String_Byte_Count(0); wire_size <= maximum; wire_size++ {
		value_position := Header_Comment_Position(position) +
			Header_Comment_Position(wire_size)
		if int(value_position) >= len(compressed) {
			return Header_Comment_Position(position), 0, STATUS_INPUT_INVALID
		}
		value := compressed[value_position]
		if value == 0 {
			return value_position + 1, text_size, STATUS_OK
		}
		if value < byte(utf8.CHARACTER_SELF) {
			text_size++
		} else {
			text_size += 2
		}
	}
	return Header_Comment_Position(position), 0, STATUS_HEADER_INVALID
}

func header_text_decode(
	destination Header_Text_Destination, source Header_String_Source,
) {
	Header_Text_Destination_Invariants(destination, "header_text_decode.destination")
	Header_String_Source_Invariants(source, "header_text_decode.source")
	position := 0
	for _, value := range source {
		if value < byte(utf8.CHARACTER_SELF) {
			destination[position] = value
			position++
			continue
		}
		destination[position] = 0xc0 | value>>6
		destination[position+1] = 0x80 | value&0x3f
		position += 2
	}
}

func header_checksum_parse(
	compressed Member, position Header_Comment_Position, flags Header_Flags,
) (next Header_Position, status Header_Validation_Status) {
	defer func() {
		Header_Position_Invariants(next, "header_checksum_parse.next")
		Header_Validation_Status_Invariants(
			status, "header_checksum_parse.status",
		)
	}()
	Member_Invariants(compressed, "header_checksum_parse.compressed")
	Header_Comment_Position_Invariants(
		position, "header_checksum_parse.position",
	)
	Header_Flags_Invariants(flags, "header_checksum_parse.flags")
	if flags&FLAG_HEADER_CHECKSUM == 0 {
		return Header_Position(position), STATUS_OK
	}
	if len(compressed)-int(position) < HEADER_CHECKSUM_SIZE {
		return Header_Position(position), STATUS_INPUT_INVALID
	}
	want := uint16(binary.Uint_16(
		binary.Bytes(
			compressed[int(position):int(position)+HEADER_CHECKSUM_SIZE],
		),
		binary.LITTLE_ENDIAN,
	))
	observed := uint16(crc32.ChecksumIEEE(compressed[:int(position)]))
	if observed != want {
		return Header_Position(position), STATUS_HEADER_INVALID
	}
	return Header_Position(position) + HEADER_CHECKSUM_SIZE, STATUS_OK
}

func trailer_valid(
	compressed Trailer_Source, decoded Destination,
) (valid Boolean) {
	defer func() { Boolean_Invariants(valid, "trailer_valid.valid") }()
	Trailer_Source_Invariants(compressed, "trailer_valid.compressed")
	Destination_Invariants(decoded, "trailer_valid.decoded")
	want_checksum := uint32(binary.Uint_32(
		binary.Bytes(compressed[:4]), binary.LITTLE_ENDIAN,
	))
	want_size := uint32(binary.Uint_32(
		binary.Bytes(compressed[4:8]), binary.LITTLE_ENDIAN,
	))
	if crc32.ChecksumIEEE(decoded) != want_checksum {
		return false
	}
	return uint32(len(decoded)) == want_size
}
