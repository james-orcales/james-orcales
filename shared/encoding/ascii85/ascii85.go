// Package ascii85 encodes and decodes Adobe ASCII85 through bounded caller-owned storage.
package ascii85

import (
	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/aver/default"
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

// BULK_GROUP_COUNT_MAXIMUM keeps paired source and destination advances inside bounds.
const BULK_GROUP_COUNT_MAXIMUM = ENCODED_INPUT_SIZE_MAXIMUM / ENCODED_GROUP_SIZE

// SIZE_MINIMUM admits empty source and output.
const SIZE_MINIMUM = 0

// DECODED_WRITE_DESTINATION_SIZE_MINIMUM admits shortest flushed group.
const DECODED_WRITE_DESTINATION_SIZE_MINIMUM = SIZE_MINIMUM + 1

// DECODED_WRITE_COUNT_SECOND follows second nonempty group prefix.
const DECODED_WRITE_COUNT_SECOND = DECODED_WRITE_DESTINATION_SIZE_MINIMUM + 1

// DECODED_WRITE_COUNT_THIRD follows third nonempty group prefix.
const DECODED_WRITE_COUNT_THIRD = DECODED_WRITE_COUNT_SECOND + 1

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
func Encode_Source_Invariants(value Encode_Source, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), SIZE_MINIMUM, ENCODE_SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Encoded is encoded input or writable encoded destination.
type Encoded []byte

// Encoded_Invariants bounds every encoded view.
func Encoded_Invariants(value Encoded, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), SIZE_MINIMUM, ENCODED_INPUT_SIZE_MAXIMUM).
		Ensure()
}

// Decoded is writable decoded destination.
type Decoded []byte

// Decoded_Invariants bounds decoded caller storage.
func Decoded_Invariants(value Decoded, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), SIZE_MINIMUM, DECODED_SIZE_MAXIMUM).
		Ensure()
}

// Decoded_Write_Destination is suffix that can receive at least one decoded byte.
type Decoded_Write_Destination []byte

// Decoded_Write_Destination_Invariants excludes storage that cannot receive any group.
func Decoded_Write_Destination_Invariants(
	value Decoded_Write_Destination, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), DECODED_WRITE_DESTINATION_SIZE_MINIMUM, DECODED_SIZE_MAXIMUM,
		).
		Ensure()
}

// Decoded_Word keeps one group value independent from host word size.
type Decoded_Word uint32

// Decoded_Word_Invariants retains every possible four-byte group.
func Decoded_Word_Invariants(value Decoded_Word, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), bits.WORD_32_MINIMUM, bits.WORD_32_MAXIMUM).
		Ensure()
}

// Decoded_Write_Count excludes an empty write from fixed group storage.
type Decoded_Write_Count int

// Decoded_Write_Count_Invariants permits each nonempty group prefix.
func Decoded_Write_Count_Invariants(
	value Decoded_Write_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_4_Int(
			int(value), DECODED_WRITE_DESTINATION_SIZE_MINIMUM,
			DECODED_WRITE_COUNT_SECOND,
			DECODED_WRITE_COUNT_THIRD,
			DECODED_GROUP_SIZE,
		).
		Ensure()
}

// Source_Count is byte count accepted by encoding.
type Source_Count int

// Source_Count_Invariants bounds size calculation input.
func Source_Count_Invariants(value Source_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), SIZE_MINIMUM, ENCODE_SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Encoded_Count is encoded bytes written or required.
type Encoded_Count int

// Encoded_Count_Invariants bounds every encoded result.
func Encoded_Count_Invariants(value Encoded_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Maximum_Encoded_Count is whole-group worst-case encoded storage.
type Maximum_Encoded_Count int

// Maximum_Encoded_Count_Invariants removes counts impossible under five-byte group rounding.
func Maximum_Encoded_Count_Invariants(
	value Maximum_Encoded_Count, namespace aver.Namespace,
) {
	aver.Always(
		int(value)%ENCODED_GROUP_SIZE == 0,
		"Maximum encoded count is whole encoded groups.",
	)
	aver.Tree(value, namespace).
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
func Decoded_Count_Invariants(value Decoded_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), SIZE_MINIMUM, DECODED_SIZE_MAXIMUM).
		Ensure()
}

// Bulk_Group_Count keeps paired source and destination advances one checked value.
type Bulk_Group_Count int

// Bulk_Group_Count_Invariants covers every complete bulk group.
func Bulk_Group_Count_Invariants(value Bulk_Group_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), SIZE_MINIMUM, BULK_GROUP_COUNT_MAXIMUM).
		Ensure()
}

// Consumed_Count is encoded source prefix fully consumed.
type Consumed_Count int

// Consumed_Count_Invariants bounds consumed input through complete source.
func Consumed_Count_Invariants(value Consumed_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), SIZE_MINIMUM, ENCODED_INPUT_SIZE_MAXIMUM).
		Ensure()
}

// Flush states source is final input.
type Flush bool

// Flush_Invariants reaches streaming and final decode modes.
func Flush_Invariants(value Flush, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "A decode flushes final input.").
		Ensure()
}

// Status is scalar result so malformed input allocates no error interface.
type Status uint8

// Status_Invariants lists complete operation outcomes.
func Status_Invariants(value Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_INPUT_INVALID),
			uint8(STATUS_OUTPUT_TOO_SMALL),
		).
		Ensure()
}

// Encode_Status excludes input failure because decoded source has no invalid representation.
type Encode_Status uint8

// Encode_Status_Invariants lists encoder outcomes.
func Encode_Status_Invariants(value Encode_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
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
	aver.Always(
		!bool(bytes.Overlap(bytes.Slice(destination), bytes.Slice(source))),
		"ASCII85 encode source and destination do not overlap.",
	)
	maximum := Encoded_Count(Encoded_Size_Maximum(Source_Count(len(source))))
	required := maximum
	if len(destination) < int(maximum) {
		required = Encoded_Size(source)
	}
	if len(destination) < int(required) {
		return 0, STATUS_OUTPUT_TOO_SMALL
	}

	source_position := 0
	destination_position := 0
	for len(source)-source_position >= DECODED_GROUP_SIZE {
		source_group := source[source_position : source_position+DECODED_GROUP_SIZE]
		shift := DECODED_GROUP_FINAL_INDEX * bits.BIT_COUNT_8_MAXIMUM
		value := uint32(source_group[0]) << shift
		shift -= bits.BIT_COUNT_8_MAXIMUM
		value |= uint32(source_group[1]) << shift
		shift -= bits.BIT_COUNT_8_MAXIMUM
		value |= uint32(source_group[2]) << shift
		value |= uint32(source_group[3])
		if value == 0 {
			destination[destination_position] = ZERO_GROUP_MARKER
			destination_position++
			source_position += DECODED_GROUP_SIZE
			continue
		}
		destination_end := destination_position + ENCODED_GROUP_SIZE
		destination_group := destination[destination_position:destination_end]
		for digit_position := ENCODED_GROUP_SIZE; digit_position > 0; digit_position-- {
			destination_group[digit_position-1] = ASCII85_DIGIT_MINIMUM +
				byte(value%ASCII85_BASE)
			value /= ASCII85_BASE
		}
		destination_position += ENCODED_GROUP_SIZE
		source_position += DECODED_GROUP_SIZE
	}
	tail_size := len(source) - source_position
	if tail_size > 0 {
		var value uint32
		for byte_position_index := range tail_size {
			byte_shift_count := DECODED_GROUP_FINAL_INDEX - byte_position_index
			shift := byte_shift_count * bits.BIT_COUNT_8_MAXIMUM
			value |= uint32(source[source_position+byte_position_index]) << shift
		}
		var group [ENCODED_GROUP_SIZE]byte
		for digit_position := ENCODED_GROUP_SIZE; digit_position > 0; digit_position-- {
			group[digit_position-1] = ASCII85_DIGIT_MINIMUM + byte(value%ASCII85_BASE)
			value /= ASCII85_BASE
		}
		encoded_group_size := tail_size + 1
		copy(
			destination[destination_position:destination_position+encoded_group_size],
			group[:encoded_group_size],
		)
		destination_position += encoded_group_size
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
	aver.Always(!bool(overlap), "ASCII85 decode source and destination do not overlap.")
	group_count := decode_bulk_unchecked(destination, source)
	decoded = Decoded_Count(int(group_count) * DECODED_GROUP_SIZE)
	consumed = Consumed_Count(int(group_count) * ENCODED_GROUP_SIZE)
	tail_decoded, tail_consumed, status := decode_tail_unchecked(
		destination[int(decoded):], source[int(consumed):], flush,
	)
	if status == STATUS_INPUT_INVALID {
		return 0, 0, status
	}
	return decoded + tail_decoded, consumed + tail_consumed, status
}

// Control syntax needs byte state after the fixed group path stops.
func decode_tail_unchecked(
	destination Decoded, source Encoded, flush Flush,
) (decoded Decoded_Count, consumed Consumed_Count, status Status) {
	defer func() {
		Decoded_Count_Invariants(decoded, "decode_tail_unchecked.decoded")
		Consumed_Count_Invariants(consumed, "decode_tail_unchecked.consumed")
		Status_Invariants(status, "decode_tail_unchecked.status")
	}()
	Decoded_Invariants(destination, "decode_tail_unchecked.destination")
	Encoded_Invariants(source, "decode_tail_unchecked.source")
	Flush_Invariants(flush, "decode_tail_unchecked.flush")
	source_position, digit_count := SIZE_MINIMUM, SIZE_MINIMUM
	value := uint32(0)
	for ; source_position < len(source); source_position++ {
		encoded_byte := source[source_position]
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
			decoded_write(destination_tail, 0, DECODED_GROUP_SIZE)
			decoded += DECODED_GROUP_SIZE
			consumed = Consumed_Count(source_position + 1)
			continue
		}
		if encoded_byte-ASCII85_DIGIT_MINIMUM >
			ASCII85_DIGIT_MAXIMUM-ASCII85_DIGIT_MINIMUM {
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
		decoded_write(destination_tail, Decoded_Word(value), DECODED_GROUP_SIZE)
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
	decoded_write(
		destination_tail, Decoded_Word(value), Decoded_Write_Count(partial_size),
	)
	decoded += Decoded_Count(partial_size)
	return decoded, Consumed_Count(len(source)), STATUS_OK
}

// Control syntax exits the fixed group path before state could be lost.
func decode_bulk_unchecked(
	destination Decoded, source Encoded,
) (group_count Bulk_Group_Count) {
	defer func() {
		Bulk_Group_Count_Invariants(group_count, "decode_bulk_unchecked.group_count")
	}()
	Decoded_Invariants(destination, "decode_bulk_unchecked.destination")
	Encoded_Invariants(source, "decode_bulk_unchecked.source")
	source_position := SIZE_MINIMUM
	destination_position := SIZE_MINIMUM
	digit_maximum := ASCII85_DIGIT_MAXIMUM - ASCII85_DIGIT_MINIMUM
	for len(source)-source_position >= ENCODED_GROUP_SIZE &&
		len(destination)-destination_position >= DECODED_GROUP_SIZE {
		source_group := source[source_position : source_position+ENCODED_GROUP_SIZE]
		if source_group[0]-ASCII85_DIGIT_MINIMUM > digit_maximum {
			break
		}
		if source_group[1]-ASCII85_DIGIT_MINIMUM > digit_maximum {
			break
		}
		if source_group[2]-ASCII85_DIGIT_MINIMUM > digit_maximum {
			break
		}
		if source_group[3]-ASCII85_DIGIT_MINIMUM > digit_maximum {
			break
		}
		if source_group[4]-ASCII85_DIGIT_MINIMUM > digit_maximum {
			break
		}
		value := uint32(source_group[0] - ASCII85_DIGIT_MINIMUM)
		value = value*ASCII85_BASE + uint32(source_group[1]-ASCII85_DIGIT_MINIMUM)
		value = value*ASCII85_BASE + uint32(source_group[2]-ASCII85_DIGIT_MINIMUM)
		value = value*ASCII85_BASE + uint32(source_group[3]-ASCII85_DIGIT_MINIMUM)
		value = value*ASCII85_BASE + uint32(source_group[4]-ASCII85_DIGIT_MINIMUM)
		destination_end := destination_position + DECODED_GROUP_SIZE
		destination_group := destination[destination_position:destination_end]
		shift := DECODED_GROUP_FINAL_INDEX * bits.BIT_COUNT_8_MAXIMUM
		destination_group[0] = byte(value >> shift)
		shift -= bits.BIT_COUNT_8_MAXIMUM
		destination_group[1] = byte(value >> shift)
		shift -= bits.BIT_COUNT_8_MAXIMUM
		destination_group[2] = byte(value >> shift)
		destination_group[3] = byte(value)
		source_position += ENCODED_GROUP_SIZE
		destination_position += DECODED_GROUP_SIZE
		group_count++
	}
	return group_count
}

func decoded_write(
	destination Decoded_Write_Destination, value Decoded_Word, count Decoded_Write_Count,
) {
	Decoded_Write_Destination_Invariants(destination, "decoded_write.destination")
	Decoded_Word_Invariants(value, "decoded_write.value")
	Decoded_Write_Count_Invariants(count, "decoded_write.count")
	for byte_position_index := range int(count) {
		byte_shift_count := DECODED_GROUP_FINAL_INDEX - byte_position_index
		shift := byte_shift_count * bits.BIT_COUNT_8_MAXIMUM
		destination[byte_position_index] = byte(value >> shift)
	}
}
