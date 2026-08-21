// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package utf8_test

import (
	"testing"

	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/testify"
	"local/james-orcales/shared/unicode/utf8"
)

// Test_Complete_Sequences verifies complete, incomplete, and invalid first encodings.
func Test_Complete_Sequences(t *testing.T) {
	t.Parallel()
	for _, value := range []string{
		"", "A", "¢", "€", "𐀀", "\xc2", "\xe2\x82", "\xf0\x90\x80", "\x80", "\xc0",
	} {
		testify.Equal(t, reference_full_character(value),
			bool(utf8.Full_Character_Text(utf8.Text(value))),
			"Full_Character_Text(%q)", value)
		testify.Equal(t, reference_full_character(value),
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
		standard_character, standard_size := reference_decode_character(value)
		shared_character, shared_size := utf8.Decode_Character_Text(
			utf8.Text(value),
		)
		testify.Equal(t, standard_character, rune(shared_character),
			"Decode_Character_Text(%q) character", value)
		testify.Equal(t, standard_size, int(shared_size),
			"Decode_Character_Text(%q) size", value)
		standard_character, standard_size = reference_decode_character(value)
		shared_character, shared_size = utf8.Decode_Character(
			utf8.Bytes(value),
		)
		testify.Equal(t, standard_character, rune(shared_character),
			"Decode_Character(%q) character", value)
		testify.Equal(t, standard_size, int(shared_size),
			"Decode_Character(%q) size", value)

		standard_character, standard_size = reference_decode_final_character(value)
		shared_character, shared_size = utf8.Decode_Final_Character_Text(
			utf8.Text(value),
		)
		testify.Equal(t, standard_character, rune(shared_character),
			"Decode_Final_Character_Text(%q) character", value)
		testify.Equal(t, standard_size, int(shared_size),
			"Decode_Final_Character_Text(%q) size", value)
		standard_character, standard_size = reference_decode_final_character(value)
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
		testify.Equal(t, reference_character_size(rune(character)),
			int(utf8.Character_Size(character)), "Character_Size(%U)", character)
		standard_buffer := make([]byte, utf8.UTF_MAXIMUM)
		standard_size := reference_encode_character(standard_buffer, rune(character))
		shared_buffer := make(utf8.Bytes, utf8.UTF_MAXIMUM)
		shared_size := utf8.Encode_Character(shared_buffer, character)
		testify.Equal(t, standard_size, int(shared_size),
			"Encode_Character(%U) size", character)
		testify.Equal(t, standard_buffer[:standard_size],
			[]byte(shared_buffer[:shared_size]),
			"Encode_Character(%U) bytes", character)
		var append_storage [utf8.UTF_MAXIMUM + 1]byte
		append_storage[0] = 'x'
		testify.Equal(t, reference_append_character([]byte("x"), rune(character)),
			[]byte(utf8.Append_Character(append_storage[:1], character)),
			"Append_Character(%U)", character)
	}
	utf8.Encode_Character(make(utf8.Bytes, 2), 0x80)
	utf8.Encode_Character(
		make(utf8.Bytes, utf8.SEQUENCE_SIZE_MAXIMUM),
		utf8.RUNE_MAX,
	)
	var append_storage [utf8.UTF_MAXIMUM]byte
	testify.Equal(t, []byte("A"),
		[]byte(utf8.Append_Character(append_storage[:0], 'A')))
	utf8.Append_Character(make(utf8.Bytes, 2, 2+utf8.UTF_MAXIMUM), 'A')
	testify.Equal(t, utf8.SEQUENCE_SIZE_MAXIMUM,
		len(utf8.Append_Character(
			make(utf8.Bytes, utf8.SEQUENCE_SIZE_MAXIMUM-1,
				utf8.SEQUENCE_SIZE_MAXIMUM), 'A',
		)))
}

// Test_Character_Count verifies counts for valid and invalid byte sequences.
func Test_Character_Count(t *testing.T) {
	t.Parallel()
	for _, value := range []string{
		"", "ASCII", "☺☻☹", "A¢€𐀀", "\x80\x80", "\xe2\x98", "a\xffb",
	} {
		testify.Equal(t, reference_character_count(value),
			int(utf8.Character_Count_Text(utf8.Text(value))),
			"Character_Count_Text(%q)", value)
		testify.Equal(t, reference_character_count(value),
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
		testify.Equal(t, byte(value)&0xc0 != 0x80,
			bool(utf8.Character_Start(utf8.Byte(value))),
			"Character_Start(%x)", value)
	}
	for _, value := range []string{
		"", "ASCII", "☺☻☹", "A¢€𐀀", "\x80", "\xc0\x80", "\xed\xa0\x80",
		"\xf4\x8f\xbf\xbf", "\xf4\x90\x80\x80",
	} {
		testify.Equal(t, reference_valid(value),
			bool(utf8.Valid_Text(utf8.Text(value))),
			"Valid_Text(%q)", value)
		testify.Equal(t, reference_valid(value),
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
		testify.Equal(t, reference_valid_character(rune(character)),
			bool(utf8.Valid_Character(character)),
			"Valid_Character(%U)", character)
	}
}

// Test_Allocation proves each public operation keeps heap allocation at zero.
func Test_Allocation(t *testing.T) {
	state := allocation_state{
		Source:  utf8.Bytes("A\xe4\xb8\x96"),
		Text:    "A世",
		Storage: make(utf8.Bytes, utf8.UTF_MAXIMUM),
	}
	for _, one := range allocation_cases(&state) {
		t.Run(one.Name, func(t *testing.T) { testify.Zero_Allocation(t, one.Run) })
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
	testify.Panics(t, func() {
		utf8.Append_Character(nil, '世')
	}, "missing append storage")
}

type allocation_case struct {
	Name string
	Run  func()
}

// Storage is utf8.Bytes made once in Test_Allocation: lint bans fixed array fields, and a
// make inside Run would count as an allocation.
type allocation_state struct {
	Bytes          utf8.Nonempty_Bytes
	Storage        utf8.Bytes
	Boolean        utf8.Boolean
	Character      utf8.Decoded_Character
	Size           utf8.Decoded_Size
	Character_Size utf8.Size
	Encoded_Size   utf8.Encoded_Size
	Count          utf8.Count
	Source         utf8.Bytes
	Text           utf8.Text
}

func allocation_cases(state *allocation_state) (cases []allocation_case) {
	return []allocation_case{
		{Name: "Full_Character", Run: func() {
			state.Boolean = utf8.Full_Character(state.Source)
		}},
		{Name: "Full_Character_Text", Run: func() {
			state.Boolean = utf8.Full_Character_Text(state.Text)
		}},
		{Name: "Decode_Character", Run: func() {
			state.Character, state.Size = utf8.Decode_Character(state.Source[1:])
		}},
		{Name: "Decode_Character_Text", Run: func() {
			state.Character, state.Size = utf8.Decode_Character_Text("世")
		}},
		{Name: "Decode_Final_Character", Run: func() {
			state.Character, state.Size = utf8.Decode_Final_Character(state.Source)
		}},
		{Name: "Decode_Final_Character_Text", Run: func() {
			state.Character, state.Size =
				utf8.Decode_Final_Character_Text("A世")
		}},
		{Name: "Character_Size", Run: func() {
			state.Character_Size = utf8.Character_Size('世')
		}},
		{Name: "Encode_Character", Run: func() {
			state.Encoded_Size = utf8.Encode_Character(state.Storage, '世')
		}},
		{Name: "Append_Character", Run: func() {
			state.Bytes = utf8.Append_Character(state.Storage[:0], '世')
		}},
		{Name: "Character_Count", Run: func() {
			state.Count = utf8.Character_Count(state.Source)
		}},
		{Name: "Character_Count_Text", Run: func() {
			state.Count = utf8.Character_Count_Text(state.Text)
		}},
		{Name: "Character_Start", Run: func() {
			state.Boolean = utf8.Character_Start('A')
		}},
		{Name: "Valid", Run: func() {
			state.Boolean = utf8.Valid(state.Source)
		}},
		{Name: "Valid_Text", Run: func() {
			state.Boolean = utf8.Valid_Text(state.Text)
		}},
		{Name: "Valid_Character", Run: func() {
			state.Boolean = utf8.Valid_Character('世')
		}},
	}
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
	storage := make([]byte, len(text)*count)
	for copy_index := 0; copy_index < count; copy_index++ {
		copy(storage[copy_index*len(text):], text)
	}
	return string(storage)
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
		standard_buffer := make([]byte, utf8.UTF_MAXIMUM)
		standard_size := reference_encode_character(standard_buffer, character)
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
		testify.Equal(t, reference_full_character(text),
			bool(utf8.Full_Character(utf8.Bytes(sequence))),
			"Full_Character(%x)", value)
		testify.Equal(t, reference_full_character(text),
			bool(utf8.Full_Character_Text(utf8.Text(text))),
			"Full_Character_Text(%x)", value)
		standard_character, standard_size := reference_decode_character(text)
		shared_character, shared_size := utf8.Decode_Character(
			utf8.Bytes(sequence),
		)
		testify.Equal(t, standard_character, rune(shared_character),
			"Decode_Character(%x) character", value)
		testify.Equal(t, standard_size, int(shared_size),
			"Decode_Character(%x) size", value)
		standard_character, standard_size = reference_decode_character(text)
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
		testify.Equal(t, reference_valid(value),
			bool(utf8.Valid_Text(utf8.Text(value))),
			"Valid_Text(%q)", value)
		testify.Equal(t, reference_valid(value),
			bool(utf8.Valid(utf8.Bytes(value))), "Valid(%q)", value)
		standard_character, standard_size := reference_decode_final_character(value)
		shared_character, shared_size := utf8.Decode_Final_Character_Text(
			utf8.Text(value),
		)
		testify.Equal(t, standard_character, rune(shared_character),
			"Decode_Final_Character_Text(%q) character", value)
		testify.Equal(t, standard_size, int(shared_size),
			"Decode_Final_Character_Text(%q) size", value)
	}
}

func reference_full_character(text string) (full bool) {
	if len(text) == 0 {
		return false
	}
	_, size := reference_decode_character(text)
	if size != 1 {
		return true
	}
	if text[0] < 0xc2 {
		return true
	}
	if text[0] > 0xf4 {
		return true
	}
	required := reference_encoded_size(text[0])
	available_count := len(text)
	if available_count > required {
		available_count = required
	}
	for byte_index := 1; byte_index < available_count; byte_index++ {
		minimum, maximum := byte(0x80), byte(0xbf)
		if byte_index == 1 {
			minimum, maximum = reference_second_byte_range(text[0])
		}
		if text[byte_index] < minimum {
			return true
		}
		if text[byte_index] > maximum {
			return true
		}
	}
	return len(text) >= required
}

func reference_decode_character(text string) (character rune, size int) {
	if len(text) == 0 {
		return '\ufffd', 0
	}
	first := text[0]
	if first < 0x80 {
		return rune(first), 1
	}
	if first < 0xc2 {
		return '\ufffd', 1
	}
	if first > 0xf4 {
		return '\ufffd', 1
	}
	required := reference_encoded_size(first)
	if required == 1 {
		return '\ufffd', 1
	}
	if len(text) < required {
		return '\ufffd', 1
	}
	second_minimum, second_maximum := reference_second_byte_range(first)
	if text[1] < second_minimum {
		return '\ufffd', 1
	}
	if text[1] > second_maximum {
		return '\ufffd', 1
	}
	character = rune(first & (0x7f >> required))
	for byte_index := 1; byte_index < required; byte_index++ {
		if byte_index > 1 {
			if text[byte_index] < 0x80 {
				return '\ufffd', 1
			}
			if text[byte_index] > 0xbf {
				return '\ufffd', 1
			}
		}
		character = character<<6 | rune(text[byte_index]&0x3f)
	}
	return character, required
}

func reference_decode_final_character(text string) (character rune, byte_size int) {
	if len(text) == 0 {
		return '\ufffd', 0
	}
	minimum_index := len(text) - utf8.UTF_MAXIMUM
	if minimum_index < 0 {
		minimum_index = 0
	}
	for byte_index := len(text) - 1; byte_index >= minimum_index; byte_index-- {
		if text[byte_index]&0xc0 == 0x80 {
			continue
		}
		character, byte_size = reference_decode_character(text[byte_index:])
		byte_count := byte_size
		final_byte_index := byte_index + byte_count
		if final_byte_index == len(text) {
			return character, byte_size
		}
		break
	}
	return '\ufffd', 1
}

func reference_encoded_size(first byte) (size int) {
	switch {
	case first < 0x80:
		return 1
	case first < 0xe0:
		return 2
	case first < 0xf0:
		return 3
	case first < 0xf5:
		return 4
	default:
		return 1
	}
}

func reference_second_byte_range(first byte) (minimum byte, maximum byte) {
	switch first {
	case 0xe0:
		return 0xa0, 0xbf
	case 0xed:
		return 0x80, 0x9f
	case 0xf0:
		return 0x90, 0xbf
	case 0xf4:
		return 0x80, 0x8f
	default:
		return 0x80, 0xbf
	}
}

func reference_character_size(character rune) (size int) {
	switch {
	case character < 0:
		return -1
	case reference_is_surrogate(character):
		return -1
	case character <= 0x7f:
		return 1
	case character <= 0x7ff:
		return 2
	case character <= 0xffff:
		return 3
	case character <= 0x10ffff:
		return 4
	default:
		return -1
	}
}

func reference_encode_character(storage []byte, character rune) (size int) {
	if !reference_valid_character(character) {
		character = '\ufffd'
	}
	size = copy(storage, string(character))
	return size
}

func reference_append_character(storage []byte, character rune) (result []byte) {
	var encoded [utf8.UTF_MAXIMUM]byte
	size := reference_encode_character(encoded[:], character)
	return append(storage, encoded[:size]...)
}

func reference_character_count(text string) (count int) {
	for len(text) > 0 {
		_, size := reference_decode_character(text)
		count++
		text = text[size:]
	}
	return count
}

func reference_valid(text string) (valid bool) {
	for len(text) > 0 {
		character, size := reference_decode_character(text)
		if character == '\ufffd' {
			if size == 1 {
				return false
			}
		}
		text = text[size:]
	}
	return true
}

func reference_valid_character(character rune) (valid bool) {
	if character < 0 {
		return false
	}
	if character > 0x10ffff {
		return false
	}
	return !reference_is_surrogate(character)
}

func reference_is_surrogate(character rune) (yes bool) {
	if character < 0xd800 {
		return false
	}
	return character < 0xe000
}
