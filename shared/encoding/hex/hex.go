// Package hex implements bounded hexadecimal encoding on caller-owned storage.
package hex

import (
	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/math/bits"
)

// NIBBLE_BIT_COUNT follows the two equal halves of one byte.
const NIBBLE_BIT_COUNT = bits.BIT_COUNT_8_MAXIMUM / 2

// ALPHABET_SIZE follows every value representable by one nibble.
const ALPHABET_SIZE = 1 << NIBBLE_BIT_COUNT

// ALPHABET_FINAL_INDEX masks one nibble without a detached hexadecimal literal.
const ALPHABET_FINAL_INDEX = ALPHABET_SIZE - 1

// ENCODED_BYTE_SIZE follows the nibbles required to represent one source byte.
const ENCODED_BYTE_SIZE = bits.BIT_COUNT_8_MAXIMUM / NIBBLE_BIT_COUNT

// ENCODED_BYTE_FINAL_OFFSET is the low-nibble distance inside one encoded byte.
const ENCODED_BYTE_FINAL_OFFSET = ENCODED_BYTE_SIZE - 1

// ENCODE_UNROLL_SOURCE_SIZE follows one machine-independent 32-bit word.
const ENCODE_UNROLL_SOURCE_SIZE = bits.BIT_COUNT_32_MAXIMUM / bits.BIT_COUNT_8_MAXIMUM

// ENCODE_UNROLL_DESTINATION_SIZE follows encoded bytes from one unrolled source word.
const ENCODE_UNROLL_DESTINATION_SIZE = ENCODE_UNROLL_SOURCE_SIZE * ENCODED_BYTE_SIZE

// SIZE_MINIMUM admits empty source and output.
const SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// COUNT_HOLE_FIRST excludes one byte from representations that require pairs or full lines.
const COUNT_HOLE_FIRST = SIZE_MINIMUM + 1

// COUNT_HOLE_SECOND excludes two bytes from representations that require a full dump line.
const COUNT_HOLE_SECOND = COUNT_HOLE_FIRST + 1

// ENCODED_SIZE_MAXIMUM follows the repository byte-slice boundary.
const ENCODED_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// SOURCE_SIZE_MAXIMUM is the largest source whose hexadecimal form stays bounded.
const SOURCE_SIZE_MAXIMUM = ENCODED_SIZE_MAXIMUM / ENCODED_BYTE_SIZE

// DECODED_SIZE_MAXIMUM is the largest result from bounded encoded input.
const DECODED_SIZE_MAXIMUM = ENCODED_SIZE_MAXIMUM / ENCODED_BYTE_SIZE

// DUMP_SOURCE_GROUP_SIZE follows the sixteen-byte canonical hexdump line.
const DUMP_SOURCE_GROUP_SIZE = ALPHABET_SIZE

// DUMP_LINE_SOURCE_SIZE_MINIMUM excludes empty input because no line exists for it.
const DUMP_LINE_SOURCE_SIZE_MINIMUM = SIZE_MINIMUM + 1

// DUMP_OFFSET_SOURCE_SIZE follows the canonical 32-bit hexdump offset.
const DUMP_OFFSET_SOURCE_SIZE = bits.BIT_COUNT_32_MAXIMUM / bits.BIT_COUNT_8_MAXIMUM

// DUMP_OFFSET_ENCODED_SIZE is the hexadecimal width of one canonical offset.
const DUMP_OFFSET_ENCODED_SIZE = DUMP_OFFSET_SOURCE_SIZE * ENCODED_BYTE_SIZE

// DUMP_OFFSET_SEPARATOR_SIZE separates offset from hexadecimal fields.
const DUMP_OFFSET_SEPARATOR_SIZE = ENCODED_BYTE_SIZE

// DUMP_FIELD_SEPARATOR_SIZE separates adjacent hexadecimal bytes.
const DUMP_FIELD_SEPARATOR_SIZE = SIZE_MINIMUM + 1

// DUMP_FIELD_SIZE holds one hexadecimal byte and its separator.
const DUMP_FIELD_SIZE = ENCODED_BYTE_SIZE + DUMP_FIELD_SEPARATOR_SIZE

// DUMP_MIDPOINT_SEPARATOR_SIZE divides the two eight-byte field groups.
const DUMP_MIDPOINT_SEPARATOR_SIZE = DUMP_FIELD_SEPARATOR_SIZE

// DUMP_ASCII_PREFIX_SIZE separates hexadecimal fields from printable bytes.
const DUMP_ASCII_PREFIX_SIZE = ENCODED_BYTE_SIZE

// DUMP_ASCII_SUFFIX_SIZE closes the printable column and line.
const DUMP_ASCII_SUFFIX_SIZE = ENCODED_BYTE_SIZE

// DUMP_LEFT_COLUMN_SIZE is fixed even when the final source group is partial.
const DUMP_LEFT_COLUMN_SIZE = DUMP_OFFSET_ENCODED_SIZE + DUMP_OFFSET_SEPARATOR_SIZE +
	DUMP_SOURCE_GROUP_SIZE*DUMP_FIELD_SIZE + DUMP_MIDPOINT_SEPARATOR_SIZE +
	DUMP_ASCII_PREFIX_SIZE

// DUMP_PARTIAL_LINE_FIXED_SIZE excludes only the variable printable bytes.
const DUMP_PARTIAL_LINE_FIXED_SIZE = DUMP_LEFT_COLUMN_SIZE + DUMP_ASCII_SUFFIX_SIZE

// DUMP_WRITE_DESTINATION_SIZE_MINIMUM fits one line containing one source byte.
const DUMP_WRITE_DESTINATION_SIZE_MINIMUM = DUMP_PARTIAL_LINE_FIXED_SIZE +
	DUMP_LINE_SOURCE_SIZE_MINIMUM

// DUMP_LINE_SIZE is one complete canonical line.
const DUMP_LINE_SIZE = DUMP_PARTIAL_LINE_FIXED_SIZE + DUMP_SOURCE_GROUP_SIZE

// DUMP_SIZE_MAXIMUM follows the repository byte-slice boundary.
const DUMP_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// DUMP_FULL_LINE_COUNT_MAXIMUM is the complete lines fitting bounded output.
const DUMP_FULL_LINE_COUNT_MAXIMUM = DUMP_SIZE_MAXIMUM / DUMP_LINE_SIZE

// DUMP_LINE_POSITION_MAXIMUM starts the final partial line fitting bounded output.
const DUMP_LINE_POSITION_MAXIMUM = DUMP_FULL_LINE_COUNT_MAXIMUM * DUMP_LINE_SIZE

// DUMP_SOURCE_POSITION_MAXIMUM starts the final partial source group.
const DUMP_SOURCE_POSITION_MAXIMUM = DUMP_FULL_LINE_COUNT_MAXIMUM * DUMP_SOURCE_GROUP_SIZE

// DUMP_OFFSET_POSITION_MINIMUM follows fixed offset field width.
const DUMP_OFFSET_POSITION_MINIMUM = DUMP_OFFSET_ENCODED_SIZE + DUMP_OFFSET_SEPARATOR_SIZE

// DUMP_OFFSET_POSITION_MAXIMUM follows final line start plus fixed offset field width.
const DUMP_OFFSET_POSITION_MAXIMUM = DUMP_LINE_POSITION_MAXIMUM + DUMP_OFFSET_POSITION_MINIMUM

// DUMP_ASCII_POSITION_MINIMUM follows fixed left-column width.
const DUMP_ASCII_POSITION_MINIMUM = DUMP_LEFT_COLUMN_SIZE

// DUMP_ASCII_POSITION_MAXIMUM follows final line start plus fixed left-column width.
const DUMP_ASCII_POSITION_MAXIMUM = DUMP_LINE_POSITION_MAXIMUM + DUMP_ASCII_POSITION_MINIMUM

// DUMP_OUTPUT_REMAINDER_SIZE is storage left after every fitting complete line.
const DUMP_OUTPUT_REMAINDER_SIZE = DUMP_SIZE_MAXIMUM % DUMP_LINE_SIZE

// DUMP_TAIL_SOURCE_SIZE_MAXIMUM is the partial source fitting output remainder.
const DUMP_TAIL_SOURCE_SIZE_MAXIMUM = DUMP_OUTPUT_REMAINDER_SIZE -
	DUMP_PARTIAL_LINE_FIXED_SIZE

// DUMP_SOURCE_SIZE_MAXIMUM is the largest source whose exact dump stays bounded.
const DUMP_SOURCE_SIZE_MAXIMUM = DUMP_FULL_LINE_COUNT_MAXIMUM*DUMP_SOURCE_GROUP_SIZE +
	DUMP_TAIL_SOURCE_SIZE_MAXIMUM

// ENCODE_ALPHABET is the canonical lowercase hexadecimal representation.
const ENCODE_ALPHABET = "0123456789abcdef"

// DECODE_ALPHABET keeps validation inside one immutable indexed lookup.
const DECODE_ALPHABET_INVALID_BLOCK = "\x10\x10\x10\x10\x10\x10\x10\x10" +
	"\x10\x10\x10\x10\x10\x10\x10\x10"

// DECODE_ALPHABET_DECIMAL_BLOCK avoids one range branch per input byte.
const DECODE_ALPHABET_DECIMAL_BLOCK = "\x00\x01\x02\x03\x04\x05\x06\x07" +
	"\x08\x09\x10\x10\x10\x10\x10\x10"

// DECODE_ALPHABET_UPPER_BLOCK avoids case conversion inside decode loop.
const DECODE_ALPHABET_UPPER_BLOCK = "\x10\x0a\x0b\x0c\x0d\x0e\x0f\x10" +
	"\x10\x10\x10\x10\x10\x10\x10\x10"

// DECODE_ALPHABET_LOWER_BLOCK shares values because ASCII case changes only index.
const DECODE_ALPHABET_LOWER_BLOCK = DECODE_ALPHABET_UPPER_BLOCK

// DECODE_ALPHABET removes data-dependent range chains from bounded input validation.
const DECODE_ALPHABET = DECODE_ALPHABET_INVALID_BLOCK +
	DECODE_ALPHABET_INVALID_BLOCK + DECODE_ALPHABET_INVALID_BLOCK +
	DECODE_ALPHABET_DECIMAL_BLOCK + DECODE_ALPHABET_UPPER_BLOCK +
	DECODE_ALPHABET_INVALID_BLOCK + DECODE_ALPHABET_LOWER_BLOCK +
	DECODE_ALPHABET_INVALID_BLOCK + DECODE_ALPHABET_INVALID_BLOCK +
	DECODE_ALPHABET_INVALID_BLOCK + DECODE_ALPHABET_INVALID_BLOCK +
	DECODE_ALPHABET_INVALID_BLOCK + DECODE_ALPHABET_INVALID_BLOCK +
	DECODE_ALPHABET_INVALID_BLOCK + DECODE_ALPHABET_INVALID_BLOCK +
	DECODE_ALPHABET_INVALID_BLOCK

// PRINTABLE_MINIMUM is the first byte retained in the hexdump text column.
const PRINTABLE_MINIMUM byte = ' '

// PRINTABLE_MAXIMUM is the final byte retained in the hexdump text column.
const PRINTABLE_MAXIMUM byte = '~'

// FIELD_SEPARATOR separates hexdump columns without an allocation-backed format string.
const FIELD_SEPARATOR byte = ' '

// ASCII_COLUMN_MARKER encloses the hexdump text column.
const ASCII_COLUMN_MARKER byte = '|'

// UNPRINTABLE_MARKER replaces bytes outside printable ASCII.
const UNPRINTABLE_MARKER byte = '.'

// LINE_FEED terminates each canonical hexdump line.
const LINE_FEED byte = '\n'

// STATUS_OK means the operation completed.
const STATUS_OK = 0

// STATUS_INPUT_INVALID means encoded input contains a non-hexadecimal digit.
const STATUS_INPUT_INVALID = STATUS_OK + 1

// STATUS_INPUT_INCOMPLETE means otherwise valid encoded input has an unmatched digit.
const STATUS_INPUT_INCOMPLETE = STATUS_INPUT_INVALID + 1

// STATUS_OUTPUT_TOO_SMALL means caller storage cannot hold the required output.
const STATUS_OUTPUT_TOO_SMALL = STATUS_INPUT_INCOMPLETE + 1

// STATUS_STORAGE_INVALID means source and destination overlap.
const STATUS_STORAGE_INVALID = STATUS_OUTPUT_TOO_SMALL + 1

// Source is decoded input whose encoded form stays within package bounds.
type Source []byte

// Source_Invariants rejects source whose encoded representation cannot stay bounded.
func Source_Invariants(value Source, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), SIZE_MINIMUM, SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Encoded is encoded input or writable encoded destination.
type Encoded []byte

// Encoded_Invariants keeps hexadecimal storage within repository bounds.
func Encoded_Invariants(value Encoded, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Decoded is writable decoded destination.
type Decoded []byte

// Decoded_Invariants keeps decoded output within its encoded-input-derived bound.
func Decoded_Invariants(value Decoded, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), SIZE_MINIMUM, DECODED_SIZE_MAXIMUM).
		Ensure()
}

// Dump_Source is source whose complete canonical dump stays within package bounds.
type Dump_Source []byte

// Dump_Source_Invariants rejects source whose exact dump cannot fit bounded output.
func Dump_Source_Invariants(value Dump_Source, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), SIZE_MINIMUM, DUMP_SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Dump is writable canonical hexdump destination.
type Dump []byte

// Dump_Invariants keeps dump output within repository bounds.
func Dump_Invariants(value Dump, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), SIZE_MINIMUM, DUMP_SIZE_MAXIMUM).
		Ensure()
}

// Source_Count is a decoded byte count accepted by Encoded_Size.
type Source_Count int

// Source_Count_Invariants follows Source's complete length domain.
func Source_Count_Invariants(value Source_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), SIZE_MINIMUM, SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Encoded_Input_Count is an encoded byte count accepted by Decoded_Size_Maximum.
type Encoded_Input_Count int

// Encoded_Input_Count_Invariants follows Encoded's complete length domain.
func Encoded_Input_Count_Invariants(
	value Encoded_Input_Count, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Encoded_Count is an exact required or written hexadecimal byte count.
type Encoded_Count int

// Encoded_Count_Invariants excludes counts that cannot represent whole source bytes.
func Encoded_Count_Invariants(value Encoded_Count, namespace invariant.Namespace) {
	invariant.Always(
		int(value)%ENCODED_BYTE_SIZE == 0,
		"Encoded count contains complete hexadecimal byte pairs.",
	)
	invariant.Tree(value, namespace).
		Range_Holed_Int(
			int(value), SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM,
			COUNT_HOLE_FIRST, COUNT_HOLE_FIRST, COUNT_HOLE_FIRST, COUNT_HOLE_FIRST,
		).
		Ensure()
}

// Decoded_Count is a maximum or written decoded byte count.
type Decoded_Count int

// Decoded_Count_Invariants follows bounded encoded input contraction.
func Decoded_Count_Invariants(value Decoded_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), SIZE_MINIMUM, DECODED_SIZE_MAXIMUM).
		Ensure()
}

// Dump_Source_Count is a source byte count accepted by Dump_Size.
type Dump_Source_Count int

// Dump_Source_Count_Invariants follows Dump_Source's complete length domain.
func Dump_Source_Count_Invariants(value Dump_Source_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), SIZE_MINIMUM, DUMP_SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Dump_Count is an exact required or written canonical dump byte count.
type Dump_Count int

// Dump_Count_Invariants excludes counts that cannot end at a canonical line boundary.
func Dump_Count_Invariants(value Dump_Count, namespace invariant.Namespace) {
	remainder := int(value) % DUMP_LINE_SIZE
	valid := remainder == SIZE_MINIMUM || remainder >= DUMP_PARTIAL_LINE_FIXED_SIZE+1
	invariant.Always(valid, "Dump count ends after a complete canonical line.")
	invariant.Tree(value, namespace).
		Range_Holed_Int(
			int(value), SIZE_MINIMUM, DUMP_SIZE_MAXIMUM,
			COUNT_HOLE_FIRST, COUNT_HOLE_SECOND, COUNT_HOLE_SECOND, COUNT_HOLE_SECOND,
		).
		Ensure()
}

// Dump_Line_Size keeps each writer inside fixed line geometry.
type Dump_Line_Size int

// Dump_Line_Size_Invariants excludes empty calls to line writers.
func Dump_Line_Size_Invariants(value Dump_Line_Size, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), DUMP_LINE_SOURCE_SIZE_MINIMUM, DUMP_SOURCE_GROUP_SIZE).
		Ensure()
}

// Dump_Write_Destination excludes storage from calls that emit no line.
type Dump_Write_Destination []byte

// Dump_Write_Destination_Invariants requires storage for one nonempty line.
func Dump_Write_Destination_Invariants(
	value Dump_Write_Destination, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), DUMP_WRITE_DESTINATION_SIZE_MINIMUM, DUMP_SIZE_MAXIMUM).
		Ensure()
}

// Dump_Write_Source excludes source from calls that emit no line.
type Dump_Write_Source []byte

// Dump_Write_Source_Invariants requires at least one source byte.
func Dump_Write_Source_Invariants(
	value Dump_Write_Source, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), DUMP_LINE_SOURCE_SIZE_MINIMUM, DUMP_SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Dump_Written_Count excludes the empty result from active line writers.
type Dump_Written_Count int

// Dump_Written_Count_Invariants preserves canonical nonempty line endings.
func Dump_Written_Count_Invariants(
	value Dump_Written_Count, namespace invariant.Namespace,
) {
	remainder := int(value) % DUMP_LINE_SIZE
	valid := remainder == SIZE_MINIMUM || remainder >= DUMP_WRITE_DESTINATION_SIZE_MINIMUM
	invariant.Always(valid, "Written dump ends after a complete canonical line.")
	invariant.Tree(value, namespace).
		Range_Int(int(value), DUMP_WRITE_DESTINATION_SIZE_MINIMUM, DUMP_SIZE_MAXIMUM).
		Ensure()
}

// Dump_Line_Position prevents drift from canonical fixed-width line starts.
type Dump_Line_Position int

// Dump_Line_Position_Invariants excludes impossible initial byte offsets.
func Dump_Line_Position_Invariants(
	value Dump_Line_Position, namespace invariant.Namespace,
) {
	invariant.Always(
		int(value)%DUMP_LINE_SIZE == SIZE_MINIMUM,
		"Dump line position follows fixed line geometry.",
	)
	invariant.Tree(value, namespace).
		Range_Holed_Int(
			int(value), SIZE_MINIMUM, DUMP_LINE_POSITION_MAXIMUM,
			COUNT_HOLE_FIRST, COUNT_HOLE_SECOND, COUNT_HOLE_SECOND, COUNT_HOLE_SECOND,
		).
		Ensure()
}

// Dump_Source_Position prevents drift from complete source-group starts.
type Dump_Source_Position int

// Dump_Source_Position_Invariants excludes impossible initial source offsets.
func Dump_Source_Position_Invariants(
	value Dump_Source_Position, namespace invariant.Namespace,
) {
	invariant.Always(
		int(value)%DUMP_SOURCE_GROUP_SIZE == SIZE_MINIMUM,
		"Dump source position follows fixed group geometry.",
	)
	invariant.Tree(value, namespace).
		Range_Holed_Int(
			int(value), SIZE_MINIMUM, DUMP_SOURCE_POSITION_MAXIMUM,
			COUNT_HOLE_FIRST, COUNT_HOLE_SECOND, COUNT_HOLE_SECOND, COUNT_HOLE_SECOND,
		).
		Ensure()
}

// Dump_Offset_Position keeps hexadecimal fields after fixed offset storage.
type Dump_Offset_Position int

// Dump_Offset_Position_Invariants preserves one fixed offset per line.
func Dump_Offset_Position_Invariants(
	value Dump_Offset_Position, namespace invariant.Namespace,
) {
	invariant.Always(
		(int(value)-DUMP_OFFSET_POSITION_MINIMUM)%DUMP_LINE_SIZE == SIZE_MINIMUM,
		"Dump offset position follows fixed line geometry.",
	)
	invariant.Tree(value, namespace).
		Range_Int(int(value), DUMP_OFFSET_POSITION_MINIMUM, DUMP_OFFSET_POSITION_MAXIMUM).
		Ensure()
}

// Dump_ASCII_Position keeps printable fields after fixed left-column storage.
type Dump_ASCII_Position int

// Dump_ASCII_Position_Invariants preserves one fixed left column per line.
func Dump_ASCII_Position_Invariants(
	value Dump_ASCII_Position, namespace invariant.Namespace,
) {
	invariant.Always(
		(int(value)-DUMP_ASCII_POSITION_MINIMUM)%DUMP_LINE_SIZE == SIZE_MINIMUM,
		"Dump ASCII position follows fixed line geometry.",
	)
	invariant.Tree(value, namespace).
		Range_Int(int(value), DUMP_ASCII_POSITION_MINIMUM, DUMP_ASCII_POSITION_MAXIMUM).
		Ensure()
}

// Encode_Status is a scalar result so storage refusal allocates no error interface.
type Encode_Status uint8

// Encode_Status_Invariants lists every encoder outcome.
func Encode_Status_Invariants(value Encode_Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_OUTPUT_TOO_SMALL),
			uint8(STATUS_STORAGE_INVALID),
		).
		Ensure()
}

// Decode_Status is a scalar result so malformed input allocates no error interface.
type Decode_Status uint8

// Decode_Status_Invariants lists every decoder outcome.
func Decode_Status_Invariants(value Decode_Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_STORAGE_INVALID),
		).
		Ensure()
}

// Dump_Status is a scalar result so storage refusal allocates no error interface.
type Dump_Status uint8

// Dump_Status_Invariants lists every dumper outcome.
func Dump_Status_Invariants(value Dump_Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_OUTPUT_TOO_SMALL),
			uint8(STATUS_STORAGE_INVALID),
		).
		Ensure()
}

// Encoded_Size reports exact caller storage required for source_count.
func Encoded_Size(source_count Source_Count) (count Encoded_Count) {
	defer func() { Encoded_Count_Invariants(count, "Encoded_Size.count") }()
	Source_Count_Invariants(source_count, "Encoded_Size.source_count")
	return Encoded_Count(int(source_count) * ENCODED_BYTE_SIZE)
}

// Decoded_Size_Maximum reports the complete bytes possible from encoded_count.
func Decoded_Size_Maximum(encoded_count Encoded_Input_Count) (count Decoded_Count) {
	defer func() {
		Decoded_Count_Invariants(count, "Decoded_Size_Maximum.count")
	}()
	Encoded_Input_Count_Invariants(encoded_count, "Decoded_Size_Maximum.encoded_count")
	return Decoded_Count(int(encoded_count) / ENCODED_BYTE_SIZE)
}

// Dump_Size reports exact canonical hexdump storage for source_count.
func Dump_Size(source_count Dump_Source_Count) (count Dump_Count) {
	defer func() { Dump_Count_Invariants(count, "Dump_Size.count") }()
	Dump_Source_Count_Invariants(source_count, "Dump_Size.source_count")
	full_line_count := int(source_count) / DUMP_SOURCE_GROUP_SIZE
	tail_size := int(source_count) % DUMP_SOURCE_GROUP_SIZE
	count = Dump_Count(full_line_count * DUMP_LINE_SIZE)
	if tail_size > SIZE_MINIMUM {
		count += Dump_Count(DUMP_PARTIAL_LINE_FIXED_SIZE + tail_size)
	}
	return count
}

// Encode_Into checks exact storage before exposing any partial representation.
func Encode_Into(
	destination Encoded, source Source,
) (count Encoded_Count, status Encode_Status) {
	defer func() {
		Encoded_Count_Invariants(count, "Encode_Into.count")
		Encode_Status_Invariants(status, "Encode_Into.status")
	}()
	Encoded_Invariants(destination, "Encode_Into.destination")
	Source_Invariants(source, "Encode_Into.source")
	if bytes.Overlap(bytes.Slice(destination), bytes.Slice(source)) {
		return 0, STATUS_STORAGE_INVALID
	}
	required := Encoded_Count(len(source) * ENCODED_BYTE_SIZE)
	if len(destination) < int(required) {
		return 0, STATUS_OUTPUT_TOO_SMALL
	}
	source_position := SIZE_MINIMUM
	destination_position := SIZE_MINIMUM
	bulk_end := len(source) / ENCODE_UNROLL_SOURCE_SIZE * ENCODE_UNROLL_SOURCE_SIZE
	const SOURCE_OFFSET_FIRST = SIZE_MINIMUM
	const SOURCE_OFFSET_STEP = ENCODED_BYTE_SIZE / ENCODED_BYTE_SIZE
	const SOURCE_OFFSET_SECOND = SOURCE_OFFSET_FIRST + SOURCE_OFFSET_STEP
	const SOURCE_OFFSET_THIRD = SOURCE_OFFSET_SECOND + SOURCE_OFFSET_STEP
	const SOURCE_OFFSET_FINAL = SOURCE_OFFSET_THIRD + SOURCE_OFFSET_STEP
	const DESTINATION_OFFSET_FIRST = SOURCE_OFFSET_FIRST * ENCODED_BYTE_SIZE
	const DESTINATION_OFFSET_SECOND = SOURCE_OFFSET_SECOND * ENCODED_BYTE_SIZE
	const DESTINATION_OFFSET_THIRD = SOURCE_OFFSET_THIRD * ENCODED_BYTE_SIZE
	const DESTINATION_OFFSET_FINAL = SOURCE_OFFSET_FINAL * ENCODED_BYTE_SIZE
	for source_position < bulk_end {
		first := source[source_position+SOURCE_OFFSET_FIRST]
		second := source[source_position+SOURCE_OFFSET_SECOND]
		third := source[source_position+SOURCE_OFFSET_THIRD]
		final := source[source_position+SOURCE_OFFSET_FINAL]
		destination[destination_position+DESTINATION_OFFSET_FIRST] =
			ENCODE_ALPHABET[first>>NIBBLE_BIT_COUNT]
		destination[destination_position+DESTINATION_OFFSET_FIRST+
			ENCODED_BYTE_FINAL_OFFSET] = ENCODE_ALPHABET[first&ALPHABET_FINAL_INDEX]
		destination[destination_position+DESTINATION_OFFSET_SECOND] =
			ENCODE_ALPHABET[second>>NIBBLE_BIT_COUNT]
		destination[destination_position+DESTINATION_OFFSET_SECOND+
			ENCODED_BYTE_FINAL_OFFSET] = ENCODE_ALPHABET[second&ALPHABET_FINAL_INDEX]
		destination[destination_position+DESTINATION_OFFSET_THIRD] =
			ENCODE_ALPHABET[third>>NIBBLE_BIT_COUNT]
		destination[destination_position+DESTINATION_OFFSET_THIRD+
			ENCODED_BYTE_FINAL_OFFSET] = ENCODE_ALPHABET[third&ALPHABET_FINAL_INDEX]
		destination[destination_position+DESTINATION_OFFSET_FINAL] =
			ENCODE_ALPHABET[final>>NIBBLE_BIT_COUNT]
		destination[destination_position+DESTINATION_OFFSET_FINAL+
			ENCODED_BYTE_FINAL_OFFSET] = ENCODE_ALPHABET[final&ALPHABET_FINAL_INDEX]
		source_position += ENCODE_UNROLL_SOURCE_SIZE
		destination_position += ENCODE_UNROLL_DESTINATION_SIZE
	}
	for source_position < len(source) {
		source_byte := source[source_position]
		destination[destination_position] = ENCODE_ALPHABET[source_byte>>NIBBLE_BIT_COUNT]
		destination[destination_position+ENCODED_BYTE_FINAL_OFFSET] =
			ENCODE_ALPHABET[source_byte&ALPHABET_FINAL_INDEX]
		source_position++
		destination_position += ENCODED_BYTE_SIZE
	}
	return required, STATUS_OK
}

// Decode_Into returns the valid prefix before malformed or unavailable output.
func Decode_Into(
	destination Decoded, source Encoded,
) (count Decoded_Count, status Decode_Status) {
	defer func() {
		Decoded_Count_Invariants(count, "Decode_Into.count")
		Decode_Status_Invariants(status, "Decode_Into.status")
	}()
	Decoded_Invariants(destination, "Decode_Into.destination")
	Encoded_Invariants(source, "Decode_Into.source")
	if bytes.Overlap(bytes.Slice(destination), bytes.Slice(source)) {
		return 0, STATUS_STORAGE_INVALID
	}
	decoded_limit_count := len(source) / ENCODED_BYTE_SIZE
	if decoded_limit_count > len(destination) {
		decoded_limit_count = len(destination)
	}
	source_end_position := decoded_limit_count * ENCODED_BYTE_SIZE
	invalid_position := decode_prefix_unchecked(
		destination, source, Encoded_Count(source_end_position),
	)
	if int(invalid_position) < source_end_position {
		return Decoded_Count(int(invalid_position) / ENCODED_BYTE_SIZE),
			STATUS_INPUT_INVALID
	}
	count = Decoded_Count(decoded_limit_count)
	if source_end_position+ENCODED_BYTE_SIZE <= len(source) {
		high := DECODE_ALPHABET[source[source_end_position]]
		low := DECODE_ALPHABET[source[source_end_position+ENCODED_BYTE_FINAL_OFFSET]]
		if high >= ALPHABET_SIZE {
			return count, STATUS_INPUT_INVALID
		}
		if low >= ALPHABET_SIZE {
			return count, STATUS_INPUT_INVALID
		}
		return count, STATUS_OUTPUT_TOO_SMALL
	}
	if source_end_position < len(source) {
		if DECODE_ALPHABET[source[source_end_position]] >= ALPHABET_SIZE {
			return count, STATUS_INPUT_INVALID
		}
		return count, STATUS_INPUT_INCOMPLETE
	}
	return count, STATUS_OK
}

func decode_prefix_unchecked(
	destination Decoded, source Encoded, source_end_position Encoded_Count,
) (source_position Encoded_Count) {
	defer func() {
		Encoded_Count_Invariants(source_position, "decode_prefix_unchecked.source_position")
	}()
	Decoded_Invariants(destination, "decode_prefix_unchecked.destination")
	Encoded_Invariants(source, "decode_prefix_unchecked.source")
	Encoded_Count_Invariants(
		source_end_position, "decode_prefix_unchecked.source_end_position",
	)
	const PAIR_OFFSET_FIRST = SIZE_MINIMUM
	const PAIR_OFFSET_SECOND = PAIR_OFFSET_FIRST + ENCODED_BYTE_SIZE
	const PAIR_OFFSET_THIRD = PAIR_OFFSET_SECOND + ENCODED_BYTE_SIZE
	const PAIR_OFFSET_FINAL = PAIR_OFFSET_THIRD + ENCODED_BYTE_SIZE
	const DESTINATION_OFFSET_FIRST = SIZE_MINIMUM
	const DESTINATION_OFFSET_STEP = ENCODE_UNROLL_SOURCE_SIZE / ENCODE_UNROLL_SOURCE_SIZE
	const DESTINATION_OFFSET_SECOND = DESTINATION_OFFSET_FIRST + DESTINATION_OFFSET_STEP
	const DESTINATION_OFFSET_THIRD = DESTINATION_OFFSET_SECOND + DESTINATION_OFFSET_STEP
	const DESTINATION_OFFSET_FINAL = DESTINATION_OFFSET_THIRD + DESTINATION_OFFSET_STEP
	const DECODE_UNROLL_SOURCE_SIZE = ENCODE_UNROLL_SOURCE_SIZE * ENCODED_BYTE_SIZE
	bulk_end_position := int(source_end_position) / DECODE_UNROLL_SOURCE_SIZE *
		DECODE_UNROLL_SOURCE_SIZE
	for int(source_position) < bulk_end_position {
		position := int(source_position)
		first_high := DECODE_ALPHABET[source[position+PAIR_OFFSET_FIRST]]
		first_low_position := position + PAIR_OFFSET_FIRST +
			ENCODED_BYTE_FINAL_OFFSET
		first_low := DECODE_ALPHABET[source[first_low_position]]
		second_high := DECODE_ALPHABET[source[position+PAIR_OFFSET_SECOND]]
		second_low_position := position + PAIR_OFFSET_SECOND +
			ENCODED_BYTE_FINAL_OFFSET
		second_low := DECODE_ALPHABET[source[second_low_position]]
		third_high := DECODE_ALPHABET[source[position+PAIR_OFFSET_THIRD]]
		third_low_position := position + PAIR_OFFSET_THIRD +
			ENCODED_BYTE_FINAL_OFFSET
		third_low := DECODE_ALPHABET[source[third_low_position]]
		final_high := DECODE_ALPHABET[source[position+PAIR_OFFSET_FINAL]]
		final_low_position := position + PAIR_OFFSET_FINAL +
			ENCODED_BYTE_FINAL_OFFSET
		final_low := DECODE_ALPHABET[source[final_low_position]]
		combined := first_high | first_low | second_high | second_low |
			third_high | third_low | final_high | final_low
		if combined >= ALPHABET_SIZE {
			break
		}
		destination_position := position / ENCODED_BYTE_SIZE
		destination[destination_position+DESTINATION_OFFSET_FIRST] =
			first_high<<NIBBLE_BIT_COUNT | first_low
		destination[destination_position+DESTINATION_OFFSET_SECOND] =
			second_high<<NIBBLE_BIT_COUNT | second_low
		destination[destination_position+DESTINATION_OFFSET_THIRD] =
			third_high<<NIBBLE_BIT_COUNT | third_low
		destination[destination_position+DESTINATION_OFFSET_FINAL] =
			final_high<<NIBBLE_BIT_COUNT | final_low
		source_position += DECODE_UNROLL_SOURCE_SIZE
	}
	for source_position < source_end_position {
		position := int(source_position)
		high := DECODE_ALPHABET[source[position]]
		low := DECODE_ALPHABET[source[position+ENCODED_BYTE_FINAL_OFFSET]]
		if high >= ALPHABET_SIZE {
			return source_position
		}
		if low >= ALPHABET_SIZE {
			return source_position
		}
		destination[position/ENCODED_BYTE_SIZE] = high<<NIBBLE_BIT_COUNT | low
		source_position += ENCODED_BYTE_SIZE
	}
	return source_position
}

// Dump_Into checks exact storage before exposing any partial line.
func Dump_Into(
	destination Dump, source Dump_Source,
) (count Dump_Count, status Dump_Status) {
	defer func() {
		Dump_Count_Invariants(count, "Dump_Into.count")
		Dump_Status_Invariants(status, "Dump_Into.status")
	}()
	Dump_Invariants(destination, "Dump_Into.destination")
	Dump_Source_Invariants(source, "Dump_Into.source")
	if bytes.Overlap(bytes.Slice(destination), bytes.Slice(source)) {
		return 0, STATUS_STORAGE_INVALID
	}
	required := Dump_Size(Dump_Source_Count(len(source)))
	if len(destination) < int(required) {
		return 0, STATUS_OUTPUT_TOO_SMALL
	}
	if required == SIZE_MINIMUM {
		return required, STATUS_OK
	}
	dump_unchecked(Dump_Write_Destination(destination), Dump_Write_Source(source))
	return required, STATUS_OK
}

func dump_unchecked(
	destination Dump_Write_Destination, source Dump_Write_Source,
) {
	Dump_Write_Destination_Invariants(destination, "dump_unchecked.destination")
	Dump_Write_Source_Invariants(source, "dump_unchecked.source")
	destination_position := Dump_Line_Position(SIZE_MINIMUM)
	for source_position := Dump_Source_Position(SIZE_MINIMUM); ; {
		line_source_size := Dump_Line_Size(len(source) - int(source_position))
		if int(line_source_size) > DUMP_SOURCE_GROUP_SIZE {
			line_source_size = DUMP_SOURCE_GROUP_SIZE
		}
		offset_position := dump_offset(
			destination, destination_position, source_position,
		)
		ascii_position := dump_hexadecimal(
			destination, source, offset_position,
			source_position, line_source_size,
		)
		next_position := dump_ascii(
			destination, source, ascii_position,
			source_position, line_source_size,
		)
		next_source_position := int(source_position) + int(line_source_size)
		if next_source_position == len(source) {
			return
		}
		destination_position = Dump_Line_Position(next_position)
		source_position = Dump_Source_Position(next_source_position)
	}
}

func dump_offset(
	destination Dump_Write_Destination, destination_position Dump_Line_Position,
	source_position Dump_Source_Position,
) (next_position Dump_Offset_Position) {
	defer func() {
		Dump_Offset_Position_Invariants(next_position, "dump_offset.next_position")
	}()
	Dump_Write_Destination_Invariants(destination, "dump_offset.destination")
	Dump_Line_Position_Invariants(destination_position, "dump_offset.destination_position")
	Dump_Source_Position_Invariants(source_position, "dump_offset.source_position")
	destination_position_value := int(destination_position)
	offset := uint32(source_position)
	for encoded_position := range DUMP_OFFSET_ENCODED_SIZE {
		shift := (DUMP_OFFSET_ENCODED_SIZE - encoded_position - 1) * NIBBLE_BIT_COUNT
		nibble := offset >> shift & uint32(ALPHABET_FINAL_INDEX)
		destination[destination_position_value+encoded_position] = ENCODE_ALPHABET[nibble]
	}
	next := destination_position_value + DUMP_OFFSET_ENCODED_SIZE
	for range DUMP_OFFSET_SEPARATOR_SIZE {
		destination[next] = FIELD_SEPARATOR
		next++
	}
	return Dump_Offset_Position(next)
}

func dump_hexadecimal(
	destination Dump_Write_Destination, source Dump_Write_Source,
	destination_position Dump_Offset_Position,
	source_position Dump_Source_Position, line_source_size Dump_Line_Size,
) (next_position Dump_ASCII_Position) {
	defer func() {
		Dump_ASCII_Position_Invariants(next_position, "dump_hexadecimal.next_position")
	}()
	Dump_Write_Destination_Invariants(destination, "dump_hexadecimal.destination")
	Dump_Write_Source_Invariants(source, "dump_hexadecimal.source")
	Dump_Offset_Position_Invariants(
		destination_position, "dump_hexadecimal.destination_position",
	)
	Dump_Source_Position_Invariants(source_position, "dump_hexadecimal.source_position")
	Dump_Line_Size_Invariants(line_source_size, "dump_hexadecimal.line_source_size")
	next := int(destination_position)
	source_index := int(source_position)
	line_size := int(line_source_size)
	for line_position := SIZE_MINIMUM; line_position < DUMP_SOURCE_GROUP_SIZE; line_position++ {
		if line_position < line_size {
			source_byte := source[source_index+line_position]
			destination[next] = ENCODE_ALPHABET[source_byte>>NIBBLE_BIT_COUNT]
			destination[next+ENCODED_BYTE_FINAL_OFFSET] =
				ENCODE_ALPHABET[source_byte&ALPHABET_FINAL_INDEX]
		} else {
			for encoded_position := range ENCODED_BYTE_SIZE {
				destination[next+encoded_position] = FIELD_SEPARATOR
			}
		}
		next += ENCODED_BYTE_SIZE
		destination[next] = FIELD_SEPARATOR
		next++
		if line_position == DUMP_SOURCE_GROUP_SIZE/ENCODED_BYTE_SIZE-1 {
			destination[next] = FIELD_SEPARATOR
			next++
		}
	}
	destination[next] = FIELD_SEPARATOR
	destination[next+DUMP_FIELD_SEPARATOR_SIZE] = ASCII_COLUMN_MARKER
	return Dump_ASCII_Position(next + DUMP_ASCII_PREFIX_SIZE)
}

func dump_ascii(
	destination Dump_Write_Destination, source Dump_Write_Source,
	destination_position Dump_ASCII_Position,
	source_position Dump_Source_Position, line_source_size Dump_Line_Size,
) (next_position Dump_Written_Count) {
	defer func() {
		Dump_Written_Count_Invariants(next_position, "dump_ascii.next_position")
	}()
	Dump_Write_Destination_Invariants(destination, "dump_ascii.destination")
	Dump_Write_Source_Invariants(source, "dump_ascii.source")
	Dump_ASCII_Position_Invariants(destination_position, "dump_ascii.destination_position")
	Dump_Source_Position_Invariants(source_position, "dump_ascii.source_position")
	Dump_Line_Size_Invariants(line_source_size, "dump_ascii.line_source_size")
	next := int(destination_position)
	source_index := int(source_position)
	for line_position := range int(line_source_size) {
		source_byte := source[source_index+line_position]
		destination[next] = UNPRINTABLE_MARKER
		if PRINTABLE_MINIMUM <= source_byte {
			if source_byte <= PRINTABLE_MAXIMUM {
				destination[next] = source_byte
			}
		}
		next++
	}
	destination[next] = ASCII_COLUMN_MARKER
	destination[next+DUMP_FIELD_SEPARATOR_SIZE] = LINE_FEED
	return Dump_Written_Count(next + DUMP_ASCII_SUFFIX_SIZE)
}
