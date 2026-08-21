// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package ucd_test

import (
	"testing"

	"local/james-orcales/shared/math/bits"
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
	testify.True(t, bool(ucd.Is(ucd.Range_Table_Handle(&table), 'A')),
		"the first 16-bit range")
	testify.True(t, bool(ucd.Is(ucd.Range_Table_Handle(&table), 'c')),
		"the 16-bit stride")
	testify.False(t, bool(ucd.Is(ucd.Range_Table_Handle(&table), 'b')),
		"a 16-bit stride gap")
	testify.True(t, bool(ucd.Is(ucd.Range_Table_Handle(&table), 0x10010)),
		"the 32-bit range")
	testify.False(t, bool(ucd.Is(ucd.Range_Table_Handle(&table), 0x1000f)),
		"a 32-bit stride gap")
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
		Name        string
		Shared      func(ucd.Character) (yes ucd.Boolean)
		Member      ucd.Character
		Nonmember   ucd.Character
		Zero_To_Two bool
	}{
		{Name: "control", Shared: ucd.Is_Control,
			Member: '\n', Nonmember: 'A', Zero_To_Two: true},
		{Name: "digit", Shared: ucd.Is_Digit,
			Member: '١', Nonmember: 'A'},
		{Name: "graphic", Shared: ucd.Is_Graphic,
			Member: ' ', Nonmember: '\n'},
		{Name: "letter", Shared: ucd.Is_Letter,
			Member: '世', Nonmember: '1'},
		{Name: "lower", Shared: ucd.Is_Lower,
			Member: 'å', Nonmember: 'Å'},
		{Name: "mark", Shared: ucd.Is_Mark,
			Member: '\u0300', Nonmember: 'A'},
		{Name: "number", Shared: ucd.Is_Number,
			Member: '\u2165', Nonmember: 'A'},
		{Name: "print", Shared: ucd.Is_Print,
			Member: ' ', Nonmember: '\n'},
		{Name: "punctuation", Shared: ucd.Is_Punctuation,
			Member: '!', Nonmember: 'A'},
		{Name: "space", Shared: ucd.Is_Space,
			Member: '\u3000', Nonmember: 'A'},
		{Name: "symbol", Shared: ucd.Is_Symbol,
			Member: '€', Nonmember: 'A'},
		{Name: "title", Shared: ucd.Is_Title,
			Member: '\u01c5', Nonmember: 'a'},
		{Name: "upper", Shared: ucd.Is_Upper,
			Member: 'Å', Nonmember: 'å'},
	}
	for _, one := range cases {
		testify.True(t, bool(one.Shared(one.Member)), one.Name)
		testify.False(t, bool(one.Shared(one.Nonmember)), one.Name)
		for _, boundary := range []ucd.Character{
			ucd.Character(bits.INTEGER_32_MINIMUM),
			-1,
			ucd.Character(bits.INTEGER_32_MAXIMUM),
		} {
			testify.False(t, bool(one.Shared(boundary)), "%s at %d", one.Name, boundary)
		}
		for _, boundary := range []ucd.Character{0, 1, 2} {
			testify.Equal(t, one.Zero_To_Two, bool(one.Shared(boundary)),
				"%s at %d", one.Name, boundary)
		}
	}
}

// Test_Case_Conversion preserves the upstream behavior coverage.
func Test_Case_Conversion(t *testing.T) {
	t.Parallel()
	for _, one := range []struct {
		Character ucd.Character
		Upper     ucd.Character
		Lower     ucd.Character
		Title     ucd.Character
		Fold      ucd.Character
	}{
		{Character: 'a', Upper: 'A', Lower: 'a', Title: 'A', Fold: 'A'},
		{Character: 'A', Upper: 'A', Lower: 'a', Title: 'A', Fold: 'a'},
		{Character: 'å', Upper: 'Å', Lower: 'å', Title: 'Å', Fold: '\u212b'},
		{Character: 'Å', Upper: 'Å', Lower: 'å', Title: 'Å', Fold: 'å'},
		{Character: '\u0131', Upper: 'I', Lower: '\u0131', Title: 'I', Fold: '\u0131'},
		{Character: '\u212a', Upper: '\u212a', Lower: 'k', Title: '\u212a', Fold: 'K'},
		{Character: ucd.Character(bits.INTEGER_32_MINIMUM),
			Upper: ucd.Character(bits.INTEGER_32_MINIMUM),
			Lower: ucd.Character(bits.INTEGER_32_MINIMUM),
			Title: ucd.Character(bits.INTEGER_32_MINIMUM),
			Fold:  ucd.Character(bits.INTEGER_32_MINIMUM)},
		{Character: -1, Upper: -1, Lower: -1, Title: -1, Fold: -1},
		{Character: 0, Upper: 0, Lower: 0, Title: 0, Fold: 0},
		{Character: 1, Upper: 1, Lower: 1, Title: 1, Fold: 1},
		{Character: 2, Upper: 2, Lower: 2, Title: 2, Fold: 2},
		{Character: ucd.RUNE_MAX, Upper: ucd.RUNE_MAX, Lower: ucd.RUNE_MAX,
			Title: ucd.RUNE_MAX, Fold: ucd.RUNE_MAX},
		{Character: ucd.Character(bits.INTEGER_32_MAXIMUM),
			Upper: ucd.Character(bits.INTEGER_32_MAXIMUM),
			Lower: ucd.Character(bits.INTEGER_32_MAXIMUM),
			Title: ucd.Character(bits.INTEGER_32_MAXIMUM),
			Fold:  ucd.Character(bits.INTEGER_32_MAXIMUM)},
	} {
		testify.Equal(t, one.Upper, ucd.To_Upper(one.Character),
			"To_Upper(%U)", one.Character)
		testify.Equal(t, one.Lower, ucd.To_Lower(one.Character),
			"To_Lower(%U)", one.Character)
		testify.Equal(t, one.Title, ucd.To_Title(one.Character),
			"To_Title(%U)", one.Character)
		testify.Equal(t, one.Fold, ucd.Simple_Fold(one.Character),
			"Simple_Fold(%U)", one.Character)
		testify.Equal(t, one.Upper, ucd.To(ucd.UPPER_CASE, one.Character))
		testify.Equal(t, one.Lower, ucd.To(ucd.LOWER_CASE, one.Character))
		testify.Equal(t, one.Title, ucd.To(ucd.TITLE_CASE, one.Character))
	}
	turkish := turkish_case()
	azerbaijani := azeri_case()
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
	table, found := named_table(ucd.TABLE_KIND_CATEGORY, "L")
	testify.True(t, bool(found), "the L category")
	testify.True(t, bool(ucd.Is(ucd.Range_Table_Handle(&table), 'A')),
		"membership in category L")
	verify_named_query_boundaries(t)
	verify_named_table_boundaries(t)
}

// Test_Unicode_Data preserves the upstream behavior coverage.
func Test_Unicode_Data(t *testing.T) {
	t.Parallel()
	testify.Equal(t, "15.0.0", ucd.VERSION)
	testify.Equal(t, '\U0010FFFF', rune(ucd.RUNE_MAX))
	testify.Equal(t, '\uFFFD',
		rune(ucd.REPLACEMENT_CHARACTER))
	testify.Equal(t, '\u007F', rune(ucd.ASCII_MAX))
	testify.Equal(t, '\u00FF', rune(ucd.LATIN_1_MAX))
	alias, alias_found := ucd.Category_Alias("Cased_Letter")
	testify.True(t, bool(alias_found), "the Cased_Letter alias")
	testify.Equal(t, ucd.Category_Alias_Name("LC"), alias)
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

// Test_Allocation proves each public operation keeps heap allocation at zero.
func Test_Allocation(t *testing.T) {
	var ranges_16 [ucd.RANGES_16_COUNT_MAXIMUM]ucd.Range_16
	var ranges_32 [ucd.RANGES_32_COUNT_MAXIMUM]ucd.Range_32
	state := allocation_state{
		Ranges_16: ranges_16[:],
		Ranges_32: ranges_32[:],
		Range:     ucd.Ranges_16{{Minimum: 'A', Maximum: 'Z', Stride: 1}},
		Special: special_case(
			1,
			ucd.Case_Range{
				Minimum: 'a', Maximum: 'z',
				Deltas: case_delta(-32, 0, -32),
			},
			ucd.Case_Range{}, ucd.Case_Range{}, ucd.Case_Range{},
		),
	}
	state.Range_Table = ucd.Range_Table{Ranges_16: state.Range, Latin_Offset: 1}
	state.Range_Tables = ucd.Range_Tables{&state.Range_Table}
	assert_zero_allocations(t, membership_allocation_cases(&state))
	assert_zero_allocations(t, conversion_allocation_cases(&state))
}

// Test_Domain_Errors preserves the upstream behavior coverage.
func Test_Domain_Errors(t *testing.T) {
	t.Parallel()
	testify.Panics(t, func() { ucd.To(-1, 'a') }, "an invalid Case")
	invalid_range := ucd.Range_Table{Ranges_16: ucd.Ranges_16{
		{Minimum: 1, Maximum: 2, Stride: 0},
	}}
	testify.Panics(t, func() {
		ucd.Is(ucd.Range_Table_Handle(&invalid_range), 1)
	}, "a zero stride")
	large_name := ucd.Name(repeat("x", ucd.NAME_SIZE_MAXIMUM+1))
	testify.Panics(t, func() {
		ucd.Named_Table(ucd.TABLE_KIND_CATEGORY, large_name, nil, nil)
	}, "an oversize Name")
	testify.Panics(t, func() {
		ucd.Named_Table(ucd.TABLE_KIND_CATEGORY, "L", nil, nil)
	}, "missing named table storage")
	short_ranges_16 := make(ucd.Ranges_16, 2)
	short_ranges_32 := make(ucd.Ranges_32, 2)
	for _, size := range []int{1, 2} {
		ucd.Named_Table(
			ucd.TABLE_KIND_CATEGORY, "unknown",
			short_ranges_16[:size], short_ranges_32[:size],
		)
		testify.Panics(t, func() {
			ucd.Named_Table(
				ucd.TABLE_KIND_CATEGORY, "L",
				short_ranges_16[:size], short_ranges_32[:size],
			)
		}, "short named table storage")
	}
	_, found := ucd.Named_Table(ucd.TABLE_KIND_CATEGORY, "unknown", nil, nil)
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
	testify.Panics(t, func() { ucd.Turkish_Case(nil) }, "missing Turkish case storage")
	testify.Panics(t, func() { ucd.Azeri_Case(nil) }, "missing Azeri case storage")
	invalid_special := ucd.Special_Case{
		Count: ucd.SPECIAL_CASE_COUNT_MAXIMUM + 1,
	}
	testify.Panics(t, func() {
		ucd.Special_Case_To_Upper(invalid_special, 'i')
	}, "oversize special case count")
}

type allocation_case struct {
	Name string
	Run  func()
}

type allocation_state struct {
	Table         ucd.Range_Table
	Found         ucd.Boolean
	Alias         ucd.Category_Alias_Name
	Language_Case ucd.Special_Case
	Ranges_16     ucd.Ranges_16
	Ranges_32     ucd.Ranges_32
	Boolean       ucd.Boolean
	Character     ucd.Character
	Range         ucd.Ranges_16
	Range_Table   ucd.Range_Table
	Range_Tables  ucd.Range_Tables
	Special       ucd.Special_Case
}

func turkish_case() (special ucd.Special_Case) {
	ucd.Turkish_Case(ucd.Special_Case_Destination(&special))
	return special
}

func azeri_case() (special ucd.Special_Case) {
	ucd.Azeri_Case(ucd.Special_Case_Destination(&special))
	return special
}

func case_delta(
	upper ucd.Upper_Case_Delta,
	lower ucd.Lower_Case_Delta,
	title ucd.Title_Case_Delta,
) (delta ucd.Case_Delta) {
	return ucd.Case_Delta{Upper: upper, Lower: lower, Title: title}
}

func special_case(
	count ucd.Special_Case_Count,
	first ucd.Case_Range,
	second ucd.Case_Range,
	third ucd.Case_Range,
	fourth ucd.Case_Range,
) (special ucd.Special_Case) {
	return ucd.Special_Case_Of(count, first, second, third, fourth)
}

func named_table(
	kind ucd.Table_Kind, name ucd.Name,
) (table ucd.Range_Table, found ucd.Boolean) {
	var ranges_16 [ucd.RANGES_16_COUNT_MAXIMUM]ucd.Range_16
	var ranges_32 [ucd.RANGES_32_COUNT_MAXIMUM]ucd.Range_32
	return ucd.Named_Table(kind, name, ranges_16[:], ranges_32[:])
}

func assert_zero_allocations(t *testing.T, cases []allocation_case) {
	t.Helper()
	for _, one := range cases {
		t.Run(one.Name, func(t *testing.T) { testify.Zero_Allocation(t, one.Run) })
	}
}

func membership_allocation_cases(state *allocation_state) (cases []allocation_case) {
	return []allocation_case{
		{Name: "Is", Run: func() {
			state.Boolean = ucd.Is(ucd.Range_Table_Handle(&state.Range_Table), 'A')
		}},
		{Name: "Is_One_Of", Run: func() {
			state.Boolean = ucd.Is_One_Of(state.Range_Tables, 'A')
		}},
		{Name: "In", Run: func() {
			state.Boolean = ucd.In('A', state.Range_Tables)
		}},
		{Name: "Is_Control", Run: func() { state.Boolean = ucd.Is_Control('\n') }},
		{Name: "Is_Digit", Run: func() { state.Boolean = ucd.Is_Digit('1') }},
		{Name: "Is_Graphic", Run: func() { state.Boolean = ucd.Is_Graphic('世') }},
		{Name: "Is_Print", Run: func() { state.Boolean = ucd.Is_Print('世') }},
		{Name: "Is_Letter", Run: func() { state.Boolean = ucd.Is_Letter('世') }},
		{Name: "Is_Lower", Run: func() { state.Boolean = ucd.Is_Lower('a') }},
		{Name: "Is_Mark", Run: func() { state.Boolean = ucd.Is_Mark('\u0300') }},
		{Name: "Is_Number", Run: func() { state.Boolean = ucd.Is_Number('1') }},
		{Name: "Is_Punctuation", Run: func() {
			state.Boolean = ucd.Is_Punctuation('.')
		}},
		{Name: "Is_Space", Run: func() { state.Boolean = ucd.Is_Space(' ') }},
		{Name: "Is_Symbol", Run: func() { state.Boolean = ucd.Is_Symbol('$') }},
		{Name: "Is_Title", Run: func() { state.Boolean = ucd.Is_Title('\u01c5') }},
		{Name: "Is_Upper", Run: func() { state.Boolean = ucd.Is_Upper('A') }},
		{Name: "Is_Category", Run: func() {
			state.Boolean = ucd.Is_Category('A', "L")
		}},
		{Name: "Is_Script", Run: func() {
			state.Boolean = ucd.Is_Script('A', "Latin")
		}},
		{Name: "Is_Property", Run: func() {
			state.Boolean = ucd.Is_Property(' ', "White_Space")
		}},
		{Name: "Is_Fold_Category", Run: func() {
			state.Boolean = ucd.Is_Fold_Category('A', "Ll")
		}},
		{Name: "Is_Fold_Script", Run: func() {
			state.Boolean = ucd.Is_Fold_Script('\u00b5', "Greek")
		}},
	}
}

func conversion_allocation_cases(state *allocation_state) (cases []allocation_case) {
	return []allocation_case{
		{Name: "Named_Table", Run: func() {
			state.Table, state.Found = ucd.Named_Table(
				ucd.TABLE_KIND_CATEGORY, "L",
				state.Ranges_16, state.Ranges_32,
			)
		}},
		{Name: "Category_Alias", Run: func() {
			state.Alias, state.Found = ucd.Category_Alias("Cased_Letter")
		}},
		{Name: "Turkish_Case", Run: func() {
			ucd.Turkish_Case(ucd.Special_Case_Destination(&state.Language_Case))
		}},
		{Name: "Azeri_Case", Run: func() {
			ucd.Azeri_Case(ucd.Special_Case_Destination(&state.Language_Case))
		}},
		{Name: "To", Run: func() {
			state.Character = ucd.To(ucd.UPPER_CASE, 'a')
		}},
		{Name: "To_Upper", Run: func() { state.Character = ucd.To_Upper('a') }},
		{Name: "To_Lower", Run: func() { state.Character = ucd.To_Lower('A') }},
		{Name: "To_Title", Run: func() { state.Character = ucd.To_Title('a') }},
		{Name: "Special_Case_To_Upper", Run: func() {
			state.Character = ucd.Special_Case_To_Upper(state.Special, 'a')
		}},
		{Name: "Special_Case_To_Lower", Run: func() {
			state.Character = ucd.Special_Case_To_Lower(state.Special, 'A')
		}},
		{Name: "Special_Case_To_Title", Run: func() {
			state.Character = ucd.Special_Case_To_Title(state.Special, 'a')
		}},
		{Name: "Simple_Fold", Run: func() {
			state.Character = ucd.Simple_Fold('K')
		}},
	}
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

func verify_maximum_range_table(t *testing.T) {
	t.Helper()
	maximum_table := maximum_range_table()
	for _, character := range []ucd.Character{
		ucd.Character(bits.INTEGER_32_MINIMUM), -1, 0, 1, 2,
		ucd.Character(ucd.RANGE_16_MAXIMUM),
		ucd.Character(ucd.RANGE_32_MINIMUM),
		ucd.Character(ucd.RANGE_32_MINIMUM + 1),
		ucd.Character(ucd.RANGE_32_MAXIMUM),
		ucd.Character(bits.INTEGER_32_MAXIMUM),
	} {
		testify.Equal(t, reference_range_table_contains(maximum_table, character),
			bool(ucd.Is(ucd.Range_Table_Handle(&maximum_table), character)),
			"Is(%d)", character)
	}
}

func reference_range_table_contains(
	table ucd.Range_Table, character ucd.Character,
) (contains bool) {
	if character < 0 {
		return false
	}
	for _, one := range table.Ranges_16 {
		if character < ucd.Character(one.Minimum) {
			break
		}
		if character <= ucd.Character(one.Maximum) {
			difference := character - ucd.Character(one.Minimum)
			return difference%ucd.Character(one.Stride) == 0
		}
	}
	for _, one := range table.Ranges_32 {
		if character < ucd.Character(one.Minimum) {
			break
		}
		if character <= ucd.Character(one.Maximum) {
			difference := character - ucd.Character(one.Minimum)
			return difference%ucd.Character(one.Stride) == 0
		}
	}
	return false
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
	testify.True(t, bool(ucd.Is(ucd.Range_Table_Handle(&one_latin_range), 0)),
		"one Latin range")
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
		ucd.Range_Table_Handle(&two_high_ranges),
		ucd.Character(ucd.RANGE_32_MINIMUM+1),
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
		testify.Equal(t,
			bool(ucd.Is(ucd.Range_Table_Handle(&maximum_table), character)),
			bool(ucd.Is_One_Of(maximum_tables, character)),
			"Is_One_Of(%d)", character)
		testify.Equal(t,
			bool(ucd.Is(ucd.Range_Table_Handle(&maximum_table), character)),
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
				rune(one.Shared(ucd.Special_Case{}, character)),
				"%s empty override at %d", one.Name, character)
		}
	}
}

func verify_zero_delta_special_cases(t *testing.T) {
	t.Helper()
	zero_ranges := special_case(
		ucd.SPECIAL_CASE_COUNT_MAXIMUM,
		ucd.Case_Range{Minimum: 0, Maximum: 0, Deltas: case_delta(0, 0, 0)},
		ucd.Case_Range{Minimum: 1, Maximum: 1, Deltas: case_delta(0, 0, 0)},
		ucd.Case_Range{Minimum: 2, Maximum: 2, Deltas: case_delta(0, 0, 0)},
		ucd.Case_Range{Minimum: ucd.Case_Range_Minimum(ucd.RUNE_MAX),
			Maximum: ucd.Case_Range_Maximum(ucd.RUNE_MAX),
			Deltas:  case_delta(0, 0, 0)},
	)
	empty_ranges := special_case(
		ucd.SPECIAL_CASE_COUNT_MINIMUM,
		ucd.Case_Range{}, ucd.Case_Range{}, ucd.Case_Range{}, ucd.Case_Range{},
	)
	testify.Equal_Values(t, ucd.SPECIAL_CASE_COUNT_MINIMUM, empty_ranges.Count)
	two_ranges := special_case(
		2,
		ucd.Case_Range{Minimum: 0, Maximum: 0, Deltas: case_delta(0, 0, 0)},
		ucd.Case_Range{Minimum: 1, Maximum: 1, Deltas: case_delta(0, 0, 0)},
		ucd.Case_Range{}, ucd.Case_Range{},
	)
	one_range := special_case(
		1,
		ucd.Case_Range{Minimum: 0, Maximum: 0, Deltas: case_delta(0, 0, 0)},
		ucd.Case_Range{}, ucd.Case_Range{}, ucd.Case_Range{},
	)
	for _, special := range []ucd.Special_Case{one_range, two_ranges} {
		turkish := special
		ucd.Turkish_Case(ucd.Special_Case_Destination(&turkish))
		testify.Equal_Values(t, ucd.SPECIAL_CASE_COUNT_MAXIMUM, turkish.Count)
		azeri := special
		ucd.Azeri_Case(ucd.Special_Case_Destination(&azeri))
		testify.Equal_Values(t, ucd.SPECIAL_CASE_COUNT_MAXIMUM, azeri.Count)
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
		testify.Equal(t, 0, int(one.Shared(two_ranges, 0)),
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
			Deltas: case_delta(
				ucd.Upper_Case_Delta(ucd.CASE_DELTA_MINIMUM),
				ucd.Lower_Case_Delta(ucd.CASE_DELTA_MINIMUM),
				ucd.Title_Case_Delta(ucd.CASE_DELTA_MINIMUM),
			),
		}, Character: ucd.RUNE_MAX},
		{Range: ucd.Case_Range{
			Minimum: 0, Maximum: 1,
			Deltas: case_delta(
				ucd.Upper_Case_Delta(ucd.CASE_DELTA_MAXIMUM),
				ucd.Lower_Case_Delta(ucd.CASE_DELTA_MAXIMUM),
				ucd.Title_Case_Delta(ucd.CASE_DELTA_MAXIMUM),
			),
		}, Character: 0},
		{Range: ucd.Case_Range{
			Minimum: 1, Maximum: 1, Deltas: case_delta(-1, -1, -1),
		}, Character: 1},
		{Range: ucd.Case_Range{
			Minimum: 0, Maximum: 0, Deltas: case_delta(1, 1, 1),
		}, Character: 0},
		{Range: ucd.Case_Range{
			Minimum: 0, Maximum: 0, Deltas: case_delta(2, 2, 2),
		}, Character: 0},
		{Range: ucd.Case_Range{
			Minimum: 2, Maximum: 2, Deltas: case_delta(0, 0, 0),
		}, Character: 2},
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
			special := special_case(
				ucd.SPECIAL_CASE_COUNT_MAXIMUM,
				delta_case.Range, delta_case.Range,
				delta_case.Range, delta_case.Range,
			)
			mapped := one.Shared(
				special,
				delta_case.Character,
			)
			testify.True(t, mapped >= 0 && mapped <= ucd.RUNE_MAX,
				"%s mapped code point", one.Name)
			turkish := special
			ucd.Turkish_Case(ucd.Special_Case_Destination(&turkish))
			testify.Equal_Values(t, ucd.SPECIAL_CASE_COUNT_MAXIMUM, turkish.Count)
			azeri := special
			ucd.Azeri_Case(ucd.Special_Case_Destination(&azeri))
			testify.Equal_Values(t, ucd.SPECIAL_CASE_COUNT_MAXIMUM, azeri.Count)
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
		_, table_found := named_table(one.Kind, one.Name)
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
			_, unknown_found := ucd.Named_Table(kind, name, nil, nil)
			testify.False(
				t, bool(unknown_found),
				"unknown table kind %d and name size %d", kind, len(name),
			)
		}
	}
}

func repeat(text string, count int) (repeated string) {
	storage := make([]byte, len(text)*count)
	for copy_index := 0; copy_index < count; copy_index++ {
		copy(storage[copy_index*len(text):], text)
	}
	return string(storage)
}
