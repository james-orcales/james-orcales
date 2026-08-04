// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package utf8

import (
	"fmt"
	"runtime"
	"testing"
	"unicode"

	"local/james-orcales/shared/testify"
)

func example_decode_final_character(output *example_buffer) {
	b := []byte("Hello, 世界")

	for len(b) > 0 {
		r, size := upstream_decode_final_character(b)
		fmt.Fprintf(output, "%c %v\n", r, size)

		b = b[:len(b)-size]
	}
}

func example_decode_final_character_text(output *example_buffer) {
	text := "Hello, 世界"

	for len(text) > 0 {
		r, size := upstream_decode_final_character_text(text)
		fmt.Fprintf(output, "%c %v\n", r, size)

		text = text[:len(text)-size]
	}
}

func example_decode_character(output *example_buffer) {
	b := []byte("Hello, 世界")

	for len(b) > 0 {
		r, size := upstream_decode_character(b)
		fmt.Fprintf(output, "%c %v\n", r, size)

		b = b[size:]
	}
}

func example_decode_character_text(output *example_buffer) {
	text := "Hello, 世界"

	for len(text) > 0 {
		r, size := upstream_decode_character_text(text)
		fmt.Fprintf(output, "%c %v\n", r, size)

		text = text[size:]
	}
}

func example_encode_character(output *example_buffer) {
	r := '世'
	buffer := make([]byte, 3)

	n_count := upstream_encode_character(buffer, r)

	fmt.Fprintln(output, buffer)
	fmt.Fprintln(output, n_count)
}

func example_encode_character_invalid(output *example_buffer) {
	runes := []rune{
		// Less than 0, out of range.
		-1,
		// Greater than 0x10FFFF, out of range.
		0x110000,
		// The Unicode replacement character.
		rune(REPLACEMENT_CHARACTER),
	}
	for item_index, c := range runes {
		buffer := make([]byte, 3)
		size := upstream_encode_character(buffer, c)
		fmt.Fprintf(output, "%d: %d %[2]s %d\n", item_index, buffer, size)
	}
}

func example_full_character(output *example_buffer) {
	buffer := []byte{228, 184, 150} // 世
	fmt.Fprintln(output, upstream_full_character(buffer))
	fmt.Fprintln(output, upstream_full_character(buffer[:2]))
}

func example_full_character_text(output *example_buffer) {
	text := "世"
	fmt.Fprintln(output, upstream_full_character_text(text))
	fmt.Fprintln(output, upstream_full_character_text(text[:2]))
}

func example_character_count(output *example_buffer) {
	buffer := []byte("Hello, 世界")
	fmt.Fprintln(output, "bytes =", len(buffer))
	fmt.Fprintln(output, "runes =", upstream_character_count(buffer))
}

func example_character_count_text(output *example_buffer) {
	text := "Hello, 世界"
	fmt.Fprintln(output, "bytes =", len(text))
	fmt.Fprintln(output, "runes =", upstream_character_count_text(text))
}

func example_character_size(output *example_buffer) {
	fmt.Fprintln(output, upstream_character_size('a'))
	fmt.Fprintln(output, upstream_character_size('界'))
}

func example_character_start(output *example_buffer) {
	buffer := []byte("a界")
	fmt.Fprintln(output, upstream_character_start(buffer[0]))
	fmt.Fprintln(output, upstream_character_start(buffer[1]))
	fmt.Fprintln(output, upstream_character_start(buffer[2]))
}

func example_valid(output *example_buffer) {
	valid := []byte("Hello, 世界")
	invalid := []byte{0xff, 0xfe, 0xfd}

	fmt.Fprintln(output, upstream_valid(valid))
	fmt.Fprintln(output, upstream_valid(invalid))
}

func example_valid_character(output *example_buffer) {
	valid := 'a'
	invalid := rune(0xfffffff)

	fmt.Fprintln(output, upstream_valid_character(valid))
	fmt.Fprintln(output, upstream_valid_character(invalid))
}

func example_valid_text(output *example_buffer) {
	valid := "Hello, 世界"
	invalid := string([]byte{0xff, 0xfe, 0xfd})

	fmt.Fprintln(output, upstream_valid_text(valid))
	fmt.Fprintln(output, upstream_valid_text(invalid))
}

func example_append_character(output *example_buffer) {
	buf1 := upstream_append_character(nil, 0x10000)
	buf2 := upstream_append_character([]byte("init"), 0x10000)
	fmt.Fprintln(output, string(buf1))
	fmt.Fprintln(output, string(buf2))
}

func upstream_full_character(source []byte) (yes bool) {
	return bool(Full_Character(Bytes(source)))
}

func upstream_full_character_text(source string) (yes bool) {
	return bool(Full_Character_Text(Text(source)))
}

func upstream_encode_character(buffer []byte, character rune) (size int) {
	return int(Encode_Character(
		Bytes(buffer), Character(character),
	))
}

func upstream_append_character(buffer []byte, character rune) (output []byte) {
	return []byte(Append_Character(
		Bytes(buffer), Character(character),
	))
}

func upstream_decode_character(source []byte) (character rune, size int) {
	decoded_character, decoded_size := Decode_Character(Bytes(source))
	return rune(decoded_character), int(decoded_size)
}

func upstream_decode_character_text(source string) (character rune, size int) {
	decoded_character, decoded_size := Decode_Character_Text(Text(source))
	return rune(decoded_character), int(decoded_size)
}

func upstream_decode_final_character(source []byte) (character rune, size int) {
	decoded_character, decoded_size := Decode_Final_Character(Bytes(source))
	return rune(decoded_character), int(decoded_size)
}

func upstream_decode_final_character_text(source string) (character rune, size int) {
	decoded_character, decoded_size := Decode_Final_Character_Text(Text(source))
	return rune(decoded_character), int(decoded_size)
}

func upstream_character_count(source []byte) (count int) {
	return int(Character_Count(Bytes(source)))
}

func upstream_character_count_text(source string) (count int) {
	return int(Character_Count_Text(Text(source)))
}

func upstream_character_size(character rune) (size int) {
	return int(Character_Size(Character(character)))
}

func upstream_character_start(value byte) (yes bool) {
	return bool(Character_Start(Byte(value)))
}

func upstream_valid(source []byte) (yes bool) {
	return bool(Valid(Bytes(source)))
}

func upstream_valid_text(source string) (yes bool) {
	return bool(Valid_Text(Text(source)))
}

func upstream_valid_character(character rune) (yes bool) {
	return bool(Valid_Character(Character(character)))
}

func check_decoded_character(
	t *testing.T,
	operation string,
	input string,
	character rune,
	size int,
	wanted_character rune,
	wanted_size int,
) {
	t.Helper()
	if character != wanted_character {
		t.Errorf("%s(%q) character = %#04x, want %#04x", operation, input,
			character, wanted_character)
	}
	if size != wanted_size {
		t.Errorf("%s(%q) size = %d, want %d", operation, input, size, wanted_size)
	}
}

// Test_Upstream_Constants preserves the upstream behavior coverage.
func Test_Upstream_Constants(t *testing.T) {
	if rune(RUNE_MAX) != unicode.MaxRune {
		t.Errorf("utf8.RUNE_MAX is wrong: %x should be %x",
			RUNE_MAX, unicode.MaxRune)
	}
	if rune(REPLACEMENT_CHARACTER) != unicode.ReplacementChar {
		t.Errorf("utf8.rune(REPLACEMENT_CHARACTER) is wrong: %x should be %x",
			rune(REPLACEMENT_CHARACTER), unicode.ReplacementChar)
	}
}

type utf8_map_entry struct {
	R    rune
	Text string
}

func utf8_map() (entries []utf8_map_entry) {
	return []utf8_map_entry{
		{0x0000, "\x00"},
		{0x0001, "\x01"},
		{0x007e, "\x7e"},
		{0x007f, "\x7f"},
		{0x0080, "\xc2\x80"},
		{0x0081, "\xc2\x81"},
		{0x00bf, "\xc2\xbf"},
		{0x00c0, "\xc3\x80"},
		{0x00c1, "\xc3\x81"},
		{0x00c8, "\xc3\x88"},
		{0x00d0, "\xc3\x90"},
		{0x00e0, "\xc3\xa0"},
		{0x00f0, "\xc3\xb0"},
		{0x00f8, "\xc3\xb8"},
		{0x00ff, "\xc3\xbf"},
		{0x0100, "\xc4\x80"},
		{0x07ff, "\xdf\xbf"},
		{0x0400, "\xd0\x80"},
		{0x0800, "\xe0\xa0\x80"},
		{0x0801, "\xe0\xa0\x81"},
		{0x1000, "\xe1\x80\x80"},
		{0xd000, "\xed\x80\x80"},
		{0xd7ff, "\xed\x9f\xbf"},
		{0xe000, "\xee\x80\x80"},
		{0xfffe, "\xef\xbf\xbe"},
		{0xffff, "\xef\xbf\xbf"},
		{0x10000, "\xf0\x90\x80\x80"},
		{0x10001, "\xf0\x90\x80\x81"},
		{0x40000, "\xf1\x80\x80\x80"},
		{0x10fffe, "\xf4\x8f\xbf\xbe"},
		{0x10ffff, "\xf4\x8f\xbf\xbf"},
		{0xFFFD, "\xef\xbf\xbd"},
	}
}

func surrogate_map() (entries []utf8_map_entry) {
	return []utf8_map_entry{
		{0xd800, "\xed\xa0\x80"},
		{0xdfff, "\xed\xbf\xbf"},
	}
}

func test_strings() (texts []string) {
	return []string{
		"",
		"abcd",
		"☺☻☹",
		"日a本b語ç日ð本Ê語þ日¥本¼語i日©",
		"日a本b語ç日ð本Ê語þ日¥本¼語i日©" +
			"日a本b語ç日ð本Ê語þ日¥本¼語i日©" +
			"日a本b語ç日ð本Ê語þ日¥本¼語i日©",
		"\x80\x80\x80\x80",
	}
}

// Test_Upstream_Full_Character preserves the upstream behavior coverage.
func Test_Upstream_Full_Character(t *testing.T) {
	for _, m := range utf8_map() {
		b := []byte(m.Text)
		if !upstream_full_character(b) {
			t.Errorf("upstream_full_character(%q) (%U) = false, want true", b, m.R)
		}
		s := m.Text
		if !upstream_full_character_text(s) {
			t.Errorf("upstream_full_character_text(%q) (%U) = false, want true", s, m.R)
		}
		b1 := b[0 : len(b)-1]
		if upstream_full_character(b1) {
			t.Errorf("upstream_full_character(%q) = true, want false", b1)
		}
		s1 := string(b1)
		if upstream_full_character_text(s1) {
			t.Errorf("upstream_full_character(%q) = true, want false", s1)
		}
	}
	for _, s := range []string{"\xc0", "\xc1"} {
		b := []byte(s)
		if !upstream_full_character(b) {
			t.Errorf("upstream_full_character(%q) = false, want true", s)
		}
		if !upstream_full_character_text(s) {
			t.Errorf("upstream_full_character_text(%q) = false, want true", s)
		}
	}
}

// Test_Upstream_Encode_Character preserves the upstream behavior coverage.
func Test_Upstream_Encode_Character(t *testing.T) {
	const ENCODE_BUFFER_SIZE = 10

	for _, m := range utf8_map() {
		b := []byte(m.Text)
		var buffer [ENCODE_BUFFER_SIZE]byte
		n_count := upstream_encode_character(buffer[0:], m.R)
		b1 := buffer[0:n_count]
		if string(b) != string(b1) {
			t.Errorf("upstream_encode_character(%#04x) = %q want %q", m.R, b1, b)
		}
	}
}

// Test_Upstream_Append_Character preserves the upstream behavior coverage.
func Test_Upstream_Append_Character(t *testing.T) {
	for _, m := range utf8_map() {
		if buffer := upstream_append_character(nil, m.R); string(buffer) != m.Text {
			t.Errorf("upstream_append_character(nil, %#04x) = %s, want %s",
				m.R, buffer, m.Text)
		}
		buffer := upstream_append_character([]byte("init"), m.R)
		if string(buffer) != "init"+m.Text {
			t.Errorf("upstream_append_character(init, %#04x) = %s, want %s",
				m.R, buffer, "init"+m.Text)
		}
	}
}

// Test_Upstream_Decode_Character preserves the upstream behavior coverage.
func Test_Upstream_Decode_Character(t *testing.T) {
	for _, m := range utf8_map() {
		b := []byte(m.Text)
		r, size := upstream_decode_character(b)
		check_decoded_character(t, "upstream_decode_character", string(b),
			r, size, m.R, len(b))
		s := m.Text
		r, size = upstream_decode_character_text(s)
		check_decoded_character(t, "upstream_decode_character_text", s,
			r, size, m.R, len(b))

		r, size = upstream_decode_character(b[0:cap(b)])
		check_decoded_character(t, "upstream_decode_character", string(b),
			r, size, m.R, len(b))
		s = m.Text + "\x00"
		r, size = upstream_decode_character_text(s)
		check_decoded_character(t, "upstream_decode_character_text", s,
			r, size, m.R, len(b))

		wantsize := 1
		if wantsize >= len(b) {
			wantsize = 0
		}
		r, size = upstream_decode_character(b[0 : len(b)-1])
		check_decoded_character(t, "upstream_decode_character",
			string(b[:len(b)-1]), r, size, rune(REPLACEMENT_CHARACTER), wantsize)
		s = m.Text[0 : len(m.Text)-1]
		r, size = upstream_decode_character_text(s)
		check_decoded_character(t, "upstream_decode_character_text", s,
			r, size, rune(REPLACEMENT_CHARACTER), wantsize)

		if len(b) == 1 {
			b[0] = 0x80
		} else {
			b[len(b)-1] = 0x7F
		}
		r, size = upstream_decode_character(b)
		check_decoded_character(t, "upstream_decode_character", string(b),
			r, size, rune(REPLACEMENT_CHARACTER), 1)
		s = string(b)
		r, size = upstream_decode_character_text(s)
		check_decoded_character(t, "upstream_decode_character_text", s,
			r, size, rune(REPLACEMENT_CHARACTER), 1)
	}
}

// Test_Upstream_Decode_Surrogate preserves the upstream behavior coverage.
func Test_Upstream_Decode_Surrogate(t *testing.T) {
	for _, m := range surrogate_map() {
		b := []byte(m.Text)
		r, size := upstream_decode_character(b)
		check_decoded_character(t, "upstream_decode_character", string(b),
			r, size, rune(REPLACEMENT_CHARACTER), 1)
		s := m.Text
		r, size = upstream_decode_character_text(s)
		check_decoded_character(t, "upstream_decode_character_text", s,
			r, size, rune(REPLACEMENT_CHARACTER), 1)
	}
}

// The range loop must use the same decoding rules as the explicit operations.
// Test_Upstream_Sequence preserves the upstream behavior coverage.
func Test_Upstream_Sequence(t *testing.T) {
	for _, test_suite := range test_strings() {
		for _, m := range utf8_map() {
			for _, s := range []string{
				test_suite + m.Text,
				m.Text + test_suite,
				test_suite + m.Text + test_suite,
			} {
				test_sequence(t, s)
			}
		}
	}
}

func runtime_character_count(s string) (count int) {
	return len([]rune(s)) // Replaced by gc with call to runtime.countrunes(s).
}

// The runtime conversions must preserve the assumption that these tests use.
// Test_Upstream_Runtime_Conversion preserves the upstream behavior coverage.
func Test_Upstream_Runtime_Conversion(t *testing.T) {
	for _, test_suite := range test_strings() {
		count := upstream_character_count_text(test_suite)
		if n_count := runtime_character_count(test_suite); n_count != count {
			t.Errorf(
				"%q: len([]rune()) counted %d runes; got %d from character count",
				test_suite, n_count, count,
			)
			break
		}

		runes := []rune(test_suite)
		if n_count := len(runes); n_count != count {
			t.Errorf("%q: []rune() has length %d; got %d from character count",
				test_suite, n_count, count)
			break
		}
		item_index := 0
		for _, r := range test_suite {
			if r != runes[item_index] {
				t.Errorf("%q[%d]: expected %c (%U); got %c (%U)",
					test_suite, item_index,
					runes[item_index], runes[item_index], r, r)
			}
			item_index++
		}
	}
}

func invalid_sequence_tests() (tests []string) {
	return []string{
		"\xed\xa0\x80\x80", // Surrogate minimum.
		"\xed\xbf\xbf\x80", // Surrogate maximum.

		// Invalid first byte.
		"\x91\x80\x80\x80",

		// Two-byte sequence.
		"\xC2\x7F\x80\x80",
		"\xC2\xC0\x80\x80",
		"\xDF\x7F\x80\x80",
		"\xDF\xC0\x80\x80",

		// Three-byte sequence after E0.
		"\xE0\x9F\xBF\x80",
		"\xE0\xA0\x7F\x80",
		"\xE0\xBF\xC0\x80",
		"\xE0\xC0\x80\x80",

		// General three-byte sequence.
		"\xE1\x7F\xBF\x80",
		"\xE1\x80\x7F\x80",
		"\xE1\xBF\xC0\x80",
		"\xE1\xC0\x80\x80",

		// Three-byte sequence before a surrogate.
		"\xED\x7F\xBF\x80",
		"\xED\x80\x7F\x80",
		"\xED\x9F\xC0\x80",
		"\xED\xA0\x80\x80",

		// Four-byte sequence after F0.
		"\xF0\x8F\xBF\xBF",
		"\xF0\x90\x7F\xBF",
		"\xF0\x90\x80\x7F",
		"\xF0\xBF\xBF\xC0",
		"\xF0\xBF\xC0\x80",
		"\xF0\xC0\x80\x80",

		// General four-byte sequence.
		"\xF1\x7F\xBF\xBF",
		"\xF1\x80\x7F\xBF",
		"\xF1\x80\x80\x7F",
		"\xF1\xBF\xBF\xC0",
		"\xF1\xBF\xC0\x80",
		"\xF1\xC0\x80\x80",

		// Four-byte sequence before the maximum.
		"\xF4\x7F\xBF\xBF",
		"\xF4\x80\x7F\xBF",
		"\xF4\x80\x80\x7F",
		"\xF4\x8F\xBF\xC0",
		"\xF4\x8F\xC0\x80",
		"\xF4\x90\x80\x80",
	}
}

func runtime_decode_character(s string) (character rune) {
	for _, r := range s {
		return r
	}
	return -1
}

// Test_Upstream_Invalid_Sequence preserves the upstream behavior coverage.
func Test_Upstream_Invalid_Sequence(t *testing.T) {
	for _, s := range invalid_sequence_tests() {
		r1, size1 := upstream_decode_character([]byte(s))
		if size1 != 1 {
			t.Errorf("upstream_decode_character(%#x) size = %d, want 1", s, size1)
		}
		if want := rune(REPLACEMENT_CHARACTER); r1 != want {
			t.Errorf("upstream_decode_character(%#x) = %#04x, want %#04x", s, r1, want)
			return
		}
		r2, size2 := upstream_decode_character_text(s)
		if size2 != 1 {
			t.Errorf("upstream_decode_character_text(%q) size = %d, want 1", s, size2)
		}
		if want := rune(REPLACEMENT_CHARACTER); r2 != want {
			t.Errorf("upstream_decode_character_text(%q) = %#04x, want %#04x",
				s, r2, want)
			return
		}
		if r1 != r2 {
			t.Errorf("byte decode of %#x = %#04x; text decode of %q = %#04x",
				s, r1, s, r2)
			return
		}
		r3 := runtime_decode_character(s)
		if r2 != r3 {
			t.Errorf("text decode of %q = %#04x; runtime decode of %q = %#04x",
				s, r2, s, r3)
			return
		}
	}
}

func test_sequence(t *testing.T, s string) {
	type info struct {
		Index int
		R     rune
	}
	index := make([]info, len(s))
	b := []byte(s)
	si := 0
	j := 0
	for item_index, r := range s {
		if si != item_index {
			t.Errorf("Sequence(%q) mismatched index %d, want %d", s, si, item_index)
			return
		}
		index[j] = info{item_index, r}
		j++
		r1, size1 := upstream_decode_character(b[item_index:])
		if r != r1 {
			t.Errorf("upstream_decode_character(%q) = %#04x, want %#04x",
				s[item_index:], r1, r)
			return
		}
		r2, size2 := upstream_decode_character_text(s[item_index:])
		if r != r2 {
			t.Errorf("upstream_decode_character_text(%q) = %#04x, want %#04x",
				s[item_index:], r2, r)
			return
		}
		if size1 != size2 {
			t.Errorf("byte and text decode sizes for %q differ: %d and %d",
				s[item_index:], size1, size2)
			return
		}
		si += size1
	}
	j--
	for si = len(s); si > 0; {
		r1, size1 := upstream_decode_final_character(b[0:si])
		r2, size2 := upstream_decode_final_character_text(s[0:si])
		if size1 != size2 {
			t.Errorf("final byte and text decode sizes for %q at %d differ: %d and %d",
				s, si, size1, size2)
			return
		}
		if r1 != index[j].R {
			t.Errorf("upstream_decode_final_character(%q, %d) = %#04x, want %#04x",
				s, si, r1, index[j].R)
			return
		}
		if r2 != index[j].R {
			t.Errorf("upstream_decode_final_character_text(%q, %d) = %#04x, want %#04x",
				s, si, r2, index[j].R)
			return
		}
		si -= size1
		if si != index[j].Index {
			t.Errorf("upstream_decode_final_character(%q) index = %d, want %d",
				s, si, index[j].Index)
			return
		}
		j--
	}
	if si != 0 {
		t.Errorf("upstream_decode_final_character(%q) finished at %d, not 0", s, si)
	}
}

// Check that negative runes encode as U+FFFD.
// Test_Upstream_Negative_Character preserves the upstream behavior coverage.
func Test_Upstream_Negative_Character(t *testing.T) {
	errorbuf := make([]byte, UTF_MAXIMUM)
	errorbuf = errorbuf[0:upstream_encode_character(errorbuf, rune(REPLACEMENT_CHARACTER))]
	buffer := make([]byte, UTF_MAXIMUM)
	buffer = buffer[0:upstream_encode_character(buffer, -1)]
	if string(buffer) != string(errorbuf) {
		t.Errorf("incorrect encoding [% x] for -1; expected [% x]", buffer, errorbuf)
	}
}

type character_count_fixture struct {
	Input  string
	Output int
}

func character_count_tests() (tests []character_count_fixture) {
	return []character_count_fixture{
		{"abcd", 4},
		{"☺☻☹", 3},
		{"1,2,3,4", 7},
		{"\xe2\x00", 2},
		{"\xe2\x80", 2},
		{"a\xe2\x80", 3},
	}
}

// Test_Upstream_Character_Count preserves the upstream behavior coverage.
func Test_Upstream_Character_Count(t *testing.T) {
	for _, tt := range character_count_tests() {
		if output := upstream_character_count_text(tt.Input); output != tt.Output {
			t.Errorf("upstream_character_count_text(%q) = %d, want %d",
				tt.Input, output, tt.Output)
		}
		if output := upstream_character_count([]byte(tt.Input)); output != tt.Output {
			t.Errorf("upstream_character_count(%q) = %d, want %d",
				tt.Input, output, tt.Output)
		}
	}
}

// Test_Upstream_Character_Count_Allocation preserves the upstream behavior coverage.
func Test_Upstream_Character_Count_Allocation(t *testing.T) {
	count := 0
	if n_count := testing.AllocsPerRun(10, func() {
		s := []byte("日本語日本語日本語日")
		count = upstream_character_count(s)
	}); n_count > 0 {
		t.Errorf("unexpected upstream_character_count allocation, got %v, want 0", n_count)
	}
	runtime.KeepAlive(count)
}

type character_size_fixture struct {
	R    rune
	Size int
}

func character_size_tests() (tests []character_size_fixture) {
	return []character_size_fixture{
		{0, 1},
		{'e', 1},
		{'é', 2},
		{'☺', 3},
		{rune(REPLACEMENT_CHARACTER), 3},
		{rune(RUNE_MAX), 4},
		{0xD800, -1},
		{0xDFFF, -1},
		{rune(RUNE_MAX) + 1, -1},
		{-1, -1},
	}
}

// Test_Upstream_Character_Size preserves the upstream behavior coverage.
func Test_Upstream_Character_Size(t *testing.T) {
	for _, tt := range character_size_tests() {
		if size := upstream_character_size(tt.R); size != tt.Size {
			t.Errorf("upstream_character_size(%#U) = %d, want %d", tt.R, size, tt.Size)
		}
	}
}

type valid_fixture struct {
	Input  string
	Output bool
}

func valid_tests() (tests []valid_fixture) {
	tests = []valid_fixture{
		{"", true},
		{"a", true},
		{"abc", true},
		{"Ж", true},
		{"ЖЖ", true},
		{"брэд-ЛГТМ", true},
		{"☺☻☹", true},
		{"aa\xe2", false},
		{string([]byte{66, 250}), false},
		{string([]byte{66, 250, 67}), false},
		{"a\uFFFDb", true},
		{string("\xF4\x8F\xBF\xBF"), true},      // U+10FFFF.
		{string("\xF4\x90\x80\x80"), false},     // Above U+10FFFF.
		{string("\xF7\xBF\xBF\xBF"), false},     // Above U+10FFFF.
		{string("\xFB\xBF\xBF\xBF\xBF"), false}, // Above U+10FFFF.
		{string("\xc0\x80"), false},             // Non-shortest U+0000.
		{string("\xed\xa0\x80"), false},         // High surrogate.
		{string("\xed\xbf\xbf"), false},         // Low surrogate.
	}
	for item_index := range 100 {
		tests = append(tests, valid_fixture{
			Input: repeat("a", item_index), Output: true,
		})
		tests = append(tests, valid_fixture{
			Input: repeat("a", item_index) + "Ж", Output: true,
		})
		tests = append(tests, valid_fixture{
			Input: repeat("a", item_index) + "\xe2", Output: false,
		})
		tests = append(tests, valid_fixture{
			Input:  repeat("a", item_index) + "Ж" + repeat("b", item_index),
			Output: true,
		})
		tests = append(tests, valid_fixture{
			Input:  repeat("a", item_index) + "\xe2" + repeat("b", item_index),
			Output: false,
		})
	}
	return tests
}

// Test_Upstream_Valid preserves the upstream behavior coverage.
func Test_Upstream_Valid(t *testing.T) {
	for _, tt := range valid_tests() {
		if upstream_valid([]byte(tt.Input)) != tt.Output {
			t.Errorf("upstream_valid(%q) = %v; want %v",
				tt.Input, !tt.Output, tt.Output)
		}
		if upstream_valid_text(tt.Input) != tt.Output {
			t.Errorf("upstream_valid_text(%q) = %v; want %v",
				tt.Input, !tt.Output, tt.Output)
		}
	}
}

type valid_character_fixture struct {
	R  rune
	Ok bool
}

func valid_character_tests() (tests []valid_character_fixture) {
	return []valid_character_fixture{
		{0, true},
		{'e', true},
		{'é', true},
		{'☺', true},
		{rune(REPLACEMENT_CHARACTER), true},
		{rune(RUNE_MAX), true},
		{0xD7FF, true},
		{0xD800, false},
		{0xDFFF, false},
		{0xE000, true},
		{rune(RUNE_MAX) + 1, false},
		{-1, false},
	}
}

// Test_Upstream_Valid_Character preserves the upstream behavior coverage.
func Test_Upstream_Valid_Character(t *testing.T) {
	for _, tt := range valid_character_tests() {
		if ok := upstream_valid_character(tt.R); ok != tt.Ok {
			t.Errorf("upstream_valid_character(%#U) = %t, want %t", tt.R, ok, tt.Ok)
		}
	}
}

// Benchmark_Character_Count_ASCII preserves the upstream performance coverage.
func Benchmark_Character_Count_ASCII(b *testing.B) {
	s := []byte("0123456789")
	for item_index := 0; item_index < b.N; item_index++ {
		upstream_character_count(s)
	}
}

// Benchmark_Character_Count_Japanese preserves the upstream performance coverage.
func Benchmark_Character_Count_Japanese(b *testing.B) {
	s := []byte("日本語日本語日本語日")
	for item_index := 0; item_index < b.N; item_index++ {
		upstream_character_count(s)
	}
}

// Benchmark_Character_Count_Text_ASCII preserves the upstream performance coverage.
func Benchmark_Character_Count_Text_ASCII(b *testing.B) {
	for item_index := 0; item_index < b.N; item_index++ {
		upstream_character_count_text("0123456789")
	}
}

// Benchmark_Character_Count_Text_Japanese preserves the upstream performance coverage.
func Benchmark_Character_Count_Text_Japanese(b *testing.B) {
	for item_index := 0; item_index < b.N; item_index++ {
		upstream_character_count_text("日本語日本語日本語日")
	}
}

func maximum_ascii() (text string) {
	return repeat("a", SEQUENCE_SIZE_MAXIMUM)
}

// Benchmark_Valid_ASCII preserves the upstream performance coverage.
func Benchmark_Valid_ASCII(b *testing.B) {
	s := []byte("0123456789")
	for item_index := 0; item_index < b.N; item_index++ {
		upstream_valid(s)
	}
}

// Benchmark_Valid_Maximum_ASCII preserves the upstream performance coverage.
func Benchmark_Valid_Maximum_ASCII(b *testing.B) {
	s := []byte(maximum_ascii())
	for item_index := 0; item_index < b.N; item_index++ {
		upstream_valid(s)
	}
}

// Benchmark_Valid_Japanese preserves the upstream performance coverage.
func Benchmark_Valid_Japanese(b *testing.B) {
	s := []byte("日本語日本語日本語日")
	for item_index := 0; item_index < b.N; item_index++ {
		upstream_valid(s)
	}
}

// Benchmark_Valid_Mostly_ASCII preserves the upstream performance coverage.
func Benchmark_Valid_Mostly_ASCII(b *testing.B) {
	mostly_ascii, _ := long_strings()
	long_mostly_ascii := []byte(mostly_ascii)
	for item_index := 0; item_index < b.N; item_index++ {
		upstream_valid(long_mostly_ascii)
	}
}

// Benchmark_Valid_Long_Japanese preserves the upstream performance coverage.
func Benchmark_Valid_Long_Japanese(b *testing.B) {
	_, japanese := long_strings()
	long_japanese := []byte(japanese)
	for item_index := 0; item_index < b.N; item_index++ {
		upstream_valid(long_japanese)
	}
}

// Benchmark_Valid_Text_ASCII preserves the upstream performance coverage.
func Benchmark_Valid_Text_ASCII(b *testing.B) {
	for item_index := 0; item_index < b.N; item_index++ {
		upstream_valid_text("0123456789")
	}
}

// Benchmark_Valid_Text_Maximum_ASCII preserves the upstream performance coverage.
func Benchmark_Valid_Text_Maximum_ASCII(b *testing.B) {
	maximum := maximum_ascii()
	for item_index := 0; item_index < b.N; item_index++ {
		upstream_valid_text(maximum)
	}
}

// Benchmark_Valid_Text_Japanese preserves the upstream performance coverage.
func Benchmark_Valid_Text_Japanese(b *testing.B) {
	for item_index := 0; item_index < b.N; item_index++ {
		upstream_valid_text("日本語日本語日本語日")
	}
}

// Benchmark_Valid_Text_Mostly_ASCII preserves the upstream performance coverage.
func Benchmark_Valid_Text_Mostly_ASCII(b *testing.B) {
	mostly_ascii, _ := long_strings()
	for item_index := 0; item_index < b.N; item_index++ {
		upstream_valid_text(mostly_ascii)
	}
}

// Benchmark_Valid_Text_Long_Japanese preserves the upstream performance coverage.
func Benchmark_Valid_Text_Long_Japanese(b *testing.B) {
	_, japanese := long_strings()
	for item_index := 0; item_index < b.N; item_index++ {
		upstream_valid_text(japanese)
	}
}

func long_strings() (mostly_ascii string, japanese string) {
	const JAPANESE = "日本語日本語日本語日"
	buffer := make([]byte, 0, SEQUENCE_SIZE_MAXIMUM+len(JAPANESE))
	for item_index := 0; len(buffer) < SEQUENCE_SIZE_MAXIMUM; item_index++ {
		if item_index%100 == 0 {
			buffer = append(buffer, JAPANESE...)
		} else {
			buffer = append(buffer, "0123456789"...)
		}
	}
	mostly_ascii = string(buffer[:SEQUENCE_SIZE_MAXIMUM])
	japanese = repeat(
		JAPANESE, SEQUENCE_SIZE_MAXIMUM/len(JAPANESE),
	)
	return mostly_ascii, japanese
}

func repeat(text string, count int) (repeated string) {
	buffer := make([]byte, len(text)*count)
	position := 0
	for range count {
		position += copy(buffer[position:], text)
	}
	return string(buffer)
}

// Benchmark_Encode_ASCII preserves the upstream performance coverage.
func Benchmark_Encode_ASCII(b *testing.B) {
	buffer := make([]byte, UTF_MAXIMUM)
	for item_index := 0; item_index < b.N; item_index++ {
		upstream_encode_character(buffer, 'a') // 1 byte
	}
}

// Benchmark_Encode_Spanish preserves the upstream performance coverage.
func Benchmark_Encode_Spanish(b *testing.B) {
	buffer := make([]byte, UTF_MAXIMUM)
	for item_index := 0; item_index < b.N; item_index++ {
		upstream_encode_character(buffer, 'Ñ') // 2 bytes
	}
}

// Benchmark_Encode_Japanese preserves the upstream performance coverage.
func Benchmark_Encode_Japanese(b *testing.B) {
	buffer := make([]byte, UTF_MAXIMUM)
	for item_index := 0; item_index < b.N; item_index++ {
		upstream_encode_character(buffer, '本') // 3 bytes
	}
}

// Benchmark_Encode_Maximum preserves the upstream performance coverage.
func Benchmark_Encode_Maximum(b *testing.B) {
	buffer := make([]byte, UTF_MAXIMUM)
	for item_index := 0; item_index < b.N; item_index++ {
		upstream_encode_character(buffer, rune(RUNE_MAX)) // 4 bytes
	}
}

// Benchmark_Encode_Above_Maximum preserves the upstream performance coverage.
func Benchmark_Encode_Above_Maximum(b *testing.B) {
	buffer := make([]byte, UTF_MAXIMUM)
	for item_index := 0; item_index < b.N; item_index++ {
		upstream_encode_character(buffer, rune(RUNE_MAX)+1)
	}
}

// Benchmark_Encode_Surrogate preserves the upstream performance coverage.
func Benchmark_Encode_Surrogate(b *testing.B) {
	buffer := make([]byte, UTF_MAXIMUM)
	for item_index := 0; item_index < b.N; item_index++ {
		upstream_encode_character(buffer, 0xD800) // 3 bytes: rune(REPLACEMENT_CHARACTER)
	}
}

// Benchmark_Encode_Negative preserves the upstream performance coverage.
func Benchmark_Encode_Negative(b *testing.B) {
	buffer := make([]byte, UTF_MAXIMUM)
	for item_index := 0; item_index < b.N; item_index++ {
		upstream_encode_character(buffer, -1) // 3 bytes: rune(REPLACEMENT_CHARACTER)
	}
}

// Benchmark_Append_ASCII preserves the upstream performance coverage.
func Benchmark_Append_ASCII(b *testing.B) {
	buffer := make([]byte, UTF_MAXIMUM)
	for item_index := 0; item_index < b.N; item_index++ {
		upstream_append_character(buffer[:0], 'a') // 1 byte
	}
}

// Benchmark_Append_Spanish preserves the upstream performance coverage.
func Benchmark_Append_Spanish(b *testing.B) {
	buffer := make([]byte, UTF_MAXIMUM)
	for item_index := 0; item_index < b.N; item_index++ {
		upstream_append_character(buffer[:0], 'Ñ') // 2 bytes
	}
}

// Benchmark_Append_Japanese preserves the upstream performance coverage.
func Benchmark_Append_Japanese(b *testing.B) {
	buffer := make([]byte, UTF_MAXIMUM)
	for item_index := 0; item_index < b.N; item_index++ {
		upstream_append_character(buffer[:0], '本') // 3 bytes
	}
}

// Benchmark_Append_Maximum preserves the upstream performance coverage.
func Benchmark_Append_Maximum(b *testing.B) {
	buffer := make([]byte, UTF_MAXIMUM)
	for item_index := 0; item_index < b.N; item_index++ {
		upstream_append_character(buffer[:0], rune(RUNE_MAX)) // 4 bytes
	}
}

// Benchmark_Append_Above_Maximum preserves the upstream performance coverage.
func Benchmark_Append_Above_Maximum(b *testing.B) {
	buffer := make([]byte, UTF_MAXIMUM)
	for item_index := 0; item_index < b.N; item_index++ {
		upstream_append_character(buffer[:0], rune(RUNE_MAX)+1)
	}
}

// Benchmark_Append_Surrogate preserves the upstream performance coverage.
func Benchmark_Append_Surrogate(b *testing.B) {
	buffer := make([]byte, UTF_MAXIMUM)
	for item_index := 0; item_index < b.N; item_index++ {
		upstream_append_character(buffer[:0], 0xD800)
	}
}

// Benchmark_Append_Negative preserves the upstream performance coverage.
func Benchmark_Append_Negative(b *testing.B) {
	buffer := make([]byte, UTF_MAXIMUM)
	for item_index := 0; item_index < b.N; item_index++ {
		upstream_append_character(buffer[:0], -1) // 3 bytes: rune(REPLACEMENT_CHARACTER)
	}
}

// Benchmark_Decode_ASCII preserves the upstream performance coverage.
func Benchmark_Decode_ASCII(b *testing.B) {
	a := []byte{'a'}
	var character rune
	var size int
	for range b.N {
		character, size = upstream_decode_character(a)
	}
	runtime.KeepAlive(character)
	runtime.KeepAlive(size)
}

// Benchmark_Decode_Japanese preserves the upstream performance coverage.
func Benchmark_Decode_Japanese(b *testing.B) {
	nihon := []byte("本")
	var character rune
	var size int
	for range b.N {
		character, size = upstream_decode_character(nihon)
	}
	runtime.KeepAlive(character)
	runtime.KeepAlive(size)
}

// Benchmark_Decode_Text_ASCII preserves the upstream performance coverage.
func Benchmark_Decode_Text_ASCII(b *testing.B) {
	a := "a"
	var character rune
	var size int
	for range b.N {
		character, size = upstream_decode_character_text(a)
	}
	runtime.KeepAlive(character)
	runtime.KeepAlive(size)
}

// Benchmark_Decode_Text_Japanese preserves the upstream performance coverage.
func Benchmark_Decode_Text_Japanese(b *testing.B) {
	nihon := "本"
	var character rune
	var size int
	for range b.N {
		character, size = upstream_decode_character_text(nihon)
	}
	runtime.KeepAlive(character)
	runtime.KeepAlive(size)
}

// Benchmark_Full_Character preserves the upstream performance coverage.
func Benchmark_Full_Character(b *testing.B) {
	var yes bool
	benchmarks := []struct {
		Name string
		Data []byte
	}{
		{"ASCII", []byte("a")},
		{"Incomplete", []byte("\xf0\x90\x80")},
		{"Japanese", []byte("本")},
	}
	for _, bm := range benchmarks {
		b.Run(bm.Name, func(b *testing.B) {
			for item_index := 0; item_index < b.N; item_index++ {
				yes = upstream_full_character(bm.Data)
			}
		})
	}
	runtime.KeepAlive(yes)
}

// UPSTREAM_EXAMPLE_OUTPUT prevents output drift during the example adaptation.
const UPSTREAM_EXAMPLE_OUTPUT = `界 3
世 3
  1
, 1
o 1
l 1
l 1
e 1
H 1
界 3
世 3
  1
, 1
o 1
l 1
l 1
e 1
H 1
H 1
e 1
l 1
l 1
o 1
, 1
  1
世 3
界 3
H 1
e 1
l 1
l 1
o 1
, 1
  1
世 3
界 3
[228 184 150]
3
0: [239 191 189] � 3
1: [239 191 189] � 3
2: [239 191 189] � 3
true
false
true
false
bytes = 13
runes = 9
bytes = 13
runes = 9
1
3
true
true
false
true
false
true
false
true
false
𐀀
init𐀀
`

type example_buffer []byte

// Write keeps captured example output inside the shared dependency graph.
func (buffer *example_buffer) Write(data []byte) (count int, err error) {
	*buffer = append(*buffer, data...)
	return len(data), nil
}

// Test_Upstream_Examples checks every example from the upstream suite.
func Test_Upstream_Examples(t *testing.T) {
	var output example_buffer
	example_decode_final_character(&output)
	example_decode_final_character_text(&output)
	example_decode_character(&output)
	example_decode_character_text(&output)
	example_encode_character(&output)
	example_encode_character_invalid(&output)
	example_full_character(&output)
	example_full_character_text(&output)
	example_character_count(&output)
	example_character_count_text(&output)
	example_character_size(&output)
	example_character_start(&output)
	example_valid(&output)
	example_valid_character(&output)
	example_valid_text(&output)
	example_append_character(&output)
	testify.Equal(t, UPSTREAM_EXAMPLE_OUTPUT, string(output))
}
