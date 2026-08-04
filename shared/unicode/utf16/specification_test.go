// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package utf16_test

import (
	"testing"
	standard_utf16 "unicode/utf16"

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
		testify.Equal(t, standard_utf16.IsSurrogate(rune(character)),
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
		testify.Equal(t, standard_utf16.RuneLen(rune(character)),
			int(utf16.Character_Size(character)),
			"Character_Size(%U)", character)
		standard_first, standard_second := standard_utf16.EncodeRune(rune(character))
		shared_first, shared_second := utf16.Encode_Character(character)
		testify.Equal(t, standard_first, rune(shared_first),
			"Encode_Character(%U) first", character)
		testify.Equal(t, standard_second, rune(shared_second),
			"Encode_Character(%U) second", character)
		testify.Equal(t, standard_utf16.DecodeRune(standard_first, standard_second),
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
			standard_utf16.DecodeRune(rune(character), 0xdc00),
			rune(utf16.Decode_Character(character, 0xdc00)),
			"Decode_Character(%U, low surrogate)", character,
		)
		testify.Equal(t,
			standard_utf16.DecodeRune(0xd800, rune(character)),
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
		testify.Equal(t, standard_utf16.Encode(characters),
			[]uint16(utf16.Encode(shared_characters)), "Encode(%x)", characters)
	}
	for _, characters := range []utf16.Characters{
		{utf16.Character(bits.INTEGER_32_MINIMUM)},
		{
			utf16.Character(bits.INTEGER_32_MAXIMUM),
			utf16.Character(bits.INTEGER_32_MAXIMUM),
		},
		make(utf16.Characters, utf16.SEQUENCE_SIZE_MAXIMUM),
	} {
		utf16.Encode(characters)
	}
	word_cases := [][]uint16{
		{},
		{0, 1, 2},
		{0xffff, 0xd800, 0xdc00, 0xd808, 0xdf45, 0xdbff, 0xdfff},
		{0xd800, 'a', 0xdfff},
	}
	for _, words := range word_cases {
		decoded := utf16.Decode(utf16.Words(words))
		shared_characters := make([]rune, len(decoded))
		for index, character := range decoded {
			shared_characters[index] = rune(character)
		}
		testify.Equal(t, standard_utf16.Decode(words), shared_characters,
			"Decode(%x)", words)
	}
	maximum_words := make(utf16.Words, utf16.SEQUENCE_SIZE_MAXIMUM)
	testify.Equal(t, utf16.SEQUENCE_SIZE_MAXIMUM,
		len(utf16.Decode(maximum_words)))
}

// Test_Append verifies that repeated append operations equal sequence encoding.
func Test_Append(t *testing.T) {
	t.Parallel()
	characters := []utf16.Character{
		0, 1, 2, '水', 0x10000, utf16.RUNE_MAX, 0xd800, -1,
	}
	var shared_words utf16.Words
	var standard_words []uint16
	for _, character := range characters {
		shared_words = utf16.Words(
			utf16.Append_Character(shared_words, character),
		)
		standard_words = standard_utf16.AppendRune(standard_words, rune(character))
	}
	testify.Equal(t, standard_words, []uint16(shared_words))
	utf16.Append_Character(nil, utf16.Character(bits.INTEGER_32_MINIMUM))
	utf16.Append_Character(nil, utf16.Character(bits.INTEGER_32_MAXIMUM))
	testify.Equal(t, utf16.SEQUENCE_SIZE_MAXIMUM,
		len(utf16.Append_Character(
			make(utf16.Words, utf16.SEQUENCE_SIZE_MAXIMUM-1), 0,
		)))
}

// Test_Domain_Errors verifies input and result sequence limits.
func Test_Domain_Errors(t *testing.T) {
	t.Parallel()
	large_words := make(utf16.Words, utf16.SEQUENCE_SIZE_MAXIMUM+1)
	testify.Panics(t, func() { utf16.Decode(large_words) }, "oversize Words")
	large_characters := make(
		utf16.Characters, utf16.SEQUENCE_SIZE_MAXIMUM+1,
	)
	testify.Panics(t, func() {
		utf16.Encode(large_characters)
	}, "oversize Characters")
	maximum := make(utf16.Words, utf16.SEQUENCE_SIZE_MAXIMUM)
	testify.Panics(t, func() {
		utf16.Append_Character(maximum, 0)
	}, "oversize append result")
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
		testify.Equal(t, standard_utf16.DecodeRune(pair.First, pair.Second),
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
		testify.Equal(t, standard_utf16.Encode(characters),
			[]uint16(utf16.Encode(shared_characters)), "Encode(%x)", characters)
	}
	for _, words := range [][]uint16{
		{1, 2, 3, 4},
		{0xffff, 0xd800, 0xdc00, 0xd800, 0xdc01, 0xd808, 0xdf45, 0xdbff, 0xdfff},
		{0xd800, 'a'},
		{0xdfff},
	} {
		decoded := utf16.Decode(utf16.Words(words))
		shared_characters := make([]rune, len(decoded))
		for index, character := range decoded {
			shared_characters[index] = rune(character)
		}
		testify.Equal(t, standard_utf16.Decode(words), shared_characters,
			"Decode(%x)", words)
	}
}
