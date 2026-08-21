// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package utf8_test

import (
	"testing"

	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/testify"
	"local/james-orcales/shared/unicode/ucd"
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

// TEXT_STORAGE_SIZE leaves room for every small transform result under test.
const TEXT_STORAGE_SIZE = 64

// TEXT_BUFFER_STORAGE_SIZE leaves room for two maximum-width encodings.
const TEXT_BUFFER_STORAGE_SIZE = 8

// Test_Text_Search keeps decoded search from drifting back into raw-byte semantics.
func Test_Text_Search(t *testing.T) {
	source := utf8.Bytes("a☺b-a")
	if !utf8.Contains_Any(source, "x☺") {
		t.Fatal("Contains_Any missed character")
	}
	if !utf8.Contains_Rune(source, '☺') {
		t.Fatal("Contains_Rune missed character")
	}
	if !utf8.Contains_Function(source, text_is_dash) {
		t.Fatal("Contains_Function missed predicate")
	}
	if utf8.Index_Rune(source, '☺') != 1 {
		t.Fatal("Index_Rune returned wrong byte index")
	}
	if utf8.Index_Any(source, "☺x") != 1 {
		t.Fatal("Index_Any returned wrong byte index")
	}
	if utf8.Last_Index_Any(source, "a") != 6 {
		t.Fatal("Last_Index_Any missed final character")
	}
	if utf8.Index_Function(source, text_is_dash) != 5 {
		t.Fatal("Index_Function returned wrong byte index")
	}
	if utf8.Last_Index_Function(source, text_is_letter_a) != 6 {
		t.Fatal("Last_Index_Function missed final predicate match")
	}
	if !utf8.Equal_Fold(utf8.Bytes("Go"), utf8.Bytes("gO")) {
		t.Fatal("Equal_Fold rejected Unicode fold")
	}
	if utf8.Equal_Fold(utf8.Bytes("Go"), utf8.Bytes("gone")) {
		t.Fatal("Equal_Fold accepted unequal input")
	}
}

// Test_Text_Fields keeps delimiter predicates attached to decoded characters.
func Test_Text_Fields(t *testing.T) {
	var slots [8]utf8.Bytes
	count := utf8.Fields_Into(slots[:], utf8.Bytes(" a\tb "))
	text_assert_slices(t, slots[:count], []string{"a", "b"})
	count = utf8.Fields_Function_Into(
		slots[:], utf8.Bytes("a-b--c"), text_is_dash,
	)
	text_assert_slices(t, slots[:count], []string{"a", "b", "c"})
	if cap(slots[0]) != len(slots[0]) {
		t.Fatal("Fields_Into returned unclipped view")
	}
}

// Test_Text_Transform keeps mapping and repair at decoded UTF-8 boundaries.
func Test_Text_Transform(t *testing.T) {
	var storage [TEXT_STORAGE_SIZE]byte
	count := utf8.Map_Into(storage[:], text_map_character, utf8.Bytes("abx"))
	if string(storage[:count]) != "☺b" {
		t.Fatal("Map_Into wrote wrong content")
	}
	count = utf8.To_Upper_Into(storage[:], utf8.Bytes("Go"))
	if string(storage[:count]) != "GO" {
		t.Fatal("To_Upper_Into wrote wrong content")
	}
	count = utf8.To_Lower_Into(storage[:], utf8.Bytes("Go"))
	if string(storage[:count]) != "go" {
		t.Fatal("To_Lower_Into wrote wrong content")
	}
	count = utf8.To_Title_Into(storage[:], utf8.Bytes("ǳ"))
	if string(storage[:count]) != "ǲ" {
		t.Fatal("To_Title_Into wrote wrong content")
	}
	var special ucd.Special_Case
	ucd.Turkish_Case(&special)
	count = utf8.To_Upper_Special_Into(storage[:], special, utf8.Bytes("i"))
	if string(storage[:count]) != "İ" {
		t.Fatal("To_Upper_Special_Into ignored language case")
	}
	count = utf8.To_Lower_Special_Into(storage[:], special, utf8.Bytes("I"))
	if string(storage[:count]) != "ı" {
		t.Fatal("To_Lower_Special_Into ignored language case")
	}
	count = utf8.To_Title_Special_Into(storage[:], special, utf8.Bytes("i"))
	if string(storage[:count]) != "İ" {
		t.Fatal("To_Title_Special_Into ignored language case")
	}
	count = utf8.To_Valid_UTF8_Into(
		storage[:], utf8.Bytes{'a', 0xff, 0xfe, 'b'}, utf8.Bytes("?"),
	)
	if string(storage[:count]) != "a?b" {
		t.Fatal("To_Valid_UTF8_Into wrote wrong repair")
	}
	count = utf8.Title_Into(storage[:], utf8.Bytes("go gopher"))
	if string(storage[:count]) != "Go Gopher" {
		t.Fatal("Title_Into wrote wrong content")
	}
	var characters [2]utf8.Decoded_Character
	character_count := utf8.Runes_Into(characters[:], utf8.Bytes{'a', 0xff})
	if character_count != 2 {
		t.Fatal("Runes_Into returned wrong count")
	}
	if characters[0] != 'a' || characters[1] != utf8.REPLACEMENT_CHARACTER {
		t.Fatal("Runes_Into wrote wrong characters")
	}
}

// Test_Text_Trim keeps cut sets and predicates on decoded character boundaries.
func Test_Text_Trim(t *testing.T) {
	source := utf8.Bytes("xx abc xx")
	if string(utf8.Trim(source, "x")) != " abc " {
		t.Fatal("Trim returned wrong view")
	}
	if string(utf8.Trim_Left(source, "x")) != " abc xx" {
		t.Fatal("Trim_Left returned wrong view")
	}
	if string(utf8.Trim_Right(source, "x")) != "xx abc " {
		t.Fatal("Trim_Right returned wrong view")
	}
	if string(utf8.Trim_Function(utf8.Bytes("--a--"), text_is_dash)) != "a" {
		t.Fatal("Trim_Function returned wrong view")
	}
	if string(utf8.Trim_Left_Function(utf8.Bytes("--a"), text_is_dash)) != "a" {
		t.Fatal("Trim_Left_Function returned wrong view")
	}
	if string(utf8.Trim_Right_Function(utf8.Bytes("a--"), text_is_dash)) != "a" {
		t.Fatal("Trim_Right_Function returned wrong view")
	}
	if string(utf8.Trim_Space(utf8.Bytes("\u2000 x \u2000"))) != "x" {
		t.Fatal("Trim_Space retained Unicode whitespace")
	}
}

// Test_Text_Iteration keeps yielded views aligned to decoded field boundaries.
func Test_Text_Iteration(t *testing.T) {
	var yielded [8]utf8.Bytes
	yielded_count := 0
	count := utf8.Fields_Sequence(
		utf8.Bytes(" a b "),
		func(field utf8.Bytes) (continued utf8.Boolean) {
			yielded[yielded_count] = field
			yielded_count++
			return true
		},
	)
	text_assert_slices(t, yielded[:count], []string{"a", "b"})
	yielded_count = 0
	count = utf8.Fields_Function_Sequence(
		utf8.Bytes("a-b-c"), text_is_dash,
		func(field utf8.Bytes) (continued utf8.Boolean) {
			yielded[yielded_count] = field
			yielded_count++
			return utf8.Boolean(yielded_count < 2)
		},
	)
	if count != 2 {
		t.Fatal("Fields_Function_Sequence ignored early stop")
	}
	text_assert_slices(t, yielded[:count], []string{"a", "b"})
}

// Test_Buffer_Character_Operations keeps UTF-8 cursor state outside raw-byte ownership.
func Test_Buffer_Character_Operations(t *testing.T) {
	var storage [TEXT_BUFFER_STORAGE_SIZE]byte
	var buffer bytes.Buffer
	bytes.Buffer_Init(&buffer, storage[:], nil)
	utf8.Buffer_Write_Character(&buffer, '☺')
	character, size, found := utf8.Buffer_Read_Character(&buffer)
	if !found {
		t.Fatal("Buffer_Read_Character returned wrong encoding")
	}
	if character != '☺' {
		t.Fatal("Buffer_Read_Character returned wrong encoding")
	}
	if size != 3 {
		t.Fatal("Buffer_Read_Character returned wrong encoding")
	}
	utf8.Buffer_Unread_Character(&buffer)
	character, size, found = utf8.Buffer_Read_Character(&buffer)
	if !found || character != '☺' || size != 3 {
		t.Fatal("Buffer_Unread_Character restored wrong boundary")
	}
}

// Test_Reader_Character_Operations keeps grouped reads above raw Reader ownership.
func Test_Reader_Character_Operations(t *testing.T) {
	var reader bytes.Reader
	bytes.Reader_Reset(&reader, bytes.Slice("☺a"))
	character, size, found := utf8.Reader_Read_Character(&reader)
	if !found || character != '☺' || size != 3 {
		t.Fatal("Reader_Read_Character returned wrong encoding")
	}
	utf8.Reader_Unread_Character(&reader)
	character, size, found = utf8.Reader_Read_Character(&reader)
	if !found || character != '☺' || size != 3 {
		t.Fatal("Reader_Unread_Character restored wrong boundary")
	}
}

// Test_Text_Allocation protects caller-owned storage across the moved API boundary.
func Test_Text_Allocation(t *testing.T) {
	fixture := text_allocation_fixture{
		Storage:     make(utf8.Bytes, 64),
		Source:      utf8.Bytes("a-b"),
		Field_Slots: make(utf8.Field_Slices, 4),
		Characters:  make(utf8.Characters, 4),
	}
	ucd.Turkish_Case(&fixture.Special)
	for _, one := range text_allocation_cases(&fixture) {
		t.Run(one.Name, func(t *testing.T) {
			testify.Zero_Allocation(t, one.Run)
		})
	}
}

// Test_Text_API_Domains keeps every moved contract observable in its new package.
func Test_Text_API_Domains(_ *testing.T) {
	maximum := make(utf8.Bytes, utf8.SEQUENCE_SIZE_MAXIMUM)
	alternate := make(utf8.Bytes, utf8.SEQUENCE_SIZE_MAXIMUM)
	fields := make(utf8.Bytes, utf8.SEQUENCE_SIZE_MAXIMUM)
	field_slots := make(utf8.Field_Slices, utf8.FIELD_COUNT_MAXIMUM)
	characters := make(utf8.Characters, utf8.CHARACTER_COUNT_MAXIMUM)
	for index := range maximum {
		maximum[index] = 'a'
		fields[index] = 'a'
		if index%2 == 1 {
			fields[index] = ' '
		}
	}
	maximum[0] = 0
	maximum[1] = 1
	maximum[2] = 2
	maximum[utf8.BYTE_INDEX_MAXIMUM] = 'z'

	text_cover_search_domains(maximum)
	text_cover_transform_domains(maximum, alternate)
	text_cover_field_domains(fields, field_slots)
	text_cover_trim_domains(maximum)
	text_cover_character_domains(maximum, characters)
	text_cover_special_domains(alternate)
	text_cover_buffer_domains(alternate)
	text_cover_reader_domains(maximum)
}

func text_cover_search_domains(maximum utf8.Bytes) {
	sources := [...]utf8.Bytes{nil, maximum[:1], maximum[:2], maximum[:]}
	texts := [...]utf8.Text{"", "a", "aa", utf8.Text(string(maximum))}
	for index, source := range sources {
		text := texts[index]
		utf8.Contains_Any(source, text)
		utf8.Contains_Function(source, text_never_match)
		utf8.Index_Any(source, text)
		utf8.Last_Index_Any(source, text)
		utf8.Index_Function(source, text_never_match)
		utf8.Last_Index_Function(source, text_never_match)
		utf8.Equal_Fold(source, source)
	}
	characters := [...]utf8.Character{-2147483648, -1, 0, 1, 2, 2147483647}
	for _, character := range characters {
		utf8.Contains_Rune(nil, character)
		utf8.Index_Rune(nil, character)
	}
	utf8.Contains_Rune(maximum[:1], 0)
	utf8.Contains_Rune(maximum[:2], 0)
	utf8.Index_Rune(maximum[:1], 0)
	utf8.Index_Rune(maximum[:2], 0)
	utf8.Contains_Any(maximum, "z")
	utf8.Contains_Rune(maximum, 'z')
	utf8.Contains_Function(maximum, text_is_z)
	utf8.Index_Rune(maximum, 0)
	utf8.Index_Rune(maximum, 1)
	utf8.Index_Rune(maximum, 2)
	utf8.Index_Rune(maximum, 'z')
	utf8.Index_Any(maximum, "\x00")
	utf8.Index_Any(maximum, "\x01")
	utf8.Index_Any(maximum, "\x02")
	utf8.Index_Any(maximum, "z")
	utf8.Last_Index_Any(maximum, "z")
	utf8.Last_Index_Any(maximum, "\x00")
	utf8.Last_Index_Any(maximum, "\x01")
	utf8.Last_Index_Any(maximum, "\x02")
	utf8.Index_Function(maximum, text_is_zero)
	utf8.Index_Function(maximum, text_is_one)
	utf8.Index_Function(maximum, text_is_two)
	utf8.Index_Function(maximum, text_is_z)
	utf8.Last_Index_Function(maximum, text_is_z)
	utf8.Last_Index_Function(maximum, text_is_zero)
	utf8.Last_Index_Function(maximum, text_is_one)
	utf8.Last_Index_Function(maximum, text_is_two)
	utf8.Equal_Fold(maximum[:1], maximum[1:2])
	var maximum_character [utf8.UTF_MAXIMUM]byte
	encoded_size := utf8.Encode_Character(maximum_character[:], utf8.RUNE_MAX)
	utf8.Index_Any(
		maximum_character[:encoded_size],
		utf8.Text(string(maximum_character[:encoded_size])),
	)
	var title_source [utf8.UTF_MAXIMUM + 1]byte
	copy(title_source[:], maximum_character[:encoded_size])
	title_source[encoded_size] = 'a'
	var title_destination [utf8.UTF_MAXIMUM + 1]byte
	utf8.Title_Into(title_destination[:], title_source[:])
}

func text_cover_transform_domains(maximum utf8.Bytes, alternate utf8.Bytes) {
	special := ucd.Special_Case{}
	ucd.Turkish_Case(&special)
	sizes := [...]int{0, 1, 2, utf8.SEQUENCE_SIZE_MAXIMUM}
	for _, size := range sizes {
		source := maximum[:size]
		destination := alternate[:size]
		utf8.Map_Into(destination, text_identity, source)
		utf8.To_Upper_Into(destination, source)
		utf8.To_Lower_Into(destination, source)
		utf8.To_Title_Into(destination, source)
		utf8.To_Upper_Special_Into(destination, special, source)
		utf8.To_Lower_Special_Into(destination, special, source)
		utf8.To_Title_Special_Into(destination, special, source)
		utf8.To_Valid_UTF8_Into(destination, source, nil)
		utf8.Title_Into(destination, source)
	}
	replacements := [...]utf8.Bytes{
		nil, maximum[:1], maximum[:2], maximum[:],
	}
	for _, replacement := range replacements {
		utf8.To_Valid_UTF8_Into(nil, nil, replacement)
	}
	utf8.Map_Into(alternate[:0], text_delete, maximum[:1])
}

func text_cover_field_domains(fields utf8.Bytes, slots utf8.Field_Slices) {
	destinations := [...]utf8.Field_Slices{nil, slots[:1], slots[:2], slots[:]}
	for _, destination := range destinations {
		utf8.Fields_Into(destination, nil)
		utf8.Fields_Function_Into(destination, nil, text_is_space)
	}
	sources := [...]utf8.Bytes{nil, fields[:1], fields[:2], fields[:3], fields[:]}
	for _, source := range sources {
		utf8.Fields_Into(slots, source)
		utf8.Fields_Function_Into(slots, source, text_is_space)
		utf8.Fields_Sequence(source, text_accept)
		utf8.Fields_Function_Sequence(source, text_is_space, text_accept)
	}
}

func text_cover_trim_domains(maximum utf8.Bytes) {
	sources := [...]utf8.Bytes{nil, maximum[:1], maximum[:2], maximum[:]}
	texts := [...]utf8.Text{"", "a", "aa", utf8.Text(string(maximum))}
	for index, source := range sources {
		utf8.Trim_Left_Function(source, text_never_match)
		utf8.Trim_Right_Function(source, text_never_match)
		utf8.Trim_Function(source, text_never_match)
		utf8.Trim(source, texts[index])
		utf8.Trim_Left(source, texts[index])
		utf8.Trim_Right(source, texts[index])
		utf8.Trim_Space(source)
	}
	utf8.Trim(maximum, "")
	utf8.Trim_Left(maximum, "")
	utf8.Trim_Right(maximum, "")
}

func text_cover_character_domains(
	maximum utf8.Bytes, characters utf8.Characters,
) {
	destinations := [...]utf8.Characters{
		nil, characters[:1], characters[:2], characters[:],
	}
	for _, destination := range destinations {
		utf8.Runes_Into(destination, nil)
	}
	sources := [...]utf8.Bytes{nil, maximum[:1], maximum[:2], maximum[:]}
	for _, source := range sources {
		utf8.Runes_Into(characters, source)
	}
}

func text_cover_special_domains(destination utf8.Bytes) {
	counts := [...]ucd.Special_Case_Count{0, 1, 2, 4, 4, 4}
	characters := [...]ucd.Character{0, 1, 2, 0, 0, ucd.RUNE_MAX}
	deltas := [...]int32{
		ucd.CASE_DELTA_MINIMUM, -1, 0, 1, 2, ucd.CASE_DELTA_MAXIMUM,
	}
	for index, character := range characters {
		delta := deltas[index]
		rule := ucd.Case_Range{
			Minimum: ucd.Case_Range_Minimum(character),
			Maximum: ucd.Case_Range_Maximum(character),
			Deltas: ucd.Case_Delta{
				Upper: ucd.Upper_Case_Delta(delta),
				Lower: ucd.Lower_Case_Delta(delta),
				Title: ucd.Title_Case_Delta(delta),
			},
		}
		special := ucd.Special_Case_Of(
			counts[index], rule, rule, rule, rule,
		)
		utf8.To_Upper_Special_Into(destination[:0], special, nil)
		utf8.To_Lower_Special_Into(destination[:0], special, nil)
		utf8.To_Title_Special_Into(destination[:0], special, nil)
	}
}

func text_cover_buffer_domains(storage utf8.Bytes) {
	states := [...]bytes.Buffer{
		{Content: bytes.Slice(storage[:0:0]), Position: 0, Operation: -1},
		{Content: bytes.Slice(storage[:1:1]), Position: 1, Operation: 0},
		{Content: bytes.Slice(storage[:2:2]), Position: 2, Operation: 1},
		{Content: bytes.Slice(storage[:2:2]), Position: 2, Operation: 2},
		{Content: bytes.Slice(storage[:3:3]), Position: 3, Operation: 3},
		{
			Content:   bytes.Slice(storage[:]),
			Position:  bytes.BOUNDARY_INDEX_MAXIMUM,
			Operation: bytes.READ_OPERATION_MAXIMUM,
		},
	}
	for _, state := range states {
		candidate := state
		text_panic(func() { utf8.Buffer_Write_Character(&candidate, 0) })
		candidate = state
		utf8.Buffer_Read_Character(&candidate)
		candidate = state
		text_panic(func() { utf8.Buffer_Unread_Character(&candidate) })
	}
	characters := [...]utf8.Character{'a', 0x80, 0x800, 0x10000}
	for _, character := range characters {
		var buffer bytes.Buffer
		bytes.Buffer_Init(&buffer, bytes.Slice(storage), nil)
		utf8.Buffer_Write_Character(&buffer, character)
	}
	character_domain := [...]utf8.Character{-2147483648, -1, 0, 1, 2, 2147483647}
	for _, character := range character_domain {
		var buffer bytes.Buffer
		bytes.Buffer_Init(&buffer, bytes.Slice(storage), nil)
		utf8.Buffer_Write_Character(&buffer, character)
	}
	encodings := [...]utf8.Bytes{
		nil, {0}, {1}, {2}, {0xc2, 0x80}, {0xe0, 0xa0, 0x80},
		{0xf4, 0x8f, 0xbf, 0xbf},
	}
	for _, encoding := range encodings {
		var buffer bytes.Buffer
		bytes.Buffer_Init(
			&buffer, bytes.Slice(storage), bytes.Slice(encoding),
		)
		utf8.Buffer_Read_Character(&buffer)
	}
}

func text_cover_reader_domains(source utf8.Bytes) {
	states := [...]bytes.Reader{
		{Source: nil, Position: 0, Previous: -1},
		{Source: bytes.Slice(source[:1]), Position: 1, Previous: 0},
		{Source: bytes.Slice(source[:2]), Position: 2, Previous: 1},
		{Source: bytes.Slice(source[:3]), Position: 3, Previous: 2},
		{
			Source:   bytes.Slice(source[:]),
			Position: bytes.Reader_Position(bytes.READER_POSITION_MAXIMUM),
			Previous: bytes.INDEX_MAXIMUM,
		},
	}
	for _, state := range states {
		candidate := state
		utf8.Reader_Read_Character(&candidate)
		candidate = state
		text_panic(func() { utf8.Reader_Unread_Character(&candidate) })
	}
	encodings := [...]utf8.Bytes{
		nil, {0}, {1}, {2}, {0xc2, 0x80}, {0xe0, 0xa0, 0x80},
		{0xf4, 0x8f, 0xbf, 0xbf},
	}
	for _, encoding := range encodings {
		reader := bytes.Reader{Source: bytes.Slice(encoding), Previous: -1}
		utf8.Reader_Read_Character(&reader)
	}
}

func text_never_match(_ rune) (matches bool) {
	return false
}

func text_is_zero(character rune) (matches bool) {
	return character == 0
}

func text_is_one(character rune) (matches bool) {
	return character == 1
}

func text_is_two(character rune) (matches bool) {
	return character == 2
}

func text_is_z(character rune) (matches bool) {
	return character == 'z'
}

func text_is_space(character rune) (matches bool) {
	return character == ' '
}

func text_is_dash(character rune) (matches bool) {
	return character == '-'
}

func text_is_letter_a(character rune) (matches bool) {
	return character == 'a'
}

func text_map_character(character rune) (mapped rune) {
	if character == 'x' {
		return -1
	}
	if character == 'a' {
		return '☺'
	}
	return character
}

func text_identity(character rune) (mapped rune) {
	return character
}

func text_delete(_ rune) (mapped rune) {
	return -1
}

func text_accept(_ utf8.Bytes) (continued utf8.Boolean) {
	return true
}

func text_panic(action func()) {
	defer func() { recover() }()
	action()
}

func text_assert_slices(t *testing.T, got []utf8.Bytes, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("Slices = %q, want %q", got, want)
	}
	for index := range got {
		if string(got[index]) != want[index] {
			t.Fatalf("Slices = %q, want %q", got, want)
		}
	}
}

type text_allocation_case struct {
	Name string
	Run  func()
}

type text_allocation_fixture struct {
	Storage           utf8.Bytes
	Source            utf8.Bytes
	Field_Slots       utf8.Field_Slices
	Characters        utf8.Characters
	Buffer            bytes.Buffer
	Reader            bytes.Reader
	Special           ucd.Special_Case
	Bytes             utf8.Bytes
	Boolean           utf8.Boolean
	Byte_Index        utf8.Byte_Index
	Byte_Count        utf8.Byte_Count
	Field_Count       utf8.Field_Count
	Count             utf8.Count
	Decoded_Character utf8.Decoded_Character
	Decoded_Size      utf8.Decoded_Size
}

func text_allocation_cases(
	fixture *text_allocation_fixture,
) (cases []text_allocation_case) {
	cases = append(cases, text_cursor_allocation_cases(fixture)...)
	cases = append(cases, text_search_allocation_cases(fixture)...)
	cases = append(cases, text_transform_allocation_cases(fixture)...)
	cases = append(cases, text_trim_allocation_cases(fixture)...)
	return cases
}

func text_cursor_allocation_cases(
	fixture *text_allocation_fixture,
) (cases []text_allocation_case) {
	return []text_allocation_case{
		{Name: "Buffer_Write_Character", Run: func() {
			bytes.Buffer_Init(
				&fixture.Buffer, bytes.Slice(fixture.Storage), nil,
			)
			fixture.Byte_Count = utf8.Byte_Count(
				utf8.Buffer_Write_Character(&fixture.Buffer, 'a'),
			)
		}},
		{Name: "Buffer_Read_Character", Run: func() {
			bytes.Buffer_Init(
				&fixture.Buffer, bytes.Slice(fixture.Storage),
				bytes.Slice(fixture.Source),
			)
			fixture.Decoded_Character, fixture.Decoded_Size, fixture.Boolean =
				utf8.Buffer_Read_Character(&fixture.Buffer)
		}},
		{Name: "Buffer_Unread_Character", Run: func() {
			bytes.Buffer_Init(
				&fixture.Buffer, bytes.Slice(fixture.Storage),
				bytes.Slice(fixture.Source),
			)
			utf8.Buffer_Read_Character(&fixture.Buffer)
			utf8.Buffer_Unread_Character(&fixture.Buffer)
		}},
		{Name: "Reader_Read_Character", Run: func() {
			bytes.Reader_Reset(&fixture.Reader, bytes.Slice(fixture.Source))
			fixture.Decoded_Character, fixture.Decoded_Size, fixture.Boolean =
				utf8.Reader_Read_Character(&fixture.Reader)
		}},
		{Name: "Reader_Unread_Character", Run: func() {
			bytes.Reader_Reset(&fixture.Reader, bytes.Slice(fixture.Source))
			utf8.Reader_Read_Character(&fixture.Reader)
			utf8.Reader_Unread_Character(&fixture.Reader)
		}},
	}
}

func text_search_allocation_cases(
	fixture *text_allocation_fixture,
) (cases []text_allocation_case) {
	return []text_allocation_case{
		{Name: "Contains_Any", Run: func() {
			fixture.Boolean = utf8.Contains_Any(fixture.Source, "a")
		}},
		{Name: "Contains_Rune", Run: func() {
			fixture.Boolean = utf8.Contains_Rune(fixture.Source, 'a')
		}},
		{Name: "Contains_Function", Run: func() {
			fixture.Boolean = utf8.Contains_Function(fixture.Source, text_is_space)
		}},
		{Name: "Index_Rune", Run: func() {
			fixture.Byte_Index = utf8.Index_Rune(fixture.Source, 'a')
		}},
		{Name: "Index_Any", Run: func() {
			fixture.Byte_Index = utf8.Index_Any(fixture.Source, "a")
		}},
		{Name: "Last_Index_Any", Run: func() {
			fixture.Byte_Index = utf8.Last_Index_Any(fixture.Source, "a")
		}},
		{Name: "Index_Function", Run: func() {
			fixture.Byte_Index = utf8.Index_Function(fixture.Source, text_is_space)
		}},
		{Name: "Last_Index_Function", Run: func() {
			fixture.Byte_Index = utf8.Last_Index_Function(fixture.Source, text_is_space)
		}},
		{Name: "Fields_Into", Run: func() {
			fixture.Field_Count = utf8.Fields_Into(fixture.Field_Slots, fixture.Source)
		}},
		{Name: "Fields_Function_Into", Run: func() {
			fixture.Field_Count = utf8.Fields_Function_Into(
				fixture.Field_Slots, fixture.Source, text_is_space,
			)
		}},
	}
}

func text_transform_allocation_cases(
	fixture *text_allocation_fixture,
) (cases []text_allocation_case) {
	return []text_allocation_case{
		{Name: "Map_Into", Run: func() {
			fixture.Byte_Count = utf8.Map_Into(
				fixture.Storage, text_identity, fixture.Source,
			)
		}},
		{Name: "To_Upper_Into", Run: func() {
			fixture.Byte_Count = utf8.To_Upper_Into(fixture.Storage, fixture.Source)
		}},
		{Name: "To_Lower_Into", Run: func() {
			fixture.Byte_Count = utf8.To_Lower_Into(fixture.Storage, fixture.Source)
		}},
		{Name: "To_Title_Into", Run: func() {
			fixture.Byte_Count = utf8.To_Title_Into(fixture.Storage, fixture.Source)
		}},
		{Name: "To_Upper_Special_Into", Run: func() {
			fixture.Byte_Count = utf8.To_Upper_Special_Into(
				fixture.Storage, fixture.Special, fixture.Source,
			)
		}},
		{Name: "To_Lower_Special_Into", Run: func() {
			fixture.Byte_Count = utf8.To_Lower_Special_Into(
				fixture.Storage, fixture.Special, fixture.Source,
			)
		}},
		{Name: "To_Title_Special_Into", Run: func() {
			fixture.Byte_Count = utf8.To_Title_Special_Into(
				fixture.Storage, fixture.Special, fixture.Source,
			)
		}},
		{Name: "To_Valid_UTF8_Into", Run: func() {
			fixture.Byte_Count = utf8.To_Valid_UTF8_Into(
				fixture.Storage, fixture.Source, utf8.Bytes("?"),
			)
		}},
		{Name: "Title_Into", Run: func() {
			fixture.Byte_Count = utf8.Title_Into(fixture.Storage, fixture.Source)
		}},
	}
}

func text_trim_allocation_cases(
	fixture *text_allocation_fixture,
) (cases []text_allocation_case) {
	return []text_allocation_case{
		{Name: "Trim_Left_Function", Run: func() {
			fixture.Bytes = utf8.Trim_Left_Function(fixture.Source, text_is_space)
		}},
		{Name: "Trim_Right_Function", Run: func() {
			fixture.Bytes = utf8.Trim_Right_Function(fixture.Source, text_is_space)
		}},
		{Name: "Trim_Function", Run: func() {
			fixture.Bytes = utf8.Trim_Function(fixture.Source, text_is_space)
		}},
		{Name: "Trim", Run: func() {
			fixture.Bytes = utf8.Trim(fixture.Source, "a")
		}},
		{Name: "Trim_Left", Run: func() {
			fixture.Bytes = utf8.Trim_Left(fixture.Source, "a")
		}},
		{Name: "Trim_Right", Run: func() {
			fixture.Bytes = utf8.Trim_Right(fixture.Source, "a")
		}},
		{Name: "Trim_Space", Run: func() {
			fixture.Bytes = utf8.Trim_Space(fixture.Source)
		}},
		{Name: "Runes_Into", Run: func() {
			fixture.Count = utf8.Runes_Into(fixture.Characters, fixture.Source)
		}},
		{Name: "Equal_Fold", Run: func() {
			fixture.Boolean = utf8.Equal_Fold(fixture.Source, fixture.Source)
		}},
		{Name: "Fields_Sequence", Run: func() {
			fixture.Field_Count = utf8.Fields_Sequence(fixture.Source, text_accept)
		}},
		{Name: "Fields_Function_Sequence", Run: func() {
			fixture.Field_Count = utf8.Fields_Function_Sequence(
				fixture.Source, text_is_space, text_accept,
			)
		}},
	}
}
