// Package utf8 supplies native UTF-8 encoding, decoding, counting, and validation.
package utf8

import (
	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/aver/default"
	"local/james-orcales/shared/unicode/ucd"
)

// REPLACEMENT_CHARACTER is the result for an invalid encoding or character.
const REPLACEMENT_CHARACTER Decoded_Character = '\uFFFD'

// CHARACTER_SELF is the first byte value that needs a multi-byte encoding.
const CHARACTER_SELF Byte = 0x80

// RUNE_MAX is the final valid Unicode code point.
const RUNE_MAX Character = '\U0010FFFF'

// UTF_MAXIMUM is the maximum byte count for one UTF-8 character.
const UTF_MAXIMUM = 4

// SEQUENCE_SIZE_MINIMUM is the size of an empty sequence.
const SEQUENCE_SIZE_MINIMUM = 0

// SEQUENCE_SIZE_MAXIMUM bounds each byte sequence that the package reads or returns.
const SEQUENCE_SIZE_MAXIMUM = 4096

// CHARACTER_SIZE_INVALID reports that a character has no valid UTF-8 encoding.
const CHARACTER_SIZE_INVALID = -1

// CHARACTER_SIZE_HOLE separates invalidity from the valid encoded sizes.
const CHARACTER_SIZE_HOLE = 0

// CHARACTER_SIZE_MINIMUM is the one-byte UTF-8 encoding size.
const CHARACTER_SIZE_MINIMUM = 1

// CHARACTER_SIZE_TWO is the two-byte UTF-8 encoding size.
const CHARACTER_SIZE_TWO = 2

// CHARACTER_SIZE_THREE is the three-byte UTF-8 encoding size.
const CHARACTER_SIZE_THREE = 3

// CHARACTER_SIZE_MAXIMUM is the four-byte UTF-8 encoding size.
const CHARACTER_SIZE_MAXIMUM = UTF_MAXIMUM

// DECODED_SIZE_MINIMUM reports that empty input consumed no bytes.
const DECODED_SIZE_MINIMUM = 0

// DECODED_SIZE_MAXIMUM is the maximum byte count that one decode consumes.
const DECODED_SIZE_MAXIMUM = UTF_MAXIMUM

// CHARACTER_COUNT_MINIMUM is the count for an empty sequence.
const CHARACTER_COUNT_MINIMUM = 0

// CHARACTER_COUNT_MAXIMUM occurs when each byte is one character or one error.
const CHARACTER_COUNT_MAXIMUM = SEQUENCE_SIZE_MAXIMUM

// CHARACTER_MINIMUM is the smallest value in rune storage.
const CHARACTER_MINIMUM int32 = bits.INTEGER_32_MINIMUM

// CHARACTER_MAXIMUM is the largest value in rune storage.
const CHARACTER_MAXIMUM int32 = bits.INTEGER_32_MAXIMUM

// DECODED_CHARACTER_MINIMUM is the first Unicode code point.
const DECODED_CHARACTER_MINIMUM int32 = 0

// DECODED_CHARACTER_MAXIMUM is the final Unicode code point.
const DECODED_CHARACTER_MAXIMUM int32 = int32(RUNE_MAX)

// SURROGATE_MINIMUM is the first UTF-16 surrogate code point.
const SURROGATE_MINIMUM Character = 0xD800

// SURROGATE_MAXIMUM is the final UTF-16 surrogate code point.
const SURROGATE_MAXIMUM Character = 0xDFFF

// FIRST_BYTE_ONE identifies the one-byte encoding prefix.
const FIRST_BYTE_ONE = 0b00000000

// CONTINUATION_BYTE identifies the continuation-byte prefix.
const CONTINUATION_BYTE = 0b10000000

// CONTINUATION_PREFIX_BIT_COUNT is the fixed-bit count in a continuation byte.
const CONTINUATION_PREFIX_BIT_COUNT = 2

// CONTINUATION_PAYLOAD_BIT_COUNT is the payload bit count in a continuation byte.
const CONTINUATION_PAYLOAD_BIT_COUNT = bits.BIT_COUNT_8_MAXIMUM - CONTINUATION_PREFIX_BIT_COUNT

// CONTINUATION_PREFIX_MASK selects the fixed continuation-byte prefix.
const CONTINUATION_PREFIX_MASK = 0b11000000

// FIRST_BYTE_TWO identifies the two-byte encoding prefix.
const FIRST_BYTE_TWO = CONTINUATION_PREFIX_MASK

// FIRST_BYTE_THREE identifies the three-byte encoding prefix.
const FIRST_BYTE_THREE = 0b11100000

// FIRST_BYTE_FOUR identifies the four-byte encoding prefix.
const FIRST_BYTE_FOUR = 0b11110000

// FIRST_BYTE_FIVE is the boundary after valid first-byte prefixes.
const FIRST_BYTE_FIVE = 0b11111000

// CONTINUATION_MASK selects the payload in a continuation byte.
const CONTINUATION_MASK = 0b00111111

// FIRST_MASK_TWO selects the payload in a two-byte first byte.
const FIRST_MASK_TWO = 0b00011111

// FIRST_MASK_THREE selects the payload in a three-byte first byte.
const FIRST_MASK_THREE = FIRST_MASK_TWO >> 1

// FIRST_MASK_FOUR selects the payload in a four-byte first byte.
const FIRST_MASK_FOUR = FIRST_MASK_THREE >> 1

// FIRST_SIZE_BIT_COUNT is the size-field width in one first-byte table entry.
const FIRST_SIZE_BIT_COUNT = 3

// FIRST_SIZE_MASK selects the size field in one first-byte table entry.
const FIRST_SIZE_MASK = 1<<FIRST_SIZE_BIT_COUNT - 1

// FIRST_ACCEPTANCE_SHIFT locates the acceptance field in one first-byte table entry.
const FIRST_ACCEPTANCE_SHIFT = bits.BIT_COUNT_8_MAXIMUM / 2

// CHARACTER_ONE_MAXIMUM is the final one-byte character.
const CHARACTER_ONE_MAXIMUM = 1<<7 - 1

// CHARACTER_TWO_MAXIMUM is the final two-byte character.
const CHARACTER_TWO_MAXIMUM = 1<<11 - 1

// CHARACTER_THREE_MAXIMUM is the final three-byte character.
const CHARACTER_THREE_MAXIMUM = 1<<16 - 1

// CONTINUATION_MINIMUM is the first continuation byte.
const CONTINUATION_MINIMUM = CONTINUATION_BYTE

// CONTINUATION_MAXIMUM is the final continuation byte.
const CONTINUATION_MAXIMUM = CONTINUATION_BYTE | CONTINUATION_MASK

// FIRST_INVALID identifies a first byte that cannot start an encoding.
const FIRST_INVALID byte = 0xF1

// FIRST_ASCII identifies a one-byte character.
const FIRST_ASCII byte = 0xF0

// FIRST_TWO identifies an unrestricted two-byte encoding.
const FIRST_TWO byte = 0x02

// FIRST_THREE_LOW_A0 raises the second-byte minimum after E0.
const FIRST_THREE_LOW_A0 byte = 0x13

// FIRST_THREE identifies an unrestricted three-byte encoding.
const FIRST_THREE byte = 0x03

// FIRST_THREE_HIGH_9F lowers the second-byte maximum after ED.
const FIRST_THREE_HIGH_9F byte = 0x23

// FIRST_FOUR_LOW_90 raises the second-byte minimum after F0.
const FIRST_FOUR_LOW_90 byte = 0x34

// FIRST_FOUR identifies an unrestricted four-byte encoding.
const FIRST_FOUR byte = 0x04

// FIRST_FOUR_HIGH_8F lowers the second-byte maximum after F4.
const FIRST_FOUR_HIGH_8F byte = 0x44

// FIRST_DATA keeps the standard decoder table immutable and directly indexable.
const FIRST_DATA = "\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0" +
	"\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0" +
	"\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0" +
	"\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0" +
	"\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0" +
	"\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0" +
	"\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0" +
	"\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0\xf0" +
	"\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1" +
	"\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1" +
	"\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1" +
	"\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1" +
	"\xf1\xf1\x02\x02\x02\x02\x02\x02\x02\x02\x02\x02\x02\x02\x02\x02" +
	"\x02\x02\x02\x02\x02\x02\x02\x02\x02\x02\x02\x02\x02\x02\x02\x02" +
	"\x13\x03\x03\x03\x03\x03\x03\x03\x03\x03\x03\x03\x03\x23\x03\x03" +
	"\x34\x04\x04\x04\x44\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1"

// ACCEPTANCE_MINIMUM_DATA supplies the lower bound for each acceptance interval.
const ACCEPTANCE_MINIMUM_DATA = "\x80\xa0\x80\x90\x80\x00\x00\x00" +
	"\x00\x00\x00\x00\x00\x00\x00\x00"

// ACCEPTANCE_MAXIMUM_DATA supplies the upper bound for each acceptance interval.
const ACCEPTANCE_MAXIMUM_DATA = "\xbf\xbf\x9f\xbf\x8f\x00\x00\x00" +
	"\x00\x00\x00\x00\x00\x00\x00\x00"

// POINTER_SIZE is the native machine-word size in bytes.
const POINTER_SIZE = bits.WORD_SIZE / bits.BIT_COUNT_8_MAXIMUM

// HIGH_BITS selects the high bit from each byte in one native machine word.
const HIGH_BITS = 0x8080808080808080 >>
	(bits.BIT_COUNT_64_MAXIMUM - bits.BIT_COUNT_8_MAXIMUM*POINTER_SIZE)

// REPLACEMENT_BYTE_ZERO is the first byte of REPLACEMENT_CHARACTER.
const REPLACEMENT_BYTE_ZERO byte = FIRST_BYTE_THREE |
	byte(rune(REPLACEMENT_CHARACTER)>>(2*CONTINUATION_PAYLOAD_BIT_COUNT))

// REPLACEMENT_BYTE_ONE is the second byte of REPLACEMENT_CHARACTER.
const REPLACEMENT_BYTE_ONE byte = CONTINUATION_BYTE |
	byte((rune(REPLACEMENT_CHARACTER)>>CONTINUATION_PAYLOAD_BIT_COUNT)&
		CONTINUATION_MASK)

// REPLACEMENT_BYTE_TWO is the third byte of REPLACEMENT_CHARACTER.
const REPLACEMENT_BYTE_TWO byte = CONTINUATION_BYTE |
	byte(rune(REPLACEMENT_CHARACTER)&CONTINUATION_MASK)

// Bytes is one bounded byte sequence.
type Bytes []byte

// Bytes_Invariants applies the package sequence limit.
func Bytes_Invariants(value Bytes, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), SEQUENCE_SIZE_MINIMUM, SEQUENCE_SIZE_MAXIMUM).
		Ensure()
}

// Text is one bounded text sequence.
type Text string

// Text_Invariants applies the package sequence limit.
func Text_Invariants(value Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), SEQUENCE_SIZE_MINIMUM, SEQUENCE_SIZE_MAXIMUM).
		Ensure()
}

// Byte is one possible UTF-8 byte.
type Byte byte

// Byte_Invariants covers all byte values.
func Byte_Invariants(value Byte, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), bits.WORD_8_MINIMUM, bits.WORD_8_MAXIMUM).
		Ensure()
}

// Character is one rune-storage value that an encoder reads.
type Character rune

// Character_Invariants covers the complete rune-storage domain.
func Character_Invariants(value Character, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), CHARACTER_MINIMUM, CHARACTER_MAXIMUM).
		Ensure()
}

// Decoded_Character is one valid Unicode code point that a decoder returns.
type Decoded_Character rune

// Decoded_Character_Invariants covers all valid decoded code points.
func Decoded_Character_Invariants(
	value Decoded_Character, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int32(
			int32(value), DECODED_CHARACTER_MINIMUM, DECODED_CHARACTER_MAXIMUM,
		).
		Ensure()
}

// Size is a valid encoded size or CHARACTER_SIZE_INVALID.
type Size int

// Size_Invariants covers all UTF-8 character sizes and invalidity.
func Size_Invariants(value Size, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Holed_Int(
			int(value), CHARACTER_SIZE_INVALID, CHARACTER_SIZE_MAXIMUM,
			CHARACTER_SIZE_HOLE, CHARACTER_SIZE_HOLE,
			CHARACTER_SIZE_HOLE, CHARACTER_SIZE_HOLE,
		).
		Ensure()
}

// Decoded_Size is the byte count that one decode consumes.
type Decoded_Size int

// Decoded_Size_Invariants covers empty input and all UTF-8 encoding sizes.
func Decoded_Size_Invariants(value Decoded_Size, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), DECODED_SIZE_MINIMUM, DECODED_SIZE_MAXIMUM).
		Ensure()
}

// Encoded_Size is the byte count that one encoding writes.
type Encoded_Size int

// Encoded_Size_Invariants lists all UTF-8 encoding sizes.
func Encoded_Size_Invariants(value Encoded_Size, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Int(
			int(value), CHARACTER_SIZE_MINIMUM, CHARACTER_SIZE_TWO,
			CHARACTER_SIZE_THREE, CHARACTER_SIZE_MAXIMUM,
		).
		Ensure()
}

// Count is the number of decoded characters in one sequence.
type Count int

// Count_Invariants applies the maximum one-character-per-byte count.
func Count_Invariants(value Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), CHARACTER_COUNT_MINIMUM, CHARACTER_COUNT_MAXIMUM).
		Ensure()
}

// Nonempty_Bytes is one append result, which always contains a new encoding.
type Nonempty_Bytes []byte

// Nonempty_Bytes_Invariants excludes an empty append result.
func Nonempty_Bytes_Invariants(value Nonempty_Bytes, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), CHARACTER_SIZE_MINIMUM, SEQUENCE_SIZE_MAXIMUM).
		Ensure()
}

// Boolean gives each UTF-8 report its own coverage identity.
type Boolean bool

// Boolean_Invariants requires both report values.
func Boolean_Invariants(value Boolean, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "A UTF-8 report is true.").
		Ensure()
}

// Full_Character reports whether the first bytes contain one complete encoding.
func Full_Character(source Bytes) (complete Boolean) {
	defer func() { Boolean_Invariants(complete, "full_character.complete") }()
	Bytes_Invariants(source, "full_character.source")
	count := len(source)
	if count == SEQUENCE_SIZE_MINIMUM {
		return false
	}
	first := FIRST_DATA[source[0]]
	if count >= int(first&FIRST_SIZE_MASK) {
		return true
	}
	acceptance := first >> FIRST_ACCEPTANCE_SHIFT
	if count > CHARACTER_SIZE_MINIMUM {
		if source[1] < ACCEPTANCE_MINIMUM_DATA[acceptance] {
			return true
		}
		if ACCEPTANCE_MAXIMUM_DATA[acceptance] < source[1] {
			return true
		}
	}
	if count > CHARACTER_SIZE_TWO {
		if source[2] < CONTINUATION_MINIMUM {
			return true
		}
		if CONTINUATION_MAXIMUM < source[2] {
			return true
		}
	}
	return false
}

// Full_Character_Text reports whether the first text bytes contain one complete encoding.
func Full_Character_Text(source Text) (complete Boolean) {
	defer func() { Boolean_Invariants(complete, "full_character_text.complete") }()
	Text_Invariants(source, "full_character_text.source")
	count := len(source)
	if count == SEQUENCE_SIZE_MINIMUM {
		return false
	}
	first := FIRST_DATA[source[0]]
	if count >= int(first&FIRST_SIZE_MASK) {
		return true
	}
	acceptance := first >> FIRST_ACCEPTANCE_SHIFT
	if count > CHARACTER_SIZE_MINIMUM {
		if source[1] < ACCEPTANCE_MINIMUM_DATA[acceptance] {
			return true
		}
		if ACCEPTANCE_MAXIMUM_DATA[acceptance] < source[1] {
			return true
		}
	}
	if count > CHARACTER_SIZE_TWO {
		if source[2] < CONTINUATION_MINIMUM {
			return true
		}
		if CONTINUATION_MAXIMUM < source[2] {
			return true
		}
	}
	return false
}

// Decode_Character decodes the first character in a byte sequence.
func Decode_Character(source Bytes) (
	character Decoded_Character, size Decoded_Size,
) {
	defer func() {
		Decoded_Character_Invariants(character, "decode_character.character")
		Decoded_Size_Invariants(size, "decode_character.size")
	}()
	Bytes_Invariants(source, "decode_character.source")
	for _, value := range source {
		if value < byte(CHARACTER_SELF) {
			return Decoded_Character(value), CHARACTER_SIZE_MINIMUM
		}
		break
	}
	if len(source) < CHARACTER_SIZE_MINIMUM {
		return REPLACEMENT_CHARACTER, DECODED_SIZE_MINIMUM
	}
	first_byte := source[0]
	first := FIRST_DATA[first_byte]
	if first >= FIRST_ASCII {
		mask := Decoded_Character(first) << (bits.BIT_COUNT_32_MAXIMUM - 1) >>
			(bits.BIT_COUNT_32_MAXIMUM - 1)
		return Decoded_Character(first_byte)&^mask |
			Decoded_Character(REPLACEMENT_CHARACTER)&mask, CHARACTER_SIZE_MINIMUM
	}
	encoded_size := int(first & FIRST_SIZE_MASK)
	acceptance := first >> FIRST_ACCEPTANCE_SHIFT
	if len(source) < encoded_size {
		return REPLACEMENT_CHARACTER, CHARACTER_SIZE_MINIMUM
	}
	second := source[1]
	if second < ACCEPTANCE_MINIMUM_DATA[acceptance] {
		return REPLACEMENT_CHARACTER, CHARACTER_SIZE_MINIMUM
	}
	if ACCEPTANCE_MAXIMUM_DATA[acceptance] < second {
		return REPLACEMENT_CHARACTER, CHARACTER_SIZE_MINIMUM
	}
	if encoded_size <= CHARACTER_SIZE_TWO {
		return Decoded_Character(first_byte&FIRST_MASK_TWO)<<
			CONTINUATION_PAYLOAD_BIT_COUNT |
			Decoded_Character(second&CONTINUATION_MASK), CHARACTER_SIZE_TWO
	}
	third := source[2]
	if third < CONTINUATION_MINIMUM {
		return REPLACEMENT_CHARACTER, CHARACTER_SIZE_MINIMUM
	}
	if CONTINUATION_MAXIMUM < third {
		return REPLACEMENT_CHARACTER, CHARACTER_SIZE_MINIMUM
	}
	if encoded_size <= CHARACTER_SIZE_THREE {
		return Decoded_Character(first_byte&FIRST_MASK_THREE)<<
			(2*CONTINUATION_PAYLOAD_BIT_COUNT) |
			Decoded_Character(second&CONTINUATION_MASK)<<
				CONTINUATION_PAYLOAD_BIT_COUNT |
			Decoded_Character(third&CONTINUATION_MASK), CHARACTER_SIZE_THREE
	}
	fourth := source[3]
	if fourth < CONTINUATION_MINIMUM {
		return REPLACEMENT_CHARACTER, CHARACTER_SIZE_MINIMUM
	}
	if CONTINUATION_MAXIMUM < fourth {
		return REPLACEMENT_CHARACTER, CHARACTER_SIZE_MINIMUM
	}
	return Decoded_Character(first_byte&FIRST_MASK_FOUR)<<
		(3*CONTINUATION_PAYLOAD_BIT_COUNT) |
		Decoded_Character(second&CONTINUATION_MASK)<<
			(2*CONTINUATION_PAYLOAD_BIT_COUNT) |
		Decoded_Character(third&CONTINUATION_MASK)<<
			CONTINUATION_PAYLOAD_BIT_COUNT |
		Decoded_Character(fourth&CONTINUATION_MASK), CHARACTER_SIZE_MAXIMUM
}

// Decode_Character_Text decodes the first character in text.
func Decode_Character_Text(source Text) (
	character Decoded_Character, size Decoded_Size,
) {
	defer func() {
		Decoded_Character_Invariants(character, "decode_character_text.character")
		Decoded_Size_Invariants(size, "decode_character_text.size")
	}()
	Text_Invariants(source, "decode_character_text.source")
	if source != "" {
		if source[0] < byte(CHARACTER_SELF) {
			return Decoded_Character(source[0]), CHARACTER_SIZE_MINIMUM
		}
	}
	if len(source) < CHARACTER_SIZE_MINIMUM {
		return REPLACEMENT_CHARACTER, DECODED_SIZE_MINIMUM
	}
	first_byte := source[0]
	first := FIRST_DATA[first_byte]
	if first >= FIRST_ASCII {
		mask := Decoded_Character(first) << (bits.BIT_COUNT_32_MAXIMUM - 1) >>
			(bits.BIT_COUNT_32_MAXIMUM - 1)
		return Decoded_Character(first_byte)&^mask |
			Decoded_Character(REPLACEMENT_CHARACTER)&mask, CHARACTER_SIZE_MINIMUM
	}
	encoded_size := int(first & FIRST_SIZE_MASK)
	acceptance := first >> FIRST_ACCEPTANCE_SHIFT
	if len(source) < encoded_size {
		return REPLACEMENT_CHARACTER, CHARACTER_SIZE_MINIMUM
	}
	second := source[1]
	if second < ACCEPTANCE_MINIMUM_DATA[acceptance] {
		return REPLACEMENT_CHARACTER, CHARACTER_SIZE_MINIMUM
	}
	if ACCEPTANCE_MAXIMUM_DATA[acceptance] < second {
		return REPLACEMENT_CHARACTER, CHARACTER_SIZE_MINIMUM
	}
	if encoded_size <= CHARACTER_SIZE_TWO {
		return Decoded_Character(first_byte&FIRST_MASK_TWO)<<
			CONTINUATION_PAYLOAD_BIT_COUNT |
			Decoded_Character(second&CONTINUATION_MASK), CHARACTER_SIZE_TWO
	}
	third := source[2]
	if third < CONTINUATION_MINIMUM {
		return REPLACEMENT_CHARACTER, CHARACTER_SIZE_MINIMUM
	}
	if CONTINUATION_MAXIMUM < third {
		return REPLACEMENT_CHARACTER, CHARACTER_SIZE_MINIMUM
	}
	if encoded_size <= CHARACTER_SIZE_THREE {
		return Decoded_Character(first_byte&FIRST_MASK_THREE)<<
			(2*CONTINUATION_PAYLOAD_BIT_COUNT) |
			Decoded_Character(second&CONTINUATION_MASK)<<
				CONTINUATION_PAYLOAD_BIT_COUNT |
			Decoded_Character(third&CONTINUATION_MASK), CHARACTER_SIZE_THREE
	}
	fourth := source[3]
	if fourth < CONTINUATION_MINIMUM {
		return REPLACEMENT_CHARACTER, CHARACTER_SIZE_MINIMUM
	}
	if CONTINUATION_MAXIMUM < fourth {
		return REPLACEMENT_CHARACTER, CHARACTER_SIZE_MINIMUM
	}
	return Decoded_Character(first_byte&FIRST_MASK_FOUR)<<
		(3*CONTINUATION_PAYLOAD_BIT_COUNT) |
		Decoded_Character(second&CONTINUATION_MASK)<<
			(2*CONTINUATION_PAYLOAD_BIT_COUNT) |
		Decoded_Character(third&CONTINUATION_MASK)<<
			CONTINUATION_PAYLOAD_BIT_COUNT |
		Decoded_Character(fourth&CONTINUATION_MASK), CHARACTER_SIZE_MAXIMUM
}

// Decode_Final_Character decodes the final character in a byte sequence.
func Decode_Final_Character(source Bytes) (
	character Decoded_Character, size Decoded_Size,
) {
	defer func() {
		Decoded_Character_Invariants(character, "decode_final_character.character")
		Decoded_Size_Invariants(size, "decode_final_character.size")
	}()
	Bytes_Invariants(source, "decode_final_character.source")
	end_count := len(source)
	if end_count == SEQUENCE_SIZE_MINIMUM {
		return REPLACEMENT_CHARACTER, DECODED_SIZE_MINIMUM
	}
	start := end_count - 1
	if source[start] < byte(CHARACTER_SELF) {
		return Decoded_Character(source[start]), CHARACTER_SIZE_MINIMUM
	}
	minimum := max(end_count-UTF_MAXIMUM, SEQUENCE_SIZE_MINIMUM)
	for start--; start >= minimum; start-- {
		if Character_Start(Byte(source[start])) {
			break
		}
	}
	if start < 0 {
		start = 0
	}
	character, size = Decode_Character(source[start:end_count])
	if start+int(size) != end_count {
		return REPLACEMENT_CHARACTER, CHARACTER_SIZE_MINIMUM
	}
	return character, size
}

// Decode_Final_Character_Text decodes the final character in text.
func Decode_Final_Character_Text(source Text) (
	character Decoded_Character, size Decoded_Size,
) {
	defer func() {
		Decoded_Character_Invariants(character, "decode_final_character_text.character")
		Decoded_Size_Invariants(size, "decode_final_character_text.size")
	}()
	Text_Invariants(source, "decode_final_character_text.source")
	end_count := len(source)
	if end_count == SEQUENCE_SIZE_MINIMUM {
		return REPLACEMENT_CHARACTER, DECODED_SIZE_MINIMUM
	}
	start := end_count - 1
	if source[start] < byte(CHARACTER_SELF) {
		return Decoded_Character(source[start]), CHARACTER_SIZE_MINIMUM
	}
	minimum := max(end_count-UTF_MAXIMUM, SEQUENCE_SIZE_MINIMUM)
	for start--; start >= minimum; start-- {
		if Character_Start(Byte(source[start])) {
			break
		}
	}
	if start < 0 {
		start = 0
	}
	character, size = Decode_Character_Text(source[start:end_count])
	if start+int(size) != end_count {
		return REPLACEMENT_CHARACTER, CHARACTER_SIZE_MINIMUM
	}
	return character, size
}

// Character_Size returns the byte count for one character or CHARACTER_SIZE_INVALID.
func Character_Size(character Character) (size Size) {
	defer func() { Size_Invariants(size, "character_size.size") }()
	Character_Invariants(character, "character_size.character")
	switch {
	case character < 0:
		return CHARACTER_SIZE_INVALID
	case character <= CHARACTER_ONE_MAXIMUM:
		return CHARACTER_SIZE_MINIMUM
	case character <= CHARACTER_TWO_MAXIMUM:
		return CHARACTER_SIZE_TWO
	case SURROGATE_MINIMUM <= character && character <= SURROGATE_MAXIMUM:
		return CHARACTER_SIZE_INVALID
	case character <= CHARACTER_THREE_MAXIMUM:
		return CHARACTER_SIZE_THREE
	case character <= RUNE_MAX:
		return CHARACTER_SIZE_MAXIMUM
	default:
		return CHARACTER_SIZE_INVALID
	}
}

// Encode_Character writes one UTF-8 encoding into caller storage.
func Encode_Character(buffer Bytes, character Character) (size Encoded_Size) {
	defer func() { Encoded_Size_Invariants(size, "encode_character.size") }()
	Bytes_Invariants(buffer, "encode_character.buffer")
	Character_Invariants(character, "encode_character.character")
	if uint32(character) <= CHARACTER_ONE_MAXIMUM {
		buffer[0] = byte(character)
		return CHARACTER_SIZE_MINIMUM
	}
	unsigned := uint32(character)
	switch {
	case unsigned <= CHARACTER_TWO_MAXIMUM:
		buffer[0] = FIRST_BYTE_TWO |
			byte(character>>CONTINUATION_PAYLOAD_BIT_COUNT)
		buffer[1] = CONTINUATION_BYTE | byte(character)&CONTINUATION_MASK
		return CHARACTER_SIZE_TWO
	case unsigned < uint32(SURROGATE_MINIMUM),
		uint32(SURROGATE_MAXIMUM) < unsigned && unsigned <= CHARACTER_THREE_MAXIMUM:
		buffer[0] = FIRST_BYTE_THREE |
			byte(character>>(2*CONTINUATION_PAYLOAD_BIT_COUNT))
		buffer[1] = CONTINUATION_BYTE |
			byte(character>>CONTINUATION_PAYLOAD_BIT_COUNT)&CONTINUATION_MASK
		buffer[2] = CONTINUATION_BYTE | byte(character)&CONTINUATION_MASK
		return CHARACTER_SIZE_THREE
	case unsigned > CHARACTER_THREE_MAXIMUM && unsigned <= uint32(RUNE_MAX):
		buffer[0] = FIRST_BYTE_FOUR |
			byte(character>>(3*CONTINUATION_PAYLOAD_BIT_COUNT))
		buffer[1] = CONTINUATION_BYTE |
			byte(character>>(2*CONTINUATION_PAYLOAD_BIT_COUNT))&CONTINUATION_MASK
		buffer[2] = CONTINUATION_BYTE |
			byte(character>>CONTINUATION_PAYLOAD_BIT_COUNT)&CONTINUATION_MASK
		buffer[3] = CONTINUATION_BYTE | byte(character)&CONTINUATION_MASK
		return CHARACTER_SIZE_MAXIMUM
	default:
		buffer[0] = REPLACEMENT_BYTE_ZERO
		buffer[1] = REPLACEMENT_BYTE_ONE
		buffer[2] = REPLACEMENT_BYTE_TWO
		return CHARACTER_SIZE_THREE
	}
}

// Append_Character adds one UTF-8 encoding within caller-owned capacity.
func Append_Character(buffer Bytes, character Character) (result Nonempty_Bytes) {
	defer func() {
		Nonempty_Bytes_Invariants(result, "append_character.result")
	}()
	Bytes_Invariants(buffer, "append_character.buffer")
	Character_Invariants(character, "append_character.character")
	encoded_size := int(Character_Size(character))
	if encoded_size == CHARACTER_SIZE_INVALID {
		encoded_size = CHARACTER_SIZE_THREE
	}
	result_size := len(buffer) + encoded_size
	aver.Always(
		result_size <= SEQUENCE_SIZE_MAXIMUM,
		"A UTF-8 append result does not exceed the sequence limit.",
	)
	aver.Always(
		result_size <= cap(buffer),
		"Caller storage holds the appended UTF-8 encoding.",
	)
	result = Nonempty_Bytes(buffer[:result_size])
	Encode_Character(Bytes(result[len(buffer):]), character)
	return result
}

// Character_Count returns the character count in a byte sequence.
func Character_Count(source Bytes) (count Count) {
	defer func() { Count_Invariants(count, "character_count.count") }()
	Bytes_Invariants(source, "character_count.source")
	byte_count := len(source)
	for ; int(count) < byte_count; count++ {
		if source[count] >= byte(CHARACTER_SELF) {
			return count + Character_Count_Text(Text(string(source[count:])))
		}
	}
	return count
}

// Character_Count_Text returns the character count in text.
func Character_Count_Text(source Text) (count Count) {
	defer func() { Count_Invariants(count, "character_count_text.count") }()
	Text_Invariants(source, "character_count_text.source")
	for range source {
		count++
	}
	return count
}

// Character_Start reports whether a byte can start an encoded character.
func Character_Start(value Byte) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "character_start.yes") }()
	Byte_Invariants(value, "character_start.value")
	return Boolean(value&CONTINUATION_PREFIX_MASK != CONTINUATION_BYTE)
}

// Valid reports whether a byte sequence contains only valid UTF-8 encodings.
func Valid(source Bytes) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "valid.yes") }()
	Bytes_Invariants(source, "valid.source")
	for len(source) > SEQUENCE_SIZE_MINIMUM {
		first_byte := source[0]
		if first_byte < byte(CHARACTER_SELF) {
			source = source[CHARACTER_SIZE_MINIMUM:]
			for len(source) > POINTER_SIZE {
				word := uintptr(source[0]) |
					uintptr(source[1])<<bits.BIT_COUNT_8_MAXIMUM |
					uintptr(source[2])<<(2*bits.BIT_COUNT_8_MAXIMUM) |
					uintptr(source[3])<<(3*bits.BIT_COUNT_8_MAXIMUM)
				if bits.WORD_SIZE != bits.BIT_COUNT_32_MAXIMUM {
					word |= uintptr(source[4]) << (4 * bits.BIT_COUNT_8_MAXIMUM)
					word |= uintptr(source[5]) << (5 * bits.BIT_COUNT_8_MAXIMUM)
					word |= uintptr(source[6]) << (6 * bits.BIT_COUNT_8_MAXIMUM)
					word |= uintptr(source[7]) << (7 * bits.BIT_COUNT_8_MAXIMUM)
				}
				if word&HIGH_BITS != 0 {
					break
				}
				source = source[POINTER_SIZE:]
			}
			continue
		}
		_, size := Decode_Character(source)
		if size == CHARACTER_SIZE_MINIMUM {
			return false
		}
		source = source[size:]
	}
	return true
}

// Valid_Text reports whether text contains only valid UTF-8 encodings.
func Valid_Text(source Text) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "valid_text.yes") }()
	Text_Invariants(source, "valid_text.source")
	for len(source) > SEQUENCE_SIZE_MINIMUM {
		first_byte := source[0]
		if first_byte < byte(CHARACTER_SELF) {
			source = source[CHARACTER_SIZE_MINIMUM:]
			for len(source) > POINTER_SIZE {
				word := uintptr(source[0]) |
					uintptr(source[1])<<bits.BIT_COUNT_8_MAXIMUM |
					uintptr(source[2])<<(2*bits.BIT_COUNT_8_MAXIMUM) |
					uintptr(source[3])<<(3*bits.BIT_COUNT_8_MAXIMUM)
				if bits.WORD_SIZE != bits.BIT_COUNT_32_MAXIMUM {
					word |= uintptr(source[4]) << (4 * bits.BIT_COUNT_8_MAXIMUM)
					word |= uintptr(source[5]) << (5 * bits.BIT_COUNT_8_MAXIMUM)
					word |= uintptr(source[6]) << (6 * bits.BIT_COUNT_8_MAXIMUM)
					word |= uintptr(source[7]) << (7 * bits.BIT_COUNT_8_MAXIMUM)
				}
				if word&HIGH_BITS != 0 {
					break
				}
				source = source[POINTER_SIZE:]
			}
			continue
		}
		_, size := Decode_Character_Text(source)
		if size == CHARACTER_SIZE_MINIMUM {
			return false
		}
		source = source[size:]
	}
	return true
}

// Valid_Character reports whether a character has a legal UTF-8 encoding.
func Valid_Character(character Character) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "valid_character.yes") }()
	Character_Invariants(character, "valid_character.character")
	return Boolean(
		0 <= character && character < SURROGATE_MINIMUM ||
			SURROGATE_MAXIMUM < character && character <= RUNE_MAX,
	)
}

// BYTE_INDEX_ABSENT separates a failed search from the first encoded byte.
const BYTE_INDEX_ABSENT = -1

// BYTE_INDEX_MAXIMUM prevents an encoded offset from escaping bounded input.
const BYTE_INDEX_MAXIMUM = SEQUENCE_SIZE_MAXIMUM - 1

// BYTE_COUNT_MINIMUM admits transforms that produce no encoding.
const BYTE_COUNT_MINIMUM = SEQUENCE_SIZE_MINIMUM

// BYTE_COUNT_MAXIMUM keeps transformed output inside one bounded sequence.
const BYTE_COUNT_MAXIMUM = SEQUENCE_SIZE_MAXIMUM

// FIELD_COUNT_MINIMUM admits input containing no fields.
const FIELD_COUNT_MINIMUM = SEQUENCE_SIZE_MINIMUM

// FIELD_COUNT_MAXIMUM follows alternating one-byte fields and separators.
const FIELD_COUNT_MAXIMUM = (SEQUENCE_SIZE_MAXIMUM + 1) / 2

// Byte_Index identifies encoded character start, not decoded character position.
type Byte_Index int

// Byte_Index_Invariants admits absence beside every sequence byte.
func Byte_Index_Invariants(value Byte_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), BYTE_INDEX_ABSENT, BYTE_INDEX_MAXIMUM).
		Ensure()
}

// Byte_Count measures encoded output without confusing bytes with characters.
type Byte_Count int

// Byte_Count_Invariants bounds output to caller sequence storage.
func Byte_Count_Invariants(value Byte_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), BYTE_COUNT_MINIMUM, BYTE_COUNT_MAXIMUM).
		Ensure()
}

// Field_Slices holds UTF-8 views between character-delimiter runs.
type Field_Slices []Bytes

// Field_Slices_Invariants bounds alternating one-byte fields and delimiters.
func Field_Slices_Invariants(value Field_Slices, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), FIELD_COUNT_MINIMUM, FIELD_COUNT_MAXIMUM).
		Ensure()
}

// Field_Count measures populated field storage.
type Field_Count int

// Field_Count_Invariants shares exact field bound with caller storage.
func Field_Count_Invariants(value Field_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), FIELD_COUNT_MINIMUM, FIELD_COUNT_MAXIMUM).
		Ensure()
}

// Characters holds decoded sequence values in caller storage.
type Characters []Decoded_Character

// Characters_Invariants permits one replacement character per invalid byte.
func Characters_Invariants(value Characters, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), CHARACTER_COUNT_MINIMUM, CHARACTER_COUNT_MAXIMUM).
		Ensure()
}

// Yield_Function receives one clipped source view until it rejects continuation.
type Yield_Function func(content Bytes) (continued Boolean)

// Predicate keeps character interpretation inside UTF-8 operations.
type Predicate func(character rune) (matches bool)

// Mapping keeps decoded input and encoded output joined by one contract.
type Mapping func(character rune) (mapped rune)

// Buffer_Write_Character appends one UTF-8 encoding to raw caller storage.
func Buffer_Write_Character(
	buffer bytes.Buffer_Handle, character Character,
) (count Encoded_Size) {
	defer func() { Encoded_Size_Invariants(count, "buffer_write_character.count") }()
	bytes.Buffer_Handle_Invariants(buffer, "buffer_write_character.buffer")
	Character_Invariants(character, "buffer_write_character.character")
	var storage [UTF_MAXIMUM]byte
	count = Encode_Character(storage[:], character)
	bytes.Buffer_Write(buffer, storage[:count])
	return count
}

// Buffer_Read_Character decodes and consumes one character from raw Buffer storage.
func Buffer_Read_Character(
	buffer bytes.Buffer_Handle,
) (character Decoded_Character, size Decoded_Size, found Boolean) {
	defer func() {
		Decoded_Character_Invariants(character, "buffer_read_character.character")
		Decoded_Size_Invariants(size, "buffer_read_character.size")
		Boolean_Invariants(found, "buffer_read_character.found")
	}()
	bytes.Buffer_Handle_Invariants(buffer, "buffer_read_character.buffer")
	source := bytes.Buffer_Bytes(buffer)
	if len(source) == 0 {
		bytes.Buffer_Reset(buffer)
		return 0, 0, false
	}
	character, size = Decode_Character(Bytes(source))
	bytes.Buffer_Next(buffer, bytes.Boundary(size))
	buffer.Operation = bytes.Read_Operation(size)
	return character, size, true
}

// Buffer_Unread_Character restores character boundary recorded by UTF-8 read.
func Buffer_Unread_Character(buffer bytes.Buffer_Handle) {
	bytes.Buffer_Handle_Invariants(buffer, "buffer_unread_character.buffer")
	aver.Always(
		buffer.Operation > 0,
		"UTF-8 Buffer unread follows character read.",
	)
	size := Encoded_Size(buffer.Operation)
	Encoded_Size_Invariants(size, "buffer_unread_character.size")
	buffer.Position -= bytes.Boundary(size)
	buffer.Operation = 0
}

// Reader_Read_Character decodes and consumes one character from raw Reader storage.
func Reader_Read_Character(
	reader bytes.Reader_Handle,
) (character Decoded_Character, size Decoded_Size, found Boolean) {
	defer func() {
		Decoded_Character_Invariants(character, "reader_read_character.character")
		Decoded_Size_Invariants(size, "reader_read_character.size")
		Boolean_Invariants(found, "reader_read_character.found")
	}()
	bytes.Reader_Handle_Invariants(reader, "reader_read_character.reader")
	if reader.Position >= bytes.Reader_Position(len(reader.Source)) {
		reader.Previous = bytes.INDEX_ABSENT
		return 0, 0, false
	}
	reader.Previous = bytes.Index_Value(reader.Position)
	character, size = Decode_Character(Bytes(reader.Source[reader.Position:]))
	reader.Position += bytes.Reader_Position(size)
	return character, size, true
}

// Reader_Unread_Character restores boundary recorded by UTF-8 read.
func Reader_Unread_Character(reader bytes.Reader_Handle) {
	bytes.Reader_Handle_Invariants(reader, "reader_unread_character.reader")
	aver.Always(
		reader.Previous >= 0,
		"UTF-8 Reader unread follows character read.",
	)
	aver.Always(
		reader.Previous < bytes.Index_Value(reader.Position),
		"UTF-8 Reader unread restores earlier boundary.",
	)
	reader.Position = bytes.Reader_Position(reader.Previous)
	reader.Previous = bytes.INDEX_ABSENT
}

// Contains_Any reports whether source contains one character from set.
func Contains_Any(source Bytes, characters Text) (contained Boolean) {
	defer func() { Boolean_Invariants(contained, "contains_any.contained") }()
	Bytes_Invariants(source, "contains_any.source")
	Text_Invariants(characters, "contains_any.characters")
	return Boolean(Index_Any(source, characters) >= 0)
}

// Contains_Rune reports whether source contains character.
func Contains_Rune(source Bytes, character Character) (contained Boolean) {
	defer func() { Boolean_Invariants(contained, "contains_rune.contained") }()
	Bytes_Invariants(source, "contains_rune.source")
	Character_Invariants(character, "contains_rune.character")
	return Boolean(Index_Rune(source, character) >= 0)
}

// Contains_Function reports whether one decoded character satisfies predicate.
func Contains_Function(
	source Bytes, predicate Predicate,
) (contained Boolean) {
	defer func() { Boolean_Invariants(contained, "contains_function.contained") }()
	Bytes_Invariants(source, "contains_function.source")
	return Boolean(Index_Function(source, predicate) >= 0)
}

// Index_Rune returns encoded byte index of character.
func Index_Rune(source Bytes, character Character) (index Byte_Index) {
	defer func() { Byte_Index_Invariants(index, "index_rune.index") }()
	Bytes_Invariants(source, "index_rune.source")
	Character_Invariants(character, "index_rune.character")
	if !Valid_Character(character) {
		return BYTE_INDEX_ABSENT
	}
	for source_index := 0; source_index < len(source); {
		source_character, size := Decode_Character(source[source_index:])
		if Character(source_character) == character {
			return Byte_Index(source_index)
		}
		source_index += int(size)
	}
	return BYTE_INDEX_ABSENT
}

// Index_Any returns encoded byte index of first character from set.
func Index_Any(source Bytes, characters Text) (index Byte_Index) {
	defer func() { Byte_Index_Invariants(index, "index_any.index") }()
	Bytes_Invariants(source, "index_any.source")
	Text_Invariants(characters, "index_any.characters")
	for source_index := 0; source_index < len(source); {
		character, size := Decode_Character(source[source_index:])
		if text_contains_character(characters, character) {
			return Byte_Index(source_index)
		}
		source_index += int(size)
	}
	return BYTE_INDEX_ABSENT
}

// Last_Index_Any returns encoded byte index of final character from set.
func Last_Index_Any(source Bytes, characters Text) (index Byte_Index) {
	defer func() { Byte_Index_Invariants(index, "last_index_any.index") }()
	Bytes_Invariants(source, "last_index_any.source")
	Text_Invariants(characters, "last_index_any.characters")
	for boundary_count := len(source); boundary_count > 0; {
		character, size := Decode_Final_Character(source[:boundary_count])
		boundary_count -= int(size)
		if text_contains_character(characters, character) {
			return Byte_Index(boundary_count)
		}
	}
	return BYTE_INDEX_ABSENT
}

// Index_Function returns encoded byte index of first predicate match.
func Index_Function(
	source Bytes, predicate Predicate,
) (index Byte_Index) {
	defer func() { Byte_Index_Invariants(index, "index_function.index") }()
	Bytes_Invariants(source, "index_function.source")
	for source_index := 0; source_index < len(source); {
		character, size := Decode_Character(source[source_index:])
		if predicate(rune(character)) {
			return Byte_Index(source_index)
		}
		source_index += int(size)
	}
	return BYTE_INDEX_ABSENT
}

// Last_Index_Function returns encoded byte index of final predicate match.
func Last_Index_Function(
	source Bytes, predicate Predicate,
) (index Byte_Index) {
	defer func() { Byte_Index_Invariants(index, "last_index_function.index") }()
	Bytes_Invariants(source, "last_index_function.source")
	for boundary_count := len(source); boundary_count > 0; {
		character, size := Decode_Final_Character(source[:boundary_count])
		boundary_count -= int(size)
		if predicate(rune(character)) {
			return Byte_Index(boundary_count)
		}
	}
	return BYTE_INDEX_ABSENT
}

// Fields_Into writes Unicode-space-delimited source views into caller storage.
func Fields_Into(destination Field_Slices, source Bytes) (count Field_Count) {
	defer func() { Field_Count_Invariants(count, "fields_into.count") }()
	Field_Slices_Invariants(destination, "fields_into.destination")
	Bytes_Invariants(source, "fields_into.source")
	return fields_into(destination, source, func(character rune) (matches bool) {
		return bool(ucd.Is_Space(ucd.Character(character)))
	})
}

// Fields_Function_Into writes predicate-delimited source views into caller storage.
func Fields_Function_Into(
	destination Field_Slices, source Bytes, predicate Predicate,
) (count Field_Count) {
	defer func() { Field_Count_Invariants(count, "fields_function_into.count") }()
	Field_Slices_Invariants(destination, "fields_function_into.destination")
	Bytes_Invariants(source, "fields_function_into.source")
	return fields_into(destination, source, predicate)
}

// Map_Into writes mapped characters into separate caller storage.
func Map_Into(
	destination Bytes, mapping Mapping, source Bytes,
) (count Byte_Count) {
	defer func() { Byte_Count_Invariants(count, "map_into.count") }()
	Bytes_Invariants(destination, "map_into.destination")
	Bytes_Invariants(source, "map_into.source")
	aver.Always(
		!bytes.Overlap(bytes.Slice(destination), bytes.Slice(source)),
		"Map destination does not overlap source.",
	)
	written := 0
	for source_index := 0; source_index < len(source); {
		character, size := Decode_Character(source[source_index:])
		source_index += int(size)
		mapped_character := mapping(rune(character))
		if mapped_character < 0 {
			continue
		}
		mapped := Character(mapped_character)
		encoded_size := Character_Size(mapped)
		if encoded_size == CHARACTER_SIZE_INVALID {
			mapped = Character(REPLACEMENT_CHARACTER)
			encoded_size = CHARACTER_SIZE_THREE
		}
		aver.Always(
			int(encoded_size) <= len(destination)-written,
			"Map destination holds complete result.",
		)
		written += int(Encode_Character(destination[written:], mapped))
	}
	return Byte_Count(written)
}

// To_Upper_Into writes Unicode uppercase mapping into caller storage.
func To_Upper_Into(destination Bytes, source Bytes) (count Byte_Count) {
	defer func() { Byte_Count_Invariants(count, "to_upper_into.count") }()
	Bytes_Invariants(destination, "to_upper_into.destination")
	Bytes_Invariants(source, "to_upper_into.source")
	return map_case_into(destination, source, ucd.UPPER_CASE, ucd.Special_Case{}, false)
}

// To_Lower_Into writes Unicode lowercase mapping into caller storage.
func To_Lower_Into(destination Bytes, source Bytes) (count Byte_Count) {
	defer func() { Byte_Count_Invariants(count, "to_lower_into.count") }()
	Bytes_Invariants(destination, "to_lower_into.destination")
	Bytes_Invariants(source, "to_lower_into.source")
	return map_case_into(destination, source, ucd.LOWER_CASE, ucd.Special_Case{}, false)
}

// To_Title_Into writes Unicode title mapping into caller storage.
func To_Title_Into(destination Bytes, source Bytes) (count Byte_Count) {
	defer func() { Byte_Count_Invariants(count, "to_title_into.count") }()
	Bytes_Invariants(destination, "to_title_into.destination")
	Bytes_Invariants(source, "to_title_into.source")
	return map_case_into(destination, source, ucd.TITLE_CASE, ucd.Special_Case{}, false)
}

// To_Upper_Special_Into applies language-specific uppercase mapping.
func To_Upper_Special_Into(
	destination Bytes, special ucd.Special_Case, source Bytes,
) (count Byte_Count) {
	defer func() { Byte_Count_Invariants(count, "to_upper_special_into.count") }()
	Bytes_Invariants(destination, "to_upper_special_into.destination")
	ucd.Special_Case_Invariants(special, "to_upper_special_into.special")
	Bytes_Invariants(source, "to_upper_special_into.source")
	return map_case_into(destination, source, ucd.UPPER_CASE, special, true)
}

// To_Lower_Special_Into applies language-specific lowercase mapping.
func To_Lower_Special_Into(
	destination Bytes, special ucd.Special_Case, source Bytes,
) (count Byte_Count) {
	defer func() { Byte_Count_Invariants(count, "to_lower_special_into.count") }()
	Bytes_Invariants(destination, "to_lower_special_into.destination")
	ucd.Special_Case_Invariants(special, "to_lower_special_into.special")
	Bytes_Invariants(source, "to_lower_special_into.source")
	return map_case_into(destination, source, ucd.LOWER_CASE, special, true)
}

// To_Title_Special_Into applies language-specific title mapping.
func To_Title_Special_Into(
	destination Bytes, special ucd.Special_Case, source Bytes,
) (count Byte_Count) {
	defer func() { Byte_Count_Invariants(count, "to_title_special_into.count") }()
	Bytes_Invariants(destination, "to_title_special_into.destination")
	ucd.Special_Case_Invariants(special, "to_title_special_into.special")
	Bytes_Invariants(source, "to_title_special_into.source")
	return map_case_into(destination, source, ucd.TITLE_CASE, special, true)
}

// To_Valid_UTF8_Into replaces each invalid-byte run in caller storage.
func To_Valid_UTF8_Into(
	destination Bytes, source Bytes, replacement Bytes,
) (count Byte_Count) {
	defer func() { Byte_Count_Invariants(count, "to_valid_utf8_into.count") }()
	Bytes_Invariants(destination, "to_valid_utf8_into.destination")
	Bytes_Invariants(source, "to_valid_utf8_into.source")
	Bytes_Invariants(replacement, "to_valid_utf8_into.replacement")
	aver.Always(
		!bytes.Overlap(bytes.Slice(destination), bytes.Slice(source)),
		"UTF-8 destination does not overlap source.",
	)
	written := 0
	invalid := false
	for source_index := 0; source_index < len(source); {
		character, size := Decode_Character(source[source_index:])
		if character == REPLACEMENT_CHARACTER {
			if size == 1 {
				source_index++
				if invalid {
					continue
				}
				invalid = true
				aver.Always(
					len(replacement) <= len(destination)-written,
					"UTF-8 repair destination holds replacement.",
				)
				written += copy(destination[written:], replacement)
				continue
			}
		}
		invalid = false
		aver.Always(
			int(size) <= len(destination)-written,
			"UTF-8 repair destination holds source character.",
		)
		boundary := source_index + int(size)
		written += copy(destination[written:], source[source_index:boundary])
		source_index = boundary
	}
	return Byte_Count(written)
}

// Title_Into writes legacy Unicode word-start title mapping.
func Title_Into(destination Bytes, source Bytes) (count Byte_Count) {
	defer func() { Byte_Count_Invariants(count, "title_into.count") }()
	Bytes_Invariants(destination, "title_into.destination")
	Bytes_Invariants(source, "title_into.source")
	aver.Always(
		!bytes.Overlap(bytes.Slice(destination), bytes.Slice(source)),
		"Title destination does not overlap source.",
	)
	written := 0
	previous := Decoded_Character(' ')
	for source_index := 0; source_index < len(source); {
		character, size := Decode_Character(source[source_index:])
		source_index += int(size)
		mapped := ucd.Character(character)
		if title_separator(previous) {
			mapped = ucd.To_Title(mapped)
		}
		previous = character
		encoded_size := Character_Size(Character(mapped))
		aver.Always(
			int(encoded_size) <= len(destination)-written,
			"Title destination holds complete result.",
		)
		written += int(Encode_Character(destination[written:], Character(mapped)))
	}
	return Byte_Count(written)
}

// Trim_Left_Function removes leading characters satisfying predicate.
func Trim_Left_Function(
	source Bytes, predicate Predicate,
) (trimmed Bytes) {
	defer func() { Bytes_Invariants(trimmed, "trim_left_function.trimmed") }()
	Bytes_Invariants(source, "trim_left_function.source")
	end := trim_left_boundary(source, predicate)
	if int(end) == len(source) {
		return nil
	}
	return source[end:]
}

// Trim_Right_Function removes trailing characters satisfying predicate.
func Trim_Right_Function(
	source Bytes, predicate Predicate,
) (trimmed Bytes) {
	defer func() { Bytes_Invariants(trimmed, "trim_right_function.trimmed") }()
	Bytes_Invariants(source, "trim_right_function.source")
	return source[:trim_right_boundary(source, predicate)]
}

// Trim_Function removes leading and trailing predicate matches.
func Trim_Function(
	source Bytes, predicate Predicate,
) (trimmed Bytes) {
	defer func() { Bytes_Invariants(trimmed, "trim_function.trimmed") }()
	Bytes_Invariants(source, "trim_function.source")
	left := trim_left_boundary(source, predicate)
	if int(left) == len(source) {
		return nil
	}
	right := trim_right_boundary(source[left:], predicate)
	return source[left : left+right]
}

// Trim removes leading and trailing characters from cut set.
func Trim(source Bytes, cutset Text) (trimmed Bytes) {
	defer func() { Bytes_Invariants(trimmed, "trim.trimmed") }()
	Bytes_Invariants(source, "trim.source")
	Text_Invariants(cutset, "trim.cutset")
	return Trim_Function(source, func(character rune) (matches bool) {
		return bool(text_contains_character(cutset, Decoded_Character(character)))
	})
}

// Trim_Left removes leading characters from cut set.
func Trim_Left(source Bytes, cutset Text) (trimmed Bytes) {
	defer func() { Bytes_Invariants(trimmed, "trim_left.trimmed") }()
	Bytes_Invariants(source, "trim_left.source")
	Text_Invariants(cutset, "trim_left.cutset")
	return Trim_Left_Function(source, func(character rune) (matches bool) {
		return bool(text_contains_character(cutset, Decoded_Character(character)))
	})
}

// Trim_Right removes trailing characters from cut set.
func Trim_Right(source Bytes, cutset Text) (trimmed Bytes) {
	defer func() { Bytes_Invariants(trimmed, "trim_right.trimmed") }()
	Bytes_Invariants(source, "trim_right.source")
	Text_Invariants(cutset, "trim_right.cutset")
	return Trim_Right_Function(source, func(character rune) (matches bool) {
		return bool(text_contains_character(cutset, Decoded_Character(character)))
	})
}

// Trim_Space removes leading and trailing Unicode whitespace.
func Trim_Space(source Bytes) (trimmed Bytes) {
	defer func() { Bytes_Invariants(trimmed, "trim_space.trimmed") }()
	Bytes_Invariants(source, "trim_space.source")
	return Trim_Function(source, func(character rune) (matches bool) {
		return bool(ucd.Is_Space(ucd.Character(character)))
	})
}

// Runes_Into decodes source into caller character storage.
func Runes_Into(destination Characters, source Bytes) (count Count) {
	defer func() { Count_Invariants(count, "runes_into.count") }()
	Characters_Invariants(destination, "runes_into.destination")
	Bytes_Invariants(source, "runes_into.source")
	for source_index := 0; source_index < len(source); {
		aver.Always(
			int(count) < len(destination),
			"Rune destination holds decoded result.",
		)
		character, size := Decode_Character(source[source_index:])
		destination[int(count)] = character
		count++
		source_index += int(size)
	}
	return count
}

// Equal_Fold compares decoded characters through Unicode simple folding.
func Equal_Fold(left Bytes, right Bytes) (equal Boolean) {
	defer func() { Boolean_Invariants(equal, "equal_fold.equal") }()
	Bytes_Invariants(left, "equal_fold.left")
	Bytes_Invariants(right, "equal_fold.right")
	left_index := 0
	right_index := 0
	for left_index < len(left) {
		if right_index == len(right) {
			return false
		}
		left_character, left_size := Decode_Character(left[left_index:])
		right_character, right_size := Decode_Character(right[right_index:])
		left_value := ucd.Character(left_character)
		right_value := ucd.Character(right_character)
		if left_value != right_value {
			folded := ucd.Simple_Fold(left_value)
			for folded != left_value && folded != right_value {
				folded = ucd.Simple_Fold(folded)
			}
			if folded != right_value {
				return false
			}
		}
		left_index += int(left_size)
		right_index += int(right_size)
	}
	return Boolean(right_index == len(right))
}

// Fields_Sequence yields Unicode-space-delimited source views.
func Fields_Sequence(source Bytes, yield Yield_Function) (count Field_Count) {
	defer func() { Field_Count_Invariants(count, "fields_sequence.count") }()
	Bytes_Invariants(source, "fields_sequence.source")
	return fields_sequence(source, func(character rune) (matches bool) {
		return bool(ucd.Is_Space(ucd.Character(character)))
	}, yield)
}

// Fields_Function_Sequence yields predicate-delimited source views.
func Fields_Function_Sequence(
	source Bytes, predicate Predicate, yield Yield_Function,
) (count Field_Count) {
	defer func() { Field_Count_Invariants(count, "fields_function_sequence.count") }()
	Bytes_Invariants(source, "fields_function_sequence.source")
	return fields_sequence(source, predicate, yield)
}

func fields_into(
	destination Field_Slices, source Bytes, predicate Predicate,
) (count Field_Count) {
	defer func() { Field_Count_Invariants(count, "fields_internal.count") }()
	Field_Slices_Invariants(destination, "fields_internal.destination")
	Bytes_Invariants(source, "fields_internal.source")
	start := BYTE_INDEX_ABSENT
	for source_index := 0; source_index < len(source); {
		character, size := Decode_Character(source[source_index:])
		if predicate(rune(character)) {
			if start >= 0 {
				aver.Always(
					int(count) < len(destination),
					"Field destination holds delimited result.",
				)
				destination[int(count)] = source[start:source_index:source_index]
				count++
				start = BYTE_INDEX_ABSENT
			}
		} else if start == BYTE_INDEX_ABSENT {
			start = source_index
		}
		source_index += int(size)
	}
	if start == BYTE_INDEX_ABSENT {
		return count
	}
	aver.Always(
		int(count) < len(destination),
		"Field destination holds final result.",
	)
	destination[int(count)] = source[start:len(source):len(source)]
	return count + 1
}

func map_case_into(
	destination Bytes, source Bytes, mapping ucd.Case,
	special ucd.Special_Case, use_special Boolean,
) (count Byte_Count) {
	defer func() { Byte_Count_Invariants(count, "map_case_internal.count") }()
	Bytes_Invariants(destination, "map_case_internal.destination")
	Bytes_Invariants(source, "map_case_internal.source")
	ucd.Case_Invariants(mapping, "map_case_internal.mapping")
	ucd.Special_Case_Invariants(special, "map_case_internal.special")
	Boolean_Invariants(use_special, "map_case_internal.use_special")
	aver.Always(
		!bytes.Overlap(bytes.Slice(destination), bytes.Slice(source)),
		"Case destination does not overlap source.",
	)
	written := 0
	for source_index := 0; source_index < len(source); {
		character, size := Decode_Character(source[source_index:])
		source_index += int(size)
		mapped := ucd.To(mapping, ucd.Character(character))
		if use_special {
			switch mapping {
			case ucd.UPPER_CASE:
				mapped = ucd.Special_Case_To_Upper(
					special, ucd.Character(character),
				)
			case ucd.LOWER_CASE:
				mapped = ucd.Special_Case_To_Lower(
					special, ucd.Character(character),
				)
			case ucd.TITLE_CASE:
				mapped = ucd.Special_Case_To_Title(
					special, ucd.Character(character),
				)
			}
		}
		encoded_size := Character_Size(Character(mapped))
		aver.Always(
			int(encoded_size) <= len(destination)-written,
			"Case destination holds complete result.",
		)
		written += int(Encode_Character(destination[written:], Character(mapped)))
	}
	return Byte_Count(written)
}

func trim_left_boundary(
	source Bytes, predicate Predicate,
) (end bytes.Boundary) {
	defer func() { bytes.Boundary_Invariants(end, "trim_left_boundary.end") }()
	Bytes_Invariants(source, "trim_left_boundary.source")
	for int(end) < len(source) {
		character, size := Decode_Character(source[int(end):])
		if !predicate(rune(character)) {
			return end
		}
		end += bytes.Boundary(size)
	}
	return end
}

func trim_right_boundary(
	source Bytes, predicate Predicate,
) (end bytes.Boundary) {
	defer func() { bytes.Boundary_Invariants(end, "trim_right_boundary.end") }()
	Bytes_Invariants(source, "trim_right_boundary.source")
	end = bytes.Boundary(len(source))
	for end > 0 {
		character, size := Decode_Final_Character(source[:int(end)])
		if !predicate(rune(character)) {
			return end
		}
		end -= bytes.Boundary(size)
	}
	return end
}

func fields_sequence(
	source Bytes, predicate Predicate, yield Yield_Function,
) (count Field_Count) {
	defer func() { Field_Count_Invariants(count, "fields_sequence_internal.count") }()
	Bytes_Invariants(source, "fields_sequence_internal.source")
	start := BYTE_INDEX_ABSENT
	for index := 0; index < len(source); {
		character, size := Decode_Character(source[index:])
		if predicate(rune(character)) {
			if start >= 0 {
				continued := yield(source[start:index:index])
				count++
				if !continued {
					return count
				}
				start = BYTE_INDEX_ABSENT
			}
		} else if start == BYTE_INDEX_ABSENT {
			start = index
		}
		index += int(size)
	}
	if start >= 0 {
		yield(source[start:len(source):len(source)])
		count++
	}
	return count
}

func text_contains_character(text Text, character Decoded_Character) (contained Boolean) {
	defer func() { Boolean_Invariants(contained, "text_contains_character.contained") }()
	Text_Invariants(text, "text_contains_character.text")
	Decoded_Character_Invariants(character, "text_contains_character.character")
	for _, text_character := range text {
		if Decoded_Character(text_character) == character {
			return true
		}
	}
	return false
}

func title_separator(character Decoded_Character) (separator Boolean) {
	defer func() { Boolean_Invariants(separator, "title_separator.separator") }()
	Decoded_Character_Invariants(character, "title_separator.character")
	if character <= 0x7F {
		if '0' <= character {
			if character <= '9' {
				return false
			}
		}
		if 'a' <= character {
			if character <= 'z' {
				return false
			}
		}
		if 'A' <= character {
			if character <= 'Z' {
				return false
			}
		}
		return Boolean(character != '_')
	}
	value := ucd.Character(character)
	if ucd.Is_Letter(value) {
		return false
	}
	if ucd.Is_Digit(value) {
		return false
	}
	return Boolean(ucd.Is_Space(value))
}
