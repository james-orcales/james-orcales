package strings_test

import (
	"testing"
	"unsafe"

	"local/james-orcales/shared/strings"
	"local/james-orcales/shared/testify"
	"local/james-orcales/shared/unicode/ucd"
	"local/james-orcales/shared/unicode/utf8"
)

// Test_Allocation proves each retained operation owns no heap storage.
func Test_Allocation(t *testing.T) {
	verify_api_is_zero_allocation(t)
}

// Test_Comparison protects lexical order and Unicode simple folding.
func Test_Comparison(t *testing.T) {
	for _, one := range []struct {
		Left  strings.Text
		Right strings.Text
		Want  strings.Order
	}{
		{Left: "", Right: "", Want: strings.ORDER_EQUAL},
		{Left: "a", Right: "b", Want: strings.ORDER_BEFORE},
		{Left: "b", Right: "a", Want: strings.ORDER_AFTER},
		{Left: "abc", Right: "ab", Want: strings.ORDER_AFTER},
	} {
		if got := strings.Compare(one.Left, one.Right); got != one.Want {
			t.Fatalf(
				"Compare(%q, %q) = %d, want %d", one.Left, one.Right, got, one.Want,
			)
		}
	}
	for _, one := range []struct {
		Left  strings.Text
		Right strings.Text
		Want  bool
	}{
		{Left: "Go", Right: "gO", Want: true},
		{Left: "Σ", Right: "ς", Want: true},
		{Left: "\xff", Right: "�", Want: false},
	} {
		got := bool(strings.Equal_Fold(one.Left, one.Right))
		if got != one.Want {
			t.Fatalf(
				"Equal_Fold(%q, %q) = %t, want %t",
				one.Left, one.Right, got, one.Want,
			)
		}
	}
}

// Test_Search protects byte indices, Unicode decoding, and empty separators.
func Test_Search(t *testing.T) {
	if strings.Count("a☺", "") != 3 {
		t.Fatal("empty separator must occur at each UTF-8 boundary")
	}
	if !strings.Contains("abc", "b") {
		t.Fatal("Contains must report presence")
	}
	if strings.Contains("abc", "x") {
		t.Fatal("Contains must report absence")
	}
	if !strings.Contains_Any("abc", "xb") {
		t.Fatal("Contains_Any must search character set")
	}
	if !strings.Contains_Rune("a☺", '☺') {
		t.Fatal("Contains_Rune must search Unicode character")
	}
	if !strings.Contains_Function("a b", allocation_space) {
		t.Fatal("Contains_Function must search decoded characters")
	}
	verify_search_indices(t)
}

// Test_Split_And_Join proves caller storage owns collections and joined bytes.
func Test_Split_And_Join(t *testing.T) {
	var part_storage [TEST_TEXT_SLOT_COUNT]strings.Text
	part_count := strings.Split_Into(part_storage[:], "a,b,", ",")
	parts := part_storage[:int(part_count)]
	if len(parts) != 3 {
		t.Fatal("Split_Into must retain final empty part")
	}
	if parts[0] != "a" {
		t.Fatal("Split_Into must return source views")
	}
	if parts[1] != "b" {
		t.Fatal("Split_Into must return source views")
	}
	if parts[2] != "" {
		t.Fatal("Split_Into must return source views")
	}
	part_count = strings.Split_After_Into(part_storage[:], "a,b,", ",")
	parts = part_storage[:int(part_count)]
	if parts[0] != "a," {
		t.Fatal("Split_After_Into must retain separators")
	}
	if parts[1] != "b," {
		t.Fatal("Split_After_Into must retain separators")
	}
	if parts[2] != "" {
		t.Fatal("Split_After_Into must retain separators")
	}
	fields := strings.Fields_Into(part_storage[:], "  a\t☺  ")
	if len(fields) != 2 {
		t.Fatal("Fields_Into must return non-space source views")
	}
	if fields[0] != "a" {
		t.Fatal("Fields_Into must return non-space source views")
	}
	if fields[1] != "☺" {
		t.Fatal("Fields_Into must return non-space source views")
	}
	verify_join_and_lines(t, strings.Texts(fields))
	verify_split_edges(t, part_storage[:])
}

// Test_Transform proves transformed bytes live only in caller storage.
func Test_Transform(t *testing.T) {
	var storage [strings.TEXT_SIZE_MAXIMUM]byte
	if got := strings.Clone_Into(storage[:], "abc"); string(got) != "abc" {
		t.Fatal("Clone_Into must copy into caller storage")
	}
	if got := strings.Map_Into(storage[:], "a☺", map_upper_a); string(got) != "A☺" {
		t.Fatal("Map_Into must encode mapped characters")
	}
	if got := strings.Repeat_Into(storage[:], "ab", 3); string(got) != "ababab" {
		t.Fatal("Repeat_Into must copy requested repetitions")
	}
	if got := strings.Replace_Into(
		storage[:], "a-b-a", "a", "xy", -1,
	); string(got) != "xy-b-xy" {
		t.Fatal("Replace_Into must replace non-overlapping matches")
	}
	verify_case_and_valid_transform(t, storage[:])
	verify_transform_edges(t, storage[:])
}

// Test_Trim_And_Cut proves each result remains an input view.
func Test_Trim_And_Cut(t *testing.T) {
	if strings.Trim("  abc  ", " ") != "abc" {
		t.Fatal("Trim must remove both sides")
	}
	if strings.Trim_Left("  abc  ", " ") != "abc  " {
		t.Fatal("Trim_Left must preserve right side")
	}
	if strings.Trim_Right("  abc  ", " ") != "  abc" {
		t.Fatal("Trim_Right must preserve left side")
	}
	if strings.Trim_Function(" abc ", allocation_space) != "abc" {
		t.Fatal("Trim_Function must remove both predicate sides")
	}
	if strings.Trim_Left_Function(" abc ", allocation_space) != "abc " {
		t.Fatal("Trim_Left_Function must preserve right side")
	}
	if strings.Trim_Right_Function(" abc ", allocation_space) != " abc" {
		t.Fatal("Trim_Right_Function must preserve left side")
	}
	if strings.Trim_Space("\u2000abc\u2000") != "abc" {
		t.Fatal("Trim_Space must apply Unicode space table")
	}
	verify_affix_views(t)
}

// Test_Builder proves zero value owns fixed bounded storage.
func Test_Builder(t *testing.T) {
	var builder strings.Builder
	strings.Builder_Write_Text(&builder, "a")
	strings.Builder_Write_Byte(&builder, 'b')
	strings.Builder_Write_Character(&builder, '☺')
	if string(strings.Builder_Bytes(&builder)) != "ab☺" {
		t.Fatal("Builder writes must remain in fixed storage")
	}
	if strings.Builder_Size(&builder) != 5 {
		t.Fatal("Builder_Size must report encoded bytes")
	}
	strings.Builder_Reset(&builder)
	if len(strings.Builder_Bytes(&builder)) != 0 {
		t.Fatal("Builder_Reset must retain storage and remove content")
	}
}

// Test_Reader proves cursor state needs no allocated wrapper.
func Test_Reader(t *testing.T) {
	var reader strings.Reader
	strings.Reader_Reset(&reader, "a☺")
	character, found := strings.Reader_Read_Character(&reader)
	if character != 'a' {
		t.Fatal("Reader_Read_Character must decode first character")
	}
	if !found {
		t.Fatal("Reader_Read_Character must report content")
	}
	strings.Reader_Unread_Character(&reader)
	character, found = strings.Reader_Read_Character(&reader)
	if character != 'a' {
		t.Fatal("Reader_Unread_Character must restore prior boundary")
	}
	if !found {
		t.Fatal("Reader_Unread_Character must preserve readable content")
	}
	var storage [utf8.CHARACTER_SIZE_MAXIMUM]byte
	read := strings.Reader_Read_Into(&reader, storage[:])
	if string(read) != "☺" {
		t.Fatal("Reader_Read_Into must copy unread bytes")
	}
}

// Test_Replacer proves rules and output remain caller-owned.
func Test_Replacer(t *testing.T) {
	rules := [...]strings.Rule{
		{Old: "ab", New: "x"},
		{Old: "a", New: "y"},
	}
	replacer := strings.Replacer_Init(rules[:])
	var storage [strings.TEXT_SIZE_MAXIMUM]byte
	replaced := strings.Replacer_Replace_Into(storage[:], replacer, "ab-a")
	if string(replaced) != "x-y" {
		t.Fatal("Replacer must honor rule order")
	}
}

// Test_Size_Limit rejects oversized malicious input.
func Test_Size_Limit(t *testing.T) {
	var maximum_storage [strings.TEXT_SIZE_MAXIMUM]byte
	maximum := repeated_text(maximum_storage[:], "x")
	var maximum_y_storage [strings.TEXT_SIZE_MAXIMUM]byte
	maximum_y := repeated_text(maximum_y_storage[:], "x")
	maximum_y_storage[len(maximum_y_storage)-1] = 'y'
	var maximum_separator_storage [strings.TEXT_SIZE_MAXIMUM]byte
	maximum_separator := repeated_text(maximum_separator_storage[:], ":")
	var maximum_fields_storage [2 * strings.FIELD_COUNT_MAXIMUM]byte
	maximum_fields := repeated_text(maximum_fields_storage[:], "x ")
	var maximum_lines_storage [strings.LINE_COUNT_MAXIMUM]byte
	maximum_lines := repeated_text(maximum_lines_storage[:], "\n")
	if strings.Index_Byte(maximum, 'x') != 0 {
		t.Fatal("maximum Text must remain valid")
	}
	exercise_invariant_boundaries(maximum, maximum_y)
	var byte_storage [strings.TEXT_SIZE_MAXIMUM]byte
	var text_storage [strings.TEXT_COUNT_MAXIMUM]strings.Text
	exercise_destination_boundaries(
		byte_storage[:], text_storage[:], maximum, maximum_separator,
		maximum_fields, maximum_lines,
	)
	var oversized_storage [strings.TEXT_SIZE_MAXIMUM + 1]byte
	oversized := repeated_text(oversized_storage[:], "x")
	defer func() {
		if recover() == nil {
			t.Fatal("oversized Text must panic")
		}
	}()
	strings.Contains(oversized, "x")
}

const TEST_TEXT_SLOT_COUNT = 8

const TEST_BYTE_STORAGE_SIZE = 16

func verify_join_and_lines(t *testing.T, parts strings.Texts) {
	var byte_storage [TEST_BYTE_STORAGE_SIZE]byte
	joined := strings.Join_Into(byte_storage[:], parts, "|")
	if string(joined) != "a|☺" {
		t.Fatal("Join_Into must write into caller storage")
	}
	var line_storage [utf8.CHARACTER_SIZE_MAXIMUM]strings.Text
	lines := strings.Lines_Into(line_storage[:], "a\nb")
	if len(lines) != 2 {
		t.Fatal("Lines_Into must return source views")
	}
	if lines[0] != "a\n" {
		t.Fatal("Lines_Into must retain newline")
	}
	if lines[1] != "b" {
		t.Fatal("Lines_Into must retain final unterminated line")
	}
}

func verify_split_edges(t *testing.T, storage strings.Texts) {
	part_count := strings.Split_Into(storage, "a☺", "")
	parts := storage[:int(part_count)]
	if len(parts) != 2 {
		t.Fatal("empty separator must split at UTF-8 boundaries")
	}
	if parts[0] != "a" {
		t.Fatal("empty separator must retain first character")
	}
	if parts[1] != "☺" {
		t.Fatal("empty separator must retain multibyte character")
	}
	part_count = strings.Split_N_Into(storage, "a:b:c", ":", 2)
	parts = storage[:int(part_count)]
	if parts[0] != "a" {
		t.Fatal("Split_N_Into must retain first limited part")
	}
	if parts[1] != "b:c" {
		t.Fatal("Split_N_Into must retain unsplit tail")
	}
	fields := strings.Fields_Function_Into(storage, "a,b", allocation_comma)
	if len(fields) != 2 {
		t.Fatal("Fields_Function_Into must use injected separator")
	}
}

func verify_case_and_valid_transform(t *testing.T, storage strings.Bytes) {
	if got := strings.To_Upper_Into(storage, "Go☺"); string(got) != "GO☺" {
		t.Fatal("To_Upper_Into must apply Unicode mapping")
	}
	if got := strings.To_Lower_Into(storage, "Go☺"); string(got) != "go☺" {
		t.Fatal("To_Lower_Into must apply Unicode mapping")
	}
	if got := strings.To_Title_Into(storage, "go☺"); string(got) != "GO☺" {
		t.Fatal("To_Title_Into must map each character to title case")
	}
	if got := strings.Title_Into(storage, "go gopher"); string(got) != "Go Gopher" {
		t.Fatal("Title_Into must map word starts")
	}
	invalid := strings.Text("a\xffb")
	if got := strings.To_Valid_UTF8_Into(storage, invalid, "?"); string(got) != "a?b" {
		t.Fatal("To_Valid_UTF8_Into must replace invalid runs")
	}
}

func verify_transform_edges(t *testing.T, storage strings.Bytes) {
	if got := strings.Map_Into(storage, "axb", allocation_drop_x); string(got) != "ab" {
		t.Fatal("Map_Into must drop negative mappings")
	}
	if got := strings.Repeat_Into(storage, "x", 0); len(got) != 0 {
		t.Fatal("Repeat_Into must permit zero copies")
	}
	if got := strings.Replace_All_Into(storage, "ab", "", "-"); string(got) != "-a-b-" {
		t.Fatal("Replace_All_Into must replace UTF-8 boundaries")
	}
	var turkish_storage [ucd.SPECIAL_CASE_COUNT_MAXIMUM]ucd.Case_Range
	turkish := ucd.Special_Case(ucd.Turkish_Case(turkish_storage[:]))
	if got := strings.To_Upper_Special_Into(storage, turkish, "i"); string(got) != "İ" {
		t.Fatal("special case transform must use supplied language rules")
	}
}

func map_upper_a(character rune) (mapped rune) {
	if character == 'a' {
		return 'A'
	}
	return character
}

func allocation_comma(character rune) (matches bool) {
	return character == ','
}

func allocation_drop_x(character rune) (mapped rune) {
	if character == 'x' {
		return -1
	}
	return character
}

func verify_search_indices(t *testing.T) {
	if strings.Index("abc", "b") != 1 {
		t.Fatal("Index must return byte index")
	}
	if strings.Last_Index("abc", "") != 3 {
		t.Fatal("Last_Index must include final empty boundary")
	}
	if strings.Index_Byte("abc", 'b') != 1 {
		t.Fatal("Index_Byte must return byte index")
	}
	if strings.Last_Index_Byte("abcb", 'b') != 3 {
		t.Fatal("Last_Index_Byte must return final byte index")
	}
	if strings.Index_Rune("a☺", '☺') != 1 {
		t.Fatal("Index_Rune must return byte index")
	}
	if strings.Index_Rune(
		strings.Text("\xff"), strings.Character(utf8.REPLACEMENT_CHARACTER),
	) != 0 {
		t.Fatal("Index_Rune must expose invalid UTF-8 through replacement character")
	}
	if strings.Index_Any("a☺", "☺") != 1 {
		t.Fatal("Index_Any must return first matching byte index")
	}
	if strings.Last_Index_Any("abcb", "xb") != 3 {
		t.Fatal("Last_Index_Any must return final matching byte index")
	}
	if strings.Index_Function("a b", allocation_space) != 1 {
		t.Fatal("Index_Function must return first predicate byte index")
	}
	if strings.Last_Index_Function("a b ", allocation_space) != 3 {
		t.Fatal("Last_Index_Function must return final predicate byte index")
	}
	verify_byte_or_non_ascii(t)
}

func verify_byte_or_non_ascii(t *testing.T) {
	for _, one := range []struct {
		Source strings.Text
		Values strings.Text
		Want   strings.Index_Value
	}{
		{Source: "abc", Values: "x", Want: strings.INDEX_ABSENT},
		{Source: "abc", Values: "b", Want: 1},
		{Source: "a☺", Values: "x", Want: 1},
		{Source: strings.Text("a\xff"), Values: "x", Want: 1},
	} {
		got := strings.Index_Byte_Or_Non_ASCII(one.Source, one.Values)
		if got != one.Want {
			t.Fatalf("Index_Byte_Or_Non_ASCII(%q, %q) = %d, want %d",
				one.Source, one.Values, got, one.Want)
		}
	}
}

func verify_affix_views(t *testing.T) {
	if strings.Trim_Prefix("prefix", "pre") != "fix" {
		t.Fatal("Trim_Prefix must remove present prefix")
	}
	if strings.Trim_Suffix("suffix", "fix") != "suf" {
		t.Fatal("Trim_Suffix must remove present suffix")
	}
	if !strings.Has_Prefix("prefix", "pre") {
		t.Fatal("Has_Prefix must report present prefix")
	}
	if !strings.Has_Suffix("suffix", "fix") {
		t.Fatal("Has_Suffix must report present suffix")
	}
	before, after, found := strings.Cut("a:b", ":")
	if before != "a" {
		t.Fatal("Cut must return both source views")
	}
	if after != "b" {
		t.Fatal("Cut must return both source views")
	}
	if !found {
		t.Fatal("Cut must report present separator")
	}
	after, found = strings.Cut_Prefix("prefix", "pre")
	if after != "fix" {
		t.Fatal("Cut_Prefix must return suffix view")
	}
	if !found {
		t.Fatal("Cut_Prefix must report present prefix")
	}
	before, found = strings.Cut_Suffix("suffix", "fix")
	if before != "suf" {
		t.Fatal("Cut_Suffix must return prefix view")
	}
	if !found {
		t.Fatal("Cut_Suffix must report present suffix")
	}
}

type allocation_check struct {
	Name string
	Call func()
}

func verify_api_is_zero_allocation(t *testing.T) {
	var maximum_storage [strings.TEXT_SIZE_MAXIMUM]byte
	maximum := repeated_text(maximum_storage[:], "x")
	var maximum_y_storage [strings.TEXT_SIZE_MAXIMUM]byte
	maximum_y := repeated_text(maximum_y_storage[:], "x")
	maximum_y_storage[len(maximum_y_storage)-1] = 'y'
	var byte_storage [strings.TEXT_SIZE_MAXIMUM]byte
	var text_storage [strings.TEXT_COUNT_MAXIMUM]strings.Text
	var builder strings.Builder
	var reader strings.Reader
	rules := [...]strings.Rule{{Old: "x", New: "y"}}
	observable := 0
	groups := [][]allocation_check{
		comparison_allocation_checks(&observable, maximum),
		search_allocation_checks(&observable, maximum_y),
		index_allocation_checks(&observable, maximum_y),
		collection_allocation_checks(
			&observable, byte_storage[:], text_storage[:], maximum_y,
		),
		transform_allocation_checks(&observable, byte_storage[:], maximum_y),
		trim_allocation_checks(&observable, maximum),
		cut_allocation_checks(&observable),
		builder_allocation_checks(&observable, &builder, byte_storage[:]),
		reader_allocation_checks(&observable, &reader, byte_storage[:]),
		replacer_allocation_checks(&observable, rules[:], byte_storage[:]),
	}
	for _, group := range groups {
		for _, one := range group {
			t.Run(one.Name, func(t *testing.T) {
				testify.Zero_Allocation(t, one.Call)
			})
		}
	}
	if observable == -1 {
		t.Fatal("allocation operations produced impossible observation")
	}
}

// Keeps boundary fixtures caller-owned so allocation tests never hide setup ownership.
func repeated_text(storage strings.Bytes, fragment strings.Text) (text strings.Text) {
	for position := 0; position < len(storage); position += len(fragment) {
		copy(storage[position:], fragment)
	}
	return strings.Text(unsafe.String(unsafe.SliceData(storage), len(storage)))
}

func comparison_allocation_checks(
	observable *int, maximum strings.Text,
) (checks []allocation_check) {
	return []allocation_check{
		{Name: "Compare", Call: func() {
			*observable = int(strings.Compare(maximum, maximum))
		}},
		{Name: "Equal_Fold", Call: func() {
			*observable = allocation_boolean(strings.Equal_Fold("Go", "gO"))
		}},
	}
}

func search_allocation_checks(
	observable *int, maximum_y strings.Text,
) (checks []allocation_check) {
	return []allocation_check{
		{Name: "Contains", Call: func() {
			*observable = allocation_boolean(strings.Contains(maximum_y, "y"))
		}},
		{Name: "Contains_Any", Call: func() {
			*observable = allocation_boolean(
				strings.Contains_Any(maximum_y, "y"),
			)
		}},
		{Name: "Contains_Rune", Call: func() {
			*observable = allocation_boolean(
				strings.Contains_Rune(maximum_y, 'y'),
			)
		}},
		{Name: "Contains_Function", Call: func() {
			*observable = allocation_boolean(
				strings.Contains_Function(maximum_y, allocation_y),
			)
		}},
		{Name: "Count", Call: func() {
			*observable = int(strings.Count("xyxy", "xy"))
		}},
		{Name: "Has_Prefix", Call: func() {
			*observable = allocation_boolean(strings.Has_Prefix(maximum_y, "x"))
		}},
		{Name: "Has_Suffix", Call: func() {
			*observable = allocation_boolean(strings.Has_Suffix(maximum_y, "y"))
		}},
	}
}

func index_allocation_checks(
	observable *int, maximum_y strings.Text,
) (checks []allocation_check) {
	return []allocation_check{
		{Name: "Index", Call: func() {
			*observable = int(strings.Index(maximum_y, "y"))
		}},
		{Name: "Last_Index", Call: func() {
			*observable = int(strings.Last_Index(maximum_y, "y"))
		}},
		{Name: "Index_Byte", Call: func() {
			*observable = int(strings.Index_Byte(maximum_y, 'y'))
		}},
		{Name: "Index_Byte_Or_Non_ASCII", Call: func() {
			*observable = int(strings.Index_Byte_Or_Non_ASCII(maximum_y, "y"))
		}},
		{Name: "Last_Index_Byte", Call: func() {
			*observable = int(strings.Last_Index_Byte(maximum_y, 'y'))
		}},
		{Name: "Index_Rune", Call: func() {
			*observable = int(strings.Index_Rune(maximum_y, 'y'))
		}},
		{Name: "Index_Any", Call: func() {
			*observable = int(strings.Index_Any(maximum_y, "y"))
		}},
		{Name: "Last_Index_Any", Call: func() {
			*observable = int(strings.Last_Index_Any(maximum_y, "y"))
		}},
		{Name: "Index_Function", Call: func() {
			*observable = int(strings.Index_Function(maximum_y, allocation_y))
		}},
		{Name: "Last_Index_Function", Call: func() {
			*observable = int(
				strings.Last_Index_Function(maximum_y, allocation_y),
			)
		}},
	}
}

func collection_allocation_checks(
	observable *int, byte_storage strings.Bytes,
	text_storage strings.Texts, maximum_y strings.Text,
) (checks []allocation_check) {
	parts := [...]strings.Text{"x", "y"}
	return []allocation_check{
		{Name: "Clone_Into", Call: func() {
			*observable = len(strings.Clone_Into(byte_storage, maximum_y))
		}},
		{Name: "Split_Into", Call: func() {
			*observable = int(strings.Split_Into(text_storage, "x:y", ":"))
		}},
		{Name: "Split_N_Into", Call: func() {
			*observable = int(strings.Split_N_Into(text_storage, "x:y", ":", 1))
		}},
		{Name: "Split_After_Into", Call: func() {
			*observable = int(strings.Split_After_Into(text_storage, "x:y", ":"))
		}},
		{Name: "Split_After_N_Into", Call: func() {
			*observable = int(
				strings.Split_After_N_Into(text_storage, "x:y", ":", 1),
			)
		}},
		{Name: "Fields_Into", Call: func() {
			*observable = len(strings.Fields_Into(text_storage, "x y"))
		}},
		{Name: "Fields_Function_Into", Call: func() {
			*observable = len(
				strings.Fields_Function_Into(
					text_storage, "x y", allocation_space,
				),
			)
		}},
		{Name: "Join_Into", Call: func() {
			*observable = len(strings.Join_Into(byte_storage, parts[:], ":"))
		}},
		{Name: "Lines_Into", Call: func() {
			*observable = len(strings.Lines_Into(text_storage, "x\ny"))
		}},
	}
}

func transform_allocation_checks(
	observable *int, storage strings.Bytes, maximum_y strings.Text,
) (checks []allocation_check) {
	var turkish_storage [ucd.SPECIAL_CASE_COUNT_MAXIMUM]ucd.Case_Range
	turkish := ucd.Special_Case(ucd.Turkish_Case(turkish_storage[:]))
	return []allocation_check{
		{Name: "Map_Into", Call: func() {
			*observable = len(strings.Map_Into(storage, maximum_y, map_upper_a))
		}},
		{Name: "Repeat_Into", Call: func() {
			*observable = len(strings.Repeat_Into(storage, "xy", 2))
		}},
		{Name: "To_Upper_Into", Call: func() {
			*observable = len(strings.To_Upper_Into(storage, "go"))
		}},
		{Name: "To_Lower_Into", Call: func() {
			*observable = len(strings.To_Lower_Into(storage, "GO"))
		}},
		{Name: "To_Title_Into", Call: func() {
			*observable = len(strings.To_Title_Into(storage, "go"))
		}},
		{Name: "To_Upper_Special_Into", Call: func() {
			*observable = len(
				strings.To_Upper_Special_Into(storage, turkish, "i"),
			)
		}},
		{Name: "To_Lower_Special_Into", Call: func() {
			*observable = len(
				strings.To_Lower_Special_Into(storage, turkish, "I"),
			)
		}},
		{Name: "To_Title_Special_Into", Call: func() {
			*observable = len(
				strings.To_Title_Special_Into(storage, turkish, "i"),
			)
		}},
		{Name: "To_Valid_UTF8_Into", Call: func() {
			*observable = len(
				strings.To_Valid_UTF8_Into(
					storage, strings.Text("x\xff"), "?",
				),
			)
		}},
		{Name: "Title_Into", Call: func() {
			*observable = len(strings.Title_Into(storage, "go gopher"))
		}},
		{Name: "Replace_Into", Call: func() {
			*observable = len(strings.Replace_Into(storage, "x:x", "x", "y", -1))
		}},
		{Name: "Replace_All_Into", Call: func() {
			*observable = len(strings.Replace_All_Into(storage, "x:x", "x", "y"))
		}},
	}
}

func trim_allocation_checks(
	observable *int, maximum strings.Text,
) (checks []allocation_check) {
	return []allocation_check{
		{Name: "Trim", Call: func() {
			*observable = len(strings.Trim(maximum, "y"))
		}},
		{Name: "Trim_Left", Call: func() {
			*observable = len(strings.Trim_Left(maximum, "y"))
		}},
		{Name: "Trim_Right", Call: func() {
			*observable = len(strings.Trim_Right(maximum, "y"))
		}},
		{Name: "Trim_Function", Call: func() {
			*observable = len(strings.Trim_Function(maximum, allocation_never))
		}},
		{Name: "Trim_Left_Function", Call: func() {
			*observable = len(
				strings.Trim_Left_Function(maximum, allocation_never),
			)
		}},
		{Name: "Trim_Right_Function", Call: func() {
			*observable = len(
				strings.Trim_Right_Function(maximum, allocation_never),
			)
		}},
		{Name: "Trim_Space", Call: func() {
			*observable = len(strings.Trim_Space(maximum))
		}},
		{Name: "Trim_Prefix", Call: func() {
			*observable = len(strings.Trim_Prefix(maximum, "y"))
		}},
		{Name: "Trim_Suffix", Call: func() {
			*observable = len(strings.Trim_Suffix(maximum, "y"))
		}},
	}
}

func cut_allocation_checks(observable *int) (checks []allocation_check) {
	return []allocation_check{
		{Name: "Cut", Call: func() {
			before, after, found := strings.Cut("x:y", ":")
			*observable = len(before) + len(after) + allocation_boolean(found)
		}},
		{Name: "Cut_Prefix", Call: func() {
			after, found := strings.Cut_Prefix("prefix", "pre")
			*observable = len(after) + allocation_boolean(found)
		}},
		{Name: "Cut_Suffix", Call: func() {
			before, found := strings.Cut_Suffix("suffix", "fix")
			*observable = len(before) + allocation_boolean(found)
		}},
	}
}

func builder_allocation_checks(
	observable *int, builder *strings.Builder, source strings.Bytes,
) (checks []allocation_check) {
	return []allocation_check{
		{Name: "Builder_Bytes", Call: func() {
			*observable = len(strings.Builder_Bytes(builder))
		}},
		{Name: "Builder_Size", Call: func() {
			*observable = int(strings.Builder_Size(builder))
		}},
		{Name: "Builder_Capacity", Call: func() {
			*observable = int(strings.Builder_Capacity(builder))
		}},
		{Name: "Builder_Reset", Call: func() {
			strings.Builder_Reset(builder)
			*observable = int(builder.Size)
		}},
		{Name: "Builder_Write", Call: func() {
			*builder = strings.Builder{}
			*observable = int(strings.Builder_Write(builder, source[:2]))
		}},
		{Name: "Builder_Write_Text", Call: func() {
			*builder = strings.Builder{}
			*observable = int(strings.Builder_Write_Text(builder, "xy"))
		}},
		{Name: "Builder_Write_Byte", Call: func() {
			*builder = strings.Builder{}
			strings.Builder_Write_Byte(builder, 'x')
			*observable = int(builder.Size)
		}},
		{Name: "Builder_Write_Character", Call: func() {
			*builder = strings.Builder{}
			strings.Builder_Write_Character(builder, '☺')
			*observable = int(builder.Size)
		}},
	}
}

func reader_allocation_checks(
	observable *int, reader *strings.Reader, storage strings.Bytes,
) (checks []allocation_check) {
	return []allocation_check{
		{Name: "Reader_Reset", Call: func() {
			strings.Reader_Reset(reader, "x")
			*observable = int(reader.Position)
		}},
		{Name: "Reader_Size", Call: func() {
			*reader = strings.Reader{Source: "x", Previous: -1}
			*observable = int(strings.Reader_Size(reader))
		}},
		{Name: "Reader_Read_Into", Call: func() {
			*reader = strings.Reader{Source: "xy", Previous: -1}
			*observable = len(strings.Reader_Read_Into(reader, storage[:2]))
		}},
		{Name: "Reader_Read_Byte", Call: func() {
			*reader = strings.Reader{Source: "x", Previous: -1}
			value, found := strings.Reader_Read_Byte(reader)
			*observable = int(value) + allocation_boolean(found)
		}},
		{Name: "Reader_Unread_Byte", Call: func() {
			*reader = strings.Reader{Source: "x", Position: 1, Previous: -1}
			strings.Reader_Unread_Byte(reader)
			*observable = int(reader.Position)
		}},
		{Name: "Reader_Read_Character", Call: func() {
			*reader = strings.Reader{Source: "☺", Previous: -1}
			character, found := strings.Reader_Read_Character(reader)
			*observable = int(character) + allocation_boolean(found)
		}},
		{Name: "Reader_Unread_Character", Call: func() {
			*reader = strings.Reader{Source: "x", Position: 1, Previous: 0}
			strings.Reader_Unread_Character(reader)
			*observable = int(reader.Position)
		}},
	}
}

func replacer_allocation_checks(
	observable *int, rules strings.Rules, storage strings.Bytes,
) (checks []allocation_check) {
	replacer := strings.Replacer{Rules: rules}
	return []allocation_check{
		{Name: "Replacer_Init", Call: func() {
			value := strings.Replacer_Init(rules)
			*observable = len(value.Rules)
		}},
		{Name: "Replacer_Replace_Into", Call: func() {
			*observable = len(
				strings.Replacer_Replace_Into(storage, replacer, "x:x"),
			)
		}},
	}
}

func exercise_invariant_boundaries(
	maximum strings.Text, maximum_y strings.Text,
) {
	exercise_comparison_boundaries(maximum)
	exercise_search_boundaries(maximum, maximum_y)
	exercise_index_boundaries(maximum, maximum_y)
	exercise_trim_boundaries(maximum)
	exercise_cut_boundaries(maximum)
}

func exercise_comparison_boundaries(maximum strings.Text) {
	strings.Compare("", "")
	strings.Compare("x", "x")
	strings.Compare("xx", "xx")
	strings.Compare(maximum, maximum)
	strings.Compare("a", "b")
	strings.Compare("b", "a")
	strings.Equal_Fold("", "")
	strings.Equal_Fold("x", "x")
	strings.Equal_Fold("xx", "xx")
	strings.Equal_Fold(maximum, maximum)
	strings.Equal_Fold("x", "y")
}

func exercise_search_boundaries(
	maximum strings.Text, maximum_y strings.Text,
) {
	exercise_text_text_boolean(strings.Contains, maximum)
	exercise_text_text_boolean(strings.Contains_Any, maximum)
	exercise_text_text_boolean(strings.Has_Prefix, maximum)
	exercise_text_text_boolean(strings.Has_Suffix, maximum)
	exercise_contains_rune_boundaries(maximum)
	strings.Contains_Function("", allocation_never)
	strings.Contains_Function("x", allocation_x)
	strings.Contains_Function("xx", allocation_never)
	strings.Contains_Function(maximum, allocation_x)
	strings.Count("", maximum)
	strings.Count("x", "x")
	strings.Count("xx", "x")
	strings.Count("xx", "xx")
	strings.Count(maximum, "")
	exercise_byte_or_non_ascii_boundaries(maximum, maximum_y)
}

func exercise_text_text_boolean(
	operation func(
		strings.Text, strings.Text,
	) (result strings.Boolean),
	maximum strings.Text,
) {
	operation("", "")
	operation("x", "x")
	operation("xx", "xx")
	operation(maximum, maximum)
	operation("", maximum)
}

func exercise_contains_rune_boundaries(maximum strings.Text) {
	strings.Contains_Rune(
		"", strings.Character(strings.CHARACTER_MINIMUM),
	)
	strings.Contains_Rune("\x00", 0)
	strings.Contains_Rune("x\x01", 1)
	strings.Contains_Rune("xx", 2)
	strings.Contains_Rune(maximum, 'x')
	strings.Contains_Rune("x", -1)
	strings.Contains_Rune(
		"x", strings.Character(strings.CHARACTER_MAXIMUM),
	)
}

func exercise_byte_or_non_ascii_boundaries(
	maximum strings.Text, maximum_y strings.Text,
) {
	strings.Index_Byte_Or_Non_ASCII("", maximum)
	strings.Index_Byte_Or_Non_ASCII("x", "")
	strings.Index_Byte_Or_Non_ASCII("xx", "xx")
	strings.Index_Byte_Or_Non_ASCII("ax", "x")
	strings.Index_Byte_Or_Non_ASCII("aax", "x")
	strings.Index_Byte_Or_Non_ASCII(maximum_y, "y")
}

func exercise_index_boundaries(
	maximum strings.Text, maximum_y strings.Text,
) {
	exercise_index_text_boundaries(strings.Index, maximum, maximum_y)
	exercise_last_index_boundaries(maximum, maximum_y)
	exercise_byte_index_boundaries(strings.Index_Byte, maximum_y)
	exercise_byte_index_boundaries(strings.Last_Index_Byte, maximum_y)
	exercise_rune_index_boundaries(maximum_y)
	exercise_index_text_boundaries(strings.Index_Any, maximum, maximum_y)
	exercise_index_text_boundaries(strings.Last_Index_Any, maximum, maximum_y)
	exercise_function_index_boundaries(strings.Index_Function, maximum_y)
	exercise_function_index_boundaries(strings.Last_Index_Function, maximum_y)
}

func exercise_index_text_boundaries(
	operation func(
		strings.Text, strings.Text,
	) (index strings.Index_Value),
	maximum strings.Text, maximum_y strings.Text,
) {
	operation("", maximum)
	operation("x", "")
	operation("x", "x")
	operation("ax", "x")
	operation("aax", "x")
	operation("xx", "xx")
	operation(maximum_y, "y")
}

func exercise_last_index_boundaries(
	maximum strings.Text, maximum_y strings.Text,
) {
	strings.Last_Index("", maximum)
	strings.Last_Index("x", "x")
	strings.Last_Index("ax", "x")
	strings.Last_Index("aax", "x")
	strings.Last_Index("xx", "xx")
	strings.Last_Index(maximum_y, "y")
	strings.Last_Index(maximum, "")
}

func exercise_byte_index_boundaries(
	operation func(
		strings.Text, strings.Byte,
	) (index strings.Index_Value),
	maximum_y strings.Text,
) {
	operation("", 0)
	operation("\x01", 1)
	operation("x\x02", 2)
	operation("xx\x02", 2)
	operation(maximum_y, 'y')
	operation("x", strings.Byte(strings.BYTE_MAXIMUM))
}

func exercise_rune_index_boundaries(maximum_y strings.Text) {
	strings.Index_Rune(
		"", strings.Character(strings.CHARACTER_MINIMUM),
	)
	strings.Index_Rune("\x00", 0)
	strings.Index_Rune("x\x01", 1)
	strings.Index_Rune("xx\x02", 2)
	strings.Index_Rune(maximum_y, 'y')
	strings.Index_Rune("x", -1)
	strings.Index_Rune(
		"x", strings.Character(strings.CHARACTER_MAXIMUM),
	)
}

func exercise_function_index_boundaries(
	operation func(
		strings.Text, func(rune) (matches bool),
	) (index strings.Index_Value),
	maximum_y strings.Text,
) {
	operation("", allocation_never)
	operation("x", allocation_x)
	operation("ax", allocation_x)
	operation("aax", allocation_x)
	operation("xx", allocation_never)
	operation(maximum_y, allocation_y)
}

func exercise_trim_boundaries(maximum strings.Text) {
	exercise_cutset_trim_boundaries(strings.Trim, maximum)
	exercise_cutset_trim_boundaries(strings.Trim_Left, maximum)
	exercise_cutset_trim_boundaries(strings.Trim_Right, maximum)
	exercise_function_trim_boundaries(strings.Trim_Function, maximum)
	exercise_function_trim_boundaries(strings.Trim_Left_Function, maximum)
	exercise_function_trim_boundaries(strings.Trim_Right_Function, maximum)
	exercise_trim_space_boundaries(maximum)
	exercise_affix_trim_boundaries(strings.Trim_Prefix, maximum)
	exercise_affix_trim_boundaries(strings.Trim_Suffix, maximum)
}

func exercise_cutset_trim_boundaries(
	operation func(
		strings.Text, strings.Text,
	) (trimmed strings.Text),
	maximum strings.Text,
) {
	operation("", maximum)
	operation("x", "x")
	operation("xx", "")
	operation(" x ", " ")
	operation(" x", " ")
	operation(" x", "yz")
	operation("x ", "yz")
	operation(maximum, "")
}

func exercise_function_trim_boundaries(
	operation func(
		strings.Text, func(rune) (matches bool),
	) (trimmed strings.Text),
	maximum strings.Text,
) {
	operation("", allocation_space)
	operation("x", allocation_x)
	operation("xx", allocation_never)
	operation(" x ", allocation_space)
	operation(" x", allocation_space)
	operation("x ", allocation_space)
	operation(maximum, allocation_never)
}

func exercise_trim_space_boundaries(maximum strings.Text) {
	strings.Trim_Space("")
	strings.Trim_Space(" ")
	strings.Trim_Space("xx")
	strings.Trim_Space(" x ")
	strings.Trim_Space(" x")
	strings.Trim_Space("x ")
	strings.Trim_Space(maximum)
}

func exercise_affix_trim_boundaries(
	operation func(
		strings.Text, strings.Text,
	) (trimmed strings.Text),
	maximum strings.Text,
) {
	operation("", maximum)
	operation("x", "x")
	operation("xx", "")
	operation("xx", "x")
	operation("xx", "xx")
	operation(maximum, "")
}

func exercise_cut_boundaries(maximum strings.Text) {
	strings.Cut("", maximum)
	strings.Cut("x", "")
	strings.Cut("xx", "x")
	strings.Cut("x:xx", ":")
	strings.Cut("xx:x", ":")
	strings.Cut("x::y", "::")
	strings.Cut(maximum, "y")
	strings.Cut(maximum, "")
	exercise_cut_prefix_boundaries(maximum)
	exercise_cut_suffix_boundaries(maximum)
}

func exercise_cut_prefix_boundaries(maximum strings.Text) {
	strings.Cut_Prefix("", maximum)
	strings.Cut_Prefix("x", "")
	strings.Cut_Prefix("x", "x")
	strings.Cut_Prefix("xx", "x")
	strings.Cut_Prefix("xx", "xx")
	strings.Cut_Prefix("xxx", "x")
	strings.Cut_Prefix(maximum, "")
	strings.Cut_Prefix(maximum, "y")
}

func exercise_cut_suffix_boundaries(maximum strings.Text) {
	strings.Cut_Suffix("", maximum)
	strings.Cut_Suffix("x", "")
	strings.Cut_Suffix("x", "x")
	strings.Cut_Suffix("xx", "x")
	strings.Cut_Suffix("xx", "xx")
	strings.Cut_Suffix("xxx", "x")
	strings.Cut_Suffix(maximum, "")
	strings.Cut_Suffix(maximum, "y")
}

func allocation_space(character rune) (matches bool) {
	return character == ' '
}

func allocation_never(rune) (matches bool) {
	return false
}

func allocation_x(character rune) (matches bool) {
	return character == 'x'
}

func allocation_y(character rune) (matches bool) {
	return character == 'y'
}

func allocation_boolean(value strings.Boolean) (number int) {
	if value {
		return 1
	}
	return 0
}

func exercise_destination_boundaries(
	byte_storage strings.Bytes, text_storage strings.Texts,
	maximum strings.Text, maximum_separator strings.Text,
	maximum_fields strings.Text, maximum_lines strings.Text,
) {
	exercise_collection_boundaries(
		byte_storage, text_storage, maximum, maximum_separator,
		maximum_fields, maximum_lines,
	)
	exercise_transform_boundaries(byte_storage, maximum)
	exercise_builder_boundaries(byte_storage, maximum)
	exercise_reader_boundaries(byte_storage, maximum)
	exercise_replacer_boundaries(byte_storage, maximum)
}

func exercise_collection_boundaries(
	byte_storage strings.Bytes, text_storage strings.Texts,
	maximum strings.Text, maximum_separator strings.Text,
	maximum_fields strings.Text, maximum_lines strings.Text,
) {
	exercise_split_boundaries(
		strings.Split_Into, text_storage, maximum, maximum_separator,
	)
	exercise_split_boundaries(
		strings.Split_After_Into, text_storage, maximum, maximum_separator,
	)
	exercise_limited_split_boundaries(
		strings.Split_N_Into, text_storage, maximum, maximum_separator,
	)
	exercise_limited_split_boundaries(
		strings.Split_After_N_Into, text_storage, maximum, maximum_separator,
	)
	exercise_fields_boundaries(text_storage, maximum_fields)
	exercise_lines_boundaries(text_storage, maximum_lines)
	exercise_join_boundaries(byte_storage, text_storage, maximum)
}

func exercise_split_boundaries(
	operation func(
		strings.Texts, strings.Text, strings.Text,
	) (count strings.Count_Value),
	storage strings.Texts, maximum strings.Text,
	maximum_separator strings.Text,
) {
	operation(storage[:0], "", "")
	operation(storage[:1], "x", "")
	operation(storage[:2], "xx", "")
	operation(storage, maximum_separator, ":")
	operation(storage[:2], "xx", "xx")
	operation(storage[:2], maximum, maximum)
}

func exercise_limited_split_boundaries(
	operation func(
		strings.Texts, strings.Text, strings.Text,
		strings.Limit,
	) (count strings.Count_Value),
	storage strings.Texts, maximum strings.Text,
	maximum_separator strings.Text,
) {
	operation(storage[:0], "", ":", 0)
	operation(storage[:1], "x", "", 1)
	operation(storage[:2], "xx", "", 2)
	operation(storage, maximum_separator, ":", strings.LIMIT_MAXIMUM)
	operation(storage, maximum, "", strings.LIMIT_MAXIMUM)
	operation(storage[:2], maximum, maximum, strings.LIMIT_MINIMUM)
	operation(storage[:2], "xx", "xx", strings.LIMIT_MINIMUM)
}

func exercise_fields_boundaries(
	storage strings.Texts, maximum_fields strings.Text,
) {
	strings.Fields_Into(storage[:0], "")
	strings.Fields_Into(storage[:1], "x")
	strings.Fields_Into(storage[:1], "xx")
	strings.Fields_Into(storage[:2], "x y")
	strings.Fields_Into(storage, maximum_fields)
	strings.Fields_Function_Into(storage[:0], "", allocation_space)
	strings.Fields_Function_Into(storage[:1], "x", allocation_space)
	strings.Fields_Function_Into(storage[:1], "xx", allocation_space)
	strings.Fields_Function_Into(storage[:2], "x y", allocation_space)
	strings.Fields_Function_Into(storage, maximum_fields, allocation_space)
}

func exercise_lines_boundaries(
	storage strings.Texts, maximum_lines strings.Text,
) {
	strings.Lines_Into(storage[:0], "")
	strings.Lines_Into(storage[:1], "x")
	strings.Lines_Into(storage[:1], "xx")
	strings.Lines_Into(storage[:2], "x\ny")
	strings.Lines_Into(storage, maximum_lines)
}

func exercise_join_boundaries(
	storage strings.Bytes, text_storage strings.Texts,
	maximum strings.Text,
) {
	text_storage[0] = "x"
	text_storage[1] = "x"
	strings.Join_Into(storage[:0], text_storage[:0], maximum)
	strings.Join_Into(storage[:1], text_storage[:1], "")
	strings.Join_Into(storage[:2], text_storage[:2], "")
	text_storage[0] = maximum
	strings.Join_Into(storage, text_storage[:1], "xx")
	text_storage[0] = "xx"
	strings.Join_Into(storage[:2], text_storage[:1], "x")
	var empty_parts [strings.TEXT_COUNT_MAXIMUM]strings.Text
	strings.Join_Into(storage[:0], empty_parts[:], "")
}

func exercise_transform_boundaries(
	storage strings.Bytes, maximum strings.Text,
) {
	exercise_text_transform_boundaries(strings.Clone_Into, storage, maximum)
	exercise_text_transform_boundaries(strings.To_Upper_Into, storage, maximum)
	exercise_text_transform_boundaries(strings.To_Lower_Into, storage, maximum)
	exercise_text_transform_boundaries(strings.To_Title_Into, storage, maximum)
	exercise_map_boundaries(storage, maximum)
	exercise_special_case_boundaries(storage, maximum)
	exercise_valid_utf8_boundaries(storage, maximum)
	exercise_title_boundaries(storage, maximum)
	exercise_repeat_boundaries(storage, maximum)
	exercise_replace_boundaries(storage, maximum)
}

func exercise_text_transform_boundaries(
	operation func(
		strings.Bytes, strings.Text,
	) (result strings.Bytes),
	storage strings.Bytes, maximum strings.Text,
) {
	operation(storage[:0], "")
	operation(storage[:1], "x")
	operation(storage[:2], "xx")
	operation(storage, maximum)
}

func exercise_map_boundaries(
	storage strings.Bytes, maximum strings.Text,
) {
	strings.Map_Into(storage[:0], "", map_upper_a)
	strings.Map_Into(storage[:1], "x", map_upper_a)
	strings.Map_Into(storage[:2], "xx", map_upper_a)
	strings.Map_Into(storage, maximum, map_upper_a)
	strings.Map_Into(storage[:0], "x", allocation_drop)
}

func exercise_special_case_boundaries(
	storage strings.Bytes, maximum strings.Text,
) {
	var turkish_storage [ucd.SPECIAL_CASE_COUNT_MAXIMUM]ucd.Case_Range
	turkish := ucd.Special_Case(ucd.Turkish_Case(turkish_storage[:]))
	exercise_special_transform_boundaries(
		strings.To_Upper_Special_Into, storage, maximum, turkish,
	)
	exercise_special_transform_boundaries(
		strings.To_Lower_Special_Into, storage, maximum, turkish,
	)
	exercise_special_transform_boundaries(
		strings.To_Title_Special_Into, storage, maximum, turkish,
	)
}

func exercise_special_transform_boundaries(
	operation func(
		strings.Bytes, ucd.Special_Case, strings.Text,
	) (result strings.Bytes),
	storage strings.Bytes, maximum strings.Text,
	turkish ucd.Special_Case,
) {
	operation(storage[:0], turkish[:0], "")
	operation(storage[:1], turkish[:1], "x")
	operation(storage[:2], turkish[:2], "xx")
	operation(storage, turkish, maximum)
}

func exercise_valid_utf8_boundaries(
	storage strings.Bytes, maximum strings.Text,
) {
	strings.To_Valid_UTF8_Into(storage[:0], "", maximum)
	strings.To_Valid_UTF8_Into(storage[:1], "x", "")
	strings.To_Valid_UTF8_Into(storage[:2], "xx", "xx")
	strings.To_Valid_UTF8_Into(storage, maximum, "x")
	strings.To_Valid_UTF8_Into(storage[:1], strings.Text("\xff"), "x")
}

func exercise_title_boundaries(
	storage strings.Bytes, maximum strings.Text,
) {
	strings.Title_Into(storage[:0], "")
	strings.Title_Into(storage[:1], "x")
	strings.Title_Into(storage[:2], "xx")
	strings.Title_Into(storage, maximum)
	strings.Title_Into(storage, "\x00\x01\x02\U0010ffff go")
}

func exercise_repeat_boundaries(
	storage strings.Bytes, maximum strings.Text,
) {
	strings.Repeat_Into(storage[:0], "", 0)
	strings.Repeat_Into(storage[:1], "x", 1)
	strings.Repeat_Into(storage[:2], "x", 2)
	strings.Repeat_Into(storage, "x", strings.REPEAT_COUNT_MAXIMUM)
	strings.Repeat_Into(storage, maximum, 1)
}

func exercise_replace_boundaries(
	storage strings.Bytes, maximum strings.Text,
) {
	strings.Replace_Into(storage[:0], "", maximum, maximum, 0)
	strings.Replace_Into(storage[:1], "x", "", "", 1)
	strings.Replace_Into(storage[:2], "xx", "xx", "xx", 2)
	strings.Replace_Into(
		storage, maximum, "x", "x", strings.REPLACEMENT_COUNT_MAXIMUM,
	)
	strings.Replace_Into(storage, maximum, "x", "x", -1)
	strings.Replace_All_Into(storage[:0], "", maximum, maximum)
	strings.Replace_All_Into(storage[:1], "x", "", "")
	strings.Replace_All_Into(storage[:2], "xx", "xx", "xx")
	strings.Replace_All_Into(storage, maximum, "x", "x")
}

func allocation_drop(rune) (mapped rune) {
	return -1
}

func exercise_builder_boundaries(
	storage strings.Bytes, maximum strings.Text,
) {
	exercise_builder_query_boundaries(maximum)
	exercise_builder_write_boundaries(storage, maximum)
	exercise_builder_scalar_write_boundaries()
}

func exercise_builder_query_boundaries(maximum strings.Text) {
	builders := [...]strings.Builder{
		{},
		{Size: 1},
		{Size: 2},
		{Size: strings.SIZE_VALUE_MAXIMUM},
	}
	for builder_index := range builders {
		builder := &builders[builder_index]
		strings.Builder_Bytes(builder)
		strings.Builder_Size(builder)
		strings.Builder_Capacity(builder)
	}
	for builder_index := range builders {
		builder := builders[builder_index]
		strings.Builder_Reset(&builder)
	}
	var full strings.Builder
	strings.Builder_Write_Text(&full, maximum)
	strings.Builder_Bytes(&full)
}

func exercise_builder_write_boundaries(
	storage strings.Bytes, maximum strings.Text,
) {
	var builder strings.Builder
	strings.Builder_Write(&builder, storage[:0])
	builder = strings.Builder{}
	strings.Builder_Write(&builder, storage[:1])
	builder = strings.Builder{}
	strings.Builder_Write(&builder, storage[:2])
	builder = strings.Builder{}
	strings.Builder_Write(&builder, storage)
	builder = strings.Builder{Size: strings.SIZE_VALUE_MAXIMUM}
	strings.Builder_Write(&builder, storage[:0])
	builder = strings.Builder{Size: 1}
	strings.Builder_Write(&builder, storage[:0])
	builder = strings.Builder{Size: 2}
	strings.Builder_Write(&builder, storage[:0])
	builder = strings.Builder{}
	strings.Builder_Write_Text(&builder, "")
	builder = strings.Builder{}
	strings.Builder_Write_Text(&builder, "x")
	builder = strings.Builder{}
	strings.Builder_Write_Text(&builder, "xx")
	builder = strings.Builder{}
	strings.Builder_Write_Text(&builder, maximum)
	builder = strings.Builder{Size: strings.SIZE_VALUE_MAXIMUM}
	strings.Builder_Write_Text(&builder, "")
	builder = strings.Builder{Size: 1}
	strings.Builder_Write_Text(&builder, "")
	builder = strings.Builder{Size: 2}
	strings.Builder_Write_Text(&builder, "")
}

func exercise_builder_scalar_write_boundaries() {
	builder := strings.Builder{}
	strings.Builder_Write_Byte(&builder, 0)
	builder = strings.Builder{Size: 1}
	strings.Builder_Write_Byte(&builder, 1)
	builder = strings.Builder{Size: 2}
	strings.Builder_Write_Byte(&builder, 2)
	builder = strings.Builder{Size: strings.SIZE_VALUE_MAXIMUM}
	exercise_panicking_call(func() {
		strings.Builder_Write_Byte(
			&builder, strings.Byte(strings.BYTE_MAXIMUM),
		)
	})
	exercise_builder_character_boundaries()
}

func exercise_builder_character_boundaries() {
	builder := strings.Builder{}
	strings.Builder_Write_Character(&builder, 0)
	builder = strings.Builder{Size: 1}
	strings.Builder_Write_Character(&builder, 1)
	builder = strings.Builder{Size: 2}
	strings.Builder_Write_Character(&builder, 2)
	builder = strings.Builder{}
	strings.Builder_Write_Character(
		&builder, strings.Character(strings.CHARACTER_MINIMUM),
	)
	builder = strings.Builder{}
	strings.Builder_Write_Character(
		&builder, strings.Character(strings.CHARACTER_MAXIMUM),
	)
	builder = strings.Builder{Size: strings.SIZE_VALUE_MAXIMUM}
	exercise_panicking_call(func() {
		strings.Builder_Write_Character(&builder, -1)
	})
}

func exercise_reader_boundaries(
	storage strings.Bytes, maximum strings.Text,
) {
	exercise_reader_reset_boundaries(maximum)
	exercise_reader_size_boundaries(maximum)
	exercise_reader_read_into_boundaries(storage, maximum)
	exercise_reader_byte_boundaries(maximum)
	exercise_reader_character_boundaries(maximum)
	exercise_reader_unread_boundaries(maximum)
}

func exercise_reader_reset_boundaries(maximum strings.Text) {
	reader := strings.Reader{Previous: -1}
	strings.Reader_Reset(&reader, "")
	reader = strings.Reader{Source: "x", Position: 1, Previous: 0}
	strings.Reader_Reset(&reader, "x")
	reader = strings.Reader{Source: "xx", Position: 2, Previous: 1}
	strings.Reader_Reset(&reader, "xx")
	reader = strings.Reader{Source: "xxx", Position: 3, Previous: 2}
	strings.Reader_Reset(&reader, "xxx")
	reader = strings.Reader{
		Source: maximum, Position: strings.SIZE_VALUE_MAXIMUM,
		Previous: strings.INDEX_MAXIMUM,
	}
	strings.Reader_Reset(&reader, maximum)
}

func exercise_reader_size_boundaries(maximum strings.Text) {
	reader := strings.Reader{Previous: -1}
	strings.Reader_Size(&reader)
	reader = strings.Reader{Source: "x", Previous: 0}
	strings.Reader_Size(&reader)
	reader = strings.Reader{Source: "xx", Previous: 1}
	strings.Reader_Size(&reader)
	reader = strings.Reader{Source: "xx", Position: 2, Previous: 1}
	strings.Reader_Size(&reader)
	reader = strings.Reader{Source: "xxx", Position: 1, Previous: 2}
	strings.Reader_Size(&reader)
	reader = strings.Reader{Source: maximum, Previous: 2}
	strings.Reader_Size(&reader)
	reader = strings.Reader{
		Source: maximum, Position: strings.SIZE_VALUE_MAXIMUM,
		Previous: strings.INDEX_MAXIMUM,
	}
	strings.Reader_Size(&reader)
}

func exercise_reader_read_into_boundaries(
	storage strings.Bytes, maximum strings.Text,
) {
	reader := strings.Reader{Previous: -1}
	strings.Reader_Read_Into(&reader, storage[:0])
	reader = strings.Reader{Source: "x", Previous: 0}
	strings.Reader_Read_Into(&reader, storage[:1])
	reader = strings.Reader{Source: "xx", Previous: 1}
	strings.Reader_Read_Into(&reader, storage[:2])
	reader = strings.Reader{Source: "xx", Position: 2, Previous: 1}
	strings.Reader_Read_Into(&reader, storage[:0])
	reader = strings.Reader{Source: maximum, Previous: 2}
	strings.Reader_Read_Into(&reader, storage)
	reader = strings.Reader{
		Source: maximum, Position: strings.SIZE_VALUE_MAXIMUM,
		Previous: strings.INDEX_MAXIMUM,
	}
	strings.Reader_Read_Into(&reader, storage[:0])
}

func exercise_reader_byte_boundaries(maximum strings.Text) {
	reader := strings.Reader{Previous: -1}
	strings.Reader_Read_Byte(&reader)
	reader = strings.Reader{Source: "\x00", Previous: 0}
	strings.Reader_Read_Byte(&reader)
	reader = strings.Reader{Source: "\x01x", Previous: 1}
	strings.Reader_Read_Byte(&reader)
	reader = strings.Reader{Source: "\x02xx", Previous: 2}
	strings.Reader_Read_Byte(&reader)
	reader = strings.Reader{Source: "\xff", Previous: -1}
	strings.Reader_Read_Byte(&reader)
	reader = strings.Reader{Source: "x", Position: 1, Previous: 0}
	strings.Reader_Read_Byte(&reader)
	reader = strings.Reader{Source: "xx", Position: 2, Previous: 1}
	strings.Reader_Read_Byte(&reader)
	reader = strings.Reader{
		Source: maximum, Position: strings.SIZE_VALUE_MAXIMUM,
		Previous: strings.INDEX_MAXIMUM,
	}
	strings.Reader_Read_Byte(&reader)
}

func exercise_reader_character_boundaries(maximum strings.Text) {
	reader := strings.Reader{Previous: -1}
	strings.Reader_Read_Character(&reader)
	reader = strings.Reader{Source: "\x00", Previous: 0}
	strings.Reader_Read_Character(&reader)
	reader = strings.Reader{Source: "\x01x", Previous: 1}
	strings.Reader_Read_Character(&reader)
	reader = strings.Reader{Source: "\x02xx", Previous: 2}
	strings.Reader_Read_Character(&reader)
	reader = strings.Reader{Source: "\U0010ffff", Previous: -1}
	strings.Reader_Read_Character(&reader)
	reader = strings.Reader{Source: "x", Position: 1, Previous: 0}
	strings.Reader_Read_Character(&reader)
	reader = strings.Reader{Source: "xx", Position: 2, Previous: 1}
	strings.Reader_Read_Character(&reader)
	reader = strings.Reader{
		Source: maximum, Position: strings.SIZE_VALUE_MAXIMUM,
		Previous: strings.INDEX_MAXIMUM,
	}
	strings.Reader_Read_Character(&reader)
}

func exercise_reader_unread_boundaries(maximum strings.Text) {
	reader := strings.Reader{Previous: -1}
	exercise_panicking_call(func() { strings.Reader_Unread_Byte(&reader) })
	reader = strings.Reader{Source: "x", Position: 1, Previous: 0}
	strings.Reader_Unread_Byte(&reader)
	reader = strings.Reader{Source: "xx", Position: 2, Previous: 1}
	strings.Reader_Unread_Byte(&reader)
	reader = strings.Reader{Source: "xxx", Position: 3, Previous: 2}
	strings.Reader_Unread_Byte(&reader)
	reader = strings.Reader{
		Source: maximum, Position: strings.SIZE_VALUE_MAXIMUM,
		Previous: strings.INDEX_MAXIMUM,
	}
	strings.Reader_Unread_Byte(&reader)
	exercise_reader_unread_character_boundaries(maximum)
}

func exercise_reader_unread_character_boundaries(maximum strings.Text) {
	reader := strings.Reader{Previous: -1}
	exercise_panicking_call(func() { strings.Reader_Unread_Character(&reader) })
	reader = strings.Reader{Source: "x", Position: 1, Previous: 0}
	strings.Reader_Unread_Character(&reader)
	reader = strings.Reader{Source: "xx", Position: 2, Previous: 1}
	strings.Reader_Unread_Character(&reader)
	reader = strings.Reader{Source: "xxx", Position: 3, Previous: 2}
	strings.Reader_Unread_Character(&reader)
	reader = strings.Reader{
		Source: maximum, Position: strings.SIZE_VALUE_MAXIMUM,
		Previous: strings.INDEX_MAXIMUM,
	}
	strings.Reader_Unread_Character(&reader)
}

func exercise_replacer_boundaries(
	storage strings.Bytes, maximum strings.Text,
) {
	var maximum_rules [strings.RULE_COUNT_MAXIMUM]strings.Rule
	exercise_replacer_init_boundaries(maximum_rules[:], maximum)
	exercise_replacer_replace_boundaries(storage, maximum_rules[:], maximum)
}

func exercise_replacer_init_boundaries(
	maximum_rules strings.Rules, maximum strings.Text,
) {
	strings.Replacer_Init(maximum_rules[:0])
	one := [...]strings.Rule{{Old: "x", New: "x"}}
	strings.Replacer_Init(one[:])
	two := [...]strings.Rule{{Old: "xx", New: "xx"}, {}}
	strings.Replacer_Init(two[:])
	strings.Replacer_Init(maximum_rules)
	one[0] = strings.Rule{Old: "", New: ""}
	strings.Replacer_Init(one[:])
	one[0] = strings.Rule{Old: "x", New: "x"}
	strings.Replacer_Init(one[:])
	one[0] = strings.Rule{Old: "xx", New: "xx"}
	strings.Replacer_Init(one[:])
	one[0] = strings.Rule{
		Old: strings.Old_Text(maximum),
		New: strings.New_Text(maximum),
	}
	strings.Replacer_Init(one[:])
}

func exercise_replacer_replace_boundaries(
	storage strings.Bytes, maximum_rules strings.Rules,
	maximum strings.Text,
) {
	strings.Replacer_Replace_Into(
		storage[:0], strings.Replacer{}, "",
	)
	one := [...]strings.Rule{{Old: "x", New: "x"}}
	strings.Replacer_Replace_Into(
		storage[:1], strings.Replacer{Rules: one[:]}, "x",
	)
	two := [...]strings.Rule{{Old: "xx", New: "xx"}, {}}
	strings.Replacer_Replace_Into(
		storage[:2], strings.Replacer{Rules: two[:]}, "xx",
	)
	maximum_rules[0] = strings.Rule{Old: "x", New: "x"}
	strings.Replacer_Replace_Into(
		storage, strings.Replacer{Rules: maximum_rules}, maximum,
	)
	one[0] = strings.Rule{Old: "", New: ""}
	strings.Replacer_Replace_Into(
		storage[:0], strings.Replacer{Rules: one[:]}, "",
	)
	one[0] = strings.Rule{
		Old: strings.Old_Text(maximum),
		New: strings.New_Text(maximum),
	}
	strings.Replacer_Replace_Into(
		storage, strings.Replacer{Rules: one[:]}, maximum,
	)
}

func exercise_panicking_call(callback func()) {
	defer func() { recover() }()
	callback()
}
