// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style.
// License that can be found in the LICENSE file.

package strings_test

import (
	"fmt"
	"iter"
	"testing"
	"unsafe"

	invariant "local/james-orcales/shared/invariant/default"
	shared_io "local/james-orcales/shared/io"
	shared_bits "local/james-orcales/shared/math/bits"
	shared_prng "local/james-orcales/shared/random/prng"
	shared_slices "local/james-orcales/shared/slices"
	shared_strconv "local/james-orcales/shared/strconv"
	shared_strings "local/james-orcales/shared/strings"
	"local/james-orcales/shared/unicode/ucd"
	shared_utf8 "local/james-orcales/shared/unicode/utf8"
)

// TestMain registers the package invariant roots before the specification runs.
func TestMain(m *testing.M) {
	invariant.Run_Test_Main(m)
}

func standard_texts(parts shared_strings.Texts) (converted []string) {
	converted = make([]string, len(parts))
	for index, part := range parts {
		converted[index] = string(part)
	}
	return converted
}

func standard_field_texts(parts shared_strings.Field_Texts) (converted []string) {
	converted = make([]string, len(parts))
	for index, part := range parts {
		converted[index] = string(part)
	}
	return converted
}

func standard_sequence(sequence iter.Seq[shared_strings.Text]) (converted iter.Seq[string]) {
	return func(yield func(string) (continue_iteration bool)) {
		for value := range sequence {
			if !yield(string(value)) {
				return
			}
		}
	}
}

func standard_lines(source string) (sequence iter.Seq[string]) {
	return standard_sequence(shared_strings.Lines(shared_strings.Text(source)))
}

func standard_compare(left string, right string) (order int) {
	return int(shared_strings.Compare(shared_strings.Text(left), shared_strings.Text(right)))
}

func standard_clone(source string) (clone string) {
	return string(shared_strings.Clone(shared_strings.Text(source)))
}

func standard_contains(source string, separator string) (contained bool) {
	return bool(shared_strings.Contains(
		shared_strings.Text(source), shared_strings.Text(separator),
	))
}

func standard_contains_any(source string, characters string) (contained bool) {
	return bool(shared_strings.Contains_Any(
		shared_strings.Text(source), shared_strings.Text(characters),
	))
}

func standard_contains_function(
	source string, predicate func(rune) (matches bool),
) (contained bool) {
	return bool(shared_strings.Contains_Function(shared_strings.Text(source), predicate))
}

func standard_contains_rune(source string, character rune) (contained bool) {
	return bool(shared_strings.Contains_Rune(
		shared_strings.Text(source), shared_strings.Character(character),
	))
}

func standard_count(source string, separator string) (count int) {
	return int(shared_strings.Count(
		shared_strings.Text(source), shared_strings.Text(separator),
	))
}

func standard_cut(
	source string, separator string,
) (before string, after string, found bool) {
	port_before, port_after, port_found := shared_strings.Cut(
		shared_strings.Text(source), shared_strings.Text(separator),
	)
	return string(port_before), string(port_after), bool(port_found)
}

func standard_cut_prefix(source string, prefix string) (after string, found bool) {
	port_after, port_found := shared_strings.Cut_Prefix(
		shared_strings.Text(source), shared_strings.Text(prefix),
	)
	return string(port_after), bool(port_found)
}

func standard_cut_suffix(source string, suffix string) (before string, found bool) {
	port_before, port_found := shared_strings.Cut_Suffix(
		shared_strings.Text(source), shared_strings.Text(suffix),
	)
	return string(port_before), bool(port_found)
}

func standard_equal_fold(left string, right string) (equal bool) {
	return bool(shared_strings.Equal_Fold(
		shared_strings.Text(left), shared_strings.Text(right),
	))
}

func standard_fields(source string) (fields []string) {
	return standard_field_texts(shared_strings.Fields(shared_strings.Text(source)))
}

func standard_fields_function(
	source string, predicate func(rune) (matches bool),
) (fields []string) {
	return standard_field_texts(shared_strings.Fields_Function(
		shared_strings.Text(source), predicate,
	))
}

func standard_has_prefix(source string, prefix string) (present bool) {
	return bool(shared_strings.Has_Prefix(
		shared_strings.Text(source), shared_strings.Text(prefix),
	))
}

func standard_has_suffix(source string, suffix string) (present bool) {
	return bool(shared_strings.Has_Suffix(
		shared_strings.Text(source), shared_strings.Text(suffix),
	))
}

func standard_index(source string, separator string) (index int) {
	return int(shared_strings.Index(
		shared_strings.Text(source), shared_strings.Text(separator),
	))
}

func standard_last_index(source string, separator string) (index int) {
	return int(shared_strings.Last_Index(
		shared_strings.Text(source), shared_strings.Text(separator),
	))
}

func standard_index_any(source string, characters string) (index int) {
	return int(shared_strings.Index_Any(
		shared_strings.Text(source), shared_strings.Text(characters),
	))
}

func standard_last_index_any(source string, characters string) (index int) {
	return int(shared_strings.Last_Index_Any(
		shared_strings.Text(source), shared_strings.Text(characters),
	))
}

func standard_index_byte(source string, value byte) (index int) {
	return int(shared_strings.Index_Byte(
		shared_strings.Text(source), shared_strings.Byte(value),
	))
}

func standard_last_index_byte(source string, value byte) (index int) {
	return int(shared_strings.Last_Index_Byte(
		shared_strings.Text(source), shared_strings.Byte(value),
	))
}

func standard_index_function(
	source string, predicate func(rune) (matches bool),
) (index int) {
	return int(shared_strings.Index_Function(shared_strings.Text(source), predicate))
}

func standard_last_index_function(
	source string, predicate func(rune) (matches bool),
) (index int) {
	return int(shared_strings.Last_Index_Function(shared_strings.Text(source), predicate))
}

func standard_index_rune(source string, character rune) (index int) {
	return int(shared_strings.Index_Rune(
		shared_strings.Text(source), shared_strings.Character(character),
	))
}

func standard_join(parts []string, separator string) (joined string) {
	converted := make(shared_strings.Texts, len(parts))
	for index, part := range parts {
		converted[index] = shared_strings.Text(part)
	}
	return string(shared_strings.Join(converted, shared_strings.Text(separator)))
}

func standard_map(mapping func(rune) (mapped rune), source string) (mapped string) {
	return string(shared_strings.Map(mapping, shared_strings.Text(source)))
}

func standard_repeat(source string, count int) (repeated string) {
	if source == "" {
		if count >= 0 {
			return ""
		}
	}
	return string(shared_strings.Repeat(
		shared_strings.Text(source), shared_strings.Repeat_Count(count),
	))
}

func standard_replace(
	source string, old string, replacement string, count int,
) (replaced string) {
	return string(shared_strings.Replace(
		shared_strings.Text(source), shared_strings.Text(old),
		shared_strings.Text(replacement), shared_strings.Replacement_Count(count),
	))
}

func standard_replace_all(
	source string, old string, replacement string,
) (replaced string) {
	return string(shared_strings.Replace_All(
		shared_strings.Text(source), shared_strings.Text(old),
		shared_strings.Text(replacement),
	))
}

func standard_split(source string, separator string) (parts []string) {
	return standard_texts(shared_strings.Split(
		shared_strings.Text(source), shared_strings.Text(separator),
	))
}

func standard_split_after(source string, separator string) (parts []string) {
	return standard_texts(shared_strings.Split_After(
		shared_strings.Text(source), shared_strings.Text(separator),
	))
}

func standard_split_n(source string, separator string, limit int) (parts []string) {
	if limit > shared_strings.LIMIT_MAXIMUM {
		limit = shared_strings.LIMIT_MAXIMUM
	}
	return standard_texts(shared_strings.Split_N(
		shared_strings.Text(source), shared_strings.Text(separator),
		shared_strings.Limit(limit),
	))
}

func standard_split_after_n(
	source string, separator string, limit int,
) (parts []string) {
	if limit > shared_strings.LIMIT_MAXIMUM {
		limit = shared_strings.LIMIT_MAXIMUM
	}
	return standard_texts(shared_strings.Split_After_N(
		shared_strings.Text(source), shared_strings.Text(separator),
		shared_strings.Limit(limit),
	))
}

func standard_title(source string) (title string) {
	return string(shared_strings.Title(shared_strings.Text(source)))
}

func standard_to_upper(source string) (upper string) {
	return string(shared_strings.To_Upper(shared_strings.Text(source)))
}

func standard_to_lower(source string) (lower string) {
	return string(shared_strings.To_Lower(shared_strings.Text(source)))
}

func standard_to_title(source string) (title string) {
	return string(shared_strings.To_Title(shared_strings.Text(source)))
}

func standard_to_upper_special(special ucd.Special_Case, source string) (upper string) {
	return string(shared_strings.To_Upper_Special(special, shared_strings.Text(source)))
}

func standard_to_lower_special(special ucd.Special_Case, source string) (lower string) {
	return string(shared_strings.To_Lower_Special(special, shared_strings.Text(source)))
}

func standard_to_title_special(special ucd.Special_Case, source string) (title string) {
	return string(shared_strings.To_Title_Special(special, shared_strings.Text(source)))
}

func standard_turkish_case() (special ucd.Special_Case) {
	return ucd.Special_Case(ucd.Turkish_Case())
}

func standard_to_valid_utf8(source string, replacement string) (valid string) {
	return string(shared_strings.To_Valid_UTF8(
		shared_strings.Text(source), shared_strings.Text(replacement),
	))
}

func standard_trim(source string, cutset string) (trimmed string) {
	return string(shared_strings.Trim(
		shared_strings.Text(source), shared_strings.Text(cutset),
	))
}

func standard_trim_space(source string) (trimmed string) {
	return string(shared_strings.Trim_Space(shared_strings.Text(source)))
}

func standard_trim_left(source string, cutset string) (trimmed string) {
	return string(shared_strings.Trim_Left(
		shared_strings.Text(source), shared_strings.Text(cutset),
	))
}

func standard_trim_right(source string, cutset string) (trimmed string) {
	return string(shared_strings.Trim_Right(
		shared_strings.Text(source), shared_strings.Text(cutset),
	))
}

func standard_trim_prefix(source string, prefix string) (trimmed string) {
	return string(shared_strings.Trim_Prefix(
		shared_strings.Text(source), shared_strings.Text(prefix),
	))
}

func standard_trim_suffix(source string, suffix string) (trimmed string) {
	return string(shared_strings.Trim_Suffix(
		shared_strings.Text(source), shared_strings.Text(suffix),
	))
}

func standard_trim_left_function(
	source string, predicate func(rune) (matches bool),
) (trimmed string) {
	return string(shared_strings.Trim_Left_Function(shared_strings.Text(source), predicate))
}

func standard_trim_right_function(
	source string, predicate func(rune) (matches bool),
) (trimmed string) {
	return string(shared_strings.Trim_Right_Function(shared_strings.Text(source), predicate))
}

func standard_trim_function(
	source string, predicate func(rune) (matches bool),
) (trimmed string) {
	return string(shared_strings.Trim_Function(shared_strings.Text(source), predicate))
}

func standard_contains_sequence(
	sequence iter.Seq[string],
) (values []string) {
	return shared_slices.Collect[[]string](sequence)
}

func standard_split_sequence(source string, separator string) (sequence iter.Seq[string]) {
	return standard_sequence(shared_strings.Split_Sequence(
		shared_strings.Text(source), shared_strings.Text(separator),
	))
}

func standard_split_after_sequence(
	source string, separator string,
) (sequence iter.Seq[string]) {
	return standard_sequence(shared_strings.Split_After_Sequence(
		shared_strings.Text(source), shared_strings.Text(separator),
	))
}

func standard_fields_sequence(source string) (sequence iter.Seq[string]) {
	return standard_sequence(shared_strings.Fields_Sequence(shared_strings.Text(source)))
}

func standard_fields_function_sequence(
	source string, predicate func(rune) (matches bool),
) (sequence iter.Seq[string]) {
	return standard_sequence(shared_strings.Fields_Function_Sequence(
		shared_strings.Text(source), predicate,
	))
}

func standard_collect(t *testing.T, sequence iter.Seq[string]) (result_1 []string) {
	output := shared_slices.Collect[[]string](sequence)
	out1 := shared_slices.Collect[[]string](sequence)
	if !shared_slices.Equal(output, out1) {
		t.Fatalf("inconsistent sequence:\n%s\n%s", output, out1)
	}
	return output
}

func standard_is_space(character rune) (matches bool) {
	return bool(ucd.Is_Space(ucd.Character(character)))
}

func standard_is_digit(character rune) (matches bool) {
	return bool(ucd.Is_Digit(ucd.Character(character)))
}

func standard_is_upper(character rune) (matches bool) {
	return bool(ucd.Is_Upper(ucd.Character(character)))
}

func standard_is_latin(character rune) (matches bool) {
	return bool(ucd.Is_Script(ucd.Character(character), "Latin"))
}

func standard_decode_rune_in_string(source string) (character rune, size int) {
	decoded, decoded_size := shared_utf8.Decode_Character_Text(shared_utf8.Text(source))
	return rune(decoded), int(decoded_size)
}

func standard_builder_write(
	builder *shared_strings.Builder,
	content string,
) (count int, err error) {
	written, write_err := shared_io.Write_String(
		shared_strings.Builder_To_Stream(builder), content,
	)
	return int(written), write_err
}

func standard_builder_write_byte(
	builder *shared_strings.Builder,
	value byte,
) (err error) {
	return shared_io.Write_Byte(shared_strings.Builder_To_Stream(builder), value)
}

func standard_new_replacer(rules ...string) (replacer *shared_strings.Replacer) {
	pairs := make(shared_strings.Replacement_Pairs, len(rules)/2)
	for pair_index := range pairs {
		pairs[pair_index] = shared_strings.Replacement_Pair{
			shared_strings.Text(rules[pair_index*2]),
			shared_strings.Text(rules[pair_index*2+1]),
		}
	}
	return shared_strings.New_Replacer(pairs)
}

func standard_replacer_replace(
	replacer *shared_strings.Replacer,
	source string,
) (replaced string) {
	return string(shared_strings.Replacer_Replace(replacer, shared_strings.Text(source)))
}

func standard_replacer_write(
	replacer *shared_strings.Replacer,
	destination shared_io.Stream,
	source string,
) (count int, err error) {
	written, write_err := shared_strings.Replacer_Write_Text(
		replacer, destination, shared_strings.Text(source),
	)
	return int(written), write_err
}

type Lines_Test struct {
	A string
	B []string
}

func lines_tests() (value []Lines_Test) {
	return []Lines_Test{
		{A: "abc\nabc\n", B: []string{"abc\n", "abc\n"}},
		{A: "abc\r\nabc", B: []string{"abc\r\n", "abc"}},
		{A: "abc\r\n", B: []string{"abc\r\n"}},
		{A: "\nabc", B: []string{"\n", "abc"}},
		{A: "\nabc\n\n", B: []string{"\n", "abc\n", "\n"}},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Lines(t *testing.T) {
	for _, s := range lines_tests() {
		result := shared_slices.Collect[[]string](standard_lines(s.A))
		if !shared_slices.Equal(result, s.B) {
			t.Errorf(
				`slices.Collect(standard_lines(%q)) = %q; want %q`,
				s.A, result, s.B,
			)
		}
	}
}

const ABCD = "abcd"
const FACES = "☺☻☹"
const COMMAS = "1,2,3,4"
const DOTS = "1....2....3....4"

type Index_Test struct {
	S         string
	Separator string
	Output    int
}

func index_tests_basic() (value []Index_Test) {
	return []Index_Test{
		{"", "", 0},
		{"", "a", -1},
		{"", "foo", -1},
		{"fo", "foo", -1},
		{"foo", "foo", 0},
		{"oofofoofooo", "f", 2},
		{"oofofoofooo", "foo", 4},
		{"barfoobarfoo", "foo", 3},
		{"foo", "", 0},
		{"foo", "o", 1},
		{"abcABCabc", "A", 3},
		{"jrzm6jjhorimglljrea4w3rlgosts0w2gia17hno2td4qd1jz", "jz", 47},
		{"ekkuk5oft4eq0ocpacknhwouic1uua46unx12l37nioq9wbpnocqks6", "ks6", 52},
		{"999f2xmimunbuyew5vrkla9cpwhmxan8o98ec", "98ec", 33},
		{"9lpt9r98i04k8bz6c6dsrthb96bhi", "96bhi", 24},
		{"55u558eqfaod2r2gu42xxsu631xf0zobs5840vl", "5840vl", 33},
		// Cases with one byte strings - test special case in standard_index().
		{"", "a", -1},
		{"x", "a", -1},
		{"x", "x", 0},
		{"abc", "a", 0},
		{"abc", "b", 1},
		{"abc", "c", 2},
		{"abc", "x", -1},
		// Test special cases in standard_index() for short strings.
		{"", "ab", -1},
		{"bc", "ab", -1},
		{"ab", "ab", 0},
		{"xab", "ab", 1},
		{"xab"[:2], "ab", -1},
		{"", "abc", -1},
		{"xbc", "abc", -1},
		{"abc", "abc", 0},
		{"xabc", "abc", 1},
		{"xabc"[:3], "abc", -1},
		{"xabxc", "abc", -1},
		{"", "abcd", -1},
		{"xbcd", "abcd", -1},
		{"abcd", "abcd", 0},
		{"xabcd", "abcd", 1},
		{"xyabcd"[:5], "abcd", -1},
		{"xbcqq", "abcqq", -1},
		{"abcqq", "abcqq", 0},
		{"xabcqq", "abcqq", 1},
		{"xyabcqq"[:6], "abcqq", -1},
		{"xabxcqq", "abcqq", -1},
		{"xabcqxq", "abcqq", -1},
	}
}

func index_tests_decimal() (value []Index_Test) {
	return []Index_Test{
		{"", "01234567", -1},
		{"32145678", "01234567", -1},
		{"01234567", "01234567", 0},
		{"x01234567", "01234567", 1},
		{"x0123456x01234567", "01234567", 9},
		{"xx01234567"[:9], "01234567", -1},
		{"", "0123456789", -1},
		{"3214567844", "0123456789", -1},
		{"0123456789", "0123456789", 0},
		{"x0123456789", "0123456789", 1},
		{"x012345678x0123456789", "0123456789", 11},
		{"xyz0123456789"[:12], "0123456789", -1},
		{"x01234567x89", "0123456789", -1},
		{"", "0123456789012345", -1},
		{"3214567889012345", "0123456789012345", -1},
		{"0123456789012345", "0123456789012345", 0},
		{"x0123456789012345", "0123456789012345", 1},
		{"x012345678901234x0123456789012345", "0123456789012345", 17},
		{"", "01234567890123456789", -1},
		{"32145678890123456789", "01234567890123456789", -1},
		{"01234567890123456789", "01234567890123456789", 0},
		{"x01234567890123456789", "01234567890123456789", 1},
		{"x0123456789012345678x01234567890123456789", "01234567890123456789", 21},
		{"xyz01234567890123456789"[:22], "01234567890123456789", -1},
		{"", "0123456789012345678901234567890", -1},
		{
			"321456788901234567890123456789012345678911",
			"0123456789012345678901234567890",
			-1,
		},
		{"0123456789012345678901234567890", "0123456789012345678901234567890", 0},
		{"x0123456789012345678901234567890", "0123456789012345678901234567890", 1},
		{
			"x012345678901234567890123456789x0123456789012345678901234567890",
			"0123456789012345678901234567890",
			32,
		},
		{"xyz0123456789012345678901234567890"[:33], "0123456789012345678901234567890", -1},
		{"", "01234567890123456789012345678901", -1},
		{
			"32145678890123456789012345678901234567890211",
			"01234567890123456789012345678901",
			-1,
		},
		{"01234567890123456789012345678901", "01234567890123456789012345678901", 0},
		{"x01234567890123456789012345678901", "01234567890123456789012345678901", 1},
		{
			"x0123456789012345678901234567890x01234567890123456789012345678901",
			"01234567890123456789012345678901",
			33,
		},
		{
			"xyz01234567890123456789012345678901"[:34],
			"01234567890123456789012345678901",
			-1,
		},
	}
}

func index_tests_long() (value []Index_Test) {
	return []Index_Test{
		{
			"xxxxxx012345678901234567890123456789012345678901234567890123456789012",
			"012345678901234567890123456789012345678901234567890123456789012",
			6,
		},
		{"", "0123456789012345678901234567890123456789", -1},
		{
			"xx012345678901234567890123456789012345678901234567890123456789012",
			"0123456789012345678901234567890123456789",
			2,
		},
		{
			"xx012345678901234567890123456789012345678901234567890123456789012"[:41],
			"0123456789012345678901234567890123456789",
			-1,
		},
		{
			"xx012345678901234567890123456789012345678901234567890123456789012",
			"0123456789012345678901234567890123456xxx",
			-1,
		},
		{
			"xx012345678901234567890123456789012345678901234567890123456789012" +
				"0123456789012345678901234567890123456xxx",
			"0123456789012345678901234567890123456xxx",
			65,
		},
		// Test fallback to Rabin-Karp.
		{"oxoxoxoxoxoxoxoxoxoxoxoy", "oy", 22},
		{"oxoxoxoxoxoxoxoxoxoxoxox", "oy", -1},
		// Test fallback to IndexRune.
		{"oxoxoxoxoxoxoxoxoxoxox☺", "☺", 22},
		// Invalid UTF-8 byte sequence (must be longer than bytealg.MaxBruteForce to.
		// Test that we don't use IndexRune).
		{
			"xx012345678901234567890123456789012345678901234567890123456789012" +
				"0123456789012345678901234567890123456xxx\xed\x9f\xc0",
			"\xed\x9f\xc0",
			105,
		},
	}
}

func index_tests() (tests []Index_Test) {
	tests = append(tests, index_tests_basic()...)
	tests = append(tests, index_tests_decimal()...)
	return append(tests, index_tests_long()...)
}
func last_index_tests() (value []Index_Test) {
	return []Index_Test{
		{"", "", 0},
		{"", "a", -1},
		{"", "foo", -1},
		{"fo", "foo", -1},
		{"foo", "foo", 0},
		{"foo", "f", 0},
		{"oofofoofooo", "f", 7},
		{"oofofoofooo", "foo", 7},
		{"barfoobarfoo", "foo", 9},
		{"foo", "", 3},
		{"foo", "o", 2},
		{"abcABCabc", "A", 3},
		{"abcABCabc", "a", 6},
	}
}
func index_any_tests() (value []Index_Test) {
	return []Index_Test{
		{"", "", -1},
		{"", "a", -1},
		{"", "abc", -1},
		{"a", "", -1},
		{"a", "a", 0},
		{"\x80", "\xffb", 0},
		{"aaa", "a", 0},
		{"abc", "xyz", -1},
		{"abc", "xcz", 2},
		{"ab☺c", "x☺yz", 2},
		{"a☺b☻c☹d", "cx", len("a☺b☻")},
		{"a☺b☻c☹d", "uvw☻xyz", len("a☺b")},
		{"aRegExp*", ".(|)*+?^$[]", 7},
		{DOTS + DOTS + DOTS, " ", -1},
		{"012abcba210", "\xffb", 4},
		{"012\x80bcb\x80210", "\xffb", 3},
		{"0123456\xcf\x80abc", "\xcfb\x80", 10},
	}
}
func last_index_any_tests() (value []Index_Test) {
	return []Index_Test{
		{"", "", -1},
		{"", "a", -1},
		{"", "abc", -1},
		{"a", "", -1},
		{"a", "a", 0},
		{"\x80", "\xffb", 0},
		{"aaa", "a", 2},
		{"abc", "xyz", -1},
		{"abc", "ab", 1},
		{"ab☺c", "x☺yz", 2},
		{"a☺b☻c☹d", "cx", len("a☺b☻")},
		{"a☺b☻c☹d", "uvw☻xyz", len("a☺b")},
		{"a.RegExp*", ".(|)*+?^$[]", 8},
		{DOTS + DOTS + DOTS, " ", -1},
		{"012abcba210", "\xffb", 6},
		{"012\x80bcb\x80210", "\xffb", 7},
		{"0123456\xcf\x80abc", "\xcfb\x80", 10},
	}
}

// Execute f on each test case.  funcName should be the name of f; it's used.
// In failure reports.
func run_index_tests(
	t *testing.T,
	f func(s, separator string) (result_2 int),
	function_name string,
	test_cases []Index_Test,
) {
	for _, test := range test_cases {
		actual := f(test.S, test.Separator)
		if actual != test.Output {
			t.Errorf(
				"%s(%q,%q) = %v; want %v",
				function_name, test.S, test.Separator, actual, test.Output,
			)
		}
	}
}

// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Index(t *testing.T) {
	run_index_tests(t, standard_index, "Index", index_tests())
}

// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Last_Index(t *testing.T) {
	run_index_tests(t, standard_last_index, "LastIndex", last_index_tests())
}

// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Index_Any(t *testing.T) {
	run_index_tests(t, standard_index_any, "IndexAny", index_any_tests())
}

// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Last_Index_Any(t *testing.T) {
	run_index_tests(t, standard_last_index_any, "LastIndexAny", last_index_any_tests())
}

// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Index_Byte(t *testing.T) {
	for _, tt := range index_tests() {
		if len(tt.Separator) != 1 {
			continue
		}
		position := standard_index_byte(tt.S, tt.Separator[0])
		if position != tt.Output {
			t.Errorf(
				`standard_index_byte(%q, %q) = %v; want %v`,
				tt.S, tt.Separator[0], position, tt.Output,
			)
		}
	}
}

// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Last_Index_Byte(t *testing.T) {
	test_cases := []Index_Test{
		{"", "q", -1},
		{"abcdef", "q", -1},
		{"abcdefabcdef", "a", len("abcdef")},      // Something in the middle.
		{"abcdefabcdef", "f", len("abcdefabcde")}, // Last byte.
		{"zabcdefabcdef", "z", 0},                 // First byte.
		{"a☺b☻c☹d", "b", len("a☺")},               // Non-ascii.
	}
	for _, test := range test_cases {
		actual := standard_last_index_byte(test.S, test.Separator[0])
		if actual != test.Output {
			t.Errorf(
				"standard_last_index_byte(%q,%c) = %v; want %v",
				test.S,
				test.Separator[0],
				actual,
				test.Output,
			)
		}
	}
}

func simple_index(s, separator string) (result_3 int) {
	n_count := len(separator)
	for i := n_count; i <= len(s); i++ {
		if s[i-n_count:i] == separator {
			return i - n_count
		}
	}
	return -1
}

// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Index_Random(t *testing.T) {
	const CHARS = "abcdefghijklmnopqrstuvwxyz0123456789"
	random := shared_prng.New(1)
	for times_index := 0; times_index < 10; times_index++ {
		// Varied sizes exercise each short-search threshold.
		for string_size := 5 + shared_prng.Generator_Below(
			&random, 5,
		); string_size < 140; string_size += 10 {
			s1 := make([]byte, string_size)
			for i := range s1 {
				s1[i] = CHARS[shared_prng.Generator_Below(&random, len(CHARS))]
			}
			s := string(s1)
			for i_index := 0; i_index < 50; i_index++ {
				begin := shared_prng.Generator_Below(&random, len(s)+1)
				end := begin + shared_prng.Generator_Below(&random, len(s)+1-begin)
				separator := s[begin:end]
				if i_index%4 == 0 {
					position := shared_prng.Generator_Below(
						&random, len(separator)+1,
					)
					separator = separator[:position] +
						"A" + separator[position:]
				}
				want := simple_index(s, separator)
				result := standard_index(s, separator)
				if result != want {
					t.Errorf(
						"standard_index(%s,%s) = %d; want %d",
						s, separator, result, want,
					)
				}
			}
		}
	}
}

type Index_Rune_Test struct {
	In   string
	Rune rune
	Want int
}

func index_rune_tests() (tests []Index_Rune_Test) {
	return []Index_Rune_Test{
		{"", 'a', -1},
		{"", '☺', -1},
		{"foo", '☹', -1},
		{"foo", 'o', 1},
		{"foo☺bar", '☺', 3},
		{"foo☺☻☹bar", '☹', 9},
		{"a A x", 'A', 2},
		{"some_text=some_value", '=', 9},
		{"☺a", 'a', 3},
		{"a☻☺b", '☺', 4},

		// RuneError should match any invalid UTF-8 byte sequence.
		{"�", '�', 0},
		{"\xff", '�', 0},
		{"☻x�", '�', len("☻x")},
		{"☻x\xe2\x98", '�', len("☻x")},
		{"☻x\xe2\x98�", '�', len("☻x")},
		{"☻x\xe2\x98x", '�', len("☻x")},

		// Invalid rune values should never match.
		{"a☺b☻c☹d\xe2\x98�\xff�\xed\xa0\x80", -1, -1},
		{"a☺b☻c☹d\xe2\x98�\xff�\xed\xa0\x80", 0xD800, -1}, // Surrogate pair.
		{"a☺b☻c☹d\xe2\x98�\xff�\xed\xa0\x80", rune(shared_utf8.RUNE_MAX) + 1, -1},

		// 2 bytes.
		{"ӆ", 'ӆ', 0},
		{"a", 'ӆ', -1},
		{"  ӆ", 'ӆ', 2},
		{"  a", 'ӆ', -1},
		{standard_repeat("ц", 64) + "ӆ", 'ӆ', 128}, // Test cutover.
		{standard_repeat("Ꙁ", 64) + "Ꚁ", '䚀', -1},

		// 3 bytes.
		{"Ꚁ", 'Ꚁ', 0},
		{"a", 'Ꚁ', -1},
		{"  Ꚁ", 'Ꚁ', 2},
		{"  a", 'Ꚁ', -1},
		{standard_repeat("Ꙁ", 64) + "Ꚁ", 'Ꚁ', 192}, // Test cutover.
		{standard_repeat("𡋀", 64) + "𡌀", '𣌀', -1},

		// 4 bytes.
		{"𡌀", '𡌀', 0},
		{"a", '𡌀', -1},
		{"  𡌀", '𡌀', 2},
		{"  a", '𡌀', -1},
		{standard_repeat("𡋀", 64) + "𡌀", '𡌀', 256}, // Test cutover.
		{standard_repeat("𡋀", 64), '𡌀', -1},

		// Test the cutover to bytealg.IndexString when it is triggered in.
		// The middle of rune that contains consecutive runs of equal bytes.
		{"aaaaaKKKK\U000bc104", '\U000bc104', 17}, // Cutover: (n + 16) / 8.
		{"aaaaaKKKK鄄", '鄄', 17},
		{"aaKKKKKa\U000bc104", '\U000bc104', 18}, // Cutover: 4 + n>>4.
		{"aaKKKKKa鄄", '鄄', 18},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Index_Rune(t *testing.T) {
	for _, tt := range index_rune_tests() {
		if got := standard_index_rune(tt.In, tt.Rune); got != tt.Want {
			t.Errorf(
				"standard_index_rune(%q, %d) = %v; want %v",
				tt.In, tt.Rune, got, tt.Want,
			)
		}
	}

	// Make sure we trigger the cutover and string(rune) conversion.
	haystack := "test" + standard_repeat("𡋀", 32) + "𡌀"
	allocs := testing.AllocsPerRun(1000, func() {
		if i := standard_index_rune(haystack, 's'); i != 2 {
			t.Fatalf("'s' at %d; want 2", i)
		}
		if i := standard_index_rune(haystack, '𡌀'); i != 132 {
			t.Fatalf("'𡌀' at %d; want 4", i)
		}
	})
	if allocs != 0 {
		if testing.CoverMode() == "" {
			t.Errorf("expected no allocations, got %f", allocs)
		}
	}
}

const BENCHMARK_STRING = "some_text=some☺value"

func Benchmark_Standard_Library_Index_Rune(b *testing.B) {
	if got := standard_index_rune(BENCHMARK_STRING, '☺'); got != 14 {
		b.Fatalf("wrong index: expected 14, got=%d", got)
	}
	for i_index := 0; i_index < b.N; i_index++ {
		standard_index_rune(BENCHMARK_STRING, '☺')
	}
}
func benchmark_long_string() (value string) {
	return standard_repeat(" ", 100) + BENCHMARK_STRING
}

func Benchmark_Standard_Library_Index_Rune_Long_String(b *testing.B) {
	if got := standard_index_rune(benchmark_long_string(), '☺'); got != 114 {
		b.Fatalf("wrong index: expected 114, got=%d", got)
	}
	for i_index := 0; i_index < b.N; i_index++ {
		standard_index_rune(benchmark_long_string(), '☺')
	}
}

func Benchmark_Standard_Library_Index_Rune_Fast_Path(b *testing.B) {
	if got := standard_index_rune(BENCHMARK_STRING, 'v'); got != 17 {
		b.Fatalf("wrong index: expected 17, got=%d", got)
	}
	for i_index := 0; i_index < b.N; i_index++ {
		standard_index_rune(BENCHMARK_STRING, 'v')
	}
}

func Benchmark_Standard_Library_Index(b *testing.B) {
	if got := standard_index(BENCHMARK_STRING, "v"); got != 17 {
		b.Fatalf("wrong index: expected 17, got=%d", got)
	}
	for i_index := 0; i_index < b.N; i_index++ {
		standard_index(BENCHMARK_STRING, "v")
	}
}

func Benchmark_Standard_Library_Last_Index(b *testing.B) {
	if got := standard_index(BENCHMARK_STRING, "v"); got != 17 {
		b.Fatalf("wrong index: expected 17, got=%d", got)
	}
	for i_index := 0; i_index < b.N; i_index++ {
		standard_last_index(BENCHMARK_STRING, "v")
	}
}

func Benchmark_Standard_Library_Index_Byte(b *testing.B) {
	if got := standard_index_byte(BENCHMARK_STRING, 'v'); got != 17 {
		b.Fatalf("wrong index: expected 17, got=%d", got)
	}
	for i_index := 0; i_index < b.N; i_index++ {
		standard_index_byte(BENCHMARK_STRING, 'v')
	}
}

type Split_Test struct {
	S         string
	Separator string
	N         int
	A         []string
}

func splittests() (value []Split_Test) {
	return []Split_Test{
		{"", "", -1, []string{}},
		{ABCD, "", 2, []string{"a", "bcd"}},
		{ABCD, "", 4, []string{"a", "b", "c", "d"}},
		{ABCD, "", -1, []string{"a", "b", "c", "d"}},
		{FACES, "", -1, []string{"☺", "☻", "☹"}},
		{FACES, "", 3, []string{"☺", "☻", "☹"}},
		{FACES, "", 17, []string{"☺", "☻", "☹"}},
		{"☺�☹", "", -1, []string{"☺", "�", "☹"}},
		{ABCD, "a", 0, nil},
		{ABCD, "a", -1, []string{"", "bcd"}},
		{ABCD, "z", -1, []string{"abcd"}},
		{COMMAS, ",", -1, []string{"1", "2", "3", "4"}},
		{DOTS, "...", -1, []string{"1", ".2", ".3", ".4"}},
		{FACES, "☹", -1, []string{"☺☻", ""}},
		{FACES, "~", -1, []string{FACES}},
		{"1 2 3 4", " ", 3, []string{"1", "2", "3 4"}},
		{"1 2", " ", 3, []string{"1", "2"}},
		{"", "T", shared_bits.INTEGER_MAXIMUM / 4, []string{""}},
		{"\xff-\xff", "", -1, []string{"\xff", "-", "\xff"}},
		{"\xff-\xff", "-", -1, []string{"\xff", "\xff"}},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Split(t *testing.T) {
	for _, tt := range splittests() {
		a := standard_split_n(tt.S, tt.Separator, tt.N)
		if !shared_slices.Equal(a, tt.A) {
			t.Errorf(
				"standard_split(%q, %q, %d) = %v; want %v",
				tt.S, tt.Separator, tt.N, a, tt.A,
			)
			continue
		}
		if tt.N < 0 {
			a2 := shared_slices.Collect[[]string](
				standard_split_sequence(tt.S, tt.Separator),
			)
			if !shared_slices.Equal(a2, tt.A) {
				t.Errorf(
					`standard_collect(standard_split_sequence(%q, %q)) = `+
						`%v; want %v`,
					tt.S, tt.Separator, a2, tt.A,
				)
			}
		}
		if tt.N == 0 {
			continue
		}
		s := standard_join(a, tt.Separator)
		if s != tt.S {
			t.Errorf(
				"standard_join(standard_split(%q, %q, %d), %q) = %q",
				tt.S, tt.Separator, tt.N, tt.Separator, s,
			)
		}
		if tt.N < 0 {
			b := standard_split(tt.S, tt.Separator)
			if !shared_slices.Equal(a, b) {
				t.Errorf(
					"Split disagrees with Split_N(%q, %q, %d) = "+
						"%v; want %v",
					tt.S, tt.Separator, tt.N, b, a,
				)
			}
		}
	}
}
func splitaftertests() (value []Split_Test) {
	return []Split_Test{
		{ABCD, "a", -1, []string{"a", "bcd"}},
		{ABCD, "z", -1, []string{"abcd"}},
		{ABCD, "", -1, []string{"a", "b", "c", "d"}},
		{COMMAS, ",", -1, []string{"1,", "2,", "3,", "4"}},
		{DOTS, "...", -1, []string{"1...", ".2...", ".3...", ".4"}},
		{FACES, "☹", -1, []string{"☺☻☹", ""}},
		{FACES, "~", -1, []string{FACES}},
		{FACES, "", -1, []string{"☺", "☻", "☹"}},
		{"1 2 3 4", " ", 3, []string{"1 ", "2 ", "3 4"}},
		{"1 2 3", " ", 3, []string{"1 ", "2 ", "3"}},
		{"1 2", " ", 3, []string{"1 ", "2"}},
		{"123", "", 2, []string{"1", "23"}},
		{"123", "", 17, []string{"1", "2", "3"}},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Split_After(t *testing.T) {
	for _, tt := range splitaftertests() {
		a := standard_split_after_n(tt.S, tt.Separator, tt.N)
		if !shared_slices.Equal(a, tt.A) {
			t.Errorf(
				`standard_split(%q, %q, %d) = %v; want %v`,
				tt.S, tt.Separator, tt.N, a, tt.A,
			)
			continue
		}
		if tt.N < 0 {
			a2 := shared_slices.Collect[[]string](
				standard_split_after_sequence(tt.S, tt.Separator),
			)
			if !shared_slices.Equal(a2, tt.A) {
				t.Errorf(
					`standard_collect(standard_split_after_sequence(%q, %q)) = `+
						`%v; want %v`,
					tt.S, tt.Separator, a2, tt.A,
				)
			}
		}
		s := standard_join(a, "")
		if s != tt.S {
			t.Errorf(
				`standard_join(standard_split(%q, %q, %d), %q) = %q`,
				tt.S, tt.Separator, tt.N, tt.Separator, s,
			)
		}
		if tt.N < 0 {
			b := standard_split_after(tt.S, tt.Separator)
			if !shared_slices.Equal(a, b) {
				t.Errorf(
					"Split_After disagrees with Split_After_N(%q, %q, %d) = "+
						"%v; want %v",
					tt.S, tt.Separator, tt.N, b, a,
				)
			}
		}
	}
}

type Fields_Test struct {
	S string
	A []string
}

func fieldstests() (value []Fields_Test) {
	return []Fields_Test{
		{"", []string{}},
		{" ", []string{}},
		{" \t ", []string{}},
		{"\u2000", []string{}},
		{"  abc  ", []string{"abc"}},
		{"1 2 3 4", []string{"1", "2", "3", "4"}},
		{"1  2  3  4", []string{"1", "2", "3", "4"}},
		{"1\t\t2\t\t3\t4", []string{"1", "2", "3", "4"}},
		{"1\u20002\u20013\u20024", []string{"1", "2", "3", "4"}},
		{"\u2000\u2001\u2002", []string{}},
		{"\n™\t™\n", []string{"™", "™"}},
		{"\n\u20001™2\u2000 \u2001 ™", []string{"1™2", "™"}},
		{"\n1\uFFFD \uFFFD2\u20003\uFFFD4", []string{"1\uFFFD", "\uFFFD2", "3\uFFFD4"}},
		{"1\xFF\u2000\xFF2\xFF \xFF", []string{"1\xFF", "\xFF2\xFF", "\xFF"}},
		{FACES, []string{FACES}},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Fields(t *testing.T) {
	for _, tt := range fieldstests() {
		a := standard_fields(tt.S)
		if !shared_slices.Equal(a, tt.A) {
			t.Errorf("standard_fields(%q) = %v; want %v", tt.S, a, tt.A)
			continue
		}
		a2 := standard_collect(t, standard_fields_sequence(tt.S))
		if !shared_slices.Equal(a2, tt.A) {
			t.Errorf(
				`standard_collect(standard_fields_sequence(%q)) = %v; want %v`,
				tt.S, a2, tt.A,
			)
		}
	}
}
func fields_function_tests() (value []Fields_Test) {
	return []Fields_Test{
		{"", []string{}},
		{"XX", []string{}},
		{"XXhiXXX", []string{"hi"}},
		{"aXXbXXXcX", []string{"a", "b", "c"}},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Fields_Function(t *testing.T) {
	for _, tt := range fieldstests() {
		a := standard_fields_function(tt.S, standard_is_space)
		if !shared_slices.Equal(a, tt.A) {
			t.Errorf(
				"standard_fields_function(%q, standard_is_space) = %v; want %v",
				tt.S, a, tt.A,
			)
			continue
		}
	}
	field_separator := func(c rune) (result_4 bool) { return c == 'X' }
	for _, tt := range fields_function_tests() {
		a := standard_fields_function(tt.S, field_separator)
		if !shared_slices.Equal(a, tt.A) {
			t.Errorf("standard_fields_function(%q) = %v, want %v", tt.S, a, tt.A)
		}
		a2 := standard_collect(t, standard_fields_function_sequence(tt.S, field_separator))
		if !shared_slices.Equal(a2, tt.A) {
			t.Errorf(
				`standard_collect(standard_fields_function_sequence(%q)) = `+
					`%v; want %v`,
				tt.S, a2, tt.A,
			)
		}
	}
}

// Test case for any function which accepts and returns a single string.
type String_Test struct {
	In, Output string
}

// Execute f on each test case.  funcName should be the name of f; it's used.
// In failure reports.
func run_string_tests(
	t *testing.T,
	f func(string) (result_5 string),
	function_name string,
	test_cases []String_Test,
) {
	for _, tc := range test_cases {
		actual := f(tc.In)
		if actual != tc.Output {
			t.Errorf("%s(%q) = %q; want %q", function_name, tc.In, actual, tc.Output)
		}
	}
}
func upper_tests() (value []String_Test) {
	return []String_Test{
		{"", ""},
		{"ONLYUPPER", "ONLYUPPER"},
		{"abc", "ABC"},
		{"AbC123", "ABC123"},
		{"azAZ09_", "AZAZ09_"},
		{"longStrinGwitHmixofsmaLLandcAps", "LONGSTRINGWITHMIXOFSMALLANDCAPS"},
		{
			"RENAN BASTOS 93 AOSDAJDJAIDJAIDAJIaidsjjaidijadsjiadjiOOKKO",
			"RENAN BASTOS 93 AOSDAJDJAIDJAIDAJIAIDSJJAIDIJADSJIADJIOOKKO",
		},
		{
			"long\u0250string\u0250with\u0250nonascii\u2C6Fchars",
			"LONG\u2C6FSTRING\u2C6FWITH\u2C6FNONASCII\u2C6FCHARS",
		},
		{"\u0250\u0250\u0250\u0250\u0250", "\u2C6F\u2C6F\u2C6F\u2C6F\u2C6F"},
		{"a\u0080\U0010FFFF", "A\u0080\U0010FFFF"},
	}
}
func lower_tests() (value []String_Test) {
	return []String_Test{
		{"", ""},
		{"abc", "abc"},
		{"AbC123", "abc123"},
		{"azAZ09_", "azaz09_"},
		{"longStrinGwitHmixofsmaLLandcAps", "longstringwithmixofsmallandcaps"},
		{
			"renan bastos 93 AOSDAJDJAIDJAIDAJIaidsjjaidijadsjiadjiOOKKO",
			"renan bastos 93 aosdajdjaidjaidajiaidsjjaidijadsjiadjiookko",
		},
		{
			"LONG\u2C6FSTRING\u2C6FWITH\u2C6FNONASCII\u2C6FCHARS",
			"long\u0250string\u0250with\u0250nonascii\u0250chars",
		},
		{"\u2C6D\u2C6D\u2C6D\u2C6D\u2C6D", "\u0251\u0251\u0251\u0251\u0251"},
		{"A\u0080\U0010FFFF", "a\u0080\U0010FFFF"},
	}
}

const SPACE = "\t\v\r\f\n\u0085\u00a0\u2000\u3000"

func trim_space_tests() (value []String_Test) {
	return []String_Test{
		{"", ""},
		{"abc", "abc"},
		{SPACE + "abc" + SPACE, "abc"},
		{" ", ""},
		{" \t\r\n \t\t\r\r\n\n ", ""},
		{" \t\r\n x\t\t\r\r\n\n ", "x"},
		{" \u2000\t\r\n x\t\t\r\r\ny\n \u3000", "x\t\t\r\r\ny"},
		{"1 \t\r\n2", "1 \t\r\n2"},
		{" x\x80", "x\x80"},
		{" x\xc0", "x\xc0"},
		{"x \xc0\xc0 ", "x \xc0\xc0"},
		{"x \xc0", "x \xc0"},
		{"x \xc0 ", "x \xc0"},
		{"x \xc0\xc0 ", "x \xc0\xc0"},
		{"x ☺\xc0\xc0 ", "x ☺\xc0\xc0"},
		{"x ☺ ", "x ☺"},
	}
}

func ten_runes(ch rune) (result_6 string) {
	r := make([]rune, 10)
	for i := range r {
		r[i] = ch
	}
	return string(r)
}

// The self-inverse mapping verifies both directions with one table.
func rot13(r rune) (result_7 rune) {
	const STEP = rune(13)
	if r >= 'a' {
		if r <= 'z' {
			return ((r - 'a' + STEP) % 26) + 'a'
		}
	}
	if r >= 'A' {
		if r <= 'Z' {
			return ((r - 'A' + STEP) % 26) + 'A'
		}
	}
	return r
}

// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Map(t *testing.T) {
	// Run a couple of awful growth/shrinkage tests.
	a := ten_runes('a')
	// 1. Grow. This triggers two reallocations in Map.
	rune_max := func(rune) (result_8 rune) { return rune(ucd.RUNE_MAX) }
	m := standard_map(rune_max, a)
	expect := ten_runes(rune(ucd.RUNE_MAX))
	if m != expect {
		t.Errorf("growing: expected %q got %q", expect, m)
	}

	// 2. Shrink.
	rune_min := func(rune) (result_9 rune) { return 'a' }
	m = standard_map(rune_min, ten_runes(rune(ucd.RUNE_MAX)))
	expect = a
	if m != expect {
		t.Errorf("shrinking: expected %q got %q", expect, m)
	}

	// 3. Rot13.
	m = standard_map(rot13, "a to zed")
	expect = "n gb mrq"
	if m != expect {
		t.Errorf("rot13: expected %q got %q", expect, m)
	}

	// 4. Rot13^2.
	m = standard_map(rot13, standard_map(rot13, "a to zed"))
	expect = "a to zed"
	if m != expect {
		t.Errorf("rot13: expected %q got %q", expect, m)
	}

	// 5. Drop.
	drop_not_latin := func(r rune) (result_10 rune) {
		if standard_is_latin(r) {
			return r
		}
		return -1
	}
	m = standard_map(drop_not_latin, "Hello, 세계")
	expect = "Hello"
	if m != expect {
		t.Errorf("drop: expected %q got %q", expect, m)
	}
	test_standard_map_identity_and_encoding(t)
}

func test_standard_map_identity_and_encoding(t *testing.T) {
	// Identity must not allocate new string storage.
	identity := func(r rune) (result_11 rune) {
		return r
	}
	original := "Input string that we expect not to be copied."
	mapped := standard_map(identity, original)
	if unsafe.StringData(original) != unsafe.StringData(mapped) {
		t.Error("unexpected copy during identity map")
	}

	// Invalid UTF-8 must enter the mapping as RuneError.
	replace_not_latin := func(r rune) (result_12 rune) {
		if standard_is_latin(r) {
			return r
		}
		return rune(shared_utf8.REPLACEMENT_CHARACTER)
	}
	mapped = standard_map(replace_not_latin, "Hello\255World")
	want := "Hello\uFFFDWorld"
	if mapped != want {
		t.Errorf("replace invalid sequence: expected %q got %q", want, mapped)
	}

	// Boundary characters verify every UTF-8 encoding path.
	encode := func(r rune) (result_13 rune) {
		switch r {
		case rune(shared_utf8.CHARACTER_SELF):
			return rune(ucd.RUNE_MAX)
		case rune(ucd.RUNE_MAX):
			return rune(shared_utf8.CHARACTER_SELF)
		}
		return r
	}
	s := string(rune(shared_utf8.CHARACTER_SELF)) +
		string(rune(shared_utf8.RUNE_MAX))
	r := string(rune(shared_utf8.RUNE_MAX)) +
		string(rune(shared_utf8.CHARACTER_SELF))
	mapped = standard_map(encode, s)
	if mapped != r {
		t.Errorf("encoding not handled correctly: expected %q got %q", r, mapped)
	}
	mapped = standard_map(encode, r)
	if mapped != s {
		t.Errorf("encoding not handled correctly: expected %q got %q", s, mapped)
	}

	// Spaced input verifies mapping at the front, middle, and back.
	trim_spaces := func(r rune) (result_14 rune) {
		if standard_is_space(r) {
			return -1
		}
		return r
	}
	mapped = standard_map(trim_spaces, "   abc    123   ")
	want = "abc123"
	if mapped != want {
		t.Errorf("trimSpaces: expected %q got %q", want, mapped)
	}
}

// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_To_Upper(t *testing.T) {
	run_string_tests(t, standard_to_upper, "ToUpper", upper_tests())
}

// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_To_Lower(t *testing.T) {
	run_string_tests(t, standard_to_lower, "ToLower", lower_tests())
}
func to_valid_utf8_tests() (value []struct {
	In     string
	Repl   string
	Output string
}) {
	return []struct {
		In     string
		Repl   string
		Output string
	}{
		{"", "\uFFFD", ""},
		{"abc", "\uFFFD", "abc"},
		{"\uFDDD", "\uFFFD", "\uFDDD"},
		{"a\xffb", "\uFFFD", "a\uFFFDb"},
		{"a\xffb\uFFFD", "X", "aXb\uFFFD"},
		{"a☺\xffb☺\xC0\xAFc☺\xff", "", "a☺b☺c☺"},
		{"a☺\xffb☺\xC0\xAFc☺\xff", "日本語", "a☺日本語b☺日本語c☺日本語"},
		{"\xC0\xAF", "\uFFFD", "\uFFFD"},
		{"\xE0\x80\xAF", "\uFFFD", "\uFFFD"},
		{"\xed\xa0\x80", "abc", "abc"},
		{"\xed\xbf\xbf", "\uFFFD", "\uFFFD"},
		{"\xF0\x80\x80\xaf", "☺", "☺"},
		{"\xF8\x80\x80\x80\xAF", "\uFFFD", "\uFFFD"},
		{"\xFC\x80\x80\x80\x80\xAF", "\uFFFD", "\uFFFD"},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_To_Valid_UTF8(t *testing.T) {
	for _, tc := range to_valid_utf8_tests() {
		got := standard_to_valid_utf8(tc.In, tc.Repl)
		if got != tc.Output {
			t.Errorf(
				"standard_to_valid_utf8(%q, %q) = %q; want %q",
				tc.In, tc.Repl, got, tc.Output,
			)
		}
	}
}

func Benchmark_Standard_Library_To_Upper(b *testing.B) {
	for _, tc := range upper_tests() {
		b.Run(tc.In, func(b *testing.B) {
			for i_index := 0; i_index < b.N; i_index++ {
				actual := standard_to_upper(tc.In)
				if actual != tc.Output {
					b.Errorf(
						"standard_to_upper(%q) = %q; want %q",
						tc.In, actual, tc.Output,
					)
				}
			}
		})
	}
}

func Benchmark_Standard_Library_To_Lower(b *testing.B) {
	for _, tc := range lower_tests() {
		b.Run(tc.In, func(b *testing.B) {
			for i_index := 0; i_index < b.N; i_index++ {
				actual := standard_to_lower(tc.In)
				if actual != tc.Output {
					b.Errorf(
						"standard_to_lower(%q) = %q; want %q",
						tc.In, actual, tc.Output,
					)
				}
			}
		})
	}
}

func Benchmark_Standard_Library_Map_No_Changes(b *testing.B) {
	identity := func(r rune) (result_15 rune) {
		return r
	}
	for i_index := 0; i_index < b.N; i_index++ {
		standard_map(identity, "Some string that won't be modified.")
	}
}

// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Special_Case(t *testing.T) {
	lower := "abcçdefgğhıijklmnoöprsştuüvyz"
	upper := "ABCÇDEFGĞHIİJKLMNOÖPRSŞTUÜVYZ"
	u := standard_to_upper_special(standard_turkish_case(), upper)
	if u != upper {
		t.Errorf("Upper(upper) is %s not %s", u, upper)
	}
	u = standard_to_upper_special(standard_turkish_case(), lower)
	if u != upper {
		t.Errorf("Upper(lower) is %s not %s", u, upper)
	}
	l := standard_to_lower_special(standard_turkish_case(), lower)
	if l != lower {
		t.Errorf("Lower(lower) is %s not %s", l, lower)
	}
	l = standard_to_lower_special(standard_turkish_case(), upper)
	if l != lower {
		t.Errorf("Lower(upper) is %s not %s", l, lower)
	}
}

// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Trim_Space(t *testing.T) {
	run_string_tests(t, standard_trim_space, "TrimSpace", trim_space_tests())
}
func trim_tests() (value []struct {
	F               string
	In, Arg, Output string
}) {
	return []struct {
		F               string
		In, Arg, Output string
	}{
		{"Trim", "abba", "a", "bb"},
		{"Trim", "abba", "ab", ""},
		{"TrimLeft", "abba", "ab", ""},
		{"TrimRight", "abba", "ab", ""},
		{"TrimLeft", "abba", "a", "bba"},
		{"TrimLeft", "abba", "b", "abba"},
		{"TrimRight", "abba", "a", "abb"},
		{"TrimRight", "abba", "b", "abba"},
		{"Trim", "<tag>", "<>", "tag"},
		{"Trim", "* listitem", " *", "listitem"},
		{"Trim", `"quote"`, `"`, "quote"},
		{"Trim", "\u2C6F\u2C6F\u0250\u0250\u2C6F\u2C6F", "\u2C6F", "\u0250\u0250"},
		{"Trim", "\x80test\xff", "\xff", "test"},
		{"Trim", " Ġ ", " ", "Ġ"},
		{"Trim", " Ġİ0", "0 ", "Ġİ"},
		// Empty string tests.
		{"Trim", "abba", "", "abba"},
		{"Trim", "", "123", ""},
		{"Trim", "", "", ""},
		{"TrimLeft", "abba", "", "abba"},
		{"TrimLeft", "", "123", ""},
		{"TrimLeft", "", "", ""},
		{"TrimRight", "abba", "", "abba"},
		{"TrimRight", "", "123", ""},
		{"TrimRight", "", "", ""},
		{"TrimRight", "☺\xc0", "☺", "☺\xc0"},
		{"TrimPrefix", "aabb", "a", "abb"},
		{"TrimPrefix", "aabb", "b", "aabb"},
		{"TrimSuffix", "aabb", "a", "aabb"},
		{"TrimSuffix", "aabb", "b", "aab"},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Trim(t *testing.T) {
	for _, tc := range trim_tests() {
		name := tc.F
		var f func(string, string) (result_16 string)
		switch name {
		case "Trim":
			f = standard_trim
		case "TrimLeft":
			f = standard_trim_left
		case "TrimRight":
			f = standard_trim_right
		case "TrimPrefix":
			f = standard_trim_prefix
		case "TrimSuffix":
			f = standard_trim_suffix
		default:
			t.Errorf("Undefined trim function %s", name)
		}
		actual := f(tc.In, tc.Arg)
		if actual != tc.Output {
			t.Errorf("%s(%q, %q) = %q; want %q", name, tc.In, tc.Arg, actual, tc.Output)
		}
	}
}

func Benchmark_Standard_Library_Trim(b *testing.B) {
	b.ReportAllocs()

	for i_index := 0; i_index < b.N; i_index++ {
		for _, tc := range trim_tests() {
			name := tc.F
			var f func(string, string) (result_17 string)
			switch name {
			case "Trim":
				f = standard_trim
			case "TrimLeft":
				f = standard_trim_left
			case "TrimRight":
				f = standard_trim_right
			case "TrimPrefix":
				f = standard_trim_prefix
			case "TrimSuffix":
				f = standard_trim_suffix
			default:
				b.Errorf("Undefined trim function %s", name)
			}
			actual := f(tc.In, tc.Arg)
			if actual != tc.Output {
				b.Errorf(
					"%s(%q, %q) = %q; want %q",
					name, tc.In, tc.Arg, actual, tc.Output,
				)
			}
		}
	}
}

func Benchmark_Standard_Library_To_Valid_UTF8(b *testing.B) {
	tests := []struct {
		Name  string
		Input string
	}{
		{"Valid", "typical"},
		{"InvalidASCII", "foo\xffbar"},
		{"InvalidNonASCII", "日本語\xff日本語"},
	}
	replacement := "\uFFFD"
	b.ResetTimer()
	for _, test := range tests {
		b.Run(test.Name, func(b *testing.B) {
			for i_index := 0; i_index < b.N; i_index++ {
				standard_to_valid_utf8(test.Input, replacement)
			}
		})
	}
}

type predicate struct {
	F    func(rune) (result_18 bool)
	Name string
}

func is_space() (value predicate) {
	return predicate{F: standard_is_space, Name: "IsSpace"}
}
func is_digit() (value predicate) {
	return predicate{F: standard_is_digit, Name: "IsDigit"}
}
func is_upper() (value predicate) {
	return predicate{F: standard_is_upper, Name: "IsUpper"}
}
func is_valid_rune() (value predicate) {
	return predicate{
		F: func(r rune) (result_19 bool) {
			return r != rune(shared_utf8.REPLACEMENT_CHARACTER)
		},
		Name: "IsValidRune",
	}
}

func not(p predicate) (result_20 predicate) {
	return predicate{
		F: func(r rune) (result_21 bool) {
			return !p.F(r)
		},
		Name: "not " + p.Name,
	}
}
func trim_function_tests() (value []struct {
	F            predicate
	In           string
	Trim_Output  string
	Left_Output  string
	Right_Output string
}) {
	return []struct {
		F            predicate
		In           string
		Trim_Output  string
		Left_Output  string
		Right_Output string
	}{
		{is_space(), SPACE + " hello " + SPACE,
			"hello",
			"hello " + SPACE,
			SPACE + " hello"},
		{is_digit(), "\u0e50\u0e5212hello34\u0e50\u0e51",
			"hello",
			"hello34\u0e50\u0e51",
			"\u0e50\u0e5212hello"},
		{is_upper(), "\u2C6F\u2C6F\u2C6F\u2C6FABCDhelloEF\u2C6F\u2C6FGH\u2C6F\u2C6F",
			"hello",
			"helloEF\u2C6F\u2C6FGH\u2C6F\u2C6F",
			"\u2C6F\u2C6F\u2C6F\u2C6FABCDhello"},
		{not(is_space()), "hello" + SPACE + "hello",
			SPACE,
			SPACE + "hello",
			"hello" + SPACE},
		{not(is_digit()), "hello\u0e50\u0e521234\u0e50\u0e51helo",
			"\u0e50\u0e521234\u0e50\u0e51",
			"\u0e50\u0e521234\u0e50\u0e51helo",
			"hello\u0e50\u0e521234\u0e50\u0e51"},
		{is_valid_rune(), "ab\xc0a\xc0cd",
			"\xc0a\xc0",
			"\xc0a\xc0cd",
			"ab\xc0a\xc0"},
		{not(is_valid_rune()), "\xc0a\xc0",
			"a",
			"a\xc0",
			"\xc0a"},
		{is_space(), "",
			"",
			"",
			""},
		{is_space(), " ",
			"",
			"",
			""},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Trim_Function(t *testing.T) {
	for _, tc := range trim_function_tests() {
		trimmers := []struct {
			Name   string
			Trim   func(s string, f func(r rune) (result_23 bool)) (result_22 string)
			Output string
		}{
			{"TrimFunc", standard_trim_function, tc.Trim_Output},
			{"TrimLeftFunc", standard_trim_left_function, tc.Left_Output},
			{"TrimRightFunc", standard_trim_right_function, tc.Right_Output},
		}
		for _, trimmer := range trimmers {
			actual := trimmer.Trim(tc.In, tc.F.F)
			if actual != trimmer.Output {
				t.Errorf(
					"%s(%q, %q) = %q; want %q",
					trimmer.Name, tc.In, tc.F.Name, actual, trimmer.Output,
				)
			}
		}
	}
}
func index_function_tests() (value []struct {
	In          string
	F           predicate
	First, Last int
}) {
	return []struct {
		In          string
		F           predicate
		First, Last int
	}{
		{"", is_valid_rune(), -1, -1},
		{"abc", is_digit(), -1, -1},
		{"0123", is_digit(), 0, 3},
		{"a1b", is_digit(), 1, 1},
		{SPACE, is_space(), 0, len(SPACE) - 3}, // Last rune in space is 3 bytes.
		{"\u0e50\u0e5212hello34\u0e50\u0e51", is_digit(), 0, 18},
		{
			"\u2C6F\u2C6F\u2C6F\u2C6FABCDhelloEF\u2C6F\u2C6FGH\u2C6F\u2C6F",
			is_upper(),
			0,
			34,
		},
		{"12\u0e50\u0e52hello34\u0e50\u0e51", not(is_digit()), 8, 12},

		// Tests of invalid UTF-8.
		{"\x801", is_digit(), 1, 1},
		{"\x80abc", is_digit(), -1, -1},
		{"\xc0a\xc0", is_valid_rune(), 1, 1},
		{"\xc0a\xc0", not(is_valid_rune()), 0, 2},
		{"\xc0☺\xc0", not(is_valid_rune()), 0, 4},
		{"\xc0☺\xc0\xc0", not(is_valid_rune()), 0, 5},
		{"ab\xc0a\xc0cd", not(is_valid_rune()), 2, 4},
		{"a\xe0\x80cd", not(is_valid_rune()), 1, 2},
		{"\x80\x80\x80\x80", not(is_valid_rune()), 0, 3},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Index_Function(t *testing.T) {
	for _, tc := range index_function_tests() {
		first := standard_index_function(tc.In, tc.F.F)
		if first != tc.First {
			t.Errorf(
				"standard_index_function(%q, %s) = %d; want %d",
				tc.In, tc.F.Name, first, tc.First,
			)
		}
		last := standard_last_index_function(tc.In, tc.F.F)
		if last != tc.Last {
			t.Errorf(
				"standard_last_index_function(%q, %s) = %d; want %d",
				tc.In, tc.F.Name, last, tc.Last,
			)
		}
	}
}

func equal(m string, s1, s2 string, t *testing.T) (result_24 bool) {
	if s1 == s2 {
		return true
	}
	e1 := standard_split(s1, "")
	e2 := standard_split(s2, "")
	for i, c1 := range e1 {
		if i >= len(e2) {
			break
		}
		r1, _ := standard_decode_rune_in_string(c1)
		r2, _ := standard_decode_rune_in_string(e2[i])
		if r1 != r2 {
			t.Errorf("%s diff at %d: U+%04X U+%04X", m, i, r1, r2)
		}
	}
	return false
}

// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Case_Consistency(t *testing.T) {
	// The bounded port uses the upstream short set because the full string exceeds Text.
	number_runes_count := 1000
	a := make([]rune, number_runes_count)
	for i := range a {
		a[i] = rune(i)
	}
	s := string(a)
	// Convert the cases.
	upper := standard_to_upper(s)
	lower := standard_to_lower(s)

	// Consistency checks.
	if n_count := int(shared_utf8.Character_Count_Text(
		shared_utf8.Text(upper),
	)); n_count != number_runes_count {
		t.Error("rune count wrong in upper:", n_count)
	}
	if n_count := int(shared_utf8.Character_Count_Text(
		shared_utf8.Text(lower),
	)); n_count != number_runes_count {
		t.Error("rune count wrong in lower:", n_count)
	}
	if !equal("standard_to_upper(upper)", standard_to_upper(upper), upper, t) {
		t.Error("standard_to_upper(upper) consistency fail")
	}
	if !equal("standard_to_lower(lower)", standard_to_lower(lower), lower, t) {
		t.Error("standard_to_lower(lower) consistency fail")
	}
	/*
		  These fail because of non-one-to-oneness of the data, such as multiple
		  upper case 'I' mapping to 'i'.  We comment them out but keep them for
		  interest.
		  For instance: CAPITAL LETTER I WITH DOT ABOVE:
			unicode.standard_to_upper(unicode.standard_to_lower('\u0130')) != '\u0130'

		if !equal("standard_to_upper(lower)", standard_to_upper(lower), upper, t) {
			t.Error("standard_to_upper(lower) consistency fail");
		}
		if !equal("standard_to_lower(upper)", standard_to_lower(upper), lower, t) {
			t.Error("standard_to_lower(upper) consistency fail");
		}
	*/
}
func long_string() (value string) {
	return "a" + string(make([]byte, shared_strings.TEXT_SIZE_MAXIMUM/2-2)) + "z"
}
func long_spaces() (result_25 string) {
	b := make([]byte, 200)
	for i := range b {
		b[i] = ' '
	}
	return string(b)
}
func Repeat_Tests() (value []struct {
	In, Output string
	Count      int
}) {
	return []struct {
		In, Output string
		Count      int
	}{
		{"", "", 0},
		{"", "", 1},
		{"", "", 2},
		{"-", "", 0},
		{"-", "-", 1},
		{"-", "----------", 10},
		{"abc ", "abc abc abc ", 3},
		{" ", " ", 1},
		{"--", "----", 2},
		{"===", "======", 2},
		{"000", "000000000", 3},
		{"\t\t\t\t", "\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t", 4},
		{" ", long_spaces(), len(long_spaces())},
		// The bounded port tests its largest result instead of an out-of-domain chunk.
		{string(rune(0)), string(make([]byte, shared_strings.TEXT_SIZE_MAXIMUM)),
			shared_strings.TEXT_SIZE_MAXIMUM},
		{long_string(), long_string() + long_string(), 2},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Repeat(t *testing.T) {
	for _, tt := range Repeat_Tests() {
		a := standard_repeat(tt.In, tt.Count)
		if !equal("standard_repeat(s)", a, tt.Output, t) {
			t.Errorf(
				"standard_repeat(%v, %d) = %v; want %v",
				tt.In, tt.Count, a, tt.Output,
			)
			continue
		}
	}
}

func repeat(s string, count int) (err error) {
	defer func() {
		if r := recover(); r != nil {
			switch v := r.(type) {
			case error:
				err = v
			default:
				err = fmt.Errorf("%s", v)
			}
		}
	}()

	standard_repeat(s, count)

	return err
}

// See Issue golang.org/issue/16237.
// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Repeat_Catches_Overflow(t *testing.T) {
	type test_case struct {
		S          string
		Count      int
		Error_Text string
	}

	run_test_cases := func(prefix string, tests []test_case) {
		for i, tt := range tests {
			err := repeat(tt.S, tt.Count)
			if tt.Error_Text == "" {
				if err != nil {
					t.Errorf("#%d panicked %v", i, err)
				}
				continue
			}

			if err == nil {
				t.Errorf("%s#%d did not panic", prefix, i)
			}
		}
	}

	const INT_MAX = int(^uint(0) >> 1)

	run_test_cases("", []test_case{
		0: {"--", -2147483647, "negative"},
		1: {"", INT_MAX, ""},
		2: {"-", 10, ""},
		3: {"gopher", 0, ""},
		4: {"-", -1, "negative"},
		5: {"--", -102, "negative"},
		6: {
			string(make([]byte, shared_bits.WORD_8_MAXIMUM)),
			int((^uint(0))/uint(shared_bits.WORD_8_MAXIMUM) + 1),
			"overflow",
		},
	})

	const IS64_BIT = 1<<(^uintptr(0)>>63)/2 != 0
	if !IS64_BIT {
		return
	}

	run_test_cases("64-bit", []test_case{
		0: {"-", INT_MAX, "out of range"},
	})
}

func runes_equal(a, b []rune) (result_26 bool) {
	if len(a) != len(b) {
		return false
	}
	for i, r := range a {
		if r != b[i] {
			return false
		}
	}
	return true
}
func Runes_Tests() (value []struct {
	In     string
	Output []rune
	Lossy  bool
}) {
	return []struct {
		In     string
		Output []rune
		Lossy  bool
	}{
		{"", []rune{}, false},
		{" ", []rune{32}, false},
		{"ABC", []rune{65, 66, 67}, false},
		{"abc", []rune{97, 98, 99}, false},
		{"\u65e5\u672c\u8a9e", []rune{26085, 26412, 35486}, false},
		{"ab\x80c", []rune{97, 98, 0xFFFD, 99}, true},
		{"ab\xc0c", []rune{97, 98, 0xFFFD, 99}, true},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Runes(t *testing.T) {
	for _, tt := range Runes_Tests() {
		a := []rune(tt.In)
		if !runes_equal(a, tt.Output) {
			t.Errorf("[]rune(%q) = %v; want %v", tt.In, a, tt.Output)
			continue
		}
		if !tt.Lossy {
			// Can only test reassembly if we didn't lose information.
			s := string(a)
			if s != tt.In {
				t.Errorf("string([]rune(%q)) = %x; want %x", tt.In, s, tt.In)
			}
		}
	}
}

// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Read_Byte(t *testing.T) {
	test_strings := []string{"", ABCD, FACES, COMMAS}
	for _, s := range test_strings {
		reader := shared_strings.New_Reader(shared_strings.Text(s))
		if e := shared_strings.Reader_Unread_Byte(reader); e == nil {
			t.Errorf("Unreading %q at beginning: expected error", s)
		}
		var result []byte
		for range shared_bits.INTEGER_MAXIMUM {
			b, e := shared_strings.Reader_Read_Byte(reader)
			if e == shared_io.Stream_EOF {
				break
			}
			if e != nil {
				t.Errorf("Reading %q: %s", s, e)
				break
			}
			result = append(result, byte(b))
			// Unread and read again.
			e = shared_strings.Reader_Unread_Byte(reader)
			if e != nil {
				t.Errorf("Unreading %q: %s", s, e)
				break
			}
			b1, e := shared_strings.Reader_Read_Byte(reader)
			if e != nil {
				t.Errorf("Reading %q after unreading: %s", s, e)
				break
			}
			if b1 != b {
				t.Errorf(
					"Reading %q after unreading: want byte %q, got %q",
					s, b, b1,
				)
				break
			}
		}
		if string(result) != s {
			t.Errorf("Reader(%q).ReadByte() produced %q", s, result)
		}
	}
}

// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Read_Rune(t *testing.T) {
	test_strings := []string{"", ABCD, FACES, COMMAS}
	for _, s := range test_strings {
		reader := shared_strings.New_Reader(shared_strings.Text(s))
		if e := shared_strings.Reader_Unread_Character(reader); e == nil {
			t.Errorf("Unreading %q at beginning: expected error", s)
		}
		result := ""
		for range shared_bits.INTEGER_MAXIMUM {
			r, z, e := shared_strings.Reader_Read_Character(reader)
			if e == shared_io.Stream_EOF {
				break
			}
			if e != nil {
				t.Errorf("Reading %q: %s", s, e)
				break
			}
			result += string(rune(r))
			// Unread and read again.
			e = shared_strings.Reader_Unread_Character(reader)
			if e != nil {
				t.Errorf("Unreading %q: %s", s, e)
				break
			}
			r1, z1, e := shared_strings.Reader_Read_Character(reader)
			if e != nil {
				t.Errorf("Reading %q after unreading: %s", s, e)
				break
			}
			if r1 != r {
				t.Errorf(
					"Reading %q after unreading: want rune %q, got %q",
					s, r, r1,
				)
				break
			}
			if z1 != z {
				t.Errorf(
					"Reading %q after unreading: want size %d, got %d",
					s, z, z1,
				)
				break
			}
		}
		if result != s {
			t.Errorf("Reader(%q).ReadRune() produced %q", s, result)
		}
	}
}
func Unread_Rune_Error_Tests() (value []struct {
	Name string
	F    func(*shared_strings.Reader)
}) {
	return []struct {
		Name string
		F    func(*shared_strings.Reader)
	}{
		{"Read", func(r *shared_strings.Reader) {
			shared_io.Read(shared_strings.Reader_To_Stream(r), []byte{0})
		}},
		{"ReadByte", func(r *shared_strings.Reader) {
			shared_strings.Reader_Read_Byte(r)
		}},
		{"UnreadRune", func(r *shared_strings.Reader) {
			shared_strings.Reader_Unread_Character(r)
		}},
		{"Seek", func(r *shared_strings.Reader) {
			shared_io.Seek(
				shared_strings.Reader_To_Stream(r), 0, shared_io.SEEK_FROM_CURRENT,
			)
		}},
		{"WriteTo", func(r *shared_strings.Reader) {
			shared_strings.Reader_Write_To(
				r, shared_io.Discard_To_Stream(&shared_io.Stream_Discard{}),
			)
		}},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Unread_Rune_Error(t *testing.T) {
	for _, tt := range Unread_Rune_Error_Tests() {
		reader := shared_strings.New_Reader("0123456789")
		if _, _, err := shared_strings.Reader_Read_Character(reader); err != nil {
			// Should not happen.
			t.Fatal(err)
		}
		tt.F(reader)
		err := shared_strings.Reader_Unread_Character(reader)
		if err == nil {
			t.Errorf("Unreading after %s: expected error", tt.Name)
		}
	}
}
func Replace_Tests() (value []struct {
	In       string
	Old, New string
	N        int
	Output   string
}) {
	return []struct {
		In       string
		Old, New string
		N        int
		Output   string
	}{
		{"hello", "l", "L", 0, "hello"},
		{"hello", "l", "L", -1, "heLLo"},
		{"hello", "x", "X", -1, "hello"},
		{"", "x", "X", -1, ""},
		{"radar", "r", "<r>", -1, "<r>ada<r>"},
		{"", "", "<>", -1, "<>"},
		{"banana", "a", "<>", -1, "b<>n<>n<>"},
		{"banana", "a", "<>", 1, "b<>nana"},
		{"banana", "a", "<>", 1000, "b<>n<>n<>"},
		{"banana", "an", "<>", -1, "b<><>a"},
		{"banana", "ana", "<>", -1, "b<>na"},
		{"banana", "", "<>", -1, "<>b<>a<>n<>a<>n<>a<>"},
		{"banana", "", "<>", 10, "<>b<>a<>n<>a<>n<>a<>"},
		{"banana", "", "<>", 6, "<>b<>a<>n<>a<>n<>a"},
		{"banana", "", "<>", 5, "<>b<>a<>n<>a<>na"},
		{"banana", "", "<>", 1, "<>banana"},
		{"banana", "a", "a", -1, "banana"},
		{"banana", "a", "a", 1, "banana"},
		{"☺☻☹", "", "<>", -1, "<>☺<>☻<>☹<>"},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Replace(t *testing.T) {
	for _, tt := range Replace_Tests() {
		if s := standard_replace(tt.In, tt.Old, tt.New, tt.N); s != tt.Output {
			t.Errorf(
				"standard_replace(%q, %q, %q, %d) = %q, want %q",
				tt.In, tt.Old, tt.New, tt.N, s, tt.Output,
			)
		}
		if tt.N == -1 {
			s := standard_replace_all(tt.In, tt.Old, tt.New)
			if s != tt.Output {
				t.Errorf(
					"standard_replace_all(%q, %q, %q) = %q, want %q",
					tt.In, tt.Old, tt.New, s, tt.Output,
				)
			}
		}
	}
}

func Fuzz_Standard_Library_Replace(f *testing.F) {
	for _, tt := range Replace_Tests() {
		f.Add(tt.In, tt.Old, tt.New, tt.N)
	}
	f.Fuzz(func(t *testing.T, in, old, new string, n int) {
		different_implementation := func(in, old, new string, n int) (result_27 string) {
			var output shared_strings.Builder
			stream := shared_strings.Builder_To_Stream(&output)
			write_text := func(text string) {
				shared_io.Write(stream, []byte(text))
			}
			if n < 0 {
				n = shared_bits.INTEGER_MAXIMUM
			}
			for i := 0; i < len(in); {
				if n == 0 {
					write_text(in[i:])
					break
				}
				if standard_has_prefix(in[i:], old) {
					write_text(new)
					i += len(old)
					n--
					if len(old) != 0 {
						continue
					}
					if i == len(in) {
						break
					}
				}
				if len(old) == 0 {
					_, size := standard_decode_rune_in_string(in[i:])
					write_text(in[i : i+size])
					i += size
				} else {
					shared_io.Write_Byte(stream, in[i])
					i++
				}
			}
			if len(old) == 0 {
				if n != 0 {
					write_text(new)
				}
			}
			return string(shared_strings.Builder_Text(&output))
		}
		simple := different_implementation(in, old, new, n)
		replace := standard_replace(in, old, new, n)
		if simple != replace {
			t.Errorf(
				"The implementations do not match %q != %q for "+
					"Replace(%q, %q, %q, %d)",
				simple, replace, in, old, new, n,
			)
		}
	})
}

func Benchmark_Standard_Library_Replace(b *testing.B) {
	for _, tt := range Replace_Tests() {
		description := fmt.Sprintf("%q %q %q %d", tt.In, tt.Old, tt.New, tt.N)
		b.Run(description, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				standard_replace(tt.In, tt.Old, tt.New, tt.N)
			}
		})
	}
}
func Title_Tests() (value []struct {
	In, Output string
}) {
	return []struct {
		In, Output string
	}{
		{"", ""},
		{"a", "A"},
		{" aaa aaa aaa ", " Aaa Aaa Aaa "},
		{" Aaa Aaa Aaa ", " Aaa Aaa Aaa "},
		{"123a456", "123a456"},
		{"double-blind", "Double-Blind"},
		{"ÿøû", "Ÿøû"},
		{"with_underscore", "With_underscore"},
		{"unicode \xe2\x80\xa8 line separator", "Unicode \xe2\x80\xa8 Line Separator"},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Title(t *testing.T) {
	for _, tt := range Title_Tests() {
		if s := standard_title(tt.In); s != tt.Output {
			t.Errorf("standard_title(%q) = %q, want %q", tt.In, s, tt.Output)
		}
	}
}
func Contains_Tests() (value []struct {
	Str, Substr string
	Expected    bool
}) {
	return []struct {
		Str, Substr string
		Expected    bool
	}{
		{"abc", "bc", true},
		{"abc", "bcd", false},
		{"abc", "", true},
		{"", "a", false},

		{"xxxxxx", "01", false},
		{"01xxxx", "01", true},
		{"xx01xx", "01", true},
		{"xxxx01", "01", true},
		{"01xxxxx"[1:], "01", false},
		{"xxxxx01"[:6], "01", false},
		{"xxxxxxx", "012", false},
		{"012xxxx", "012", true},
		{"xx012xx", "012", true},
		{"xxxx012", "012", true},
		{"012xxxxx"[1:], "012", false},
		{"xxxxx012"[:7], "012", false},
		{"xxxxxxxx", "0123", false},
		{"0123xxxx", "0123", true},
		{"xx0123xx", "0123", true},
		{"xxxx0123", "0123", true},
		{"0123xxxxx"[1:], "0123", false},
		{"xxxxx0123"[:8], "0123", false},
		{"xxxxxxxxx", "01234", false},
		{"01234xxxx", "01234", true},
		{"xx01234xx", "01234", true},
		{"xxxx01234", "01234", true},
		{"01234xxxxx"[1:], "01234", false},
		{"xxxxx01234"[:9], "01234", false},
		{"xxxxxxxxxxxx", "01234567", false},
		{"01234567xxxx", "01234567", true},
		{"xx01234567xx", "01234567", true},
		{"xxxx01234567", "01234567", true},
		{"01234567xxxxx"[1:], "01234567", false},
		{"xxxxx01234567"[:12], "01234567", false},
		{"xxxxxxxxxxxxx", "012345678", false},
		{"012345678xxxx", "012345678", true},
		{"xx012345678xx", "012345678", true},
		{"xxxx012345678", "012345678", true},
		{"012345678xxxxx"[1:], "012345678", false},
		{"xxxxx012345678"[:13], "012345678", false},
		{"xxxxxxxxxxxxxxxxxxxx", "0123456789ABCDEF", false},
		{"0123456789ABCDEFxxxx", "0123456789ABCDEF", true},
		{"xx0123456789ABCDEFxx", "0123456789ABCDEF", true},
		{"xxxx0123456789ABCDEF", "0123456789ABCDEF", true},
		{"0123456789ABCDEFxxxxx"[1:], "0123456789ABCDEF", false},
		{"xxxxx0123456789ABCDEF"[:20], "0123456789ABCDEF", false},
		{"xxxxxxxxxxxxxxxxxxxxx", "0123456789ABCDEFG", false},
		{"0123456789ABCDEFGxxxx", "0123456789ABCDEFG", true},
		{"xx0123456789ABCDEFGxx", "0123456789ABCDEFG", true},
		{"xxxx0123456789ABCDEFG", "0123456789ABCDEFG", true},
		{"0123456789ABCDEFGxxxxx"[1:], "0123456789ABCDEFG", false},
		{"xxxxx0123456789ABCDEFG"[:21], "0123456789ABCDEFG", false},

		{"xx01x", "012", false},                             // 3.
		{"xx0123x", "01234", false},                         // 5-7.
		{"xx01234567x", "012345678", false},                 // 9-15.
		{"xx0123456789ABCDEFx", "0123456789ABCDEFG", false}, // 17-31, issue 15679.
	}
}

// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Contains(t *testing.T) {
	for _, ct := range Contains_Tests() {
		if standard_contains(ct.Str, ct.Substr) != ct.Expected {
			t.Errorf("standard_contains(%s, %s) = %v, want %v",
				ct.Str, ct.Substr, !ct.Expected, ct.Expected)
		}
	}
}
func Contains_Any_Tests() (value []struct {
	Str, Substr string
	Expected    bool
}) {
	return []struct {
		Str, Substr string
		Expected    bool
	}{
		{"", "", false},
		{"", "a", false},
		{"", "abc", false},
		{"a", "", false},
		{"a", "a", true},
		{"aaa", "a", true},
		{"abc", "xyz", false},
		{"abc", "xcz", true},
		{"a☺b☻c☹d", "uvw☻xyz", true},
		{"aRegExp*", ".(|)*+?^$[]", true},
		{DOTS + DOTS + DOTS, " ", false},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Contains_Any(t *testing.T) {
	for _, ct := range Contains_Any_Tests() {
		if standard_contains_any(ct.Str, ct.Substr) != ct.Expected {
			t.Errorf("standard_contains_any(%s, %s) = %v, want %v",
				ct.Str, ct.Substr, !ct.Expected, ct.Expected)
		}
	}
}
func Contains_Rune_Tests() (value []struct {
	Str      string
	R        rune
	Expected bool
}) {
	return []struct {
		Str      string
		R        rune
		Expected bool
	}{
		{"", 'a', false},
		{"a", 'a', true},
		{"aaa", 'a', true},
		{"abc", 'y', false},
		{"abc", 'c', true},
		{"a☺b☻c☹d", 'x', false},
		{"a☺b☻c☹d", '☻', true},
		{"aRegExp*", '*', true},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Contains_Rune(t *testing.T) {
	for _, ct := range Contains_Rune_Tests() {
		if standard_contains_rune(ct.Str, ct.R) != ct.Expected {
			t.Errorf("standard_contains_rune(%q, %q) = %v, want %v",
				ct.Str, ct.R, !ct.Expected, ct.Expected)
		}
	}
}

// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Contains_Function(t *testing.T) {
	for _, ct := range Contains_Rune_Tests() {
		if standard_contains_function(ct.Str, func(r rune) (result_28 bool) {
			return ct.R == r
		}) != ct.Expected {
			t.Errorf("standard_contains_function(%q, func(%q)) = %v, want %v",
				ct.Str, ct.R, !ct.Expected, ct.Expected)
		}
	}
}
func Equal_Fold_Tests() (value []struct {
	S, T   string
	Output bool
}) {
	return []struct {
		S, T   string
		Output bool
	}{
		{"abc", "abc", true},
		{"ABcd", "ABcd", true},
		{"123abc", "123ABC", true},
		{"αβδ", "ΑΒΔ", true},
		{"abc", "xyz", false},
		{"abc", "XYZ", false},
		{"abcdefghijk", "abcdefghijX", false},
		{"abcdefghijk", "abcdefghij\u212A", true},
		{"abcdefghijK", "abcdefghij\u212A", true},
		{"abcdefghijkz", "abcdefghij\u212Ay", false},
		{"abcdefghijKz", "abcdefghij\u212Ay", false},
		{"1", "2", false},
		{"utf-8", "US-ASCII", false},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Equal_Fold(t *testing.T) {
	for _, tt := range Equal_Fold_Tests() {
		if output := standard_equal_fold(tt.S, tt.T); output != tt.Output {
			t.Errorf(
				"standard_equal_fold(%#q, %#q) = %v, want %v",
				tt.S, tt.T, output, tt.Output,
			)
		}
		if output := standard_equal_fold(tt.T, tt.S); output != tt.Output {
			t.Errorf(
				"standard_equal_fold(%#q, %#q) = %v, want %v",
				tt.T, tt.S, output, tt.Output,
			)
		}
	}
}

func Benchmark_Standard_Library_Equal_Fold(b *testing.B) {
	b.Run("Tests", func(b *testing.B) {
		for i_index := 0; i_index < b.N; i_index++ {
			for _, tt := range Equal_Fold_Tests() {
				if output := standard_equal_fold(tt.S, tt.T); output != tt.Output {
					b.Fatal("wrong result")
				}
			}
		}
	})

	const S1 = "abcdefghijKz"
	const S2 = "abcDefGhijKz"

	b.Run("ASCII", func(b *testing.B) {
		for i_index := 0; i_index < b.N; i_index++ {
			standard_equal_fold(S1, S2)
		}
	})

	b.Run("UnicodePrefix", func(b *testing.B) {
		for i_index := 0; i_index < b.N; i_index++ {
			standard_equal_fold("αβδ"+S1, "ΑΒΔ"+S2)
		}
	})

	b.Run("UnicodeSuffix", func(b *testing.B) {
		for i_index := 0; i_index < b.N; i_index++ {
			standard_equal_fold(S1+"αβδ", S2+"ΑΒΔ")
		}
	})
}
func Count_Tests() (value []struct {
	S, Separator string
	Num          int
}) {
	return []struct {
		S, Separator string
		Num          int
	}{
		{"", "", 1},
		{"", "notempty", 0},
		{"notempty", "", 9},
		{"smaller", "not smaller", 0},
		{"12345678987654321", "6", 2},
		{"611161116", "6", 3},
		{"notequal", "NotEqual", 0},
		{"equal", "equal", 1},
		{"abc1231231123q", "123", 3},
		{"11111", "11", 2},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Count(t *testing.T) {
	for _, tt := range Count_Tests() {
		if number := standard_count(tt.S, tt.Separator); number != tt.Num {
			t.Errorf(
				"standard_count(%q, %q) = %d, want %d",
				tt.S, tt.Separator, number, tt.Num,
			)
		}
	}
}
func cut_tests() (value []struct {
	S, Separator  string
	Before, After string
	Found         bool
}) {
	return []struct {
		S, Separator  string
		Before, After string
		Found         bool
	}{
		{"abc", "b", "a", "c", true},
		{"abc", "a", "", "bc", true},
		{"abc", "c", "ab", "", true},
		{"abc", "abc", "", "", true},
		{"abc", "", "", "abc", true},
		{"abc", "d", "abc", "", false},
		{"", "d", "", "", false},
		{"", "", "", "", true},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Cut(t *testing.T) {
	for _, tt := range cut_tests() {
		before, after, found := standard_cut(tt.S, tt.Separator)
		if before != tt.Before {
			t.Errorf("standard_cut(%q, %q) before = %q; want %q",
				tt.S, tt.Separator, before, tt.Before)
		}
		if after != tt.After {
			t.Errorf("standard_cut(%q, %q) after = %q; want %q",
				tt.S, tt.Separator, after, tt.After)
		}
		if found != tt.Found {
			t.Errorf("standard_cut(%q, %q) found = %v; want %v",
				tt.S, tt.Separator, found, tt.Found)
		}
	}
}
func cut_prefix_tests() (value []struct {
	S, Separator string
	After        string
	Found        bool
}) {
	return []struct {
		S, Separator string
		After        string
		Found        bool
	}{
		{"abc", "a", "bc", true},
		{"abc", "abc", "", true},
		{"abc", "", "abc", true},
		{"abc", "d", "abc", false},
		{"", "d", "", false},
		{"", "", "", true},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Cut_Prefix(t *testing.T) {
	for _, tt := range cut_prefix_tests() {
		after, found := standard_cut_prefix(tt.S, tt.Separator)
		if after != tt.After {
			t.Errorf("standard_cut_prefix(%q, %q) after = %q; want %q",
				tt.S, tt.Separator, after, tt.After)
		}
		if found != tt.Found {
			t.Errorf("standard_cut_prefix(%q, %q) found = %v; want %v",
				tt.S, tt.Separator, found, tt.Found)
		}
	}
}
func cut_suffix_tests() (value []struct {
	S, Separator string
	Before       string
	Found        bool
}) {
	return []struct {
		S, Separator string
		Before       string
		Found        bool
	}{
		{"abc", "bc", "a", true},
		{"abc", "abc", "", true},
		{"abc", "", "abc", true},
		{"abc", "d", "abc", false},
		{"", "d", "", false},
		{"", "", "", true},
	}
}

// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Cut_Suffix(t *testing.T) {
	for _, tt := range cut_suffix_tests() {
		before, found := standard_cut_suffix(tt.S, tt.Separator)
		if before != tt.Before {
			t.Errorf("standard_cut_suffix(%q, %q) before = %q; want %q",
				tt.S, tt.Separator, before, tt.Before)
		}
		if found != tt.Found {
			t.Errorf("standard_cut_suffix(%q, %q) found = %v; want %v",
				tt.S, tt.Separator, found, tt.Found)
		}
	}
}

func make_bench_input_hard() (result_29 string) {
	tokens := [...]string{
		"<a>", "<p>", "<b>", "<strong>",
		"</a>", "</p>", "</b>", "</strong>",
		"hello", "world",
	}
	x := make([]byte, 0, shared_strings.TEXT_SIZE_MAXIMUM)
	random := shared_prng.New(99)
	for range shared_bits.INTEGER_MAXIMUM {
		i := shared_prng.Generator_Below(&random, len(tokens))
		if len(x)+len(tokens[i]) >= shared_strings.TEXT_SIZE_MAXIMUM {
			break
		}
		x = append(x, tokens[i]...)
	}
	return string(x)
}
func bench_input_hard() (value string) {
	return make_bench_input_hard()
}

func benchmark_index_hard(b *testing.B, separator string) {
	for i_index := 0; i_index < b.N; i_index++ {
		standard_index(bench_input_hard(), separator)
	}
}

func benchmark_last_index_hard(b *testing.B, separator string) {
	for i_index := 0; i_index < b.N; i_index++ {
		standard_last_index(bench_input_hard(), separator)
	}
}

func benchmark_count_hard(b *testing.B, separator string) {
	for i_index := 0; i_index < b.N; i_index++ {
		standard_count(bench_input_hard(), separator)
	}
}

func Benchmark_Standard_Library_Index_Hard1(b *testing.B) { benchmark_index_hard(b, "<>") }
func Benchmark_Standard_Library_Index_Hard2(b *testing.B) { benchmark_index_hard(b, "</pre>") }
func Benchmark_Standard_Library_Index_Hard3(b *testing.B) {
	benchmark_index_hard(b, "<b>hello world</b>")
}
func Benchmark_Standard_Library_Index_Hard4(b *testing.B) {
	benchmark_index_hard(b, "<pre><b>hello</b><strong>world</strong></pre>")
}

func Benchmark_Standard_Library_Last_Index_Hard1(b *testing.B) {
	benchmark_last_index_hard(b, "<>")
}
func Benchmark_Standard_Library_Last_Index_Hard2(b *testing.B) {
	benchmark_last_index_hard(b, "</pre>")
}
func Benchmark_Standard_Library_Last_Index_Hard3(b *testing.B) {
	benchmark_last_index_hard(b, "<b>hello world</b>")
}

func Benchmark_Standard_Library_Count_Hard1(b *testing.B) { benchmark_count_hard(b, "<>") }
func Benchmark_Standard_Library_Count_Hard2(b *testing.B) { benchmark_count_hard(b, "</pre>") }
func Benchmark_Standard_Library_Count_Hard3(b *testing.B) {
	benchmark_count_hard(b, "<b>hello world</b>")
}
func bench_input_torture() (value string) {
	repeat_count := (shared_strings.TEXT_SIZE_MAXIMUM - len("123")) / (2 * len("ABC"))
	return standard_repeat("ABC", repeat_count) + "123" +
		standard_repeat("ABC", repeat_count)
}
func bench_needle_torture() (value string) {
	return standard_repeat("ABC", shared_strings.TEXT_SIZE_MAXIMUM/len("ABC"))
}

func Benchmark_Standard_Library_Index_Torture(b *testing.B) {
	for i_index := 0; i_index < b.N; i_index++ {
		standard_index(bench_input_torture(), bench_needle_torture())
	}
}

func Benchmark_Standard_Library_Count_Torture(b *testing.B) {
	for i_index := 0; i_index < b.N; i_index++ {
		standard_count(bench_input_torture(), bench_needle_torture())
	}
}

func Benchmark_Standard_Library_Count_Torture_Overlap(b *testing.B) {
	A := standard_repeat("ABC", shared_strings.TEXT_SIZE_MAXIMUM/len("ABC"))
	B := standard_repeat("ABC", 1<<10)
	for i_index := 0; i_index < b.N; i_index++ {
		standard_count(A, B)
	}
}

func Benchmark_Standard_Library_Count_Byte(b *testing.B) {
	index_sizes := []int{10, 32, shared_strings.TEXT_SIZE_MAXIMUM}
	repeat_count := shared_strings.TEXT_SIZE_MAXIMUM / len(BENCHMARK_STRING)
	bench_string := standard_repeat(BENCHMARK_STRING, repeat_count)
	remainder := shared_strings.TEXT_SIZE_MAXIMUM - len(bench_string)
	bench_string += BENCHMARK_STRING[:remainder]
	bench_function := func(b *testing.B, bench_string string) {
		b.SetBytes(int64(len(bench_string)))
		for i_index := 0; i_index < b.N; i_index++ {
			standard_count(bench_string, "=")
		}
	}
	for _, size := range index_sizes {
		b.Run(fmt.Sprintf("%d", size), func(b *testing.B) {
			bench_function(b, bench_string[:size])
		})
	}

}

func make_fields_input() (result_30 string) {
	x := make([]byte, shared_strings.TEXT_SIZE_MAXIMUM)
	// Input is ~10% space, ~10% 2-byte UTF-8, rest ASCII non-space.
	random := shared_prng.New(99)
	for i := range x {
		switch shared_prng.Generator_Below(&random, 10) {
		case 0:
			x[i] = ' '
		case 1:
			if i > 0 {
				if x[i-1] == 'x' {
					copy(x[i-1:], "χ")
					break
				}
			}
			x[i] = 'x'
		default:
			x[i] = 'x'
		}
	}
	return string(x)
}

func make_fields_input_ascii() (result_31 string) {
	x := make([]byte, shared_strings.TEXT_SIZE_MAXIMUM)
	// Input is ~10% space, rest ASCII non-space.
	random := shared_prng.New(99)
	for i := range x {
		if shared_prng.Generator_Below(&random, 10) == 0 {
			x[i] = ' '
		} else {
			x[i] = 'x'
		}
	}
	return string(x)
}
func stringdata() (value []struct{ Name, Data string }) {
	return []struct{ Name, Data string }{
		{"ASCII", make_fields_input_ascii()},
		{"Mixed", make_fields_input()},
	}
}

func Benchmark_Standard_Library_Fields(b *testing.B) {
	for _, sd := range stringdata() {
		b.Run(sd.Name, func(b *testing.B) {
			for j := 1 << 4; j <= shared_strings.TEXT_SIZE_MAXIMUM; j <<= 4 {
				b.Run(fmt.Sprintf("%d", j), func(b *testing.B) {
					b.ReportAllocs()
					b.SetBytes(int64(j))
					data := sd.Data[:j]
					for i_index := 0; i_index < b.N; i_index++ {
						standard_fields(data)
					}
				})
			}
		})
	}
}

func Benchmark_Standard_Library_Fields_Function(b *testing.B) {
	for _, sd := range stringdata() {
		b.Run(sd.Name, func(b *testing.B) {
			for j := 1 << 4; j <= shared_strings.TEXT_SIZE_MAXIMUM; j <<= 4 {
				b.Run(fmt.Sprintf("%d", j), func(b *testing.B) {
					b.ReportAllocs()
					b.SetBytes(int64(j))
					data := sd.Data[:j]
					for i_index := 0; i_index < b.N; i_index++ {
						standard_fields_function(data, standard_is_space)
					}
				})
			}
		})
	}
}

func Benchmark_Standard_Library_Split_Empty_Separator(b *testing.B) {
	for i_index := 0; i_index < b.N; i_index++ {
		standard_split(bench_input_hard(), "")
	}
}

func Benchmark_Standard_Library_Split_Single_Byte_Separator(b *testing.B) {
	for i_index := 0; i_index < b.N; i_index++ {
		standard_split(bench_input_hard(), "/")
	}
}

func Benchmark_Standard_Library_Split_Multi_Byte_Separator(b *testing.B) {
	for i_index := 0; i_index < b.N; i_index++ {
		standard_split(bench_input_hard(), "hello")
	}
}

func Benchmark_Standard_Library_Split_N_Single_Byte_Separator(b *testing.B) {
	for i_index := 0; i_index < b.N; i_index++ {
		standard_split_n(bench_input_hard(), "/", 10)
	}
}

func Benchmark_Standard_Library_Split_N_Multi_Byte_Separator(b *testing.B) {
	for i_index := 0; i_index < b.N; i_index++ {
		standard_split_n(bench_input_hard(), "hello", 10)
	}
}

func Benchmark_Standard_Library_Repeat(b *testing.B) {
	s := "0123456789"
	for _, n := range []int{5, 10} {
		for _, c := range []int{0, 1, 2, 6} {
			b.Run(fmt.Sprintf("%dx%d", n, c), func(b *testing.B) {
				for i_index := 0; i_index < b.N; i_index++ {
					standard_repeat(s[:n], c)
				}
			})
		}
	}
}

func Benchmark_Standard_Library_Repeat_Large(b *testing.B) {
	source := standard_repeat("@", shared_strings.TEXT_SIZE_MAXIMUM)
	for j := 8; j <= 12; j++ {
		for _, k := range []int{1, 16, shared_strings.TEXT_SIZE_MAXIMUM} {
			fragment := source[:k]
			n := (1 << j) / k
			if n == 0 {
				continue
			}
			b.Run(fmt.Sprintf("%d/%d", 1<<j, k), func(b *testing.B) {
				for i_index := 0; i_index < b.N; i_index++ {
					standard_repeat(fragment, n)
				}
				b.SetBytes(int64(n * len(fragment)))
			})
		}
	}
}

func Benchmark_Standard_Library_Repeat_Spaces(b *testing.B) {
	b.ReportAllocs()
	for i_index := 0; i_index < b.N; i_index++ {
		standard_repeat(" ", 2)
	}
}

func Benchmark_Standard_Library_Index_Any_ASCII(b *testing.B) {
	x := standard_repeat("#", 2048) // Never matches set.
	cs := "0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz"
	for k := 1; k <= 2048; k <<= 4 {
		for j := 1; j <= 64; j <<= 1 {
			b.Run(fmt.Sprintf("%d:%d", k, j), func(b *testing.B) {
				for i_index := 0; i_index < b.N; i_index++ {
					standard_index_any(x[:k], cs[:j])
				}
			})
		}
	}
}

func Benchmark_Standard_Library_Index_Any_UTF8(b *testing.B) {
	x := standard_repeat("#", 2048) // Never matches set.
	cs := "你好世界, hello world. 你好世界, hello world. 你好世界, hello world."
	for k := 1; k <= 2048; k <<= 4 {
		for j := 1; j <= 64; j <<= 1 {
			b.Run(fmt.Sprintf("%d:%d", k, j), func(b *testing.B) {
				for i_index := 0; i_index < b.N; i_index++ {
					standard_index_any(x[:k], cs[:j])
				}
			})
		}
	}
}

func Benchmark_Standard_Library_Last_Index_Any_ASCII(b *testing.B) {
	x := standard_repeat("#", 2048) // Never matches set.
	cs := "0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz"
	for k := 1; k <= 2048; k <<= 4 {
		for j := 1; j <= 64; j <<= 1 {
			b.Run(fmt.Sprintf("%d:%d", k, j), func(b *testing.B) {
				for i_index := 0; i_index < b.N; i_index++ {
					standard_last_index_any(x[:k], cs[:j])
				}
			})
		}
	}
}

func Benchmark_Standard_Library_Last_Index_Any_UTF8(b *testing.B) {
	x := standard_repeat("#", 2048) // Never matches set.
	cs := "你好世界, hello world. 你好世界, hello world. 你好世界, hello world."
	for k := 1; k <= 2048; k <<= 4 {
		for j := 1; j <= 64; j <<= 1 {
			b.Run(fmt.Sprintf("%d:%d", k, j), func(b *testing.B) {
				for i_index := 0; i_index < b.N; i_index++ {
					standard_last_index_any(x[:k], cs[:j])
				}
			})
		}
	}
}

func Benchmark_Standard_Library_Trim_ASCII(b *testing.B) {
	cs := "0123456789abcdef"
	for k := 1; k <= 4096; k <<= 4 {
		for j := 1; j <= 16; j <<= 1 {
			b.Run(fmt.Sprintf("%d:%d", k, j), func(b *testing.B) {
				x := standard_repeat(cs[:j], (k+j-1)/j) // Always matches set.
				for i_index := 0; i_index < b.N; i_index++ {
					standard_trim(x[:k], cs[:j])
				}
			})
		}
	}
}

func Benchmark_Standard_Library_Trim_Byte(b *testing.B) {
	x := "  the quick brown fox   "
	for i_index := 0; i_index < b.N; i_index++ {
		standard_trim(x, " ")
	}
}

func Benchmark_Standard_Library_Index_Periodic(b *testing.B) {
	key := "aa"
	for _, skip := range [...]int{2, 4, 8, 16, 32, 64} {
		b.Run(fmt.Sprintf("IndexPeriodic%d", skip), func(b *testing.B) {
			s := standard_repeat(
				"a"+standard_repeat(" ", skip-1),
				shared_strings.TEXT_SIZE_MAXIMUM/skip,
			)
			for i_index := 0; i_index < b.N; i_index++ {
				standard_index(s, key)
			}
		})
	}
}

func Benchmark_Standard_Library_Join(b *testing.B) {
	vals := []string{"red", "yellow", "pink", "green", "purple", "orange", "blue"}
	for l := 0; l <= len(vals); l++ {
		name := shared_strconv.Format_Decimal(shared_strconv.Machine_Integer(l))
		b.Run(string(name), func(b *testing.B) {
			b.ReportAllocs()
			vals := vals[:l]
			for i_index := 0; i_index < b.N; i_index++ {
				standard_join(vals, " and ")
			}
		})
	}
}

func Benchmark_Standard_Library_Trim_Space(b *testing.B) {
	tests := []struct{ Name, Input string }{
		{"NoTrim", "typical"},
		{"ASCII", "  foo bar  "},
		{"SomeNonASCII", "    \u2000\t\r\n x\t\t\r\r\ny\n \u3000    "},
		{"JustNonASCII", "\u2000\u2000\u2000☺☺☺☺\u3000\u3000\u3000"},
	}
	for _, test := range tests {
		b.Run(test.Name, func(b *testing.B) {
			for i_index := 0; i_index < b.N; i_index++ {
				standard_trim_space(test.Input)
			}
		})
	}
}

func Benchmark_Standard_Library_Replace_All(b *testing.B) {
	b.ReportAllocs()
	for i_index := 0; i_index < b.N; i_index++ {
		standard_replace_all("banana", "a", "<>")
	}
}

func check_standard_builder(
	t *testing.T,
	builder *shared_strings.Builder,
	want string,
) {
	t.Helper()
	got := string(shared_strings.Builder_Text(builder))
	if got != want {
		t.Errorf("Builder text = %q; want %q", got, want)
	}
	if int(shared_strings.Builder_Size(builder)) != len(got) {
		t.Errorf("Builder size does not equal its text size")
	}
	if int(shared_strings.Builder_Capacity(builder)) < len(got) {
		t.Errorf("Builder capacity is smaller than its text size")
	}
}

// Test_Standard_Library_Builder preserves the upstream Builder write sequence.
// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Builder(t *testing.T) {
	var builder shared_strings.Builder
	check_standard_builder(t, &builder, "")
	count, write_err := standard_builder_write(&builder, "hello")
	if count != len("hello") {
		t.Errorf("Builder write count = %d; want %d", count, len("hello"))
	}
	if write_err != nil {
		t.Errorf("Builder write error = %v; want nil", write_err)
	}
	if byte_err := standard_builder_write_byte(&builder, ' '); byte_err != nil {
		t.Errorf("Builder byte write error = %v; want nil", byte_err)
	}
	if _, text_err := standard_builder_write(&builder, "world"); text_err != nil {
		t.Errorf("Builder write error = %v; want nil", text_err)
	}
	check_standard_builder(t, &builder, "hello world")
}

// Test_Standard_Library_Builder_Text preserves earlier returned text values.
// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Builder_Text(t *testing.T) {
	var builder shared_strings.Builder
	standard_builder_write(&builder, "alpha")
	alpha := string(shared_strings.Builder_Text(&builder))
	standard_builder_write(&builder, "beta")
	alpha_beta := string(shared_strings.Builder_Text(&builder))
	standard_builder_write(&builder, "gamma")
	alpha_beta_gamma := string(shared_strings.Builder_Text(&builder))
	if alpha != "alpha" {
		t.Errorf("first Builder text changed to %q", alpha)
	}
	if alpha_beta != "alphabeta" {
		t.Errorf("second Builder text changed to %q", alpha_beta)
	}
	if alpha_beta_gamma != "alphabetagamma" {
		t.Errorf("third Builder text = %q", alpha_beta_gamma)
	}
}

// Test_Standard_Library_Builder_Reset preserves text returned before a reset.
// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Builder_Reset(t *testing.T) {
	var builder shared_strings.Builder
	standard_builder_write(&builder, "aaa")
	previous := string(shared_strings.Builder_Text(&builder))
	shared_strings.Builder_Reset(&builder)
	check_standard_builder(t, &builder, "")
	standard_builder_write(&builder, "bbb")
	check_standard_builder(t, &builder, "bbb")
	if previous != "aaa" {
		t.Errorf("text returned before Reset changed to %q", previous)
	}
}

// Test_Standard_Library_Builder_Grow checks each in-domain upstream reservation.
// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Builder_Grow(t *testing.T) {
	for _, growth := range []int{0, 100, 1000, shared_strings.TEXT_SIZE_MAXIMUM} {
		var builder shared_strings.Builder
		shared_strings.Builder_Grow(&builder, shared_strings.Growth_Count(growth))
		if int(shared_strings.Builder_Capacity(&builder)) < growth {
			t.Errorf("Builder capacity = %d; want at least %d",
				shared_strings.Builder_Capacity(&builder), growth)
		}
		content := standard_repeat("a", growth)
		count, err := shared_io.Write(
			shared_strings.Builder_To_Stream(&builder), []byte(content),
		)
		if count != int64(growth) {
			t.Errorf("Builder write count = %d; want %d", count, growth)
		}
		if err != nil {
			t.Errorf("Builder write after Grow returned %v", err)
		}
		check_standard_builder(t, &builder, content)
	}
}

// Test_Standard_Library_Builder_Write_Forms preserves every upstream write form.
// This test keeps upstream standard-library regression coverage on shared/io.Stream.
func Test_Standard_Library_Builder_Write_Forms(t *testing.T) {
	const SOURCE = "hello 世界"
	tests := []struct {
		Name      string
		Operation func(*shared_strings.Builder) (count int, err error)
		Count     int
		Want      string
	}{
		{
			Name: "Write",
			Operation: func(builder *shared_strings.Builder) (count int, err error) {
				written, write_err := shared_io.Write(
					shared_strings.Builder_To_Stream(builder), []byte(SOURCE),
				)
				return int(written), write_err
			},
			Count: len(SOURCE),
			Want:  SOURCE,
		},
		{
			Name: "Write_Rune",
			Operation: func(builder *shared_strings.Builder) (count int, err error) {
				written, write_err := shared_io.Write_Rune(
					shared_strings.Builder_To_Stream(builder), 'a',
				)
				return int(written), write_err
			},
			Count: 1,
			Want:  "a",
		},
		{
			Name: "Write_Rune_Wide",
			Operation: func(builder *shared_strings.Builder) (count int, err error) {
				written, write_err := shared_io.Write_Rune(
					shared_strings.Builder_To_Stream(builder), '世',
				)
				return int(written), write_err
			},
			Count: 3,
			Want:  "世",
		},
		{
			Name: "Write_String",
			Operation: func(builder *shared_strings.Builder) (count int, err error) {
				return standard_builder_write(builder, SOURCE)
			},
			Count: len(SOURCE),
			Want:  SOURCE,
		},
	}
	for _, test := range tests {
		var builder shared_strings.Builder
		for call := 1; call <= 2; call++ {
			count, err := test.Operation(&builder)
			if err != nil {
				t.Fatalf("%s call %d returned %v", test.Name, call, err)
			}
			if count != test.Count {
				t.Errorf(
					"%s call %d count = %d; want %d",
					test.Name, call, count, test.Count,
				)
			}
			check_standard_builder(t, &builder, standard_repeat(test.Want, call))
		}
	}
}

// Test_Standard_Library_Builder_Write_Byte preserves NUL byte output.
// This test keeps upstream standard-library regression coverage on shared/io.Stream.
func Test_Standard_Library_Builder_Write_Byte(t *testing.T) {
	var builder shared_strings.Builder
	if err := standard_builder_write_byte(&builder, 'a'); err != nil {
		t.Errorf("Builder byte write returned %v", err)
	}
	if err := standard_builder_write_byte(&builder, 0); err != nil {
		t.Errorf("Builder NUL byte write returned %v", err)
	}
	check_standard_builder(t, &builder, "a\x00")
}

// Test_Standard_Library_Builder_Write_Invalid_Rune preserves replacement encoding.
// This test keeps upstream standard-library regression coverage on shared/io.Stream.
func Test_Standard_Library_Builder_Write_Invalid_Rune(t *testing.T) {
	var builder shared_strings.Builder
	count, err := shared_io.Write_Rune(shared_strings.Builder_To_Stream(&builder), -1)
	if count != int64(shared_utf8.CHARACTER_SIZE_THREE) {
		t.Errorf(
			"invalid rune write count = %d; want %d",
			count, shared_utf8.CHARACTER_SIZE_THREE,
		)
	}
	if err != nil {
		t.Errorf("invalid rune write returned %v", err)
	}
	check_standard_builder(t, &builder, string(shared_utf8.REPLACEMENT_CHARACTER))
}

// Test_Standard_Library_Clone checks equality and independent nonempty storage.
// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Clone(t *testing.T) {
	clone_tests := []string{
		"",
		standard_clone(""),
		standard_repeat("a", 42)[:0],
		"short",
		standard_repeat("a", 42),
	}
	for _, input := range clone_tests {
		clone := standard_clone(input)
		if clone != input {
			t.Errorf("Clone(%q) = %q; want %q", input, clone, input)
		}
		if len(input) > 0 {
			if unsafe.StringData(clone) == unsafe.StringData(input) {
				t.Errorf("Clone(%q) retained the input storage", input)
			}
		}
	}
}

type Compare_Test struct {
	Left  string
	Right string
	Order int
}

func compare_tests() (tests []Compare_Test) {
	return []Compare_Test{
		{Left: "", Right: "", Order: 0},
		{Left: "a", Right: "", Order: 1},
		{Left: "", Right: "a", Order: -1},
		{Left: "abc", Right: "abc", Order: 0},
		{Left: "ab", Right: "abc", Order: -1},
		{Left: "abc", Right: "ab", Order: 1},
		{Left: "x", Right: "ab", Order: 1},
		{Left: "ab", Right: "x", Order: -1},
		{Left: "abcdefghi", Right: "abcdefghj", Order: -1},
	}
}

// Test_Standard_Library_Compare preserves shifted-string comparison cases.
// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Compare(t *testing.T) {
	const SHIFT_COUNT = 16
	for _, test := range compare_tests() {
		for offset := 0; offset <= SHIFT_COUNT; offset++ {
			shifted := (standard_repeat("*", offset) + test.Right)[offset:]
			order := standard_compare(test.Left, shifted)
			if order != test.Order {
				t.Errorf("Compare(%q, %q), offset %d = %d; want %d",
					test.Left, test.Right, offset, order, test.Order)
			}
		}
	}
}

// Test_Standard_Library_Compare_Identical_String checks shared-prefix order.
// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Compare_Identical_String(t *testing.T) {
	const SOURCE = "Hello Gophers!"
	if standard_compare(SOURCE, SOURCE) != 0 {
		t.Error("a string must compare equal to itself")
	}
	if standard_compare(SOURCE, SOURCE[:1]) != 1 {
		t.Error("a string must compare after its proper prefix")
	}
}

func standard_unsafe_string(source []byte) (text string) {
	return unsafe.String(unsafe.SliceData(source), len(source))
}

func verify_compare_size(
	t *testing.T,
	left []byte,
	right []byte,
	size int,
	previous_size int,
) {
	left_text := standard_unsafe_string(left)
	right_text := standard_unsafe_string(right)
	if order := standard_compare(left_text[:size], right_text[:size]); order != 0 {
		t.Errorf("CompareIdentical(%d) = %d", size, order)
	}
	if size > 0 {
		if order := standard_compare(left_text[:size-1], right_text[:size]); order != -1 {
			t.Errorf("CompareLeftShorter(%d) = %d", size, order)
		}
		if order := standard_compare(left_text[:size], right_text[:size-1]); order != 1 {
			t.Errorf("CompareRightShorter(%d) = %d", size, order)
		}
	}
	for difference_index := previous_size; difference_index < size; difference_index++ {
		right[difference_index] = left[difference_index] - 1
		if order := standard_compare(left_text[:size], right_text[:size]); order != 1 {
			t.Errorf("CompareLeftBigger(%d,%d) = %d", size, difference_index, order)
		}
		right[difference_index] = left[difference_index] + 1
		if order := standard_compare(left_text[:size], right_text[:size]); order != -1 {
			t.Errorf("CompareRightBigger(%d,%d) = %d", size, difference_index, order)
		}
		right[difference_index] = left[difference_index]
	}
}

// This test preserves bounded exhaustive comparison sizes and alignments.
func Test_Standard_Library_Compare_Strings(t *testing.T) {
	sizes := make([]int, 0)
	for size := 0; size <= 128; size++ {
		sizes = append(sizes, size)
	}
	sizes = append(sizes, 256, 512, 1024, 1333, 4095, 4096)
	maximum := sizes[len(sizes)-1]
	left := make([]byte, maximum+1)
	right := make([]byte, maximum+1)
	previous_size := 0
	for _, size := range sizes {
		for index := 0; index < size; index++ {
			left[index] = byte(1 + 31*index%254)
			right[index] = left[index]
		}
		for index := size; index <= maximum; index++ {
			left[index] = 8
			right[index] = 9
		}
		verify_compare_size(t, left, right, size, previous_size)
		previous_size = size
	}
}

func standard_reader_read(
	reader *shared_strings.Reader,
	buffer []byte,
) (count int, err error) {
	read_count, read_err := shared_io.Read(
		shared_strings.Reader_To_Stream(reader), buffer,
	)
	return int(read_count), read_err
}

func standard_reader_read_at(
	reader *shared_strings.Reader,
	buffer []byte,
	offset int64,
) (count int, err error) {
	read_count, read_err := shared_io.Read_At(
		shared_strings.Reader_To_Stream(reader), buffer, offset,
	)
	return int(read_count), read_err
}

// Test_Standard_Library_Reader preserves seek and sequential read cases.
// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Reader(t *testing.T) {
	reader := shared_strings.New_Reader("0123456789")
	tests := []struct {
		Offset   int64
		Whence   shared_io.Seek_From
		Size     int
		Want     string
		Position int64
		Error    error
	}{
		{Whence: shared_io.SEEK_FROM_START, Size: 20, Want: "0123456789"},
		{Whence: shared_io.SEEK_FROM_START, Offset: 1, Size: 1, Want: "1", Position: 1},
		{Whence: shared_io.SEEK_FROM_CURRENT, Offset: 1, Size: 2, Want: "34", Position: 3},
		{
			Whence: shared_io.SEEK_FROM_START,
			Offset: -1,
			Error:  shared_io.Stream_Invalid_Offset,
		},
		{Whence: shared_io.SEEK_FROM_START, Size: 5, Want: "01234"},
		{Whence: shared_io.SEEK_FROM_CURRENT, Size: 5, Want: "56789", Position: 5},
		{Whence: shared_io.SEEK_FROM_END, Offset: -1, Size: 1, Want: "9", Position: 9},
	}
	for test_index, test := range tests {
		stream := shared_strings.Reader_To_Stream(reader)
		position, err := shared_io.Seek(stream, test.Offset, test.Whence)
		if err != test.Error {
			t.Errorf("case %d: seek error = %v; want %v", test_index, err, test.Error)
			continue
		}
		if err != nil {
			continue
		}
		if position != test.Position {
			t.Errorf("case %d: position = %d; want %d",
				test_index, position, test.Position)
		}
		buffer := make([]byte, test.Size)
		count, read_err := shared_io.Read(stream, buffer)
		if read_err != nil {
			t.Errorf("case %d: read error = %v; want nil", test_index, read_err)
		}
		if got := string(buffer[:count]); got != test.Want {
			t.Errorf("case %d: read = %q; want %q", test_index, got, test.Want)
		}
	}
}

// Test_Standard_Library_Reader_After_Large_Seek preserves the large-offset case.
// This test keeps upstream standard-library regression coverage on shared/io.Stream.
func Test_Standard_Library_Reader_After_Large_Seek(t *testing.T) {
	reader := shared_strings.New_Reader("0123456789")
	stream := shared_strings.Reader_To_Stream(reader)
	first := int64(1) << (shared_bits.BIT_COUNT_32_MAXIMUM + 1)
	check_standard_reader_eof_after_seek(
		t, stream, first, shared_io.SEEK_FROM_START, first,
	)
	check_standard_reader_eof_after_seek(
		t, stream, 1, shared_io.SEEK_FROM_CURRENT, first+1,
	)
	second := int64(1)<<shared_bits.BIT_COUNT_32_MAXIMUM + 5
	check_standard_reader_eof_after_seek(
		t, stream, second, shared_io.SEEK_FROM_START, second,
	)
}

func check_standard_reader_eof_after_seek(
	t *testing.T,
	stream shared_io.Stream,
	offset int64,
	whence shared_io.Seek_From,
	want int64,
) {
	t.Helper()
	position, seek_err := shared_io.Seek(stream, offset, whence)
	if position != want {
		t.Errorf("Reader seek position = %d; want %d", position, want)
	}
	if seek_err != nil {
		t.Fatalf("Reader large seek returned %v", seek_err)
	}
	count, read_err := shared_io.Read(stream, make([]byte, 10))
	if count != 0 {
		t.Errorf("Reader after large seek read %d bytes; want 0", count)
	}
	if read_err != shared_io.Stream_EOF {
		t.Errorf("Reader after large seek returned %v; want end of file", read_err)
	}
}

// Test_Standard_Library_Reader_At preserves positional read cases.
// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Reader_At(t *testing.T) {
	reader := shared_strings.New_Reader("0123456789")
	tests := []struct {
		Offset int64
		Size   int
		Want   string
		Error  error
	}{
		{Offset: 0, Size: 10, Want: "0123456789"},
		{Offset: 1, Size: 10, Want: "123456789", Error: shared_io.Stream_EOF},
		{Offset: 1, Size: 9, Want: "123456789"},
		{Offset: 11, Size: 10, Error: shared_io.Stream_EOF},
		{Offset: 0, Size: 0, Want: ""},
		{Offset: -1, Error: shared_io.Stream_Invalid_Offset},
	}
	for test_index, test := range tests {
		buffer := make([]byte, test.Size)
		count, err := standard_reader_read_at(reader, buffer, test.Offset)
		if got := string(buffer[:count]); got != test.Want {
			t.Errorf("case %d: Read_At = %q; want %q", test_index, got, test.Want)
		}
		if err != test.Error {
			t.Errorf(
				"case %d: Read_At error = %v; want %v",
				test_index, err, test.Error,
			)
		}
	}
}

// Test_Standard_Library_Reader_Write_To preserves complete remaining output.
// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Reader_Write_To(t *testing.T) {
	const SOURCE = "0123456789"
	for source_index := 0; source_index <= len(SOURCE); source_index++ {
		suffix := SOURCE[source_index:]
		reader := shared_strings.New_Reader(shared_strings.Text(suffix))
		memory := make([]byte, len(suffix))
		destination := shared_io.Stream_Memory{Memory: memory}
		count, err := shared_strings.Reader_Write_To(
			reader, shared_io.Memory_To_Stream(&destination),
		)
		if err != nil {
			t.Errorf("offset %d: Write_To error = %v", source_index, err)
		}
		if int(count) != len(suffix) {
			t.Errorf("offset %d: Write_To count = %d; want %d",
				source_index, count, len(suffix))
		}
		if string(memory) != suffix {
			t.Errorf("offset %d: Write_To = %q; want %q",
				source_index, memory, suffix)
		}
	}
}

// Test_Standard_Library_Reader_Size checks unread and total byte counts.
// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Reader_Size(t *testing.T) {
	reader := shared_strings.New_Reader("abc")
	if _, err := standard_reader_read(reader, make([]byte, 1)); err != nil {
		t.Errorf("Reader read error = %v", err)
	}
	if shared_strings.Reader_Unread_Size(reader) != 2 {
		t.Errorf("Reader unread size = %d; want 2",
			shared_strings.Reader_Unread_Size(reader))
	}
	size, err := shared_io.Size(shared_strings.Reader_To_Stream(reader))
	if size != 3 {
		t.Errorf("Reader size = %d; want 3", size)
	}
	if err != nil {
		t.Errorf("Reader size error = %v; want nil", err)
	}
}

// Test_Standard_Library_Reader_Reset preserves the upstream reset state.
// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Reader_Reset(t *testing.T) {
	reader := shared_strings.New_Reader("世界")
	shared_strings.Reader_Read_Character(reader)
	shared_strings.Reader_Reset(reader, "abcdef")
	if err := shared_strings.Reader_Unread_Character(reader); err == nil {
		t.Error("Unread_Character after Reset returned nil")
	}
	buffer := make([]byte, len("abcdef"))
	count, err := standard_reader_read(reader, buffer)
	if count != len(buffer) {
		t.Errorf("Reader after Reset count = %d; want %d", count, len(buffer))
	}
	if err != nil {
		t.Errorf("Reader after Reset error = %v; want nil", err)
	}
	if string(buffer) != "abcdef" {
		t.Errorf("Reader after Reset = %q; want abcdef", buffer)
	}
}

// This test keeps upstream zero-value Reader coverage on the shared stream boundary.
func Test_Standard_Library_Reader_Zero(t *testing.T) {
	reader := &shared_strings.Reader{}
	if shared_strings.Reader_Unread_Size(reader) != 0 {
		t.Error("a zero Reader has unread content")
	}
	if _, err := standard_reader_read(reader, nil); err != shared_io.Stream_EOF {
		t.Errorf("zero Reader read error = %v; want end of file", err)
	}
	if _, err := standard_reader_read_at(reader, nil, 11); err != shared_io.Stream_EOF {
		t.Errorf("zero Reader Read_At error = %v; want end of file", err)
	}
	if _, err := shared_strings.Reader_Read_Byte(reader); err != shared_io.Stream_EOF {
		t.Errorf("zero Reader byte error = %v; want end of file", err)
	}
	if _, _, err := shared_strings.Reader_Read_Character(reader); err != shared_io.Stream_EOF {
		t.Errorf("zero Reader character error = %v; want end of file", err)
	}
	if err := shared_strings.Reader_Unread_Byte(reader); err == nil {
		t.Error("zero Reader Unread_Byte returned nil")
	}
	if err := shared_strings.Reader_Unread_Character(reader); err == nil {
		t.Error("zero Reader Unread_Character returned nil")
	}
	size, size_err := shared_io.Size(shared_strings.Reader_To_Stream(reader))
	if size != 0 {
		t.Errorf("zero Reader size = %d; want 0", size)
	}
	if size_err != nil {
		t.Errorf("zero Reader size error = %v; want nil", size_err)
	}
	position, seek_err := shared_io.Seek(
		shared_strings.Reader_To_Stream(reader), 11, shared_io.SEEK_FROM_START,
	)
	if position != 11 {
		t.Errorf("zero Reader seek position = %d; want 11", position)
	}
	if seek_err != nil {
		t.Errorf("zero Reader seek returned %v; want nil", seek_err)
	}
	count, write_err := shared_strings.Reader_Write_To(
		reader, shared_io.Discard_To_Stream(&shared_io.Stream_Discard{}),
	)
	if count != 0 {
		t.Errorf("zero Reader Write_To count = %d; want 0", count)
	}
	if write_err != nil {
		t.Errorf("zero Reader Write_To error = %v; want nil", write_err)
	}
}

type Replacer_Test struct {
	Replacer *shared_strings.Replacer
	Input    string
	Output   string
}

func standard_replacer_byte_tests() (tests []Replacer_Test) {
	capital_letters := standard_new_replacer("a", "A", "b", "B")
	html_escape := standard_new_replacer(
		"&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;",
	)
	html_unescape := standard_new_replacer(
		"&amp;", "&", "&lt;", "<", "&gt;", ">", "&quot;", `"`, "&apos;", "'",
	)
	tests = []Replacer_Test{
		{Replacer: capital_letters, Input: "brad", Output: "BrAd"},
		{Replacer: capital_letters, Input: standard_repeat("a", 2048),
			Output: standard_repeat("A", 2048)},
		{Replacer: capital_letters, Input: "", Output: ""},
		{Replacer: standard_new_replacer("a", "1", "a", "2"),
			Input: "brad", Output: "br1d"},
	}
	increment_pairs := make(shared_strings.Replacement_Pairs, 0)
	repeat_pairs := make(shared_strings.Replacement_Pairs, 0)
	for value := 0; value <= int(shared_bits.WORD_8_MAXIMUM); value++ {
		input := string([]byte{byte(value)})
		output := string([]byte{byte(value + 1)})
		increment_pairs = append(
			increment_pairs, shared_strings.Replacement_Pair{
				shared_strings.Text(input), shared_strings.Text(output),
			},
		)
		repeat_count := value + 1 - int('a')
		if repeat_count < 1 {
			repeat_count = 1
		}
		repeat_pairs = append(repeat_pairs, shared_strings.Replacement_Pair{
			shared_strings.Text(input),
			shared_strings.Text(standard_repeat(input, repeat_count)),
		})
	}
	increment := shared_strings.New_Replacer(increment_pairs)
	byte_repeat_replacer := shared_strings.New_Replacer(repeat_pairs)
	tests = append(tests,
		Replacer_Test{Replacer: increment, Input: "brad", Output: "csbe"},
		Replacer_Test{Replacer: increment, Input: "\x00\xff", Output: "\x01\x00"},
		Replacer_Test{Replacer: increment, Input: "", Output: ""},
		Replacer_Test{Replacer: html_escape, Input: "No changes", Output: "No changes"},
		Replacer_Test{Replacer: html_escape, Input: "I <3 escaping & stuff",
			Output: "I &lt;3 escaping &amp; stuff"},
		Replacer_Test{Replacer: html_escape, Input: "&&&", Output: "&amp;&amp;&amp;"},
		Replacer_Test{Replacer: html_escape, Input: "", Output: ""},
		Replacer_Test{
			Replacer: byte_repeat_replacer, Input: "brad",
			Output: "bbrrrrrrrrrrrrrrrrrradddd",
		},
		Replacer_Test{Replacer: byte_repeat_replacer, Input: "abba", Output: "abbbba"},
		Replacer_Test{Replacer: byte_repeat_replacer, Input: "", Output: ""},
		Replacer_Test{Replacer: standard_new_replacer("a", "11", "a", "22"),
			Input: "brad", Output: "br11d"},
		Replacer_Test{Replacer: html_unescape, Input: "&amp;amp;", Output: "&amp;"},
		Replacer_Test{Replacer: html_unescape, Input: "&lt;b&gt;HTML&apos;s neat&lt;/b&gt;",
			Output: "<b>HTML's neat</b>"},
		Replacer_Test{Replacer: html_unescape, Input: "", Output: ""},
		Replacer_Test{Replacer: standard_new_replacer("a", "1", "a", "2", "xxx", "xxx"),
			Input: "brad", Output: "br1d"},
		Replacer_Test{Replacer: standard_new_replacer("a", "1", "aa", "2", "aaa", "3"),
			Input: "aaaa", Output: "1111"},
		Replacer_Test{Replacer: standard_new_replacer("aaa", "3", "aa", "2", "a", "1"),
			Input: "aaaa", Output: "31"},
	)
	return tests
}

func standard_replacer_general_tests() (tests []Replacer_Test) {
	general_one := standard_new_replacer(
		"aaa", "3[aaa]", "aa", "2[aa]", "a", "1[a]", "i", "i",
		"longerst", "most long", "longer", "medium", "long", "short",
		"xx", "xx", "x", "X", "X", "Y", "Y", "Z",
	)
	tests = append(tests,
		Replacer_Test{Replacer: general_one, Input: "fooaaabar", Output: "foo3[aaa]b1[a]r"},
		Replacer_Test{
			Replacer: general_one, Input: "long, longerst, longer",
			Output: "short, most long, medium",
		},
		Replacer_Test{Replacer: general_one, Input: "xxxxx", Output: "xxxxX"},
		Replacer_Test{Replacer: general_one, Input: "XiX", Output: "YiY"},
		Replacer_Test{Replacer: general_one, Input: "", Output: ""},
	)
	general_two := standard_new_replacer(
		"roses", "red", "violets", "blue", "sugar", "sweet",
	)
	tests = append(tests,
		Replacer_Test{
			Replacer: general_two, Input: "roses are red, violets are blue...",
			Output: "red are red, blue are blue...",
		},
		Replacer_Test{Replacer: general_two, Input: "", Output: ""},
	)
	general_three := standard_new_replacer(
		"abracadabra", "poof", "abracadabrakazam", "splat", "abraham", "lincoln",
		"abrasion", "scrape", "abraham", "isaac",
	)
	tests = append(tests,
		Replacer_Test{
			Replacer: general_three, Input: "abracadabrakazam abraham",
			Output: "poofkazam lincoln",
		},
		Replacer_Test{
			Replacer: general_three, Input: "abrasion abracad",
			Output: "scrape abracad",
		},
		Replacer_Test{
			Replacer: general_three, Input: "abba abram abrasive",
			Output: "abba abram abrasive",
		},
		Replacer_Test{Replacer: general_three, Input: "", Output: ""},
	)
	return tests
}

func standard_replacer_prefix_tests() (tests []Replacer_Test) {
	foo_one := standard_new_replacer("foo1", "A", "foo2", "B", "foo3", "C")
	foo_two := standard_new_replacer(
		"foo1", "A", "foo2", "B", "foo31", "C", "foo32", "D",
	)
	foo_three := standard_new_replacer(
		"foo11", "A", "foo12", "B", "foo31", "C", "foo32", "D",
	)
	foo_four := standard_new_replacer("foo12", "B", "foo32", "D")
	tests = append(tests,
		Replacer_Test{Replacer: foo_one, Input: "fofoofoo12foo32oo", Output: "fofooA2C2oo"},
		Replacer_Test{Replacer: foo_one, Input: "", Output: ""},
		Replacer_Test{Replacer: foo_two, Input: "fofoofoo12foo32oo", Output: "fofooA2Doo"},
		Replacer_Test{Replacer: foo_two, Input: "", Output: ""},
		Replacer_Test{Replacer: foo_three, Input: "fofoofoo12foo32oo", Output: "fofooBDoo"},
		Replacer_Test{Replacer: foo_three, Input: "", Output: ""},
		Replacer_Test{Replacer: foo_four, Input: "fofoofoo12foo32oo", Output: "fofooBDoo"},
		Replacer_Test{Replacer: foo_four, Input: "", Output: ""},
	)
	all_bytes := make([]byte, int(shared_bits.WORD_8_MAXIMUM)+1)
	for index := range all_bytes {
		all_bytes[index] = byte(index)
	}
	all_text := string(all_bytes)
	all_replacer := standard_new_replacer(
		all_text, "[all]", "\xff", "[ff]", "\x00", "[00]",
	)
	tests = append(tests,
		Replacer_Test{Replacer: all_replacer, Input: all_text, Output: "[all]"},
		Replacer_Test{
			Replacer: all_replacer, Input: "a\xff" + all_text + "\x00",
			Output: "a[ff][all][00]",
		},
		Replacer_Test{Replacer: all_replacer, Input: "", Output: ""},
	)
	return tests
}

func standard_replacer_empty_tests() (tests []Replacer_Test) {
	blank_to_x_one := standard_new_replacer("", "X")
	blank_to_x_two := standard_new_replacer("", "X", "", "")
	blank_high_priority := standard_new_replacer("", "X", "o", "O")
	blank_low_priority := standard_new_replacer("o", "O", "", "X")
	blank_no_operation_one := standard_new_replacer("", "")
	blank_no_operation_two := standard_new_replacer("", "", "", "A")
	blank_foo := standard_new_replacer("", "X", "foobar", "R", "foobaz", "Z")
	tests = append(tests,
		Replacer_Test{Replacer: blank_to_x_one, Input: "foo", Output: "XfXoXoX"},
		Replacer_Test{Replacer: blank_to_x_one, Input: "", Output: "X"},
		Replacer_Test{Replacer: blank_to_x_two, Input: "foo", Output: "XfXoXoX"},
		Replacer_Test{Replacer: blank_to_x_two, Input: "", Output: "X"},
		Replacer_Test{Replacer: blank_high_priority, Input: "oo", Output: "XOXOX"},
		Replacer_Test{Replacer: blank_high_priority, Input: "ii", Output: "XiXiX"},
		Replacer_Test{Replacer: blank_high_priority, Input: "oiio", Output: "XOXiXiXOX"},
		Replacer_Test{Replacer: blank_high_priority, Input: "iooi", Output: "XiXOXOXiX"},
		Replacer_Test{Replacer: blank_high_priority, Input: "", Output: "X"},
		Replacer_Test{Replacer: blank_low_priority, Input: "oo", Output: "OOX"},
		Replacer_Test{Replacer: blank_low_priority, Input: "ii", Output: "XiXiX"},
		Replacer_Test{Replacer: blank_low_priority, Input: "oiio", Output: "OXiXiOX"},
		Replacer_Test{Replacer: blank_low_priority, Input: "iooi", Output: "XiOOXiX"},
		Replacer_Test{Replacer: blank_low_priority, Input: "", Output: "X"},
		Replacer_Test{Replacer: blank_no_operation_one, Input: "foo", Output: "foo"},
		Replacer_Test{Replacer: blank_no_operation_one, Input: "", Output: ""},
		Replacer_Test{Replacer: blank_no_operation_two, Input: "foo", Output: "foo"},
		Replacer_Test{Replacer: blank_no_operation_two, Input: "", Output: ""},
		Replacer_Test{Replacer: blank_foo, Input: "foobarfoobaz", Output: "XRXZX"},
		Replacer_Test{Replacer: blank_foo, Input: "foobar-foobaz", Output: "XRX-XZX"},
		Replacer_Test{Replacer: blank_foo, Input: "", Output: "X"},
	)
	return tests
}

func standard_replacer_single_tests() (tests []Replacer_Test) {
	abc_matcher := standard_new_replacer("abc", "[match]")
	no_hello := standard_new_replacer("Hello", "")
	no_operation := standard_new_replacer()
	tests = append(tests,
		Replacer_Test{Replacer: abc_matcher, Input: "", Output: ""},
		Replacer_Test{Replacer: abc_matcher, Input: "ab", Output: "ab"},
		Replacer_Test{Replacer: abc_matcher, Input: "abc", Output: "[match]"},
		Replacer_Test{Replacer: abc_matcher, Input: "abcd", Output: "[match]d"},
		Replacer_Test{
			Replacer: abc_matcher, Input: "cabcabcdabca",
			Output: "c[match][match]d[match]a",
		},
		Replacer_Test{Replacer: no_hello, Input: "Hello", Output: ""},
		Replacer_Test{Replacer: no_hello, Input: "Hellox", Output: "x"},
		Replacer_Test{Replacer: no_hello, Input: "xHello", Output: "x"},
		Replacer_Test{Replacer: no_hello, Input: "xHellox", Output: "xx"},
		Replacer_Test{Replacer: no_operation, Input: "abc", Output: "abc"},
		Replacer_Test{Replacer: no_operation, Input: "", Output: ""},
	)
	return tests
}

func standard_replacer_tests() (tests []Replacer_Test) {
	tests = standard_replacer_byte_tests()
	tests = append(tests, standard_replacer_general_tests()...)
	tests = append(tests, standard_replacer_prefix_tests()...)
	tests = append(tests, standard_replacer_empty_tests()...)
	tests = append(tests, standard_replacer_single_tests()...)
	return tests
}

// Test_Standard_Library_Replacer preserves rule priority and empty-match cases.
// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Replacer(t *testing.T) {
	for test_index, test := range standard_replacer_tests() {
		got := standard_replacer_replace(test.Replacer, test.Input)
		if got != test.Output {
			t.Errorf("case %d: Replacer = %q; want %q",
				test_index, got, test.Output)
		}
		memory := make([]byte, len(test.Output))
		destination := shared_io.Stream_Memory{Memory: memory}
		count, err := standard_replacer_write(
			test.Replacer, shared_io.Memory_To_Stream(&destination), test.Input,
		)
		if err != nil {
			t.Errorf("case %d: Replacer write error = %v", test_index, err)
		}
		if count != len(test.Output) {
			t.Errorf("case %d: Replacer write count = %d; want %d",
				test_index, count, len(test.Output))
		}
		if string(memory) != test.Output {
			t.Errorf("case %d: Replacer write = %q; want %q",
				test_index, memory, test.Output)
		}
	}
}

// Test_Standard_Library_Replacer_Copies_Rules checks constructor ownership.
// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Replacer_Copies_Rules(t *testing.T) {
	pairs := shared_strings.Replacement_Pairs{{"a", "A"}}
	replacer := shared_strings.New_Replacer(pairs)
	pairs[0] = shared_strings.Replacement_Pair{"a", "wrong"}
	if got := standard_replacer_replace(replacer, "a"); got != "A" {
		t.Errorf("Replacer changed with caller rules: got %q; want A", got)
	}
}

// Test_Standard_Library_Replacer_Write_Error preserves destination errors.
// This test keeps upstream standard-library regression coverage in this bounded port.
func Test_Standard_Library_Replacer_Write_Error(t *testing.T) {
	replacer := standard_new_replacer("a", "A")
	destination := shared_io.Stream{
		Procedure: func(
			_ any,
			mode shared_io.Stream_Mode,
			_ []byte,
			_ int64,
			_ shared_io.Seek_From,
		) (count int64, err error) {
			if mode == shared_io.STREAM_MODE_QUERY {
				return int64(1 << shared_io.STREAM_MODE_WRITE), nil
			}
			return 0, shared_io.Stream_Short_Write
		},
	}
	count, err := standard_replacer_write(replacer, destination, "abc")
	if count != 0 {
		t.Errorf("Replacer write count = %d; want 0", count)
	}
	if err != shared_io.Stream_Short_Write {
		t.Errorf("Replacer write error = %v; want short write", err)
	}
}

func check_standard_example[T comparable](
	t *testing.T, name string, got T, want T,
) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v; want %v", name, got, want)
	}
}

func check_standard_example_texts(
	t *testing.T, name string, got []string, want []string,
) {
	t.Helper()
	if !shared_slices.Equal(got, want) {
		t.Errorf("%s = %q; want %q", name, got, want)
	}
}

// Test_Standard_Library_Examples_Queries preserves the upstream Boolean examples.
func Test_Standard_Library_Examples_Queries(t *testing.T) {
	vowel := func(character rune) (matches bool) {
		return character == 'a' || character == 'e' || character == 'i' ||
			character == 'o' || character == 'u'
	}
	tests := []struct {
		Name string
		Got  bool
		Want bool
	}{
		{Name: "Contains present", Got: standard_contains("seafood", "foo"), Want: true},
		{Name: "Contains absent", Got: standard_contains("seafood", "bar"), Want: false},
		{Name: "Contains empty", Got: standard_contains("seafood", ""), Want: true},
		{Name: "Contains empty source", Got: standard_contains("", ""), Want: true},
		{Name: "Contains_Any absent", Got: standard_contains_any("team", "i"), Want: false},
		{
			Name: "Contains_Any present", Got: standard_contains_any("fail", "ui"),
			Want: true,
		},
		{Name: "Contains_Any empty", Got: standard_contains_any("foo", ""), Want: false},
		{
			Name: "Contains_Rune present", Got: standard_contains_rune("aardvark", 97),
			Want: true,
		},
		{
			Name: "Contains_Rune absent", Got: standard_contains_rune("timeout", 97),
			Want: false,
		},
		{
			Name: "Contains_Function present",
			Got:  standard_contains_function("hello", vowel), Want: true,
		},
		{
			Name: "Contains_Function absent",
			Got:  standard_contains_function("rhythms", vowel), Want: false,
		},
		{Name: "Equal_Fold", Got: standard_equal_fold("Go", "go"), Want: true},
		{Name: "Equal_Fold simple", Got: standard_equal_fold("AB", "ab"), Want: true},
		{Name: "Equal_Fold full", Got: standard_equal_fold("ß", "ss"), Want: false},
		{Name: "Has_Prefix", Got: standard_has_prefix("Gopher", "Go"), Want: true},
		{Name: "Has_Prefix absent", Got: standard_has_prefix("Gopher", "C"), Want: false},
		{Name: "Has_Suffix", Got: standard_has_suffix("Amigo", "go"), Want: true},
		{Name: "Has_Suffix absent", Got: standard_has_suffix("Amigo", "O"), Want: false},
	}
	for _, test := range tests {
		check_standard_example(t, test.Name, test.Got, test.Want)
	}
}

// Test_Standard_Library_Examples_Search preserves the upstream index examples.
func Test_Standard_Library_Examples_Search(t *testing.T) {
	tests := []struct {
		Name string
		Got  int
		Want int
	}{
		{Name: "Compare before", Got: standard_compare("a", "b"), Want: -1},
		{Name: "Compare equal", Got: standard_compare("a", "a"), Want: 0},
		{Name: "Compare after", Got: standard_compare("b", "a"), Want: 1},
		{Name: "Count", Got: standard_count("cheese", "e"), Want: 3},
		{Name: "Count empty", Got: standard_count("five", ""), Want: 5},
		{Name: "Index", Got: standard_index("chicken", "ken"), Want: 4},
		{Name: "Index absent", Got: standard_index("chicken", "dmr"), Want: -1},
		{Name: "Index_Any", Got: standard_index_any("chicken", "aeiouy"), Want: 2},
		{Name: "Index_Byte", Got: standard_index_byte("gophers", 'h'), Want: 3},
		{Name: "Index_Rune", Got: standard_index_rune("chicken", 'k'), Want: 4},
		{Name: "Last_Index", Got: standard_last_index("go gopher", "go"), Want: 3},
		{Name: "Last_Index_Any", Got: standard_last_index_any("go gopher", "go"), Want: 4},
		{
			Name: "Last_Index_Byte", Got: standard_last_index_byte("Hello, world", 'l'),
			Want: 10,
		},
		{Name: "Last_Index_Function", Got: standard_last_index_function(
			"go 123", standard_is_digit,
		), Want: 5},
	}
	for _, test := range tests {
		check_standard_example(t, test.Name, test.Got, test.Want)
	}
	check_standard_example(t, "Index_Function Han",
		standard_index_function("Hello, 世界", standard_is_han), 7)
}

func standard_is_han(character rune) (matches bool) {
	return bool(ucd.Is_Script(ucd.Character(character), "Han"))
}

func standard_is_not_letter_or_number(character rune) (matches bool) {
	if ucd.Is_Letter(ucd.Character(character)) {
		return false
	}
	return !bool(ucd.Is_Number(ucd.Character(character)))
}

// Test_Standard_Library_Examples_Fields preserves the upstream field examples.
func Test_Standard_Library_Examples_Fields(t *testing.T) {
	check_standard_example_texts(
		t, "Fields", standard_fields("  foo bar  baz   "),
		[]string{"foo", "bar", "baz"},
	)
	check_standard_example_texts(
		t, "Fields_Function",
		standard_fields_function("  foo1;bar2,baz3...", standard_is_not_letter_or_number),
		[]string{"foo1", "bar2", "baz3"},
	)
}

func standard_cut_example(source string, separator string) (result string) {
	before, after, found := standard_cut(source, separator)
	return fmt.Sprintf("%s|%s|%t", before, after, found)
}

func standard_cut_prefix_example(source string, prefix string) (result string) {
	after, found := standard_cut_prefix(source, prefix)
	return fmt.Sprintf("%s|%t", after, found)
}

func standard_cut_suffix_example(source string, suffix string) (result string) {
	before, found := standard_cut_suffix(source, suffix)
	return fmt.Sprintf("%s|%t", before, found)
}

// Test_Standard_Library_Examples_Cut preserves the upstream cut examples.
func Test_Standard_Library_Examples_Cut(t *testing.T) {
	check_standard_example(t, "Cut prefix", standard_cut_example("Gopher", "Go"), "|pher|true")
	check_standard_example(t, "Cut middle", standard_cut_example("Gopher", "ph"), "Go|er|true")
	check_standard_example(t, "Cut suffix", standard_cut_example("Gopher", "er"), "Goph||true")
	check_standard_example(
		t, "Cut absent", standard_cut_example("Gopher", "Badger"), "Gopher||false",
	)
	check_standard_example(
		t, "Cut_Prefix present", standard_cut_prefix_example("Gopher", "Go"), "pher|true",
	)
	check_standard_example(
		t, "Cut_Prefix absent", standard_cut_prefix_example("Gopher", "ph"), "Gopher|false",
	)
	check_standard_example(
		t, "Cut_Suffix absent", standard_cut_suffix_example("Gopher", "Go"), "Gopher|false",
	)
	check_standard_example(
		t, "Cut_Suffix present", standard_cut_suffix_example("Gopher", "er"), "Goph|true",
	)
}

// Test_Standard_Library_Examples_Split preserves the upstream split examples.
func Test_Standard_Library_Examples_Split(t *testing.T) {
	check_standard_example_texts(
		t, "Split", standard_split("a,b,c", ","), []string{"a", "b", "c"},
	)
	check_standard_example_texts(
		t, "Split words", standard_split("a man a plan a canal panama", "a "),
		[]string{"", "man ", "plan ", "canal panama"},
	)
	check_standard_example_texts(
		t, "Split empty separator", standard_split(" xyz ", ""),
		[]string{" ", "x", "y", "z", " "},
	)
	check_standard_example_texts(
		t, "Split_N", standard_split_n("a,b,c", ",", 2), []string{"a", "b,c"},
	)
	check_standard_example_texts(
		t, "Split_After", standard_split_after("a,b,c", ","), []string{"a,", "b,", "c"},
	)
	check_standard_example_texts(
		t, "Split_After_N", standard_split_after_n("a,b,c", ",", 2),
		[]string{"a,", "b,c"},
	)
	zero := shared_strings.Split_N("a,b,c", ",", 0)
	check_standard_example(t, "Split_N zero is nil", zero == nil, true)
	check_standard_example(t, "Join", standard_join([]string{"foo", "bar", "baz"}, ", "),
		"foo, bar, baz")
}

func standard_rot_13(character rune) (mapped rune) {
	if character >= 'A' {
		if character <= 'Z' {
			return 'A' + (character-'A'+13)%26
		}
	}
	if character >= 'a' {
		if character <= 'z' {
			return 'a' + (character-'a'+13)%26
		}
	}
	return character
}

// Test_Standard_Library_Examples_Transform preserves the upstream transform examples.
func Test_Standard_Library_Examples_Transform(t *testing.T) {
	turkish := standard_turkish_case()
	tests := []struct {
		Name string
		Got  string
		Want string
	}{
		{Name: "Map", Got: standard_map(standard_rot_13,
			"'Twas brillig and the slithy gopher..."),
			Want: "'Gjnf oevyyvt naq gur fyvgul tbcure..."},
		{Name: "Repeat", Got: "ba" + standard_repeat("na", 2), Want: "banana"},
		{Name: "Replace", Got: standard_replace("oink oink oink", "k", "ky", 2),
			Want: "oinky oinky oink"},
		{Name: "Replace all", Got: standard_replace("oink oink oink", "oink", "moo", -1),
			Want: "moo moo moo"},
		{Name: "Replace_All", Got: standard_replace_all("oink oink oink", "oink", "moo"),
			Want: "moo moo moo"},
		{
			Name: "Title", Got: standard_title("her royal highness"),
			Want: "Her Royal Highness",
		},
		{Name: "To_Title", Got: standard_to_title("брат"), Want: "БРАТ"},
		{Name: "To_Title_Special", Got: standard_to_title_special(
			turkish, "dünyanın ilk borsa yapısı Aizonai kabul edilir",
		), Want: "DÜNYANIN İLK BORSA YAPISI AİZONAİ KABUL EDİLİR"},
		{Name: "To_Upper", Got: standard_to_upper("Gopher"), Want: "GOPHER"},
		{Name: "To_Upper_Special", Got: standard_to_upper_special(
			turkish, "örnek iş",
		), Want: "ÖRNEK İŞ"},
		{Name: "To_Lower", Got: standard_to_lower("Gopher"), Want: "gopher"},
		{Name: "To_Lower_Special", Got: standard_to_lower_special(
			turkish, "Örnek İş",
		), Want: "örnek iş"},
		{Name: "To_Valid_UTF8 valid", Got: standard_to_valid_utf8("abc", "�"), Want: "abc"},
		{Name: "To_Valid_UTF8 invalid", Got: standard_to_valid_utf8(
			"a\xffb\xC0\xAFc\xff", "",
		), Want: "abc"},
		{Name: "To_Valid_UTF8 surrogate", Got: standard_to_valid_utf8(
			"\xed\xa0\x80", "abc",
		), Want: "abc"},
	}
	for _, test := range tests {
		check_standard_example(t, test.Name, test.Got, test.Want)
	}
	replacer := standard_new_replacer("<", "&lt;", ">", "&gt;")
	check_standard_example(t, "New_Replacer",
		standard_replacer_replace(replacer, "This is <b>HTML</b>!"),
		"This is &lt;b&gt;HTML&lt;/b&gt;!")
}

// Test_Standard_Library_Examples_Trim preserves the upstream trim examples.
func Test_Standard_Library_Examples_Trim(t *testing.T) {
	tests := []struct {
		Name string
		Got  string
		Want string
	}{
		{Name: "Trim", Got: standard_trim("¡¡¡Hello, Gophers!!!", "!¡"),
			Want: "Hello, Gophers"},
		{Name: "Trim_Space", Got: standard_trim_space(" \t\n Hello, Gophers \n\t\r\n"),
			Want: "Hello, Gophers"},
		{Name: "Trim_Function", Got: standard_trim_function(
			"¡¡¡Hello, Gophers!!!", standard_is_not_letter_or_number,
		), Want: "Hello, Gophers"},
		{Name: "Trim_Left", Got: standard_trim_left("¡¡¡Hello, Gophers!!!", "!¡"),
			Want: "Hello, Gophers!!!"},
		{Name: "Trim_Left_Function", Got: standard_trim_left_function(
			"¡¡¡Hello, Gophers!!!", standard_is_not_letter_or_number,
		), Want: "Hello, Gophers!!!"},
		{Name: "Trim_Right", Got: standard_trim_right("¡¡¡Hello, Gophers!!!", "!¡"),
			Want: "¡¡¡Hello, Gophers"},
		{Name: "Trim_Right_Function", Got: standard_trim_right_function(
			"¡¡¡Hello, Gophers!!!", standard_is_not_letter_or_number,
		), Want: "¡¡¡Hello, Gophers"},
	}
	for _, test := range tests {
		check_standard_example(t, test.Name, test.Got, test.Want)
	}
	prefix := standard_trim_prefix("¡¡¡Hello, Gophers!!!", "¡¡¡Hello, ")
	prefix = standard_trim_prefix(prefix, "¡¡¡Howdy, ")
	check_standard_example(t, "Trim_Prefix", prefix, "Gophers!!!")
	suffix := standard_trim_suffix("¡¡¡Hello, Gophers!!!", ", Gophers!!!")
	suffix = standard_trim_suffix(suffix, ", Marmots!!!")
	check_standard_example(t, "Trim_Suffix", suffix, "¡¡¡Hello")
}

// Test_Standard_Library_Examples_Sequences preserves the upstream iterator examples.
func Test_Standard_Library_Examples_Sequences(t *testing.T) {
	check_standard_example_texts(
		t, "Lines", standard_contains_sequence(standard_lines(
			"Hello\nWorld\nGo Programming\n",
		)), []string{"Hello\n", "World\n", "Go Programming\n"},
	)
	check_standard_example_texts(
		t, "Split_Sequence", standard_contains_sequence(
			standard_split_sequence("a,b,c,d", ","),
		), []string{"a", "b", "c", "d"},
	)
	check_standard_example_texts(
		t, "Split_After_Sequence", standard_contains_sequence(
			standard_split_after_sequence("a,b,c,d", ","),
		), []string{"a,", "b,", "c,", "d"},
	)
	check_standard_example_texts(
		t, "Fields_Sequence", standard_contains_sequence(
			standard_fields_sequence("  lots   of   spaces  "),
		), []string{"lots", "of", "spaces"},
	)
	check_standard_example_texts(
		t, "Fields_Function_Sequence", standard_contains_sequence(
			standard_fields_function_sequence("abc123def456ghi", standard_is_digit),
		), []string{"abc", "def", "ghi"},
	)
}

// Test_Standard_Library_Examples_Stateful preserves the upstream stateful examples.
// Stateful operations stay on shared/io.Stream and do not add standard interfaces.
func Test_Standard_Library_Examples_Stateful(t *testing.T) {
	source := "abc"
	clone := standard_clone(source)
	check_standard_example(t, "Clone value", clone, source)
	check_standard_example(
		t, "Clone storage", unsafe.StringData(clone) == unsafe.StringData(source), false,
	)
	var builder shared_strings.Builder
	for count := 3; count >= 1; count-- {
		digits := shared_strconv.Format_Decimal(shared_strconv.Machine_Integer(count))
		_, err := standard_builder_write(&builder, string(digits)+"...")
		if err != nil {
			t.Fatalf("Builder example write returned %v", err)
		}
	}
	if _, err := standard_builder_write(&builder, "ignition"); err != nil {
		t.Fatalf("Builder example final write returned %v", err)
	}
	check_standard_example(
		t, "Builder", string(shared_strings.Builder_Text(&builder)), "3...2...1...ignition",
	)
	short_reader := shared_strings.New_Reader("Hi!")
	check_standard_example(
		t, "Reader ASCII size", int(shared_strings.Reader_Unread_Size(short_reader)), 3,
	)
	wide_reader := shared_strings.New_Reader("こんにちは!")
	check_standard_example(
		t, "Reader Unicode size", int(shared_strings.Reader_Unread_Size(wide_reader)), 16,
	)
}
