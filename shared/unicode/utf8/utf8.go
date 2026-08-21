// Package utf8 supplies native UTF-8 encoding, decoding, counting, and validation.
package utf8

import (
	"local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/math/bits"
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

func machine_word[Data ~string | ~[]byte](source Data) (value uintptr) {
	if POINTER_SIZE == bits.BIT_COUNT_32_MAXIMUM/bits.BIT_COUNT_8_MAXIMUM {
		return uintptr(source[0]) |
			uintptr(source[1])<<bits.BIT_COUNT_8_MAXIMUM |
			uintptr(source[2])<<(2*bits.BIT_COUNT_8_MAXIMUM) |
			uintptr(source[3])<<(3*bits.BIT_COUNT_8_MAXIMUM)
	}
	return uintptr(uint64(source[0]) |
		uint64(source[1])<<bits.BIT_COUNT_8_MAXIMUM |
		uint64(source[2])<<(2*bits.BIT_COUNT_8_MAXIMUM) |
		uint64(source[3])<<(3*bits.BIT_COUNT_8_MAXIMUM) |
		uint64(source[4])<<(4*bits.BIT_COUNT_8_MAXIMUM) |
		uint64(source[5])<<(5*bits.BIT_COUNT_8_MAXIMUM) |
		uint64(source[6])<<(6*bits.BIT_COUNT_8_MAXIMUM) |
		uint64(source[7])<<(7*bits.BIT_COUNT_8_MAXIMUM))
}

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
func Bytes_Invariants(value Bytes, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), SEQUENCE_SIZE_MINIMUM, SEQUENCE_SIZE_MAXIMUM).
		Ensure()
}

// Text is one bounded text sequence.
type Text string

// Text_Invariants applies the package sequence limit.
func Text_Invariants(value Text, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), SEQUENCE_SIZE_MINIMUM, SEQUENCE_SIZE_MAXIMUM).
		Ensure()
}

// Byte is one possible UTF-8 byte.
type Byte byte

// Byte_Invariants covers all byte values.
func Byte_Invariants(value Byte, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint8(uint8(value), bits.WORD_8_MINIMUM, bits.WORD_8_MAXIMUM).
		Ensure()
}

// Character is one rune-storage value that an encoder reads.
type Character rune

// Character_Invariants covers the complete rune-storage domain.
func Character_Invariants(value Character, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int32(int32(value), CHARACTER_MINIMUM, CHARACTER_MAXIMUM).
		Ensure()
}

// Decoded_Character is one valid Unicode code point that a decoder returns.
type Decoded_Character rune

// Decoded_Character_Invariants covers all valid decoded code points.
func Decoded_Character_Invariants(
	value Decoded_Character, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int32(
			int32(value), DECODED_CHARACTER_MINIMUM, DECODED_CHARACTER_MAXIMUM,
		).
		Ensure()
}

// Size is a valid encoded size or CHARACTER_SIZE_INVALID.
type Size int

// Size_Invariants covers all UTF-8 character sizes and invalidity.
func Size_Invariants(value Size, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
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
func Decoded_Size_Invariants(value Decoded_Size, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), DECODED_SIZE_MINIMUM, DECODED_SIZE_MAXIMUM).
		Ensure()
}

// Encoded_Size is the byte count that one encoding writes.
type Encoded_Size int

// Encoded_Size_Invariants lists all UTF-8 encoding sizes.
func Encoded_Size_Invariants(value Encoded_Size, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_4_Int(
			int(value), CHARACTER_SIZE_MINIMUM, CHARACTER_SIZE_TWO,
			CHARACTER_SIZE_THREE, CHARACTER_SIZE_MAXIMUM,
		).
		Ensure()
}

// Count is the number of decoded characters in one sequence.
type Count int

// Count_Invariants applies the maximum one-character-per-byte count.
func Count_Invariants(value Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), CHARACTER_COUNT_MINIMUM, CHARACTER_COUNT_MAXIMUM).
		Ensure()
}

// Nonempty_Bytes is one append result, which always contains a new encoding.
type Nonempty_Bytes []byte

// Nonempty_Bytes_Invariants excludes an empty append result.
func Nonempty_Bytes_Invariants(value Nonempty_Bytes, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), CHARACTER_SIZE_MINIMUM, SEQUENCE_SIZE_MAXIMUM).
		Ensure()
}

// Boolean gives each UTF-8 report its own coverage identity.
type Boolean bool

// Boolean_Invariants requires both report values.
func Boolean_Invariants(value Boolean, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
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
	invariant.Always(
		result_size <= SEQUENCE_SIZE_MAXIMUM,
		"A UTF-8 append result does not exceed the sequence limit.",
	)
	invariant.Always(
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
				if machine_word(source)&HIGH_BITS != 0 {
					break
				}
				source = source[POINTER_SIZE:]
			}
			continue
		}
		first := FIRST_DATA[first_byte]
		size := int(first & FIRST_SIZE_MASK)
		acceptance := first >> FIRST_ACCEPTANCE_SHIFT
		switch size {
		case CHARACTER_SIZE_TWO:
			if len(source) < CHARACTER_SIZE_TWO {
				return false
			}
			second_valid := ACCEPTANCE_MINIMUM_DATA[acceptance] <= source[1] &&
				source[1] <= ACCEPTANCE_MAXIMUM_DATA[acceptance]
			if !second_valid {
				return false
			}
			source = source[CHARACTER_SIZE_TWO:]
		case CHARACTER_SIZE_THREE:
			if len(source) < CHARACTER_SIZE_THREE {
				return false
			}
			second_valid := ACCEPTANCE_MINIMUM_DATA[acceptance] <= source[1] &&
				source[1] <= ACCEPTANCE_MAXIMUM_DATA[acceptance]
			if !second_valid {
				return false
			}
			third_valid := CONTINUATION_MINIMUM <= source[2] &&
				source[2] <= CONTINUATION_MAXIMUM
			if !third_valid {
				return false
			}
			source = source[CHARACTER_SIZE_THREE:]
		case CHARACTER_SIZE_MAXIMUM:
			if len(source) < CHARACTER_SIZE_MAXIMUM {
				return false
			}
			second_valid := ACCEPTANCE_MINIMUM_DATA[acceptance] <= source[1] &&
				source[1] <= ACCEPTANCE_MAXIMUM_DATA[acceptance]
			if !second_valid {
				return false
			}
			third_valid := CONTINUATION_MINIMUM <= source[2] &&
				source[2] <= CONTINUATION_MAXIMUM
			if !third_valid {
				return false
			}
			fourth_valid := CONTINUATION_MINIMUM <= source[3] &&
				source[3] <= CONTINUATION_MAXIMUM
			if !fourth_valid {
				return false
			}
			source = source[CHARACTER_SIZE_MAXIMUM:]
		default:
			return false
		}
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
				if machine_word(source)&HIGH_BITS != 0 {
					break
				}
				source = source[POINTER_SIZE:]
			}
			continue
		}
		first := FIRST_DATA[first_byte]
		size := int(first & FIRST_SIZE_MASK)
		acceptance := first >> FIRST_ACCEPTANCE_SHIFT
		switch size {
		case CHARACTER_SIZE_TWO:
			if len(source) < CHARACTER_SIZE_TWO {
				return false
			}
			second_valid := ACCEPTANCE_MINIMUM_DATA[acceptance] <= source[1] &&
				source[1] <= ACCEPTANCE_MAXIMUM_DATA[acceptance]
			if !second_valid {
				return false
			}
			source = source[CHARACTER_SIZE_TWO:]
		case CHARACTER_SIZE_THREE:
			if len(source) < CHARACTER_SIZE_THREE {
				return false
			}
			second_valid := ACCEPTANCE_MINIMUM_DATA[acceptance] <= source[1] &&
				source[1] <= ACCEPTANCE_MAXIMUM_DATA[acceptance]
			if !second_valid {
				return false
			}
			third_valid := CONTINUATION_MINIMUM <= source[2] &&
				source[2] <= CONTINUATION_MAXIMUM
			if !third_valid {
				return false
			}
			source = source[CHARACTER_SIZE_THREE:]
		case CHARACTER_SIZE_MAXIMUM:
			if len(source) < CHARACTER_SIZE_MAXIMUM {
				return false
			}
			second_valid := ACCEPTANCE_MINIMUM_DATA[acceptance] <= source[1] &&
				source[1] <= ACCEPTANCE_MAXIMUM_DATA[acceptance]
			if !second_valid {
				return false
			}
			third_valid := CONTINUATION_MINIMUM <= source[2] &&
				source[2] <= CONTINUATION_MAXIMUM
			if !third_valid {
				return false
			}
			fourth_valid := CONTINUATION_MINIMUM <= source[3] &&
				source[3] <= CONTINUATION_MAXIMUM
			if !fourth_valid {
				return false
			}
			source = source[CHARACTER_SIZE_MAXIMUM:]
		default:
			return false
		}
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
