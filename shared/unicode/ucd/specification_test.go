// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package ucd_test

import (
	"testing"
	standard_unicode "unicode"

	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/strings"
	"local/james-orcales/shared/testify"
	"local/james-orcales/shared/unicode/ucd"
)

// Test_Range_Tables preserves the upstream behavior coverage.
func Test_Range_Tables(t *testing.T) {
	t.Parallel()
	table := ucd.Range_Table{
		Ranges_16: ucd.Ranges_16{
			{Minimum: 'A', Maximum: 'Z', Stride: 1},
			{Minimum: 'a', Maximum: 'z', Stride: 2},
		},
		Ranges_32: ucd.Ranges_32{
			{Minimum: 0x10000, Maximum: 0x10010, Stride: 2},
		},
		Latin_Offset: 2,
	}
	testify.True(t, bool(ucd.Is(&table, 'A')), "the first 16-bit range")
	testify.True(t, bool(ucd.Is(&table, 'c')), "the 16-bit stride")
	testify.False(t, bool(ucd.Is(&table, 'b')), "a 16-bit stride gap")
	testify.True(t, bool(ucd.Is(&table, 0x10010)), "the 32-bit range")
	testify.False(t, bool(ucd.Is(&table, 0x1000f)), "a 32-bit stride gap")
	tables := ucd.Range_Tables{&table}
	testify.True(t, bool(ucd.Is_One_Of(tables, 'A')), "Is_One_Of")
	testify.True(t, bool(ucd.In('A', tables)), "In")
	testify.False(t, bool(ucd.In('0', tables)), "an absent character")
	verify_maximum_range_table(t)
	verify_range_table_collections(t)
}

// Test_Classification preserves the upstream behavior coverage.
func Test_Classification(t *testing.T) {
	t.Parallel()
	cases := []struct {
		Name      string
		Shared    func(ucd.Character) (yes ucd.Boolean)
		Standard  func(rune) (yes bool)
		Member    ucd.Character
		Nonmember ucd.Character
	}{
		{Name: "control", Shared: ucd.Is_Control,
			Standard: standard_unicode.IsControl, Member: '\n', Nonmember: 'A'},
		{Name: "digit", Shared: ucd.Is_Digit,
			Standard: standard_unicode.IsDigit, Member: '١', Nonmember: 'A'},
		{Name: "graphic", Shared: ucd.Is_Graphic,
			Standard: standard_unicode.IsGraphic, Member: ' ', Nonmember: '\n'},
		{Name: "letter", Shared: ucd.Is_Letter,
			Standard: standard_unicode.IsLetter, Member: '世', Nonmember: '1'},
		{Name: "lower", Shared: ucd.Is_Lower,
			Standard: standard_unicode.IsLower, Member: 'å', Nonmember: 'Å'},
		{Name: "mark", Shared: ucd.Is_Mark,
			Standard: standard_unicode.IsMark, Member: '\u0300', Nonmember: 'A'},
		{Name: "number", Shared: ucd.Is_Number,
			Standard: standard_unicode.IsNumber, Member: '\u2165', Nonmember: 'A'},
		{Name: "print", Shared: ucd.Is_Print,
			Standard: standard_unicode.IsPrint, Member: ' ', Nonmember: '\n'},
		{Name: "punctuation", Shared: ucd.Is_Punctuation,
			Standard: standard_unicode.IsPunct, Member: '!', Nonmember: 'A'},
		{Name: "space", Shared: ucd.Is_Space,
			Standard: standard_unicode.IsSpace, Member: '\u3000', Nonmember: 'A'},
		{Name: "symbol", Shared: ucd.Is_Symbol,
			Standard: standard_unicode.IsSymbol, Member: '€', Nonmember: 'A'},
		{Name: "title", Shared: ucd.Is_Title,
			Standard: standard_unicode.IsTitle, Member: '\u01c5', Nonmember: 'a'},
		{Name: "upper", Shared: ucd.Is_Upper,
			Standard: standard_unicode.IsUpper, Member: 'Å', Nonmember: 'å'},
	}
	for _, one := range cases {
		testify.Equal(
			t, one.Standard(rune(one.Member)), bool(one.Shared(one.Member)), one.Name,
		)
		testify.Equal(
			t, one.Standard(rune(one.Nonmember)),
			bool(one.Shared(one.Nonmember)), one.Name,
		)
		for _, boundary := range []ucd.Character{
			ucd.Character(bits.INTEGER_32_MINIMUM),
			-1, 0, 1, 2,
			ucd.Character(bits.INTEGER_32_MAXIMUM),
		} {
			testify.Equal(t, one.Standard(rune(boundary)), bool(one.Shared(boundary)),
				"%s at %d", one.Name, boundary)
		}
	}
}

// Test_Case_Conversion preserves the upstream behavior coverage.
func Test_Case_Conversion(t *testing.T) {
	t.Parallel()
	for _, character := range []ucd.Character{
		'a', 'A', 'å', 'Å', '\u0131', '\u212a', ucd.Character(bits.INTEGER_32_MINIMUM),
		-1, 0, 1, 2, ucd.RUNE_MAX, ucd.Character(bits.INTEGER_32_MAXIMUM),
	} {
		testify.Equal(t, standard_unicode.ToUpper(rune(character)),
			rune(ucd.To_Upper(character)), "To_Upper(%U)", character)
		testify.Equal(t, standard_unicode.ToLower(rune(character)),
			rune(ucd.To_Lower(character)), "To_Lower(%U)", character)
		testify.Equal(t, standard_unicode.ToTitle(rune(character)),
			rune(ucd.To_Title(character)), "To_Title(%U)", character)
		testify.Equal(t, standard_unicode.SimpleFold(rune(character)),
			rune(ucd.Simple_Fold(character)), "Simple_Fold(%U)", character)
	}
	for _, case_value := range []ucd.Case{
		ucd.UPPER_CASE, ucd.LOWER_CASE, ucd.TITLE_CASE,
	} {
		for _, character := range []ucd.Character{
			ucd.Character(bits.INTEGER_32_MINIMUM), -1, 0, 1, 2, 'a', 'A', '\u01c5',
			ucd.RUNE_MAX, ucd.Character(bits.INTEGER_32_MAXIMUM),
		} {
			testify.Equal(t, standard_unicode.To(int(case_value), rune(character)),
				rune(ucd.To(case_value, character)), "To(%d, %U)",
				case_value, character)
		}
	}
	turkish := ucd.Special_Case(ucd.Turkish_Case())
	azerbaijani := ucd.Special_Case(ucd.Azeri_Case())
	testify.Equal(t, 'İ', rune(ucd.Special_Case_To_Upper(turkish, 'i')))
	testify.Equal(t, 'ı', rune(ucd.Special_Case_To_Lower(turkish, 'I')))
	testify.Equal(t, 'İ', rune(ucd.Special_Case_To_Title(azerbaijani, 'i')))
	verify_empty_special_cases(t)
	verify_zero_delta_special_cases(t)
	verify_delta_special_cases(t)
}

// Test_Named_Tables preserves the upstream behavior coverage.
func Test_Named_Tables(t *testing.T) {
	t.Parallel()
	testify.True(t, bool(ucd.Is_Category('A', "L")), "Is_Category")
	testify.True(t, bool(ucd.Is_Script('A', "Latin")), "Is_Script")
	testify.True(t, bool(ucd.Is_Property(' ', "White_Space")), "Is_Property")
	testify.True(t, bool(ucd.Is_Fold_Category('A', "Ll")),
		"Is_Fold_Category")
	testify.True(t, bool(ucd.Is_Fold_Script('\u00b5', "Greek")),
		"Is_Fold_Script")
	table, found := ucd.Named_Table(ucd.TABLE_KIND_CATEGORY, "L")
	testify.True(t, bool(found), "the L category")
	testify.True(t, bool(ucd.Is(&table, 'A')), "membership in category L")
	verify_named_query_boundaries(t)
	verify_named_table_boundaries(t)
}

// Test_Unicode_Data preserves the upstream behavior coverage.
func Test_Unicode_Data(t *testing.T) {
	t.Parallel()
	testify.Equal(t, standard_unicode.Version, ucd.VERSION)
	testify.Equal(t, standard_unicode.MaxRune, rune(ucd.RUNE_MAX))
	testify.Equal(t, standard_unicode.ReplacementChar,
		rune(ucd.REPLACEMENT_CHARACTER))
	testify.Equal(t, standard_unicode.MaxASCII, rune(ucd.ASCII_MAX))
	testify.Equal(t, standard_unicode.MaxLatin1, rune(ucd.LATIN_1_MAX))
	alias, alias_found := ucd.Category_Alias("Cased_Letter")
	testify.True(t, bool(alias_found), "the Cased_Letter alias")
	testify.Equal(t, ucd.Category_Alias_Name("LC"), alias)
	testify.Equal(t, len(standard_unicode.TurkishCase), len(ucd.Turkish_Case()))
	testify.Equal(t, len(standard_unicode.AzeriCase), len(ucd.Azeri_Case()))
	for _, one := range []struct {
		Name      ucd.Name
		Canonical ucd.Category_Alias_Name
		Found     bool
	}{
		{Name: "", Canonical: "", Found: false},
		{Name: "x", Canonical: "", Found: false},
		{Name: "xx", Canonical: "", Found: false},
		{Name: ucd.Name(repeat("x", ucd.NAME_SIZE_MAXIMUM)),
			Canonical: "", Found: false},
		{Name: "Letter", Canonical: "L", Found: true},
		{Name: "Cased_Letter", Canonical: "LC", Found: true},
	} {
		canonical, found := ucd.Category_Alias(one.Name)
		testify.Equal(t, one.Found, bool(found), "alias %q found", one.Name)
		testify.Equal(t, one.Canonical, canonical, "alias %q", one.Name)
	}
}

// Test_Domain_Errors preserves the upstream behavior coverage.
func Test_Domain_Errors(t *testing.T) {
	t.Parallel()
	testify.Panics(t, func() { ucd.To(-1, 'a') }, "an invalid Case")
	invalid_range := ucd.Range_Table{Ranges_16: ucd.Ranges_16{
		{Minimum: 1, Maximum: 2, Stride: 0},
	}}
	testify.Panics(t, func() { ucd.Is(&invalid_range, 1) }, "a zero stride")
	large_name := ucd.Name(repeat("x", ucd.NAME_SIZE_MAXIMUM+1))
	testify.Panics(t, func() {
		ucd.Named_Table(ucd.TABLE_KIND_CATEGORY, large_name)
	}, "an oversize Name")
	_, found := ucd.Named_Table(ucd.TABLE_KIND_CATEGORY, "unknown")
	testify.False(t, bool(found), "an unknown category")
	testify.False(t, bool(ucd.Is_Category('A', "unknown")),
		"an unknown category query")
	testify.False(t, bool(ucd.Is_Script('A', "unknown")),
		"an unknown script query")
	testify.False(t, bool(ucd.Is_Property('A', "unknown")),
		"an unknown property query")
	testify.False(t, bool(ucd.Is_Fold_Category('A', "unknown")),
		"an unknown fold category query")
	testify.False(t, bool(ucd.Is_Fold_Script('A', "unknown")),
		"an unknown fold script query")
}

// Test_Encoding_Constants verifies shared machine limits, widths, and Turkic code points.
func Test_Encoding_Constants(t *testing.T) {
	t.Parallel()
	testify.Equal_Values(t, bits.INTEGER_32_MINIMUM, ucd.CHARACTER_MINIMUM)
	testify.Equal_Values(t, bits.INTEGER_32_MAXIMUM, ucd.CHARACTER_MAXIMUM)
	testify.Equal_Values(t, bits.WORD_16_MINIMUM, ucd.RANGE_16_MINIMUM)
	testify.Equal_Values(t, bits.WORD_16_MAXIMUM, ucd.RANGE_16_MAXIMUM)
	testify.Equal_Values(t, bits.WORD_32_MAXIMUM, ucd.ENCODED_NUMBER_MAXIMUM)
	testify.Equal_Values(t, 'I', ucd.LATIN_CAPITAL_I)
	testify.Equal_Values(t, 'i', ucd.LATIN_SMALL_I)
	testify.Equal_Values(t, 'İ', ucd.LATIN_CAPITAL_I_WITH_DOT)
	testify.Equal_Values(t, 'ı', ucd.LATIN_SMALL_DOTLESS_I)
}

func maximum_range_table() (table ucd.Range_Table) {
	table = ucd.Range_Table{
		Ranges_16: make(
			ucd.Ranges_16, ucd.RANGES_16_COUNT_MAXIMUM,
		),
		Ranges_32: make(
			ucd.Ranges_32, ucd.RANGES_32_COUNT_MAXIMUM,
		),
		Latin_Offset: ucd.LATIN_OFFSET_MAXIMUM,
	}
	for index := range table.Ranges_16 {
		value := uint16(index)
		table.Ranges_16[index] = ucd.Range_16{
			Minimum: ucd.Range_16_Minimum(value),
			Maximum: ucd.Range_16_Maximum(value),
			Stride:  1,
		}
	}
	table.Ranges_16[2].Stride = 2
	table.Ranges_16[len(table.Ranges_16)-1] = ucd.Range_16{
		Minimum: ucd.Range_16_Minimum(ucd.RANGE_16_MAXIMUM),
		Maximum: ucd.Range_16_Maximum(ucd.RANGE_16_MAXIMUM),
		Stride:  ucd.Range_16_Stride(ucd.RANGE_16_STRIDE_MAXIMUM),
	}
	for index := range table.Ranges_32 {
		value := uint32(ucd.RANGE_32_MINIMUM) + uint32(index)
		table.Ranges_32[index] = ucd.Range_32{
			Minimum: ucd.Range_32_Minimum(value),
			Maximum: ucd.Range_32_Maximum(value),
			Stride:  1,
		}
	}
	table.Ranges_32[1].Stride = 2
	table.Ranges_32[len(table.Ranges_32)-1] = ucd.Range_32{
		Minimum: ucd.Range_32_Minimum(ucd.RANGE_32_MAXIMUM),
		Maximum: ucd.Range_32_Maximum(ucd.RANGE_32_MAXIMUM),
		Stride:  ucd.Range_32_Stride(ucd.RANGE_32_STRIDE_MAXIMUM),
	}
	return table
}

func standard_range_table(
	table ucd.Range_Table,
) (standard_table *standard_unicode.RangeTable) {
	standard_table = &standard_unicode.RangeTable{
		R16:         make([]standard_unicode.Range16, len(table.Ranges_16)),
		R32:         make([]standard_unicode.Range32, len(table.Ranges_32)),
		LatinOffset: int(table.Latin_Offset),
	}
	for index, one := range table.Ranges_16 {
		standard_table.R16[index] = standard_unicode.Range16{
			Lo: uint16(one.Minimum), Hi: uint16(one.Maximum),
			Stride: uint16(one.Stride),
		}
	}
	for index, one := range table.Ranges_32 {
		standard_table.R32[index] = standard_unicode.Range32{
			Lo: uint32(one.Minimum), Hi: uint32(one.Maximum),
			Stride: uint32(one.Stride),
		}
	}
	return standard_table
}

func verify_maximum_range_table(t *testing.T) {
	t.Helper()
	maximum_table := maximum_range_table()
	standard_table := standard_range_table(maximum_table)
	for _, character := range []ucd.Character{
		ucd.Character(bits.INTEGER_32_MINIMUM), -1, 0, 1, 2,
		ucd.Character(ucd.RANGE_16_MAXIMUM),
		ucd.Character(ucd.RANGE_32_MINIMUM),
		ucd.Character(ucd.RANGE_32_MINIMUM + 1),
		ucd.Character(ucd.RANGE_32_MAXIMUM),
		ucd.Character(bits.INTEGER_32_MAXIMUM),
	} {
		testify.Equal(t, standard_unicode.Is(standard_table, rune(character)),
			bool(ucd.Is(&maximum_table, character)), "Is(%d)", character)
	}
}

func verify_range_table_collections(t *testing.T) {
	t.Helper()
	maximum_table := maximum_range_table()
	one_latin_range := ucd.Range_Table{
		Ranges_16: ucd.Ranges_16{
			{Minimum: 0, Maximum: 0, Stride: 1},
		},
		Latin_Offset: 1,
	}
	testify.True(t, bool(ucd.Is(&one_latin_range, 0)), "one Latin range")
	two_high_ranges := ucd.Range_Table{Ranges_32: ucd.Ranges_32{
		{Minimum: ucd.Range_32_Minimum(ucd.RANGE_32_MINIMUM),
			Maximum: ucd.Range_32_Maximum(ucd.RANGE_32_MINIMUM),
			Stride:  1},
		{Minimum: ucd.Range_32_Minimum(ucd.RANGE_32_MINIMUM + 1),
			Maximum: ucd.Range_32_Maximum(
				ucd.RANGE_32_MINIMUM + 1,
			),
			Stride: 1},
	}}
	testify.True(t, bool(ucd.Is(
		&two_high_ranges, ucd.Character(ucd.RANGE_32_MINIMUM+1),
	)), "two 32-bit ranges without a 16-bit range")
	maximum_tables := make(
		ucd.Range_Tables, ucd.RANGE_TABLES_COUNT_MAXIMUM,
	)
	for index := range maximum_tables {
		maximum_tables[index] = &maximum_table
	}
	for _, character := range []ucd.Character{
		ucd.Character(bits.INTEGER_32_MINIMUM),
		-1, 0, 1, 2,
		ucd.Character(bits.INTEGER_32_MAXIMUM),
	} {
		testify.Equal(t, bool(ucd.Is(&maximum_table, character)),
			bool(ucd.Is_One_Of(maximum_tables, character)),
			"Is_One_Of(%d)", character)
		testify.Equal(t, bool(ucd.Is(&maximum_table, character)),
			bool(ucd.In(character, maximum_tables)), "In(%d)", character)
	}
	testify.False(t, bool(ucd.Is_One_Of(nil, 0)), "an empty Is_One_Of")
	testify.False(t, bool(ucd.In(0, nil)), "an empty In")
	two_tables := ucd.Range_Tables{&one_latin_range, &maximum_table}
	testify.True(t, bool(ucd.Is_One_Of(two_tables, 0)), "two Is_One_Of tables")
	testify.True(t, bool(ucd.In(0, two_tables)), "two In tables")
}

func verify_empty_special_cases(t *testing.T) {
	t.Helper()
	case_functions := []struct {
		Name       string
		Case_Value ucd.Case
		Shared     func(ucd.Special_Case, ucd.Character) (
			mapped ucd.Character,
		)
	}{
		{Name: "upper", Case_Value: ucd.UPPER_CASE,
			Shared: ucd.Special_Case_To_Upper},
		{Name: "lower", Case_Value: ucd.LOWER_CASE,
			Shared: ucd.Special_Case_To_Lower},
		{Name: "title", Case_Value: ucd.TITLE_CASE,
			Shared: ucd.Special_Case_To_Title},
	}
	for _, one := range case_functions {
		for _, character := range []ucd.Character{
			ucd.Character(bits.INTEGER_32_MINIMUM), -1, 0, 1, 2, ucd.RUNE_MAX,
			ucd.Character(bits.INTEGER_32_MAXIMUM),
		} {
			testify.Equal(t, rune(ucd.To(one.Case_Value, character)),
				rune(one.Shared(nil, character)),
				"%s empty override at %d", one.Name, character)
		}
	}
}

func verify_zero_delta_special_cases(t *testing.T) {
	t.Helper()
	zero_ranges := ucd.Special_Case{
		{Minimum: 0, Maximum: 0, Deltas: ucd.Case_Delta{0, 0, 0}},
		{Minimum: 1, Maximum: 1, Deltas: ucd.Case_Delta{0, 0, 0}},
		{Minimum: 2, Maximum: 2, Deltas: ucd.Case_Delta{0, 0, 0}},
		{Minimum: ucd.Case_Range_Minimum(ucd.RUNE_MAX),
			Maximum: ucd.Case_Range_Maximum(ucd.RUNE_MAX),
			Deltas:  ucd.Case_Delta{0, 0, 0}},
	}
	case_functions := []struct {
		Name   string
		Shared func(ucd.Special_Case, ucd.Character) (
			mapped ucd.Character,
		)
	}{
		{Name: "upper", Shared: ucd.Special_Case_To_Upper},
		{Name: "lower", Shared: ucd.Special_Case_To_Lower},
		{Name: "title", Shared: ucd.Special_Case_To_Title},
	}
	for _, one := range case_functions {
		for _, character := range []ucd.Character{
			0, 1, 2, ucd.RUNE_MAX,
		} {
			testify.Equal(t, rune(character), rune(one.Shared(zero_ranges, character)),
				"%s zero delta at %d", one.Name, character)
		}
		testify.Equal(t, 0, int(one.Shared(zero_ranges[:2], 0)),
			"%s two override rules", one.Name)
	}
}

func verify_delta_special_cases(t *testing.T) {
	t.Helper()
	delta_cases := []struct {
		Range     ucd.Case_Range
		Character ucd.Character
	}{
		{Range: ucd.Case_Range{
			Minimum: ucd.Case_Range_Minimum(ucd.RUNE_MAX),
			Maximum: ucd.Case_Range_Maximum(ucd.RUNE_MAX),
			Deltas: ucd.Case_Delta{
				ucd.CASE_DELTA_MINIMUM,
				ucd.CASE_DELTA_MINIMUM,
				ucd.CASE_DELTA_MINIMUM,
			},
		}, Character: ucd.RUNE_MAX},
		{Range: ucd.Case_Range{
			Minimum: 0, Maximum: 1,
			Deltas: ucd.Case_Delta{
				ucd.CASE_DELTA_MAXIMUM,
				ucd.CASE_DELTA_MAXIMUM,
				ucd.CASE_DELTA_MAXIMUM,
			},
		}, Character: 0},
		{Range: ucd.Case_Range{
			Minimum: 1, Maximum: 1, Deltas: ucd.Case_Delta{-1, -1, -1},
		}, Character: 1},
		{Range: ucd.Case_Range{
			Minimum: 0, Maximum: 0, Deltas: ucd.Case_Delta{1, 1, 1},
		}, Character: 0},
		{Range: ucd.Case_Range{
			Minimum: 0, Maximum: 0, Deltas: ucd.Case_Delta{2, 2, 2},
		}, Character: 0},
	}
	case_functions := []struct {
		Name   string
		Shared func(ucd.Special_Case, ucd.Character) (
			mapped ucd.Character,
		)
	}{
		{Name: "upper", Shared: ucd.Special_Case_To_Upper},
		{Name: "lower", Shared: ucd.Special_Case_To_Lower},
		{Name: "title", Shared: ucd.Special_Case_To_Title},
	}
	for _, one := range case_functions {
		for _, delta_case := range delta_cases {
			mapped := one.Shared(
				ucd.Special_Case{delta_case.Range}, delta_case.Character,
			)
			testify.True(t, mapped >= 0 && mapped <= ucd.RUNE_MAX,
				"%s mapped code point", one.Name)
		}
	}
}

func verify_named_query_boundaries(t *testing.T) {
	t.Helper()
	named_queries := []struct {
		Name  string
		Query func(
			ucd.Character, ucd.Name,
		) (yes ucd.Boolean)
	}{
		{Name: "category", Query: ucd.Is_Category},
		{Name: "script", Query: ucd.Is_Script},
		{Name: "property", Query: ucd.Is_Property},
		{Name: "fold category", Query: ucd.Is_Fold_Category},
		{Name: "fold script", Query: ucd.Is_Fold_Script},
	}
	maximum_name := ucd.Name(repeat("x", ucd.NAME_SIZE_MAXIMUM))
	for _, one := range named_queries {
		for _, character := range []ucd.Character{
			ucd.Character(bits.INTEGER_32_MINIMUM),
			-1, 0, 1, 2,
			ucd.Character(bits.INTEGER_32_MAXIMUM),
		} {
			testify.False(t, bool(one.Query(character, "xx")),
				"%s at %d", one.Name, character)
		}
		for _, name := range []ucd.Name{"", "x", "xx", maximum_name} {
			testify.False(t, bool(one.Query('A', name)),
				"%s name size %d", one.Name, len(name))
		}
	}
}

func verify_named_table_boundaries(t *testing.T) {
	t.Helper()
	maximum_name := ucd.Name(repeat("x", ucd.NAME_SIZE_MAXIMUM))
	table_cases := []struct {
		Kind ucd.Table_Kind
		Name ucd.Name
	}{
		{Kind: ucd.TABLE_KIND_CATEGORY, Name: "L"},
		{Kind: ucd.TABLE_KIND_CATEGORY, Name: "Cc"},
		{Kind: ucd.TABLE_KIND_CATEGORY, Name: "Cn"},
		{Kind: ucd.TABLE_KIND_CATEGORY, Name: "P"},
		{Kind: ucd.TABLE_KIND_CATEGORY, Name: "Zs"},
		{Kind: ucd.TABLE_KIND_SCRIPT, Name: "Shavian"},
		{Kind: ucd.TABLE_KIND_SCRIPT, Name: "Khojki"},
		{Kind: ucd.TABLE_KIND_PROPERTY, Name: "Regional_Indicator"},
		{Kind: ucd.TABLE_KIND_PROPERTY, Name: "IDS_Binary_Operator"},
		{Kind: ucd.TABLE_KIND_FOLD_CATEGORY, Name: "L"},
		{Kind: ucd.TABLE_KIND_FOLD_CATEGORY, Name: "M"},
		{Kind: ucd.TABLE_KIND_FOLD_SCRIPT, Name: "Greek"},
		{Kind: ucd.TABLE_KIND_FOLD_SCRIPT, Name: "Inherited"},
	}
	for _, one := range table_cases {
		_, table_found := ucd.Named_Table(one.Kind, one.Name)
		testify.True(
			t, bool(table_found), "table kind %d and name %s", one.Kind, one.Name,
		)
	}
	for _, kind := range []ucd.Table_Kind{
		ucd.TABLE_KIND_CATEGORY,
		ucd.TABLE_KIND_SCRIPT,
		ucd.TABLE_KIND_PROPERTY,
		ucd.TABLE_KIND_FOLD_CATEGORY,
		ucd.TABLE_KIND_FOLD_SCRIPT,
	} {
		for _, name := range []ucd.Name{"", "x", "xx", maximum_name} {
			_, unknown_found := ucd.Named_Table(kind, name)
			testify.False(
				t, bool(unknown_found),
				"unknown table kind %d and name size %d", kind, len(name),
			)
		}
	}
}

// Test_Standard_Library_Tables preserves the upstream behavior coverage.
func Test_Standard_Library_Tables(t *testing.T) {
	t.Parallel()
	families := []struct {
		Kind   ucd.Table_Kind
		Tables map[string]*standard_unicode.RangeTable
	}{
		{Kind: ucd.TABLE_KIND_CATEGORY, Tables: standard_unicode.Categories},
		{Kind: ucd.TABLE_KIND_SCRIPT, Tables: standard_unicode.Scripts},
		{Kind: ucd.TABLE_KIND_PROPERTY, Tables: standard_unicode.Properties},
		{Kind: ucd.TABLE_KIND_FOLD_CATEGORY,
			Tables: standard_unicode.FoldCategory},
		{Kind: ucd.TABLE_KIND_FOLD_SCRIPT, Tables: standard_unicode.FoldScript},
	}
	for _, family := range families {
		for name, standard_table := range family.Tables {
			shared_table, found := ucd.Named_Table(
				family.Kind, ucd.Name(name),
			)
			testify.True(t, bool(found), "table kind %d and name %s", family.Kind, name)
			testify.Equal(t, standard_table.LatinOffset, int(shared_table.Latin_Offset),
				"Latin offset for %s", name)
			testify.Equal(t, len(standard_table.R16), len(shared_table.Ranges_16),
				"16-bit range count for %s", name)
			testify.Equal(t, len(standard_table.R32), len(shared_table.Ranges_32),
				"32-bit range count for %s", name)
			for index, standard_range := range standard_table.R16 {
				shared_range := shared_table.Ranges_16[index]
				testify.Equal(t, standard_range.Lo, uint16(shared_range.Minimum),
					"16-bit minimum for %s at %d", name, index)
				testify.Equal(t, standard_range.Hi, uint16(shared_range.Maximum),
					"16-bit maximum for %s at %d", name, index)
				testify.Equal(t, standard_range.Stride, uint16(shared_range.Stride),
					"16-bit stride for %s at %d", name, index)
			}
			for index, standard_range := range standard_table.R32 {
				shared_range := shared_table.Ranges_32[index]
				testify.Equal(t, standard_range.Lo, uint32(shared_range.Minimum),
					"32-bit minimum for %s at %d", name, index)
				testify.Equal(t, standard_range.Hi, uint32(shared_range.Maximum),
					"32-bit maximum for %s at %d", name, index)
				testify.Equal(t, standard_range.Stride, uint32(shared_range.Stride),
					"32-bit stride for %s at %d", name, index)
			}
		}
	}
}

// Test_Standard_Library_Classification preserves the upstream behavior coverage.
func Test_Standard_Library_Classification(t *testing.T) {
	t.Parallel()
	classifications := []struct {
		Name     string
		Shared   func(ucd.Character) (yes ucd.Boolean)
		Standard func(rune) (yes bool)
	}{
		{Name: "control", Shared: ucd.Is_Control,
			Standard: standard_unicode.IsControl},
		{Name: "digit", Shared: ucd.Is_Digit,
			Standard: standard_unicode.IsDigit},
		{Name: "graphic", Shared: ucd.Is_Graphic,
			Standard: standard_unicode.IsGraphic},
		{Name: "letter", Shared: ucd.Is_Letter,
			Standard: standard_unicode.IsLetter},
		{Name: "lower", Shared: ucd.Is_Lower,
			Standard: standard_unicode.IsLower},
		{Name: "mark", Shared: ucd.Is_Mark,
			Standard: standard_unicode.IsMark},
		{Name: "number", Shared: ucd.Is_Number,
			Standard: standard_unicode.IsNumber},
		{Name: "print", Shared: ucd.Is_Print,
			Standard: standard_unicode.IsPrint},
		{Name: "punctuation", Shared: ucd.Is_Punctuation,
			Standard: standard_unicode.IsPunct},
		{Name: "space", Shared: ucd.Is_Space,
			Standard: standard_unicode.IsSpace},
		{Name: "symbol", Shared: ucd.Is_Symbol,
			Standard: standard_unicode.IsSymbol},
		{Name: "title", Shared: ucd.Is_Title,
			Standard: standard_unicode.IsTitle},
		{Name: "upper", Shared: ucd.Is_Upper,
			Standard: standard_unicode.IsUpper},
	}
	characters := make([]rune, 0, int(standard_unicode.MaxLatin1)+20)
	for character := rune(0); character <= standard_unicode.MaxLatin1; character++ {
		characters = append(characters, character)
	}
	characters = append(characters,
		-0x100, -0x101, -0x1c5, -0x300, -0x660, -0x37e, -0x2c2, -0x1680,
		0x10000, 0x10400, 0x10428, 0x1d7ce, 0x1f1ff, 0x20000, 0x2fa1d,
		standard_unicode.MaxRune,
	)
	for _, classification := range classifications {
		for _, character := range characters {
			testify.Equal(t, classification.Standard(character),
				bool(classification.Shared(ucd.Character(character))),
				"%s at %U", classification.Name, character)
		}
	}
}

// Test_Standard_Library_Case_Data preserves the upstream behavior coverage.
func Test_Standard_Library_Case_Data(t *testing.T) {
	t.Parallel()
	for _, case_range := range standard_unicode.CaseRanges {
		for character := case_range.Lo; character <= case_range.Hi; character++ {
			for _, case_value := range []ucd.Case{
				ucd.UPPER_CASE,
				ucd.LOWER_CASE,
				ucd.TITLE_CASE,
			} {
				testify.Equal(
					t, standard_unicode.To(int(case_value), rune(character)),
					rune(ucd.To(
						case_value, ucd.Character(character),
					)),
					"case %d at %U", case_value, character)
			}
			testify.Equal(t, standard_unicode.SimpleFold(rune(character)),
				rune(ucd.Simple_Fold(
					ucd.Character(character),
				)),
				"simple fold at %U", character)
		}
	}
	for _, cycle := range []string{
		"Aa", "δΔ", "KkK", "Ssſ", "ρϱΡ", "ͅΙιι", "İ", "ı", "\u13b0\uab80",
	} {
		for _, character := range cycle {
			testify.Equal(t, standard_unicode.SimpleFold(character),
				rune(ucd.Simple_Fold(
					ucd.Character(character),
				)),
				"simple fold orbit at %U", character)
		}
	}
}

// Test_Standard_Library_Remaining_Data preserves the upstream behavior coverage.
func Test_Standard_Library_Remaining_Data(t *testing.T) {
	t.Parallel()
	for alias, standard_name := range standard_unicode.CategoryAliases {
		shared_name, found := ucd.Category_Alias(ucd.Name(alias))
		testify.True(t, bool(found), "category alias %s", alias)
		testify.Equal(t, standard_name, string(shared_name), "category alias %s", alias)
	}
	standard_special_cases := []standard_unicode.SpecialCase{
		standard_unicode.TurkishCase,
		standard_unicode.AzeriCase,
	}
	shared_special_cases := []ucd.Language_Case{
		ucd.Turkish_Case(),
		ucd.Azeri_Case(),
	}
	for case_index, standard_special := range standard_special_cases {
		shared_special := shared_special_cases[case_index]
		testify.Equal(t, len(standard_special), len(shared_special),
			"special case %d range count", case_index)
		for range_index, standard_range := range standard_special {
			shared_range := shared_special[range_index]
			testify.Equal(t, standard_range.Lo, uint32(shared_range.Minimum),
				"special case %d minimum at %d", case_index, range_index)
			testify.Equal(t, standard_range.Hi, uint32(shared_range.Maximum),
				"special case %d maximum at %d", case_index, range_index)
			for delta_index, standard_delta := range standard_range.Delta {
				testify.Equal(
					t, standard_delta, rune(shared_range.Deltas[delta_index]),
					"special case %d delta at %d and %d",
					case_index, range_index, delta_index)
			}
		}
	}
}

func repeat(text string, count int) (repeated string) {
	return string(strings.Repeat(
		strings.Text(text), strings.Repeat_Count(count),
	))
}
