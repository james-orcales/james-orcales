// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package utf16

import (
	"runtime"
	"testing"
	"unicode"

	"local/james-orcales/shared/slices"
)

func upstream_character_size(character rune) (size int) {
	return int(Character_Size(Character(character)))
}

func upstream_encode(characters []rune) (words []uint16) {
	source := make(Characters, len(characters))
	for index, character := range characters {
		source[index] = Character(character)
	}
	return []uint16(Encode(source))
}

func upstream_append_character(words []uint16, character rune) (result []uint16) {
	return []uint16(Append_Character(
		Words(words), Character(character),
	))
}

func upstream_encode_character(
	character rune,
) (first_result rune, second_result rune) {
	first, second := Encode_Character(
		Character(character),
	)
	return rune(first), rune(second)
}

func upstream_decode(words []uint16) (characters []rune) {
	decoded := Decode(Words(words))
	characters = make([]rune, len(decoded))
	for index, character := range decoded {
		characters[index] = rune(character)
	}
	return characters
}

func upstream_decode_character(first rune, second rune) (character rune) {
	return rune(Decode_Character(
		Character(first), Character(second),
	))
}

func upstream_is_surrogate(character rune) (yes bool) {
	return bool(Is_Surrogate(Character(character)))
}

// Test_Upstream_Constants checks the constants copied from package unicode.
func Test_Upstream_Constants(t *testing.T) {
	if RUNE_MAX != unicode.MaxRune {
		t.Errorf("utf16.maxRune is wrong: %x should be %x", RUNE_MAX, unicode.MaxRune)
	}
	if REPLACEMENT_CHARACTER != unicode.ReplacementChar {
		t.Errorf(
			"utf16.replacementChar is wrong: %x should be %x",
			REPLACEMENT_CHARACTER, unicode.ReplacementChar,
		)
	}
}

// Test_Upstream_Character_Size checks the word count of boundary characters.
func Test_Upstream_Character_Size(t *testing.T) {
	for _, tt := range []struct {
		R    rune
		Size int
	}{
		{0, 1},
		{rune(SURROGATE_FIRST) - 1, 1},
		{rune(SURROGATE_LIMIT), 1},
		{rune(SUPPLEMENTARY_FIRST) - 1, 1},
		{rune(SUPPLEMENTARY_FIRST), 2},
		{rune(RUNE_MAX), 2},
		{rune(RUNE_MAX) + 1, -1},
		{-1, -1},
	} {
		if size := upstream_character_size(tt.R); size != tt.Size {
			t.Errorf(
				"upstream_character_size(%#U) = %d, want %d",
				tt.R, size, tt.Size,
			)
		}
	}
}

type encode_fixture struct {
	Input  []rune
	Output []uint16
}

func encode_tests() (tests []encode_fixture) {
	return []encode_fixture{
		{[]rune{1, 2, 3, 4}, []uint16{1, 2, 3, 4}},
		{[]rune{0xffff, 0x10000, 0x10001, 0x12345, 0x10ffff},
			[]uint16{
				0xffff, 0xd800, 0xdc00, 0xd800, 0xdc01,
				0xd808, 0xdf45, 0xdbff, 0xdfff,
			}},
		{[]rune{'a', 'b', 0xd7ff, 0xd800, 0xdfff, 0xe000, 0x110000, -1},
			[]uint16{'a', 'b', 0xd7ff, 0xfffd, 0xfffd, 0xe000, 0xfffd, 0xfffd}},
	}
}

// Test_Upstream_Encode checks each upstream sequence fixture.
func Test_Upstream_Encode(t *testing.T) {
	for _, tt := range encode_tests() {
		output := upstream_encode(tt.Input)
		if !bool(slices.Equal(output, tt.Output)) {
			t.Errorf(
				"upstream_encode(%x) = %x; want %x",
				tt.Input, output, tt.Output,
			)
		}
	}
}

// Test_Upstream_Append_Character checks repeated single-character encoding.
func Test_Upstream_Append_Character(t *testing.T) {
	for _, tt := range encode_tests() {
		var output []uint16
		for _, character := range tt.Input {
			output = upstream_append_character(output, character)
		}
		if !bool(slices.Equal(output, tt.Output)) {
			t.Errorf(
				"upstream_append_character(%x) = %x; want %x",
				tt.Input, output, tt.Output,
			)
		}
	}
}

// Test_Upstream_Encode_Character checks every character in the sequence fixtures.
func Test_Upstream_Encode_Character(t *testing.T) {
	for fixture_index, tt := range encode_tests() {
		output_index := 0
		for _, character := range tt.Input {
			first, second := upstream_encode_character(character)
			invalid := character < 0x10000
			if !invalid {
				invalid = character > unicode.MaxRune
			}
			if invalid {
				if output_index >= len(tt.Output) {
					t.Errorf("#%d: ran out of tt.Output", fixture_index)
					break
				}
				incorrect := first != unicode.ReplacementChar
				if !incorrect {
					incorrect = second != unicode.ReplacementChar
				}
				if incorrect {
					t.Errorf(
						"upstream_encode_character(%#x) = %#x, %#x; "+
							"want 0xfffd, 0xfffd",
						character, first, second,
					)
				}
				output_index++
			} else {
				if output_index+1 >= len(tt.Output) {
					t.Errorf("#%d: ran out of tt.Output", fixture_index)
					break
				}
				incorrect := first != rune(tt.Output[output_index])
				if !incorrect {
					incorrect = second != rune(tt.Output[output_index+1])
				}
				if incorrect {
					t.Errorf(
						"upstream_encode_character(%#x) = %#x, %#x; "+
							"want %#x, %#x",
						character, first, second, tt.Output[output_index],
						tt.Output[output_index+1],
					)
				}
				output_index += 2
				decoded := upstream_decode_character(first, second)
				if decoded != character {
					t.Errorf(
						"upstream_decode_character(%#x, %#x) = %#x; "+
							"want %#x",
						first, second, decoded, character,
					)
				}
			}
		}
		if output_index != len(tt.Output) {
			t.Errorf(
				"#%d: upstream_encode_character did not generate "+
					"enough output",
				fixture_index,
			)
		}
	}
}

type decode_fixture struct {
	Input  []uint16
	Output []rune
}

func decode_tests() (tests []decode_fixture) {
	return []decode_fixture{
		{[]uint16{1, 2, 3, 4}, []rune{1, 2, 3, 4}},
		{[]uint16{0xffff, 0xd800, 0xdc00, 0xd800, 0xdc01, 0xd808, 0xdf45, 0xdbff, 0xdfff},
			[]rune{0xffff, 0x10000, 0x10001, 0x12345, 0x10ffff}},
		{[]uint16{0xd800, 'a'}, []rune{0xfffd, 'a'}},
		{[]uint16{0xdfff}, []rune{0xfffd}},
	}
}

// Test_Upstream_Decode_Allocations keeps short decode results on the caller stack.
func Test_Upstream_Decode_Allocations(t *testing.T) {
	for _, tt := range decode_tests() {
		allocs := testing.AllocsPerRun(10, func() {
			output := Decode(Words(tt.Input))
			if output == nil {
				t.Errorf("upstream_decode(%x) = nil", tt.Input)
			}
		})
		if allocs > 0 {
			t.Errorf("upstream_decode allocated %v times", allocs)
		}
	}
}

// Test_Upstream_Decode checks each upstream sequence fixture.
func Test_Upstream_Decode(t *testing.T) {
	for _, tt := range decode_tests() {
		output := upstream_decode(tt.Input)
		if !bool(slices.Equal(output, tt.Output)) {
			t.Errorf(
				"upstream_decode(%x) = %x; want %x",
				tt.Input, output, tt.Output,
			)
		}
	}
}

type decode_character_fixture struct {
	R1   rune
	R2   rune
	Want rune
}

func decode_character_tests() (tests []decode_character_fixture) {
	return []decode_character_fixture{
		{0xd800, 0xdc00, 0x10000},
		{0xd800, 0xdc01, 0x10001},
		{0xd808, 0xdf45, 0x12345},
		{0xdbff, 0xdfff, 0x10ffff},
		{0xd800, 'a', 0xfffd},
	}
}

// Test_Upstream_Decode_Character checks valid pairs and an invalid pair.
func Test_Upstream_Decode_Character(t *testing.T) {
	for test_index, tt := range decode_character_tests() {
		got := upstream_decode_character(tt.R1, tt.R2)
		if got != tt.Want {
			t.Errorf(
				"%d: upstream_decode_character(%q, %q) = %v; want %v",
				test_index, tt.R1, tt.R2, got, tt.Want,
			)
		}
	}
}

type surrogate_fixture struct {
	R    rune
	Want bool
}

func surrogate_tests() (tests []surrogate_fixture) {
	return []surrogate_fixture{
		// The published examples keep this fixture independent of the implementation.
		{'\u007A', false},     // LATIN SMALL LETTER Z.
		{'\u6C34', false},     // CJK UNIFIED IDEOGRAPH-6C34.
		{'\uFEFF', false},     // Byte order mark.
		{'\U00010000', false}, // First non-BMP code point.
		{'\U0001D11E', false}, // Musical symbol G clef.
		{'\U0010FFFD', false}, // Last Unicode code point.

		{rune(0xd7ff), false}, // Before the surrogate range.
		{rune(0xd800), true},  // First high surrogate.
		{rune(0xdc00), true},  // First low surrogate.
		{rune(0xe000), false}, // After the surrogate range.
		{rune(0xdfff), true},  // Last low surrogate.
	}
}

// Test_Upstream_Is_Surrogate checks the complete published boundary fixture.
func Test_Upstream_Is_Surrogate(t *testing.T) {
	for test_index, tt := range surrogate_tests() {
		got := upstream_is_surrogate(tt.R)
		if got != tt.Want {
			t.Errorf(
				"%d: upstream_is_surrogate(%q) = %v; want %v",
				test_index, tt.R, got, tt.Want,
			)
		}
	}
}

// Benchmark_Decode_ASCII measures one-word decoding.
func Benchmark_Decode_ASCII(b *testing.B) {
	data := Words{104, 101, 108, 108, 111, 32, 119, 111, 114, 108, 100}
	for run_index := 0; run_index < b.N; run_index++ {
		Decode(data)
	}
}

// Benchmark_Decode_Japanese measures non-ASCII one-word decoding.
func Benchmark_Decode_Japanese(b *testing.B) {
	data := Words{26085, 26412, 35486, 26085, 26412, 35486, 26085, 26412, 35486}
	for run_index := 0; run_index < b.N; run_index++ {
		Decode(data)
	}
}

// Benchmark_Decode_Character measures surrogate-pair decoding.
func Benchmark_Decode_Character(b *testing.B) {
	rs := make([]rune, 10)
	// Supplementary characters force pair decoding instead of the one-word path.
	for character_index, character := range []rune{'𝓐', '𝓑', '𝓒', '𝓓', '𝓔'} {
		rs[2*character_index], rs[2*character_index+1] =
			upstream_encode_character(character)
	}

	b.ResetTimer()
	var result Combined_Character
	for run_index := 0; run_index < b.N; run_index++ {
		for character_index := 0; character_index < 5; character_index++ {
			first := Character(rs[2*character_index])
			second := Character(rs[2*character_index+1])
			result = Decode_Character(
				first, second,
			)
		}
	}
	runtime.KeepAlive(result)
}

// Benchmark_Encode_ASCII measures one-word sequence encoding.
func Benchmark_Encode_ASCII(b *testing.B) {
	data := Characters{'h', 'e', 'l', 'l', 'o'}
	var result Words
	for run_index := 0; run_index < b.N; run_index++ {
		result = Encode(data)
	}
	runtime.KeepAlive(result)
}

// Benchmark_Encode_Japanese measures non-ASCII one-word sequence encoding.
func Benchmark_Encode_Japanese(b *testing.B) {
	data := Characters{'日', '本', '語'}
	var result Words
	for run_index := 0; run_index < b.N; run_index++ {
		result = Encode(data)
	}
	runtime.KeepAlive(result)
}

// Benchmark_Append_ASCII measures repeated one-word append operations.
func Benchmark_Append_ASCII(b *testing.B) {
	data := Characters{'h', 'e', 'l', 'l', 'o'}
	a := make(Words, 0, len(data)*2)
	for run_index := 0; run_index < b.N; run_index++ {
		for _, u := range data {
			a = Words(Append_Character(a, u))
		}
		a = a[:0]
	}
	runtime.KeepAlive(a)
}

// Benchmark_Append_Japanese measures non-ASCII append operations.
func Benchmark_Append_Japanese(b *testing.B) {
	data := Characters{'日', '本', '語'}
	a := make(Words, 0, len(data)*2)
	for run_index := 0; run_index < b.N; run_index++ {
		for _, u := range data {
			a = Words(Append_Character(a, u))
		}
		a = a[:0]
	}
	runtime.KeepAlive(a)
}

// Benchmark_Encode_Character measures surrogate-pair encoding.
func Benchmark_Encode_Character(b *testing.B) {
	var first First_Encoded_Character
	var second Second_Encoded_Character
	for run_index := 0; run_index < b.N; run_index++ {
		for _, u := range (Characters{'𝓐', '𝓑', '𝓒', '𝓓', '𝓔'}) {
			first, second = Encode_Character(u)
		}
	}
	runtime.KeepAlive(first)
	runtime.KeepAlive(second)
}
