package bytes_test

import (
	"reflect"
	"testing"

	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/testify"
	"local/james-orcales/shared/unicode/ucd"
)

// Test_Allocation proves each exported operation owns no heap storage.
func Test_Allocation(t *testing.T) {
	verify_api_is_zero_allocation(t)
}

// Test_Constant_Facts protects shared byte, result, and cursor boundaries.
func Test_Constant_Facts(t *testing.T) {
	if bytes.TEXT_SIZE_MAXIMUM != bytes.SLICE_SIZE_MAXIMUM {
		t.Fatal("Text and Slice maximums differ")
	}
	if bytes.SLICES_COUNT_MAXIMUM != bytes.SLICE_SIZE_MAXIMUM+1 {
		t.Fatal("split count does not include empty boundaries")
	}
	if bytes.COUNT_VALUE_MAXIMUM != bytes.SLICE_SIZE_MAXIMUM {
		t.Fatal("item count maximum differs from source maximum")
	}
	if bytes.OCCURRENCE_COUNT_MAXIMUM != bytes.SLICES_COUNT_MAXIMUM {
		t.Fatal("occurrence count missed empty boundaries")
	}
}

// Test_Comparison protects byte equality, lexical order, and Unicode folding.
func Test_Comparison(t *testing.T) {
	testify.True(t, bool(bytes.Equal(nil, []byte{})))
	testify.False(t, bool(bytes.Equal([]byte("a"), []byte("b"))))
	testify.Equal(t, bytes.Order(bytes.ORDER_BEFORE), bytes.Compare([]byte("ab"), []byte("ac")))
	testify.Equal(t, bytes.Order(bytes.ORDER_AFTER), bytes.Compare([]byte("ac"), []byte("ab")))
	testify.Equal(t, bytes.Order(bytes.ORDER_EQUAL), bytes.Compare([]byte("ab"), []byte("ab")))
	testify.True(t, bool(bytes.Equal_Fold([]byte("Go"), []byte("gO"))))
	storage := []byte("abcdef")
	testify.True(t, bool(bytes.Overlap(storage[:3], storage[2:])))
	testify.False(t, bool(bytes.Overlap(storage[:3], []byte("abc"))))
	testify.False(t, bool(bytes.Overlap(storage[:0], storage)))
	testify.True(t, bool(bytes.Has_Text_Suffix([]byte("archive.txt"), ".txt")))
	testify.False(t, bool(bytes.Has_Text_Suffix([]byte("archive.txt"), ".zip")))
	testify.False(t, bool(bytes.Has_Text_Suffix([]byte("x"), "longer")))
}

// Test_Search protects every search form and empty UTF-8 boundaries.
func Test_Search(t *testing.T) {
	source := []byte("a☺b-a")
	if bytes.Count(source, []byte("a")) != 2 {
		t.Fatal("Count returned wrong match count")
	}
	if bytes.Count([]byte("a☺"), nil) != 3 {
		t.Fatal("Count missed UTF-8 boundaries")
	}
	if !bytes.Contains(source, []byte("☺b")) {
		t.Fatal("Contains missed Slice")
	}
	if !bytes.Contains_Any(source, "x☺") {
		t.Fatal("Contains_Any missed character")
	}
	if !bytes.Contains_Rune(source, '☺') {
		t.Fatal("Contains_Rune missed character")
	}
	if !bytes.Contains_Function(source, is_dash) {
		t.Fatal("Contains_Function missed predicate")
	}
	if bytes.Index(source, []byte("b")) != 4 {
		t.Fatal("Index returned wrong byte index")
	}
	if bytes.Index_Byte(source, 'a') != 0 {
		t.Fatal("Index_Byte missed first byte")
	}
	if bytes.Last_Index(source, []byte("a")) != 6 {
		t.Fatal("Last_Index missed final Slice")
	}
	if bytes.Last_Index_Byte(source, 'a') != 6 {
		t.Fatal("Last_Index_Byte missed final byte")
	}
	if bytes.Index_Rune(source, '☺') != 1 {
		t.Fatal("Index_Rune returned wrong byte index")
	}
	if bytes.Index_Any(source, "☺x") != 1 {
		t.Fatal("Index_Any returned wrong byte index")
	}
	if bytes.Last_Index_Any(source, "a") != 6 {
		t.Fatal("Last_Index_Any missed final character")
	}
	if bytes.Index_Function(source, is_dash) != 5 {
		t.Fatal("Index_Function returned wrong byte index")
	}
	if bytes.Last_Index_Function(source, is_letter_a) != 6 {
		t.Fatal("Last_Index_Function missed final predicate match")
	}
}

// Test_Split_And_Join protects caller slots, limits, fields, views, and join storage.
func Test_Split_And_Join(t *testing.T) {
	var slots [TEST_SLOT_COUNT][]byte
	source := []byte("a,b,c")
	count := bytes.Split_Into(slots[:], source, []byte(","))
	assert_slices(t, slots[:int(count)], [][]byte{[]byte("a"), []byte("b"), []byte("c")})
	count = bytes.Split_N_Into(slots[:], source, []byte(","), 2)
	assert_slices(t, slots[:int(count)], [][]byte{[]byte("a"), []byte("b,c")})
	count = bytes.Split_After_Into(slots[:], source, []byte(","))
	assert_slices(t, slots[:int(count)], [][]byte{[]byte("a,"), []byte("b,"), []byte("c")})
	count = bytes.Split_After_N_Into(slots[:], source, []byte(","), 2)
	assert_slices(t, slots[:int(count)], [][]byte{[]byte("a,"), []byte("b,c")})
	count = bytes.Split_Into(slots[:], []byte("a☺"), nil)
	assert_slices(t, slots[:int(count)], [][]byte{[]byte("a"), []byte("☺")})
	var fields [TEST_SLOT_COUNT][]byte
	field_count := bytes.Fields_Into(fields[:], []byte(" a\tb "))
	assert_slices(t, fields[:int(field_count)], [][]byte{[]byte("a"), []byte("b")})
	field_count = bytes.Fields_Function_Into(fields[:], []byte("a-b--c"), is_dash)
	assert_slices(t, fields[:int(field_count)], [][]byte{[]byte("a"), []byte("b"), []byte("c")})
	var storage [TEST_STORAGE_COUNT]byte
	joined := bytes.Join_Into(storage[:], slots[:3], []byte("-"))
	if string(storage[:int(joined)]) != "a-☺-c" {
		t.Fatal("Join_Into wrote wrong content")
	}
	if cap(slots[0]) != len(slots[0]) {
		t.Fatal("Split_Into returned unclipped view")
	}
}

// Test_Transform protects mapped, repeated, case, repaired, and rune output.
func Test_Transform(t *testing.T) {
	var storage [TEST_STORAGE_COUNT]byte
	count := bytes.Map_Into(storage[:], map_character, []byte("abx"))
	if string(storage[:int(count)]) != "☺b" {
		t.Fatal("Map_Into wrote wrong content")
	}
	count = bytes.Repeat_Into(storage[:], []byte("ab"), 3)
	if string(storage[:int(count)]) != "ababab" {
		t.Fatal("Repeat_Into wrote wrong content")
	}
	count = bytes.To_Upper_Into(storage[:], []byte("Go"))
	if string(storage[:int(count)]) != "GO" {
		t.Fatal("To_Upper_Into wrote wrong content")
	}
	count = bytes.To_Lower_Into(storage[:], []byte("Go"))
	if string(storage[:int(count)]) != "go" {
		t.Fatal("To_Lower_Into wrote wrong content")
	}
	count = bytes.To_Title_Into(storage[:], []byte("ǳ"))
	if string(storage[:int(count)]) != "ǲ" {
		t.Fatal("To_Title_Into wrote wrong content")
	}
	var special_storage [ucd.SPECIAL_CASE_COUNT_MAXIMUM]ucd.Case_Range
	special := ucd.Special_Case(ucd.Turkish_Case(special_storage[:]))
	count = bytes.To_Upper_Special_Into(storage[:], special, []byte("i"))
	if string(storage[:int(count)]) != "İ" {
		t.Fatal("To_Upper_Special_Into ignored special mapping")
	}
	count = bytes.To_Lower_Special_Into(storage[:], special, []byte("I"))
	if string(storage[:int(count)]) != "ı" {
		t.Fatal("To_Lower_Special_Into ignored special mapping")
	}
	count = bytes.To_Title_Special_Into(storage[:], special, []byte("i"))
	if string(storage[:int(count)]) != "İ" {
		t.Fatal("To_Title_Special_Into ignored special mapping")
	}
	count = bytes.To_Valid_UTF8_Into(storage[:], []byte{'a', 0xff, 0xfe, 'b'}, []byte("?"))
	if string(storage[:int(count)]) != "a?b" {
		t.Fatal("To_Valid_UTF8_Into wrote wrong repair")
	}
	count = bytes.Title_Into(storage[:], []byte("go gopher"))
	if string(storage[:int(count)]) != "Go Gopher" {
		t.Fatal("Title_Into wrote wrong content")
	}
	var characters [TEST_CHARACTER_COUNT]rune
	character_count := bytes.Runes_Into(characters[:], []byte{'a', 0xff})
	want_characters := []rune{'a', '�'}
	if !reflect.DeepEqual(characters[:int(character_count)], want_characters) {
		t.Fatal("Runes_Into wrote wrong characters")
	}
}

// Test_Trim protects predicate, cut-set, space, prefix, and suffix views.
func Test_Trim(t *testing.T) {
	source := []byte("xx abc xx")
	if string(bytes.Trim(source, "x")) != " abc " {
		t.Fatal("Trim returned wrong view")
	}
	if string(bytes.Trim_Left(source, "x")) != " abc xx" {
		t.Fatal("Trim_Left returned wrong view")
	}
	if string(bytes.Trim_Right(source, "x")) != "xx abc " {
		t.Fatal("Trim_Right returned wrong view")
	}
	if string(bytes.Trim_Function([]byte("--a--"), is_dash)) != "a" {
		t.Fatal("Trim_Function returned wrong view")
	}
	if string(bytes.Trim_Left_Function([]byte("--a"), is_dash)) != "a" {
		t.Fatal("Trim_Left_Function returned wrong view")
	}
	if string(bytes.Trim_Right_Function([]byte("a--"), is_dash)) != "a" {
		t.Fatal("Trim_Right_Function returned wrong view")
	}
	if string(bytes.Trim_Space([]byte(" \ta\n"))) != "a" {
		t.Fatal("Trim_Space returned wrong view")
	}
	if string(bytes.Trim_Prefix([]byte("abc"), []byte("a"))) != "bc" {
		t.Fatal("Trim_Prefix returned wrong view")
	}
	if string(bytes.Trim_Suffix([]byte("abc"), []byte("c"))) != "ab" {
		t.Fatal("Trim_Suffix returned wrong view")
	}
}

// Test_Replace protects bounded, unlimited, and empty-match substitutions.
func Test_Replace(t *testing.T) {
	var storage [TEST_STORAGE_COUNT]byte
	count := bytes.Replace_Into(storage[:], []byte("abcabc"), []byte("a"), []byte("z"), 1)
	if string(storage[:int(count)]) != "zbcabc" {
		t.Fatal("Replace_Into ignored count")
	}
	count = bytes.Replace_All_Into(storage[:], []byte("abcabc"), []byte("a"), []byte("z"))
	if string(storage[:int(count)]) != "zbczbc" {
		t.Fatal("Replace_All_Into missed match")
	}
	count = bytes.Replace_All_Into(storage[:], []byte("ab"), nil, []byte("-"))
	if string(storage[:int(count)]) != "-a-b-" {
		t.Fatal("Replace_All_Into missed empty boundaries")
	}
}

// Test_Cut_And_Clone protects aliased cuts and caller-owned clones.
func Test_Cut_And_Clone(t *testing.T) {
	source := []byte("a,b")
	before, after, found := bytes.Cut(source, []byte(","))
	if !found {
		t.Fatal("Cut returned wrong views")
	}
	if string(before) != "a" {
		t.Fatal("Cut returned wrong views")
	}
	if string(after) != "b" {
		t.Fatal("Cut returned wrong views")
	}
	after, found = bytes.Cut_Prefix(source, []byte("a"))
	if !found {
		t.Fatal("Cut_Prefix returned wrong view")
	}
	if string(after) != ",b" {
		t.Fatal("Cut_Prefix returned wrong view")
	}
	before, found = bytes.Cut_Suffix(source, []byte("b"))
	if !found {
		t.Fatal("Cut_Suffix returned wrong view")
	}
	if string(before) != "a," {
		t.Fatal("Cut_Suffix returned wrong view")
	}
	var destination [TEST_STORAGE_COUNT]byte
	count := bytes.Clone_Into(destination[:], source)
	destination[0] = 'z'
	if source[0] != 'a' {
		t.Fatal("Clone_Into did not copy into caller storage")
	}
	if count != 3 {
		t.Fatal("Clone_Into did not copy into caller storage")
	}
}

// Test_Iteration protects synchronous order, clipped views, and early stop.
func Test_Iteration(t *testing.T) {
	var yielded [TEST_SLOT_COUNT][]byte
	yielded_count := 0
	count := bytes.Lines([]byte("a\nb"), func(line bytes.Slice) (continued bytes.Boolean) {
		yielded[yielded_count] = line
		yielded_count++
		return true
	})
	assert_slices(t, yielded[:int(count)], [][]byte{[]byte("a\n"), []byte("b")})
	yielded_count = 0
	count = bytes.Split_Sequence(
		[]byte("a,b,c"), []byte(","),
		func(part bytes.Slice) (continued bytes.Boolean) {
			yielded[yielded_count] = part
			yielded_count++
			return yielded_count < 2
		},
	)
	if count != 2 {
		t.Fatal("Split_Sequence ignored early stop")
	}
	yielded_count = 0
	count = bytes.Split_After_Sequence(
		[]byte("a,b"), []byte(","), collect_yield(yielded[:], &yielded_count),
	)
	assert_slices(t, yielded[:int(count)], [][]byte{[]byte("a,"), []byte("b")})
	yielded_count = 0
	field_count := bytes.Fields_Sequence(
		[]byte(" a b "), collect_yield(yielded[:], &yielded_count),
	)
	assert_slices(t, yielded[:int(field_count)], [][]byte{[]byte("a"), []byte("b")})
	yielded_count = 0
	field_count = bytes.Fields_Function_Sequence(
		[]byte("a-b"), is_dash, collect_yield(yielded[:], &yielded_count),
	)
	assert_slices(t, yielded[:int(field_count)], [][]byte{[]byte("a"), []byte("b")})
}

// Test_Buffer protects caller storage, compaction, writes, reads, and unread state.
func Test_Buffer(t *testing.T) {
	test_buffer_reads(t)
	test_buffer_writes(t)
}

// Test_Reader protects caller reads, rune state, positioned reads, seek, and reset.
func Test_Reader(t *testing.T) {
	var reader bytes.Reader
	bytes.Reader_Reset(&reader, []byte("a☺b"))
	if bytes.Reader_Size(&reader) != 5 {
		t.Fatal("Reader size reports are wrong")
	}
	if bytes.Reader_Unread_Size(&reader) != 5 {
		t.Fatal("Reader size reports are wrong")
	}
	character, size, found := bytes.Reader_Read_Character(&reader)
	if !found {
		t.Fatal("Reader_Read_Character returned wrong character")
	}
	if character != 'a' {
		t.Fatal("Reader_Read_Character returned wrong character")
	}
	if size != 1 {
		t.Fatal("Reader_Read_Character returned wrong character")
	}
	bytes.Reader_Unread_Character(&reader)
	value, found := bytes.Reader_Read_Byte(&reader)
	if !found {
		t.Fatal("Reader_Read_Byte returned wrong byte")
	}
	if value != 'a' {
		t.Fatal("Reader_Read_Byte returned wrong byte")
	}
	bytes.Reader_Unread_Byte(&reader)
	var destination [TEST_READ_COUNT]byte
	if bytes.Reader_Read_Into(&reader, destination[:]) != TEST_READ_COUNT {
		t.Fatal("Reader_Read_Into returned wrong count")
	}
	if bytes.Reader_Read_At_Into(&reader, destination[:1], 4) != 1 {
		t.Fatal("Reader_Read_At_Into returned wrong count")
	}
	if destination[0] != 'b' {
		t.Fatal("Reader_Read_At_Into copied wrong byte")
	}
	if bytes.Reader_Seek(&reader, -1, bytes.SEEK_FROM_END) != 4 {
		t.Fatal("Reader_Seek returned wrong position")
	}
	bytes.Reader_Reset(&reader, []byte("xy"))
	if bytes.Reader_Unread_Size(&reader) != 2 {
		t.Fatal("Reader_Reset retained cursor")
	}
}

// Test_Size_Limits protects package boundaries against oversized malicious input.
func Test_Size_Limits(t *testing.T) {
	var oversized [bytes.SLICE_SIZE_MAXIMUM + 1]byte
	if panic_text(func() { bytes.Equal(oversized[:], nil) }) == "" {
		t.Fatal("Equal accepted oversized Slice")
	}
	var storage [TEST_SINGLE_COUNT]byte
	if panic_text(func() { bytes.Clone_Into(storage[:], []byte("ab")) }) == "" {
		t.Fatal("Clone_Into accepted short destination")
	}
}

// Test_Domain_Errors protects invalid limits, overlaps, cursor states, and storage.
func Test_Domain_Errors(t *testing.T) {
	var storage [TEST_STORAGE_COUNT]byte
	var slots [TEST_SINGLE_COUNT][]byte
	var buffer bytes.Buffer
	var reader bytes.Reader
	bytes.Reader_Reset(&reader, []byte("a"))
	panic_cases := [...]func(){
		func() { bytes.Split_N_Into(slots[:], []byte("a"), []byte(","), -2) },
		func() { bytes.Repeat_Into(storage[:], []byte("a"), -1) },
		func() { bytes.Replace_Into(storage[:], []byte("a"), nil, nil, -2) },
		func() { bytes.Buffer_Init(&buffer, storage[:1], []byte("ab")) },
		func() { bytes.Buffer_Unread_Byte(&buffer) },
		func() { bytes.Reader_Unread_Byte(&reader) },
		func() { bytes.Reader_Seek(&reader, -1, bytes.SEEK_FROM_START) },
		func() { bytes.Join_Into(storage[:], bytes.Slices{storage[:1]}, nil) },
	}
	for index, action := range panic_cases {
		if panic_text(action) == "" {
			t.Fatalf("domain error %d did not panic", index)
		}
	}
}

// Test_Invariant_Domains protects every declared typed boundary through public operations.
func Test_Invariant_Domains(t *testing.T) {
	fixture := domain_fixture_value()
	cover_boolean_domains(&fixture)
	cover_order_domains(&fixture)
	cover_count_domains(&fixture)
	cover_search_domains(&fixture)
	cover_state_domains(&fixture)
}

func test_buffer_reads(t *testing.T) {
	var storage [TEST_STORAGE_COUNT]byte
	var buffer bytes.Buffer
	if bytes.Buffer_Init(&buffer, storage[:], []byte("a☺b")) != 5 {
		t.Fatal("Buffer_Init returned wrong count")
	}
	if bytes.Buffer_Capacity(&buffer) != TEST_STORAGE_COUNT {
		t.Fatal("Buffer_Capacity returned wrong value")
	}
	if string(bytes.Buffer_Peek(&buffer, 1)) != "a" {
		t.Fatal("Buffer_Peek returned wrong view")
	}
	value, found := bytes.Buffer_Read_Byte(&buffer)
	if !found {
		t.Fatal("Buffer_Read_Byte returned wrong byte")
	}
	if value != 'a' {
		t.Fatal("Buffer_Read_Byte returned wrong byte")
	}
	bytes.Buffer_Unread_Byte(&buffer)
	character, size, found := bytes.Buffer_Read_Character(&buffer)
	if !found {
		t.Fatal("Buffer_Read_Character returned wrong character")
	}
	if character != 'a' {
		t.Fatal("Buffer_Read_Character returned wrong character")
	}
	if size != 1 {
		t.Fatal("Buffer_Read_Character returned wrong character")
	}
	character, size, found = bytes.Buffer_Read_Character(&buffer)
	if !found {
		t.Fatal("Buffer_Read_Character returned wrong UTF-8 character")
	}
	if character != '☺' {
		t.Fatal("Buffer_Read_Character returned wrong UTF-8 character")
	}
	if size != 3 {
		t.Fatal("Buffer_Read_Character returned wrong UTF-8 character")
	}
	bytes.Buffer_Unread_Character(&buffer)
	if bytes.Buffer_Size(&buffer) != 4 {
		t.Fatal("Buffer_Unread_Character restored wrong position")
	}
	var read [TEST_READ_COUNT]byte
	if bytes.Buffer_Read_Into(&buffer, read[:]) != TEST_READ_COUNT {
		t.Fatal("Buffer_Read_Into returned wrong count")
	}
	bytes.Buffer_Init_Text(&buffer, storage[:], "a,b")
	line, found := bytes.Buffer_Read_Until(&buffer, ',')
	if !found {
		t.Fatal("Buffer_Read_Until returned wrong view")
	}
	if string(line) != "a," {
		t.Fatal("Buffer_Read_Until returned wrong view")
	}
	if string(bytes.Buffer_Next(&buffer, 1)) != "b" {
		t.Fatal("Buffer_Next returned wrong view")
	}
}

func test_buffer_writes(t *testing.T) {
	var storage [TEST_STORAGE_COUNT]byte
	var buffer bytes.Buffer
	bytes.Buffer_Init(&buffer, storage[:], []byte("a"))
	bytes.Buffer_Grow(&buffer, 4)
	if bytes.Buffer_Write(&buffer, []byte("xy")) != 2 {
		t.Fatal("Buffer_Write returned wrong count")
	}
	if bytes.Buffer_Write_Text(&buffer, "z") != 1 {
		t.Fatal("Buffer_Write_Text returned wrong count")
	}
	bytes.Buffer_Write_Byte(&buffer, '!')
	if bytes.Buffer_Write_Character(&buffer, '☺') != 3 {
		t.Fatal("Buffer_Write_Character returned wrong count")
	}
	if bytes.Buffer_Available(&buffer) == bytes.Buffer_Capacity(&buffer) {
		t.Fatal("Buffer_Available ignored content")
	}
	if len(bytes.Buffer_Available_Slice(&buffer)) != 0 {
		t.Fatal("Buffer_Available_Slice exposed readable bytes")
	}
	bytes.Buffer_Reset(&buffer)
	if len(bytes.Buffer_Bytes(&buffer)) != 0 {
		t.Fatal("Buffer_Reset retained readable bytes")
	}
	bytes.Buffer_Init(&buffer, storage[:], []byte("abcd"))
	bytes.Buffer_Truncate(&buffer, 2)
	if string(bytes.Buffer_Bytes(&buffer)) != "ab" {
		t.Fatal("Buffer_Truncate retained wrong prefix")
	}
}

const TEST_STORAGE_COUNT = 64

const TEST_SLOT_COUNT = 8

const TEST_CHARACTER_COUNT = 4

const TEST_READ_COUNT = 2

const TEST_SINGLE_COUNT = 1

const TEST_TRIPLE_COUNT = 3

const TEST_FIVE_COUNT = 5

func assert_slices(t *testing.T, got [][]byte, want [][]byte) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Slices = %q, want %q", got, want)
	}
}

func collect_yield(
	destination [][]byte, count *int,
) (yield bytes.Yield_Function) {
	return func(content bytes.Slice) (continued bytes.Boolean) {
		destination[*count] = content
		*count++
		return true
	}
}

func is_dash(character rune) (matches bool) {
	return character == '-'
}

func is_letter_a(character rune) (matches bool) {
	return character == 'a'
}

func is_zero(character rune) (matches bool) {
	return character == 0
}

func is_one(character rune) (matches bool) {
	return character == 1
}

func is_two(character rune) (matches bool) {
	return character == 2
}

func is_replacement(character rune) (matches bool) {
	return character == '�'
}

func is_space(character rune) (matches bool) {
	return character == ' '
}

func never_match(_ rune) (matches bool) {
	return false
}

func identity_character(character rune) (mapped rune) {
	return character
}

func reject_yield(_ bytes.Slice) (continued bytes.Boolean) {
	return false
}

func map_character(character rune) (mapped_character rune) {
	if character == 'a' {
		return '☺'
	}
	if character == 'x' {
		return -1
	}
	return character
}

func panic_text(action func()) (message string) {
	defer func() {
		if recover() != nil {
			message = "panicked"
		}
	}()
	action()
	return ""
}

type domain_fixture struct {
	Maximum     [bytes.SLICE_SIZE_MAXIMUM]byte
	Alternate   [bytes.SLICE_SIZE_MAXIMUM]byte
	Lines       [bytes.SLICE_SIZE_MAXIMUM]byte
	Fields      [bytes.SLICE_SIZE_MAXIMUM]byte
	Slots       [bytes.SLICES_COUNT_MAXIMUM][]byte
	Field_Slots [bytes.FIELDS_COUNT_MAXIMUM][]byte
	Characters  [bytes.CHARACTERS_COUNT_MAXIMUM]rune
}

func domain_fixture_value() (fixture domain_fixture) {
	for index := range fixture.Maximum {
		fixture.Maximum[index] = 0
		fixture.Alternate[index] = 0
		fixture.Lines[index] = '\n'
		fixture.Fields[index] = 'a'
		if index%2 == 1 {
			fixture.Fields[index] = ' '
		}
	}
	fixture.Maximum[1] = 1
	fixture.Maximum[2] = 2
	fixture.Maximum[bytes.INDEX_MAXIMUM] = 255
	return fixture
}

func cover_boolean_domains(fixture *domain_fixture) {
	cover_slice_input_domains(fixture)
	cover_boolean_query_results(fixture)
	cover_boolean_read_results(fixture)
	cover_boolean_yield_results(fixture)
}

func cover_order_domains(fixture *domain_fixture) {
	bytes.Compare(nil, nil)
	bytes.Compare(fixture.Maximum[:1], fixture.Maximum[:2])
	bytes.Compare(fixture.Maximum[:2], fixture.Maximum[:1])
}

func cover_count_domains(fixture *domain_fixture) {
	cover_count_results(fixture)
	cover_collection_domains(fixture)
	cover_output_count_domains(fixture)
	cover_cut_output_domains(fixture)
	cover_transform_maximum_outputs(fixture)
	cover_argument_output_domains(fixture)
	cover_trim_output_domains(fixture)
}

func cover_search_domains(fixture *domain_fixture) {
	cover_search_result_domains(fixture)
	cover_scalar_input_domains(fixture)
}

func cover_state_domains(fixture *domain_fixture) {
	cover_buffer_domains(fixture)
	cover_reader_domains(fixture)
	cover_transform_special_domains(fixture)
}

func cover_slice_input_domains(fixture *domain_fixture) {
	sources := [...]bytes.Slice{
		nil, fixture.Maximum[:1], fixture.Maximum[:2], fixture.Maximum[:],
	}
	destinations := [...]bytes.Slice{
		nil, fixture.Alternate[:1], fixture.Alternate[:2], fixture.Alternate[:],
	}
	for index, source := range sources {
		cover_query_slice_inputs(source)
		cover_output_slice_inputs(fixture, destinations[index], source)
		cover_view_slice_inputs(source)
	}
}

func cover_query_slice_inputs(source bytes.Slice) {
	text := bytes.Text(string(source))
	bytes.Equal(source, source)
	bytes.Compare(source, source)
	bytes.Count(source, source)
	bytes.Contains(source, source)
	bytes.Contains_Any(source, text)
	bytes.Contains_Rune(source, 0)
	bytes.Contains_Function(source, is_zero)
	bytes.Index_Byte(source, 0)
	bytes.Last_Index(source, source)
	bytes.Last_Index_Byte(source, 0)
	bytes.Index_Rune(source, 0)
	bytes.Index_Any(source, text)
	bytes.Last_Index_Any(source, text)
	bytes.Has_Prefix(source, source)
	bytes.Has_Suffix(source, source)
	bytes.Has_Text_Suffix(source, text)
	bytes.Index_Function(source, is_zero)
	bytes.Last_Index_Function(source, is_zero)
	bytes.Equal_Fold(source, source)
	bytes.Index(source, source)
}

func cover_output_slice_inputs(
	fixture *domain_fixture, destination bytes.Slice, source bytes.Slice,
) {
	parts := [...][]byte{source}
	panic_text(func() { bytes.Join_Into(destination, parts[:], source[:0]) })
	panic_text(func() { bytes.Map_Into(destination, identity_character, source) })
	panic_text(func() { bytes.Repeat_Into(destination, source, 1) })
	panic_text(func() { bytes.To_Upper_Into(destination, source) })
	panic_text(func() { bytes.To_Lower_Into(destination, source) })
	panic_text(func() { bytes.To_Title_Into(destination, source) })
	panic_text(func() { bytes.To_Upper_Special_Into(destination, nil, source) })
	panic_text(func() { bytes.To_Lower_Special_Into(destination, nil, source) })
	panic_text(func() { bytes.To_Title_Special_Into(destination, nil, source) })
	panic_text(func() { bytes.To_Valid_UTF8_Into(destination, source, nil) })
	panic_text(func() { bytes.Title_Into(destination, source) })
	panic_text(func() { bytes.Runes_Into(fixture.Characters[:len(source)], source) })
	panic_text(func() { bytes.Replace_Into(destination, source, nil, nil, -1) })
	panic_text(func() { bytes.Replace_All_Into(destination, source, nil, nil) })
	panic_text(func() { bytes.Clone_Into(destination, source) })
}

func cover_view_slice_inputs(source bytes.Slice) {
	text := bytes.Text(string(source))
	bytes.Trim_Left_Function(source, is_zero)
	bytes.Trim_Right_Function(source, is_zero)
	bytes.Trim_Function(source, is_zero)
	bytes.Trim_Prefix(source, source)
	bytes.Trim_Suffix(source, source)
	bytes.Trim(source, text)
	bytes.Trim_Left(source, text)
	bytes.Trim_Right(source, text)
	bytes.Trim_Space(source)
	bytes.Cut(source, source)
	bytes.Cut_Prefix(source, source)
	bytes.Cut_Suffix(source, source)
	bytes.Lines(source, zero_allocation_yield)
	bytes.Split_Sequence(source, nil, zero_allocation_yield)
	bytes.Split_After_Sequence(source, nil, zero_allocation_yield)
	bytes.Fields_Sequence(source, zero_allocation_yield)
	bytes.Fields_Function_Sequence(source, is_zero, zero_allocation_yield)
}

func cover_boolean_query_results(fixture *domain_fixture) {
	one := fixture.Maximum[:1]
	different := fixture.Alternate[:1]
	different[0] = 'x'
	bytes.Equal(one, different)
	bytes.Contains(one, different)
	bytes.Contains_Any(one, "x")
	bytes.Contains_Rune(one, 'x')
	bytes.Contains_Function(one, never_match)
	bytes.Has_Prefix(one, different)
	bytes.Has_Suffix(one, different)
	bytes.Equal_Fold(one, different)
	bytes.Cut(one, different)
	bytes.Cut_Prefix(one, different)
	bytes.Cut_Suffix(one, different)
	panic_text(func() { bytes.Map_Into(one, identity_character, one) })
	var title_destination [TEST_READ_COUNT]byte
	bytes.Title_Into(title_destination[:], []byte("aa"))
}

func cover_boolean_read_results(fixture *domain_fixture) {
	var buffer bytes.Buffer
	bytes.Buffer_Init(&buffer, fixture.Alternate[:], nil)
	bytes.Buffer_Read_Byte(&buffer)
	bytes.Buffer_Read_Character(&buffer)
	bytes.Buffer_Read_Until(&buffer, 0)
	bytes.Buffer_Init(&buffer, fixture.Alternate[:], fixture.Maximum[:1])
	bytes.Buffer_Read_Byte(&buffer)
	bytes.Buffer_Init(&buffer, fixture.Alternate[:], fixture.Maximum[:1])
	bytes.Buffer_Read_Character(&buffer)
	bytes.Buffer_Init(&buffer, fixture.Alternate[:], fixture.Maximum[:1])
	bytes.Buffer_Read_Until(&buffer, 0)
	var reader bytes.Reader
	bytes.Reader_Reset(&reader, nil)
	bytes.Reader_Read_Byte(&reader)
	bytes.Reader_Read_Character(&reader)
	bytes.Reader_Reset(&reader, fixture.Maximum[:1])
	bytes.Reader_Read_Byte(&reader)
	bytes.Reader_Reset(&reader, fixture.Maximum[:1])
	bytes.Reader_Read_Character(&reader)
}

func cover_boolean_yield_results(fixture *domain_fixture) {
	bytes.Lines(fixture.Maximum[:1], zero_allocation_yield)
	bytes.Lines(fixture.Maximum[:1], reject_yield)
	bytes.Split_N_Into(fixture.Slots[:], nil, nil, -1)
	bytes.Split_After_N_Into(fixture.Slots[:], nil, nil, -1)
}

func cover_count_results(fixture *domain_fixture) {
	bytes.Count(nil, fixture.Alternate[:1])
	bytes.Count(fixture.Maximum[:1], fixture.Maximum[:1])
	bytes.Count(fixture.Maximum[:2], fixture.Maximum[:1])
	bytes.Count(fixture.Maximum[:], nil)
	bytes.Lines(nil, zero_allocation_yield)
	bytes.Lines(fixture.Lines[:1], zero_allocation_yield)
	bytes.Lines(fixture.Lines[:2], zero_allocation_yield)
	bytes.Lines(fixture.Lines[:], zero_allocation_yield)
	bytes.Runes_Into(fixture.Characters[:], nil)
	bytes.Runes_Into(fixture.Characters[:], fixture.Maximum[:1])
	bytes.Runes_Into(fixture.Characters[:], fixture.Maximum[:2])
	bytes.Runes_Into(fixture.Characters[:], fixture.Maximum[:])
	bytes.Fields_Sequence(fixture.Fields[:3], zero_allocation_yield)
}

func cover_output_count_domains(fixture *domain_fixture) {
	cover_split_count_domains(fixture)
	cover_field_count_domains(fixture)
	cover_repeat_replacement_count_domains(fixture)
}

func cover_split_count_domains(fixture *domain_fixture) {
	bytes.Split_Into(fixture.Slots[:], nil, nil)
	bytes.Split_Into(fixture.Slots[:], fixture.Maximum[:1], nil)
	bytes.Split_Into(fixture.Slots[:], fixture.Maximum[:2], nil)
	bytes.Split_Into(fixture.Slots[:], fixture.Maximum[:], nil)
	bytes.Split_After_Into(fixture.Slots[:], nil, nil)
	bytes.Split_After_Into(fixture.Slots[:], fixture.Maximum[:1], nil)
	bytes.Split_After_Into(fixture.Slots[:], fixture.Maximum[:2], nil)
	bytes.Split_After_Into(fixture.Slots[:], fixture.Maximum[:], nil)
	bytes.Split_After_N_Into(fixture.Slots[:], fixture.Maximum[:], nil, -1)
	limits := [...]bytes.Limit{-1, 0, 1, 2, bytes.LIMIT_MAXIMUM}
	for _, limit := range limits {
		bytes.Split_N_Into(fixture.Slots[:], fixture.Maximum[:2], nil, limit)
		bytes.Split_After_N_Into(fixture.Slots[:], fixture.Maximum[:2], nil, limit)
	}
}

func cover_field_count_domains(fixture *domain_fixture) {
	bytes.Fields_Into(fixture.Field_Slots[:], nil)
	bytes.Fields_Into(fixture.Field_Slots[:], fixture.Fields[:1])
	bytes.Fields_Into(fixture.Field_Slots[:], fixture.Fields[:3])
	bytes.Fields_Into(fixture.Field_Slots[:], fixture.Fields[:bytes.INDEX_MAXIMUM])
	bytes.Fields_Function_Into(fixture.Field_Slots[:], nil, is_space)
	bytes.Fields_Function_Into(fixture.Field_Slots[:], fixture.Fields[:1], is_space)
	bytes.Fields_Function_Into(fixture.Field_Slots[:], fixture.Fields[:3], is_space)
	bytes.Fields_Function_Into(
		fixture.Field_Slots[:], fixture.Fields[:bytes.INDEX_MAXIMUM], is_space,
	)
	bytes.Fields_Sequence(fixture.Fields[:bytes.INDEX_MAXIMUM], zero_allocation_yield)
	bytes.Fields_Function_Sequence(
		fixture.Fields[:bytes.INDEX_MAXIMUM], is_space, zero_allocation_yield,
	)
}

func cover_repeat_replacement_count_domains(fixture *domain_fixture) {
	repeat_counts := [...]bytes.Repeat_Count{0, 1, 2, bytes.REPEAT_COUNT_MAXIMUM}
	for _, count := range repeat_counts {
		panic_text(func() { bytes.Repeat_Into(fixture.Alternate[:], nil, count) })
	}
	replacement_counts := [...]bytes.Replacement_Count{
		-1, 0, 1, 2, bytes.REPLACEMENT_COUNT_MAXIMUM,
	}
	for _, count := range replacement_counts {
		bytes.Replace_Into(fixture.Alternate[:1], fixture.Maximum[:1], nil, nil, count)
	}
}

func cover_collection_domains(fixture *domain_fixture) {
	cover_split_collection_domains(fixture)
	cover_field_collection_domains(fixture)
	cover_join_collection_domains(fixture)
	cover_character_collection_domains(fixture)
	cover_separator_size_domains(fixture)
}

func cover_split_collection_domains(fixture *domain_fixture) {
	destinations := [...]bytes.Slices{
		nil, fixture.Slots[:1], fixture.Slots[:2], fixture.Slots[:],
	}
	for _, destination := range destinations {
		bytes.Split_Into(destination, nil, nil)
		bytes.Split_N_Into(destination, nil, nil, -1)
		bytes.Split_After_Into(destination, nil, nil)
		bytes.Split_After_N_Into(destination, nil, nil, -1)
	}
	sources := [...]bytes.Slice{
		nil, fixture.Maximum[:1], fixture.Maximum[:2], fixture.Maximum[:],
	}
	for _, source := range sources {
		bytes.Split_Into(fixture.Slots[:], source, source)
		bytes.Split_N_Into(fixture.Slots[:], source, source, -1)
		bytes.Split_After_Into(fixture.Slots[:], source, source)
		bytes.Split_After_N_Into(fixture.Slots[:], source, source, -1)
	}
}

func cover_field_collection_domains(fixture *domain_fixture) {
	destinations := [...]bytes.Field_Slices{
		nil, fixture.Field_Slots[:1], fixture.Field_Slots[:2],
		fixture.Field_Slots[:],
	}
	for _, destination := range destinations {
		bytes.Fields_Into(destination, nil)
		bytes.Fields_Function_Into(destination, nil, is_space)
	}
	sources := [...]bytes.Slice{
		nil, fixture.Fields[:1], fixture.Fields[:2], fixture.Fields[:],
	}
	for _, source := range sources {
		bytes.Fields_Into(fixture.Field_Slots[:], source)
		bytes.Fields_Function_Into(fixture.Field_Slots[:], source, is_space)
	}
}

func cover_join_collection_domains(fixture *domain_fixture) {
	collections := [...]bytes.Slices{
		nil, fixture.Slots[:1], fixture.Slots[:2], fixture.Slots[:],
	}
	for _, parts := range collections {
		bytes.Join_Into(fixture.Alternate[:], parts, nil)
	}
	parts := [...][]byte{
		nil, fixture.Maximum[:1], fixture.Maximum[:2], fixture.Maximum[:],
	}
	for _, part := range parts {
		one_part := [...][]byte{part}
		bytes.Join_Into(fixture.Alternate[:], one_part[:], nil)
	}
}

func cover_character_collection_domains(fixture *domain_fixture) {
	destinations := [...]bytes.Characters{
		nil, fixture.Characters[:1], fixture.Characters[:2], fixture.Characters[:],
	}
	for _, destination := range destinations {
		bytes.Runes_Into(destination, nil)
	}
}

func cover_separator_size_domains(fixture *domain_fixture) {
	separators := [...]bytes.Slice{
		nil, fixture.Maximum[:1], fixture.Maximum[:2], fixture.Maximum[:],
	}
	for _, separator := range separators {
		bytes.Split_Sequence(separator, separator, zero_allocation_yield)
		bytes.Split_After_Sequence(separator, separator, zero_allocation_yield)
	}
}

func cover_transform_special_domains(fixture *domain_fixture) {
	var special_storage [ucd.SPECIAL_CASE_COUNT_MAXIMUM]ucd.Case_Range
	turkish := ucd.Special_Case(ucd.Turkish_Case(special_storage[:]))
	specials := [...]ucd.Special_Case{
		turkish[:0], turkish[:1], turkish[:2], turkish[:],
	}
	for _, special := range specials {
		bytes.To_Upper_Special_Into(fixture.Alternate[:0], special, nil)
		bytes.To_Lower_Special_Into(fixture.Alternate[:0], special, nil)
		bytes.To_Title_Special_Into(fixture.Alternate[:0], special, nil)
	}
}

func cover_search_result_domains(fixture *domain_fixture) {
	cover_first_index_domains(fixture)
	cover_last_index_domains(fixture)
	cover_function_index_domains(fixture)
}

func cover_first_index_domains(fixture *domain_fixture) {
	source := fixture.Maximum[:]
	bytes.Index(source, fixture.Alternate[:1])
	bytes.Index(source, source[:1])
	bytes.Index(source, source[1:2])
	bytes.Index(source, source[2:3])
	bytes.Index(source, source[bytes.INDEX_MAXIMUM:])
	bytes.Index_Byte(source, 'x')
	bytes.Index_Byte(source, 0)
	bytes.Index_Byte(source, 1)
	bytes.Index_Byte(source, 2)
	bytes.Index_Byte(source, 255)
	bytes.Index_Rune(source, 'x')
	bytes.Index_Rune(source, 0)
	bytes.Index_Rune(source, 1)
	bytes.Index_Rune(source, 2)
	bytes.Index_Rune(source, '�')
	bytes.Index_Any(source, "x")
	bytes.Index_Any(source, "\x00")
	bytes.Index_Any(source, "\x01")
	bytes.Index_Any(source, "\x02")
	bytes.Index_Any(source, "\xff")
}

func cover_last_index_domains(fixture *domain_fixture) {
	source := fixture.Maximum[:]
	bytes.Last_Index(source, fixture.Alternate[:1])
	bytes.Last_Index(source, source[:1])
	bytes.Last_Index(source, source[1:2])
	bytes.Last_Index(source, source[2:3])
	bytes.Last_Index(source, nil)
	bytes.Last_Index_Byte(source, 'x')
	bytes.Last_Index_Byte(source, 0)
	bytes.Last_Index_Byte(source, 1)
	bytes.Last_Index_Byte(source, 2)
	bytes.Last_Index_Byte(source, 255)
	bytes.Last_Index_Any(source, "x")
	bytes.Last_Index_Any(source, "\x00")
	bytes.Last_Index_Any(source, "\x01")
	bytes.Last_Index_Any(source, "\x02")
	bytes.Last_Index_Any(source, "\xff")
}

func cover_function_index_domains(fixture *domain_fixture) {
	source := fixture.Maximum[:]
	predicates := [...]func(rune) (matches bool){
		never_match, is_zero, is_one, is_two, is_replacement,
	}
	for _, predicate := range predicates {
		bytes.Index_Function(source, predicate)
		bytes.Last_Index_Function(source, predicate)
	}
}

func cover_scalar_input_domains(fixture *domain_fixture) {
	byte_values := [...]bytes.Byte{0, 1, 2, 255}
	for _, value := range byte_values {
		cover_byte_input_domains(fixture, value)
	}
	character_values := [...]bytes.Character{
		-2147483648, -1, 0, 1, 2, 2147483647,
	}
	for _, character := range character_values {
		cover_character_input_domains(fixture, character)
	}
	cover_reader_offset_domains(fixture)
}

func cover_byte_input_domains(fixture *domain_fixture, value bytes.Byte) {
	bytes.Index_Byte(nil, value)
	bytes.Last_Index_Byte(nil, value)
	var buffer bytes.Buffer
	bytes.Buffer_Init(&buffer, fixture.Alternate[:], nil)
	bytes.Buffer_Write_Byte(&buffer, value)
	bytes.Buffer_Init(&buffer, fixture.Alternate[:], nil)
	bytes.Buffer_Read_Until(&buffer, value)
}

func cover_character_input_domains(
	fixture *domain_fixture, character bytes.Character,
) {
	bytes.Contains_Rune(nil, character)
	bytes.Index_Rune(nil, character)
	var buffer bytes.Buffer
	bytes.Buffer_Init(&buffer, fixture.Alternate[:], nil)
	bytes.Buffer_Write_Character(&buffer, character)
}

func cover_reader_offset_domains(fixture *domain_fixture) {
	offsets := [...]bytes.Reader_Offset{
		bytes.Reader_Offset(-bytes.READER_POSITION_MAXIMUM), -1, 0, 1, 2,
		bytes.Reader_Offset(bytes.READER_POSITION_MAXIMUM),
	}
	var reader bytes.Reader
	bytes.Reader_Reset(&reader, fixture.Maximum[:])
	for _, offset := range offsets {
		panic_text(func() {
			bytes.Reader_Read_At_Into(&reader, fixture.Alternate[:1], offset)
		})
		panic_text(func() { bytes.Reader_Seek(&reader, offset, bytes.SEEK_FROM_START) })
	}
}

func cover_buffer_domains(fixture *domain_fixture) {
	cover_buffer_state_inputs(fixture)
	cover_buffer_boundary_outputs(fixture)
	cover_buffer_scalar_outputs(fixture)
}

func cover_buffer_state_inputs(fixture *domain_fixture) {
	states := [...]bytes.Buffer{
		{Content: fixture.Alternate[:0:0], Position: 0, Operation: -1},
		{Content: fixture.Alternate[:1:1], Position: 1, Operation: 0},
		{Content: fixture.Alternate[:2:2], Position: 2, Operation: 1},
		{Content: fixture.Alternate[:2:2], Position: 2, Operation: 2},
		{
			Content: fixture.Alternate[:], Position: bytes.BOUNDARY_INDEX_MAXIMUM,
			Operation: bytes.DECODED_SIZE_MAXIMUM,
		},
	}
	for _, state := range states {
		cover_buffer_state_queries(fixture, state)
		cover_buffer_state_edits(fixture, state)
		cover_buffer_state_reads(fixture, state)
	}
}

func cover_buffer_state_queries(fixture *domain_fixture, state bytes.Buffer) {
	candidate := state
	bytes.Buffer_Init(&candidate, fixture.Alternate[:], nil)
	candidate = state
	bytes.Buffer_Init_Text(&candidate, fixture.Alternate[:], "")
	candidate = state
	bytes.Buffer_Bytes(&candidate)
	candidate = state
	bytes.Buffer_Available_Slice(&candidate)
	candidate = state
	bytes.Buffer_Peek(&candidate, 0)
	candidate = state
	bytes.Buffer_Size(&candidate)
	candidate = state
	bytes.Buffer_Capacity(&candidate)
	candidate = state
	bytes.Buffer_Available(&candidate)
}

func cover_buffer_state_edits(fixture *domain_fixture, state bytes.Buffer) {
	candidate := state
	bytes.Buffer_Truncate(&candidate, 0)
	candidate = state
	bytes.Buffer_Reset(&candidate)
	candidate = state
	panic_text(func() { bytes.Buffer_Grow(&candidate, 0) })
	candidate = state
	panic_text(func() { bytes.Buffer_Write(&candidate, fixture.Maximum[:0]) })
	candidate = state
	panic_text(func() { bytes.Buffer_Write_Text(&candidate, "") })
	candidate = state
	panic_text(func() { bytes.Buffer_Write_Byte(&candidate, 0) })
	candidate = state
	panic_text(func() { bytes.Buffer_Write_Character(&candidate, 0) })
}

func cover_buffer_state_reads(fixture *domain_fixture, state bytes.Buffer) {
	candidate := state
	bytes.Buffer_Read_Into(&candidate, fixture.Maximum[:0])
	candidate = state
	bytes.Buffer_Next(&candidate, 0)
	candidate = state
	bytes.Buffer_Read_Byte(&candidate)
	candidate = state
	bytes.Buffer_Read_Character(&candidate)
	candidate = state
	panic_text(func() { bytes.Buffer_Unread_Character(&candidate) })
	candidate = state
	panic_text(func() { bytes.Buffer_Unread_Byte(&candidate) })
	candidate = state
	bytes.Buffer_Read_Until(&candidate, 0)
}

func cover_buffer_boundary_outputs(fixture *domain_fixture) {
	sources := [...]bytes.Slice{
		nil, fixture.Maximum[:1], fixture.Maximum[:2], fixture.Maximum[:],
	}
	for _, source := range sources {
		var buffer bytes.Buffer
		bytes.Buffer_Init(&buffer, fixture.Alternate[:len(source)], source)
		bytes.Buffer_Init_Text(
			&buffer, fixture.Alternate[:len(source)], bytes.Text(string(source)),
		)
		bytes.Buffer_Bytes(&buffer)
		bytes.Buffer_Size(&buffer)
		bytes.Buffer_Capacity(&buffer)
		bytes.Buffer_Peek(&buffer, bytes.Boundary(len(source)))
		bytes.Buffer_Next(&buffer, bytes.Boundary(len(source)))
		bytes.Buffer_Init(&buffer, fixture.Alternate[:], source)
		bytes.Buffer_Read_Into(&buffer, fixture.Maximum[:len(source)])
		bytes.Buffer_Init(&buffer, fixture.Alternate[:], nil)
		bytes.Buffer_Write(&buffer, source)
		bytes.Buffer_Init(&buffer, fixture.Alternate[:], nil)
		bytes.Buffer_Write_Text(&buffer, bytes.Text(string(source)))
	}
	cover_buffer_available_outputs(fixture)
	cover_buffer_size_inputs(fixture)
}

func cover_buffer_available_outputs(fixture *domain_fixture) {
	content_sizes := [...]int{
		bytes.SLICE_SIZE_MAXIMUM,
		bytes.SLICE_SIZE_MAXIMUM - 1,
		bytes.SLICE_SIZE_MAXIMUM - 2,
		0,
	}
	for _, content_size := range content_sizes {
		buffer := bytes.Buffer{Content: fixture.Alternate[:content_size]}
		bytes.Buffer_Available(&buffer)
		bytes.Buffer_Available_Slice(&buffer)
	}
}

func cover_buffer_size_inputs(fixture *domain_fixture) {
	values := [...]bytes.Boundary{0, 1, 2, bytes.BOUNDARY_INDEX_MAXIMUM}
	for _, value := range values {
		buffer := bytes.Buffer{Content: fixture.Alternate[:]}
		bytes.Buffer_Peek(&buffer, value)
		buffer = bytes.Buffer{Content: fixture.Alternate[:]}
		bytes.Buffer_Next(&buffer, value)
		buffer = bytes.Buffer{Content: fixture.Alternate[:]}
		bytes.Buffer_Truncate(&buffer, value)
	}
	growth := [...]bytes.Growth_Count{0, 1, 2, bytes.GROWTH_COUNT_MAXIMUM}
	for _, value := range growth {
		buffer := bytes.Buffer{Content: fixture.Alternate[:0]}
		bytes.Buffer_Grow(&buffer, value)
	}
}

func cover_buffer_scalar_outputs(fixture *domain_fixture) {
	cover_buffer_byte_outputs(fixture)
	cover_buffer_character_outputs(fixture)
	cover_buffer_until_outputs(fixture)
}

func cover_buffer_byte_outputs(fixture *domain_fixture) {
	positions := [...]int{0, 1, 2, bytes.INDEX_MAXIMUM}
	for _, position := range positions {
		var buffer bytes.Buffer
		bytes.Buffer_Init(
			&buffer, fixture.Alternate[:], fixture.Maximum[position:position+1],
		)
		bytes.Buffer_Read_Byte(&buffer)
	}
}

func cover_buffer_character_outputs(fixture *domain_fixture) {
	var buffer bytes.Buffer
	characters := [...]bytes.Character{'a', 0x80, 0x800, 0x10000}
	for _, character := range characters {
		bytes.Buffer_Init(&buffer, fixture.Alternate[:], nil)
		bytes.Buffer_Write_Character(&buffer, character)
	}
	encoded_zero := [TEST_SINGLE_COUNT]byte{0}
	encoded_one := [TEST_SINGLE_COUNT]byte{1}
	encoded_two := [TEST_SINGLE_COUNT]byte{2}
	encoded_size_two := [TEST_READ_COUNT]byte{0xc2, 0x80}
	encoded_size_three := [TEST_TRIPLE_COUNT]byte{0xe0, 0xa0, 0x80}
	encoded_maximum := [TEST_CHARACTER_COUNT]byte{0xf4, 0x8f, 0xbf, 0xbf}
	values := [...]bytes.Slice{
		encoded_zero[:], encoded_one[:], encoded_two[:], encoded_size_two[:],
		encoded_size_three[:], encoded_maximum[:],
	}
	for _, value := range values {
		bytes.Buffer_Init(&buffer, fixture.Alternate[:], value)
		bytes.Buffer_Read_Character(&buffer)
	}
}

func cover_buffer_until_outputs(fixture *domain_fixture) {
	var buffer bytes.Buffer
	bytes.Buffer_Init(&buffer, fixture.Alternate[:], nil)
	bytes.Buffer_Read_Until(&buffer, 0)
	bytes.Buffer_Init(&buffer, fixture.Alternate[:], fixture.Maximum[:1])
	bytes.Buffer_Read_Until(&buffer, 0)
	bytes.Buffer_Init(&buffer, fixture.Alternate[:], fixture.Maximum[:2])
	bytes.Buffer_Read_Until(&buffer, 1)
	bytes.Buffer_Init(&buffer, fixture.Alternate[:], fixture.Maximum[:])
	bytes.Buffer_Read_Until(&buffer, 255)
}

func cover_reader_domains(fixture *domain_fixture) {
	cover_reader_state_inputs(fixture)
	cover_reader_boundary_outputs(fixture)
	cover_reader_scalar_outputs(fixture)
}

func cover_reader_state_inputs(fixture *domain_fixture) {
	states := [...]bytes.Reader{
		{Source: nil, Position: 0, Previous: -1},
		{Source: fixture.Maximum[:1], Position: 1, Previous: 0},
		{Source: fixture.Maximum[:2], Position: 2, Previous: 1},
		{Source: fixture.Maximum[:3], Position: 3, Previous: 2},
		{
			Source:   fixture.Maximum[:],
			Position: bytes.Reader_Position(bytes.READER_POSITION_MAXIMUM),
			Previous: bytes.INDEX_MAXIMUM,
		},
	}
	for _, state := range states {
		cover_reader_state_queries(fixture, state)
		cover_reader_state_cursor(fixture, state)
	}
}

func cover_reader_state_queries(fixture *domain_fixture, state bytes.Reader) {
	candidate := state
	bytes.Reader_Unread_Size(&candidate)
	candidate = state
	bytes.Reader_Size(&candidate)
	candidate = state
	bytes.Reader_Read_Into(&candidate, fixture.Alternate[:0])
	candidate = state
	bytes.Reader_Read_At_Into(&candidate, fixture.Alternate[:0], 0)
	candidate = state
	bytes.Reader_Read_Byte(&candidate)
	candidate = state
	bytes.Reader_Read_Character(&candidate)
}

func cover_reader_state_cursor(fixture *domain_fixture, state bytes.Reader) {
	candidate := state
	panic_text(func() { bytes.Reader_Unread_Byte(&candidate) })
	candidate = state
	panic_text(func() { bytes.Reader_Unread_Character(&candidate) })
	candidate = state
	bytes.Reader_Seek(&candidate, 0, bytes.SEEK_FROM_START)
	candidate = state
	bytes.Reader_Reset(&candidate, fixture.Maximum[:0])
}

func cover_reader_boundary_outputs(fixture *domain_fixture) {
	sources := [...]bytes.Slice{
		nil, fixture.Maximum[:1], fixture.Maximum[:2], fixture.Maximum[:],
	}
	for _, source := range sources {
		var reader bytes.Reader
		bytes.Reader_Reset(&reader, source)
		bytes.Reader_Unread_Size(&reader)
		bytes.Reader_Size(&reader)
		bytes.Reader_Read_Into(&reader, fixture.Alternate[:len(source)])
		bytes.Reader_Reset(&reader, source)
		bytes.Reader_Read_At_Into(&reader, fixture.Alternate[:len(source)], 0)
	}
	cover_reader_seek_outputs(fixture)
	cover_reader_origins(fixture)
}

func cover_reader_seek_outputs(fixture *domain_fixture) {
	positions := [...]bytes.Reader_Offset{
		0, 1, 2, bytes.Reader_Offset(bytes.READER_POSITION_MAXIMUM),
	}
	for _, position := range positions {
		reader := bytes.Reader{Source: fixture.Maximum[:]}
		bytes.Reader_Seek(&reader, position, bytes.SEEK_FROM_START)
	}
}

func cover_reader_origins(fixture *domain_fixture) {
	origins := [...]bytes.Seek_From{
		bytes.SEEK_FROM_START, bytes.SEEK_FROM_CURRENT, bytes.SEEK_FROM_END,
	}
	for _, origin := range origins {
		reader := bytes.Reader{Source: fixture.Maximum[:]}
		bytes.Reader_Seek(&reader, 0, origin)
	}
}

func cover_reader_scalar_outputs(fixture *domain_fixture) {
	cover_reader_byte_outputs(fixture)
	cover_reader_character_outputs(fixture)
}

func cover_reader_byte_outputs(fixture *domain_fixture) {
	positions := [...]int{0, 1, 2, bytes.INDEX_MAXIMUM}
	for _, position := range positions {
		var reader bytes.Reader
		bytes.Reader_Reset(&reader, fixture.Maximum[position:position+1])
		bytes.Reader_Read_Byte(&reader)
	}
}

func cover_reader_character_outputs(fixture *domain_fixture) {
	encoded_zero := [TEST_SINGLE_COUNT]byte{0}
	encoded_one := [TEST_SINGLE_COUNT]byte{1}
	encoded_two := [TEST_SINGLE_COUNT]byte{2}
	encoded_size_two := [TEST_READ_COUNT]byte{0xc2, 0x80}
	encoded_size_three := [TEST_TRIPLE_COUNT]byte{0xe0, 0xa0, 0x80}
	encoded_maximum := [TEST_CHARACTER_COUNT]byte{0xf4, 0x8f, 0xbf, 0xbf}
	values := [...]bytes.Slice{
		encoded_zero[:], encoded_one[:], encoded_two[:], encoded_size_two[:],
		encoded_size_three[:], encoded_maximum[:],
	}
	for _, value := range values {
		var reader bytes.Reader
		bytes.Reader_Reset(&reader, value)
		bytes.Reader_Read_Character(&reader)
	}
}

func cover_cut_output_domains(fixture *domain_fixture) {
	sources := [...]bytes.Slice{
		nil, fixture.Lines[:1], fixture.Lines[:2], fixture.Lines[:],
	}
	absent := []byte("x")
	for _, source := range sources {
		bytes.Cut(source, nil)
		bytes.Cut(source, absent)
		bytes.Cut_Prefix(source, absent)
		bytes.Cut_Suffix(source, absent)
	}
}

func cover_transform_maximum_outputs(fixture *domain_fixture) {
	source := fixture.Lines[:]
	destination := fixture.Alternate[:]
	bytes.Map_Into(destination, identity_character, source)
	bytes.To_Upper_Into(destination, source)
	bytes.To_Lower_Into(destination, source)
	bytes.To_Title_Into(destination, source)
	bytes.To_Upper_Special_Into(destination, nil, source)
	bytes.To_Lower_Special_Into(destination, nil, source)
	bytes.To_Title_Special_Into(destination, nil, source)
	bytes.To_Valid_UTF8_Into(destination, source, nil)
	bytes.Title_Into(destination, source)
	maximum_character := [TEST_CHARACTER_COUNT]byte{0xf4, 0x8f, 0xbf, 0xbf}
	bytes.Contains_Any(
		maximum_character[:], bytes.Text(string(maximum_character[:])),
	)
	var title_source [TEST_FIVE_COUNT]byte
	copy(title_source[:], maximum_character[:])
	title_source[4] = 'a'
	var title_destination [TEST_FIVE_COUNT]byte
	bytes.Title_Into(title_destination[:], title_source[:])
}

func cover_argument_output_domains(fixture *domain_fixture) {
	values := [...]bytes.Slice{
		nil, fixture.Maximum[:1], fixture.Maximum[:2], fixture.Maximum[:],
	}
	for _, value := range values {
		bytes.Join_Into(nil, nil, value)
		panic_text(func() { bytes.Replace_Into(nil, nil, value, value, -1) })
		panic_text(func() { bytes.Replace_All_Into(nil, nil, value, value) })
		panic_text(func() { bytes.To_Valid_UTF8_Into(nil, nil, value) })
	}
}

func cover_trim_output_domains(fixture *domain_fixture) {
	sources := [...]bytes.Slice{
		nil, fixture.Lines[:1], fixture.Lines[:2], fixture.Lines[:],
	}
	absent := []byte("x")
	for _, source := range sources {
		bytes.Trim(source, "")
		bytes.Trim_Left(source, "")
		bytes.Trim_Right(source, "")
		bytes.Trim_Prefix(source, absent)
		bytes.Trim_Suffix(source, absent)
		bytes.Trim_Function(source, never_match)
		bytes.Trim_Left_Function(source, never_match)
		bytes.Trim_Right_Function(source, never_match)
	}
}

type zero_allocation_check struct {
	Name string
	Call func()
}

type zero_allocation_fixture struct {
	Storage     [TEST_STORAGE_COUNT]byte
	Alternate   [TEST_STORAGE_COUNT]byte
	Source      [TEST_READ_COUNT]byte
	Parts       [TEST_READ_COUNT][]byte
	Slots       [TEST_SLOT_COUNT][]byte
	Characters  [TEST_CHARACTER_COUNT]rune
	Buffer      bytes.Buffer
	Reader      bytes.Reader
	Special     ucd.Special_Case
	Case_Ranges [ucd.SPECIAL_CASE_COUNT_MAXIMUM]ucd.Case_Range
	Observable  int
}

func verify_api_is_zero_allocation(t *testing.T) {
	fixture := zero_allocation_fixture{
		Source: [TEST_READ_COUNT]byte{'a', 'b'},
	}
	fixture.Special = ucd.Special_Case(ucd.Turkish_Case(fixture.Case_Ranges[:]))
	fixture.Parts[0] = fixture.Source[:1]
	fixture.Parts[1] = fixture.Source[1:]
	groups := [][]zero_allocation_check{
		buffer_allocation_checks(&fixture),
		buffer_io_allocation_checks(&fixture),
		reader_allocation_checks(&fixture),
		reader_cursor_allocation_checks(&fixture),
		query_allocation_checks(&fixture),
		query_suffix_allocation_checks(&fixture),
		output_allocation_checks(&fixture),
		transform_allocation_checks(&fixture),
		copy_allocation_checks(&fixture),
		view_iteration_allocation_checks(&fixture),
		cut_iteration_allocation_checks(&fixture),
	}
	for _, group := range groups {
		for _, check := range group {
			t.Run(check.Name, func(t *testing.T) {
				testify.Zero_Allocation(t, check.Call)
			})
		}
	}
	if fixture.Observable == -1 {
		t.Fatal("operations produced impossible observation")
	}
}

func buffer_allocation_checks(
	fixture *zero_allocation_fixture,
) (checks []zero_allocation_check) {
	return []zero_allocation_check{
		{Name: "Buffer_Init", Call: func() {
			fixture.Observable = int(bytes.Buffer_Init(
				&fixture.Buffer, fixture.Storage[:], fixture.Source[:],
			))
		}},
		{Name: "Buffer_Init_Text", Call: func() {
			fixture.Observable = int(bytes.Buffer_Init_Text(
				&fixture.Buffer, fixture.Storage[:], "ab",
			))
		}},
		{Name: "Buffer_Bytes", Call: func() {
			bytes.Buffer_Init(&fixture.Buffer, fixture.Storage[:], fixture.Source[:])
			fixture.Observable = len(bytes.Buffer_Bytes(&fixture.Buffer))
		}},
		{Name: "Buffer_Available_Slice", Call: func() {
			bytes.Buffer_Init(&fixture.Buffer, fixture.Storage[:], fixture.Source[:])
			fixture.Observable = cap(bytes.Buffer_Available_Slice(&fixture.Buffer))
		}},
		{Name: "Buffer_Peek", Call: func() {
			bytes.Buffer_Init(&fixture.Buffer, fixture.Storage[:], fixture.Source[:])
			fixture.Observable = len(bytes.Buffer_Peek(&fixture.Buffer, 1))
		}},
		{Name: "Buffer_Size", Call: func() {
			bytes.Buffer_Init(&fixture.Buffer, fixture.Storage[:], fixture.Source[:])
			fixture.Observable = int(bytes.Buffer_Size(&fixture.Buffer))
		}},
		{Name: "Buffer_Capacity", Call: func() {
			bytes.Buffer_Init(&fixture.Buffer, fixture.Storage[:], fixture.Source[:])
			fixture.Observable = int(bytes.Buffer_Capacity(&fixture.Buffer))
		}},
		{Name: "Buffer_Available", Call: func() {
			bytes.Buffer_Init(&fixture.Buffer, fixture.Storage[:], fixture.Source[:])
			fixture.Observable = int(bytes.Buffer_Available(&fixture.Buffer))
		}},
		{Name: "Buffer_Truncate", Call: func() {
			bytes.Buffer_Init(&fixture.Buffer, fixture.Storage[:], fixture.Source[:])
			bytes.Buffer_Truncate(&fixture.Buffer, 1)
			fixture.Observable = int(bytes.Buffer_Size(&fixture.Buffer))
		}},
		{Name: "Buffer_Reset", Call: func() {
			bytes.Buffer_Init(&fixture.Buffer, fixture.Storage[:], fixture.Source[:])
			bytes.Buffer_Reset(&fixture.Buffer)
			fixture.Observable = len(fixture.Buffer.Content)
		}},
		{Name: "Buffer_Grow", Call: func() {
			bytes.Buffer_Init(&fixture.Buffer, fixture.Storage[:], fixture.Source[:])
			bytes.Buffer_Grow(&fixture.Buffer, 1)
			fixture.Observable = cap(fixture.Buffer.Content)
		}},
	}
}

func buffer_io_allocation_checks(
	fixture *zero_allocation_fixture,
) (checks []zero_allocation_check) {
	return []zero_allocation_check{
		{Name: "Buffer_Write", Call: func() {
			bytes.Buffer_Init(&fixture.Buffer, fixture.Storage[:], nil)
			fixture.Observable = int(bytes.Buffer_Write(
				&fixture.Buffer, fixture.Source[:],
			))
		}},
		{Name: "Buffer_Write_Text", Call: func() {
			bytes.Buffer_Init(&fixture.Buffer, fixture.Storage[:], nil)
			fixture.Observable = int(bytes.Buffer_Write_Text(&fixture.Buffer, "ab"))
		}},
		{Name: "Buffer_Write_Byte", Call: func() {
			bytes.Buffer_Init(&fixture.Buffer, fixture.Storage[:], nil)
			bytes.Buffer_Write_Byte(&fixture.Buffer, 'a')
			fixture.Observable = len(fixture.Buffer.Content)
		}},
		{Name: "Buffer_Write_Character", Call: func() {
			bytes.Buffer_Init(&fixture.Buffer, fixture.Storage[:], nil)
			fixture.Observable = int(bytes.Buffer_Write_Character(
				&fixture.Buffer, '☺',
			))
		}},
		{Name: "Buffer_Read_Into", Call: func() {
			bytes.Buffer_Init(&fixture.Buffer, fixture.Storage[:], fixture.Source[:])
			fixture.Observable = int(bytes.Buffer_Read_Into(
				&fixture.Buffer, fixture.Alternate[:TEST_READ_COUNT],
			))
		}},
		{Name: "Buffer_Next", Call: func() {
			bytes.Buffer_Init(&fixture.Buffer, fixture.Storage[:], fixture.Source[:])
			fixture.Observable = len(bytes.Buffer_Next(&fixture.Buffer, 1))
		}},
		{Name: "Buffer_Read_Byte", Call: func() {
			bytes.Buffer_Init(&fixture.Buffer, fixture.Storage[:], fixture.Source[:])
			value, found := bytes.Buffer_Read_Byte(&fixture.Buffer)
			fixture.Observable = int(value) + zero_allocation_boolean(found)
		}},
		{Name: "Buffer_Read_Character", Call: func() {
			bytes.Buffer_Init(&fixture.Buffer, fixture.Storage[:], fixture.Source[:])
			character, size, found := bytes.Buffer_Read_Character(&fixture.Buffer)
			fixture.Observable = int(character) + int(size)
			fixture.Observable += zero_allocation_boolean(found)
		}},
		{Name: "Buffer_Unread_Character", Call: func() {
			bytes.Buffer_Init(&fixture.Buffer, fixture.Storage[:], fixture.Source[:])
			bytes.Buffer_Read_Character(&fixture.Buffer)
			bytes.Buffer_Unread_Character(&fixture.Buffer)
			fixture.Observable = int(fixture.Buffer.Position)
		}},
		{Name: "Buffer_Unread_Byte", Call: func() {
			bytes.Buffer_Init(&fixture.Buffer, fixture.Storage[:], fixture.Source[:])
			bytes.Buffer_Read_Byte(&fixture.Buffer)
			bytes.Buffer_Unread_Byte(&fixture.Buffer)
			fixture.Observable = int(fixture.Buffer.Position)
		}},
		{Name: "Buffer_Read_Until", Call: func() {
			bytes.Buffer_Init(&fixture.Buffer, fixture.Storage[:], fixture.Source[:])
			content, found := bytes.Buffer_Read_Until(&fixture.Buffer, 'b')
			fixture.Observable = len(content) + zero_allocation_boolean(found)
		}},
	}
}

func reader_allocation_checks(
	fixture *zero_allocation_fixture,
) (checks []zero_allocation_check) {
	return []zero_allocation_check{
		{Name: "Reader_Unread_Size", Call: func() {
			bytes.Reader_Reset(&fixture.Reader, fixture.Source[:])
			fixture.Observable = int(bytes.Reader_Unread_Size(&fixture.Reader))
		}},
		{Name: "Reader_Size", Call: func() {
			bytes.Reader_Reset(&fixture.Reader, fixture.Source[:])
			fixture.Observable = int(bytes.Reader_Size(&fixture.Reader))
		}},
		{Name: "Reader_Read_Into", Call: func() {
			bytes.Reader_Reset(&fixture.Reader, fixture.Source[:])
			fixture.Observable = int(bytes.Reader_Read_Into(
				&fixture.Reader, fixture.Alternate[:TEST_READ_COUNT],
			))
		}},
		{Name: "Reader_Read_At_Into", Call: func() {
			bytes.Reader_Reset(&fixture.Reader, fixture.Source[:])
			fixture.Observable = int(bytes.Reader_Read_At_Into(
				&fixture.Reader, fixture.Alternate[:TEST_SINGLE_COUNT], 1,
			))
		}},
		{Name: "Reader_Read_Byte", Call: func() {
			bytes.Reader_Reset(&fixture.Reader, fixture.Source[:])
			value, found := bytes.Reader_Read_Byte(&fixture.Reader)
			fixture.Observable = int(value) + zero_allocation_boolean(found)
		}},
	}
}

func reader_cursor_allocation_checks(
	fixture *zero_allocation_fixture,
) (checks []zero_allocation_check) {
	return []zero_allocation_check{
		{Name: "Reader_Unread_Byte", Call: func() {
			bytes.Reader_Reset(&fixture.Reader, fixture.Source[:])
			bytes.Reader_Read_Byte(&fixture.Reader)
			bytes.Reader_Unread_Byte(&fixture.Reader)
			fixture.Observable = int(fixture.Reader.Position)
		}},
		{Name: "Reader_Read_Character", Call: func() {
			bytes.Reader_Reset(&fixture.Reader, fixture.Source[:])
			character, size, found := bytes.Reader_Read_Character(&fixture.Reader)
			fixture.Observable = int(character) + int(size)
			fixture.Observable += zero_allocation_boolean(found)
		}},
		{Name: "Reader_Unread_Character", Call: func() {
			bytes.Reader_Reset(&fixture.Reader, fixture.Source[:])
			bytes.Reader_Read_Character(&fixture.Reader)
			bytes.Reader_Unread_Character(&fixture.Reader)
			fixture.Observable = int(fixture.Reader.Position)
		}},
		{Name: "Reader_Seek", Call: func() {
			bytes.Reader_Reset(&fixture.Reader, fixture.Source[:])
			fixture.Observable = int(bytes.Reader_Seek(
				&fixture.Reader, 1, bytes.SEEK_FROM_START,
			))
		}},
		{Name: "Reader_Reset", Call: func() {
			bytes.Reader_Reset(&fixture.Reader, fixture.Source[:])
			fixture.Observable = int(fixture.Reader.Position)
		}},
	}
}

func query_allocation_checks(
	fixture *zero_allocation_fixture,
) (checks []zero_allocation_check) {
	return []zero_allocation_check{
		{Name: "Equal", Call: func() {
			fixture.Observable = zero_allocation_boolean(bytes.Equal(
				fixture.Source[:], fixture.Source[:],
			))
		}},
		{Name: "Compare", Call: func() {
			fixture.Observable = int(bytes.Compare(
				fixture.Source[:], fixture.Source[:],
			))
		}},
		{Name: "Count", Call: func() {
			fixture.Observable = int(bytes.Count(fixture.Source[:], fixture.Source[:1]))
		}},
		{Name: "Contains", Call: func() {
			fixture.Observable = zero_allocation_boolean(bytes.Contains(
				fixture.Source[:], fixture.Source[:1],
			))
		}},
		{Name: "Contains_Any", Call: func() {
			fixture.Observable = zero_allocation_boolean(bytes.Contains_Any(
				fixture.Source[:], "a",
			))
		}},
		{Name: "Contains_Rune", Call: func() {
			fixture.Observable = zero_allocation_boolean(bytes.Contains_Rune(
				fixture.Source[:], 'a',
			))
		}},
		{Name: "Contains_Function", Call: func() {
			fixture.Observable = zero_allocation_boolean(bytes.Contains_Function(
				fixture.Source[:], is_letter_a,
			))
		}},
		{Name: "Index_Byte", Call: func() {
			fixture.Observable = int(bytes.Index_Byte(fixture.Source[:], 'a'))
		}},
		{Name: "Last_Index", Call: func() {
			fixture.Observable = int(bytes.Last_Index(
				fixture.Source[:], fixture.Source[:1],
			))
		}},
		{Name: "Last_Index_Byte", Call: func() {
			fixture.Observable = int(bytes.Last_Index_Byte(fixture.Source[:], 'a'))
		}},
		{Name: "Index_Rune", Call: func() {
			fixture.Observable = int(bytes.Index_Rune(fixture.Source[:], 'a'))
		}},
		{Name: "Index_Any", Call: func() {
			fixture.Observable = int(bytes.Index_Any(fixture.Source[:], "a"))
		}},
		{Name: "Last_Index_Any", Call: func() {
			fixture.Observable = int(bytes.Last_Index_Any(fixture.Source[:], "a"))
		}},
	}
}

func query_suffix_allocation_checks(
	fixture *zero_allocation_fixture,
) (checks []zero_allocation_check) {
	return []zero_allocation_check{
		{Name: "Has_Prefix", Call: func() {
			fixture.Observable = zero_allocation_boolean(bytes.Has_Prefix(
				fixture.Source[:], fixture.Source[:1],
			))
		}},
		{Name: "Has_Suffix", Call: func() {
			fixture.Observable = zero_allocation_boolean(bytes.Has_Suffix(
				fixture.Source[:], fixture.Source[1:],
			))
		}},
		{Name: "Has_Text_Suffix", Call: func() {
			fixture.Observable = zero_allocation_boolean(bytes.Has_Text_Suffix(
				fixture.Source[:], "b",
			))
		}},
		{Name: "Overlap", Call: func() {
			fixture.Observable = zero_allocation_boolean(bytes.Overlap(
				fixture.Source[:], fixture.Source[1:],
			))
		}},
		{Name: "Index_Function", Call: func() {
			fixture.Observable = int(bytes.Index_Function(
				fixture.Source[:], is_letter_a,
			))
		}},
		{Name: "Last_Index_Function", Call: func() {
			fixture.Observable = int(bytes.Last_Index_Function(
				fixture.Source[:], is_letter_a,
			))
		}},
		{Name: "Equal_Fold", Call: func() {
			fixture.Observable = zero_allocation_boolean(bytes.Equal_Fold(
				fixture.Source[:], fixture.Source[:],
			))
		}},
		{Name: "Index", Call: func() {
			fixture.Observable = int(bytes.Index(fixture.Source[:], fixture.Source[:1]))
		}},
	}
}

func output_allocation_checks(
	fixture *zero_allocation_fixture,
) (checks []zero_allocation_check) {
	return []zero_allocation_check{
		{Name: "Split_Into", Call: func() {
			fixture.Observable = int(bytes.Split_Into(
				fixture.Slots[:], fixture.Source[:], fixture.Source[:1],
			))
		}},
		{Name: "Split_N_Into", Call: func() {
			fixture.Observable = int(bytes.Split_N_Into(
				fixture.Slots[:], fixture.Source[:], fixture.Source[:1], 1,
			))
		}},
		{Name: "Split_After_Into", Call: func() {
			fixture.Observable = int(bytes.Split_After_Into(
				fixture.Slots[:], fixture.Source[:], fixture.Source[:1],
			))
		}},
		{Name: "Split_After_N_Into", Call: func() {
			fixture.Observable = int(bytes.Split_After_N_Into(
				fixture.Slots[:], fixture.Source[:], fixture.Source[:1], 1,
			))
		}},
		{Name: "Fields_Into", Call: func() {
			fixture.Observable = int(bytes.Fields_Into(
				fixture.Slots[:], fixture.Source[:],
			))
		}},
		{Name: "Fields_Function_Into", Call: func() {
			fixture.Observable = int(bytes.Fields_Function_Into(
				fixture.Slots[:], fixture.Source[:], is_dash,
			))
		}},
		{Name: "Join_Into", Call: func() {
			fixture.Observable = int(bytes.Join_Into(
				fixture.Storage[:], fixture.Parts[:], fixture.Alternate[:1],
			))
		}},
	}
}

func transform_allocation_checks(
	fixture *zero_allocation_fixture,
) (checks []zero_allocation_check) {
	return []zero_allocation_check{
		{Name: "Map_Into", Call: func() {
			fixture.Observable = int(bytes.Map_Into(
				fixture.Storage[:], map_character, fixture.Source[:],
			))
		}},
		{Name: "Repeat_Into", Call: func() {
			fixture.Observable = int(bytes.Repeat_Into(
				fixture.Storage[:], fixture.Source[:], 2,
			))
		}},
		{Name: "To_Upper_Into", Call: func() {
			fixture.Observable = int(bytes.To_Upper_Into(
				fixture.Storage[:], fixture.Source[:],
			))
		}},
		{Name: "To_Lower_Into", Call: func() {
			fixture.Observable = int(bytes.To_Lower_Into(
				fixture.Storage[:], fixture.Source[:],
			))
		}},
		{Name: "To_Title_Into", Call: func() {
			fixture.Observable = int(bytes.To_Title_Into(
				fixture.Storage[:], fixture.Source[:],
			))
		}},
		{Name: "To_Upper_Special_Into", Call: func() {
			fixture.Observable = int(bytes.To_Upper_Special_Into(
				fixture.Storage[:], fixture.Special, fixture.Source[:],
			))
		}},
		{Name: "To_Lower_Special_Into", Call: func() {
			fixture.Observable = int(bytes.To_Lower_Special_Into(
				fixture.Storage[:], fixture.Special, fixture.Source[:],
			))
		}},
		{Name: "To_Title_Special_Into", Call: func() {
			fixture.Observable = int(bytes.To_Title_Special_Into(
				fixture.Storage[:], fixture.Special, fixture.Source[:],
			))
		}},
		{Name: "To_Valid_UTF8_Into", Call: func() {
			fixture.Observable = int(bytes.To_Valid_UTF8_Into(
				fixture.Storage[:], fixture.Source[:], fixture.Alternate[:1],
			))
		}},
		{Name: "Title_Into", Call: func() {
			fixture.Observable = int(bytes.Title_Into(
				fixture.Storage[:], fixture.Source[:],
			))
		}},
	}
}

func copy_allocation_checks(
	fixture *zero_allocation_fixture,
) (checks []zero_allocation_check) {
	return []zero_allocation_check{
		{Name: "Runes_Into", Call: func() {
			fixture.Observable = int(bytes.Runes_Into(
				fixture.Characters[:], fixture.Source[:],
			))
		}},
		{Name: "Replace_Into", Call: func() {
			fixture.Observable = int(bytes.Replace_Into(
				fixture.Storage[:], fixture.Source[:], fixture.Source[:1],
				fixture.Alternate[:1], 1,
			))
		}},
		{Name: "Replace_All_Into", Call: func() {
			fixture.Observable = int(bytes.Replace_All_Into(
				fixture.Storage[:], fixture.Source[:], fixture.Source[:1],
				fixture.Alternate[:1],
			))
		}},
		{Name: "Clone_Into", Call: func() {
			fixture.Observable = int(bytes.Clone_Into(
				fixture.Storage[:], fixture.Source[:],
			))
		}},
	}
}

func view_iteration_allocation_checks(
	fixture *zero_allocation_fixture,
) (checks []zero_allocation_check) {
	return []zero_allocation_check{
		{Name: "Trim_Left_Function", Call: func() {
			fixture.Observable = len(bytes.Trim_Left_Function(
				fixture.Source[:], is_dash,
			))
		}},
		{Name: "Trim_Right_Function", Call: func() {
			fixture.Observable = len(bytes.Trim_Right_Function(
				fixture.Source[:], is_dash,
			))
		}},
		{Name: "Trim_Function", Call: func() {
			fixture.Observable = len(bytes.Trim_Function(
				fixture.Source[:], is_dash,
			))
		}},
		{Name: "Trim_Prefix", Call: func() {
			fixture.Observable = len(bytes.Trim_Prefix(
				fixture.Source[:], fixture.Source[:1],
			))
		}},
		{Name: "Trim_Suffix", Call: func() {
			fixture.Observable = len(bytes.Trim_Suffix(
				fixture.Source[:], fixture.Source[1:],
			))
		}},
		{Name: "Trim", Call: func() {
			fixture.Observable = len(bytes.Trim(fixture.Source[:], "a"))
		}},
		{Name: "Trim_Left", Call: func() {
			fixture.Observable = len(bytes.Trim_Left(fixture.Source[:], "a"))
		}},
		{Name: "Trim_Right", Call: func() {
			fixture.Observable = len(bytes.Trim_Right(fixture.Source[:], "b"))
		}},
		{Name: "Trim_Space", Call: func() {
			fixture.Observable = len(bytes.Trim_Space(fixture.Source[:]))
		}},
	}
}

func cut_iteration_allocation_checks(
	fixture *zero_allocation_fixture,
) (checks []zero_allocation_check) {
	return []zero_allocation_check{
		{Name: "Cut", Call: func() {
			before, after, found := bytes.Cut(
				fixture.Source[:], fixture.Source[:1],
			)
			fixture.Observable = len(before) + len(after)
			fixture.Observable += zero_allocation_boolean(found)
		}},
		{Name: "Cut_Prefix", Call: func() {
			after, found := bytes.Cut_Prefix(
				fixture.Source[:], fixture.Source[:1],
			)
			fixture.Observable = len(after) + zero_allocation_boolean(found)
		}},
		{Name: "Cut_Suffix", Call: func() {
			before, found := bytes.Cut_Suffix(
				fixture.Source[:], fixture.Source[1:],
			)
			fixture.Observable = len(before) + zero_allocation_boolean(found)
		}},
		{Name: "Lines", Call: func() {
			fixture.Observable = int(bytes.Lines(
				fixture.Source[:], zero_allocation_yield,
			))
		}},
		{Name: "Split_Sequence", Call: func() {
			fixture.Observable = int(bytes.Split_Sequence(
				fixture.Source[:], fixture.Source[:1], zero_allocation_yield,
			))
		}},
		{Name: "Split_After_Sequence", Call: func() {
			fixture.Observable = int(bytes.Split_After_Sequence(
				fixture.Source[:], fixture.Source[:1], zero_allocation_yield,
			))
		}},
		{Name: "Fields_Sequence", Call: func() {
			fixture.Observable = int(bytes.Fields_Sequence(
				fixture.Source[:], zero_allocation_yield,
			))
		}},
		{Name: "Fields_Function_Sequence", Call: func() {
			fixture.Observable = int(bytes.Fields_Function_Sequence(
				fixture.Source[:], is_dash, zero_allocation_yield,
			))
		}},
	}
}

func zero_allocation_boolean(value bytes.Boolean) (number int) {
	if value {
		return 1
	}
	return 0
}

func zero_allocation_yield(_ bytes.Slice) (continued bytes.Boolean) {
	return true
}
