package strings_test

import (
	"reflect"
	"testing"

	shared_io "local/james-orcales/shared/io"
	shared_strings "local/james-orcales/shared/strings"
	"local/james-orcales/shared/unicode/ucd"
	"local/james-orcales/shared/unicode/utf8"
)

// Test_Comparison verifies lexical order, Unicode folding, and independent cloning.
func Test_Comparison(t *testing.T) {
	t.Parallel()
	for _, one := range []struct {
		Left  string
		Right string
		Want  shared_strings.Order
	}{
		{Left: "", Right: "", Want: shared_strings.ORDER_EQUAL},
		{Left: "a", Right: "b", Want: shared_strings.ORDER_BEFORE},
		{Left: "b", Right: "a", Want: shared_strings.ORDER_AFTER},
		{Left: "abc", Right: "ab", Want: shared_strings.ORDER_AFTER},
	} {
		got := shared_strings.Compare(text(one.Left), text(one.Right))
		if got != one.Want {
			t.Fatalf(
				"Compare(%q, %q) = %d, want %d", one.Left, one.Right, got, one.Want,
			)
		}
	}
	if !shared_strings.Equal_Fold("Go", "gO") {
		t.Fatal("Equal_Fold must apply Unicode simple folding")
	}
	if shared_strings.Equal_Fold("Go", "stop") {
		t.Fatal("Equal_Fold must reject different text")
	}
	source := text("clone")
	clone := shared_strings.Clone(source)
	if clone != source {
		t.Fatal("Clone must keep the text value")
	}
}

// Test_Search verifies every search form, empty separators, invalid UTF-8, and absence.
func Test_Search(t *testing.T) {
	t.Parallel()
	full := text(fixture_repeat("a", shared_strings.TEXT_SIZE_MAXIMUM-1) + "z")
	if shared_strings.Count(full, "") != shared_strings.TEXT_SIZE_MAXIMUM+1 {
		t.Fatal("an empty separator must occur at every character boundary")
	}
	if !shared_strings.Contains(full, "z") {
		t.Fatal("Contains must report present and absent text")
	}
	if shared_strings.Contains(full, "x") {
		t.Fatal("Contains must report present and absent text")
	}
	if !shared_strings.Contains_Any("abc", "zc") {
		t.Fatal("Contains_Any must search a character set")
	}
	if shared_strings.Contains_Any("abc", "xy") {
		t.Fatal("Contains_Any must search a character set")
	}
	if !shared_strings.Contains_Rune("a☺", '☺') {
		t.Fatal("Contains_Rune must find a Unicode character")
	}
	space := func(character rune) (matches bool) {
		return bool(ucd.Is_Space(ucd.Character(character)))
	}
	if !shared_strings.Contains_Function("a b", space) {
		t.Fatal("Contains_Function must find a predicate match")
	}
	if shared_strings.Index(full, "z") != shared_strings.TEXT_SIZE_MAXIMUM-1 {
		t.Fatal("Index must reach the final byte")
	}
	if shared_strings.Last_Index("abcabc", "bc") != 4 {
		t.Fatal("Last_Index must find the final match")
	}
	if shared_strings.Last_Index("abc", "") != 3 {
		t.Fatal("Last_Index must return the final boundary for an empty separator")
	}
	if shared_strings.Index_Byte(full, 'z') != shared_strings.TEXT_SIZE_MAXIMUM-1 {
		t.Fatal("Index_Byte must reach the final byte")
	}
	verify_byte_or_non_ascii_search(t)
	if shared_strings.Last_Index_Byte(full, 'x') != shared_strings.INDEX_ABSENT {
		t.Fatal("Last_Index_Byte must report an absent byte")
	}
	if shared_strings.Index_Rune("a☺b", '☺') != 1 {
		t.Fatal("Index_Rune must return a byte index")
	}
	if shared_strings.Index_Rune(
		text("\xff"), shared_strings.Character(utf8.REPLACEMENT_CHARACTER),
	) != 0 {
		t.Fatal("Index_Rune must find invalid UTF-8 through RuneError")
	}
	if shared_strings.Index_Any(full, "z") != shared_strings.TEXT_SIZE_MAXIMUM-1 {
		t.Fatal("Index_Any must reach the final byte")
	}
	if shared_strings.Last_Index_Any("abcabc", "ca") != 5 {
		t.Fatal("Last_Index_Any must find the last set member")
	}
	if shared_strings.Index_Function("a b", space) != 1 {
		t.Fatal("Index_Function must find the first predicate match")
	}
	if shared_strings.Last_Index_Function("a b c", space) != 3 {
		t.Fatal("Last_Index_Function must find the last predicate match")
	}
}

// Test_Split_And_Join verifies collection forms, field forms, limits, and empty separators.
func Test_Split_And_Join(t *testing.T) {
	t.Parallel()
	assert_texts(t, shared_strings.Split("a,b,c", ","), []string{"a", "b", "c"})
	assert_texts(t, shared_strings.Split_After("a,b,c", ","), []string{"a,", "b,", "c"})
	assert_texts(t, shared_strings.Split_N("a,b,c", ",", 2), []string{"a", "b,c"})
	assert_texts(
		t, shared_strings.Split_After_N("a,b,c", ",", 2), []string{"a,", "b,c"},
	)
	assert_texts(t, shared_strings.Split("", ""), []string{})
	assert_texts(t, shared_strings.Split_After("", ""), []string{})
	assert_texts(t, shared_strings.Split("\xff", ""), []string{"\xff"})
	separators := text(fixture_repeat("x", shared_strings.TEXT_SIZE_MAXIMUM))
	if len(shared_strings.Split(separators, "x")) != shared_strings.TEXTS_COUNT_MAXIMUM {
		t.Fatal("Split must reach the text-count maximum")
	}
	space := func(character rune) (matches bool) {
		return bool(ucd.Is_Space(ucd.Character(character)))
	}
	for _, one := range []struct {
		Source string
		Want   []string
	}{
		{Source: "", Want: []string{}},
		{Source: " a\tb\n", Want: []string{"a", "b"}},
		{Source: "\u2000a\u2000b", Want: []string{"a", "b"}},
		{Source: "abc", Want: []string{"abc"}},
	} {
		assert_texts(t, shared_strings.Fields(text(one.Source)), one.Want)
		assert_texts(t, shared_strings.Fields_Function(text(one.Source), space), one.Want)
	}
	parts := shared_strings.Texts{"a", "b", ""}
	if got := shared_strings.Join(parts, ","); got != "a,b," {
		t.Fatalf("Join = %q, want %q", got, "a,b,")
	}
}

// Test_Transform verifies mapping, repetition, case forms, UTF-8 repair, and replacement.
func Test_Transform(t *testing.T) {
	t.Parallel()
	mapping := func(character rune) (mapped rune) {
		if character == 'a' {
			return '☺'
		}
		if character == 'x' {
			return -1
		}
		return character
	}
	for _, one := range []struct {
		Source string
		Want   string
	}{
		{Source: "", Want: ""},
		{Source: "abx", Want: "☺b"},
		{Source: "Hello, 世界", Want: "Hello, 世界"},
	} {
		if got := shared_strings.Map(mapping, text(one.Source)); string(got) != one.Want {
			t.Fatalf("Map(%q) = %q, want %q", one.Source, got, one.Want)
		}
	}
	if got := shared_strings.Repeat("ab", 3); got != "ababab" {
		t.Fatalf("Repeat = %q, want %q", got, "ababab")
	}
	upper := shared_strings.To_Upper("Hello, 世界")
	if string(upper) != "HELLO, 世界" {
		t.Fatalf("To_Upper = %q, want %q", upper, "HELLO, 世界")
	}
	lower := shared_strings.To_Lower("Hello, 世界")
	if string(lower) != "hello, 世界" {
		t.Fatalf("To_Lower = %q, want %q", lower, "hello, 世界")
	}
	title := shared_strings.To_Title("hello")
	if string(title) != "HELLO" {
		t.Fatalf("To_Title = %q, want %q", title, "HELLO")
	}
	if got := shared_strings.To_Upper_Special(turkish_case(), "i"); got != "İ" {
		t.Fatal("To_Upper_Special must apply the supplied case")
	}
	if got := shared_strings.To_Lower_Special(turkish_case(), "I"); got != "ı" {
		t.Fatal("To_Lower_Special must apply the supplied case")
	}
	if got := shared_strings.To_Title_Special(turkish_case(), "i"); got != "İ" {
		t.Fatal("To_Title_Special must apply the supplied case")
	}
	if got := shared_strings.To_Valid_UTF8(text("a\xff\xfeb"), "?"); got != "a?b" {
		t.Fatalf("To_Valid_UTF8 = %q, want %q", got, "a?b")
	}
	word_title := shared_strings.Title("hello-world")
	if string(word_title) != "Hello-World" {
		t.Fatalf("Title = %q, want %q", word_title, "Hello-World")
	}
	verify_transform_replacement(t)
}

// Test_Trim_And_Cut verifies Unicode trim forms and the prefix, suffix, and cut forms.
func Test_Trim_And_Cut(t *testing.T) {
	t.Parallel()
	verify_trim_forms(t)
	if shared_strings.Trim_Prefix("prefix", "pre") != "fix" {
		t.Fatal("Trim_Prefix must remove a present prefix")
	}
	if shared_strings.Trim_Suffix("suffix", "fix") != "suf" {
		t.Fatal("Trim_Suffix must remove a present suffix")
	}
	before, after, found := shared_strings.Cut("a:b", ":")
	if before != "a" {
		t.Fatal("Cut must divide text around the first separator")
	}
	if after != "b" {
		t.Fatal("Cut must divide text around the first separator")
	}
	if !found {
		t.Fatal("Cut must divide text around the first separator")
	}
	after, found = shared_strings.Cut_Prefix("prefix", "pre")
	if after != "fix" {
		t.Fatal("Cut_Prefix must report a present prefix")
	}
	if !found {
		t.Fatal("Cut_Prefix must report a present prefix")
	}
	before, found = shared_strings.Cut_Suffix("suffix", "fix")
	if before != "suf" {
		t.Fatal("Cut_Suffix must report a present suffix")
	}
	if !found {
		t.Fatal("Cut_Suffix must report a present suffix")
	}
	if !shared_strings.Has_Prefix("prefix", "pre") {
		t.Fatal("the prefix and suffix queries must report present values")
	}
	if !shared_strings.Has_Suffix("suffix", "fix") {
		t.Fatal("the prefix and suffix queries must report present values")
	}
}

// Test_Iteration verifies all sequence forms and an early consumer stop.
func Test_Iteration(t *testing.T) {
	t.Parallel()
	assert_texts(t, collect(shared_strings.Lines("a\nb")), []string{"a\n", "b"})
	assert_texts(t, collect(shared_strings.Split_Sequence("a,b,c", ",")),
		[]string{"a", "b", "c"})
	assert_texts(t, collect(shared_strings.Split_After_Sequence("a,b,c", ",")),
		[]string{"a,", "b,", "c"})
	assert_texts(t, collect(shared_strings.Fields_Sequence(" a\tb ")),
		[]string{"a", "b"})
	comma := func(character rune) (matches bool) { return character == ',' }
	assert_texts(t, collect(shared_strings.Fields_Function_Sequence("a,,b", comma)),
		[]string{"a", "b"})
	count := 0
	for range shared_strings.Lines("a\nb\n") {
		count++
		break
	}
	if count != 1 {
		t.Fatal("a sequence must stop after its consumer stops")
	}
}

// Test_Builder verifies each write form, capacity growth, and reset.
func Test_Builder(t *testing.T) {
	t.Parallel()
	var builder shared_strings.Builder
	shared_strings.Builder_Grow(&builder, 16)
	if shared_strings.Builder_Capacity(&builder) < 16 {
		t.Fatal("Builder_Grow must reserve the requested capacity")
	}
	stream := shared_strings.Builder_To_Stream(&builder)
	write_count, write_error := shared_io.Write(stream, []byte("ab"))
	if write_count != 2 {
		t.Fatalf("the Builder stream write count = %d, want 2", write_count)
	}
	if write_error != nil {
		t.Fatalf("the Builder stream write returned %v", write_error)
	}
	byte_error := shared_io.Write_Byte(stream, 'c')
	if byte_error != nil {
		t.Fatalf("the Builder stream byte write returned %v", byte_error)
	}
	character_count, character_error := shared_io.Write_Rune(stream, '☺')
	if character_count != 3 {
		t.Fatalf("the Builder character write count = %d, want 3", character_count)
	}
	if character_error != nil {
		t.Fatalf("the Builder character write returned %v", character_error)
	}
	text_count, text_error := shared_io.Write_String(stream, "de")
	if text_count != 2 {
		t.Fatalf("the Builder text write count = %d, want 2", text_count)
	}
	if text_error != nil {
		t.Fatalf("the Builder text write returned %v", text_error)
	}
	if shared_strings.Builder_Text(&builder) != "abc☺de" {
		t.Fatalf(
			"Builder_Text = %q, want %q",
			shared_strings.Builder_Text(&builder),
			"abc☺de",
		)
	}
	if int(shared_strings.Builder_Size(&builder)) != len("abc☺de") {
		t.Fatal("Builder_Size must report the byte count")
	}
	shared_strings.Builder_Reset(&builder)
	if shared_strings.Builder_Text(&builder) != "" {
		t.Fatal("Builder_Reset must remove all content")
	}
	if shared_strings.Builder_Size(&builder) != 0 {
		t.Fatal("Builder_Reset must remove all content")
	}
}

// Test_Interface_Boundary verifies that package values expose only package functions.
func Test_Interface_Boundary(t *testing.T) {
	t.Parallel()
	for _, value := range []any{
		&shared_strings.Builder{},
		&shared_strings.Reader{},
		&shared_strings.Replacer{},
	} {
		value_type := reflect.TypeOf(value)
		if value_type.NumMethod() != 0 {
			t.Fatalf(
				"%s has %d exported methods, want none",
				value_type,
				value_type.NumMethod(),
			)
		}
	}
}

// Test_Reader verifies sequential, random, byte, character, seek, reset, and writer reads.
func Test_Reader(t *testing.T) {
	t.Parallel()
	reader := shared_strings.New_Reader("a☺b")
	stream := shared_strings.Reader_To_Stream(reader)
	verify_reader_sequential(t, reader, stream)
	verify_reader_random_and_reset(t, reader, stream)
}

// Test_Replacer verifies rule priority, empty matches, output, and writer behavior.
func Test_Replacer(t *testing.T) {
	t.Parallel()
	replacer := shared_strings.New_Replacer(shared_strings.Replacement_Pairs{
		{"ab", "x"}, {"a", "y"}, {"", "."},
	})
	want := "xxy."
	if got := shared_strings.Replacer_Replace(replacer, "ababa"); string(got) != want {
		t.Fatalf("Replacer_Replace = %q, want %q", got, want)
	}
	output := make([]byte, len(want))
	output_state := shared_io.Stream_Memory{Memory: output}
	count, err := shared_strings.Replacer_Write_Text(
		replacer, shared_io.Memory_To_Stream(&output_state), "ababa",
	)
	if int(count) != len(want) {
		t.Fatalf("Replacer_Write_Text count = %d", count)
	}
	if err != nil {
		t.Fatalf("Replacer_Write_Text returned %v", err)
	}
	if string(output) != want {
		t.Fatalf("Replacer_Write_Text content = %q, want %q", output, want)
	}
}

// Test_Size_Limits verifies exact maximum results and rejects each larger owned result.
func Test_Size_Limits(t *testing.T) {
	t.Parallel()
	exercise_invariant_boundaries()
	maximum := text(fixture_repeat("x", shared_strings.TEXT_SIZE_MAXIMUM))
	if shared_strings.Repeat("x", shared_strings.TEXT_SIZE_MAXIMUM) != maximum {
		t.Fatal("Repeat must reach the text-size maximum")
	}
	assert_panic(t, func() { shared_strings.Repeat("xx", shared_strings.TEXT_SIZE_MAXIMUM) })
	assert_panic(t, func() { shared_strings.Join(shared_strings.Texts{maximum, "x"}, "") })
	assert_panic(t, func() { shared_strings.Replace_All(maximum, "x", "xx") })
	var builder shared_strings.Builder
	stream := shared_strings.Builder_To_Stream(&builder)
	shared_io.Write_String(stream, string(maximum))
	assert_panic(t, func() { shared_io.Write_Byte(stream, 'x') })
	replacer := shared_strings.New_Replacer(shared_strings.Replacement_Pairs{{"x", "xx"}})
	assert_panic(t, func() { shared_strings.Replacer_Replace(replacer, maximum) })
}

// Test_Domain_Errors verifies that invalid bounded arguments fail at their boundaries.
func Test_Domain_Errors(t *testing.T) {
	t.Parallel()
	assert_panic(t, func() { shared_strings.Split_N("", "", -2) })
	assert_panic(t, func() { shared_strings.Repeat("", -1) })
	assert_panic(t, func() { shared_strings.Replace("", "", "", -2) })
	assert_panic(t, func() { shared_strings.Builder_Grow(&shared_strings.Builder{}, -1) })
	oversized_builder := shared_strings.Builder{
		Content: make(
			shared_strings.Slice,
			0,
			shared_strings.BUILDER_CAPACITY_MAXIMUM+1,
		),
	}
	assert_panic(t, func() { shared_strings.Builder_To_Stream(&oversized_builder) })
	too_many_pairs := make(
		shared_strings.Replacement_Pairs,
		shared_strings.REPLACEMENT_PAIRS_COUNT_MAXIMUM+1,
	)
	assert_panic(t, func() { shared_strings.New_Replacer(too_many_pairs) })
}

func verify_transform_replacement(t *testing.T) {
	t.Helper()
	for _, one := range []struct {
		Count shared_strings.Replacement_Count
		Want  string
	}{
		{Count: -1, Want: "xybcxybc"},
		{Count: 0, Want: "abcabc"},
		{Count: 1, Want: "xybcabc"},
		{Count: 2, Want: "xybcxybc"},
	} {
		got := shared_strings.Replace("abcabc", "a", "xy", one.Count)
		if string(got) != one.Want {
			t.Fatalf("Replace count %d = %q, want %q", one.Count, got, one.Want)
		}
	}
	if got := shared_strings.Replace_All("abcabc", "a", "xy"); got != "xybcxybc" {
		t.Fatalf("Replace_All = %q, want %q", got, "xybcxybc")
	}
}

func verify_trim_forms(t *testing.T) {
	t.Helper()
	space := func(character rune) (matches bool) {
		return bool(ucd.Is_Space(ucd.Character(character)))
	}
	for _, one := range []struct {
		Source      string
		Cut         string
		Cut_Left    string
		Cut_Right   string
		Whitespace  string
		Space_Left  string
		Space_Right string
	}{
		{
			Source: "", Cut: "", Cut_Left: "", Cut_Right: "", Whitespace: "",
			Space_Left: "", Space_Right: "",
		},
		{
			Source: "  abc  ", Cut: "abc", Cut_Left: "abc  ", Cut_Right: "  abc",
			Whitespace: "abc", Space_Left: "abc  ", Space_Right: "  abc",
		},
		{
			Source: "xxabcxx", Cut: "abc", Cut_Left: "abcxx", Cut_Right: "xxabc",
			Whitespace: "xxabcxx", Space_Left: "xxabcxx", Space_Right: "xxabcxx",
		},
		{
			Source: "\u2000abc\u2000", Cut: "\u2000abc\u2000",
			Cut_Left: "\u2000abc\u2000", Cut_Right: "\u2000abc\u2000",
			Whitespace: "abc",
			Space_Left: "abc\u2000", Space_Right: "\u2000abc",
		},
	} {
		if shared_strings.Trim(text(one.Source), " x") != text(one.Cut) {
			t.Fatalf("Trim(%q) did not return %q", one.Source, one.Cut)
		}
		if shared_strings.Trim_Left(text(one.Source), " x") != text(one.Cut_Left) {
			t.Fatalf("Trim_Left(%q) did not return %q", one.Source, one.Cut_Left)
		}
		if shared_strings.Trim_Right(text(one.Source), " x") != text(one.Cut_Right) {
			t.Fatalf("Trim_Right(%q) did not return %q", one.Source, one.Cut_Right)
		}
		if shared_strings.Trim_Space(text(one.Source)) != text(one.Whitespace) {
			t.Fatalf("Trim_Space(%q) did not return %q", one.Source, one.Whitespace)
		}
		if shared_strings.Trim_Function(text(one.Source), space) != text(one.Whitespace) {
			t.Fatalf("Trim_Function(%q) did not return %q", one.Source, one.Whitespace)
		}
		if shared_strings.Trim_Left_Function(
			text(one.Source), space,
		) != text(one.Space_Left) {
			t.Fatalf(
				"Trim_Left_Function(%q) did not return %q",
				one.Source, one.Space_Left,
			)
		}
		if shared_strings.Trim_Right_Function(
			text(one.Source), space,
		) != text(one.Space_Right) {
			t.Fatalf(
				"Trim_Right_Function(%q) did not return %q",
				one.Source, one.Space_Right,
			)
		}
	}
}

func verify_reader_sequential(
	t *testing.T, reader *shared_strings.Reader, stream shared_io.Stream,
) {
	t.Helper()
	stream_size, size_error := shared_io.Size(stream)
	if stream_size != int64(len("a☺b")) {
		t.Fatalf("the Reader stream size = %d", stream_size)
	}
	if size_error != nil {
		t.Fatalf("the Reader stream size returned %v", size_error)
	}
	value, byte_error := shared_io.Read_Byte(stream)
	if value != 'a' {
		t.Fatalf("the Reader stream byte read = %q, want 'a'", value)
	}
	if byte_error != nil {
		t.Fatalf("the Reader stream byte read returned %v", byte_error)
	}
	character, character_size, character_error := shared_strings.Reader_Read_Character(reader)
	if character != '☺' {
		t.Fatalf("the Reader character = %U, want %U", character, '☺')
	}
	if character_size != 3 {
		t.Fatalf("the Reader character size = %d, want 3", character_size)
	}
	if character_error != nil {
		t.Fatalf("the Reader character read returned %v", character_error)
	}
	unread_character_error := shared_strings.Reader_Unread_Character(reader)
	if unread_character_error != nil {
		t.Fatalf("Reader_Unread_Character returned %v", unread_character_error)
	}
	second_character, second_size, second_error := shared_strings.Reader_Read_Character(reader)
	if second_character != '☺' {
		t.Fatal("Reader_Unread_Character must restore the last character")
	}
	if second_size != 3 {
		t.Fatal("Reader_Unread_Character must restore the character size")
	}
	if second_error != nil {
		t.Fatalf("the restored character read returned %v", second_error)
	}
	unread_byte_error := shared_strings.Reader_Unread_Byte(reader)
	if unread_byte_error != nil {
		t.Fatalf("Reader_Unread_Byte returned %v", unread_byte_error)
	}
}

func verify_reader_random_and_reset(
	t *testing.T, reader *shared_strings.Reader, stream shared_io.Stream,
) {
	t.Helper()
	position, seek_error := shared_io.Seek(stream, 0, shared_io.SEEK_FROM_START)
	if position != 0 {
		t.Fatalf("the Reader stream seek position = %d, want 0", position)
	}
	if seek_error != nil {
		t.Fatalf("the Reader stream seek returned %v", seek_error)
	}
	destination := make(shared_strings.Slice, 3)
	read_count, read_error := shared_io.Read_At(stream, destination, 1)
	if read_count != 3 {
		t.Fatalf("the Reader read-at count = %d, want 3", read_count)
	}
	if read_error != nil {
		t.Fatalf("the Reader read-at returned %v", read_error)
	}
	if string(destination) != "☺" {
		t.Fatalf("the Reader read-at content = %q", destination)
	}
	output := make([]byte, len("a☺b"))
	output_state := shared_io.Stream_Memory{Memory: output}
	written, write_error := shared_strings.Reader_Write_To(
		reader, shared_io.Memory_To_Stream(&output_state),
	)
	if int64(written) != int64(len("a☺b")) {
		t.Fatalf("Reader_Write_To count = %d", written)
	}
	if write_error != nil {
		t.Fatalf("Reader_Write_To returned %v", write_error)
	}
	if string(output) != "a☺b" {
		t.Fatalf("Reader_Write_To content = %q", output)
	}
	shared_strings.Reader_Reset(reader, "xy")
	if shared_strings.Reader_Unread_Size(reader) != 2 {
		t.Fatal("Reader_Reset must restore the first position")
	}
	read := make([]byte, 2)
	stream = shared_strings.Reader_To_Stream(reader)
	reset_count, reset_error := shared_io.Read(stream, read)
	if reset_count != 2 {
		t.Fatalf("the reset Reader count = %d, want 2", reset_count)
	}
	if reset_error != nil {
		t.Fatalf("the reset Reader returned %v", reset_error)
	}
	if string(read) != "xy" {
		t.Fatalf("the reset Reader content = %q", read)
	}
}

func verify_byte_or_non_ascii_search(t *testing.T) {
	t.Helper()
	if shared_strings.Index_Byte_Or_Non_ASCII("aaaaaaaa", "\"\\\n") !=
		shared_strings.INDEX_ABSENT {
		t.Fatal("Index_Byte_Or_Non_ASCII must prove a uniform byte absent")
	}
	if shared_strings.Index_Byte_Or_Non_ASCII("abcdefgh", "\"\\\n") !=
		shared_strings.INDEX_ABSENT {
		t.Fatal("Index_Byte_Or_Non_ASCII must report an absent selected byte")
	}
	if shared_strings.Index_Byte_Or_Non_ASCII("abc\\def", "\"\\\n") != 3 {
		t.Fatal("Index_Byte_Or_Non_ASCII must find a selected byte")
	}
	if shared_strings.Index_Byte_Or_Non_ASCII("abcdef☺", "\"\\\n") != 6 {
		t.Fatal("Index_Byte_Or_Non_ASCII must find a non-ASCII byte")
	}
}

func exercise_invariant_boundaries() {
	values := boundary_texts()
	exercise_comparison_boundaries(values)
	exercise_search_boundaries(values)
	exercise_split_boundaries(values)
	exercise_transform_boundaries(values)
	exercise_trim_boundaries(values)
	exercise_replace_boundaries(values)
	exercise_cut_boundaries(values)
	exercise_iteration_boundaries(values)
	exercise_builder_boundaries(values)
	exercise_reader_boundaries(values)
	exercise_replacer_boundaries(values)
}

func boundary_texts() (values []shared_strings.Text) {
	return []shared_strings.Text{
		"",
		"a",
		"aa",
		text(fixture_repeat("a", shared_strings.TEXT_SIZE_MAXIMUM)),
	}
}

func exercise_comparison_boundaries(values []shared_strings.Text) {
	for _, value := range values {
		shared_strings.Compare(value, value)
		shared_strings.Equal_Fold(value, value)
		shared_strings.Has_Prefix(value, value)
		shared_strings.Has_Suffix(value, value)
		shared_strings.Clone(value)
	}
	shared_strings.Compare("a", "b")
	shared_strings.Compare("b", "a")
	shared_strings.Equal_Fold("a", "b")
	shared_strings.Has_Prefix("a", "b")
	shared_strings.Has_Suffix("a", "b")
}

func exercise_search_boundaries(values []shared_strings.Text) {
	full := values[len(values)-1]
	last := text(
		fixture_repeat("a", shared_strings.TEXT_SIZE_MAXIMUM-1) + "z",
	)
	for _, value := range values {
		shared_strings.Count(value, value)
		shared_strings.Contains(value, value)
		shared_strings.Index(value, value)
		shared_strings.Last_Index(value, value)
	}
	shared_strings.Count("", "z")
	shared_strings.Count("a", "a")
	shared_strings.Count("aa", "a")
	shared_strings.Count(full, "")
	index_sources := []shared_strings.Text{"", "z", "az", "aaz", last}
	for _, source := range index_sources {
		shared_strings.Index(source, "z")
		shared_strings.Last_Index(source, "z")
		shared_strings.Index_Byte(source, 'z')
		shared_strings.Last_Index_Byte(source, 'z')
		shared_strings.Index_Rune(source, 'z')
		shared_strings.Index_Any(source, "z")
		shared_strings.Last_Index_Any(source, "z")
	}
	exercise_byte_or_non_ascii_boundaries(last)
	shared_strings.Last_Index(full, "")
	for _, value := range []shared_strings.Byte{0, 1, 2, 255} {
		shared_strings.Index_Byte("", value)
		shared_strings.Last_Index_Byte("", value)
	}
	characters := []shared_strings.Character{
		shared_strings.Character(shared_strings.CHARACTER_MINIMUM),
		-1,
		0,
		1,
		2,
		shared_strings.Character(shared_strings.CHARACTER_MAXIMUM),
	}
	for _, character := range characters {
		shared_strings.Contains_Rune("", character)
		shared_strings.Index_Rune("", character)
	}
	for _, value := range values {
		shared_strings.Contains_Rune(value, 'z')
	}
	shared_strings.Contains_Rune("z", 'z')
	for index, character_set := range values {
		source := values[index]
		shared_strings.Contains_Any(source, character_set)
		shared_strings.Index_Any(source, character_set)
		shared_strings.Last_Index_Any(source, character_set)
	}
	match_z := func(character rune) (matches bool) { return character == 'z' }
	never := func(character rune) (matches bool) { return false }
	for _, source := range values {
		shared_strings.Contains_Function(source, never)
		shared_strings.Index_Function(source, never)
		shared_strings.Last_Index_Function(source, never)
	}
	for _, source := range index_sources[1:] {
		shared_strings.Contains_Function(source, match_z)
		shared_strings.Index_Function(source, match_z)
		shared_strings.Last_Index_Function(source, match_z)
	}
}

func exercise_byte_or_non_ascii_boundaries(last shared_strings.Text) {
	shared_strings.Index_Byte_Or_Non_ASCII("", "")
	shared_strings.Index_Byte_Or_Non_ASCII("z", "z")
	shared_strings.Index_Byte_Or_Non_ASCII("az", "xz")
	shared_strings.Index_Byte_Or_Non_ASCII("aaz", "z")
	shared_strings.Index_Byte_Or_Non_ASCII(last, "z")
	shared_strings.Index_Byte_Or_Non_ASCII(
		"a", text("a"+fixture_repeat("x", shared_strings.TEXT_SIZE_MAXIMUM-1)),
	)
}

func exercise_split_boundaries(values []shared_strings.Text) {
	space := func(character rune) (matches bool) {
		return bool(ucd.Is_Space(ucd.Character(character)))
	}
	full := values[len(values)-1]
	separators := text(fixture_repeat("x", shared_strings.TEXT_SIZE_MAXIMUM))
	for index, value := range values {
		other := values[len(values)-1-index]
		shared_strings.Split(value, other)
		shared_strings.Split_After(value, other)
		shared_strings.Split_N(value, other, shared_strings.Limit(index))
		shared_strings.Split_After_N(value, other, shared_strings.Limit(index))
	}
	for _, limit := range []shared_strings.Limit{
		-1, 0, 1, 2, shared_strings.LIMIT_MAXIMUM,
	} {
		shared_strings.Split_N(separators, "x", limit)
		shared_strings.Split_After_N(separators, "x", limit)
	}
	shared_strings.Split(full, "")
	shared_strings.Split(full, "a")
	shared_strings.Split_After(separators, "x")
	shared_strings.Split("a", "a")
	shared_strings.Split_After("a", "a")
	full_fields := text(fixture_repeat("a ", shared_strings.FIELDS_COUNT_MAXIMUM))
	for _, value := range values {
		shared_strings.Fields(value)
		shared_strings.Fields_Function(value, space)
	}
	shared_strings.Fields("a b")
	shared_strings.Fields_Function("a b", space)
	shared_strings.Fields(full_fields)
	shared_strings.Fields_Function(full_fields, space)
	parts_values := []shared_strings.Texts{
		nil,
		{""},
		{"", ""},
		make(shared_strings.Texts, shared_strings.TEXTS_COUNT_MAXIMUM),
	}
	for _, parts := range parts_values {
		shared_strings.Join(parts, "")
	}
	for _, value := range values {
		shared_strings.Join(shared_strings.Texts{""}, value)
		shared_strings.Join(shared_strings.Texts{value}, "")
	}
}

func exercise_transform_boundaries(values []shared_strings.Text) {
	identity := func(character rune) (mapped rune) { return character }
	special := turkish_case()
	for _, size := range []int{0, 1, 2, len(special)} {
		shared_strings.To_Upper_Special(special[:size], "")
		shared_strings.To_Lower_Special(special[:size], "")
		shared_strings.To_Title_Special(special[:size], "")
	}
	for _, value := range values {
		shared_strings.Map(identity, value)
		shared_strings.To_Upper(value)
		shared_strings.To_Lower(value)
		shared_strings.To_Title(value)
		shared_strings.To_Upper_Special(special, value)
		shared_strings.To_Lower_Special(special, value)
		shared_strings.To_Title_Special(special, value)
		shared_strings.To_Valid_UTF8(value, value)
		shared_strings.Title(value)
	}
	for _, count := range []shared_strings.Repeat_Count{
		0, 1, 2, shared_strings.REPEAT_COUNT_MAXIMUM,
	} {
		shared_strings.Repeat("a", count)
	}
	shared_strings.Repeat("", 0)
	shared_strings.Repeat(values[len(values)-1], 1)
}

func exercise_trim_boundaries(values []shared_strings.Text) {
	never := func(character rune) (matches bool) { return false }
	for index, value := range values {
		cutset := values[index]
		shared_strings.Trim(value, cutset)
		shared_strings.Trim_Left(value, cutset)
		shared_strings.Trim_Right(value, cutset)
		shared_strings.Trim(value, "z")
		shared_strings.Trim_Left(value, "z")
		shared_strings.Trim_Right(value, "z")
		shared_strings.Trim_Function(value, never)
		shared_strings.Trim_Left_Function(value, never)
		shared_strings.Trim_Right_Function(value, never)
		shared_strings.Trim_Space(value)
		shared_strings.Trim_Prefix(value, "")
		shared_strings.Trim_Suffix(value, "")
		shared_strings.Trim_Prefix("", value)
		shared_strings.Trim_Suffix("", value)
	}
}

func exercise_replace_boundaries(values []shared_strings.Text) {
	empty := values[0]
	full := values[len(values)-1]
	for _, value := range values {
		shared_strings.Replace(value, full, empty, 0)
		shared_strings.Replace(empty, value, empty, 1)
		shared_strings.Replace(empty, full, value, 2)
		shared_strings.Replace_All(value, full, empty)
		shared_strings.Replace_All(empty, value, empty)
		shared_strings.Replace_All(empty, full, value)
		shared_strings.Replace_All(value, "z", empty)
	}
	for _, count := range []shared_strings.Replacement_Count{
		-1, 0, 1, 2, shared_strings.REPLACEMENT_COUNT_MAXIMUM,
	} {
		shared_strings.Replace(empty, full, empty, count)
	}
}

func exercise_cut_boundaries(values []shared_strings.Text) {
	for _, value := range values {
		shared_strings.Cut(value, value)
		shared_strings.Cut(value, "z")
		shared_strings.Cut(value, "")
		shared_strings.Cut_Prefix(value, "")
		shared_strings.Cut_Suffix(value, "")
		shared_strings.Cut_Prefix("", value)
		shared_strings.Cut_Suffix("", value)
	}
	shared_strings.Cut("za", "z")
	shared_strings.Cut("zaa", "z")
}

func exercise_iteration_boundaries(values []shared_strings.Text) {
	space := func(character rune) (matches bool) {
		return bool(ucd.Is_Space(ucd.Character(character)))
	}
	for index, value := range values {
		other := values[len(values)-1-index]
		collect(shared_strings.Lines(value))
		collect(shared_strings.Split_Sequence(value, other))
		collect(shared_strings.Split_After_Sequence(value, other))
		collect(shared_strings.Fields_Sequence(value))
		collect(shared_strings.Fields_Function_Sequence(value, space))
	}
}

func exercise_builder_boundaries(values []shared_strings.Text) {
	for _, value := range values {
		content := make(shared_strings.Slice, len(value), len(value))
		copy(content, value)
		prefilled := &shared_strings.Builder{
			Content: content,
		}
		shared_strings.Builder_To_Stream(prefilled)
		builder := &shared_strings.Builder{}
		stream := shared_strings.Builder_To_Stream(builder)
		shared_io.Write_String(stream, string(value))
		shared_io.Query(stream)
		shared_strings.Builder_Text(builder)
		shared_strings.Builder_Size(builder)
		shared_strings.Builder_Capacity(builder)
		shared_strings.Builder_Grow(builder, 0)
		shared_strings.Builder_Reset(builder)
	}
	for _, count := range []shared_strings.Growth_Count{
		0, 1, 2, shared_strings.GROWTH_COUNT_MAXIMUM,
	} {
		shared_strings.Builder_Grow(&shared_strings.Builder{}, count)
	}
}

func exercise_reader_boundaries(values []shared_strings.Text) {
	for _, value := range values {
		shared_strings.New_Reader(value)
	}
	exercise_reader_state_boundaries(func(reader *shared_strings.Reader) {
		shared_strings.Reader_To_Stream(reader)
	})
	exercise_reader_state_boundaries(func(reader *shared_strings.Reader) {
		shared_io.Query(shared_strings.Reader_To_Stream(reader))
	})
	exercise_reader_state_boundaries(func(reader *shared_strings.Reader) {
		shared_strings.Reader_Unread_Size(reader)
	})
	exercise_reader_state_boundaries(func(reader *shared_strings.Reader) {
		shared_strings.Reader_Read_Byte(reader)
	})
	exercise_reader_state_boundaries(func(reader *shared_strings.Reader) {
		shared_strings.Reader_Unread_Byte(reader)
	})
	exercise_reader_state_boundaries(func(reader *shared_strings.Reader) {
		shared_strings.Reader_Read_Character(reader)
	})
	exercise_reader_state_boundaries(func(reader *shared_strings.Reader) {
		shared_strings.Reader_Unread_Character(reader)
	})
	exercise_reader_state_boundaries(func(reader *shared_strings.Reader) {
		shared_strings.Reader_Write_To(reader, discard_stream())
	})
	exercise_reader_state_boundaries(func(reader *shared_strings.Reader) {
		shared_strings.Reader_Reset(reader, "")
	})
	for _, value := range values {
		shared_strings.Reader_Reset(shared_strings.New_Reader(""), value)
	}
	for _, value := range []byte{0, 1, 2, 255} {
		reader := shared_strings.New_Reader(text(string([]byte{value})))
		shared_strings.Reader_Read_Byte(reader)
	}
	for _, character := range []rune{0, 1, 2, 0x80, rune(utf8.RUNE_MAX)} {
		shared_strings.Reader_Read_Character(
			shared_strings.New_Reader(text(string(character))),
		)
	}
	shared_strings.Reader_Read_Character(shared_strings.New_Reader(""))
	reader := shared_strings.New_Reader(values[len(values)-1])
	stream := shared_strings.Reader_To_Stream(reader)
	for _, size := range []int{0, 1, 2, shared_strings.SLICE_SIZE_MAXIMUM} {
		shared_io.Read_At(stream, make([]byte, size), 0)
	}
	exercise_reader_stream_offsets(values)
}

func exercise_reader_stream_offsets(values []shared_strings.Text) {
	offsets := []int64{
		shared_strings.STREAM_OFFSET_MINIMUM,
		-1,
		0,
		1,
		2,
		shared_strings.STREAM_OFFSET_MAXIMUM,
	}
	for _, value := range values {
		stream := shared_strings.Reader_To_Stream(shared_strings.New_Reader(value))
		shared_io.Read_At(stream, make([]byte, 2), 0)
		shared_io.Seek(stream, 0, shared_io.SEEK_FROM_START)
	}
	maximum := values[len(values)-1]
	reader := shared_strings.New_Reader(maximum)
	stream := shared_strings.Reader_To_Stream(reader)
	for _, offset := range offsets {
		shared_io.Read_At(stream, make([]byte, 2), offset)
		shared_io.Seek(stream, offset, shared_io.SEEK_FROM_START)
		shared_strings.Reader_Unread_Size(reader)
	}
}

func exercise_reader_state_boundaries(operation func(*shared_strings.Reader)) {
	for _, reader := range reader_boundary_states() {
		operation(reader)
	}
}

func reader_boundary_states() (readers []*shared_strings.Reader) {
	maximum := text(fixture_repeat("a", shared_strings.TEXT_SIZE_MAXIMUM))
	readers = append(readers, shared_strings.New_Reader(""))
	readers = append(readers, shared_strings.New_Reader(maximum))
	for _, position := range []int64{0, 1, 2, shared_strings.TEXT_SIZE_MAXIMUM - 1} {
		reader := shared_strings.New_Reader(maximum)
		stream := shared_strings.Reader_To_Stream(reader)
		shared_io.Seek(stream, position, shared_io.SEEK_FROM_START)
		shared_strings.Reader_Read_Character(reader)
		readers = append(readers, reader)
	}
	reader := shared_strings.New_Reader(maximum)
	stream := shared_strings.Reader_To_Stream(reader)
	shared_io.Seek(stream, shared_strings.TEXT_SIZE_MAXIMUM, shared_io.SEEK_FROM_START)
	readers = append(readers, reader)
	readers = append(readers, shared_strings.New_Reader("a"))
	readers = append(readers, shared_strings.New_Reader("aa"))
	return readers
}

func exercise_replacer_boundaries(values []shared_strings.Text) {
	for _, value := range values {
		shared_strings.New_Replacer(shared_strings.Replacement_Pairs{{value, ""}})
		replacer := shared_strings.New_Replacer(shared_strings.Replacement_Pairs{{"z", ""}})
		shared_strings.Replacer_Replace(replacer, value)
		shared_strings.Replacer_Write_Text(replacer, discard_stream(), value)
	}
	maximum := values[len(values)-1]
	maximum_pairs := make(
		shared_strings.Replacement_Pairs,
		shared_strings.REPLACEMENT_PAIRS_COUNT_MAXIMUM,
	)
	pair_sets := []shared_strings.Replacement_Pairs{
		nil,
		{{maximum, maximum}},
		{{"a", "b"}, {"c", "d"}},
		maximum_pairs,
	}
	for _, pairs := range pair_sets {
		replacer := shared_strings.New_Replacer(pairs)
		shared_strings.Replacer_Replace(replacer, "")
		shared_strings.Replacer_Write_Text(replacer, discard_stream(), "")
	}
}

func discard_stream() (stream shared_io.Stream) {
	state := &shared_io.Stream_Discard{}
	return shared_io.Discard_To_Stream(state)
}

func fixture_repeat(fragment string, count int) (value string) {
	size := len(fragment) * count
	content := make([]byte, size)
	for offset := 0; offset < size; offset += len(fragment) {
		copy(content[offset:], fragment)
	}
	return string(content)
}

func text(value string) (converted shared_strings.Text) {
	return shared_strings.Text(value)
}

func collect(
	sequence func(func(shared_strings.Text) (continue_iteration bool)),
) (values shared_strings.Texts) {
	for value := range sequence {
		values = append(values, value)
	}
	return values
}

func assert_texts[Collection ~[]shared_strings.Text](
	t *testing.T, got Collection, want []string,
) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("text count = %d, want %d: %q", len(got), len(want), got)
	}
	for index := range want {
		if string(got[index]) != want[index] {
			t.Fatalf("text %d = %q, want %q", index, got[index], want[index])
		}
	}
}

func assert_panic(t *testing.T, operation func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("the operation must panic")
		}
	}()
	operation()
}

func turkish_case() (special ucd.Special_Case) {
	return ucd.Special_Case(ucd.Turkish_Case())
}
