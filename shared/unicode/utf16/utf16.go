// Package utf16 supplies native UTF-16 character and sequence conversion.
package utf16

import (
	invariant "local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/math/bits"
)

// REPLACEMENT_CHARACTER substitutes for invalid characters and surrogate sequences.
const REPLACEMENT_CHARACTER Character = '\uFFFD'

// RUNE_MAX is the final valid Unicode code point.
const RUNE_MAX Character = '\U0010FFFF'

// SURROGATE_FIRST is the first high surrogate.
const SURROGATE_FIRST Character = 0xD800

// SURROGATE_SHIFT is the bit count in one surrogate payload.
const SURROGATE_SHIFT = 10

// SURROGATE_BLOCK_SIZE is the code-point count in each surrogate interval.
const SURROGATE_BLOCK_SIZE Character = 1 << SURROGATE_SHIFT

// SURROGATE_MASK selects one surrogate payload.
const SURROGATE_MASK Character = SURROGATE_BLOCK_SIZE - 1

// LOW_SURROGATE_FIRST is the first low surrogate.
const LOW_SURROGATE_FIRST Character = SURROGATE_FIRST + SURROGATE_BLOCK_SIZE

// SURROGATE_FINAL is the final low surrogate.
const SURROGATE_FINAL Character = SURROGATE_LIMIT - 1

// SURROGATE_LIMIT is the boundary after the surrogate interval.
const SURROGATE_LIMIT Character = LOW_SURROGATE_FIRST + SURROGATE_BLOCK_SIZE

// SUPPLEMENTARY_FIRST is the first character that needs a surrogate pair.
const SUPPLEMENTARY_FIRST Character = Character(bits.WORD_16_MAXIMUM) + 1

// SEQUENCE_SIZE_MINIMUM is the count for an empty sequence.
const SEQUENCE_SIZE_MINIMUM = 0

// SEQUENCE_SIZE_MAXIMUM bounds each character or word sequence.
const SEQUENCE_SIZE_MAXIMUM = 4096

// CHARACTER_SIZE_INVALID reports that a character has no valid UTF-16 encoding.
const CHARACTER_SIZE_INVALID = -1

// CHARACTER_SIZE_MINIMUM is the one-word UTF-16 encoding size.
const CHARACTER_SIZE_MINIMUM = 1

// CHARACTER_SIZE_MAXIMUM is the two-word UTF-16 encoding size.
const CHARACTER_SIZE_MAXIMUM = 2

// CHARACTER_MINIMUM is the smallest value in rune storage.
const CHARACTER_MINIMUM int32 = bits.INTEGER_32_MINIMUM

// CHARACTER_MAXIMUM is the largest value in rune storage.
const CHARACTER_MAXIMUM int32 = bits.INTEGER_32_MAXIMUM

// DECODED_CHARACTER_MINIMUM is the first Unicode code point.
const DECODED_CHARACTER_MINIMUM int32 = 0

// DECODED_CHARACTER_MAXIMUM is the final Unicode code point.
const DECODED_CHARACTER_MAXIMUM int32 = int32(RUNE_MAX)

// FIRST_ENCODED_CHARACTER_MINIMUM is the first high surrogate.
const FIRST_ENCODED_CHARACTER_MINIMUM int32 = int32(SURROGATE_FIRST)

// FIRST_ENCODED_CHARACTER_MAXIMUM includes the invalid-input replacement.
const FIRST_ENCODED_CHARACTER_MAXIMUM int32 = int32(REPLACEMENT_CHARACTER)

// SECOND_ENCODED_CHARACTER_MINIMUM is the first low surrogate.
const SECOND_ENCODED_CHARACTER_MINIMUM int32 = int32(LOW_SURROGATE_FIRST)

// SECOND_ENCODED_CHARACTER_MAXIMUM includes the invalid-input replacement.
const SECOND_ENCODED_CHARACTER_MAXIMUM int32 = int32(REPLACEMENT_CHARACTER)

// COMBINED_CHARACTER_MINIMUM is the replacement below supplementary characters.
const COMBINED_CHARACTER_MINIMUM int32 = int32(REPLACEMENT_CHARACTER)

// COMBINED_CHARACTER_MAXIMUM is the final supplementary character.
const COMBINED_CHARACTER_MAXIMUM int32 = int32(RUNE_MAX)

// COMBINED_CHARACTER_HOLE_FIRST cannot result from a surrogate pair.
const COMBINED_CHARACTER_HOLE_FIRST int32 = COMBINED_CHARACTER_HOLE_FINAL - 1

// COMBINED_CHARACTER_HOLE_FINAL cannot result from a surrogate pair.
const COMBINED_CHARACTER_HOLE_FINAL int32 = int32(bits.WORD_16_MAXIMUM)

// Character is one rune-storage value that a conversion reads.
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

// Combined_Character is a decoded surrogate pair or its replacement.
type Combined_Character rune

// Combined_Character_Invariants excludes the two noncharacters below supplementary storage.
func Combined_Character_Invariants(
	value Combined_Character, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Holed_Int32(
			int32(value), COMBINED_CHARACTER_MINIMUM, COMBINED_CHARACTER_MAXIMUM,
			COMBINED_CHARACTER_HOLE_FIRST, COMBINED_CHARACTER_HOLE_FINAL,
			COMBINED_CHARACTER_HOLE_FINAL, COMBINED_CHARACTER_HOLE_FINAL,
		).
		Ensure()
}

// First_Encoded_Character is a high surrogate or REPLACEMENT_CHARACTER.
type First_Encoded_Character rune

// First_Encoded_Character_Invariants permits the valid first results.
func First_Encoded_Character_Invariants(
	value First_Encoded_Character, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int32(
			int32(value), FIRST_ENCODED_CHARACTER_MINIMUM,
			FIRST_ENCODED_CHARACTER_MAXIMUM,
		).
		Ensure()
	invariant.Always(
		(Character(value) < LOW_SURROGATE_FIRST) ==
			(Character(value) != REPLACEMENT_CHARACTER),
		"An encoded first character is a high surrogate or the replacement character.",
	)
}

// Second_Encoded_Character is a low surrogate or REPLACEMENT_CHARACTER.
type Second_Encoded_Character rune

// Second_Encoded_Character_Invariants permits the valid second results.
func Second_Encoded_Character_Invariants(
	value Second_Encoded_Character, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int32(
			int32(value), SECOND_ENCODED_CHARACTER_MINIMUM,
			SECOND_ENCODED_CHARACTER_MAXIMUM,
		).
		Ensure()
	invariant.Always(
		(Character(value) < SURROGATE_LIMIT) ==
			(Character(value) != REPLACEMENT_CHARACTER),
		"An encoded second character is a low surrogate or the replacement character.",
	)
}

// Size is a valid encoded size or CHARACTER_SIZE_INVALID.
type Size int

// Size_Invariants covers invalidity and both UTF-16 encoding sizes.
func Size_Invariants(value Size, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_3_Int(
			int(value), CHARACTER_SIZE_INVALID,
			CHARACTER_SIZE_MINIMUM, CHARACTER_SIZE_MAXIMUM,
		).
		Ensure()
}

// Words is one bounded UTF-16 sequence.
type Words []uint16

// Words_Invariants applies the package sequence limit.
func Words_Invariants(value Words, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), SEQUENCE_SIZE_MINIMUM, SEQUENCE_SIZE_MAXIMUM).
		Ensure()
}

// Nonempty_Words is one append result, which always contains a new encoding.
type Nonempty_Words []uint16

// Nonempty_Words_Invariants excludes an empty append result.
func Nonempty_Words_Invariants(value Nonempty_Words, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), CHARACTER_SIZE_MINIMUM, SEQUENCE_SIZE_MAXIMUM).
		Ensure()
}

// Characters is one bounded sequence that an encoder reads.
type Characters []Character

// Characters_Invariants applies the package sequence limit.
func Characters_Invariants(value Characters, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), SEQUENCE_SIZE_MINIMUM, SEQUENCE_SIZE_MAXIMUM).
		Ensure()
}

// Decoded_Characters is one bounded sequence that Decode returns.
type Decoded_Characters []Decoded_Character

// Decoded_Characters_Invariants applies the package sequence limit.
func Decoded_Characters_Invariants(
	value Decoded_Characters, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), SEQUENCE_SIZE_MINIMUM, SEQUENCE_SIZE_MAXIMUM).
		Ensure()
}

// Boolean gives each UTF-16 report its own coverage identity.
type Boolean bool

// Boolean_Invariants requires both report values.
func Boolean_Invariants(value Boolean, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "A UTF-16 report is true.").
		Ensure()
}

// Is_Surrogate reports whether a character can occur in a surrogate pair.
func Is_Surrogate(character Character) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "is_surrogate.yes") }()
	Character_Invariants(character, "is_surrogate.character")
	return Boolean(SURROGATE_FIRST <= character && character < SURROGATE_LIMIT)
}

// Decode_Character combines one high surrogate and one low surrogate.
func Decode_Character(
	first Character, second Character,
) (character Combined_Character) {
	defer func() {
		Combined_Character_Invariants(character, "decode_character.character")
	}()
	Character_Invariants(first, "decode_character.first")
	Character_Invariants(second, "decode_character.second")
	if SURROGATE_FIRST <= first {
		if first < LOW_SURROGATE_FIRST {
			if LOW_SURROGATE_FIRST <= second {
				if second < SURROGATE_LIMIT {
					high := (first - SURROGATE_FIRST) << SURROGATE_SHIFT
					low := second - LOW_SURROGATE_FIRST
					return Combined_Character(
						high | low + SUPPLEMENTARY_FIRST,
					)
				}
			}
		}
	}
	return Combined_Character(REPLACEMENT_CHARACTER)
}

// Encode_Character divides one supplementary character into a surrogate pair.
func Encode_Character(
	character Character,
) (first First_Encoded_Character, second Second_Encoded_Character) {
	defer func() {
		First_Encoded_Character_Invariants(first, "encode_character.first")
		Second_Encoded_Character_Invariants(second, "encode_character.second")
	}()
	Character_Invariants(character, "encode_character.character")
	if character < SUPPLEMENTARY_FIRST {
		return First_Encoded_Character(REPLACEMENT_CHARACTER),
			Second_Encoded_Character(REPLACEMENT_CHARACTER)
	}
	if character > RUNE_MAX {
		return First_Encoded_Character(REPLACEMENT_CHARACTER),
			Second_Encoded_Character(REPLACEMENT_CHARACTER)
	}
	character -= SUPPLEMENTARY_FIRST
	return First_Encoded_Character(
		SURROGATE_FIRST + (character>>SURROGATE_SHIFT)&SURROGATE_MASK,
	), Second_Encoded_Character(LOW_SURROGATE_FIRST + character&SURROGATE_MASK)
}

// Character_Size returns the word count or CHARACTER_SIZE_INVALID.
func Character_Size(character Character) (size Size) {
	defer func() { Size_Invariants(size, "character_size.size") }()
	Character_Invariants(character, "character_size.character")
	switch {
	case 0 <= character && character < SURROGATE_FIRST:
		return CHARACTER_SIZE_MINIMUM
	case SURROGATE_LIMIT <= character && character < SUPPLEMENTARY_FIRST:
		return CHARACTER_SIZE_MINIMUM
	case SUPPLEMENTARY_FIRST <= character && character <= RUNE_MAX:
		return CHARACTER_SIZE_MAXIMUM
	default:
		return CHARACTER_SIZE_INVALID
	}
}

// Encode converts a character sequence to UTF-16 words.
func Encode(source Characters) (result Words) {
	defer func() { Words_Invariants(result, "encode.result") }()
	Characters_Invariants(source, "encode.source")
	count := len(source)
	for _, character := range source {
		if character >= SUPPLEMENTARY_FIRST {
			if character <= RUNE_MAX {
				count++
			}
		}
	}
	invariant.Always(
		count <= SEQUENCE_SIZE_MAXIMUM,
		"A UTF-16 encoding result does not exceed the sequence limit.",
	)
	result = make(Words, count)
	position := 0
	for _, character := range source {
		switch {
		case 0 <= character && character < SURROGATE_FIRST,
			SURROGATE_LIMIT <= character && character < SUPPLEMENTARY_FIRST:
			result[position] = uint16(character)
			position++
		case SUPPLEMENTARY_FIRST <= character && character <= RUNE_MAX:
			character -= SUPPLEMENTARY_FIRST
			result[position] = uint16(
				SURROGATE_FIRST + (character>>SURROGATE_SHIFT)&SURROGATE_MASK,
			)
			result[position+CHARACTER_SIZE_MINIMUM] = uint16(
				LOW_SURROGATE_FIRST + character&SURROGATE_MASK,
			)
			position += CHARACTER_SIZE_MAXIMUM
		default:
			result[position] = uint16(REPLACEMENT_CHARACTER)
			position++
		}
	}
	return result[:position]
}

// Append_Character adds one UTF-16 encoding to a word sequence.
func Append_Character(buffer Words, character Character) (result Nonempty_Words) {
	defer func() {
		Nonempty_Words_Invariants(result, "append_character.result")
	}()
	Words_Invariants(buffer, "append_character.buffer")
	Character_Invariants(character, "append_character.character")
	switch {
	case 0 <= character && character < SURROGATE_FIRST,
		SURROGATE_LIMIT <= character && character < SUPPLEMENTARY_FIRST:
		return Nonempty_Words(append(buffer, uint16(character)))
	case SUPPLEMENTARY_FIRST <= character && character <= RUNE_MAX:
		character -= SUPPLEMENTARY_FIRST
		first := uint16(
			SURROGATE_FIRST + (character>>SURROGATE_SHIFT)&SURROGATE_MASK,
		)
		second := uint16(LOW_SURROGATE_FIRST + character&SURROGATE_MASK)
		return Nonempty_Words(append(buffer, first, second))
	}
	return Nonempty_Words(append(buffer, uint16(REPLACEMENT_CHARACTER)))
}

// DECODE_BUFFER_SIZE keeps common decode results on the caller stack.
const DECODE_BUFFER_SIZE = 64

// Decode converts UTF-16 words to Unicode characters.
func Decode[Source interface{ Words }](source Source) (_ Decoded_Characters) {
	var buffer [DECODE_BUFFER_SIZE]Decoded_Character
	return decode(source, &buffer)
}

func decode[Source interface{ Words }](
	source Source, buffer *[DECODE_BUFFER_SIZE]Decoded_Character,
) (result Decoded_Characters) {
	defer func() { Decoded_Characters_Invariants(result, "decode.result") }()
	Words_Invariants(Words(source), "decode.source")
	result = buffer[:0]
	for index := 0; index < len(source); index++ {
		first := source[index]
		var character Decoded_Character
		switch {
		case Character(first) < SURROGATE_FIRST,
			SURROGATE_LIMIT <= Character(first):
			character = Decoded_Character(first)
		case Character(first) < LOW_SURROGATE_FIRST &&
			index+CHARACTER_SIZE_MINIMUM < len(source) &&
			LOW_SURROGATE_FIRST <= Character(source[index+CHARACTER_SIZE_MINIMUM]) &&
			Character(source[index+CHARACTER_SIZE_MINIMUM]) < SURROGATE_LIMIT:
			second := source[index+CHARACTER_SIZE_MINIMUM]
			high := (Character(first) - SURROGATE_FIRST) << SURROGATE_SHIFT
			low := Character(second) - LOW_SURROGATE_FIRST
			character = Decoded_Character(
				high | low + SUPPLEMENTARY_FIRST,
			)
			index++
		default:
			character = Decoded_Character(REPLACEMENT_CHARACTER)
		}
		result = append(result, character)
	}
	return result
}
