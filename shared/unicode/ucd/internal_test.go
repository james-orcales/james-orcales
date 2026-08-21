package ucd

import (
	"fmt"
	"runtime"
	"testing"

	"local/james-orcales/shared/math/bits"
)

func test_equal(
	t *testing.T, expected any, actual any, message_and_arguments ...any,
) {
	t.Helper()
	if encoded, ok := expected.(Encoded_Data); ok {
		expected = string(encoded)
	}
	if encoded, ok := actual.(Encoded_Data); ok {
		actual = string(encoded)
	}
	if expected != actual {
		message := "values differ"
		if len(message_and_arguments) > 0 {
			message = fmt.Sprintf(
				message_and_arguments[0].(string), message_and_arguments[1:]...,
			)
		}
		t.Errorf("%s: expected %v, actual %v", message, expected, actual)
	}
}

func test_true(t *testing.T, actual bool, message_and_arguments ...any) {
	t.Helper()
	if !actual {
		t.Errorf(message_and_arguments[0].(string), message_and_arguments[1:]...)
	}
}

func test_false(t *testing.T, actual bool, message_and_arguments ...any) {
	t.Helper()
	if actual {
		t.Errorf(message_and_arguments[0].(string), message_and_arguments[1:]...)
	}
}

func test_panics(t *testing.T, operation func(), message_and_arguments ...any) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf(message_and_arguments[0].(string), message_and_arguments[1:]...)
		}
	}()
	operation()
}

func named_table(
	kind Table_Kind, name Name,
) (table Range_Table, found Boolean) {
	var ranges_16 [RANGES_16_COUNT_MAXIMUM]Range_16
	var ranges_32 [RANGES_32_COUNT_MAXIMUM]Range_32
	return Named_Table(kind, name, ranges_16[:], ranges_32[:])
}

func turkish_case() (special Special_Case) {
	Turkish_Case(Special_Case_Destination(&special))
	return special
}

func encoded_ranges_16(count int) (data string) {
	buffer := make(
		[]byte, 0,
		count*RANGE_16_BYTE_COUNT*HEXADECIMAL_BYTE_CHARACTER_COUNT,
	)
	for index := range count {
		buffer = append_fixed_hexadecimal(buffer, uint32(index), 4)
		buffer = append_fixed_hexadecimal(buffer, uint32(index), 4)
		buffer = append_fixed_hexadecimal(buffer, 1, 4)
	}
	return string(buffer)
}

func encoded_ranges_32(count int) (data string) {
	buffer := make(
		[]byte, 0,
		count*RANGE_32_BYTE_COUNT*HEXADECIMAL_BYTE_CHARACTER_COUNT,
	)
	for index := range count {
		value := uint32(RANGE_32_MINIMUM) + uint32(index)
		buffer = append_fixed_hexadecimal(buffer, value, 8)
		buffer = append_fixed_hexadecimal(buffer, value, 8)
		buffer = append_fixed_hexadecimal(buffer, 1, 8)
	}
	return string(buffer)
}

func append_fixed_hexadecimal(
	buffer []byte, value uint32, digit_size int,
) (extended []byte) {
	const HEXADECIMAL_DIGITS = "0123456789abcdef"
	buffer_size := len(buffer)
	buffer = append(buffer, make([]byte, digit_size)...)
	for digit_index := buffer_size + digit_size - 1; digit_index >= buffer_size; digit_index-- {
		buffer[digit_index] = HEXADECIMAL_DIGITS[value&0xF]
		value >>= 4
	}
	return buffer
}

func binary_case_range(one Case_Range) (data string) {
	buffer := make([]byte, 0, CASE_RANGE_BYTE_COUNT)
	values := [...]uint32{
		uint32(one.Minimum), uint32(one.Maximum),
		uint32(one.Deltas.Upper), uint32(one.Deltas.Lower), uint32(one.Deltas.Title),
	}
	for _, value := range values {
		buffer = append(
			buffer,
			byte(value>>ENCODED_32_FIRST_BYTE_SHIFT),
			byte(value>>ENCODED_32_SECOND_BYTE_SHIFT),
			byte(value>>bits.BIT_COUNT_8_MAXIMUM),
			byte(value),
		)
	}
	return string(buffer)
}

// Test_Encoded_Range_Boundaries verifies each private binary-search boundary.
func Test_Encoded_Range_Boundaries(t *testing.T) {
	t.Parallel()
	test_false(t, bool(encoded_ranges_16_contain(
		"", Data_Position(DATA_POSITION_MAXIMUM),
		DATA_COUNT_MINIMUM, Code_Point_16(RANGE_16_MAXIMUM),
	)), "an empty 16-bit range collection")
	test_true(t, bool(encoded_ranges_16_contain(
		Encoded_Data(encoded_ranges_16(1)), DATA_POSITION_MINIMUM, 1, 0,
	)), "one 16-bit range")
	test_true(t, bool(encoded_ranges_16_contain(
		Encoded_Data("00"+encoded_ranges_16(2)), 1, 2, 1,
	)), "two 16-bit ranges after one byte")
	test_true(t, bool(encoded_ranges_16_contain(
		Encoded_Data("0000"+encoded_ranges_16(DATA_COUNT_MAXIMUM)),
		2, DATA_COUNT_MAXIMUM, 2,
	)), "the maximum 16-bit range collection after two bytes")

	test_false(t, bool(encoded_ranges_32_contain(
		"", Data_Position(DATA_POSITION_MAXIMUM),
		DATA_COUNT_MINIMUM, Code_Point_32(RANGE_32_MAXIMUM),
	)), "an empty 32-bit range collection")
	test_true(t, bool(encoded_ranges_32_contain(
		Encoded_Data(encoded_ranges_32(1)),
		DATA_POSITION_MINIMUM, 1, Code_Point_32(RANGE_32_MINIMUM),
	)), "one 32-bit range")
	test_true(t, bool(encoded_ranges_32_contain(
		Encoded_Data("00"+encoded_ranges_32(2)),
		1, 2, Code_Point_32(RANGE_32_MINIMUM+1),
	)), "two 32-bit ranges after one byte")
	test_true(t, bool(encoded_ranges_32_contain(
		Encoded_Data("0000"+encoded_ranges_32(DATA_COUNT_MAXIMUM)),
		2, DATA_COUNT_MAXIMUM, Code_Point_32(RANGE_32_MINIMUM+2),
	)), "the maximum 32-bit range collection after two bytes")
	maximum_range := fmt.Sprintf(
		"%08x%08x%08x", RANGE_32_MAXIMUM, RANGE_32_MAXIMUM, RANGE_32_STRIDE_MINIMUM,
	)
	test_true(t, bool(encoded_ranges_32_contain(
		Encoded_Data(maximum_range),
		DATA_POSITION_MINIMUM, 1, Code_Point_32(RANGE_32_MAXIMUM),
	)), "the maximum 32-bit code point")
}

// Test_Encoded_Data_Boundaries verifies every encoded storage boundary.
func Test_Encoded_Data_Boundaries(t *testing.T) {
	t.Parallel()
	maximum_text := Encoded_Data(repeat("0", ENCODED_DATA_SIZE_MAXIMUM))
	maximum_binary := Encoded_Data(repeat("\x00", ENCODED_DATA_SIZE_MAXIMUM))

	test_false(t, bool(binary_range_table_contains("", "", 1)), "empty binary ranges")
	test_false(t, bool(binary_range_table_contains("0", "", 1)), "one 16-bit byte")
	test_false(t, bool(binary_range_table_contains("00", "", 1)), "two 16-bit bytes")
	test_false(t, bool(binary_range_table_contains(maximum_binary, "", 1)),
		"maximum 16-bit data")
	test_false(t, bool(binary_range_table_contains("", "0", RUNE_MAX)),
		"one 32-bit byte")
	test_false(t, bool(binary_range_table_contains("", "00", RUNE_MAX)),
		"two 32-bit bytes")
	test_false(t, bool(binary_range_table_contains("", maximum_binary, RUNE_MAX)),
		"maximum 32-bit data")

	for _, data := range []Encoded_Data{"", "0", "00", maximum_binary} {
		_, found := encoded_case_range(data, RUNE_MAX)
		test_false(t, bool(found), "case range in boundary data")
	}
	for _, data := range []Encoded_Data{"0", "00", maximum_text} {
		test_false(t, bool(encoded_ranges_16_contain(data, 0, 0, 0)),
			"16-bit range in boundary data")
		test_false(t, bool(encoded_ranges_32_contain(
			data, 0, 0, Code_Point_32(RANGE_32_MINIMUM),
		)), "32-bit range in boundary data")
	}

	test_true(t, bool(encoded_name_equal("0", 0, 0, "")), "one name data byte")
	test_true(t, bool(encoded_name_equal("00", 0, 0, "")), "two name data bytes")
	test_true(t, bool(encoded_name_equal(maximum_text, 0, 0, "")),
		"maximum name data")
	test_panics(t, func() { encoded_number("", 0, ENCODED_WIDTH_BYTE) },
		"an empty encoded number")
	test_panics(t, func() { encoded_number("0", 0, ENCODED_WIDTH_BYTE) },
		"a partial encoded number")
	test_equal(t, Encoded_Number(0), encoded_number("00", 0, ENCODED_WIDTH_BYTE))
}

// Test_Encoded_Directory_Data_Boundaries verifies named table storage boundaries.
func Test_Encoded_Directory_Data_Boundaries(t *testing.T) {
	t.Parallel()
	test_false(t, bool(named_table_contains("", "x", 0)), "empty named data")
	test_false(t, bool(named_table_contains("0", "x", 0)), "one named data byte")
	test_panics(t, func() { named_table_contains("00", "x", 0) },
		"partial named data")
	maximum_named_data := Encoded_Data(
		"00000fffff" + repeat("0", ENCODED_DATA_SIZE_MAXIMUM-10),
	)
	test_false(t, bool(named_table_contains(maximum_named_data, "x", 0)),
		"maximum named data")
	_, found := named_table_data("", "x")
	test_false(t, bool(found), "empty named table data")
	_, found = named_table_data("0", "x")
	test_false(t, bool(found), "one named table data byte")
	test_panics(t, func() { named_table_data("00", "x") }, "partial named table data")
	_, found = named_table_data(maximum_named_data, "x")
	test_false(t, bool(found), "maximum named table data")
	one_byte_table, found := named_table_data("000000000100", "")
	test_true(t, bool(found), "one-byte named table")
	test_equal(t, 2, len(one_byte_table))
	two_byte_table, found := named_table_data("00000000020000", "")
	test_true(t, bool(found), "two-byte named table")
	test_equal(t, 4, len(two_byte_table))

	var ranges_16 [RANGES_16_COUNT_MAXIMUM]Range_16
	var ranges_32 [RANGES_32_COUNT_MAXIMUM]Range_32
	test_panics(t, func() { decode_range_table("", ranges_16[:], ranges_32[:]) },
		"empty decoded table data")
	test_panics(t, func() { decode_range_table("00", ranges_16[:], ranges_32[:]) },
		"one decoded table byte")
	test_panics(t, func() { decode_range_table("0000", ranges_16[:], ranges_32[:]) },
		"two decoded table bytes")
	test_panics(t, func() { encoded_range_table_contains("", 0) },
		"empty searched table data")
	test_panics(t, func() { encoded_range_table_contains("00", 0) },
		"one searched table byte")
	test_panics(t, func() { encoded_range_table_contains("0000", 0) },
		"two searched table bytes")
	maximum_table_data := Table_Data(repeat(
		"0", TABLE_DATA_BYTE_COUNT_MAXIMUM*HEXADECIMAL_BYTE_CHARACTER_COUNT,
	))
	test_false(t, bool(encoded_range_table_contains(maximum_table_data, 0)),
		"maximum searched table data")
}

// Test_Encoded_Case_Range_Boundaries verifies every decoded case field boundary.
func Test_Encoded_Case_Range_Boundaries(t *testing.T) {
	t.Parallel()
	cases := [...]struct {
		Range     Case_Range
		Character Character
	}{
		{Case_Range{
			Minimum: Case_Range_Minimum(RUNE_MAX),
			Maximum: Case_Range_Maximum(RUNE_MAX),
			Deltas: Case_Delta{
				Upper: Upper_Case_Delta(CASE_DELTA_MINIMUM),
				Lower: Lower_Case_Delta(CASE_DELTA_MINIMUM),
				Title: Title_Case_Delta(CASE_DELTA_MINIMUM),
			},
		}, RUNE_MAX},
		{Case_Range{Minimum: 1, Maximum: 1, Deltas: Case_Delta{
			Upper: 1, Lower: 1, Title: 1,
		}}, 1},
		{Case_Range{Minimum: 2, Maximum: 2, Deltas: Case_Delta{
			Upper: 2, Lower: 2, Title: 2,
		}}, 2},
		{Case_Range{Minimum: 1, Maximum: 1, Deltas: Case_Delta{
			Upper: -1, Lower: -1, Title: -1,
		}}, 1},
		{Case_Range{
			Minimum: 0, Maximum: 1,
			Deltas: Case_Delta{
				Upper: Upper_Case_Delta(CASE_DELTA_MAXIMUM),
				Lower: Lower_Case_Delta(CASE_DELTA_MAXIMUM),
				Title: Title_Case_Delta(CASE_DELTA_MAXIMUM),
			},
		}, 0},
	}
	for _, one := range cases {
		actual, found := encoded_case_range(
			Encoded_Data(binary_case_range(one.Range)), one.Character,
		)
		test_true(t, bool(found), "encoded case range")
		test_equal(t, one.Range, actual, "decoded case range")
	}
}

// Test_Encoded_Primitive_Boundaries verifies primitive decoder positions, sizes, and values.
func Test_Encoded_Primitive_Boundaries(t *testing.T) {
	t.Parallel()
	test_true(t, bool(encoded_name_equal(
		"", DATA_POSITION_MINIMUM, DATA_COUNT_MINIMUM, "",
	)), "an empty name at the minimum position_count")
	test_true(t, bool(encoded_name_equal("", 1, 0, "")),
		"an empty name at position_count one")
	test_true(t, bool(encoded_name_equal("", 2, 0, "")),
		"an empty name at position_count two")
	test_true(t, bool(encoded_name_equal(
		"", Data_Position(DATA_POSITION_MAXIMUM), DATA_COUNT_MINIMUM, "",
	)), "an empty name at the maximum position_count")
	maximum_name := Name(repeat("x", NAME_SIZE_MAXIMUM))
	test_false(t, bool(encoded_name_equal(
		"", Data_Position(DATA_POSITION_MAXIMUM), DATA_COUNT_MAXIMUM, maximum_name,
	)), "a maximum count cannot equal a bounded name")

	maximum_position_buffer := make([]byte, ENCODED_DATA_SIZE_MAXIMUM)
	for index := range maximum_position_buffer {
		maximum_position_buffer[index] = '0'
	}
	maximum_position := DATA_POSITION_MAXIMUM * HEXADECIMAL_BYTE_CHARACTER_COUNT
	maximum_position_buffer[maximum_position] = 'f'
	maximum_position_buffer[maximum_position+1] = 'f'
	maximum_position_data := string(maximum_position_buffer)
	test_equal(t, Encoded_Number(0xff), encoded_number(
		Encoded_Data(maximum_position_data),
		Data_Position(DATA_POSITION_MAXIMUM), ENCODED_WIDTH_BYTE,
	))
	test_equal(t, Encoded_Number(ENCODED_NUMBER_MAXIMUM), encoded_number(
		"ffffffff", DATA_POSITION_MINIMUM, ENCODED_WIDTH_32,
	))
}

// Test_Case_Data_Uses_Native_Width prevents text decoding in case-conversion lookups.
func Test_Case_Data_Uses_Native_Width(t *testing.T) {
	t.Parallel()
	test_equal(
		t, CASE_RANGE_COUNT*CASE_RANGE_BYTE_COUNT, len(CASE_RANGE_DATA),
		"case ranges use one byte per encoded byte",
	)
	test_equal(
		t, CASE_ORBIT_COUNT*CASE_ORBIT_PAIR_BYTE_COUNT, len(CASE_ORBIT_DATA),
		"case orbits use one byte per encoded byte",
	)
}

// Test_ASCII_Fold_Data checks the complete upstream ASCII fold table.
func Test_ASCII_Fold_Data(t *testing.T) {
	t.Parallel()
	test_equal(t, (int(ASCII_MAX)+1)*2, len(ASCII_FOLD_DATA))
	for character := Character(0); character <= ASCII_MAX; character++ {
		position := int(character) * 2
		folded := Character(
			uint16(ASCII_FOLD_DATA[position])<<8 |
				uint16(ASCII_FOLD_DATA[position+1]),
		)
		test_equal(
			t, reference_ascii_fold(character), rune(folded),
			"the ASCII fold for %U", character,
		)
	}
}

func reference_ascii_fold(character Character) (folded rune) {
	switch character {
	case 'k':
		return '\u212a'
	case 's':
		return '\u017f'
	}
	if 'A' <= character {
		if character <= 'Z' {
			return rune(character + ('a' - 'A'))
		}
	}
	if 'a' <= character {
		if character <= 'z' {
			return rune(character - ('a' - 'A'))
		}
	}
	return rune(character)
}

func test_digit() (tests []rune) {
	return []rune{
		0x0030,
		0x0039,
		0x0661,
		0x06F1,
		0x07C9,
		0x0966,
		0x09EF,
		0x0A66,
		0x0AEF,
		0x0B66,
		0x0B6F,
		0x0BE6,
		0x0BEF,
		0x0C66,
		0x0CEF,
		0x0D66,
		0x0D6F,
		0x0E50,
		0x0E59,
		0x0ED0,
		0x0ED9,
		0x0F20,
		0x0F29,
		0x1040,
		0x1049,
		0x1090,
		0x1091,
		0x1099,
		0x17E0,
		0x17E9,
		0x1810,
		0x1819,
		0x1946,
		0x194F,
		0x19D0,
		0x19D9,
		0x1B50,
		0x1B59,
		0x1BB0,
		0x1BB9,
		0x1C40,
		0x1C49,
		0x1C50,
		0x1C59,
		0xA620,
		0xA629,
		0xA8D0,
		0xA8D9,
		0xA900,
		0xA909,
		0xAA50,
		0xAA59,
		0xFF10,
		0xFF19,
		0x104A1,
		0x1D7CE,
	}
}

func test_letter() (tests []rune) {
	return []rune{
		0x0041,
		0x0061,
		0x00AA,
		0x00BA,
		0x00C8,
		0x00DB,
		0x00F9,
		0x02EC,
		0x0535,
		0x06E6,
		0x093D,
		0x0A15,
		0x0B99,
		0x0DC0,
		0x0EDD,
		0x1000,
		0x1200,
		0x1312,
		0x1401,
		0x1885,
		0x2C00,
		0xA800,
		0xF900,
		0xFA30,
		0xFFDA,
		0xFFDC,
		0x10000,
		0x10300,
		0x10400,
		0x20000,
		0x2F800,
		0x2FA1D,
	}
}

// Test_Digit preserves the upstream behavior coverage.
func Test_Digit(t *testing.T) {
	for _, r := range test_digit() {
		if !bool(Is_Digit(Character(r))) {
			t.Errorf("bool(Is_Digit(Character(U+%04X))) = false, want true", r)
		}
	}
	for _, r := range test_letter() {
		if bool(Is_Digit(Character(r))) {
			t.Errorf("bool(Is_Digit(Character(U+%04X))) = true, want false", r)
		}
	}
}

// The Latin-1 fast path must agree with the complete digit table.

// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Functions starting with "Is" can be used to inspect which table of range a
// rune belongs to. Note that runes may fit into more than one range.

// The mixed fixture exercises overlapping character classes.

// Test_Digit_Optimization preserves the upstream behavior coverage.
func Test_Digit_Optimization(t *testing.T) {
	table, found := named_table(TABLE_KIND_CATEGORY, "Nd")
	if !found {
		t.Fatal("Nd is not a known category")
	}
	for i := rune(0); i <= rune(LATIN_1_MAX); i++ {
		from_table := Is(Range_Table_Handle(&table), Character(i))
		if bool(from_table) != bool(
			Is_Digit(Character(i)),
		) {
			t.Errorf(

				"Is_Digit(U+%04X) disagrees with category Nd", i)
		}
	}
}

func example_is(output *example_buffer) {

	const MIXED = "\b5Ὂg̀9! ℃ᾭG"
	for _, c := range MIXED {
		*output = fmt.Appendf(*output, "For %q:\n", c)
		if bool(Is_Control(Character(c))) {
			*output = fmt.Appendln(*output, "\tis control rune")
		}
		if bool(Is_Digit(Character(c))) {
			*output = fmt.Appendln(*output, "\tis digit rune")
		}
		if bool(Is_Graphic(Character(c))) {
			*output = fmt.Appendln(*output, "\tis graphic rune")
		}
		if bool(Is_Letter(Character(c))) {
			*output = fmt.Appendln(*output, "\tis letter rune")
		}
		if bool(Is_Lower(Character(c))) {
			*output = fmt.Appendln(*output, "\tis lower case rune")
		}
		if bool(Is_Mark(Character(c))) {
			*output = fmt.Appendln(*output, "\tis mark rune")
		}
		if bool(Is_Number(Character(c))) {
			*output = fmt.Appendln(*output, "\tis number rune")
		}
		if bool(Is_Print(Character(c))) {
			*output = fmt.Appendln(*output, "\tis printable rune")
		}
		if !bool(Is_Print(Character(c))) {
			*output = fmt.Appendln(*output, "\tis not printable rune")
		}
		if bool(Is_Punctuation(Character(c))) {
			*output = fmt.Appendln(*output, "\tis punct rune")
		}
		if bool(Is_Space(Character(c))) {
			*output = fmt.Appendln(*output, "\tis space rune")
		}
		if bool(Is_Symbol(Character(c))) {
			*output = fmt.Appendln(*output, "\tis symbol rune")
		}
		if bool(Is_Title(Character(c))) {
			*output = fmt.Appendln(*output, "\tis title case rune")
		}
		if bool(Is_Upper(Character(c))) {
			*output = fmt.Appendln(*output, "\tis upper case rune")
		}
	}

}

func example_simple_fold(output *example_buffer) {
	*output = fmt.Appendf(*output, "%#U\n", Simple_Fold(Character('A')))
	*output = fmt.Appendf(*output, "%#U\n", Simple_Fold(Character('a')))
	*output = fmt.Appendf(*output, "%#U\n", Simple_Fold(Character('K')))
	*output = fmt.Appendf(*output, "%#U\n", Simple_Fold(Character('k')))
	*output = fmt.Appendf(*output, "%#U\n", Simple_Fold(Character('\u212A')))
	*output = fmt.Appendf(*output, "%#U\n", Simple_Fold(Character('1')))
}

func example_to(output *example_buffer) {
	const LC_G = 'g'
	*output = fmt.Appendf(*output, "%#U\n", To(UPPER_CASE, Character(LC_G)))
	*output = fmt.Appendf(*output, "%#U\n", To(LOWER_CASE, Character(LC_G)))
	*output = fmt.Appendf(*output, "%#U\n", To(TITLE_CASE, Character(LC_G)))

	const UC_G = 'G'
	*output = fmt.Appendf(*output, "%#U\n", To(UPPER_CASE, Character(UC_G)))
	*output = fmt.Appendf(*output, "%#U\n", To(LOWER_CASE, Character(UC_G)))
	*output = fmt.Appendf(*output, "%#U\n", To(TITLE_CASE, Character(UC_G)))
}

func example_to_lower(output *example_buffer) {
	const UC_G = 'G'
	*output = fmt.Appendf(*output, "%#U\n", To_Lower(Character(UC_G)))
}
func example_to_title(output *example_buffer) {
	const UC_G = 'g'
	*output = fmt.Appendf(*output, "%#U\n", To_Title(Character(UC_G)))
}

func example_to_upper(output *example_buffer) {
	const UC_G = 'g'
	*output = fmt.Appendf(*output, "%#U\n", To_Upper(Character(UC_G)))
}

func example_special_case(output *example_buffer) {
	special := turkish_case()

	const LCI = 'i'
	*output = fmt.Appendf(*output, "%#U\n", Special_Case_To_Lower(special, LCI))
	*output = fmt.Appendf(*output, "%#U\n", Special_Case_To_Title(special, LCI))
	*output = fmt.Appendf(*output, "%#U\n", Special_Case_To_Upper(special, LCI))

	const UCI = 'İ'
	*output = fmt.Appendf(*output, "%#U\n", Special_Case_To_Lower(special, UCI))
	*output = fmt.Appendf(*output, "%#U\n", Special_Case_To_Title(special, UCI))
	*output = fmt.Appendf(*output, "%#U\n", Special_Case_To_Upper(special, UCI))
}

func example_is_digit(output *example_buffer) {
	*output = fmt.Appendf(*output, "%t\n", bool(Is_Digit(Character('৩'))))
	*output = fmt.Appendf(*output, "%t\n", bool(Is_Digit(Character('A'))))
}

func example_is_number(output *example_buffer) {
	*output = fmt.Appendf(*output, "%t\n", bool(Is_Number(Character('Ⅷ'))))
	*output = fmt.Appendf(*output, "%t\n", bool(Is_Number(Character('A'))))
}

func example_is_letter(output *example_buffer) {
	*output = fmt.Appendf(*output, "%t\n", bool(Is_Letter(Character('A'))))
	*output = fmt.Appendf(*output, "%t\n", bool(Is_Letter(Character('7'))))
}

func example_is_lower(output *example_buffer) {
	*output = fmt.Appendf(*output, "%t\n", bool(Is_Lower(Character('a'))))
	*output = fmt.Appendf(*output, "%t\n", bool(Is_Lower(Character('A'))))
}

func example_is_upper(output *example_buffer) {
	*output = fmt.Appendf(*output, "%t\n", bool(Is_Upper(Character('A'))))
	*output = fmt.Appendf(*output, "%t\n", bool(Is_Upper(Character('a'))))
}

func example_is_title(output *example_buffer) {
	*output = fmt.Appendf(*output, "%t\n", bool(Is_Title(Character('ǅ'))))
	*output = fmt.Appendf(*output, "%t\n", bool(Is_Title(Character('a'))))
}

func example_is_space(output *example_buffer) {
	*output = fmt.Appendf(*output, "%t\n", bool(Is_Space(Character(' '))))
	*output = fmt.Appendf(*output, "%t\n", bool(Is_Space(Character('\n'))))
	*output = fmt.Appendf(*output, "%t\n", bool(Is_Space(Character('\t'))))
	*output = fmt.Appendf(*output, "%t\n", bool(Is_Space(Character('a'))))
}

// UPSTREAM_EXAMPLE_OUTPUT prevents output drift during the example adaptation.
const UPSTREAM_EXAMPLE_OUTPUT = `For '\b':
	is control rune
	is not printable rune
For '5':
	is digit rune
	is graphic rune
	is number rune
	is printable rune
For 'Ὂ':
	is graphic rune
	is letter rune
	is printable rune
	is upper case rune
For 'g':
	is graphic rune
	is letter rune
	is lower case rune
	is printable rune
For '̀':
	is graphic rune
	is mark rune
	is printable rune
For '9':
	is digit rune
	is graphic rune
	is number rune
	is printable rune
For '!':
	is graphic rune
	is printable rune
	is punct rune
For ' ':
	is graphic rune
	is printable rune
	is space rune
For '℃':
	is graphic rune
	is printable rune
	is symbol rune
For 'ᾭ':
	is graphic rune
	is letter rune
	is printable rune
	is title case rune
For 'G':
	is graphic rune
	is letter rune
	is printable rune
	is upper case rune
U+0061 'a'
U+0041 'A'
U+006B 'k'
U+212A 'K'
U+004B 'K'
U+0031 '1'
U+0047 'G'
U+0067 'g'
U+0047 'G'
U+0047 'G'
U+0067 'g'
U+0047 'G'
U+0067 'g'
U+0047 'G'
U+0047 'G'
U+0069 'i'
U+0130 'İ'
U+0130 'İ'
U+0069 'i'
U+0130 'İ'
U+0130 'İ'
true
false
true
false
true
false
true
false
true
false
true
false
true
true
true
false
`

type example_buffer []byte

// Test_Upstream_Examples preserves every upstream example.
func Test_Upstream_Examples(t *testing.T) {
	var output example_buffer
	example_is(&output)
	example_simple_fold(&output)
	example_to(&output)
	example_to_lower(&output)
	example_to_title(&output)
	example_to_upper(&output)
	example_special_case(&output)
	example_is_digit(&output)
	example_is_number(&output)
	example_is_letter(&output)
	example_is_lower(&output)
	example_is_upper(&output)
	example_is_title(&output)
	example_is_space(&output)
	test_equal(t, UPSTREAM_EXAMPLE_OUTPUT, string(output))
}

// Test_Is_Control_Latin_1 preserves the upstream behavior coverage.
func Test_Is_Control_Latin_1(t *testing.T) {
	for i := rune(0); i <= rune(LATIN_1_MAX); i++ {
		got := bool(Is_Control(Character(i)))
		want := false
		switch {
		case 0x00 <= i && i <= 0x1F:
			want = true
		case 0x7F <= i && i <= 0x9F:
			want = true
		}
		if got != want {
			t.Errorf("%U incorrect: got %t; want %t", i, got, want)
		}
	}
}

// Test_Is_Letter_Latin_1 preserves the upstream behavior coverage.
func Test_Is_Letter_Latin_1(t *testing.T) {
	for i := rune(0); i <= rune(LATIN_1_MAX); i++ {
		got := bool(Is_Letter(Character(i)))
		want := in_upstream_categories(i, "L")
		if got != want {
			t.Errorf("%U incorrect: got %t; want %t", i, got, want)
		}
	}
}

// Test_Is_Upper_Latin_1 preserves the upstream behavior coverage.
func Test_Is_Upper_Latin_1(t *testing.T) {
	for i := rune(0); i <= rune(LATIN_1_MAX); i++ {
		got := bool(Is_Upper(Character(i)))
		want := in_upstream_categories(i, "Lu")
		if got != want {
			t.Errorf("%U incorrect: got %t; want %t", i, got, want)
		}
	}
}

// Test_Is_Lower_Latin_1 preserves the upstream behavior coverage.
func Test_Is_Lower_Latin_1(t *testing.T) {
	for i := rune(0); i <= rune(LATIN_1_MAX); i++ {
		got := bool(Is_Lower(Character(i)))
		want := in_upstream_categories(i, "Ll")
		if got != want {
			t.Errorf("%U incorrect: got %t; want %t", i, got, want)
		}
	}
}

// Test_Number_Latin_1 preserves the upstream behavior coverage.
func Test_Number_Latin_1(t *testing.T) {
	for i := rune(0); i <= rune(LATIN_1_MAX); i++ {
		got := bool(Is_Number(Character(i)))
		want := in_upstream_categories(i, "N")
		if got != want {
			t.Errorf("%U incorrect: got %t; want %t", i, got, want)
		}
	}
}

// Test_Is_Print_Latin_1 preserves the upstream behavior coverage.
func Test_Is_Print_Latin_1(t *testing.T) {
	for i := rune(0); i <= rune(LATIN_1_MAX); i++ {
		got := bool(Is_Print(Character(i)))
		want := in_upstream_categories(i, "L", "M", "N", "P", "S")
		if i == ' ' {
			want = true
		}
		if got != want {
			t.Errorf("%U incorrect: got %t; want %t", i, got, want)
		}
	}
}

// Test_Is_Graphic_Latin_1 preserves the upstream behavior coverage.
func Test_Is_Graphic_Latin_1(t *testing.T) {
	for i := rune(0); i <= rune(LATIN_1_MAX); i++ {
		got := bool(Is_Graphic(Character(i)))
		want := in_upstream_categories(i, "L", "M", "N", "P", "S", "Zs")
		if got != want {
			t.Errorf("%U incorrect: got %t; want %t", i, got, want)
		}
	}
}

// Test_Derived_Classification_Tables prevents generated data from drifting from Unicode.
func Test_Derived_Classification_Tables(t *testing.T) {
	t.Parallel()
	print_tables := reference_classification_tables(t, "L", "M", "N", "P", "S")
	graphic_tables := reference_classification_tables(t, "L", "M", "N", "P", "S", "Zs")
	print := func(character rune) (yes bool) {
		return reference_is_print(print_tables, character)
	}
	graphic := func(character rune) (yes bool) {
		return reference_is_graphic(graphic_tables, character)
	}
	print_ranges_16, print_ranges_32 := derived_classification_data(
		print,
	)
	graphic_ranges_16, graphic_ranges_32 := derived_classification_data(
		graphic,
	)
	test_equal(t, PRINT_RANGES_16_DATA, print_ranges_16,
		"the generated 16-bit print ranges")
	test_equal(t, PRINT_RANGES_32_DATA, print_ranges_32,
		"the generated 32-bit print ranges")
	test_equal(t, GRAPHIC_RANGES_16_DATA, graphic_ranges_16,
		"the generated 16-bit graphic ranges")
	test_equal(t, GRAPHIC_RANGES_32_DATA, graphic_ranges_32,
		"the generated 32-bit graphic ranges")
	for character := rune(0); character <= rune(RUNE_MAX); character++ {
		shared_print := bool(Is_Print(Character(character)))
		if shared_print != print(character) {
			t.Errorf(
				"the print ranges contain %U: got %t", character, shared_print,
			)
		}
		shared_graphic := bool(Is_Graphic(Character(character)))
		if shared_graphic != graphic(character) {
			t.Errorf(
				"the graphic ranges contain %U: got %t", character, shared_graphic,
			)
		}
	}
}

func reference_is_print(tables []Range_Table, character rune) (yes bool) {
	if character == ' ' {
		return true
	}
	return reference_in_tables(tables, character)
}

func reference_is_graphic(tables []Range_Table, character rune) (yes bool) {
	return reference_in_tables(tables, character)
}

func reference_classification_tables(
	t *testing.T, names ...Name,
) (tables []Range_Table) {
	t.Helper()
	for _, name := range names {
		table, found := named_table(TABLE_KIND_CATEGORY, name)
		if !found {
			t.Fatalf("missing classification table %q", name)
		}
		tables = append(tables, table)
	}
	return tables
}

func reference_in_tables(tables []Range_Table, character rune) (yes bool) {
	for _, table := range tables {
		if reference_range_table_contains(table, character) {
			return true
		}
	}
	return false
}

func reference_range_table_contains(table Range_Table, character rune) (yes bool) {
	if character < 0 {
		return false
	}
	if character <= rune(RANGE_16_MAXIMUM) {
		low_index, high_count := 0, len(table.Ranges_16)
		for low_index < high_count {
			middle_index := int(uint(low_index+high_count) >> 1)
			one := table.Ranges_16[middle_index]
			if character < rune(one.Minimum) {
				high_count = middle_index
				continue
			}
			if character <= rune(one.Maximum) {
				difference := character - rune(one.Minimum)
				return difference%rune(one.Stride) == 0
			}
			low_index = middle_index + 1
		}
		return false
	}
	low_index, high_count := 0, len(table.Ranges_32)
	for low_index < high_count {
		middle_index := int(uint(low_index+high_count) >> 1)
		one := table.Ranges_32[middle_index]
		if character < rune(one.Minimum) {
			high_count = middle_index
			continue
		}
		if character <= rune(one.Maximum) {
			difference := character - rune(one.Minimum)
			return difference%rune(one.Stride) == 0
		}
		low_index = middle_index + 1
	}
	return false
}

func derived_classification_data(
	contains func(rune) (yes bool),
) (ranges_16_data string, ranges_32_data string) {
	points := make([]uint32, 0, int(RUNE_MAX)+1)
	for point := rune(0); point <= rune(RUNE_MAX); point++ {
		if contains(point) {
			points = append(points, uint32(point))
		}
	}
	ranges := make([][RANGE_VALUE_COUNT]uint32, 0, len(points))
	for position := 0; position < len(points); {
		minimum := points[position]
		if position+1 >= len(points) {
			ranges = append(ranges,
				[RANGE_VALUE_COUNT]uint32{minimum, minimum, 1})
			break
		}
		stride := points[position+1] - minimum
		end := position + 1
		for end+1 < len(points) && points[end+1]-points[end] == stride {
			end++
		}
		ranges = append(ranges,
			[RANGE_VALUE_COUNT]uint32{minimum, points[end], stride})
		position = end + 1
	}
	range_16_count := 0
	for _, one := range ranges {
		if one[1] > uint32(RANGE_16_MAXIMUM) {
			break
		}
		range_16_count++
	}
	range_32_count := len(ranges) - range_16_count
	ranges_16 := make([]byte, 0, range_16_count*RANGE_16_BYTE_COUNT)
	for _, one := range ranges[:range_16_count] {
		for _, value := range one {
			ranges_16 = append_fixed_binary(
				ranges_16, value, ENCODED_WIDTH_16,
			)
		}
	}
	ranges_32 := make([]byte, 0, range_32_count*RANGE_32_BYTE_COUNT)
	for _, one := range ranges[range_16_count:] {
		for _, value := range one {
			ranges_32 = append_fixed_binary(
				ranges_32, value, ENCODED_WIDTH_32,
			)
		}
	}
	return string(ranges_16), string(ranges_32)
}

func append_fixed_binary(
	buffer []byte, value uint32, width Encoded_Width,
) (extended []byte) {
	for byte_index := int(width) - 1; byte_index >= 0; byte_index-- {
		shift := byte_index * bits.BIT_COUNT_8_MAXIMUM
		buffer = append(buffer, byte(value>>shift))
	}
	return buffer
}

// Test_Is_Punctuation_Latin_1 preserves the upstream behavior coverage.
func Test_Is_Punctuation_Latin_1(t *testing.T) {
	for i := rune(0); i <= rune(LATIN_1_MAX); i++ {
		got := bool(Is_Punctuation(Character(i)))
		want := in_upstream_categories(i, "P")
		if got != want {
			t.Errorf("%U incorrect: got %t; want %t", i, got, want)
		}
	}
}

// Test_Is_Space_Latin_1 preserves the upstream behavior coverage.
func Test_Is_Space_Latin_1(t *testing.T) {
	for i := rune(0); i <= rune(LATIN_1_MAX); i++ {
		got := bool(Is_Space(Character(i)))
		want := bool(Is_Property(
			Character(i), "White_Space",
		))
		if got != want {
			t.Errorf("%U incorrect: got %t; want %t", i, got, want)
		}
	}
}

// Test_Is_Symbol_Latin_1 preserves the upstream behavior coverage.
func Test_Is_Symbol_Latin_1(t *testing.T) {
	for i := rune(0); i <= rune(LATIN_1_MAX); i++ {
		got := bool(Is_Symbol(Character(i)))
		want := in_upstream_categories(i, "S")
		if got != want {
			t.Errorf("%U incorrect: got %t; want %t", i, got, want)
		}
	}
}

func in_upstream_categories(character rune, names ...Name) (found bool) {
	for _, name := range names {
		if Is_Category(Character(character), name) {
			return true
		}
	}
	return false
}

func upper_test() (tests []rune) {
	return []rune{
		0x41,
		0xc0,
		0xd8,
		0x100,
		0x139,
		0x14a,
		0x178,
		0x181,
		0x376,
		0x3cf,
		0x13bd,
		0x1f2a,
		0x2102,
		0x2c00,
		0x2c10,
		0x2c20,
		0xa650,
		0xa722,
		0xff3a,
		0x10400,
		0x1d400,
		0x1d7ca,
	}
}

func not_upper_test() (tests []rune) {
	return []rune{
		0x40,
		0x5b,
		0x61,
		0x185,
		0x1b0,
		0x377,
		0x387,
		0x2150,
		0xab7d,
		0xffff,
		0x10000,
	}
}

func letter_test() (tests []rune) {
	return []rune{
		0x41,
		0x61,
		0xaa,
		0xba,
		0xc8,
		0xdb,
		0xf9,
		0x2ec,
		0x535,
		0x620,
		0x6e6,
		0x93d,
		0xa15,
		0xb99,
		0xdc0,
		0xedd,
		0x1000,
		0x1200,
		0x1312,
		0x1401,
		0x2c00,
		0xa800,
		0xf900,
		0xfa30,
		0xffda,
		0xffdc,
		0x10000,
		0x10300,
		0x10400,
		0x20000,
		0x2f800,
		0x2fa1d,
	}
}

func not_letter_test() (tests []rune) {
	return []rune{
		0x20,
		0x35,
		0x375,
		0x619,
		0x700,
		0x1885,
		0xfffe,
		0x1ffff,
		0x10ffff,
	}
}

func space_test() (tests []rune) {
	return []rune{
		0x09,
		0x0a,
		0x0b,
		0x0c,
		0x0d,
		0x20,
		0x85,
		0xA0,
		0x2000,
		0x3000,
	}
}

type case_fixture struct {
	Cas        Case
	In, Output rune
}

func case_test() (tests []case_fixture) {
	tests = case_test_general()
	return append(tests, case_test_supplementary()...)
}

func case_test_general() (tests []case_fixture) {
	return []case_fixture{

		{-1, '\n', 0xFFFD},
		{UPPER_CASE, -1, -1},
		{UPPER_CASE, 1 << 30, 1 << 30},

		{UPPER_CASE, '\n', '\n'},
		{UPPER_CASE, 'a', 'A'},
		{UPPER_CASE, 'A', 'A'},
		{UPPER_CASE, '7', '7'},
		{LOWER_CASE, '\n', '\n'},
		{LOWER_CASE, 'a', 'a'},
		{LOWER_CASE, 'A', 'a'},
		{LOWER_CASE, '7', '7'},
		{TITLE_CASE, '\n', '\n'},
		{TITLE_CASE, 'a', 'A'},
		{TITLE_CASE, 'A', 'A'},
		{TITLE_CASE, '7', '7'},

		{UPPER_CASE, 0x80, 0x80},
		{UPPER_CASE, 'Å', 'Å'},
		{UPPER_CASE, 'å', 'Å'},
		{LOWER_CASE, 0x80, 0x80},
		{LOWER_CASE, 'Å', 'å'},
		{LOWER_CASE, 'å', 'å'},
		{TITLE_CASE, 0x80, 0x80},
		{TITLE_CASE, 'Å', 'Å'},
		{TITLE_CASE, 'å', 'Å'},

		{UPPER_CASE, 0x0131, 'I'},
		{LOWER_CASE, 0x0131, 0x0131},
		{TITLE_CASE, 0x0131, 'I'},

		{UPPER_CASE, 0x0133, 0x0132},
		{LOWER_CASE, 0x0133, 0x0133},
		{TITLE_CASE, 0x0133, 0x0132},

		{UPPER_CASE, 0x212A, 0x212A},
		{LOWER_CASE, 0x212A, 'k'},
		{TITLE_CASE, 0x212A, 0x212A},
	}
}

func case_test_supplementary() (tests []case_fixture) {
	return []case_fixture{
		{UPPER_CASE, 0xA640, 0xA640},
		{LOWER_CASE, 0xA640, 0xA641},
		{TITLE_CASE, 0xA640, 0xA640},

		{UPPER_CASE, 0xA641, 0xA640},
		{LOWER_CASE, 0xA641, 0xA641},
		{TITLE_CASE, 0xA641, 0xA640},

		{UPPER_CASE, 0xA64E, 0xA64E},
		{LOWER_CASE, 0xA64E, 0xA64F},
		{TITLE_CASE, 0xA64E, 0xA64E},

		{UPPER_CASE, 0xA65F, 0xA65E},
		{LOWER_CASE, 0xA65F, 0xA65F},
		{TITLE_CASE, 0xA65F, 0xA65E},

		{UPPER_CASE, 0x0139, 0x0139},
		{LOWER_CASE, 0x0139, 0x013A},
		{TITLE_CASE, 0x0139, 0x0139},

		{UPPER_CASE, 0x013f, 0x013f},
		{LOWER_CASE, 0x013f, 0x0140},
		{TITLE_CASE, 0x013f, 0x013f},

		{UPPER_CASE, 0x0148, 0x0147},
		{LOWER_CASE, 0x0148, 0x0148},
		{TITLE_CASE, 0x0148, 0x0147},

		{UPPER_CASE, 0xab78, 0x13a8},
		{LOWER_CASE, 0xab78, 0xab78},
		{TITLE_CASE, 0xab78, 0x13a8},
		{UPPER_CASE, 0x13a8, 0x13a8},
		{LOWER_CASE, 0x13a8, 0xab78},
		{TITLE_CASE, 0x13a8, 0x13a8},

		{UPPER_CASE, 0x10400, 0x10400},
		{LOWER_CASE, 0x10400, 0x10428},
		{TITLE_CASE, 0x10400, 0x10400},

		{UPPER_CASE, 0x10427, 0x10427},
		{LOWER_CASE, 0x10427, 0x1044F},
		{TITLE_CASE, 0x10427, 0x10427},

		{UPPER_CASE, 0x10428, 0x10400},
		{LOWER_CASE, 0x10428, 0x10428},
		{TITLE_CASE, 0x10428, 0x10400},

		{UPPER_CASE, 0x1044F, 0x10427},
		{LOWER_CASE, 0x1044F, 0x1044F},
		{TITLE_CASE, 0x1044F, 0x10427},

		{UPPER_CASE, 0x10450, 0x10450},
		{LOWER_CASE, 0x10450, 0x10450},
		{TITLE_CASE, 0x10450, 0x10450},

		{LOWER_CASE, 0x2161, 0x2171},
		{UPPER_CASE, 0x0345, 0x0399},
	}
}

// Test_Is_Letter preserves the upstream behavior coverage.
func Test_Is_Letter(t *testing.T) {
	for _, r := range upper_test() {
		if !bool(Is_Letter(Character(r))) {
			t.Errorf("bool(Is_Letter(Character(U+%04X))) = false, want true", r)
		}
	}
	for _, r := range letter_test() {
		if !bool(Is_Letter(Character(r))) {
			t.Errorf("bool(Is_Letter(Character(U+%04X))) = false, want true", r)
		}
	}
	for _, r := range not_letter_test() {
		if bool(Is_Letter(Character(r))) {
			t.Errorf("bool(Is_Letter(Character(U+%04X))) = true, want false", r)
		}
	}
}

// Test_Is_Upper preserves the upstream behavior coverage.
func Test_Is_Upper(t *testing.T) {
	for _, r := range upper_test() {
		if !bool(Is_Upper(Character(r))) {
			t.Errorf("bool(Is_Upper(Character(U+%04X))) = false, want true", r)
		}
	}
	for _, r := range not_upper_test() {
		if bool(Is_Upper(Character(r))) {
			t.Errorf("bool(Is_Upper(Character(U+%04X))) = true, want false", r)
		}
	}
	for _, r := range not_letter_test() {
		if bool(Is_Upper(Character(r))) {
			t.Errorf("bool(Is_Upper(Character(U+%04X))) = true, want false", r)
		}
	}
}

func case_name(c Case) (name string) {
	switch c {
	case UPPER_CASE:
		return "UPPER_CASE"
	case LOWER_CASE:
		return "LOWER_CASE"
	case TITLE_CASE:
		return "TITLE_CASE"
	}
	return "ErrorCase"
}

// Test_To preserves the upstream behavior coverage.
func Test_To(t *testing.T) {
	for _, c := range case_test() {
		if c.Cas < UPPER_CASE {
			test_panics(t, func() {
				To(c.Cas, Character(c.In))
			}, "To rejects case %d", c.Cas)
			continue
		}
		r := rune(To(c.Cas, Character(c.In)))
		if c.Output != r {
			t.Errorf("To(U+%04X, %s) = U+%04X want U+%04X",
				c.In, case_name(c.Cas), r, c.Output)
		}
	}
}

// Test_To_Upper_Case preserves the upstream behavior coverage.
func Test_To_Upper_Case(t *testing.T) {
	for _, c := range case_test() {
		if c.Cas != UPPER_CASE {
			continue
		}
		r := rune(To_Upper(Character(c.In)))
		if c.Output != r {
			t.Errorf("To_Upper(U+%04X) = U+%04X want U+%04X",
				c.In, r, c.Output)
		}
	}
}

// Test_To_Lower_Case preserves the upstream behavior coverage.
func Test_To_Lower_Case(t *testing.T) {
	for _, c := range case_test() {
		if c.Cas != LOWER_CASE {
			continue
		}
		r := rune(To_Lower(Character(c.In)))
		if c.Output != r {
			t.Errorf("To_Lower(U+%04X) = U+%04X want U+%04X",
				c.In, r, c.Output)
		}
	}
}

// Test_To_Title_Case preserves the upstream behavior coverage.
func Test_To_Title_Case(t *testing.T) {
	for _, c := range case_test() {
		if c.Cas != TITLE_CASE {
			continue
		}
		r := rune(To_Title(Character(c.In)))
		if c.Output != r {
			t.Errorf("To_Title(U+%04X) = U+%04X want U+%04X",
				c.In, r, c.Output)
		}
	}
}

// Test_Is_Space preserves the upstream behavior coverage.
func Test_Is_Space(t *testing.T) {
	for _, c := range space_test() {
		if !bool(Is_Space(Character(c))) {
			t.Errorf("bool(Is_Space(Character(U+%04X))) = false; want true", c)
		}
	}
	for _, c := range letter_test() {
		if bool(Is_Space(Character(c))) {
			t.Errorf("bool(Is_Space(Character(U+%04X))) = true; want false", c)
		}
	}
}

// Check that the optimizations for IsLetter etc. agree with the tables.
// We only need to check the Latin-1 range.
// Test_Letter_Optimizations preserves the upstream behavior coverage.
func Test_Letter_Optimizations(t *testing.T) {
	for i := rune(0); i <= rune(LATIN_1_MAX); i++ {
		if in_upstream_categories(i, "L") !=
			bool(Is_Letter(Character(i))) {
			t.Errorf("Is_Letter(U+%04X) disagrees with category L", i)
		}
		if in_upstream_categories(i, "Lu") !=
			bool(Is_Upper(Character(i))) {
			t.Errorf("Is_Upper(U+%04X) disagrees with category Lu", i)
		}
		if in_upstream_categories(i, "Ll") !=
			bool(Is_Lower(Character(i))) {
			t.Errorf("Is_Lower(U+%04X) disagrees with category Ll", i)
		}
		if in_upstream_categories(i, "Lt") !=
			bool(Is_Title(Character(i))) {
			t.Errorf("Is_Title(U+%04X) disagrees with category Lt", i)
		}
		if bool(Is_Property(
			Character(i), "White_Space",
		)) != bool(Is_Space(Character(i))) {
			t.Errorf("Is_Space(U+%04X) disagrees with White_Space", i)
		}
		if rune(To(UPPER_CASE, Character(i))) != rune(To_Upper(Character(i))) {
			t.Errorf("rune(To_Upper(Character(U+%04X))) disagrees with To(Upper)", i)
		}
		if rune(To(LOWER_CASE, Character(i))) != rune(To_Lower(Character(i))) {
			t.Errorf("rune(To_Lower(Character(U+%04X))) disagrees with To(Lower)", i)
		}
		if rune(To(TITLE_CASE, Character(i))) != rune(To_Title(Character(i))) {
			t.Errorf("rune(To_Title(Character(U+%04X))) disagrees with To(Title)", i)
		}
	}
}

// Test_Turkish_Case preserves the upstream behavior coverage.
func Test_Turkish_Case(t *testing.T) {
	lower := []rune("abcçdefgğhıijklmnoöprsştuüvyz")
	upper := []rune("ABCÇDEFGĞHIİJKLMNOÖPRSŞTUÜVYZ")
	special := turkish_case()
	for i, l := range lower {
		u := upper[i]
		if rune(Special_Case_To_Lower(special, Character(l))) != l {
			t.Errorf("lower(U+%04X) is not U+%04X", l, l)
		}
		if rune(Special_Case_To_Upper(special, Character(u))) != u {
			t.Errorf("upper(U+%04X) is not U+%04X", u, u)
		}
		if rune(Special_Case_To_Upper(special, Character(l))) != u {
			t.Errorf("upper(U+%04X) is not U+%04X", l, u)
		}
		if rune(Special_Case_To_Lower(special, Character(u))) != l {
			t.Errorf("lower(U+%04X) is not U+%04X", u, l)
		}
		if rune(Special_Case_To_Title(special, Character(u))) != u {
			t.Errorf("title(U+%04X) is not U+%04X", u, u)
		}
		if rune(Special_Case_To_Title(special, Character(l))) != u {
			t.Errorf("title(U+%04X) is not U+%04X", l, u)
		}
	}
}

func simple_fold_tests() (tests []string) {
	return []string{
		// Simple_Fold returns the next equivalent character or wraps.
		// around to smaller values.

		// Easy cases.
		"Aa",
		"δΔ",

		// ASCII special cases.
		"KkK",
		"Ssſ",

		// Non-ASCII special cases.
		"ρϱΡ",
		"ͅΙιι",

		// Extra special cases: has lower/upper but no case fold.
		"İ",
		"ı",

		// Upper comes before lower (Cherokee).
		"\u13b0\uab80",
	}
}

// Test_Simple_Fold preserves the upstream behavior coverage.
func Test_Simple_Fold(t *testing.T) {
	for _, tt := range simple_fold_tests() {
		cycle := []rune(tt)
		r := cycle[len(cycle)-1]
		for _, output := range cycle {
			folded := rune(Simple_Fold(Character(r)))
			if folded != output {
				t.Errorf("rune(Simple_Fold(Character(%#U))) = %#U, want %#U",
					r, folded, output)
			}
			r = output
		}
	}

	if r := rune(Simple_Fold(Character(-42))); r != -42 {
		t.Errorf("rune(Simple_Fold(Character(-42))) = %v, want -42", r)
	}
}

func repeat(text string, count int) (repeated string) {
	buffer := make([]byte, len(text)*count)
	position := 0
	for range count {
		position += copy(buffer[position:], text)
	}
	return string(buffer)
}

// Test_Special_Case_No_Mapping preserves the upstream behavior coverage.
func Test_Special_Case_No_Mapping(t *testing.T) {
	// Issue 25636 needs zero delta to win over standard case conversion.
	special := Special_Case_Of(
		1,
		Case_Range{
			Minimum: 'A', Maximum: 'A',
			Deltas: Case_Delta{Upper: 0, Lower: 0, Title: 0},
		},
		Case_Range{},
		Case_Range{},
		Case_Range{},
	)
	mapped := make([]rune, 0, 3)
	for _, character := range "ABC" {
		mapped = append(mapped, rune(Special_Case_To_Lower(
			special, Character(character),
		)))
	}
	got := string(mapped)
	want := "Abc"
	if got != want {
		t.Errorf("got %q; want %q", got, want)
	}
}

// Test_Negative_Rune preserves the upstream behavior coverage.
func Test_Negative_Rune(t *testing.T) {
	// Negative values can resemble valid values after an unsigned conversion.
	non_latin1 := []uint32{
		0x0100, // Lu.
		0x0101, // Ll.
		0x01C5, // Lt.
		0x0300, // M.
		0x0660, // Nd.
		0x037E, // P.
		0x02C2, // S.
		0x1680, // Z.
	}
	for i_index := 0; i_index < int(LATIN_1_MAX)+len(non_latin1); i_index++ {
		base := uint32(i_index)
		if i_index >= int(LATIN_1_MAX) {
			base = non_latin1[i_index-int(LATIN_1_MAX)]
		}
		check_negative_character(t, base)
	}
}

func check_negative_character(t *testing.T, base uint32) {
	t.Helper()
	character := Character(rune(base - 1<<31))
	checks := []struct {
		Name  string
		Check func(Character) (yes Boolean)
	}{
		{"Is_Category", func(value Character) (yes Boolean) {
			return Is_Category(value, "L")
		}},
		{"Is_Control", Is_Control},
		{"Is_Digit", Is_Digit},
		{"Is_Graphic", Is_Graphic},
		{"Is_Letter", Is_Letter},
		{"Is_Lower", Is_Lower},
		{"Is_Mark", Is_Mark},
		{"Is_Number", Is_Number},
		{"Is_Print", Is_Print},
		{"Is_Punctuation", Is_Punctuation},
		{"Is_Space", Is_Space},
		{"Is_Symbol", Is_Symbol},
		{"Is_Title", Is_Title},
		{"Is_Upper", Is_Upper},
	}
	for _, check := range checks {
		if check.Check(character) {
			t.Errorf("%s(0x%x - 1<<31) = true, want false", check.Name, base)
		}
	}
}

// Benchmark_To_Upper preserves the upstream performance coverage.
func Benchmark_To_Upper(b *testing.B) {
	var result rune
	for i_index := 0; i_index < b.N; i_index++ {
		result = rune(To_Upper(Character('δ')))
	}
	runtime.KeepAlive(result)
}

// Benchmark_Is_Print_Letter guards the non-Latin-1 printable path.
func Benchmark_Is_Print_Letter(b *testing.B) {
	var result Boolean
	b.ReportAllocs()
	for range b.N {
		result = Is_Print('世')
	}
	runtime.KeepAlive(result)
}

// Benchmark_Is_Print_Unassigned guards the full printable-table search.
func Benchmark_Is_Print_Unassigned(b *testing.B) {
	var result Boolean
	b.ReportAllocs()
	for range b.N {
		result = Is_Print(0x0378)
	}
	runtime.KeepAlive(result)
}

// Benchmark_Is_Graphic_Unassigned guards the full graphic-table search.
func Benchmark_Is_Graphic_Unassigned(b *testing.B) {
	var result Boolean
	b.ReportAllocs()
	for range b.N {
		result = Is_Graphic(0x0378)
	}
	runtime.KeepAlive(result)
}

// Benchmark_To_Lower preserves the upstream performance coverage.
func Benchmark_To_Lower(b *testing.B) {
	var result rune
	for i_index := 0; i_index < b.N; i_index++ {
		result = rune(To_Lower(Character('Δ')))
	}
	runtime.KeepAlive(result)
}

// Benchmark_Simple_Fold preserves the upstream performance coverage.
func Benchmark_Simple_Fold(b *testing.B) {
	bench := func(name string, r rune) {
		b.Run(name, func(b *testing.B) {
			var result rune
			for i_index := 0; i_index < b.N; i_index++ {
				result = rune(Simple_Fold(Character(r)))
			}
			runtime.KeepAlive(result)
		})
	}
	bench("Upper", 'Δ')
	bench("Lower", 'δ')
	bench("Fold", '\u212A')
	bench("NoFold", '習')
}

type table_fixture struct {
	Rune   rune
	Script string
}

func in_category_test() (tests []table_fixture) {
	return []table_fixture{
		{0x0081, "Cc"},
		{0x200B, "Cf"},
		{0xf0000, "Co"},
		{0xdb80, "Cs"},
		{0x0236, "Ll"},
		{0x1d9d, "Lm"},
		{0x07cf, "Lo"},
		{0x1f8a, "Lt"},
		{0x03ff, "Lu"},
		{0x0bc1, "Mc"},
		{0x20df, "Me"},
		{0x07f0, "Mn"},
		{0x1bb2, "Nd"},
		{0x10147, "Nl"},
		{0x2478, "No"},
		{0xfe33, "Pc"},
		{0x2011, "Pd"},
		{0x301e, "Pe"},
		{0x2e03, "Pf"},
		{0x2e02, "Pi"},
		{0x0022, "Po"},
		{0x2770, "Ps"},
		{0x00a4, "Sc"},
		{0xa711, "Sk"},
		{0x25f9, "Sm"},
		{0x2108, "So"},
		{0x2028, "Zl"},
		{0x2029, "Zp"},
		{0x202f, "Zs"},

		{0x04aa, "L"},
		{0x0009, "C"},
		{0x1712, "M"},
		{0x0031, "N"},
		{0x00bb, "P"},
		{0x00a2, "S"},
		{0x00a0, "Z"},
		{0x0065, "LC"},

		{0x0378, "Cn"},
		{0x0378, "C"},
	}
}

func in_property_test() (tests []table_fixture) {
	return []table_fixture{
		{0x0046, "ASCII_Hex_Digit"},
		{0x200F, "Bidi_Control"},
		{0x2212, "Dash"},
		{0xE0001, "Deprecated"},
		{0x00B7, "Diacritic"},
		{0x30FE, "Extender"},
		{0xFF46, "Hex_Digit"},
		{0x2E17, "Hyphen"},
		{0x2FFB, "IDS_Binary_Operator"},
		{0x2FF3, "IDS_Trinary_Operator"},
		{0xFA6A, "Ideographic"},
		{0x200D, "Join_Control"},
		{0x0EC4, "Logical_Order_Exception"},
		{0x2FFFF, "Noncharacter_Code_Point"},
		{0x065E, "Other_Alphabetic"},
		{0x2065, "Other_Default_Ignorable_Code_Point"},
		{0x0BD7, "Other_Grapheme_Extend"},
		{0x0387, "Other_ID_Continue"},
		{0x212E, "Other_ID_Start"},
		{0x2094, "Other_Lowercase"},
		{0x2040, "Other_Math"},
		{0x216F, "Other_Uppercase"},
		{0x0027, "Pattern_Syntax"},
		{0x0020, "Pattern_White_Space"},
		{0x06DD, "Prepended_Concatenation_Mark"},
		{0x300D, "Quotation_Mark"},
		{0x2EF3, "Radical"},
		{0x1f1ff, "Regional_Indicator"},
		{0x061F, "STerm"},
		{0x061F, "Sentence_Terminal"},
		{0x2071, "Soft_Dotted"},
		{0x003A, "Terminal_Punctuation"},
		{0x9FC3, "Unified_Ideograph"},
		{0xFE0F, "Variation_Selector"},
		{0x0020, "White_Space"},
	}
}

// Test_Categories preserves the upstream behavior coverage.
func Test_Categories(t *testing.T) {
	for _, test := range in_category_test() {
		table, found := named_table(TABLE_KIND_CATEGORY, Name(test.Script))
		if !found {
			t.Fatal(test.Script, "not a known category")
		}
		if !Is(Range_Table_Handle(&table), Character(test.Rune)) {
			t.Errorf("IsCategory(%U, %s) = false, want true", test.Rune, test.Script)
		}
	}
}

// Test_Properties preserves the upstream behavior coverage.
func Test_Properties(t *testing.T) {
	for _, test := range in_property_test() {
		table, found := named_table(TABLE_KIND_PROPERTY, Name(test.Script))
		if !found {
			t.Fatal(test.Script, "not a known prop")
		}
		if !Is(Range_Table_Handle(&table), Character(test.Rune)) {
			t.Errorf("IsCategory(%U, %s) = false, want true", test.Rune, test.Script)
		}
	}
}
