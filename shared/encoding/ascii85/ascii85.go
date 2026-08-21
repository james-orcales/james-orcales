// Package ascii85 encodes and decodes Adobe ASCII85 through bounded caller-owned storage.
package ascii85

import (
	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/math/bits"
)

// DECODED_GROUP_SIZE follows ASCII85's 32-bit source word.
const DECODED_GROUP_SIZE = bits.BIT_COUNT_32_MAXIMUM / bits.BIT_COUNT_8_MAXIMUM

// ENCODED_GROUP_SIZE adds one radix digit to each decoded group.
const ENCODED_GROUP_SIZE = DECODED_GROUP_SIZE + 1

// DECODED_GROUP_FINAL_INDEX is final byte position in one decoded group.
const DECODED_GROUP_FINAL_INDEX = DECODED_GROUP_SIZE - 1

// ENCODED_INPUT_SIZE_MAXIMUM follows repository byte-slice boundary.
const ENCODED_INPUT_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// DECODED_SIZE_MAXIMUM follows repository byte-slice boundary.
const DECODED_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// ENCODE_SOURCE_SIZE_MAXIMUM is largest whole source group whose worst encoding stays bounded.
const ENCODE_SOURCE_SIZE_MAXIMUM = ENCODED_INPUT_SIZE_MAXIMUM /
	ENCODED_GROUP_SIZE * DECODED_GROUP_SIZE

// ENCODED_SIZE_MAXIMUM is worst output for largest admitted encode source.
const ENCODED_SIZE_MAXIMUM = (ENCODE_SOURCE_SIZE_MAXIMUM + DECODED_GROUP_SIZE - 1) /
	DECODED_GROUP_SIZE * ENCODED_GROUP_SIZE

// ENCODED_GROUP_COUNT_MAXIMUM is complete groups in largest encode source.
const ENCODED_GROUP_COUNT_MAXIMUM = ENCODE_SOURCE_SIZE_MAXIMUM / DECODED_GROUP_SIZE

// SIZE_MINIMUM admits empty source and output.
const SIZE_MINIMUM = 0

// DECODED_WRITE_DESTINATION_SIZE_MINIMUM admits shortest flushed group.
const DECODED_WRITE_DESTINATION_SIZE_MINIMUM = SIZE_MINIMUM + 1

// MAXIMUM_ENCODED_COUNT_HOLE_FIRST excludes a count below one complete encoded group.
const MAXIMUM_ENCODED_COUNT_HOLE_FIRST = DECODED_WRITE_DESTINATION_SIZE_MINIMUM

// MAXIMUM_ENCODED_COUNT_HOLE_SECOND excludes the next incomplete encoded group count.
const MAXIMUM_ENCODED_COUNT_HOLE_SECOND = MAXIMUM_ENCODED_COUNT_HOLE_FIRST +
	DECODED_WRITE_DESTINATION_SIZE_MINIMUM

// MAXIMUM_ENCODED_COUNT_HOLE_THIRD excludes the next incomplete encoded group count.
const MAXIMUM_ENCODED_COUNT_HOLE_THIRD = MAXIMUM_ENCODED_COUNT_HOLE_SECOND +
	DECODED_WRITE_DESTINATION_SIZE_MINIMUM

// MAXIMUM_ENCODED_COUNT_HOLE_FINAL closes every remainder before one complete encoded group.
const MAXIMUM_ENCODED_COUNT_HOLE_FINAL = MAXIMUM_ENCODED_COUNT_HOLE_THIRD +
	DECODED_WRITE_DESTINATION_SIZE_MINIMUM

// ASCII85_BASE is wire-format radix.
const ASCII85_BASE = 85

// ASCII85_DIGIT_MINIMUM is zero wire digit.
const ASCII85_DIGIT_MINIMUM byte = '!'

// ASCII85_DIGIT_MAXIMUM is final wire digit.
const ASCII85_DIGIT_MAXIMUM byte = ASCII85_DIGIT_MINIMUM + ASCII85_BASE - 1

// ASCII85_DIGIT_VALUE_MAXIMUM pads truncated groups with largest possible digit.
const ASCII85_DIGIT_VALUE_MAXIMUM = ASCII85_BASE - 1

// ZERO_GROUP_MARKER shortens one complete zero word.
const ZERO_GROUP_MARKER byte = 'z'

// IGNORED_BYTE_MAXIMUM includes ASCII whitespace and control bytes ignored by decoder.
const IGNORED_BYTE_MAXIMUM byte = ' '

// STATUS_OK means complete requested operation succeeded.
const STATUS_OK = 0

// STATUS_INPUT_INVALID means encoded input violated ASCII85 grammar.
const STATUS_INPUT_INVALID = STATUS_OK + 1

// STATUS_OUTPUT_TOO_SMALL means caller destination cannot hold next complete output group.
const STATUS_OUTPUT_TOO_SMALL = STATUS_INPUT_INVALID + 1

// Encode_Source is decoded input whose worst encoded form stays in package bound.
type Encode_Source []byte

// Encode_Source_Invariants bounds input before encoder reads it.
func Encode_Source_Invariants(value Encode_Source, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), SIZE_MINIMUM, ENCODE_SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Encoded is encoded input or writable encoded destination.
type Encoded []byte

// Encoded_Invariants bounds every encoded view.
func Encoded_Invariants(value Encoded, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), SIZE_MINIMUM, ENCODED_INPUT_SIZE_MAXIMUM).
		Ensure()
}

// Decoded is writable decoded destination.
type Decoded []byte

// Decoded_Invariants bounds decoded caller storage.
func Decoded_Invariants(value Decoded, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), SIZE_MINIMUM, DECODED_SIZE_MAXIMUM).
		Ensure()
}

// Decoded_Write_Destination is suffix that can receive at least one decoded byte.
type Decoded_Write_Destination []byte

// Decoded_Write_Destination_Invariants excludes storage that cannot receive any group.
func Decoded_Write_Destination_Invariants(
	value Decoded_Write_Destination, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			len(value), DECODED_WRITE_DESTINATION_SIZE_MINIMUM, DECODED_SIZE_MAXIMUM,
		).
		Ensure()
}

// Source_Count is byte count accepted by encoding.
type Source_Count int

// Source_Count_Invariants bounds size calculation input.
func Source_Count_Invariants(value Source_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), SIZE_MINIMUM, ENCODE_SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Encoded_Count is encoded bytes written or required.
type Encoded_Count int

// Encoded_Count_Invariants bounds every encoded result.
func Encoded_Count_Invariants(value Encoded_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Maximum_Encoded_Count is whole-group worst-case encoded storage.
type Maximum_Encoded_Count int

// Maximum_Encoded_Count_Invariants removes counts impossible under five-byte group rounding.
func Maximum_Encoded_Count_Invariants(
	value Maximum_Encoded_Count, namespace invariant.Namespace,
) {
	invariant.Always(
		int(value)%ENCODED_GROUP_SIZE == 0,
		"Maximum encoded count is whole encoded groups.",
	)
	invariant.Tree(value, namespace).
		Range_Holed_Int(
			int(value), SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM,
			MAXIMUM_ENCODED_COUNT_HOLE_FIRST, MAXIMUM_ENCODED_COUNT_HOLE_SECOND,
			MAXIMUM_ENCODED_COUNT_HOLE_THIRD, MAXIMUM_ENCODED_COUNT_HOLE_FINAL,
		).
		Ensure()
}

// Decoded_Count is decoded bytes published.
type Decoded_Count int

// Decoded_Count_Invariants bounds every decoded result.
func Decoded_Count_Invariants(value Decoded_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), SIZE_MINIMUM, DECODED_SIZE_MAXIMUM).
		Ensure()
}

// Consumed_Count is encoded source prefix fully consumed.
type Consumed_Count int

// Consumed_Count_Invariants bounds consumed input through complete source.
func Consumed_Count_Invariants(value Consumed_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), SIZE_MINIMUM, ENCODED_INPUT_SIZE_MAXIMUM).
		Ensure()
}

// Flush states source is final input.
type Flush bool

// Flush_Invariants reaches streaming and final decode modes.
func Flush_Invariants(value Flush, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "A decode flushes final input.").
		Ensure()
}

// Status is scalar result so malformed input allocates no error interface.
type Status uint8

// Status_Invariants lists complete operation outcomes.
func Status_Invariants(value Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_INPUT_INVALID),
			uint8(STATUS_OUTPUT_TOO_SMALL),
		).
		Ensure()
}

// Encode_Status excludes input failure because decoded source has no invalid representation.
type Encode_Status uint8

// Encode_Status_Invariants lists encoder outcomes.
func Encode_Status_Invariants(value Encode_Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_OUTPUT_TOO_SMALL),
		).
		Ensure()
}

// Encoded_Size_Maximum lets caller prove capacity before source exists.
func Encoded_Size_Maximum(source_count Source_Count) (count Maximum_Encoded_Count) {
	defer func() {
		Maximum_Encoded_Count_Invariants(count, "Encoded_Size_Maximum.count")
	}()
	Source_Count_Invariants(source_count, "Encoded_Size_Maximum.source_count")
	return Maximum_Encoded_Count(
		(int(source_count) + DECODED_GROUP_SIZE - 1) /
			DECODED_GROUP_SIZE * ENCODED_GROUP_SIZE,
	)
}

// Encoded_Size avoids worst-case refusal when complete zero groups shorten to one byte.
func Encoded_Size(source Encode_Source) (count Encoded_Count) {
	defer func() { Encoded_Count_Invariants(count, "Encoded_Size.count") }()
	Encode_Source_Invariants(source, "Encoded_Size.source")
	position := 0
	for len(source)-position >= DECODED_GROUP_SIZE {
		var value uint32
		for byte_position_index := range DECODED_GROUP_SIZE {
			byte_shift_count := DECODED_GROUP_FINAL_INDEX - byte_position_index
			shift := byte_shift_count * bits.BIT_COUNT_8_MAXIMUM
			value |= uint32(source[position+byte_position_index]) << shift
		}
		if value == 0 {
			count++
		} else {
			count += ENCODED_GROUP_SIZE
		}
		position += DECODED_GROUP_SIZE
	}
	tail_size := len(source) - position
	if tail_size > 0 {
		count += Encoded_Count(tail_size + 1)
	}
	return count
}

// Encode_Into checks exact output first so short destination exposes no partial representation.
func Encode_Into(
	destination Encoded, source Encode_Source,
) (count Encoded_Count, status Encode_Status) {
	defer func() {
		Encoded_Count_Invariants(count, "Encode_Into.count")
		Encode_Status_Invariants(status, "Encode_Into.status")
	}()
	Encoded_Invariants(destination, "Encode_Into.destination")
	Encode_Source_Invariants(source, "Encode_Into.source")
	invariant.Always(
		!bool(bytes.Overlap(bytes.Slice(destination), bytes.Slice(source))),
		"ASCII85 encode source and destination do not overlap.",
	)
	required := Encoded_Size(source)
	if len(destination) < int(required) {
		return 0, STATUS_OUTPUT_TOO_SMALL
	}

	source_position := 0
	destination_position := 0
	for source_position < len(source) {
		tail_size := len(source) - source_position
		group_size := DECODED_GROUP_SIZE
		if tail_size < group_size {
			group_size = tail_size
		}
		var value uint32
		for byte_position_index := range group_size {
			byte_shift_count := DECODED_GROUP_FINAL_INDEX - byte_position_index
			shift := byte_shift_count * bits.BIT_COUNT_8_MAXIMUM
			value |= uint32(source[source_position+byte_position_index]) << shift
		}
		if value == 0 {
			if group_size == DECODED_GROUP_SIZE {
				destination[destination_position] = ZERO_GROUP_MARKER
				destination_position++
				source_position += group_size
				continue
			}
		}
		var group [ENCODED_GROUP_SIZE]byte
		for digit_position := ENCODED_GROUP_SIZE; digit_position > 0; digit_position-- {
			group[digit_position-1] = ASCII85_DIGIT_MINIMUM + byte(value%ASCII85_BASE)
			value /= ASCII85_BASE
		}
		encoded_group_size := group_size + 1
		copy(
			destination[destination_position:destination_position+encoded_group_size],
			group[:encoded_group_size],
		)
		destination_position += encoded_group_size
		source_position += group_size
	}
	return Encoded_Count(destination_position), STATUS_OK
}

// Decode_Into retains incomplete group through consumed count when flush is false.
func Decode_Into(
	destination Decoded, source Encoded, flush Flush,
) (decoded Decoded_Count, consumed Consumed_Count, status Status) {
	defer func() {
		Decoded_Count_Invariants(decoded, "Decode_Into.decoded")
		Consumed_Count_Invariants(consumed, "Decode_Into.consumed")
		Status_Invariants(status, "Decode_Into.status")
	}()
	Decoded_Invariants(destination, "Decode_Into.destination")
	Encoded_Invariants(source, "Decode_Into.source")
	Flush_Invariants(flush, "Decode_Into.flush")
	overlap := bytes.Overlap(bytes.Slice(destination), bytes.Slice(source))
	invariant.Always(!bool(overlap), "ASCII85 decode source and destination do not overlap.")
	var value uint32
	digit_count := 0
	for source_position, encoded_byte := range source {
		if encoded_byte <= IGNORED_BYTE_MAXIMUM {
			continue
		}
		destination_tail := Decoded_Write_Destination(destination[int(decoded):])
		if encoded_byte == ZERO_GROUP_MARKER {
			if digit_count != 0 {
				return 0, 0, STATUS_INPUT_INVALID
			}
			if len(destination)-int(decoded) < DECODED_GROUP_SIZE {
				return decoded, consumed, STATUS_OUTPUT_TOO_SMALL
			}
			decoded_write(destination_tail, uint32(0), DECODED_GROUP_SIZE)
			decoded += DECODED_GROUP_SIZE
			consumed = Consumed_Count(source_position + 1)
			continue
		}
		if encoded_byte < ASCII85_DIGIT_MINIMUM {
			return 0, 0, STATUS_INPUT_INVALID
		}
		if encoded_byte > ASCII85_DIGIT_MAXIMUM {
			return 0, 0, STATUS_INPUT_INVALID
		}
		value = value*ASCII85_BASE + uint32(encoded_byte-ASCII85_DIGIT_MINIMUM)
		digit_count++
		if digit_count != ENCODED_GROUP_SIZE {
			continue
		}
		if len(destination)-int(decoded) < DECODED_GROUP_SIZE {
			return decoded, consumed, STATUS_OUTPUT_TOO_SMALL
		}
		decoded_write(destination_tail, value, DECODED_GROUP_SIZE)
		decoded += DECODED_GROUP_SIZE
		consumed = Consumed_Count(source_position + 1)
		value = 0
		digit_count = 0
	}
	switch {
	case !bool(flush):
		return decoded, consumed, STATUS_OK
	case digit_count == 1:
		return 0, 0, STATUS_INPUT_INVALID
	case digit_count == 0:
		return decoded, Consumed_Count(len(source)), STATUS_OK
	}
	for digit_position := digit_count; digit_position < ENCODED_GROUP_SIZE; digit_position++ {
		value = value*ASCII85_BASE + ASCII85_DIGIT_VALUE_MAXIMUM
	}
	partial_size := digit_count - 1
	if len(destination)-int(decoded) < partial_size {
		return decoded, consumed, STATUS_OUTPUT_TOO_SMALL
	}
	destination_tail := Decoded_Write_Destination(destination[int(decoded):])
	decoded_write(destination_tail, value, partial_size)
	decoded += Decoded_Count(partial_size)
	return decoded, Consumed_Count(len(source)), STATUS_OK
}

func decoded_write[Word ~uint32, Count ~int](
	destination Decoded_Write_Destination, value Word, count Count,
) {
	Decoded_Write_Destination_Invariants(destination, "decoded_write.destination")
	invariant.Always(
		int(count) >= DECODED_WRITE_DESTINATION_SIZE_MINIMUM,
		"Decoded write emits at least one byte.",
	)
	invariant.Always(
		int(count) <= DECODED_GROUP_SIZE,
		"Decoded write emits at most one group.",
	)
	for byte_position_index := range int(count) {
		byte_shift_count := DECODED_GROUP_FINAL_INDEX - byte_position_index
		shift := byte_shift_count * bits.BIT_COUNT_8_MAXIMUM
		destination[byte_position_index] = byte(value >> shift)
	}
}
