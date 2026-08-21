// Package tar reads and writes bounded TAR archives in caller-owned storage.
package tar

import (
	"unsafe"

	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/aver/default"
	"local/james-orcales/shared/sim/nbio"
	"local/james-orcales/shared/sim/time"
	"local/james-orcales/shared/strconv"
	"local/james-orcales/shared/unicode/utf8"
)

// NAME_FIELD_SIZE is V7 path field width.
const NAME_FIELD_SIZE = 100

// MODE_FIELD_SIZE is V7 mode field width.
const MODE_FIELD_SIZE = 8

// USER_IDENTIFIER_FIELD_SIZE is V7 owner identifier field width.
const USER_IDENTIFIER_FIELD_SIZE = MODE_FIELD_SIZE

// GROUP_IDENTIFIER_FIELD_SIZE is V7 group identifier field width.
const GROUP_IDENTIFIER_FIELD_SIZE = USER_IDENTIFIER_FIELD_SIZE

// ENTRY_SIZE_FIELD_SIZE is V7 content-size field width.
const ENTRY_SIZE_FIELD_SIZE = 12

// TIMESTAMP_FIELD_SIZE is V7 timestamp field width.
const TIMESTAMP_FIELD_SIZE = ENTRY_SIZE_FIELD_SIZE

// CHECKSUM_FIELD_SIZE is V7 checksum field width.
const CHECKSUM_FIELD_SIZE = MODE_FIELD_SIZE

// NUMERIC_FIELD_SIZE_MINIMUM admits the empty rejected parser boundary.
const NUMERIC_FIELD_SIZE_MINIMUM = 0

// NUMERIC_FIELD_SIZE_MAXIMUM follows the widest TAR integer field.
const NUMERIC_FIELD_SIZE_MAXIMUM = ENTRY_SIZE_FIELD_SIZE

// TYPE_FLAG_FIELD_SIZE is V7 type field width.
const TYPE_FLAG_FIELD_SIZE = 1

// LINK_NAME_FIELD_SIZE is V7 link target field width.
const LINK_NAME_FIELD_SIZE = NAME_FIELD_SIZE

// MAGIC_FIELD_SIZE is USTAR format marker width.
const MAGIC_FIELD_SIZE = 6

// VERSION_FIELD_SIZE is USTAR version field width.
const VERSION_FIELD_SIZE = 2

// USER_NAME_FIELD_SIZE is USTAR owner name field width.
const USER_NAME_FIELD_SIZE = 32

// GROUP_NAME_FIELD_SIZE is USTAR group name field width.
const GROUP_NAME_FIELD_SIZE = USER_NAME_FIELD_SIZE

// DEVICE_MAJOR_FIELD_SIZE is USTAR device-major field width.
const DEVICE_MAJOR_FIELD_SIZE = MODE_FIELD_SIZE

// DEVICE_MINOR_FIELD_SIZE is USTAR device-minor field width.
const DEVICE_MINOR_FIELD_SIZE = DEVICE_MAJOR_FIELD_SIZE

// PREFIX_FIELD_SIZE is USTAR path prefix width.
const PREFIX_FIELD_SIZE = 155

// HEADER_PADDING_FIELD_SIZE is unused USTAR trailer width.
const HEADER_PADDING_FIELD_SIZE = 12

// BLOCK_SIZE is sum of all USTAR wire fields.
const BLOCK_SIZE = NAME_FIELD_SIZE + MODE_FIELD_SIZE +
	USER_IDENTIFIER_FIELD_SIZE + GROUP_IDENTIFIER_FIELD_SIZE +
	ENTRY_SIZE_FIELD_SIZE + TIMESTAMP_FIELD_SIZE + CHECKSUM_FIELD_SIZE +
	TYPE_FLAG_FIELD_SIZE + LINK_NAME_FIELD_SIZE + MAGIC_FIELD_SIZE +
	VERSION_FIELD_SIZE + USER_NAME_FIELD_SIZE + GROUP_NAME_FIELD_SIZE +
	DEVICE_MAJOR_FIELD_SIZE + DEVICE_MINOR_FIELD_SIZE + PREFIX_FIELD_SIZE +
	HEADER_PADDING_FIELD_SIZE

// NAME_FIELD_OFFSET starts V7 header layout.
const NAME_FIELD_OFFSET = 0

// MODE_FIELD_OFFSET follows path field.
const MODE_FIELD_OFFSET = NAME_FIELD_OFFSET + NAME_FIELD_SIZE

// USER_IDENTIFIER_FIELD_OFFSET follows mode field.
const USER_IDENTIFIER_FIELD_OFFSET = MODE_FIELD_OFFSET + MODE_FIELD_SIZE

// GROUP_IDENTIFIER_FIELD_OFFSET follows owner identifier field.
const GROUP_IDENTIFIER_FIELD_OFFSET = USER_IDENTIFIER_FIELD_OFFSET + USER_IDENTIFIER_FIELD_SIZE

// ENTRY_SIZE_FIELD_OFFSET follows group identifier field.
const ENTRY_SIZE_FIELD_OFFSET = GROUP_IDENTIFIER_FIELD_OFFSET + GROUP_IDENTIFIER_FIELD_SIZE

// TIMESTAMP_FIELD_OFFSET follows content-size field.
const TIMESTAMP_FIELD_OFFSET = ENTRY_SIZE_FIELD_OFFSET + ENTRY_SIZE_FIELD_SIZE

// CHECKSUM_FIELD_OFFSET follows timestamp field.
const CHECKSUM_FIELD_OFFSET = TIMESTAMP_FIELD_OFFSET + TIMESTAMP_FIELD_SIZE

// TYPE_FLAG_FIELD_OFFSET follows checksum field.
const TYPE_FLAG_FIELD_OFFSET = CHECKSUM_FIELD_OFFSET + CHECKSUM_FIELD_SIZE

// LINK_NAME_FIELD_OFFSET follows type field.
const LINK_NAME_FIELD_OFFSET = TYPE_FLAG_FIELD_OFFSET + TYPE_FLAG_FIELD_SIZE

// MAGIC_FIELD_OFFSET follows link target field.
const MAGIC_FIELD_OFFSET = LINK_NAME_FIELD_OFFSET + LINK_NAME_FIELD_SIZE

// VERSION_FIELD_OFFSET follows format marker.
const VERSION_FIELD_OFFSET = MAGIC_FIELD_OFFSET + MAGIC_FIELD_SIZE

// USER_NAME_FIELD_OFFSET follows format version.
const USER_NAME_FIELD_OFFSET = VERSION_FIELD_OFFSET + VERSION_FIELD_SIZE

// GROUP_NAME_FIELD_OFFSET follows owner name field.
const GROUP_NAME_FIELD_OFFSET = USER_NAME_FIELD_OFFSET + USER_NAME_FIELD_SIZE

// DEVICE_MAJOR_FIELD_OFFSET follows group name field.
const DEVICE_MAJOR_FIELD_OFFSET = GROUP_NAME_FIELD_OFFSET + GROUP_NAME_FIELD_SIZE

// DEVICE_MINOR_FIELD_OFFSET follows device-major field.
const DEVICE_MINOR_FIELD_OFFSET = DEVICE_MAJOR_FIELD_OFFSET + DEVICE_MAJOR_FIELD_SIZE

// PREFIX_FIELD_OFFSET follows device-minor field.
const PREFIX_FIELD_OFFSET = DEVICE_MINOR_FIELD_OFFSET + DEVICE_MINOR_FIELD_SIZE

// HEADER_PADDING_FIELD_OFFSET follows USTAR prefix field.
const HEADER_PADDING_FIELD_OFFSET = PREFIX_FIELD_OFFSET + PREFIX_FIELD_SIZE

// STAR_TIMESTAMP_FIELD_COUNT is access and metadata-change time.
const STAR_TIMESTAMP_FIELD_COUNT = 2

// STAR_PREFIX_FIELD_SIZE leaves STAR timestamps in USTAR prefix storage.
const STAR_PREFIX_FIELD_SIZE = PREFIX_FIELD_SIZE -
	STAR_TIMESTAMP_FIELD_COUNT*TIMESTAMP_FIELD_SIZE

// STAR_ACCESS_TIMESTAMP_FIELD_OFFSET follows STAR prefix.
const STAR_ACCESS_TIMESTAMP_FIELD_OFFSET = PREFIX_FIELD_OFFSET + STAR_PREFIX_FIELD_SIZE

// STAR_CHANGE_TIMESTAMP_FIELD_OFFSET follows STAR access timestamp.
const STAR_CHANGE_TIMESTAMP_FIELD_OFFSET = STAR_ACCESS_TIMESTAMP_FIELD_OFFSET + TIMESTAMP_FIELD_SIZE

// STAR_TRAILER_VERSION_FIELD_COUNT is the number of version-width trailer pieces.
const STAR_TRAILER_VERSION_FIELD_COUNT = 2

// STAR_TRAILER_FIELD_SIZE derives the STAR trailer width.
const STAR_TRAILER_FIELD_SIZE = STAR_TRAILER_VERSION_FIELD_COUNT * VERSION_FIELD_SIZE

// STAR_TRAILER_FIELD_OFFSET ends STAR trailer at block boundary.
const STAR_TRAILER_FIELD_OFFSET = BLOCK_SIZE - STAR_TRAILER_FIELD_SIZE

// GNU_ACCESS_TIMESTAMP_FIELD_OFFSET reuses USTAR prefix position.
const GNU_ACCESS_TIMESTAMP_FIELD_OFFSET = PREFIX_FIELD_OFFSET

// GNU_CHANGE_TIMESTAMP_FIELD_OFFSET follows GNU access timestamp.
const GNU_CHANGE_TIMESTAMP_FIELD_OFFSET = GNU_ACCESS_TIMESTAMP_FIELD_OFFSET + TIMESTAMP_FIELD_SIZE

// USTAR_PATH_SIZE_MAXIMUM includes prefix separator and direct path.
const USTAR_PATH_SIZE_MAXIMUM = PREFIX_FIELD_SIZE + TYPE_FLAG_FIELD_SIZE + NAME_FIELD_SIZE

// USTAR_PATH_SIZE_MINIMUM excludes the empty path rejected before wire staging.
const USTAR_PATH_SIZE_MINIMUM = TYPE_FLAG_FIELD_SIZE

// OCTAL_BASE is TAR text-number radix.
const OCTAL_BASE = 8

// DECIMAL_BASE is PAX text-number radix.
const DECIMAL_BASE = 10

// OCTAL_BITS_PER_DIGIT is binary payload in one octal digit.
const OCTAL_BITS_PER_DIGIT = 3

// BASE_256_MARKER_MASK identifies GNU binary numeric fields.
const BASE_256_MARKER_MASK byte = 1 << (bits.BIT_COUNT_8_MAXIMUM - 1)

// BASE_256_SIGN_MASK identifies negative GNU binary numbers.
const BASE_256_SIGN_MASK byte = BASE_256_MARKER_MASK >> 1

// BASE_256_FIRST_DATA_MASK removes marker bit.
const BASE_256_FIRST_DATA_MASK byte = BASE_256_MARKER_MASK - 1

// INTEGER_SIGN_SHIFT locates signed 64-bit sign.
const INTEGER_SIGN_SHIFT = bits.BIT_COUNT_64_MAXIMUM - 1

// INTEGER_BYTE_SHIFT consumes one byte from 64-bit workspace.
const INTEGER_BYTE_SHIFT = bits.BIT_COUNT_64_MAXIMUM - bits.BIT_COUNT_8_MAXIMUM

// PAX_COUNT_SEPARATOR_SIZE counts space after decimal record count.
const PAX_COUNT_SEPARATOR_SIZE = len(" ")

// PAX_KEY_SEPARATOR_SIZE counts equals byte after key.
const PAX_KEY_SEPARATOR_SIZE = len("=")

// PAX_RECORD_TERMINATOR_SIZE counts newline after value.
const PAX_RECORD_TERMINATOR_SIZE = len("\n")

// GNU_LONG_FIELD_TERMINATOR_SIZE counts trailing NUL.
const GNU_LONG_FIELD_TERMINATOR_SIZE = len("\x00")

// TIMESTAMP_FRACTION_SEPARATOR_SIZE is the decimal point width.
const TIMESTAMP_FRACTION_SEPARATOR_SIZE = len(".")

// PAX_PATH_KEY names logical path override.
const PAX_PATH_KEY = "path"

// PAX_LINK_PATH_KEY names logical link override.
const PAX_LINK_PATH_KEY = "linkpath"

// PAX_SIZE_KEY names logical content-size override.
const PAX_SIZE_KEY = "size"

// PAX_USER_IDENTIFIER_KEY names owner identifier override.
const PAX_USER_IDENTIFIER_KEY = "uid"

// PAX_GROUP_IDENTIFIER_KEY names group identifier override.
const PAX_GROUP_IDENTIFIER_KEY = "gid"

// PAX_USER_NAME_KEY names owner text override.
const PAX_USER_NAME_KEY = "uname"

// PAX_GROUP_NAME_KEY names group text override.
const PAX_GROUP_NAME_KEY = "gname"

// PAX_MODIFICATION_TIME_KEY names modification timestamp override.
const PAX_MODIFICATION_TIME_KEY = "mtime"

// PAX_ACCESS_TIME_KEY names access timestamp override.
const PAX_ACCESS_TIME_KEY = "atime"

// PAX_CHANGE_TIME_KEY names metadata-change timestamp override.
const PAX_CHANGE_TIME_KEY = "ctime"

// PAX_KEY_NAME_SIZE_MINIMUM is the shortest supported keyword.
const PAX_KEY_NAME_SIZE_MINIMUM = len(PAX_USER_IDENTIFIER_KEY)

// PAX_KEY_NAME_SIZE_MAXIMUM is the longest supported keyword.
const PAX_KEY_NAME_SIZE_MAXIMUM = len(PAX_LINK_PATH_KEY)

// PAX_TEXT_KEY_SIZE_MINIMUM is the shortest generated text keyword.
const PAX_TEXT_KEY_SIZE_MINIMUM = len(PAX_PATH_KEY)

// PAX_TEXT_KEY_SIZE_MAXIMUM is the longest generated text keyword.
const PAX_TEXT_KEY_SIZE_MAXIMUM = len(PAX_LINK_PATH_KEY)

// PAX_DECIMAL_KEY_SIZE_MINIMUM is the owner-identifier keyword width.
const PAX_DECIMAL_KEY_SIZE_MINIMUM = len(PAX_USER_IDENTIFIER_KEY)

// PAX_DECIMAL_KEY_SIZE_MAXIMUM is the size keyword width.
const PAX_DECIMAL_KEY_SIZE_MAXIMUM = len(PAX_SIZE_KEY)

// PAX_TIME_KEY_SIZE is the common generated timestamp keyword width.
const PAX_TIME_KEY_SIZE = len(PAX_MODIFICATION_TIME_KEY)

// PAX_GNU_SPARSE_KEY_PREFIX identifies unsupported GNU sparse records.
const PAX_GNU_SPARSE_KEY_PREFIX = "GNU.sparse."

// PAX_HEADER_NAME is conventional local metadata entry path.
const PAX_HEADER_NAME = "PaxHeaders.0"

// GNU_LONG_HEADER_NAME is conventional GNU long-field entry path.
const GNU_LONG_HEADER_NAME = "././@LongLink"

// USTAR_MAGIC identifies POSIX USTAR header.
const USTAR_MAGIC = "ustar\x00"

// USTAR_VERSION identifies POSIX USTAR revision.
const USTAR_VERSION = "00"

// GNU_MAGIC identifies GNU header.
const GNU_MAGIC = "ustar "

// GNU_VERSION identifies GNU revision.
const GNU_VERSION = " \x00"

// STAR_TRAILER identifies Schily STAR header.
const STAR_TRAILER = "tar\x00"

// UTF8_FIRST_TWO_MINIMUM starts valid two-byte encodings after overlong forms.
const UTF8_FIRST_TWO_MINIMUM byte = 0b11000010

// UTF8_FIRST_TWO_MAXIMUM closes two-byte prefix range.
const UTF8_FIRST_TWO_MAXIMUM byte = 0b11011111

// UTF8_FIRST_THREE_MINIMUM starts three-byte prefix range.
const UTF8_FIRST_THREE_MINIMUM byte = 0b11100000

// UTF8_FIRST_THREE_MAXIMUM closes three-byte prefix range.
const UTF8_FIRST_THREE_MAXIMUM byte = 0b11101111

// UTF8_FIRST_FOUR_MINIMUM starts four-byte prefix range.
const UTF8_FIRST_FOUR_MINIMUM byte = 0b11110000

// UTF8_FIRST_FOUR_MAXIMUM closes Unicode-limited four-byte prefix range.
const UTF8_FIRST_FOUR_MAXIMUM byte = 0b11110100

// ARCHIVE_MEBIBYTE_COUNT_MAXIMUM states the 64-MiB policy as a binary formula.
const ARCHIVE_MEBIBYTE_COUNT_MAXIMUM = 1 << 6

// ARCHIVE_SIZE_MAXIMUM derives byte bound from shared IEC unit.
const ARCHIVE_SIZE_MAXIMUM = ARCHIVE_MEBIBYTE_COUNT_MAXIMUM * bits.MEBIBYTE_BYTES

// SPECIAL_FILE_SIZE_MAXIMUM matches libarchive one-MiB metadata defense.
const SPECIAL_FILE_SIZE_MAXIMUM = bits.MEBIBYTE_BYTES

// PAX_RECORD_COUNT_TEXT_SIZE_MAXIMUM follows the decimal width of the metadata bound.
const PAX_RECORD_COUNT_TEXT_SIZE_MAXIMUM = len("1000000")

// PAX_RECORD_COUNT_TEXT_SIZE_MINIMUM is one decimal count digit.
const PAX_RECORD_COUNT_TEXT_SIZE_MINIMUM = TYPE_FLAG_FIELD_SIZE

// PAX_SMALL_RECORD_COUNT_TEXT_SIZE covers every generated record below one hundred bytes.
const PAX_SMALL_RECORD_COUNT_TEXT_SIZE = PAX_RECORD_COUNT_TEXT_SIZE_MINIMUM +
	TYPE_FLAG_FIELD_SIZE

// PAX_RECORD_FIXED_SIZE_MAXIMUM reserves count text and the three grammar separators.
const PAX_RECORD_FIXED_SIZE_MAXIMUM = PAX_RECORD_COUNT_TEXT_SIZE_MAXIMUM +
	PAX_COUNT_SEPARATOR_SIZE + PAX_KEY_SEPARATOR_SIZE + PAX_RECORD_TERMINATOR_SIZE

// PAX_RECORD_FIXED_SIZE_MINIMUM is one count digit plus grammar separators.
const PAX_RECORD_FIXED_SIZE_MINIMUM = TYPE_FLAG_FIELD_SIZE +
	PAX_COUNT_SEPARATOR_SIZE + PAX_KEY_SEPARATOR_SIZE + PAX_RECORD_TERMINATOR_SIZE

// PAX_MANDATORY_RECORD_COUNT is the generated path and size pair.
const PAX_MANDATORY_RECORD_COUNT = 2

// PAX_PAYLOAD_SIZE_MINIMUM is mandatory one-byte path and zero-size records.
const PAX_PAYLOAD_SIZE_MINIMUM = PAX_MANDATORY_RECORD_COUNT*PAX_RECORD_FIXED_SIZE_MINIMUM +
	len(PAX_PATH_KEY) + TYPE_FLAG_FIELD_SIZE + len(PAX_SIZE_KEY) + TYPE_FLAG_FIELD_SIZE

// PAX_RECORD_SIZE_MINIMUM is one one-byte key with an empty value.
const PAX_RECORD_SIZE_MINIMUM = PAX_RECORD_FIXED_SIZE_MINIMUM + TYPE_FLAG_FIELD_SIZE

// PAX_PATH_RECORD_OVERHEAD_MAXIMUM excludes the path value itself.
const PAX_PATH_RECORD_OVERHEAD_MAXIMUM = PAX_RECORD_COUNT_TEXT_SIZE_MAXIMUM +
	PAX_COUNT_SEPARATOR_SIZE + len(PAX_PATH_KEY) + PAX_KEY_SEPARATOR_SIZE +
	PAX_RECORD_TERMINATOR_SIZE

// PAX_SIZE_RECORD_SIZE_MINIMUM is the mandatory zero-size record.
const PAX_SIZE_RECORD_SIZE_MINIMUM = PAX_RECORD_FIXED_SIZE_MINIMUM +
	len(PAX_SIZE_KEY) + TYPE_FLAG_FIELD_SIZE

// PAX_GENERATED_RECORD_SIZE_MINIMUM is one shortest nonzero decimal record.
const PAX_GENERATED_RECORD_SIZE_MINIMUM = PAX_RECORD_FIXED_SIZE_MINIMUM +
	len(PAX_USER_IDENTIFIER_KEY) + TYPE_FLAG_FIELD_SIZE

// PAX_TEXT_RECORD_SIZE_MINIMUM is one-byte path in its shortest frame.
const PAX_TEXT_RECORD_SIZE_MINIMUM = PAX_RECORD_COUNT_TEXT_SIZE_MINIMUM +
	PAX_COUNT_SEPARATOR_SIZE +
	len(PAX_PATH_KEY) + PAX_KEY_SEPARATOR_SIZE + TYPE_FLAG_FIELD_SIZE +
	PAX_RECORD_TERMINATOR_SIZE

// PAX_TEXT_RECORD_DESTINATION_SIZE_MINIMUM is one final one-byte owner record.
const PAX_TEXT_RECORD_DESTINATION_SIZE_MINIMUM = PAX_SMALL_RECORD_COUNT_TEXT_SIZE +
	PAX_COUNT_SEPARATOR_SIZE + len(PAX_USER_NAME_KEY) + PAX_KEY_SEPARATOR_SIZE +
	TYPE_FLAG_FIELD_SIZE + PAX_RECORD_TERMINATOR_SIZE

// PAX_DECIMAL_RECORD_DESTINATION_SIZE_MAXIMUM follows the shortest path record.
const PAX_DECIMAL_RECORD_DESTINATION_SIZE_MAXIMUM = SPECIAL_FILE_SIZE_MAXIMUM -
	PAX_TEXT_RECORD_SIZE_MINIMUM

// PAX_NAME_SIZE_MAXIMUM leaves mandatory, caller, and timestamp records intact.
const PAX_NAME_SIZE_MAXIMUM = SPECIAL_FILE_SIZE_MAXIMUM -
	PAX_PATH_RECORD_OVERHEAD_MAXIMUM - PAX_SIZE_RECORD_SIZE_MINIMUM -
	PAX_RECORD_SIZE_MINIMUM - PAX_TIMESTAMP_RECORD_SIZE_MINIMUM

// PAX_BASE_PAYLOAD_SIZE_MINIMUM is the shortest mandatory path and size pair.
const PAX_BASE_PAYLOAD_SIZE_MINIMUM = PAX_PAYLOAD_SIZE_MINIMUM

// PAX_LINK_RECORD_OVERHEAD_MAXIMUM excludes the link value itself.
const PAX_LINK_RECORD_OVERHEAD_MAXIMUM = PAX_RECORD_COUNT_TEXT_SIZE_MAXIMUM +
	PAX_COUNT_SEPARATOR_SIZE + len(PAX_LINK_PATH_KEY) + PAX_KEY_SEPARATOR_SIZE +
	PAX_RECORD_TERMINATOR_SIZE

// PAX_LINK_NAME_SIZE_MAXIMUM leaves mandatory, caller, and timestamp records intact.
const PAX_LINK_NAME_SIZE_MAXIMUM = SPECIAL_FILE_SIZE_MAXIMUM -
	PAX_BASE_PAYLOAD_SIZE_MINIMUM - PAX_LINK_RECORD_OVERHEAD_MAXIMUM -
	PAX_RECORD_SIZE_MINIMUM - PAX_TIMESTAMP_RECORD_SIZE_MINIMUM

// PAX_USER_NAME_RECORD_OVERHEAD_MAXIMUM excludes the owner value itself.
const PAX_USER_NAME_RECORD_OVERHEAD_MAXIMUM = PAX_RECORD_COUNT_TEXT_SIZE_MAXIMUM +
	PAX_COUNT_SEPARATOR_SIZE + len(PAX_USER_NAME_KEY) + PAX_KEY_SEPARATOR_SIZE +
	PAX_RECORD_TERMINATOR_SIZE

// HEADER_USER_NAME_SIZE_MAXIMUM is the largest readable owner-name record value.
const HEADER_USER_NAME_SIZE_MAXIMUM = SPECIAL_FILE_SIZE_MAXIMUM -
	PAX_USER_NAME_RECORD_OVERHEAD_MAXIMUM

// HEADER_USER_NAME_SIZE_UNVALIDATED_MAXIMUM admits the first rejected owner byte.
const HEADER_USER_NAME_SIZE_UNVALIDATED_MAXIMUM = HEADER_USER_NAME_SIZE_MAXIMUM +
	TYPE_FLAG_FIELD_SIZE

// PAX_USER_NAME_SIZE_MAXIMUM leaves mandatory, caller, and timestamp records intact.
const PAX_USER_NAME_SIZE_MAXIMUM = SPECIAL_FILE_SIZE_MAXIMUM -
	PAX_BASE_PAYLOAD_SIZE_MINIMUM - PAX_USER_NAME_RECORD_OVERHEAD_MAXIMUM -
	PAX_RECORD_SIZE_MINIMUM - PAX_TIMESTAMP_RECORD_SIZE_MINIMUM

// PAX_GROUP_NAME_RECORD_OVERHEAD_MAXIMUM excludes the group value itself.
const PAX_GROUP_NAME_RECORD_OVERHEAD_MAXIMUM = PAX_RECORD_COUNT_TEXT_SIZE_MAXIMUM +
	PAX_COUNT_SEPARATOR_SIZE + len(PAX_GROUP_NAME_KEY) + PAX_KEY_SEPARATOR_SIZE +
	PAX_RECORD_TERMINATOR_SIZE

// HEADER_GROUP_NAME_SIZE_MAXIMUM is the largest readable group-name record value.
const HEADER_GROUP_NAME_SIZE_MAXIMUM = SPECIAL_FILE_SIZE_MAXIMUM -
	PAX_GROUP_NAME_RECORD_OVERHEAD_MAXIMUM

// HEADER_GROUP_NAME_SIZE_UNVALIDATED_MAXIMUM admits the first rejected group byte.
const HEADER_GROUP_NAME_SIZE_UNVALIDATED_MAXIMUM = HEADER_GROUP_NAME_SIZE_MAXIMUM +
	TYPE_FLAG_FIELD_SIZE

// PAX_GROUP_NAME_SIZE_MAXIMUM leaves mandatory, caller, and timestamp records intact.
const PAX_GROUP_NAME_SIZE_MAXIMUM = SPECIAL_FILE_SIZE_MAXIMUM -
	PAX_BASE_PAYLOAD_SIZE_MINIMUM - PAX_GROUP_NAME_RECORD_OVERHEAD_MAXIMUM -
	PAX_RECORD_SIZE_MINIMUM - PAX_TIMESTAMP_RECORD_SIZE_MINIMUM

// PAX_HEADER_RECORDS_SIZE_MAXIMUM leaves both generated mandatory records intact.
const PAX_HEADER_RECORDS_SIZE_MAXIMUM = SPECIAL_FILE_SIZE_MAXIMUM -
	PAX_BASE_PAYLOAD_SIZE_MINIMUM

// PAX_RECORDS_PAYLOAD_SIZE_MINIMUM adds the shortest caller record.
const PAX_RECORDS_PAYLOAD_SIZE_MINIMUM = PAX_PAYLOAD_SIZE_MINIMUM +
	PAX_RECORD_SIZE_MINIMUM

// ENTRY_SIZE_DECIMAL_SIZE_MAXIMUM is decimal width of the archive policy bound.
const ENTRY_SIZE_DECIMAL_SIZE_MAXIMUM = PAX_RECORD_COUNT_TEXT_SIZE_MAXIMUM +
	TYPE_FLAG_FIELD_SIZE

// INTEGER_DECIMAL_SIZE_MAXIMUM includes sign on the widest signed value.
const INTEGER_DECIMAL_SIZE_MAXIMUM = strconv.DECIMAL_TEXT_SIZE_MAXIMUM

// DECIMAL_MAGNITUDE_SIZE_MAXIMUM excludes the optional signed prefix.
const DECIMAL_MAGNITUDE_SIZE_MAXIMUM = INTEGER_DECIMAL_SIZE_MAXIMUM -
	TYPE_FLAG_FIELD_SIZE

// DECIMAL_MAGNITUDE_MAXIMUM is absolute value of the smallest signed integer.
const DECIMAL_MAGNITUDE_MAXIMUM uint64 = 1 << (bits.BIT_COUNT_64_MAXIMUM - 1)

// TIMESTAMP_VALUE_SIZE_MAXIMUM includes widest seconds and full precision.
const TIMESTAMP_VALUE_SIZE_MAXIMUM = INTEGER_DECIMAL_SIZE_MAXIMUM +
	TIMESTAMP_FRACTION_SEPARATOR_SIZE + TIMESTAMP_FRACTION_DIGIT_COUNT_MAXIMUM

// PAX_PATH_RECORD_SIZE_CANDIDATE_MAXIMUM frames the largest bounded path.
const PAX_PATH_RECORD_SIZE_CANDIDATE_MAXIMUM = PAX_RECORD_FIXED_SIZE_MAXIMUM +
	len(PAX_PATH_KEY) + HEADER_TEXT_SIZE_MAXIMUM

// PAX_LINK_RECORD_SIZE_CANDIDATE_MAXIMUM frames the largest bounded link.
const PAX_LINK_RECORD_SIZE_CANDIDATE_MAXIMUM = PAX_RECORD_FIXED_SIZE_MAXIMUM +
	len(PAX_LINK_PATH_KEY) + HEADER_TEXT_SIZE_MAXIMUM

// PAX_USER_NAME_RECORD_SIZE_CANDIDATE_MAXIMUM frames the largest bounded owner.
const PAX_USER_NAME_RECORD_SIZE_CANDIDATE_MAXIMUM = PAX_RECORD_FIXED_SIZE_MAXIMUM +
	len(PAX_USER_NAME_KEY) + HEADER_USER_NAME_SIZE_MAXIMUM

// PAX_GROUP_NAME_RECORD_SIZE_CANDIDATE_MAXIMUM frames the largest bounded group.
const PAX_GROUP_NAME_RECORD_SIZE_CANDIDATE_MAXIMUM = PAX_RECORD_FIXED_SIZE_MAXIMUM +
	len(PAX_GROUP_NAME_KEY) + HEADER_GROUP_NAME_SIZE_MAXIMUM

// PAX_SIZE_RECORD_SIZE_CANDIDATE_MAXIMUM frames the largest logical size.
const PAX_SIZE_RECORD_SIZE_CANDIDATE_MAXIMUM = PAX_SMALL_RECORD_COUNT_TEXT_SIZE +
	PAX_COUNT_SEPARATOR_SIZE + len(PAX_SIZE_KEY) + PAX_KEY_SEPARATOR_SIZE +
	ENTRY_SIZE_DECIMAL_SIZE_MAXIMUM + PAX_RECORD_TERMINATOR_SIZE

// PAX_IDENTIFIER_BODY_SIZE_MAXIMUM contains an identifier key and signed decimal value.
const PAX_IDENTIFIER_BODY_SIZE_MAXIMUM = PAX_DECIMAL_KEY_SIZE_MINIMUM +
	INTEGER_DECIMAL_SIZE_MAXIMUM

// PAX_IDENTIFIER_RECORD_SIZE_MAXIMUM frames one signed identifier record.
const PAX_IDENTIFIER_RECORD_SIZE_MAXIMUM = PAX_SMALL_RECORD_COUNT_TEXT_SIZE +
	PAX_COUNT_SEPARATOR_SIZE +
	PAX_IDENTIFIER_BODY_SIZE_MAXIMUM + PAX_KEY_SEPARATOR_SIZE +
	PAX_RECORD_TERMINATOR_SIZE

// PAX_TIMESTAMP_BODY_SIZE_MAXIMUM excludes its self-count prefix.
const PAX_TIMESTAMP_BODY_SIZE_MAXIMUM = PAX_COUNT_SEPARATOR_SIZE +
	len(PAX_MODIFICATION_TIME_KEY) + PAX_KEY_SEPARATOR_SIZE +
	TIMESTAMP_VALUE_SIZE_MAXIMUM + PAX_RECORD_TERMINATOR_SIZE

// PAX_TIMESTAMP_RECORD_SIZE_MAXIMUM includes decimal width of its complete size.
const PAX_TIMESTAMP_RECORD_SIZE_MAXIMUM = PAX_TIMESTAMP_BODY_SIZE_MAXIMUM +
	PAX_SMALL_RECORD_COUNT_TEXT_SIZE

// PAX_TIMESTAMP_BODY_SIZE_MINIMUM excludes the self-count prefix.
const PAX_TIMESTAMP_BODY_SIZE_MINIMUM = PAX_COUNT_SEPARATOR_SIZE +
	len(PAX_MODIFICATION_TIME_KEY) + PAX_KEY_SEPARATOR_SIZE +
	TYPE_FLAG_FIELD_SIZE + PAX_RECORD_TERMINATOR_SIZE

// PAX_TIMESTAMP_RECORD_SIZE_MINIMUM includes its two-digit complete size.
const PAX_TIMESTAMP_RECORD_SIZE_MINIMUM = PAX_TIMESTAMP_BODY_SIZE_MINIMUM +
	PAX_SMALL_RECORD_COUNT_TEXT_SIZE

// PAX_TIMESTAMP_RECORD_COUNT is number of generated time overrides.
const PAX_TIMESTAMP_RECORD_COUNT = (len(PAX_MODIFICATION_TIME_KEY) +
	len(PAX_ACCESS_TIME_KEY) + len(PAX_CHANGE_TIME_KEY)) / PAX_TIME_KEY_SIZE

// PAX_PATH_WRITE_RECORD_SIZE_MAXIMUM frames the largest writable path.
const PAX_PATH_WRITE_RECORD_SIZE_MAXIMUM = PAX_RECORD_FIXED_SIZE_MAXIMUM +
	len(PAX_PATH_KEY) + PAX_NAME_SIZE_MAXIMUM

// PAX_LINK_WRITE_RECORD_SIZE_MAXIMUM frames the largest writable link.
const PAX_LINK_WRITE_RECORD_SIZE_MAXIMUM = PAX_RECORD_FIXED_SIZE_MAXIMUM +
	len(PAX_LINK_PATH_KEY) + PAX_LINK_NAME_SIZE_MAXIMUM

// PAX_USER_NAME_WRITE_RECORD_SIZE_MAXIMUM frames the largest writable owner.
const PAX_USER_NAME_WRITE_RECORD_SIZE_MAXIMUM = PAX_RECORD_FIXED_SIZE_MAXIMUM +
	len(PAX_USER_NAME_KEY) + PAX_USER_NAME_SIZE_MAXIMUM

// PAX_GROUP_NAME_WRITE_RECORD_SIZE_MAXIMUM frames the largest writable group.
const PAX_GROUP_NAME_WRITE_RECORD_SIZE_MAXIMUM = PAX_RECORD_FIXED_SIZE_MAXIMUM +
	len(PAX_GROUP_NAME_KEY) + PAX_GROUP_NAME_SIZE_MAXIMUM

// PAX_PAYLOAD_SIZE_CANDIDATE_MAXIMUM sums every independently bounded field.
const PAX_PAYLOAD_SIZE_CANDIDATE_MAXIMUM = PAX_PATH_RECORD_SIZE_CANDIDATE_MAXIMUM +
	PAX_LINK_RECORD_SIZE_CANDIDATE_MAXIMUM +
	PAX_USER_NAME_RECORD_SIZE_CANDIDATE_MAXIMUM +
	PAX_GROUP_NAME_RECORD_SIZE_CANDIDATE_MAXIMUM +
	PAX_SIZE_RECORD_SIZE_CANDIDATE_MAXIMUM +
	PAX_IDENTIFIER_RECORD_SIZE_MAXIMUM +
	PAX_IDENTIFIER_RECORD_SIZE_MAXIMUM +
	PAX_TIMESTAMP_RECORD_COUNT*PAX_TIMESTAMP_RECORD_SIZE_MAXIMUM

// PAX_TIME_PAYLOAD_BASE_SIZE_MAXIMUM excludes optional timestamp records.
const PAX_TIME_PAYLOAD_BASE_SIZE_MAXIMUM = PAX_PAYLOAD_SIZE_CANDIDATE_MAXIMUM -
	PAX_TIMESTAMP_RECORD_COUNT*PAX_TIMESTAMP_RECORD_SIZE_MAXIMUM

// PAX_TIME_SIZE_MAXIMUM contains all generated timestamp records.
const PAX_TIME_SIZE_MAXIMUM = PAX_TIMESTAMP_RECORD_COUNT *
	PAX_TIMESTAMP_RECORD_SIZE_MAXIMUM

// PAX_KEY_SIZE_MAXIMUM leaves an empty value inside the largest record.
const PAX_KEY_SIZE_MAXIMUM = SPECIAL_FILE_SIZE_MAXIMUM - PAX_RECORD_FIXED_SIZE_MAXIMUM

// PAX_VALUE_SIZE_MAXIMUM also leaves the shortest valid key.
const PAX_VALUE_SIZE_MAXIMUM = PAX_KEY_SIZE_MAXIMUM - TYPE_FLAG_FIELD_SIZE

// PAX_NUMERIC_VALUE_SIZE_MAXIMUM leaves shortest numeric key and framing.
const PAX_NUMERIC_VALUE_SIZE_MAXIMUM = SPECIAL_FILE_SIZE_MAXIMUM -
	PAX_RECORD_FIXED_SIZE_MAXIMUM - PAX_DECIMAL_KEY_SIZE_MINIMUM

// PAX_TIMESTAMP_VALUE_SIZE_MAXIMUM leaves one timestamp key and framing.
const PAX_TIMESTAMP_VALUE_SIZE_MAXIMUM = SPECIAL_FILE_SIZE_MAXIMUM -
	PAX_RECORD_FIXED_SIZE_MAXIMUM - PAX_TIME_KEY_SIZE

// PAX_APPLY_USER_NAME_SIZE_MAXIMUM leaves shortest following record.
const PAX_APPLY_USER_NAME_SIZE_MAXIMUM = HEADER_USER_NAME_SIZE_MAXIMUM -
	PAX_RECORD_SIZE_MINIMUM

// PAX_APPLY_GROUP_NAME_SIZE_MAXIMUM leaves shortest following record.
const PAX_APPLY_GROUP_NAME_SIZE_MAXIMUM = HEADER_GROUP_NAME_SIZE_MAXIMUM -
	PAX_RECORD_SIZE_MINIMUM

// UTF8_POSITION_MAXIMUM leaves one hostile leading byte inside text.
const UTF8_POSITION_MAXIMUM = HEADER_TEXT_SIZE_MAXIMUM - TYPE_FLAG_FIELD_SIZE

// USTAR_SUFFIX_START_MAXIMUM follows full prefix and separator.
const USTAR_SUFFIX_START_MAXIMUM = PREFIX_FIELD_SIZE + TYPE_FLAG_FIELD_SIZE

// SPECIAL_FILE_SIZE_UNVALIDATED_MAXIMUM admits the first rejected metadata byte.
const SPECIAL_FILE_SIZE_UNVALIDATED_MAXIMUM = SPECIAL_FILE_SIZE_MAXIMUM + TYPE_FLAG_FIELD_SIZE

// HEADER_TEXT_SIZE_MAXIMUM reserves GNU long-field terminator space.
const HEADER_TEXT_SIZE_MAXIMUM = SPECIAL_FILE_SIZE_MAXIMUM - GNU_LONG_FIELD_TERMINATOR_SIZE

// GNU_EXTENDED_FIELD_SIZE_MINIMUM is the first value needing a long header.
const GNU_EXTENDED_FIELD_SIZE_MINIMUM = NAME_FIELD_SIZE + TYPE_FLAG_FIELD_SIZE

// HEADER_TEXT_SIZE_UNVALIDATED_MAXIMUM admits first rejected logical text byte.
const HEADER_TEXT_SIZE_UNVALIDATED_MAXIMUM = HEADER_TEXT_SIZE_MAXIMUM + TYPE_FLAG_FIELD_SIZE

// ARCHIVE_FOOTER_BLOCK_COUNT is TAR end-marker record count.
const ARCHIVE_FOOTER_BLOCK_COUNT = 2

// ARCHIVE_FOOTER_SIZE derives end-marker width from record count.
const ARCHIVE_FOOTER_SIZE = ARCHIVE_FOOTER_BLOCK_COUNT * BLOCK_SIZE

// PADDING_SIZE_MAXIMUM is largest gap before next block boundary.
const PADDING_SIZE_MAXIMUM = BLOCK_SIZE - TYPE_FLAG_FIELD_SIZE

// METADATA_BLOCK_COUNT_MAXIMUM is maximum bounded metadata rounded to blocks.
const METADATA_BLOCK_COUNT_MAXIMUM = SPECIAL_FILE_SIZE_MAXIMUM / BLOCK_SIZE

// WIRE_NAME_SIZE_MAXIMUM joins largest prefix, separator, and suffix fields.
const WIRE_NAME_SIZE_MAXIMUM = PREFIX_FIELD_SIZE + TYPE_FLAG_FIELD_SIZE + NAME_FIELD_SIZE

// SMALL_NUMERIC_PAYLOAD_BIT_COUNT follows base-256 space below its marker byte.
const SMALL_NUMERIC_PAYLOAD_BIT_COUNT = (MODE_FIELD_SIZE - TYPE_FLAG_FIELD_SIZE) *
	bits.BIT_COUNT_8_MAXIMUM

// SMALL_NUMERIC_MINIMUM is the smallest value in an eight-byte base-256 field.
const SMALL_NUMERIC_MINIMUM int64 = -(1 << SMALL_NUMERIC_PAYLOAD_BIT_COUNT)

// SMALL_NUMERIC_MAXIMUM is the largest value in an eight-byte base-256 field.
const SMALL_NUMERIC_MAXIMUM int64 = 1<<SMALL_NUMERIC_PAYLOAD_BIT_COUNT - 1

// SMALL_OCTAL_PAYLOAD_BIT_COUNT follows the digits before one field terminator.
const SMALL_OCTAL_PAYLOAD_BIT_COUNT = (MODE_FIELD_SIZE - TYPE_FLAG_FIELD_SIZE) *
	OCTAL_BITS_PER_DIGIT

// SMALL_OCTAL_MAXIMUM is the largest value preserved by an eight-byte octal field.
const SMALL_OCTAL_MAXIMUM int64 = 1<<SMALL_OCTAL_PAYLOAD_BIT_COUNT - 1

// LARGE_OCTAL_PAYLOAD_BIT_COUNT follows the digits before a twelve-byte field terminator.
const LARGE_OCTAL_PAYLOAD_BIT_COUNT = (ENTRY_SIZE_FIELD_SIZE - TYPE_FLAG_FIELD_SIZE) *
	OCTAL_BITS_PER_DIGIT

// LARGE_OCTAL_MAXIMUM is the largest value preserved by a twelve-byte octal field.
const LARGE_OCTAL_MAXIMUM int64 = 1<<LARGE_OCTAL_PAYLOAD_BIT_COUNT - 1

// OCTAL_PARSE_BIT_COUNT_MAXIMUM admits a field whose final digit has no terminator.
const OCTAL_PARSE_BIT_COUNT_MAXIMUM = NUMERIC_FIELD_SIZE_MAXIMUM *
	OCTAL_BITS_PER_DIGIT

// OCTAL_PARSE_MAXIMUM is largest unsigned value the octal grammar accepts.
const OCTAL_PARSE_MAXIMUM int64 = 1<<OCTAL_PARSE_BIT_COUNT_MAXIMUM - 1

// HEADER_CHECKSUM_DATA_BYTE_COUNT excludes the checksum field replaced by spaces.
const HEADER_CHECKSUM_DATA_BYTE_COUNT = BLOCK_SIZE - CHECKSUM_FIELD_SIZE

// HEADER_CHECKSUM_SPACE_SUM is the normalized checksum-field contribution.
const HEADER_CHECKSUM_SPACE_SUM = CHECKSUM_FIELD_SIZE * int(' ')

// HEADER_CHECKSUM_UNSIGNED_MINIMUM has one nonzero data byte so parsing reaches checksum work.
const HEADER_CHECKSUM_UNSIGNED_MINIMUM = HEADER_CHECKSUM_SPACE_SUM + TYPE_FLAG_FIELD_SIZE

// HEADER_CHECKSUM_UNSIGNED_MAXIMUM fills every data byte with its unsigned maximum.
const HEADER_CHECKSUM_UNSIGNED_MAXIMUM = HEADER_CHECKSUM_DATA_BYTE_COUNT*
	int(bits.WORD_8_MAXIMUM) + HEADER_CHECKSUM_SPACE_SUM

// HEADER_CHECKSUM_SIGNED_MINIMUM fills every data byte with signed-byte minimum.
const HEADER_CHECKSUM_SIGNED_MINIMUM = -HEADER_CHECKSUM_DATA_BYTE_COUNT*
	(1<<(bits.BIT_COUNT_8_MAXIMUM-1)) + HEADER_CHECKSUM_SPACE_SUM

// HEADER_CHECKSUM_SIGNED_MAXIMUM fills every data byte with signed-byte maximum.
const HEADER_CHECKSUM_SIGNED_MAXIMUM = HEADER_CHECKSUM_DATA_BYTE_COUNT*
	((1<<(bits.BIT_COUNT_8_MAXIMUM-1))-1) + HEADER_CHECKSUM_SPACE_SUM

// GNU_LONG_FIELD_COUNT_MAXIMUM covers path and link target extensions.
const GNU_LONG_FIELD_COUNT_MAXIMUM = 2

// READER_METADATA_SIZE_MAXIMUM retains both GNU long fields and one local PAX body.
const READER_METADATA_SIZE_MAXIMUM = (GNU_LONG_FIELD_COUNT_MAXIMUM + TYPE_FLAG_FIELD_SIZE) *
	SPECIAL_FILE_SIZE_MAXIMUM

// READER_METADATA_BLOCK_COUNT_MAXIMUM converts retained extension bytes to blocks.
const READER_METADATA_BLOCK_COUNT_MAXIMUM = READER_METADATA_SIZE_MAXIMUM / BLOCK_SIZE

// WRITER_STORAGE_SIZE_MINIMUM stages largest footer with prior padding.
const WRITER_STORAGE_SIZE_MINIMUM = PADDING_SIZE_MAXIMUM + ARCHIVE_FOOTER_SIZE

// GNU_LONG_FIELD_WIRE_SIZE_MAXIMUM includes metadata header and payload.
const GNU_LONG_FIELD_WIRE_SIZE_MAXIMUM = BLOCK_SIZE + SPECIAL_FILE_SIZE_MAXIMUM

// WRITER_STORAGE_SIZE_MAXIMUM stages two GNU fields and main header after padding.
const WRITER_STORAGE_SIZE_MAXIMUM = PADDING_SIZE_MAXIMUM +
	GNU_LONG_FIELD_COUNT_MAXIMUM*GNU_LONG_FIELD_WIRE_SIZE_MAXIMUM + BLOCK_SIZE

// ENCODED_HEADER_SIZE_MAXIMUM excludes padding owned by prior content.
const ENCODED_HEADER_SIZE_MAXIMUM = WRITER_STORAGE_SIZE_MAXIMUM -
	PADDING_SIZE_MAXIMUM

// HEADER_BLOCK_COUNT is one block expressed from its own byte width.
const HEADER_BLOCK_COUNT = BLOCK_SIZE / BLOCK_SIZE

// PAX_HEADER_WIRE_SIZE_MINIMUM is metadata header, padded payload, and main header.
const PAX_HEADER_WIRE_SIZE_MINIMUM = (ARCHIVE_FOOTER_BLOCK_COUNT +
	HEADER_BLOCK_COUNT) * BLOCK_SIZE

// PAX_HEADER_END_POSITION_MAXIMUM includes prior content padding.
const PAX_HEADER_END_POSITION_MAXIMUM = PADDING_SIZE_MAXIMUM +
	ARCHIVE_FOOTER_SIZE + SPECIAL_FILE_SIZE_MAXIMUM

// PAX_ENCODED_HEADER_SIZE_MAXIMUM excludes padding owned by prior content.
const PAX_ENCODED_HEADER_SIZE_MAXIMUM = PAX_HEADER_END_POSITION_MAXIMUM -
	PADDING_SIZE_MAXIMUM

// PAX_PAYLOAD_START_POSITION_MINIMUM follows one metadata header block.
const PAX_PAYLOAD_START_POSITION_MINIMUM = BLOCK_SIZE

// PAX_PAYLOAD_START_POSITION_MAXIMUM includes prior content padding.
const PAX_PAYLOAD_START_POSITION_MAXIMUM = PADDING_SIZE_MAXIMUM + BLOCK_SIZE

// GNU_LONG_POSITION_MAXIMUM starts the second possible long field.
const GNU_LONG_POSITION_MAXIMUM = PADDING_SIZE_MAXIMUM +
	GNU_LONG_FIELD_WIRE_SIZE_MAXIMUM

// GNU_LONG_END_POSITION_MINIMUM is one metadata header plus padded long body.
const GNU_LONG_END_POSITION_MINIMUM = ARCHIVE_FOOTER_SIZE

// GNU_LONG_DESTINATION_SIZE_MINIMUM leaves room for the required main header.
const GNU_LONG_DESTINATION_SIZE_MINIMUM = GNU_LONG_END_POSITION_MINIMUM + BLOCK_SIZE

// GNU_LONG_END_POSITION_MAXIMUM leaves the final main header unstaged.
const GNU_LONG_END_POSITION_MAXIMUM = WRITER_STORAGE_SIZE_MAXIMUM - BLOCK_SIZE

// FORMAT_MINIMUM starts format domain at unknown selection.
const FORMAT_MINIMUM uint8 = uint8(FORMAT_UNKNOWN)

// FORMAT_MAXIMUM closes format domain at STAR.
const FORMAT_MAXIMUM uint8 = uint8(FORMAT_STAR)

// STATUS_MINIMUM starts operation result domain at success.
const STATUS_MINIMUM uint8 = uint8(STATUS_OK)

// STATUS_MAXIMUM closes operation result domain at incomplete content.
const STATUS_MAXIMUM uint8 = uint8(STATUS_TRANSPORT_FAILED)

// COUNT_MINIMUM admits empty archive and empty entry.
const COUNT_MINIMUM = bits.BIT_COUNT_MINIMUM

// COUNT_MAXIMUM matches archive byte boundary.
const COUNT_MAXIMUM = ARCHIVE_SIZE_MAXIMUM

// TIMESTAMP_NANOSECOND_MINIMUM is first sub-second value.
const TIMESTAMP_NANOSECOND_MINIMUM Nanosecond_Count = Nanosecond_Count(
	time.NANOSECOND - time.NANOSECOND,
)

// TIMESTAMP_NANOSECOND_COUNT is nanoseconds in one second.
const TIMESTAMP_NANOSECOND_COUNT = time.SECOND / time.NANOSECOND

// TIMESTAMP_NANOSECOND_MAXIMUM is final valid sub-second value.
const TIMESTAMP_NANOSECOND_MAXIMUM Nanosecond_Count = Nanosecond_Count(
	TIMESTAMP_NANOSECOND_COUNT - 1,
)

// TIMESTAMP_FRACTION_DIGIT_COUNT_MAXIMUM is decimal width below one second.
const TIMESTAMP_FRACTION_DIGIT_COUNT_MAXIMUM = DECIMAL_BASE - 1

// TIMESTAMP_FIRST_FRACTION_DIVISOR selects first decimal fraction digit.
const TIMESTAMP_FIRST_FRACTION_DIVISOR = TIMESTAMP_NANOSECOND_COUNT / DECIMAL_BASE

// INTEGER_MINIMUM is lowest PAX signed value.
const INTEGER_MINIMUM Integer = Integer(bits.INTEGER_64_MINIMUM)

// INTEGER_MAXIMUM is highest PAX signed value.
const INTEGER_MAXIMUM Integer = Integer(bits.INTEGER_64_MAXIMUM)

// UNSIGNED_INTEGER_MINIMUM begins unsigned formatting domain.
const UNSIGNED_INTEGER_MINIMUM Unsigned_Integer = Unsigned_Integer(bits.WORD_64_MINIMUM)

// UNSIGNED_INTEGER_MAXIMUM closes unsigned formatting domain.
const UNSIGNED_INTEGER_MAXIMUM Unsigned_Integer = Unsigned_Integer(bits.WORD_64_MAXIMUM)

// TYPE_FLAG_MINIMUM is lowest extension byte.
const TYPE_FLAG_MINIMUM Type_Flag = Type_Flag(bits.WORD_8_MINIMUM)

// TYPE_FLAG_MAXIMUM is highest extension byte.
const TYPE_FLAG_MAXIMUM Type_Flag = Type_Flag(bits.WORD_8_MAXIMUM)

// Format identifies one TAR header family.
type Format uint8

// Format_Invariants bounds recognized format identity.
func Format_Invariants(value Format, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), FORMAT_MINIMUM, FORMAT_MAXIMUM).
		Ensure()
}

// Format_Unvalidated preserves every caller format byte until validation.
type Format_Unvalidated uint8

// Format_Unvalidated_Invariants covers the complete hostile byte domain.
func Format_Unvalidated_Invariants(
	value Format_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), bits.WORD_8_MINIMUM, bits.WORD_8_MAXIMUM).
		Ensure()
}

// FORMAT_UNKNOWN asks writer to select narrowest lossless format.
const FORMAT_UNKNOWN Format = Format(bits.WORD_8_MINIMUM)

// FORMAT_V7 identifies original Unix V7 header.
const FORMAT_V7 Format = FORMAT_UNKNOWN + 1

// FORMAT_USTAR identifies POSIX.1-1988 USTAR header.
const FORMAT_USTAR Format = FORMAT_V7 + 1

// FORMAT_PAX identifies POSIX.1-2001 extended header.
const FORMAT_PAX Format = FORMAT_USTAR + 1

// FORMAT_GNU identifies GNU header.
const FORMAT_GNU Format = FORMAT_PAX + 1

// FORMAT_STAR identifies read-only Schily STAR header.
const FORMAT_STAR Format = FORMAT_GNU + 1

// Status reports bounded operation result without owned error storage.
type Status uint8

// Status_Invariants bounds every operation result.
func Status_Invariants(value Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), STATUS_MINIMUM, STATUS_MAXIMUM).
		Ensure()
}

// Initialization_Status contains transport-storage binding outcomes.
type Initialization_Status uint8

// Initialization_Status_Invariants keeps lifecycle results exact.
func Initialization_Status_Invariants(
	value Initialization_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_STORAGE_INVALID),
		).
		Ensure()
}

// Header_Validation_Status contains hostile-header boundary outcomes.
type Header_Validation_Status uint8

// Header_Validation_Status_Invariants keeps validation failures exact.
func Header_Validation_Status_Invariants(
	value Header_Validation_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_INPUT_INVALID),
			uint8(STATUS_FORMAT_UNSUPPORTED), uint8(STATUS_FIELD_TOO_LONG),
		).
		Ensure()
}

// Header_Encoding_Status contains pure writer-validation outcomes only.
type Header_Encoding_Status uint8

// Header_Encoding_Status_Invariants keeps header-size validation exact.
func Header_Encoding_Status_Invariants(
	value Header_Encoding_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_FORMAT_UNSUPPORTED),
			uint8(STATUS_FIELD_TOO_LONG),
		).
		Ensure()
}

// Format_Encoding_Status contains one selected format's size outcomes.
type Format_Encoding_Status uint8

// Format_Encoding_Status_Invariants excludes generic dispatch failure.
func Format_Encoding_Status_Invariants(
	value Format_Encoding_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(STATUS_OK), uint8(STATUS_FIELD_TOO_LONG)).
		Ensure()
}

// Header_Stage_Status contains preflight and capacity outcomes only.
type Header_Stage_Status uint8

// Header_Stage_Status_Invariants keeps writer lifecycle failures outside staging.
func Header_Stage_Status_Invariants(
	value Header_Stage_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_OUTPUT_TOO_SMALL),
			uint8(STATUS_FORMAT_UNSUPPORTED), uint8(STATUS_FIELD_TOO_LONG),
		).
		Ensure()
}

// PAX_Records_Stage_Status excludes unsupported dispatch after record routing.
type PAX_Records_Stage_Status uint8

// PAX_Records_Stage_Status_Invariants admits record staging outcomes only.
func PAX_Records_Stage_Status_Invariants(
	value PAX_Records_Stage_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_OUTPUT_TOO_SMALL),
			uint8(STATUS_FIELD_TOO_LONG),
		).
		Ensure()
}

// Parse_Status contains complete fixed-header outcomes.
type Parse_Status uint8

// Parse_Status_Invariants keeps end-of-archive distinct from malformed fields.
func Parse_Status_Invariants(value Parse_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_END),
			uint8(STATUS_INPUT_INVALID), uint8(STATUS_FIELD_TOO_LONG),
		).
		Ensure()
}

// Basic_Parse_Status contains fixed numeric-header outcomes.
type Basic_Parse_Status uint8

// Basic_Parse_Status_Invariants excludes failures owned by later phases.
func Basic_Parse_Status_Invariants(
	value Basic_Parse_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_INPUT_INVALID),
			uint8(STATUS_FIELD_TOO_LONG),
		).
		Ensure()
}

// Input_Status reports success or malformed fixed metadata.
type Input_Status uint8

// Input_Status_Invariants keeps two-outcome parser helpers exact.
func Input_Status_Invariants(value Input_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(STATUS_OK), uint8(STATUS_INPUT_INVALID)).
		Ensure()
}

// PAX_Apply_Status contains extension grammar and sparse-feature outcomes.
type PAX_Apply_Status uint8

// PAX_Apply_Status_Invariants keeps PAX mutation failures exact.
func PAX_Apply_Status_Invariants(
	value PAX_Apply_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_INPUT_INVALID),
			uint8(STATUS_FORMAT_UNSUPPORTED),
		).
		Ensure()
}

// STATUS_OK means operation completed.
const STATUS_OK = bits.BIT_COUNT_MINIMUM

// STATUS_END means archive or current entry has no unread data.
const STATUS_END = STATUS_OK + TYPE_FLAG_FIELD_SIZE

// STATUS_INPUT_INVALID means archive bytes violate TAR grammar.
const STATUS_INPUT_INVALID = STATUS_END + TYPE_FLAG_FIELD_SIZE

// STATUS_OUTPUT_TOO_SMALL means caller output cannot hold complete requested result.
const STATUS_OUTPUT_TOO_SMALL = STATUS_INPUT_INVALID + TYPE_FLAG_FIELD_SIZE

// STATUS_FORMAT_UNSUPPORTED means recognized feature lacks bounded implementation.
const STATUS_FORMAT_UNSUPPORTED = STATUS_OUTPUT_TOO_SMALL + TYPE_FLAG_FIELD_SIZE

// STATUS_FIELD_TOO_LONG means metadata exceeds format or package boundary.
const STATUS_FIELD_TOO_LONG = STATUS_FORMAT_UNSUPPORTED + TYPE_FLAG_FIELD_SIZE

// STATUS_WRITE_TOO_LONG means input exceeds declared entry size.
const STATUS_WRITE_TOO_LONG = STATUS_FIELD_TOO_LONG + TYPE_FLAG_FIELD_SIZE

// STATUS_CLOSED means writer operation followed successful close.
const STATUS_CLOSED = STATUS_WRITE_TOO_LONG + TYPE_FLAG_FIELD_SIZE

// STATUS_STORAGE_INVALID means caller storage is missing, oversized, or overlaps input.
const STATUS_STORAGE_INVALID = STATUS_CLOSED + TYPE_FLAG_FIELD_SIZE

// STATUS_CONTENT_INCOMPLETE means next header arrived before declared content.
const STATUS_CONTENT_INCOMPLETE = STATUS_STORAGE_INVALID + TYPE_FLAG_FIELD_SIZE

// STATUS_TRANSPORT_FAILED leaves the concrete cause in Completion.Error.
const STATUS_TRANSPORT_FAILED = STATUS_CONTENT_INCOMPLETE + TYPE_FLAG_FIELD_SIZE

// Count reports bounded bytes consumed or written.
type Count int

// Count_Invariants bounds archive coordinates.
func Count_Invariants(value Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), COUNT_MINIMUM, COUNT_MAXIMUM).
		Ensure()
}

// Decimal_Magnitude is absolute magnitude of one signed 64-bit value.
type Decimal_Magnitude uint64

// Decimal_Magnitude_Invariants bounds conversion without signed overflow.
func Decimal_Magnitude_Invariants(
	value Decimal_Magnitude, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, DECIMAL_MAGNITUDE_MAXIMUM).
		Ensure()
}

// Decimal_Magnitude_Size is decimal width without a sign.
type Decimal_Magnitude_Size int

// Decimal_Magnitude_Size_Invariants bounds zero through largest magnitude text.
func Decimal_Magnitude_Size_Invariants(
	value Decimal_Magnitude_Size, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), TYPE_FLAG_FIELD_SIZE, DECIMAL_MAGNITUDE_SIZE_MAXIMUM,
		).
		Ensure()
}

// Decimal_Value_Size is decimal width including an optional sign.
type Decimal_Value_Size int

// Decimal_Value_Size_Invariants bounds every signed integer rendering.
func Decimal_Value_Size_Invariants(
	value Decimal_Value_Size, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), TYPE_FLAG_FIELD_SIZE, INTEGER_DECIMAL_SIZE_MAXIMUM).
		Ensure()
}

// Fractional_Nanoseconds is one nonzero sub-second remainder.
type Fractional_Nanoseconds int32

// Fractional_Nanoseconds_Invariants excludes whole-second timestamps.
func Fractional_Nanoseconds_Invariants(
	value Fractional_Nanoseconds, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int32(
			int32(value), int32(TYPE_FLAG_FIELD_SIZE),
			int32(TIMESTAMP_NANOSECOND_MAXIMUM),
		).
		Ensure()
}

// Fraction_Digit_Count is retained decimal precision after trailing-zero removal.
type Fraction_Digit_Count int

// Fraction_Digit_Count_Invariants bounds one through nanosecond precision.
func Fraction_Digit_Count_Invariants(
	value Fraction_Digit_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), TYPE_FLAG_FIELD_SIZE,
			TIMESTAMP_FRACTION_DIGIT_COUNT_MAXIMUM,
		).
		Ensure()
}

// Timestamp_Value_Size is decimal seconds with optional fractional precision.
type Timestamp_Value_Size int

// Timestamp_Value_Size_Invariants bounds every PAX timestamp rendering.
func Timestamp_Value_Size_Invariants(
	value Timestamp_Value_Size, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), TYPE_FLAG_FIELD_SIZE, TIMESTAMP_VALUE_SIZE_MAXIMUM).
		Ensure()
}

// Unpadded_Size is one nonnegative archive span before block alignment.
type Unpadded_Size int64

// Unpadded_Size_Invariants bounds every content or metadata alignment input.
func Unpadded_Size_Invariants(value Unpadded_Size, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), COUNT_MINIMUM, ARCHIVE_SIZE_MAXIMUM).
		Ensure()
}

// Block_Padding is bytes required to reach the next block boundary.
type Block_Padding int64

// Block_Padding_Invariants bounds one alignment remainder.
func Block_Padding_Invariants(value Block_Padding, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), COUNT_MINIMUM, PADDING_SIZE_MAXIMUM).
		Ensure()
}

// Metadata_Payload_Size is one extension body before block alignment.
type Metadata_Payload_Size int

// Metadata_Payload_Size_Invariants bounds hostile and generated metadata bodies.
func Metadata_Payload_Size_Invariants(
	value Metadata_Payload_Size, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), COUNT_MINIMUM, SPECIAL_FILE_SIZE_MAXIMUM).
		Ensure()
}

// Metadata_Block_Count is the complete blocks occupied by one metadata body.
type Metadata_Block_Count int

// Metadata_Block_Count_Invariants bounds zero through maximum metadata blocks.
func Metadata_Block_Count_Invariants(
	value Metadata_Block_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), COUNT_MINIMUM, METADATA_BLOCK_COUNT_MAXIMUM,
		).
		Ensure()
}

// Metadata_Block_Position is retained extension cursor in wire blocks.
type Metadata_Block_Position int

// Metadata_Block_Position_Invariants bounds retained extension cursor.
func Metadata_Block_Position_Invariants(
	value Metadata_Block_Position, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), COUNT_MINIMUM, READER_METADATA_BLOCK_COUNT_MAXIMUM,
		).
		Ensure()
}

// Metadata_Available_Boundary is retained extension storage end.
type Metadata_Available_Boundary int

// Metadata_Available_Boundary_Invariants bounds retained extension storage.
func Metadata_Available_Boundary_Invariants(
	value Metadata_Available_Boundary, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), COUNT_MINIMUM, READER_METADATA_SIZE_MAXIMUM).
		Ensure()
}

// UTF8_Position is cursor within validated metadata candidate.
type UTF8_Position int

// UTF8_Position_Invariants bounds metadata cursor.
func UTF8_Position_Invariants(value UTF8_Position, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), COUNT_MINIMUM,
			UTF8_POSITION_MAXIMUM,
		).
		Ensure()
}

// UTF8_Character_Size is non-ASCII encoded rune width.
type UTF8_Character_Size int

// UTF8_Character_Size_Invariants admits encoded rune widths only.
func UTF8_Character_Size_Invariants(
	value UTF8_Character_Size, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_3_Int(
			int(value), utf8.CHARACTER_SIZE_TWO, utf8.CHARACTER_SIZE_THREE,
			utf8.CHARACTER_SIZE_MAXIMUM,
		).
		Ensure()
}

// UTF8_Boundary is metadata candidate end.
type UTF8_Boundary int

// UTF8_Boundary_Invariants bounds metadata candidate end.
func UTF8_Boundary_Invariants(value UTF8_Boundary, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), TYPE_FLAG_FIELD_SIZE, HEADER_TEXT_SIZE_MAXIMUM,
		).
		Ensure()
}

// PAX_Payload_Size_Candidate is one complete mandatory-record size calculation.
type PAX_Payload_Size_Candidate int

// PAX_Payload_Size_Candidate_Invariants bounds preflight before storage capacity.
func PAX_Payload_Size_Candidate_Invariants(
	value PAX_Payload_Size_Candidate, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), PAX_PAYLOAD_SIZE_MINIMUM,
			PAX_PAYLOAD_SIZE_CANDIDATE_MAXIMUM,
		).
		Ensure()
}

// PAX_Time_Payload_Base_Size is generated metadata before timestamp records.
type PAX_Time_Payload_Base_Size int

// PAX_Time_Payload_Base_Size_Invariants excludes optional time-record space.
func PAX_Time_Payload_Base_Size_Invariants(
	value PAX_Time_Payload_Base_Size, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), PAX_PAYLOAD_SIZE_MINIMUM,
			PAX_TIME_PAYLOAD_BASE_SIZE_MAXIMUM,
		).
		Ensure()
}

// Destination is caller-owned decoded or encoded storage.
type Destination []byte

// Destination_Invariants bounds output storage.
func Destination_Invariants(value Destination, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, ARCHIVE_SIZE_MAXIMUM).
		Ensure()
}

// Source is caller-owned entry content.
type Source []byte

// Source_Invariants bounds entry content.
func Source_Invariants(value Source, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, ARCHIVE_SIZE_MAXIMUM).
		Ensure()
}

// Integer is complete signed TAR numeric domain.
type Integer int64

// Integer_Invariants covers signed wire and PAX values.
func Integer_Invariants(value Integer, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), int64(INTEGER_MINIMUM), int64(INTEGER_MAXIMUM)).
		Ensure()
}

// Octal_Integer is one nonnegative parsed text-octal value.
type Octal_Integer int64

// Octal_Integer_Invariants bounds widest unterminated numeric field.
func Octal_Integer_Invariants(
	value Octal_Integer, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), COUNT_MINIMUM, OCTAL_PARSE_MAXIMUM).
		Ensure()
}

// Octal_Value is one formatter input proven to fit its terminated field.
type Octal_Value int64

// Octal_Value_Invariants bounds largest terminated octal field.
func Octal_Value_Invariants(value Octal_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), COUNT_MINIMUM, LARGE_OCTAL_MAXIMUM).
		Ensure()
}

// Unsigned_Header_Checksum is one normalized unsigned header sum.
type Unsigned_Header_Checksum int64

// Unsigned_Header_Checksum_Invariants bounds every possible header byte pattern.
func Unsigned_Header_Checksum_Invariants(
	value Unsigned_Header_Checksum, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int64(
			int64(value), int64(HEADER_CHECKSUM_UNSIGNED_MINIMUM),
			int64(HEADER_CHECKSUM_UNSIGNED_MAXIMUM),
		).
		Ensure()
}

// Signed_Header_Checksum is one normalized signed-byte header sum.
type Signed_Header_Checksum int64

// Signed_Header_Checksum_Invariants bounds every possible header byte pattern.
func Signed_Header_Checksum_Invariants(
	value Signed_Header_Checksum, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int64(
			int64(value), int64(HEADER_CHECKSUM_SIGNED_MINIMUM),
			int64(HEADER_CHECKSUM_SIGNED_MAXIMUM),
		).
		Ensure()
}

// Numeric_Field is one bounded hostile TAR integer field.
type Numeric_Field []byte

// Numeric_Field_Invariants admits only the two TAR numeric field widths.
func Numeric_Field_Invariants(value Numeric_Field, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Int(len(value), MODE_FIELD_SIZE, ENTRY_SIZE_FIELD_SIZE).
		Ensure()
}

// Optional_Numeric_Field is one GNU optional timestamp field.
type Optional_Numeric_Field []byte

// Optional_Numeric_Field_Invariants fixes optional timestamp width.
func Optional_Numeric_Field_Invariants(
	value Optional_Numeric_Field, namespace aver.Namespace,
) {
	aver.Always(
		len(value) == TIMESTAMP_FIELD_SIZE,
		"GNU optional time occupies timestamp field.",
	)
}

// Numeric_Destination is one non-empty TAR integer field.
type Numeric_Destination []byte

// Numeric_Destination_Invariants admits only the two encoded field widths.
func Numeric_Destination_Invariants(
	value Numeric_Destination, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Int(len(value), MODE_FIELD_SIZE, ENTRY_SIZE_FIELD_SIZE).
		Ensure()
}

// Numeric_Field_Size is one encoded integer field width.
type Numeric_Field_Size int

// Numeric_Field_Size_Invariants admits only the two TAR numeric widths.
func Numeric_Field_Size_Invariants(
	value Numeric_Field_Size, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Int(int(value), MODE_FIELD_SIZE, ENTRY_SIZE_FIELD_SIZE).
		Ensure()
}

// Unsigned_Integer is complete unsigned numeric workspace domain.
type Unsigned_Integer uint64

// Unsigned_Integer_Invariants covers decimal formatting magnitude.
func Unsigned_Integer_Invariants(
	value Unsigned_Integer, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint64(
			uint64(value), uint64(UNSIGNED_INTEGER_MINIMUM),
			uint64(UNSIGNED_INTEGER_MAXIMUM),
		).
		Ensure()
}

// Header_Name separates a validated logical path from other metadata bytes.
type Header_Name []byte

// Header_Name_Invariants bounds one logical path.
func Header_Name_Invariants(value Header_Name, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, HEADER_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Writer_Name is one nonempty path admitted to format preflight.
type Writer_Name []byte

// Writer_Name_Invariants excludes the empty path rejected at validation.
func Writer_Name_Invariants(value Writer_Name, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), TYPE_FLAG_FIELD_SIZE, HEADER_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Header_Name_Unvalidated admits the first rejected path byte.
type Header_Name_Unvalidated []byte

// Header_Name_Unvalidated_Invariants bounds hostile path storage.
func Header_Name_Unvalidated_Invariants(
	value Header_Name_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, HEADER_TEXT_SIZE_UNVALIDATED_MAXIMUM).
		Ensure()
}

// Header_Link_Name separates a validated link target from entry path bytes.
type Header_Link_Name []byte

// Header_Link_Name_Invariants bounds one logical link target.
func Header_Link_Name_Invariants(
	value Header_Link_Name, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, HEADER_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Header_Link_Name_Unvalidated admits the first rejected link byte.
type Header_Link_Name_Unvalidated []byte

// Header_Link_Name_Unvalidated_Invariants bounds hostile link storage.
func Header_Link_Name_Unvalidated_Invariants(
	value Header_Link_Name_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, HEADER_TEXT_SIZE_UNVALIDATED_MAXIMUM).
		Ensure()
}

// Header_User_Name separates owner text from other header fields.
type Header_User_Name []byte

// Header_User_Name_Invariants bounds one owner name.
func Header_User_Name_Invariants(
	value Header_User_Name, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, HEADER_USER_NAME_SIZE_MAXIMUM).
		Ensure()
}

// Header_User_Name_Unvalidated admits the first rejected owner byte.
type Header_User_Name_Unvalidated []byte

// Header_User_Name_Unvalidated_Invariants bounds hostile owner storage.
func Header_User_Name_Unvalidated_Invariants(
	value Header_User_Name_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), COUNT_MINIMUM, HEADER_USER_NAME_SIZE_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Header_Group_Name separates group text from other header fields.
type Header_Group_Name []byte

// Header_Group_Name_Invariants bounds one group name.
func Header_Group_Name_Invariants(
	value Header_Group_Name, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, HEADER_GROUP_NAME_SIZE_MAXIMUM).
		Ensure()
}

// UTF8_Text is one bounded metadata field validated as UTF-8.
type UTF8_Text []byte

// UTF8_Text_Invariants bounds a complete metadata field.
func UTF8_Text_Invariants(value UTF8_Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, HEADER_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// GNU_Extended_Field is one value that cannot fit the fixed name field.
type GNU_Extended_Field []byte

// GNU_Extended_Field_Invariants bounds values requiring a GNU long extension.
func GNU_Extended_Field_Invariants(
	value GNU_Extended_Field, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), GNU_EXTENDED_FIELD_SIZE_MINIMUM,
			HEADER_TEXT_SIZE_MAXIMUM,
		).
		Ensure()
}

// Header_Group_Name_Unvalidated admits the first rejected group byte.
type Header_Group_Name_Unvalidated []byte

// Header_Group_Name_Unvalidated_Invariants bounds hostile group storage.
func Header_Group_Name_Unvalidated_Invariants(
	value Header_Group_Name_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), COUNT_MINIMUM, HEADER_GROUP_NAME_SIZE_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// PAX_Records separates extension grammar from ordinary metadata bytes.
type PAX_Records []byte

// PAX_Records_Invariants keeps extension parsing bounded.
func PAX_Records_Invariants(value PAX_Records, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, SPECIAL_FILE_SIZE_MAXIMUM).
		Ensure()
}

// PAX_Record_Input has at least one byte left to parse.
type PAX_Record_Input []byte

// PAX_Record_Input_Invariants excludes empty parser calls.
func PAX_Record_Input_Invariants(
	value PAX_Record_Input, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), TYPE_FLAG_FIELD_SIZE, SPECIAL_FILE_SIZE_MAXIMUM).
		Ensure()
}

// PAX_Parsed_Record_Size is one complete record relative to remaining input.
type PAX_Parsed_Record_Size int

// PAX_Parsed_Record_Size_Invariants bounds one trusted record.
func PAX_Parsed_Record_Size_Invariants(
	value PAX_Parsed_Record_Size, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), PAX_RECORD_SIZE_MINIMUM, SPECIAL_FILE_SIZE_MAXIMUM,
		).
		Ensure()
}

// PAX_Valid_Records is one nonempty, grammar-checked local metadata payload.
type PAX_Valid_Records []byte

// PAX_Valid_Records_Invariants excludes lengths no complete record can occupy.
func PAX_Valid_Records_Invariants(
	value PAX_Valid_Records, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), PAX_RECORD_SIZE_MINIMUM, SPECIAL_FILE_SIZE_MAXIMUM).
		Ensure()
}

// PAX_Header_Records is one nonempty caller record set that leaves mandatory fields.
type PAX_Header_Records []byte

// PAX_Header_Records_Invariants bounds one writer-supplied record set.
func PAX_Header_Records_Invariants(
	value PAX_Header_Records, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), PAX_RECORD_SIZE_MINIMUM, PAX_HEADER_RECORDS_SIZE_MAXIMUM,
		).
		Ensure()
}

// PAX_Key is one record key bounded by complete record grammar.
type PAX_Key []byte

// PAX_Key_Invariants protects record scans and literal comparisons.
func PAX_Key_Invariants(value PAX_Key, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, PAX_KEY_SIZE_MAXIMUM).
		Ensure()
}

// PAX_Parsed_Key is one nonempty key from a complete record.
type PAX_Parsed_Key []byte

// PAX_Parsed_Key_Invariants excludes the empty-key parser failure.
func PAX_Parsed_Key_Invariants(
	value PAX_Parsed_Key, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), TYPE_FLAG_FIELD_SIZE, PAX_KEY_SIZE_MAXIMUM).
		Ensure()
}

// PAX_Key_Name is one supported PAX keyword.
type PAX_Key_Name string

// PAX_Key_Name_Invariants bounds shortest and longest supported keywords.
func PAX_Key_Name_Invariants(value PAX_Key_Name, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), PAX_KEY_NAME_SIZE_MINIMUM, PAX_KEY_NAME_SIZE_MAXIMUM,
		).
		Ensure()
}

// Wire_Literal is one fixed marker literal.
type Wire_Literal string

// Wire_Literal_Invariants matches fixed marker width bounds.
func Wire_Literal_Invariants(value Wire_Literal, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), VERSION_FIELD_SIZE, MAGIC_FIELD_SIZE).
		Ensure()
}

// Wire_Literal_Offset locates one fixed marker inside a header block.
type Wire_Literal_Offset int

// Wire_Literal_Offset_Invariants admits only fields compared with fixed markers.
func Wire_Literal_Offset_Invariants(
	value Wire_Literal_Offset, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_3_Int(
			int(value), MAGIC_FIELD_OFFSET, VERSION_FIELD_OFFSET,
			STAR_TRAILER_FIELD_OFFSET,
		).
		Ensure()
}

// PAX_Value is one record value bounded by complete record grammar.
type PAX_Value []byte

// PAX_Value_Invariants protects text and number parsing.
func PAX_Value_Invariants(value PAX_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, PAX_VALUE_SIZE_MAXIMUM).
		Ensure()
}

// Decimal_Candidate is one supported numeric value or empty timestamp prefix.
type Decimal_Candidate []byte

// Decimal_Candidate_Invariants bounds numeric parser input.
func Decimal_Candidate_Invariants(
	value Decimal_Candidate, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, PAX_NUMERIC_VALUE_SIZE_MAXIMUM).
		Ensure()
}

// PAX_Timestamp_Candidate is one nonempty timestamp record value.
type PAX_Timestamp_Candidate []byte

// PAX_Timestamp_Candidate_Invariants bounds timestamp parser input.
func PAX_Timestamp_Candidate_Invariants(
	value PAX_Timestamp_Candidate, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), TYPE_FLAG_FIELD_SIZE, PAX_TIMESTAMP_VALUE_SIZE_MAXIMUM,
		).
		Ensure()
}

// Decimal_Destination exactly holds one signed decimal value.
type Decimal_Destination []byte

// Decimal_Destination_Invariants bounds signed decimal width.
func Decimal_Destination_Invariants(
	value Decimal_Destination, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), TYPE_FLAG_FIELD_SIZE, INTEGER_DECIMAL_SIZE_MAXIMUM,
		).
		Ensure()
}

// Unsigned_Decimal_Destination exactly holds one decimal magnitude.
type Unsigned_Decimal_Destination []byte

// Unsigned_Decimal_Destination_Invariants bounds magnitude width.
func Unsigned_Decimal_Destination_Invariants(
	value Unsigned_Decimal_Destination, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), TYPE_FLAG_FIELD_SIZE, DECIMAL_MAGNITUDE_SIZE_MAXIMUM,
		).
		Ensure()
}

// Timestamp_Destination exactly holds one PAX timestamp value.
type Timestamp_Destination []byte

// Timestamp_Destination_Invariants bounds timestamp text width.
func Timestamp_Destination_Invariants(
	value Timestamp_Destination, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), TYPE_FLAG_FIELD_SIZE, TIMESTAMP_VALUE_SIZE_MAXIMUM).
		Ensure()
}

// PAX_Payload_Destination is one complete generated metadata body.
type PAX_Payload_Destination []byte

// PAX_Payload_Destination_Invariants bounds mandatory through maximum payload.
func PAX_Payload_Destination_Invariants(
	value PAX_Payload_Destination, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), PAX_PAYLOAD_SIZE_MINIMUM, SPECIAL_FILE_SIZE_MAXIMUM).
		Ensure()
}

// PAX_Header_Field_Size is caller records plus generated non-time records.
type PAX_Header_Field_Size int

// PAX_Header_Field_Size_Invariants bounds mandatory through complete payload.
func PAX_Header_Field_Size_Invariants(
	value PAX_Header_Field_Size, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), PAX_PAYLOAD_SIZE_MINIMUM, SPECIAL_FILE_SIZE_MAXIMUM).
		Ensure()
}

// PAX_Time_Size is one to three generated timestamp records.
type PAX_Time_Size int

// PAX_Time_Size_Invariants bounds a nonempty generated timestamp span.
func PAX_Time_Size_Invariants(value PAX_Time_Size, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), PAX_TIMESTAMP_RECORD_SIZE_MINIMUM, PAX_TIME_SIZE_MAXIMUM,
		).
		Ensure()
}

// PAX_Time_Destination is nonempty remaining storage for generated timestamps.
type PAX_Time_Destination []byte

// PAX_Time_Destination_Invariants bounds one through three timestamp records.
func PAX_Time_Destination_Invariants(
	value PAX_Time_Destination, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), PAX_TIMESTAMP_RECORD_SIZE_MINIMUM, PAX_TIME_SIZE_MAXIMUM,
		).
		Ensure()
}

// PAX_Text_Value is one nonempty generated text-record value.
type PAX_Text_Value []byte

// PAX_Text_Value_Invariants bounds every writer-generated text value.
func PAX_Text_Value_Invariants(
	value PAX_Text_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), TYPE_FLAG_FIELD_SIZE, PAX_NAME_SIZE_MAXIMUM).
		Ensure()
}

// PAX_Record_Key_Size is one supported generated-key width.
type PAX_Record_Key_Size int

// PAX_Record_Key_Size_Invariants bounds decimal through link-path keys.
func PAX_Record_Key_Size_Invariants(
	value PAX_Record_Key_Size, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), PAX_KEY_NAME_SIZE_MINIMUM, PAX_KEY_NAME_SIZE_MAXIMUM,
		).
		Ensure()
}

// PAX_Record_Value_Size is one nonempty generated value width.
type PAX_Record_Value_Size int

// PAX_Record_Value_Size_Invariants bounds one generated record value.
func PAX_Record_Value_Size_Invariants(
	value PAX_Record_Value_Size, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), TYPE_FLAG_FIELD_SIZE, HEADER_TEXT_SIZE_MAXIMUM,
		).
		Ensure()
}

// PAX_Record_Size_Candidate includes records rejected by aggregate preflight.
type PAX_Record_Size_Candidate int

// PAX_Record_Size_Candidate_Invariants bounds every independently bounded record.
func PAX_Record_Size_Candidate_Invariants(
	value PAX_Record_Size_Candidate, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), PAX_GENERATED_RECORD_SIZE_MINIMUM,
			PAX_LINK_RECORD_SIZE_CANDIDATE_MAXIMUM,
		).
		Ensure()
}

// PAX_Text_Record_Size is one complete generated text record.
type PAX_Text_Record_Size int

// PAX_Text_Record_Size_Invariants bounds shortest through largest path record.
func PAX_Text_Record_Size_Invariants(
	value PAX_Text_Record_Size, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), PAX_TEXT_RECORD_SIZE_MINIMUM,
			PAX_PATH_WRITE_RECORD_SIZE_MAXIMUM,
		).
		Ensure()
}

// PAX_Decimal_Record_Size is one complete generated integer record.
type PAX_Decimal_Record_Size int

// PAX_Decimal_Record_Size_Invariants bounds identifiers and logical size.
func PAX_Decimal_Record_Size_Invariants(
	value PAX_Decimal_Record_Size, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), PAX_GENERATED_RECORD_SIZE_MINIMUM,
			PAX_IDENTIFIER_RECORD_SIZE_MAXIMUM,
		).
		Ensure()
}

// PAX_Timestamp_Record_Size is one complete generated time record.
type PAX_Timestamp_Record_Size int

// PAX_Timestamp_Record_Size_Invariants bounds shortest through widest time.
func PAX_Timestamp_Record_Size_Invariants(
	value PAX_Timestamp_Record_Size, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), PAX_TIMESTAMP_RECORD_SIZE_MINIMUM,
			PAX_TIMESTAMP_RECORD_SIZE_MAXIMUM,
		).
		Ensure()
}

// PAX_Text_Record_Destination is remaining storage before a text record.
type PAX_Text_Record_Destination []byte

// PAX_Text_Record_Destination_Invariants bounds valid text write windows.
func PAX_Text_Record_Destination_Invariants(
	value PAX_Text_Record_Destination, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), PAX_TEXT_RECORD_DESTINATION_SIZE_MINIMUM,
			SPECIAL_FILE_SIZE_MAXIMUM,
		).
		Ensure()
}

// PAX_Decimal_Record_Destination is remaining storage after mandatory path.
type PAX_Decimal_Record_Destination []byte

// PAX_Decimal_Record_Destination_Invariants bounds integer write windows.
func PAX_Decimal_Record_Destination_Invariants(
	value PAX_Decimal_Record_Destination, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), PAX_GENERATED_RECORD_SIZE_MINIMUM,
			PAX_DECIMAL_RECORD_DESTINATION_SIZE_MAXIMUM,
		).
		Ensure()
}

// PAX_Timestamp_Record_Destination is remaining timestamp-only storage.
type PAX_Timestamp_Record_Destination []byte

// PAX_Timestamp_Record_Destination_Invariants bounds time write windows.
func PAX_Timestamp_Record_Destination_Invariants(
	value PAX_Timestamp_Record_Destination, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), PAX_TIMESTAMP_RECORD_SIZE_MINIMUM, PAX_TIME_SIZE_MAXIMUM,
		).
		Ensure()
}

// PAX_Text_Key is one generated text-record key.
type PAX_Text_Key string

// PAX_Text_Key_Invariants bounds path through link-path keyword widths.
func PAX_Text_Key_Invariants(value PAX_Text_Key, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), PAX_TEXT_KEY_SIZE_MINIMUM, PAX_TEXT_KEY_SIZE_MAXIMUM).
		Ensure()
}

// PAX_Decimal_Key is one generated integer-record key.
type PAX_Decimal_Key string

// PAX_Decimal_Key_Invariants bounds owner identifiers through size keyword.
func PAX_Decimal_Key_Invariants(
	value PAX_Decimal_Key, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Int(len(value), PAX_DECIMAL_KEY_SIZE_MINIMUM, PAX_DECIMAL_KEY_SIZE_MAXIMUM).
		Ensure()
}

// PAX_Time_Key is one generated timestamp-record key.
type PAX_Time_Key string

// PAX_Time_Key_Invariants fixes the shared timestamp keyword width.
func PAX_Time_Key_Invariants(value PAX_Time_Key, namespace aver.Namespace) {
	aver.Always(
		len(value) == PAX_TIME_KEY_SIZE,
		"Generated PAX timestamp keys share one width.",
	)
}

// PAX_Records_Unvalidated admits the first rejected extension byte.
type PAX_Records_Unvalidated []byte

// PAX_Records_Unvalidated_Invariants bounds hostile extension storage.
func PAX_Records_Unvalidated_Invariants(
	value PAX_Records_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, SPECIAL_FILE_SIZE_UNVALIDATED_MAXIMUM).
		Ensure()
}

// Entry_Size is one bounded logical content size.
type Entry_Size int64

// Entry_Size_Invariants protects caller destination arithmetic.
func Entry_Size_Invariants(value Entry_Size, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), COUNT_MINIMUM, ARCHIVE_SIZE_MAXIMUM).
		Ensure()
}

// Entry_Size_Unvalidated preserves every caller size until validation.
type Entry_Size_Unvalidated int64

// Entry_Size_Unvalidated_Invariants covers the complete hostile integer domain.
func Entry_Size_Unvalidated_Invariants(
	value Entry_Size_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int64(
			int64(value), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM,
		).
		Ensure()
}

// File_Mode preserves the complete signed TAR numeric field.
type File_Mode int64

// File_Mode_Invariants covers octal and base-256 encodings.
func File_Mode_Invariants(value File_Mode, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), int64(INTEGER_MINIMUM), int64(INTEGER_MAXIMUM)).
		Ensure()
}

// User_Identifier preserves the complete signed owner field.
type User_Identifier int64

// User_Identifier_Invariants covers octal, base-256, and PAX forms.
func User_Identifier_Invariants(
	value User_Identifier, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), int64(INTEGER_MINIMUM), int64(INTEGER_MAXIMUM)).
		Ensure()
}

// Group_Identifier preserves the complete signed group field.
type Group_Identifier int64

// Group_Identifier_Invariants covers octal, base-256, and PAX forms.
func Group_Identifier_Invariants(
	value Group_Identifier, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), int64(INTEGER_MINIMUM), int64(INTEGER_MAXIMUM)).
		Ensure()
}

// Device_Major preserves a complete signed device class field.
type Device_Major int64

// Device_Major_Invariants covers octal and base-256 forms.
func Device_Major_Invariants(value Device_Major, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), int64(INTEGER_MINIMUM), int64(INTEGER_MAXIMUM)).
		Ensure()
}

// Device_Minor preserves a complete signed device instance field.
type Device_Minor int64

// Device_Minor_Invariants covers octal and base-256 forms.
func Device_Minor_Invariants(value Device_Minor, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), int64(INTEGER_MINIMUM), int64(INTEGER_MAXIMUM)).
		Ensure()
}

// Access_Seconds separates access time from other signed header values.
type Access_Seconds int64

// Access_Seconds_Invariants covers complete Unix time.
func Access_Seconds_Invariants(
	value Access_Seconds, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), int64(INTEGER_MINIMUM), int64(INTEGER_MAXIMUM)).
		Ensure()
}

// Access_Nanosecond_Count is one normalized access-time fraction.
type Access_Nanosecond_Count int32

// Access_Nanosecond_Count_Invariants bounds one fractional second.
func Access_Nanosecond_Count_Invariants(
	value Access_Nanosecond_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int32(
			int32(value), int32(TIMESTAMP_NANOSECOND_MINIMUM),
			int32(TIMESTAMP_NANOSECOND_MAXIMUM),
		).
		Ensure()
}

// Access_Nanosecond_Count_Unvalidated preserves every hostile fraction.
type Access_Nanosecond_Count_Unvalidated int32

// Access_Nanosecond_Count_Unvalidated_Invariants covers the complete scalar domain.
func Access_Nanosecond_Count_Unvalidated_Invariants(
	value Access_Nanosecond_Count_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), bits.INTEGER_32_MINIMUM, bits.INTEGER_32_MAXIMUM).
		Ensure()
}

// Access_Time_Set keeps absent access time distinct from Unix epoch.
type Access_Time_Set bool

// Access_Time_Set_Invariants covers absent and present metadata.
func Access_Time_Set_Invariants(
	value Access_Time_Set, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Access time is present.").
		Ensure()
}

// Access_Timestamp retains optional access time without sharing modification state.
type Access_Timestamp struct {
	// Seconds holds Unix epoch offset.
	Seconds Access_Seconds
	// Nanoseconds holds fractional second.
	Nanoseconds Access_Nanosecond_Count
	// Set distinguishes Unix epoch from absent metadata.
	Set Access_Time_Set
}

// Access_Timestamp_Invariants composes one normalized optional instant.
func Access_Timestamp_Invariants(
	value Access_Timestamp, namespace aver.Namespace,
) {
	Access_Seconds_Invariants(value.Seconds, namespace)
	Access_Nanosecond_Count_Invariants(value.Nanoseconds, namespace)
	Access_Time_Set_Invariants(value.Set, namespace)
}

// Access_Timestamp_Unvalidated retains hostile access-time fractions.
type Access_Timestamp_Unvalidated struct {
	// Seconds preserves the complete caller epoch offset.
	Seconds Access_Seconds
	// Nanoseconds remains hostile until Header_Validate.
	Nanoseconds Access_Nanosecond_Count_Unvalidated
	// Set distinguishes Unix epoch from absent metadata.
	Set Access_Time_Set
}

// Access_Timestamp_Unvalidated_Invariants composes hostile access time.
func Access_Timestamp_Unvalidated_Invariants(
	value Access_Timestamp_Unvalidated, namespace aver.Namespace,
) {
	Access_Seconds_Invariants(value.Seconds, namespace)
	Access_Nanosecond_Count_Unvalidated_Invariants(value.Nanoseconds, namespace)
	Access_Time_Set_Invariants(value.Set, namespace)
}

// Change_Seconds separates metadata change time from other signed fields.
type Change_Seconds int64

// Change_Seconds_Invariants covers complete Unix time.
func Change_Seconds_Invariants(
	value Change_Seconds, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), int64(INTEGER_MINIMUM), int64(INTEGER_MAXIMUM)).
		Ensure()
}

// Change_Nanosecond_Count is one normalized metadata-change fraction.
type Change_Nanosecond_Count int32

// Change_Nanosecond_Count_Invariants bounds one fractional second.
func Change_Nanosecond_Count_Invariants(
	value Change_Nanosecond_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int32(
			int32(value), int32(TIMESTAMP_NANOSECOND_MINIMUM),
			int32(TIMESTAMP_NANOSECOND_MAXIMUM),
		).
		Ensure()
}

// Change_Nanosecond_Count_Unvalidated preserves every hostile fraction.
type Change_Nanosecond_Count_Unvalidated int32

// Change_Nanosecond_Count_Unvalidated_Invariants covers the complete scalar domain.
func Change_Nanosecond_Count_Unvalidated_Invariants(
	value Change_Nanosecond_Count_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), bits.INTEGER_32_MINIMUM, bits.INTEGER_32_MAXIMUM).
		Ensure()
}

// Change_Time_Set keeps absent metadata time distinct from Unix epoch.
type Change_Time_Set bool

// Change_Time_Set_Invariants covers absent and present metadata.
func Change_Time_Set_Invariants(
	value Change_Time_Set, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Metadata change time is present.").
		Ensure()
}

// Change_Timestamp retains optional metadata time without sharing access state.
type Change_Timestamp struct {
	// Seconds holds Unix epoch offset.
	Seconds Change_Seconds
	// Nanoseconds holds fractional second.
	Nanoseconds Change_Nanosecond_Count
	// Set distinguishes Unix epoch from absent metadata.
	Set Change_Time_Set
}

// Change_Timestamp_Invariants composes one normalized optional instant.
func Change_Timestamp_Invariants(
	value Change_Timestamp, namespace aver.Namespace,
) {
	Change_Seconds_Invariants(value.Seconds, namespace)
	Change_Nanosecond_Count_Invariants(value.Nanoseconds, namespace)
	Change_Time_Set_Invariants(value.Set, namespace)
}

// Change_Timestamp_Unvalidated retains hostile metadata-time fractions.
type Change_Timestamp_Unvalidated struct {
	// Seconds preserves the complete caller epoch offset.
	Seconds Change_Seconds
	// Nanoseconds remains hostile until Header_Validate.
	Nanoseconds Change_Nanosecond_Count_Unvalidated
	// Set distinguishes Unix epoch from absent metadata.
	Set Change_Time_Set
}

// Change_Timestamp_Unvalidated_Invariants composes hostile metadata time.
func Change_Timestamp_Unvalidated_Invariants(
	value Change_Timestamp_Unvalidated, namespace aver.Namespace,
) {
	Change_Seconds_Invariants(value.Seconds, namespace)
	Change_Nanosecond_Count_Unvalidated_Invariants(value.Nanoseconds, namespace)
	Change_Time_Set_Invariants(value.Set, namespace)
}

// Nanosecond_Count is one normalized fractional second.
type Nanosecond_Count int32

// Nanosecond_Count_Invariants bounds one fractional second.
func Nanosecond_Count_Invariants(
	value Nanosecond_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int32(
			int32(value), int32(TIMESTAMP_NANOSECOND_MINIMUM),
			int32(TIMESTAMP_NANOSECOND_MAXIMUM),
		).
		Ensure()
}

// Nanosecond_Count_Unvalidated preserves every hostile modification fraction.
type Nanosecond_Count_Unvalidated int32

// Nanosecond_Count_Unvalidated_Invariants covers the complete scalar domain.
func Nanosecond_Count_Unvalidated_Invariants(
	value Nanosecond_Count_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), bits.INTEGER_32_MINIMUM, bits.INTEGER_32_MAXIMUM).
		Ensure()
}

// Type_Flag identifies one TAR entry kind.
type Type_Flag byte

// Type_Flag_Invariants covers complete extension byte domain.
func Type_Flag_Invariants(value Type_Flag, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), uint8(TYPE_FLAG_MINIMUM), uint8(TYPE_FLAG_MAXIMUM)).
		Ensure()
}

// Wire_Type_Flag excludes the legacy zero byte normalized during parsing.
type Wire_Type_Flag byte

// Wire_Type_Flag_Invariants bounds normalized fixed-header entry kinds.
func Wire_Type_Flag_Invariants(
	value Wire_Type_Flag, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value), uint8(TYPE_FLAG_FIELD_SIZE), uint8(TYPE_FLAG_MAXIMUM),
		).
		Ensure()
}

// GNU_Long_Type_Flag distinguishes long path from long link metadata.
type GNU_Long_Type_Flag byte

// GNU_Long_Type_Flag_Invariants keeps GNU metadata dispatch exact.
func GNU_Long_Type_Flag_Invariants(
	value GNU_Long_Type_Flag, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(TYPE_GNU_LONG_LINK), uint8(TYPE_GNU_LONG_NAME),
		).
		Ensure()
}

// Metadata_Type_Flag contains every writer-generated metadata entry kind.
type Metadata_Type_Flag byte

// Metadata_Type_Flag_Invariants keeps generated metadata families exact.
func Metadata_Type_Flag_Invariants(
	value Metadata_Type_Flag, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(TYPE_GNU_LONG_LINK), uint8(TYPE_GNU_LONG_NAME),
			uint8(TYPE_PAX_LOCAL),
		).
		Ensure()
}

// Metadata_Size is one generated metadata payload.
type Metadata_Size int64

// Metadata_Size_Invariants bounds PAX or GNU metadata bodies.
func Metadata_Size_Invariants(value Metadata_Size, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(
			int64(value), int64(PAX_PAYLOAD_SIZE_MINIMUM),
			int64(SPECIAL_FILE_SIZE_MAXIMUM),
		).
		Ensure()
}

// Timestamp preserves seconds, nanoseconds, and zero-value presence without ambient clock.
type Timestamp struct {
	// Seconds holds Unix epoch offset.
	Seconds Integer
	// Nanoseconds holds fractional second.
	Nanoseconds Nanosecond_Count
	// Set distinguishes Unix epoch from absent metadata.
	Set bytes.Boolean
}

// Timestamp_Invariants composes signed instant, fraction, and presence.
func Timestamp_Invariants(value Timestamp, namespace aver.Namespace) {
	Integer_Invariants(value.Seconds, namespace)
	Nanosecond_Count_Invariants(value.Nanoseconds, namespace)
	bytes.Boolean_Invariants(value.Set, namespace)
}

// Timestamp_Unvalidated retains hostile modification-time fractions.
type Timestamp_Unvalidated struct {
	// Seconds preserves the complete caller epoch offset.
	Seconds Integer
	// Nanoseconds remains hostile until Header_Validate.
	Nanoseconds Nanosecond_Count_Unvalidated
	// Set distinguishes Unix epoch from absent metadata.
	Set bytes.Boolean
}

// Timestamp_Unvalidated_Invariants composes hostile modification time.
func Timestamp_Unvalidated_Invariants(
	value Timestamp_Unvalidated, namespace aver.Namespace,
) {
	Integer_Invariants(value.Seconds, namespace)
	Nanosecond_Count_Unvalidated_Invariants(value.Nanoseconds, namespace)
	bytes.Boolean_Invariants(value.Set, namespace)
}

// Header carries one logical TAR entry without owning field storage.
type Header struct {
	// Format identifies decoded or requested wire family.
	Format Format
	// Type_Flag selects file kind.
	Type_Flag Type_Flag
	// Name aliases caller field storage.
	Name Header_Name
	// Link_Name aliases caller field storage.
	Link_Name Header_Link_Name
	// Size is logical content size.
	Size Entry_Size
	// Mode holds permission and mode bits.
	Mode File_Mode
	// User_Identifier identifies owner.
	User_Identifier User_Identifier
	// Group_Identifier identifies owner group.
	Group_Identifier Group_Identifier
	// User_Name aliases caller field storage.
	User_Name Header_User_Name
	// Group_Name aliases caller field storage.
	Group_Name Header_Group_Name
	// Modification_Time records content change.
	Modification_Time Timestamp
	// Access_Time records last access where format supports it.
	Access_Time Access_Timestamp
	// Change_Time records metadata change where format supports it.
	Change_Time Change_Timestamp
	// Device_Major identifies device class.
	Device_Major Device_Major
	// Device_Minor identifies device instance.
	Device_Minor Device_Minor
}

// Header_Invariants keeps bounded fields and normalized times together.
func Header_Invariants(value Header, namespace aver.Namespace) {
	Format_Invariants(value.Format, namespace)
	Type_Flag_Invariants(value.Type_Flag, namespace)
	Header_Name_Invariants(value.Name, namespace)
	Header_Link_Name_Invariants(value.Link_Name, namespace)
	Entry_Size_Invariants(value.Size, namespace)
	File_Mode_Invariants(value.Mode, namespace)
	User_Identifier_Invariants(value.User_Identifier, namespace)
	Group_Identifier_Invariants(value.Group_Identifier, namespace)
	Header_User_Name_Invariants(value.User_Name, namespace)
	Header_Group_Name_Invariants(value.Group_Name, namespace)
	Timestamp_Invariants(value.Modification_Time, namespace)
	Access_Timestamp_Invariants(value.Access_Time, namespace)
	Change_Timestamp_Invariants(value.Change_Time, namespace)
	Device_Major_Invariants(value.Device_Major, namespace)
	Device_Minor_Invariants(value.Device_Minor, namespace)
}

// Header_Unvalidated keeps hostile caller fields outside Writer state.
type Header_Unvalidated struct {
	// Format remains hostile until Header_Validate.
	Format Format_Unvalidated
	// Type_Flag preserves the complete extension byte domain.
	Type_Flag Type_Flag
	// Name remains hostile until Header_Validate.
	Name Header_Name_Unvalidated
	// Link_Name remains hostile until Header_Validate.
	Link_Name Header_Link_Name_Unvalidated
	// Size remains hostile until Header_Validate.
	Size Entry_Size_Unvalidated
	// Mode preserves the complete signed wire domain.
	Mode File_Mode
	// User_Identifier preserves the complete signed wire domain.
	User_Identifier User_Identifier
	// Group_Identifier preserves the complete signed wire domain.
	Group_Identifier Group_Identifier
	// User_Name remains hostile until Header_Validate.
	User_Name Header_User_Name_Unvalidated
	// Group_Name remains hostile until Header_Validate.
	Group_Name Header_Group_Name_Unvalidated
	// Modification_Time remains hostile until Header_Validate.
	Modification_Time Timestamp_Unvalidated
	// Access_Time remains hostile until Header_Validate.
	Access_Time Access_Timestamp_Unvalidated
	// Change_Time remains hostile until Header_Validate.
	Change_Time Change_Timestamp_Unvalidated
	// Device_Major preserves the complete signed wire domain.
	Device_Major Device_Major
	// Device_Minor preserves the complete signed wire domain.
	Device_Minor Device_Minor
	// PAX_Records remains hostile until Header_Validate.
	PAX_Records PAX_Records_Unvalidated
}

// Header_Unvalidated_Invariants composes the complete caller domain.
func Header_Unvalidated_Invariants(
	value *Header_Unvalidated, namespace aver.Namespace,
) {
	aver.Always(value != nil, "Unvalidated TAR Header state exists.")
	Format_Unvalidated_Invariants(value.Format, namespace)
	Type_Flag_Invariants(value.Type_Flag, namespace)
	Header_Name_Unvalidated_Invariants(value.Name, namespace)
	Header_Link_Name_Unvalidated_Invariants(value.Link_Name, namespace)
	Entry_Size_Unvalidated_Invariants(value.Size, namespace)
	File_Mode_Invariants(value.Mode, namespace)
	User_Identifier_Invariants(value.User_Identifier, namespace)
	Group_Identifier_Invariants(value.Group_Identifier, namespace)
	Header_User_Name_Unvalidated_Invariants(value.User_Name, namespace)
	Header_Group_Name_Unvalidated_Invariants(value.Group_Name, namespace)
	Timestamp_Unvalidated_Invariants(value.Modification_Time, namespace)
	Access_Timestamp_Unvalidated_Invariants(value.Access_Time, namespace)
	Change_Timestamp_Unvalidated_Invariants(value.Change_Time, namespace)
	Device_Major_Invariants(value.Device_Major, namespace)
	Device_Minor_Invariants(value.Device_Minor, namespace)
	PAX_Records_Unvalidated_Invariants(value.PAX_Records, namespace)
}

// Header_Validate is the only boundary where hostile Writer fields enter TAR state.
func Header_Validate(
	value *Header_Unvalidated,
) (header Header, status Header_Validation_Status) {
	defer func() {
		Header_Invariants(header, "Header_Validate.header")
		Header_Validation_Status_Invariants(status, "Header_Validate.status")
	}()
	Header_Unvalidated_Invariants(value, "Header_Validate.value")
	bounds_status := header_bounds_status(value)
	if bounds_status != Header_Validation_Status(STATUS_OK) {
		return Header{}, bounds_status
	}
	if !header_times_valid(
		value.Modification_Time, value.Access_Time, value.Change_Time,
	) {
		return Header{}, STATUS_INPUT_INVALID
	}
	if len(value.Name) == 0 {
		return Header{}, STATUS_INPUT_INVALID
	}
	if contains_nul(NUL_Checked_Text(value.Name)) {
		return Header{}, STATUS_INPUT_INVALID
	}
	if contains_nul(NUL_Checked_Text(value.Link_Name)) {
		return Header{}, STATUS_INPUT_INVALID
	}
	if contains_nul(NUL_Checked_Text(value.User_Name)) {
		return Header{}, STATUS_INPUT_INVALID
	}
	if contains_nul(NUL_Checked_Text(value.Group_Name)) {
		return Header{}, STATUS_INPUT_INVALID
	}
	if len(value.PAX_Records) != 0 {
		if !pax_records_valid(PAX_Records(value.PAX_Records)) {
			return Header{}, STATUS_INPUT_INVALID
		}
	}
	header = Header{
		Format: Format(value.Format), Type_Flag: value.Type_Flag,
		Name: Header_Name(value.Name), Link_Name: Header_Link_Name(value.Link_Name),
		Size: Entry_Size(value.Size), Mode: value.Mode,
		User_Identifier:  value.User_Identifier,
		Group_Identifier: value.Group_Identifier,
		User_Name:        Header_User_Name(value.User_Name),
		Group_Name:       Header_Group_Name(value.Group_Name),
		Modification_Time: Timestamp{
			Seconds:     value.Modification_Time.Seconds,
			Nanoseconds: Nanosecond_Count(value.Modification_Time.Nanoseconds),
			Set:         value.Modification_Time.Set,
		},
		Access_Time: Access_Timestamp{
			Seconds:     value.Access_Time.Seconds,
			Nanoseconds: Access_Nanosecond_Count(value.Access_Time.Nanoseconds),
			Set:         value.Access_Time.Set,
		},
		Change_Time: Change_Timestamp{
			Seconds:     value.Change_Time.Seconds,
			Nanoseconds: Change_Nanosecond_Count(value.Change_Time.Nanoseconds),
			Set:         value.Change_Time.Set,
		},
		Device_Major: value.Device_Major, Device_Minor: value.Device_Minor,
	}
	return header, STATUS_OK
}

func header_bounds_status(
	value *Header_Unvalidated,
) (status Header_Validation_Status) {
	defer func() {
		Header_Validation_Status_Invariants(status, "header_bounds_status.status")
	}()
	Header_Unvalidated_Invariants(value, "header_bounds_status.value")
	if uint8(value.Format) > FORMAT_MAXIMUM {
		return Header_Validation_Status(STATUS_FORMAT_UNSUPPORTED)
	}
	if len(value.Name) > HEADER_TEXT_SIZE_MAXIMUM {
		return Header_Validation_Status(STATUS_FIELD_TOO_LONG)
	}
	if len(value.Link_Name) > HEADER_TEXT_SIZE_MAXIMUM {
		return Header_Validation_Status(STATUS_FIELD_TOO_LONG)
	}
	if len(value.User_Name) > HEADER_USER_NAME_SIZE_MAXIMUM {
		return Header_Validation_Status(STATUS_FIELD_TOO_LONG)
	}
	if len(value.Group_Name) > HEADER_GROUP_NAME_SIZE_MAXIMUM {
		return Header_Validation_Status(STATUS_FIELD_TOO_LONG)
	}
	if len(value.PAX_Records) > SPECIAL_FILE_SIZE_MAXIMUM {
		return Header_Validation_Status(STATUS_FIELD_TOO_LONG)
	}
	if value.Size < 0 {
		return Header_Validation_Status(STATUS_INPUT_INVALID)
	}
	if value.Size > ARCHIVE_SIZE_MAXIMUM {
		return Header_Validation_Status(STATUS_INPUT_INVALID)
	}
	return Header_Validation_Status(STATUS_OK)
}

func header_times_valid(
	modified Timestamp_Unvalidated,
	accessed Access_Timestamp_Unvalidated,
	changed Change_Timestamp_Unvalidated,
) (valid bytes.Boolean) {
	defer func() { bytes.Boolean_Invariants(valid, "header_times_valid.valid") }()
	Timestamp_Unvalidated_Invariants(modified, "header_times_valid.modified")
	Access_Timestamp_Unvalidated_Invariants(accessed, "header_times_valid.accessed")
	Change_Timestamp_Unvalidated_Invariants(changed, "header_times_valid.changed")
	if modified.Nanoseconds <
		Nanosecond_Count_Unvalidated(TIMESTAMP_NANOSECOND_MINIMUM) {
		return false
	}
	if modified.Nanoseconds >
		Nanosecond_Count_Unvalidated(TIMESTAMP_NANOSECOND_MAXIMUM) {
		return false
	}
	if accessed.Nanoseconds <
		Access_Nanosecond_Count_Unvalidated(TIMESTAMP_NANOSECOND_MINIMUM) {
		return false
	}
	if accessed.Nanoseconds >
		Access_Nanosecond_Count_Unvalidated(TIMESTAMP_NANOSECOND_MAXIMUM) {
		return false
	}
	if changed.Nanoseconds <
		Change_Nanosecond_Count_Unvalidated(TIMESTAMP_NANOSECOND_MINIMUM) {
		return false
	}
	return changed.Nanoseconds <=
		Change_Nanosecond_Count_Unvalidated(TIMESTAMP_NANOSECOND_MAXIMUM)
}

// Header_Name_Destination is caller-owned decoded path storage.
type Header_Name_Destination []byte

// Header_Name_Destination_Invariants bounds decoded path storage.
func Header_Name_Destination_Invariants(
	value Header_Name_Destination, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, HEADER_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Header_Link_Name_Destination is caller-owned decoded link storage.
type Header_Link_Name_Destination []byte

// Header_Link_Name_Destination_Invariants bounds decoded link storage.
func Header_Link_Name_Destination_Invariants(
	value Header_Link_Name_Destination, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, HEADER_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Header_User_Name_Destination is caller-owned decoded owner storage.
type Header_User_Name_Destination []byte

// Header_User_Name_Destination_Invariants bounds decoded owner storage.
func Header_User_Name_Destination_Invariants(
	value Header_User_Name_Destination, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, HEADER_USER_NAME_SIZE_MAXIMUM).
		Ensure()
}

// Header_Group_Name_Destination is caller-owned decoded group storage.
type Header_Group_Name_Destination []byte

// Header_Group_Name_Destination_Invariants bounds decoded group storage.
func Header_Group_Name_Destination_Invariants(
	value Header_Group_Name_Destination, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, HEADER_GROUP_NAME_SIZE_MAXIMUM).
		Ensure()
}

// Header_Storage supplies mutable field output.
type Header_Storage struct {
	// Name receives logical path.
	Name Header_Name_Destination
	// Link_Name receives logical link path.
	Link_Name Header_Link_Name_Destination
	// User_Name receives owner name.
	User_Name Header_User_Name_Destination
	// Group_Name receives owner group.
	Group_Name Header_Group_Name_Destination
}

// Header_Storage_Invariants keeps caller fields inside metadata bounds.
func Header_Storage_Invariants(
	value Header_Storage, namespace aver.Namespace,
) {
	Header_Name_Destination_Invariants(value.Name, namespace)
	Header_Link_Name_Destination_Invariants(value.Link_Name, namespace)
	Header_User_Name_Destination_Invariants(value.User_Name, namespace)
	Header_Group_Name_Destination_Invariants(value.Group_Name, namespace)
}

// Logical_Count is one bounded materialized entry size.
type Logical_Count int

// Logical_Count_Invariants protects caller output arithmetic.
func Logical_Count_Invariants(
	value Logical_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), COUNT_MINIMUM, COUNT_MAXIMUM).
		Ensure()
}

// Logical_Position is one materialized content coordinate.
type Logical_Position int

// Logical_Position_Invariants keeps read progress bounded.
func Logical_Position_Invariants(
	value Logical_Position, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), COUNT_MINIMUM, COUNT_MAXIMUM).
		Ensure()
}

// Reader_Initialized separates unbound storage from a bound archive.
type Reader_Initialized bool

// Reader_Initialized_Invariants covers both reader lifecycle states.
func Reader_Initialized_Invariants(
	value Reader_Initialized, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Reader is bound to caller archive.").
		Ensure()
}

// Entry_Active separates initial cursor from a selected empty entry.
type Entry_Active bool

// Entry_Active_Invariants covers absent and selected entry states.
func Entry_Active_Invariants(value Entry_Active, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Reader has selected one entry.").
		Ensure()
}

// Reader_Block is the one fixed wire record borrowed by Reader.
type Reader_Block []byte

// Reader_Block_Invariants bounds one fixed wire record.
func Reader_Block_Invariants(value Reader_Block, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, BLOCK_SIZE).
		Ensure()
}

// Reader_Header_Block is validated fixed header workspace.
type Reader_Header_Block []byte

// Reader_Header_Block_Invariants protects each fixed-offset header access.
func Reader_Header_Block_Invariants(
	value Reader_Header_Block, namespace aver.Namespace,
) {
	aver.Always(
		len(value) == BLOCK_SIZE,
		"Initialized Reader header workspace has one complete TAR block.",
	)
}

// Reader_Metadata is caller extension storage.
type Reader_Metadata []byte

// Reader_Metadata_Invariants bounds extension workspace.
func Reader_Metadata_Invariants(value Reader_Metadata, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, READER_METADATA_SIZE_MAXIMUM).
		Ensure()
}

// Header_Block is one complete fixed wire header after storage validation.
type Header_Block []byte

// Header_Block_Invariants fixes every parser and formatter to one wire block.
func Header_Block_Invariants(value Header_Block, namespace aver.Namespace) {
	aver.Always(
		len(value) == BLOCK_SIZE,
		"A header occupies one complete wire block.",
	)
}

// Overlap_Left is a header field or fixed block checked for harmful aliasing.
type Overlap_Left []byte

// Overlap_Left_Invariants bounds the larger header-field side.
func Overlap_Left_Invariants(value Overlap_Left, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, HEADER_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Overlap_Right is reader metadata or a header field checked for aliasing.
type Overlap_Right []byte

// Overlap_Right_Invariants bounds the larger reader-workspace side.
func Overlap_Right_Invariants(value Overlap_Right, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, READER_METADATA_SIZE_MAXIMUM).
		Ensure()
}

// Reader_Transfer_Buffer is the current internal Stream borrow.
type Reader_Transfer_Buffer []byte

// Reader_Transfer_Buffer_Invariants bounds one transport borrow.
func Reader_Transfer_Buffer_Invariants(
	value Reader_Transfer_Buffer, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, ARCHIVE_SIZE_MAXIMUM).
		Ensure()
}

// Reader_Long_Name keeps one GNU override.
type Reader_Long_Name []byte

// Reader_Long_Name_Invariants bounds one GNU name override.
func Reader_Long_Name_Invariants(
	value Reader_Long_Name, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, HEADER_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Reader_Long_Link keeps one GNU link override.
type Reader_Long_Link []byte

// Reader_Long_Link_Invariants bounds one GNU link override.
func Reader_Long_Link_Invariants(
	value Reader_Long_Link, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, HEADER_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Reader_Storage keeps parser memory outside Reader lifetime policy.
type Reader_Storage struct {
	// Block isolates fixed wire records from metadata that survives the next read.
	Block Reader_Block
	// Metadata retains extension values until the next logical header request.
	Metadata Reader_Metadata
}

// Reader_Storage_Invariants bounds both caller workspaces.
func Reader_Storage_Invariants(value Reader_Storage, namespace aver.Namespace) {
	Reader_Block_Invariants(value.Block, namespace)
	Reader_Metadata_Invariants(value.Metadata, namespace)
}

// Reader_Stage names one pending wire action.
type Reader_Stage uint8

// Reader_Stage_Invariants bounds transport continuation state.
func Reader_Stage_Invariants(value Reader_Stage, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), uint8(READER_STAGE_NONE), uint8(READER_STAGE_CONTENT)).
		Ensure()
}

// READER_STAGE_NONE means no Stream transfer waits.
const READER_STAGE_NONE Reader_Stage = Reader_Stage(bits.WORD_8_MINIMUM)

// READER_STAGE_SKIP_CONTENT discards unread entry bytes.
const READER_STAGE_SKIP_CONTENT Reader_Stage = READER_STAGE_NONE + 1

// READER_STAGE_SKIP_PADDING discards entry alignment bytes.
const READER_STAGE_SKIP_PADDING Reader_Stage = READER_STAGE_SKIP_CONTENT + 1

// READER_STAGE_HEADER reads one fixed header block.
const READER_STAGE_HEADER Reader_Stage = READER_STAGE_SKIP_PADDING + 1

// READER_STAGE_METADATA reads one extension payload.
const READER_STAGE_METADATA Reader_Stage = READER_STAGE_HEADER + 1

// READER_STAGE_CONTENT reads requested entry content.
const READER_STAGE_CONTENT Reader_Stage = READER_STAGE_METADATA + 1

// Physical_Debt counts stored entry bytes not consumed from Stream.
type Physical_Debt int

// Physical_Debt_Invariants bounds unread stored content.
func Physical_Debt_Invariants(
	value Physical_Debt, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), COUNT_MINIMUM, COUNT_MAXIMUM).
		Ensure()
}

// Metadata_Position is first unused extension byte.
type Metadata_Position int

// Metadata_Position_Invariants bounds first unused extension byte.
func Metadata_Position_Invariants(
	value Metadata_Position, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), COUNT_MINIMUM, READER_METADATA_SIZE_MAXIMUM).
		Ensure()
}

// Pending_PAX_Position locates one retained local extension body.
type Pending_PAX_Position int

// Pending_PAX_Position_Invariants bounds one retained metadata origin.
func Pending_PAX_Position_Invariants(
	value Pending_PAX_Position, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), COUNT_MINIMUM, READER_METADATA_SIZE_MAXIMUM).
		Ensure()
}

// Pending_PAX_Size counts one retained local extension body.
type Pending_PAX_Size int

// Pending_PAX_Size_Invariants bounds absent through maximum local metadata.
func Pending_PAX_Size_Invariants(
	value Pending_PAX_Size, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), COUNT_MINIMUM, SPECIAL_FILE_SIZE_MAXIMUM).
		Ensure()
}

// Reader_Skip_Debt counts bytes owed to discard alignment.
type Reader_Skip_Debt int

// Reader_Skip_Debt_Invariants bounds one discard stage.
func Reader_Skip_Debt_Invariants(
	value Reader_Skip_Debt, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), COUNT_MINIMUM, COUNT_MAXIMUM).
		Ensure()
}

// Reader_Transfer_Offset counts retired bytes in one transfer.
type Reader_Transfer_Offset int

// Reader_Transfer_Offset_Invariants bounds partial transfer progress.
func Reader_Transfer_Offset_Invariants(
	value Reader_Transfer_Offset, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), COUNT_MINIMUM, COUNT_MAXIMUM).
		Ensure()
}

// Reader_Transfer_Needed counts bytes required by one transfer.
type Reader_Transfer_Needed int

// Reader_Transfer_Needed_Invariants bounds one transfer request.
func Reader_Transfer_Needed_Invariants(
	value Reader_Transfer_Needed, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), COUNT_MINIMUM, COUNT_MAXIMUM).
		Ensure()
}

// Reader_Extension_Type retains one metadata entry kind.
type Reader_Extension_Type byte

// Reader_Extension_Type_Invariants bounds extension identity.
func Reader_Extension_Type_Invariants(
	value Reader_Extension_Type, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), uint8(TYPE_FLAG_MINIMUM), uint8(TYPE_FLAG_MAXIMUM)).
		Ensure()
}

// Reader_Extension_Size counts one metadata payload retained across a read.
type Reader_Extension_Size int

// Reader_Extension_Size_Invariants protects metadata slicing.
func Reader_Extension_Size_Invariants(
	value Reader_Extension_Size, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), COUNT_MINIMUM, SPECIAL_FILE_SIZE_MAXIMUM).
		Ensure()
}

// Reader_Header_Complete reports whether header work retires the public request.
type Reader_Header_Complete bool

// Reader_Header_Complete_Invariants covers extension continuation and logical completion.
func Reader_Header_Complete_Invariants(
	value Reader_Header_Complete, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Reader header work completed one public request.").
		Ensure()
}

// Reader_Header_Status is the contiguous grammar result set produced while resolving a header.
type Reader_Header_Status uint8

// Reader_Header_Status_Invariants excludes lifecycle and transport results from header work.
func Reader_Header_Status_Invariants(
	value Reader_Header_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_FIELD_TOO_LONG),
		).
		Ensure()
}

// Reader_Resolve_Failure excludes archive end from logical header failures.
type Reader_Resolve_Failure uint8

// Reader_Resolve_Failure_Invariants admits logical header failures only.
func Reader_Resolve_Failure_Invariants(
	value Reader_Resolve_Failure, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), uint8(STATUS_INPUT_INVALID),
			uint8(STATUS_OUTPUT_TOO_SMALL), uint8(STATUS_FORMAT_UNSUPPORTED),
			uint8(STATUS_FIELD_TOO_LONG),
		).
		Ensure()
}

// Reader_Resolve_Success reports logical header commitment.
type Reader_Resolve_Success bool

// Reader_Resolve_Success_Invariants covers commitment and failure.
func Reader_Resolve_Success_Invariants(
	value Reader_Resolve_Success, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Logical header resolution succeeds.").
		Ensure()
}

// Reader_Active separates idle and retained callback state.
type Reader_Active bool

// Reader_Active_Invariants covers idle and active operations.
func Reader_Active_Invariants(value Reader_Active, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Reader has an operation in flight.").
		Ensure()
}

// Reader_Submission_Active marks a Stream Procedure frame.
type Reader_Submission_Active bool

// Reader_Submission_Active_Invariants covers callback timing.
func Reader_Submission_Active_Invariants(
	value Reader_Submission_Active, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Reader is inside Stream Procedure.").
		Ensure()
}

// Reader_Wait_Active marks one unretired Stream request.
type Reader_Wait_Active bool

// Reader_Wait_Active_Invariants covers inline and deferred retirement.
func Reader_Wait_Active_Invariants(
	value Reader_Wait_Active, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Reader waits for Stream retirement.").
		Ensure()
}

// Reader_Continue requests another trampoline iteration.
type Reader_Continue bool

// Reader_Continue_Invariants covers inline trampoline state.
func Reader_Continue_Invariants(
	value Reader_Continue, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Reader has inline retirement to process.").
		Ensure()
}

// Reader_Transfer owns one Stream request and its inline-retirement trampoline.
type Reader_Transfer struct {
	// Buffer stays borrowed until the complete request retires.
	Buffer Reader_Transfer_Buffer
	// Offset counts bytes already retired for the request.
	Offset Reader_Transfer_Offset
	// Needed is the exact complete request size.
	Needed Reader_Transfer_Needed
	// Stage selects the state transition after retirement.
	Stage Reader_Stage
	// Submission_Active marks the Stream Procedure frame.
	Submission_Active Reader_Submission_Active
	// Wait_Active marks one request not yet retired.
	Wait_Active Reader_Wait_Active
	// Continue turns inline callback recursion into iteration.
	Continue Reader_Continue
}

// Reader_Transfer_Invariants composes one bounded Stream request.
func Reader_Transfer_Invariants(
	value Reader_Transfer, namespace aver.Namespace,
) {
	Reader_Transfer_Buffer_Invariants(value.Buffer, namespace)
	Reader_Transfer_Offset_Invariants(value.Offset, namespace)
	Reader_Transfer_Needed_Invariants(value.Needed, namespace)
	Reader_Stage_Invariants(value.Stage, namespace)
	Reader_Submission_Active_Invariants(value.Submission_Active, namespace)
	Reader_Wait_Active_Invariants(value.Wait_Active, namespace)
	Reader_Continue_Invariants(value.Continue, namespace)
	aver.Always(
		int(value.Offset) <= int(value.Needed),
		"Reader transfer cursor does not cross requested bytes.",
	)
	aver.Always(
		int(value.Needed) <= len(value.Buffer),
		"Reader transfer request stays inside its borrowed buffer.",
	)
}

// Reader_Archive owns retained header metadata and the selected entry cursor.
type Reader_Archive struct {
	// Block is the fixed wire-record workspace.
	Block Reader_Block
	// Metadata retains PAX and GNU values across the following header.
	Metadata Reader_Metadata
	// Metadata_Position is first unused extension byte.
	Metadata_Position Metadata_Position
	// Header is the latest decoded logical entry.
	Header Header
	// Header_Storage remains borrowed until one logical header resolves.
	Header_Storage Header_Storage
	// Pending_PAX_Position locates one validated local metadata payload.
	Pending_PAX_Position Pending_PAX_Position
	// Pending_PAX_Size retains its nonzero validated byte count.
	Pending_PAX_Size Pending_PAX_Size
	// Long_Name retains one GNU path override.
	Long_Name Reader_Long_Name
	// Long_Link retains one GNU link override.
	Long_Link Reader_Long_Link
	// Extension_Type selects metadata interpretation after transfer.
	Extension_Type Reader_Extension_Type
	// Extension_Size retains metadata payload width before padding.
	Extension_Size Reader_Extension_Size
	// Physical_Debt counts stored entry bytes not consumed from Stream.
	Physical_Debt Physical_Debt
	// Padding aligns the next header block.
	Padding Padding_Count
	// Logical_Size bounds bytes exposed for current entry.
	Logical_Size Logical_Count
	// Logical_Position counts bytes already exposed.
	Logical_Position Logical_Position
	// Entry_Active distinguishes initial cursor from a selected empty entry.
	Entry_Active Entry_Active
}

// Reader_Archive_Invariants composes retained archive state.
func Reader_Archive_Invariants(
	value Reader_Archive, namespace aver.Namespace,
) {
	Reader_Block_Invariants(value.Block, namespace)
	Reader_Metadata_Invariants(value.Metadata, namespace)
	Metadata_Position_Invariants(value.Metadata_Position, namespace)
	Header_Invariants(value.Header, namespace)
	Header_Storage_Invariants(value.Header_Storage, namespace)
	Pending_PAX_Position_Invariants(value.Pending_PAX_Position, namespace)
	Pending_PAX_Size_Invariants(value.Pending_PAX_Size, namespace)
	Reader_Long_Name_Invariants(value.Long_Name, namespace)
	Reader_Long_Link_Invariants(value.Long_Link, namespace)
	Reader_Extension_Type_Invariants(value.Extension_Type, namespace)
	Reader_Extension_Size_Invariants(value.Extension_Size, namespace)
	Physical_Debt_Invariants(value.Physical_Debt, namespace)
	Padding_Count_Invariants(value.Padding, namespace)
	Logical_Count_Invariants(value.Logical_Size, namespace)
	Logical_Position_Invariants(value.Logical_Position, namespace)
	Entry_Active_Invariants(value.Entry_Active, namespace)
	aver.Always(
		int(value.Logical_Position) <= int(value.Logical_Size),
		"Reader logical cursor stays inside entry.",
	)
}

// Reader retains bounded coordinates into caller archive.
type Reader struct {
	// Completion stays first because a static nbio callback recovers this caller-owned Reader
	// from its submitted completion. A captured callback would allocate.
	Completion nbio.Completion
	// Stream owns timing; Reader owns only continuation state.
	Stream nbio.Stream
	// Archive owns retained headers, extensions, and selected-entry coordinates.
	Archive Reader_Archive
	// Callback retires after the complete TAR operation, not each stream transfer.
	Callback nbio.Callback
	// Transfer owns one Stream request and callback timing.
	Transfer Reader_Transfer
	// Skip_Debt bounds the current discard operation.
	Skip_Debt Reader_Skip_Debt
	// Status reports grammar result while Completion.Error reports transport.
	Status Status
	// Count reports the latest Reader_Read result.
	Count Count
	// Active prevents one callback slot from serving overlapping calls.
	Active Reader_Active
	// Initialized rejects use before archive binding.
	Initialized Reader_Initialized
}

// Reader_Invariants keeps every cursor within caller archive.
func Reader_Invariants(value *Reader, namespace aver.Namespace) {
	aver.Always(value != nil, "Reader state exists.")
	aver.Always(
		unsafe.Pointer(value) == unsafe.Pointer(&value.Completion),
		"Reader completion stays first so a static Stream callback recovers its owner.",
	)
	Reader_Archive_Invariants(value.Archive, namespace)
	Reader_Transfer_Invariants(value.Transfer, namespace)
	Reader_Skip_Debt_Invariants(value.Skip_Debt, namespace)
	Status_Invariants(value.Status, namespace)
	Count_Invariants(value.Count, namespace)
	Reader_Active_Invariants(value.Active, namespace)
	Reader_Initialized_Invariants(value.Initialized, namespace)
}

// Destination_Position is one encoded archive coordinate.
type Destination_Position int

// Destination_Position_Invariants keeps writes inside caller storage.
func Destination_Position_Invariants(
	value Destination_Position, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), COUNT_MINIMUM, WRITER_STORAGE_SIZE_MAXIMUM).
		Ensure()
}

// Header_Start_Position is prior content padding before one header sequence.
type Header_Start_Position int

// Header_Start_Position_Invariants bounds one header staging origin.
func Header_Start_Position_Invariants(
	value Header_Start_Position, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), COUNT_MINIMUM, PADDING_SIZE_MAXIMUM).
		Ensure()
}

// PAX_Payload_Start_Position follows metadata header after prior content padding.
type PAX_Payload_Start_Position int

// PAX_Payload_Start_Position_Invariants bounds the generated payload origin.
func PAX_Payload_Start_Position_Invariants(
	value PAX_Payload_Start_Position, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), PAX_PAYLOAD_START_POSITION_MINIMUM,
			PAX_PAYLOAD_START_POSITION_MAXIMUM,
		).
		Ensure()
}

// Encoded_Header_Size is one complete staged header sequence.
type Encoded_Header_Size int

// Encoded_Header_Size_Invariants bounds one through maximum staged sequence.
func Encoded_Header_Size_Invariants(
	value Encoded_Header_Size, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), BLOCK_SIZE, ENCODED_HEADER_SIZE_MAXIMUM).
		Ensure()
}

// PAX_Encoded_Header_Size is one complete generated PAX header sequence.
type PAX_Encoded_Header_Size int

// PAX_Encoded_Header_Size_Invariants bounds PAX metadata and main header.
func PAX_Encoded_Header_Size_Invariants(
	value PAX_Encoded_Header_Size, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), PAX_HEADER_WIRE_SIZE_MINIMUM,
			PAX_ENCODED_HEADER_SIZE_MAXIMUM,
		).
		Ensure()
}

// GNU_Encoded_Header_Size is one fixed through two-long-field GNU sequence.
type GNU_Encoded_Header_Size int

// GNU_Encoded_Header_Size_Invariants bounds one complete GNU header sequence.
func GNU_Encoded_Header_Size_Invariants(
	value GNU_Encoded_Header_Size, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), BLOCK_SIZE, ENCODED_HEADER_SIZE_MAXIMUM).
		Ensure()
}

// Header_Required_Size includes prior content padding and encoded headers.
type Header_Required_Size int

// Header_Required_Size_Invariants bounds every successful staging requirement.
func Header_Required_Size_Invariants(
	value Header_Required_Size, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), BLOCK_SIZE, WRITER_STORAGE_SIZE_MAXIMUM).
		Ensure()
}

// PAX_Header_End_Position ends one complete PAX sequence in staging storage.
type PAX_Header_End_Position int

// PAX_Header_End_Position_Invariants bounds shortest and largest PAX sequences.
func PAX_Header_End_Position_Invariants(
	value PAX_Header_End_Position, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), PAX_HEADER_WIRE_SIZE_MINIMUM,
			PAX_HEADER_END_POSITION_MAXIMUM,
		).
		Ensure()
}

// GNU_Header_End_Position ends one complete GNU sequence in staging storage.
type GNU_Header_End_Position int

// GNU_Header_End_Position_Invariants bounds fixed through two-long-field sequences.
func GNU_Header_End_Position_Invariants(
	value GNU_Header_End_Position, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), BLOCK_SIZE, WRITER_STORAGE_SIZE_MAXIMUM).
		Ensure()
}

// GNU_Long_Position starts one of two possible GNU long-field records.
type GNU_Long_Position int

// GNU_Long_Position_Invariants bounds first through second metadata origins.
func GNU_Long_Position_Invariants(
	value GNU_Long_Position, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), COUNT_MINIMUM, GNU_LONG_POSITION_MAXIMUM).
		Ensure()
}

// GNU_Long_End_Position ends one staged GNU long-field record.
type GNU_Long_End_Position int

// GNU_Long_End_Position_Invariants bounds first minimum through second maximum.
func GNU_Long_End_Position_Invariants(
	value GNU_Long_End_Position, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), GNU_LONG_END_POSITION_MINIMUM,
			GNU_LONG_END_POSITION_MAXIMUM,
		).
		Ensure()
}

// GNU_Long_Encoded_Size is one metadata header plus one padded long value.
type GNU_Long_Encoded_Size int

// GNU_Long_Encoded_Size_Invariants bounds one complete long-field record.
func GNU_Long_Encoded_Size_Invariants(
	value GNU_Long_Encoded_Size, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), GNU_LONG_END_POSITION_MINIMUM,
			GNU_LONG_FIELD_WIRE_SIZE_MAXIMUM,
		).
		Ensure()
}

// Content_Count is payload still owed by current writer entry.
type Content_Count int

// Content_Count_Invariants protects declared entry completion.
func Content_Count_Invariants(value Content_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), COUNT_MINIMUM, COUNT_MAXIMUM).
		Ensure()
}

// Padding_Count is zero fill owed before next header or footer.
type Padding_Count int

// Padding_Count_Invariants bounds one TAR record remainder.
func Padding_Count_Invariants(value Padding_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), COUNT_MINIMUM, PADDING_SIZE_MAXIMUM).
		Ensure()
}

// Writer_Initialized separates unbound storage from caller output.
type Writer_Initialized bool

// Writer_Initialized_Invariants covers both writer binding states.
func Writer_Initialized_Invariants(
	value Writer_Initialized, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Writer is bound to caller destination.").
		Ensure()
}

// Writer_Closed stops mutation after footer emission.
type Writer_Closed bool

// Writer_Closed_Invariants covers open and closed lifecycle states.
func Writer_Closed_Invariants(value Writer_Closed, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Writer emitted archive footer.").
		Ensure()
}

// Writer_Header_Storage owns retained header staging bytes.
type Writer_Header_Storage []byte

// Writer_Header_Storage_Invariants bounds retained staging bytes.
func Writer_Header_Storage_Invariants(
	value Writer_Header_Storage, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, WRITER_STORAGE_SIZE_MAXIMUM).
		Ensure()
}

// Zero_Destination is one staged block, padding span, or footer span.
type Zero_Destination []byte

// Zero_Destination_Invariants bounds every zero-filled staging span.
func Zero_Destination_Invariants(
	value Zero_Destination, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, WRITER_STORAGE_SIZE_MINIMUM).
		Ensure()
}

// PAX_Header_Destination is storage proven to hold the shortest PAX sequence.
type PAX_Header_Destination []byte

// PAX_Header_Destination_Invariants excludes storage rejected by preflight.
func PAX_Header_Destination_Invariants(
	value PAX_Header_Destination, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), PAX_HEADER_WIRE_SIZE_MINIMUM, WRITER_STORAGE_SIZE_MAXIMUM).
		Ensure()
}

// GNU_Header_Destination is storage proven to hold one fixed GNU header.
type GNU_Header_Destination []byte

// GNU_Header_Destination_Invariants excludes storage rejected by preflight.
func GNU_Header_Destination_Invariants(
	value GNU_Header_Destination, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), WRITER_STORAGE_SIZE_MINIMUM, WRITER_STORAGE_SIZE_MAXIMUM).
		Ensure()
}

// GNU_Long_Destination is storage proven to hold a long field and main header.
type GNU_Long_Destination []byte

// GNU_Long_Destination_Invariants excludes storage rejected by long preflight.
func GNU_Long_Destination_Invariants(
	value GNU_Long_Destination, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), GNU_LONG_DESTINATION_SIZE_MINIMUM,
			WRITER_STORAGE_SIZE_MAXIMUM,
		).
		Ensure()
}

// Writer_Operation names the state transition waiting on Stream.
type Writer_Operation uint8

// Writer_Operation_Invariants bounds callback commit identity.
func Writer_Operation_Invariants(value Writer_Operation, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), uint8(WRITER_OPERATION_NONE),
			uint8(WRITER_OPERATION_HEADER), uint8(WRITER_OPERATION_CONTENT),
			uint8(WRITER_OPERATION_CLOSE),
		).
		Ensure()
}

// WRITER_OPERATION_NONE means no public operation owns retained state.
const WRITER_OPERATION_NONE Writer_Operation = Writer_Operation(bits.WORD_8_MINIMUM)

// WRITER_OPERATION_HEADER submits staged header bytes.
const WRITER_OPERATION_HEADER Writer_Operation = WRITER_OPERATION_NONE + 1

// WRITER_OPERATION_CONTENT submits caller content bytes.
const WRITER_OPERATION_CONTENT Writer_Operation = WRITER_OPERATION_HEADER + 1

// WRITER_OPERATION_CLOSE submits padding and footer bytes.
const WRITER_OPERATION_CLOSE Writer_Operation = WRITER_OPERATION_CONTENT + 1

// Archive_Count reports bytes accepted by Stream for one archive.
type Archive_Count int

// Archive_Count_Invariants bounds accepted archive bytes.
func Archive_Count_Invariants(value Archive_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), COUNT_MINIMUM, COUNT_MAXIMUM).
		Ensure()
}

// Pending_Content_Count retains content debt until header retirement.
type Pending_Content_Count int

// Pending_Content_Count_Invariants bounds staged content debt.
func Pending_Content_Count_Invariants(
	value Pending_Content_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), COUNT_MINIMUM, COUNT_MAXIMUM).
		Ensure()
}

// Pending_Padding_Count retains alignment debt until header retirement.
type Pending_Padding_Count int

// Pending_Padding_Count_Invariants bounds staged alignment debt.
func Pending_Padding_Count_Invariants(
	value Pending_Padding_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), COUNT_MINIMUM, PADDING_SIZE_MAXIMUM).
		Ensure()
}

// Writer_Active separates idle and retained callback state.
type Writer_Active bool

// Writer_Active_Invariants covers idle and active operations.
func Writer_Active_Invariants(value Writer_Active, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Writer has an operation in flight.").
		Ensure()
}

// Writer_Submission_Active marks a Stream Procedure frame.
type Writer_Submission_Active bool

// Writer_Submission_Active_Invariants covers callback timing.
func Writer_Submission_Active_Invariants(
	value Writer_Submission_Active, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Writer is inside Stream Procedure.").
		Ensure()
}

// Writer_Wait_Active marks one unretired Stream request.
type Writer_Wait_Active bool

// Writer_Wait_Active_Invariants covers inline and deferred retirement.
func Writer_Wait_Active_Invariants(
	value Writer_Wait_Active, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Writer waits for Stream retirement.").
		Ensure()
}

// Writer_Continue requests another trampoline iteration.
type Writer_Continue bool

// Writer_Continue_Invariants covers inline trampoline state.
func Writer_Continue_Invariants(
	value Writer_Continue, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Writer has inline retirement to process.").
		Ensure()
}

// Writer retains bounded coordinates into caller destination.
type Writer struct {
	// Completion stays first because a static nbio callback recovers this caller-owned Writer
	// from its submitted completion. A captured callback would allocate.
	Completion nbio.Completion
	// Stream owns transport and callback timing.
	Stream nbio.Stream
	// Callback retires after one complete public operation.
	Callback nbio.Callback
	// Source remains borrowed until its write retires.
	Source Source
	// Pending_Content commits only after the encoded header reaches Stream.
	Pending_Content Pending_Content_Count
	// Pending_Padding commits only after the encoded header reaches Stream.
	Pending_Padding Pending_Padding_Count
	// Archive_Count bounds total successful wire bytes.
	Archive_Count Archive_Count
	// Operation selects callback commit rules.
	Operation Writer_Operation
	// Status reports TAR validation separately from transport failure.
	Status Status
	// Count reports latest transfer or final archive size.
	Count Count
	// Active protects retained callback and borrowed buffer state.
	Active Writer_Active
	// Submitting handles a stream that retires before Procedure returns.
	Submission_Active Writer_Submission_Active
	// Waiting distinguishes deferred retirement.
	Wait_Active Writer_Wait_Active
	// Continue asks the submitting frame to finish inline retirement.
	Continue Writer_Continue
	// Destination retains caller header staging storage.
	Destination Writer_Header_Storage
	// Position is first unwritten output byte.
	Position Destination_Position
	// Content_Count counts content still required by current header.
	Content_Count Content_Count
	// Padding counts zero bytes due before next header.
	Padding Padding_Count
	// Initialized rejects use before output binding.
	Initialized Writer_Initialized
	// Closed rejects mutation after footer emission.
	Closed Writer_Closed
}

// Writer_Invariants keeps output cursor inside caller destination.
func Writer_Invariants(value *Writer, namespace aver.Namespace) {
	aver.Always(value != nil, "Writer state exists.")
	aver.Always(
		unsafe.Pointer(value) == unsafe.Pointer(&value.Completion),
		"Writer completion stays first so a static Stream callback recovers its owner.",
	)
	Source_Invariants(value.Source, namespace)
	Pending_Content_Count_Invariants(value.Pending_Content, namespace)
	Pending_Padding_Count_Invariants(value.Pending_Padding, namespace)
	Archive_Count_Invariants(value.Archive_Count, namespace)
	Writer_Operation_Invariants(value.Operation, namespace)
	Status_Invariants(value.Status, namespace)
	Count_Invariants(value.Count, namespace)
	Writer_Active_Invariants(value.Active, namespace)
	Writer_Submission_Active_Invariants(value.Submission_Active, namespace)
	Writer_Wait_Active_Invariants(value.Wait_Active, namespace)
	Writer_Continue_Invariants(value.Continue, namespace)
	Writer_Header_Storage_Invariants(value.Destination, namespace)
	Destination_Position_Invariants(value.Position, namespace)
	Content_Count_Invariants(value.Content_Count, namespace)
	Padding_Count_Invariants(value.Padding, namespace)
	Writer_Initialized_Invariants(value.Initialized, namespace)
	Writer_Closed_Invariants(value.Closed, namespace)
	aver.Always(
		int(value.Position) <= len(value.Destination),
		"Writer cursor stays inside destination.",
	)
}

// Regular file type.
const TYPE_REGULAR Type_Flag = '0'

// Legacy zero regular-file type.
const TYPE_REGULAR_LEGACY Type_Flag = TYPE_FLAG_MINIMUM

// Hard-link type.
const TYPE_LINK Type_Flag = '1'

// Symbolic-link type.
const TYPE_SYMBOLIC_LINK Type_Flag = '2'

// Character-device type.
const TYPE_CHARACTER Type_Flag = '3'

// Block-device type.
const TYPE_BLOCK Type_Flag = '4'

// Directory type.
const TYPE_DIRECTORY Type_Flag = '5'

// FIFO type.
const TYPE_FIFO Type_Flag = '6'

// Contiguous-file reserved type.
const TYPE_CONTIGUOUS Type_Flag = '7'

// Local PAX extended-header type.
const TYPE_PAX_LOCAL Type_Flag = 'x'

// Global PAX extended-header type.
const TYPE_PAX_GLOBAL Type_Flag = 'g'

// GNU sparse-file type.
const TYPE_GNU_SPARSE Type_Flag = 'S'

// GNU long-name metadata type.
const TYPE_GNU_LONG_NAME Type_Flag = 'L'

// GNU long-link metadata type.
const TYPE_GNU_LONG_LINK Type_Flag = 'K'

// Field_Value retains one contiguous or split archive field view.
type Name_Prefix []byte

// Name_Prefix_Invariants bounds one split path prefix.
func Name_Prefix_Invariants(value Name_Prefix, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, PREFIX_FIELD_SIZE).
		Ensure()
}

// Name_Suffix retains one final path component or contiguous path.
type Name_Suffix []byte

// Name_Suffix_Invariants bounds one split path suffix.
func Name_Suffix_Invariants(value Name_Suffix, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, HEADER_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Name_Joined records whether split path needs one separator.
type Name_Joined bool

// Name_Joined_Invariants covers contiguous and split path views.
func Name_Joined_Invariants(value Name_Joined, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Archive path uses prefix field.").
		Ensure()
}

// Field_Value retains split archive path without copying.
type Field_Value struct {
	// Prefix holds optional USTAR or STAR path prefix.
	Prefix Name_Prefix
	// Suffix holds final field or path component.
	Suffix Name_Suffix
	// Joined inserts path separator between prefix and suffix.
	Joined Name_Joined
}

// Field_Value_Invariants bounds combined metadata view.
func Field_Value_Invariants(value Field_Value, namespace aver.Namespace) {
	Name_Prefix_Invariants(value.Prefix, namespace)
	Name_Suffix_Invariants(value.Suffix, namespace)
	Name_Joined_Invariants(value.Joined, namespace)
	aver.Always(
		len(value.Prefix)+len(value.Suffix) <= SPECIAL_FILE_SIZE_MAXIMUM,
		"Split archive field stays inside metadata bound.",
	)
}

// Field is one bounded logical or physical metadata view.
type Field []byte

// Field_Invariants bounds one complete metadata view.
func Field_Invariants(value Field, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, SPECIAL_FILE_SIZE_MAXIMUM).
		Ensure()
}

// NUL_Checked_Text is one header field or string record checked for NUL.
type NUL_Checked_Text []byte

// NUL_Checked_Text_Invariants bounds largest checked header field.
func NUL_Checked_Text_Invariants(
	value NUL_Checked_Text, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, HEADER_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Field_Size is one decoded path width.
type Field_Size int

// Field_Size_Invariants bounds decoded path width.
func Field_Size_Invariants(value Field_Size, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), COUNT_MINIMUM, HEADER_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Wire_Text_Field is one fixed textual header field.
type Wire_Text_Field []byte

// Wire_Text_Field_Invariants admits fixed text widths only.
func Wire_Text_Field_Invariants(
	value Wire_Text_Field, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_4_Int(
			len(value), USER_NAME_FIELD_SIZE, NAME_FIELD_SIZE,
			STAR_PREFIX_FIELD_SIZE, PREFIX_FIELD_SIZE,
		).
		Ensure()
}

// Wire_Text is text before first fixed-field NUL.
type Wire_Text []byte

// Wire_Text_Invariants bounds largest fixed text field.
func Wire_Text_Invariants(value Wire_Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, PREFIX_FIELD_SIZE).
		Ensure()
}

// GNU_Prefix_Text is fallback path prefix after failed optional time parse.
type GNU_Prefix_Text []byte

// GNU_Prefix_Text_Invariants bounds one GNU prefix field.
func GNU_Prefix_Text_Invariants(
	value GNU_Prefix_Text, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, PREFIX_FIELD_SIZE).
		Ensure()
}

// Wire_File_Mode is one value decoded from the eight-byte mode field.
type Wire_File_Mode int64

// Wire_File_Mode_Invariants bounds an eight-byte base-256 value.
func Wire_File_Mode_Invariants(value Wire_File_Mode, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), SMALL_NUMERIC_MINIMUM, SMALL_NUMERIC_MAXIMUM).
		Ensure()
}

// Wire_User_Identifier is one value decoded from the eight-byte owner field.
type Wire_User_Identifier int64

// Wire_User_Identifier_Invariants bounds an eight-byte base-256 value.
func Wire_User_Identifier_Invariants(
	value Wire_User_Identifier, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), SMALL_NUMERIC_MINIMUM, SMALL_NUMERIC_MAXIMUM).
		Ensure()
}

// Wire_Group_Identifier is one value decoded from the eight-byte group field.
type Wire_Group_Identifier int64

// Wire_Group_Identifier_Invariants bounds an eight-byte base-256 value.
func Wire_Group_Identifier_Invariants(
	value Wire_Group_Identifier, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), SMALL_NUMERIC_MINIMUM, SMALL_NUMERIC_MAXIMUM).
		Ensure()
}

// Wire_Device_Major is one value decoded from the eight-byte device field.
type Wire_Device_Major int64

// Wire_Device_Major_Invariants bounds an eight-byte base-256 value.
func Wire_Device_Major_Invariants(
	value Wire_Device_Major, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), SMALL_NUMERIC_MINIMUM, SMALL_NUMERIC_MAXIMUM).
		Ensure()
}

// Wire_Device_Minor is one value decoded from the eight-byte device field.
type Wire_Device_Minor int64

// Wire_Device_Minor_Invariants bounds an eight-byte base-256 value.
func Wire_Device_Minor_Invariants(
	value Wire_Device_Minor, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), SMALL_NUMERIC_MINIMUM, SMALL_NUMERIC_MAXIMUM).
		Ensure()
}

// Wire_Name_Prefix aliases the fixed USTAR or STAR prefix field.
type Wire_Name_Prefix []byte

// Wire_Name_Prefix_Invariants bounds the largest prefix field.
func Wire_Name_Prefix_Invariants(
	value Wire_Name_Prefix, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, PREFIX_FIELD_SIZE).
		Ensure()
}

// Wire_Prefix_Field is one STAR or USTAR prefix field view.
type Wire_Prefix_Field []byte

// Wire_Prefix_Field_Invariants spans the two fixed prefix widths.
func Wire_Prefix_Field_Invariants(
	value Wire_Prefix_Field, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), STAR_PREFIX_FIELD_SIZE, PREFIX_FIELD_SIZE).
		Ensure()
}

// Wire_Name_Suffix aliases the fixed V7 name field.
type Wire_Name_Suffix []byte

// Wire_Name_Suffix_Invariants bounds the fixed name field.
func Wire_Name_Suffix_Invariants(
	value Wire_Name_Suffix, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, NAME_FIELD_SIZE).
		Ensure()
}

// STAR_Wire_Name_Prefix is one STAR prefix field.
type STAR_Wire_Name_Prefix []byte

// STAR_Wire_Name_Prefix_Invariants bounds STAR prefix bytes.
func STAR_Wire_Name_Prefix_Invariants(
	value STAR_Wire_Name_Prefix, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, STAR_PREFIX_FIELD_SIZE).
		Ensure()
}

// STAR_Wire_Name_Suffix is one STAR name field.
type STAR_Wire_Name_Suffix []byte

// STAR_Wire_Name_Suffix_Invariants bounds STAR suffix bytes.
func STAR_Wire_Name_Suffix_Invariants(
	value STAR_Wire_Name_Suffix, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, NAME_FIELD_SIZE).
		Ensure()
}

// STAR_Wire_Name retains exact STAR prefix and suffix fields.
type STAR_Wire_Name struct {
	// Prefix keeps the split boundary that concatenated bytes cannot recover.
	Prefix STAR_Wire_Name_Prefix
	// Suffix keeps an empty prefix distinct from a separator-bearing name.
	Suffix STAR_Wire_Name_Suffix
	// Joined prevents an absent separator from being inferred from byte content.
	Joined Wire_Name_Joined
}

// STAR_Wire_Name_Invariants composes exact STAR path fields.
func STAR_Wire_Name_Invariants(
	value STAR_Wire_Name, namespace aver.Namespace,
) {
	STAR_Wire_Name_Prefix_Invariants(value.Prefix, namespace)
	STAR_Wire_Name_Suffix_Invariants(value.Suffix, namespace)
	Wire_Name_Joined_Invariants(value.Joined, namespace)
}

// Wire_Name_Joined records whether one separator joins fixed name fields.
type Wire_Name_Joined bool

// Wire_Name_Joined_Invariants covers contiguous and split wire names.
func Wire_Name_Joined_Invariants(
	value Wire_Name_Joined, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Wire path uses prefix field.").
		Ensure()
}

// Wire_Name retains one fixed or split wire path.
type Wire_Name struct {
	// Prefix aliases the USTAR or STAR prefix field.
	Prefix Wire_Name_Prefix
	// Suffix aliases the V7 name field.
	Suffix Wire_Name_Suffix
	// Joined inserts one path separator.
	Joined Wire_Name_Joined
}

// Wire_Name_Invariants composes one bounded wire path.
func Wire_Name_Invariants(value Wire_Name, namespace aver.Namespace) {
	Wire_Name_Prefix_Invariants(value.Prefix, namespace)
	Wire_Name_Suffix_Invariants(value.Suffix, namespace)
	Wire_Name_Joined_Invariants(value.Joined, namespace)
	aver.Always(
		len(value.Prefix)+len(value.Suffix) <= WIRE_NAME_SIZE_MAXIMUM,
		"Wire path stays inside fixed fields.",
	)
}

// Wire_Link_Name aliases the fixed V7 link field.
type Wire_Link_Name []byte

// Wire_Link_Name_Invariants bounds the fixed link field.
func Wire_Link_Name_Invariants(value Wire_Link_Name, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, LINK_NAME_FIELD_SIZE).
		Ensure()
}

// Wire_User_Name aliases the fixed USTAR owner field.
type Wire_User_Name []byte

// Wire_User_Name_Invariants bounds the fixed owner field.
func Wire_User_Name_Invariants(value Wire_User_Name, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, USER_NAME_FIELD_SIZE).
		Ensure()
}

// Wire_Group_Name aliases the fixed USTAR group field.
type Wire_Group_Name []byte

// Wire_Group_Name_Invariants bounds the fixed group field.
func Wire_Group_Name_Invariants(value Wire_Group_Name, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, GROUP_NAME_FIELD_SIZE).
		Ensure()
}

// USTAR_Name is one path proven to fit the prefix and suffix fields.
type USTAR_Name []byte

// USTAR_Name_Invariants bounds one encodable USTAR path.
func USTAR_Name_Invariants(value USTAR_Name, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), USTAR_PATH_SIZE_MINIMUM, USTAR_PATH_SIZE_MAXIMUM).
		Ensure()
}

// USTAR_User_Name is one fixed-field owner name.
type USTAR_User_Name []byte

// USTAR_User_Name_Invariants bounds one USTAR owner field.
func USTAR_User_Name_Invariants(
	value USTAR_User_Name, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, USER_NAME_FIELD_SIZE).
		Ensure()
}

// USTAR_Group_Name is one fixed-field group name.
type USTAR_Group_Name []byte

// USTAR_Group_Name_Invariants bounds one USTAR group field.
func USTAR_Group_Name_Invariants(
	value USTAR_Group_Name, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, GROUP_NAME_FIELD_SIZE).
		Ensure()
}

// USTAR_Device_Major is one device class proven to fit its octal field.
type USTAR_Device_Major int64

// USTAR_Device_Major_Invariants bounds one device-class field.
func USTAR_Device_Major_Invariants(
	value USTAR_Device_Major, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), COUNT_MINIMUM, SMALL_OCTAL_MAXIMUM).
		Ensure()
}

// USTAR_Device_Minor is one device instance proven to fit its octal field.
type USTAR_Device_Minor int64

// USTAR_Device_Minor_Invariants bounds one device-instance field.
func USTAR_Device_Minor_Invariants(
	value USTAR_Device_Minor, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), COUNT_MINIMUM, SMALL_OCTAL_MAXIMUM).
		Ensure()
}

// USTAR_Fields contains the values unique to one USTAR-compatible header.
type USTAR_Fields struct {
	// Name supplies the optional prefix.
	Name USTAR_Name
	// User_Name fills the fixed owner field.
	User_Name USTAR_User_Name
	// Group_Name fills the fixed group field.
	Group_Name USTAR_Group_Name
	// Device_Major fills the fixed device-class field.
	Device_Major USTAR_Device_Major
	// Device_Minor fills the fixed device-instance field.
	Device_Minor USTAR_Device_Minor
}

// USTAR_Fields_Invariants composes one encodable USTAR extension.
func USTAR_Fields_Invariants(value USTAR_Fields, namespace aver.Namespace) {
	USTAR_Name_Invariants(value.Name, namespace)
	USTAR_User_Name_Invariants(value.User_Name, namespace)
	USTAR_Group_Name_Invariants(value.Group_Name, namespace)
	USTAR_Device_Major_Invariants(value.Device_Major, namespace)
	USTAR_Device_Minor_Invariants(value.Device_Minor, namespace)
}

// USTAR_Prefix_Count locates the final separator in one encodable path.
type USTAR_Prefix_Count int

// USTAR_Prefix_Count_Invariants bounds one prefix field write.
func USTAR_Prefix_Count_Invariants(
	value USTAR_Prefix_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), COUNT_MINIMUM, PREFIX_FIELD_SIZE).
		Ensure()
}

// USTAR_Suffix_Start follows selected path separator.
type USTAR_Suffix_Start int

// USTAR_Suffix_Start_Invariants bounds fixed name-field origin.
func USTAR_Suffix_Start_Invariants(
	value USTAR_Suffix_Start, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), COUNT_MINIMUM, USTAR_SUFFIX_START_MAXIMUM,
		).
		Ensure()
}

// Basic_Format excludes selection and read-only families from fixed writer headers.
type Basic_Format uint8

// Basic_Format_Invariants bounds one V7-compatible writer format.
func Basic_Format_Invariants(value Basic_Format, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(FORMAT_V7), uint8(FORMAT_USTAR), uint8(FORMAT_PAX),
		).
		Ensure()
}

// Basic_Fit_Format contains only fixed formats selected by direct-fit checks.
type Basic_Fit_Format uint8

// Basic_Fit_Format_Invariants distinguishes V7 from USTAR path rules.
func Basic_Fit_Format_Invariants(
	value Basic_Fit_Format, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(FORMAT_V7), uint8(FORMAT_USTAR)).
		Ensure()
}

// Writer_Selected_Format is one writable family after automatic selection.
type Writer_Selected_Format uint8

// Writer_Selected_Format_Invariants excludes selection and read-only STAR.
func Writer_Selected_Format_Invariants(
	value Writer_Selected_Format, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), uint8(FORMAT_V7), uint8(FORMAT_USTAR),
			uint8(FORMAT_PAX), uint8(FORMAT_GNU),
		).
		Ensure()
}

// Encoding_Format excludes automatic selection before size dispatch.
type Encoding_Format uint8

// Encoding_Format_Invariants admits wire families accepted by dispatch.
func Encoding_Format_Invariants(
	value Encoding_Format, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), uint8(FORMAT_V7), uint8(FORMAT_STAR)).
		Ensure()
}

// Basic_Link_Name is proven to fit the fixed V7 field.
type Basic_Link_Name []byte

// Basic_Link_Name_Invariants bounds one fixed link field.
func Basic_Link_Name_Invariants(
	value Basic_Link_Name, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, LINK_NAME_FIELD_SIZE).
		Ensure()
}

// Basic_File_Mode is proven to fit the fixed octal mode field.
type Basic_File_Mode int64

// Basic_File_Mode_Invariants bounds one fixed mode field.
func Basic_File_Mode_Invariants(
	value Basic_File_Mode, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), COUNT_MINIMUM, SMALL_OCTAL_MAXIMUM).
		Ensure()
}

// Basic_User_Identifier is proven to fit the fixed octal owner field.
type Basic_User_Identifier int64

// Basic_User_Identifier_Invariants bounds one fixed owner field.
func Basic_User_Identifier_Invariants(
	value Basic_User_Identifier, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), COUNT_MINIMUM, SMALL_OCTAL_MAXIMUM).
		Ensure()
}

// Basic_Group_Identifier is proven to fit the fixed octal group field.
type Basic_Group_Identifier int64

// Basic_Group_Identifier_Invariants bounds one fixed group field.
func Basic_Group_Identifier_Invariants(
	value Basic_Group_Identifier, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), COUNT_MINIMUM, SMALL_OCTAL_MAXIMUM).
		Ensure()
}

// Basic_Modification_Seconds is proven to fit the fixed octal timestamp field.
type Basic_Modification_Seconds int64

// Basic_Modification_Seconds_Invariants bounds one fixed timestamp field.
func Basic_Modification_Seconds_Invariants(
	value Basic_Modification_Seconds, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), COUNT_MINIMUM, LARGE_OCTAL_MAXIMUM).
		Ensure()
}

// Basic_Header contains exactly the values consumed by a V7-compatible wire record.
type Basic_Header struct {
	// Format cannot admit selection or read-only families after staging.
	Format Basic_Format
	// Type_Flag remains one complete extension byte on the wire.
	Type_Flag Type_Flag
	// Name is already proven to fit fixed prefix and suffix fields.
	Name USTAR_Name
	// Link_Name is already proven to fit its fixed field.
	Link_Name Basic_Link_Name
	// Mode cannot lose high bits during octal formatting.
	Mode Basic_File_Mode
	// User_Identifier cannot lose high bits during octal formatting.
	User_Identifier Basic_User_Identifier
	// Group_Identifier cannot lose high bits during octal formatting.
	Group_Identifier Basic_Group_Identifier
	// Size remains inside both package and fixed-field bounds.
	Size Entry_Size
	// Modification_Seconds excludes fractions the fixed field cannot preserve.
	Modification_Seconds Basic_Modification_Seconds
	// User_Name is already proven to fit its fixed field.
	User_Name USTAR_User_Name
	// Group_Name is already proven to fit its fixed field.
	Group_Name USTAR_Group_Name
	// Device_Major cannot lose high bits during octal formatting.
	Device_Major USTAR_Device_Major
	// Device_Minor cannot lose high bits during octal formatting.
	Device_Minor USTAR_Device_Minor
}

// Basic_Header_Invariants composes one fixed writer record.
func Basic_Header_Invariants(value Basic_Header, namespace aver.Namespace) {
	Basic_Format_Invariants(value.Format, namespace)
	Type_Flag_Invariants(value.Type_Flag, namespace)
	USTAR_Name_Invariants(value.Name, namespace)
	Basic_Link_Name_Invariants(value.Link_Name, namespace)
	Basic_File_Mode_Invariants(value.Mode, namespace)
	Basic_User_Identifier_Invariants(value.User_Identifier, namespace)
	Basic_Group_Identifier_Invariants(value.Group_Identifier, namespace)
	Entry_Size_Invariants(value.Size, namespace)
	Basic_Modification_Seconds_Invariants(value.Modification_Seconds, namespace)
	USTAR_User_Name_Invariants(value.User_Name, namespace)
	USTAR_Group_Name_Invariants(value.Group_Name, namespace)
	USTAR_Device_Major_Invariants(value.Device_Major, namespace)
	USTAR_Device_Minor_Invariants(value.Device_Minor, namespace)
}

// GNU_Name is one non-empty path accepted by GNU long-name encoding.
type GNU_Name []byte

// GNU_Name_Invariants reserves one wire terminator inside the metadata bound.
func GNU_Name_Invariants(value GNU_Name, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), TYPE_FLAG_FIELD_SIZE, HEADER_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// GNU_Link_Name is one optional fixed or GNU long-link value.
type GNU_Link_Name []byte

// GNU_Link_Name_Invariants reserves one wire terminator inside the metadata bound.
func GNU_Link_Name_Invariants(value GNU_Link_Name, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, HEADER_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// GNU_Modification_Time is one whole-second GNU modification instant.
type GNU_Modification_Time struct {
	// Seconds remains complete because the GNU field exceeds one signed word.
	Seconds Integer
	// Set distinguishes an omitted timestamp from the Unix epoch.
	Set bytes.Boolean
}

// GNU_Modification_Time_Invariants composes one lossless fixed timestamp.
func GNU_Modification_Time_Invariants(
	value GNU_Modification_Time, namespace aver.Namespace,
) {
	Integer_Invariants(value.Seconds, namespace)
	bytes.Boolean_Invariants(value.Set, namespace)
}

// GNU_Header contains exactly the values encoded by one GNU main header sequence.
type GNU_Header struct {
	// Type_Flag preserves the complete GNU entry-kind byte.
	Type_Flag Type_Flag
	// Name selects fixed-name or long-name encoding.
	Name GNU_Name
	// Link_Name selects fixed-link or long-link encoding.
	Link_Name GNU_Link_Name
	// Mode is proven to fit the GNU base-256 field.
	Mode Wire_File_Mode
	// User_Identifier is proven to fit the GNU base-256 field.
	User_Identifier Wire_User_Identifier
	// Group_Identifier is proven to fit the GNU base-256 field.
	Group_Identifier Wire_Group_Identifier
	// Size remains bounded below the wider GNU field limit.
	Size Entry_Size
	// Modification_Time excludes fractions GNU cannot encode.
	Modification_Time GNU_Modification_Time
	// User_Name is proven to fit its fixed field.
	User_Name Wire_User_Name
	// Group_Name is proven to fit its fixed field.
	Group_Name Wire_Group_Name
	// Device_Major is proven to fit the GNU base-256 field.
	Device_Major Wire_Device_Major
	// Device_Minor is proven to fit the GNU base-256 field.
	Device_Minor Wire_Device_Minor
	// Access_Time excludes fractions GNU cannot encode.
	Access_Time Wire_Access_Time
	// Change_Time excludes fractions GNU cannot encode.
	Change_Time Wire_Change_Time
}

// GNU_Header_Invariants composes one lossless GNU writer sequence.
func GNU_Header_Invariants(value GNU_Header, namespace aver.Namespace) {
	Type_Flag_Invariants(value.Type_Flag, namespace)
	GNU_Name_Invariants(value.Name, namespace)
	GNU_Link_Name_Invariants(value.Link_Name, namespace)
	Wire_File_Mode_Invariants(value.Mode, namespace)
	Wire_User_Identifier_Invariants(value.User_Identifier, namespace)
	Wire_Group_Identifier_Invariants(value.Group_Identifier, namespace)
	Entry_Size_Invariants(value.Size, namespace)
	GNU_Modification_Time_Invariants(value.Modification_Time, namespace)
	Wire_User_Name_Invariants(value.User_Name, namespace)
	Wire_Group_Name_Invariants(value.Group_Name, namespace)
	Wire_Device_Major_Invariants(value.Device_Major, namespace)
	Wire_Device_Minor_Invariants(value.Device_Minor, namespace)
	Wire_Access_Time_Invariants(value.Access_Time, namespace)
	Wire_Change_Time_Invariants(value.Change_Time, namespace)
}

// PAX_Payload_Size is one complete generated metadata payload.
type PAX_Payload_Size int

// PAX_Payload_Size_Invariants bounds mandatory records and the metadata cap.
func PAX_Payload_Size_Invariants(
	value PAX_Payload_Size, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), PAX_PAYLOAD_SIZE_MINIMUM, SPECIAL_FILE_SIZE_MAXIMUM).
		Ensure()
}

// PAX_Records_Payload_Size includes at least one caller-supplied record.
type PAX_Records_Payload_Size int

// PAX_Records_Payload_Size_Invariants bounds record-bearing metadata.
func PAX_Records_Payload_Size_Invariants(
	value PAX_Records_Payload_Size, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), PAX_RECORDS_PAYLOAD_SIZE_MINIMUM,
			SPECIAL_FILE_SIZE_MAXIMUM,
		).
		Ensure()
}

// PAX_Header_Name is one mandatory path proven to fit the complete payload.
type PAX_Header_Name []byte

// PAX_Header_Name_Invariants bounds the path after record framing.
func PAX_Header_Name_Invariants(
	value PAX_Header_Name, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), TYPE_FLAG_FIELD_SIZE, PAX_NAME_SIZE_MAXIMUM).
		Ensure()
}

// PAX_Fallback_Name is nonempty fixed-header basename suffix.
type PAX_Fallback_Name []byte

// PAX_Fallback_Name_Invariants bounds PAX fixed-header fallback.
func PAX_Fallback_Name_Invariants(
	value PAX_Fallback_Name, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), TYPE_FLAG_FIELD_SIZE, NAME_FIELD_SIZE).
		Ensure()
}

// PAX_Header_Link_Name is one optional link proven to fit the complete payload.
type PAX_Header_Link_Name []byte

// PAX_Header_Link_Name_Invariants bounds the link after record framing.
func PAX_Header_Link_Name_Invariants(
	value PAX_Header_Link_Name, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, PAX_LINK_NAME_SIZE_MAXIMUM).
		Ensure()
}

// PAX_Header_User_Name is one optional owner proven to fit the complete payload.
type PAX_Header_User_Name []byte

// PAX_Header_User_Name_Invariants bounds the owner after record framing.
func PAX_Header_User_Name_Invariants(
	value PAX_Header_User_Name, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, PAX_USER_NAME_SIZE_MAXIMUM).
		Ensure()
}

// PAX_Header_Group_Name is one optional group proven to fit the complete payload.
type PAX_Header_Group_Name []byte

// PAX_Header_Group_Name_Invariants bounds the group after record framing.
func PAX_Header_Group_Name_Invariants(
	value PAX_Header_Group_Name, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, PAX_GROUP_NAME_SIZE_MAXIMUM).
		Ensure()
}

// PAX_Header contains exactly the values consumed by one PAX header sequence.
type PAX_Header struct {
	// Type_Flag preserves the complete entry-kind byte.
	Type_Flag Type_Flag
	// Name becomes the mandatory path record.
	Name PAX_Header_Name
	// Link_Name becomes an optional link-path record.
	Link_Name PAX_Header_Link_Name
	// Size becomes the mandatory logical-size record.
	Size Entry_Size
	// Mode is proven to fit the fixed fallback header.
	Mode Basic_File_Mode
	// User_Identifier becomes an optional decimal owner record.
	User_Identifier User_Identifier
	// Group_Identifier becomes an optional decimal group record.
	Group_Identifier Group_Identifier
	// User_Name becomes an optional owner-name record.
	User_Name PAX_Header_User_Name
	// Group_Name becomes an optional group-name record.
	Group_Name PAX_Header_Group_Name
	// Modification_Time becomes an optional timestamp record.
	Modification_Time Timestamp
	// Access_Time becomes an optional timestamp record.
	Access_Time Access_Timestamp
	// Change_Time becomes an optional timestamp record.
	Change_Time Change_Timestamp
	// Device_Major is proven to fit the fixed fallback header.
	Device_Major USTAR_Device_Major
	// Device_Minor is proven to fit the fixed fallback header.
	Device_Minor USTAR_Device_Minor
}

// PAX_Header_Invariants composes one bounded PAX writer sequence.
func PAX_Header_Invariants(value PAX_Header, namespace aver.Namespace) {
	Type_Flag_Invariants(value.Type_Flag, namespace)
	PAX_Header_Name_Invariants(value.Name, namespace)
	PAX_Header_Link_Name_Invariants(value.Link_Name, namespace)
	Entry_Size_Invariants(value.Size, namespace)
	Basic_File_Mode_Invariants(value.Mode, namespace)
	User_Identifier_Invariants(value.User_Identifier, namespace)
	Group_Identifier_Invariants(value.Group_Identifier, namespace)
	PAX_Header_User_Name_Invariants(value.User_Name, namespace)
	PAX_Header_Group_Name_Invariants(value.Group_Name, namespace)
	Timestamp_Invariants(value.Modification_Time, namespace)
	Access_Timestamp_Invariants(value.Access_Time, namespace)
	Change_Timestamp_Invariants(value.Change_Time, namespace)
	USTAR_Device_Major_Invariants(value.Device_Major, namespace)
	USTAR_Device_Minor_Invariants(value.Device_Minor, namespace)
}

// Wire_Modification_Seconds is one decoded fixed-header timestamp.
type Wire_Modification_Seconds int64

// Wire_Modification_Seconds_Invariants preserves the complete timestamp domain.
func Wire_Modification_Seconds_Invariants(
	value Wire_Modification_Seconds, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Optional_Integer retains one absent or decoded fixed numeric field.
type Optional_Integer struct {
	// Value preserves the complete decoded integer domain.
	Value Integer
	// Set distinguishes an absent zero-filled field from numeric zero.
	Set bytes.Boolean
}

// Optional_Integer_Invariants composes one optional fixed numeric result.
func Optional_Integer_Invariants(
	value Optional_Integer, namespace aver.Namespace,
) {
	Integer_Invariants(value.Value, namespace)
	bytes.Boolean_Invariants(value.Set, namespace)
}

// Wire_Access_Time retains one optional fixed-header timestamp.
type Wire_Access_Time struct {
	// Seconds preserves the complete decoded field.
	Seconds Access_Seconds
	// Set distinguishes absent GNU time from Unix epoch.
	Set Access_Time_Set
}

// Wire_Access_Time_Invariants composes optional fixed-header access time.
func Wire_Access_Time_Invariants(
	value Wire_Access_Time, namespace aver.Namespace,
) {
	Access_Seconds_Invariants(value.Seconds, namespace)
	Access_Time_Set_Invariants(value.Set, namespace)
}

// Wire_Change_Time retains one optional fixed-header timestamp.
type Wire_Change_Time struct {
	// Seconds preserves the complete decoded field.
	Seconds Change_Seconds
	// Set distinguishes absent GNU time from Unix epoch.
	Set Change_Time_Set
}

// Wire_Change_Time_Invariants composes optional fixed-header change time.
func Wire_Change_Time_Invariants(
	value Wire_Change_Time, namespace aver.Namespace,
) {
	Change_Seconds_Invariants(value.Seconds, namespace)
	Change_Time_Set_Invariants(value.Set, namespace)
}

// Wire_Header_Basic retains fields common to every fixed header.
type Wire_Header_Basic struct {
	// Type_Flag preserves the fixed header entry kind.
	Type_Flag Wire_Type_Flag
	// Size preserves validated logical content size.
	Size Entry_Size
	// Physical_Size preserves stored content size.
	Physical_Size Physical_Size
	// Mode preserves the fixed mode field.
	Mode Wire_File_Mode
	// User_Identifier preserves the fixed owner field.
	User_Identifier Wire_User_Identifier
	// Group_Identifier preserves the fixed group field.
	Group_Identifier Wire_Group_Identifier
	// Modified_Seconds preserves the fixed modification field.
	Modified_Seconds Wire_Modification_Seconds
}

// Wire_Header_Basic_Invariants composes fixed numeric fields.
func Wire_Header_Basic_Invariants(
	value Wire_Header_Basic, namespace aver.Namespace,
) {
	Wire_Type_Flag_Invariants(value.Type_Flag, namespace)
	Entry_Size_Invariants(value.Size, namespace)
	Physical_Size_Invariants(value.Physical_Size, namespace)
	Wire_File_Mode_Invariants(value.Mode, namespace)
	Wire_User_Identifier_Invariants(value.User_Identifier, namespace)
	Wire_Group_Identifier_Invariants(value.Group_Identifier, namespace)
	Wire_Modification_Seconds_Invariants(value.Modified_Seconds, namespace)
}

// Wire_Header_Names retains fixed text and device fields.
type Wire_Header_Names struct {
	// Name preserves the fixed or split path.
	Name Wire_Name
	// Link_Name preserves the fixed link target.
	Link_Name Wire_Link_Name
	// User_Name preserves the fixed owner name.
	User_Name Wire_User_Name
	// Group_Name preserves the fixed group name.
	Group_Name Wire_Group_Name
	// Device_Major preserves the fixed device class.
	Device_Major Wire_Device_Major
	// Device_Minor preserves the fixed device instance.
	Device_Minor Wire_Device_Minor
}

// Wire_Header_Names_Invariants composes fixed text and device fields.
func Wire_Header_Names_Invariants(
	value Wire_Header_Names, namespace aver.Namespace,
) {
	Wire_Name_Invariants(value.Name, namespace)
	Wire_Link_Name_Invariants(value.Link_Name, namespace)
	Wire_User_Name_Invariants(value.User_Name, namespace)
	Wire_Group_Name_Invariants(value.Group_Name, namespace)
	Wire_Device_Major_Invariants(value.Device_Major, namespace)
	Wire_Device_Minor_Invariants(value.Device_Minor, namespace)
}

// Wire_Header_Fixed_Names retains fields before format-specific path resolution.
type Wire_Header_Fixed_Names struct {
	// Name is the suffix held by every fixed header.
	Name Wire_Name_Suffix
	// Link_Name preserves the fixed link target.
	Link_Name Wire_Link_Name
	// User_Name preserves the fixed owner name.
	User_Name Wire_User_Name
	// Group_Name preserves the fixed group name.
	Group_Name Wire_Group_Name
	// Device_Major preserves the fixed device class.
	Device_Major Wire_Device_Major
	// Device_Minor preserves the fixed device instance.
	Device_Minor Wire_Device_Minor
}

// Wire_Header_Fixed_Names_Invariants composes pre-extension fixed fields.
func Wire_Header_Fixed_Names_Invariants(
	value Wire_Header_Fixed_Names, namespace aver.Namespace,
) {
	Wire_Name_Suffix_Invariants(value.Name, namespace)
	Wire_Link_Name_Invariants(value.Link_Name, namespace)
	Wire_User_Name_Invariants(value.User_Name, namespace)
	Wire_Group_Name_Invariants(value.Group_Name, namespace)
	Wire_Device_Major_Invariants(value.Device_Major, namespace)
	Wire_Device_Minor_Invariants(value.Device_Minor, namespace)
}

// Wire_Header_Extensions retains format-specific fixed fields.
type Wire_Header_Extensions struct {
	// Format preserves the resolved fixed-header family.
	Format Reader_Format
	// Access_Time preserves the optional access field.
	Access_Time Wire_Access_Time
	// Change_Time preserves the optional metadata-change field.
	Change_Time Wire_Change_Time
}

// Wire_Header_Extensions_Invariants composes format-specific fields.
func Wire_Header_Extensions_Invariants(
	value Wire_Header_Extensions, namespace aver.Namespace,
) {
	Reader_Format_Invariants(value.Format, namespace)
	Wire_Access_Time_Invariants(value.Access_Time, namespace)
	Wire_Change_Time_Invariants(value.Change_Time, namespace)
}

// Wire_Header is one completely decoded fixed header.
type Wire_Header struct {
	// Basic contains common fixed fields.
	Basic Wire_Header_Basic
	// Names contains fixed text and device fields.
	Names Wire_Header_Names
	// Extensions contains format-specific fixed fields.
	Extensions Wire_Header_Extensions
}

// Wire_Header_Invariants composes one decoded fixed header.
func Wire_Header_Invariants(value Wire_Header, namespace aver.Namespace) {
	Wire_Header_Basic_Invariants(value.Basic, namespace)
	Wire_Header_Names_Invariants(value.Names, namespace)
	Wire_Header_Extensions_Invariants(value.Extensions, namespace)
}

// Physical_Size is stored payload size before sparse expansion.
type Physical_Size int64

// Physical_Size_Invariants protects archive payload slicing.
func Physical_Size_Invariants(value Physical_Size, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), COUNT_MINIMUM, ARCHIVE_SIZE_MAXIMUM).
		Ensure()
}

// Reader_Format excludes selection and unrecognized states after wire parsing.
type Reader_Format uint8

// Reader_Format_Invariants bounds the five readable wire families.
func Reader_Format_Invariants(
	value Reader_Format, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), uint8(FORMAT_V7), uint8(FORMAT_STAR)).
		Ensure()
}

// Raw_Header carries validated archive views before caller-storage copy.
type Raw_Header struct {
	// Format retains wire family before extension resolution.
	Format Reader_Format
	// Type_Flag retains raw file kind.
	Type_Flag Wire_Type_Flag
	// Name retains archive path views.
	Name Field_Value
	// Link_Name retains archive link views.
	Link_Name Header_Link_Name
	// Size is resolved logical entry size.
	Size Entry_Size
	// Physical_Size is payload bytes present in archive.
	Physical_Size Physical_Size
	// Mode retains permission and mode bits.
	Mode Wire_File_Mode
	// User_Identifier retains numeric owner.
	User_Identifier User_Identifier
	// Group_Identifier retains numeric owner group.
	Group_Identifier Group_Identifier
	// User_Name retains archive owner name view.
	User_Name Header_User_Name
	// Group_Name retains archive owner group view.
	Group_Name Header_Group_Name
	// Modification_Time is always present in fixed TAR header.
	Modification_Time Present_Timestamp
	// Access_Time retains optional access instant.
	Access_Time Access_Timestamp
	// Change_Time retains optional metadata-change instant.
	Change_Time Change_Timestamp
	// Device_Major retains device class.
	Device_Major Wire_Device_Major
	// Device_Minor retains device instance.
	Device_Minor Wire_Device_Minor
}

// Raw_Header_Invariants keeps parsed logical and physical sizes bounded.
func Raw_Header_Invariants(value Raw_Header, namespace aver.Namespace) {
	Reader_Format_Invariants(value.Format, namespace)
	Wire_Type_Flag_Invariants(value.Type_Flag, namespace)
	Field_Value_Invariants(value.Name, namespace)
	Header_Link_Name_Invariants(value.Link_Name, namespace)
	Entry_Size_Invariants(value.Size, namespace)
	Physical_Size_Invariants(value.Physical_Size, namespace)
	Wire_File_Mode_Invariants(value.Mode, namespace)
	User_Identifier_Invariants(value.User_Identifier, namespace)
	Group_Identifier_Invariants(value.Group_Identifier, namespace)
	Header_User_Name_Invariants(value.User_Name, namespace)
	Header_Group_Name_Invariants(value.Group_Name, namespace)
	Present_Timestamp_Invariants(value.Modification_Time, namespace)
	Access_Timestamp_Invariants(value.Access_Time, namespace)
	Change_Timestamp_Invariants(value.Change_Time, namespace)
	Wire_Device_Major_Invariants(value.Device_Major, namespace)
	Wire_Device_Minor_Invariants(value.Device_Minor, namespace)
}

// GNU_Numeric_Fields contains values tested against GNU fixed numeric fields.
type GNU_Numeric_Fields struct {
	// Mode occupies GNU mode field.
	Mode File_Mode
	// User_Identifier occupies GNU owner field.
	User_Identifier User_Identifier
	// Group_Identifier occupies GNU group field.
	Group_Identifier Group_Identifier
	// Size occupies GNU entry-size field.
	Size Entry_Size
	// Modification_Time occupies GNU modification field.
	Modification_Time Timestamp
	// Access_Time occupies GNU access field.
	Access_Time Access_Timestamp
	// Change_Time occupies GNU metadata-change field.
	Change_Time Change_Timestamp
	// Device_Major occupies GNU device-class field.
	Device_Major Device_Major
	// Device_Minor occupies GNU device-instance field.
	Device_Minor Device_Minor
}

// GNU_Numeric_Fields_Invariants composes GNU numeric encoding inputs.
func GNU_Numeric_Fields_Invariants(
	value GNU_Numeric_Fields, namespace aver.Namespace,
) {
	File_Mode_Invariants(value.Mode, namespace)
	User_Identifier_Invariants(value.User_Identifier, namespace)
	Group_Identifier_Invariants(value.Group_Identifier, namespace)
	Entry_Size_Invariants(value.Size, namespace)
	Timestamp_Invariants(value.Modification_Time, namespace)
	Access_Timestamp_Invariants(value.Access_Time, namespace)
	Change_Timestamp_Invariants(value.Change_Time, namespace)
	Device_Major_Invariants(value.Device_Major, namespace)
	Device_Minor_Invariants(value.Device_Minor, namespace)
}

// PAX_Text_Fields contains logical text changed by PAX records.
type PAX_Text_Fields struct {
	// Name holds logical path.
	Name Field_Value
	// Link_Name holds logical link target.
	Link_Name Header_Link_Name
	// User_Name holds logical owner name.
	User_Name Header_User_Name
	// Group_Name holds logical group name.
	Group_Name Header_Group_Name
}

// PAX_Text_Fields_Invariants composes PAX text overrides.
func PAX_Text_Fields_Invariants(
	value PAX_Text_Fields, namespace aver.Namespace,
) {
	Field_Value_Invariants(value.Name, namespace)
	Header_Link_Name_Invariants(value.Link_Name, namespace)
	Header_User_Name_Invariants(value.User_Name, namespace)
	Header_Group_Name_Invariants(value.Group_Name, namespace)
}

// PAX_Apply_User_Name leaves one current record after prior override.
type PAX_Apply_User_Name []byte

// PAX_Apply_User_Name_Invariants bounds prior owner override.
func PAX_Apply_User_Name_Invariants(
	value PAX_Apply_User_Name, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, PAX_APPLY_USER_NAME_SIZE_MAXIMUM).
		Ensure()
}

// PAX_Apply_Group_Name leaves one current record after prior override.
type PAX_Apply_Group_Name []byte

// PAX_Apply_Group_Name_Invariants bounds prior group override.
func PAX_Apply_Group_Name_Invariants(
	value PAX_Apply_Group_Name, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COUNT_MINIMUM, PAX_APPLY_GROUP_NAME_SIZE_MAXIMUM).
		Ensure()
}

// PAX_Input_Text_Fields retains fixed and GNU text before local overrides.
type PAX_Input_Text_Fields struct {
	// Name may already contain a GNU long-name override.
	Name Field_Value
	// Link_Name may already contain a GNU long-link override.
	Link_Name Header_Link_Name
	// User_Name still comes from the fixed header.
	User_Name Wire_User_Name
	// Group_Name still comes from the fixed header.
	Group_Name Wire_Group_Name
}

// PAX_Input_Text_Fields_Invariants composes pre-PAX text domains.
func PAX_Input_Text_Fields_Invariants(
	value PAX_Input_Text_Fields, namespace aver.Namespace,
) {
	Field_Value_Invariants(value.Name, namespace)
	Header_Link_Name_Invariants(value.Link_Name, namespace)
	Wire_User_Name_Invariants(value.User_Name, namespace)
	Wire_Group_Name_Invariants(value.Group_Name, namespace)
}

// Header_Text_Fields contains caller-backed text after one decoded header resolves.
type Header_Text_Fields struct {
	// Name holds the copied logical path.
	Name Header_Name
	// Link_Name holds the copied link target.
	Link_Name Header_Link_Name
	// User_Name holds the copied owner name.
	User_Name Header_User_Name
	// Group_Name holds the copied group name.
	Group_Name Header_Group_Name
}

// Header_Text_Fields_Invariants composes decoded caller-backed text.
func Header_Text_Fields_Invariants(
	value Header_Text_Fields, namespace aver.Namespace,
) {
	Header_Name_Invariants(value.Name, namespace)
	Header_Link_Name_Invariants(value.Link_Name, namespace)
	Header_User_Name_Invariants(value.User_Name, namespace)
	Header_Group_Name_Invariants(value.Group_Name, namespace)
}

// PAX_Entry_Size retains nonnegative decimal size until policy validation.
type PAX_Entry_Size int64

// PAX_Entry_Size_Invariants covers parsed nonnegative decimal domain.
func PAX_Entry_Size_Invariants(
	value PAX_Entry_Size, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), COUNT_MINIMUM, int64(INTEGER_MAXIMUM)).
		Ensure()
}

// PAX_Physical_Size retains nonnegative stored size until policy validation.
type PAX_Physical_Size int64

// PAX_Physical_Size_Invariants covers parsed nonnegative decimal domain.
func PAX_Physical_Size_Invariants(
	value PAX_Physical_Size, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), COUNT_MINIMUM, int64(INTEGER_MAXIMUM)).
		Ensure()
}

// PAX_Numeric_Fields contains logical numbers changed by PAX records.
type PAX_Numeric_Fields struct {
	// Size stays hostile until all records apply.
	Size PAX_Entry_Size
	// Physical_Size stays hostile until all records apply.
	Physical_Size PAX_Physical_Size
	// User_Identifier holds numeric owner.
	User_Identifier User_Identifier
	// Group_Identifier holds numeric group.
	Group_Identifier Group_Identifier
}

// PAX_Numeric_Fields_Invariants composes PAX numeric overrides.
func PAX_Numeric_Fields_Invariants(
	value PAX_Numeric_Fields, namespace aver.Namespace,
) {
	PAX_Entry_Size_Invariants(value.Size, namespace)
	PAX_Physical_Size_Invariants(value.Physical_Size, namespace)
	User_Identifier_Invariants(value.User_Identifier, namespace)
	Group_Identifier_Invariants(value.Group_Identifier, namespace)
}

// PAX_Input_Numeric_Fields retains fixed numbers before local overrides.
type PAX_Input_Numeric_Fields struct {
	// Size is the logical fixed-header content size.
	Size Entry_Size
	// Physical_Size is the fixed-header stored size.
	Physical_Size Physical_Size
	// User_Identifier still comes from the fixed numeric field.
	User_Identifier Wire_User_Identifier
	// Group_Identifier still comes from the fixed numeric field.
	Group_Identifier Wire_Group_Identifier
}

// PAX_Input_Numeric_Fields_Invariants composes pre-PAX numeric domains.
func PAX_Input_Numeric_Fields_Invariants(
	value PAX_Input_Numeric_Fields, namespace aver.Namespace,
) {
	Entry_Size_Invariants(value.Size, namespace)
	Physical_Size_Invariants(value.Physical_Size, namespace)
	Wire_User_Identifier_Invariants(value.User_Identifier, namespace)
	Wire_Group_Identifier_Invariants(value.Group_Identifier, namespace)
}

// PAX_Timestamp_Fields contains instants changed by PAX records.
type PAX_Timestamp_Fields struct {
	// Modification_Time holds content-change instant.
	Modification_Time Present_Timestamp
	// Access_Time holds access instant.
	Access_Time Access_Timestamp
	// Change_Time holds metadata-change instant.
	Change_Time Change_Timestamp
}

// PAX_Timestamp_Fields_Invariants composes PAX timestamp overrides.
func PAX_Timestamp_Fields_Invariants(
	value PAX_Timestamp_Fields, namespace aver.Namespace,
) {
	Present_Timestamp_Invariants(value.Modification_Time, namespace)
	Access_Timestamp_Invariants(value.Access_Time, namespace)
	Change_Timestamp_Invariants(value.Change_Time, namespace)
}

// Present_Timestamp is one instant whose fixed header guarantees presence.
type Present_Timestamp struct {
	// Seconds holds Unix epoch offset.
	Seconds Integer
	// Nanoseconds holds fractional second.
	Nanoseconds Nanosecond_Count
}

// Present_Timestamp_Invariants composes one required normalized instant.
func Present_Timestamp_Invariants(
	value Present_Timestamp, namespace aver.Namespace,
) {
	Integer_Invariants(value.Seconds, namespace)
	Nanosecond_Count_Invariants(value.Nanoseconds, namespace)
}

// PAX_Timestamp_Input contains fixed-header instants before PAX overrides.
type PAX_Timestamp_Input struct {
	// Modification_Seconds is always present in a valid fixed header.
	Modification_Seconds Wire_Modification_Seconds
	// Access_Time preserves optional GNU or STAR access time.
	Access_Time Wire_Access_Time
	// Change_Time preserves optional GNU or STAR metadata time.
	Change_Time Wire_Change_Time
}

// PAX_Timestamp_Input_Invariants composes fixed-header timestamp inputs.
func PAX_Timestamp_Input_Invariants(
	value PAX_Timestamp_Input, namespace aver.Namespace,
) {
	Wire_Modification_Seconds_Invariants(value.Modification_Seconds, namespace)
	Wire_Access_Time_Invariants(value.Access_Time, namespace)
	Wire_Change_Time_Invariants(value.Change_Time, namespace)
}

// Reader_Init binds transport and caller parser storage without starting work.
func Reader_Init(
	reader *Reader, stream nbio.Stream, storage Reader_Storage,
) (status Initialization_Status) {
	defer func() {
		Initialization_Status_Invariants(status, "Reader_Init.status")
	}()
	Reader_Invariants(reader, "Reader_Init.reader")
	Reader_Storage_Invariants(storage, "Reader_Init.storage")
	if stream.Procedure == nil {
		return STATUS_STORAGE_INVALID
	}
	if len(storage.Block) != BLOCK_SIZE {
		return STATUS_STORAGE_INVALID
	}
	if len(storage.Metadata) > READER_METADATA_SIZE_MAXIMUM {
		return STATUS_STORAGE_INVALID
	}
	if slices_overlap(Overlap_Left(storage.Block), Overlap_Right(storage.Metadata)) {
		return STATUS_STORAGE_INVALID
	}
	*reader = Reader{
		Stream: stream,
		Archive: Reader_Archive{
			Block: storage.Block, Metadata: storage.Metadata,
		},
		Initialized: true,
	}
	return STATUS_OK
}

// Reader_Next resolves the next logical header after Stream retires required bytes.
func Reader_Next(
	reader *Reader, completion *nbio.Completion, storage Header_Storage,
	callback nbio.Callback,
) {
	Reader_Invariants(reader, "Reader_Next.reader")
	Header_Storage_Invariants(storage, "Reader_Next.storage")
	aver.Always(completion != nil, "Reader_Next has a completion.")
	aver.Always(
		completion == &reader.Completion,
		"Reader_Next submits the completion owned by its Reader.",
	)
	aver.Always(callback != nil, "Reader_Next has a callback.")
	aver.Always(!reader.Active, "Reader_Next owns a free callback slot.")
	reader.Active = true
	reader.Callback = callback
	reader.Status = STATUS_OK
	reader.Count = 0
	reader.Transfer = Reader_Transfer{}
	completion.Data = 0
	completion.Error = nil
	reader.Archive.Header_Storage = storage
	if !bool(reader.Initialized) {
		reader.Status = STATUS_STORAGE_INVALID
		reader_finish(completion)
		return
	}
	if !bool(reader_header_storage_valid(
		Reader_Header_Block(reader.Archive.Block), reader.Archive.Metadata, storage,
	)) {
		reader.Status = STATUS_STORAGE_INVALID
		reader_finish(completion)
		return
	}
	reader.Archive.Pending_PAX_Position = 0
	reader.Archive.Pending_PAX_Size = 0
	reader.Archive.Long_Name = nil
	reader.Archive.Long_Link = nil
	reader.Archive.Metadata_Position = 0
	reader.Skip_Debt = Reader_Skip_Debt(reader.Archive.Physical_Debt)
	if reader.Skip_Debt != 0 {
		reader.Transfer.Stage = READER_STAGE_SKIP_CONTENT
	} else if reader.Archive.Padding != 0 {
		reader.Skip_Debt = Reader_Skip_Debt(reader.Archive.Padding)
		reader.Transfer.Stage = READER_STAGE_SKIP_PADDING
	} else {
		reader.Transfer.Stage = READER_STAGE_HEADER
	}
	reader_progress(completion)
}

// Reader_Read fills one bounded caller destination from current entry content.
func Reader_Read(
	reader *Reader, completion *nbio.Completion, destination Destination,
	callback nbio.Callback,
) {
	Reader_Invariants(reader, "Reader_Read.reader")
	Destination_Invariants(destination, "Reader_Read.destination")
	aver.Always(completion != nil, "Reader_Read has a completion.")
	aver.Always(
		completion == &reader.Completion,
		"Reader_Read submits the completion owned by its Reader.",
	)
	aver.Always(callback != nil, "Reader_Read has a callback.")
	aver.Always(!reader.Active, "Reader_Read owns a free callback slot.")
	reader.Active = true
	reader.Callback = callback
	reader.Status = STATUS_OK
	reader.Count = 0
	reader.Transfer = Reader_Transfer{}
	completion.Data = 0
	completion.Error = nil
	if !reader.Initialized {
		reader.Status = STATUS_STORAGE_INVALID
		reader_finish(completion)
		return
	}
	if !bool(reader.Archive.Entry_Active) {
		reader.Status = STATUS_END
		reader_finish(completion)
		return
	}
	if int(reader.Archive.Logical_Position) == int(reader.Archive.Logical_Size) {
		reader.Status = STATUS_END
		reader_finish(completion)
		return
	}
	if len(destination) == 0 {
		reader.Status = STATUS_OUTPUT_TOO_SMALL
		reader_finish(completion)
		return
	}
	available_count := int(reader.Archive.Logical_Size) -
		int(reader.Archive.Logical_Position)
	if len(destination) > available_count {
		destination = destination[:available_count]
	}
	reader.Transfer.Stage = READER_STAGE_CONTENT
	reader.Transfer.Buffer = Reader_Transfer_Buffer(destination)
	reader.Transfer.Offset = 0
	reader.Transfer.Needed = Reader_Transfer_Needed(len(destination))
	reader_progress(completion)
}

// Reader progress needs a trampoline because Memory Stream may retire inline.
func reader_progress(completion *nbio.Completion) {
	reader := (*Reader)(unsafe.Pointer(completion))
	for bool(reader.Active) && !bool(reader.Transfer.Wait_Active) {
		if int(reader.Transfer.Offset) < int(reader.Transfer.Needed) {
			start := reader.Transfer.Offset
			end := reader.Transfer.Needed
			buffer := reader.Transfer.Buffer[start:end]
			reader.Transfer.Wait_Active = true
			reader.Transfer.Submission_Active = true
			reader.Transfer.Continue = false
			nbio.Read(reader.Stream, completion, buffer, reader_stream_complete)
			reader.Transfer.Submission_Active = false
			retired_inline := reader.Transfer.Continue
			reader.Transfer.Continue = false
			if !retired_inline {
				return
			}
			continue
		}
		if reader.Transfer.Needed != 0 {
			reader_transfer_complete(completion)
			continue
		}
		switch reader.Transfer.Stage {
		case READER_STAGE_SKIP_CONTENT:
			if reader.Skip_Debt == 0 {
				reader.Archive.Physical_Debt = 0
				reader.Skip_Debt = Reader_Skip_Debt(reader.Archive.Padding)
				reader.Transfer.Stage = READER_STAGE_SKIP_PADDING
				continue
			}
			amount := reader.Skip_Debt
			if int(amount) > len(reader.Archive.Block) {
				amount = Reader_Skip_Debt(len(reader.Archive.Block))
			}
			reader.Transfer.Buffer = Reader_Transfer_Buffer(
				reader.Archive.Block[:amount],
			)
			reader.Transfer.Offset = 0
			reader.Transfer.Needed = Reader_Transfer_Needed(amount)
		case READER_STAGE_SKIP_PADDING:
			if reader.Skip_Debt == 0 {
				reader.Archive.Padding = 0
				reader.Archive.Entry_Active = false
				reader.Transfer.Stage = READER_STAGE_HEADER
				continue
			}
			amount := reader.Skip_Debt
			if int(amount) > len(reader.Archive.Block) {
				amount = Reader_Skip_Debt(len(reader.Archive.Block))
			}
			reader.Transfer.Buffer = Reader_Transfer_Buffer(
				reader.Archive.Block[:amount],
			)
			reader.Transfer.Offset = 0
			reader.Transfer.Needed = Reader_Transfer_Needed(amount)
		case READER_STAGE_HEADER:
			reader.Transfer.Buffer = Reader_Transfer_Buffer(
				reader.Archive.Block[:BLOCK_SIZE],
			)
			reader.Transfer.Offset = 0
			reader.Transfer.Needed = BLOCK_SIZE
		default:
			reader.Status = STATUS_STORAGE_INVALID
			reader_finish(completion)
		}
	}
}

func reader_stream_complete(completion *nbio.Completion) {
	reader := (*Reader)(unsafe.Pointer(completion))
	aver.Always(reader.Active, "Reader callback belongs to one active operation.")
	aver.Always(
		reader.Transfer.Wait_Active,
		"Reader callback retires one submitted transfer.",
	)
	reader.Transfer.Wait_Active = false
	requested := int(reader.Transfer.Needed) - int(reader.Transfer.Offset)
	count := completion.Data
	if count < 0 {
		count = 0
		completion.Error = nbio.Stream_Negative_Read
	}
	if count > requested {
		count = 0
		completion.Error = nbio.Stream_Short_Buffer
	}
	reader.Transfer.Offset += Reader_Transfer_Offset(count)
	completion.Data = count
	failed := completion.Error != nil
	if count == 0 {
		failed = true
	}
	if failed {
		if int(reader.Transfer.Offset) == int(reader.Transfer.Needed) {
			completion.Error = nil
		} else {
			if completion.Error == nil {
				completion.Error = nbio.Stream_No_Progress
				reader.Status = STATUS_TRANSPORT_FAILED
			} else {
				switch completion.Error {
				case nbio.Stream_EOF, nbio.Stream_Unexpected_EOF:
					completion.Error = nil
					if reader.Transfer.Stage == READER_STAGE_HEADER {
						if reader.Transfer.Offset == 0 {
							reader.Status = STATUS_END
						} else {
							reader.Status = STATUS_INPUT_INVALID
						}
					} else {
						reader.Status = STATUS_INPUT_INVALID
					}
				default:
					reader.Status = STATUS_TRANSPORT_FAILED
				}
			}
			if reader.Transfer.Stage == READER_STAGE_CONTENT {
				reader.Archive.Logical_Position += Logical_Position(
					reader.Transfer.Offset,
				)
				reader.Archive.Physical_Debt -= Physical_Debt(
					reader.Transfer.Offset,
				)
			}
			reader.Transfer.Needed = Reader_Transfer_Needed(reader.Transfer.Offset)
			reader_finish(completion)
		}
	}
	if !reader.Active {
		return
	}
	if reader.Transfer.Submission_Active {
		reader.Transfer.Continue = true
		return
	}
	reader_progress(completion)
}

func reader_transfer_complete(completion *nbio.Completion) {
	reader := (*Reader)(unsafe.Pointer(completion))
	amount := reader.Transfer.Needed
	reader.Transfer.Buffer = nil
	reader.Transfer.Offset = 0
	reader.Transfer.Needed = 0
	switch reader.Transfer.Stage {
	case READER_STAGE_SKIP_CONTENT:
		reader.Skip_Debt -= Reader_Skip_Debt(amount)
		reader.Archive.Physical_Debt -= Physical_Debt(amount)
	case READER_STAGE_SKIP_PADDING:
		reader.Skip_Debt -= Reader_Skip_Debt(amount)
	case READER_STAGE_HEADER:
		complete, header_status := reader_header_complete(completion)
		reader.Status = Status(header_status)
		if bool(complete) {
			reader_finish(completion)
			return
		}
		start := Count(reader.Archive.Metadata_Position)
		blocks := padded_block_count(
			Metadata_Payload_Size(reader.Archive.Extension_Size),
		)
		padded := Count(blocks) * BLOCK_SIZE
		if padded == 0 {
			reader_metadata_complete(completion)
			return
		}
		reader.Transfer.Stage = READER_STAGE_METADATA
		reader.Transfer.Buffer = Reader_Transfer_Buffer(
			reader.Archive.Metadata[start : start+Count(padded)],
		)
		reader.Transfer.Offset = 0
		reader.Transfer.Needed = Reader_Transfer_Needed(padded)
	case READER_STAGE_METADATA:
		reader_metadata_complete(completion)
	case READER_STAGE_CONTENT:
		reader.Archive.Logical_Position += Logical_Position(amount)
		reader.Archive.Physical_Debt -= Physical_Debt(amount)
		reader.Count = Count(amount)
		reader_finish(completion)
	default:
		reader.Status = STATUS_STORAGE_INVALID
		reader_finish(completion)
	}
}

func reader_metadata_complete(completion *nbio.Completion) {
	reader := (*Reader)(unsafe.Pointer(completion))
	start := Count(reader.Archive.Metadata_Position)
	physical_size := Count(reader.Archive.Extension_Size)
	blocks := padded_block_count(Metadata_Payload_Size(physical_size))
	padded := Count(blocks) * BLOCK_SIZE
	payload := Field(reader.Archive.Metadata[start : start+physical_size])
	reader.Archive.Metadata_Position += Metadata_Position(padded)
	switch Type_Flag(reader.Archive.Extension_Type) {
	case TYPE_PAX_LOCAL:
		if !pax_records_valid(PAX_Records(payload)) {
			reader.Status = STATUS_INPUT_INVALID
			reader_finish(completion)
			return
		}
		reader.Archive.Pending_PAX_Position = Pending_PAX_Position(start)
		reader.Archive.Pending_PAX_Size = Pending_PAX_Size(physical_size)
	case TYPE_PAX_GLOBAL:
		reader.Status = STATUS_FORMAT_UNSUPPORTED
		reader_finish(completion)
		return
	case TYPE_GNU_LONG_NAME:
		value := metadata_field_bytes(payload)
		if len(value) > HEADER_TEXT_SIZE_MAXIMUM {
			reader.Status = STATUS_FIELD_TOO_LONG
			reader_finish(completion)
			return
		}
		reader.Archive.Long_Name = Reader_Long_Name(value)
	case TYPE_GNU_LONG_LINK:
		value := metadata_field_bytes(payload)
		if len(value) > HEADER_TEXT_SIZE_MAXIMUM {
			reader.Status = STATUS_FIELD_TOO_LONG
			reader_finish(completion)
			return
		}
		reader.Archive.Long_Link = Reader_Long_Link(value)
	default:
		reader.Status = STATUS_INPUT_INVALID
		reader_finish(completion)
		return
	}
	reader.Transfer.Stage = READER_STAGE_HEADER
}

func reader_header_complete(
	completion *nbio.Completion,
) (complete Reader_Header_Complete, status Reader_Header_Status) {
	defer func() {
		Reader_Header_Complete_Invariants(complete, "reader_header_complete.complete")
		Reader_Header_Status_Invariants(status, "reader_header_complete.status")
	}()
	reader := (*Reader)(unsafe.Pointer(completion))
	archive := &reader.Archive
	header, parse_status := parse_header(Reader_Header_Block(archive.Block))
	if parse_status != Parse_Status(STATUS_OK) {
		return true, Reader_Header_Status(parse_status)
	}
	switch header.Basic.Type_Flag {
	case Wire_Type_Flag(TYPE_PAX_LOCAL), Wire_Type_Flag(TYPE_PAX_GLOBAL),
		Wire_Type_Flag(TYPE_GNU_LONG_NAME), Wire_Type_Flag(TYPE_GNU_LONG_LINK):
		if header.Basic.Physical_Size > SPECIAL_FILE_SIZE_MAXIMUM {
			return true, Reader_Header_Status(STATUS_FIELD_TOO_LONG)
		}
		blocks := padded_block_count(
			Metadata_Payload_Size(header.Basic.Physical_Size),
		)
		start := Count(archive.Metadata_Position)
		if !metadata_available(
			Metadata_Block_Position(start/BLOCK_SIZE), blocks,
			Metadata_Available_Boundary(len(archive.Metadata)),
		) {
			return true, Reader_Header_Status(STATUS_OUTPUT_TOO_SMALL)
		}
		archive.Extension_Type = Reader_Extension_Type(header.Basic.Type_Flag)
		archive.Extension_Size = Reader_Extension_Size(header.Basic.Physical_Size)
		return false, Reader_Header_Status(STATUS_OK)
	}
	failure, successful := reader_header_resolve(completion, header)
	if !successful {
		return true, Reader_Header_Status(failure)
	}
	return true, Reader_Header_Status(STATUS_OK)
}

func reader_header_resolve(
	completion *nbio.Completion, header Wire_Header,
) (failure Reader_Resolve_Failure, successful Reader_Resolve_Success) {
	defer func() {
		Reader_Resolve_Failure_Invariants(failure, "reader_header_resolve.failure")
		Reader_Resolve_Success_Invariants(successful, "reader_header_resolve.successful")
	}()
	Wire_Header_Invariants(header, "reader_header_resolve.header")
	reader := (*Reader)(unsafe.Pointer(completion))
	archive := &reader.Archive
	text_input, format := reader_header_text_input(
		header, archive.Long_Name, archive.Long_Link,
	)
	text := PAX_Text_Fields{
		Name: text_input.Name, Link_Name: text_input.Link_Name,
		User_Name:  Header_User_Name(text_input.User_Name),
		Group_Name: Header_Group_Name(text_input.Group_Name),
	}
	numeric_input := PAX_Input_Numeric_Fields{
		Size: header.Basic.Size, Physical_Size: header.Basic.Physical_Size,
		User_Identifier:  header.Basic.User_Identifier,
		Group_Identifier: header.Basic.Group_Identifier,
	}
	timestamp_input := PAX_Timestamp_Input{
		Modification_Seconds: header.Basic.Modified_Seconds,
		Access_Time:          header.Extensions.Access_Time,
		Change_Time:          header.Extensions.Change_Time,
	}
	numeric := PAX_Numeric_Fields{
		Size:             PAX_Entry_Size(numeric_input.Size),
		Physical_Size:    PAX_Physical_Size(numeric_input.Physical_Size),
		User_Identifier:  User_Identifier(numeric_input.User_Identifier),
		Group_Identifier: Group_Identifier(numeric_input.Group_Identifier),
	}
	modification_seconds := Integer(timestamp_input.Modification_Seconds)
	access := timestamp_input.Access_Time
	change := timestamp_input.Change_Time
	timestamps := PAX_Timestamp_Fields{
		Modification_Time: Present_Timestamp{Seconds: modification_seconds},
		Access_Time:       Access_Timestamp{Seconds: access.Seconds, Set: access.Set},
		Change_Time:       Change_Timestamp{Seconds: change.Seconds, Set: change.Set},
	}
	if archive.Pending_PAX_Size != 0 {
		pax_start := Count(archive.Pending_PAX_Position)
		pax_end := pax_start + Count(archive.Pending_PAX_Size)
		pax_records := PAX_Valid_Records(archive.Metadata[pax_start:pax_end])
		var pax_status PAX_Apply_Status
		text, numeric, timestamps, pax_status = pax_apply(
			text_input, numeric_input, timestamp_input, pax_records,
		)
		if pax_status != PAX_Apply_Status(STATUS_OK) {
			return Reader_Resolve_Failure(pax_status), false
		}
		format = Reader_Format(FORMAT_PAX)
	}
	if !reader_header_numeric_valid(numeric) {
		return Reader_Resolve_Failure(STATUS_FIELD_TOO_LONG), false
	}
	raw := reader_raw_header(
		header, format, text, Entry_Size(numeric.Size),
		Physical_Size(numeric.Physical_Size), numeric.User_Identifier,
		numeric.Group_Identifier, timestamps,
	)
	if !reader_header_commit(completion, raw) {
		return Reader_Resolve_Failure(reader.Status), false
	}
	return Reader_Resolve_Failure(STATUS_INPUT_INVALID), true
}

func reader_header_text_input(
	header Wire_Header, long_name Reader_Long_Name, long_link Reader_Long_Link,
) (input PAX_Input_Text_Fields, format Reader_Format) {
	defer func() {
		PAX_Input_Text_Fields_Invariants(input, "reader_header_text_input.input")
		Reader_Format_Invariants(format, "reader_header_text_input.format")
	}()
	Wire_Header_Invariants(header, "reader_header_text_input.header")
	Reader_Long_Name_Invariants(long_name, "reader_header_text_input.long_name")
	Reader_Long_Link_Invariants(long_link, "reader_header_text_input.long_link")
	input = PAX_Input_Text_Fields{
		Name: Field_Value{
			Prefix: Name_Prefix(header.Names.Name.Prefix),
			Suffix: Name_Suffix(header.Names.Name.Suffix),
			Joined: Name_Joined(header.Names.Name.Joined),
		},
		Link_Name: Header_Link_Name(header.Names.Link_Name),
		User_Name: header.Names.User_Name, Group_Name: header.Names.Group_Name,
	}
	format = header.Extensions.Format
	if len(long_name) != 0 {
		input.Name = Field_Value{Suffix: Name_Suffix(long_name)}
		format = Reader_Format(FORMAT_GNU)
	}
	if len(long_link) != 0 {
		input.Link_Name = Header_Link_Name(long_link)
		format = Reader_Format(FORMAT_GNU)
	}
	return input, format
}

func reader_raw_header(
	header Wire_Header, format Reader_Format, text PAX_Text_Fields,
	size Entry_Size, physical_size Physical_Size,
	user_identifier User_Identifier, group_identifier Group_Identifier,
	timestamps PAX_Timestamp_Fields,
) (raw Raw_Header) {
	defer func() { Raw_Header_Invariants(raw, "reader_raw_header.raw") }()
	Wire_Header_Invariants(header, "reader_raw_header.header")
	Reader_Format_Invariants(format, "reader_raw_header.format")
	PAX_Text_Fields_Invariants(text, "reader_raw_header.text")
	Entry_Size_Invariants(size, "reader_raw_header.size")
	Physical_Size_Invariants(physical_size, "reader_raw_header.physical_size")
	User_Identifier_Invariants(user_identifier, "reader_raw_header.user_identifier")
	Group_Identifier_Invariants(group_identifier, "reader_raw_header.group_identifier")
	PAX_Timestamp_Fields_Invariants(timestamps, "reader_raw_header.timestamps")
	return Raw_Header{
		Format: format, Type_Flag: header.Basic.Type_Flag,
		Name: text.Name, Link_Name: text.Link_Name,
		Size: size, Physical_Size: physical_size,
		Mode: header.Basic.Mode, User_Identifier: user_identifier,
		Group_Identifier: group_identifier,
		User_Name:        text.User_Name, Group_Name: text.Group_Name,
		Modification_Time: Present_Timestamp{
			Seconds:     timestamps.Modification_Time.Seconds,
			Nanoseconds: timestamps.Modification_Time.Nanoseconds,
		},
		Access_Time: timestamps.Access_Time, Change_Time: timestamps.Change_Time,
		Device_Major: header.Names.Device_Major,
		Device_Minor: header.Names.Device_Minor,
	}
}

func reader_header_numeric_valid(value PAX_Numeric_Fields) (valid bytes.Boolean) {
	defer func() {
		bytes.Boolean_Invariants(valid, "reader_header_numeric_valid.valid")
	}()
	PAX_Numeric_Fields_Invariants(value, "reader_header_numeric_valid.value")
	if value.Size < 0 {
		return false
	}
	if value.Size > ARCHIVE_SIZE_MAXIMUM {
		return false
	}
	if value.Physical_Size < 0 {
		return false
	}
	return value.Physical_Size <= ARCHIVE_SIZE_MAXIMUM
}

func reader_header_commit(
	completion *nbio.Completion, raw Raw_Header,
) (committed bytes.Boolean) {
	defer func() {
		bytes.Boolean_Invariants(committed, "reader_header_commit.committed")
	}()
	Raw_Header_Invariants(raw, "reader_header_commit.raw")
	reader := (*Reader)(unsafe.Pointer(completion))
	archive := &reader.Archive
	if raw.Type_Flag == Wire_Type_Flag(TYPE_GNU_SPARSE) {
		reader.Status = STATUS_FORMAT_UNSUPPORTED
		return false
	}
	text := PAX_Text_Fields{
		Name: raw.Name, Link_Name: raw.Link_Name,
		User_Name: raw.User_Name, Group_Name: raw.Group_Name,
	}
	if !field_storage_fits(text, archive.Header_Storage) {
		reader.Status = STATUS_OUTPUT_TOO_SMALL
		return false
	}
	stored_text := reader_header_store_text(archive.Header_Storage, text)
	archive.Header = Header{
		Format: Format(raw.Format), Type_Flag: Type_Flag(raw.Type_Flag),
		Name: stored_text.Name, Link_Name: stored_text.Link_Name,
		Size: raw.Size, Mode: File_Mode(raw.Mode),
		User_Identifier:  raw.User_Identifier,
		Group_Identifier: raw.Group_Identifier,
		User_Name:        stored_text.User_Name, Group_Name: stored_text.Group_Name,
		Modification_Time: Timestamp{
			Seconds:     raw.Modification_Time.Seconds,
			Nanoseconds: raw.Modification_Time.Nanoseconds, Set: true,
		},
		Access_Time: raw.Access_Time, Change_Time: raw.Change_Time,
		Device_Major: Device_Major(raw.Device_Major),
		Device_Minor: Device_Minor(raw.Device_Minor),
	}
	archive.Physical_Debt = Physical_Debt(raw.Physical_Size)
	archive.Logical_Size = Logical_Count(raw.Size)
	if header_only_type(Type_Flag(raw.Type_Flag)) {
		archive.Physical_Debt = 0
		archive.Logical_Size = 0
	}
	archive.Logical_Position = 0
	archive.Padding = Padding_Count(block_padding(Unpadded_Size(raw.Physical_Size)))
	archive.Entry_Active = true
	return true
}

func reader_header_store_text(
	storage Header_Storage, value PAX_Text_Fields,
) (text Header_Text_Fields) {
	defer func() {
		Header_Text_Fields_Invariants(text, "reader_header_store_text.text")
	}()
	Header_Storage_Invariants(storage, "reader_header_store_text.storage")
	PAX_Text_Fields_Invariants(value, "reader_header_store_text.value")
	text.Name = copy_field(storage.Name, value.Name)
	link_count := copy(storage.Link_Name, value.Link_Name)
	text.Link_Name = Header_Link_Name(storage.Link_Name[:link_count])
	user_count := copy(storage.User_Name, value.User_Name)
	text.User_Name = Header_User_Name(storage.User_Name[:user_count])
	group_count := copy(storage.Group_Name, value.Group_Name)
	text.Group_Name = Header_Group_Name(storage.Group_Name[:group_count])
	return text
}

func reader_finish(completion *nbio.Completion) {
	reader := (*Reader)(unsafe.Pointer(completion))
	callback := reader.Callback
	completion.Data = int(reader.Count)
	reader.Callback = nil
	reader.Archive.Header_Storage = Header_Storage{}
	reader.Transfer = Reader_Transfer{}
	reader.Active = false
	callback(completion)
}

func reader_header_storage_valid(
	block Reader_Header_Block, metadata Reader_Metadata, storage Header_Storage,
) (valid bytes.Boolean) {
	defer func() {
		bytes.Boolean_Invariants(valid, "reader_header_storage_valid.valid")
	}()
	Reader_Header_Block_Invariants(block, "reader_header_storage_valid.block")
	Reader_Metadata_Invariants(metadata, "reader_header_storage_valid.metadata")
	Header_Storage_Invariants(storage, "reader_header_storage_valid.storage")
	if !header_storage_valid(storage) {
		return false
	}
	if reader_storage_overlaps(Overlap_Right(block), storage) {
		return false
	}
	return !reader_storage_overlaps(Overlap_Right(metadata), storage)
}

func reader_storage_overlaps(
	workspace Overlap_Right, storage Header_Storage,
) (overlaps bytes.Boolean) {
	defer func() {
		bytes.Boolean_Invariants(overlaps, "reader_storage_overlaps.overlaps")
	}()
	Overlap_Right_Invariants(workspace, "reader_storage_overlaps.workspace")
	Header_Storage_Invariants(storage, "reader_storage_overlaps.storage")
	if slices_overlap(Overlap_Left(storage.Name), Overlap_Right(workspace)) {
		return true
	}
	if slices_overlap(Overlap_Left(storage.Link_Name), Overlap_Right(workspace)) {
		return true
	}
	if slices_overlap(Overlap_Left(storage.User_Name), Overlap_Right(workspace)) {
		return true
	}
	return slices_overlap(Overlap_Left(storage.Group_Name), Overlap_Right(workspace))
}

// Writer_Init binds transport and caller header staging without submitting work.
func Writer_Init(
	writer *Writer, stream nbio.Stream, storage Writer_Header_Storage,
) (status Initialization_Status) {
	defer func() {
		Initialization_Status_Invariants(status, "Writer_Init.status")
	}()
	Writer_Invariants(writer, "Writer_Init.writer")
	Writer_Header_Storage_Invariants(storage, "Writer_Init.storage")
	if stream.Procedure == nil {
		return STATUS_STORAGE_INVALID
	}
	if len(storage) < WRITER_STORAGE_SIZE_MINIMUM {
		return STATUS_STORAGE_INVALID
	}
	if len(storage) > WRITER_STORAGE_SIZE_MAXIMUM {
		return STATUS_STORAGE_INVALID
	}
	*writer = Writer{
		Stream: stream, Destination: storage, Initialized: true,
	}
	return STATUS_OK
}

// Writer_Write_Header stages and submits one complete header sequence.
func Writer_Write_Header(
	writer *Writer, completion *nbio.Completion, header *Header_Unvalidated,
	callback nbio.Callback,
) {
	Writer_Invariants(writer, "Writer_Write_Header.writer")
	Header_Unvalidated_Invariants(header, "Writer_Write_Header.header")
	aver.Always(completion != nil, "Writer_Write_Header has a completion.")
	aver.Always(
		completion == &writer.Completion,
		"Writer_Write_Header submits the completion owned by its Writer.",
	)
	aver.Always(header != nil, "Writer_Write_Header has a header.")
	aver.Always(callback != nil, "Writer_Write_Header has a callback.")
	aver.Always(!writer.Active, "Writer_Write_Header owns a free callback slot.")
	writer.Active = true
	writer.Callback = callback
	writer.Operation = WRITER_OPERATION_HEADER
	writer.Status = STATUS_OK
	writer.Count = 0
	writer.Source = nil
	writer.Position = 0
	completion.Data = 0
	completion.Error = nil
	validated, validation_status := Header_Validate(header)
	if validation_status != STATUS_OK {
		writer.Status = Status(validation_status)
		writer_finish(completion)
		return
	}
	if !writer_header_state_valid(completion) {
		writer_finish(completion)
		return
	}
	status := Header_Stage_Status(STATUS_OK)
	if len(header.PAX_Records) == 0 {
		status = writer_stage_header(
			completion, validated.Format, validated.Type_Flag,
			Writer_Name(validated.Name), validated.Link_Name, validated.Size,
			validated.Mode, validated.User_Identifier, validated.Group_Identifier,
			validated.User_Name, validated.Group_Name,
			validated.Modification_Time, validated.Access_Time, validated.Change_Time,
			validated.Device_Major, validated.Device_Minor,
		)
	} else {
		if len(header.PAX_Records) > PAX_HEADER_RECORDS_SIZE_MAXIMUM {
			writer.Status = STATUS_FIELD_TOO_LONG
			writer_finish(completion)
			return
		}
		status = Header_Stage_Status(writer_stage_header_records(
			completion, validated.Format, validated.Type_Flag,
			Writer_Name(validated.Name), validated.Link_Name, validated.Size,
			validated.Mode, validated.User_Identifier, validated.Group_Identifier,
			validated.User_Name, validated.Group_Name,
			validated.Modification_Time, validated.Access_Time, validated.Change_Time,
			validated.Device_Major, validated.Device_Minor,
			PAX_Header_Records(header.PAX_Records),
		))
	}
	if status != Header_Stage_Status(STATUS_OK) {
		writer.Status = Status(status)
		writer_finish(completion)
		return
	}
	writer.Source = Source(writer.Destination[:writer.Position])
	writer_submit_progress(completion)
}

func writer_stage_header(
	completion *nbio.Completion,
	format Format,
	type_flag Type_Flag,
	name Writer_Name,
	link_name Header_Link_Name,
	size Entry_Size,
	mode File_Mode,
	user_identifier User_Identifier,
	group_identifier Group_Identifier,
	user_name Header_User_Name,
	group_name Header_Group_Name,
	modified Timestamp,
	accessed Access_Timestamp,
	changed Change_Timestamp,
	device_major Device_Major,
	device_minor Device_Minor,
) (status Header_Stage_Status) {
	defer func() { Header_Stage_Status_Invariants(status, "writer_stage_header.status") }()
	Format_Invariants(format, "writer_stage_header.format")
	Type_Flag_Invariants(type_flag, "writer_stage_header.type_flag")
	Writer_Name_Invariants(name, "writer_stage_header.name")
	Header_Link_Name_Invariants(link_name, "writer_stage_header.link_name")
	Entry_Size_Invariants(size, "writer_stage_header.size")
	File_Mode_Invariants(mode, "writer_stage_header.mode")
	User_Identifier_Invariants(user_identifier, "writer_stage_header.user_identifier")
	Group_Identifier_Invariants(group_identifier, "writer_stage_header.group_identifier")
	Header_User_Name_Invariants(user_name, "writer_stage_header.user_name")
	Header_Group_Name_Invariants(group_name, "writer_stage_header.group_name")
	Timestamp_Invariants(modified, "writer_stage_header.modified")
	Access_Timestamp_Invariants(accessed, "writer_stage_header.accessed")
	Change_Timestamp_Invariants(changed, "writer_stage_header.changed")
	Device_Major_Invariants(device_major, "writer_stage_header.device_major")
	Device_Minor_Invariants(device_minor, "writer_stage_header.device_minor")
	selected, header_size, size_status := writer_header_preflight(
		format, name, link_name, size, mode, user_identifier, group_identifier,
		user_name, group_name, modified, accessed, changed, device_major, device_minor,
	)
	if size_status != Header_Encoding_Status(STATUS_OK) {
		return Header_Stage_Status(size_status)
	}
	if !writer_header_destination(completion, header_size) {
		return Header_Stage_Status(STATUS_OUTPUT_TOO_SMALL)
	}
	modified_seconds := Basic_Modification_Seconds(COUNT_MINIMUM)
	if modified.Set {
		modified_seconds = Basic_Modification_Seconds(modified.Seconds)
	}
	switch Format(selected) {
	case FORMAT_V7, FORMAT_USTAR:
		writer_stage_basic(
			completion, Basic_Fit_Format(selected), type_flag, USTAR_Name(name),
			Basic_Link_Name(link_name), size, Basic_File_Mode(mode),
			Basic_User_Identifier(user_identifier),
			Basic_Group_Identifier(group_identifier),
			USTAR_User_Name(user_name), USTAR_Group_Name(group_name),
			modified_seconds, device_major, device_minor,
		)
	case FORMAT_PAX:
		payload_size, _ := pax_payload_size(
			name, link_name, size, mode, user_identifier, group_identifier,
			user_name, group_name, modified, accessed, changed,
			device_major, device_minor,
		)
		writer_stage_pax(completion, writer_pax_header(
			type_flag, PAX_Header_Name(name), PAX_Header_Link_Name(link_name),
			size, Basic_File_Mode(mode), user_identifier, group_identifier,
			PAX_Header_User_Name(user_name), PAX_Header_Group_Name(group_name),
			modified, accessed, changed, USTAR_Device_Major(device_major),
			USTAR_Device_Minor(device_minor),
		), PAX_Payload_Size(payload_size))
	case FORMAT_GNU:
		writer_stage_gnu(completion, writer_gnu_header(
			type_flag, GNU_Name(name), GNU_Link_Name(link_name),
			Wire_File_Mode(mode), Wire_User_Identifier(user_identifier),
			Wire_Group_Identifier(group_identifier), size,
			GNU_Modification_Time{Seconds: modified.Seconds, Set: modified.Set},
			Wire_User_Name(user_name), Wire_Group_Name(group_name),
			Wire_Device_Major(device_major), Wire_Device_Minor(device_minor),
			Wire_Access_Time{Seconds: accessed.Seconds, Set: accessed.Set},
			Wire_Change_Time{Seconds: changed.Seconds, Set: changed.Set},
		))
	}
	writer_stage_content(completion, type_flag, size)
	return Header_Stage_Status(STATUS_OK)
}

func writer_header_destination(
	completion *nbio.Completion, header_size Encoded_Header_Size,
) (valid bytes.Boolean) {
	defer func() { bytes.Boolean_Invariants(valid, "writer_header_destination.valid") }()
	Encoded_Header_Size_Invariants(header_size, "writer_header_destination.header_size")
	writer := (*Writer)(unsafe.Pointer(completion))
	required := Header_Required_Size(int(writer.Padding) + int(header_size))
	if !writer_header_capacity_valid(completion, required) {
		return false
	}
	writer.Position = Destination_Position(writer.Padding)
	zero(Zero_Destination(writer.Destination[:writer.Position]))
	return true
}

func writer_header_preflight(
	format Format, name Writer_Name, link_name Header_Link_Name,
	size Entry_Size, mode File_Mode,
	user_identifier User_Identifier, group_identifier Group_Identifier,
	user_name Header_User_Name, group_name Header_Group_Name,
	modified Timestamp, accessed Access_Timestamp, changed Change_Timestamp,
	device_major Device_Major, device_minor Device_Minor,
) (
	selected Writer_Selected_Format, size_value Encoded_Header_Size,
	status Header_Encoding_Status,
) {
	defer func() {
		Writer_Selected_Format_Invariants(selected, "writer_header_preflight.selected")
		Encoded_Header_Size_Invariants(size_value, "writer_header_preflight.size_value")
		Header_Encoding_Status_Invariants(status, "writer_header_preflight.status")
	}()
	Format_Invariants(format, "writer_header_preflight.format")
	Writer_Name_Invariants(name, "writer_header_preflight.name")
	Header_Link_Name_Invariants(link_name, "writer_header_preflight.link_name")
	Entry_Size_Invariants(size, "writer_header_preflight.size")
	File_Mode_Invariants(mode, "writer_header_preflight.mode")
	User_Identifier_Invariants(user_identifier, "writer_header_preflight.user_identifier")
	Group_Identifier_Invariants(group_identifier, "writer_header_preflight.group_identifier")
	Header_User_Name_Invariants(user_name, "writer_header_preflight.user_name")
	Header_Group_Name_Invariants(group_name, "writer_header_preflight.group_name")
	Timestamp_Invariants(modified, "writer_header_preflight.modified")
	Access_Timestamp_Invariants(accessed, "writer_header_preflight.accessed")
	Change_Timestamp_Invariants(changed, "writer_header_preflight.changed")
	Device_Major_Invariants(device_major, "writer_header_preflight.device_major")
	Device_Minor_Invariants(device_minor, "writer_header_preflight.device_minor")
	if format == FORMAT_UNKNOWN {
		format = FORMAT_PAX
		if header_fits_basic(
			name, link_name, size, mode, user_identifier, group_identifier,
			user_name, group_name, modified, accessed, changed,
			device_major, device_minor, Basic_Fit_Format(FORMAT_USTAR),
		) {
			format = FORMAT_USTAR
		}
	}
	size_value, status = encoded_header_size(
		Encoding_Format(format), name, link_name, size, mode,
		user_identifier, group_identifier,
		user_name, group_name, modified, accessed, changed,
		device_major, device_minor,
	)
	if status != Header_Encoding_Status(STATUS_OK) {
		return Writer_Selected_Format(FORMAT_V7), size_value, status
	}
	return Writer_Selected_Format(format), size_value, status
}

func writer_stage_header_records(
	completion *nbio.Completion, format Format, type_flag Type_Flag,
	name Writer_Name, link_name Header_Link_Name, size Entry_Size, mode File_Mode,
	user_identifier User_Identifier, group_identifier Group_Identifier,
	user_name Header_User_Name, group_name Header_Group_Name,
	modified Timestamp, accessed Access_Timestamp, changed Change_Timestamp,
	device_major Device_Major, device_minor Device_Minor,
	records PAX_Header_Records,
) (status PAX_Records_Stage_Status) {
	defer func() {
		PAX_Records_Stage_Status_Invariants(
			status, "writer_stage_header_records.status",
		)
	}()
	Format_Invariants(format, "writer_stage_header_records.format")
	Type_Flag_Invariants(type_flag, "writer_stage_header_records.type_flag")
	Writer_Name_Invariants(name, "writer_stage_header_records.name")
	Header_Link_Name_Invariants(link_name, "writer_stage_header_records.link_name")
	Entry_Size_Invariants(size, "writer_stage_header_records.size")
	File_Mode_Invariants(mode, "writer_stage_header_records.mode")
	User_Identifier_Invariants(
		user_identifier, "writer_stage_header_records.user_identifier",
	)
	Group_Identifier_Invariants(
		group_identifier, "writer_stage_header_records.group_identifier",
	)
	Header_User_Name_Invariants(user_name, "writer_stage_header_records.user_name")
	Header_Group_Name_Invariants(group_name, "writer_stage_header_records.group_name")
	Timestamp_Invariants(modified, "writer_stage_header_records.modified")
	Access_Timestamp_Invariants(accessed, "writer_stage_header_records.accessed")
	Change_Timestamp_Invariants(changed, "writer_stage_header_records.changed")
	Device_Major_Invariants(device_major, "writer_stage_header_records.device_major")
	Device_Minor_Invariants(device_minor, "writer_stage_header_records.device_minor")
	PAX_Header_Records_Invariants(records, "writer_stage_header_records.records")
	if format != FORMAT_UNKNOWN {
		if format != FORMAT_PAX {
			return PAX_Records_Stage_Status(STATUS_FIELD_TOO_LONG)
		}
	}
	payload_size, valid := pax_payload_size(
		name, link_name, size, mode, user_identifier, group_identifier,
		user_name, group_name, modified, accessed, changed,
		device_major, device_minor,
	)
	if !valid {
		return PAX_Records_Stage_Status(STATUS_FIELD_TOO_LONG)
	}
	payload_size += PAX_Payload_Size_Candidate(len(records))
	if payload_size > SPECIAL_FILE_SIZE_MAXIMUM {
		return PAX_Records_Stage_Status(STATUS_FIELD_TOO_LONG)
	}
	blocks := padded_block_count(Metadata_Payload_Size(payload_size))
	header_size := Encoded_Header_Size(
		(HEADER_BLOCK_COUNT + int(blocks) + HEADER_BLOCK_COUNT) * BLOCK_SIZE,
	)
	writer := (*Writer)(unsafe.Pointer(completion))
	required := Header_Required_Size(int(writer.Padding) + int(header_size))
	if !writer_header_capacity_valid(completion, required) {
		return PAX_Records_Stage_Status(STATUS_OUTPUT_TOO_SMALL)
	}
	writer.Position = Destination_Position(writer.Padding)
	zero(Zero_Destination(writer.Destination[:writer.Position]))
	pax_header := writer_pax_header(
		type_flag, PAX_Header_Name(name), PAX_Header_Link_Name(link_name), size,
		Basic_File_Mode(mode), user_identifier, group_identifier,
		PAX_Header_User_Name(user_name), PAX_Header_Group_Name(group_name),
		modified, accessed, changed,
		USTAR_Device_Major(device_major), USTAR_Device_Minor(device_minor),
	)
	writer_stage_pax_records(
		completion, pax_header, PAX_Records_Payload_Size(payload_size), records,
	)
	writer_stage_content(completion, type_flag, size)
	return PAX_Records_Stage_Status(STATUS_OK)
}

func writer_stage_content(
	completion *nbio.Completion, type_flag Type_Flag, size Entry_Size,
) {
	Type_Flag_Invariants(type_flag, "writer_stage_content.type_flag")
	Entry_Size_Invariants(size, "writer_stage_content.size")
	writer := (*Writer)(unsafe.Pointer(completion))
	writer.Pending_Content = Pending_Content_Count(size)
	if header_only_type(type_flag) {
		writer.Pending_Content = 0
	}
	writer.Pending_Padding = Pending_Padding_Count(
		block_padding(Unpadded_Size(writer.Pending_Content)),
	)
}

func writer_basic_header(
	format Basic_Format, type_flag Type_Flag, name USTAR_Name,
	link_name Basic_Link_Name, size Entry_Size, mode Basic_File_Mode,
	user_identifier Basic_User_Identifier,
	group_identifier Basic_Group_Identifier,
	user_name USTAR_User_Name, group_name USTAR_Group_Name,
	modified_seconds Basic_Modification_Seconds,
	device_major Device_Major, device_minor Device_Minor,
) (header Basic_Header) {
	defer func() { Basic_Header_Invariants(header, "writer_basic_header.header") }()
	Basic_Format_Invariants(format, "writer_basic_header.format")
	Type_Flag_Invariants(type_flag, "writer_basic_header.type_flag")
	USTAR_Name_Invariants(name, "writer_basic_header.name")
	Basic_Link_Name_Invariants(link_name, "writer_basic_header.link_name")
	Entry_Size_Invariants(size, "writer_basic_header.size")
	Basic_File_Mode_Invariants(mode, "writer_basic_header.mode")
	Basic_User_Identifier_Invariants(
		user_identifier, "writer_basic_header.user_identifier",
	)
	Basic_Group_Identifier_Invariants(
		group_identifier, "writer_basic_header.group_identifier",
	)
	USTAR_User_Name_Invariants(user_name, "writer_basic_header.user_name")
	USTAR_Group_Name_Invariants(group_name, "writer_basic_header.group_name")
	Basic_Modification_Seconds_Invariants(
		modified_seconds, "writer_basic_header.modified_seconds",
	)
	Device_Major_Invariants(device_major, "writer_basic_header.device_major")
	Device_Minor_Invariants(device_minor, "writer_basic_header.device_minor")
	major := USTAR_Device_Major(COUNT_MINIMUM)
	minor := USTAR_Device_Minor(COUNT_MINIMUM)
	if format != Basic_Format(FORMAT_V7) {
		major = USTAR_Device_Major(device_major)
		minor = USTAR_Device_Minor(device_minor)
	}
	return Basic_Header{
		Format: format, Type_Flag: type_flag,
		Name: name, Link_Name: link_name,
		Mode:             mode,
		User_Identifier:  user_identifier,
		Group_Identifier: group_identifier, Size: size,
		Modification_Seconds: modified_seconds,
		User_Name:            user_name,
		Group_Name:           group_name,
		Device_Major:         major, Device_Minor: minor,
	}
}

func writer_pax_header(
	type_flag Type_Flag, name PAX_Header_Name, link_name PAX_Header_Link_Name,
	size Entry_Size, mode Basic_File_Mode,
	user_identifier User_Identifier, group_identifier Group_Identifier,
	user_name PAX_Header_User_Name, group_name PAX_Header_Group_Name,
	modified Timestamp, accessed Access_Timestamp, changed Change_Timestamp,
	device_major USTAR_Device_Major, device_minor USTAR_Device_Minor,
) (header PAX_Header) {
	defer func() { PAX_Header_Invariants(header, "writer_pax_header.header") }()
	Type_Flag_Invariants(type_flag, "writer_pax_header.type_flag")
	PAX_Header_Name_Invariants(name, "writer_pax_header.name")
	PAX_Header_Link_Name_Invariants(link_name, "writer_pax_header.link_name")
	Entry_Size_Invariants(size, "writer_pax_header.size")
	Basic_File_Mode_Invariants(mode, "writer_pax_header.mode")
	User_Identifier_Invariants(user_identifier, "writer_pax_header.user_identifier")
	Group_Identifier_Invariants(group_identifier, "writer_pax_header.group_identifier")
	PAX_Header_User_Name_Invariants(user_name, "writer_pax_header.user_name")
	PAX_Header_Group_Name_Invariants(group_name, "writer_pax_header.group_name")
	Timestamp_Invariants(modified, "writer_pax_header.modified")
	Access_Timestamp_Invariants(accessed, "writer_pax_header.accessed")
	Change_Timestamp_Invariants(changed, "writer_pax_header.changed")
	USTAR_Device_Major_Invariants(device_major, "writer_pax_header.device_major")
	USTAR_Device_Minor_Invariants(device_minor, "writer_pax_header.device_minor")
	return PAX_Header{
		Type_Flag: type_flag, Name: name,
		Link_Name: link_name, Size: size,
		Mode: mode, User_Identifier: user_identifier,
		Group_Identifier:  group_identifier,
		User_Name:         user_name,
		Group_Name:        group_name,
		Modification_Time: modified, Access_Time: accessed, Change_Time: changed,
		Device_Major: device_major, Device_Minor: device_minor,
	}
}

func writer_gnu_header(
	type_flag Type_Flag, name GNU_Name, link_name GNU_Link_Name,
	mode Wire_File_Mode, user_identifier Wire_User_Identifier,
	group_identifier Wire_Group_Identifier, size Entry_Size,
	modified GNU_Modification_Time,
	user_name Wire_User_Name, group_name Wire_Group_Name,
	device_major Wire_Device_Major, device_minor Wire_Device_Minor,
	accessed Wire_Access_Time, changed Wire_Change_Time,
) (header GNU_Header) {
	defer func() { GNU_Header_Invariants(header, "writer_gnu_header.header") }()
	Type_Flag_Invariants(type_flag, "writer_gnu_header.type_flag")
	GNU_Name_Invariants(name, "writer_gnu_header.name")
	GNU_Link_Name_Invariants(link_name, "writer_gnu_header.link_name")
	Entry_Size_Invariants(size, "writer_gnu_header.size")
	Wire_File_Mode_Invariants(mode, "writer_gnu_header.mode")
	Wire_User_Identifier_Invariants(user_identifier, "writer_gnu_header.user_identifier")
	Wire_Group_Identifier_Invariants(group_identifier, "writer_gnu_header.group_identifier")
	GNU_Modification_Time_Invariants(modified, "writer_gnu_header.modified")
	Wire_User_Name_Invariants(user_name, "writer_gnu_header.user_name")
	Wire_Group_Name_Invariants(group_name, "writer_gnu_header.group_name")
	Wire_Device_Major_Invariants(device_major, "writer_gnu_header.device_major")
	Wire_Device_Minor_Invariants(device_minor, "writer_gnu_header.device_minor")
	Wire_Access_Time_Invariants(accessed, "writer_gnu_header.accessed")
	Wire_Change_Time_Invariants(changed, "writer_gnu_header.changed")
	return GNU_Header{
		Type_Flag: type_flag, Name: name, Link_Name: link_name,
		Mode: mode, User_Identifier: user_identifier,
		Group_Identifier: group_identifier, Size: size,
		Modification_Time: modified,
		User_Name:         user_name, Group_Name: group_name,
		Device_Major: device_major, Device_Minor: device_minor,
		Access_Time: accessed, Change_Time: changed,
	}
}

func writer_stage_basic(
	completion *nbio.Completion, format Basic_Fit_Format,
	type_flag Type_Flag, name USTAR_Name, link_name Basic_Link_Name,
	size Entry_Size, mode Basic_File_Mode,
	user_identifier Basic_User_Identifier,
	group_identifier Basic_Group_Identifier,
	user_name USTAR_User_Name, group_name USTAR_Group_Name,
	modified_seconds Basic_Modification_Seconds,
	device_major Device_Major, device_minor Device_Minor,
) {
	Basic_Fit_Format_Invariants(format, "writer_stage_basic.format")
	Type_Flag_Invariants(type_flag, "writer_stage_basic.type_flag")
	USTAR_Name_Invariants(name, "writer_stage_basic.name")
	Basic_Link_Name_Invariants(link_name, "writer_stage_basic.link_name")
	Entry_Size_Invariants(size, "writer_stage_basic.size")
	Basic_File_Mode_Invariants(mode, "writer_stage_basic.mode")
	Basic_User_Identifier_Invariants(user_identifier, "writer_stage_basic.user_identifier")
	Basic_Group_Identifier_Invariants(group_identifier, "writer_stage_basic.group_identifier")
	USTAR_User_Name_Invariants(user_name, "writer_stage_basic.user_name")
	USTAR_Group_Name_Invariants(group_name, "writer_stage_basic.group_name")
	Basic_Modification_Seconds_Invariants(
		modified_seconds, "writer_stage_basic.modified_seconds",
	)
	Device_Major_Invariants(device_major, "writer_stage_basic.device_major")
	Device_Minor_Invariants(device_minor, "writer_stage_basic.device_minor")
	writer := (*Writer)(unsafe.Pointer(completion))
	block := writer.Destination[writer.Position : writer.Position+BLOCK_SIZE]
	zero(Zero_Destination(block))
	write_basic_header(Header_Block(block), writer_basic_header(
		Basic_Format(format), type_flag, name, link_name, size, mode,
		user_identifier, group_identifier, user_name, group_name,
		modified_seconds, device_major, device_minor,
	))
	writer.Position += BLOCK_SIZE
}

func writer_stage_pax(
	completion *nbio.Completion, header PAX_Header, payload_size PAX_Payload_Size,
) {
	PAX_Header_Invariants(header, "writer_stage_pax.header")
	PAX_Payload_Size_Invariants(payload_size, "writer_stage_pax.payload_size")
	writer := (*Writer)(unsafe.Pointer(completion))
	writer.Position = Destination_Position(write_pax_headers(
		PAX_Header_Destination(writer.Destination),
		Header_Start_Position(writer.Position), header, payload_size,
	))
}

func writer_stage_pax_records(
	completion *nbio.Completion, header PAX_Header,
	payload_size PAX_Records_Payload_Size, records PAX_Header_Records,
) {
	PAX_Header_Invariants(header, "writer_stage_pax_records.header")
	PAX_Records_Payload_Size_Invariants(
		payload_size, "writer_stage_pax_records.payload_size",
	)
	PAX_Header_Records_Invariants(records, "writer_stage_pax_records.records")
	writer := (*Writer)(unsafe.Pointer(completion))
	writer.Position = Destination_Position(write_pax_headers_records(
		PAX_Header_Destination(writer.Destination),
		Header_Start_Position(writer.Position), header, payload_size, records,
	))
}

func writer_stage_gnu(completion *nbio.Completion, header GNU_Header) {
	GNU_Header_Invariants(header, "writer_stage_gnu.header")
	writer := (*Writer)(unsafe.Pointer(completion))
	writer.Position = Destination_Position(write_gnu_headers(
		GNU_Header_Destination(writer.Destination),
		Header_Start_Position(writer.Position), header,
	))
}

func writer_header_state_valid(completion *nbio.Completion) (valid bytes.Boolean) {
	defer func() {
		bytes.Boolean_Invariants(valid, "writer_header_state_valid.valid")
	}()
	writer := (*Writer)(unsafe.Pointer(completion))
	if !writer.Initialized {
		writer.Status = STATUS_STORAGE_INVALID
		return false
	}
	if writer.Closed {
		writer.Status = STATUS_CLOSED
		return false
	}
	if writer.Content_Count != 0 {
		writer.Status = STATUS_CONTENT_INCOMPLETE
		return false
	}
	return true
}

func writer_header_capacity_valid(
	completion *nbio.Completion, required Header_Required_Size,
) (valid bytes.Boolean) {
	defer func() {
		bytes.Boolean_Invariants(valid, "writer_header_capacity_valid.valid")
	}()
	Header_Required_Size_Invariants(required, "writer_header_capacity_valid.required")
	writer := (*Writer)(unsafe.Pointer(completion))
	if int(required) > len(writer.Destination) {
		return false
	}
	return int(writer.Archive_Count)+int(required) <= ARCHIVE_SIZE_MAXIMUM
}

// Writer_Write submits current entry content without exceeding declared Size.
func Writer_Write(
	writer *Writer, completion *nbio.Completion, source Source,
	callback nbio.Callback,
) {
	Writer_Invariants(writer, "Writer_Write.writer")
	Source_Invariants(source, "Writer_Write.source")
	aver.Always(completion != nil, "Writer_Write has a completion.")
	aver.Always(
		completion == &writer.Completion,
		"Writer_Write submits the completion owned by its Writer.",
	)
	aver.Always(callback != nil, "Writer_Write has a callback.")
	aver.Always(!writer.Active, "Writer_Write owns a free callback slot.")
	writer.Active = true
	writer.Callback = callback
	writer.Operation = WRITER_OPERATION_CONTENT
	writer.Status = STATUS_OK
	writer.Count = 0
	writer.Source = nil
	writer.Position = 0
	completion.Data = 0
	completion.Error = nil
	if !writer.Initialized {
		writer.Status = STATUS_STORAGE_INVALID
		writer_finish(completion)
		return
	}
	if writer.Closed {
		writer.Status = STATUS_CLOSED
		writer_finish(completion)
		return
	}
	if Content_Count(len(source)) > writer.Content_Count {
		writer.Status = STATUS_WRITE_TOO_LONG
		writer_finish(completion)
		return
	}
	if Count(writer.Archive_Count)+Count(len(source)) > ARCHIVE_SIZE_MAXIMUM {
		writer.Status = STATUS_OUTPUT_TOO_SMALL
		writer_finish(completion)
		return
	}
	if len(source) == 0 {
		writer_finish(completion)
		return
	}
	writer.Source = source
	writer_submit_progress(completion)
}

// Writer_Close submits final padding and two zero records.
func Writer_Close(
	writer *Writer, completion *nbio.Completion, callback nbio.Callback,
) {
	Writer_Invariants(writer, "Writer_Close.writer")
	aver.Always(completion != nil, "Writer_Close has a completion.")
	aver.Always(
		completion == &writer.Completion,
		"Writer_Close submits the completion owned by its Writer.",
	)
	aver.Always(callback != nil, "Writer_Close has a callback.")
	aver.Always(!writer.Active, "Writer_Close owns a free callback slot.")
	writer.Active = true
	writer.Callback = callback
	writer.Operation = WRITER_OPERATION_CLOSE
	writer.Status = STATUS_OK
	writer.Count = 0
	writer.Source = nil
	writer.Position = 0
	completion.Data = 0
	completion.Error = nil
	if !writer.Initialized {
		writer.Status = STATUS_STORAGE_INVALID
		writer_finish(completion)
		return
	}
	if writer.Closed {
		writer.Status = STATUS_CLOSED
		writer.Count = Count(writer.Archive_Count)
		writer_finish(completion)
		return
	}
	if writer.Content_Count != 0 {
		writer.Status = STATUS_CONTENT_INCOMPLETE
		writer.Count = Count(writer.Archive_Count)
		writer_finish(completion)
		return
	}
	required := Count(writer.Padding) + ARCHIVE_FOOTER_SIZE
	if required > Count(len(writer.Destination)) {
		writer.Status = STATUS_OUTPUT_TOO_SMALL
		writer.Count = Count(writer.Archive_Count)
		writer_finish(completion)
		return
	}
	if Count(writer.Archive_Count)+required > ARCHIVE_SIZE_MAXIMUM {
		writer.Status = STATUS_OUTPUT_TOO_SMALL
		writer.Count = Count(writer.Archive_Count)
		writer_finish(completion)
		return
	}
	zero(Zero_Destination(writer.Destination[:required]))
	writer.Position = Destination_Position(required)
	writer.Source = Source(writer.Destination[:required])
	writer_submit_progress(completion)
}

// Writer submission needs a trampoline for partial inline Stream writes.
func writer_submit_progress(completion *nbio.Completion) {
	writer := (*Writer)(unsafe.Pointer(completion))
	for bool(writer.Active) && !bool(writer.Wait_Active) {
		if int(writer.Count) == len(writer.Source) {
			writer_transfer_complete(completion)
			return
		}
		writer.Wait_Active = true
		writer.Submission_Active = true
		writer.Continue = false
		buffer := writer.Source[writer.Count:]
		nbio.Write(writer.Stream, completion, buffer, writer_stream_complete)
		writer.Submission_Active = false
		if !writer.Continue {
			return
		}
		writer.Continue = false
	}
}

func writer_stream_complete(completion *nbio.Completion) {
	writer := (*Writer)(unsafe.Pointer(completion))
	aver.Always(writer.Wait_Active, "Writer callback retires one submitted transfer.")
	writer.Wait_Active = false
	requested := len(writer.Source) - int(writer.Count)
	count := completion.Data
	if count < 0 {
		count = 0
		completion.Error = nbio.Stream_Negative_Write
	}
	if count > requested {
		count = 0
		completion.Error = nbio.Stream_Invalid_Write
	}
	writer.Count += Count(count)
	completion.Data = count
	if completion.Error != nil {
		writer.Status = STATUS_TRANSPORT_FAILED
		writer_transfer_failed(completion)
	} else if count == 0 {
		if completion.Error == nil {
			completion.Error = nbio.Stream_No_Progress
		}
		writer.Status = STATUS_TRANSPORT_FAILED
		writer_transfer_failed(completion)
	}
	if !writer.Active {
		return
	}
	if writer.Submission_Active {
		writer.Continue = true
		return
	}
	writer_submit_progress(completion)
}

func writer_transfer_failed(completion *nbio.Completion) {
	writer := (*Writer)(unsafe.Pointer(completion))
	writer.Archive_Count += Archive_Count(writer.Count)
	if writer.Operation == WRITER_OPERATION_CONTENT {
		writer.Content_Count -= Content_Count(writer.Count)
	} else {
		writer.Closed = true
	}
	writer_finish(completion)
}

func writer_transfer_complete(completion *nbio.Completion) {
	writer := (*Writer)(unsafe.Pointer(completion))
	writer.Archive_Count += Archive_Count(writer.Count)
	switch writer.Operation {
	case WRITER_OPERATION_HEADER:
		writer.Content_Count = Content_Count(writer.Pending_Content)
		writer.Padding = Padding_Count(writer.Pending_Padding)
	case WRITER_OPERATION_CONTENT:
		writer.Content_Count -= Content_Count(writer.Count)
	case WRITER_OPERATION_CLOSE:
		writer.Padding = 0
		writer.Closed = true
		writer.Count = Count(writer.Archive_Count)
	default:
		writer.Status = STATUS_STORAGE_INVALID
	}
	writer_finish(completion)
}

func writer_finish(completion *nbio.Completion) {
	writer := (*Writer)(unsafe.Pointer(completion))
	callback := writer.Callback
	completion.Data = int(writer.Count)
	writer.Callback = nil
	writer.Source = nil
	writer.Pending_Content = 0
	writer.Pending_Padding = 0
	writer.Position = 0
	writer.Operation = WRITER_OPERATION_NONE
	writer.Active = false
	writer.Wait_Active = false
	writer.Submission_Active = false
	writer.Continue = false
	callback(completion)
}

func parse_header(
	block_storage Reader_Header_Block,
) (header Wire_Header, status Parse_Status) {
	defer func() {
		Wire_Header_Invariants(header, "parse_header.header")
		Parse_Status_Invariants(status, "parse_header.status")
	}()
	Reader_Header_Block_Invariants(block_storage, "parse_header.block_storage")
	fallback := Wire_Header{
		Basic: Wire_Header_Basic{Type_Flag: Wire_Type_Flag(TYPE_REGULAR)},
		Extensions: Wire_Header_Extensions{
			Format: Reader_Format(FORMAT_V7),
		},
	}
	block := Header_Block(block_storage)
	if zero_block(block) {
		return fallback, Parse_Status(STATUS_END)
	}
	format, valid := block_format(block)
	if !valid {
		return fallback, Parse_Status(STATUS_INPUT_INVALID)
	}
	reader_format := Reader_Format(format)
	basic, basic_status := parse_header_basic(block)
	if basic_status != Basic_Parse_Status(STATUS_OK) {
		return fallback, Parse_Status(basic_status)
	}
	fixed_names, names_status := parse_header_names(block, reader_format)
	if names_status != Input_Status(STATUS_OK) {
		return fallback, Parse_Status(names_status)
	}
	extensions, name, extension_status := parse_header_extensions(
		block, reader_format, fixed_names.Name,
	)
	if extension_status != Input_Status(STATUS_OK) {
		return fallback, Parse_Status(extension_status)
	}
	names := Wire_Header_Names{
		Name: name, Link_Name: fixed_names.Link_Name,
		User_Name: fixed_names.User_Name, Group_Name: fixed_names.Group_Name,
		Device_Major: fixed_names.Device_Major,
		Device_Minor: fixed_names.Device_Minor,
	}
	if header_only_type(Type_Flag(basic.Type_Flag)) {
		basic.Physical_Size = 0
	}
	return Wire_Header{
		Basic: basic, Names: names, Extensions: extensions,
	}, Parse_Status(STATUS_OK)
}

func parse_header_basic(
	block Header_Block,
) (basic Wire_Header_Basic, status Basic_Parse_Status) {
	defer func() {
		Wire_Header_Basic_Invariants(basic, "parse_header_basic.basic")
		Basic_Parse_Status_Invariants(status, "parse_header_basic.status")
	}()
	Header_Block_Invariants(block, "parse_header_basic.block")
	fallback := Wire_Header_Basic{Type_Flag: Wire_Type_Flag(TYPE_REGULAR)}
	type_flag := Type_Flag(block[TYPE_FLAG_FIELD_OFFSET])
	if type_flag == TYPE_REGULAR_LEGACY {
		type_flag = TYPE_REGULAR
	}
	mode, valid := parse_numeric(Numeric_Field(
		block[MODE_FIELD_OFFSET:][:MODE_FIELD_SIZE],
	))
	if !valid {
		return fallback, Basic_Parse_Status(STATUS_INPUT_INVALID)
	}
	user_identifier, valid := parse_numeric(Numeric_Field(
		block[USER_IDENTIFIER_FIELD_OFFSET:][:USER_IDENTIFIER_FIELD_SIZE],
	))
	if !valid {
		return fallback, Basic_Parse_Status(STATUS_INPUT_INVALID)
	}
	group_identifier, valid := parse_numeric(Numeric_Field(
		block[GROUP_IDENTIFIER_FIELD_OFFSET:][:GROUP_IDENTIFIER_FIELD_SIZE],
	))
	if !valid {
		return fallback, Basic_Parse_Status(STATUS_INPUT_INVALID)
	}
	entry_size, valid := parse_numeric(Numeric_Field(
		block[ENTRY_SIZE_FIELD_OFFSET:][:ENTRY_SIZE_FIELD_SIZE],
	))
	if !valid {
		return fallback, Basic_Parse_Status(STATUS_INPUT_INVALID)
	}
	if entry_size < 0 {
		return fallback, Basic_Parse_Status(STATUS_INPUT_INVALID)
	}
	if entry_size > ARCHIVE_SIZE_MAXIMUM {
		return fallback, Basic_Parse_Status(STATUS_FIELD_TOO_LONG)
	}
	modified, valid := parse_numeric(Numeric_Field(
		block[TIMESTAMP_FIELD_OFFSET:][:TIMESTAMP_FIELD_SIZE],
	))
	if !valid {
		return fallback, Basic_Parse_Status(STATUS_INPUT_INVALID)
	}
	return Wire_Header_Basic{
		Type_Flag: Wire_Type_Flag(type_flag), Size: Entry_Size(entry_size),
		Physical_Size: Physical_Size(entry_size), Mode: Wire_File_Mode(mode),
		User_Identifier:  Wire_User_Identifier(user_identifier),
		Group_Identifier: Wire_Group_Identifier(group_identifier),
		Modified_Seconds: Wire_Modification_Seconds(modified),
	}, Basic_Parse_Status(STATUS_OK)
}

func parse_header_names(
	block Header_Block, format Reader_Format,
) (names Wire_Header_Fixed_Names, status Input_Status) {
	defer func() {
		Wire_Header_Fixed_Names_Invariants(names, "parse_header_names.names")
		Input_Status_Invariants(status, "parse_header_names.status")
	}()
	Header_Block_Invariants(block, "parse_header_names.block")
	Reader_Format_Invariants(format, "parse_header_names.format")
	names.Name = Wire_Name_Suffix(wire_field_bytes(
		Wire_Text_Field(block[NAME_FIELD_OFFSET:][:NAME_FIELD_SIZE]),
	))
	names.Link_Name = Wire_Link_Name(wire_field_bytes(
		Wire_Text_Field(block[LINK_NAME_FIELD_OFFSET:][:LINK_NAME_FIELD_SIZE]),
	))
	if format == Reader_Format(FORMAT_V7) {
		return names, Input_Status(STATUS_OK)
	}
	names.User_Name = Wire_User_Name(wire_field_bytes(
		Wire_Text_Field(block[USER_NAME_FIELD_OFFSET:][:USER_NAME_FIELD_SIZE]),
	))
	names.Group_Name = Wire_Group_Name(wire_field_bytes(
		Wire_Text_Field(block[GROUP_NAME_FIELD_OFFSET:][:GROUP_NAME_FIELD_SIZE]),
	))
	device_major, valid := parse_numeric(Numeric_Field(
		block[DEVICE_MAJOR_FIELD_OFFSET:][:DEVICE_MAJOR_FIELD_SIZE],
	))
	if !valid {
		return Wire_Header_Fixed_Names{}, Input_Status(STATUS_INPUT_INVALID)
	}
	device_minor, valid := parse_numeric(Numeric_Field(
		block[DEVICE_MINOR_FIELD_OFFSET:][:DEVICE_MINOR_FIELD_SIZE],
	))
	if !valid {
		return Wire_Header_Fixed_Names{}, Input_Status(STATUS_INPUT_INVALID)
	}
	names.Device_Major = Wire_Device_Major(device_major)
	names.Device_Minor = Wire_Device_Minor(device_minor)
	return names, Input_Status(STATUS_OK)
}

func parse_header_extensions(
	block Header_Block, format Reader_Format, name Wire_Name_Suffix,
) (
	extensions Wire_Header_Extensions,
	resolved Wire_Name,
	status Input_Status,
) {
	defer func() {
		Wire_Header_Extensions_Invariants(
			extensions, "parse_header_extensions.extensions",
		)
		Wire_Name_Invariants(resolved, "parse_header_extensions.resolved")
		Input_Status_Invariants(status, "parse_header_extensions.status")
	}()
	Header_Block_Invariants(block, "parse_header_extensions.block")
	Reader_Format_Invariants(format, "parse_header_extensions.format")
	Wire_Name_Suffix_Invariants(name, "parse_header_extensions.name")
	extensions.Format = format
	resolved.Suffix = name
	switch format {
	case Reader_Format(FORMAT_USTAR), Reader_Format(FORMAT_PAX):
		resolved = parse_header_prefix(
			Wire_Prefix_Field(block[PREFIX_FIELD_OFFSET:][:PREFIX_FIELD_SIZE]), name,
		)
	case Reader_Format(FORMAT_STAR):
		accessed, changed, star_name, star_status := parse_header_star(block, name)
		extensions.Access_Time = Wire_Access_Time{Seconds: accessed, Set: true}
		extensions.Change_Time = Wire_Change_Time{Seconds: changed, Set: true}
		return extensions, Wire_Name{
			Prefix: Wire_Name_Prefix(star_name.Prefix),
			Suffix: Wire_Name_Suffix(star_name.Suffix), Joined: star_name.Joined,
		}, star_status
	case Reader_Format(FORMAT_GNU):
		accessed, changed, gnu_name, gnu_status := parse_header_gnu(block, name)
		extensions.Access_Time = accessed
		extensions.Change_Time = changed
		return extensions, gnu_name, gnu_status
	}
	return extensions, resolved, Input_Status(STATUS_OK)
}

func parse_header_prefix(
	field Wire_Prefix_Field, name Wire_Name_Suffix,
) (resolved Wire_Name) {
	defer func() { Wire_Name_Invariants(resolved, "parse_header_prefix.resolved") }()
	Wire_Prefix_Field_Invariants(field, "parse_header_prefix.field")
	Wire_Name_Suffix_Invariants(name, "parse_header_prefix.name")
	resolved.Suffix = name
	prefix := wire_field_bytes(Wire_Text_Field(field))
	if len(prefix) != 0 {
		resolved = Wire_Name{
			Prefix: Wire_Name_Prefix(prefix), Suffix: name, Joined: true,
		}
	}
	return resolved
}

func parse_header_star(
	block Header_Block, name Wire_Name_Suffix,
) (
	access_time Access_Seconds,
	change_time Change_Seconds,
	resolved STAR_Wire_Name,
	status Input_Status,
) {
	defer func() {
		Access_Seconds_Invariants(access_time, "parse_header_star.access_time")
		Change_Seconds_Invariants(change_time, "parse_header_star.change_time")
		STAR_Wire_Name_Invariants(resolved, "parse_header_star.resolved")
		Input_Status_Invariants(status, "parse_header_star.status")
	}()
	Header_Block_Invariants(block, "parse_header_star.block")
	Wire_Name_Suffix_Invariants(name, "parse_header_star.name")
	wire_name := parse_header_prefix(
		Wire_Prefix_Field(
			block[PREFIX_FIELD_OFFSET:][:STAR_PREFIX_FIELD_SIZE],
		),
		name,
	)
	resolved = STAR_Wire_Name{
		Prefix: STAR_Wire_Name_Prefix(wire_name.Prefix),
		Suffix: STAR_Wire_Name_Suffix(wire_name.Suffix), Joined: wire_name.Joined,
	}
	accessed, access_valid := parse_numeric(Numeric_Field(
		block[STAR_ACCESS_TIMESTAMP_FIELD_OFFSET:][:TIMESTAMP_FIELD_SIZE],
	))
	changed, change_valid := parse_numeric(Numeric_Field(
		block[STAR_CHANGE_TIMESTAMP_FIELD_OFFSET:][:TIMESTAMP_FIELD_SIZE],
	))
	if !access_valid {
		return 0, 0, STAR_Wire_Name{}, Input_Status(STATUS_INPUT_INVALID)
	}
	if !change_valid {
		return 0, 0, STAR_Wire_Name{}, Input_Status(STATUS_INPUT_INVALID)
	}
	return Access_Seconds(accessed), Change_Seconds(changed), resolved,
		Input_Status(STATUS_OK)
}

func parse_header_gnu(
	block Header_Block, name Wire_Name_Suffix,
) (
	access_time Wire_Access_Time,
	change_time Wire_Change_Time,
	resolved Wire_Name,
	status Input_Status,
) {
	defer func() {
		Wire_Access_Time_Invariants(access_time, "parse_header_gnu.access_time")
		Wire_Change_Time_Invariants(change_time, "parse_header_gnu.change_time")
		Wire_Name_Invariants(resolved, "parse_header_gnu.resolved")
		Input_Status_Invariants(status, "parse_header_gnu.status")
	}()
	Header_Block_Invariants(block, "parse_header_gnu.block")
	Wire_Name_Suffix_Invariants(name, "parse_header_gnu.name")
	resolved.Suffix = name
	accessed, access_valid := parse_optional_numeric(Optional_Numeric_Field(
		block[GNU_ACCESS_TIMESTAMP_FIELD_OFFSET:][:TIMESTAMP_FIELD_SIZE],
	))
	changed, change_valid := parse_optional_numeric(Optional_Numeric_Field(
		block[GNU_CHANGE_TIMESTAMP_FIELD_OFFSET:][:TIMESTAMP_FIELD_SIZE],
	))
	if !access_valid {
		resolved, status = parse_header_gnu_prefix(block, name)
		return Wire_Access_Time{}, Wire_Change_Time{}, resolved, status
	}
	if !change_valid {
		resolved, status = parse_header_gnu_prefix(block, name)
		return Wire_Access_Time{}, Wire_Change_Time{}, resolved, status
	}
	access_time = Wire_Access_Time{
		Seconds: Access_Seconds(accessed.Value),
		Set:     Access_Time_Set(accessed.Set),
	}
	change_time = Wire_Change_Time{
		Seconds: Change_Seconds(changed.Value),
		Set:     Change_Time_Set(changed.Set),
	}
	return access_time, change_time, resolved, Input_Status(STATUS_OK)
}

func parse_header_gnu_prefix(
	block Header_Block, name Wire_Name_Suffix,
) (resolved Wire_Name, status Input_Status) {
	defer func() {
		Wire_Name_Invariants(resolved, "parse_header_gnu_prefix.resolved")
		Input_Status_Invariants(status, "parse_header_gnu_prefix.status")
	}()
	Header_Block_Invariants(block, "parse_header_gnu_prefix.block")
	Wire_Name_Suffix_Invariants(name, "parse_header_gnu_prefix.name")
	resolved.Suffix = name
	prefix := wire_field_bytes(
		Wire_Text_Field(block[PREFIX_FIELD_OFFSET:][:PREFIX_FIELD_SIZE]),
	)
	if !ascii_bytes(GNU_Prefix_Text(prefix)) {
		return Wire_Name{}, Input_Status(STATUS_INPUT_INVALID)
	}
	if len(prefix) != 0 {
		resolved = Wire_Name{
			Prefix: Wire_Name_Prefix(prefix), Suffix: name, Joined: true,
		}
	}
	return resolved, Input_Status(STATUS_OK)
}

func block_format(block Header_Block) (format Format, valid bytes.Boolean) {
	defer func() {
		Format_Invariants(format, "block_format.format")
		bytes.Boolean_Invariants(valid, "block_format.valid")
	}()
	Header_Block_Invariants(block, "block_format.block")
	wanted, numeric_valid := parse_octal(
		Numeric_Field(
			block[CHECKSUM_FIELD_OFFSET : CHECKSUM_FIELD_OFFSET+CHECKSUM_FIELD_SIZE],
		),
	)
	if !numeric_valid {
		return FORMAT_UNKNOWN, false
	}
	unsigned, signed := header_checksum(block)
	if Integer(wanted) != Integer(unsigned) {
		if Integer(wanted) != Integer(signed) {
			return FORMAT_UNKNOWN, false
		}
	}
	if wire_literal_equal(
		block, Wire_Literal_Offset(MAGIC_FIELD_OFFSET),
		Wire_Literal(USTAR_MAGIC),
	) {
		if wire_literal_equal(
			block, Wire_Literal_Offset(STAR_TRAILER_FIELD_OFFSET),
			Wire_Literal(STAR_TRAILER),
		) {
			return FORMAT_STAR, true
		}
		return FORMAT_USTAR, true
	}
	if wire_literal_equal(
		block, Wire_Literal_Offset(MAGIC_FIELD_OFFSET),
		Wire_Literal(GNU_MAGIC),
	) {
		if wire_literal_equal(
			block, Wire_Literal_Offset(VERSION_FIELD_OFFSET),
			Wire_Literal(GNU_VERSION),
		) {
			return FORMAT_GNU, true
		}
	}
	return FORMAT_V7, true
}

func parse_numeric(field Numeric_Field) (value Integer, valid bytes.Boolean) {
	defer func() {
		Integer_Invariants(value, "parse_numeric.value")
		bytes.Boolean_Invariants(valid, "parse_numeric.valid")
	}()
	Numeric_Field_Invariants(field, "parse_numeric.field")
	if len(field) == 0 {
		return 0, false
	}
	if field[0]&BASE_256_MARKER_MASK != 0 {
		return parse_base_256(field)
	}
	octal, valid := parse_octal(field)
	return Integer(octal), valid
}

func parse_optional_numeric(
	field Optional_Numeric_Field,
) (value Optional_Integer, valid bytes.Boolean) {
	defer func() {
		Optional_Integer_Invariants(value, "parse_optional_numeric.value")
		bytes.Boolean_Invariants(valid, "parse_optional_numeric.valid")
	}()
	Optional_Numeric_Field_Invariants(field, "parse_optional_numeric.field")
	if field[0] == 0 {
		return Optional_Integer{}, true
	}
	parsed, valid := parse_numeric(Numeric_Field(field))
	return Optional_Integer{Value: parsed, Set: bytes.Boolean(valid)}, valid
}

func parse_octal(field Numeric_Field) (value Octal_Integer, valid bytes.Boolean) {
	defer func() {
		Octal_Integer_Invariants(value, "parse_octal.value")
		bytes.Boolean_Invariants(valid, "parse_octal.valid")
	}()
	Numeric_Field_Invariants(field, "parse_octal.field")
	position := 0
	for position < len(field) {
		if field[position] != ' ' {
			if field[position] != 0 {
				break
			}
		}
		position++
	}
	digits := 0
	for position < len(field) {
		character := field[position]
		if character == 0 {
			break
		}
		if character == ' ' {
			break
		}
		if character < '0' {
			return 0, false
		}
		if character > '7' {
			return 0, false
		}
		if value > (Octal_Integer(OCTAL_PARSE_MAXIMUM)-
			Octal_Integer(character-'0'))/OCTAL_BASE {
			return 0, false
		}
		value = value*OCTAL_BASE + Octal_Integer(character-'0')
		digits++
		position++
	}
	for position < len(field) {
		if field[position] != 0 {
			if field[position] != ' ' {
				return 0, false
			}
		}
		position++
	}
	if digits != 0 {
		return value, true
	}
	return value, value == 0
}

func parse_base_256(field Numeric_Field) (value Integer, valid bytes.Boolean) {
	defer func() {
		Integer_Invariants(value, "parse_base_256.value")
		bytes.Boolean_Invariants(valid, "parse_base_256.valid")
	}()
	Numeric_Field_Invariants(field, "parse_base_256.field")
	inversion := byte(0)
	if field[0]&BASE_256_SIGN_MASK != 0 {
		inversion = byte(TYPE_FLAG_MAXIMUM)
	}
	unsigned := uint64(0)
	for position, character := range field {
		character ^= inversion
		if position == 0 {
			character &= BASE_256_FIRST_DATA_MASK
		}
		if unsigned>>INTEGER_BYTE_SHIFT != 0 {
			return 0, false
		}
		unsigned = unsigned<<bits.BIT_COUNT_8_MAXIMUM | uint64(character)
	}
	if unsigned>>INTEGER_SIGN_SHIFT != 0 {
		return 0, false
	}
	if inversion == byte(TYPE_FLAG_MAXIMUM) {
		return Integer(^int64(unsigned)), true
	}
	return Integer(unsigned), true
}

func header_checksum(
	block Header_Block,
) (unsigned Unsigned_Header_Checksum, signed Signed_Header_Checksum) {
	defer func() {
		Unsigned_Header_Checksum_Invariants(unsigned, "header_checksum.unsigned")
		Signed_Header_Checksum_Invariants(signed, "header_checksum.signed")
	}()
	Header_Block_Invariants(block, "header_checksum.block")
	for position, value := range block {
		if position >= CHECKSUM_FIELD_OFFSET {
			if position < CHECKSUM_FIELD_OFFSET+CHECKSUM_FIELD_SIZE {
				value = ' '
			}
		}
		unsigned += Unsigned_Header_Checksum(value)
		signed += Signed_Header_Checksum(int8(value))
	}
	return unsigned, signed
}

func pax_records_valid(records PAX_Records) (valid bytes.Boolean) {
	defer func() {
		bytes.Boolean_Invariants(valid, "pax_records_valid.valid")
	}()
	PAX_Records_Invariants(records, "pax_records_valid.records")
	position := Count(COUNT_MINIMUM)
	for position < Count(len(records)) {
		size, _, _, record_valid := pax_record(PAX_Record_Input(records[position:]))
		if !record_valid {
			return false
		}
		position += Count(size)
	}
	return position == Count(len(records))
}

func pax_record(
	records PAX_Record_Input,
) (size PAX_Parsed_Record_Size, key PAX_Key, value PAX_Value, valid bytes.Boolean) {
	defer func() {
		PAX_Parsed_Record_Size_Invariants(size, "pax_record.size")
		PAX_Key_Invariants(key, "pax_record.key")
		PAX_Value_Invariants(value, "pax_record.value")
		bytes.Boolean_Invariants(valid, "pax_record.valid")
	}()
	PAX_Record_Input_Invariants(records, "pax_record.records")
	invalid_size := PAX_Parsed_Record_Size(PAX_RECORD_SIZE_MINIMUM)
	record_size := 0
	digit_count := 0
	for cursor := COUNT_MINIMUM; cursor < len(records); cursor++ {
		character := records[cursor]
		if character == ' ' {
			if digit_count == 0 {
				return invalid_size, nil, nil, false
			}
			start := cursor + PAX_COUNT_SEPARATOR_SIZE
			size = PAX_Parsed_Record_Size(record_size)
			if int(size) > len(records) {
				return invalid_size, nil, nil, false
			}
			if int(size) <= start {
				return invalid_size, nil, nil, false
			}
			if records[int(size)-PAX_RECORD_TERMINATOR_SIZE] != '\n' {
				return invalid_size, nil, nil, false
			}
			equals := start
			for equals < int(size)-PAX_RECORD_TERMINATOR_SIZE {
				if records[equals] == '=' {
					break
				}
				equals++
			}
			if equals == start {
				return invalid_size, nil, nil, false
			}
			if equals == int(size)-PAX_RECORD_TERMINATOR_SIZE {
				return invalid_size, nil, nil, false
			}
			key = PAX_Key(records[start:equals])
			value_start := equals + PAX_KEY_SEPARATOR_SIZE
			value_end := int(size) - PAX_RECORD_TERMINATOR_SIZE
			value = PAX_Value(records[value_start:value_end])
			if contains_nul(NUL_Checked_Text(key)) {
				return invalid_size, nil, nil, false
			}
			if pax_string_key(PAX_Parsed_Key(key)) {
				if contains_nul(NUL_Checked_Text(value)) {
					return invalid_size, nil, nil, false
				}
			}
			return size, key, value, true
		}
		if character < '0' {
			return invalid_size, nil, nil, false
		}
		if character > '9' {
			return invalid_size, nil, nil, false
		}
		if record_size > SPECIAL_FILE_SIZE_MAXIMUM/DECIMAL_BASE {
			return invalid_size, nil, nil, false
		}
		record_size = record_size*DECIMAL_BASE + int(character-'0')
		digit_count++
	}
	return invalid_size, nil, nil, false
}

func pax_apply(
	text_input PAX_Input_Text_Fields,
	numeric_input PAX_Input_Numeric_Fields,
	timestamp_input PAX_Timestamp_Input,
	records PAX_Valid_Records,
) (
	text PAX_Text_Fields,
	numeric PAX_Numeric_Fields,
	timestamps PAX_Timestamp_Fields,
	status PAX_Apply_Status,
) {
	defer func() {
		PAX_Text_Fields_Invariants(text, "pax_apply.text")
		PAX_Numeric_Fields_Invariants(numeric, "pax_apply.numeric")
		PAX_Timestamp_Fields_Invariants(timestamps, "pax_apply.timestamps")
		PAX_Apply_Status_Invariants(status, "pax_apply.status")
	}()
	PAX_Input_Text_Fields_Invariants(text_input, "pax_apply.text_input")
	PAX_Input_Numeric_Fields_Invariants(numeric_input, "pax_apply.numeric_input")
	PAX_Timestamp_Input_Invariants(timestamp_input, "pax_apply.timestamp_input")
	PAX_Valid_Records_Invariants(records, "pax_apply.records")
	text = PAX_Text_Fields{
		Name: text_input.Name, Link_Name: text_input.Link_Name,
		User_Name:  Header_User_Name(text_input.User_Name),
		Group_Name: Header_Group_Name(text_input.Group_Name),
	}
	numeric = PAX_Numeric_Fields{
		Size:             PAX_Entry_Size(numeric_input.Size),
		Physical_Size:    PAX_Physical_Size(numeric_input.Physical_Size),
		User_Identifier:  User_Identifier(numeric_input.User_Identifier),
		Group_Identifier: Group_Identifier(numeric_input.Group_Identifier),
	}
	timestamps = PAX_Timestamp_Fields{
		Modification_Time: Present_Timestamp{
			Seconds: Integer(timestamp_input.Modification_Seconds),
		},
		Access_Time: Access_Timestamp{
			Seconds: timestamp_input.Access_Time.Seconds,
			Set:     timestamp_input.Access_Time.Set,
		},
		Change_Time: Change_Timestamp{
			Seconds: timestamp_input.Change_Time.Seconds,
			Set:     timestamp_input.Change_Time.Set,
		},
	}
	position := Count(COUNT_MINIMUM)
	for position < Count(len(records)) {
		size, key, value, record_valid := pax_record(
			PAX_Record_Input(records[position:]),
		)
		if !record_valid {
			return text, numeric, timestamps, PAX_Apply_Status(STATUS_INPUT_INVALID)
		}
		if len(value) == 0 {
			if pax_basic_key(PAX_Parsed_Key(key)) {
				position += Count(size)
				continue
			}
		}
		parsed_key := PAX_Parsed_Key(key)
		var record_status PAX_Apply_Status
		text, numeric, timestamps, record_status = pax_apply_record(
			text.Name, text.Link_Name, PAX_Apply_User_Name(text.User_Name),
			PAX_Apply_Group_Name(text.Group_Name), numeric, timestamps,
			parsed_key, value,
		)
		if record_status != PAX_Apply_Status(STATUS_OK) {
			return text, numeric, timestamps, record_status
		}
		position += Count(size)
	}
	return text, numeric, timestamps, PAX_Apply_Status(STATUS_OK)
}

func pax_apply_record(
	name Field_Value, link_name Header_Link_Name,
	user_name PAX_Apply_User_Name, group_name PAX_Apply_Group_Name,
	numeric PAX_Numeric_Fields, timestamps PAX_Timestamp_Fields,
	key PAX_Parsed_Key, value PAX_Value,
) (
	text_result PAX_Text_Fields, numeric_result PAX_Numeric_Fields,
	timestamp_result PAX_Timestamp_Fields, status PAX_Apply_Status,
) {
	defer func() {
		PAX_Text_Fields_Invariants(text_result, "pax_apply_record.text_result")
		PAX_Numeric_Fields_Invariants(numeric_result, "pax_apply_record.numeric_result")
		PAX_Timestamp_Fields_Invariants(
			timestamp_result, "pax_apply_record.timestamp_result",
		)
		PAX_Apply_Status_Invariants(status, "pax_apply_record.status")
	}()
	Field_Value_Invariants(name, "pax_apply_record.name")
	Header_Link_Name_Invariants(link_name, "pax_apply_record.link_name")
	PAX_Apply_User_Name_Invariants(user_name, "pax_apply_record.user_name")
	PAX_Apply_Group_Name_Invariants(group_name, "pax_apply_record.group_name")
	PAX_Numeric_Fields_Invariants(numeric, "pax_apply_record.numeric")
	PAX_Timestamp_Fields_Invariants(timestamps, "pax_apply_record.timestamps")
	PAX_Parsed_Key_Invariants(key, "pax_apply_record.key")
	PAX_Value_Invariants(value, "pax_apply_record.value")
	text_result, matched := pax_apply_text_record(
		name, link_name, user_name, group_name, key, value,
	)
	numeric_result, timestamp_result = numeric, timestamps
	if matched {
		return text_result, numeric_result, timestamp_result, PAX_Apply_Status(STATUS_OK)
	}
	var numeric_status Input_Status
	numeric_result, matched, numeric_status = pax_apply_numeric_record(numeric, key, value)
	if numeric_status != Input_Status(STATUS_OK) {
		return text_result, numeric_result, timestamp_result,
			PAX_Apply_Status(numeric_status)
	}
	if matched {
		return text_result, numeric_result, timestamp_result, PAX_Apply_Status(STATUS_OK)
	}
	var timestamp_status Input_Status
	timestamp_result, matched, timestamp_status = pax_apply_timestamp_record(
		timestamps, key, value,
	)
	if timestamp_status != Input_Status(STATUS_OK) {
		return text_result, numeric_result, timestamp_result,
			PAX_Apply_Status(timestamp_status)
	}
	if !matched {
		if pax_sparse_key(key) {
			return text_result, numeric_result, timestamp_result,
				PAX_Apply_Status(STATUS_FORMAT_UNSUPPORTED)
		}
	}
	return text_result, numeric_result, timestamp_result, PAX_Apply_Status(STATUS_OK)
}

func pax_apply_text_record(
	name Field_Value, link_name Header_Link_Name,
	user_name PAX_Apply_User_Name, group_name PAX_Apply_Group_Name,
	key PAX_Parsed_Key, value PAX_Value,
) (result PAX_Text_Fields, matched bytes.Boolean) {
	defer func() {
		PAX_Text_Fields_Invariants(result, "pax_apply_text_record.result")
		bytes.Boolean_Invariants(matched, "pax_apply_text_record.matched")
	}()
	Field_Value_Invariants(name, "pax_apply_text_record.name")
	Header_Link_Name_Invariants(link_name, "pax_apply_text_record.link_name")
	PAX_Apply_User_Name_Invariants(user_name, "pax_apply_text_record.user_name")
	PAX_Apply_Group_Name_Invariants(group_name, "pax_apply_text_record.group_name")
	PAX_Parsed_Key_Invariants(key, "pax_apply_text_record.key")
	PAX_Value_Invariants(value, "pax_apply_text_record.value")
	result = PAX_Text_Fields{
		Name: name, Link_Name: link_name,
		User_Name:  Header_User_Name(user_name),
		Group_Name: Header_Group_Name(group_name),
	}
	switch {
	case bool(pax_key_equal(key, PAX_PATH_KEY)):
		result.Name = Field_Value{Suffix: Name_Suffix(value)}
	case bool(pax_key_equal(key, PAX_LINK_PATH_KEY)):
		result.Link_Name = Header_Link_Name(value)
	case bool(pax_key_equal(key, PAX_USER_NAME_KEY)):
		result.User_Name = Header_User_Name(value)
	case bool(pax_key_equal(key, PAX_GROUP_NAME_KEY)):
		result.Group_Name = Header_Group_Name(value)
	default:
		return result, false
	}
	return result, true
}

func pax_apply_numeric_record(
	numeric PAX_Numeric_Fields, key PAX_Parsed_Key, value PAX_Value,
) (result PAX_Numeric_Fields, matched bytes.Boolean, status Input_Status) {
	defer func() {
		PAX_Numeric_Fields_Invariants(result, "pax_apply_numeric_record.result")
		bytes.Boolean_Invariants(matched, "pax_apply_numeric_record.matched")
		Input_Status_Invariants(status, "pax_apply_numeric_record.status")
	}()
	PAX_Numeric_Fields_Invariants(numeric, "pax_apply_numeric_record.numeric")
	PAX_Parsed_Key_Invariants(key, "pax_apply_numeric_record.key")
	PAX_Value_Invariants(value, "pax_apply_numeric_record.value")
	result = numeric
	if pax_key_equal(key, PAX_SIZE_KEY) {
		parsed, valid := parse_decimal(Decimal_Candidate(value))
		if !valid {
			return result, true, Input_Status(STATUS_INPUT_INVALID)
		}
		if parsed < 0 {
			return result, true, Input_Status(STATUS_INPUT_INVALID)
		}
		result.Size = PAX_Entry_Size(parsed)
		result.Physical_Size = PAX_Physical_Size(parsed)
		return result, true, Input_Status(STATUS_OK)
	}
	if pax_key_equal(key, PAX_USER_IDENTIFIER_KEY) {
		parsed, valid := parse_decimal(Decimal_Candidate(value))
		if !valid {
			return result, true, Input_Status(STATUS_INPUT_INVALID)
		}
		result.User_Identifier = User_Identifier(parsed)
		return result, true, Input_Status(STATUS_OK)
	}
	if pax_key_equal(key, PAX_GROUP_IDENTIFIER_KEY) {
		parsed, valid := parse_decimal(Decimal_Candidate(value))
		if !valid {
			return result, true, Input_Status(STATUS_INPUT_INVALID)
		}
		result.Group_Identifier = Group_Identifier(parsed)
		return result, true, Input_Status(STATUS_OK)
	}
	return result, false, Input_Status(STATUS_OK)
}

func pax_apply_timestamp_record(
	timestamps PAX_Timestamp_Fields, key PAX_Parsed_Key, value PAX_Value,
) (result PAX_Timestamp_Fields, matched bytes.Boolean, status Input_Status) {
	defer func() {
		PAX_Timestamp_Fields_Invariants(result, "pax_apply_timestamp_record.result")
		bytes.Boolean_Invariants(matched, "pax_apply_timestamp_record.matched")
		Input_Status_Invariants(status, "pax_apply_timestamp_record.status")
	}()
	PAX_Timestamp_Fields_Invariants(timestamps, "pax_apply_timestamp_record.timestamps")
	PAX_Parsed_Key_Invariants(key, "pax_apply_timestamp_record.key")
	PAX_Value_Invariants(value, "pax_apply_timestamp_record.value")
	result = timestamps
	parsed, matched, valid := pax_timestamp_record(key, value)
	if !matched {
		return result, false, Input_Status(STATUS_OK)
	}
	if !valid {
		return result, true, Input_Status(STATUS_INPUT_INVALID)
	}
	if pax_key_equal(key, PAX_MODIFICATION_TIME_KEY) {
		result.Modification_Time = Present_Timestamp{
			Seconds: parsed.Seconds, Nanoseconds: parsed.Nanoseconds,
		}
		return result, true, Input_Status(STATUS_OK)
	}
	if pax_key_equal(key, PAX_ACCESS_TIME_KEY) {
		result.Access_Time = Access_Timestamp{
			Seconds:     Access_Seconds(parsed.Seconds),
			Nanoseconds: Access_Nanosecond_Count(parsed.Nanoseconds),
			Set:         Access_Time_Set(parsed.Set),
		}
		return result, true, Input_Status(STATUS_OK)
	}
	result.Change_Time = Change_Timestamp{
		Seconds:     Change_Seconds(parsed.Seconds),
		Nanoseconds: Change_Nanosecond_Count(parsed.Nanoseconds),
		Set:         Change_Time_Set(parsed.Set),
	}
	return result, true, Input_Status(STATUS_OK)
}

func pax_timestamp_record(
	key PAX_Parsed_Key, value PAX_Value,
) (timestamp Timestamp, matched bytes.Boolean, valid bytes.Boolean) {
	defer func() {
		Timestamp_Invariants(timestamp, "pax_timestamp_record.timestamp")
		bytes.Boolean_Invariants(matched, "pax_timestamp_record.matched")
		bytes.Boolean_Invariants(valid, "pax_timestamp_record.valid")
	}()
	PAX_Parsed_Key_Invariants(key, "pax_timestamp_record.key")
	PAX_Value_Invariants(value, "pax_timestamp_record.value")
	if pax_key_equal(key, PAX_MODIFICATION_TIME_KEY) {
		timestamp, valid = parse_timestamp(PAX_Timestamp_Candidate(value))
		return timestamp, true, valid
	}
	if pax_key_equal(key, PAX_ACCESS_TIME_KEY) {
		timestamp, valid = parse_timestamp(PAX_Timestamp_Candidate(value))
		return timestamp, true, valid
	}
	if pax_key_equal(key, PAX_CHANGE_TIME_KEY) {
		timestamp, valid = parse_timestamp(PAX_Timestamp_Candidate(value))
		return timestamp, true, valid
	}
	return Timestamp{}, false, true
}

func parse_decimal(value Decimal_Candidate) (parsed Integer, valid bytes.Boolean) {
	defer func() {
		Integer_Invariants(parsed, "parse_decimal.parsed")
		bytes.Boolean_Invariants(valid, "parse_decimal.valid")
	}()
	Decimal_Candidate_Invariants(value, "parse_decimal.value")
	if len(value) == 0 {
		return 0, false
	}
	position := 0
	negative := false
	if value[0] == '-' {
		negative = true
		position++
	}
	if position == len(value) {
		return 0, false
	}
	limit := uint64(INTEGER_MAXIMUM)
	if negative {
		limit++
	}
	unsigned := uint64(0)
	for ; position < len(value); position++ {
		digit := value[position]
		if digit < '0' {
			return 0, false
		}
		if digit > '9' {
			return 0, false
		}
		unsigned_digit := uint64(digit - '0')
		if unsigned > (limit-unsigned_digit)/DECIMAL_BASE {
			return 0, false
		}
		unsigned = unsigned*DECIMAL_BASE + unsigned_digit
	}
	if negative {
		if unsigned == uint64(INTEGER_MAXIMUM)+1 {
			return INTEGER_MINIMUM, true
		}
		return Integer(-int64(unsigned)), true
	}
	return Integer(unsigned), true
}

func parse_timestamp(
	value PAX_Timestamp_Candidate,
) (timestamp Timestamp, valid bytes.Boolean) {
	defer func() {
		Timestamp_Invariants(timestamp, "parse_timestamp.timestamp")
		bytes.Boolean_Invariants(valid, "parse_timestamp.valid")
	}()
	PAX_Timestamp_Candidate_Invariants(value, "parse_timestamp.value")
	seconds_size := len(value)
	for index, character := range value {
		if character == '.' {
			seconds_size = index
			break
		}
	}
	seconds, seconds_valid := parse_decimal(Decimal_Candidate(value[:seconds_size]))
	if !seconds_valid {
		return Timestamp{}, false
	}
	nanoseconds := int32(0)
	if seconds_size != len(value) {
		fraction_start_size := seconds_size + TIMESTAMP_FRACTION_SEPARATOR_SIZE
		fraction := value[fraction_start_size:]
		if len(fraction) == 0 {
			return Timestamp{}, false
		}
		if len(fraction) > TIMESTAMP_FRACTION_DIGIT_COUNT_MAXIMUM {
			return Timestamp{}, false
		}
		for _, digit := range fraction {
			if digit < '0' {
				return Timestamp{}, false
			}
			if digit > '9' {
				return Timestamp{}, false
			}
			nanoseconds = nanoseconds*DECIMAL_BASE + int32(digit-'0')
		}
		fraction_count := len(fraction)
		for fraction_count < TIMESTAMP_FRACTION_DIGIT_COUNT_MAXIMUM {
			nanoseconds *= DECIMAL_BASE
			fraction_count++
		}
	}
	if len(value) != 0 {
		if value[0] == '-' {
			if nanoseconds != 0 {
				seconds--
				nanoseconds = int32(TIMESTAMP_NANOSECOND_COUNT) - nanoseconds
			}
		}
	}
	return Timestamp{
		Seconds: seconds, Nanoseconds: Nanosecond_Count(nanoseconds), Set: true,
	}, true
}

func encoded_header_size(
	format Encoding_Format,
	name Writer_Name,
	link_name Header_Link_Name,
	size_value Entry_Size,
	mode File_Mode,
	user_identifier User_Identifier,
	group_identifier Group_Identifier,
	user_name Header_User_Name,
	group_name Header_Group_Name,
	modified Timestamp,
	accessed Access_Timestamp,
	changed Change_Timestamp,
	device_major Device_Major,
	device_minor Device_Minor,
) (size Encoded_Header_Size, status Header_Encoding_Status) {
	defer func() {
		Encoded_Header_Size_Invariants(size, "encoded_header_size.size")
		Header_Encoding_Status_Invariants(status, "encoded_header_size.status")
	}()
	Encoding_Format_Invariants(format, "encoded_header_size.format")
	Writer_Name_Invariants(name, "encoded_header_size.name")
	Header_Link_Name_Invariants(link_name, "encoded_header_size.link_name")
	Entry_Size_Invariants(size_value, "encoded_header_size.size_value")
	File_Mode_Invariants(mode, "encoded_header_size.mode")
	User_Identifier_Invariants(user_identifier, "encoded_header_size.user_identifier")
	Group_Identifier_Invariants(group_identifier, "encoded_header_size.group_identifier")
	Header_User_Name_Invariants(user_name, "encoded_header_size.user_name")
	Header_Group_Name_Invariants(group_name, "encoded_header_size.group_name")
	Timestamp_Invariants(modified, "encoded_header_size.modified")
	Access_Timestamp_Invariants(accessed, "encoded_header_size.accessed")
	Change_Timestamp_Invariants(changed, "encoded_header_size.changed")
	Device_Major_Invariants(device_major, "encoded_header_size.device_major")
	Device_Minor_Invariants(device_minor, "encoded_header_size.device_minor")
	switch Format(format) {
	case FORMAT_V7, FORMAT_USTAR:
		format_status := basic_encoded_header_size(
			Basic_Fit_Format(format),
			name, link_name, size_value, mode, user_identifier, group_identifier,
			user_name, group_name, modified, accessed, changed,
			device_major, device_minor,
		)
		return Encoded_Header_Size(BLOCK_SIZE), Header_Encoding_Status(format_status)
	case FORMAT_PAX:
		pax_size, format_status := pax_encoded_header_size(
			name, link_name, size_value, mode, user_identifier, group_identifier,
			user_name, group_name, modified, accessed, changed,
			device_major, device_minor,
		)
		return Encoded_Header_Size(pax_size), Header_Encoding_Status(format_status)
	case FORMAT_GNU:
		gnu_size, format_status := gnu_encoded_header_size(
			name, link_name, user_name, group_name,
			GNU_Numeric_Fields{
				Mode: mode, User_Identifier: user_identifier,
				Group_Identifier: group_identifier, Size: size_value,
				Modification_Time: modified,
				Access_Time:       accessed, Change_Time: changed,
				Device_Major: device_major, Device_Minor: device_minor,
			},
		)
		return Encoded_Header_Size(gnu_size), Header_Encoding_Status(format_status)
	default:
		return Encoded_Header_Size(BLOCK_SIZE),
			Header_Encoding_Status(STATUS_FORMAT_UNSUPPORTED)
	}
}

func basic_encoded_header_size(
	format Basic_Fit_Format,
	name Writer_Name, link_name Header_Link_Name, size Entry_Size, mode File_Mode,
	user_identifier User_Identifier, group_identifier Group_Identifier,
	user_name Header_User_Name, group_name Header_Group_Name,
	modified Timestamp, accessed Access_Timestamp, changed Change_Timestamp,
	device_major Device_Major, device_minor Device_Minor,
) (status Format_Encoding_Status) {
	defer func() {
		Format_Encoding_Status_Invariants(status, "basic_encoded_header_size.status")
	}()
	Basic_Fit_Format_Invariants(format, "basic_encoded_header_size.format")
	Writer_Name_Invariants(name, "basic_encoded_header_size.name")
	Header_Link_Name_Invariants(link_name, "basic_encoded_header_size.link_name")
	Entry_Size_Invariants(size, "basic_encoded_header_size.size")
	File_Mode_Invariants(mode, "basic_encoded_header_size.mode")
	User_Identifier_Invariants(
		user_identifier, "basic_encoded_header_size.user_identifier",
	)
	Group_Identifier_Invariants(
		group_identifier, "basic_encoded_header_size.group_identifier",
	)
	Header_User_Name_Invariants(user_name, "basic_encoded_header_size.user_name")
	Header_Group_Name_Invariants(group_name, "basic_encoded_header_size.group_name")
	Timestamp_Invariants(modified, "basic_encoded_header_size.modified")
	Access_Timestamp_Invariants(accessed, "basic_encoded_header_size.accessed")
	Change_Timestamp_Invariants(changed, "basic_encoded_header_size.changed")
	Device_Major_Invariants(device_major, "basic_encoded_header_size.device_major")
	Device_Minor_Invariants(device_minor, "basic_encoded_header_size.device_minor")
	if !header_fits_basic(
		name, link_name, size, mode, user_identifier, group_identifier,
		user_name, group_name, modified, accessed, changed,
		device_major, device_minor, format,
	) {
		return Format_Encoding_Status(STATUS_FIELD_TOO_LONG)
	}
	return Format_Encoding_Status(STATUS_OK)
}

func pax_encoded_header_size(
	name Writer_Name, link_name Header_Link_Name, size Entry_Size, mode File_Mode,
	user_identifier User_Identifier, group_identifier Group_Identifier,
	user_name Header_User_Name, group_name Header_Group_Name,
	modified Timestamp, accessed Access_Timestamp, changed Change_Timestamp,
	device_major Device_Major, device_minor Device_Minor,
) (size_value PAX_Encoded_Header_Size, status Format_Encoding_Status) {
	defer func() {
		PAX_Encoded_Header_Size_Invariants(size_value, "pax_encoded_header_size.size_value")
		Format_Encoding_Status_Invariants(status, "pax_encoded_header_size.status")
	}()
	Writer_Name_Invariants(name, "pax_encoded_header_size.name")
	Header_Link_Name_Invariants(link_name, "pax_encoded_header_size.link_name")
	Entry_Size_Invariants(size, "pax_encoded_header_size.size")
	File_Mode_Invariants(mode, "pax_encoded_header_size.mode")
	User_Identifier_Invariants(user_identifier, "pax_encoded_header_size.user_identifier")
	Group_Identifier_Invariants(group_identifier, "pax_encoded_header_size.group_identifier")
	Header_User_Name_Invariants(user_name, "pax_encoded_header_size.user_name")
	Header_Group_Name_Invariants(group_name, "pax_encoded_header_size.group_name")
	Timestamp_Invariants(modified, "pax_encoded_header_size.modified")
	Access_Timestamp_Invariants(accessed, "pax_encoded_header_size.accessed")
	Change_Timestamp_Invariants(changed, "pax_encoded_header_size.changed")
	Device_Major_Invariants(device_major, "pax_encoded_header_size.device_major")
	Device_Minor_Invariants(device_minor, "pax_encoded_header_size.device_minor")
	payload_size, valid := pax_payload_size(
		name, link_name, size, mode, user_identifier, group_identifier,
		user_name, group_name, modified, accessed, changed,
		device_major, device_minor,
	)
	if !valid {
		return PAX_Encoded_Header_Size(PAX_HEADER_WIRE_SIZE_MINIMUM),
			Format_Encoding_Status(STATUS_FIELD_TOO_LONG)
	}
	blocks := padded_block_count(Metadata_Payload_Size(payload_size))
	wire_size := (HEADER_BLOCK_COUNT + int(blocks) + HEADER_BLOCK_COUNT) * BLOCK_SIZE
	return PAX_Encoded_Header_Size(wire_size), Format_Encoding_Status(STATUS_OK)
}

func gnu_encoded_header_size(
	name Writer_Name, link_name Header_Link_Name,
	user_name Header_User_Name, group_name Header_Group_Name,
	numeric GNU_Numeric_Fields,
) (size GNU_Encoded_Header_Size, status Format_Encoding_Status) {
	defer func() {
		GNU_Encoded_Header_Size_Invariants(size, "gnu_encoded_header_size.size")
		Format_Encoding_Status_Invariants(status, "gnu_encoded_header_size.status")
	}()
	Writer_Name_Invariants(name, "gnu_encoded_header_size.name")
	Header_Link_Name_Invariants(link_name, "gnu_encoded_header_size.link_name")
	Header_User_Name_Invariants(user_name, "gnu_encoded_header_size.user_name")
	Header_Group_Name_Invariants(group_name, "gnu_encoded_header_size.group_name")
	GNU_Numeric_Fields_Invariants(numeric, "gnu_encoded_header_size.numeric")
	if !gnu_numeric_fields_fit(numeric) {
		return GNU_Encoded_Header_Size(BLOCK_SIZE),
			Format_Encoding_Status(STATUS_FIELD_TOO_LONG)
	}
	if len(user_name) > USER_NAME_FIELD_SIZE {
		return GNU_Encoded_Header_Size(BLOCK_SIZE),
			Format_Encoding_Status(STATUS_FIELD_TOO_LONG)
	}
	if len(group_name) > GROUP_NAME_FIELD_SIZE {
		return GNU_Encoded_Header_Size(BLOCK_SIZE),
			Format_Encoding_Status(STATUS_FIELD_TOO_LONG)
	}
	size = GNU_Encoded_Header_Size(BLOCK_SIZE)
	if len(name) > NAME_FIELD_SIZE {
		long_size := gnu_long_encoded_size(GNU_Extended_Field(name))
		size += GNU_Encoded_Header_Size(long_size)
	}
	if len(link_name) > NAME_FIELD_SIZE {
		long_size := gnu_long_encoded_size(GNU_Extended_Field(link_name))
		size += GNU_Encoded_Header_Size(long_size)
	}
	return size, Format_Encoding_Status(STATUS_OK)
}

func gnu_long_encoded_size(
	value GNU_Extended_Field,
) (size GNU_Long_Encoded_Size) {
	defer func() {
		GNU_Long_Encoded_Size_Invariants(size, "gnu_long_encoded_size.size")
	}()
	GNU_Extended_Field_Invariants(value, "gnu_long_encoded_size.value")
	physical_size := len(value) + GNU_LONG_FIELD_TERMINATOR_SIZE
	blocks := padded_block_count(Metadata_Payload_Size(physical_size))
	return GNU_Long_Encoded_Size((HEADER_BLOCK_COUNT + int(blocks)) * BLOCK_SIZE)
}

func gnu_numeric_fields_fit(fields GNU_Numeric_Fields) (fits bytes.Boolean) {
	defer func() {
		bytes.Boolean_Invariants(fits, "gnu_numeric_fields_fit.fits")
	}()
	GNU_Numeric_Fields_Invariants(fields, "gnu_numeric_fields_fit.fields")
	if !base_256_fits(MODE_FIELD_SIZE, Integer(fields.Mode)) {
		return false
	}
	if !base_256_fits(USER_IDENTIFIER_FIELD_SIZE, Integer(fields.User_Identifier)) {
		return false
	}
	if !base_256_fits(GROUP_IDENTIFIER_FIELD_SIZE, Integer(fields.Group_Identifier)) {
		return false
	}
	if !base_256_fits(ENTRY_SIZE_FIELD_SIZE, Integer(fields.Size)) {
		return false
	}
	if fields.Modification_Time.Set {
		if !base_256_fits(TIMESTAMP_FIELD_SIZE, fields.Modification_Time.Seconds) {
			return false
		}
	}
	if fields.Access_Time.Set {
		if !base_256_fits(
			TIMESTAMP_FIELD_SIZE, Integer(fields.Access_Time.Seconds),
		) {
			return false
		}
	}
	if fields.Change_Time.Set {
		if !base_256_fits(
			TIMESTAMP_FIELD_SIZE, Integer(fields.Change_Time.Seconds),
		) {
			return false
		}
	}
	if !base_256_fits(DEVICE_MAJOR_FIELD_SIZE, Integer(fields.Device_Major)) {
		return false
	}
	return base_256_fits(DEVICE_MINOR_FIELD_SIZE, Integer(fields.Device_Minor))
}

func header_fits_basic(
	name Writer_Name,
	link_name Header_Link_Name,
	size Entry_Size,
	mode File_Mode,
	user_identifier User_Identifier,
	group_identifier Group_Identifier,
	user_name Header_User_Name,
	group_name Header_Group_Name,
	modified Timestamp,
	accessed Access_Timestamp,
	changed Change_Timestamp,
	device_major Device_Major,
	device_minor Device_Minor,
	format Basic_Fit_Format,
) (fits bytes.Boolean) {
	defer func() {
		bytes.Boolean_Invariants(fits, "header_fits_basic.fits")
	}()
	Writer_Name_Invariants(name, "header_fits_basic.name")
	Header_Link_Name_Invariants(link_name, "header_fits_basic.link_name")
	Entry_Size_Invariants(size, "header_fits_basic.size")
	File_Mode_Invariants(mode, "header_fits_basic.mode")
	User_Identifier_Invariants(user_identifier, "header_fits_basic.user_identifier")
	Group_Identifier_Invariants(group_identifier, "header_fits_basic.group_identifier")
	Header_User_Name_Invariants(user_name, "header_fits_basic.user_name")
	Header_Group_Name_Invariants(group_name, "header_fits_basic.group_name")
	Timestamp_Invariants(modified, "header_fits_basic.modified")
	Access_Timestamp_Invariants(accessed, "header_fits_basic.accessed")
	Change_Timestamp_Invariants(changed, "header_fits_basic.changed")
	Device_Major_Invariants(device_major, "header_fits_basic.device_major")
	Device_Minor_Invariants(device_minor, "header_fits_basic.device_minor")
	Basic_Fit_Format_Invariants(format, "header_fits_basic.format")
	if len(link_name) > NAME_FIELD_SIZE {
		return false
	}
	if len(user_name) > USER_NAME_FIELD_SIZE {
		return false
	}
	if len(group_name) > GROUP_NAME_FIELD_SIZE {
		return false
	}
	if format == Basic_Fit_Format(FORMAT_V7) {
		if len(name) > NAME_FIELD_SIZE {
			return false
		}
	} else if !ustar_name_fits(name) {
		return false
	}
	if !octal_fits(MODE_FIELD_SIZE, Integer(mode)) {
		return false
	}
	if !octal_fits(USER_IDENTIFIER_FIELD_SIZE, Integer(user_identifier)) {
		return false
	}
	if !octal_fits(GROUP_IDENTIFIER_FIELD_SIZE, Integer(group_identifier)) {
		return false
	}
	if !octal_fits(ENTRY_SIZE_FIELD_SIZE, Integer(size)) {
		return false
	}
	if modified.Set {
		if modified.Nanoseconds != 0 {
			return false
		}
		if !octal_fits(TIMESTAMP_FIELD_SIZE, modified.Seconds) {
			return false
		}
	}
	if bool(accessed.Set) {
		return false
	}
	if bool(changed.Set) {
		return false
	}
	if format != Basic_Fit_Format(FORMAT_V7) {
		if !octal_fits(DEVICE_MAJOR_FIELD_SIZE, Integer(device_major)) {
			return false
		}
		if !octal_fits(DEVICE_MINOR_FIELD_SIZE, Integer(device_minor)) {
			return false
		}
	}
	return true
}

func pax_payload_size(
	name Writer_Name,
	link_name Header_Link_Name,
	size_value Entry_Size,
	mode File_Mode,
	user_identifier User_Identifier,
	group_identifier Group_Identifier,
	user_name Header_User_Name,
	group_name Header_Group_Name,
	modified Timestamp,
	accessed Access_Timestamp,
	changed Change_Timestamp,
	device_major Device_Major,
	device_minor Device_Minor,
) (size PAX_Payload_Size_Candidate, valid bytes.Boolean) {
	defer func() {
		PAX_Payload_Size_Candidate_Invariants(size, "pax_payload_size.size")
		bytes.Boolean_Invariants(valid, "pax_payload_size.valid")
	}()
	Writer_Name_Invariants(name, "pax_payload_size.name")
	Header_Link_Name_Invariants(link_name, "pax_payload_size.link_name")
	Entry_Size_Invariants(size_value, "pax_payload_size.size_value")
	File_Mode_Invariants(mode, "pax_payload_size.mode")
	User_Identifier_Invariants(user_identifier, "pax_payload_size.user_identifier")
	Group_Identifier_Invariants(group_identifier, "pax_payload_size.group_identifier")
	Header_User_Name_Invariants(user_name, "pax_payload_size.user_name")
	Header_Group_Name_Invariants(group_name, "pax_payload_size.group_name")
	Timestamp_Invariants(modified, "pax_payload_size.modified")
	Access_Timestamp_Invariants(accessed, "pax_payload_size.accessed")
	Change_Timestamp_Invariants(changed, "pax_payload_size.changed")
	Device_Major_Invariants(device_major, "pax_payload_size.device_major")
	Device_Minor_Invariants(device_minor, "pax_payload_size.device_minor")
	if !pax_header_text_valid(name, link_name, user_name, group_name) {
		return PAX_Payload_Size_Candidate(PAX_PAYLOAD_SIZE_MINIMUM), false
	}
	if !pax_fixed_fields_fit(mode, device_major, device_minor) {
		return PAX_Payload_Size_Candidate(PAX_PAYLOAD_SIZE_MINIMUM), false
	}
	size = PAX_Payload_Size_Candidate(
		pax_record_size(
			PAX_Record_Key_Size(len(PAX_PATH_KEY)), PAX_Record_Value_Size(len(name)),
		),
	)
	size += PAX_Payload_Size_Candidate(pax_record_size(
		PAX_Record_Key_Size(len(PAX_SIZE_KEY)),
		PAX_Record_Value_Size(decimal_value_size(Integer(size_value))),
	))
	if len(link_name) != 0 {
		size += PAX_Payload_Size_Candidate(pax_record_size(
			PAX_Record_Key_Size(len(PAX_LINK_PATH_KEY)),
			PAX_Record_Value_Size(len(link_name)),
		))
	}
	if user_identifier != 0 {
		size += PAX_Payload_Size_Candidate(pax_record_size(
			PAX_Record_Key_Size(len(PAX_USER_IDENTIFIER_KEY)),
			PAX_Record_Value_Size(decimal_value_size(Integer(user_identifier))),
		))
	}
	if group_identifier != 0 {
		size += PAX_Payload_Size_Candidate(pax_record_size(
			PAX_Record_Key_Size(len(PAX_GROUP_IDENTIFIER_KEY)),
			PAX_Record_Value_Size(decimal_value_size(Integer(group_identifier))),
		))
	}
	if len(user_name) != 0 {
		size += PAX_Payload_Size_Candidate(pax_record_size(
			PAX_Record_Key_Size(len(PAX_USER_NAME_KEY)),
			PAX_Record_Value_Size(len(user_name)),
		))
	}
	if len(group_name) != 0 {
		size += PAX_Payload_Size_Candidate(pax_record_size(
			PAX_Record_Key_Size(len(PAX_GROUP_NAME_KEY)),
			PAX_Record_Value_Size(len(group_name)),
		))
	}
	size = pax_time_payload_size(
		PAX_Time_Payload_Base_Size(size), modified, accessed, changed,
	)
	return size, size <= SPECIAL_FILE_SIZE_MAXIMUM
}

func pax_time_payload_size(
	size PAX_Time_Payload_Base_Size,
	modified Timestamp, accessed Access_Timestamp, changed Change_Timestamp,
) (result PAX_Payload_Size_Candidate) {
	defer func() {
		PAX_Payload_Size_Candidate_Invariants(result, "pax_time_payload_size.result")
	}()
	PAX_Time_Payload_Base_Size_Invariants(size, "pax_time_payload_size.size")
	Timestamp_Invariants(modified, "pax_time_payload_size.modified")
	Access_Timestamp_Invariants(accessed, "pax_time_payload_size.accessed")
	Change_Timestamp_Invariants(changed, "pax_time_payload_size.changed")
	result = PAX_Payload_Size_Candidate(size)
	if modified.Set {
		result += PAX_Payload_Size_Candidate(pax_record_size(
			PAX_Record_Key_Size(len(PAX_MODIFICATION_TIME_KEY)),
			PAX_Record_Value_Size(timestamp_value_size(
				modified.Seconds, modified.Nanoseconds,
			)),
		))
	}
	if accessed.Set {
		result += PAX_Payload_Size_Candidate(pax_record_size(
			PAX_Record_Key_Size(len(PAX_ACCESS_TIME_KEY)),
			PAX_Record_Value_Size(timestamp_value_size(
				Integer(accessed.Seconds), Nanosecond_Count(accessed.Nanoseconds),
			)),
		))
	}
	if changed.Set {
		result += PAX_Payload_Size_Candidate(pax_record_size(
			PAX_Record_Key_Size(len(PAX_CHANGE_TIME_KEY)),
			PAX_Record_Value_Size(timestamp_value_size(
				Integer(changed.Seconds), Nanosecond_Count(changed.Nanoseconds),
			)),
		))
	}
	return result
}

func pax_fixed_fields_fit(
	mode File_Mode, device_major Device_Major, device_minor Device_Minor,
) (fits bytes.Boolean) {
	defer func() {
		bytes.Boolean_Invariants(fits, "pax_fixed_fields_fit.fits")
	}()
	File_Mode_Invariants(mode, "pax_fixed_fields_fit.mode")
	Device_Major_Invariants(device_major, "pax_fixed_fields_fit.device_major")
	Device_Minor_Invariants(device_minor, "pax_fixed_fields_fit.device_minor")
	if !octal_fits(MODE_FIELD_SIZE, Integer(mode)) {
		return false
	}
	if !octal_fits(DEVICE_MAJOR_FIELD_SIZE, Integer(device_major)) {
		return false
	}
	return octal_fits(DEVICE_MINOR_FIELD_SIZE, Integer(device_minor))
}

func pax_header_text_valid(
	name Writer_Name,
	link_name Header_Link_Name,
	user_name Header_User_Name,
	group_name Header_Group_Name,
) (valid bytes.Boolean) {
	defer func() {
		bytes.Boolean_Invariants(valid, "pax_header_text_valid.valid")
	}()
	Writer_Name_Invariants(name, "pax_header_text_valid.name")
	Header_Link_Name_Invariants(link_name, "pax_header_text_valid.link_name")
	Header_User_Name_Invariants(user_name, "pax_header_text_valid.user_name")
	Header_Group_Name_Invariants(group_name, "pax_header_text_valid.group_name")
	if !valid_utf8(UTF8_Text(name)) {
		return false
	}
	if !valid_utf8(UTF8_Text(link_name)) {
		return false
	}
	if !valid_utf8(UTF8_Text(user_name)) {
		return false
	}
	return valid_utf8(UTF8_Text(group_name))
}

func pax_record_size(
	key_size PAX_Record_Key_Size, value_size PAX_Record_Value_Size,
) (size PAX_Record_Size_Candidate) {
	defer func() {
		PAX_Record_Size_Candidate_Invariants(size, "pax_record_size.size")
	}()
	PAX_Record_Key_Size_Invariants(key_size, "pax_record_size.key_size")
	PAX_Record_Value_Size_Invariants(value_size, "pax_record_size.value_size")
	body_size := Count(PAX_COUNT_SEPARATOR_SIZE) + Count(key_size) +
		Count(PAX_KEY_SEPARATOR_SIZE) + Count(value_size) +
		Count(PAX_RECORD_TERMINATOR_SIZE)
	size = PAX_Record_Size_Candidate(body_size + Count(decimal_value_size(
		Integer(body_size+Count(PAX_COUNT_SEPARATOR_SIZE)),
	)))
	for Count(decimal_value_size(Integer(size)))+body_size != Count(size) {
		size = PAX_Record_Size_Candidate(
			Count(decimal_value_size(Integer(size))) + body_size,
		)
	}
	return size
}

func write_pax_headers(
	destination PAX_Header_Destination, position Header_Start_Position,
	header PAX_Header, payload_size_value PAX_Payload_Size,
) (end PAX_Header_End_Position) {
	defer func() {
		PAX_Header_End_Position_Invariants(end, "write_pax_headers.end")
	}()
	PAX_Header_Destination_Invariants(destination, "write_pax_headers.destination")
	Header_Start_Position_Invariants(position, "write_pax_headers.position")
	PAX_Header_Invariants(header, "write_pax_headers.header")
	PAX_Payload_Size_Invariants(payload_size_value, "write_pax_headers.payload_size_value")
	payload_size := Count(payload_size_value)
	block := Header_Block(
		destination[position : position+BLOCK_SIZE],
	)
	zero(Zero_Destination(block))
	write_metadata_header(
		block, Metadata_Type_Flag(TYPE_PAX_LOCAL), Metadata_Size(payload_size),
	)
	position += BLOCK_SIZE
	payload_start := PAX_Payload_Start_Position(position)
	payload_end := position + Header_Start_Position(payload_size)
	payload := PAX_Payload_Destination(destination[position:payload_end])
	payload_position := pax_write_header_fields(payload, header)
	payload_position = pax_write_header_time_fields(
		payload, payload_position,
		header.Modification_Time, header.Access_Time, header.Change_Time,
	)
	end = pax_write_header_end(
		destination, payload_start, payload_position, header,
	)
	return end
}

func write_pax_headers_records(
	destination PAX_Header_Destination, position Header_Start_Position,
	header PAX_Header, payload_size_value PAX_Records_Payload_Size,
	records PAX_Header_Records,
) (end PAX_Header_End_Position) {
	defer func() {
		PAX_Header_End_Position_Invariants(end, "write_pax_headers_records.end")
	}()
	PAX_Header_Destination_Invariants(destination, "write_pax_headers_records.destination")
	Header_Start_Position_Invariants(position, "write_pax_headers_records.position")
	PAX_Header_Invariants(header, "write_pax_headers_records.header")
	PAX_Records_Payload_Size_Invariants(
		payload_size_value, "write_pax_headers_records.payload_size_value",
	)
	PAX_Header_Records_Invariants(records, "write_pax_headers_records.records")
	payload_size := Count(payload_size_value)
	block := Header_Block(destination[position : position+BLOCK_SIZE])
	zero(Zero_Destination(block))
	write_metadata_header(
		block, Metadata_Type_Flag(TYPE_PAX_LOCAL), Metadata_Size(payload_size),
	)
	position += BLOCK_SIZE
	payload_start := PAX_Payload_Start_Position(position)
	payload_end := position + Header_Start_Position(payload_size)
	payload := PAX_Payload_Destination(destination[position:payload_end])
	record_count := Count(copy(payload, records))
	generated := PAX_Payload_Destination(payload[record_count:])
	generated_count := pax_write_header_fields(generated, header)
	payload_position := PAX_Header_Field_Size(record_count) + generated_count
	payload_position = pax_write_header_time_fields(
		payload, payload_position,
		header.Modification_Time, header.Access_Time, header.Change_Time,
	)
	return pax_write_header_end(destination, payload_start, payload_position, header)
}

func pax_write_header_time_fields(
	payload PAX_Payload_Destination, position PAX_Header_Field_Size,
	modified Timestamp, accessed Access_Timestamp, changed Change_Timestamp,
) (result PAX_Header_Field_Size) {
	defer func() {
		PAX_Header_Field_Size_Invariants(result, "pax_write_header_time_fields.result")
	}()
	PAX_Payload_Destination_Invariants(payload, "pax_write_header_time_fields.payload")
	PAX_Header_Field_Size_Invariants(position, "pax_write_header_time_fields.position")
	Timestamp_Invariants(modified, "pax_write_header_time_fields.modified")
	Access_Timestamp_Invariants(accessed, "pax_write_header_time_fields.accessed")
	Change_Timestamp_Invariants(changed, "pax_write_header_time_fields.changed")
	result = position
	if modified.Set {
		time_size := pax_write_header_times(
			PAX_Time_Destination(payload[result:]), modified, accessed, changed,
		)
		result += PAX_Header_Field_Size(time_size)
	} else if accessed.Set {
		time_size := pax_write_header_times(
			PAX_Time_Destination(payload[result:]), modified, accessed, changed,
		)
		result += PAX_Header_Field_Size(time_size)
	} else if changed.Set {
		time_size := pax_write_header_times(
			PAX_Time_Destination(payload[result:]), modified, accessed, changed,
		)
		result += PAX_Header_Field_Size(time_size)
	}
	return result
}

func pax_write_header_end(
	destination PAX_Header_Destination, payload_start PAX_Payload_Start_Position,
	payload_size PAX_Header_Field_Size, header PAX_Header,
) (end PAX_Header_End_Position) {
	defer func() {
		PAX_Header_End_Position_Invariants(end, "pax_write_header_end.end")
	}()
	PAX_Header_Destination_Invariants(destination, "pax_write_header_end.destination")
	PAX_Payload_Start_Position_Invariants(
		payload_start, "pax_write_header_end.payload_start",
	)
	PAX_Header_Field_Size_Invariants(payload_size, "pax_write_header_end.payload_size")
	PAX_Header_Invariants(header, "pax_write_header_end.header")
	end = PAX_Header_End_Position(
		int(payload_start) + int(payload_size),
	)
	padding := Count(block_padding(Unpadded_Size(payload_size)))
	padding_end := end + PAX_Header_End_Position(padding)
	zero(Zero_Destination(destination[end:padding_end]))
	end = padding_end
	block := Header_Block(
		destination[end : end+BLOCK_SIZE],
	)
	zero(Zero_Destination(block))
	write_pax_main_header(block, header)
	return end + PAX_Header_End_Position(BLOCK_SIZE)
}

func pax_write_header_fields(
	payload PAX_Payload_Destination, header PAX_Header,
) (position PAX_Header_Field_Size) {
	defer func() {
		PAX_Header_Field_Size_Invariants(position, "pax_write_header_fields.position")
	}()
	PAX_Payload_Destination_Invariants(payload, "pax_write_header_fields.payload")
	PAX_Header_Invariants(header, "pax_write_header_fields.header")
	position = PAX_Header_Field_Size(pax_write_bytes(
		PAX_Text_Record_Destination(payload[position:]), PAX_PATH_KEY,
		PAX_Text_Value(header.Name),
	))
	position += PAX_Header_Field_Size(pax_write_decimal(
		PAX_Decimal_Record_Destination(payload[position:]), PAX_SIZE_KEY,
		Integer(header.Size),
	))
	if len(header.Link_Name) != 0 {
		position += PAX_Header_Field_Size(pax_write_bytes(
			PAX_Text_Record_Destination(payload[position:]), PAX_LINK_PATH_KEY,
			PAX_Text_Value(header.Link_Name),
		))
	}
	if header.User_Identifier != 0 {
		position += PAX_Header_Field_Size(pax_write_decimal(
			PAX_Decimal_Record_Destination(payload[position:]),
			PAX_USER_IDENTIFIER_KEY,
			Integer(header.User_Identifier),
		))
	}
	if header.Group_Identifier != 0 {
		position += PAX_Header_Field_Size(pax_write_decimal(
			PAX_Decimal_Record_Destination(payload[position:]),
			PAX_GROUP_IDENTIFIER_KEY,
			Integer(header.Group_Identifier),
		))
	}
	if len(header.User_Name) != 0 {
		position += PAX_Header_Field_Size(pax_write_bytes(
			PAX_Text_Record_Destination(payload[position:]), PAX_USER_NAME_KEY,
			PAX_Text_Value(header.User_Name),
		))
	}
	if len(header.Group_Name) != 0 {
		position += PAX_Header_Field_Size(pax_write_bytes(
			PAX_Text_Record_Destination(payload[position:]), PAX_GROUP_NAME_KEY,
			PAX_Text_Value(header.Group_Name),
		))
	}
	return position
}

func pax_write_header_times(
	payload PAX_Time_Destination,
	modified Timestamp, accessed Access_Timestamp, changed Change_Timestamp,
) (position PAX_Time_Size) {
	defer func() {
		PAX_Time_Size_Invariants(position, "pax_write_header_times.position")
	}()
	PAX_Time_Destination_Invariants(payload, "pax_write_header_times.payload")
	Timestamp_Invariants(modified, "pax_write_header_times.modified")
	Access_Timestamp_Invariants(accessed, "pax_write_header_times.accessed")
	Change_Timestamp_Invariants(changed, "pax_write_header_times.changed")
	if modified.Set {
		position += PAX_Time_Size(pax_write_timestamp(
			PAX_Timestamp_Record_Destination(payload[position:]),
			PAX_MODIFICATION_TIME_KEY,
			Present_Timestamp{
				Seconds: modified.Seconds, Nanoseconds: modified.Nanoseconds,
			},
		))
	}
	if accessed.Set {
		position += PAX_Time_Size(pax_write_timestamp(
			PAX_Timestamp_Record_Destination(payload[position:]), PAX_ACCESS_TIME_KEY,
			Present_Timestamp{
				Seconds:     Integer(accessed.Seconds),
				Nanoseconds: Nanosecond_Count(accessed.Nanoseconds),
			},
		))
	}
	if changed.Set {
		position += PAX_Time_Size(pax_write_timestamp(
			PAX_Timestamp_Record_Destination(payload[position:]), PAX_CHANGE_TIME_KEY,
			Present_Timestamp{
				Seconds:     Integer(changed.Seconds),
				Nanoseconds: Nanosecond_Count(changed.Nanoseconds),
			},
		))
	}
	return position
}

func write_pax_main_header(block Header_Block, header PAX_Header) {
	Header_Block_Invariants(block, "write_pax_main_header.block")
	PAX_Header_Invariants(header, "write_pax_main_header.header")
	main := header
	main.Name = PAX_Header_Name(pax_fallback_name(header.Name))
	if len(main.Link_Name) > NAME_FIELD_SIZE {
		main.Link_Name = main.Link_Name[:NAME_FIELD_SIZE]
	}
	if len(main.User_Name) > USER_NAME_FIELD_SIZE {
		main.User_Name = main.User_Name[:USER_NAME_FIELD_SIZE]
	}
	if len(main.Group_Name) > GROUP_NAME_FIELD_SIZE {
		main.Group_Name = main.Group_Name[:GROUP_NAME_FIELD_SIZE]
	}
	if !octal_fits(USER_IDENTIFIER_FIELD_SIZE, Integer(main.User_Identifier)) {
		main.User_Identifier = 0
	}
	if !octal_fits(GROUP_IDENTIFIER_FIELD_SIZE, Integer(main.Group_Identifier)) {
		main.Group_Identifier = 0
	}
	if !octal_fits(ENTRY_SIZE_FIELD_SIZE, Integer(main.Size)) {
		main.Size = 0
	}
	if !octal_fits(TIMESTAMP_FIELD_SIZE, main.Modification_Time.Seconds) {
		main.Modification_Time = Timestamp{}
	} else {
		main.Modification_Time.Nanoseconds = 0
	}
	modified := Basic_Modification_Seconds(0)
	if main.Modification_Time.Set {
		modified = Basic_Modification_Seconds(main.Modification_Time.Seconds)
	}
	write_basic_header(block, writer_basic_header(
		Basic_Format(FORMAT_PAX), main.Type_Flag,
		USTAR_Name(main.Name), Basic_Link_Name(main.Link_Name), main.Size,
		Basic_File_Mode(main.Mode), Basic_User_Identifier(main.User_Identifier),
		Basic_Group_Identifier(main.Group_Identifier),
		USTAR_User_Name(main.User_Name), USTAR_Group_Name(main.Group_Name), modified,
		Device_Major(main.Device_Major), Device_Minor(main.Device_Minor),
	))
}

func pax_fallback_name(name PAX_Header_Name) (fallback PAX_Fallback_Name) {
	defer func() {
		PAX_Fallback_Name_Invariants(fallback, "pax_fallback_name.fallback")
	}()
	PAX_Header_Name_Invariants(name, "pax_fallback_name.name")
	start := 0
	for position, character := range name {
		if character == '/' {
			start = position + 1
		}
	}
	if len(name)-start > NAME_FIELD_SIZE {
		start = len(name) - NAME_FIELD_SIZE
	}
	return PAX_Fallback_Name(name[start:])
}

func pax_write_bytes(
	destination PAX_Text_Record_Destination,
	key PAX_Text_Key, value PAX_Text_Value,
) (count PAX_Text_Record_Size) {
	defer func() {
		PAX_Text_Record_Size_Invariants(count, "pax_write_bytes.count")
	}()
	PAX_Text_Record_Destination_Invariants(destination, "pax_write_bytes.destination")
	PAX_Text_Key_Invariants(key, "pax_write_bytes.key")
	PAX_Text_Value_Invariants(value, "pax_write_bytes.value")
	count = PAX_Text_Record_Size(pax_record_size(
		PAX_Record_Key_Size(len(key)), PAX_Record_Value_Size(len(value)),
	))
	count_size := decimal_value_size(Integer(count))
	position := Count(format_decimal(
		Decimal_Destination(destination[:count_size]), Integer(count),
	))
	destination[position] = ' '
	position += Count(PAX_COUNT_SEPARATOR_SIZE)
	position += Count(copy(destination[position:], key))
	destination[position] = '='
	position += Count(PAX_KEY_SEPARATOR_SIZE)
	position += Count(copy(destination[position:], value))
	destination[position] = '\n'
	return count
}

func pax_write_decimal(
	destination PAX_Decimal_Record_Destination,
	key PAX_Decimal_Key, value Integer,
) (count PAX_Decimal_Record_Size) {
	defer func() {
		PAX_Decimal_Record_Size_Invariants(count, "pax_write_decimal.count")
	}()
	PAX_Decimal_Record_Destination_Invariants(
		destination, "pax_write_decimal.destination",
	)
	PAX_Decimal_Key_Invariants(key, "pax_write_decimal.key")
	Integer_Invariants(value, "pax_write_decimal.value")
	value_size := decimal_value_size(value)
	count = PAX_Decimal_Record_Size(pax_record_size(
		PAX_Record_Key_Size(len(key)), PAX_Record_Value_Size(value_size),
	))
	count_size := decimal_value_size(Integer(count))
	position := Count(format_decimal(
		Decimal_Destination(destination[:count_size]), Integer(count),
	))
	destination[position] = ' '
	position += Count(PAX_COUNT_SEPARATOR_SIZE)
	position += Count(copy(destination[position:], key))
	destination[position] = '='
	position += Count(PAX_KEY_SEPARATOR_SIZE)
	position += Count(format_decimal(
		Decimal_Destination(destination[position:position+Count(value_size)]), value,
	))
	destination[position] = '\n'
	return count
}

func pax_write_timestamp(
	destination PAX_Timestamp_Record_Destination, key PAX_Time_Key,
	timestamp Present_Timestamp,
) (count PAX_Timestamp_Record_Size) {
	defer func() {
		PAX_Timestamp_Record_Size_Invariants(count, "pax_write_timestamp.count")
	}()
	PAX_Timestamp_Record_Destination_Invariants(
		destination, "pax_write_timestamp.destination",
	)
	PAX_Time_Key_Invariants(key, "pax_write_timestamp.key")
	Present_Timestamp_Invariants(timestamp, "pax_write_timestamp.timestamp")
	value_size := timestamp_value_size(timestamp.Seconds, timestamp.Nanoseconds)
	count = PAX_Timestamp_Record_Size(pax_record_size(
		PAX_Record_Key_Size(len(key)), PAX_Record_Value_Size(value_size),
	))
	count_size := decimal_value_size(Integer(count))
	position := Count(format_decimal(
		Decimal_Destination(destination[:count_size]), Integer(count),
	))
	destination[position] = ' '
	position += Count(PAX_COUNT_SEPARATOR_SIZE)
	position += Count(copy(destination[position:], key))
	destination[position] = '='
	position += Count(PAX_KEY_SEPARATOR_SIZE)
	position += Count(format_timestamp(
		Timestamp_Destination(destination[position:position+Count(value_size)]),
		timestamp.Seconds, timestamp.Nanoseconds,
	))
	destination[position] = '\n'
	return count
}

func timestamp_value_size(
	seconds_value Integer, nanoseconds_value Nanosecond_Count,
) (size Timestamp_Value_Size) {
	defer func() {
		Timestamp_Value_Size_Invariants(size, "timestamp_value_size.size")
	}()
	Integer_Invariants(seconds_value, "timestamp_value_size.seconds")
	Nanosecond_Count_Invariants(nanoseconds_value, "timestamp_value_size.nanoseconds")
	seconds, nanoseconds, negative := pax_timestamp_parts(
		seconds_value, nanoseconds_value,
	)
	size = Timestamp_Value_Size(unsigned_decimal_value_size(seconds))
	if negative {
		size += Timestamp_Value_Size(PAX_COUNT_SEPARATOR_SIZE)
	}
	if nanoseconds != 0 {
		size += Timestamp_Value_Size(PAX_KEY_SEPARATOR_SIZE) +
			Timestamp_Value_Size(fractional_value_size(
				Fractional_Nanoseconds(nanoseconds),
			))
	}
	return size
}

func format_timestamp(
	destination Timestamp_Destination,
	seconds_value Integer,
	nanoseconds_value Nanosecond_Count,
) (count Timestamp_Value_Size) {
	defer func() {
		Timestamp_Value_Size_Invariants(count, "format_timestamp.count")
	}()
	Timestamp_Destination_Invariants(destination, "format_timestamp.destination")
	Integer_Invariants(seconds_value, "format_timestamp.seconds")
	Nanosecond_Count_Invariants(nanoseconds_value, "format_timestamp.nanoseconds")
	seconds, nanoseconds, negative := pax_timestamp_parts(
		seconds_value, nanoseconds_value,
	)
	if negative {
		destination[count] = '-'
		count += Timestamp_Value_Size(PAX_COUNT_SEPARATOR_SIZE)
	}
	seconds_size := unsigned_decimal_value_size(seconds)
	count += Timestamp_Value_Size(format_unsigned_decimal(
		Unsigned_Decimal_Destination(
			destination[count:count+Timestamp_Value_Size(seconds_size)],
		),
		seconds,
	))
	if nanoseconds == 0 {
		return count
	}
	destination[count] = '.'
	count += Timestamp_Value_Size(PAX_KEY_SEPARATOR_SIZE)
	fraction_size := fractional_value_size(Fractional_Nanoseconds(nanoseconds))
	divisor := Nanosecond_Count(TIMESTAMP_FIRST_FRACTION_DIVISOR)
	for position := Fraction_Digit_Count(0); position < fraction_size; position++ {
		destination[count] = byte(nanoseconds/divisor) + '0'
		nanoseconds %= divisor
		divisor /= DECIMAL_BASE
		count += Timestamp_Value_Size(PAX_RECORD_TERMINATOR_SIZE)
	}
	return count
}

func pax_timestamp_parts(
	seconds_value Integer,
	nanoseconds_value Nanosecond_Count,
) (
	seconds Decimal_Magnitude,
	nanoseconds Nanosecond_Count,
	negative bytes.Boolean,
) {
	defer func() {
		Decimal_Magnitude_Invariants(seconds, "pax_timestamp_parts.seconds")
		Nanosecond_Count_Invariants(nanoseconds, "pax_timestamp_parts.nanoseconds")
		bytes.Boolean_Invariants(negative, "pax_timestamp_parts.negative")
	}()
	Integer_Invariants(seconds_value, "pax_timestamp_parts.seconds_value")
	Nanosecond_Count_Invariants(nanoseconds_value, "pax_timestamp_parts.nanoseconds_value")
	nanoseconds = nanoseconds_value
	if seconds_value >= 0 {
		return Decimal_Magnitude(seconds_value), nanoseconds, false
	}
	negative = true
	seconds = Decimal_Magnitude(-(seconds_value + 1))
	if nanoseconds != 0 {
		nanoseconds = Nanosecond_Count(TIMESTAMP_NANOSECOND_COUNT) - nanoseconds
	} else {
		seconds++
	}
	return seconds, nanoseconds, negative
}

func fractional_value_size(
	nanoseconds Fractional_Nanoseconds,
) (size Fraction_Digit_Count) {
	defer func() {
		Fraction_Digit_Count_Invariants(size, "fractional_value_size.size")
	}()
	Fractional_Nanoseconds_Invariants(nanoseconds, "fractional_value_size.nanoseconds")
	size = TIMESTAMP_FRACTION_DIGIT_COUNT_MAXIMUM
	for nanoseconds%DECIMAL_BASE == 0 {
		nanoseconds /= DECIMAL_BASE
		size--
	}
	return size
}

func decimal_value_size(value Integer) (size Decimal_Value_Size) {
	defer func() {
		Decimal_Value_Size_Invariants(size, "decimal_value_size.size")
	}()
	Integer_Invariants(value, "decimal_value_size.value")
	magnitude := Decimal_Magnitude(value)
	if value < 0 {
		size++
		magnitude = Decimal_Magnitude(-(value + 1)) + 1
	}
	return size + Decimal_Value_Size(unsigned_decimal_value_size(magnitude))
}

func unsigned_decimal_value_size(
	value Decimal_Magnitude,
) (size Decimal_Magnitude_Size) {
	defer func() {
		Decimal_Magnitude_Size_Invariants(size, "unsigned_decimal_value_size.size")
	}()
	Decimal_Magnitude_Invariants(value, "unsigned_decimal_value_size.value")
	size++
	for value >= DECIMAL_BASE {
		value /= DECIMAL_BASE
		size++
	}
	return size
}

func format_decimal(
	destination Decimal_Destination, value Integer,
) (count Decimal_Value_Size) {
	defer func() {
		Decimal_Value_Size_Invariants(count, "format_decimal.count")
	}()
	Decimal_Destination_Invariants(destination, "format_decimal.destination")
	Integer_Invariants(value, "format_decimal.value")
	if value < 0 {
		destination[count] = '-'
		count++
		unsigned := Decimal_Magnitude(-(value + 1)) + 1
		unsigned_size := unsigned_decimal_value_size(unsigned)
		return count + Decimal_Value_Size(
			format_unsigned_decimal(
				Unsigned_Decimal_Destination(
					destination[count:count+Decimal_Value_Size(unsigned_size)],
				),
				unsigned,
			),
		)
	}
	unsigned := Decimal_Magnitude(value)
	unsigned_size := unsigned_decimal_value_size(unsigned)
	return Decimal_Value_Size(
		format_unsigned_decimal(
			Unsigned_Decimal_Destination(destination[:unsigned_size]), unsigned,
		),
	)
}

func format_unsigned_decimal(
	destination Unsigned_Decimal_Destination, value Decimal_Magnitude,
) (count Decimal_Magnitude_Size) {
	defer func() {
		Decimal_Magnitude_Size_Invariants(count, "format_unsigned_decimal.count")
	}()
	Unsigned_Decimal_Destination_Invariants(
		destination, "format_unsigned_decimal.destination",
	)
	Decimal_Magnitude_Invariants(value, "format_unsigned_decimal.value")
	divisor := Decimal_Magnitude(1)
	for divisor <= value/DECIMAL_BASE {
		divisor *= DECIMAL_BASE
	}
	for divisor != 0 {
		destination[count] = byte(value/divisor) + '0'
		value %= divisor
		divisor /= DECIMAL_BASE
		count++
	}
	return count
}

func write_gnu_headers(
	destination GNU_Header_Destination, position Header_Start_Position,
	header GNU_Header,
) (end GNU_Header_End_Position) {
	defer func() {
		GNU_Header_End_Position_Invariants(end, "write_gnu_headers.end")
	}()
	GNU_Header_Destination_Invariants(destination, "write_gnu_headers.destination")
	Header_Start_Position_Invariants(position, "write_gnu_headers.position")
	GNU_Header_Invariants(header, "write_gnu_headers.header")
	cursor := int(position)
	if len(header.Name) > NAME_FIELD_SIZE {
		cursor = int(write_gnu_long_field(
			GNU_Long_Destination(destination), GNU_Long_Position(cursor),
			GNU_Extended_Field(header.Name),
			GNU_Long_Type_Flag(TYPE_GNU_LONG_NAME),
		))
	}
	if len(header.Link_Name) > NAME_FIELD_SIZE {
		cursor = int(write_gnu_long_field(
			GNU_Long_Destination(destination), GNU_Long_Position(cursor),
			GNU_Extended_Field(header.Link_Name),
			GNU_Long_Type_Flag(TYPE_GNU_LONG_LINK),
		))
	}
	block := destination[cursor : cursor+BLOCK_SIZE]
	zero(Zero_Destination(block))
	write_gnu_main_header(Header_Block(block), header)
	return GNU_Header_End_Position(cursor + BLOCK_SIZE)
}

func write_gnu_long_field(
	destination GNU_Long_Destination, position GNU_Long_Position,
	value GNU_Extended_Field, type_flag GNU_Long_Type_Flag,
) (end GNU_Long_End_Position) {
	defer func() {
		GNU_Long_End_Position_Invariants(end, "write_gnu_long_field.end")
	}()
	GNU_Long_Destination_Invariants(destination, "write_gnu_long_field.destination")
	GNU_Long_Position_Invariants(position, "write_gnu_long_field.position")
	GNU_Extended_Field_Invariants(value, "write_gnu_long_field.value")
	GNU_Long_Type_Flag_Invariants(type_flag, "write_gnu_long_field.type_flag")
	block := destination[position : position+BLOCK_SIZE]
	zero(Zero_Destination(block))
	write_metadata_header(
		Header_Block(block), Metadata_Type_Flag(type_flag),
		Metadata_Size(len(value)+GNU_LONG_FIELD_TERMINATOR_SIZE),
	)
	position += BLOCK_SIZE
	copy(destination[position:], value)
	position += GNU_Long_Position(len(value))
	destination[position] = 0
	position++
	padding := Count(block_padding(
		Unpadded_Size(len(value) + GNU_LONG_FIELD_TERMINATOR_SIZE),
	))
	end = GNU_Long_End_Position(position) + GNU_Long_End_Position(padding)
	zero(Zero_Destination(destination[position:end]))
	return end
}

func write_gnu_main_header(block Header_Block, header GNU_Header) {
	Header_Block_Invariants(block, "write_gnu_main_header.block")
	GNU_Header_Invariants(header, "write_gnu_main_header.header")
	copy(block[NAME_FIELD_OFFSET:NAME_FIELD_OFFSET+NAME_FIELD_SIZE], header.Name)
	write_gnu_numeric_fields(
		block, header.Mode, header.User_Identifier, header.Group_Identifier,
		header.Size, header.Modification_Time,
	)
	checksum_end_offset := CHECKSUM_FIELD_OFFSET + CHECKSUM_FIELD_SIZE
	for position := CHECKSUM_FIELD_OFFSET; position < checksum_end_offset; position++ {
		block[position] = ' '
	}
	type_flag := header.Type_Flag
	if type_flag == TYPE_REGULAR_LEGACY {
		type_flag = TYPE_REGULAR
	}
	block[TYPE_FLAG_FIELD_OFFSET] = byte(type_flag)
	copy(
		block[LINK_NAME_FIELD_OFFSET:LINK_NAME_FIELD_OFFSET+LINK_NAME_FIELD_SIZE],
		header.Link_Name,
	)
	copy(block[MAGIC_FIELD_OFFSET:MAGIC_FIELD_OFFSET+MAGIC_FIELD_SIZE], GNU_MAGIC)
	copy(
		block[VERSION_FIELD_OFFSET:VERSION_FIELD_OFFSET+VERSION_FIELD_SIZE],
		GNU_VERSION,
	)
	copy(
		block[USER_NAME_FIELD_OFFSET:USER_NAME_FIELD_OFFSET+USER_NAME_FIELD_SIZE],
		header.User_Name,
	)
	copy(
		block[GROUP_NAME_FIELD_OFFSET:GROUP_NAME_FIELD_OFFSET+GROUP_NAME_FIELD_SIZE],
		header.Group_Name,
	)
	format_numeric(
		Numeric_Destination(block[DEVICE_MAJOR_FIELD_OFFSET:][:DEVICE_MAJOR_FIELD_SIZE]),
		Integer(header.Device_Major),
	)
	format_numeric(
		Numeric_Destination(block[DEVICE_MINOR_FIELD_OFFSET:][:DEVICE_MINOR_FIELD_SIZE]),
		Integer(header.Device_Minor),
	)
	write_gnu_time_fields(block, header.Access_Time, header.Change_Time)
	write_checksum(block)
}

func write_gnu_numeric_fields(
	block Header_Block,
	mode Wire_File_Mode,
	user_identifier Wire_User_Identifier,
	group_identifier Wire_Group_Identifier,
	size Entry_Size,
	modified GNU_Modification_Time,
) {
	Header_Block_Invariants(block, "write_gnu_numeric_fields.block")
	Wire_File_Mode_Invariants(mode, "write_gnu_numeric_fields.mode")
	Wire_User_Identifier_Invariants(
		user_identifier, "write_gnu_numeric_fields.user_identifier",
	)
	Wire_Group_Identifier_Invariants(
		group_identifier, "write_gnu_numeric_fields.group_identifier",
	)
	Entry_Size_Invariants(size, "write_gnu_numeric_fields.size")
	GNU_Modification_Time_Invariants(modified, "write_gnu_numeric_fields.modified")
	format_numeric(
		Numeric_Destination(block[MODE_FIELD_OFFSET:MODE_FIELD_OFFSET+MODE_FIELD_SIZE]),
		Integer(mode),
	)
	format_numeric(
		Numeric_Destination(
			block[USER_IDENTIFIER_FIELD_OFFSET:][:USER_IDENTIFIER_FIELD_SIZE],
		),
		Integer(user_identifier),
	)
	format_numeric(
		Numeric_Destination(
			block[GROUP_IDENTIFIER_FIELD_OFFSET:][:GROUP_IDENTIFIER_FIELD_SIZE],
		),
		Integer(group_identifier),
	)
	format_numeric(
		Numeric_Destination(block[ENTRY_SIZE_FIELD_OFFSET:][:ENTRY_SIZE_FIELD_SIZE]),
		Integer(size),
	)
	seconds := Integer(0)
	if modified.Set {
		seconds = modified.Seconds
	}
	format_numeric(
		Numeric_Destination(
			block[TIMESTAMP_FIELD_OFFSET:TIMESTAMP_FIELD_OFFSET+TIMESTAMP_FIELD_SIZE],
		),
		seconds,
	)
}

func write_gnu_time_fields(
	block Header_Block, accessed Wire_Access_Time, changed Wire_Change_Time,
) {
	Header_Block_Invariants(block, "write_gnu_time_fields.block")
	Wire_Access_Time_Invariants(accessed, "write_gnu_time_fields.accessed")
	Wire_Change_Time_Invariants(changed, "write_gnu_time_fields.changed")
	if accessed.Set {
		format_numeric(
			Numeric_Destination(
				block[GNU_ACCESS_TIMESTAMP_FIELD_OFFSET:][:TIMESTAMP_FIELD_SIZE],
			),
			Integer(accessed.Seconds),
		)
	}
	if changed.Set {
		format_numeric(
			Numeric_Destination(
				block[GNU_CHANGE_TIMESTAMP_FIELD_OFFSET:][:TIMESTAMP_FIELD_SIZE],
			),
			Integer(changed.Seconds),
		)
	}
}

func write_metadata_header(
	block Header_Block,
	type_flag Metadata_Type_Flag,
	size Metadata_Size,
) {
	Header_Block_Invariants(block, "write_metadata_header.block")
	Metadata_Type_Flag_Invariants(type_flag, "write_metadata_header.type_flag")
	Metadata_Size_Invariants(size, "write_metadata_header.size")
	name := GNU_LONG_HEADER_NAME
	if type_flag == Metadata_Type_Flag(TYPE_PAX_LOCAL) {
		name = PAX_HEADER_NAME
	}
	copy(block[NAME_FIELD_OFFSET:NAME_FIELD_OFFSET+NAME_FIELD_SIZE], name)
	format_octal(
		Numeric_Destination(block[MODE_FIELD_OFFSET:MODE_FIELD_OFFSET+MODE_FIELD_SIZE]),
		0,
	)
	format_octal(
		Numeric_Destination(
			block[USER_IDENTIFIER_FIELD_OFFSET:][:USER_IDENTIFIER_FIELD_SIZE],
		),
		0,
	)
	format_octal(
		Numeric_Destination(
			block[GROUP_IDENTIFIER_FIELD_OFFSET:][:GROUP_IDENTIFIER_FIELD_SIZE],
		),
		0,
	)
	format_numeric(
		Numeric_Destination(block[ENTRY_SIZE_FIELD_OFFSET:][:ENTRY_SIZE_FIELD_SIZE]),
		Integer(size),
	)
	format_octal(
		Numeric_Destination(
			block[TIMESTAMP_FIELD_OFFSET:TIMESTAMP_FIELD_OFFSET+TIMESTAMP_FIELD_SIZE],
		),
		0,
	)
	checksum_end_offset := CHECKSUM_FIELD_OFFSET + CHECKSUM_FIELD_SIZE
	for position := CHECKSUM_FIELD_OFFSET; position < checksum_end_offset; position++ {
		block[position] = ' '
	}
	block[TYPE_FLAG_FIELD_OFFSET] = byte(type_flag)
	if type_flag != Metadata_Type_Flag(TYPE_PAX_LOCAL) {
		copy(
			block[MAGIC_FIELD_OFFSET:MAGIC_FIELD_OFFSET+MAGIC_FIELD_SIZE],
			GNU_MAGIC,
		)
		copy(
			block[VERSION_FIELD_OFFSET:VERSION_FIELD_OFFSET+VERSION_FIELD_SIZE],
			GNU_VERSION,
		)
	} else {
		copy(
			block[MAGIC_FIELD_OFFSET:MAGIC_FIELD_OFFSET+MAGIC_FIELD_SIZE],
			USTAR_MAGIC,
		)
		copy(
			block[VERSION_FIELD_OFFSET:VERSION_FIELD_OFFSET+VERSION_FIELD_SIZE],
			USTAR_VERSION,
		)
	}
	write_checksum(block)
}

func format_numeric(field Numeric_Destination, value Integer) {
	Numeric_Destination_Invariants(field, "format_numeric.field")
	Integer_Invariants(value, "format_numeric.value")
	if octal_fits(Numeric_Field_Size(len(field)), value) {
		format_octal(field, Octal_Value(value))
		return
	}
	for position := len(field) - 1; position >= 0; position-- {
		field[position] = byte(value)
		value >>= bits.BIT_COUNT_8_MAXIMUM
	}
	field[0] |= BASE_256_MARKER_MASK
}

func base_256_fits(
	field_size Numeric_Field_Size, value Integer,
) (fits bytes.Boolean) {
	defer func() { bytes.Boolean_Invariants(fits, "base_256_fits.fits") }()
	Numeric_Field_Size_Invariants(field_size, "base_256_fits.field_size")
	Integer_Invariants(value, "base_256_fits.value")
	if field_size >= MODE_FIELD_SIZE+TYPE_FLAG_FIELD_SIZE {
		return true
	}
	payload_bits := uint(field_size-TYPE_FLAG_FIELD_SIZE) * bits.BIT_COUNT_8_MAXIMUM
	if value < -1<<payload_bits {
		return false
	}
	return value < 1<<payload_bits
}

func valid_utf8(value UTF8_Text) (valid bytes.Boolean) {
	defer func() { bytes.Boolean_Invariants(valid, "valid_utf8.valid") }()
	UTF8_Text_Invariants(value, "valid_utf8.value")
	for position := 0; position < len(value); {
		character := value[position]
		if character < byte(utf8.CHARACTER_SELF) {
			position++
			continue
		}
		width := 0
		minimum := uint32(0)
		decoded := uint32(0)
		switch {
		case character < UTF8_FIRST_TWO_MINIMUM:
			return false
		case character <= UTF8_FIRST_TWO_MAXIMUM:
			width = utf8.CHARACTER_SIZE_TWO
			minimum = uint32(utf8.CHARACTER_ONE_MAXIMUM + 1)
			decoded = uint32(character & byte(utf8.FIRST_MASK_TWO))
		case character < UTF8_FIRST_THREE_MINIMUM:
			return false
		case character <= UTF8_FIRST_THREE_MAXIMUM:
			width = utf8.CHARACTER_SIZE_THREE
			minimum = uint32(utf8.CHARACTER_TWO_MAXIMUM + 1)
			decoded = uint32(character & byte(utf8.FIRST_MASK_THREE))
		case character < UTF8_FIRST_FOUR_MINIMUM:
			return false
		case character <= UTF8_FIRST_FOUR_MAXIMUM:
			width = utf8.CHARACTER_SIZE_MAXIMUM
			minimum = uint32(utf8.CHARACTER_THREE_MAXIMUM + 1)
			decoded = uint32(character & byte(utf8.FIRST_MASK_FOUR))
		default:
			return false
		}
		if !utf8_available(
			UTF8_Position(position), UTF8_Character_Size(width),
			UTF8_Boundary(len(value)),
		) {
			return false
		}
		for offset := utf8.CHARACTER_SIZE_MINIMUM; offset < width; offset++ {
			continuation := value[position+offset]
			if continuation < byte(utf8.CONTINUATION_MINIMUM) {
				return false
			}
			if continuation > byte(utf8.CONTINUATION_MAXIMUM) {
				return false
			}
			decoded = decoded<<utf8.CONTINUATION_PAYLOAD_BIT_COUNT |
				uint32(continuation&byte(utf8.CONTINUATION_MASK))
		}
		if decoded < minimum {
			return false
		}
		if decoded > uint32(utf8.RUNE_MAX) {
			return false
		}
		if decoded >= uint32(utf8.SURROGATE_MINIMUM) {
			if decoded <= uint32(utf8.SURROGATE_MAXIMUM) {
				return false
			}
		}
		position += width
	}
	return true
}

func write_basic_header(block Header_Block, header Basic_Header) {
	Header_Block_Invariants(block, "write_basic_header.block")
	Basic_Header_Invariants(header, "write_basic_header.header")
	prefix_count, suffix_start := ustar_name_split(Writer_Name(header.Name))
	copy(
		block[NAME_FIELD_OFFSET:NAME_FIELD_OFFSET+NAME_FIELD_SIZE],
		header.Name[suffix_start:],
	)
	write_octal_numeric_fields(
		block, header.Mode, header.User_Identifier, header.Group_Identifier,
		header.Size, header.Modification_Seconds,
	)
	checksum_end_offset := CHECKSUM_FIELD_OFFSET + CHECKSUM_FIELD_SIZE
	for position := CHECKSUM_FIELD_OFFSET; position < checksum_end_offset; position++ {
		block[position] = ' '
	}
	type_flag := header.Type_Flag
	if type_flag == TYPE_REGULAR_LEGACY {
		if len(header.Name) != 0 {
			if header.Name[len(header.Name)-1] == '/' {
				type_flag = TYPE_DIRECTORY
			} else {
				type_flag = TYPE_REGULAR
			}
		} else {
			type_flag = TYPE_REGULAR
		}
	}
	block[TYPE_FLAG_FIELD_OFFSET] = byte(type_flag)
	copy(
		block[LINK_NAME_FIELD_OFFSET:LINK_NAME_FIELD_OFFSET+LINK_NAME_FIELD_SIZE],
		header.Link_Name,
	)
	ustar_fields := USTAR_Fields{
		Name: USTAR_Name(header.Name), User_Name: USTAR_User_Name(header.User_Name),
		Group_Name:   USTAR_Group_Name(header.Group_Name),
		Device_Major: USTAR_Device_Major(header.Device_Major),
		Device_Minor: USTAR_Device_Minor(header.Device_Minor),
	}
	if Format(header.Format) == FORMAT_USTAR {
		write_ustar_fields(block, ustar_fields, USTAR_Prefix_Count(prefix_count))
	} else if Format(header.Format) == FORMAT_PAX {
		write_ustar_fields(block, ustar_fields, USTAR_Prefix_Count(prefix_count))
	}
	write_checksum(block)
}

func write_octal_numeric_fields(
	block Header_Block,
	mode Basic_File_Mode,
	user_identifier Basic_User_Identifier,
	group_identifier Basic_Group_Identifier,
	size Entry_Size,
	modified Basic_Modification_Seconds,
) {
	Header_Block_Invariants(block, "write_octal_numeric_fields.block")
	Basic_File_Mode_Invariants(mode, "write_octal_numeric_fields.mode")
	Basic_User_Identifier_Invariants(
		user_identifier, "write_octal_numeric_fields.user_identifier",
	)
	Basic_Group_Identifier_Invariants(
		group_identifier, "write_octal_numeric_fields.group_identifier",
	)
	Entry_Size_Invariants(size, "write_octal_numeric_fields.size")
	Basic_Modification_Seconds_Invariants(
		modified, "write_octal_numeric_fields.modified",
	)
	format_octal(
		Numeric_Destination(block[MODE_FIELD_OFFSET:MODE_FIELD_OFFSET+MODE_FIELD_SIZE]),
		Octal_Value(mode),
	)
	format_octal(
		Numeric_Destination(
			block[USER_IDENTIFIER_FIELD_OFFSET:][:USER_IDENTIFIER_FIELD_SIZE],
		),
		Octal_Value(user_identifier),
	)
	format_octal(
		Numeric_Destination(
			block[GROUP_IDENTIFIER_FIELD_OFFSET:][:GROUP_IDENTIFIER_FIELD_SIZE],
		),
		Octal_Value(group_identifier),
	)
	format_octal(
		Numeric_Destination(block[ENTRY_SIZE_FIELD_OFFSET:][:ENTRY_SIZE_FIELD_SIZE]),
		Octal_Value(size),
	)
	format_octal(
		Numeric_Destination(
			block[TIMESTAMP_FIELD_OFFSET:TIMESTAMP_FIELD_OFFSET+TIMESTAMP_FIELD_SIZE],
		),
		Octal_Value(modified),
	)
}

func write_ustar_fields(
	block Header_Block, fields USTAR_Fields, prefix_count USTAR_Prefix_Count,
) {
	Header_Block_Invariants(block, "write_ustar_fields.block")
	USTAR_Fields_Invariants(fields, "write_ustar_fields.fields")
	USTAR_Prefix_Count_Invariants(prefix_count, "write_ustar_fields.prefix_count")
	copy(
		block[MAGIC_FIELD_OFFSET:MAGIC_FIELD_OFFSET+MAGIC_FIELD_SIZE],
		USTAR_MAGIC,
	)
	copy(
		block[VERSION_FIELD_OFFSET:VERSION_FIELD_OFFSET+VERSION_FIELD_SIZE],
		USTAR_VERSION,
	)
	copy(
		block[USER_NAME_FIELD_OFFSET:USER_NAME_FIELD_OFFSET+USER_NAME_FIELD_SIZE],
		fields.User_Name,
	)
	copy(
		block[GROUP_NAME_FIELD_OFFSET:GROUP_NAME_FIELD_OFFSET+GROUP_NAME_FIELD_SIZE],
		fields.Group_Name,
	)
	format_octal(
		Numeric_Destination(block[DEVICE_MAJOR_FIELD_OFFSET:][:DEVICE_MAJOR_FIELD_SIZE]),
		Octal_Value(fields.Device_Major),
	)
	format_octal(
		Numeric_Destination(block[DEVICE_MINOR_FIELD_OFFSET:][:DEVICE_MINOR_FIELD_SIZE]),
		Octal_Value(fields.Device_Minor),
	)
	copy(
		block[PREFIX_FIELD_OFFSET:PREFIX_FIELD_OFFSET+PREFIX_FIELD_SIZE],
		fields.Name[:prefix_count],
	)
}

func format_octal(field Numeric_Destination, value Octal_Value) {
	Numeric_Destination_Invariants(field, "format_octal.field")
	Octal_Value_Invariants(value, "format_octal.value")
	for position := range field {
		field[position] = 0
	}
	position := len(field) - VERSION_FIELD_SIZE
	for position >= 0 {
		field[position] = byte(value&(OCTAL_BASE-1)) + '0'
		value >>= OCTAL_BITS_PER_DIGIT
		position--
	}
}

func write_checksum(block Header_Block) {
	Header_Block_Invariants(block, "write_checksum.block")
	unsigned, _ := header_checksum(block)
	field := Numeric_Destination(
		block[CHECKSUM_FIELD_OFFSET : CHECKSUM_FIELD_OFFSET+CHECKSUM_FIELD_SIZE],
	)
	format_octal(field, Octal_Value(unsigned))
	field[len(field)-TYPE_FLAG_FIELD_SIZE] = ' '
}

func octal_fits(
	field_size Numeric_Field_Size, value Integer,
) (fits bytes.Boolean) {
	defer func() { bytes.Boolean_Invariants(fits, "octal_fits.fits") }()
	Numeric_Field_Size_Invariants(field_size, "octal_fits.field_size")
	Integer_Invariants(value, "octal_fits.value")
	if value < 0 {
		return false
	}
	payload_bits := uint(field_size-TYPE_FLAG_FIELD_SIZE) * OCTAL_BITS_PER_DIGIT
	if payload_bits >= INTEGER_SIGN_SHIFT {
		return true
	}
	return value < 1<<payload_bits
}

func ustar_name_fits(name Writer_Name) (fits bytes.Boolean) {
	defer func() { bytes.Boolean_Invariants(fits, "ustar_name_fits.fits") }()
	Writer_Name_Invariants(name, "ustar_name_fits.name")
	if len(name) <= NAME_FIELD_SIZE {
		return true
	}
	prefix, suffix := ustar_name_split(name)
	if prefix == 0 {
		return false
	}
	if prefix > PREFIX_FIELD_SIZE {
		return false
	}
	return USTAR_Suffix_Start(len(name))-suffix <= NAME_FIELD_SIZE
}

func ustar_name_split(
	name Writer_Name,
) (prefix_count USTAR_Prefix_Count, suffix_start USTAR_Suffix_Start) {
	defer func() {
		USTAR_Prefix_Count_Invariants(prefix_count, "ustar_name_split.prefix_count")
		USTAR_Suffix_Start_Invariants(suffix_start, "ustar_name_split.suffix_start")
	}()
	Writer_Name_Invariants(name, "ustar_name_split.name")
	if len(name) <= NAME_FIELD_SIZE {
		return 0, 0
	}
	minimum := len(name) - NAME_FIELD_SIZE - TYPE_FLAG_FIELD_SIZE
	maximum := PREFIX_FIELD_SIZE
	last := len(name) - TYPE_FLAG_FIELD_SIZE
	if maximum > last {
		maximum = last
	}
	for position := maximum; position >= minimum; position-- {
		if name[position] == '/' {
			return USTAR_Prefix_Count(position), USTAR_Suffix_Start(position + 1)
		}
	}
	return 0, 0
}

func field_storage_fits(
	fields PAX_Text_Fields, storage Header_Storage,
) (fits bytes.Boolean) {
	defer func() {
		bytes.Boolean_Invariants(fits, "field_storage_fits.fits")
	}()
	PAX_Text_Fields_Invariants(fields, "field_storage_fits.fields")
	Header_Storage_Invariants(storage, "field_storage_fits.storage")
	if field_size(fields.Name) > Field_Size(len(storage.Name)) {
		return false
	}
	if Count(len(fields.Link_Name)) > Count(len(storage.Link_Name)) {
		return false
	}
	if Count(len(fields.User_Name)) > Count(len(storage.User_Name)) {
		return false
	}
	return Count(len(fields.Group_Name)) <= Count(len(storage.Group_Name))
}

func header_storage_valid(storage Header_Storage) (valid bytes.Boolean) {
	defer func() {
		bytes.Boolean_Invariants(valid, "header_storage_valid.valid")
	}()
	Header_Storage_Invariants(storage, "header_storage_valid.storage")
	if len(storage.Name) > SPECIAL_FILE_SIZE_MAXIMUM {
		return false
	}
	if len(storage.Link_Name) > SPECIAL_FILE_SIZE_MAXIMUM {
		return false
	}
	if len(storage.User_Name) > HEADER_USER_NAME_SIZE_MAXIMUM {
		return false
	}
	if len(storage.Group_Name) > HEADER_GROUP_NAME_SIZE_MAXIMUM {
		return false
	}
	if slices_overlap(Overlap_Left(storage.Name), Overlap_Right(storage.Link_Name)) {
		return false
	}
	if slices_overlap(Overlap_Left(storage.Name), Overlap_Right(storage.User_Name)) {
		return false
	}
	if slices_overlap(Overlap_Left(storage.Name), Overlap_Right(storage.Group_Name)) {
		return false
	}
	if slices_overlap(
		Overlap_Left(storage.Link_Name), Overlap_Right(storage.User_Name),
	) {
		return false
	}
	if slices_overlap(
		Overlap_Left(storage.Link_Name), Overlap_Right(storage.Group_Name),
	) {
		return false
	}
	return !slices_overlap(
		Overlap_Left(storage.User_Name), Overlap_Right(storage.Group_Name),
	)
}

func field_size(value Field_Value) (size Field_Size) {
	defer func() { Field_Size_Invariants(size, "field_size.size") }()
	Field_Value_Invariants(value, "field_size.value")
	size = Field_Size(len(value.Prefix) + len(value.Suffix))
	if value.Joined {
		size++
	}
	return size
}

func copy_field(
	destination Header_Name_Destination, value Field_Value,
) (field Header_Name) {
	defer func() { Header_Name_Invariants(field, "copy_field.field") }()
	Header_Name_Destination_Invariants(destination, "copy_field.destination")
	Field_Value_Invariants(value, "copy_field.value")
	position := copy(destination, value.Prefix)
	if value.Joined {
		destination[position] = '/'
		position++
	}
	position += copy(destination[position:], value.Suffix)
	return Header_Name(destination[:position])
}

func metadata_field_bytes(field Field) (value Field) {
	defer func() { Field_Invariants(value, "metadata_field_bytes.value") }()
	Field_Invariants(field, "metadata_field_bytes.field")
	end := 0
	for end < len(field) {
		if field[end] == 0 {
			break
		}
		end++
	}
	return field[:end]
}

func wire_field_bytes(field Wire_Text_Field) (value Wire_Text) {
	defer func() { Wire_Text_Invariants(value, "wire_field_bytes.value") }()
	Wire_Text_Field_Invariants(field, "wire_field_bytes.field")
	end := 0
	for end < len(field) {
		if field[end] == 0 {
			break
		}
		end++
	}
	return Wire_Text(field[:end])
}

func padded_block_count(
	size Metadata_Payload_Size,
) (count Metadata_Block_Count) {
	defer func() {
		Metadata_Block_Count_Invariants(count, "padded_block_count.count")
	}()
	Metadata_Payload_Size_Invariants(size, "padded_block_count.size")
	return Metadata_Block_Count(
		(int(size) + BLOCK_SIZE - TYPE_FLAG_FIELD_SIZE) / BLOCK_SIZE,
	)
}

func block_padding(size Unpadded_Size) (padding Block_Padding) {
	defer func() { Block_Padding_Invariants(padding, "block_padding.padding") }()
	Unpadded_Size_Invariants(size, "block_padding.size")
	return Block_Padding(-size & (BLOCK_SIZE - TYPE_FLAG_FIELD_SIZE))
}

func header_only_type(type_flag Type_Flag) (header_only bytes.Boolean) {
	defer func() {
		bytes.Boolean_Invariants(header_only, "header_only_type.header_only")
	}()
	Type_Flag_Invariants(type_flag, "header_only_type.type_flag")
	switch type_flag {
	case TYPE_LINK, TYPE_SYMBOLIC_LINK, TYPE_CHARACTER,
		TYPE_BLOCK, TYPE_DIRECTORY, TYPE_FIFO:
		return true
	default:
		return false
	}
}

func zero_block(block Header_Block) (empty bytes.Boolean) {
	defer func() { bytes.Boolean_Invariants(empty, "zero_block.empty") }()
	Header_Block_Invariants(block, "zero_block.block")
	for _, value := range block {
		if value != 0 {
			return false
		}
	}
	return true
}

func zero(destination Zero_Destination) {
	Zero_Destination_Invariants(destination, "zero.destination")
	for position := range destination {
		destination[position] = 0
	}
}

func metadata_available(
	position Metadata_Block_Position,
	size Metadata_Block_Count,
	boundary Metadata_Available_Boundary,
) (fits bytes.Boolean) {
	defer func() { bytes.Boolean_Invariants(fits, "metadata_available.fits") }()
	Metadata_Block_Position_Invariants(position, "metadata_available.position")
	Metadata_Block_Count_Invariants(size, "metadata_available.size")
	Metadata_Available_Boundary_Invariants(boundary, "metadata_available.boundary")
	position_count := int(position) * BLOCK_SIZE
	size_count := int(size) * BLOCK_SIZE
	boundary_count := int(boundary)
	if position_count > boundary_count {
		return false
	}
	return size_count <= boundary_count-position_count
}

func utf8_available(
	position UTF8_Position, size UTF8_Character_Size, boundary UTF8_Boundary,
) (fits bytes.Boolean) {
	defer func() { bytes.Boolean_Invariants(fits, "utf8_available.fits") }()
	UTF8_Position_Invariants(position, "utf8_available.position")
	UTF8_Character_Size_Invariants(size, "utf8_available.size")
	UTF8_Boundary_Invariants(boundary, "utf8_available.boundary")
	position_count := int(position)
	size_count := int(size)
	boundary_count := int(boundary)
	if position_count > boundary_count {
		return false
	}
	return size_count <= boundary_count-position_count
}

func contains_nul(value NUL_Checked_Text) (contains bytes.Boolean) {
	defer func() { bytes.Boolean_Invariants(contains, "contains_nul.contains") }()
	NUL_Checked_Text_Invariants(value, "contains_nul.value")
	for _, character := range value {
		if character == 0 {
			return true
		}
	}
	return false
}

func wire_literal_equal(
	block Header_Block, offset Wire_Literal_Offset, literal Wire_Literal,
) (equal bytes.Boolean) {
	defer func() { bytes.Boolean_Invariants(equal, "wire_literal_equal.equal") }()
	Header_Block_Invariants(block, "wire_literal_equal.block")
	Wire_Literal_Offset_Invariants(offset, "wire_literal_equal.offset")
	Wire_Literal_Invariants(literal, "wire_literal_equal.literal")
	start_position := int(offset)
	if len(literal) > len(block)-start_position {
		return false
	}
	for position := range literal {
		if block[start_position+position] != literal[position] {
			return false
		}
	}
	return true
}

func pax_key_equal(
	key PAX_Parsed_Key, literal PAX_Key_Name,
) (equal bytes.Boolean) {
	defer func() {
		bytes.Boolean_Invariants(equal, "pax_key_equal.equal")
	}()
	PAX_Parsed_Key_Invariants(key, "pax_key_equal.key")
	PAX_Key_Name_Invariants(literal, "pax_key_equal.literal")
	if len(key) != len(literal) {
		return false
	}
	for position := range key {
		if key[position] != literal[position] {
			return false
		}
	}
	return true
}

func pax_sparse_key(key PAX_Parsed_Key) (sparse bytes.Boolean) {
	defer func() {
		bytes.Boolean_Invariants(sparse, "pax_sparse_key.sparse")
	}()
	PAX_Parsed_Key_Invariants(key, "pax_sparse_key.key")
	if len(key) < len(PAX_GNU_SPARSE_KEY_PREFIX) {
		return false
	}
	for position := range PAX_GNU_SPARSE_KEY_PREFIX {
		if key[position] != PAX_GNU_SPARSE_KEY_PREFIX[position] {
			return false
		}
	}
	return true
}

func pax_string_key(key PAX_Parsed_Key) (string_key bytes.Boolean) {
	defer func() {
		bytes.Boolean_Invariants(string_key, "pax_string_key.string_key")
	}()
	PAX_Parsed_Key_Invariants(key, "pax_string_key.key")
	if pax_key_equal(key, PAX_PATH_KEY) {
		return true
	}
	if pax_key_equal(key, PAX_LINK_PATH_KEY) {
		return true
	}
	if pax_key_equal(key, PAX_USER_NAME_KEY) {
		return true
	}
	return pax_key_equal(key, PAX_GROUP_NAME_KEY)
}

func pax_basic_key(key PAX_Parsed_Key) (basic bytes.Boolean) {
	defer func() { bytes.Boolean_Invariants(basic, "pax_basic_key.basic") }()
	PAX_Parsed_Key_Invariants(key, "pax_basic_key.key")
	if pax_string_key(key) {
		return true
	}
	if pax_key_equal(key, PAX_SIZE_KEY) {
		return true
	}
	if pax_key_equal(key, PAX_USER_IDENTIFIER_KEY) {
		return true
	}
	if pax_key_equal(key, PAX_GROUP_IDENTIFIER_KEY) {
		return true
	}
	if pax_key_equal(key, PAX_MODIFICATION_TIME_KEY) {
		return true
	}
	if pax_key_equal(key, PAX_ACCESS_TIME_KEY) {
		return true
	}
	return pax_key_equal(key, PAX_CHANGE_TIME_KEY)
}

func ascii_bytes(value GNU_Prefix_Text) (ascii bytes.Boolean) {
	defer func() { bytes.Boolean_Invariants(ascii, "ascii_bytes.ascii") }()
	GNU_Prefix_Text_Invariants(value, "ascii_bytes.value")
	for _, character := range value {
		if character >= byte(utf8.CHARACTER_SELF) {
			return false
		}
	}
	return true
}

func slices_overlap(
	left Overlap_Left, right Overlap_Right,
) (overlap bytes.Boolean) {
	defer func() { bytes.Boolean_Invariants(overlap, "slices_overlap.overlap") }()
	Overlap_Left_Invariants(left, "slices_overlap.left")
	Overlap_Right_Invariants(right, "slices_overlap.right")
	if len(left) == 0 {
		return false
	}
	if len(right) == 0 {
		return false
	}
	for position := range left {
		if &left[position] == &right[0] {
			return true
		}
	}
	for position := range right {
		if &right[position] == &left[0] {
			return true
		}
	}
	return false
}
