// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package utf16_test

import (
	"testing"

	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/testify"
	"local/james-orcales/shared/unicode/utf16"
)

// Test_Surrogates verifies the complete surrogate interval and its adjacent values.
func Test_Surrogates(t *testing.T) {
	t.Parallel()
	for _, character := range []utf16.Character{
		utf16.Character(bits.INTEGER_32_MINIMUM), -1, 0, 1, 2, utf16.SURROGATE_FIRST - 1,
		utf16.SURROGATE_FIRST, utf16.LOW_SURROGATE_FIRST,
		utf16.SURROGATE_FINAL, utf16.SURROGATE_FINAL + 1,
		utf16.RUNE_MAX, utf16.Character(bits.INTEGER_32_MAXIMUM),
	} {
		testify.Equal(t, reference_is_surrogate(rune(character)),
			bool(utf16.Is_Surrogate(character)), "Is_Surrogate(%U)", character)
	}
}

// Test_Character_Conversion verifies pair conversion and character sizes.
func Test_Character_Conversion(t *testing.T) {
	t.Parallel()
	for _, character := range []utf16.Character{
		utf16.Character(bits.INTEGER_32_MINIMUM), -1, 0, 1, 2, utf16.SURROGATE_FIRST - 1,
		utf16.SURROGATE_FIRST, utf16.SURROGATE_FINAL,
		utf16.SUPPLEMENTARY_FIRST - 1, utf16.SUPPLEMENTARY_FIRST,
		utf16.RUNE_MAX, utf16.RUNE_MAX + 1,
		utf16.Character(bits.INTEGER_32_MAXIMUM),
	} {
		testify.Equal(t, reference_character_size(rune(character)),
			int(utf16.Character_Size(character)),
			"Character_Size(%U)", character)
		standard_first, standard_second := reference_encode_character(rune(character))
		shared_first, shared_second := utf16.Encode_Character(character)
		testify.Equal(t, standard_first, rune(shared_first),
			"Encode_Character(%U) first", character)
		testify.Equal(t, standard_second, rune(shared_second),
			"Encode_Character(%U) second", character)
		testify.Equal(t, reference_decode_character(standard_first, standard_second),
			rune(utf16.Decode_Character(
				utf16.Character(standard_first),
				utf16.Character(standard_second),
			)), "Decode_Character(%U)", character)
	}
	for _, character := range []utf16.Character{
		utf16.Character(bits.INTEGER_32_MINIMUM),
		0, 1, 2,
		utf16.Character(bits.INTEGER_32_MAXIMUM),
	} {
		testify.Equal(t,
			reference_decode_character(rune(character), 0xdc00),
			rune(utf16.Decode_Character(character, 0xdc00)),
			"Decode_Character(%U, low surrogate)", character,
		)
		testify.Equal(t,
			reference_decode_character(0xd800, rune(character)),
			rune(utf16.Decode_Character(0xd800, character)),
			"Decode_Character(high surrogate, %U)", character,
		)
	}
}

// Test_Sequence_Conversion verifies valid sequences, invalid characters, and unpaired surrogates.
func Test_Sequence_Conversion(t *testing.T) {
	t.Parallel()
	character_cases := [][]rune{
		{},
		{0, 1, 2},
		{'A', '水', 0x10000, 0x12345, 0x10ffff},
		{0xd7ff, 0xd800, 0xdfff, 0xe000, 0x110000, -1},
	}
	for _, characters := range character_cases {
		shared_characters := make(utf16.Characters, len(characters))
		for index, character := range characters {
			shared_characters[index] = utf16.Character(character)
		}
		testify.Equal(t, reference_encode(characters),
			[]uint16(encode(shared_characters)), "Encode(%x)", characters)
	}
	for _, characters := range []utf16.Characters{
		{utf16.Character(bits.INTEGER_32_MINIMUM)},
		{
			utf16.Character(bits.INTEGER_32_MAXIMUM),
			utf16.Character(bits.INTEGER_32_MAXIMUM),
		},
		make(utf16.Characters, utf16.SEQUENCE_SIZE_MAXIMUM),
	} {
		encode(characters)
	}
	word_cases := [][]uint16{
		{},
		{0, 1, 2},
		{0xffff, 0xd800, 0xdc00, 0xd808, 0xdf45, 0xdbff, 0xdfff},
		{0xd800, 'a', 0xdfff},
	}
	for _, words := range word_cases {
		decoded := decode(utf16.Words(words))
		shared_characters := make([]rune, len(decoded))
		for index, character := range decoded {
			shared_characters[index] = rune(character)
		}
		testify.Equal(t, reference_decode(words), shared_characters,
			"Decode(%x)", words)
	}
	maximum_words := make(utf16.Words, utf16.SEQUENCE_SIZE_MAXIMUM)
	testify.Equal(t, utf16.SEQUENCE_SIZE_MAXIMUM,
		len(decode(maximum_words)))
	word_buffer := make(utf16.Words, utf16.SEQUENCE_SIZE_MAXIMUM)
	character_buffer := make(
		utf16.Decoded_Characters, utf16.SEQUENCE_SIZE_MAXIMUM,
	)
	for _, size := range []int{0, 1, 2, utf16.SEQUENCE_SIZE_MAXIMUM} {
		testify.Equal(t, size,
			len(utf16.Encode(word_buffer[:size], nil)),
			"Encode preserves a %d-word prefix", size)
		testify.Equal(t, size,
			len(utf16.Decode(character_buffer[:size], nil)),
			"Decode preserves a %d-character prefix", size)
	}
}

// Test_Append verifies that repeated append operations equal sequence encoding.
func Test_Append(t *testing.T) {
	t.Parallel()
	characters := []utf16.Character{
		0, 1, 2, '水', 0x10000, utf16.RUNE_MAX, 0xd800, -1,
	}
	var shared_word_storage [utf16.SEQUENCE_SIZE_MAXIMUM]uint16
	shared_words := utf16.Words(shared_word_storage[:0])
	var standard_words []uint16
	for _, character := range characters {
		shared_words = utf16.Words(
			utf16.Append_Character(shared_words, character),
		)
		standard_words = reference_append_character(standard_words, rune(character))
	}
	testify.Equal(t, standard_words, []uint16(shared_words))
	var invalid_storage [utf16.CHARACTER_SIZE_MAXIMUM]uint16
	utf16.Append_Character(
		invalid_storage[:0], utf16.Character(bits.INTEGER_32_MINIMUM),
	)
	utf16.Append_Character(
		invalid_storage[:0], utf16.Character(bits.INTEGER_32_MAXIMUM),
	)
	testify.Equal(t, utf16.SEQUENCE_SIZE_MAXIMUM,
		len(utf16.Append_Character(
			make(utf16.Words, utf16.SEQUENCE_SIZE_MAXIMUM-1,
				utf16.SEQUENCE_SIZE_MAXIMUM), 0,
		)))
}

// Test_Allocation proves each public operation keeps heap allocation at zero.
func Test_Allocation(t *testing.T) {
	var word_storage [utf16.SEQUENCE_SIZE_MAXIMUM]uint16
	var character_storage [utf16.SEQUENCE_SIZE_MAXIMUM]utf16.Decoded_Character
	state := allocation_state{
		Source_Characters: utf16.Characters{'A', '\U00010000'},
		Source_Words:      utf16.Words{'A', 0xd800, 0xdc00},
		Word_Storage:      word_storage[:],
		Character_Storage: character_storage[:],
	}
	for _, one := range allocation_cases(&state) {
		t.Run(one.Name, func(t *testing.T) { testify.Zero_Allocation(t, one.Run) })
	}
}

// Test_Domain_Errors verifies input and result sequence limits.
func Test_Domain_Errors(t *testing.T) {
	t.Parallel()
	large_words := make(utf16.Words, utf16.SEQUENCE_SIZE_MAXIMUM+1)
	testify.Panics(t, func() { utf16.Decode(nil, large_words) }, "oversize Words")
	large_characters := make(
		utf16.Characters, utf16.SEQUENCE_SIZE_MAXIMUM+1,
	)
	testify.Panics(t, func() {
		utf16.Encode(nil, large_characters)
	}, "oversize Characters")
	maximum := make(utf16.Words, utf16.SEQUENCE_SIZE_MAXIMUM)
	testify.Panics(t, func() {
		utf16.Append_Character(maximum, 0)
	}, "oversize append result")
	testify.Panics(t, func() {
		utf16.Encode(nil, utf16.Characters{'A'})
	}, "missing encode storage")
	testify.Panics(t, func() {
		utf16.Append_Character(nil, 'A')
	}, "missing append storage")
	testify.Panics(t, func() {
		utf16.Decode(nil, utf16.Words{'A'})
	}, "missing decode storage")
}

type allocation_case struct {
	Name string
	Run  func()
}

type allocation_state struct {
	Words      utf16.Words
	Characters utf16.Decoded_Characters
	// Word_Storage and Character_Storage view fixed arrays owned by Test_Allocation: lint
	// bans fixed array fields, and a make inside Run would count as an allocation.
	Word_Storage      utf16.Words
	Character_Storage utf16.Decoded_Characters
	Boolean           utf16.Boolean
	Combined          utf16.Combined_Character
	First             utf16.First_Encoded_Character
	Second            utf16.Second_Encoded_Character
	Size              utf16.Size
	Source_Characters utf16.Characters
	Source_Words      utf16.Words
}

func encode(source utf16.Characters) (result utf16.Words) {
	buffer := make(utf16.Words, 0, utf16.SEQUENCE_SIZE_MAXIMUM)
	return utf16.Encode(buffer, source)
}

func decode(source utf16.Words) (result utf16.Decoded_Characters) {
	buffer := make(
		utf16.Decoded_Characters, 0, utf16.SEQUENCE_SIZE_MAXIMUM,
	)
	return utf16.Decode(buffer, source)
}

func allocation_cases(state *allocation_state) (cases []allocation_case) {
	return []allocation_case{
		{Name: "Is_Surrogate", Run: func() {
			state.Boolean = utf16.Is_Surrogate(utf16.SURROGATE_FIRST)
		}},
		{Name: "Decode_Character", Run: func() {
			state.Combined = utf16.Decode_Character(0xd800, 0xdc00)
		}},
		{Name: "Encode_Character", Run: func() {
			state.First, state.Second =
				utf16.Encode_Character('\U00010000')
		}},
		{Name: "Character_Size", Run: func() {
			state.Size = utf16.Character_Size('\U00010000')
		}},
		{Name: "Encode", Run: func() {
			state.Words = utf16.Encode(
				state.Word_Storage[:0], state.Source_Characters,
			)
		}},
		{Name: "Append_Character", Run: func() {
			state.Words = utf16.Words(utf16.Append_Character(
				state.Word_Storage[:0], '\U00010000',
			))
		}},
		{Name: "Decode", Run: func() {
			state.Characters = utf16.Decode(
				state.Character_Storage[:0], state.Source_Words,
			)
		}},
	}
}

// Test_Encoding_Constants verifies the shared facts behind surrogate conversion.
func Test_Encoding_Constants(t *testing.T) {
	t.Parallel()
	testify.Equal_Values(t, utf16.SURROGATE_MASK+1, utf16.SURROGATE_BLOCK_SIZE)
	testify.Equal_Values(t, utf16.SURROGATE_FIRST+utf16.SURROGATE_BLOCK_SIZE,
		utf16.LOW_SURROGATE_FIRST)
	testify.Equal_Values(t, utf16.LOW_SURROGATE_FIRST+utf16.SURROGATE_BLOCK_SIZE,
		utf16.SURROGATE_LIMIT)
	testify.Equal_Values(t, utf16.SURROGATE_LIMIT-1, utf16.SURROGATE_FINAL)
	testify.Equal_Values(t, uint32(bits.WORD_16_MAXIMUM)+1,
		utf16.SUPPLEMENTARY_FIRST)
}

// SURROGATE_FIRST exposes the first surrogate to the upstream tests.
const SURROGATE_FIRST rune = rune(utf16.SURROGATE_FIRST)

// SURROGATE_LIMIT exposes the boundary after the surrogate range.
const SURROGATE_LIMIT rune = rune(utf16.SURROGATE_LIMIT)

// SUPPLEMENTARY_FIRST exposes the first character that needs a pair.
const SUPPLEMENTARY_FIRST rune = rune(utf16.SUPPLEMENTARY_FIRST)

// RUNE_MAX exposes the final Unicode code point.
const RUNE_MAX rune = rune(utf16.RUNE_MAX)

// REPLACEMENT_CHARACTER exposes the invalid-input result.
const REPLACEMENT_CHARACTER rune = rune(utf16.REPLACEMENT_CHARACTER)

// Test_Standard_Library_Surrogate_Pairs verifies representative pairs.
func Test_Standard_Library_Surrogate_Pairs(t *testing.T) {
	t.Parallel()
	for _, pair := range []struct {
		First  rune
		Second rune
	}{
		{First: 0xd800, Second: 0xdc00},
		{First: 0xd800, Second: 0xdc01},
		{First: 0xd808, Second: 0xdf45},
		{First: 0xdbff, Second: 0xdfff},
		{First: 0xd800, Second: 'a'},
		{First: 'a', Second: 0xdc00},
		{First: -1, Second: -1},
	} {
		testify.Equal(t, reference_decode_character(pair.First, pair.Second),
			rune(utf16.Decode_Character(
				utf16.Character(pair.First),
				utf16.Character(pair.Second),
			)), "Decode_Character(%x, %x)", pair.First, pair.Second)
	}
}

// Test_Standard_Library_Sequences verifies standard sequence examples.
func Test_Standard_Library_Sequences(t *testing.T) {
	t.Parallel()
	for _, characters := range [][]rune{
		{1, 2, 3, 4},
		{0xffff, 0x10000, 0x10001, 0x12345, 0x10ffff},
		{'a', 'b', 0xd7ff, 0xd800, 0xdfff, 0xe000, 0x110000, -1},
	} {
		shared_characters := make(utf16.Characters, len(characters))
		for index, character := range characters {
			shared_characters[index] = utf16.Character(character)
		}
		testify.Equal(t, reference_encode(characters),
			[]uint16(encode(shared_characters)), "Encode(%x)", characters)
	}
	for _, words := range [][]uint16{
		{1, 2, 3, 4},
		{0xffff, 0xd800, 0xdc00, 0xd800, 0xdc01, 0xd808, 0xdf45, 0xdbff, 0xdfff},
		{0xd800, 'a'},
		{0xdfff},
	} {
		decoded := decode(utf16.Words(words))
		shared_characters := make([]rune, len(decoded))
		for index, character := range decoded {
			shared_characters[index] = rune(character)
		}
		testify.Equal(t, reference_decode(words), shared_characters,
			"Decode(%x)", words)
	}
}

func reference_is_surrogate(character rune) (yes bool) {
	return 0xd800 <= character && character < 0xe000
}

func reference_character_size(character rune) (size int) {
	if character < 0 {
		return -1
	}
	if reference_is_surrogate(character) {
		return -1
	}
	if character > 0x10ffff {
		return -1
	}
	if character < 0x10000 {
		return 1
	}
	return 2
}

func reference_encode_character(character rune) (first rune, second rune) {
	if character < 0x10000 {
		return '\ufffd', '\ufffd'
	}
	if character > 0x10ffff {
		return '\ufffd', '\ufffd'
	}
	character -= 0x10000
	return 0xd800 + character>>10, 0xdc00 + character&0x3ff
}

func reference_decode_character(first rune, second rune) (character rune) {
	if 0xd800 <= first {
		if first < 0xdc00 {
			if 0xdc00 <= second {
				if second < 0xe000 {
					return (first-0xd800)<<10 | (second - 0xdc00) + 0x10000
				}
			}
		}
	}
	return '\ufffd'
}

func reference_append_character(words []uint16, character rune) (result []uint16) {
	if reference_is_surrogate(character) {
		character = '\ufffd'
	}
	if character < 0 {
		character = '\ufffd'
	}
	if character > 0x10ffff {
		character = '\ufffd'
	}
	if character < 0x10000 {
		return append(words, uint16(character))
	}
	first, second := reference_encode_character(character)
	return append(words, uint16(first), uint16(second))
}

func reference_encode(characters []rune) (words []uint16) {
	words = make([]uint16, 0, len(characters)*2)
	for _, character := range characters {
		words = reference_append_character(words, character)
	}
	return words
}

func reference_decode(words []uint16) (characters []rune) {
	characters = make([]rune, 0, len(words))
	for word_index := 0; word_index < len(words); word_index++ {
		first := rune(words[word_index])
		decoded_character, paired := reference_decode_pair(words, word_index)
		if paired {
			characters = append(characters, decoded_character)
			word_index++
			continue
		}
		if reference_is_surrogate(first) {
			first = '\ufffd'
		}
		characters = append(characters, first)
	}
	return characters
}

func reference_decode_pair(words []uint16, word_index int) (character rune, paired bool) {
	first := rune(words[word_index])
	if first < 0xd800 {
		return 0, false
	}
	if first >= 0xdc00 {
		return 0, false
	}
	if word_index+1 >= len(words) {
		return 0, false
	}
	second := rune(words[word_index+1])
	if second < 0xdc00 {
		return 0, false
	}
	if second >= 0xe000 {
		return 0, false
	}
	return reference_decode_character(first, second), true
}
