// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package utf8_test

import (
	"testing"
	standard_utf8 "unicode/utf8"

	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/strings"
	"local/james-orcales/shared/testify"
	"local/james-orcales/shared/unicode/utf8"
)

// Test_Complete_Sequences verifies complete, incomplete, and invalid first encodings.
func Test_Complete_Sequences(t *testing.T) {
	t.Parallel()
	for _, value := range []string{
		"", "A", "¢", "€", "𐀀", "\xc2", "\xe2\x82", "\xf0\x90\x80", "\x80", "\xc0",
	} {
		testify.Equal(t, standard_utf8.FullRuneInString(value),
			bool(utf8.Full_Character_Text(utf8.Text(value))),
			"Full_Character_Text(%q)", value)
		testify.Equal(t, standard_utf8.FullRune([]byte(value)),
			bool(utf8.Full_Character(utf8.Bytes(value))),
			"Full_Character(%q)", value)
	}
	maximum := repeat("a", utf8.SEQUENCE_SIZE_MAXIMUM)
	testify.True(t, bool(utf8.Full_Character_Text(utf8.Text(maximum))))
	testify.True(t, bool(utf8.Full_Character(utf8.Bytes(maximum))))
}

// Test_Decode verifies first and final decoding for valid and invalid input.
func Test_Decode(t *testing.T) {
	t.Parallel()
	for _, value := range []string{
		"", "A", "¢", "€", "𐀀", "A¢€𐀀", "\x80", "\xc0", "\xed\xa0\x80", "\xf4\x90\x80\x80",
	} {
		standard_character, standard_size := standard_utf8.DecodeRuneInString(value)
		shared_character, shared_size := utf8.Decode_Character_Text(
			utf8.Text(value),
		)
		testify.Equal(t, standard_character, rune(shared_character),
			"Decode_Character_Text(%q) character", value)
		testify.Equal(t, standard_size, int(shared_size),
			"Decode_Character_Text(%q) size", value)
		standard_character, standard_size = standard_utf8.DecodeRune([]byte(value))
		shared_character, shared_size = utf8.Decode_Character(
			utf8.Bytes(value),
		)
		testify.Equal(t, standard_character, rune(shared_character),
			"Decode_Character(%q) character", value)
		testify.Equal(t, standard_size, int(shared_size),
			"Decode_Character(%q) size", value)

		standard_character, standard_size = standard_utf8.DecodeLastRuneInString(value)
		shared_character, shared_size = utf8.Decode_Final_Character_Text(
			utf8.Text(value),
		)
		testify.Equal(t, standard_character, rune(shared_character),
			"Decode_Final_Character_Text(%q) character", value)
		testify.Equal(t, standard_size, int(shared_size),
			"Decode_Final_Character_Text(%q) size", value)
		standard_character, standard_size = standard_utf8.DecodeLastRune([]byte(value))
		shared_character, shared_size = utf8.Decode_Final_Character(
			utf8.Bytes(value),
		)
		testify.Equal(t, standard_character, rune(shared_character),
			"Decode_Final_Character(%q) character", value)
		testify.Equal(t, standard_size, int(shared_size),
			"Decode_Final_Character(%q) size", value)
	}
	for _, value := range []string{
		repeat("a", utf8.SEQUENCE_SIZE_MAXIMUM-1) + "\x00",
		repeat("a", utf8.SEQUENCE_SIZE_MAXIMUM-1) + "\x01",
		repeat("a", utf8.SEQUENCE_SIZE_MAXIMUM-1) + "\x02",
		repeat("a", utf8.SEQUENCE_SIZE_MAXIMUM-4) + "\U0010ffff",
	} {
		utf8.Decode_Character_Text(utf8.Text(value))
		utf8.Decode_Character(utf8.Bytes(value))
		utf8.Decode_Final_Character_Text(utf8.Text(value))
		utf8.Decode_Final_Character(utf8.Bytes(value))
	}
}

// Test_Encoding preserves the upstream behavior coverage.
func Test_Encoding(t *testing.T) {
	t.Parallel()
	for _, character := range []utf8.Character{
		utf8.Character(bits.INTEGER_32_MINIMUM), -1, 0, 1, 2, 0x7f, 0x80, 0x7ff, 0x800,
		0xd7ff, 0xd800,
		0xdfff, 0xe000, 0xffff, 0x10000, utf8.RUNE_MAX,
		utf8.Character(bits.INTEGER_32_MAXIMUM),
	} {
		testify.Equal(t, standard_utf8.RuneLen(rune(character)),
			int(utf8.Character_Size(character)), "Character_Size(%U)", character)
		standard_buffer := make([]byte, standard_utf8.UTFMax)
		standard_size := standard_utf8.EncodeRune(standard_buffer, rune(character))
		shared_buffer := make(utf8.Bytes, utf8.UTF_MAXIMUM)
		shared_size := utf8.Encode_Character(shared_buffer, character)
		testify.Equal(t, standard_size, int(shared_size),
			"Encode_Character(%U) size", character)
		testify.Equal(t, standard_buffer[:standard_size],
			[]byte(shared_buffer[:shared_size]),
			"Encode_Character(%U) bytes", character)
		testify.Equal(t, standard_utf8.AppendRune([]byte("x"), rune(character)),
			[]byte(utf8.Append_Character(utf8.Bytes("x"), character)),
			"Append_Character(%U)", character)
	}
	utf8.Encode_Character(make(utf8.Bytes, 2), 0x80)
	utf8.Encode_Character(
		make(utf8.Bytes, utf8.SEQUENCE_SIZE_MAXIMUM),
		utf8.RUNE_MAX,
	)
	testify.Equal(t, []byte("A"),
		[]byte(utf8.Append_Character(nil, 'A')))
	utf8.Append_Character(make(utf8.Bytes, 2), 'A')
	testify.Equal(t, utf8.SEQUENCE_SIZE_MAXIMUM,
		len(utf8.Append_Character(
			make(utf8.Bytes, utf8.SEQUENCE_SIZE_MAXIMUM-1), 'A',
		)))
}

// Test_Character_Count verifies counts for valid and invalid byte sequences.
func Test_Character_Count(t *testing.T) {
	t.Parallel()
	for _, value := range []string{
		"", "ASCII", "☺☻☹", "A¢€𐀀", "\x80\x80", "\xe2\x98", "a\xffb",
	} {
		testify.Equal(t, standard_utf8.RuneCountInString(value),
			int(utf8.Character_Count_Text(utf8.Text(value))),
			"Character_Count_Text(%q)", value)
		testify.Equal(t, standard_utf8.RuneCount([]byte(value)),
			int(utf8.Character_Count(utf8.Bytes(value))),
			"Character_Count(%q)", value)
	}
	maximum := repeat("a", utf8.SEQUENCE_SIZE_MAXIMUM)
	testify.Equal(t, utf8.SEQUENCE_SIZE_MAXIMUM,
		int(utf8.Character_Count_Text(utf8.Text(maximum))))
	testify.Equal(t, utf8.SEQUENCE_SIZE_MAXIMUM,
		int(utf8.Character_Count(utf8.Bytes(maximum))))
	testify.Equal(t, 1, int(utf8.Character_Count_Text("a")))
	testify.Equal(t, 1, int(utf8.Character_Count(utf8.Bytes("a"))))
}

// Test_Validation verifies byte starts, sequences, text, and character validity.
func Test_Validation(t *testing.T) {
	t.Parallel()
	for value := 0; value <= int(bits.WORD_8_MAXIMUM); value++ {
		testify.Equal(t, standard_utf8.RuneStart(byte(value)),
			bool(utf8.Character_Start(utf8.Byte(value))),
			"Character_Start(%x)", value)
	}
	for _, value := range []string{
		"", "ASCII", "☺☻☹", "A¢€𐀀", "\x80", "\xc0\x80", "\xed\xa0\x80",
		"\xf4\x8f\xbf\xbf", "\xf4\x90\x80\x80",
	} {
		testify.Equal(t, standard_utf8.ValidString(value),
			bool(utf8.Valid_Text(utf8.Text(value))),
			"Valid_Text(%q)", value)
		testify.Equal(t, standard_utf8.Valid([]byte(value)),
			bool(utf8.Valid(utf8.Bytes(value))), "Valid(%q)", value)
	}
	maximum := repeat("a", utf8.SEQUENCE_SIZE_MAXIMUM)
	testify.True(t, bool(utf8.Valid_Text(utf8.Text(maximum))))
	testify.True(t, bool(utf8.Valid(utf8.Bytes(maximum))))
	for _, character := range []utf8.Character{
		utf8.Character(bits.INTEGER_32_MINIMUM),
		-1, 0, 1, 2, 0xd7ff, 0xd800, 0xdfff, 0xe000,
		utf8.RUNE_MAX, utf8.RUNE_MAX + 1, utf8.Character(bits.INTEGER_32_MAXIMUM),
	} {
		testify.Equal(t, standard_utf8.ValidRune(rune(character)),
			bool(utf8.Valid_Character(character)),
			"Valid_Character(%U)", character)
	}
}

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Test_Domain_Errors preserves the upstream behavior coverage.
func Test_Domain_Errors(t *testing.T) {
	t.Parallel()
	large_text := string(make([]byte, utf8.SEQUENCE_SIZE_MAXIMUM+1))
	testify.Panics(t, func() {
		utf8.Valid_Text(utf8.Text(large_text))
	}, "oversize Text")
	testify.Panics(t, func() {
		utf8.Valid(utf8.Bytes(large_text))
	}, "oversize Bytes")
	testify.Panics(t, func() {
		utf8.Encode_Character(make(utf8.Bytes, 1), 0x10000)
	}, "short caller storage")
	testify.Panics(t, func() {
		utf8.Encode_Character(nil, 'A')
	}, "empty caller storage")
	maximum := make(utf8.Bytes, utf8.SEQUENCE_SIZE_MAXIMUM)
	testify.Panics(t, func() {
		utf8.Append_Character(maximum, 'A')
	}, "oversize append result")
}

// Test_Encoding_Constants verifies the shared facts behind UTF-8 metadata and payloads.
func Test_Encoding_Constants(t *testing.T) {
	t.Parallel()
	testify.Equal_Values(t, bits.WORD_SIZE/bits.BIT_COUNT_8_MAXIMUM,
		utf8.POINTER_SIZE)
	testify.Equal_Values(t, utf8.CONTINUATION_BYTE, utf8.CONTINUATION_MINIMUM)
	testify.Equal_Values(t, utf8.CONTINUATION_BYTE|utf8.CONTINUATION_MASK,
		utf8.CONTINUATION_MAXIMUM)
	testify.Equal_Values(t, 6, utf8.CONTINUATION_PAYLOAD_BIT_COUNT)
	testify.Equal_Values(t, 7, utf8.FIRST_SIZE_MASK)
	testify.Equal_Values(t, 4, utf8.FIRST_ACCEPTANCE_SHIFT)
	testify.Equal_Values(t, 0b11000000, utf8.CONTINUATION_PREFIX_MASK)
}

func repeat(text string, count int) (repeated string) {
	return string(strings.Repeat(
		strings.Text(text), strings.Repeat_Count(count),
	))
}

// Test_Standard_Library_Character_Map preserves the upstream behavior coverage.
func Test_Standard_Library_Character_Map(t *testing.T) {
	t.Parallel()
	for _, character := range []rune{
		0x0000, 0x0001, 0x007e, 0x007f, 0x0080, 0x0081, 0x00bf, 0x00c0,
		0x00ff, 0x0100, 0x0400, 0x07ff, 0x0800, 0x0801, 0x1000, 0xd000,
		0xd7ff, 0xe000, 0xfffe, 0xffff, 0x10000, 0x10001, 0x40000,
		0x10fffe, 0x10ffff, 0xfffd,
	} {
		standard_buffer := make([]byte, standard_utf8.UTFMax)
		standard_size := standard_utf8.EncodeRune(standard_buffer, character)
		standard_encoding := standard_buffer[:standard_size]
		shared_buffer := make(utf8.Bytes, utf8.UTF_MAXIMUM)
		shared_size := utf8.Encode_Character(
			shared_buffer, utf8.Character(character),
		)
		testify.Equal(t, standard_encoding, []byte(shared_buffer[:shared_size]),
			"Encode_Character(%U)", character)
		shared_character, decoded_size := utf8.Decode_Character(
			utf8.Bytes(standard_encoding),
		)
		testify.Equal(t, character, rune(shared_character),
			"Decode_Character(%U)", character)
		testify.Equal(t, standard_size, int(decoded_size),
			"Decode_Character(%U) size", character)
		shared_character, decoded_size = utf8.Decode_Character_Text(
			utf8.Text(standard_encoding),
		)
		testify.Equal(t, character, rune(shared_character),
			"Decode_Character_Text(%U)", character)
		testify.Equal(t, standard_size, int(decoded_size),
			"Decode_Character_Text(%U) size", character)
	}
}

// Test_Standard_Library_First_Bytes preserves the upstream behavior coverage.
func Test_Standard_Library_First_Bytes(t *testing.T) {
	t.Parallel()
	for value := 0; value <= 0xff; value++ {
		sequence := []byte{byte(value)}
		text := string(sequence)
		testify.Equal(t, standard_utf8.FullRune(sequence),
			bool(utf8.Full_Character(utf8.Bytes(sequence))),
			"Full_Character(%x)", value)
		testify.Equal(t, standard_utf8.FullRuneInString(text),
			bool(utf8.Full_Character_Text(utf8.Text(text))),
			"Full_Character_Text(%x)", value)
		standard_character, standard_size := standard_utf8.DecodeRune(sequence)
		shared_character, shared_size := utf8.Decode_Character(
			utf8.Bytes(sequence),
		)
		testify.Equal(t, standard_character, rune(shared_character),
			"Decode_Character(%x) character", value)
		testify.Equal(t, standard_size, int(shared_size),
			"Decode_Character(%x) size", value)
		standard_character, standard_size = standard_utf8.DecodeRuneInString(text)
		shared_character, shared_size = utf8.Decode_Character_Text(
			utf8.Text(text),
		)
		testify.Equal(t, standard_character, rune(shared_character),
			"Decode_Character_Text(%x) character", value)
		testify.Equal(t, standard_size, int(shared_size),
			"Decode_Character_Text(%x) size", value)
	}
}

// Test_Standard_Library_Invalid_Sequences preserves the upstream behavior coverage.
func Test_Standard_Library_Invalid_Sequences(t *testing.T) {
	t.Parallel()
	for _, value := range []string{
		"\x80", "\xbf", "\xc0\x80", "\xc1\xbf", "\xe0\x80\x80", "\xed\xa0\x80",
		"\xf0\x80\x80\x80", "\xf4\x90\x80\x80", "\xf5\x80\x80\x80",
		"\xe2\x28\xa1", "\xf0\x90\x28\xbc", "\xf0\x28\x8c\xbc",
	} {
		testify.Equal(t, standard_utf8.ValidString(value),
			bool(utf8.Valid_Text(utf8.Text(value))),
			"Valid_Text(%q)", value)
		testify.Equal(t, standard_utf8.Valid([]byte(value)),
			bool(utf8.Valid(utf8.Bytes(value))), "Valid(%q)", value)
		standard_character, standard_size := standard_utf8.DecodeLastRuneInString(value)
		shared_character, shared_size := utf8.Decode_Final_Character_Text(
			utf8.Text(value),
		)
		testify.Equal(t, standard_character, rune(shared_character),
			"Decode_Final_Character_Text(%q) character", value)
		testify.Equal(t, standard_size, int(shared_size),
			"Decode_Final_Character_Text(%q) size", value)
	}
}
