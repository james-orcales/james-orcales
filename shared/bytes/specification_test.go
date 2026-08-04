package bytes_test

import (
	standard_bytes "bytes"
	"go/ast"
	"go/parser"
	"go/token"
	standard_strings "strings"
	"testing"
	standard_unicode "unicode"
	standard_utf8 "unicode/utf8"

	shared_bytes "local/james-orcales/shared/bytes"
	"local/james-orcales/shared/io"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/unicode/ucd"
)

// Test_Constant_Facts verifies that equal domain facts have one source.
func Test_Constant_Facts(t *testing.T) {
	t.Parallel()
	file, err := parser.ParseFile(token.NewFileSet(), "bytes.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	expressions := constant_expressions(file)
	assert_constant_sources(t, expressions)
	assert_bit_constant_sources(t, expressions, bit_constant_counts(file))
	assert_package_imports(t, file)
}

// Test_Comparison verifies equality, lexical order, nil equivalence, and Unicode folding.
func Test_Comparison(t *testing.T) {
	t.Parallel()
	comparison_cases := []struct {
		Left  string
		Right string
	}{
		{Left: "", Right: ""},
		{Left: "a", Right: "a"},
		{Left: "a", Right: "b"},
		{Left: "b", Right: "a"},
		{Left: "abc", Right: "ab"},
		{Left: standard_strings.Repeat("x", shared_bytes.SLICE_SIZE_MAXIMUM), Right: ""},
	}
	for _, one := range comparison_cases {
		left := shared_bytes.Slice(one.Left)
		right := shared_bytes.Slice(one.Right)
		if bool(shared_bytes.Equal(left, right)) != standard_bytes.Equal(
			[]byte(one.Left), []byte(one.Right),
		) {
			t.Fatalf(
				"Equal(%q, %q) differs from the standard library",
				one.Left,
				one.Right,
			)
		}
		if int(shared_bytes.Compare(left, right)) != standard_bytes.Compare(
			[]byte(one.Left), []byte(one.Right),
		) {
			t.Fatalf(
				"Compare(%q, %q) differs from the standard library",
				one.Left,
				one.Right,
			)
		}
	}
	if !shared_bytes.Equal(nil, shared_bytes.Slice{}) {
		t.Fatal("a nil Slice must equal an empty Slice")
	}
	for _, one := range comparison_cases {
		got := shared_bytes.Equal_Fold(
			shared_bytes.Slice(one.Left), shared_bytes.Slice(one.Right),
		)
		want := standard_bytes.EqualFold([]byte(one.Left), []byte(one.Right))
		if bool(got) != want {
			t.Fatalf("Equal_Fold(%q, %q) = %t, want %t", one.Left, one.Right, got, want)
		}
	}
	if !shared_bytes.Equal_Fold(slice("Go"), slice("gO")) {
		t.Fatal("Equal_Fold must apply Unicode simple folding")
	}
}

// Test_Search verifies every search form, including empty separators and invalid UTF-8.
func Test_Search(t *testing.T) {
	t.Parallel()
	full := shared_bytes.Slice(
		standard_strings.Repeat("a", shared_bytes.SLICE_SIZE_MAXIMUM-1) + "z",
	)
	search_slice_cases(t, full)
	if shared_bytes.Count(full, slice("")) != shared_bytes.Count_Value(
		shared_bytes.SLICE_SIZE_MAXIMUM+1,
	) {
		t.Fatal("an empty separator must occur before and after every byte character")
	}
	if shared_bytes.Index_Byte(full, 'z') != shared_bytes.Index_Value(
		shared_bytes.SLICE_SIZE_MAXIMUM-1,
	) {
		t.Fatal("Index_Byte must reach the final byte")
	}
	if shared_bytes.Last_Index_Byte(full, 'x') != shared_bytes.INDEX_ABSENT {
		t.Fatal("Last_Index_Byte must report an absent byte")
	}
	if shared_bytes.Index_Rune(slice("a☺b"), '☺') != 1 {
		t.Fatal("Index_Rune must return a byte index")
	}
	if shared_bytes.Index_Rune(slice("\xff"), standard_utf8.RuneError) != 0 {
		t.Fatal("Index_Rune must find invalid UTF-8 through RuneError")
	}
	if shared_bytes.Index_Any(full, "z") != shared_bytes.Index_Value(
		shared_bytes.SLICE_SIZE_MAXIMUM-1,
	) {
		t.Fatal("Index_Any must reach the final byte")
	}
	if shared_bytes.Last_Index_Any(slice("abcabc"), "ca") != 5 {
		t.Fatal("Last_Index_Any must find the last set member")
	}
	space_match := func(character rune) (matches bool) {
		return standard_unicode.IsSpace(character)
	}
	if shared_bytes.Index_Function(slice("a b"), space_match) != 1 {
		t.Fatal("Index_Function must find the first predicate match")
	}
	if shared_bytes.Last_Index_Function(slice("a b c"), space_match) != 3 {
		t.Fatal("Last_Index_Function must find the last predicate match")
	}
	if !shared_bytes.Contains_Any(slice("abc"), "zx") {
		if shared_bytes.Contains_Any(slice("abc"), "x") {
			t.Fatal("Contains_Any must reject an absent set")
		}
	}
	if !shared_bytes.Contains_Rune(slice("a☺"), '☺') {
		t.Fatal("Contains_Rune must find a Unicode character")
	}
	if !shared_bytes.Contains_Function(slice("a b"), space_match) {
		t.Fatal("Contains_Function must find a predicate match")
	}
}

// Test_Split_And_Join verifies separator placement, N limits, fields, aliasing, and joins.
func Test_Split_And_Join(t *testing.T) {
	t.Parallel()
	for _, source := range []shared_bytes.Slice{
		slice(""), slice("a,b,c"), slice("☺,☻"), slice("\xff"),
	} {
		for _, separator := range []shared_bytes.Slice{slice(""), slice(","), slice("x")} {
			assert_slices(
				t,
				shared_bytes.Split(source, separator),
				standard_bytes.Split(source, separator),
			)
			assert_slices(
				t,
				shared_bytes.Split_After(source, separator),
				standard_bytes.SplitAfter(source, separator),
			)
			for _, limit := range []shared_bytes.Limit{-1, 0, 1, 2} {
				assert_slices(
					t,
					shared_bytes.Split_N(source, separator, limit),
					standard_bytes.SplitN(source, separator, int(limit)),
				)
				assert_slices(
					t,
					shared_bytes.Split_After_N(source, separator, limit),
					standard_bytes.SplitAfterN(source, separator, int(limit)),
				)
			}
		}
	}
	separators := shared_bytes.Slice(standard_strings.Repeat(
		"x", shared_bytes.SLICE_SIZE_MAXIMUM,
	))
	if len(shared_bytes.Split(separators, slice("x"))) != shared_bytes.SLICES_COUNT_MAXIMUM {
		t.Fatal("Split must reach the declared slice-count maximum")
	}
	for _, source := range []shared_bytes.Slice{
		slice(""), slice(" a\tb\n"), slice(" a b"), slice("abc"),
	} {
		assert_slices(t, shared_bytes.Fields(source), standard_bytes.Fields(source))
		field_match := func(character rune) (matches bool) { return character == 'a' }
		assert_slices(
			t,
			shared_bytes.Fields_Function(source, field_match),
			standard_bytes.FieldsFunc(source, field_match),
		)
	}
	full_fields := shared_bytes.Slice(standard_strings.Repeat(
		"a ", shared_bytes.SLICE_SIZE_MAXIMUM/2,
	))
	if len(shared_bytes.Fields(full_fields)) != shared_bytes.FIELDS_COUNT_MAXIMUM {
		t.Fatal("Fields must reach the declared field-count maximum")
	}
	parts := shared_bytes.Slices{shared_bytes.Slice("a"), shared_bytes.Slice("b"), nil}
	if got := shared_bytes.Join(parts, slice(",")); string(got) != "a,b," {
		t.Fatalf("Join = %q, want %q", got, "a,b,")
	}
	joined_maximum := shared_bytes.Slices{shared_bytes.Slice(standard_strings.Repeat(
		"x", shared_bytes.SLICE_SIZE_MAXIMUM,
	))}
	if len(shared_bytes.Join(joined_maximum, slice(""))) != shared_bytes.SLICE_SIZE_MAXIMUM {
		t.Fatal("Join must reach the Slice size maximum")
	}
	aliased := shared_bytes.Slice("a,b")
	parts = shared_bytes.Split(aliased, slice(","))
	parts[0][0] = 'z'
	if aliased[0] != 'z' {
		t.Fatal("Split results must alias their input")
	}
}

// Test_Transform verifies character mapping, repetition, case mapping, UTF-8 repair, and decoding.
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
	for _, source := range []shared_bytes.Slice{
		slice(""), slice("abx"), slice("Hello, 世界"),
		shared_bytes.Slice(standard_strings.Repeat("b", shared_bytes.SLICE_SIZE_MAXIMUM)),
	} {
		assert_slice(
			t, shared_bytes.Map(mapping, source), standard_bytes.Map(mapping, source),
		)
	}
	for _, one := range []struct {
		Source shared_bytes.Slice
		Count  shared_bytes.Repeat_Count
	}{
		{Source: slice(""), Count: 0},
		{Source: slice("ab"), Count: 3},
		{Source: slice("x"), Count: shared_bytes.SLICE_SIZE_MAXIMUM},
	} {
		assert_slice(
			t,
			shared_bytes.Repeat(one.Source, one.Count),
			standard_bytes.Repeat(one.Source, int(one.Count)),
		)
	}
	case_mapping_cases(t)
	invalid := shared_bytes.Slice("a\xff\xfeb")
	assert_slice(
		t,
		shared_bytes.To_Valid_UTF8(invalid, slice("?")),
		standard_bytes.ToValidUTF8(invalid, []byte("?")),
	)
	assert_slice(
		t,
		shared_bytes.To_Valid_UTF8(slice(""), slice("")),
		standard_bytes.ToValidUTF8(nil, nil),
	)
	got_runes := shared_bytes.Runes(slice("a☺\xff"))
	want_runes := standard_bytes.Runes([]byte("a☺\xff"))
	if len(got_runes) != len(want_runes) {
		t.Fatalf("Runes gives %d characters, want %d", len(got_runes), len(want_runes))
	}
	for index := range want_runes {
		if got_runes[index] != want_runes[index] {
			t.Fatalf(
				"Runes character %d = %U, want %U",
				index,
				got_runes[index],
				want_runes[index],
			)
		}
	}
}

// Test_Trim verifies cut sets, predicates, Unicode space, prefix and suffix aliasing.
func Test_Trim(t *testing.T) {
	t.Parallel()
	space_match := func(character rune) (matches bool) {
		return standard_unicode.IsSpace(character)
	}
	for _, source := range []shared_bytes.Slice{
		slice(""), slice("  abc  "), slice("xxabcxx"), slice(" abc "),
		shared_bytes.Slice(standard_strings.Repeat("x", shared_bytes.SLICE_SIZE_MAXIMUM)),
	} {
		for _, cutset := range []shared_bytes.Text{"", " x", " "} {
			assert_slice(
				t,
				shared_bytes.Trim(source, cutset),
				standard_bytes.Trim(source, string(cutset)),
			)
			assert_slice(
				t,
				shared_bytes.Trim_Left(source, cutset),
				standard_bytes.TrimLeft(source, string(cutset)),
			)
			assert_slice(
				t,
				shared_bytes.Trim_Right(source, cutset),
				standard_bytes.TrimRight(source, string(cutset)),
			)
		}
		assert_slice(t, shared_bytes.Trim_Space(source), standard_bytes.TrimSpace(source))
		assert_slice(
			t,
			shared_bytes.Trim_Function(source, space_match),
			standard_bytes.TrimFunc(source, space_match),
		)
		assert_slice(
			t,
			shared_bytes.Trim_Left_Function(source, space_match),
			standard_bytes.TrimLeftFunc(source, space_match),
		)
		assert_slice(
			t,
			shared_bytes.Trim_Right_Function(source, space_match),
			standard_bytes.TrimRightFunc(source, space_match),
		)
		assert_slice(
			t,
			shared_bytes.Trim_Prefix(source, slice("xx")),
			standard_bytes.TrimPrefix(source, []byte("xx")),
		)
		assert_slice(
			t,
			shared_bytes.Trim_Suffix(source, slice("xx")),
			standard_bytes.TrimSuffix(source, []byte("xx")),
		)
	}
	aliased := shared_bytes.Slice("xxabcxx")
	trimmed := shared_bytes.Trim(aliased, "x")
	trimmed[0] = 'z'
	if aliased[2] != 'z' {
		t.Fatal("Trim results must alias their input")
	}
}

// Test_Replace verifies bounded and unlimited replacement, empty matches, and output limits.
func Test_Replace(t *testing.T) {
	t.Parallel()
	for _, count := range []shared_bytes.Replacement_Count{-1, 0, 1, 2, 9} {
		for _, source := range []shared_bytes.Slice{
			slice(""), slice("abcabc"), slice("☺☺"),
		} {
			for _, old := range []shared_bytes.Slice{
				slice(""), slice("a"), slice("bc"), slice("x"),
			} {
				assert_slice(
					t,
					shared_bytes.Replace(source, old, slice("Z"), count),
					standard_bytes.Replace(
						source, old, []byte("Z"), int(count),
					),
				)
			}
		}
	}
	assert_slice(
		t,
		shared_bytes.Replace_All(slice("abcabc"), slice("a"), slice("z")),
		standard_bytes.ReplaceAll([]byte("abcabc"), []byte("a"), []byte("z")),
	)
	full := shared_bytes.Slice(standard_strings.Repeat("x", shared_bytes.SLICE_SIZE_MAXIMUM))
	if len(shared_bytes.Replace_All(
		full, slice("x"), slice("y"),
	)) != shared_bytes.SLICE_SIZE_MAXIMUM {
		t.Fatal("Replace_All must reach the Slice size maximum")
	}
}

// Test_Cut_And_Clone verifies found and absent cuts, edge cuts, aliasing, and nil cloning.
func Test_Cut_And_Clone(t *testing.T) {
	t.Parallel()
	for _, separator := range []shared_bytes.Slice{slice(""), slice(","), slice("x")} {
		before, after, found := shared_bytes.Cut(slice("a,b"), separator)
		want_before, want_after, want_found := standard_bytes.Cut([]byte("a,b"), separator)
		assert_slice(t, before, want_before)
		assert_slice(t, after, want_after)
		if bool(found) != want_found {
			t.Fatalf("Cut found = %t, want %t", found, want_found)
		}
	}
	after, found := shared_bytes.Cut_Prefix(slice("abc"), slice("a"))
	assert_slice(t, after, []byte("bc"))
	if !found {
		t.Fatal("Cut_Prefix must report a present prefix")
	}
	before, found := shared_bytes.Cut_Suffix(slice("abc"), slice("c"))
	assert_slice(t, before, []byte("ab"))
	if !found {
		t.Fatal("Cut_Suffix must report a present suffix")
	}
	if clone := shared_bytes.Clone(nil); clone != nil {
		t.Fatal("Clone(nil) must return nil")
	}
	full := shared_bytes.Slice(standard_strings.Repeat("x", shared_bytes.SLICE_SIZE_MAXIMUM))
	clone := shared_bytes.Clone(full)
	assert_slice(t, clone, full)
	clone[0] = 'z'
	if full[0] != 'x' {
		t.Fatal("Clone must not alias its input")
	}
	before, after, found = shared_bytes.Cut(full, slice("missing"))
	if !found {
		if len(before) != shared_bytes.SLICE_SIZE_MAXIMUM {
			t.Fatal("an absent Cut must return the full input and an empty tail")
		}
		if len(after) != 0 {
			t.Fatal("an absent Cut must return the full input and an empty tail")
		}
	}
}

// Test_Iteration verifies line, split, and field sequences against their standard forms.
func Test_Iteration(t *testing.T) {
	t.Parallel()
	for _, source := range []shared_bytes.Slice{
		slice(""), slice("a"), slice("a\nb\n"), slice("a\nb"),
	} {
		assert_slices(
			t,
			collect(shared_bytes.Lines(source)),
			collect_standard(standard_bytes.Lines(source)),
		)
		assert_slices(
			t,
			collect(shared_bytes.Split_Sequence(source, slice(","))),
			collect_standard(standard_bytes.SplitSeq(source, []byte(","))),
		)
		assert_slices(
			t,
			collect(shared_bytes.Split_After_Sequence(source, slice(","))),
			collect_standard(standard_bytes.SplitAfterSeq(source, []byte(","))),
		)
		assert_slices(
			t,
			collect(shared_bytes.Fields_Sequence(source)),
			collect_standard(standard_bytes.FieldsSeq(source)),
		)
		field_match := func(character rune) (matches bool) {
			return standard_unicode.IsSpace(character)
		}
		assert_slices(
			t,
			collect(shared_bytes.Fields_Function_Sequence(source, field_match)),
			collect_standard(standard_bytes.FieldsFuncSeq(source, field_match)),
		)
	}
	full := shared_bytes.Slice(standard_strings.Repeat("\n", shared_bytes.SLICE_SIZE_MAXIMUM))
	if len(collect(shared_bytes.Lines(full))) != shared_bytes.SLICE_SIZE_MAXIMUM {
		t.Fatal("Lines must yield every newline-terminated empty line")
	}
}

// Test_Buffer verifies reads, writes, growth, aliases, and reset behavior.
func Test_Buffer(t *testing.T) {
	buffer := shared_bytes.New_Buffer(slice("ab"))
	stream := shared_bytes.Buffer_To_Stream(buffer)
	if _, err := io.Write(stream, []byte("cd")); err != nil {
		t.Fatalf("Stream write failed: %s", err)
	}
	if shared_bytes.Buffer_String(buffer) != "abcd" {
		t.Fatalf(
			"Buffer content = %q, want %q",
			shared_bytes.Buffer_String(buffer),
			"abcd",
		)
	}
	if string(shared_bytes.Buffer_Next(buffer, 2)) != "ab" {
		t.Fatal("Buffer_Next must return and consume the requested prefix")
	}
	if string(shared_bytes.Buffer_Bytes(buffer)) != "cd" {
		t.Fatal("Buffer_Bytes must return the unread content")
	}
	shared_bytes.Buffer_Grow(buffer, 8)
	if shared_bytes.Buffer_Available(buffer) < 8 {
		t.Fatal("Buffer_Grow must reserve the requested capacity")
	}
	shared_bytes.Buffer_Reset(buffer)
	if shared_bytes.Buffer_Size(buffer) != 0 {
		t.Fatal("Buffer_Reset must remove all content")
	}
}

// Test_Reader verifies reads, rune handling, seeking, writing, and reset behavior.
func Test_Reader(t *testing.T) {
	reader := shared_bytes.New_Reader(slice("a☺b"))
	stream := shared_bytes.Reader_To_Stream(reader)
	character, size, err := io.Read_Rune(stream)
	if err != nil {
		t.Fatalf("ReadRune = %U, %d, %v; want %U, 1, nil", character, size, err, 'a')
	}
	if character != 'a' {
		t.Fatalf("ReadRune = %U, %d, %v; want %U, 1, nil", character, size, err, 'a')
	}
	if size != 1 {
		t.Fatalf("ReadRune = %U, %d, %v; want %U, 1, nil", character, size, err, 'a')
	}
	position, err := io.Seek(stream, -1, io.SEEK_FROM_END)
	if err != nil {
		t.Fatalf("Seek = %d, %v; want 4, nil", position, err)
	}
	if position != 4 {
		t.Fatalf("Seek = %d, %v; want 4, nil", position, err)
	}
	destination := make([]byte, 1)
	if _, err = io.Read(stream, destination); err != nil {
		t.Fatalf("Stream read failed: %s", err)
	}
	if string(destination) != "b" {
		t.Fatalf("Stream content = %q, want %q", destination, "b")
	}
	shared_bytes.Reader_Reset(reader, slice("xy"))
	reader_size, size_err := io.Size(stream)
	if size_err != nil {
		t.Fatalf("Stream size failed: %s", size_err)
	}
	if reader_size != 2 {
		t.Fatal("Reader_Reset must replace the source and reset the position")
	}
	if shared_bytes.Reader_Unread_Size(reader) != 2 {
		t.Fatal("Reader_Reset must replace the source and reset the position")
	}
}

// Test_Size_Limits verifies every collection domain at its minimum and maximum.
func Test_Size_Limits(t *testing.T) {
	t.Parallel()
	exercise_invariant_boundaries()
	full := shared_bytes.Slice(standard_strings.Repeat("x", shared_bytes.SLICE_SIZE_MAXIMUM))
	if len(shared_bytes.Clone(full)) != shared_bytes.SLICE_SIZE_MAXIMUM {
		t.Fatal("Clone must accept the Slice size maximum")
	}
	if len(shared_bytes.Clone(slice(""))) != 0 {
		t.Fatal("Clone must accept the Slice size minimum")
	}
	oversize := shared_bytes.Slice(standard_strings.Repeat(
		"x", shared_bytes.SLICE_SIZE_MAXIMUM+1,
	))
	if !panicked(func() { shared_bytes.Clone(oversize) }) {
		t.Fatal("a Slice above the size maximum must panic")
	}
	text_oversize := shared_bytes.Text(standard_strings.Repeat(
		"x", shared_bytes.TEXT_SIZE_MAXIMUM+1,
	))
	if !panicked(func() { shared_bytes.Trim(slice(""), text_oversize) }) {
		t.Fatal("Text above the size maximum must panic")
	}
}

// Test_Domain_Errors verifies bounded numeric inputs and standard state errors.
func Test_Domain_Errors(t *testing.T) {
	t.Parallel()
	if !panicked(func() {
		shared_bytes.Split_N(slice("a"), slice(","), shared_bytes.Limit(-2))
	}) {
		t.Fatal("a Limit below negative one must panic")
	}
	if !panicked(func() { shared_bytes.Repeat(slice("a"), shared_bytes.Repeat_Count(-1)) }) {
		t.Fatal("a negative Repeat_Count must panic")
	}
	if !panicked(func() {
		shared_bytes.Replace(
			slice("a"), slice("a"), slice("b"), shared_bytes.Replacement_Count(-2),
		)
	}) {
		t.Fatal("a Replacement_Count below negative one must panic")
	}
}

func assert_bit_constant_sources(
	t *testing.T,
	expressions map[string]ast.Expr,
	counts map[string]int,
) {
	t.Helper()
	expected := []string{
		"INTEGER_32_MINIMUM",
		"INTEGER_32_MAXIMUM",
		"INTEGER_64_MINIMUM",
		"INTEGER_64_MAXIMUM",
		"WORD_8_MINIMUM",
		"WORD_8_MAXIMUM",
	}
	removed := []string{
		"CHARACTER_MINIMUM",
		"CHARACTER_MAXIMUM",
		"BYTE_MINIMUM",
		"BYTE_MAXIMUM",
		"READER_OFFSET_MINIMUM",
		"READER_OFFSET_MAXIMUM",
	}
	for _, constant := range removed {
		if _, exists := expressions[constant]; exists {
			t.Errorf("%s must come from shared/math/bits", constant)
		}
	}
	for _, constant := range expected {
		if counts[constant] == 0 {
			t.Errorf("shared/math/bits.%s must define a bytes domain", constant)
		}
	}
}

func assert_constant_sources(t *testing.T, expressions map[string]ast.Expr) {
	t.Helper()
	expected := map[string]string{
		"TEXT_SIZE_MINIMUM":         "SLICE_SIZE_MINIMUM",
		"TEXT_SIZE_MAXIMUM":         "SLICE_SIZE_MAXIMUM",
		"SLICES_COUNT_MINIMUM":      "SLICE_SIZE_MINIMUM",
		"FIELDS_COUNT_MINIMUM":      "SLICES_COUNT_MINIMUM",
		"CHARACTERS_COUNT_MINIMUM":  "SLICE_SIZE_MINIMUM",
		"CHARACTERS_COUNT_MAXIMUM":  "SLICE_SIZE_MAXIMUM",
		"BOUNDARY_INDEX_MAXIMUM":    "SLICE_SIZE_MAXIMUM",
		"COUNT_VALUE_MINIMUM":       "SLICES_COUNT_MINIMUM",
		"COUNT_VALUE_MAXIMUM":       "SLICES_COUNT_MAXIMUM",
		"LIMIT_MAXIMUM":             "SLICE_SIZE_MAXIMUM",
		"REPEAT_COUNT_MINIMUM":      "SLICE_SIZE_MINIMUM",
		"REPEAT_COUNT_MAXIMUM":      "SLICE_SIZE_MAXIMUM",
		"REPLACEMENT_COUNT_MINIMUM": "LIMIT_MINIMUM",
		"REPLACEMENT_COUNT_MAXIMUM": "LIMIT_MAXIMUM",
		"ENCODED_SIZE_MAXIMUM":      "DECODED_SIZE_MAXIMUM",
		"READ_OPERATION_MINIMUM":    "READ_OPERATION_OTHER",
		"READ_OPERATION_MAXIMUM":    "DECODED_SIZE_MAXIMUM",
		"READ_OPERATION_ABSENT":     "DECODED_SIZE_MINIMUM",
		"GROWTH_COUNT_MINIMUM":      "SLICE_SIZE_MINIMUM",
		"GROWTH_COUNT_MAXIMUM":      "SLICE_SIZE_MAXIMUM",
		"READER_POSITION_MINIMUM":   "SLICE_SIZE_MINIMUM",
		"READER_POSITION_MAXIMUM":   "SLICE_SIZE_MAXIMUM",
		"SIZE_VALUE_MINIMUM":        "READER_POSITION_MINIMUM",
		"SIZE_VALUE_MAXIMUM":        "READER_POSITION_MAXIMUM",
		"READ_OPERATION_RUNE_1":     "ENCODED_SIZE_MINIMUM",
	}
	for constant, expected_source := range expected {
		expression, exists := expressions[constant]
		if !exists {
			t.Errorf("%s must use %s", constant, expected_source)
			continue
		}
		identifier, is_identifier := expression.(*ast.Ident)
		if !is_identifier {
			t.Errorf("%s must use %s", constant, expected_source)
			continue
		}
		if identifier.Name != expected_source {
			t.Errorf("%s must use %s", constant, expected_source)
		}
	}
}

func bit_constant_counts(file *ast.File) (counts map[string]int) {
	counts = make(map[string]int)
	ast.Inspect(file, func(node ast.Node) (proceed bool) {
		selector, is_selector := node.(*ast.SelectorExpr)
		if !is_selector {
			return true
		}
		package_name, is_identifier := selector.X.(*ast.Ident)
		if !is_identifier {
			return true
		}
		if package_name.Name != "bits" {
			return true
		}
		counts[selector.Sel.Name]++
		return true
	})
	return counts
}

func constant_expressions(file *ast.File) (expressions map[string]ast.Expr) {
	expressions = make(map[string]ast.Expr)
	for _, declaration := range file.Decls {
		general, is_general := declaration.(*ast.GenDecl)
		if !is_general {
			continue
		}
		if general.Tok != token.CONST {
			continue
		}
		for _, specification := range general.Specs {
			value, is_value := specification.(*ast.ValueSpec)
			if !is_value {
				continue
			}
			for index, name := range value.Names {
				if index < len(value.Values) {
					expressions[name.Name] = value.Values[index]
				}
			}
		}
	}
	return expressions
}

func assert_package_imports(t *testing.T, file *ast.File) {
	t.Helper()
	expected := map[string]bool{
		`"local/james-orcales/shared/io"`:           false,
		`"local/james-orcales/shared/strings"`:      false,
		`"local/james-orcales/shared/unicode/ucd"`:  false,
		`"local/james-orcales/shared/unicode/utf8"`: false,
	}
	for _, imported := range file.Imports {
		standard_replacement := map[string]string{
			`"strings"`:      "shared/strings",
			`"unicode"`:      "shared/unicode/ucd",
			`"unicode/utf8"`: "shared/unicode/utf8",
		}
		if replacement, standard := standard_replacement[imported.Path.Value]; standard {
			t.Errorf("shared/bytes must use %s", replacement)
		}
		if _, required := expected[imported.Path.Value]; !required {
			continue
		}
		expected[imported.Path.Value] = true
		if imported.Name != nil {
			t.Errorf("%s must use its package name", imported.Path.Value)
		}
	}
	for path, present := range expected {
		if !present {
			t.Errorf("shared/bytes must import %s", path)
		}
	}
}

func assert_slice(t *testing.T, got shared_bytes.Slice, want []byte) {
	t.Helper()
	if !standard_bytes.Equal(got, want) {
		t.Fatalf("slice = %q, want %q", got, want)
	}
}

func assert_slices[S ~[][]byte](t *testing.T, got S, want [][]byte) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("slice count = %d, want %d", len(got), len(want))
	}
	for index := range want {
		assert_slice(t, got[index], want[index])
	}
}

func slice(text string) (value shared_bytes.Slice) {
	return shared_bytes.Slice(text)
}

func search_slice_cases(t *testing.T, full shared_bytes.Slice) {
	t.Helper()
	cases := []struct {
		Source    shared_bytes.Slice
		Separator shared_bytes.Slice
	}{
		{Source: slice(""), Separator: slice("")},
		{Source: slice("abcabc"), Separator: slice("bc")},
		{Source: slice("abc"), Separator: slice("x")},
		{Source: full, Separator: slice("z")},
	}
	for _, one := range cases {
		source := []byte(one.Source)
		separator := []byte(one.Separator)
		if int(shared_bytes.Count(one.Source, one.Separator)) != standard_bytes.Count(
			source, separator,
		) {
			t.Fatalf("Count(%q, %q) differs", one.Source, one.Separator)
		}
		if int(shared_bytes.Index(one.Source, one.Separator)) != standard_bytes.Index(
			source, separator,
		) {
			t.Fatalf("Index(%q, %q) differs", one.Source, one.Separator)
		}
		if int(shared_bytes.Last_Index(
			one.Source, one.Separator,
		)) != standard_bytes.LastIndex(
			source, separator,
		) {
			t.Fatalf("Last_Index(%q, %q) differs", one.Source, one.Separator)
		}
		if bool(shared_bytes.Contains(
			one.Source, one.Separator,
		)) != standard_bytes.Contains(
			source, separator,
		) {
			t.Fatalf("Contains(%q, %q) differs", one.Source, one.Separator)
		}
	}
}

func case_mapping_cases(t *testing.T) {
	t.Helper()
	values := []shared_bytes.Slice{
		slice(""), slice("Hello, 世界"), slice("ÿß"),
		shared_bytes.Slice(standard_strings.Repeat("x", shared_bytes.SLICE_SIZE_MAXIMUM)),
	}
	for _, source := range values {
		assert_slice(t, shared_bytes.To_Upper(source), standard_bytes.ToUpper(source))
		assert_slice(t, shared_bytes.To_Lower(source), standard_bytes.ToLower(source))
		assert_slice(t, shared_bytes.To_Title(source), standard_bytes.ToTitle(source))
		assert_slice(t, shared_bytes.Title(source), standard_bytes.Title(source))
		assert_slice(
			t,
			shared_bytes.To_Upper_Special(turkish_case(), source),
			standard_bytes.ToUpperSpecial(standard_unicode.TurkishCase, source),
		)
		assert_slice(
			t,
			shared_bytes.To_Lower_Special(turkish_case(), source),
			standard_bytes.ToLowerSpecial(standard_unicode.TurkishCase, source),
		)
		assert_slice(
			t,
			shared_bytes.To_Title_Special(turkish_case(), source),
			standard_bytes.ToTitleSpecial(standard_unicode.TurkishCase, source),
		)
	}
}

func exercise_invariant_boundaries() {
	values := boundary_slices()
	exercise_comparison_boundaries(values)
	exercise_search_boundaries(values)
	exercise_split_boundaries(values)
	exercise_transform_boundaries(values)
	exercise_trim_boundaries(values)
	exercise_replace_boundaries(values)
	exercise_cut_boundaries(values)
	exercise_iteration_boundaries(values)
	exercise_buffer_boundaries()
	exercise_reader_boundaries()
}

func boundary_slices() (values []shared_bytes.Slice) {
	return []shared_bytes.Slice{
		slice(""),
		slice("a"),
		slice("aa"),
		shared_bytes.Slice(standard_strings.Repeat("a", shared_bytes.SLICE_SIZE_MAXIMUM)),
	}
}

func exercise_comparison_boundaries(values []shared_bytes.Slice) {
	for _, value := range values {
		shared_bytes.Equal(value, value)
		shared_bytes.Compare(value, value)
		shared_bytes.Equal_Fold(value, value)
		shared_bytes.Has_Prefix(value, value)
		shared_bytes.Has_Suffix(value, value)
	}
	shared_bytes.Equal(slice("a"), slice("b"))
	shared_bytes.Equal_Fold(slice("a"), slice("b"))
	shared_bytes.Has_Prefix(slice("a"), slice("b"))
	shared_bytes.Has_Suffix(slice("a"), slice("b"))
}

func exercise_search_boundaries(values []shared_bytes.Slice) {
	full := values[len(values)-1]
	last := shared_bytes.Slice(
		standard_strings.Repeat("a", shared_bytes.SLICE_SIZE_MAXIMUM-1) + "z",
	)
	for _, value := range values {
		shared_bytes.Count(value, value)
		shared_bytes.Contains(value, value)
		shared_bytes.Index(value, value)
		shared_bytes.Last_Index(value, value)
	}
	shared_bytes.Count(slice(""), slice("z"))
	shared_bytes.Count(slice("a"), slice("a"))
	shared_bytes.Count(slice("aa"), slice("a"))
	shared_bytes.Count(full, slice(""))
	index_sources := []shared_bytes.Slice{
		slice(""), slice("z"), slice("az"), slice("aaz"), last,
	}
	for _, source := range index_sources {
		shared_bytes.Index(source, slice("z"))
		shared_bytes.Last_Index(source, slice("z"))
		shared_bytes.Index_Byte(source, 'z')
		shared_bytes.Last_Index_Byte(source, 'z')
		shared_bytes.Index_Rune(source, 'z')
		shared_bytes.Index_Any(source, "z")
		shared_bytes.Last_Index_Any(source, "z")
	}
	shared_bytes.Last_Index(full, slice(""))
	exercise_character_search_boundaries(values)
	text_values := []shared_bytes.Text{
		"",
		"z",
		"zz",
		shared_bytes.Text(standard_strings.Repeat("z", shared_bytes.TEXT_SIZE_MAXIMUM)),
	}
	for index, text := range text_values {
		source := values[index]
		shared_bytes.Contains_Any(source, text)
		shared_bytes.Index_Any(source, text)
		shared_bytes.Last_Index_Any(source, text)
	}
	shared_bytes.Contains_Any(slice("z"), "z")
	match_z := func(character rune) (matches bool) { return character == 'z' }
	never := func(character rune) (matches bool) { return false }
	for _, source := range values {
		shared_bytes.Contains_Function(source, never)
		shared_bytes.Index_Function(source, never)
		shared_bytes.Last_Index_Function(source, never)
	}
	for _, source := range index_sources[1:] {
		shared_bytes.Contains_Function(source, match_z)
		shared_bytes.Index_Function(source, match_z)
		shared_bytes.Last_Index_Function(source, match_z)
	}
}

func exercise_character_search_boundaries(values []shared_bytes.Slice) {
	for _, value := range []shared_bytes.Byte{
		shared_bytes.Byte(bits.WORD_8_MINIMUM),
		1,
		2,
		shared_bytes.Byte(bits.WORD_8_MAXIMUM),
	} {
		shared_bytes.Index_Byte(slice(""), value)
		shared_bytes.Last_Index_Byte(slice(""), value)
	}
	characters := []shared_bytes.Character{
		shared_bytes.Character(bits.INTEGER_32_MINIMUM),
		-1,
		0,
		1,
		2,
		shared_bytes.Character(bits.INTEGER_32_MAXIMUM),
	}
	for _, character := range characters {
		shared_bytes.Contains_Rune(slice(""), character)
		shared_bytes.Index_Rune(slice(""), character)
	}
	for _, source := range values {
		shared_bytes.Contains_Rune(source, 'z')
	}
	shared_bytes.Contains_Rune(slice("z"), 'z')
}

func exercise_split_boundaries(values []shared_bytes.Slice) {
	full := values[len(values)-1]
	separators := shared_bytes.Slice(standard_strings.Repeat(
		"x", shared_bytes.SLICE_SIZE_MAXIMUM,
	))
	for index, value := range values {
		other := values[len(values)-1-index]
		shared_bytes.Split(value, other)
		shared_bytes.Split_After(value, other)
		shared_bytes.Split_N(value, other, shared_bytes.Limit(index))
		shared_bytes.Split_After_N(value, other, shared_bytes.Limit(index))
	}
	limits := []shared_bytes.Limit{-1, 0, 1, 2, shared_bytes.LIMIT_MAXIMUM}
	for _, limit := range limits {
		shared_bytes.Split_N(separators, slice("x"), limit)
		shared_bytes.Split_After_N(separators, slice("x"), limit)
	}
	shared_bytes.Split_N(full, slice(""), 0)
	shared_bytes.Split_N(slice("aa"), slice(""), shared_bytes.LIMIT_MAXIMUM)
	shared_bytes.Split_N(full, slice(""), shared_bytes.LIMIT_MAXIMUM)
	shared_bytes.Split_After(separators, slice("x"))
	full_fields := shared_bytes.Slice(standard_strings.Repeat(
		"a ", shared_bytes.FIELDS_COUNT_MAXIMUM,
	))
	space_match := func(character rune) (matches bool) {
		return standard_unicode.IsSpace(character)
	}
	for _, value := range values {
		shared_bytes.Fields(value)
		shared_bytes.Fields_Function(value, space_match)
	}
	shared_bytes.Fields_Function(full_fields, space_match)
	parts_values := []shared_bytes.Slices{
		nil,
		{nil},
		{nil, nil},
		make(shared_bytes.Slices, shared_bytes.SLICES_COUNT_MAXIMUM),
	}
	for _, parts := range parts_values {
		shared_bytes.Join(parts, slice(""))
	}
	for _, value := range values {
		shared_bytes.Join(shared_bytes.Slices{nil}, value)
		shared_bytes.Join(shared_bytes.Slices{value}, slice(""))
	}
	shared_bytes.Join(shared_bytes.Slices{full}, slice(""))
}

func exercise_transform_boundaries(values []shared_bytes.Slice) {
	identity := func(character rune) (mapped rune) { return character }
	turkish := turkish_case()
	for _, value := range values {
		shared_bytes.Map(identity, value)
		shared_bytes.To_Upper(value)
		shared_bytes.To_Lower(value)
		shared_bytes.To_Title(value)
		shared_bytes.To_Upper_Special(turkish, value)
		shared_bytes.To_Lower_Special(turkish, value)
		shared_bytes.To_Title_Special(turkish, value)
		shared_bytes.Title(value)
		shared_bytes.To_Valid_UTF8(value, value)
		shared_bytes.Runes(value)
		shared_bytes.Clone(value)
	}
	for _, count := range []int{0, 1, 2, len(turkish)} {
		special := ucd.Special_Case(turkish[:count])
		shared_bytes.To_Upper_Special(special, slice("i"))
		shared_bytes.To_Lower_Special(special, slice("I"))
		shared_bytes.To_Title_Special(special, slice("i"))
	}
	counts := []shared_bytes.Repeat_Count{
		0, 1, 2, shared_bytes.REPEAT_COUNT_MAXIMUM,
	}
	for _, count := range counts {
		shared_bytes.Repeat(slice("a"), count)
	}
	shared_bytes.Repeat(values[len(values)-1], 1)
}

func exercise_trim_boundaries(values []shared_bytes.Slice) {
	never := func(character rune) (matches bool) { return false }
	text_values := []shared_bytes.Text{
		"",
		"z",
		"zz",
		shared_bytes.Text(standard_strings.Repeat("z", shared_bytes.TEXT_SIZE_MAXIMUM)),
	}
	for index, value := range values {
		cutset := text_values[index]
		shared_bytes.Trim(value, cutset)
		shared_bytes.Trim_Left(value, cutset)
		shared_bytes.Trim_Right(value, cutset)
		shared_bytes.Trim_Function(value, never)
		shared_bytes.Trim_Left_Function(value, never)
		shared_bytes.Trim_Right_Function(value, never)
		shared_bytes.Trim_Space(value)
		shared_bytes.Trim_Prefix(value, slice(""))
		shared_bytes.Trim_Suffix(value, slice(""))
	}
	match_z := func(character rune) (matches bool) { return character == 'z' }
	shared_bytes.Trim_Left_Function(slice("za"), match_z)
	for _, value := range values {
		shared_bytes.Trim_Prefix(slice(""), value)
		shared_bytes.Trim_Suffix(slice(""), value)
	}
}

func exercise_replace_boundaries(values []shared_bytes.Slice) {
	empty_value := values[0]
	full := values[len(values)-1]
	for _, value := range values {
		shared_bytes.Replace(value, full, empty_value, 0)
		shared_bytes.Replace(empty_value, value, empty_value, 1)
		shared_bytes.Replace(empty_value, full, value, 2)
		shared_bytes.Replace_All(value, full, empty_value)
		shared_bytes.Replace_All(empty_value, value, empty_value)
		shared_bytes.Replace_All(empty_value, full, value)
	}
	for _, count := range []shared_bytes.Replacement_Count{
		-1, 0, 1, 2, shared_bytes.REPLACEMENT_COUNT_MAXIMUM,
	} {
		shared_bytes.Replace(empty_value, full, empty_value, count)
	}
}

func exercise_cut_boundaries(values []shared_bytes.Slice) {
	for _, value := range values {
		shared_bytes.Cut(value, value)
		shared_bytes.Cut(value, slice("z"))
		shared_bytes.Cut(value, slice(""))
		shared_bytes.Cut_Prefix(value, slice(""))
		shared_bytes.Cut_Suffix(value, slice(""))
		shared_bytes.Cut_Prefix(slice(""), value)
		shared_bytes.Cut_Suffix(slice(""), value)
	}
	shared_bytes.Cut(slice("za"), slice("z"))
	shared_bytes.Cut(slice("zaa"), slice("z"))
}

func exercise_iteration_boundaries(values []shared_bytes.Slice) {
	for index, value := range values {
		other := values[len(values)-1-index]
		collect(shared_bytes.Lines(value))
		collect(shared_bytes.Split_Sequence(value, other))
		collect(shared_bytes.Split_After_Sequence(value, other))
		collect(shared_bytes.Fields_Sequence(value))
		collect(shared_bytes.Fields_Function_Sequence(value, standard_unicode.IsSpace))
	}
}

func exercise_buffer_boundaries() {
	for _, size := range []int{0, 1, 2, shared_bytes.SLICE_SIZE_MAXIMUM} {
		exercise_buffer_size(size)
	}
	for _, available := range []int{0, 1, 2, shared_bytes.SLICE_SIZE_MAXIMUM} {
		content := make(shared_bytes.Slice, 0, available)
		buffer := shared_bytes.New_Buffer(content)
		shared_bytes.Buffer_Available(buffer)
		shared_bytes.Buffer_Available_Slice(buffer)
	}
	for _, count := range []shared_bytes.Growth_Count{
		0, 1, 2, shared_bytes.GROWTH_COUNT_MAXIMUM,
	} {
		shared_bytes.Buffer_Grow(buffer_with_size(0), count)
	}
	for _, value := range []byte{bits.WORD_8_MINIMUM, 1, 2, bits.WORD_8_MAXIMUM} {
		shared_bytes.Buffer_Write_Byte(buffer_with_size(0), shared_bytes.Byte(value))
	}
	for _, character := range []rune{
		bits.INTEGER_32_MINIMUM,
		-1,
		0,
		1,
		2,
		0x80,
		0x800,
		0x10000,
		bits.INTEGER_32_MAXIMUM,
	} {
		shared_bytes.Buffer_Write_Character(
			buffer_with_size(0),
			shared_bytes.Character(character),
		)
	}
	for _, delimiter := range []shared_bytes.Byte{
		shared_bytes.Byte(bits.WORD_8_MINIMUM),
		1,
		2,
		shared_bytes.Byte(bits.WORD_8_MAXIMUM),
	} {
		shared_bytes.Buffer_Read_Bytes(buffer_with_size(0), delimiter)
		shared_bytes.Buffer_Read_Text(buffer_with_size(0), delimiter)
	}
	four_byte_rune := shared_bytes.New_Buffer(slice("𐀀"))
	shared_bytes.Buffer_Read_Character(four_byte_rune)
	shared_bytes.Buffer_Unread_Character(four_byte_rune)
	exercise_buffer_read_boundaries()
}

func exercise_buffer_read_boundaries() {
	for _, value := range []byte{bits.WORD_8_MINIMUM, 1, 2, bits.WORD_8_MAXIMUM} {
		shared_bytes.Buffer_Read_Byte(shared_bytes.New_Buffer(shared_bytes.Slice{value}))
	}
	for _, character := range []rune{
		0, 1, 2, 0x80, 0x800, 0x10000, standard_utf8.MaxRune,
	} {
		shared_bytes.Buffer_Read_Character(
			shared_bytes.New_Buffer(slice(string(character))),
		)
	}
}

func exercise_buffer_size(size int) {
	shared_bytes.New_Buffer(make(shared_bytes.Slice, size, size))
	shared_bytes.New_Buffer_Text(shared_bytes.Text(standard_strings.Repeat("a", size)))
	shared_bytes.Buffer_Bytes(buffer_with_size(size))
	shared_bytes.Buffer_Available_Slice(buffer_with_size(size))
	shared_bytes.Buffer_String(buffer_with_size(size))
	shared_bytes.Buffer_Size(buffer_with_size(size))
	shared_bytes.Buffer_Capacity(buffer_with_size(size))
	shared_bytes.Buffer_Available(buffer_with_size(size))
	shared_bytes.Buffer_Peek(buffer_with_size(size), shared_bytes.Boundary(size))
	shared_bytes.Buffer_Truncate(buffer_with_size(size), shared_bytes.Boundary(size))
	shared_bytes.Buffer_Reset(buffer_with_size(size))
	shared_bytes.Buffer_Grow(buffer_with_size(size), 0)
	shared_bytes.Buffer_Write(buffer_with_size(size), nil)
	shared_bytes.Buffer_Write(buffer_with_size(0), make(shared_bytes.Slice, size))
	shared_bytes.Buffer_Write_Text(buffer_with_size(size), "")
	shared_bytes.Buffer_Write_Text(
		buffer_with_size(0),
		shared_bytes.Text(standard_strings.Repeat("a", size)),
	)
	shared_bytes.Buffer_Read_From(
		buffer_with_size(0),
		memory_stream(make(shared_bytes.Slice, size)),
	)
	shared_bytes.Buffer_Read_From(buffer_with_size(size), memory_stream(nil))
	shared_bytes.Buffer_Write_To(buffer_with_size(size), discard_stream())
	exercise_buffer_write_byte(size)
	exercise_buffer_write_rune(size)
	shared_bytes.Buffer_Read(
		buffer_with_size(size),
		make(shared_bytes.Slice, size),
	)
	buffer_stream := shared_bytes.Buffer_To_Stream(buffer_with_size(size))
	io.Query(buffer_stream)
	io.Read(buffer_stream, make([]byte, size))
	io.Write(
		shared_bytes.Buffer_To_Stream(buffer_with_size(0)),
		make([]byte, size),
	)
	shared_bytes.Buffer_Next(buffer_with_size(size), shared_bytes.Boundary(size))
	shared_bytes.Buffer_Read_Byte(buffer_with_size(size))
	shared_bytes.Buffer_Read_Character(buffer_with_size(size))
	shared_bytes.Buffer_Unread_Character(buffer_with_size(size))
	shared_bytes.Buffer_Unread_Byte(buffer_with_size(size))
	shared_bytes.Buffer_Read_Bytes(
		buffer_with_size(size),
		shared_bytes.Byte(bits.WORD_8_MAXIMUM),
	)
	shared_bytes.Buffer_Read_Text(
		buffer_with_size(size),
		shared_bytes.Byte(bits.WORD_8_MAXIMUM),
	)
	consumed := buffer_with_size(size)
	shared_bytes.Buffer_Next(consumed, shared_bytes.Boundary(size))
	shared_bytes.Buffer_Bytes(consumed)
}

func exercise_buffer_write_byte(size int) {
	buffer := buffer_with_size(size)
	if size == shared_bytes.SLICE_SIZE_MAXIMUM {
		shared_bytes.Buffer_Read_Byte(buffer)
	}
	shared_bytes.Buffer_Write_Byte(buffer, 0)
}

func exercise_buffer_write_rune(size int) {
	buffer := buffer_with_size(size)
	if size == shared_bytes.SLICE_SIZE_MAXIMUM {
		shared_bytes.Buffer_Next(buffer, 4)
	}
	shared_bytes.Buffer_Write_Character(buffer, 'a')
}

func buffer_with_size(size int) (buffer *shared_bytes.Buffer) {
	content := make(shared_bytes.Slice, size, size)
	for index := range content {
		content[index] = 'a'
	}
	return shared_bytes.New_Buffer(content)
}

func exercise_reader_boundaries() {
	for _, size := range []int{0, 1, 2, shared_bytes.SLICE_SIZE_MAXIMUM} {
		exercise_reader_size(size)
	}
	for _, receiver_size := range []int{0, 1, 2, shared_bytes.SLICE_SIZE_MAXIMUM} {
		reader := reader_with_size(receiver_size)
		shared_bytes.Reader_Reset(reader, nil)
	}
	for _, source_size := range []int{0, 1, 2, shared_bytes.SLICE_SIZE_MAXIMUM} {
		shared_bytes.Reader_Reset(
			reader_with_size(0),
			make(shared_bytes.Slice, source_size),
		)
	}
	reader := reader_with_size(shared_bytes.SLICE_SIZE_MAXIMUM)
	shared_bytes.Reader_Seek(
		reader,
		shared_bytes.SLICE_SIZE_MAXIMUM,
		io.SEEK_FROM_START,
	)
	shared_bytes.Reader_Unread_Size(reader)
	for _, offset := range []shared_bytes.Reader_Offset{
		shared_bytes.Reader_Offset(bits.INTEGER_64_MINIMUM),
		-2,
		-1,
		0,
		1,
		2,
		shared_bytes.Reader_Offset(bits.INTEGER_64_MAXIMUM),
	} {
		shared_bytes.Reader_Read_At(reader, nil, offset)
		shared_bytes.Reader_Seek(reader, offset, io.SEEK_FROM_START)
		io.Read_At(
			shared_bytes.Reader_To_Stream(reader),
			nil,
			int64(offset),
		)
	}
	for _, position := range []int64{1, 2, shared_bytes.SLICE_SIZE_MAXIMUM - 1} {
		reader = reader_with_size(shared_bytes.SLICE_SIZE_MAXIMUM)
		shared_bytes.Reader_Seek(
			reader,
			shared_bytes.Reader_Offset(position),
			io.SEEK_FROM_START,
		)
		shared_bytes.Reader_Read_Character(reader)
		shared_bytes.Reader_Unread_Character(reader)
	}
	exercise_reader_read_boundaries()
}

func exercise_reader_read_boundaries() {
	for _, value := range []byte{bits.WORD_8_MINIMUM, 1, 2, bits.WORD_8_MAXIMUM} {
		shared_bytes.Reader_Read_Byte(shared_bytes.New_Reader(shared_bytes.Slice{value}))
	}
	for _, character := range []rune{
		0, 1, 2, 0x80, 0x800, 0x10000, standard_utf8.MaxRune,
	} {
		shared_bytes.Reader_Read_Character(
			shared_bytes.New_Reader(slice(string(character))),
		)
	}
	reader := reader_with_size(shared_bytes.SLICE_SIZE_MAXIMUM)
	for _, size := range []int{0, 1, 2, shared_bytes.SLICE_SIZE_MAXIMUM} {
		shared_bytes.Reader_Read_At(reader, make(shared_bytes.Slice, size), 0)
	}
}

func exercise_reader_size(size int) {
	shared_bytes.Reader_Unread_Size(reader_with_size(size))
	shared_bytes.Reader_Size(reader_with_size(size))
	shared_bytes.Reader_Read(
		reader_with_size(size),
		make(shared_bytes.Slice, size),
	)
	shared_bytes.Reader_Read_At(reader_with_size(size), nil, 0)
	shared_bytes.Reader_Read_Byte(reader_with_size(size))
	shared_bytes.Reader_Unread_Byte(reader_with_size(size))
	shared_bytes.Reader_Read_Character(reader_with_size(size))
	exercise_reader_unread_rune(size)
	shared_bytes.Reader_Seek(
		reader_with_size(size),
		0,
		io.SEEK_FROM_START,
	)
	shared_bytes.Reader_Write_To(reader_with_size(size), discard_stream())
	reader_stream := shared_bytes.Reader_To_Stream(reader_with_size(size))
	io.Query(reader_stream)
	io.Read(reader_stream, make([]byte, size))
	io.Read_At(reader_stream, make([]byte, size), 0)
	io.Seek(reader_stream, 0, io.SEEK_FROM_START)
	io.Size(reader_stream)
}

func exercise_reader_unread_rune(size int) {
	reader := reader_with_size(size)
	if size == 0 {
		shared_bytes.Reader_Seek(reader, 1, io.SEEK_FROM_START)
	} else {
		shared_bytes.Reader_Read_Character(reader)
	}
	shared_bytes.Reader_Unread_Character(reader)
}

func reader_with_size(size int) (reader *shared_bytes.Reader) {
	content := make(shared_bytes.Slice, size)
	for index := range content {
		content[index] = 'a'
	}
	return shared_bytes.New_Reader(content)
}

func memory_stream(content []byte) (stream io.Stream) {
	state := &io.Stream_Memory{Memory: content}
	return io.Memory_To_Stream(state)
}

func discard_stream() (stream io.Stream) {
	state := &io.Stream_Discard{}
	return io.Discard_To_Stream(state)
}

func collect(
	sequence func(func(shared_bytes.Slice) (continue_iteration bool)),
) (slices shared_bytes.Slices) {
	for value := range sequence {
		slices = append(slices, value)
	}
	return slices
}

func collect_standard(
	sequence func(func([]byte) (continue_iteration bool)),
) (slices [][]byte) {
	for value := range sequence {
		slices = append(slices, value)
	}
	return slices
}

func turkish_case() (special ucd.Special_Case) {
	return ucd.Special_Case(ucd.Turkish_Case())
}

func panicked(action func()) (yes bool) {
	defer func() {
		if recover() != nil {
			yes = true
		}
	}()
	action()
	return false
}
