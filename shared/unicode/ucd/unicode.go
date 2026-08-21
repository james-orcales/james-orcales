// Package ucd supplies Unicode property tests and simple case conversion. It ports the Go
// standard library algorithms and data into repository domain types and immutable constants.
package ucd

import (
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/aver/default"
)

// VERSION is the Unicode edition from which the tables derive.
const VERSION = "15.0.0"

// RUNE_MAX is the final valid Unicode code point.
const RUNE_MAX Character = '\U0010FFFF'

// REPLACEMENT_CHARACTER represents an invalid code point.
const REPLACEMENT_CHARACTER Character = '\uFFFD'

// ASCII_MAX is the final ASCII code point.
const ASCII_MAX Character = '\u007F'

// ASCII_UPPER_FIRST is the first uppercase ASCII letter.
const ASCII_UPPER_FIRST Character = 'A'

// ASCII_UPPER_FINAL is the final uppercase ASCII letter.
const ASCII_UPPER_FINAL Character = 'Z'

// ASCII_LOWER_FIRST is the first lowercase ASCII letter.
const ASCII_LOWER_FIRST Character = 'a'

// ASCII_LOWER_FINAL is the final lowercase ASCII letter.
const ASCII_LOWER_FINAL Character = 'z'

// ASCII_CASE_DELTA is the distance between corresponding ASCII letters.
const ASCII_CASE_DELTA Character = ASCII_LOWER_FIRST - ASCII_UPPER_FIRST

// LATIN_CAPITAL_I is the Latin capital letter I.
const LATIN_CAPITAL_I Character = 'I'

// LATIN_SMALL_I is the Latin small letter i.
const LATIN_SMALL_I Character = 'i'

// LATIN_CAPITAL_I_WITH_DOT is the Latin capital letter I with dot above.
const LATIN_CAPITAL_I_WITH_DOT Character = 'İ'

// LATIN_SMALL_DOTLESS_I is the Latin small dotless letter i.
const LATIN_SMALL_DOTLESS_I Character = 'ı'

// ASCII_FOLD_DATA stores the next simple-fold character for each ASCII value.
const ASCII_FOLD_DATA = "" +
	"\x00\x00\x00\x01\x00\x02\x00\x03\x00\x04\x00\x05\x00\x06\x00\x07\x00\x08\x00\x09" +
	"\x00\x0a\x00\x0b\x00\x0c\x00\x0d\x00\x0e\x00\x0f\x00\x10\x00\x11\x00\x12\x00\x13" +
	"\x00\x14\x00\x15\x00\x16\x00\x17\x00\x18\x00\x19\x00\x1a\x00\x1b\x00\x1c\x00\x1d" +
	"\x00\x1e\x00\x1f\x00\x20\x00\x21\x00\x22\x00\x23\x00\x24\x00\x25\x00\x26\x00\x27" +
	"\x00\x28\x00\x29\x00\x2a\x00\x2b\x00\x2c\x00\x2d\x00\x2e\x00\x2f\x00\x30\x00\x31" +
	"\x00\x32\x00\x33\x00\x34\x00\x35\x00\x36\x00\x37\x00\x38\x00\x39\x00\x3a\x00\x3b" +
	"\x00\x3c\x00\x3d\x00\x3e\x00\x3f\x00\x40\x00\x61\x00\x62\x00\x63\x00\x64\x00\x65" +
	"\x00\x66\x00\x67\x00\x68\x00\x69\x00\x6a\x00\x6b\x00\x6c\x00\x6d\x00\x6e\x00\x6f" +
	"\x00\x70\x00\x71\x00\x72\x00\x73\x00\x74\x00\x75\x00\x76\x00\x77\x00\x78\x00\x79" +
	"\x00\x7a\x00\x5b\x00\x5c\x00\x5d\x00\x5e\x00\x5f\x00\x60\x00\x41\x00\x42\x00\x43" +
	"\x00\x44\x00\x45\x00\x46\x00\x47\x00\x48\x00\x49\x00\x4a\x21\x2a\x00\x4c\x00\x4d" +
	"\x00\x4e\x00\x4f\x00\x50\x00\x51\x00\x52\x01\x7f\x00\x54\x00\x55\x00\x56\x00\x57" +
	"\x00\x58\x00\x59\x00\x5a\x00\x7b\x00\x7c\x00\x7d\x00\x7e\x00\x7f"

// LATIN_1_MAX is the final Latin-1 code point.
const LATIN_1_MAX Character = '\u00FF'

// CJK_UNIFIED_IDEOGRAPHS_MINIMUM is the first unified ideograph in the basic block.
const CJK_UNIFIED_IDEOGRAPHS_MINIMUM Character = '\u4E00'

// CJK_UNIFIED_IDEOGRAPHS_MAXIMUM is the final unified ideograph in the basic block.
const CJK_UNIFIED_IDEOGRAPHS_MAXIMUM Character = '\u9FFF'

// CHARACTER_MINIMUM is the smallest value in rune storage.
const CHARACTER_MINIMUM int32 = bits.INTEGER_32_MINIMUM

// CHARACTER_MAXIMUM is the largest value in rune storage.
const CHARACTER_MAXIMUM int32 = bits.INTEGER_32_MAXIMUM

// NAME_SIZE_MINIMUM permits an unknown empty name.
const NAME_SIZE_MINIMUM = 0

// NAME_SIZE_MAXIMUM bounds each Unicode table name.
const NAME_SIZE_MAXIMUM = 64

// CATEGORY_ALIAS_NAME_SIZE_SINGLE is the size of a unified category name.
const CATEGORY_ALIAS_NAME_SIZE_SINGLE = 1

// CATEGORY_ALIAS_NAME_SIZE_MAXIMUM is the size of a specific category name.
const CATEGORY_ALIAS_NAME_SIZE_MAXIMUM = 2

// CATEGORY_ALIAS_NAME_DATA owns each canonical alias so returned text remains immutable.
const CATEGORY_ALIAS_NAME_DATA = "CLMNPSZ" +
	"CcCfCoCsCnLuLlLtLmLoLCMcMeMnNdNlNoPcPdPePfPiPoPsScSkSmSoZlZpZs"

// CATEGORY_ALIAS_NAME_SINGLE_COUNT separates one-byte and two-byte canonical names.
const CATEGORY_ALIAS_NAME_SINGLE_COUNT = 7

// RANGE_TABLES_COUNT_MINIMUM permits an empty table collection.
const RANGE_TABLES_COUNT_MINIMUM = 0

// RANGE_TABLES_COUNT_MAXIMUM bounds a caller-supplied table collection.
const RANGE_TABLES_COUNT_MAXIMUM = 4096

// RANGES_16_COUNT_MINIMUM permits a table without a 16-bit range.
const RANGES_16_COUNT_MINIMUM = 0

// RANGES_32_COUNT_MINIMUM permits a table without a 32-bit range.
const RANGES_32_COUNT_MINIMUM = 0

// RANGE_16_MINIMUM is the first 16-bit code point.
const RANGE_16_MINIMUM uint16 = bits.WORD_16_MINIMUM

// RANGE_16_MAXIMUM is the final 16-bit code point.
const RANGE_16_MAXIMUM uint16 = bits.WORD_16_MAXIMUM

// RANGE_32_MINIMUM is the first code point that needs 32-bit range storage.
const RANGE_32_MINIMUM uint32 = uint32(RANGE_16_MAXIMUM) + 1

// RANGE_32_MAXIMUM is the final Unicode code point.
const RANGE_32_MAXIMUM uint32 = uint32(RUNE_MAX)

// CASE_RANGE_CODE_POINT_MINIMUM is the first Unicode code point.
const CASE_RANGE_CODE_POINT_MINIMUM uint32 = bits.WORD_32_MINIMUM

// RANGE_16_STRIDE_MINIMUM prevents a range from repeating one code point forever.
const RANGE_16_STRIDE_MINIMUM uint16 = 1

// RANGE_16_STRIDE_MAXIMUM admits the complete 16-bit interval.
const RANGE_16_STRIDE_MAXIMUM uint16 = RANGE_16_MAXIMUM

// RANGE_32_STRIDE_MINIMUM prevents a range from repeating one code point forever.
const RANGE_32_STRIDE_MINIMUM uint32 = 1

// RANGE_32_STRIDE_MAXIMUM admits the complete Unicode interval.
const RANGE_32_STRIDE_MAXIMUM uint32 = RANGE_32_MAXIMUM

// LATIN_OFFSET_MINIMUM permits a table without a Latin-1 range.
const LATIN_OFFSET_MINIMUM = 0

// LATIN_OFFSET_MAXIMUM is the largest offset in the native Unicode tables.
const LATIN_OFFSET_MAXIMUM = 11

// CASE_DELTA_COUNT is the upper, lower, and title mapping count.
const CASE_DELTA_COUNT = 3

// CASE_DELTA_MINIMUM is the most negative delta between two Unicode code points.
const CASE_DELTA_MINIMUM int32 = -int32(RUNE_MAX)

// CASE_DELTA_MAXIMUM includes the alternating-case sentinel.
const CASE_DELTA_MAXIMUM int32 = UPPER_LOWER

// SPECIAL_CASE_COUNT_MINIMUM permits the standard mapping without an override.
const SPECIAL_CASE_COUNT_MINIMUM = 0

// SPECIAL_CASE_COUNT_MAXIMUM is the rule count in each supplied language override.
const SPECIAL_CASE_COUNT_MAXIMUM = 4

// UPPER_CASE selects the uppercase mapping.
const UPPER_CASE Case = 0

// LOWER_CASE selects the lowercase mapping.
const LOWER_CASE Case = 1

// TITLE_CASE selects the title-case mapping.
const TITLE_CASE Case = 2

// UPPER_LOWER marks a range whose uppercase and lowercase points alternate.
const UPPER_LOWER int32 = int32(RUNE_MAX) + 1

// TABLE_KIND_CATEGORY selects the Unicode category tables.
const TABLE_KIND_CATEGORY Table_Kind = 0

// TABLE_KIND_SCRIPT selects the Unicode script tables.
const TABLE_KIND_SCRIPT Table_Kind = TABLE_KIND_CATEGORY + 1

// TABLE_KIND_PROPERTY selects the Unicode property tables.
const TABLE_KIND_PROPERTY Table_Kind = TABLE_KIND_SCRIPT + 1

// TABLE_KIND_FOLD_CATEGORY selects the category fold tables.
const TABLE_KIND_FOLD_CATEGORY Table_Kind = TABLE_KIND_PROPERTY + 1

// TABLE_KIND_FOLD_SCRIPT selects the script fold tables.
const TABLE_KIND_FOLD_SCRIPT Table_Kind = TABLE_KIND_FOLD_CATEGORY + 1

// TABLE_KIND_MINIMUM is the first named table family.
const TABLE_KIND_MINIMUM = TABLE_KIND_CATEGORY

// TABLE_KIND_MAXIMUM is the final named table family.
const TABLE_KIND_MAXIMUM = TABLE_KIND_FOLD_SCRIPT

// HEXADECIMAL_BYTE_CHARACTER_COUNT is the encoded character count for one byte.
const HEXADECIMAL_BYTE_CHARACTER_COUNT = 2

// RANGE_BOUND_COUNT keeps encoded offsets aligned with the two range bounds.
const RANGE_BOUND_COUNT = 2

// RANGE_VALUE_COUNT keeps range storage aligned with both bounds and the stride.
const RANGE_VALUE_COUNT = RANGE_BOUND_COUNT + 1

// RANGE_16_BYTE_COUNT is the storage for three 16-bit values.
const RANGE_16_BYTE_COUNT = RANGE_VALUE_COUNT * int(ENCODED_WIDTH_16)

// RANGE_32_BYTE_COUNT is the storage for three 32-bit values.
const RANGE_32_BYTE_COUNT = RANGE_VALUE_COUNT * int(ENCODED_WIDTH_32)

// CASE_RANGE_BYTE_COUNT is the storage for two bounds and three deltas.
const CASE_RANGE_BYTE_COUNT = (RANGE_BOUND_COUNT + CASE_DELTA_COUNT) *
	int(ENCODED_WIDTH_32)

// CASE_ORBIT_PAIR_BYTE_COUNT is the storage for two 16-bit code points.
const CASE_ORBIT_PAIR_BYTE_COUNT = 2 * int(ENCODED_WIDTH_16)

// CASE_RANGE_COUNT is the standard simple-case range count.
const CASE_RANGE_COUNT = 328

// CASE_ORBIT_COUNT is the standard exceptional simple-fold pair count.
const CASE_ORBIT_COUNT = 88

// TABLE_HEADER_BYTE_COUNT stores the 16-bit range count and Latin offset.
const TABLE_HEADER_BYTE_COUNT = 2 * int(ENCODED_WIDTH_16)

// TABLE_RANGE_32_COUNT_BYTE_COUNT stores the 32-bit range count.
const TABLE_RANGE_32_COUNT_BYTE_COUNT = int(ENCODED_WIDTH_16)

// NAMED_TABLE_NAME_SIZE_BYTE_COUNT stores one table-name size.
const NAMED_TABLE_NAME_SIZE_BYTE_COUNT = int(ENCODED_WIDTH_BYTE)

// NAMED_TABLE_DATA_SIZE_BYTE_COUNT stores one encoded table size.
const NAMED_TABLE_DATA_SIZE_BYTE_COUNT = int(ENCODED_WIDTH_32)

// ALIAS_NAME_SIZE_BYTE_COUNT stores an alias or canonical-name size.
const ALIAS_NAME_SIZE_BYTE_COUNT = int(ENCODED_WIDTH_BYTE)

// DATA_POSITION_MINIMUM is the first byte in encoded data.
const DATA_POSITION_MINIMUM = 0

// DATA_POSITION_MAXIMUM admits a final 32-bit value in the largest encoded table.
const DATA_POSITION_MAXIMUM = len(CATEGORY_TABLE_DATA)/HEXADECIMAL_BYTE_CHARACTER_COUNT -
	int(ENCODED_WIDTH_32)

// DATA_COUNT_MINIMUM permits an empty encoded range collection.
const DATA_COUNT_MINIMUM = 0

// DATA_COUNT_MAXIMUM is the largest generated range collection.
const DATA_COUNT_MAXIMUM = RANGES_16_COUNT_MAXIMUM

// ENCODED_DATA_SIZE_MINIMUM permits an empty encoded table.
const ENCODED_DATA_SIZE_MINIMUM = 0

// ENCODED_DATA_SIZE_MAXIMUM is the largest generated encoded table.
const ENCODED_DATA_SIZE_MAXIMUM = len(CATEGORY_TABLE_DATA)

// TABLE_DATA_BYTE_COUNT_MAXIMUM is the largest generated named table.
const TABLE_DATA_BYTE_COUNT_MAXIMUM = 5580

// TABLE_DATA_SIZE_MINIMUM permits an empty named table.
const TABLE_DATA_SIZE_MINIMUM = 0

// TABLE_DATA_SIZE_MAXIMUM is the encoded size of the largest generated named table.
const TABLE_DATA_SIZE_MAXIMUM = TABLE_DATA_BYTE_COUNT_MAXIMUM *
	HEXADECIMAL_BYTE_CHARACTER_COUNT

// TABLE_DATA_SIZE_HOLE excludes an incomplete encoded byte.
const TABLE_DATA_SIZE_HOLE = 1

// ENCODED_NUMBER_MINIMUM is the smallest decoded unsigned value.
const ENCODED_NUMBER_MINIMUM uint64 = uint64(bits.WORD_32_MINIMUM)

// ENCODED_NUMBER_MAXIMUM is the largest decoded unsigned value.
const ENCODED_NUMBER_MAXIMUM uint64 = uint64(bits.WORD_32_MAXIMUM)

// ENCODED_WIDTH_BYTE selects an 8-bit encoded number.
const ENCODED_WIDTH_BYTE Encoded_Width = 1

// ENCODED_WIDTH_16 selects a 16-bit encoded number.
const ENCODED_WIDTH_16 Encoded_Width = 2

// ENCODED_WIDTH_32 selects a 32-bit encoded number.
const ENCODED_WIDTH_32 Encoded_Width = 4

// ENCODED_32_FIRST_BYTE_SHIFT aligns the first byte with big-endian storage.
const ENCODED_32_FIRST_BYTE_SHIFT = (int(ENCODED_WIDTH_32) - int(ENCODED_WIDTH_BYTE)) *
	bits.BIT_COUNT_8_MAXIMUM

// ENCODED_32_SECOND_BYTE_SHIFT aligns the second byte with big-endian storage.
const ENCODED_32_SECOND_BYTE_SHIFT = int(ENCODED_WIDTH_16) * bits.BIT_COUNT_8_MAXIMUM

// HEXADECIMAL_NIBBLE_BIT_COUNT is the bit width of one hexadecimal digit.
const HEXADECIMAL_NIBBLE_BIT_COUNT = bits.BIT_COUNT_8_MAXIMUM / HEXADECIMAL_BYTE_CHARACTER_COUNT

// HEXADECIMAL_LETTER_VALUE_OFFSET converts a hexadecimal letter to its value.
const HEXADECIMAL_LETTER_VALUE_OFFSET = 10

// Character keeps a rune argument separate from an integer argument.
type Character rune

// Character_Invariants covers the complete rune storage domain.
func Character_Invariants(value Character, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), CHARACTER_MINIMUM, CHARACTER_MAXIMUM).
		Ensure()
}

// Case_Character is a valid Unicode code point inside a case range.
type Case_Character rune

// Case_Character_Invariants excludes rune-storage values that no case range can contain.
func Case_Character_Invariants(value Case_Character, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(
			int32(value), int32(CASE_RANGE_CODE_POINT_MINIMUM), int32(RUNE_MAX),
		).
		Ensure()
}

// Boolean gives each property report a coverage identity.
type Boolean bool

// Boolean_Invariants requires both report values.
func Boolean_Invariants(value Boolean, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "A Unicode report is true.").
		Ensure()
}

// Name is a bounded Unicode table or alias name.
type Name string

// Name_Invariants applies the Unicode name size limit.
func Name_Invariants(value Name, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), NAME_SIZE_MINIMUM, NAME_SIZE_MAXIMUM).
		Ensure()
}

// Category_Alias_Name is an empty result or a canonical category name.
type Category_Alias_Name string

// Category_Alias_Name_Invariants admits unknown names and two-character category names.
func Category_Alias_Name_Invariants(
	value Category_Alias_Name, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_3_Int(
			len(value),
			NAME_SIZE_MINIMUM,
			CATEGORY_ALIAS_NAME_SIZE_SINGLE,
			CATEGORY_ALIAS_NAME_SIZE_MAXIMUM,
		).
		Ensure()
}

// Case selects one simple case mapping.
type Case int

// Case_Invariants permits the three Unicode case mappings.
func Case_Invariants(value Case, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Int(int(value), int(UPPER_CASE), int(LOWER_CASE), int(TITLE_CASE)).
		Ensure()
}

// Table_Kind selects one named Unicode table family.
type Table_Kind int

// Table_Kind_Invariants permits the five immutable table families.
func Table_Kind_Invariants(value Table_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), int(TABLE_KIND_MINIMUM), int(TABLE_KIND_MAXIMUM)).
		Ensure()
}

// Data_Position is a byte position in encoded Unicode data.
type Data_Position int

// Data_Position_Invariants bounds a position by the encoded-data budget.
func Data_Position_Invariants(value Data_Position, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), DATA_POSITION_MINIMUM, DATA_POSITION_MAXIMUM).
		Ensure()
}

// Data_Count is a bounded item count in encoded Unicode data.
type Data_Count int

// Data_Count_Invariants keeps an encoded item count inside its data budget.
func Data_Count_Invariants(value Data_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), DATA_COUNT_MINIMUM, DATA_COUNT_MAXIMUM).
		Ensure()
}

// Encoded_Data distinguishes immutable hexadecimal Unicode tables from user text.
type Encoded_Data string

// Encoded_Data_Invariants prevents internal table scans from exceeding data budget.
func Encoded_Data_Invariants(value Encoded_Data, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), ENCODED_DATA_SIZE_MINIMUM, ENCODED_DATA_SIZE_MAXIMUM).
		Ensure()
}

// Table_Data is one hexadecimal range table selected from a named directory.
type Table_Data string

// Table_Data_Invariants bounds decoded bytes by the largest generated named table.
func Table_Data_Invariants(value Table_Data, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Holed_Int(
			len(value), TABLE_DATA_SIZE_MINIMUM, TABLE_DATA_SIZE_MAXIMUM,
			TABLE_DATA_SIZE_HOLE, TABLE_DATA_SIZE_HOLE,
			TABLE_DATA_SIZE_HOLE, TABLE_DATA_SIZE_HOLE,
		).
		Ensure()
	aver.Always(
		len(value)%HEXADECIMAL_BYTE_CHARACTER_COUNT == 0,
		"A hexadecimal range table contains complete encoded bytes.",
	)
}

// Encoded_Width selects the byte width of one encoded unsigned number.
type Encoded_Width int

// Encoded_Width_Invariants permits the three encoded unsigned widths.
func Encoded_Width_Invariants(value Encoded_Width, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Int(
			int(value),
			int(ENCODED_WIDTH_BYTE),
			int(ENCODED_WIDTH_16),
			int(ENCODED_WIDTH_32),
		).
		Ensure()
}

// Encoded_Number holds one decoded unsigned value.
type Encoded_Number uint64

// Encoded_Number_Invariants keeps a decoded number in 32-bit storage.
func Encoded_Number_Invariants(value Encoded_Number, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), ENCODED_NUMBER_MINIMUM, ENCODED_NUMBER_MAXIMUM).
		Ensure()
}

// Code_Point_16 holds one character that fits in 16 bits.
type Code_Point_16 uint16

// Code_Point_16_Invariants covers the complete 16-bit code-point interval.
func Code_Point_16_Invariants(value Code_Point_16, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value), RANGE_16_MINIMUM, RANGE_16_MAXIMUM).
		Ensure()
}

// Code_Point_32 holds one character that needs 32-bit range storage.
type Code_Point_32 uint32

// Code_Point_32_Invariants covers the Unicode code points above 16-bit storage.
func Code_Point_32_Invariants(value Code_Point_32, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), RANGE_32_MINIMUM, RANGE_32_MAXIMUM).
		Ensure()
}

// Range_16_Minimum is the first code point in a 16-bit range.
type Range_16_Minimum uint16

// Range_16_Minimum_Invariants covers the complete 16-bit interval.
func Range_16_Minimum_Invariants(
	value Range_16_Minimum, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value), RANGE_16_MINIMUM, RANGE_16_MAXIMUM).
		Ensure()
}

// Range_16_Maximum is the final code point in a 16-bit range.
type Range_16_Maximum uint16

// Range_16_Maximum_Invariants covers the complete 16-bit interval.
func Range_16_Maximum_Invariants(
	value Range_16_Maximum, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value), RANGE_16_MINIMUM, RANGE_16_MAXIMUM).
		Ensure()
}

// Range_16_Stride is the distance between 16-bit range members.
type Range_16_Stride uint16

// Range_16_Stride_Invariants excludes a zero stride.
func Range_16_Stride_Invariants(
	value Range_16_Stride, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), RANGE_16_STRIDE_MINIMUM, RANGE_16_STRIDE_MAXIMUM,
		).
		Ensure()
}

// Range_16 represents one range whose values fit in 16 bits.
type Range_16 struct {
	// Minimum is the first code point in the range.
	Minimum Range_16_Minimum
	// Maximum is the final code point in the range.
	Maximum Range_16_Maximum
	// Stride is the distance between members.
	Stride Range_16_Stride
}

// Range_16_Invariants verifies each bound, the stride, and the bound order.
func Range_16_Invariants(value Range_16, namespace aver.Namespace) {
	Range_16_Minimum_Invariants(value.Minimum, namespace)
	Range_16_Maximum_Invariants(value.Maximum, namespace)
	Range_16_Stride_Invariants(value.Stride, namespace)
	aver.Always(
		uint16(value.Minimum) <= uint16(value.Maximum),
		"A 16-bit Unicode range minimum does not exceed its maximum.",
	)
}

// Ranges_16 is a bounded collection of 16-bit ranges.
type Ranges_16 []Range_16

// Ranges_16_Invariants bounds the collection by the largest native table.
func Ranges_16_Invariants(value Ranges_16, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), RANGES_16_COUNT_MINIMUM, RANGES_16_COUNT_MAXIMUM,
		).
		Ensure()
}

// Range_32_Minimum is the first code point in a 32-bit range.
type Range_32_Minimum uint32

// Range_32_Minimum_Invariants covers the code points that need 32-bit storage.
func Range_32_Minimum_Invariants(
	value Range_32_Minimum, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), RANGE_32_MINIMUM, RANGE_32_MAXIMUM).
		Ensure()
}

// Range_32_Maximum is the final code point in a 32-bit range.
type Range_32_Maximum uint32

// Range_32_Maximum_Invariants covers the code points that need 32-bit storage.
func Range_32_Maximum_Invariants(
	value Range_32_Maximum, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), RANGE_32_MINIMUM, RANGE_32_MAXIMUM).
		Ensure()
}

// Range_32_Stride is the distance between 32-bit range members.
type Range_32_Stride uint32

// Range_32_Stride_Invariants excludes a zero stride.
func Range_32_Stride_Invariants(
	value Range_32_Stride, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint32(
			uint32(value), RANGE_32_STRIDE_MINIMUM, RANGE_32_STRIDE_MAXIMUM,
		).
		Ensure()
}

// Range_32 represents one range whose values need 32 bits.
type Range_32 struct {
	// Minimum is the first code point in the range.
	Minimum Range_32_Minimum
	// Maximum is the final code point in the range.
	Maximum Range_32_Maximum
	// Stride is the distance between members.
	Stride Range_32_Stride
}

// Range_32_Invariants verifies each bound, the stride, and the bound order.
func Range_32_Invariants(value Range_32, namespace aver.Namespace) {
	Range_32_Minimum_Invariants(value.Minimum, namespace)
	Range_32_Maximum_Invariants(value.Maximum, namespace)
	Range_32_Stride_Invariants(value.Stride, namespace)
	aver.Always(
		uint32(value.Minimum) <= uint32(value.Maximum),
		"A 32-bit Unicode range minimum does not exceed its maximum.",
	)
}

// Ranges_32 is a bounded collection of 32-bit ranges.
type Ranges_32 []Range_32

// Ranges_32_Invariants bounds the collection by the largest native table.
func Ranges_32_Invariants(value Ranges_32, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), RANGES_32_COUNT_MINIMUM, RANGES_32_COUNT_MAXIMUM,
		).
		Ensure()
}

// Latin_Offset counts the 16-bit ranges that end in Latin-1.
type Latin_Offset int

// Latin_Offset_Invariants bounds the offset by the complete 16-bit collection.
func Latin_Offset_Invariants(value Latin_Offset, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), LATIN_OFFSET_MINIMUM, LATIN_OFFSET_MAXIMUM).
		Ensure()
}

// Range_Table stores sorted, non-overlapping 16-bit and 32-bit ranges.
type Range_Table struct {
	// Ranges_16 holds the ranges that fit in 16 bits.
	Ranges_16 Ranges_16
	// Ranges_32 holds the ranges that need 32 bits.
	Ranges_32 Ranges_32
	// Latin_Offset counts Ranges_16 entries that end in Latin-1.
	Latin_Offset Latin_Offset
}

// Range_Table_Invariants composes both range collections and the Latin offset.
func Range_Table_Invariants(value Range_Table, namespace aver.Namespace) {
	Ranges_16_Invariants(value.Ranges_16, namespace)
	Ranges_32_Invariants(value.Ranges_32, namespace)
	Latin_Offset_Invariants(value.Latin_Offset, namespace)
	aver.Always(
		int(value.Latin_Offset) <= len(value.Ranges_16),
		"A Unicode Latin offset does not exceed its 16-bit range count.",
	)
}

// Range_Table_Handle is nonnil caller-owned table storage.
type Range_Table_Handle *Range_Table

// Range_Table_Handle_Invariants rejects missing table storage before reading it.
func Range_Table_Handle_Invariants(value Range_Table_Handle, namespace aver.Namespace) {
	aver.Always(value != nil, "Unicode range table handle exists.")
	aver.Tree(Ranges_16(value.Ranges_16), namespace).
		Range_Int(
			len(value.Ranges_16), RANGES_16_COUNT_MINIMUM, RANGES_16_COUNT_MAXIMUM,
		).
		Ensure()
	aver.Tree(Ranges_32(value.Ranges_32), namespace).
		Range_Int(
			len(value.Ranges_32), RANGES_32_COUNT_MINIMUM, RANGES_32_COUNT_MAXIMUM,
		).
		Ensure()
	aver.Tree(Latin_Offset(value.Latin_Offset), namespace).
		Range_Int(int(value.Latin_Offset), LATIN_OFFSET_MINIMUM, LATIN_OFFSET_MAXIMUM).
		Ensure()
	aver.Always(
		int(value.Latin_Offset) <= len(value.Ranges_16),
		"Unicode range table handle has valid Latin offset.",
	)
}

// Range_Tables is a bounded collection for a union query.
type Range_Tables []*Range_Table

// Range_Tables_Invariants applies the caller-supplied table-count limit.
func Range_Tables_Invariants(value Range_Tables, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), RANGE_TABLES_COUNT_MINIMUM, RANGE_TABLES_COUNT_MAXIMUM).
		Ensure()
}

// Upper_Case_Delta keeps uppercase mapping separate from other case axes.
type Upper_Case_Delta int32

// Upper_Case_Delta_Invariants bounds code-point movement and alternating sentinel.
func Upper_Case_Delta_Invariants(value Upper_Case_Delta, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), CASE_DELTA_MINIMUM, CASE_DELTA_MAXIMUM).
		Ensure()
}

// Lower_Case_Delta keeps lowercase mapping separate from other case axes.
type Lower_Case_Delta int32

// Lower_Case_Delta_Invariants bounds code-point movement and alternating sentinel.
func Lower_Case_Delta_Invariants(value Lower_Case_Delta, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), CASE_DELTA_MINIMUM, CASE_DELTA_MAXIMUM).
		Ensure()
}

// Title_Case_Delta keeps title mapping separate from other case axes.
type Title_Case_Delta int32

// Title_Case_Delta_Invariants bounds code-point movement and alternating sentinel.
func Title_Case_Delta_Invariants(value Title_Case_Delta, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), CASE_DELTA_MINIMUM, CASE_DELTA_MAXIMUM).
		Ensure()
}

// Case_Delta names each mapping because three positions form semantic shape.
type Case_Delta struct {
	// Upper separates uppercase mapping from selector order.
	Upper Upper_Case_Delta
	// Lower separates lowercase mapping from selector order.
	Lower Lower_Case_Delta
	// Title separates title mapping from selector order.
	Title Title_Case_Delta
}

// Case_Delta_Invariants composes all three mapping domains.
func Case_Delta_Invariants(value Case_Delta, namespace aver.Namespace) {
	Upper_Case_Delta_Invariants(value.Upper, namespace)
	Lower_Case_Delta_Invariants(value.Lower, namespace)
	Title_Case_Delta_Invariants(value.Title, namespace)
}

// Case_Range_Minimum is the first code point in a case range.
type Case_Range_Minimum uint32

// Case_Range_Minimum_Invariants covers all valid Unicode code points.
func Case_Range_Minimum_Invariants(
	value Case_Range_Minimum, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint32(
			uint32(value), CASE_RANGE_CODE_POINT_MINIMUM, RANGE_32_MAXIMUM,
		).
		Ensure()
}

// Case_Range_Maximum is the final code point in a case range.
type Case_Range_Maximum uint32

// Case_Range_Maximum_Invariants covers all valid Unicode code points.
func Case_Range_Maximum_Invariants(
	value Case_Range_Maximum, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint32(
			uint32(value), CASE_RANGE_CODE_POINT_MINIMUM, RANGE_32_MAXIMUM,
		).
		Ensure()
}

// Case_Range represents one range for simple case conversion.
type Case_Range struct {
	// Minimum is the first code point in the range.
	Minimum Case_Range_Minimum
	// Maximum is the final code point in the range.
	Maximum Case_Range_Maximum
	// Deltas hold the three case mappings.
	Deltas Case_Delta
}

// Case_Range_Invariants verifies the bounds, their order, and the mapping deltas.
func Case_Range_Invariants(value Case_Range, namespace aver.Namespace) {
	Case_Range_Minimum_Invariants(value.Minimum, namespace)
	Case_Range_Maximum_Invariants(value.Maximum, namespace)
	Case_Delta_Invariants(value.Deltas, namespace)
	aver.Always(
		uint32(value.Minimum) <= uint32(value.Maximum),
		"A Unicode case range minimum does not exceed its maximum.",
	)
}

// First_Special_Case_Minimum gives first rule minimum separate invariant identity.
type First_Special_Case_Minimum uint32

// First_Special_Case_Minimum_Invariants bounds first rule start to Unicode.
func First_Special_Case_Minimum_Invariants(
	value First_Special_Case_Minimum, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), CASE_RANGE_CODE_POINT_MINIMUM, RANGE_32_MAXIMUM).
		Ensure()
}

// First_Special_Case_Maximum gives first rule maximum separate invariant identity.
type First_Special_Case_Maximum uint32

// First_Special_Case_Maximum_Invariants bounds first rule end to Unicode.
func First_Special_Case_Maximum_Invariants(
	value First_Special_Case_Maximum, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), CASE_RANGE_CODE_POINT_MINIMUM, RANGE_32_MAXIMUM).
		Ensure()
}

// First_Special_Case_Upper_Delta separates first upper mapping axis.
type First_Special_Case_Upper_Delta int32

// First_Special_Case_Upper_Delta_Invariants bounds first upper movement.
func First_Special_Case_Upper_Delta_Invariants(
	value First_Special_Case_Upper_Delta, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), CASE_DELTA_MINIMUM, CASE_DELTA_MAXIMUM).
		Ensure()
}

// First_Special_Case_Lower_Delta separates first lower mapping axis.
type First_Special_Case_Lower_Delta int32

// First_Special_Case_Lower_Delta_Invariants bounds first lower movement.
func First_Special_Case_Lower_Delta_Invariants(
	value First_Special_Case_Lower_Delta, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), CASE_DELTA_MINIMUM, CASE_DELTA_MAXIMUM).
		Ensure()
}

// First_Special_Case_Title_Delta separates first title mapping axis.
type First_Special_Case_Title_Delta int32

// First_Special_Case_Title_Delta_Invariants bounds first title movement.
func First_Special_Case_Title_Delta_Invariants(
	value First_Special_Case_Title_Delta, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), CASE_DELTA_MINIMUM, CASE_DELTA_MAXIMUM).
		Ensure()
}

// First_Special_Case_Deltas names first rule mapping axes.
type First_Special_Case_Deltas struct {
	// Upper prevents selector position from implying meaning.
	Upper First_Special_Case_Upper_Delta
	// Lower prevents selector position from implying meaning.
	Lower First_Special_Case_Lower_Delta
	// Title prevents selector position from implying meaning.
	Title First_Special_Case_Title_Delta
}

// First_Special_Case_Deltas_Invariants composes first rule mappings.
func First_Special_Case_Deltas_Invariants(
	value First_Special_Case_Deltas, namespace aver.Namespace,
) {
	First_Special_Case_Upper_Delta_Invariants(value.Upper, namespace)
	First_Special_Case_Lower_Delta_Invariants(value.Lower, namespace)
	First_Special_Case_Title_Delta_Invariants(value.Title, namespace)
}

// First_Special_Case_Range gives first ordered rule structural identity.
type First_Special_Case_Range struct {
	// Minimum prevents range order from hiding behind collection position.
	Minimum First_Special_Case_Minimum
	// Maximum prevents range order from hiding behind collection position.
	Maximum First_Special_Case_Maximum
	// Deltas keep all mapping axes beside range bounds.
	Deltas First_Special_Case_Deltas
}

// First_Special_Case_Range_Invariants composes first ordered rule.
func First_Special_Case_Range_Invariants(
	value First_Special_Case_Range, namespace aver.Namespace,
) {
	First_Special_Case_Minimum_Invariants(value.Minimum, namespace)
	First_Special_Case_Maximum_Invariants(value.Maximum, namespace)
	First_Special_Case_Deltas_Invariants(value.Deltas, namespace)
	aver.Always(
		uint32(value.Minimum) <= uint32(value.Maximum),
		"First special case range minimum does not exceed maximum.",
	)
}

// Second_Special_Case_Minimum gives second rule minimum separate invariant identity.
type Second_Special_Case_Minimum uint32

// Second_Special_Case_Minimum_Invariants bounds second rule start to Unicode.
func Second_Special_Case_Minimum_Invariants(
	value Second_Special_Case_Minimum, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), CASE_RANGE_CODE_POINT_MINIMUM, RANGE_32_MAXIMUM).
		Ensure()
}

// Second_Special_Case_Maximum gives second rule maximum separate invariant identity.
type Second_Special_Case_Maximum uint32

// Second_Special_Case_Maximum_Invariants bounds second rule end to Unicode.
func Second_Special_Case_Maximum_Invariants(
	value Second_Special_Case_Maximum, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), CASE_RANGE_CODE_POINT_MINIMUM, RANGE_32_MAXIMUM).
		Ensure()
}

// Second_Special_Case_Upper_Delta separates second upper mapping axis.
type Second_Special_Case_Upper_Delta int32

// Second_Special_Case_Upper_Delta_Invariants bounds second upper movement.
func Second_Special_Case_Upper_Delta_Invariants(
	value Second_Special_Case_Upper_Delta, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), CASE_DELTA_MINIMUM, CASE_DELTA_MAXIMUM).
		Ensure()
}

// Second_Special_Case_Lower_Delta separates second lower mapping axis.
type Second_Special_Case_Lower_Delta int32

// Second_Special_Case_Lower_Delta_Invariants bounds second lower movement.
func Second_Special_Case_Lower_Delta_Invariants(
	value Second_Special_Case_Lower_Delta, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), CASE_DELTA_MINIMUM, CASE_DELTA_MAXIMUM).
		Ensure()
}

// Second_Special_Case_Title_Delta separates second title mapping axis.
type Second_Special_Case_Title_Delta int32

// Second_Special_Case_Title_Delta_Invariants bounds second title movement.
func Second_Special_Case_Title_Delta_Invariants(
	value Second_Special_Case_Title_Delta, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), CASE_DELTA_MINIMUM, CASE_DELTA_MAXIMUM).
		Ensure()
}

// Second_Special_Case_Deltas names second rule mapping axes.
type Second_Special_Case_Deltas struct {
	// Upper prevents selector position from implying meaning.
	Upper Second_Special_Case_Upper_Delta
	// Lower prevents selector position from implying meaning.
	Lower Second_Special_Case_Lower_Delta
	// Title prevents selector position from implying meaning.
	Title Second_Special_Case_Title_Delta
}

// Second_Special_Case_Deltas_Invariants composes second rule mappings.
func Second_Special_Case_Deltas_Invariants(
	value Second_Special_Case_Deltas, namespace aver.Namespace,
) {
	Second_Special_Case_Upper_Delta_Invariants(value.Upper, namespace)
	Second_Special_Case_Lower_Delta_Invariants(value.Lower, namespace)
	Second_Special_Case_Title_Delta_Invariants(value.Title, namespace)
}

// Second_Special_Case_Range gives second ordered rule structural identity.
type Second_Special_Case_Range struct {
	// Minimum prevents range order from hiding behind collection position.
	Minimum Second_Special_Case_Minimum
	// Maximum prevents range order from hiding behind collection position.
	Maximum Second_Special_Case_Maximum
	// Deltas keep all mapping axes beside range bounds.
	Deltas Second_Special_Case_Deltas
}

// Second_Special_Case_Range_Invariants composes second ordered rule.
func Second_Special_Case_Range_Invariants(
	value Second_Special_Case_Range, namespace aver.Namespace,
) {
	Second_Special_Case_Minimum_Invariants(value.Minimum, namespace)
	Second_Special_Case_Maximum_Invariants(value.Maximum, namespace)
	Second_Special_Case_Deltas_Invariants(value.Deltas, namespace)
	aver.Always(
		uint32(value.Minimum) <= uint32(value.Maximum),
		"Second special case range minimum does not exceed maximum.",
	)
}

// Third_Special_Case_Minimum gives third rule minimum separate invariant identity.
type Third_Special_Case_Minimum uint32

// Third_Special_Case_Minimum_Invariants bounds third rule start to Unicode.
func Third_Special_Case_Minimum_Invariants(
	value Third_Special_Case_Minimum, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), CASE_RANGE_CODE_POINT_MINIMUM, RANGE_32_MAXIMUM).
		Ensure()
}

// Third_Special_Case_Maximum gives third rule maximum separate invariant identity.
type Third_Special_Case_Maximum uint32

// Third_Special_Case_Maximum_Invariants bounds third rule end to Unicode.
func Third_Special_Case_Maximum_Invariants(
	value Third_Special_Case_Maximum, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), CASE_RANGE_CODE_POINT_MINIMUM, RANGE_32_MAXIMUM).
		Ensure()
}

// Third_Special_Case_Upper_Delta separates third upper mapping axis.
type Third_Special_Case_Upper_Delta int32

// Third_Special_Case_Upper_Delta_Invariants bounds third upper movement.
func Third_Special_Case_Upper_Delta_Invariants(
	value Third_Special_Case_Upper_Delta, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), CASE_DELTA_MINIMUM, CASE_DELTA_MAXIMUM).
		Ensure()
}

// Third_Special_Case_Lower_Delta separates third lower mapping axis.
type Third_Special_Case_Lower_Delta int32

// Third_Special_Case_Lower_Delta_Invariants bounds third lower movement.
func Third_Special_Case_Lower_Delta_Invariants(
	value Third_Special_Case_Lower_Delta, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), CASE_DELTA_MINIMUM, CASE_DELTA_MAXIMUM).
		Ensure()
}

// Third_Special_Case_Title_Delta separates third title mapping axis.
type Third_Special_Case_Title_Delta int32

// Third_Special_Case_Title_Delta_Invariants bounds third title movement.
func Third_Special_Case_Title_Delta_Invariants(
	value Third_Special_Case_Title_Delta, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), CASE_DELTA_MINIMUM, CASE_DELTA_MAXIMUM).
		Ensure()
}

// Third_Special_Case_Deltas names third rule mapping axes.
type Third_Special_Case_Deltas struct {
	// Upper prevents selector position from implying meaning.
	Upper Third_Special_Case_Upper_Delta
	// Lower prevents selector position from implying meaning.
	Lower Third_Special_Case_Lower_Delta
	// Title prevents selector position from implying meaning.
	Title Third_Special_Case_Title_Delta
}

// Third_Special_Case_Deltas_Invariants composes third rule mappings.
func Third_Special_Case_Deltas_Invariants(
	value Third_Special_Case_Deltas, namespace aver.Namespace,
) {
	Third_Special_Case_Upper_Delta_Invariants(value.Upper, namespace)
	Third_Special_Case_Lower_Delta_Invariants(value.Lower, namespace)
	Third_Special_Case_Title_Delta_Invariants(value.Title, namespace)
}

// Third_Special_Case_Range gives third ordered rule structural identity.
type Third_Special_Case_Range struct {
	// Minimum prevents range order from hiding behind collection position.
	Minimum Third_Special_Case_Minimum
	// Maximum prevents range order from hiding behind collection position.
	Maximum Third_Special_Case_Maximum
	// Deltas keep all mapping axes beside range bounds.
	Deltas Third_Special_Case_Deltas
}

// Third_Special_Case_Range_Invariants composes third ordered rule.
func Third_Special_Case_Range_Invariants(
	value Third_Special_Case_Range, namespace aver.Namespace,
) {
	Third_Special_Case_Minimum_Invariants(value.Minimum, namespace)
	Third_Special_Case_Maximum_Invariants(value.Maximum, namespace)
	Third_Special_Case_Deltas_Invariants(value.Deltas, namespace)
	aver.Always(
		uint32(value.Minimum) <= uint32(value.Maximum),
		"Third special case range minimum does not exceed maximum.",
	)
}

// Fourth_Special_Case_Minimum gives fourth rule minimum separate invariant identity.
type Fourth_Special_Case_Minimum uint32

// Fourth_Special_Case_Minimum_Invariants bounds fourth rule start to Unicode.
func Fourth_Special_Case_Minimum_Invariants(
	value Fourth_Special_Case_Minimum, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), CASE_RANGE_CODE_POINT_MINIMUM, RANGE_32_MAXIMUM).
		Ensure()
}

// Fourth_Special_Case_Maximum gives fourth rule maximum separate invariant identity.
type Fourth_Special_Case_Maximum uint32

// Fourth_Special_Case_Maximum_Invariants bounds fourth rule end to Unicode.
func Fourth_Special_Case_Maximum_Invariants(
	value Fourth_Special_Case_Maximum, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), CASE_RANGE_CODE_POINT_MINIMUM, RANGE_32_MAXIMUM).
		Ensure()
}

// Fourth_Special_Case_Upper_Delta separates fourth upper mapping axis.
type Fourth_Special_Case_Upper_Delta int32

// Fourth_Special_Case_Upper_Delta_Invariants bounds fourth upper movement.
func Fourth_Special_Case_Upper_Delta_Invariants(
	value Fourth_Special_Case_Upper_Delta, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), CASE_DELTA_MINIMUM, CASE_DELTA_MAXIMUM).
		Ensure()
}

// Fourth_Special_Case_Lower_Delta separates fourth lower mapping axis.
type Fourth_Special_Case_Lower_Delta int32

// Fourth_Special_Case_Lower_Delta_Invariants bounds fourth lower movement.
func Fourth_Special_Case_Lower_Delta_Invariants(
	value Fourth_Special_Case_Lower_Delta, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), CASE_DELTA_MINIMUM, CASE_DELTA_MAXIMUM).
		Ensure()
}

// Fourth_Special_Case_Title_Delta separates fourth title mapping axis.
type Fourth_Special_Case_Title_Delta int32

// Fourth_Special_Case_Title_Delta_Invariants bounds fourth title movement.
func Fourth_Special_Case_Title_Delta_Invariants(
	value Fourth_Special_Case_Title_Delta, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), CASE_DELTA_MINIMUM, CASE_DELTA_MAXIMUM).
		Ensure()
}

// Fourth_Special_Case_Deltas names fourth rule mapping axes.
type Fourth_Special_Case_Deltas struct {
	// Upper prevents selector position from implying meaning.
	Upper Fourth_Special_Case_Upper_Delta
	// Lower prevents selector position from implying meaning.
	Lower Fourth_Special_Case_Lower_Delta
	// Title prevents selector position from implying meaning.
	Title Fourth_Special_Case_Title_Delta
}

// Fourth_Special_Case_Deltas_Invariants composes fourth rule mappings.
func Fourth_Special_Case_Deltas_Invariants(
	value Fourth_Special_Case_Deltas, namespace aver.Namespace,
) {
	Fourth_Special_Case_Upper_Delta_Invariants(value.Upper, namespace)
	Fourth_Special_Case_Lower_Delta_Invariants(value.Lower, namespace)
	Fourth_Special_Case_Title_Delta_Invariants(value.Title, namespace)
}

// Fourth_Special_Case_Range gives fourth ordered rule structural identity.
type Fourth_Special_Case_Range struct {
	// Minimum prevents range order from hiding behind collection position.
	Minimum Fourth_Special_Case_Minimum
	// Maximum prevents range order from hiding behind collection position.
	Maximum Fourth_Special_Case_Maximum
	// Deltas keep all mapping axes beside range bounds.
	Deltas Fourth_Special_Case_Deltas
}

// Fourth_Special_Case_Range_Invariants composes fourth ordered rule.
func Fourth_Special_Case_Range_Invariants(
	value Fourth_Special_Case_Range, namespace aver.Namespace,
) {
	Fourth_Special_Case_Minimum_Invariants(value.Minimum, namespace)
	Fourth_Special_Case_Maximum_Invariants(value.Maximum, namespace)
	Fourth_Special_Case_Deltas_Invariants(value.Deltas, namespace)
	aver.Always(
		uint32(value.Minimum) <= uint32(value.Maximum),
		"Fourth special case range minimum does not exceed maximum.",
	)
}

// Special_Case_Count selects active prefix from four named rules.
type Special_Case_Count int

// Special_Case_Count_Invariants bounds work to complete language override.
func Special_Case_Count_Invariants(value Special_Case_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), SPECIAL_CASE_COUNT_MINIMUM, SPECIAL_CASE_COUNT_MAXIMUM).
		Ensure()
}

// Special_Case names bounded rules because four positions form fixed shape.
type Special_Case struct {
	// Count keeps zero value on standard mapping.
	Count Special_Case_Count
	// First retains first ordered override rule.
	First First_Special_Case_Range
	// Second retains second ordered override rule.
	Second Second_Special_Case_Range
	// Third retains third ordered override rule.
	Third Third_Special_Case_Range
	// Fourth retains fourth ordered override rule.
	Fourth Fourth_Special_Case_Range
}

// Special_Case_Invariants composes active count and every fixed rule.
func Special_Case_Invariants(value Special_Case, namespace aver.Namespace) {
	Special_Case_Count_Invariants(value.Count, namespace)
	First_Special_Case_Range_Invariants(value.First, namespace)
	Second_Special_Case_Range_Invariants(value.Second, namespace)
	Third_Special_Case_Range_Invariants(value.Third, namespace)
	Fourth_Special_Case_Range_Invariants(value.Fourth, namespace)
}

// Special_Case_Destination is nonnil caller-owned override storage.
type Special_Case_Destination *Special_Case

// Special_Case_Destination_Invariants rejects missing storage before writing it.
func Special_Case_Destination_Invariants(
	value Special_Case_Destination, namespace aver.Namespace,
) {
	aver.Always(value != nil, "Unicode special case destination exists.")
	aver.Tree(Special_Case_Count(value.Count), namespace).
		Range_Int(
			int(value.Count), SPECIAL_CASE_COUNT_MINIMUM, SPECIAL_CASE_COUNT_MAXIMUM,
		).
		Ensure()
	First_Special_Case_Range_Invariants(value.First, namespace)
	Second_Special_Case_Range_Invariants(value.Second, namespace)
	Third_Special_Case_Range_Invariants(value.Third, namespace)
	Fourth_Special_Case_Range_Invariants(value.Fourth, namespace)
}

// Special_Case_Of copies bounded rules so no caller collection can escape.
func Special_Case_Of(
	count Special_Case_Count,
	first Case_Range,
	second Case_Range,
	third Case_Range,
	fourth Case_Range,
) (special Special_Case) {
	defer func() { Special_Case_Invariants(special, "special_case_of.special") }()
	Special_Case_Count_Invariants(count, "special_case_of.count")
	Case_Range_Invariants(first, "special_case_of.first")
	Case_Range_Invariants(second, "special_case_of.second")
	Case_Range_Invariants(third, "special_case_of.third")
	Case_Range_Invariants(fourth, "special_case_of.fourth")
	return Special_Case{
		Count: count,
		First: First_Special_Case_Range{
			Minimum: First_Special_Case_Minimum(first.Minimum),
			Maximum: First_Special_Case_Maximum(first.Maximum),
			Deltas: First_Special_Case_Deltas{
				Upper: First_Special_Case_Upper_Delta(first.Deltas.Upper),
				Lower: First_Special_Case_Lower_Delta(first.Deltas.Lower),
				Title: First_Special_Case_Title_Delta(first.Deltas.Title),
			},
		},
		Second: Second_Special_Case_Range{
			Minimum: Second_Special_Case_Minimum(second.Minimum),
			Maximum: Second_Special_Case_Maximum(second.Maximum),
			Deltas: Second_Special_Case_Deltas{
				Upper: Second_Special_Case_Upper_Delta(second.Deltas.Upper),
				Lower: Second_Special_Case_Lower_Delta(second.Deltas.Lower),
				Title: Second_Special_Case_Title_Delta(second.Deltas.Title),
			},
		},
		Third: Third_Special_Case_Range{
			Minimum: Third_Special_Case_Minimum(third.Minimum),
			Maximum: Third_Special_Case_Maximum(third.Maximum),
			Deltas: Third_Special_Case_Deltas{
				Upper: Third_Special_Case_Upper_Delta(third.Deltas.Upper),
				Lower: Third_Special_Case_Lower_Delta(third.Deltas.Lower),
				Title: Third_Special_Case_Title_Delta(third.Deltas.Title),
			},
		},
		Fourth: Fourth_Special_Case_Range{
			Minimum: Fourth_Special_Case_Minimum(fourth.Minimum),
			Maximum: Fourth_Special_Case_Maximum(fourth.Maximum),
			Deltas: Fourth_Special_Case_Deltas{
				Upper: Fourth_Special_Case_Upper_Delta(fourth.Deltas.Upper),
				Lower: Fourth_Special_Case_Lower_Delta(fourth.Deltas.Lower),
				Title: Fourth_Special_Case_Title_Delta(fourth.Deltas.Title),
			},
		},
	}
}

// Is reports whether a character is in a range table.
func Is(table Range_Table_Handle, character Character) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "is.yes") }()
	Character_Invariants(character, "is.character")
	Range_Table_Handle_Invariants(table, "is.table")
	if character >= 0 {
		if uint32(character) <= uint32(RANGE_16_MAXIMUM) {
			for _, one := range table.Ranges_16 {
				Range_16_Invariants(one, "is.range_16")
				if range_16_contains(one, Code_Point_16(character)) {
					return true
				}
			}
		}
	}
	if character < Character(RANGE_32_MINIMUM) {
		return false
	}
	if character > RUNE_MAX {
		return false
	}
	for _, one := range table.Ranges_32 {
		Range_32_Invariants(one, "is.range_32")
		if range_32_contains(one, Code_Point_32(character)) {
			return true
		}
	}
	return false
}

// Is_One_Of reports whether a character is in one table of the collection.
func Is_One_Of(tables Range_Tables, character Character) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "is_one_of.yes") }()
	Range_Tables_Invariants(tables, "is_one_of.tables")
	Character_Invariants(character, "is_one_of.character")
	for _, table := range tables {
		if Is(Range_Table_Handle(table), character) {
			return true
		}
	}
	return false
}

// In reports whether a character is in one table of the collection.
func In(character Character, tables Range_Tables) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "in.yes") }()
	Character_Invariants(character, "in.character")
	Range_Tables_Invariants(tables, "in.tables")
	for _, table := range tables {
		if Is(Range_Table_Handle(table), character) {
			return true
		}
	}
	return false
}

// Is_Control reports whether a character is in the Unicode control category.
func Is_Control(character Character) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "is_control.yes") }()
	Character_Invariants(character, "is_control.character")
	return named_table_contains(CATEGORY_TABLE_DATA, "Cc", character)
}

// Is_Digit reports whether a character is a Unicode decimal digit.
func Is_Digit(character Character) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "is_digit.yes") }()
	Character_Invariants(character, "is_digit.character")
	return named_table_contains(CATEGORY_TABLE_DATA, "Nd", character)
}

// Is_Graphic reports whether Unicode defines a character as graphic.
func Is_Graphic(character Character) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "is_graphic.yes") }()
	Character_Invariants(character, "is_graphic.character")
	if character >= CJK_UNIFIED_IDEOGRAPHS_MINIMUM {
		if character <= CJK_UNIFIED_IDEOGRAPHS_MAXIMUM {
			return true
		}
	}
	return binary_range_table_contains(
		GRAPHIC_RANGES_16_DATA, GRAPHIC_RANGES_32_DATA, character,
	)
}

// Is_Print reports whether Go defines a character as printable.
func Is_Print(character Character) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "is_print.yes") }()
	Character_Invariants(character, "is_print.character")
	if character >= CJK_UNIFIED_IDEOGRAPHS_MINIMUM {
		if character <= CJK_UNIFIED_IDEOGRAPHS_MAXIMUM {
			return true
		}
	}
	return binary_range_table_contains(
		PRINT_RANGES_16_DATA, PRINT_RANGES_32_DATA, character,
	)
}

// Is_Letter reports whether a character is a Unicode letter.
func Is_Letter(character Character) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "is_letter.yes") }()
	Character_Invariants(character, "is_letter.character")
	return named_table_contains(CATEGORY_TABLE_DATA, "L", character)
}

// Is_Lower reports whether a character is a lowercase letter.
func Is_Lower(character Character) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "is_lower.yes") }()
	Character_Invariants(character, "is_lower.character")
	return named_table_contains(CATEGORY_TABLE_DATA, "Ll", character)
}

// Is_Mark reports whether a character is a Unicode mark.
func Is_Mark(character Character) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "is_mark.yes") }()
	Character_Invariants(character, "is_mark.character")
	return named_table_contains(CATEGORY_TABLE_DATA, "M", character)
}

// Is_Number reports whether a character is a Unicode number.
func Is_Number(character Character) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "is_number.yes") }()
	Character_Invariants(character, "is_number.character")
	return named_table_contains(CATEGORY_TABLE_DATA, "N", character)
}

// Is_Punctuation reports whether a character is Unicode punctuation.
func Is_Punctuation(character Character) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "is_punctuation.yes") }()
	Character_Invariants(character, "is_punctuation.character")
	return named_table_contains(CATEGORY_TABLE_DATA, "P", character)
}

// Is_Space reports whether a character has the Unicode White_Space property.
func Is_Space(character Character) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "is_space.yes") }()
	Character_Invariants(character, "is_space.character")
	return named_table_contains(PROPERTY_TABLE_DATA, "White_Space", character)
}

// Is_Symbol reports whether a character is a Unicode symbol.
func Is_Symbol(character Character) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "is_symbol.yes") }()
	Character_Invariants(character, "is_symbol.character")
	return named_table_contains(CATEGORY_TABLE_DATA, "S", character)
}

// Is_Title reports whether a character is a title-case letter.
func Is_Title(character Character) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "is_title.yes") }()
	Character_Invariants(character, "is_title.character")
	return named_table_contains(CATEGORY_TABLE_DATA, "Lt", character)
}

// Is_Upper reports whether a character is an uppercase letter.
func Is_Upper(character Character) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "is_upper.yes") }()
	Character_Invariants(character, "is_upper.character")
	return named_table_contains(CATEGORY_TABLE_DATA, "Lu", character)
}

// Is_Category reports membership in a named Unicode category table.
func Is_Category(character Character, name Name) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "is_category.yes") }()
	Character_Invariants(character, "is_category.character")
	Name_Invariants(name, "is_category.name")
	return named_table_contains(CATEGORY_TABLE_DATA, name, character)
}

// Is_Script reports membership in a named Unicode script table.
func Is_Script(character Character, name Name) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "is_script.yes") }()
	Character_Invariants(character, "is_script.character")
	Name_Invariants(name, "is_script.name")
	return named_table_contains(SCRIPT_TABLE_DATA, name, character)
}

// Is_Property reports membership in a named Unicode property table.
func Is_Property(character Character, name Name) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "is_property.yes") }()
	Character_Invariants(character, "is_property.character")
	Name_Invariants(name, "is_property.name")
	return named_table_contains(PROPERTY_TABLE_DATA, name, character)
}

// Is_Fold_Category reports membership in a named category fold table.
func Is_Fold_Category(character Character, name Name) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "is_fold_category.yes") }()
	Character_Invariants(character, "is_fold_category.character")
	Name_Invariants(name, "is_fold_category.name")
	return named_table_contains(FOLD_CATEGORY_TABLE_DATA, name, character)
}

// Is_Fold_Script reports membership in a named script fold table.
func Is_Fold_Script(character Character, name Name) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "is_fold_script.yes") }()
	Character_Invariants(character, "is_fold_script.character")
	Name_Invariants(name, "is_fold_script.name")
	return named_table_contains(FOLD_SCRIPT_TABLE_DATA, name, character)
}

// Named_Table decodes one named Unicode table into caller-owned range storage.
func Named_Table(
	kind Table_Kind, name Name, ranges_16 Ranges_16, ranges_32 Ranges_32,
) (table Range_Table, found Boolean) {
	defer func() {
		Range_Table_Invariants(table, "named_table.table")
		Boolean_Invariants(found, "named_table.found")
	}()
	Table_Kind_Invariants(kind, "named_table.kind")
	Name_Invariants(name, "named_table.name")
	Ranges_16_Invariants(ranges_16, "named_table.ranges_16")
	Ranges_32_Invariants(ranges_32, "named_table.ranges_32")
	data := CATEGORY_TABLE_DATA
	switch kind {
	case TABLE_KIND_CATEGORY:
		data = CATEGORY_TABLE_DATA
	case TABLE_KIND_SCRIPT:
		data = SCRIPT_TABLE_DATA
	case TABLE_KIND_PROPERTY:
		data = PROPERTY_TABLE_DATA
	case TABLE_KIND_FOLD_CATEGORY:
		data = FOLD_CATEGORY_TABLE_DATA
	case TABLE_KIND_FOLD_SCRIPT:
		data = FOLD_SCRIPT_TABLE_DATA
	}
	table_data, known := named_table_data(data, name)
	if !known {
		return Range_Table{}, false
	}
	return decode_range_table(table_data, ranges_16, ranges_32), true
}

// Category_Alias returns the canonical category name for one alias.
func Category_Alias(
	name Name,
) (canonical Category_Alias_Name, found Boolean) {
	defer func() {
		Category_Alias_Name_Invariants(canonical, "category_alias.canonical")
		Boolean_Invariants(found, "category_alias.found")
	}()
	Name_Invariants(name, "category_alias.name")
	return category_alias_data(name)
}

// To applies one simple Unicode case mapping.
func To(case_value Case, character Character) (mapped Character) {
	defer func() { Character_Invariants(mapped, "to.mapped") }()
	Case_Invariants(case_value, "to.case")
	Character_Invariants(character, "to.character")
	case_range, found := encoded_case_range(CASE_RANGE_DATA, character)
	if !found {
		return character
	}
	return Character(convert_case_range(
		case_value, Case_Character(character), case_range,
	))
}

// To_Upper maps a character to uppercase.
func To_Upper(character Character) (mapped Character) {
	defer func() { Character_Invariants(mapped, "to_upper.mapped") }()
	Character_Invariants(character, "to_upper.character")
	if character <= ASCII_MAX {
		if ASCII_LOWER_FIRST <= character {
			if character <= ASCII_LOWER_FINAL {
				return character - ASCII_CASE_DELTA
			}
		}
		return character
	}
	case_range, found := encoded_case_range(CASE_RANGE_DATA, character)
	if !found {
		return character
	}
	one := case_range
	delta := int32(one.Deltas.Upper)
	if delta > int32(RUNE_MAX) {
		offset := character - Character(one.Minimum)
		offset = offset &^ 1
		return Character(one.Minimum) + offset
	}
	return character + Character(delta)
}

// To_Lower maps a character to lowercase.
func To_Lower(character Character) (mapped Character) {
	defer func() { Character_Invariants(mapped, "to_lower.mapped") }()
	Character_Invariants(character, "to_lower.character")
	if character <= ASCII_MAX {
		if ASCII_UPPER_FIRST <= character {
			if character <= ASCII_UPPER_FINAL {
				return character + ASCII_CASE_DELTA
			}
		}
		return character
	}
	case_range, found := encoded_case_range(CASE_RANGE_DATA, character)
	if !found {
		return character
	}
	one := case_range
	delta := int32(one.Deltas.Lower)
	if delta > int32(RUNE_MAX) {
		offset := character - Character(one.Minimum)
		offset = offset &^ 1
		offset |= Character(LOWER_CASE & 1)
		return Character(one.Minimum) + offset
	}
	return character + Character(delta)
}

// To_Title maps a character to title case.
func To_Title(character Character) (mapped Character) {
	defer func() { Character_Invariants(mapped, "to_title.mapped") }()
	Character_Invariants(character, "to_title.character")
	if character <= ASCII_MAX {
		if ASCII_LOWER_FIRST <= character {
			if character <= ASCII_LOWER_FINAL {
				return character - ASCII_CASE_DELTA
			}
		}
		return character
	}
	case_range, found := encoded_case_range(CASE_RANGE_DATA, character)
	if !found {
		return character
	}
	one := case_range
	delta := int32(one.Deltas.Title)
	if delta > int32(RUNE_MAX) {
		offset := character - Character(one.Minimum)
		offset = offset &^ 1
		return Character(one.Minimum) + offset
	}
	return character + Character(delta)
}

// Special_Case_To_Upper applies a language override before the uppercase mapping.
func Special_Case_To_Upper(
	special Special_Case, character Character,
) (mapped Character) {
	defer func() { Character_Invariants(mapped, "special_case_to_upper.mapped") }()
	Special_Case_Invariants(special, "special_case_to_upper.special")
	Character_Invariants(character, "special_case_to_upper.character")
	return special_case_to(special, UPPER_CASE, character)
}

// Special_Case_To_Lower applies a language override before the lowercase mapping.
func Special_Case_To_Lower(
	special Special_Case, character Character,
) (mapped Character) {
	defer func() { Character_Invariants(mapped, "special_case_to_lower.mapped") }()
	Special_Case_Invariants(special, "special_case_to_lower.special")
	Character_Invariants(character, "special_case_to_lower.character")
	return special_case_to(special, LOWER_CASE, character)
}

// Special_Case_To_Title applies a language override before the title-case mapping.
func Special_Case_To_Title(
	special Special_Case, character Character,
) (mapped Character) {
	defer func() { Character_Invariants(mapped, "special_case_to_title.mapped") }()
	Special_Case_Invariants(special, "special_case_to_title.special")
	Character_Invariants(character, "special_case_to_title.character")
	return special_case_to(special, TITLE_CASE, character)
}

// Turkish_Case writes complete Turkish rules into caller-owned structural storage.
func Turkish_Case(special Special_Case_Destination) {
	Special_Case_Destination_Invariants(special, "turkish_case.special")
	*special = Special_Case_Of(
		SPECIAL_CASE_COUNT_MAXIMUM,
		Case_Range{
			Minimum: Case_Range_Minimum(LATIN_CAPITAL_I),
			Maximum: Case_Range_Maximum(LATIN_CAPITAL_I),
			Deltas: Case_Delta{
				Upper: 0,
				Lower: Lower_Case_Delta(LATIN_SMALL_DOTLESS_I - LATIN_CAPITAL_I),
				Title: 0,
			},
		},
		Case_Range{
			Minimum: Case_Range_Minimum(LATIN_SMALL_I),
			Maximum: Case_Range_Maximum(LATIN_SMALL_I),
			Deltas: Case_Delta{
				Upper: Upper_Case_Delta(LATIN_CAPITAL_I_WITH_DOT - LATIN_SMALL_I),
				Lower: 0,
				Title: Title_Case_Delta(LATIN_CAPITAL_I_WITH_DOT - LATIN_SMALL_I),
			},
		},
		Case_Range{
			Minimum: Case_Range_Minimum(LATIN_CAPITAL_I_WITH_DOT),
			Maximum: Case_Range_Maximum(LATIN_CAPITAL_I_WITH_DOT),
			Deltas: Case_Delta{
				Upper: 0,
				Lower: Lower_Case_Delta(LATIN_SMALL_I - LATIN_CAPITAL_I_WITH_DOT),
				Title: 0,
			},
		},
		Case_Range{
			Minimum: Case_Range_Minimum(LATIN_SMALL_DOTLESS_I),
			Maximum: Case_Range_Maximum(LATIN_SMALL_DOTLESS_I),
			Deltas: Case_Delta{
				Upper: Upper_Case_Delta(LATIN_CAPITAL_I - LATIN_SMALL_DOTLESS_I),
				Lower: 0,
				Title: Title_Case_Delta(LATIN_CAPITAL_I - LATIN_SMALL_DOTLESS_I),
			},
		},
	)
}

// Azeri_Case writes the Azerbaijani language-specific case rules into caller storage.
func Azeri_Case(special Special_Case_Destination) {
	Special_Case_Destination_Invariants(special, "azeri_case.special")
	Turkish_Case(special)
}

// Simple_Fold returns the next character in a simple case-fold orbit.
func Simple_Fold(character Character) (folded Character) {
	defer func() { Character_Invariants(folded, "simple_fold.folded") }()
	Character_Invariants(character, "simple_fold.character")
	if character < 0 {
		return character
	}
	if character > RUNE_MAX {
		return character
	}
	if character <= ASCII_MAX {
		position := int(character) * int(ENCODED_WIDTH_16)
		return Character(
			uint16(ASCII_FOLD_DATA[position])<<bits.BIT_COUNT_8_MAXIMUM |
				uint16(ASCII_FOLD_DATA[position+1]),
		)
	}
	orbit_lower := 0
	orbit_upper := len(CASE_ORBIT_DATA) / CASE_ORBIT_PAIR_BYTE_COUNT
	for orbit_lower < orbit_upper {
		middle := int(uint(orbit_lower+orbit_upper) >> 1)
		position := middle * CASE_ORBIT_PAIR_BYTE_COUNT
		from := Character(
			uint16(CASE_ORBIT_DATA[position])<<bits.BIT_COUNT_8_MAXIMUM |
				uint16(CASE_ORBIT_DATA[position+1]),
		)
		if from < character {
			orbit_lower = middle + 1
			continue
		}
		orbit_upper = middle
	}
	if orbit_lower < len(CASE_ORBIT_DATA)/CASE_ORBIT_PAIR_BYTE_COUNT {
		position := orbit_lower * CASE_ORBIT_PAIR_BYTE_COUNT
		from := Character(
			uint16(CASE_ORBIT_DATA[position])<<bits.BIT_COUNT_8_MAXIMUM |
				uint16(CASE_ORBIT_DATA[position+1]),
		)
		if from == character {
			to_position := position + int(ENCODED_WIDTH_16)
			return Character(
				uint16(CASE_ORBIT_DATA[to_position])<<bits.BIT_COUNT_8_MAXIMUM |
					uint16(CASE_ORBIT_DATA[to_position+1]),
			)
		}
	}
	case_range, range_found := encoded_case_range(CASE_RANGE_DATA, character)
	if !range_found {
		return character
	}
	one := case_range
	lower_delta := int32(one.Deltas.Lower)
	lower := character + Character(lower_delta)
	if lower_delta > int32(RUNE_MAX) {
		offset := character - Character(one.Minimum)
		offset = offset &^ 1
		offset |= Character(LOWER_CASE & 1)
		lower = Character(one.Minimum) + offset
	}
	if lower != character {
		return lower
	}
	upper_delta := int32(one.Deltas.Upper)
	if upper_delta > int32(RUNE_MAX) {
		offset := character - Character(one.Minimum)
		offset = offset &^ 1
		return Character(one.Minimum) + offset
	}
	return character + Character(upper_delta)
}

func range_16_contains(one Range_16, character Code_Point_16) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "range_16_contains.yes") }()
	Range_16_Invariants(one, "range_16_contains.range")
	Code_Point_16_Invariants(character, "range_16_contains.character")
	if uint16(character) < uint16(one.Minimum) {
		return false
	}
	if uint16(character) > uint16(one.Maximum) {
		return false
	}
	distance := uint16(character) - uint16(one.Minimum)
	return Boolean(distance%uint16(one.Stride) == 0)
}

func range_32_contains(one Range_32, character Code_Point_32) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "range_32_contains.yes") }()
	Range_32_Invariants(one, "range_32_contains.range")
	Code_Point_32_Invariants(character, "range_32_contains.character")
	if uint32(character) < uint32(one.Minimum) {
		return false
	}
	if uint32(character) > uint32(one.Maximum) {
		return false
	}
	distance := uint32(character) - uint32(one.Minimum)
	return Boolean(distance%uint32(one.Stride) == 0)
}

func special_case_to(
	special Special_Case, case_value Case, character Character,
) (mapped Character) {
	defer func() { Character_Invariants(mapped, "special_case_to.mapped") }()
	Special_Case_Invariants(special, "special_case_to.special")
	Case_Invariants(case_value, "special_case_to.case")
	Character_Invariants(character, "special_case_to.character")
	ranges := [...]Case_Range{
		{
			Minimum: Case_Range_Minimum(special.First.Minimum),
			Maximum: Case_Range_Maximum(special.First.Maximum),
			Deltas: Case_Delta{
				Upper: Upper_Case_Delta(special.First.Deltas.Upper),
				Lower: Lower_Case_Delta(special.First.Deltas.Lower),
				Title: Title_Case_Delta(special.First.Deltas.Title),
			},
		},
		{
			Minimum: Case_Range_Minimum(special.Second.Minimum),
			Maximum: Case_Range_Maximum(special.Second.Maximum),
			Deltas: Case_Delta{
				Upper: Upper_Case_Delta(special.Second.Deltas.Upper),
				Lower: Lower_Case_Delta(special.Second.Deltas.Lower),
				Title: Title_Case_Delta(special.Second.Deltas.Title),
			},
		},
		{
			Minimum: Case_Range_Minimum(special.Third.Minimum),
			Maximum: Case_Range_Maximum(special.Third.Maximum),
			Deltas: Case_Delta{
				Upper: Upper_Case_Delta(special.Third.Deltas.Upper),
				Lower: Lower_Case_Delta(special.Third.Deltas.Lower),
				Title: Title_Case_Delta(special.Third.Deltas.Title),
			},
		},
		{
			Minimum: Case_Range_Minimum(special.Fourth.Minimum),
			Maximum: Case_Range_Maximum(special.Fourth.Maximum),
			Deltas: Case_Delta{
				Upper: Upper_Case_Delta(special.Fourth.Deltas.Upper),
				Lower: Lower_Case_Delta(special.Fourth.Deltas.Lower),
				Title: Title_Case_Delta(special.Fourth.Deltas.Title),
			},
		},
	}
	for _, one := range ranges[:int(special.Count)] {
		Case_Range_Invariants(one, "special_case_to.range")
		if character < Character(one.Minimum) {
			continue
		}
		if character > Character(one.Maximum) {
			continue
		}
		return Character(convert_case_range(
			case_value, Case_Character(character), one,
		))
	}
	return To(case_value, character)
}

func convert_case_range(
	case_value Case, character Case_Character, case_range Case_Range,
) (mapped Case_Character) {
	defer func() {
		Case_Character_Invariants(mapped, "convert_case_range.mapped")
	}()
	Case_Invariants(case_value, "convert_case_range.case")
	Case_Character_Invariants(character, "convert_case_range.character")
	Case_Range_Invariants(case_range, "convert_case_range.range")
	delta := int32(case_range.Deltas.Upper)
	switch case_value {
	case LOWER_CASE:
		delta = int32(case_range.Deltas.Lower)
	case TITLE_CASE:
		delta = int32(case_range.Deltas.Title)
	}
	if delta > int32(RUNE_MAX) {
		offset := character - Case_Character(case_range.Minimum)
		offset = offset &^ 1
		offset |= Case_Character(case_value & 1)
		return Case_Character(case_range.Minimum) + offset
	}
	return character + Case_Character(delta)
}

func named_table_contains(
	data Encoded_Data, name Name, character Character,
) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "named_table_contains.yes") }()
	Encoded_Data_Invariants(data, "named_table_contains.data")
	Name_Invariants(name, "named_table_contains.name")
	Character_Invariants(character, "named_table_contains.character")
	table_data, found := named_table_data(data, name)
	if !found {
		return false
	}
	return encoded_range_table_contains(table_data, character)
}

func named_table_data(
	data Encoded_Data, name Name,
) (table_data Table_Data, found Boolean) {
	defer func() {
		Table_Data_Invariants(table_data, "named_table_data.table_data")
		Boolean_Invariants(found, "named_table_data.found")
	}()
	Encoded_Data_Invariants(data, "named_table_data.data")
	Name_Invariants(name, "named_table_data.name")
	byte_count := len(data) / HEXADECIMAL_BYTE_CHARACTER_COUNT
	for position := 0; position < byte_count; {
		name_size := int(encoded_number(
			data, Data_Position(position), ENCODED_WIDTH_BYTE,
		))
		name_position := position + NAMED_TABLE_NAME_SIZE_BYTE_COUNT
		size_position := name_position + name_size
		table_size := int(encoded_number(
			data, Data_Position(size_position), ENCODED_WIDTH_32,
		))
		table_position := size_position + NAMED_TABLE_DATA_SIZE_BYTE_COUNT
		if encoded_name_equal(
			data, Data_Position(name_position), Data_Count(name_size), name,
		) {
			start := table_position * HEXADECIMAL_BYTE_CHARACTER_COUNT
			end := (table_position + table_size) * HEXADECIMAL_BYTE_CHARACTER_COUNT
			return Table_Data(data[start:end]), true
		}
		position = table_position + table_size
	}
	return "", false
}

func category_alias_data(
	name Name,
) (canonical Category_Alias_Name, found Boolean) {
	defer func() {
		Category_Alias_Name_Invariants(canonical, "category_alias_data.canonical")
		Boolean_Invariants(found, "category_alias_data.found")
	}()
	Name_Invariants(name, "category_alias_data.name")
	byte_count := len(CATEGORY_ALIAS_DATA) / HEXADECIMAL_BYTE_CHARACTER_COUNT
	for position := 0; position < byte_count; {
		name_size := int(encoded_number(
			CATEGORY_ALIAS_DATA, Data_Position(position), ENCODED_WIDTH_BYTE,
		))
		name_position := position + ALIAS_NAME_SIZE_BYTE_COUNT
		canonical_size_position := name_position + name_size
		canonical_size := int(encoded_number(
			CATEGORY_ALIAS_DATA,
			Data_Position(canonical_size_position),
			ENCODED_WIDTH_BYTE,
		))
		canonical_position := canonical_size_position + ALIAS_NAME_SIZE_BYTE_COUNT
		if encoded_name_equal(
			CATEGORY_ALIAS_DATA,
			Data_Position(name_position),
			Data_Count(name_size),
			name,
		) {
			first := byte(encoded_number(
				CATEGORY_ALIAS_DATA,
				Data_Position(canonical_position),
				ENCODED_WIDTH_BYTE,
			))
			name_data := CATEGORY_ALIAS_NAME_DATA
			if canonical_size == CATEGORY_ALIAS_NAME_SIZE_SINGLE {
				single_count := CATEGORY_ALIAS_NAME_SINGLE_COUNT
				position_index := 0
				for position_index < single_count {
					if name_data[position_index] == first {
						name_end := position_index + 1
						name_text := name_data[position_index:name_end]
						return Category_Alias_Name(name_text), true
					}
					position_index++
				}
				return "", false
			}
			second := byte(encoded_number(
				CATEGORY_ALIAS_DATA,
				Data_Position(canonical_position+1),
				ENCODED_WIDTH_BYTE,
			))
			name_count := len(name_data)
			position_index := CATEGORY_ALIAS_NAME_SINGLE_COUNT
			for position_index < name_count {
				if name_data[position_index] == first {
					if name_data[position_index+1] != second {
						position_index += 2
						continue
					}
					name_end := position_index + 2
					name_text := name_data[position_index:name_end]
					return Category_Alias_Name(name_text), true
				}
				position_index += 2
			}
			return "", false
		}
		position = canonical_position + canonical_size
	}
	return "", false
}

func decode_range_table(
	data Table_Data, ranges_16 Ranges_16, ranges_32 Ranges_32,
) (table Range_Table) {
	defer func() { Range_Table_Invariants(table, "decode_range_table.table") }()
	Table_Data_Invariants(data, "decode_range_table.data")
	Ranges_16_Invariants(ranges_16, "decode_range_table.ranges_16")
	Ranges_32_Invariants(ranges_32, "decode_range_table.ranges_32")
	range_16_count := int(encoded_number(Encoded_Data(data), 0, ENCODED_WIDTH_16))
	aver.Always(
		range_16_count <= len(ranges_16),
		"Caller storage holds every decoded 16-bit Unicode range.",
	)
	table.Latin_Offset = Latin_Offset(encoded_number(
		Encoded_Data(data), Data_Position(ENCODED_WIDTH_16), ENCODED_WIDTH_16,
	))
	table.Ranges_16 = ranges_16[:range_16_count]
	position := TABLE_HEADER_BYTE_COUNT
	for index := range table.Ranges_16 {
		table.Ranges_16[index] = Range_16{
			Minimum: Range_16_Minimum(encoded_number(
				Encoded_Data(data), Data_Position(position), ENCODED_WIDTH_16,
			)),
			Maximum: Range_16_Maximum(encoded_number(
				Encoded_Data(data),
				Data_Position(position+int(ENCODED_WIDTH_16)),
				ENCODED_WIDTH_16,
			)),
			Stride: Range_16_Stride(encoded_number(
				Encoded_Data(data),
				Data_Position(position+RANGE_BOUND_COUNT*int(ENCODED_WIDTH_16)),
				ENCODED_WIDTH_16,
			)),
		}
		position += RANGE_16_BYTE_COUNT
	}
	range_32_count := int(encoded_number(
		Encoded_Data(data), Data_Position(position), ENCODED_WIDTH_16,
	))
	aver.Always(
		range_32_count <= len(ranges_32),
		"Caller storage holds every decoded 32-bit Unicode range.",
	)
	position += TABLE_RANGE_32_COUNT_BYTE_COUNT
	table.Ranges_32 = ranges_32[:range_32_count]
	for index := range table.Ranges_32 {
		table.Ranges_32[index] = Range_32{
			Minimum: Range_32_Minimum(encoded_number(
				Encoded_Data(data), Data_Position(position), ENCODED_WIDTH_32,
			)),
			Maximum: Range_32_Maximum(encoded_number(
				Encoded_Data(data),
				Data_Position(position+int(ENCODED_WIDTH_32)),
				ENCODED_WIDTH_32,
			)),
			Stride: Range_32_Stride(encoded_number(
				Encoded_Data(data),
				Data_Position(position+RANGE_BOUND_COUNT*int(ENCODED_WIDTH_32)),
				ENCODED_WIDTH_32,
			)),
		}
		position += RANGE_32_BYTE_COUNT
	}
	return table
}

func encoded_range_table_contains(
	data Table_Data, character Character,
) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "encoded_range_table_contains.yes") }()
	Table_Data_Invariants(data, "encoded_range_table_contains.data")
	Character_Invariants(character, "encoded_range_table_contains.character")
	range_16_count := int(encoded_number(Encoded_Data(data), 0, ENCODED_WIDTH_16))
	range_16_position := TABLE_HEADER_BYTE_COUNT
	if character >= 0 {
		if uint32(character) <= uint32(RANGE_16_MAXIMUM) {
			if encoded_ranges_16_contain(
				Encoded_Data(data),
				Data_Position(range_16_position),
				Data_Count(range_16_count),
				Code_Point_16(character),
			) {
				return true
			}
		}
	}
	range_32_count_position := range_16_position + range_16_count*RANGE_16_BYTE_COUNT
	range_32_count := int(encoded_number(
		Encoded_Data(data), Data_Position(range_32_count_position), ENCODED_WIDTH_16,
	))
	range_32_position := range_32_count_position + TABLE_RANGE_32_COUNT_BYTE_COUNT
	if character < Character(RANGE_32_MINIMUM) {
		return false
	}
	if character > RUNE_MAX {
		return false
	}
	return encoded_ranges_32_contain(
		Encoded_Data(data),
		Data_Position(range_32_position),
		Data_Count(range_32_count),
		Code_Point_32(character),
	)
}

func binary_range_table_contains(
	ranges_16 Encoded_Data, ranges_32 Encoded_Data, character Character,
) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "binary_range_table_contains.yes") }()
	Encoded_Data_Invariants(ranges_16, "binary_range_table_contains.ranges_16")
	Encoded_Data_Invariants(ranges_32, "binary_range_table_contains.ranges_32")
	Character_Invariants(character, "binary_range_table_contains.character")
	if uint32(character) <= uint32(RANGE_16_MAXIMUM) {
		value, lower, upper := uint16(character), 0, len(ranges_16)/RANGE_16_BYTE_COUNT
		for lower < upper {
			middle := int(uint(lower+upper) >> 1)
			position := middle * RANGE_16_BYTE_COUNT
			minimum := uint16(ranges_16[position])<<bits.BIT_COUNT_8_MAXIMUM |
				uint16(ranges_16[position+1])
			maximum_position := position + int(ENCODED_WIDTH_16)
			maximum := uint16(ranges_16[maximum_position])<<bits.BIT_COUNT_8_MAXIMUM |
				uint16(ranges_16[maximum_position+1])
			if value < minimum {
				upper = middle
				continue
			}
			if value > maximum {
				lower = middle + 1
				continue
			}
			stride_position := position + RANGE_BOUND_COUNT*int(ENCODED_WIDTH_16)
			stride := uint16(ranges_16[stride_position])<<bits.BIT_COUNT_8_MAXIMUM |
				uint16(ranges_16[stride_position+1])
			return Boolean(stride == 1 || (value-minimum)%stride == 0)
		}
		return false
	}
	if character < Character(RANGE_32_MINIMUM) {
		return false
	}
	if character > RUNE_MAX {
		return false
	}
	value := uint32(character)
	lower := 0
	upper := len(ranges_32) / RANGE_32_BYTE_COUNT
	for lower < upper {
		middle := int(uint(lower+upper) >> 1)
		position := middle * RANGE_32_BYTE_COUNT
		minimum := uint32(ranges_32[position])<<ENCODED_32_FIRST_BYTE_SHIFT |
			uint32(ranges_32[position+1])<<ENCODED_32_SECOND_BYTE_SHIFT |
			uint32(ranges_32[position+2])<<bits.BIT_COUNT_8_MAXIMUM |
			uint32(ranges_32[position+3])
		maximum_position := position + int(ENCODED_WIDTH_32)
		maximum := uint32(ranges_32[maximum_position])<<ENCODED_32_FIRST_BYTE_SHIFT |
			uint32(ranges_32[maximum_position+1])<<
				ENCODED_32_SECOND_BYTE_SHIFT |
			uint32(ranges_32[maximum_position+2])<<bits.BIT_COUNT_8_MAXIMUM |
			uint32(ranges_32[maximum_position+3])
		if value < minimum {
			upper = middle
			continue
		}
		if value > maximum {
			lower = middle + 1
			continue
		}
		stride_position := position + RANGE_BOUND_COUNT*int(ENCODED_WIDTH_32)
		stride := uint32(ranges_32[stride_position])<<ENCODED_32_FIRST_BYTE_SHIFT |
			uint32(ranges_32[stride_position+1])<<
				ENCODED_32_SECOND_BYTE_SHIFT |
			uint32(ranges_32[stride_position+2])<<bits.BIT_COUNT_8_MAXIMUM |
			uint32(ranges_32[stride_position+3])
		return Boolean(stride == 1 || (value-minimum)%stride == 0)
	}
	return false
}

func encoded_ranges_16_contain(
	data Encoded_Data,
	start Data_Position,
	count Data_Count,
	character Code_Point_16,
) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "encoded_ranges_16_contain.yes") }()
	Encoded_Data_Invariants(data, "encoded_ranges_16_contain.data")
	Data_Position_Invariants(start, "encoded_ranges_16_contain.start")
	Data_Count_Invariants(count, "encoded_ranges_16_contain.count")
	Code_Point_16_Invariants(character, "encoded_ranges_16_contain.character")
	lower := 0
	upper := int(count)
	for lower < upper {
		middle := int(uint(lower+upper) >> 1)
		position := int(start) + middle*RANGE_16_BYTE_COUNT
		minimum := uint16(encoded_number(
			data, Data_Position(position), ENCODED_WIDTH_16,
		))
		maximum := uint16(encoded_number(
			data, Data_Position(position+int(ENCODED_WIDTH_16)), ENCODED_WIDTH_16,
		))
		if uint16(character) < minimum {
			upper = middle
			continue
		}
		if uint16(character) > maximum {
			lower = middle + 1
			continue
		}
		stride := uint16(encoded_number(
			data,
			Data_Position(position+RANGE_BOUND_COUNT*int(ENCODED_WIDTH_16)),
			ENCODED_WIDTH_16,
		))
		return Boolean((uint16(character)-minimum)%stride == 0)
	}
	return false
}

func encoded_ranges_32_contain(
	data Encoded_Data,
	start Data_Position,
	count Data_Count,
	character Code_Point_32,
) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "encoded_ranges_32_contain.yes") }()
	Encoded_Data_Invariants(data, "encoded_ranges_32_contain.data")
	Data_Position_Invariants(start, "encoded_ranges_32_contain.start")
	Data_Count_Invariants(count, "encoded_ranges_32_contain.count")
	Code_Point_32_Invariants(character, "encoded_ranges_32_contain.character")
	lower := 0
	upper := int(count)
	for lower < upper {
		middle := int(uint(lower+upper) >> 1)
		position := int(start) + middle*RANGE_32_BYTE_COUNT
		minimum := uint32(encoded_number(
			data, Data_Position(position), ENCODED_WIDTH_32,
		))
		maximum := uint32(encoded_number(
			data, Data_Position(position+int(ENCODED_WIDTH_32)), ENCODED_WIDTH_32,
		))
		if uint32(character) < minimum {
			upper = middle
			continue
		}
		if uint32(character) > maximum {
			lower = middle + 1
			continue
		}
		stride := uint32(encoded_number(
			data,
			Data_Position(position+RANGE_BOUND_COUNT*int(ENCODED_WIDTH_32)),
			ENCODED_WIDTH_32,
		))
		return Boolean((uint32(character)-minimum)%stride == 0)
	}
	return false
}

func encoded_case_range(
	data Encoded_Data, character Character,
) (case_range Case_Range, found Boolean) {
	defer func() {
		Case_Range_Invariants(case_range, "encoded_case_range.range")
		Boolean_Invariants(found, "encoded_case_range.found")
	}()
	Encoded_Data_Invariants(data, "encoded_case_range.data")
	Character_Invariants(character, "encoded_case_range.character")
	if len(data) == 0 {
		return Case_Range{}, false
	}
	lower := 0
	upper := len(data) / CASE_RANGE_BYTE_COUNT
	for lower < upper {
		middle := int(uint(lower+upper) >> 1)
		position := middle * CASE_RANGE_BYTE_COUNT
		minimum := uint32(data[position])<<ENCODED_32_FIRST_BYTE_SHIFT |
			uint32(data[position+1])<<ENCODED_32_SECOND_BYTE_SHIFT |
			uint32(data[position+2])<<bits.BIT_COUNT_8_MAXIMUM |
			uint32(data[position+3])
		maximum_position := position + int(ENCODED_WIDTH_32)
		maximum := uint32(data[maximum_position])<<ENCODED_32_FIRST_BYTE_SHIFT |
			uint32(data[maximum_position+1])<<ENCODED_32_SECOND_BYTE_SHIFT |
			uint32(data[maximum_position+2])<<bits.BIT_COUNT_8_MAXIMUM |
			uint32(data[maximum_position+3])
		if character < Character(minimum) {
			upper = middle
			continue
		}
		if character > Character(maximum) {
			lower = middle + 1
			continue
		}
		delta_position := maximum_position + int(ENCODED_WIDTH_32)
		upper_position := delta_position + int(UPPER_CASE)*int(ENCODED_WIDTH_32)
		lower_position := delta_position + int(LOWER_CASE)*int(ENCODED_WIDTH_32)
		title_position := delta_position + int(TITLE_CASE)*int(ENCODED_WIDTH_32)
		case_range = Case_Range{
			Minimum: Case_Range_Minimum(minimum),
			Maximum: Case_Range_Maximum(maximum),
			Deltas: Case_Delta{
				Upper: Upper_Case_Delta(
					uint32(data[upper_position])<<ENCODED_32_FIRST_BYTE_SHIFT |
						uint32(data[upper_position+1])<<
							ENCODED_32_SECOND_BYTE_SHIFT |
						uint32(data[upper_position+2])<<
							bits.BIT_COUNT_8_MAXIMUM |
						uint32(data[upper_position+3]),
				),
				Lower: Lower_Case_Delta(
					uint32(data[lower_position])<<ENCODED_32_FIRST_BYTE_SHIFT |
						uint32(data[lower_position+1])<<
							ENCODED_32_SECOND_BYTE_SHIFT |
						uint32(data[lower_position+2])<<
							bits.BIT_COUNT_8_MAXIMUM |
						uint32(data[lower_position+3]),
				),
				Title: Title_Case_Delta(
					uint32(data[title_position])<<ENCODED_32_FIRST_BYTE_SHIFT |
						uint32(data[title_position+1])<<
							ENCODED_32_SECOND_BYTE_SHIFT |
						uint32(data[title_position+2])<<
							bits.BIT_COUNT_8_MAXIMUM |
						uint32(data[title_position+3]),
				),
			},
		}
		return case_range, true
	}
	return Case_Range{}, false
}

func encoded_name_equal(
	data Encoded_Data, position Data_Position, size Data_Count, name Name,
) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "encoded_name_equal.yes") }()
	Encoded_Data_Invariants(data, "encoded_name_equal.data")
	Data_Position_Invariants(position, "encoded_name_equal.position")
	Data_Count_Invariants(size, "encoded_name_equal.size")
	Name_Invariants(name, "encoded_name_equal.name")
	if int(size) != len(name) {
		return false
	}
	for index := 0; index < int(size); index++ {
		value := byte(encoded_number(
			data, Data_Position(int(position)+index), ENCODED_WIDTH_BYTE,
		))
		if value != name[index] {
			return false
		}
	}
	return true
}

func encoded_number(
	data Encoded_Data, position Data_Position, width Encoded_Width,
) (value Encoded_Number) {
	defer func() { Encoded_Number_Invariants(value, "encoded_number.value") }()
	Encoded_Data_Invariants(data, "encoded_number.data")
	Data_Position_Invariants(position, "encoded_number.position")
	Encoded_Width_Invariants(width, "encoded_number.width")
	for byte_index := 0; byte_index < int(width); byte_index++ {
		character_position := (int(position) + byte_index) *
			HEXADECIMAL_BYTE_CHARACTER_COUNT
		high_character := data[character_position]
		low_character := data[character_position+1]
		high := high_character - '0'
		if high_character > '9' {
			high = high_character - 'a' + HEXADECIMAL_LETTER_VALUE_OFFSET
		}
		low := low_character - '0'
		if low_character > '9' {
			low = low_character - 'a' + HEXADECIMAL_LETTER_VALUE_OFFSET
		}
		value = value<<bits.BIT_COUNT_8_MAXIMUM |
			Encoded_Number(high<<HEXADECIMAL_NIBBLE_BIT_COUNT|low)
	}
	return value
}

// Generated Unicode data follows this marker.

// RANGES_16_COUNT_MAXIMUM is the largest native 16-bit range collection.
const RANGES_16_COUNT_MAXIMUM = 359

// RANGES_32_COUNT_MAXIMUM is the largest native 32-bit range collection.
const RANGES_32_COUNT_MAXIMUM = 322

// PRINT_RANGES_16_DATA avoids hexadecimal decoding for 16-bit print searches.
const PRINT_RANGES_16_DATA Encoded_Data = "" +
	"\x00\x20\x00\x7e\x00\x01\x00\xa1\x00\xac\x00\x01\x00\xae\x03\x77\x00\x01\x03\x7a\x03\x7f" +
	"\x00\x01\x03\x84\x03\x8a\x00\x01\x03\x8c\x03\x8e\x00\x02\x03\x8f\x03\xa1\x00\x01\x03\xa3" +
	"\x05\x2f\x00\x01\x05\x31\x05\x56\x00\x01\x05\x59\x05\x8a\x00\x01\x05\x8d\x05\x8f\x00\x01" +
	"\x05\x91\x05\xc7\x00\x01\x05\xd0\x05\xea\x00\x01\x05\xef\x05\xf4\x00\x01\x06\x06\x06\x1b" +
	"\x00\x01\x06\x1d\x06\xdc\x00\x01\x06\xde\x07\x0d\x00\x01\x07\x10\x07\x4a\x00\x01\x07\x4d" +
	"\x07\xb1\x00\x01\x07\xc0\x07\xfa\x00\x01\x07\xfd\x08\x2d\x00\x01\x08\x30\x08\x3e\x00\x01" +
	"\x08\x40\x08\x5b\x00\x01\x08\x5e\x08\x60\x00\x02\x08\x61\x08\x6a\x00\x01\x08\x70\x08\x8e" +
	"\x00\x01\x08\x98\x08\xe1\x00\x01\x08\xe3\x09\x83\x00\x01\x09\x85\x09\x8c\x00\x01\x09\x8f" +
	"\x09\x90\x00\x01\x09\x93\x09\xa8\x00\x01\x09\xaa\x09\xb0\x00\x01\x09\xb2\x09\xb6\x00\x04" +
	"\x09\xb7\x09\xb9\x00\x01\x09\xbc\x09\xc4\x00\x01\x09\xc7\x09\xc8\x00\x01\x09\xcb\x09\xce" +
	"\x00\x01\x09\xd7\x09\xdc\x00\x05\x09\xdd\x09\xdf\x00\x02\x09\xe0\x09\xe3\x00\x01\x09\xe6" +
	"\x09\xfe\x00\x01\x0a\x01\x0a\x03\x00\x01\x0a\x05\x0a\x0a\x00\x01\x0a\x0f\x0a\x10\x00\x01" +
	"\x0a\x13\x0a\x28\x00\x01\x0a\x2a\x0a\x30\x00\x01\x0a\x32\x0a\x33\x00\x01\x0a\x35\x0a\x36" +
	"\x00\x01\x0a\x38\x0a\x39\x00\x01\x0a\x3c\x0a\x3e\x00\x02\x0a\x3f\x0a\x42\x00\x01\x0a\x47" +
	"\x0a\x48\x00\x01\x0a\x4b\x0a\x4d\x00\x01\x0a\x51\x0a\x59\x00\x08\x0a\x5a\x0a\x5c\x00\x01" +
	"\x0a\x5e\x0a\x66\x00\x08\x0a\x67\x0a\x76\x00\x01\x0a\x81\x0a\x83\x00\x01\x0a\x85\x0a\x8d" +
	"\x00\x01\x0a\x8f\x0a\x91\x00\x01\x0a\x93\x0a\xa8\x00\x01\x0a\xaa\x0a\xb0\x00\x01\x0a\xb2" +
	"\x0a\xb3\x00\x01\x0a\xb5\x0a\xb9\x00\x01\x0a\xbc\x0a\xc5\x00\x01\x0a\xc7\x0a\xc9\x00\x01" +
	"\x0a\xcb\x0a\xcd\x00\x01\x0a\xd0\x0a\xe0\x00\x10\x0a\xe1\x0a\xe3\x00\x01\x0a\xe6\x0a\xf1" +
	"\x00\x01\x0a\xf9\x0a\xff\x00\x01\x0b\x01\x0b\x03\x00\x01\x0b\x05\x0b\x0c\x00\x01\x0b\x0f" +
	"\x0b\x10\x00\x01\x0b\x13\x0b\x28\x00\x01\x0b\x2a\x0b\x30\x00\x01\x0b\x32\x0b\x33\x00\x01" +
	"\x0b\x35\x0b\x39\x00\x01\x0b\x3c\x0b\x44\x00\x01\x0b\x47\x0b\x48\x00\x01\x0b\x4b\x0b\x4d" +
	"\x00\x01\x0b\x55\x0b\x57\x00\x01\x0b\x5c\x0b\x5d\x00\x01\x0b\x5f\x0b\x63\x00\x01\x0b\x66" +
	"\x0b\x77\x00\x01\x0b\x82\x0b\x83\x00\x01\x0b\x85\x0b\x8a\x00\x01\x0b\x8e\x0b\x90\x00\x01" +
	"\x0b\x92\x0b\x95\x00\x01\x0b\x99\x0b\x9a\x00\x01\x0b\x9c\x0b\x9e\x00\x02\x0b\x9f\x0b\xa3" +
	"\x00\x04\x0b\xa4\x0b\xa8\x00\x04\x0b\xa9\x0b\xaa\x00\x01\x0b\xae\x0b\xb9\x00\x01\x0b\xbe" +
	"\x0b\xc2\x00\x01\x0b\xc6\x0b\xc8\x00\x01\x0b\xca\x0b\xcd\x00\x01\x0b\xd0\x0b\xd7\x00\x07" +
	"\x0b\xe6\x0b\xfa\x00\x01\x0c\x00\x0c\x0c\x00\x01\x0c\x0e\x0c\x10\x00\x01\x0c\x12\x0c\x28" +
	"\x00\x01\x0c\x2a\x0c\x39\x00\x01\x0c\x3c\x0c\x44\x00\x01\x0c\x46\x0c\x48\x00\x01\x0c\x4a" +
	"\x0c\x4d\x00\x01\x0c\x55\x0c\x56\x00\x01\x0c\x58\x0c\x5a\x00\x01\x0c\x5d\x0c\x60\x00\x03" +
	"\x0c\x61\x0c\x63\x00\x01\x0c\x66\x0c\x6f\x00\x01\x0c\x77\x0c\x8c\x00\x01\x0c\x8e\x0c\x90" +
	"\x00\x01\x0c\x92\x0c\xa8\x00\x01\x0c\xaa\x0c\xb3\x00\x01\x0c\xb5\x0c\xb9\x00\x01\x0c\xbc" +
	"\x0c\xc4\x00\x01\x0c\xc6\x0c\xc8\x00\x01\x0c\xca\x0c\xcd\x00\x01\x0c\xd5\x0c\xd6\x00\x01" +
	"\x0c\xdd\x0c\xde\x00\x01\x0c\xe0\x0c\xe3\x00\x01\x0c\xe6\x0c\xef\x00\x01\x0c\xf1\x0c\xf3" +
	"\x00\x01\x0d\x00\x0d\x0c\x00\x01\x0d\x0e\x0d\x10\x00\x01\x0d\x12\x0d\x44\x00\x01\x0d\x46" +
	"\x0d\x48\x00\x01\x0d\x4a\x0d\x4f\x00\x01\x0d\x54\x0d\x63\x00\x01\x0d\x66\x0d\x7f\x00\x01" +
	"\x0d\x81\x0d\x83\x00\x01\x0d\x85\x0d\x96\x00\x01\x0d\x9a\x0d\xb1\x00\x01\x0d\xb3\x0d\xbb" +
	"\x00\x01\x0d\xbd\x0d\xc0\x00\x03\x0d\xc1\x0d\xc6\x00\x01\x0d\xca\x0d\xcf\x00\x05\x0d\xd0" +
	"\x0d\xd4\x00\x01\x0d\xd6\x0d\xd8\x00\x02\x0d\xd9\x0d\xdf\x00\x01\x0d\xe6\x0d\xef\x00\x01" +
	"\x0d\xf2\x0d\xf4\x00\x01\x0e\x01\x0e\x3a\x00\x01\x0e\x3f\x0e\x5b\x00\x01\x0e\x81\x0e\x82" +
	"\x00\x01\x0e\x84\x0e\x86\x00\x02\x0e\x87\x0e\x8a\x00\x01\x0e\x8c\x0e\xa3\x00\x01\x0e\xa5" +
	"\x0e\xa7\x00\x02\x0e\xa8\x0e\xbd\x00\x01\x0e\xc0\x0e\xc4\x00\x01\x0e\xc6\x0e\xc8\x00\x02" +
	"\x0e\xc9\x0e\xce\x00\x01\x0e\xd0\x0e\xd9\x00\x01\x0e\xdc\x0e\xdf\x00\x01\x0f\x00\x0f\x47" +
	"\x00\x01\x0f\x49\x0f\x6c\x00\x01\x0f\x71\x0f\x97\x00\x01\x0f\x99\x0f\xbc\x00\x01\x0f\xbe" +
	"\x0f\xcc\x00\x01\x0f\xce\x0f\xda\x00\x01\x10\x00\x10\xc5\x00\x01\x10\xc7\x10\xcd\x00\x06" +
	"\x10\xd0\x12\x48\x00\x01\x12\x4a\x12\x4d\x00\x01\x12\x50\x12\x56\x00\x01\x12\x58\x12\x5a" +
	"\x00\x02\x12\x5b\x12\x5d\x00\x01\x12\x60\x12\x88\x00\x01\x12\x8a\x12\x8d\x00\x01\x12\x90" +
	"\x12\xb0\x00\x01\x12\xb2\x12\xb5\x00\x01\x12\xb8\x12\xbe\x00\x01\x12\xc0\x12\xc2\x00\x02" +
	"\x12\xc3\x12\xc5\x00\x01\x12\xc8\x12\xd6\x00\x01\x12\xd8\x13\x10\x00\x01\x13\x12\x13\x15" +
	"\x00\x01\x13\x18\x13\x5a\x00\x01\x13\x5d\x13\x7c\x00\x01\x13\x80\x13\x99\x00\x01\x13\xa0" +
	"\x13\xf5\x00\x01\x13\xf8\x13\xfd\x00\x01\x14\x00\x16\x7f\x00\x01\x16\x81\x16\x9c\x00\x01" +
	"\x16\xa0\x16\xf8\x00\x01\x17\x00\x17\x15\x00\x01\x17\x1f\x17\x36\x00\x01\x17\x40\x17\x53" +
	"\x00\x01\x17\x60\x17\x6c\x00\x01\x17\x6e\x17\x70\x00\x01\x17\x72\x17\x73\x00\x01\x17\x80" +
	"\x17\xdd\x00\x01\x17\xe0\x17\xe9\x00\x01\x17\xf0\x17\xf9\x00\x01\x18\x00\x18\x0d\x00\x01" +
	"\x18\x0f\x18\x19\x00\x01\x18\x20\x18\x78\x00\x01\x18\x80\x18\xaa\x00\x01\x18\xb0\x18\xf5" +
	"\x00\x01\x19\x00\x19\x1e\x00\x01\x19\x20\x19\x2b\x00\x01\x19\x30\x19\x3b\x00\x01\x19\x40" +
	"\x19\x44\x00\x04\x19\x45\x19\x6d\x00\x01\x19\x70\x19\x74\x00\x01\x19\x80\x19\xab\x00\x01" +
	"\x19\xb0\x19\xc9\x00\x01\x19\xd0\x19\xda\x00\x01\x19\xde\x1a\x1b\x00\x01\x1a\x1e\x1a\x5e" +
	"\x00\x01\x1a\x60\x1a\x7c\x00\x01\x1a\x7f\x1a\x89\x00\x01\x1a\x90\x1a\x99\x00\x01\x1a\xa0" +
	"\x1a\xad\x00\x01\x1a\xb0\x1a\xce\x00\x01\x1b\x00\x1b\x4c\x00\x01\x1b\x50\x1b\x7e\x00\x01" +
	"\x1b\x80\x1b\xf3\x00\x01\x1b\xfc\x1c\x37\x00\x01\x1c\x3b\x1c\x49\x00\x01\x1c\x4d\x1c\x88" +
	"\x00\x01\x1c\x90\x1c\xba\x00\x01\x1c\xbd\x1c\xc7\x00\x01\x1c\xd0\x1c\xfa\x00\x01\x1d\x00" +
	"\x1f\x15\x00\x01\x1f\x18\x1f\x1d\x00\x01\x1f\x20\x1f\x45\x00\x01\x1f\x48\x1f\x4d\x00\x01" +
	"\x1f\x50\x1f\x57\x00\x01\x1f\x59\x1f\x5f\x00\x02\x1f\x60\x1f\x7d\x00\x01\x1f\x80\x1f\xb4" +
	"\x00\x01\x1f\xb6\x1f\xc4\x00\x01\x1f\xc6\x1f\xd3\x00\x01\x1f\xd6\x1f\xdb\x00\x01\x1f\xdd" +
	"\x1f\xef\x00\x01\x1f\xf2\x1f\xf4\x00\x01\x1f\xf6\x1f\xfe\x00\x01\x20\x10\x20\x27\x00\x01" +
	"\x20\x30\x20\x5e\x00\x01\x20\x70\x20\x71\x00\x01\x20\x74\x20\x8e\x00\x01\x20\x90\x20\x9c" +
	"\x00\x01\x20\xa0\x20\xc0\x00\x01\x20\xd0\x20\xf0\x00\x01\x21\x00\x21\x8b\x00\x01\x21\x90" +
	"\x24\x26\x00\x01\x24\x40\x24\x4a\x00\x01\x24\x60\x2b\x73\x00\x01\x2b\x76\x2b\x95\x00\x01" +
	"\x2b\x97\x2c\xf3\x00\x01\x2c\xf9\x2d\x25\x00\x01\x2d\x27\x2d\x2d\x00\x06\x2d\x30\x2d\x67" +
	"\x00\x01\x2d\x6f\x2d\x70\x00\x01\x2d\x7f\x2d\x96\x00\x01\x2d\xa0\x2d\xa6\x00\x01\x2d\xa8" +
	"\x2d\xae\x00\x01\x2d\xb0\x2d\xb6\x00\x01\x2d\xb8\x2d\xbe\x00\x01\x2d\xc0\x2d\xc6\x00\x01" +
	"\x2d\xc8\x2d\xce\x00\x01\x2d\xd0\x2d\xd6\x00\x01\x2d\xd8\x2d\xde\x00\x01\x2d\xe0\x2e\x5d" +
	"\x00\x01\x2e\x80\x2e\x99\x00\x01\x2e\x9b\x2e\xf3\x00\x01\x2f\x00\x2f\xd5\x00\x01\x2f\xf0" +
	"\x2f\xfb\x00\x01\x30\x01\x30\x3f\x00\x01\x30\x41\x30\x96\x00\x01\x30\x99\x30\xff\x00\x01" +
	"\x31\x05\x31\x2f\x00\x01\x31\x31\x31\x8e\x00\x01\x31\x90\x31\xe3\x00\x01\x31\xf0\x32\x1e" +
	"\x00\x01\x32\x20\xa4\x8c\x00\x01\xa4\x90\xa4\xc6\x00\x01\xa4\xd0\xa6\x2b\x00\x01\xa6\x40" +
	"\xa6\xf7\x00\x01\xa7\x00\xa7\xca\x00\x01\xa7\xd0\xa7\xd1\x00\x01\xa7\xd3\xa7\xd5\x00\x02" +
	"\xa7\xd6\xa7\xd9\x00\x01\xa7\xf2\xa8\x2c\x00\x01\xa8\x30\xa8\x39\x00\x01\xa8\x40\xa8\x77" +
	"\x00\x01\xa8\x80\xa8\xc5\x00\x01\xa8\xce\xa8\xd9\x00\x01\xa8\xe0\xa9\x53\x00\x01\xa9\x5f" +
	"\xa9\x7c\x00\x01\xa9\x80\xa9\xcd\x00\x01\xa9\xcf\xa9\xd9\x00\x01\xa9\xde\xa9\xfe\x00\x01" +
	"\xaa\x00\xaa\x36\x00\x01\xaa\x40\xaa\x4d\x00\x01\xaa\x50\xaa\x59\x00\x01\xaa\x5c\xaa\xc2" +
	"\x00\x01\xaa\xdb\xaa\xf6\x00\x01\xab\x01\xab\x06\x00\x01\xab\x09\xab\x0e\x00\x01\xab\x11" +
	"\xab\x16\x00\x01\xab\x20\xab\x26\x00\x01\xab\x28\xab\x2e\x00\x01\xab\x30\xab\x6b\x00\x01" +
	"\xab\x70\xab\xed\x00\x01\xab\xf0\xab\xf9\x00\x01\xac\x00\xd7\xa3\x00\x01\xd7\xb0\xd7\xc6" +
	"\x00\x01\xd7\xcb\xd7\xfb\x00\x01\xf9\x00\xfa\x6d\x00\x01\xfa\x70\xfa\xd9\x00\x01\xfb\x00" +
	"\xfb\x06\x00\x01\xfb\x13\xfb\x17\x00\x01\xfb\x1d\xfb\x36\x00\x01\xfb\x38\xfb\x3c\x00\x01" +
	"\xfb\x3e\xfb\x40\x00\x02\xfb\x41\xfb\x43\x00\x02\xfb\x44\xfb\x46\x00\x02\xfb\x47\xfb\xc2" +
	"\x00\x01\xfb\xd3\xfd\x8f\x00\x01\xfd\x92\xfd\xc7\x00\x01\xfd\xcf\xfd\xf0\x00\x21\xfd\xf1" +
	"\xfe\x19\x00\x01\xfe\x20\xfe\x52\x00\x01\xfe\x54\xfe\x66\x00\x01\xfe\x68\xfe\x6b\x00\x01" +
	"\xfe\x70\xfe\x74\x00\x01\xfe\x76\xfe\xfc\x00\x01\xff\x01\xff\xbe\x00\x01\xff\xc2\xff\xc7" +
	"\x00\x01\xff\xca\xff\xcf\x00\x01\xff\xd2\xff\xd7\x00\x01\xff\xda\xff\xdc\x00\x01\xff\xe0" +
	"\xff\xe6\x00\x01\xff\xe8\xff\xee\x00\x01\xff\xfc\xff\xfd\x00\x01"

// PRINT_RANGES_32_DATA avoids hexadecimal decoding for 32-bit print searches.
const PRINT_RANGES_32_DATA Encoded_Data = "" +
	"\x00\x01\x00\x00\x00\x01\x00\x0b\x00\x00\x00\x01\x00\x01\x00\x0d\x00\x01\x00\x26\x00\x00" +
	"\x00\x01\x00\x01\x00\x28\x00\x01\x00\x3a\x00\x00\x00\x01\x00\x01\x00\x3c\x00\x01\x00\x3d" +
	"\x00\x00\x00\x01\x00\x01\x00\x3f\x00\x01\x00\x4d\x00\x00\x00\x01\x00\x01\x00\x50\x00\x01" +
	"\x00\x5d\x00\x00\x00\x01\x00\x01\x00\x80\x00\x01\x00\xfa\x00\x00\x00\x01\x00\x01\x01\x00" +
	"\x00\x01\x01\x02\x00\x00\x00\x01\x00\x01\x01\x07\x00\x01\x01\x33\x00\x00\x00\x01\x00\x01" +
	"\x01\x37\x00\x01\x01\x8e\x00\x00\x00\x01\x00\x01\x01\x90\x00\x01\x01\x9c\x00\x00\x00\x01" +
	"\x00\x01\x01\xa0\x00\x01\x01\xd0\x00\x00\x00\x30\x00\x01\x01\xd1\x00\x01\x01\xfd\x00\x00" +
	"\x00\x01\x00\x01\x02\x80\x00\x01\x02\x9c\x00\x00\x00\x01\x00\x01\x02\xa0\x00\x01\x02\xd0" +
	"\x00\x00\x00\x01\x00\x01\x02\xe0\x00\x01\x02\xfb\x00\x00\x00\x01\x00\x01\x03\x00\x00\x01" +
	"\x03\x23\x00\x00\x00\x01\x00\x01\x03\x2d\x00\x01\x03\x4a\x00\x00\x00\x01\x00\x01\x03\x50" +
	"\x00\x01\x03\x7a\x00\x00\x00\x01\x00\x01\x03\x80\x00\x01\x03\x9d\x00\x00\x00\x01\x00\x01" +
	"\x03\x9f\x00\x01\x03\xc3\x00\x00\x00\x01\x00\x01\x03\xc8\x00\x01\x03\xd5\x00\x00\x00\x01" +
	"\x00\x01\x04\x00\x00\x01\x04\x9d\x00\x00\x00\x01\x00\x01\x04\xa0\x00\x01\x04\xa9\x00\x00" +
	"\x00\x01\x00\x01\x04\xb0\x00\x01\x04\xd3\x00\x00\x00\x01\x00\x01\x04\xd8\x00\x01\x04\xfb" +
	"\x00\x00\x00\x01\x00\x01\x05\x00\x00\x01\x05\x27\x00\x00\x00\x01\x00\x01\x05\x30\x00\x01" +
	"\x05\x63\x00\x00\x00\x01\x00\x01\x05\x6f\x00\x01\x05\x7a\x00\x00\x00\x01\x00\x01\x05\x7c" +
	"\x00\x01\x05\x8a\x00\x00\x00\x01\x00\x01\x05\x8c\x00\x01\x05\x92\x00\x00\x00\x01\x00\x01" +
	"\x05\x94\x00\x01\x05\x95\x00\x00\x00\x01\x00\x01\x05\x97\x00\x01\x05\xa1\x00\x00\x00\x01" +
	"\x00\x01\x05\xa3\x00\x01\x05\xb1\x00\x00\x00\x01\x00\x01\x05\xb3\x00\x01\x05\xb9\x00\x00" +
	"\x00\x01\x00\x01\x05\xbb\x00\x01\x05\xbc\x00\x00\x00\x01\x00\x01\x06\x00\x00\x01\x07\x36" +
	"\x00\x00\x00\x01\x00\x01\x07\x40\x00\x01\x07\x55\x00\x00\x00\x01\x00\x01\x07\x60\x00\x01" +
	"\x07\x67\x00\x00\x00\x01\x00\x01\x07\x80\x00\x01\x07\x85\x00\x00\x00\x01\x00\x01\x07\x87" +
	"\x00\x01\x07\xb0\x00\x00\x00\x01\x00\x01\x07\xb2\x00\x01\x07\xba\x00\x00\x00\x01\x00\x01" +
	"\x08\x00\x00\x01\x08\x05\x00\x00\x00\x01\x00\x01\x08\x08\x00\x01\x08\x0a\x00\x00\x00\x02" +
	"\x00\x01\x08\x0b\x00\x01\x08\x35\x00\x00\x00\x01\x00\x01\x08\x37\x00\x01\x08\x38\x00\x00" +
	"\x00\x01\x00\x01\x08\x3c\x00\x01\x08\x3f\x00\x00\x00\x03\x00\x01\x08\x40\x00\x01\x08\x55" +
	"\x00\x00\x00\x01\x00\x01\x08\x57\x00\x01\x08\x9e\x00\x00\x00\x01\x00\x01\x08\xa7\x00\x01" +
	"\x08\xaf\x00\x00\x00\x01\x00\x01\x08\xe0\x00\x01\x08\xf2\x00\x00\x00\x01\x00\x01\x08\xf4" +
	"\x00\x01\x08\xf5\x00\x00\x00\x01\x00\x01\x08\xfb\x00\x01\x09\x1b\x00\x00\x00\x01\x00\x01" +
	"\x09\x1f\x00\x01\x09\x39\x00\x00\x00\x01\x00\x01\x09\x3f\x00\x01\x09\x80\x00\x00\x00\x41" +
	"\x00\x01\x09\x81\x00\x01\x09\xb7\x00\x00\x00\x01\x00\x01\x09\xbc\x00\x01\x09\xcf\x00\x00" +
	"\x00\x01\x00\x01\x09\xd2\x00\x01\x0a\x03\x00\x00\x00\x01\x00\x01\x0a\x05\x00\x01\x0a\x06" +
	"\x00\x00\x00\x01\x00\x01\x0a\x0c\x00\x01\x0a\x13\x00\x00\x00\x01\x00\x01\x0a\x15\x00\x01" +
	"\x0a\x17\x00\x00\x00\x01\x00\x01\x0a\x19\x00\x01\x0a\x35\x00\x00\x00\x01\x00\x01\x0a\x38" +
	"\x00\x01\x0a\x3a\x00\x00\x00\x01\x00\x01\x0a\x3f\x00\x01\x0a\x48\x00\x00\x00\x01\x00\x01" +
	"\x0a\x50\x00\x01\x0a\x58\x00\x00\x00\x01\x00\x01\x0a\x60\x00\x01\x0a\x9f\x00\x00\x00\x01" +
	"\x00\x01\x0a\xc0\x00\x01\x0a\xe6\x00\x00\x00\x01\x00\x01\x0a\xeb\x00\x01\x0a\xf6\x00\x00" +
	"\x00\x01\x00\x01\x0b\x00\x00\x01\x0b\x35\x00\x00\x00\x01\x00\x01\x0b\x39\x00\x01\x0b\x55" +
	"\x00\x00\x00\x01\x00\x01\x0b\x58\x00\x01\x0b\x72\x00\x00\x00\x01\x00\x01\x0b\x78\x00\x01" +
	"\x0b\x91\x00\x00\x00\x01\x00\x01\x0b\x99\x00\x01\x0b\x9c\x00\x00\x00\x01\x00\x01\x0b\xa9" +
	"\x00\x01\x0b\xaf\x00\x00\x00\x01\x00\x01\x0c\x00\x00\x01\x0c\x48\x00\x00\x00\x01\x00\x01" +
	"\x0c\x80\x00\x01\x0c\xb2\x00\x00\x00\x01\x00\x01\x0c\xc0\x00\x01\x0c\xf2\x00\x00\x00\x01" +
	"\x00\x01\x0c\xfa\x00\x01\x0d\x27\x00\x00\x00\x01\x00\x01\x0d\x30\x00\x01\x0d\x39\x00\x00" +
	"\x00\x01\x00\x01\x0e\x60\x00\x01\x0e\x7e\x00\x00\x00\x01\x00\x01\x0e\x80\x00\x01\x0e\xa9" +
	"\x00\x00\x00\x01\x00\x01\x0e\xab\x00\x01\x0e\xad\x00\x00\x00\x01\x00\x01\x0e\xb0\x00\x01" +
	"\x0e\xb1\x00\x00\x00\x01\x00\x01\x0e\xfd\x00\x01\x0f\x27\x00\x00\x00\x01\x00\x01\x0f\x30" +
	"\x00\x01\x0f\x59\x00\x00\x00\x01\x00\x01\x0f\x70\x00\x01\x0f\x89\x00\x00\x00\x01\x00\x01" +
	"\x0f\xb0\x00\x01\x0f\xcb\x00\x00\x00\x01\x00\x01\x0f\xe0\x00\x01\x0f\xf6\x00\x00\x00\x01" +
	"\x00\x01\x10\x00\x00\x01\x10\x4d\x00\x00\x00\x01\x00\x01\x10\x52\x00\x01\x10\x75\x00\x00" +
	"\x00\x01\x00\x01\x10\x7f\x00\x01\x10\xbc\x00\x00\x00\x01\x00\x01\x10\xbe\x00\x01\x10\xc2" +
	"\x00\x00\x00\x01\x00\x01\x10\xd0\x00\x01\x10\xe8\x00\x00\x00\x01\x00\x01\x10\xf0\x00\x01" +
	"\x10\xf9\x00\x00\x00\x01\x00\x01\x11\x00\x00\x01\x11\x34\x00\x00\x00\x01\x00\x01\x11\x36" +
	"\x00\x01\x11\x47\x00\x00\x00\x01\x00\x01\x11\x50\x00\x01\x11\x76\x00\x00\x00\x01\x00\x01" +
	"\x11\x80\x00\x01\x11\xdf\x00\x00\x00\x01\x00\x01\x11\xe1\x00\x01\x11\xf4\x00\x00\x00\x01" +
	"\x00\x01\x12\x00\x00\x01\x12\x11\x00\x00\x00\x01\x00\x01\x12\x13\x00\x01\x12\x41\x00\x00" +
	"\x00\x01\x00\x01\x12\x80\x00\x01\x12\x86\x00\x00\x00\x01\x00\x01\x12\x88\x00\x01\x12\x8a" +
	"\x00\x00\x00\x02\x00\x01\x12\x8b\x00\x01\x12\x8d\x00\x00\x00\x01\x00\x01\x12\x8f\x00\x01" +
	"\x12\x9d\x00\x00\x00\x01\x00\x01\x12\x9f\x00\x01\x12\xa9\x00\x00\x00\x01\x00\x01\x12\xb0" +
	"\x00\x01\x12\xea\x00\x00\x00\x01\x00\x01\x12\xf0\x00\x01\x12\xf9\x00\x00\x00\x01\x00\x01" +
	"\x13\x00\x00\x01\x13\x03\x00\x00\x00\x01\x00\x01\x13\x05\x00\x01\x13\x0c\x00\x00\x00\x01" +
	"\x00\x01\x13\x0f\x00\x01\x13\x10\x00\x00\x00\x01\x00\x01\x13\x13\x00\x01\x13\x28\x00\x00" +
	"\x00\x01\x00\x01\x13\x2a\x00\x01\x13\x30\x00\x00\x00\x01\x00\x01\x13\x32\x00\x01\x13\x33" +
	"\x00\x00\x00\x01\x00\x01\x13\x35\x00\x01\x13\x39\x00\x00\x00\x01\x00\x01\x13\x3b\x00\x01" +
	"\x13\x44\x00\x00\x00\x01\x00\x01\x13\x47\x00\x01\x13\x48\x00\x00\x00\x01\x00\x01\x13\x4b" +
	"\x00\x01\x13\x4d\x00\x00\x00\x01\x00\x01\x13\x50\x00\x01\x13\x57\x00\x00\x00\x07\x00\x01" +
	"\x13\x5d\x00\x01\x13\x63\x00\x00\x00\x01\x00\x01\x13\x66\x00\x01\x13\x6c\x00\x00\x00\x01" +
	"\x00\x01\x13\x70\x00\x01\x13\x74\x00\x00\x00\x01\x00\x01\x14\x00\x00\x01\x14\x5b\x00\x00" +
	"\x00\x01\x00\x01\x14\x5d\x00\x01\x14\x61\x00\x00\x00\x01\x00\x01\x14\x80\x00\x01\x14\xc7" +
	"\x00\x00\x00\x01\x00\x01\x14\xd0\x00\x01\x14\xd9\x00\x00\x00\x01\x00\x01\x15\x80\x00\x01" +
	"\x15\xb5\x00\x00\x00\x01\x00\x01\x15\xb8\x00\x01\x15\xdd\x00\x00\x00\x01\x00\x01\x16\x00" +
	"\x00\x01\x16\x44\x00\x00\x00\x01\x00\x01\x16\x50\x00\x01\x16\x59\x00\x00\x00\x01\x00\x01" +
	"\x16\x60\x00\x01\x16\x6c\x00\x00\x00\x01\x00\x01\x16\x80\x00\x01\x16\xb9\x00\x00\x00\x01" +
	"\x00\x01\x16\xc0\x00\x01\x16\xc9\x00\x00\x00\x01\x00\x01\x17\x00\x00\x01\x17\x1a\x00\x00" +
	"\x00\x01\x00\x01\x17\x1d\x00\x01\x17\x2b\x00\x00\x00\x01\x00\x01\x17\x30\x00\x01\x17\x46" +
	"\x00\x00\x00\x01\x00\x01\x18\x00\x00\x01\x18\x3b\x00\x00\x00\x01\x00\x01\x18\xa0\x00\x01" +
	"\x18\xf2\x00\x00\x00\x01\x00\x01\x18\xff\x00\x01\x19\x06\x00\x00\x00\x01\x00\x01\x19\x09" +
	"\x00\x01\x19\x0c\x00\x00\x00\x03\x00\x01\x19\x0d\x00\x01\x19\x13\x00\x00\x00\x01\x00\x01" +
	"\x19\x15\x00\x01\x19\x16\x00\x00\x00\x01\x00\x01\x19\x18\x00\x01\x19\x35\x00\x00\x00\x01" +
	"\x00\x01\x19\x37\x00\x01\x19\x38\x00\x00\x00\x01\x00\x01\x19\x3b\x00\x01\x19\x46\x00\x00" +
	"\x00\x01\x00\x01\x19\x50\x00\x01\x19\x59\x00\x00\x00\x01\x00\x01\x19\xa0\x00\x01\x19\xa7" +
	"\x00\x00\x00\x01\x00\x01\x19\xaa\x00\x01\x19\xd7\x00\x00\x00\x01\x00\x01\x19\xda\x00\x01" +
	"\x19\xe4\x00\x00\x00\x01\x00\x01\x1a\x00\x00\x01\x1a\x47\x00\x00\x00\x01\x00\x01\x1a\x50" +
	"\x00\x01\x1a\xa2\x00\x00\x00\x01\x00\x01\x1a\xb0\x00\x01\x1a\xf8\x00\x00\x00\x01\x00\x01" +
	"\x1b\x00\x00\x01\x1b\x09\x00\x00\x00\x01\x00\x01\x1c\x00\x00\x01\x1c\x08\x00\x00\x00\x01" +
	"\x00\x01\x1c\x0a\x00\x01\x1c\x36\x00\x00\x00\x01\x00\x01\x1c\x38\x00\x01\x1c\x45\x00\x00" +
	"\x00\x01\x00\x01\x1c\x50\x00\x01\x1c\x6c\x00\x00\x00\x01\x00\x01\x1c\x70\x00\x01\x1c\x8f" +
	"\x00\x00\x00\x01\x00\x01\x1c\x92\x00\x01\x1c\xa7\x00\x00\x00\x01\x00\x01\x1c\xa9\x00\x01" +
	"\x1c\xb6\x00\x00\x00\x01\x00\x01\x1d\x00\x00\x01\x1d\x06\x00\x00\x00\x01\x00\x01\x1d\x08" +
	"\x00\x01\x1d\x09\x00\x00\x00\x01\x00\x01\x1d\x0b\x00\x01\x1d\x36\x00\x00\x00\x01\x00\x01" +
	"\x1d\x3a\x00\x01\x1d\x3c\x00\x00\x00\x02\x00\x01\x1d\x3d\x00\x01\x1d\x3f\x00\x00\x00\x02" +
	"\x00\x01\x1d\x40\x00\x01\x1d\x47\x00\x00\x00\x01\x00\x01\x1d\x50\x00\x01\x1d\x59\x00\x00" +
	"\x00\x01\x00\x01\x1d\x60\x00\x01\x1d\x65\x00\x00\x00\x01\x00\x01\x1d\x67\x00\x01\x1d\x68" +
	"\x00\x00\x00\x01\x00\x01\x1d\x6a\x00\x01\x1d\x8e\x00\x00\x00\x01\x00\x01\x1d\x90\x00\x01" +
	"\x1d\x91\x00\x00\x00\x01\x00\x01\x1d\x93\x00\x01\x1d\x98\x00\x00\x00\x01\x00\x01\x1d\xa0" +
	"\x00\x01\x1d\xa9\x00\x00\x00\x01\x00\x01\x1e\xe0\x00\x01\x1e\xf8\x00\x00\x00\x01\x00\x01" +
	"\x1f\x00\x00\x01\x1f\x10\x00\x00\x00\x01\x00\x01\x1f\x12\x00\x01\x1f\x3a\x00\x00\x00\x01" +
	"\x00\x01\x1f\x3e\x00\x01\x1f\x59\x00\x00\x00\x01\x00\x01\x1f\xb0\x00\x01\x1f\xc0\x00\x00" +
	"\x00\x10\x00\x01\x1f\xc1\x00\x01\x1f\xf1\x00\x00\x00\x01\x00\x01\x1f\xff\x00\x01\x23\x99" +
	"\x00\x00\x00\x01\x00\x01\x24\x00\x00\x01\x24\x6e\x00\x00\x00\x01\x00\x01\x24\x70\x00\x01" +
	"\x24\x74\x00\x00\x00\x01\x00\x01\x24\x80\x00\x01\x25\x43\x00\x00\x00\x01\x00\x01\x2f\x90" +
	"\x00\x01\x2f\xf2\x00\x00\x00\x01\x00\x01\x30\x00\x00\x01\x34\x2f\x00\x00\x00\x01\x00\x01" +
	"\x34\x40\x00\x01\x34\x55\x00\x00\x00\x01\x00\x01\x44\x00\x00\x01\x46\x46\x00\x00\x00\x01" +
	"\x00\x01\x68\x00\x00\x01\x6a\x38\x00\x00\x00\x01\x00\x01\x6a\x40\x00\x01\x6a\x5e\x00\x00" +
	"\x00\x01\x00\x01\x6a\x60\x00\x01\x6a\x69\x00\x00\x00\x01\x00\x01\x6a\x6e\x00\x01\x6a\xbe" +
	"\x00\x00\x00\x01\x00\x01\x6a\xc0\x00\x01\x6a\xc9\x00\x00\x00\x01\x00\x01\x6a\xd0\x00\x01" +
	"\x6a\xed\x00\x00\x00\x01\x00\x01\x6a\xf0\x00\x01\x6a\xf5\x00\x00\x00\x01\x00\x01\x6b\x00" +
	"\x00\x01\x6b\x45\x00\x00\x00\x01\x00\x01\x6b\x50\x00\x01\x6b\x59\x00\x00\x00\x01\x00\x01" +
	"\x6b\x5b\x00\x01\x6b\x61\x00\x00\x00\x01\x00\x01\x6b\x63\x00\x01\x6b\x77\x00\x00\x00\x01" +
	"\x00\x01\x6b\x7d\x00\x01\x6b\x8f\x00\x00\x00\x01\x00\x01\x6e\x40\x00\x01\x6e\x9a\x00\x00" +
	"\x00\x01\x00\x01\x6f\x00\x00\x01\x6f\x4a\x00\x00\x00\x01\x00\x01\x6f\x4f\x00\x01\x6f\x87" +
	"\x00\x00\x00\x01\x00\x01\x6f\x8f\x00\x01\x6f\x9f\x00\x00\x00\x01\x00\x01\x6f\xe0\x00\x01" +
	"\x6f\xe4\x00\x00\x00\x01\x00\x01\x6f\xf0\x00\x01\x6f\xf1\x00\x00\x00\x01\x00\x01\x70\x00" +
	"\x00\x01\x87\xf7\x00\x00\x00\x01\x00\x01\x88\x00\x00\x01\x8c\xd5\x00\x00\x00\x01\x00\x01" +
	"\x8d\x00\x00\x01\x8d\x08\x00\x00\x00\x01\x00\x01\xaf\xf0\x00\x01\xaf\xf3\x00\x00\x00\x01" +
	"\x00\x01\xaf\xf5\x00\x01\xaf\xfb\x00\x00\x00\x01\x00\x01\xaf\xfd\x00\x01\xaf\xfe\x00\x00" +
	"\x00\x01\x00\x01\xb0\x00\x00\x01\xb1\x22\x00\x00\x00\x01\x00\x01\xb1\x32\x00\x01\xb1\x50" +
	"\x00\x00\x00\x1e\x00\x01\xb1\x51\x00\x01\xb1\x52\x00\x00\x00\x01\x00\x01\xb1\x55\x00\x01" +
	"\xb1\x64\x00\x00\x00\x0f\x00\x01\xb1\x65\x00\x01\xb1\x67\x00\x00\x00\x01\x00\x01\xb1\x70" +
	"\x00\x01\xb2\xfb\x00\x00\x00\x01\x00\x01\xbc\x00\x00\x01\xbc\x6a\x00\x00\x00\x01\x00\x01" +
	"\xbc\x70\x00\x01\xbc\x7c\x00\x00\x00\x01\x00\x01\xbc\x80\x00\x01\xbc\x88\x00\x00\x00\x01" +
	"\x00\x01\xbc\x90\x00\x01\xbc\x99\x00\x00\x00\x01\x00\x01\xbc\x9c\x00\x01\xbc\x9f\x00\x00" +
	"\x00\x01\x00\x01\xcf\x00\x00\x01\xcf\x2d\x00\x00\x00\x01\x00\x01\xcf\x30\x00\x01\xcf\x46" +
	"\x00\x00\x00\x01\x00\x01\xcf\x50\x00\x01\xcf\xc3\x00\x00\x00\x01\x00\x01\xd0\x00\x00\x01" +
	"\xd0\xf5\x00\x00\x00\x01\x00\x01\xd1\x00\x00\x01\xd1\x26\x00\x00\x00\x01\x00\x01\xd1\x29" +
	"\x00\x01\xd1\x72\x00\x00\x00\x01\x00\x01\xd1\x7b\x00\x01\xd1\xea\x00\x00\x00\x01\x00\x01" +
	"\xd2\x00\x00\x01\xd2\x45\x00\x00\x00\x01\x00\x01\xd2\xc0\x00\x01\xd2\xd3\x00\x00\x00\x01" +
	"\x00\x01\xd2\xe0\x00\x01\xd2\xf3\x00\x00\x00\x01\x00\x01\xd3\x00\x00\x01\xd3\x56\x00\x00" +
	"\x00\x01\x00\x01\xd3\x60\x00\x01\xd3\x78\x00\x00\x00\x01\x00\x01\xd4\x00\x00\x01\xd4\x54" +
	"\x00\x00\x00\x01\x00\x01\xd4\x56\x00\x01\xd4\x9c\x00\x00\x00\x01\x00\x01\xd4\x9e\x00\x01" +
	"\xd4\x9f\x00\x00\x00\x01\x00\x01\xd4\xa2\x00\x01\xd4\xa5\x00\x00\x00\x03\x00\x01\xd4\xa6" +
	"\x00\x01\xd4\xa9\x00\x00\x00\x03\x00\x01\xd4\xaa\x00\x01\xd4\xac\x00\x00\x00\x01\x00\x01" +
	"\xd4\xae\x00\x01\xd4\xb9\x00\x00\x00\x01\x00\x01\xd4\xbb\x00\x01\xd4\xbd\x00\x00\x00\x02" +
	"\x00\x01\xd4\xbe\x00\x01\xd4\xc3\x00\x00\x00\x01\x00\x01\xd4\xc5\x00\x01\xd5\x05\x00\x00" +
	"\x00\x01\x00\x01\xd5\x07\x00\x01\xd5\x0a\x00\x00\x00\x01\x00\x01\xd5\x0d\x00\x01\xd5\x14" +
	"\x00\x00\x00\x01\x00\x01\xd5\x16\x00\x01\xd5\x1c\x00\x00\x00\x01\x00\x01\xd5\x1e\x00\x01" +
	"\xd5\x39\x00\x00\x00\x01\x00\x01\xd5\x3b\x00\x01\xd5\x3e\x00\x00\x00\x01\x00\x01\xd5\x40" +
	"\x00\x01\xd5\x44\x00\x00\x00\x01\x00\x01\xd5\x46\x00\x01\xd5\x4a\x00\x00\x00\x04\x00\x01" +
	"\xd5\x4b\x00\x01\xd5\x50\x00\x00\x00\x01\x00\x01\xd5\x52\x00\x01\xd6\xa5\x00\x00\x00\x01" +
	"\x00\x01\xd6\xa8\x00\x01\xd7\xcb\x00\x00\x00\x01\x00\x01\xd7\xce\x00\x01\xda\x8b\x00\x00" +
	"\x00\x01\x00\x01\xda\x9b\x00\x01\xda\x9f\x00\x00\x00\x01\x00\x01\xda\xa1\x00\x01\xda\xaf" +
	"\x00\x00\x00\x01\x00\x01\xdf\x00\x00\x01\xdf\x1e\x00\x00\x00\x01\x00\x01\xdf\x25\x00\x01" +
	"\xdf\x2a\x00\x00\x00\x01\x00\x01\xe0\x00\x00\x01\xe0\x06\x00\x00\x00\x01\x00\x01\xe0\x08" +
	"\x00\x01\xe0\x18\x00\x00\x00\x01\x00\x01\xe0\x1b\x00\x01\xe0\x21\x00\x00\x00\x01\x00\x01" +
	"\xe0\x23\x00\x01\xe0\x24\x00\x00\x00\x01\x00\x01\xe0\x26\x00\x01\xe0\x2a\x00\x00\x00\x01" +
	"\x00\x01\xe0\x30\x00\x01\xe0\x6d\x00\x00\x00\x01\x00\x01\xe0\x8f\x00\x01\xe1\x00\x00\x00" +
	"\x00\x71\x00\x01\xe1\x01\x00\x01\xe1\x2c\x00\x00\x00\x01\x00\x01\xe1\x30\x00\x01\xe1\x3d" +
	"\x00\x00\x00\x01\x00\x01\xe1\x40\x00\x01\xe1\x49\x00\x00\x00\x01\x00\x01\xe1\x4e\x00\x01" +
	"\xe1\x4f\x00\x00\x00\x01\x00\x01\xe2\x90\x00\x01\xe2\xae\x00\x00\x00\x01\x00\x01\xe2\xc0" +
	"\x00\x01\xe2\xf9\x00\x00\x00\x01\x00\x01\xe2\xff\x00\x01\xe4\xd0\x00\x00\x01\xd1\x00\x01" +
	"\xe4\xd1\x00\x01\xe4\xf9\x00\x00\x00\x01\x00\x01\xe7\xe0\x00\x01\xe7\xe6\x00\x00\x00\x01" +
	"\x00\x01\xe7\xe8\x00\x01\xe7\xeb\x00\x00\x00\x01\x00\x01\xe7\xed\x00\x01\xe7\xee\x00\x00" +
	"\x00\x01\x00\x01\xe7\xf0\x00\x01\xe7\xfe\x00\x00\x00\x01\x00\x01\xe8\x00\x00\x01\xe8\xc4" +
	"\x00\x00\x00\x01\x00\x01\xe8\xc7\x00\x01\xe8\xd6\x00\x00\x00\x01\x00\x01\xe9\x00\x00\x01" +
	"\xe9\x4b\x00\x00\x00\x01\x00\x01\xe9\x50\x00\x01\xe9\x59\x00\x00\x00\x01\x00\x01\xe9\x5e" +
	"\x00\x01\xe9\x5f\x00\x00\x00\x01\x00\x01\xec\x71\x00\x01\xec\xb4\x00\x00\x00\x01\x00\x01" +
	"\xed\x01\x00\x01\xed\x3d\x00\x00\x00\x01\x00\x01\xee\x00\x00\x01\xee\x03\x00\x00\x00\x01" +
	"\x00\x01\xee\x05\x00\x01\xee\x1f\x00\x00\x00\x01\x00\x01\xee\x21\x00\x01\xee\x22\x00\x00" +
	"\x00\x01\x00\x01\xee\x24\x00\x01\xee\x27\x00\x00\x00\x03\x00\x01\xee\x29\x00\x01\xee\x32" +
	"\x00\x00\x00\x01\x00\x01\xee\x34\x00\x01\xee\x37\x00\x00\x00\x01\x00\x01\xee\x39\x00\x01" +
	"\xee\x3b\x00\x00\x00\x02\x00\x01\xee\x42\x00\x01\xee\x47\x00\x00\x00\x05\x00\x01\xee\x49" +
	"\x00\x01\xee\x4d\x00\x00\x00\x02\x00\x01\xee\x4e\x00\x01\xee\x4f\x00\x00\x00\x01\x00\x01" +
	"\xee\x51\x00\x01\xee\x52\x00\x00\x00\x01\x00\x01\xee\x54\x00\x01\xee\x57\x00\x00\x00\x03" +
	"\x00\x01\xee\x59\x00\x01\xee\x61\x00\x00\x00\x02\x00\x01\xee\x62\x00\x01\xee\x64\x00\x00" +
	"\x00\x02\x00\x01\xee\x67\x00\x01\xee\x6a\x00\x00\x00\x01\x00\x01\xee\x6c\x00\x01\xee\x72" +
	"\x00\x00\x00\x01\x00\x01\xee\x74\x00\x01\xee\x77\x00\x00\x00\x01\x00\x01\xee\x79\x00\x01" +
	"\xee\x7c\x00\x00\x00\x01\x00\x01\xee\x7e\x00\x01\xee\x80\x00\x00\x00\x02\x00\x01\xee\x81" +
	"\x00\x01\xee\x89\x00\x00\x00\x01\x00\x01\xee\x8b\x00\x01\xee\x9b\x00\x00\x00\x01\x00\x01" +
	"\xee\xa1\x00\x01\xee\xa3\x00\x00\x00\x01\x00\x01\xee\xa5\x00\x01\xee\xa9\x00\x00\x00\x01" +
	"\x00\x01\xee\xab\x00\x01\xee\xbb\x00\x00\x00\x01\x00\x01\xee\xf0\x00\x01\xee\xf1\x00\x00" +
	"\x00\x01\x00\x01\xf0\x00\x00\x01\xf0\x2b\x00\x00\x00\x01\x00\x01\xf0\x30\x00\x01\xf0\x93" +
	"\x00\x00\x00\x01\x00\x01\xf0\xa0\x00\x01\xf0\xae\x00\x00\x00\x01\x00\x01\xf0\xb1\x00\x01" +
	"\xf0\xbf\x00\x00\x00\x01\x00\x01\xf0\xc1\x00\x01\xf0\xcf\x00\x00\x00\x01\x00\x01\xf0\xd1" +
	"\x00\x01\xf0\xf5\x00\x00\x00\x01\x00\x01\xf1\x00\x00\x01\xf1\xad\x00\x00\x00\x01\x00\x01" +
	"\xf1\xe6\x00\x01\xf2\x02\x00\x00\x00\x01\x00\x01\xf2\x10\x00\x01\xf2\x3b\x00\x00\x00\x01" +
	"\x00\x01\xf2\x40\x00\x01\xf2\x48\x00\x00\x00\x01\x00\x01\xf2\x50\x00\x01\xf2\x51\x00\x00" +
	"\x00\x01\x00\x01\xf2\x60\x00\x01\xf2\x65\x00\x00\x00\x01\x00\x01\xf3\x00\x00\x01\xf6\xd7" +
	"\x00\x00\x00\x01\x00\x01\xf6\xdc\x00\x01\xf6\xec\x00\x00\x00\x01\x00\x01\xf6\xf0\x00\x01" +
	"\xf6\xfc\x00\x00\x00\x01\x00\x01\xf7\x00\x00\x01\xf7\x76\x00\x00\x00\x01\x00\x01\xf7\x7b" +
	"\x00\x01\xf7\xd9\x00\x00\x00\x01\x00\x01\xf7\xe0\x00\x01\xf7\xeb\x00\x00\x00\x01\x00\x01" +
	"\xf7\xf0\x00\x01\xf8\x00\x00\x00\x00\x10\x00\x01\xf8\x01\x00\x01\xf8\x0b\x00\x00\x00\x01" +
	"\x00\x01\xf8\x10\x00\x01\xf8\x47\x00\x00\x00\x01\x00\x01\xf8\x50\x00\x01\xf8\x59\x00\x00" +
	"\x00\x01\x00\x01\xf8\x60\x00\x01\xf8\x87\x00\x00\x00\x01\x00\x01\xf8\x90\x00\x01\xf8\xad" +
	"\x00\x00\x00\x01\x00\x01\xf8\xb0\x00\x01\xf8\xb1\x00\x00\x00\x01\x00\x01\xf9\x00\x00\x01" +
	"\xfa\x53\x00\x00\x00\x01\x00\x01\xfa\x60\x00\x01\xfa\x6d\x00\x00\x00\x01\x00\x01\xfa\x70" +
	"\x00\x01\xfa\x7c\x00\x00\x00\x01\x00\x01\xfa\x80\x00\x01\xfa\x88\x00\x00\x00\x01\x00\x01" +
	"\xfa\x90\x00\x01\xfa\xbd\x00\x00\x00\x01\x00\x01\xfa\xbf\x00\x01\xfa\xc5\x00\x00\x00\x01" +
	"\x00\x01\xfa\xce\x00\x01\xfa\xdb\x00\x00\x00\x01\x00\x01\xfa\xe0\x00\x01\xfa\xe8\x00\x00" +
	"\x00\x01\x00\x01\xfa\xf0\x00\x01\xfa\xf8\x00\x00\x00\x01\x00\x01\xfb\x00\x00\x01\xfb\x92" +
	"\x00\x00\x00\x01\x00\x01\xfb\x94\x00\x01\xfb\xca\x00\x00\x00\x01\x00\x01\xfb\xf0\x00\x01" +
	"\xfb\xf9\x00\x00\x00\x01\x00\x02\x00\x00\x00\x02\xa6\xdf\x00\x00\x00\x01\x00\x02\xa7\x00" +
	"\x00\x02\xb7\x39\x00\x00\x00\x01\x00\x02\xb7\x40\x00\x02\xb8\x1d\x00\x00\x00\x01\x00\x02" +
	"\xb8\x20\x00\x02\xce\xa1\x00\x00\x00\x01\x00\x02\xce\xb0\x00\x02\xeb\xe0\x00\x00\x00\x01" +
	"\x00\x02\xf8\x00\x00\x02\xfa\x1d\x00\x00\x00\x01\x00\x03\x00\x00\x00\x03\x13\x4a\x00\x00" +
	"\x00\x01\x00\x03\x13\x50\x00\x03\x23\xaf\x00\x00\x00\x01\x00\x0e\x01\x00\x00\x0e\x01\xef" +
	"\x00\x00\x00\x01"

// GRAPHIC_RANGES_16_DATA avoids hexadecimal decoding for 16-bit graphic searches.
const GRAPHIC_RANGES_16_DATA Encoded_Data = "" +
	"\x00\x20\x00\x7e\x00\x01\x00\xa0\x00\xac\x00\x01\x00\xae\x03\x77\x00\x01\x03\x7a\x03\x7f" +
	"\x00\x01\x03\x84\x03\x8a\x00\x01\x03\x8c\x03\x8e\x00\x02\x03\x8f\x03\xa1\x00\x01\x03\xa3" +
	"\x05\x2f\x00\x01\x05\x31\x05\x56\x00\x01\x05\x59\x05\x8a\x00\x01\x05\x8d\x05\x8f\x00\x01" +
	"\x05\x91\x05\xc7\x00\x01\x05\xd0\x05\xea\x00\x01\x05\xef\x05\xf4\x00\x01\x06\x06\x06\x1b" +
	"\x00\x01\x06\x1d\x06\xdc\x00\x01\x06\xde\x07\x0d\x00\x01\x07\x10\x07\x4a\x00\x01\x07\x4d" +
	"\x07\xb1\x00\x01\x07\xc0\x07\xfa\x00\x01\x07\xfd\x08\x2d\x00\x01\x08\x30\x08\x3e\x00\x01" +
	"\x08\x40\x08\x5b\x00\x01\x08\x5e\x08\x60\x00\x02\x08\x61\x08\x6a\x00\x01\x08\x70\x08\x8e" +
	"\x00\x01\x08\x98\x08\xe1\x00\x01\x08\xe3\x09\x83\x00\x01\x09\x85\x09\x8c\x00\x01\x09\x8f" +
	"\x09\x90\x00\x01\x09\x93\x09\xa8\x00\x01\x09\xaa\x09\xb0\x00\x01\x09\xb2\x09\xb6\x00\x04" +
	"\x09\xb7\x09\xb9\x00\x01\x09\xbc\x09\xc4\x00\x01\x09\xc7\x09\xc8\x00\x01\x09\xcb\x09\xce" +
	"\x00\x01\x09\xd7\x09\xdc\x00\x05\x09\xdd\x09\xdf\x00\x02\x09\xe0\x09\xe3\x00\x01\x09\xe6" +
	"\x09\xfe\x00\x01\x0a\x01\x0a\x03\x00\x01\x0a\x05\x0a\x0a\x00\x01\x0a\x0f\x0a\x10\x00\x01" +
	"\x0a\x13\x0a\x28\x00\x01\x0a\x2a\x0a\x30\x00\x01\x0a\x32\x0a\x33\x00\x01\x0a\x35\x0a\x36" +
	"\x00\x01\x0a\x38\x0a\x39\x00\x01\x0a\x3c\x0a\x3e\x00\x02\x0a\x3f\x0a\x42\x00\x01\x0a\x47" +
	"\x0a\x48\x00\x01\x0a\x4b\x0a\x4d\x00\x01\x0a\x51\x0a\x59\x00\x08\x0a\x5a\x0a\x5c\x00\x01" +
	"\x0a\x5e\x0a\x66\x00\x08\x0a\x67\x0a\x76\x00\x01\x0a\x81\x0a\x83\x00\x01\x0a\x85\x0a\x8d" +
	"\x00\x01\x0a\x8f\x0a\x91\x00\x01\x0a\x93\x0a\xa8\x00\x01\x0a\xaa\x0a\xb0\x00\x01\x0a\xb2" +
	"\x0a\xb3\x00\x01\x0a\xb5\x0a\xb9\x00\x01\x0a\xbc\x0a\xc5\x00\x01\x0a\xc7\x0a\xc9\x00\x01" +
	"\x0a\xcb\x0a\xcd\x00\x01\x0a\xd0\x0a\xe0\x00\x10\x0a\xe1\x0a\xe3\x00\x01\x0a\xe6\x0a\xf1" +
	"\x00\x01\x0a\xf9\x0a\xff\x00\x01\x0b\x01\x0b\x03\x00\x01\x0b\x05\x0b\x0c\x00\x01\x0b\x0f" +
	"\x0b\x10\x00\x01\x0b\x13\x0b\x28\x00\x01\x0b\x2a\x0b\x30\x00\x01\x0b\x32\x0b\x33\x00\x01" +
	"\x0b\x35\x0b\x39\x00\x01\x0b\x3c\x0b\x44\x00\x01\x0b\x47\x0b\x48\x00\x01\x0b\x4b\x0b\x4d" +
	"\x00\x01\x0b\x55\x0b\x57\x00\x01\x0b\x5c\x0b\x5d\x00\x01\x0b\x5f\x0b\x63\x00\x01\x0b\x66" +
	"\x0b\x77\x00\x01\x0b\x82\x0b\x83\x00\x01\x0b\x85\x0b\x8a\x00\x01\x0b\x8e\x0b\x90\x00\x01" +
	"\x0b\x92\x0b\x95\x00\x01\x0b\x99\x0b\x9a\x00\x01\x0b\x9c\x0b\x9e\x00\x02\x0b\x9f\x0b\xa3" +
	"\x00\x04\x0b\xa4\x0b\xa8\x00\x04\x0b\xa9\x0b\xaa\x00\x01\x0b\xae\x0b\xb9\x00\x01\x0b\xbe" +
	"\x0b\xc2\x00\x01\x0b\xc6\x0b\xc8\x00\x01\x0b\xca\x0b\xcd\x00\x01\x0b\xd0\x0b\xd7\x00\x07" +
	"\x0b\xe6\x0b\xfa\x00\x01\x0c\x00\x0c\x0c\x00\x01\x0c\x0e\x0c\x10\x00\x01\x0c\x12\x0c\x28" +
	"\x00\x01\x0c\x2a\x0c\x39\x00\x01\x0c\x3c\x0c\x44\x00\x01\x0c\x46\x0c\x48\x00\x01\x0c\x4a" +
	"\x0c\x4d\x00\x01\x0c\x55\x0c\x56\x00\x01\x0c\x58\x0c\x5a\x00\x01\x0c\x5d\x0c\x60\x00\x03" +
	"\x0c\x61\x0c\x63\x00\x01\x0c\x66\x0c\x6f\x00\x01\x0c\x77\x0c\x8c\x00\x01\x0c\x8e\x0c\x90" +
	"\x00\x01\x0c\x92\x0c\xa8\x00\x01\x0c\xaa\x0c\xb3\x00\x01\x0c\xb5\x0c\xb9\x00\x01\x0c\xbc" +
	"\x0c\xc4\x00\x01\x0c\xc6\x0c\xc8\x00\x01\x0c\xca\x0c\xcd\x00\x01\x0c\xd5\x0c\xd6\x00\x01" +
	"\x0c\xdd\x0c\xde\x00\x01\x0c\xe0\x0c\xe3\x00\x01\x0c\xe6\x0c\xef\x00\x01\x0c\xf1\x0c\xf3" +
	"\x00\x01\x0d\x00\x0d\x0c\x00\x01\x0d\x0e\x0d\x10\x00\x01\x0d\x12\x0d\x44\x00\x01\x0d\x46" +
	"\x0d\x48\x00\x01\x0d\x4a\x0d\x4f\x00\x01\x0d\x54\x0d\x63\x00\x01\x0d\x66\x0d\x7f\x00\x01" +
	"\x0d\x81\x0d\x83\x00\x01\x0d\x85\x0d\x96\x00\x01\x0d\x9a\x0d\xb1\x00\x01\x0d\xb3\x0d\xbb" +
	"\x00\x01\x0d\xbd\x0d\xc0\x00\x03\x0d\xc1\x0d\xc6\x00\x01\x0d\xca\x0d\xcf\x00\x05\x0d\xd0" +
	"\x0d\xd4\x00\x01\x0d\xd6\x0d\xd8\x00\x02\x0d\xd9\x0d\xdf\x00\x01\x0d\xe6\x0d\xef\x00\x01" +
	"\x0d\xf2\x0d\xf4\x00\x01\x0e\x01\x0e\x3a\x00\x01\x0e\x3f\x0e\x5b\x00\x01\x0e\x81\x0e\x82" +
	"\x00\x01\x0e\x84\x0e\x86\x00\x02\x0e\x87\x0e\x8a\x00\x01\x0e\x8c\x0e\xa3\x00\x01\x0e\xa5" +
	"\x0e\xa7\x00\x02\x0e\xa8\x0e\xbd\x00\x01\x0e\xc0\x0e\xc4\x00\x01\x0e\xc6\x0e\xc8\x00\x02" +
	"\x0e\xc9\x0e\xce\x00\x01\x0e\xd0\x0e\xd9\x00\x01\x0e\xdc\x0e\xdf\x00\x01\x0f\x00\x0f\x47" +
	"\x00\x01\x0f\x49\x0f\x6c\x00\x01\x0f\x71\x0f\x97\x00\x01\x0f\x99\x0f\xbc\x00\x01\x0f\xbe" +
	"\x0f\xcc\x00\x01\x0f\xce\x0f\xda\x00\x01\x10\x00\x10\xc5\x00\x01\x10\xc7\x10\xcd\x00\x06" +
	"\x10\xd0\x12\x48\x00\x01\x12\x4a\x12\x4d\x00\x01\x12\x50\x12\x56\x00\x01\x12\x58\x12\x5a" +
	"\x00\x02\x12\x5b\x12\x5d\x00\x01\x12\x60\x12\x88\x00\x01\x12\x8a\x12\x8d\x00\x01\x12\x90" +
	"\x12\xb0\x00\x01\x12\xb2\x12\xb5\x00\x01\x12\xb8\x12\xbe\x00\x01\x12\xc0\x12\xc2\x00\x02" +
	"\x12\xc3\x12\xc5\x00\x01\x12\xc8\x12\xd6\x00\x01\x12\xd8\x13\x10\x00\x01\x13\x12\x13\x15" +
	"\x00\x01\x13\x18\x13\x5a\x00\x01\x13\x5d\x13\x7c\x00\x01\x13\x80\x13\x99\x00\x01\x13\xa0" +
	"\x13\xf5\x00\x01\x13\xf8\x13\xfd\x00\x01\x14\x00\x16\x9c\x00\x01\x16\xa0\x16\xf8\x00\x01" +
	"\x17\x00\x17\x15\x00\x01\x17\x1f\x17\x36\x00\x01\x17\x40\x17\x53\x00\x01\x17\x60\x17\x6c" +
	"\x00\x01\x17\x6e\x17\x70\x00\x01\x17\x72\x17\x73\x00\x01\x17\x80\x17\xdd\x00\x01\x17\xe0" +
	"\x17\xe9\x00\x01\x17\xf0\x17\xf9\x00\x01\x18\x00\x18\x0d\x00\x01\x18\x0f\x18\x19\x00\x01" +
	"\x18\x20\x18\x78\x00\x01\x18\x80\x18\xaa\x00\x01\x18\xb0\x18\xf5\x00\x01\x19\x00\x19\x1e" +
	"\x00\x01\x19\x20\x19\x2b\x00\x01\x19\x30\x19\x3b\x00\x01\x19\x40\x19\x44\x00\x04\x19\x45" +
	"\x19\x6d\x00\x01\x19\x70\x19\x74\x00\x01\x19\x80\x19\xab\x00\x01\x19\xb0\x19\xc9\x00\x01" +
	"\x19\xd0\x19\xda\x00\x01\x19\xde\x1a\x1b\x00\x01\x1a\x1e\x1a\x5e\x00\x01\x1a\x60\x1a\x7c" +
	"\x00\x01\x1a\x7f\x1a\x89\x00\x01\x1a\x90\x1a\x99\x00\x01\x1a\xa0\x1a\xad\x00\x01\x1a\xb0" +
	"\x1a\xce\x00\x01\x1b\x00\x1b\x4c\x00\x01\x1b\x50\x1b\x7e\x00\x01\x1b\x80\x1b\xf3\x00\x01" +
	"\x1b\xfc\x1c\x37\x00\x01\x1c\x3b\x1c\x49\x00\x01\x1c\x4d\x1c\x88\x00\x01\x1c\x90\x1c\xba" +
	"\x00\x01\x1c\xbd\x1c\xc7\x00\x01\x1c\xd0\x1c\xfa\x00\x01\x1d\x00\x1f\x15\x00\x01\x1f\x18" +
	"\x1f\x1d\x00\x01\x1f\x20\x1f\x45\x00\x01\x1f\x48\x1f\x4d\x00\x01\x1f\x50\x1f\x57\x00\x01" +
	"\x1f\x59\x1f\x5f\x00\x02\x1f\x60\x1f\x7d\x00\x01\x1f\x80\x1f\xb4\x00\x01\x1f\xb6\x1f\xc4" +
	"\x00\x01\x1f\xc6\x1f\xd3\x00\x01\x1f\xd6\x1f\xdb\x00\x01\x1f\xdd\x1f\xef\x00\x01\x1f\xf2" +
	"\x1f\xf4\x00\x01\x1f\xf6\x1f\xfe\x00\x01\x20\x00\x20\x0a\x00\x01\x20\x10\x20\x27\x00\x01" +
	"\x20\x2f\x20\x5f\x00\x01\x20\x70\x20\x71\x00\x01\x20\x74\x20\x8e\x00\x01\x20\x90\x20\x9c" +
	"\x00\x01\x20\xa0\x20\xc0\x00\x01\x20\xd0\x20\xf0\x00\x01\x21\x00\x21\x8b\x00\x01\x21\x90" +
	"\x24\x26\x00\x01\x24\x40\x24\x4a\x00\x01\x24\x60\x2b\x73\x00\x01\x2b\x76\x2b\x95\x00\x01" +
	"\x2b\x97\x2c\xf3\x00\x01\x2c\xf9\x2d\x25\x00\x01\x2d\x27\x2d\x2d\x00\x06\x2d\x30\x2d\x67" +
	"\x00\x01\x2d\x6f\x2d\x70\x00\x01\x2d\x7f\x2d\x96\x00\x01\x2d\xa0\x2d\xa6\x00\x01\x2d\xa8" +
	"\x2d\xae\x00\x01\x2d\xb0\x2d\xb6\x00\x01\x2d\xb8\x2d\xbe\x00\x01\x2d\xc0\x2d\xc6\x00\x01" +
	"\x2d\xc8\x2d\xce\x00\x01\x2d\xd0\x2d\xd6\x00\x01\x2d\xd8\x2d\xde\x00\x01\x2d\xe0\x2e\x5d" +
	"\x00\x01\x2e\x80\x2e\x99\x00\x01\x2e\x9b\x2e\xf3\x00\x01\x2f\x00\x2f\xd5\x00\x01\x2f\xf0" +
	"\x2f\xfb\x00\x01\x30\x00\x30\x3f\x00\x01\x30\x41\x30\x96\x00\x01\x30\x99\x30\xff\x00\x01" +
	"\x31\x05\x31\x2f\x00\x01\x31\x31\x31\x8e\x00\x01\x31\x90\x31\xe3\x00\x01\x31\xf0\x32\x1e" +
	"\x00\x01\x32\x20\xa4\x8c\x00\x01\xa4\x90\xa4\xc6\x00\x01\xa4\xd0\xa6\x2b\x00\x01\xa6\x40" +
	"\xa6\xf7\x00\x01\xa7\x00\xa7\xca\x00\x01\xa7\xd0\xa7\xd1\x00\x01\xa7\xd3\xa7\xd5\x00\x02" +
	"\xa7\xd6\xa7\xd9\x00\x01\xa7\xf2\xa8\x2c\x00\x01\xa8\x30\xa8\x39\x00\x01\xa8\x40\xa8\x77" +
	"\x00\x01\xa8\x80\xa8\xc5\x00\x01\xa8\xce\xa8\xd9\x00\x01\xa8\xe0\xa9\x53\x00\x01\xa9\x5f" +
	"\xa9\x7c\x00\x01\xa9\x80\xa9\xcd\x00\x01\xa9\xcf\xa9\xd9\x00\x01\xa9\xde\xa9\xfe\x00\x01" +
	"\xaa\x00\xaa\x36\x00\x01\xaa\x40\xaa\x4d\x00\x01\xaa\x50\xaa\x59\x00\x01\xaa\x5c\xaa\xc2" +
	"\x00\x01\xaa\xdb\xaa\xf6\x00\x01\xab\x01\xab\x06\x00\x01\xab\x09\xab\x0e\x00\x01\xab\x11" +
	"\xab\x16\x00\x01\xab\x20\xab\x26\x00\x01\xab\x28\xab\x2e\x00\x01\xab\x30\xab\x6b\x00\x01" +
	"\xab\x70\xab\xed\x00\x01\xab\xf0\xab\xf9\x00\x01\xac\x00\xd7\xa3\x00\x01\xd7\xb0\xd7\xc6" +
	"\x00\x01\xd7\xcb\xd7\xfb\x00\x01\xf9\x00\xfa\x6d\x00\x01\xfa\x70\xfa\xd9\x00\x01\xfb\x00" +
	"\xfb\x06\x00\x01\xfb\x13\xfb\x17\x00\x01\xfb\x1d\xfb\x36\x00\x01\xfb\x38\xfb\x3c\x00\x01" +
	"\xfb\x3e\xfb\x40\x00\x02\xfb\x41\xfb\x43\x00\x02\xfb\x44\xfb\x46\x00\x02\xfb\x47\xfb\xc2" +
	"\x00\x01\xfb\xd3\xfd\x8f\x00\x01\xfd\x92\xfd\xc7\x00\x01\xfd\xcf\xfd\xf0\x00\x21\xfd\xf1" +
	"\xfe\x19\x00\x01\xfe\x20\xfe\x52\x00\x01\xfe\x54\xfe\x66\x00\x01\xfe\x68\xfe\x6b\x00\x01" +
	"\xfe\x70\xfe\x74\x00\x01\xfe\x76\xfe\xfc\x00\x01\xff\x01\xff\xbe\x00\x01\xff\xc2\xff\xc7" +
	"\x00\x01\xff\xca\xff\xcf\x00\x01\xff\xd2\xff\xd7\x00\x01\xff\xda\xff\xdc\x00\x01\xff\xe0" +
	"\xff\xe6\x00\x01\xff\xe8\xff\xee\x00\x01\xff\xfc\xff\xfd\x00\x01"

// GRAPHIC_RANGES_32_DATA reuses the identical 32-bit print ranges.
const GRAPHIC_RANGES_32_DATA Encoded_Data = PRINT_RANGES_32_DATA

// CATEGORY_TABLE_DATA stores the Unicode category tables.
const CATEGORY_TABLE_DATA Encoded_Data = "" +
	"0143000015cc012100020000001f0001007f009f000100ad037802cb03790380" +
	"0007038103830001038b038d000203a20530018e055705580001058b058c0001" +
	"059005c8003805c905cf000105eb05ee000105f506050001061c06dd00c1070e" +
	"070f0001074b074c000107b207bf000107fb07fc0001082e082f0001083f085c" +
	"001d085d085f0002086b086f0001088f0897000108e2098400a2098d098e0001" +
	"09910992000109a909b1000809b309b5000109ba09bb000109c509c6000109c9" +
	"09ca000109cf09d6000109d809db000109de09e4000609e509ff001a0a000a04" +
	"00040a0b0a0e00010a110a1200010a290a3100080a340a3a00030a3b0a3d0002" +
	"0a430a4600010a490a4a00010a4e0a5000010a520a5800010a5d0a5f00020a60" +
	"0a6500010a770a8000010a840a8e000a0a920aa900170ab10ab400030aba0abb" +
	"00010ac60ace00040acf0ad100020ad20adf00010ae40ae500010af20af80001" +
	"0b000b0400040b0d0b0e00010b110b1200010b290b3100080b340b3a00060b3b" +
	"0b45000a0b460b4900030b4a0b4e00040b4f0b5400010b580b5b00010b5e0b64" +
	"00060b650b7800130b790b8100010b840b8b00070b8c0b8d00010b910b960005" +
	"0b970b9800010b9b0b9d00020ba00ba200010ba50ba700010bab0bad00010bba" +
	"0bbd00010bc30bc500010bc90bce00050bcf0bd100020bd20bd600010bd80be5" +
	"00010bfb0bff00010c0d0c1100040c290c3a00110c3b0c45000a0c490c4e0005" +
	"0c4f0c5400010c570c5b00040c5c0c5e00020c5f0c6400050c650c70000b0c71" +
	"0c7600010c8d0c9100040ca90cb4000b0cba0cbb00010cc50cc900040cce0cd4" +
	"00010cd70cdc00010cdf0ce400050ce50cf0000b0cf40cff00010d0d0d110004" +
	"0d450d4900040d500d5300010d640d6500010d800d8400040d970d9900010db2" +
	"0dbc000a0dbe0dbf00010dc70dc900010dcb0dce00010dd50dd700020de00de5" +
	"00010df00df100010df50e0000010e3b0e3e00010e5c0e8000010e830e850002" +
	"0e8b0ea400190ea60ebe00180ebf0ec500060ec70ecf00080eda0edb00010ee0" +
	"0eff00010f480f6d00250f6e0f7000010f980fbd00250fcd0fdb000e0fdc0fff" +
	"000110c610c8000210c910cc000110ce10cf00011249124e0005124f12570008" +
	"1259125e0005125f1289002a128e128f000112b112b6000512b712bf000812c1" +
	"12c6000512c712d700101311131600051317135b0044135c137d0021137e137f" +
	"0001139a139f000113f613f7000113fe13ff0001169d169f000116f916ff0001" +
	"1716171e00011737173f00011754175f0001176d177100041774177f000117de" +
	"17df000117ea17ef000117fa17ff0001180e181a000c181b181f00011879187f" +
	"000118ab18af000118f618ff0001191f192c000d192d192f0001193c193f0001" +
	"194119430001196e196f00011975197f000119ac19af000119ca19cf000119db" +
	"19dd00011a1c1a1d00011a5f1a7d001e1a7e1a8a000c1a8b1a8f00011a9a1a9f" +
	"00011aae1aaf00011acf1aff00011b4d1b4f00011b7f1bf400751bf51bfb0001" +
	"1c381c3a00011c4a1c4c00011c891c8f00011cbb1cbc00011cc81ccf00011cfb" +
	"1cff00011f161f1700011f1e1f1f00011f461f4700011f4e1f4f00011f581f5e" +
	"00021f7e1f7f00011fb51fc500101fd41fd500011fdc1ff000141ff11ff50004" +
	"1fff200b000c200c200f0001202a202e00012060206f0001207220730001208f" +
	"209d000e209e209f000120c120cf000120f120ff0001218c218f00012427243f" +
	"0001244b245f00012b742b7500012b962cf4015e2cf52cf800012d262d280002" +
	"2d292d2c00012d2e2d2f00012d682d6e00012d712d7e00012d972d9f00012da7" +
	"2ddf00082e5e2e7f00012e9a2ef4005a2ef52eff00012fd62fef00012ffc2fff" +
	"00013040309700573098310000683101310400013130318f005f31e431ef0001" +
	"321fa48d726ea48ea48f0001a4c7a4cf0001a62ca63f0001a6f8a6ff0001a7cb" +
	"a7cf0001a7d2a7d40002a7daa7f10001a82da82f0001a83aa83f0001a878a87f" +
	"0001a8c6a8cd0001a8daa8df0001a954a95e0001a97da97f0001a9cea9da000c" +
	"a9dba9dd0001a9ffaa370038aa38aa3f0001aa4eaa4f0001aa5aaa5b0001aac3" +
	"aada0001aaf7ab000001ab07ab080001ab0fab100001ab17ab1f0001ab27ab2f" +
	"0008ab6cab6f0001abeeabef0001abfaabff0001d7a4d7af0001d7c7d7ca0001" +
	"d7fcf8ff0001fa6efa6f0001fadafaff0001fb07fb120001fb18fb1c0001fb37" +
	"fb3d0006fb3ffb450003fbc3fbd20001fd90fd910001fdc8fdce0001fdd0fdef" +
	"0001fe1afe1f0001fe53fe670014fe6cfe6f0001fe75fefd0088fefeff000001" +
	"ffbfffc10001ffc8ffc90001ffd0ffd10001ffd8ffd90001ffddffdf0001ffe7" +
	"ffef0008fff0fffb0001fffeffff000101400001000c000100270000001b0001" +
	"003b0001003e000000030001004e0001004f000000010001005e0001007f0000" +
	"0001000100fb000100ff00000001000101030001010600000001000101340001" +
	"0136000000010001018f0001019d0000000e0001019e0001019f000000010001" +
	"01a1000101cf00000001000101fe0001027f000000010001029d0001029f0000" +
	"0001000102d1000102df00000001000102fc000102ff00000001000103240001" +
	"032c000000010001034b0001034f000000010001037b0001037f000000010001" +
	"039e000103c400000026000103c5000103c700000001000103d6000103ff0000" +
	"00010001049e0001049f00000001000104aa000104af00000001000104d40001" +
	"04d700000001000104fc000104ff00000001000105280001052f000000010001" +
	"05640001056e000000010001057b0001058b0000001000010593000105960000" +
	"0003000105a2000105b200000010000105ba000105bd00000003000105be0001" +
	"05ff00000001000107370001073f00000001000107560001075f000000010001" +
	"07680001077f0000000100010786000107b10000002b000107bb000107ff0000" +
	"000100010806000108070000000100010809000108360000002d000108390001" +
	"083b000000010001083d0001083e00000001000108560001089f000000490001" +
	"08a0000108a600000001000108b0000108df00000001000108f3000108f60000" +
	"0003000108f7000108fa000000010001091c0001091e000000010001093a0001" +
	"093e00000001000109400001097f00000001000109b8000109bb000000010001" +
	"09d0000109d10000000100010a0400010a070000000300010a0800010a0b0000" +
	"000100010a1400010a180000000400010a3600010a370000000100010a3b0001" +
	"0a3e0000000100010a4900010a4f0000000100010a5900010a5f000000010001" +
	"0aa000010abf0000000100010ae700010aea0000000100010af700010aff0000" +
	"000100010b3600010b380000000100010b5600010b570000000100010b730001" +
	"0b770000000100010b9200010b980000000100010b9d00010ba8000000010001" +
	"0bb000010bff0000000100010c4900010c7f0000000100010cb300010cbf0000" +
	"000100010cf300010cf90000000100010d2800010d2f0000000100010d3a0001" +
	"0e5f0000000100010e7f00010eaa0000002b00010eae00010eaf000000010001" +
	"0eb200010efc0000000100010f2800010f2f0000000100010f5a00010f6f0000" +
	"000100010f8a00010faf0000000100010fcc00010fdf0000000100010ff70001" +
	"0fff000000010001104e0001105100000001000110760001107e000000010001" +
	"10bd000110c300000006000110c4000110cf00000001000110e9000110ef0000" +
	"0001000110fa000110ff00000001000111350001114800000013000111490001" +
	"114f00000001000111770001117f00000001000111e0000111f5000000150001" +
	"11f6000111ff00000001000112120001124200000030000112430001127f0000" +
	"00010001128700011289000000020001128e0001129e00000010000112aa0001" +
	"12af00000001000112eb000112ef00000001000112fa000112ff000000010001" +
	"13040001130d000000090001130e000113110000000300011312000113290000" +
	"00170001133100011334000000030001133a000113450000000b000113460001" +
	"1349000000030001134a0001134e000000040001134f00011351000000020001" +
	"13520001135600000001000113580001135c0000000100011364000113650000" +
	"00010001136d0001136f0000000100011375000113ff000000010001145c0001" +
	"146200000006000114630001147f00000001000114c8000114cf000000010001" +
	"14da0001157f00000001000115b6000115b700000001000115de000115ff0000" +
	"0001000116450001164f000000010001165a0001165f000000010001166d0001" +
	"167f00000001000116ba000116bf00000001000116ca000116ff000000010001" +
	"171b0001171c000000010001172c0001172f0000000100011747000117ff0000" +
	"00010001183c0001189f00000001000118f3000118fe00000001000119070001" +
	"1908000000010001190a0001190b000000010001191400011917000000030001" +
	"193600011939000000030001193a000119470000000d000119480001194f0000" +
	"00010001195a0001199f00000001000119a8000119a900000001000119d80001" +
	"19d900000001000119e5000119ff0000000100011a4800011a4f000000010001" +
	"1aa300011aaf0000000100011af900011aff0000000100011b0a00011bff0000" +
	"000100011c0900011c370000002e00011c4600011c4f0000000100011c6d0001" +
	"1c6f0000000100011c9000011c910000000100011ca800011cb70000000f0001" +
	"1cb800011cff0000000100011d0700011d0a0000000300011d3700011d390000" +
	"000100011d3b00011d3e0000000300011d4800011d4f0000000100011d5a0001" +
	"1d5f0000000100011d6600011d690000000300011d8f00011d92000000030001" +
	"1d9900011d9f0000000100011daa00011edf0000000100011ef900011eff0000" +
	"000100011f1100011f3b0000002a00011f3c00011f3d0000000100011f5a0001" +
	"1faf0000000100011fb100011fbf0000000100011ff200011ffe000000010001" +
	"239a000123ff000000010001246f0001247500000006000124760001247f0000" +
	"00010001254400012f8f0000000100012ff300012fff00000001000134300001" +
	"343f0000000100013456000143ff0000000100014647000167ff000000010001" +
	"6a3900016a3f0000000100016a5f00016a6a0000000b00016a6b00016a6d0000" +
	"000100016abf00016aca0000000b00016acb00016acf0000000100016aee0001" +
	"6aef0000000100016af600016aff0000000100016b4600016b4f000000010001" +
	"6b5a00016b620000000800016b7800016b7c0000000100016b9000016e3f0000" +
	"000100016e9b00016eff0000000100016f4b00016f4e0000000100016f880001" +
	"6f8e0000000100016fa000016fdf0000000100016fe500016fef000000010001" +
	"6ff200016fff00000001000187f8000187ff0000000100018cd600018cff0000" +
	"000100018d090001afef000000010001aff40001affc000000080001afff0001" +
	"b123000001240001b1240001b131000000010001b1330001b14f000000010001" +
	"b1530001b154000000010001b1560001b163000000010001b1680001b16f0000" +
	"00010001b2fc0001bbff000000010001bc6b0001bc6f000000010001bc7d0001" +
	"bc7f000000010001bc890001bc8f000000010001bc9a0001bc9b000000010001" +
	"bca00001ceff000000010001cf2e0001cf2f000000010001cf470001cf4f0000" +
	"00010001cfc40001cfff000000010001d0f60001d0ff000000010001d1270001" +
	"d128000000010001d1730001d17a000000010001d1eb0001d1ff000000010001" +
	"d2460001d2bf000000010001d2d40001d2df000000010001d2f40001d2ff0000" +
	"00010001d3570001d35f000000010001d3790001d3ff000000010001d4550001" +
	"d49d000000480001d4a00001d4a1000000010001d4a30001d4a4000000010001" +
	"d4a70001d4a8000000010001d4ad0001d4ba0000000d0001d4bc0001d4c40000" +
	"00080001d5060001d50b000000050001d50c0001d515000000090001d51d0001" +
	"d53a0000001d0001d53f0001d545000000060001d5470001d549000000010001" +
	"d5510001d6a6000001550001d6a70001d7cc000001250001d7cd0001da8c0000" +
	"02bf0001da8d0001da9a000000010001daa00001dab0000000100001dab10001" +
	"deff000000010001df1f0001df24000000010001df2b0001dfff000000010001" +
	"e0070001e019000000120001e01a0001e022000000080001e0250001e02b0000" +
	"00060001e02c0001e02f000000010001e06e0001e08e000000010001e0900001" +
	"e0ff000000010001e12d0001e12f000000010001e13e0001e13f000000010001" +
	"e14a0001e14d000000010001e1500001e28f000000010001e2af0001e2bf0000" +
	"00010001e2fa0001e2fe000000010001e3000001e4cf000000010001e4fa0001" +
	"e7df000000010001e7e70001e7ec000000050001e7ef0001e7ff000000100001" +
	"e8c50001e8c6000000010001e8d70001e8ff000000010001e94c0001e94f0000" +
	"00010001e95a0001e95d000000010001e9600001ec70000000010001ecb50001" +
	"ed00000000010001ed3e0001edff000000010001ee040001ee200000001c0001" +
	"ee230001ee25000000020001ee260001ee28000000020001ee330001ee380000" +
	"00050001ee3a0001ee3c000000020001ee3d0001ee41000000010001ee430001" +
	"ee46000000010001ee480001ee4c000000020001ee500001ee53000000030001" +
	"ee550001ee56000000010001ee580001ee60000000020001ee630001ee650000" +
	"00020001ee660001ee6b000000050001ee730001ee7d000000050001ee7f0001" +
	"ee8a0000000b0001ee9c0001eea0000000010001eea40001eeaa000000060001" +
	"eebc0001eeef000000010001eef20001efff000000010001f02c0001f02f0000" +
	"00010001f0940001f09f000000010001f0af0001f0b0000000010001f0c00001" +
	"f0d0000000100001f0f60001f0ff000000010001f1ae0001f1e5000000010001" +
	"f2030001f20f000000010001f23c0001f23f000000010001f2490001f24f0000" +
	"00010001f2520001f25f000000010001f2660001f2ff000000010001f6d80001" +
	"f6db000000010001f6ed0001f6ef000000010001f6fd0001f6ff000000010001" +
	"f7770001f77a000000010001f7da0001f7df000000010001f7ec0001f7ef0000" +
	"00010001f7f10001f7ff000000010001f80c0001f80f000000010001f8480001" +
	"f84f000000010001f85a0001f85f000000010001f8880001f88f000000010001" +
	"f8ae0001f8af000000010001f8b20001f8ff000000010001fa540001fa5f0000" +
	"00010001fa6e0001fa6f000000010001fa7d0001fa7f000000010001fa890001" +
	"fa8f000000010001fabe0001fac6000000080001fac70001facd000000010001" +
	"fadc0001fadf000000010001fae90001faef000000010001faf90001faff0000" +
	"00010001fb930001fbcb000000380001fbcc0001fbef000000010001fbfa0001" +
	"ffff000000010002a6e00002a6ff000000010002b73a0002b73f000000010002" +
	"b81e0002b81f000000010002cea20002ceaf000000010002ebe10002f7ff0000" +
	"00010002fa1e0002ffff000000010003134b0003134f00000001000323b0000e" +
	"00ff00000001000e01f00010ffff000000010243630000001200020002000000" +
	"1f0001007f009f0001000002436600000096000c000000ad0600055306010605" +
	"0001061c06dd00c1070f08900181089108e20051180e200b07fd200c200f0001" +
	"202a202e00012060206400012066206f0001fefffff900fafffafffb00010006" +
	"000110bd000110cd00000010000134300001343f000000010001bca00001bca3" +
	"000000010001d1730001d17a00000001000e0001000e00200000001f000e0021" +
	"000e007f0000000102436e000015ba011a000003780379000103800383000103" +
	"8b038d000203a20530018e055705580001058b058c0001059005c8003805c905" +
	"cf000105eb05ee000105f505ff0001070e074b003d074c07b2006607b307bf00" +
	"0107fb07fc0001082e082f0001083f085c001d085d085f0002086b086f000108" +
	"8f089200030893089700010984098d0009098e09910003099209a9001709b109" +
	"b3000209b409b5000109ba09bb000109c509c6000109c909ca000109cf09d600" +
	"0109d809db000109de09e4000609e509ff001a0a000a0400040a0b0a0e00010a" +
	"110a1200010a290a3100080a340a3a00030a3b0a3d00020a430a4600010a490a" +
	"4a00010a4e0a5000010a520a5800010a5d0a5f00020a600a6500010a770a8000" +
	"010a840a8e000a0a920aa900170ab10ab400030aba0abb00010ac60ace00040a" +
	"cf0ad100020ad20adf00010ae40ae500010af20af800010b000b0400040b0d0b" +
	"0e00010b110b1200010b290b3100080b340b3a00060b3b0b45000a0b460b4900" +
	"030b4a0b4e00040b4f0b5400010b580b5b00010b5e0b6400060b650b7800130b" +
	"790b8100010b840b8b00070b8c0b8d00010b910b9600050b970b9800010b9b0b" +
	"9d00020ba00ba200010ba50ba700010bab0bad00010bba0bbd00010bc30bc500" +
	"010bc90bce00050bcf0bd100020bd20bd600010bd80be500010bfb0bff00010c" +
	"0d0c1100040c290c3a00110c3b0c45000a0c490c4e00050c4f0c5400010c570c" +
	"5b00040c5c0c5e00020c5f0c6400050c650c70000b0c710c7600010c8d0c9100" +
	"040ca90cb4000b0cba0cbb00010cc50cc900040cce0cd400010cd70cdc00010c" +
	"df0ce400050ce50cf0000b0cf40cff00010d0d0d1100040d450d4900040d500d" +
	"5300010d640d6500010d800d8400040d970d9900010db20dbc000a0dbe0dbf00" +
	"010dc70dc900010dcb0dce00010dd50dd700020de00de500010df00df100010d" +
	"f50e0000010e3b0e3e00010e5c0e8000010e830e8500020e8b0ea400190ea60e" +
	"be00180ebf0ec500060ec70ecf00080eda0edb00010ee00eff00010f480f6d00" +
	"250f6e0f7000010f980fbd00250fcd0fdb000e0fdc0fff000110c610c8000210" +
	"c910cc000110ce10cf00011249124e0005124f125700081259125e0005125f12" +
	"89002a128e128f000112b112b6000512b712bf000812c112c6000512c712d700" +
	"101311131600051317135b0044135c137d0021137e137f0001139a139f000113" +
	"f613f7000113fe13ff0001169d169f000116f916ff00011716171e0001173717" +
	"3f00011754175f0001176d177100041774177f000117de17df000117ea17ef00" +
	"0117fa17ff0001181a181f00011879187f000118ab18af000118f618ff000119" +
	"1f192c000d192d192f0001193c193f0001194119430001196e196f0001197519" +
	"7f000119ac19af000119ca19cf000119db19dd00011a1c1a1d00011a5f1a7d00" +
	"1e1a7e1a8a000c1a8b1a8f00011a9a1a9f00011aae1aaf00011acf1aff00011b" +
	"4d1b4f00011b7f1bf400751bf51bfb00011c381c3a00011c4a1c4c00011c891c" +
	"8f00011cbb1cbc00011cc81ccf00011cfb1cff00011f161f1700011f1e1f1f00" +
	"011f461f4700011f4e1f4f00011f581f5e00021f7e1f7f00011fb51fc500101f" +
	"d41fd500011fdc1ff000141ff11ff500041fff20650066207220730001208f20" +
	"9d000e209e209f000120c120cf000120f120ff0001218c218f00012427243f00" +
	"01244b245f00012b742b7500012b962cf4015e2cf52cf800012d262d2800022d" +
	"292d2c00012d2e2d2f00012d682d6e00012d712d7e00012d972d9f00012da72d" +
	"df00082e5e2e7f00012e9a2ef4005a2ef52eff00012fd62fef00012ffc2fff00" +
	"013040309700573098310000683101310400013130318f005f31e431ef000132" +
	"1fa48d726ea48ea48f0001a4c7a4cf0001a62ca63f0001a6f8a6ff0001a7cba7" +
	"cf0001a7d2a7d40002a7daa7f10001a82da82f0001a83aa83f0001a878a87f00" +
	"01a8c6a8cd0001a8daa8df0001a954a95e0001a97da97f0001a9cea9da000ca9" +
	"dba9dd0001a9ffaa370038aa38aa3f0001aa4eaa4f0001aa5aaa5b0001aac3aa" +
	"da0001aaf7ab000001ab07ab080001ab0fab100001ab17ab1f0001ab27ab2f00" +
	"08ab6cab6f0001abeeabef0001abfaabff0001d7a4d7af0001d7c7d7ca0001d7" +
	"fcd7ff0001fa6efa6f0001fadafaff0001fb07fb120001fb18fb1c0001fb37fb" +
	"3d0006fb3ffb450003fbc3fbd20001fd90fd910001fdc8fdce0001fdd0fdef00" +
	"01fe1afe1f0001fe53fe670014fe6cfe6f0001fe75fefd0088fefeff000002ff" +
	"bfffc10001ffc8ffc90001ffd0ffd10001ffd8ffd90001ffddffdf0001ffe7ff" +
	"ef0008fff0fff80001fffeffff000101420001000c000100270000001b000100" +
	"3b0001003e000000030001004e0001004f000000010001005e0001007f000000" +
	"01000100fb000100ff0000000100010103000101060000000100010134000101" +
	"36000000010001018f0001019d0000000e0001019e0001019f00000001000101" +
	"a1000101cf00000001000101fe0001027f000000010001029d0001029f000000" +
	"01000102d1000102df00000001000102fc000102ff0000000100010324000103" +
	"2c000000010001034b0001034f000000010001037b0001037f00000001000103" +
	"9e000103c400000026000103c5000103c700000001000103d6000103ff000000" +
	"010001049e0001049f00000001000104aa000104af00000001000104d4000104" +
	"d700000001000104fc000104ff00000001000105280001052f00000001000105" +
	"640001056e000000010001057b0001058b000000100001059300010596000000" +
	"03000105a2000105b200000010000105ba000105bd00000003000105be000105" +
	"ff00000001000107370001073f00000001000107560001075f00000001000107" +
	"680001077f0000000100010786000107b10000002b000107bb000107ff000000" +
	"0100010806000108070000000100010809000108360000002d00010839000108" +
	"3b000000010001083d0001083e00000001000108560001089f00000049000108" +
	"a0000108a600000001000108b0000108df00000001000108f3000108f6000000" +
	"03000108f7000108fa000000010001091c0001091e000000010001093a000109" +
	"3e00000001000109400001097f00000001000109b8000109bb00000001000109" +
	"d0000109d10000000100010a0400010a070000000300010a0800010a0b000000" +
	"0100010a1400010a180000000400010a3600010a370000000100010a3b00010a" +
	"3e0000000100010a4900010a4f0000000100010a5900010a5f0000000100010a" +
	"a000010abf0000000100010ae700010aea0000000100010af700010aff000000" +
	"0100010b3600010b380000000100010b5600010b570000000100010b7300010b" +
	"770000000100010b9200010b980000000100010b9d00010ba80000000100010b" +
	"b000010bff0000000100010c4900010c7f0000000100010cb300010cbf000000" +
	"0100010cf300010cf90000000100010d2800010d2f0000000100010d3a00010e" +
	"5f0000000100010e7f00010eaa0000002b00010eae00010eaf0000000100010e" +
	"b200010efc0000000100010f2800010f2f0000000100010f5a00010f6f000000" +
	"0100010f8a00010faf0000000100010fcc00010fdf0000000100010ff700010f" +
	"ff000000010001104e0001105100000001000110760001107e00000001000110" +
	"c3000110cc00000001000110ce000110cf00000001000110e9000110ef000000" +
	"01000110fa000110ff0000000100011135000111480000001300011149000111" +
	"4f00000001000111770001117f00000001000111e0000111f500000015000111" +
	"f6000111ff00000001000112120001124200000030000112430001127f000000" +
	"010001128700011289000000020001128e0001129e00000010000112aa000112" +
	"af00000001000112eb000112ef00000001000112fa000112ff00000001000113" +
	"040001130d000000090001130e00011311000000030001131200011329000000" +
	"170001133100011334000000030001133a000113450000000b00011346000113" +
	"49000000030001134a0001134e000000040001134f0001135100000002000113" +
	"520001135600000001000113580001135c000000010001136400011365000000" +
	"010001136d0001136f0000000100011375000113ff000000010001145c000114" +
	"6200000006000114630001147f00000001000114c8000114cf00000001000114" +
	"da0001157f00000001000115b6000115b700000001000115de000115ff000000" +
	"01000116450001164f000000010001165a0001165f000000010001166d000116" +
	"7f00000001000116ba000116bf00000001000116ca000116ff00000001000117" +
	"1b0001171c000000010001172c0001172f0000000100011747000117ff000000" +
	"010001183c0001189f00000001000118f3000118fe0000000100011907000119" +
	"08000000010001190a0001190b00000001000119140001191700000003000119" +
	"3600011939000000030001193a000119470000000d000119480001194f000000" +
	"010001195a0001199f00000001000119a8000119a900000001000119d8000119" +
	"d900000001000119e5000119ff0000000100011a4800011a4f0000000100011a" +
	"a300011aaf0000000100011af900011aff0000000100011b0a00011bff000000" +
	"0100011c0900011c370000002e00011c4600011c4f0000000100011c6d00011c" +
	"6f0000000100011c9000011c910000000100011ca800011cb70000000f00011c" +
	"b800011cff0000000100011d0700011d0a0000000300011d3700011d39000000" +
	"0100011d3b00011d3e0000000300011d4800011d4f0000000100011d5a00011d" +
	"5f0000000100011d6600011d690000000300011d8f00011d920000000300011d" +
	"9900011d9f0000000100011daa00011edf0000000100011ef900011eff000000" +
	"0100011f1100011f3b0000002a00011f3c00011f3d0000000100011f5a00011f" +
	"af0000000100011fb100011fbf0000000100011ff200011ffe00000001000123" +
	"9a000123ff000000010001246f0001247500000006000124760001247f000000" +
	"010001254400012f8f0000000100012ff300012fff0000000100013456000143" +
	"ff0000000100014647000167ff0000000100016a3900016a3f0000000100016a" +
	"5f00016a6a0000000b00016a6b00016a6d0000000100016abf00016aca000000" +
	"0b00016acb00016acf0000000100016aee00016aef0000000100016af600016a" +
	"ff0000000100016b4600016b4f0000000100016b5a00016b620000000800016b" +
	"7800016b7c0000000100016b9000016e3f0000000100016e9b00016eff000000" +
	"0100016f4b00016f4e0000000100016f8800016f8e0000000100016fa000016f" +
	"df0000000100016fe500016fef0000000100016ff200016fff00000001000187" +
	"f8000187ff0000000100018cd600018cff0000000100018d090001afef000000" +
	"010001aff40001affc000000080001afff0001b123000001240001b1240001b1" +
	"31000000010001b1330001b14f000000010001b1530001b154000000010001b1" +
	"560001b163000000010001b1680001b16f000000010001b2fc0001bbff000000" +
	"010001bc6b0001bc6f000000010001bc7d0001bc7f000000010001bc890001bc" +
	"8f000000010001bc9a0001bc9b000000010001bca40001ceff000000010001cf" +
	"2e0001cf2f000000010001cf470001cf4f000000010001cfc40001cfff000000" +
	"010001d0f60001d0ff000000010001d1270001d128000000010001d1eb0001d1" +
	"ff000000010001d2460001d2bf000000010001d2d40001d2df000000010001d2" +
	"f40001d2ff000000010001d3570001d35f000000010001d3790001d3ff000000" +
	"010001d4550001d49d000000480001d4a00001d4a1000000010001d4a30001d4" +
	"a4000000010001d4a70001d4a8000000010001d4ad0001d4ba0000000d0001d4" +
	"bc0001d4c4000000080001d5060001d50b000000050001d50c0001d515000000" +
	"090001d51d0001d53a0000001d0001d53f0001d545000000060001d5470001d5" +
	"49000000010001d5510001d6a6000001550001d6a70001d7cc000001250001d7" +
	"cd0001da8c000002bf0001da8d0001da9a000000010001daa00001dab0000000" +
	"100001dab10001deff000000010001df1f0001df24000000010001df2b0001df" +
	"ff000000010001e0070001e019000000120001e01a0001e022000000080001e0" +
	"250001e02b000000060001e02c0001e02f000000010001e06e0001e08e000000" +
	"010001e0900001e0ff000000010001e12d0001e12f000000010001e13e0001e1" +
	"3f000000010001e14a0001e14d000000010001e1500001e28f000000010001e2" +
	"af0001e2bf000000010001e2fa0001e2fe000000010001e3000001e4cf000000" +
	"010001e4fa0001e7df000000010001e7e70001e7ec000000050001e7ef0001e7" +
	"ff000000100001e8c50001e8c6000000010001e8d70001e8ff000000010001e9" +
	"4c0001e94f000000010001e95a0001e95d000000010001e9600001ec70000000" +
	"010001ecb50001ed00000000010001ed3e0001edff000000010001ee040001ee" +
	"200000001c0001ee230001ee25000000020001ee260001ee28000000020001ee" +
	"330001ee38000000050001ee3a0001ee3c000000020001ee3d0001ee41000000" +
	"010001ee430001ee46000000010001ee480001ee4c000000020001ee500001ee" +
	"53000000030001ee550001ee56000000010001ee580001ee60000000020001ee" +
	"630001ee65000000020001ee660001ee6b000000050001ee730001ee7d000000" +
	"050001ee7f0001ee8a0000000b0001ee9c0001eea0000000010001eea40001ee" +
	"aa000000060001eebc0001eeef000000010001eef20001efff000000010001f0" +
	"2c0001f02f000000010001f0940001f09f000000010001f0af0001f0b0000000" +
	"010001f0c00001f0d0000000100001f0f60001f0ff000000010001f1ae0001f1" +
	"e5000000010001f2030001f20f000000010001f23c0001f23f000000010001f2" +
	"490001f24f000000010001f2520001f25f000000010001f2660001f2ff000000" +
	"010001f6d80001f6db000000010001f6ed0001f6ef000000010001f6fd0001f6" +
	"ff000000010001f7770001f77a000000010001f7da0001f7df000000010001f7" +
	"ec0001f7ef000000010001f7f10001f7ff000000010001f80c0001f80f000000" +
	"010001f8480001f84f000000010001f85a0001f85f000000010001f8880001f8" +
	"8f000000010001f8ae0001f8af000000010001f8b20001f8ff000000010001fa" +
	"540001fa5f000000010001fa6e0001fa6f000000010001fa7d0001fa7f000000" +
	"010001fa890001fa8f000000010001fabe0001fac6000000080001fac70001fa" +
	"cd000000010001fadc0001fadf000000010001fae90001faef000000010001fa" +
	"f90001faff000000010001fb930001fbcb000000380001fbcc0001fbef000000" +
	"010001fbfa0001ffff000000010002a6e00002a6ff000000010002b73a0002b7" +
	"3f000000010002b81e0002b81f000000010002cea20002ceaf000000010002eb" +
	"e10002f7ff000000010002fa1e0002ffff000000010003134b0003134f000000" +
	"01000323b0000e000000000001000e0002000e001f00000001000e0080000e00" +
	"ff00000001000e01f0000effff00000001000ffffe000fffff000000010010ff" +
	"fe0010ffff0000000102436f0000002400010000e000f8ff00010002000f0000" +
	"000ffffd00000001001000000010fffd000000010243730000000c00010000d8" +
	"00dfff00010000014c000014d0016700060041005a00010061007a000100aa00" +
	"b5000b00ba00c0000600c100d6000100d800f6000100f802c1000102c602d100" +
	"0102e002e4000102ec02ee0002037003740001037603770001037a037d000103" +
	"7f038600070388038a0001038c038e0002038f03a1000103a303f5000103f704" +
	"810001048a052f000105310556000105590560000705610588000105d005ea00" +
	"0105ef05f200010620064a0001066e066f0001067106d3000106d506e5001006" +
	"e606ee000806ef06fa000b06fb06fc000106ff071000110712072f0001074d07" +
	"a5000107b107ca001907cb07ea000107f407f5000107fa080000060801081500" +
	"01081a0824000a0828084000180841085800010860086a000108700887000108" +
	"89088e000108a008c90001090409390001093d09500013095809610001097109" +
	"8000010985098c0001098f09900001099309a8000109aa09b0000109b209b600" +
	"0409b709b9000109bd09ce001109dc09dd000109df09e1000109f009f1000109" +
	"fc0a0500090a060a0a00010a0f0a1000010a130a2800010a2a0a3000010a320a" +
	"3300010a350a3600010a380a3900010a590a5c00010a5e0a7200140a730a7400" +
	"010a850a8d00010a8f0a9100010a930aa800010aaa0ab000010ab20ab300010a" +
	"b50ab900010abd0ad000130ae00ae100010af90b05000c0b060b0c00010b0f0b" +
	"1000010b130b2800010b2a0b3000010b320b3300010b350b3900010b3d0b5c00" +
	"1f0b5d0b5f00020b600b6100010b710b8300120b850b8a00010b8e0b9000010b" +
	"920b9500010b990b9a00010b9c0b9e00020b9f0ba300040ba40ba800040ba90b" +
	"aa00010bae0bb900010bd00c0500350c060c0c00010c0e0c1000010c120c2800" +
	"010c2a0c3900010c3d0c58001b0c590c5a00010c5d0c6000030c610c80001f0c" +
	"850c8c00010c8e0c9000010c920ca800010caa0cb300010cb50cb900010cbd0c" +
	"dd00200cde0ce000020ce10cf100100cf20d0400120d050d0c00010d0e0d1000" +
	"010d120d3a00010d3d0d4e00110d540d5600010d5f0d6100010d7a0d7f00010d" +
	"850d9600010d9a0db100010db30dbb00010dbd0dc000030dc10dc600010e010e" +
	"3000010e320e3300010e400e4600010e810e8200010e840e8600020e870e8a00" +
	"010e8c0ea300010ea50ea700020ea80eb000010eb20eb300010ebd0ec000030e" +
	"c10ec400010ec60edc00160edd0edf00010f000f4000400f410f4700010f490f" +
	"6c00010f880f8c00011000102a0001103f10500011105110550001105a105d00" +
	"011061106500041066106e0008106f10700001107510810001108e10a0001210" +
	"a110c5000110c710cd000610d010fa000110fc12480001124a124d0001125012" +
	"5600011258125a0002125b125d0001126012880001128a128d0001129012b000" +
	"0112b212b5000112b812be000112c012c2000212c312c5000112c812d6000112" +
	"d8131000011312131500011318135a00011380138f000113a013f5000113f813" +
	"fd00011401166c0001166f167f00011681169a000116a016ea000116f116f800" +
	"01170017110001171f173100011740175100011760176c0001176e1770000117" +
	"8017b3000117d717dc0005182018780001188018840001188718a8000118aa18" +
	"b0000618b118f500011900191e00011950196d0001197019740001198019ab00" +
	"0119b019c900011a001a1600011a201a5400011aa71b05005e1b061b3300011b" +
	"451b4c00011b831ba000011bae1baf00011bba1be500011c001c2300011c4d1c" +
	"4f00011c5a1c7d00011c801c8800011c901cba00011cbd1cbf00011ce91cec00" +
	"011cee1cf300011cf51cf600011cfa1d0000061d011dbf00011e001f1500011f" +
	"181f1d00011f201f4500011f481f4d00011f501f5700011f591f5f00021f601f" +
	"7d00011f801fb400011fb61fbc00011fbe1fc200041fc31fc400011fc61fcc00" +
	"011fd01fd300011fd61fdb00011fe01fec00011ff21ff400011ff61ffc000120" +
	"71207f000e2090209c0001210221070005210a21130001211521190004211a21" +
	"1d00012124212a0002212b212d0001212f21390001213c213f00012145214900" +
	"01214e2183003521842c000a7c2c012ce400012ceb2cee00012cf22cf300012d" +
	"002d2500012d272d2d00062d302d6700012d6f2d8000112d812d9600012da02d" +
	"a600012da82dae00012db02db600012db82dbe00012dc02dc600012dc82dce00" +
	"012dd02dd600012dd82dde00012e2f300501d630063031002b30323035000130" +
	"3b303c0001304130960001309d309f000130a130fa000130fc30ff0001310531" +
	"2f00013131318e000131a031bf000131f031ff000134004dbf00014e00a48c00" +
	"01a4d0a4fd0001a500a60c0001a610a61f0001a62aa62b0001a640a66e0001a6" +
	"7fa69d0001a6a0a6e50001a717a71f0001a722a7880001a78ba7ca0001a7d0a7" +
	"d10001a7d3a7d50002a7d6a7d90001a7f2a8010001a803a8050001a807a80a00" +
	"01a80ca8220001a840a8730001a882a8b30001a8f2a8f70001a8fba8fd0002a8" +
	"fea90a000ca90ba9250001a930a9460001a960a97c0001a984a9b20001a9cfa9" +
	"e00011a9e1a9e40001a9e6a9ef0001a9faa9fe0001aa00aa280001aa40aa4200" +
	"01aa44aa4b0001aa60aa760001aa7aaa7e0004aa7faaaf0001aab1aab50004aa" +
	"b6aab90003aabaaabd0001aac0aac20002aadbaadd0001aae0aaea0001aaf2aa" +
	"f40001ab01ab060001ab09ab0e0001ab11ab160001ab20ab260001ab28ab2e00" +
	"01ab30ab5a0001ab5cab690001ab70abe20001ac00d7a30001d7b0d7c60001d7" +
	"cbd7fb0001f900fa6d0001fa70fad90001fb00fb060001fb13fb170001fb1dfb" +
	"1f0002fb20fb280001fb2afb360001fb38fb3c0001fb3efb400002fb41fb4300" +
	"02fb44fb460002fb47fbb10001fbd3fd3d0001fd50fd8f0001fd92fdc70001fd" +
	"f0fdfb0001fe70fe740001fe76fefc0001ff21ff3a0001ff41ff5a0001ff66ff" +
	"be0001ffc2ffc70001ffcaffcf0001ffd2ffd70001ffdaffdc00010108000100" +
	"000001000b000000010001000d0001002600000001000100280001003a000000" +
	"010001003c0001003d000000010001003f0001004d0000000100010050000100" +
	"5d0000000100010080000100fa00000001000102800001029c00000001000102" +
	"a0000102d000000001000103000001031f000000010001032d00010340000000" +
	"0100010342000103490000000100010350000103750000000100010380000103" +
	"9d00000001000103a0000103c300000001000103c8000103cf00000001000104" +
	"000001049d00000001000104b0000104d300000001000104d8000104fb000000" +
	"0100010500000105270000000100010530000105630000000100010570000105" +
	"7a000000010001057c0001058a000000010001058c0001059200000001000105" +
	"94000105950000000100010597000105a100000001000105a3000105b1000000" +
	"01000105b3000105b900000001000105bb000105bc0000000100010600000107" +
	"3600000001000107400001075500000001000107600001076700000001000107" +
	"80000107850000000100010787000107b000000001000107b2000107ba000000" +
	"01000108000001080500000001000108080001080a000000020001080b000108" +
	"35000000010001083700010838000000010001083c0001083f00000003000108" +
	"400001085500000001000108600001087600000001000108800001089e000000" +
	"01000108e0000108f200000001000108f4000108f50000000100010900000109" +
	"150000000100010920000109390000000100010980000109b700000001000109" +
	"be000109bf0000000100010a0000010a100000001000010a1100010a13000000" +
	"0100010a1500010a170000000100010a1900010a350000000100010a6000010a" +
	"7c0000000100010a8000010a9c0000000100010ac000010ac70000000100010a" +
	"c900010ae40000000100010b0000010b350000000100010b4000010b55000000" +
	"0100010b6000010b720000000100010b8000010b910000000100010c0000010c" +
	"480000000100010c8000010cb20000000100010cc000010cf20000000100010d" +
	"0000010d230000000100010e8000010ea90000000100010eb000010eb1000000" +
	"0100010f0000010f1c0000000100010f2700010f300000000900010f3100010f" +
	"450000000100010f7000010f810000000100010fb000010fc40000000100010f" +
	"e000010ff6000000010001100300011037000000010001107100011072000000" +
	"0100011075000110830000000e00011084000110af00000001000110d0000110" +
	"e800000001000111030001112600000001000111440001114700000003000111" +
	"50000111720000000100011176000111830000000d00011184000111b2000000" +
	"01000111c1000111c400000001000111da000111dc0000000200011200000112" +
	"1100000001000112130001122b000000010001123f0001124000000001000112" +
	"800001128600000001000112880001128a000000020001128b0001128d000000" +
	"010001128f0001129d000000010001129f000112a800000001000112b0000112" +
	"de00000001000113050001130c000000010001130f0001131000000001000113" +
	"1300011328000000010001132a00011330000000010001133200011333000000" +
	"010001133500011339000000010001133d00011350000000130001135d000113" +
	"6100000001000114000001143400000001000114470001144a00000001000114" +
	"5f000114610000000100011480000114af00000001000114c4000114c5000000" +
	"01000114c700011580000000b900011581000115ae00000001000115d8000115" +
	"db00000001000116000001162f0000000100011644000116800000003c000116" +
	"81000116aa00000001000116b80001170000000048000117010001171a000000" +
	"01000117400001174600000001000118000001182b00000001000118a0000118" +
	"df00000001000118ff0001190600000001000119090001190c00000003000119" +
	"0d0001191300000001000119150001191600000001000119180001192f000000" +
	"010001193f0001194100000002000119a0000119a700000001000119aa000119" +
	"d000000001000119e1000119e30000000200011a0000011a0b0000000b00011a" +
	"0c00011a320000000100011a3a00011a500000001600011a5c00011a89000000" +
	"0100011a9d00011ab00000001300011ab100011af80000000100011c0000011c" +
	"080000000100011c0a00011c2e0000000100011c4000011c720000003200011c" +
	"7300011c8f0000000100011d0000011d060000000100011d0800011d09000000" +
	"0100011d0b00011d300000000100011d4600011d600000001a00011d6100011d" +
	"650000000100011d6700011d680000000100011d6a00011d890000000100011d" +
	"9800011ee00000014800011ee100011ef20000000100011f0200011f04000000" +
	"0200011f0500011f100000000100011f1200011f330000000100011fb0000120" +
	"000000005000012001000123990000000100012480000125430000000100012f" +
	"9000012ff000000001000130000001342f000000010001344100013446000000" +
	"010001440000014646000000010001680000016a380000000100016a4000016a" +
	"5e0000000100016a7000016abe0000000100016ad000016aed0000000100016b" +
	"0000016b2f0000000100016b4000016b430000000100016b6300016b77000000" +
	"0100016b7d00016b8f0000000100016e4000016e7f0000000100016f0000016f" +
	"4a0000000100016f5000016f930000004300016f9400016f9f0000000100016f" +
	"e000016fe10000000100016fe3000170000000001d00017001000187f7000000" +
	"010001880000018cd50000000100018d0000018d08000000010001aff00001af" +
	"f3000000010001aff50001affb000000010001affd0001affe000000010001b0" +
	"000001b122000000010001b1320001b1500000001e0001b1510001b152000000" +
	"010001b1550001b1640000000f0001b1650001b167000000010001b1700001b2" +
	"fb000000010001bc000001bc6a000000010001bc700001bc7c000000010001bc" +
	"800001bc88000000010001bc900001bc99000000010001d4000001d454000000" +
	"010001d4560001d49c000000010001d49e0001d49f000000010001d4a20001d4" +
	"a5000000030001d4a60001d4a9000000030001d4aa0001d4ac000000010001d4" +
	"ae0001d4b9000000010001d4bb0001d4bd000000020001d4be0001d4c3000000" +
	"010001d4c50001d505000000010001d5070001d50a000000010001d50d0001d5" +
	"14000000010001d5160001d51c000000010001d51e0001d539000000010001d5" +
	"3b0001d53e000000010001d5400001d544000000010001d5460001d54a000000" +
	"040001d54b0001d550000000010001d5520001d6a5000000010001d6a80001d6" +
	"c0000000010001d6c20001d6da000000010001d6dc0001d6fa000000010001d6" +
	"fc0001d714000000010001d7160001d734000000010001d7360001d74e000000" +
	"010001d7500001d76e000000010001d7700001d788000000010001d78a0001d7" +
	"a8000000010001d7aa0001d7c2000000010001d7c40001d7cb000000010001df" +
	"000001df1e000000010001df250001df2a000000010001e0300001e06d000000" +
	"010001e1000001e12c000000010001e1370001e13d000000010001e14e0001e2" +
	"90000001420001e2910001e2ad000000010001e2c00001e2eb000000010001e4" +
	"d00001e4eb000000010001e7e00001e7e6000000010001e7e80001e7eb000000" +
	"010001e7ed0001e7ee000000010001e7f00001e7fe000000010001e8000001e8" +
	"c4000000010001e9000001e943000000010001e94b0001ee00000004b50001ee" +
	"010001ee03000000010001ee050001ee1f000000010001ee210001ee22000000" +
	"010001ee240001ee27000000030001ee290001ee32000000010001ee340001ee" +
	"37000000010001ee390001ee3b000000020001ee420001ee47000000050001ee" +
	"490001ee4d000000020001ee4e0001ee4f000000010001ee510001ee52000000" +
	"010001ee540001ee57000000030001ee590001ee61000000020001ee620001ee" +
	"64000000020001ee670001ee6a000000010001ee6c0001ee72000000010001ee" +
	"740001ee77000000010001ee790001ee7c000000010001ee7e0001ee80000000" +
	"020001ee810001ee89000000010001ee8b0001ee9b000000010001eea10001ee" +
	"a3000000010001eea50001eea9000000010001eeab0001eebb00000001000200" +
	"000002a6df000000010002a7000002b739000000010002b7400002b81d000000" +
	"010002b8200002cea1000000010002ceb00002ebe0000000010002f8000002fa" +
	"1d00000001000300000003134a0000000100031350000323af00000001024c43" +
	"00000456005600050041005a00010061007a000100b500c0000b00c100d60001" +
	"00d800f6000100f801ba000101bc01bf000101c402930001029502af00010370" +
	"03730001037603770001037b037d0001037f038600070388038a0001038c038e" +
	"0002038f03a1000103a303f5000103f704810001048a052f0001053105560001" +
	"05600588000110a010c5000110c710cd000610d010fa000110fd10ff000113a0" +
	"13f5000113f813fd00011c801c8800011c901cba00011cbd1cbf00011d001d2b" +
	"00011d6b1d7700011d791d9a00011e001f1500011f181f1d00011f201f450001" +
	"1f481f4d00011f501f5700011f591f5f00021f601f7d00011f801fb400011fb6" +
	"1fbc00011fbe1fc200041fc31fc400011fc61fcc00011fd01fd300011fd61fdb" +
	"00011fe01fec00011ff21ff400011ff61ffc0001210221070005210a21130001" +
	"211521190004211a211d00012124212a0002212b212d0001212f213400012139" +
	"213c0003213d213f0001214521490001214e2183003521842c000a7c2c012c7b" +
	"00012c7e2ce400012ceb2cee00012cf22cf300012d002d2500012d272d2d0006" +
	"a640a66d0001a680a69b0001a722a76f0001a771a7870001a78ba78e0001a790" +
	"a7ca0001a7d0a7d10001a7d3a7d50002a7d6a7d90001a7f5a7f60001a7faab30" +
	"0336ab31ab5a0001ab60ab680001ab70abbf0001fb00fb060001fb13fb170001" +
	"ff21ff3a0001ff41ff5a00010031000104000001044f00000001000104b00001" +
	"04d300000001000104d8000104fb00000001000105700001057a000000010001" +
	"057c0001058a000000010001058c000105920000000100010594000105950000" +
	"000100010597000105a100000001000105a3000105b100000001000105b30001" +
	"05b900000001000105bb000105bc0000000100010c8000010cb2000000010001" +
	"0cc000010cf200000001000118a0000118df0000000100016e4000016e7f0000" +
	"00010001d4000001d454000000010001d4560001d49c000000010001d49e0001" +
	"d49f000000010001d4a20001d4a5000000030001d4a60001d4a9000000030001" +
	"d4aa0001d4ac000000010001d4ae0001d4b9000000010001d4bb0001d4bd0000" +
	"00020001d4be0001d4c3000000010001d4c50001d505000000010001d5070001" +
	"d50a000000010001d50d0001d514000000010001d5160001d51c000000010001" +
	"d51e0001d539000000010001d53b0001d53e000000010001d5400001d5440000" +
	"00010001d5460001d54a000000040001d54b0001d550000000010001d5520001" +
	"d6a5000000010001d6a80001d6c0000000010001d6c20001d6da000000010001" +
	"d6dc0001d6fa000000010001d6fc0001d714000000010001d7160001d7340000" +
	"00010001d7360001d74e000000010001d7500001d76e000000010001d7700001" +
	"d788000000010001d78a0001d7a8000000010001d7aa0001d7c2000000010001" +
	"d7c40001d7cb000000010001df000001df09000000010001df0b0001df1e0000" +
	"00010001df250001df2a000000010001e9000001e94300000001024c6c000004" +
	"ce007a00040061007a000100b500df002a00e000f6000100f800ff0001010101" +
	"370002013801480002014901770002017a017e0002017f018000010183018500" +
	"020188018c0004018d01920005019501990004019a019b0001019e01a1000301" +
	"a301a5000201a801aa000201ab01ad000201b001b4000401b601b9000301ba01" +
	"bd000301be01bf000101c601cc000301ce01dc000201dd01ef000201f001f300" +
	"0301f501f9000401fb02330002023402390001023c023f000302400242000202" +
	"47024f0002025002930001029502af00010371037300020377037b0004037c03" +
	"7d0001039003ac001c03ad03ce000103d003d1000103d503d7000103d903ef00" +
	"0203f003f3000103f503fb000303fc043000340431045f000104610481000204" +
	"8b04bf000204c204ce000204cf052f000205600588000110d010fa000110fd10" +
	"ff000113f813fd00011c801c8800011d001d2b00011d6b1d7700011d791d9a00" +
	"011e011e9500021e961e9d00011e9f1eff00021f001f0700011f101f1500011f" +
	"201f2700011f301f3700011f401f4500011f501f5700011f601f6700011f701f" +
	"7d00011f801f8700011f901f9700011fa01fa700011fb01fb400011fb61fb700" +
	"011fbe1fc200041fc31fc400011fc61fc700011fd01fd300011fd61fd700011f" +
	"e01fe700011ff21ff400011ff61ff70001210a210e0004210f21130004212f21" +
	"390005213c213d0001214621490001214e218400362c302c5f00012c612c6500" +
	"042c662c6c00022c712c7300022c742c7600022c772c7b00012c812ce300022c" +
	"e42cec00082cee2cf300052d002d2500012d272d2d0006a641a66d0002a681a6" +
	"9b0002a723a72f0002a730a7310001a733a7710002a772a7780001a77aa77c00" +
	"02a77fa7870002a78ca78e0002a791a7930002a794a7950001a797a7a90002a7" +
	"afa7b50006a7b7a7c30002a7c8a7ca0002a7d1a7d90002a7f6a7fa0004ab30ab" +
	"5a0001ab60ab680001ab70abbf0001fb00fb060001fb13fb170001ff41ff5a00" +
	"010029000104280001044f00000001000104d8000104fb000000010001059700" +
	"0105a100000001000105a3000105b100000001000105b3000105b90000000100" +
	"0105bb000105bc0000000100010cc000010cf200000001000118c0000118df00" +
	"00000100016e6000016e7f000000010001d41a0001d433000000010001d44e00" +
	"01d454000000010001d4560001d467000000010001d4820001d49b0000000100" +
	"01d4b60001d4b9000000010001d4bb0001d4bd000000020001d4be0001d4c300" +
	"0000010001d4c50001d4cf000000010001d4ea0001d503000000010001d51e00" +
	"01d537000000010001d5520001d56b000000010001d5860001d59f0000000100" +
	"01d5ba0001d5d3000000010001d5ee0001d607000000010001d6220001d63b00" +
	"0000010001d6560001d66f000000010001d68a0001d6a5000000010001d6c200" +
	"01d6da000000010001d6dc0001d6e1000000010001d6fc0001d7140000000100" +
	"01d7160001d71b000000010001d7360001d74e000000010001d7500001d75500" +
	"0000010001d7700001d788000000010001d78a0001d78f000000010001d7aa00" +
	"01d7c2000000010001d7c40001d7c9000000010001d7cb0001df000000073500" +
	"01df010001df09000000010001df0b0001df1e000000010001df250001df2a00" +
	"0000010001e9220001e94300000001024c6d000001980029000002b002c10001" +
	"02c602d1000102e002e4000102ec02ee00020374037a00060559064000e706e5" +
	"06e6000107f407f5000107fa081a002008240828000408c9097100a80e460ec6" +
	"008010fc17d706db18431aa702641c781c7d00011d2c1d6a00011d781d9b0023" +
	"1d9c1dbf00012071207f000e2090209c00012c7c2c7d00012d6f2e2f00c03005" +
	"3031002c303230350001303b309d0062309e30fc005e30fd30fe0001a015a4f8" +
	"04e3a4f9a4fd0001a60ca67f0073a69ca69d0001a717a71f0001a770a7880018" +
	"a7f2a7f40001a7f8a7f90001a9cfa9e60017aa70aadd006daaf3aaf40001ab5c" +
	"ab5f0001ab69ff705407ff9eff9f0001000d0001078000010785000000010001" +
	"0787000107b000000001000107b2000107ba0000000100016b4000016b430000" +
	"000100016f9300016f9f0000000100016fe000016fe10000000100016fe30001" +
	"aff00000400d0001aff10001aff3000000010001aff50001affb000000010001" +
	"affd0001affe000000010001e0300001e06d000000010001e1370001e13d0000" +
	"00010001e4eb0001e94b00000460024c6f0000102c0117000100aa00ba001001" +
	"bb01c0000501c101c30001029405d0033c05d105ea000105ef05f20001062006" +
	"3f00010641064a0001066e066f0001067106d3000106d506ee001906ef06fa00" +
	"0b06fb06fc000106ff071000110712072f0001074d07a5000107b107ca001907" +
	"cb07ea00010800081500010840085800010860086a0001087008870001088908" +
	"8e000108a008c80001090409390001093d095000130958096100010972098000" +
	"010985098c0001098f09900001099309a8000109aa09b0000109b209b6000409" +
	"b709b9000109bd09ce001109dc09dd000109df09e1000109f009f1000109fc0a" +
	"0500090a060a0a00010a0f0a1000010a130a2800010a2a0a3000010a320a3300" +
	"010a350a3600010a380a3900010a590a5c00010a5e0a7200140a730a7400010a" +
	"850a8d00010a8f0a9100010a930aa800010aaa0ab000010ab20ab300010ab50a" +
	"b900010abd0ad000130ae00ae100010af90b05000c0b060b0c00010b0f0b1000" +
	"010b130b2800010b2a0b3000010b320b3300010b350b3900010b3d0b5c001f0b" +
	"5d0b5f00020b600b6100010b710b8300120b850b8a00010b8e0b9000010b920b" +
	"9500010b990b9a00010b9c0b9e00020b9f0ba300040ba40ba800040ba90baa00" +
	"010bae0bb900010bd00c0500350c060c0c00010c0e0c1000010c120c2800010c" +
	"2a0c3900010c3d0c58001b0c590c5a00010c5d0c6000030c610c80001f0c850c" +
	"8c00010c8e0c9000010c920ca800010caa0cb300010cb50cb900010cbd0cdd00" +
	"200cde0ce000020ce10cf100100cf20d0400120d050d0c00010d0e0d1000010d" +
	"120d3a00010d3d0d4e00110d540d5600010d5f0d6100010d7a0d7f00010d850d" +
	"9600010d9a0db100010db30dbb00010dbd0dc000030dc10dc600010e010e3000" +
	"010e320e3300010e400e4500010e810e8200010e840e8600020e870e8a00010e" +
	"8c0ea300010ea50ea700020ea80eb000010eb20eb300010ebd0ec000030ec10e" +
	"c400010edc0edf00010f000f4000400f410f4700010f490f6c00010f880f8c00" +
	"011000102a0001103f10500011105110550001105a105d000110611065000410" +
	"66106e0008106f10700001107510810001108e11000072110112480001124a12" +
	"4d00011250125600011258125a0002125b125d0001126012880001128a128d00" +
	"01129012b0000112b212b5000112b812be000112c012c2000212c312c5000112" +
	"c812d6000112d8131000011312131500011318135a00011380138f0001140116" +
	"6c0001166f167f00011681169a000116a016ea000116f116f800011700171100" +
	"01171f173100011740175100011760176c0001176e17700001178017b3000117" +
	"dc18200044182118420001184418780001188018840001188718a8000118aa18" +
	"b0000618b118f500011900191e00011950196d0001197019740001198019ab00" +
	"0119b019c900011a001a1600011a201a5400011b051b3300011b451b4c00011b" +
	"831ba000011bae1baf00011bba1be500011c001c2300011c4d1c4f00011c5a1c" +
	"7700011ce91cec00011cee1cf300011cf51cf600011cfa2135043b2136213800" +
	"012d302d6700012d802d9600012da02da600012da82dae00012db02db600012d" +
	"b82dbe00012dc02dc600012dc82dce00012dd02dd600012dd82dde0001300630" +
	"3c0036304130960001309f30a1000230a230fa000130ff310500063106312f00" +
	"013131318e000131a031bf000131f031ff000134004dbf00014e00a0140001a0" +
	"16a48c0001a4d0a4f70001a500a60b0001a610a61f0001a62aa62b0001a66ea6" +
	"a00032a6a1a6e50001a78fa7f70068a7fba8010001a803a8050001a807a80a00" +
	"01a80ca8220001a840a8730001a882a8b30001a8f2a8f70001a8fba8fd0002a8" +
	"fea90a000ca90ba9250001a930a9460001a960a97c0001a984a9b20001a9e0a9" +
	"e40001a9e7a9ef0001a9faa9fe0001aa00aa280001aa40aa420001aa44aa4b00" +
	"01aa60aa6f0001aa71aa760001aa7aaa7e0004aa7faaaf0001aab1aab50004aa" +
	"b6aab90003aabaaabd0001aac0aac20002aadbaadc0001aae0aaea0001aaf2ab" +
	"01000fab02ab060001ab09ab0e0001ab11ab160001ab20ab260001ab28ab2e00" +
	"01abc0abe20001ac00d7a30001d7b0d7c60001d7cbd7fb0001f900fa6d0001fa" +
	"70fad90001fb1dfb1f0002fb20fb280001fb2afb360001fb38fb3c0001fb3efb" +
	"400002fb41fb430002fb44fb460002fb47fbb10001fbd3fd3d0001fd50fd8f00" +
	"01fd92fdc70001fdf0fdfb0001fe70fe740001fe76fefc0001ff66ff6f0001ff" +
	"71ff9d0001ffa0ffbe0001ffc2ffc70001ffcaffcf0001ffd2ffd70001ffdaff" +
	"dc000100cd000100000001000b000000010001000d0001002600000001000100" +
	"280001003a000000010001003c0001003d000000010001003f0001004d000000" +
	"01000100500001005d0000000100010080000100fa0000000100010280000102" +
	"9c00000001000102a0000102d000000001000103000001031f00000001000103" +
	"2d00010340000000010001034200010349000000010001035000010375000000" +
	"01000103800001039d00000001000103a0000103c300000001000103c8000103" +
	"cf00000001000104500001049d00000001000105000001052700000001000105" +
	"3000010563000000010001060000010736000000010001074000010755000000" +
	"0100010760000107670000000100010800000108050000000100010808000108" +
	"0a000000020001080b0001083500000001000108370001083800000001000108" +
	"3c0001083f000000030001084000010855000000010001086000010876000000" +
	"01000108800001089e00000001000108e0000108f200000001000108f4000108" +
	"f500000001000109000001091500000001000109200001093900000001000109" +
	"80000109b700000001000109be000109bf0000000100010a0000010a10000000" +
	"1000010a1100010a130000000100010a1500010a170000000100010a1900010a" +
	"350000000100010a6000010a7c0000000100010a8000010a9c0000000100010a" +
	"c000010ac70000000100010ac900010ae40000000100010b0000010b35000000" +
	"0100010b4000010b550000000100010b6000010b720000000100010b8000010b" +
	"910000000100010c0000010c480000000100010d0000010d230000000100010e" +
	"8000010ea90000000100010eb000010eb10000000100010f0000010f1c000000" +
	"0100010f2700010f300000000900010f3100010f450000000100010f7000010f" +
	"810000000100010fb000010fc40000000100010fe000010ff600000001000110" +
	"0300011037000000010001107100011072000000010001107500011083000000" +
	"0e00011084000110af00000001000110d0000110e80000000100011103000111" +
	"2600000001000111440001114700000003000111500001117200000001000111" +
	"76000111830000000d00011184000111b200000001000111c1000111c4000000" +
	"01000111da000111dc0000000200011200000112110000000100011213000112" +
	"2b000000010001123f0001124000000001000112800001128600000001000112" +
	"880001128a000000020001128b0001128d000000010001128f0001129d000000" +
	"010001129f000112a800000001000112b0000112de0000000100011305000113" +
	"0c000000010001130f0001131000000001000113130001132800000001000113" +
	"2a00011330000000010001133200011333000000010001133500011339000000" +
	"010001133d00011350000000130001135d000113610000000100011400000114" +
	"3400000001000114470001144a000000010001145f0001146100000001000114" +
	"80000114af00000001000114c4000114c500000001000114c700011580000000" +
	"b900011581000115ae00000001000115d8000115db0000000100011600000116" +
	"2f0000000100011644000116800000003c00011681000116aa00000001000116" +
	"b80001170000000048000117010001171a000000010001174000011746000000" +
	"01000118000001182b00000001000118ff000119060000000100011909000119" +
	"0c000000030001190d0001191300000001000119150001191600000001000119" +
	"180001192f000000010001193f0001194100000002000119a0000119a7000000" +
	"01000119aa000119d000000001000119e1000119e30000000200011a0000011a" +
	"0b0000000b00011a0c00011a320000000100011a3a00011a500000001600011a" +
	"5c00011a890000000100011a9d00011ab00000001300011ab100011af8000000" +
	"0100011c0000011c080000000100011c0a00011c2e0000000100011c4000011c" +
	"720000003200011c7300011c8f0000000100011d0000011d060000000100011d" +
	"0800011d090000000100011d0b00011d300000000100011d4600011d60000000" +
	"1a00011d6100011d650000000100011d6700011d680000000100011d6a00011d" +
	"890000000100011d9800011ee00000014800011ee100011ef20000000100011f" +
	"0200011f040000000200011f0500011f100000000100011f1200011f33000000" +
	"0100011fb0000120000000005000012001000123990000000100012480000125" +
	"430000000100012f9000012ff000000001000130000001342f00000001000134" +
	"4100013446000000010001440000014646000000010001680000016a38000000" +
	"0100016a4000016a5e0000000100016a7000016abe0000000100016ad000016a" +
	"ed0000000100016b0000016b2f0000000100016b6300016b770000000100016b" +
	"7d00016b8f0000000100016f0000016f4a0000000100016f5000017000000000" +
	"b000017001000187f7000000010001880000018cd50000000100018d0000018d" +
	"08000000010001b0000001b122000000010001b1320001b1500000001e0001b1" +
	"510001b152000000010001b1550001b1640000000f0001b1650001b167000000" +
	"010001b1700001b2fb000000010001bc000001bc6a000000010001bc700001bc" +
	"7c000000010001bc800001bc88000000010001bc900001bc99000000010001df" +
	"0a0001e100000001f60001e1010001e12c000000010001e14e0001e290000001" +
	"420001e2910001e2ad000000010001e2c00001e2eb000000010001e4d00001e4" +
	"ea000000010001e7e00001e7e6000000010001e7e80001e7eb000000010001e7" +
	"ed0001e7ee000000010001e7f00001e7fe000000010001e8000001e8c4000000" +
	"010001ee000001ee03000000010001ee050001ee1f000000010001ee210001ee" +
	"22000000010001ee240001ee27000000030001ee290001ee32000000010001ee" +
	"340001ee37000000010001ee390001ee3b000000020001ee420001ee47000000" +
	"050001ee490001ee4d000000020001ee4e0001ee4f000000010001ee510001ee" +
	"52000000010001ee540001ee57000000030001ee590001ee61000000020001ee" +
	"620001ee64000000020001ee670001ee6a000000010001ee6c0001ee72000000" +
	"010001ee740001ee77000000010001ee790001ee7c000000010001ee7e0001ee" +
	"80000000020001ee810001ee89000000010001ee8b0001ee9b000000010001ee" +
	"a10001eea3000000010001eea50001eea9000000010001eeab0001eebb000000" +
	"01000200000002a6df000000010002a7000002b739000000010002b7400002b8" +
	"1d000000010002b8200002cea1000000010002ceb00002ebe0000000010002f8" +
	"000002fa1d00000001000300000003134a0000000100031350000323af000000" +
	"01024c74000000300007000001c501cb000301f21f881d961f891f8f00011f98" +
	"1f9f00011fa81faf00011fbc1fcc00101ffc1ffc00010000024c750000047400" +
	"6d00030041005a000100c000d6000100d800de00010100013600020139014700" +
	"02014a017800020179017d000201810182000101840186000201870189000201" +
	"8a018b0001018e01910001019301940001019601980001019c019d0001019f01" +
	"a0000101a201a6000201a701a9000201ac01ae000201af01b1000201b201b300" +
	"0101b501b7000201b801bc000401c401cd000301cf01db000201de01ee000201" +
	"f101f4000301f601f8000101fa02320002023a023b0001023d023e0001024102" +
	"4300020244024600010248024e00020370037200020376037f00090386038800" +
	"020389038a0001038c038e0002038f03910002039203a1000103a303ab000103" +
	"cf03d2000303d303d4000103d803ee000203f403f7000303f903fa000103fd04" +
	"2f0001046004800002048a04c0000204c104cd000204d0052e00020531055600" +
	"0110a010c5000110c710cd000613a013f500011c901cba00011cbd1cbf00011e" +
	"001e9400021e9e1efe00021f081f0f00011f181f1d00011f281f2f00011f381f" +
	"3f00011f481f4d00011f591f5f00021f681f6f00011fb81fbb00011fc81fcb00" +
	"011fd81fdb00011fe81fec00011ff81ffb0001210221070005210b210d000121" +
	"1021120001211521190004211a211d00012124212a0002212b212d0001213021" +
	"330001213e213f000121452183003e2c002c2f00012c602c6200022c632c6400" +
	"012c672c6d00022c6e2c7000012c722c7500032c7e2c8000012c822ce200022c" +
	"eb2ced00022cf2a640794ea642a66c0002a680a69a0002a722a72e0002a732a7" +
	"6e0002a779a77d0002a77ea7860002a78ba78d0002a790a7920002a796a7aa00" +
	"02a7aba7ae0001a7b0a7b40001a7b6a7c40002a7c5a7c70001a7c9a7d00007a7" +
	"d6a7d80002a7f5ff21572cff22ff3a0001002800010400000104270000000100" +
	"0104b0000104d300000001000105700001057a000000010001057c0001058a00" +
	"0000010001058c000105920000000100010594000105950000000100010c8000" +
	"010cb200000001000118a0000118bf0000000100016e4000016e5f0000000100" +
	"01d4000001d419000000010001d4340001d44d000000010001d4680001d48100" +
	"0000010001d49c0001d49e000000020001d49f0001d4a5000000030001d4a600" +
	"01d4a9000000030001d4aa0001d4ac000000010001d4ae0001d4b50000000100" +
	"01d4d00001d4e9000000010001d5040001d505000000010001d5070001d50a00" +
	"0000010001d50d0001d514000000010001d5160001d51c000000010001d53800" +
	"01d539000000010001d53b0001d53e000000010001d5400001d5440000000100" +
	"01d5460001d54a000000040001d54b0001d550000000010001d56c0001d58500" +
	"0000010001d5a00001d5b9000000010001d5d40001d5ed000000010001d60800" +
	"01d621000000010001d63c0001d655000000010001d6700001d6890000000100" +
	"01d6a80001d6c0000000010001d6e20001d6fa000000010001d71c0001d73400" +
	"0000010001d7560001d76e000000010001d7900001d7a8000000010001d7ca00" +
	"01e900000011360001e9010001e92100000001014d000009c600b60000030003" +
	"6f0001048304890001059105bd000105bf05c1000205c205c4000205c505c700" +
	"020610061a0001064b065f0001067006d6006606d706dc000106df06e4000106" +
	"e706e8000106ea06ed000107110730001f0731074a000107a607b0000107eb07" +
	"f3000107fd08160019081708190001081b082300010825082700010829082d00" +
	"010859085b00010898089f000108ca08e1000108e309030001093a093c000109" +
	"3e094f000109510957000109620963000109810983000109bc09be000209bf09" +
	"c4000109c709c8000109cb09cd000109d709e2000b09e309fe001b0a010a0300" +
	"010a3c0a3e00020a3f0a4200010a470a4800010a4b0a4d00010a510a70001f0a" +
	"710a7500040a810a8300010abc0abe00020abf0ac500010ac70ac900010acb0a" +
	"cd00010ae20ae300010afa0aff00010b010b0300010b3c0b3e00020b3f0b4400" +
	"010b470b4800010b4b0b4d00010b550b5700010b620b6300010b820bbe003c0b" +
	"bf0bc200010bc60bc800010bca0bcd00010bd70c0000290c010c0400010c3c0c" +
	"3e00020c3f0c4400010c460c4800010c4a0c4d00010c550c5600010c620c6300" +
	"010c810c8300010cbc0cbe00020cbf0cc400010cc60cc800010cca0ccd00010c" +
	"d50cd600010ce20ce300010cf30d00000d0d010d0300010d3b0d3c00010d3e0d" +
	"4400010d460d4800010d4a0d4d00010d570d62000b0d630d81001e0d820d8300" +
	"010dca0dcf00050dd00dd400010dd60dd800020dd90ddf00010df20df300010e" +
	"310e3400030e350e3a00010e470e4e00010eb10eb400030eb50ebc00010ec80e" +
	"ce00010f180f1900010f350f3900020f3e0f3f00010f710f8400010f860f8700" +
	"010f8d0f9700010f990fbc00010fc6102b0065102c103e000110561059000110" +
	"5e106000011062106400011067106d00011071107400011082108d0001108f10" +
	"9a000b109b109d0001135d135f00011712171500011732173400011752175300" +
	"0117721773000117b417d3000117dd180b002e180c180d0001180f1885007618" +
	"8618a900231920192b00011930193b00011a171a1b00011a551a5e00011a601a" +
	"7c00011a7f1ab000311ab11ace00011b001b0400011b341b4400011b6b1b7300" +
	"011b801b8200011ba11bad00011be61bf300011c241c3700011cd01cd200011c" +
	"d41ce800011ced1cf400071cf71cf900011dc01dff000120d020f000012cef2c" +
	"f100012d7f2de000612de12dff0001302a302f00013099309a0001a66fa67200" +
	"01a674a67d0001a69ea69f0001a6f0a6f10001a802a8060004a80ba8230018a8" +
	"24a8270001a82ca8800054a881a8b40033a8b5a8c50001a8e0a8f10001a8ffa9" +
	"260027a927a92d0001a947a9530001a980a9830001a9b3a9c00001a9e5aa2900" +
	"44aa2aaa360001aa43aa4c0009aa4daa7b002eaa7caa7d0001aab0aab20002aa" +
	"b3aab40001aab7aab80001aabeaabf0001aac1aaeb002aaaecaaef0001aaf5aa" +
	"f60001abe3abea0001abecabed0001fb1efe0002e2fe01fe0f0001fe20fe2f00" +
	"010075000101fd000102e0000000e3000103760001037a0000000100010a0100" +
	"010a030000000100010a0500010a060000000100010a0c00010a0f0000000100" +
	"010a3800010a3a0000000100010a3f00010ae5000000a600010ae600010d2400" +
	"00023e00010d2500010d270000000100010eab00010eac0000000100010efd00" +
	"010eff0000000100010f4600010f500000000100010f8200010f850000000100" +
	"0110000001100200000001000110380001104600000001000110700001107300" +
	"000003000110740001107f0000000b000110800001108200000001000110b000" +
	"0110ba00000001000110c2000111000000003e00011101000111020000000100" +
	"0111270001113400000001000111450001114600000001000111730001118000" +
	"00000d000111810001118200000001000111b3000111c000000001000111c900" +
	"0111cc00000001000111ce000111cf000000010001122c000112370000000100" +
	"01123e0001124100000003000112df000112ea00000001000113000001130300" +
	"0000010001133b0001133c000000010001133e00011344000000010001134700" +
	"011348000000010001134b0001134d0000000100011357000113620000000b00" +
	"0113630001136600000003000113670001136c00000001000113700001137400" +
	"0000010001143500011446000000010001145e000114b000000052000114b100" +
	"0114c300000001000115af000115b500000001000115b8000115c00000000100" +
	"0115dc000115dd00000001000116300001164000000001000116ab000116b700" +
	"0000010001171d0001172b000000010001182c0001183a000000010001193000" +
	"011935000000010001193700011938000000010001193b0001193e0000000100" +
	"011940000119420000000200011943000119d10000008e000119d2000119d700" +
	"000001000119da000119e000000001000119e400011a010000001d00011a0200" +
	"011a0a0000000100011a3300011a390000000100011a3b00011a3e0000000100" +
	"011a4700011a510000000a00011a5200011a5b0000000100011a8a00011a9900" +
	"00000100011c2f00011c360000000100011c3800011c3f0000000100011c9200" +
	"011ca70000000100011ca900011cb60000000100011d3100011d360000000100" +
	"011d3a00011d3c0000000200011d3d00011d3f0000000200011d4000011d4500" +
	"00000100011d4700011d8a0000004300011d8b00011d8e0000000100011d9000" +
	"011d910000000100011d9300011d970000000100011ef300011ef60000000100" +
	"011f0000011f010000000100011f0300011f340000003100011f3500011f3a00" +
	"00000100011f3e00011f42000000010001344000013447000000070001344800" +
	"0134550000000100016af000016af40000000100016b3000016b360000000100" +
	"016f4f00016f510000000200016f5200016f870000000100016f8f00016f9200" +
	"00000100016fe400016ff00000000c00016ff10001bc9d00004cac0001bc9e00" +
	"01cf00000012620001cf010001cf2d000000010001cf300001cf460000000100" +
	"01d1650001d169000000010001d16d0001d172000000010001d17b0001d18200" +
	"0000010001d1850001d18b000000010001d1aa0001d1ad000000010001d24200" +
	"01d244000000010001da000001da36000000010001da3b0001da6c0000000100" +
	"01da750001da840000000f0001da9b0001da9f000000010001daa10001daaf00" +
	"0000010001e0000001e006000000010001e0080001e018000000010001e01b00" +
	"01e021000000010001e0230001e024000000010001e0260001e02a0000000100" +
	"01e08f0001e130000000a10001e1310001e136000000010001e2ae0001e2ec00" +
	"00003e0001e2ed0001e2ef000000010001e4ec0001e4ef000000010001e8d000" +
	"01e8d6000000010001e9440001e94a00000001000e0100000e01ef0000000102" +
	"4d6300000522006200000903093b0038093e094000010949094c0001094e094f" +
	"000109820983000109be09c0000109c709c8000109cb09cc000109d70a03002c" +
	"0a3e0a4000010a830abe003b0abf0ac000010ac90acb00020acc0b0200360b03" +
	"0b3e003b0b400b4700070b480b4b00030b4c0b57000b0bbe0bbf00010bc10bc2" +
	"00010bc60bc800010bca0bcc00010bd70c01002a0c020c0300010c410c440001" +
	"0c820c8300010cbe0cc000020cc10cc400010cc70cc800010cca0ccb00010cd5" +
	"0cd600010cf30d02000f0d030d3e003b0d3f0d4000010d460d4800010d4a0d4c" +
	"00010d570d82002b0d830dcf004c0dd00dd100010dd80ddf00010df20df30001" +
	"0f3e0f3f00010f7f102b00ac102c103100051038103b0003103c1056001a1057" +
	"1062000b1063106400011067106d00011083108400011087108c0001108f109a" +
	"000b109b109c000117151734001f17b617be000817bf17c5000117c717c80001" +
	"1923192600011929192b00011930193100011933193800011a191a1a00011a55" +
	"1a5700021a611a6300021a641a6d00091a6e1a7200011b041b3500311b3b1b3d" +
	"00021b3e1b4100011b431b4400011b821ba1001f1ba61ba700011baa1be7003d" +
	"1bea1bec00011bee1bf200041bf31c2400311c251c2b00011c341c3500011ce1" +
	"1cf70016302e302f0001a823a8240001a827a8800059a881a8b40033a8b5a8c3" +
	"0001a952a9530001a983a9b40031a9b5a9ba0005a9bba9be0003a9bfa9c00001" +
	"aa2faa300001aa33aa340001aa4daa7b002eaa7daaeb006eaaeeaaef0001aaf5" +
	"abe300eeabe4abe60002abe7abe90002abeaabec0002003c0001100000011002" +
	"0000000200011082000110b00000002e000110b1000110b200000001000110b7" +
	"000110b8000000010001112c000111450000001900011146000111820000003c" +
	"000111b3000111b500000001000111bf000111c000000001000111ce0001122c" +
	"0000005e0001122d0001122e0000000100011232000112330000000100011235" +
	"000112e0000000ab000112e1000112e200000001000113020001130300000001" +
	"0001133e0001133f000000010001134100011344000000010001134700011348" +
	"000000010001134b0001134d0000000100011357000113620000000b00011363" +
	"00011435000000d2000114360001143700000001000114400001144100000001" +
	"00011445000114b00000006b000114b1000114b200000001000114b9000114bb" +
	"00000002000114bc000114be00000001000114c1000115af000000ee000115b0" +
	"000115b100000001000115b8000115bb00000001000115be0001163000000072" +
	"0001163100011632000000010001163b0001163c000000010001163e000116ac" +
	"0000006e000116ae000116af00000001000116b6000117200000006a00011721" +
	"00011726000000050001182c0001182e000000010001183800011930000000f8" +
	"0001193100011935000000010001193700011938000000010001193d00011940" +
	"0000000300011942000119d10000008f000119d2000119d300000001000119dc" +
	"000119df00000001000119e400011a390000005500011a5700011a5800000001" +
	"00011a9700011c2f0000019800011c3e00011ca90000006b00011cb100011cb4" +
	"0000000300011d8a00011d8e0000000100011d9300011d940000000100011d96" +
	"00011ef50000015f00011ef600011f030000000d00011f3400011f3500000001" +
	"00011f3e00011f3f0000000100011f4100016f510000501000016f5200016f87" +
	"0000000100016ff000016ff1000000010001d1650001d166000000010001d16d" +
	"0001d17200000001024d6500000024000500000488048900011abe20dd061f20" +
	"de20e0000120e220e40001a670a67200010000024d6e00000a2c00b700000300" +
	"036f0001048304870001059105bd000105bf05c1000205c205c4000205c505c7" +
	"00020610061a0001064b065f0001067006d6006606d706dc000106df06e40001" +
	"06e706e8000106ea06ed000107110730001f0731074a000107a607b0000107eb" +
	"07f3000107fd08160019081708190001081b082300010825082700010829082d" +
	"00010859085b00010898089f000108ca08e1000108e309020001093a093c0002" +
	"094109480001094d09510004095209570001096209630001098109bc003b09c1" +
	"09c4000109cd09e2001509e309fe001b0a010a0200010a3c0a4100050a420a47" +
	"00050a480a4b00030a4c0a4d00010a510a70001f0a710a7500040a810a820001" +
	"0abc0ac100050ac20ac500010ac70ac800010acd0ae200150ae30afa00170afb" +
	"0aff00010b010b3c003b0b3f0b4100020b420b4400010b4d0b5500080b560b62" +
	"000c0b630b82001f0bc00bcd000d0c000c0400040c3c0c3e00020c3f0c400001" +
	"0c460c4800010c4a0c4d00010c550c5600010c620c6300010c810cbc003b0cbf" +
	"0cc600070ccc0ccd00010ce20ce300010d000d0100010d3b0d3c00010d410d44" +
	"00010d4d0d6200150d630d81001e0dca0dd200080dd30dd400010dd60e31005b" +
	"0e340e3a00010e470e4e00010eb10eb400030eb50ebc00010ec80ece00010f18" +
	"0f1900010f350f3900020f710f7e00010f800f8400010f860f8700010f8d0f97" +
	"00010f990fbc00010fc6102d0067102e103000011032103700011039103a0001" +
	"103d103e0001105810590001105e106000011071107400011082108500031086" +
	"108d0007109d135d02c0135e135f000117121714000117321733000117521753" +
	"000117721773000117b417b5000117b717bd000117c617c9000317ca17d30001" +
	"17dd180b002e180c180d0001180f18850076188618a900231920192200011927" +
	"19280001193219390007193a193b00011a171a1800011a1b1a56003b1a581a5e" +
	"00011a601a6200021a651a6c00011a731a7c00011a7f1ab000311ab11abd0001" +
	"1abf1ace00011b001b0300011b341b3600021b371b3a00011b3c1b4200061b6b" +
	"1b7300011b801b8100011ba21ba500011ba81ba900011bab1bad00011be61be8" +
	"00021be91bed00041bef1bf100011c2c1c3300011c361c3700011cd01cd20001" +
	"1cd41ce000011ce21ce800011ced1cf400071cf81cf900011dc01dff000120d0" +
	"20dc000120e120e5000420e620f000012cef2cf100012d7f2de000612de12dff" +
	"0001302a302d00013099309a0001a66fa6740005a675a67d0001a69ea69f0001" +
	"a6f0a6f10001a802a8060004a80ba825001aa826a82c0006a8c4a8c50001a8e0" +
	"a8f10001a8ffa9260027a927a92d0001a947a9510001a980a9820001a9b3a9b6" +
	"0003a9b7a9b90001a9bca9bd0001a9e5aa290044aa2aaa2e0001aa31aa320001" +
	"aa35aa360001aa43aa4c0009aa7caab00034aab2aab40001aab7aab80001aabe" +
	"aabf0001aac1aaec002baaedaaf60009abe5abe80003abedfb1e4f31fe00fe0f" +
	"0001fe20fe2f0001007d000101fd000102e0000000e3000103760001037a0000" +
	"000100010a0100010a030000000100010a0500010a060000000100010a0c0001" +
	"0a0f0000000100010a3800010a3a0000000100010a3f00010ae5000000a60001" +
	"0ae600010d240000023e00010d2500010d270000000100010eab00010eac0000" +
	"000100010efd00010eff0000000100010f4600010f500000000100010f820001" +
	"0f85000000010001100100011038000000370001103900011046000000010001" +
	"10700001107300000003000110740001107f0000000b00011080000110810000" +
	"0001000110b3000110b600000001000110b9000110ba00000001000110c20001" +
	"11000000003e000111010001110200000001000111270001112b000000010001" +
	"112d000111340000000100011173000111800000000d00011181000111b60000" +
	"0035000111b7000111be00000001000111c9000111cc00000001000111cf0001" +
	"122f000000600001123000011231000000010001123400011236000000020001" +
	"12370001123e0000000700011241000112df0000009e000112e3000112ea0000" +
	"00010001130000011301000000010001133b0001133c00000001000113400001" +
	"136600000026000113670001136c000000010001137000011374000000010001" +
	"14380001143f00000001000114420001144400000001000114460001145e0000" +
	"0018000114b3000114b800000001000114ba000114bf00000005000114c00001" +
	"14c200000002000114c3000115b2000000ef000115b3000115b5000000010001" +
	"15bc000115bd00000001000115bf000115c000000001000115dc000115dd0000" +
	"0001000116330001163a000000010001163d0001163f00000002000116400001" +
	"16ab0000006b000116ad000116b000000003000116b1000116b5000000010001" +
	"16b70001171d000000660001171e0001171f0000000100011722000117250000" +
	"0001000117270001172b000000010001182f0001183700000001000118390001" +
	"183a000000010001193b0001193c000000010001193e00011943000000050001" +
	"19d4000119d700000001000119da000119db00000001000119e000011a010000" +
	"002100011a0200011a0a0000000100011a3300011a380000000100011a3b0001" +
	"1a3e0000000100011a4700011a510000000a00011a5200011a56000000010001" +
	"1a5900011a5b0000000100011a8a00011a960000000100011a9800011a990000" +
	"000100011c3000011c360000000100011c3800011c3d0000000100011c3f0001" +
	"1c920000005300011c9300011ca70000000100011caa00011cb0000000010001" +
	"1cb200011cb30000000100011cb500011cb60000000100011d3100011d360000" +
	"000100011d3a00011d3c0000000200011d3d00011d3f0000000200011d400001" +
	"1d450000000100011d4700011d900000004900011d9100011d95000000040001" +
	"1d9700011ef30000015c00011ef400011f000000000c00011f0100011f360000" +
	"003500011f3700011f3a0000000100011f4000011f4200000002000134400001" +
	"34470000000700013448000134550000000100016af000016af4000000010001" +
	"6b3000016b360000000100016f4f00016f8f0000004000016f9000016f920000" +
	"000100016fe40001bc9d00004cb90001bc9e0001cf00000012620001cf010001" +
	"cf2d000000010001cf300001cf46000000010001d1670001d169000000010001" +
	"d17b0001d182000000010001d1850001d18b000000010001d1aa0001d1ad0000" +
	"00010001d2420001d244000000010001da000001da36000000010001da3b0001" +
	"da6c000000010001da750001da840000000f0001da9b0001da9f000000010001" +
	"daa10001daaf000000010001e0000001e006000000010001e0080001e0180000" +
	"00010001e01b0001e021000000010001e0230001e024000000010001e0260001" +
	"e02a000000010001e08f0001e130000000a10001e1310001e136000000010001" +
	"e2ae0001e2ec0000003e0001e2ed0001e2ef000000010001e4ec0001e4ef0000" +
	"00010001e8d00001e8d6000000010001e9440001e94a00000001000e0100000e" +
	"01ef00000001014e000004ce0042000400300039000100b200b3000100b900bc" +
	"000300bd00be000106600669000106f006f9000107c007c900010966096f0001" +
	"09e609ef000109f409f900010a660a6f00010ae60aef00010b660b6f00010b72" +
	"0b7700010be60bf200010c660c6f00010c780c7e00010ce60cef00010d580d5e" +
	"00010d660d7800010de60def00010e500e5900010ed00ed900010f200f330001" +
	"1040104900011090109900011369137c000116ee16f0000117e017e9000117f0" +
	"17f900011810181900011946194f000119d019da00011a801a8900011a901a99" +
	"00011b501b5900011bb01bb900011c401c4900011c501c590001207020740004" +
	"2075207900012080208900012150218200012185218900012460249b000124ea" +
	"24ff00012776279300012cfd3007030a3021302900013038303a000131923195" +
	"00013220322900013248324f00013251325f000132803289000132b132bf0001" +
	"a620a6290001a6e6a6ef0001a830a8350001a8d0a8d90001a900a9090001a9d0" +
	"a9d90001a9f0a9f90001aa50aa590001abf0abf90001ff10ff19000100450001" +
	"010700010133000000010001014000010178000000010001018a0001018b0000" +
	"0001000102e1000102fb00000001000103200001032300000001000103410001" +
	"034a00000009000103d1000103d500000001000104a0000104a9000000010001" +
	"08580001085f00000001000108790001087f00000001000108a7000108af0000" +
	"0001000108fb000108ff00000001000109160001091b00000001000109bc0001" +
	"09bd00000001000109c0000109cf00000001000109d2000109ff000000010001" +
	"0a4000010a480000000100010a7d00010a7e0000000100010a9d00010a9f0000" +
	"000100010aeb00010aef0000000100010b5800010b5f0000000100010b780001" +
	"0b7f0000000100010ba900010baf0000000100010cfa00010cff000000010001" +
	"0d3000010d390000000100010e6000010e7e0000000100010f1d00010f260000" +
	"000100010f5100010f540000000100010fc500010fcb00000001000110520001" +
	"106f00000001000110f0000110f900000001000111360001113f000000010001" +
	"11d0000111d900000001000111e1000111f400000001000112f0000112f90000" +
	"0001000114500001145900000001000114d0000114d900000001000116500001" +
	"165900000001000116c0000116c900000001000117300001173b000000010001" +
	"18e0000118f20000000100011950000119590000000100011c5000011c6c0000" +
	"000100011d5000011d590000000100011da000011da90000000100011f500001" +
	"1f590000000100011fc000011fd400000001000124000001246e000000010001" +
	"6a6000016a690000000100016ac000016ac90000000100016b5000016b590000" +
	"000100016b5b00016b610000000100016e8000016e96000000010001d2c00001" +
	"d2d3000000010001d2e00001d2f3000000010001d3600001d378000000010001" +
	"d7ce0001d7ff000000010001e1400001e149000000010001e2f00001e2f90000" +
	"00010001e4f00001e4f9000000010001e8c70001e8cf000000010001e9500001" +
	"e959000000010001ec710001ecab000000010001ecad0001ecaf000000010001" +
	"ecb10001ecb4000000010001ed010001ed2d000000010001ed2f0001ed3d0000" +
	"00010001f1000001f10c000000010001fbf00001fbf900000001024e64000002" +
	"280025000100300039000106600669000106f006f9000107c007c90001096609" +
	"6f000109e609ef00010a660a6f00010ae60aef00010b660b6f00010be60bef00" +
	"010c660c6f00010ce60cef00010d660d6f00010de60def00010e500e5900010e" +
	"d00ed900010f200f29000110401049000110901099000117e017e90001181018" +
	"1900011946194f000119d019d900011a801a8900011a901a9900011b501b5900" +
	"011bb01bb900011c401c4900011c501c590001a620a6290001a8d0a8d90001a9" +
	"00a9090001a9d0a9d90001a9f0a9f90001aa50aa590001abf0abf90001ff10ff" +
	"190001001b000104a0000104a90000000100010d3000010d3900000001000110" +
	"660001106f00000001000110f0000110f900000001000111360001113f000000" +
	"01000111d0000111d900000001000112f0000112f90000000100011450000114" +
	"5900000001000114d0000114d900000001000116500001165900000001000116" +
	"c0000116c900000001000117300001173900000001000118e0000118e9000000" +
	"0100011950000119590000000100011c5000011c590000000100011d5000011d" +
	"590000000100011da000011da90000000100011f5000011f590000000100016a" +
	"6000016a690000000100016ac000016ac90000000100016b5000016b59000000" +
	"010001d7ce0001d7ff000000010001e1400001e149000000010001e2f00001e2" +
	"f9000000010001e4f00001e4f9000000010001e9500001e959000000010001fb" +
	"f00001fbf900000001024e6c000000600007000016ee16f00001216021820001" +
	"21852188000130073021001a3022302900013038303a0001a6e6a6ef00010004" +
	"000101400001017400000001000103410001034a00000009000103d1000103d5" +
	"00000001000124000001246e00000001024e6f000002b2001c000300b200b300" +
	"0100b900bc000300bd00be000109f409f900010b720b7700010bf00bf200010c" +
	"780c7e00010d580d5e00010d700d7800010f2a0f3300011369137c000117f017" +
	"f9000119da207006962074207900012080208900012150215f00012189246002" +
	"d72461249b000124ea24ff00012776279300012cfd3192049531933195000132" +
	"20322900013248324f00013251325f000132803289000132b132bf0001a830a8" +
	"350001002b000101070001013300000001000101750001017800000001000101" +
	"8a0001018b00000001000102e1000102fb000000010001032000010323000000" +
	"01000108580001085f00000001000108790001087f00000001000108a7000108" +
	"af00000001000108fb000108ff00000001000109160001091b00000001000109" +
	"bc000109bd00000001000109c0000109cf00000001000109d2000109ff000000" +
	"0100010a4000010a480000000100010a7d00010a7e0000000100010a9d00010a" +
	"9f0000000100010aeb00010aef0000000100010b5800010b5f0000000100010b" +
	"7800010b7f0000000100010ba900010baf0000000100010cfa00010cff000000" +
	"0100010e6000010e7e0000000100010f1d00010f260000000100010f5100010f" +
	"540000000100010fc500010fcb00000001000110520001106500000001000111" +
	"e1000111f4000000010001173a0001173b00000001000118ea000118f2000000" +
	"0100011c5a00011c6c0000000100011fc000011fd40000000100016b5b00016b" +
	"610000000100016e8000016e96000000010001d2c00001d2d3000000010001d2" +
	"e00001d2f3000000010001d3600001d378000000010001e8c70001e8cf000000" +
	"010001ec710001ecab000000010001ecad0001ecaf000000010001ecb10001ec" +
	"b4000000010001ed010001ed2d000000010001ed2f0001ed3d000000010001f1" +
	"000001f10c000000010150000005280073000b0021002300010025002a000100" +
	"2c002f0001003a003b0001003f00400001005b005d0001005f007b001c007d00" +
	"a1002400a700ab000400b600b7000100bb00bf0004037e03870009055a055f00" +
	"010589058a000105be05c0000205c305c6000305f305f400010609060a000106" +
	"0c060d0001061b061d0002061e061f0001066a066d000106d40700002c070107" +
	"0d000107f707f900010830083e0001085e0964010609650970000b09fd0a7600" +
	"790af00c7701870c840df401700e4f0e5a000b0e5b0f0400a90f050f1200010f" +
	"140f3a00260f3b0f3d00010f850fd0004b0fd10fd400010fd90fda0001104a10" +
	"4f000110fb136002651361136800011400166e026e169b169c000116eb16ed00" +
	"0117351736000117d417d6000117d817da00011800180a00011944194500011a" +
	"1e1a1f00011aa01aa600011aa81aad00011b5a1b6000011b7d1b7e00011bfc1b" +
	"ff00011c3b1c3f00011c7e1c7f00011cc01cc700011cd32010033d2011202700" +
	"012030204300012045205100012053205e0001207d207e0001208d208e000123" +
	"08230b00012329232a000127682775000127c527c6000127e627ef0001298329" +
	"98000129d829db000129fc29fd00012cf92cfc00012cfe2cff00012d702e0000" +
	"902e012e2e00012e302e4f00012e522e5d000130013003000130083011000130" +
	"14301f00013030303d000d30a030fb005ba4fea4ff0001a60da60f0001a673a6" +
	"7e000ba6f2a6f70001a874a8770001a8cea8cf0001a8f8a8fa0001a8fca92e00" +
	"32a92fa95f0030a9c1a9cd0001a9dea9df0001aa5caa5f0001aadeaadf0001aa" +
	"f0aaf10001abebfd3e5153fd3ffe1000d1fe11fe190001fe30fe520001fe54fe" +
	"610001fe63fe680005fe6afe6b0001ff01ff030001ff05ff0a0001ff0cff0f00" +
	"01ff1aff1b0001ff1fff200001ff3bff3d0001ff3fff5b001cff5dff5f0002ff" +
	"60ff65000100340001010000010102000000010001039f000103d00000003100" +
	"01056f00010857000002e80001091f0001093f0000002000010a5000010a5800" +
	"00000100010a7f00010af00000007100010af100010af60000000100010b3900" +
	"010b3f0000000100010b9900010b9c0000000100010ead00010f55000000a800" +
	"010f5600010f590000000100010f8600010f8900000001000110470001104d00" +
	"000001000110bb000110bc00000001000110be000110c1000000010001114000" +
	"01114300000001000111740001117500000001000111c5000111c80000000100" +
	"0111cd000111db0000000e000111dd000111df00000001000112380001123d00" +
	"000001000112a90001144b000001a20001144c0001144f000000010001145a00" +
	"01145b000000010001145d000114c600000069000115c1000115d70000000100" +
	"0116410001164300000001000116600001166c00000001000116b90001173c00" +
	"0000830001173d0001173e000000010001183b00011944000001090001194500" +
	"01194600000001000119e200011a3f0000005d00011a4000011a460000000100" +
	"011a9a00011a9c0000000100011a9e00011aa20000000100011b0000011b0900" +
	"00000100011c4100011c450000000100011c7000011c710000000100011ef700" +
	"011ef80000000100011f4300011f4f0000000100011fff000124700000047100" +
	"012471000124740000000100012ff100012ff20000000100016a6e00016a6f00" +
	"00000100016af500016b370000004200016b3800016b3b0000000100016b4400" +
	"016e970000035300016e9800016e9a0000000100016fe20001bc9f00004cbd00" +
	"01da870001da8b000000010001e95e0001e95f00000001025063000000240005" +
	"0000005f203f1fe0204020540014fe33fe340001fe4dfe4f0001ff3fff3f0001" +
	"000002506400000054000b0000002d058a055d05be14000e4218062010080a20" +
	"11201500012e172e1a00032e3a2e3b00012e402e5d001d301c3030001430a0fe" +
	"31cd91fe32fe580026fe63ff0d00aa000100010ead00010ead00000001025065" +
	"00000090001700010029005d0034007d0f3b0ebe0f3d169c075f2046207e0038" +
	"208e2309027b230b232a001f27692775000227c627e7002127e927ef00022984" +
	"2998000229d929db000229fd2e2304262e252e2900022e562e5c000230093011" +
	"00023015301b0002301e301f0001fd3efe1800dafe36fe440002fe48fe5a0012" +
	"fe5cfe5e0002ff09ff3d0034ff5dff6300030000025066000000240005000000" +
	"bb20191f5e201d203a001d2e032e0500022e0a2e0d00032e1d2e210004000002" +
	"50690000002a0006000000ab20181f6d201b201c0001201f2039001a2e022e04" +
	"00022e092e0c00032e1c2e200004000002506f00000510007100080021002300" +
	"01002500270001002a002e0002002f003a000b003b003f00040040005c001c00" +
	"a100a7000600b600b7000100bf037e02bf0387055a01d3055b055f0001058905" +
	"c0003705c305c6000305f305f400010609060a0001060c060d0001061b061d00" +
	"02061e061f0001066a066d000106d40700002c0701070d000107f707f9000108" +
	"30083e0001085e0964010609650970000b09fd0a7600790af00c7701870c840d" +
	"f401700e4f0e5a000b0e5b0f0400a90f050f1200010f140f8500710fd00fd400" +
	"010fd90fda0001104a104f000110fb13600265136113680001166e16eb007d16" +
	"ec16ed000117351736000117d417d6000117d817da0001180018050001180718" +
	"0a00011944194500011a1e1a1f00011aa01aa600011aa81aad00011b5a1b6000" +
	"011b7d1b7e00011bfc1bff00011c3b1c3f00011c7e1c7f00011cc01cc700011c" +
	"d320160343201720200009202120270001203020380001203b203e0001204120" +
	"4300012047205100012053205500022056205e00012cf92cfc00012cfe2cff00" +
	"012d702e0000902e012e0600052e072e0800012e0b2e0e00032e0f2e1600012e" +
	"182e1900012e1b2e1e00032e1f2e2a000b2e2b2e2e00012e302e3900012e3c2e" +
	"3f00012e412e4300022e442e4f00012e522e540001300130030001303d30fb00" +
	"bea4fea4ff0001a60da60f0001a673a67e000ba6f2a6f70001a874a8770001a8" +
	"cea8cf0001a8f8a8fa0001a8fca92e0032a92fa95f0030a9c1a9cd0001a9dea9" +
	"df0001aa5caa5f0001aadeaadf0001aaf0aaf10001abebfe105225fe11fe1600" +
	"01fe19fe300017fe45fe460001fe49fe4c0001fe50fe520001fe54fe570001fe" +
	"5ffe610001fe68fe6a0002fe6bff010096ff02ff030001ff05ff070001ff0aff" +
	"0e0002ff0fff1a000bff1bff1f0004ff20ff3c001cff61ff640003ff65ff6500" +
	"0100330001010000010102000000010001039f000103d0000000310001056f00" +
	"010857000002e80001091f0001093f0000002000010a5000010a580000000100" +
	"010a7f00010af00000007100010af100010af60000000100010b3900010b3f00" +
	"00000100010b9900010b9c0000000100010f5500010f590000000100010f8600" +
	"010f8900000001000110470001104d00000001000110bb000110bc0000000100" +
	"0110be000110c100000001000111400001114300000001000111740001117500" +
	"000001000111c5000111c800000001000111cd000111db0000000e000111dd00" +
	"0111df00000001000112380001123d00000001000112a90001144b000001a200" +
	"01144c0001144f000000010001145a0001145b000000010001145d000114c600" +
	"000069000115c1000115d7000000010001164100011643000000010001166000" +
	"01166c00000001000116b90001173c000000830001173d0001173e0000000100" +
	"01183b0001194400000109000119450001194600000001000119e200011a3f00" +
	"00005d00011a4000011a460000000100011a9a00011a9c0000000100011a9e00" +
	"011aa20000000100011b0000011b090000000100011c4100011c450000000100" +
	"011c7000011c710000000100011ef700011ef80000000100011f4300011f4f00" +
	"00000100011fff000124700000047100012471000124740000000100012ff100" +
	"012ff20000000100016a6e00016a6f0000000100016af500016b370000004200" +
	"016b3800016b3b0000000100016b4400016e970000035300016e9800016e9a00" +
	"00000100016fe20001bc9f00004cbd0001da870001da8b000000010001e95e00" +
	"01e95f00000001025073000000a2001a00010028005b0033007b0f3a0ebf0f3c" +
	"169b075f201a201e00042045207d0038208d2308027b230a2329001f27682774" +
	"000227c527e6002127e827ee000229832997000229d829da000229fc2e220426" +
	"2e242e2800022e422e5500132e572e5b00023008301000023014301a0002301d" +
	"fd3fcd22fe17fe35001efe37fe430002fe47fe590012fe5bfe5d0002ff08ff3b" +
	"0033ff5bff5f0004ff62ff620001000001530000066c0081000a0024002b0007" +
	"003c003e0001005e00600002007c007e000200a200a6000100a800a9000100ac" +
	"00ae000200af00b1000100b400b8000400d700f7002002c202c5000102d202df" +
	"000102e502eb000102ed02ef000202f002ff000103750384000f038503f60071" +
	"0482058d010b058e058f0001060606080001060b060e0003060f06de00cf06e9" +
	"06fd001406fe07f600f807fe07ff0001088809f2016a09f309fa000709fb0af1" +
	"00f60b700bf300830bf40bfa00010c7f0d4f00d00d790e3f00c60f010f030001" +
	"0f130f1500020f160f1700010f1a0f1f00010f340f3800020fbe0fc500010fc7" +
	"0fcc00010fce0fcf00010fd50fd80001109e109f0001139013990001166d17db" +
	"016e194019de009e19df19ff00011b611b6a00011b741b7c00011fbd1fbf0002" +
	"1fc01fc100011fcd1fcf00011fdd1fdf00011fed1fef00011ffd1ffe00012044" +
	"2052000e207a207c0001208a208c000120a020c0000121002101000121032106" +
	"0001210821090001211421160002211721180001211e21230001212521290002" +
	"212e213a000c213b21400005214121440001214a214d0001214f218a003b218b" +
	"21900005219123070001230c23280001232b242600012440244a0001249c24e9" +
	"0001250027670001279427c4000127c727e5000127f029820001299929d70001" +
	"29dc29fb000129fe2b7300012b762b9500012b972bff00012ce52cea00012e50" +
	"2e5100012e802e9900012e9b2ef300012f002fd500012ff02ffb000130043012" +
	"000e30133020000d303630370001303e303f0001309b309c0001319031910001" +
	"3196319f000131c031e300013200321e0001322a324700013250326000103261" +
	"327f0001328a32b0000132c033ff00014dc04dff0001a490a4c60001a700a716" +
	"0001a720a7210001a789a78a0001a828a82b0001a836a8390001aa77aa790001" +
	"ab5bab6a000fab6bfb294fbefbb2fbc20001fd40fd4f0001fdcffdfc002dfdfd" +
	"fdff0001fe62fe640002fe65fe660001fe69ff04009bff0bff1c0011ff1dff1e" +
	"0001ff3eff400002ff5cff5e0002ffe0ffe60001ffe8ffee0001fffcfffd0001" +
	"0048000101370001013f000000010001017900010189000000010001018c0001" +
	"018e00000001000101900001019c00000001000101a0000101d0000000300001" +
	"01d1000101fc0000000100010877000108780000000100010ac80001173f0000" +
	"0c7700011fd500011ff10000000100016b3c00016b3f0000000100016b450001" +
	"bc9c000051570001cf500001cfc3000000010001d0000001d0f5000000010001" +
	"d1000001d126000000010001d1290001d164000000010001d16a0001d16c0000" +
	"00010001d1830001d184000000010001d18c0001d1a9000000010001d1ae0001" +
	"d1ea000000010001d2000001d241000000010001d2450001d300000000bb0001" +
	"d3010001d356000000010001d6c10001d6db0000001a0001d6fb0001d7150000" +
	"001a0001d7350001d74f0000001a0001d76f0001d7890000001a0001d7a90001" +
	"d7c30000001a0001d8000001d9ff000000010001da370001da3a000000010001" +
	"da6d0001da74000000010001da760001da83000000010001da850001da860000" +
	"00010001e14f0001e2ff000001b00001ecac0001ecb0000000040001ed2e0001" +
	"eef0000001c20001eef10001f0000000010f0001f0010001f02b000000010001" +
	"f0300001f093000000010001f0a00001f0ae000000010001f0b10001f0bf0000" +
	"00010001f0c10001f0cf000000010001f0d10001f0f5000000010001f10d0001" +
	"f1ad000000010001f1e60001f202000000010001f2100001f23b000000010001" +
	"f2400001f248000000010001f2500001f251000000010001f2600001f2650000" +
	"00010001f3000001f6d7000000010001f6dc0001f6ec000000010001f6f00001" +
	"f6fc000000010001f7000001f776000000010001f77b0001f7d9000000010001" +
	"f7e00001f7eb000000010001f7f00001f800000000100001f8010001f80b0000" +
	"00010001f8100001f847000000010001f8500001f859000000010001f8600001" +
	"f887000000010001f8900001f8ad000000010001f8b00001f8b1000000010001" +
	"f9000001fa53000000010001fa600001fa6d000000010001fa700001fa7c0000" +
	"00010001fa800001fa88000000010001fa900001fabd000000010001fabf0001" +
	"fac5000000010001face0001fadb000000010001fae00001fae8000000010001" +
	"faf00001faf8000000010001fb000001fb92000000010001fb940001fbca0000" +
	"00010253630000006c000d0002002400a2007e00a300a50001058f060b007c07" +
	"fe07ff000109f209f3000109fb0af100f60bf90e3f024617db20a008c520a120" +
	"c00001a838fdfc55c4fe69ff04009bffe0ffe10001ffe5ffe60001000200011f" +
	"dd00011fe0000000010001e2ff0001ecb0000009b102536b000000a800190003" +
	"005e0060000200a800af000700b400b8000402c202c5000102d202df000102e5" +
	"02eb000102ed02ef000202f002ff000103750384000f0385088805031fbd1fbf" +
	"00021fc01fc100011fcd1fcf00011fdd1fdf00011fed1fef00011ffd1ffe0001" +
	"309b309c0001a700a7160001a720a7210001a789a78a0001ab5bab6a000fab6b" +
	"fbb25047fbb3fbc20001ff3eff400002ffe3ffe3000100010001f3fb0001f3ff" +
	"0000000102536d00000150002b0005002b003c0011003d003e0001007c007e00" +
	"0200ac00b1000500d700f7002003f60606021006070608000120442052000e20" +
	"7a207c0001208a208c0001211821400028214121440001214b21900045219121" +
	"940001219a219b000121a021a6000321ae21ce002021cf21d2000321d421f400" +
	"2021f522ff0001232023210001237c239b001f239c23b3000123dc23e1000125" +
	"b725c1000a25f825ff0001266f27c0015127c127c4000127c727e5000127f027" +
	"ff0001290029820001299929d7000129dc29fb000129fe2aff00012b302b4400" +
	"012b472b4c0001fb29fe620339fe64fe660001ff0bff1c0011ff1dff1e0001ff" +
	"5cff5e0002ffe2ffe90007ffeaffec000100060001d6c10001d6db0000001a00" +
	"01d6fb0001d7150000001a0001d7350001d74f0000001a0001d76f0001d78900" +
	"00001a0001d7a90001d7c30000001a0001eef00001eef10000000102536f0000" +
	"057c0063000200a600a9000300ae00b000020482058d010b058e060e0080060f" +
	"06de00cf06e906fd001406fe07f600f809fa0b7001760bf30bf800010bfa0c7f" +
	"00850d4f0d79002a0f010f0300010f130f1500020f160f1700010f1a0f1f0001" +
	"0f340f3800020fbe0fc500010fc70fcc00010fce0fcf00010fd50fd80001109e" +
	"109f0001139013990001166d194002d319de19ff00011b611b6a00011b741b7c" +
	"00012100210100012103210600012108210900012114211600022117211e0007" +
	"211f21230001212521290002212e213a000c213b214a000f214c214d0001214f" +
	"218a003b218b2195000a219621990001219c219f000121a121a2000121a421a5" +
	"000121a721ad000121af21cd000121d021d1000121d321d5000221d621f30001" +
	"230023070001230c231f0001232223280001232b237b0001237d239a000123b4" +
	"23db000123e2242600012440244a0001249c24e90001250025b6000125b825c0" +
	"000125c225f700012600266e0001267027670001279427bf0001280028ff0001" +
	"2b002b2f00012b452b4600012b4d2b7300012b762b9500012b972bff00012ce5" +
	"2cea00012e502e5100012e802e9900012e9b2ef300012f002fd500012ff02ffb" +
	"000130043012000e30133020000d303630370001303e303f0001319031910001" +
	"3196319f000131c031e300013200321e0001322a324700013250326000103261" +
	"327f0001328a32b0000132c033ff00014dc04dff0001a490a4c60001a828a82b" +
	"0001a836a8370001a839aa77023eaa78aa790001fd40fd4f0001fdcffdfd002e" +
	"fdfefdff0001ffe4ffe80004ffedffee0001fffcfffd00010043000101370001" +
	"013f000000010001017900010189000000010001018c0001018e000000010001" +
	"01900001019c00000001000101a0000101d000000030000101d1000101fc0000" +
	"000100010877000108780000000100010ac80001173f00000c7700011fd50001" +
	"1fdc0000000100011fe100011ff10000000100016b3c00016b3f000000010001" +
	"6b450001bc9c000051570001cf500001cfc3000000010001d0000001d0f50000" +
	"00010001d1000001d126000000010001d1290001d164000000010001d16a0001" +
	"d16c000000010001d1830001d184000000010001d18c0001d1a9000000010001" +
	"d1ae0001d1ea000000010001d2000001d241000000010001d2450001d3000000" +
	"00bb0001d3010001d356000000010001d8000001d9ff000000010001da370001" +
	"da3a000000010001da6d0001da74000000010001da760001da83000000010001" +
	"da850001da86000000010001e14f0001ecac00000b5d0001ed2e0001f0000000" +
	"02d20001f0010001f02b000000010001f0300001f093000000010001f0a00001" +
	"f0ae000000010001f0b10001f0bf000000010001f0c10001f0cf000000010001" +
	"f0d10001f0f5000000010001f10d0001f1ad000000010001f1e60001f2020000" +
	"00010001f2100001f23b000000010001f2400001f248000000010001f2500001" +
	"f251000000010001f2600001f265000000010001f3000001f3fa000000010001" +
	"f4000001f6d7000000010001f6dc0001f6ec000000010001f6f00001f6fc0000" +
	"00010001f7000001f776000000010001f77b0001f7d9000000010001f7e00001" +
	"f7eb000000010001f7f00001f800000000100001f8010001f80b000000010001" +
	"f8100001f847000000010001f8500001f859000000010001f8600001f8870000" +
	"00010001f8900001f8ad000000010001f8b00001f8b1000000010001f9000001" +
	"fa53000000010001fa600001fa6d000000010001fa700001fa7c000000010001" +
	"fa800001fa88000000010001fa900001fabd000000010001fabf0001fac50000" +
	"00010001face0001fadb000000010001fae00001fae8000000010001faf00001" +
	"faf8000000010001fb000001fb92000000010001fb940001fbca00000001015a" +
	"0000002a00060001002000a000801680200009802001200a0001202820290001" +
	"202f205f00303000300000010000025a6c0000000c0001000020282028000100" +
	"00025a700000000c000100002029202900010000025a73000000240005000100" +
	"2000a000801680200009802001200a0001202f205f00303000300000010000"

// SCRIPT_TABLE_DATA stores the Unicode script tables.
const SCRIPT_TABLE_DATA Encoded_Data = "" +
	"0541646c616d0000002a0000000000030001e9000001e94b000000010001e950" +
	"0001e959000000010001e95e0001e95f000000010441686f6d0000002a000000" +
	"000003000117000001171a000000010001171d0001172b000000010001173000" +
	"0117460000000115416e61746f6c69616e5f486965726f676c79706873000000" +
	"1200000000000100014400000146460000000106417261626963000001ce0016" +
	"00000600060400010606060b0001060d061a0001061c061e00010620063f0001" +
	"0641064a00010656066f0001067106dc000106de06ff00010750077f00010870" +
	"088e0001089008910001089808e1000108e308ff0001fb50fbc20001fbd3fd3d" +
	"0001fd40fd8f0001fd92fdc70001fdcffdf00021fdf1fdff0001fe70fe740001" +
	"fe76fefc0001001b00010e6000010e7e0000000100010efd00010eff00000001" +
	"0001ee000001ee03000000010001ee050001ee1f000000010001ee210001ee22" +
	"000000010001ee240001ee27000000030001ee290001ee32000000010001ee34" +
	"0001ee37000000010001ee390001ee3b000000020001ee420001ee4700000005" +
	"0001ee490001ee4d000000020001ee4e0001ee4f000000010001ee510001ee52" +
	"000000010001ee540001ee57000000030001ee590001ee61000000020001ee62" +
	"0001ee64000000020001ee670001ee6a000000010001ee6c0001ee7200000001" +
	"0001ee740001ee77000000010001ee790001ee7c000000010001ee7e0001ee80" +
	"000000020001ee810001ee89000000010001ee8b0001ee9b000000010001eea1" +
	"0001eea3000000010001eea50001eea9000000010001eeab0001eebb00000001" +
	"0001eef00001eef1000000010841726d656e69616e0000001e00040000053105" +
	"5600010559058a0001058d058f0001fb13fb1700010000074176657374616e00" +
	"00001e00000000000200010b0000010b350000000100010b3900010b3f000000" +
	"010842616c696e65736500000012000200001b001b4c00011b501b7e00010000" +
	"0542616d756d0000001800010000a6a0a6f7000100010001680000016a380000" +
	"00010942617373615f5661680000001e00000000000200016ad000016aed0000" +
	"000100016af000016af50000000105426174616b00000012000200001bc01bf3" +
	"00011bfc1bff000100000742656e67616c690000005a000e0000098009830001" +
	"0985098c0001098f09900001099309a8000109aa09b0000109b209b6000409b7" +
	"09b9000109bc09c4000109c709c8000109cb09ce000109d709dc000509dd09df" +
	"000209e009e3000109e609fe0001000009426861696b73756b69000000360000" +
	"0000000400011c0000011c080000000100011c0a00011c360000000100011c38" +
	"00011c450000000100011c5000011c6c0000000108426f706f6d6f666f000000" +
	"180003000002ea02eb00013105312f000131a031bf0001000006427261686d69" +
	"0000002a000000000003000110000001104d0000000100011052000110750000" +
	"00010001107f0001107f0000000107427261696c6c650000000c000100002800" +
	"28ff0001000008427567696e65736500000012000200001a001a1b00011a1e1a" +
	"1f000100000542756869640000000c0001000017401753000100001343616e61" +
	"6469616e5f41626f726967696e616c0000001e000200001400167f000118b018" +
	"f50001000100011ab000011abf000000010643617269616e0000001200000000" +
	"0001000102a0000102d0000000011243617563617369616e5f416c62616e6961" +
	"6e0000001e0000000000020001053000010563000000010001056f0001056f00" +
	"000001064368616b6d610000001e000000000002000111000001113400000001" +
	"000111360001114700000001044368616d0000001e00040000aa00aa360001aa" +
	"40aa4d0001aa50aa590001aa5caa5f0001000008436865726f6b656500000018" +
	"0003000013a013f5000113f813fd0001ab70abbf000100000a43686f7261736d" +
	"69616e0000001200000000000100010fb000010fcb0000000106436f6d6d6f6e" +
	"000005ca00520006000000400001005b00600001007b00a9000100ab00b90001" +
	"00bb00bf000100d700f7002002b902df000102e502e9000102ec02ff00010374" +
	"037e000a0385038700020605060c0007061b061f0004064006dd009d08e20964" +
	"008209650e3f04da0fd50fd8000110fb16eb05f016ec16ed0001173517360001" +
	"18021803000118051cd304ce1ce11ce900081cea1cec00011cee1cf300011cf5" +
	"1cf700011cfa200003062001200b0001200e206400012066207000012074207e" +
	"00012080208e000120a020c00001210021250001212721290001212c21310001" +
	"2133214d0001214f215f00012189218b00012190242600012440244a00012460" +
	"27ff000129002b7300012b762b9500012b972bff00012e002e5d00012ff02ffb" +
	"0001300030040001300630080002300930200001303030370001303c303f0001" +
	"309b309c000130a030fb005b30fc319000943191319f000131c031e300013220" +
	"325f0001327f32cf000132ff33580059335933ff00014dc04dff0001a700a721" +
	"0001a788a78a0001a830a8390001a92ea9cf00a1ab5bab6a000fab6bfd3e51d3" +
	"fd3ffe1000d1fe11fe190001fe30fe520001fe54fe660001fe68fe6b0001feff" +
	"ff010002ff02ff200001ff3bff400001ff5bff650001ff70ff9e002eff9fffe0" +
	"0041ffe1ffe60001ffe8ffee0001fff9fffd0001005200010100000101020000" +
	"0001000101070001013300000001000101370001013f00000001000101900001" +
	"019c00000001000101d0000101fc00000001000102e1000102fb000000010001" +
	"bca00001bca3000000010001cf500001cfc3000000010001d0000001d0f50000" +
	"00010001d1000001d126000000010001d1290001d166000000010001d16a0001" +
	"d17a000000010001d1830001d184000000010001d18c0001d1a9000000010001" +
	"d1ae0001d1ea000000010001d2c00001d2d3000000010001d2e00001d2f30000" +
	"00010001d3000001d356000000010001d3600001d378000000010001d4000001" +
	"d454000000010001d4560001d49c000000010001d49e0001d49f000000010001" +
	"d4a20001d4a5000000030001d4a60001d4a9000000030001d4aa0001d4ac0000" +
	"00010001d4ae0001d4b9000000010001d4bb0001d4bd000000020001d4be0001" +
	"d4c3000000010001d4c50001d505000000010001d5070001d50a000000010001" +
	"d50d0001d514000000010001d5160001d51c000000010001d51e0001d5390000" +
	"00010001d53b0001d53e000000010001d5400001d544000000010001d5460001" +
	"d54a000000040001d54b0001d550000000010001d5520001d6a5000000010001" +
	"d6a80001d7cb000000010001d7ce0001d7ff000000010001ec710001ecb40000" +
	"00010001ed010001ed3d000000010001f0000001f02b000000010001f0300001" +
	"f093000000010001f0a00001f0ae000000010001f0b10001f0bf000000010001" +
	"f0c10001f0cf000000010001f0d10001f0f5000000010001f1000001f1ad0000" +
	"00010001f1e60001f1ff000000010001f2010001f202000000010001f2100001" +
	"f23b000000010001f2400001f248000000010001f2500001f251000000010001" +
	"f2600001f265000000010001f3000001f6d7000000010001f6dc0001f6ec0000" +
	"00010001f6f00001f6fc000000010001f7000001f776000000010001f77b0001" +
	"f7d9000000010001f7e00001f7eb000000010001f7f00001f800000000100001" +
	"f8010001f80b000000010001f8100001f847000000010001f8500001f8590000" +
	"00010001f8600001f887000000010001f8900001f8ad000000010001f8b00001" +
	"f8b1000000010001f9000001fa53000000010001fa600001fa6d000000010001" +
	"fa700001fa7c000000010001fa800001fa88000000010001fa900001fabd0000" +
	"00010001fabf0001fac5000000010001face0001fadb000000010001fae00001" +
	"fae8000000010001faf00001faf8000000010001fb000001fb92000000010001" +
	"fb940001fbca000000010001fbf00001fbf900000001000e0001000e00200000" +
	"001f000e0021000e007f0000000106436f70746963000000180003000003e203" +
	"ef00012c802cf300012cf92cff000100000943756e6569666f726d0000003600" +
	"0000000004000120000001239900000001000124000001246e00000001000124" +
	"7000012474000000010001248000012543000000010743797072696f74000000" +
	"42000000000005000108000001080500000001000108080001080a0000000200" +
	"01080b00010835000000010001083700010838000000010001083c0001083f00" +
	"0000030c437970726f5f4d696e6f616e0000001200000000000100012f900001" +
	"2ff20000000108437972696c6c69630000004800070000040004840001048705" +
	"2f00011c801c8800011d2b1d78004d2de02dff0001a640a69f0001fe2efe2f00" +
	"0100020001e0300001e06d000000010001e08f0001e08f000000010744657365" +
	"72657400000012000000000001000104000001044f000000010a446576616e61" +
	"676172690000002a000400000900095000010955096300010966097f0001a8e0" +
	"a8ff0001000100011b0000011b09000000010b44697665735f416b7572750000" +
	"0066000000000008000119000001190600000001000119090001190c00000003" +
	"0001190d00011913000000010001191500011916000000010001191800011935" +
	"000000010001193700011938000000010001193b000119460000000100011950" +
	"000119590000000105446f67726100000012000000000001000118000001183b" +
	"00000001084475706c6f79616e000000420000000000050001bc000001bc6a00" +
	"0000010001bc700001bc7c000000010001bc800001bc88000000010001bc9000" +
	"01bc99000000010001bc9c0001bc9f0000000114456779707469616e5f486965" +
	"726f676c79706873000000120000000000010001300000013455000000010745" +
	"6c626173616e0000001200000000000100010500000105270000000107456c79" +
	"6d6169630000001200000000000100010fe000010ff60000000108457468696f" +
	"706963000000f600200000120012480001124a124d0001125012560001125812" +
	"5a0002125b125d0001126012880001128a128d0001129012b0000112b212b500" +
	"0112b812be000112c012c2000212c312c5000112c812d6000112d81310000113" +
	"12131500011318135a0001135d137c00011380139900012d802d9600012da02d" +
	"a600012da82dae00012db02db600012db82dbe00012dc02dc600012dc82dce00" +
	"012dd02dd600012dd82dde0001ab01ab060001ab09ab0e0001ab11ab160001ab" +
	"20ab260001ab28ab2e000100040001e7e00001e7e6000000010001e7e80001e7" +
	"eb000000010001e7ed0001e7ee000000010001e7f00001e7fe00000001084765" +
	"6f726769616e000000360008000010a010c5000110c710cd000610d010fa0001" +
	"10fc10ff00011c901cba00011cbd1cbf00012d002d2500012d272d2d00060000" +
	"0a476c61676f6c6974696300000048000100002c002c5f000100050001e00000" +
	"01e006000000010001e0080001e018000000010001e01b0001e0210000000100" +
	"01e0230001e024000000010001e0260001e02a0000000106476f746869630000" +
	"0012000000000001000103300001034a00000001074772616e746861000000ae" +
	"00000000000e000113000001130300000001000113050001130c000000010001" +
	"130f00011310000000010001131300011328000000010001132a000113300000" +
	"00010001133200011333000000010001133500011339000000010001133c0001" +
	"1344000000010001134700011348000000010001134b0001134d000000010001" +
	"135000011357000000070001135d0001136300000001000113660001136c0000" +
	"000100011370000113740000000105477265656b000000d8001d000003700373" +
	"0001037503770001037a037d0001037f038400050386038800020389038a0001" +
	"038c038e0002038f03a1000103a303e1000103f003ff00011d261d2a00011d5d" +
	"1d6100011d661d6a00011dbf1f0001411f011f1500011f181f1d00011f201f45" +
	"00011f481f4d00011f501f5700011f591f5f00021f601f7d00011f801fb40001" +
	"1fb61fc400011fc61fd300011fd61fdb00011fdd1fef00011ff21ff400011ff6" +
	"1ffe00012126ab658a3f0003000101400001018e00000001000101a00001d200" +
	"0000d0600001d2010001d245000000010847756a61726174690000005a000e00" +
	"000a810a8300010a850a8d00010a8f0a9100010a930aa800010aaa0ab000010a" +
	"b20ab300010ab50ab900010abc0ac500010ac70ac900010acb0acd00010ad00a" +
	"e000100ae10ae300010ae60af100010af90aff000100000d47756e6a616c615f" +
	"476f6e64690000004e00000000000600011d6000011d650000000100011d6700" +
	"011d680000000100011d6a00011d8e0000000100011d9000011d910000000100" +
	"011d9300011d980000000100011da000011da900000001084775726d756b6869" +
	"00000066001000000a010a0300010a050a0a00010a0f0a1000010a130a280001" +
	"0a2a0a3000010a320a3300010a350a3600010a380a3900010a3c0a3e00020a3f" +
	"0a4200010a470a4800010a4b0a4d00010a510a5900080a5a0a5c00010a5e0a66" +
	"00080a670a76000100000348616e000000ba000a00002e802e9900012e9b2ef3" +
	"00012f002fd500013005300700023021302900013038303b000134004dbf0001" +
	"4e009fff0001f900fa6d0001fa70fad90001000a00016fe200016fe300000001" +
	"00016ff000016ff100000001000200000002a6df000000010002a7000002b739" +
	"000000010002b7400002b81d000000010002b8200002cea1000000010002ceb0" +
	"0002ebe0000000010002f8000002fa1d00000001000300000003134a00000001" +
	"00031350000323af000000010648616e67756c0000005a000e0000110011ff00" +
	"01302e302f00013131318e00013200321e00013260327e0001a960a97c0001ac" +
	"00d7a30001d7b0d7c60001d7cbd7fb0001ffa0ffbe0001ffc2ffc70001ffcaff" +
	"cf0001ffd2ffd70001ffdaffdc000100000f48616e6966695f526f68696e6779" +
	"610000001e00000000000200010d0000010d270000000100010d3000010d3900" +
	"0000010748616e756e6f6f0000000c0001000017201734000100000648617472" +
	"616e0000002a000000000003000108e0000108f200000001000108f4000108f5" +
	"00000001000108fb000108ff00000001064865627265770000003c0009000005" +
	"9105c7000105d005ea000105ef05f40001fb1dfb360001fb38fb3c0001fb3efb" +
	"400002fb41fb430002fb44fb460002fb47fb4f00010000084869726167616e61" +
	"0000004200020000304130960001309d309f000100040001b0010001b11f0000" +
	"00010001b1320001b1500000001e0001b1510001b152000000010001f2000001" +
	"f2000000000110496d70657269616c5f4172616d6169630000001e0000000000" +
	"02000108400001085500000001000108570001085f0000000109496e68657269" +
	"746564000000de001200000300036f0001048504860001064b06550001067009" +
	"5102e10952095400011ab01ace00011cd01cd200011cd41ce000011ce21ce800" +
	"011ced1cf400071cf81cf900011dc01dff0001200c200d000120d020f0000130" +
	"2a302d00013099309a0001fe00fe0f0001fe20fe2d00010009000101fd000102" +
	"e0000000e30001133b0001cf000000bbc50001cf010001cf2d000000010001cf" +
	"300001cf46000000010001d1670001d169000000010001d17b0001d182000000" +
	"010001d1850001d18b000000010001d1aa0001d1ad00000001000e0100000e01" +
	"ef0000000115496e736372697074696f6e616c5f5061686c6176690000001e00" +
	"000000000200010b6000010b720000000100010b7800010b7f0000000116496e" +
	"736372697074696f6e616c5f506172746869616e0000001e0000000000020001" +
	"0b4000010b550000000100010b5800010b5f00000001084a6176616e65736500" +
	"00001800030000a980a9cd0001a9d0a9d90001a9dea9df00010000064b616974" +
	"68690000001e00000000000200011080000110c200000001000110cd000110cd" +
	"00000001074b616e6e61646100000054000d00000c800c8c00010c8e0c900001" +
	"0c920ca800010caa0cb300010cb50cb900010cbc0cc400010cc60cc800010cca" +
	"0ccd00010cd50cd600010cdd0cde00010ce00ce300010ce60cef00010cf10cf3" +
	"00010000084b6174616b616e61000000840007000030a130fa000130fd30ff00" +
	"0131f031ff000132d032fe0001330033570001ff66ff6f0001ff71ff9d000100" +
	"070001aff00001aff3000000010001aff50001affb000000010001affd0001af" +
	"fe000000010001b0000001b120000001200001b1210001b122000000010001b1" +
	"550001b1640000000f0001b1650001b16700000001044b6177690000002a0000" +
	"0000000300011f0000011f100000000100011f1200011f3a0000000100011f3e" +
	"00011f5900000001084b617961685f4c690000001200020000a900a92d0001a9" +
	"2fa92f000100000a4b6861726f73687468690000006600000000000800010a00" +
	"00010a030000000100010a0500010a060000000100010a0c00010a1300000001" +
	"00010a1500010a170000000100010a1900010a350000000100010a3800010a3a" +
	"0000000100010a3f00010a480000000100010a5000010a5800000001134b6869" +
	"74616e5f536d616c6c5f5363726970740000001e00000000000200016fe40001" +
	"8b0000001b1c00018b0100018cd500000001054b686d65720000001e00040000" +
	"178017dd000117e017e9000117f017f9000119e019ff00010000064b686f6a6b" +
	"690000001e000000000002000112000001121100000001000112130001124100" +
	"000001094b68756461776164690000001e000000000002000112b0000112ea00" +
	"000001000112f0000112f900000001034c616f00000048000b00000e810e8200" +
	"010e840e8600020e870e8a00010e8c0ea300010ea50ea700020ea80ebd00010e" +
	"c00ec400010ec60ec800020ec90ece00010ed00ed900010edc0edf0001000005" +
	"4c6174696e000000fc001f00050041005a00010061007a000100aa00ba001000" +
	"c000d6000100d800f6000100f802b8000102e002e400011d001d2500011d2c1d" +
	"5c00011d621d6500011d6b1d7700011d791dbe00011e001eff00012071207f00" +
	"0e2090209c0001212a212b00012132214e001c2160218800012c602c7f0001a7" +
	"22a7870001a78ba7ca0001a7d0a7d10001a7d3a7d50002a7d6a7d90001a7f2a7" +
	"ff0001ab30ab5a0001ab5cab640001ab66ab690001fb00fb060001ff21ff3a00" +
	"01ff41ff5a0001000500010780000107850000000100010787000107b0000000" +
	"01000107b2000107ba000000010001df000001df1e000000010001df250001df" +
	"2a00000001064c657063686100000018000300001c001c3700011c3b1c490001" +
	"1c4d1c4f00010000054c696d627500000024000500001900191e00011920192b" +
	"00011930193b00011940194400041945194f00010000084c696e6561725f4100" +
	"00002a0000000000030001060000010736000000010001074000010755000000" +
	"01000107600001076700000001084c696e6561725f420000005a000000000007" +
	"000100000001000b000000010001000d0001002600000001000100280001003a" +
	"000000010001003c0001003d000000010001003f0001004d0000000100010050" +
	"0001005d0000000100010080000100fa00000001044c69737500000018000100" +
	"00a4d0a4ff0001000100011fb000011fb000000001064c796369616e00000012" +
	"000000000001000102800001029c00000001064c796469616e0000001e000000" +
	"0000020001092000010939000000010001093f0001093f00000001084d616861" +
	"6a616e6900000012000000000001000111500001117600000001074d616b6173" +
	"61720000001200000000000100011ee000011ef800000001094d616c6179616c" +
	"616d00000030000700000d000d0c00010d0e0d1000010d120d4400010d460d48" +
	"00010d4a0d4f00010d540d6300010d660d7f00010000074d616e646169630000" +
	"0012000200000840085b0001085e085e000100000a4d616e6963686165616e00" +
	"00001e00000000000200010ac000010ae60000000100010aeb00010af6000000" +
	"01074d61726368656e0000002a00000000000300011c7000011c8f0000000100" +
	"011c9200011ca70000000100011ca900011cb6000000010d4d61736172616d5f" +
	"476f6e64690000005a00000000000700011d0000011d060000000100011d0800" +
	"011d090000000100011d0b00011d360000000100011d3a00011d3c0000000200" +
	"011d3d00011d3f0000000200011d4000011d470000000100011d5000011d5900" +
	"0000010b4d6564656661696472696e0000001200000000000100016e4000016e" +
	"9a000000010c4d65657465695f4d6179656b0000001800030000aae0aaf60001" +
	"abc0abed0001abf0abf9000100000d4d656e64655f4b696b616b75690000001e" +
	"0000000000020001e8000001e8c4000000010001e8c70001e8d600000001104d" +
	"65726f697469635f437572736976650000002a000000000003000109a0000109" +
	"b700000001000109bc000109cf00000001000109d2000109ff00000001144d65" +
	"726f697469635f486965726f676c797068730000001200000000000100010980" +
	"0001099f00000001044d69616f0000002a00000000000300016f0000016f4a00" +
	"00000100016f4f00016f870000000100016f8f00016f9f00000001044d6f6469" +
	"0000001e00000000000200011600000116440000000100011650000116590000" +
	"0001094d6f6e676f6c69616e0000003000050000180018010001180418060002" +
	"180718190001182018780001188018aa00010001000116600001166c00000001" +
	"034d726f0000002a00000000000300016a4000016a5e0000000100016a600001" +
	"6a690000000100016a6e00016a6f00000001074d756c74616e69000000420000" +
	"00000005000112800001128600000001000112880001128a000000020001128b" +
	"0001128d000000010001128f0001129d000000010001129f000112a900000001" +
	"074d79616e6d617200000018000300001000109f0001a9e0a9fe0001aa60aa7f" +
	"00010000094e616261746165616e0000001e000000000002000108800001089e" +
	"00000001000108a7000108af000000010b4e61675f4d756e6461726900000012" +
	"0000000000010001e4d00001e4f9000000010b4e616e64696e61676172690000" +
	"002a000000000003000119a0000119a700000001000119aa000119d700000001" +
	"000119da000119e4000000010b4e65775f5461695f4c75650000001e00040000" +
	"198019ab000119b019c9000119d019da000119de19df00010000044e65776100" +
	"00001e000000000002000114000001145b000000010001145d00011461000000" +
	"01034e6b6f000000120002000007c007fa000107fd07ff00010000054e757368" +
	"750000001e00000000000200016fe10001b1700000418f0001b1710001b2fb00" +
	"000001164e7969616b656e675f507561636875655f486d6f6e67000000360000" +
	"000000040001e1000001e12c000000010001e1300001e13d000000010001e140" +
	"0001e149000000010001e14e0001e14f00000001054f6768616d0000000c0001" +
	"00001680169c00010000084f6c5f4368696b690000000c000100001c501c7f00" +
	"0100000d4f6c645f48756e67617269616e0000002a00000000000300010c8000" +
	"010cb20000000100010cc000010cf20000000100010cfa00010cff000000010a" +
	"4f6c645f4974616c69630000001e000000000002000103000001032300000001" +
	"0001032d0001032f00000001114f6c645f4e6f7274685f4172616269616e0000" +
	"001200000000000100010a8000010a9f000000010a4f6c645f5065726d696300" +
	"000012000000000001000103500001037a000000010b4f6c645f506572736961" +
	"6e0000001e000000000002000103a0000103c300000001000103c8000103d500" +
	"0000010b4f6c645f536f676469616e0000001200000000000100010f0000010f" +
	"2700000001114f6c645f536f7574685f4172616269616e000000120000000000" +
	"0100010a6000010a7f000000010a4f6c645f5475726b69630000001200000000" +
	"000100010c0000010c48000000010a4f6c645f55796768757200000012000000" +
	"00000100010f7000010f8900000001054f726979610000005a000e00000b010b" +
	"0300010b050b0c00010b0f0b1000010b130b2800010b2a0b3000010b320b3300" +
	"010b350b3900010b3c0b4400010b470b4800010b4b0b4d00010b550b5700010b" +
	"5c0b5d00010b5f0b6300010b660b7700010000054f736167650000001e000000" +
	"000002000104b0000104d300000001000104d8000104fb00000001074f736d61" +
	"6e79610000001e000000000002000104800001049d00000001000104a0000104" +
	"a9000000010c5061686177685f486d6f6e670000004200000000000500016b00" +
	"00016b450000000100016b5000016b590000000100016b5b00016b6100000001" +
	"00016b6300016b770000000100016b7d00016b8f000000010950616c6d797265" +
	"6e6500000012000000000001000108600001087f000000010b5061755f43696e" +
	"5f4861750000001200000000000100011ac000011af800000001085068616773" +
	"5f50610000000c00010000a840a877000100000a50686f656e696369616e0000" +
	"001e000000000002000109000001091b000000010001091f0001091f00000001" +
	"0f5073616c7465725f5061686c6176690000002a00000000000300010b800001" +
	"0b910000000100010b9900010b9c0000000100010ba900010baf000000010652" +
	"656a616e670000001200020000a930a9530001a95fa95f000100000552756e69" +
	"63000000120002000016a016ea000116ee16f8000100000953616d6172697461" +
	"6e00000012000200000800082d00010830083e000100000a5361757261736874" +
	"72610000001200020000a880a8c50001a8cea8d9000100000753686172616461" +
	"0000001200000000000100011180000111df00000001075368617669616e0000" +
	"0012000000000001000104500001047f00000001075369646468616d0000001e" +
	"00000000000200011580000115b500000001000115b8000115dd000000010b53" +
	"69676e57726974696e670000002a0000000000030001d8000001da8b00000001" +
	"0001da9b0001da9f000000010001daa10001daaf000000010753696e68616c61" +
	"0000005a000c00000d810d8300010d850d9600010d9a0db100010db30dbb0001" +
	"0dbd0dc000030dc10dc600010dca0dcf00050dd00dd400010dd60dd800020dd9" +
	"0ddf00010de60def00010df20df400010001000111e1000111f4000000010753" +
	"6f676469616e0000001200000000000100010f3000010f59000000010c536f72" +
	"615f536f6d70656e670000001e000000000002000110d0000110e80000000100" +
	"0110f0000110f90000000107536f796f6d626f0000001200000000000100011a" +
	"5000011aa2000000010953756e64616e65736500000012000200001b801bbf00" +
	"011cc01cc7000100000c53796c6f74695f4e616772690000000c00010000a800" +
	"a82c00010000065379726961630000001e000400000700070d0001070f074a00" +
	"01074d074f00010860086a0001000007546167616c6f67000000120002000017" +
	"0017150001171f171f000100000854616762616e776100000018000300001760" +
	"176c0001176e177000011772177300010000065461695f4c6500000012000200" +
	"001950196d00011970197400010000085461695f5468616d0000002400050000" +
	"1a201a5e00011a601a7c00011a7f1a8900011a901a9900011aa01aad00010000" +
	"085461695f566965740000001200020000aa80aac20001aadbaadf0001000005" +
	"54616b72690000001e00000000000200011680000116b900000001000116c000" +
	"0116c9000000010554616d696c00000078000f00000b820b8300010b850b8a00" +
	"010b8e0b9000010b920b9500010b990b9a00010b9c0b9e00020b9f0ba300040b" +
	"a40ba800040ba90baa00010bae0bb900010bbe0bc200010bc60bc800010bca0b" +
	"cd00010bd00bd700070be60bfa0001000200011fc000011ff10000000100011f" +
	"ff00011fff000000010654616e6773610000001e00000000000200016a700001" +
	"6abe0000000100016ac000016ac9000000010654616e67757400000036000000" +
	"00000400016fe0000170000000002000017001000187f7000000010001880000" +
	"018aff0000000100018d0000018d08000000010654656c75677500000054000d" +
	"00000c000c0c00010c0e0c1000010c120c2800010c2a0c3900010c3c0c440001" +
	"0c460c4800010c4a0c4d00010c550c5600010c580c5a00010c5d0c6000030c61" +
	"0c6300010c660c6f00010c770c7f0001000006546861616e610000000c000100" +
	"00078007b100010000045468616900000012000200000e010e3a00010e400e5b" +
	"00010000075469626574616e00000030000700000f000f4700010f490f6c0001" +
	"0f710f9700010f990fbc00010fbe0fcc00010fce0fd400010fd90fda00010000" +
	"08546966696e61676800000018000300002d302d6700012d6f2d7000012d7f2d" +
	"7f0001000007546972687574610000001e00000000000200011480000114c700" +
	"000001000114d0000114d90000000104546f746f000000120000000000010001" +
	"e2900001e2ae000000010855676172697469630000001e000000000002000103" +
	"800001039d000000010001039f0001039f00000001035661690000000c000100" +
	"00a500a62b0001000008566974686b7571690000006600000000000800010570" +
	"0001057a000000010001057c0001058a000000010001058c0001059200000001" +
	"00010594000105950000000100010597000105a100000001000105a3000105b1" +
	"00000001000105b3000105b900000001000105bb000105bc000000010657616e" +
	"63686f0000001e0000000000020001e2c00001e2f9000000010001e2ff0001e2" +
	"ff000000010b576172616e675f436974690000001e000000000002000118a000" +
	"0118f200000001000118ff000118ff000000010659657a6964690000002a0000" +
	"0000000300010e8000010ea90000000100010eab00010ead0000000100010eb0" +
	"00010eb1000000010259690000001200020000a000a48c0001a490a4c6000100" +
	"00105a616e6162617a61725f5371756172650000001200000000000100011a00" +
	"00011a4700000001"

// PROPERTY_TABLE_DATA stores the Unicode property tables.
const PROPERTY_TABLE_DATA Encoded_Data = "" +
	"0f41534349495f4865785f446967697400000018000300030030003900010041" +
	"0046000100610066000100000c426964695f436f6e74726f6c0000001e000400" +
	"00061c200e19f2200f202a001b202b202e000120662069000100000444617368" +
	"00000060000d0000002d058a055d05be14000e4218062010080a201120150001" +
	"2053207b0028208b221201872e172e1a00032e3a2e3b00012e402e5d001d301c" +
	"3030001430a0fe31cd91fe32fe580026fe63ff0d00aa000100010ead00010ead" +
	"000000010a44657072656361746564000000300005000001490673052a0f770f" +
	"79000217a317a40001206a206f00012329232a00010001000e0001000e000100" +
	"0000010944696163726974696300000516006e0003005e0060000200a800af00" +
	"0700b400b7000300b802b001f802b1034e0001035003570001035d0362000103" +
	"7403750001037a0384000a0385048300fe048404870001055905910038059205" +
	"a1000105a305bd000105bf05c1000205c205c40002064b065200010657065800" +
	"0106df06e0000106e506e6000106ea06ec00010730074a000107a607b0000107" +
	"eb07f500010818081900010898089f000108c908d2000108e308fe0001093c09" +
	"4d0011095109540001097109bc004b09cd0a3c006f0a4d0abc006f0acd0afd00" +
	"300afe0aff00010b3c0b4d00110b550bcd00780c3c0c4d00110cbc0ccd00110d" +
	"3b0d3c00010d4d0e47007d0e480e4c00010e4e0eba006c0ec80ecc00010f180f" +
	"1900010f350f3900020f3e0f3f00010f820f8400010f860f8700010fc6103700" +
	"711039103a00011063106400011069106d00011087108d0001108f109a000b10" +
	"9b135d02c2135e135f000117141715000117c917d3000117dd1939015c193a19" +
	"3b00011a751a7c00011a7f1ab000311ab11abe00011ac11acb00011b341b4400" +
	"101b6b1b7300011baa1bab00011c361c3700011c781c7d00011cd01ce800011c" +
	"ed1cf400071cf71cf900011d2c1d6a00011dc41dcf00011df51dff00011fbd1f" +
	"bf00021fc01fc100011fcd1fcf00011fdd1fdf00011fed1fef00011ffd1ffe00" +
	"012cef2cf100012e2f302a01fb302b302f00013099309c000130fca66f7573a6" +
	"7ca67d0001a67fa69c001da69da6f00053a6f1a700000fa701a7210001a788a7" +
	"8a0001a7f8a7f90001a8c4a8e0001ca8e1a8f10001a92ba92e0001a953a9b300" +
	"60a9c0a9e50025aa7baa7d0001aabfaac20001aaf6ab5b0065ab5cab5f0001ab" +
	"69ab6b0001abecabed0001fb1efe200302fe21fe2f0001ff3eff400002ff70ff" +
	"9e002eff9fffe300440035000102e000010780000004a0000107810001078500" +
	"00000100010787000107b000000001000107b2000107ba0000000100010ae500" +
	"010ae60000000100010d2200010d270000000100010efd00010eff0000000100" +
	"010f4600010f500000000100010f8200010f8500000001000110460001107000" +
	"00002a000110b9000110ba000000010001113300011134000000010001117300" +
	"0111c00000004d000111ca000111cc0000000100011235000112360000000100" +
	"0112e9000112ea000000010001133c0001134d00000011000113660001136c00" +
	"000001000113700001137400000001000114420001144600000004000114c200" +
	"0114c300000001000115bf000115c0000000010001163f000116b60000007700" +
	"0116b70001172b00000074000118390001183a000000010001193d0001193e00" +
	"00000100011943000119e00000009d00011a3400011a470000001300011a9900" +
	"011c3f000001a600011d4200011d440000000200011d4500011d970000005200" +
	"013447000134550000000100016af000016af40000000100016b3000016b3600" +
	"00000100016f8f00016f9f0000000100016ff000016ff1000000010001aff000" +
	"01aff3000000010001aff50001affb000000010001affd0001affe0000000100" +
	"01cf000001cf2d000000010001cf300001cf46000000010001d1670001d16900" +
	"0000010001d16d0001d172000000010001d17b0001d182000000010001d18500" +
	"01d18b000000010001d1aa0001d1ad000000010001e0300001e06d0000000100" +
	"01e1300001e136000000010001e2ae0001e2ec0000003e0001e2ed0001e2ef00" +
	"0000010001e8d00001e8d6000000010001e9440001e946000000010001e94800" +
	"01e94a0000000108457874656e646572000000c0000f000000b702d0021902d1" +
	"0640036f07fa0b55035b0e460ec60080180a184300391aa71c36018f1c7b3005" +
	"138a303130350001309d309e000130fc30fe0001a015a60c05f7a9cfa9e60017" +
	"aa70aadd006daaf3aaf40001ff70ff7000010008000107810001078200000001" +
	"0001135d000115c600000269000115c7000115c80000000100011a9800016b42" +
	"000050aa00016b4300016fe00000049d00016fe100016fe3000000020001e13c" +
	"0001e13d000000010001e9440001e94600000001094865785f44696769740000" +
	"002a00060003003000390001004100460001006100660001ff10ff190001ff21" +
	"ff260001ff41ff46000100000648797068656e0000002a00060001002d00ad00" +
	"80058a1806127c2010201100012e1730fb02e4fe63ff0d00aaff65ff65000100" +
	"00134944535f42696e6172795f4f70657261746f7200000012000200002ff02f" +
	"f100012ff42ffb00010000144944535f5472696e6172795f4f70657261746f72" +
	"0000000c000100002ff22ff3000100000b4964656f67726170686963000000cc" +
	"000700003006300700013021302900013038303a000134004dbf00014e009fff" +
	"0001f900fa6d0001fa70fad90001000d00016fe4000170000000001c00017001" +
	"000187f7000000010001880000018cd50000000100018d0000018d0800000001" +
	"0001b1700001b2fb00000001000200000002a6df000000010002a7000002b739" +
	"000000010002b7400002b81d000000010002b8200002cea1000000010002ceb0" +
	"0002ebe0000000010002f8000002fa1d00000001000300000003134a00000001" +
	"00031350000323af000000010c4a6f696e5f436f6e74726f6c0000000c000100" +
	"00200c200d00010000174c6f676963616c5f4f726465725f457863657074696f" +
	"6e0000002a000600000e400e4400010ec00ec4000119b519b7000119baaab590" +
	"fbaab6aab90003aabbaabc00010000174e6f6e6368617261637465725f436f64" +
	"655f506f696e74000000d200020000fdd0fdef0001fffeffff000100100001ff" +
	"fe0001ffff000000010002fffe0002ffff000000010003fffe0003ffff000000" +
	"010004fffe0004ffff000000010005fffe0005ffff000000010006fffe0006ff" +
	"ff000000010007fffe0007ffff000000010008fffe0008ffff000000010009ff" +
	"fe0009ffff00000001000afffe000affff00000001000bfffe000bffff000000" +
	"01000cfffe000cffff00000001000dfffe000dffff00000001000efffe000eff" +
	"ff00000001000ffffe000fffff000000010010fffe0010ffff00000001104f74" +
	"6865725f416c70686162657469630000075600940000034505b0026b05b105bd" +
	"000105bf05c1000205c205c4000205c505c700020610061a0001064b06570001" +
	"0659065f0001067006d6006606d706dc000106e106e4000106e706e8000106ed" +
	"071100240730073f000107a607b00001081608170001081b0823000108250827" +
	"00010829082c000108d408df000108e308e9000108f009030001093a093b0001" +
	"093e094c0001094e094f000109550957000109620963000109810983000109be" +
	"09c4000109c709c8000109cb09cc000109d709e2000b09e30a01001e0a020a03" +
	"00010a3e0a4200010a470a4800010a4b0a4c00010a510a70001f0a710a750004" +
	"0a810a8300010abe0ac500010ac70ac900010acb0acc00010ae20ae300010afa" +
	"0afc00010b010b0300010b3e0b4400010b470b4800010b4b0b4c00010b560b57" +
	"00010b620b6300010b820bbe003c0bbf0bc200010bc60bc800010bca0bcc0001" +
	"0bd70c0000290c010c0400010c3e0c4400010c460c4800010c4a0c4c00010c55" +
	"0c5600010c620c6300010c810c8300010cbe0cc400010cc60cc800010cca0ccc" +
	"00010cd50cd600010ce20ce300010cf30d00000d0d010d0300010d3e0d440001" +
	"0d460d4800010d4a0d4c00010d570d62000b0d630d81001e0d820d8300010dcf" +
	"0dd400010dd60dd800020dd90ddf00010df20df300010e310e3400030e350e3a" +
	"00010e4d0eb100640eb40eb900010ebb0ebc00010ecd0f7100a40f720f830001" +
	"0f8d0f9700010f990fbc0001102b103600011038103b0003103c103e00011056" +
	"10590001105e106000011062106400011067106d00011071107400011082108d" +
	"0001108f109a000b109b109d0001171217130001173217330001175217530001" +
	"17721773000117b617c8000118851886000118a9192000771921192b00011930" +
	"193800011a171a1b00011a551a5e00011a611a7400011abf1ac000011acc1ace" +
	"00011b001b0400011b351b4300011b801b8200011ba11ba900011bac1bad0001" +
	"1be71bf100011c241c3600011de71df4000124b624e900012de02dff0001a674" +
	"a67b0001a69ea69f0001a802a80b0009a823a8270001a880a8810001a8b4a8c3" +
	"0001a8c5a8ff003aa926a92a0001a947a9520001a980a9830001a9b4a9bf0001" +
	"a9e5aa290044aa2aaa360001aa43aa4c0009aa4daa7b002eaa7caa7d0001aab0" +
	"aab20002aab3aab40001aab7aab80001aabeaaeb002daaecaaef0001aaf5abe3" +
	"00eeabe4abea0001fb1efb1e00010052000103760001037a0000000100010a01" +
	"00010a030000000100010a0500010a060000000100010a0c00010a0f00000001" +
	"00010d2400010d270000000100010eab00010eac000000010001100000011002" +
	"0000000100011038000110450000000100011073000110740000000100011080" +
	"0001108200000001000110b0000110b800000001000110c2000111000000003e" +
	"0001110100011102000000010001112700011132000000010001114500011146" +
	"00000001000111800001118200000001000111b3000111bf00000001000111ce" +
	"000111cf000000010001122c0001123400000001000112370001123e00000007" +
	"00011241000112df0000009e000112e0000112e8000000010001130000011303" +
	"000000010001133e00011344000000010001134700011348000000010001134b" +
	"0001134c0000000100011357000113620000000b0001136300011435000000d2" +
	"000114360001144100000001000114430001144500000001000114b0000114c1" +
	"00000001000115af000115b500000001000115b8000115be00000001000115dc" +
	"000115dd00000001000116300001163e0000000100011640000116ab0000006b" +
	"000116ac000116b5000000010001171d0001172a000000010001182c00011838" +
	"000000010001193000011935000000010001193700011938000000010001193b" +
	"0001193c00000001000119400001194200000002000119d1000119d700000001" +
	"000119da000119df00000001000119e400011a010000001d00011a0200011a0a" +
	"0000000100011a3500011a390000000100011a3b00011a3e0000000100011a51" +
	"00011a5b0000000100011a8a00011a970000000100011c2f00011c3600000001" +
	"00011c3800011c3e0000000100011c9200011ca70000000100011ca900011cb6" +
	"0000000100011d3100011d360000000100011d3a00011d3c0000000200011d3d" +
	"00011d3f0000000200011d4000011d410000000100011d4300011d4700000004" +
	"00011d8a00011d8e0000000100011d9000011d910000000100011d9300011d96" +
	"0000000100011ef300011ef60000000100011f0000011f010000000100011f03" +
	"00011f340000003100011f3500011f3a0000000100011f3e00011f4000000001" +
	"00016f4f00016f510000000200016f5200016f870000000100016f8f00016f92" +
	"0000000100016ff000016ff1000000010001bc9e0001e000000023620001e001" +
	"0001e006000000010001e0080001e018000000010001e01b0001e02100000001" +
	"0001e0230001e024000000010001e0260001e02a000000010001e08f0001e947" +
	"000008b80001f1300001f149000000010001f1500001f169000000010001f170" +
	"0001f18900000001224f746865725f44656661756c745f49676e6f7261626c65" +
	"5f436f64655f506f696e740000005400050000034f115f0e10116017b4065417" +
	"b5206508b03164ffa0ce3cfff0fff800010004000e0000000e00020000000200" +
	"0e0003000e001f00000001000e0080000e00ff00000001000e01f0000e0fff00" +
	"000001154f746865725f4772617068656d655f457874656e640000008a000a00" +
	"0009be09d700190b3e0b5700190bbe0bd700190cc20cd500130cd60d3e00680d" +
	"570dcf00780ddf1b350d56200c302e1022302fff9ecf6fff9fff9f0001000600" +
	"01133e0001135700000019000114b0000114bd0000000d000115af0001193000" +
	"0003810001d1650001d16e000000090001d16f0001d17200000001000e002000" +
	"0e007f00000001114f746865725f49445f436f6e74696e756500000018000300" +
	"0000b7038702d013691371000119da19da000100000e4f746865725f49445f53" +
	"7461727400000018000300001885188600012118212e0016309b309c00010000" +
	"0f4f746865725f4c6f77657263617365000000ba0014000100aa00ba001002b0" +
	"02b8000102c002c1000102e002e400010345037a003510fc1d2c0c301d2d1d6a" +
	"00011d781d9b00231d9c1dbf00012071207f000e2090209c00012170217f0001" +
	"24d024e900012c7c2c7d0001a69ca69d0001a770a7f20082a7f3a7f40001a7f8" +
	"a7f90001ab5cab5f0001ab69ab69000100050001078000010783000000030001" +
	"0784000107850000000100010787000107b000000001000107b2000107ba0000" +
	"00010001e0300001e06d000000010a4f746865725f4d61746800000414003f00" +
	"00005e03d0037203d103d2000103d503f0001b03f103f4000303f520161c2120" +
	"3220340001204020610021206220640001207d207e0001208d208e000120d020" +
	"dc000120e120e5000420e620eb000520ec20ef0001210221070005210a211300" +
	"01211521190004211a211d00012124212800042129212c0003212d212f000221" +
	"3021310001213321380001213c213f0001214521490001219521990001219c21" +
	"9f000121a121a2000121a421a5000121a721a9000221aa21ad000121b021b100" +
	"0121b621b7000121bc21cd000121d021d1000121d321d5000221d621db000121" +
	"dd21e4000721e5230801232309230b000123b423b5000123b723d0001923e225" +
	"a001be25a125ae000d25af25b6000125bc25c0000125c625c7000125ca25cb00" +
	"0125cf25d3000125e225e4000225e725ec000126052606000126402642000226" +
	"6026630001266d266e000127c527c6000127e627ef000129832998000129d829" +
	"db000129fc29fd0001fe61fe630002fe68ff3c00d4ff3eff3e000100370001d4" +
	"000001d454000000010001d4560001d49c000000010001d49e0001d49f000000" +
	"010001d4a20001d4a5000000030001d4a60001d4a9000000030001d4aa0001d4" +
	"ac000000010001d4ae0001d4b9000000010001d4bb0001d4bd000000020001d4" +
	"be0001d4c3000000010001d4c50001d505000000010001d5070001d50a000000" +
	"010001d50d0001d514000000010001d5160001d51c000000010001d51e0001d5" +
	"39000000010001d53b0001d53e000000010001d5400001d544000000010001d5" +
	"460001d54a000000040001d54b0001d550000000010001d5520001d6a5000000" +
	"010001d6a80001d6c0000000010001d6c20001d6da000000010001d6dc0001d6" +
	"fa000000010001d6fc0001d714000000010001d7160001d734000000010001d7" +
	"360001d74e000000010001d7500001d76e000000010001d7700001d788000000" +
	"010001d78a0001d7a8000000010001d7aa0001d7c2000000010001d7c40001d7" +
	"cb000000010001d7ce0001d7ff000000010001ee000001ee03000000010001ee" +
	"050001ee1f000000010001ee210001ee22000000010001ee240001ee27000000" +
	"030001ee290001ee32000000010001ee340001ee37000000010001ee390001ee" +
	"3b000000020001ee420001ee47000000050001ee490001ee4d000000020001ee" +
	"4e0001ee4f000000010001ee510001ee52000000010001ee540001ee57000000" +
	"030001ee590001ee61000000020001ee620001ee64000000020001ee670001ee" +
	"6a000000010001ee6c0001ee72000000010001ee740001ee77000000010001ee" +
	"790001ee7c000000010001ee7e0001ee80000000020001ee810001ee89000000" +
	"010001ee8b0001ee9b000000010001eea10001eea3000000010001eea50001ee" +
	"a9000000010001eeab0001eebb000000010f4f746865725f5570706572636173" +
	"6500000036000200002160216f000124b624cf000100030001f1300001f14900" +
	"0000010001f1500001f169000000010001f1700001f189000000010e50617474" +
	"65726e5f53796e746178000000960018000a0021002f0001003a00400001005b" +
	"005e00010060007b001b007c007e000100a100a7000100a900ab000200ac00b0" +
	"000200b100bb000500bf00d7001800f720101f192011202700012030203e0001" +
	"2041205300012055205e00012190245f000125002775000127942bff00012e00" +
	"2e7f00013001300300013008302000013030fd3ecd0efd3ffe450106fe46fe46" +
	"00010000135061747465726e5f57686974655f53706163650000001e00040002" +
	"0009000d0001002000850065200e200f000120282029000100001c5072657065" +
	"6e6465645f436f6e636174656e6174696f6e5f4d61726b0000002a0004000006" +
	"000605000106dd070f003208900891000108e208e200010001000110bd000110" +
	"cd000000100e51756f746174696f6e5f4d61726b00000042000a000200220027" +
	"000500ab00bb00102018201f00012039203a00012e42300c01ca300d300f0001" +
	"301d301f0001fe41fe440001ff02ff070005ff62ff6300010000075261646963" +
	"616c00000018000300002e802e9900012e9b2ef300012f002fd5000100001252" +
	"6567696f6e616c5f496e64696361746f72000000120000000000010001f1e600" +
	"01f1ff0000000105535465726d0000022e002600010021002e000d003f058905" +
	"4a061d061f000106d40700002c07010702000107f90837003e0839083d000408" +
	"3e096401260965104a06e5104b13620317136713680001166e173500c7173618" +
	"0300cd18091944013b19451aa801631aa91aab00011b5a1b5b00011b5e1b5f00" +
	"011b7d1b7e00011c3b1c3c00011c7e1c7f0001203c203d00012047204900012e" +
	"2e2e3c000e2e532e5400013002a4ff74fda60ea60f0001a6f3a6f70004a876a8" +
	"770001a8cea8cf0001a92fa9c80099a9c9aa5d0094aa5eaa5f0001aaf0aaf100" +
	"01abebfe525267fe56fe570001ff01ff0e000dff1fff610042001b00010a5600" +
	"010a570000000100010f5500010f590000000100010f8600010f890000000100" +
	"0110470001104800000001000110be000110c100000001000111410001114300" +
	"000001000111c5000111c600000001000111cd000111de00000011000111df00" +
	"01123800000059000112390001123b000000020001123c000112a90000006d00" +
	"01144b0001144c00000001000115c2000115c300000001000115c9000115d700" +
	"0000010001164100011642000000010001173c0001173e000000010001194400" +
	"0119460000000200011a4200011a430000000100011a9b00011a9c0000000100" +
	"011c4100011c420000000100011ef700011ef80000000100011f4300011f4400" +
	"00000100016a6e00016a6f0000000100016af500016b370000004200016b3800" +
	"016b440000000c00016e980001bc9f00004e070001da880001da880000000111" +
	"53656e74656e63655f5465726d696e616c0000022e002600010021002e000d00" +
	"3f0589054a061d061f000106d40700002c07010702000107f90837003e083908" +
	"3d0004083e096401260965104a06e5104b13620317136713680001166e173500" +
	"c71736180300cd18091944013b19451aa801631aa91aab00011b5a1b5b00011b" +
	"5e1b5f00011b7d1b7e00011c3b1c3c00011c7e1c7f0001203c203d0001204720" +
	"4900012e2e2e3c000e2e532e5400013002a4ff74fda60ea60f0001a6f3a6f700" +
	"04a876a8770001a8cea8cf0001a92fa9c80099a9c9aa5d0094aa5eaa5f0001aa" +
	"f0aaf10001abebfe525267fe56fe570001ff01ff0e000dff1fff610042001b00" +
	"010a5600010a570000000100010f5500010f590000000100010f8600010f8900" +
	"000001000110470001104800000001000110be000110c1000000010001114100" +
	"01114300000001000111c5000111c600000001000111cd000111de0000001100" +
	"0111df0001123800000059000112390001123b000000020001123c000112a900" +
	"00006d0001144b0001144c00000001000115c2000115c300000001000115c900" +
	"0115d7000000010001164100011642000000010001173c0001173e0000000100" +
	"011944000119460000000200011a4200011a430000000100011a9b00011a9c00" +
	"00000100011c4100011c420000000100011ef700011ef80000000100011f4300" +
	"011f440000000100016a6e00016a6f0000000100016af500016b370000004200" +
	"016b3800016b440000000c00016e980001bc9f00004e070001da880001da8800" +
	"0000010b536f66745f446f74746564000000f6000a00010069006a0001012f02" +
	"49011a0268029d003502b203f301410456045800021d621d9600341da41da800" +
	"041e2d1ecb009e2071214800d721492c7c0b33000f0001d4220001d423000000" +
	"010001d4560001d457000000010001d48a0001d48b000000010001d4be0001d4" +
	"bf000000010001d4f20001d4f3000000010001d5260001d527000000010001d5" +
	"5a0001d55b000000010001d58e0001d58f000000010001d5c20001d5c3000000" +
	"010001d5f60001d5f7000000010001d62a0001d62b000000010001d65e0001d6" +
	"5f000000010001d6920001d693000000010001df1a0001e04c000001320001e0" +
	"4d0001e0680000001b145465726d696e616c5f50756e6374756174696f6e0000" +
	"030c003700030021002c000b002e003a000c003b003f0004037e038700090589" +
	"05c3003a060c061b000f061d061f000106d40700002c0701070a0001070c07f8" +
	"00ec07f9083000370831083e0001085e0964010609650e5a04f50e5b0f0800ad" +
	"0f0d0f120001104a104b0001136113680001166e16eb007d16ec16ed00011735" +
	"1736000117d417d6000117da1802002818031805000118081809000119441945" +
	"00011aa81aab00011b5a1b5b00011b5d1b5f00011b7d1b7e00011c3b1c3f0001" +
	"1c7e1c7f0001203c203d00012047204900012e2e2e3c000e2e412e4c000b2e4e" +
	"2e4f00012e532e540001300130020001a4fea4ff0001a60da60f0001a6f3a6f7" +
	"0001a876a8770001a8cea8cf0001a92fa9c70098a9c8a9c90001aa5daa5f0001" +
	"aadfaaf00011aaf1abeb00fafe50fe520001fe54fe570001ff01ff0c000bff0e" +
	"ff1a000cff1bff1f0004ff61ff64000300250001039f000103d0000000310001" +
	"08570001091f000000c800010a5600010a570000000100010af000010af50000" +
	"000100010b3a00010b3f0000000100010b9900010b9c0000000100010f550001" +
	"0f590000000100010f8600010f8900000001000110470001104d000000010001" +
	"10be000110c100000001000111410001114300000001000111c5000111c60000" +
	"0001000111cd000111de00000011000111df0001123800000059000112390001" +
	"123c00000001000112a90001144b000001a20001144c0001144d000000010001" +
	"145a0001145b00000001000115c2000115c500000001000115c9000115d70000" +
	"00010001164100011642000000010001173c0001173e00000001000119440001" +
	"19460000000200011a4200011a430000000100011a9b00011a9c000000010001" +
	"1aa100011aa20000000100011c4100011c430000000100011c7100011ef70000" +
	"028600011ef800011f430000004b00011f44000124700000052c000124710001" +
	"24740000000100016a6e00016a6f0000000100016af500016b37000000420001" +
	"6b3800016b390000000100016b4400016e970000035300016e980001bc9f0000" +
	"4e070001da870001da8a0000000111556e69666965645f4964656f6772617068" +
	"0000008a0008000034004dbf00014e009fff0001fa0efa0f0001fa11fa130002" +
	"fa14fa1f000bfa21fa230002fa24fa270003fa28fa2900010007000200000002" +
	"a6df000000010002a7000002b739000000010002b7400002b81d000000010002" +
	"b8200002cea1000000010002ceb00002ebe000000001000300000003134a0000" +
	"000100031350000323af0000000112566172696174696f6e5f53656c6563746f" +
	"720000002400030000180b180d0001180ffe00e5f1fe01fe0f00010001000e01" +
	"00000e01ef000000010b57686974655f53706163650000003000070002000900" +
	"0d000100200085006500a0168015e02000200a0001202820290001202f205f00" +
	"303000300000010000"

// FOLD_CATEGORY_TABLE_DATA stores the Unicode category fold tables.
const FOLD_CATEGORY_TABLE_DATA Encoded_Data = "" +
	"014c0000000c000100000345034500010000024c6c00000306006c0003004100" +
	"5a000100c000d6000100d800de00010100012e00020132013600020139014700" +
	"02014a017800020179017d000201810182000101840186000201870189000201" +
	"8a018b0001018e01910001019301940001019601980001019c019d0001019f01" +
	"a0000101a201a6000201a701a9000201ac01ae000201af01b1000201b201b300" +
	"0101b501b7000201b801bc000401c401c5000101c701c8000101ca01cb000101" +
	"cd01db000201de01ee000201f101f2000101f401f6000201f701f8000101fa02" +
	"320002023a023b0001023d023e00010241024300020244024600010248024e00" +
	"0203450370002b037203760004037f038600070388038a0001038c038e000203" +
	"8f03910002039203a1000103a303ab000103cf03d8000903da03ee000203f403" +
	"f7000303f903fa000103fd042f0001046004800002048a04c0000204c104cd00" +
	"0204d0052e000205310556000110a010c5000110c710cd000613a013f500011c" +
	"901cba00011cbd1cbf00011e001e9400021e9e1efe00021f081f0f00011f181f" +
	"1d00011f281f2f00011f381f3f00011f481f4d00011f591f5f00021f681f6f00" +
	"011f881f8f00011f981f9f00011fa81faf00011fb81fbc00011fc81fcc00011f" +
	"d81fdb00011fe81fec00011ff81ffc00012126212a0004212b2132000721832c" +
	"000a7d2c012c2f00012c602c6200022c632c6400012c672c6d00022c6e2c7000" +
	"012c722c7500032c7e2c8000012c822ce200022ceb2ced00022cf2a640794ea6" +
	"42a66c0002a680a69a0002a722a72e0002a732a76e0002a779a77d0002a77ea7" +
	"860002a78ba78d0002a790a7920002a796a7aa0002a7aba7ae0001a7b0a7b400" +
	"01a7b6a7c40002a7c5a7c70001a7c9a7d00007a7d6a7d80002a7f5ff21572cff" +
	"22ff3a0001000a000104000001042700000001000104b0000104d30000000100" +
	"0105700001057a000000010001057c0001058a000000010001058c0001059200" +
	"00000100010594000105950000000100010c8000010cb200000001000118a000" +
	"0118bf0000000100016e4000016e5f000000010001e9000001e9210000000102" +
	"4c740000003c0009000001c401c6000201c701c9000201ca01cc000201f101f3" +
	"00021f801f8700011f901f9700011fa01fa700011fb31fc300101ff31ff30001" +
	"0000024c750000030c006d00040061007a000100b500df002a00e000f6000100" +
	"f800ff00010101012f0002013301370002013a01480002014b01770002017a01" +
	"7e0002017f018000010183018500020188018c00040192019500030199019a00" +
	"01019e01a1000301a301a5000201a801ad000501b001b4000401b601b9000301" +
	"bd01bf000201c501c6000101c801c9000101cb01cc000101ce01dc000201dd01" +
	"ef000201f201f3000101f501f9000401fb021f0002022302330002023c023f00" +
	"030240024200020247024f00020250025400010256025700010259025b000202" +
	"5c026000040261026500020266026800020269026c0001026f02710002027202" +
	"750003027d028000030282028300010287028c00010292029d000b029e034500" +
	"a70371037300020377037b0004037c037d000103ac03af000103b103ce000103" +
	"d003d1000103d503d7000103d903ef000203f003f3000103f503fb0003043004" +
	"5f0001046104810002048b04bf000204c204ce000204cf052f00020561058600" +
	"0110d010fa000110fd10ff000113f813fd00011c801c8800011d791d7d00041d" +
	"8e1e0100731e031e9500021e9b1ea100061ea31eff00021f001f0700011f101f" +
	"1500011f201f2700011f301f3700011f401f4500011f511f5700021f601f6700" +
	"011f701f7d00011fb01fb100011fbe1fd000121fd11fe0000f1fe11fe5000421" +
	"4e218400362c302c5f00012c612c6500042c662c6c00022c732c7600032c812c" +
	"e300022cec2cee00022cf32d00000d2d012d2500012d272d2d0006a641a66d00" +
	"02a681a69b0002a723a72f0002a733a76f0002a77aa77c0002a77fa7870002a7" +
	"8ca7910005a793a7940001a797a7a90002a7b5a7c30002a7c8a7ca0002a7d1a7" +
	"d70006a7d9a7f6001dab53ab70001dab71abbf0001ff41ff5a0001000a000104" +
	"280001044f00000001000104d8000104fb0000000100010597000105a1000000" +
	"01000105a3000105b100000001000105b3000105b900000001000105bb000105" +
	"bc0000000100010cc000010cf200000001000118c0000118df0000000100016e" +
	"6000016e7f000000010001e9220001e94300000001014d000000120002000003" +
	"9903b900201fbe1fbe00010000024d6e0000001200020000039903b900201fbe" +
	"1fbe00010000"

// FOLD_SCRIPT_TABLE_DATA stores the Unicode script fold tables.
const FOLD_SCRIPT_TABLE_DATA Encoded_Data = "" +
	"06436f6d6d6f6e0000000c00010000039c03bc0020000005477265656b000000" +
	"0c0001000000b503450290000009496e68657269746564000000120002000003" +
	"9903b900201fbe1fbe00010000"

// CATEGORY_ALIAS_DATA stores the category aliases and canonical names.
const CATEGORY_ALIAS_DATA Encoded_Data = "" +
	"0c43617365645f4c6574746572024c4311436c6f73655f50756e637475617469" +
	"6f6e0250650e436f6d62696e696e675f4d61726b014d15436f6e6e6563746f72" +
	"5f50756e6374756174696f6e02506307436f6e74726f6c0243630f4375727265" +
	"6e63795f53796d626f6c02536310446173685f50756e6374756174696f6e0250" +
	"640e446563696d616c5f4e756d626572024e640e456e636c6f73696e675f4d61" +
	"726b024d651146696e616c5f50756e6374756174696f6e02506606466f726d61" +
	"7402436613496e697469616c5f50756e6374756174696f6e025069064c657474" +
	"6572014c0d4c65747465725f4e756d626572024e6c0e4c696e655f5365706172" +
	"61746f72025a6c104c6f776572636173655f4c6574746572024c6c044d61726b" +
	"014d0b4d6174685f53796d626f6c02536d0f4d6f6469666965725f4c65747465" +
	"72024c6d0f4d6f6469666965725f53796d626f6c02536b0f4e6f6e7370616369" +
	"6e675f4d61726b024d6e064e756d626572014e104f70656e5f50756e63747561" +
	"74696f6e025073054f7468657201430c4f746865725f4c6574746572024c6f0c" +
	"4f746865725f4e756d626572024e6f114f746865725f50756e6374756174696f" +
	"6e02506f0c4f746865725f53796d626f6c02536f135061726167726170685f53" +
	"6570617261746f72025a700b507269766174655f55736502436f0b50756e6374" +
	"756174696f6e015009536570617261746f72015a0f53706163655f5365706172" +
	"61746f72025a730c53706163696e675f4d61726b024d6309537572726f676174" +
	"650243730653796d626f6c0153105469746c65636173655f4c6574746572024c" +
	"740a556e61737369676e656402436e105570706572636173655f4c6574746572" +
	"024c7505636e74726c024363056469676974024e640570756e63740150"

// CASE_RANGE_DATA stores the simple case mapping ranges.
const CASE_RANGE_DATA Encoded_Data = "" +
	"\x00\x00\x00\x41\x00\x00\x00\x5a\x00\x00\x00\x00\x00\x00\x00\x20\x00\x00\x00\x00" +
	"\x00\x00\x00\x61\x00\x00\x00\x7a\xff\xff\xff\xe0\x00\x00\x00\x00\xff\xff\xff\xe0" +
	"\x00\x00\x00\xb5\x00\x00\x00\xb5\x00\x00\x02\xe7\x00\x00\x00\x00\x00\x00\x02\xe7" +
	"\x00\x00\x00\xc0\x00\x00\x00\xd6\x00\x00\x00\x00\x00\x00\x00\x20\x00\x00\x00\x00" +
	"\x00\x00\x00\xd8\x00\x00\x00\xde\x00\x00\x00\x00\x00\x00\x00\x20\x00\x00\x00\x00" +
	"\x00\x00\x00\xe0\x00\x00\x00\xf6\xff\xff\xff\xe0\x00\x00\x00\x00\xff\xff\xff\xe0" +
	"\x00\x00\x00\xf8\x00\x00\x00\xfe\xff\xff\xff\xe0\x00\x00\x00\x00\xff\xff\xff\xe0" +
	"\x00\x00\x00\xff\x00\x00\x00\xff\x00\x00\x00\x79\x00\x00\x00\x00\x00\x00\x00\x79" +
	"\x00\x00\x01\x00\x00\x00\x01\x2f\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\x01\x30\x00\x00\x01\x30\x00\x00\x00\x00\xff\xff\xff\x39\x00\x00\x00\x00" +
	"\x00\x00\x01\x31\x00\x00\x01\x31\xff\xff\xff\x18\x00\x00\x00\x00\xff\xff\xff\x18" +
	"\x00\x00\x01\x32\x00\x00\x01\x37\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\x01\x39\x00\x00\x01\x48\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\x01\x4a\x00\x00\x01\x77\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\x01\x78\x00\x00\x01\x78\x00\x00\x00\x00\xff\xff\xff\x87\x00\x00\x00\x00" +
	"\x00\x00\x01\x79\x00\x00\x01\x7e\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\x01\x7f\x00\x00\x01\x7f\xff\xff\xfe\xd4\x00\x00\x00\x00\xff\xff\xfe\xd4" +
	"\x00\x00\x01\x80\x00\x00\x01\x80\x00\x00\x00\xc3\x00\x00\x00\x00\x00\x00\x00\xc3" +
	"\x00\x00\x01\x81\x00\x00\x01\x81\x00\x00\x00\x00\x00\x00\x00\xd2\x00\x00\x00\x00" +
	"\x00\x00\x01\x82\x00\x00\x01\x85\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\x01\x86\x00\x00\x01\x86\x00\x00\x00\x00\x00\x00\x00\xce\x00\x00\x00\x00" +
	"\x00\x00\x01\x87\x00\x00\x01\x88\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\x01\x89\x00\x00\x01\x8a\x00\x00\x00\x00\x00\x00\x00\xcd\x00\x00\x00\x00" +
	"\x00\x00\x01\x8b\x00\x00\x01\x8c\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\x01\x8e\x00\x00\x01\x8e\x00\x00\x00\x00\x00\x00\x00\x4f\x00\x00\x00\x00" +
	"\x00\x00\x01\x8f\x00\x00\x01\x8f\x00\x00\x00\x00\x00\x00\x00\xca\x00\x00\x00\x00" +
	"\x00\x00\x01\x90\x00\x00\x01\x90\x00\x00\x00\x00\x00\x00\x00\xcb\x00\x00\x00\x00" +
	"\x00\x00\x01\x91\x00\x00\x01\x92\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\x01\x93\x00\x00\x01\x93\x00\x00\x00\x00\x00\x00\x00\xcd\x00\x00\x00\x00" +
	"\x00\x00\x01\x94\x00\x00\x01\x94\x00\x00\x00\x00\x00\x00\x00\xcf\x00\x00\x00\x00" +
	"\x00\x00\x01\x95\x00\x00\x01\x95\x00\x00\x00\x61\x00\x00\x00\x00\x00\x00\x00\x61" +
	"\x00\x00\x01\x96\x00\x00\x01\x96\x00\x00\x00\x00\x00\x00\x00\xd3\x00\x00\x00\x00" +
	"\x00\x00\x01\x97\x00\x00\x01\x97\x00\x00\x00\x00\x00\x00\x00\xd1\x00\x00\x00\x00" +
	"\x00\x00\x01\x98\x00\x00\x01\x99\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\x01\x9a\x00\x00\x01\x9a\x00\x00\x00\xa3\x00\x00\x00\x00\x00\x00\x00\xa3" +
	"\x00\x00\x01\x9c\x00\x00\x01\x9c\x00\x00\x00\x00\x00\x00\x00\xd3\x00\x00\x00\x00" +
	"\x00\x00\x01\x9d\x00\x00\x01\x9d\x00\x00\x00\x00\x00\x00\x00\xd5\x00\x00\x00\x00" +
	"\x00\x00\x01\x9e\x00\x00\x01\x9e\x00\x00\x00\x82\x00\x00\x00\x00\x00\x00\x00\x82" +
	"\x00\x00\x01\x9f\x00\x00\x01\x9f\x00\x00\x00\x00\x00\x00\x00\xd6\x00\x00\x00\x00" +
	"\x00\x00\x01\xa0\x00\x00\x01\xa5\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\x01\xa6\x00\x00\x01\xa6\x00\x00\x00\x00\x00\x00\x00\xda\x00\x00\x00\x00" +
	"\x00\x00\x01\xa7\x00\x00\x01\xa8\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\x01\xa9\x00\x00\x01\xa9\x00\x00\x00\x00\x00\x00\x00\xda\x00\x00\x00\x00" +
	"\x00\x00\x01\xac\x00\x00\x01\xad\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\x01\xae\x00\x00\x01\xae\x00\x00\x00\x00\x00\x00\x00\xda\x00\x00\x00\x00" +
	"\x00\x00\x01\xaf\x00\x00\x01\xb0\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\x01\xb1\x00\x00\x01\xb2\x00\x00\x00\x00\x00\x00\x00\xd9\x00\x00\x00\x00" +
	"\x00\x00\x01\xb3\x00\x00\x01\xb6\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\x01\xb7\x00\x00\x01\xb7\x00\x00\x00\x00\x00\x00\x00\xdb\x00\x00\x00\x00" +
	"\x00\x00\x01\xb8\x00\x00\x01\xb9\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\x01\xbc\x00\x00\x01\xbd\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\x01\xbf\x00\x00\x01\xbf\x00\x00\x00\x38\x00\x00\x00\x00\x00\x00\x00\x38" +
	"\x00\x00\x01\xc4\x00\x00\x01\xc4\x00\x00\x00\x00\x00\x00\x00\x02\x00\x00\x00\x01" +
	"\x00\x00\x01\xc5\x00\x00\x01\xc5\xff\xff\xff\xff\x00\x00\x00\x01\x00\x00\x00\x00" +
	"\x00\x00\x01\xc6\x00\x00\x01\xc6\xff\xff\xff\xfe\x00\x00\x00\x00\xff\xff\xff\xff" +
	"\x00\x00\x01\xc7\x00\x00\x01\xc7\x00\x00\x00\x00\x00\x00\x00\x02\x00\x00\x00\x01" +
	"\x00\x00\x01\xc8\x00\x00\x01\xc8\xff\xff\xff\xff\x00\x00\x00\x01\x00\x00\x00\x00" +
	"\x00\x00\x01\xc9\x00\x00\x01\xc9\xff\xff\xff\xfe\x00\x00\x00\x00\xff\xff\xff\xff" +
	"\x00\x00\x01\xca\x00\x00\x01\xca\x00\x00\x00\x00\x00\x00\x00\x02\x00\x00\x00\x01" +
	"\x00\x00\x01\xcb\x00\x00\x01\xcb\xff\xff\xff\xff\x00\x00\x00\x01\x00\x00\x00\x00" +
	"\x00\x00\x01\xcc\x00\x00\x01\xcc\xff\xff\xff\xfe\x00\x00\x00\x00\xff\xff\xff\xff" +
	"\x00\x00\x01\xcd\x00\x00\x01\xdc\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\x01\xdd\x00\x00\x01\xdd\xff\xff\xff\xb1\x00\x00\x00\x00\xff\xff\xff\xb1" +
	"\x00\x00\x01\xde\x00\x00\x01\xef\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\x01\xf1\x00\x00\x01\xf1\x00\x00\x00\x00\x00\x00\x00\x02\x00\x00\x00\x01" +
	"\x00\x00\x01\xf2\x00\x00\x01\xf2\xff\xff\xff\xff\x00\x00\x00\x01\x00\x00\x00\x00" +
	"\x00\x00\x01\xf3\x00\x00\x01\xf3\xff\xff\xff\xfe\x00\x00\x00\x00\xff\xff\xff\xff" +
	"\x00\x00\x01\xf4\x00\x00\x01\xf5\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\x01\xf6\x00\x00\x01\xf6\x00\x00\x00\x00\xff\xff\xff\x9f\x00\x00\x00\x00" +
	"\x00\x00\x01\xf7\x00\x00\x01\xf7\x00\x00\x00\x00\xff\xff\xff\xc8\x00\x00\x00\x00" +
	"\x00\x00\x01\xf8\x00\x00\x02\x1f\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\x02\x20\x00\x00\x02\x20\x00\x00\x00\x00\xff\xff\xff\x7e\x00\x00\x00\x00" +
	"\x00\x00\x02\x22\x00\x00\x02\x33\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\x02\x3a\x00\x00\x02\x3a\x00\x00\x00\x00\x00\x00\x2a\x2b\x00\x00\x00\x00" +
	"\x00\x00\x02\x3b\x00\x00\x02\x3c\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\x02\x3d\x00\x00\x02\x3d\x00\x00\x00\x00\xff\xff\xff\x5d\x00\x00\x00\x00" +
	"\x00\x00\x02\x3e\x00\x00\x02\x3e\x00\x00\x00\x00\x00\x00\x2a\x28\x00\x00\x00\x00" +
	"\x00\x00\x02\x3f\x00\x00\x02\x40\x00\x00\x2a\x3f\x00\x00\x00\x00\x00\x00\x2a\x3f" +
	"\x00\x00\x02\x41\x00\x00\x02\x42\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\x02\x43\x00\x00\x02\x43\x00\x00\x00\x00\xff\xff\xff\x3d\x00\x00\x00\x00" +
	"\x00\x00\x02\x44\x00\x00\x02\x44\x00\x00\x00\x00\x00\x00\x00\x45\x00\x00\x00\x00" +
	"\x00\x00\x02\x45\x00\x00\x02\x45\x00\x00\x00\x00\x00\x00\x00\x47\x00\x00\x00\x00" +
	"\x00\x00\x02\x46\x00\x00\x02\x4f\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\x02\x50\x00\x00\x02\x50\x00\x00\x2a\x1f\x00\x00\x00\x00\x00\x00\x2a\x1f" +
	"\x00\x00\x02\x51\x00\x00\x02\x51\x00\x00\x2a\x1c\x00\x00\x00\x00\x00\x00\x2a\x1c" +
	"\x00\x00\x02\x52\x00\x00\x02\x52\x00\x00\x2a\x1e\x00\x00\x00\x00\x00\x00\x2a\x1e" +
	"\x00\x00\x02\x53\x00\x00\x02\x53\xff\xff\xff\x2e\x00\x00\x00\x00\xff\xff\xff\x2e" +
	"\x00\x00\x02\x54\x00\x00\x02\x54\xff\xff\xff\x32\x00\x00\x00\x00\xff\xff\xff\x32" +
	"\x00\x00\x02\x56\x00\x00\x02\x57\xff\xff\xff\x33\x00\x00\x00\x00\xff\xff\xff\x33" +
	"\x00\x00\x02\x59\x00\x00\x02\x59\xff\xff\xff\x36\x00\x00\x00\x00\xff\xff\xff\x36" +
	"\x00\x00\x02\x5b\x00\x00\x02\x5b\xff\xff\xff\x35\x00\x00\x00\x00\xff\xff\xff\x35" +
	"\x00\x00\x02\x5c\x00\x00\x02\x5c\x00\x00\xa5\x4f\x00\x00\x00\x00\x00\x00\xa5\x4f" +
	"\x00\x00\x02\x60\x00\x00\x02\x60\xff\xff\xff\x33\x00\x00\x00\x00\xff\xff\xff\x33" +
	"\x00\x00\x02\x61\x00\x00\x02\x61\x00\x00\xa5\x4b\x00\x00\x00\x00\x00\x00\xa5\x4b" +
	"\x00\x00\x02\x63\x00\x00\x02\x63\xff\xff\xff\x31\x00\x00\x00\x00\xff\xff\xff\x31" +
	"\x00\x00\x02\x65\x00\x00\x02\x65\x00\x00\xa5\x28\x00\x00\x00\x00\x00\x00\xa5\x28" +
	"\x00\x00\x02\x66\x00\x00\x02\x66\x00\x00\xa5\x44\x00\x00\x00\x00\x00\x00\xa5\x44" +
	"\x00\x00\x02\x68\x00\x00\x02\x68\xff\xff\xff\x2f\x00\x00\x00\x00\xff\xff\xff\x2f" +
	"\x00\x00\x02\x69\x00\x00\x02\x69\xff\xff\xff\x2d\x00\x00\x00\x00\xff\xff\xff\x2d" +
	"\x00\x00\x02\x6a\x00\x00\x02\x6a\x00\x00\xa5\x44\x00\x00\x00\x00\x00\x00\xa5\x44" +
	"\x00\x00\x02\x6b\x00\x00\x02\x6b\x00\x00\x29\xf7\x00\x00\x00\x00\x00\x00\x29\xf7" +
	"\x00\x00\x02\x6c\x00\x00\x02\x6c\x00\x00\xa5\x41\x00\x00\x00\x00\x00\x00\xa5\x41" +
	"\x00\x00\x02\x6f\x00\x00\x02\x6f\xff\xff\xff\x2d\x00\x00\x00\x00\xff\xff\xff\x2d" +
	"\x00\x00\x02\x71\x00\x00\x02\x71\x00\x00\x29\xfd\x00\x00\x00\x00\x00\x00\x29\xfd" +
	"\x00\x00\x02\x72\x00\x00\x02\x72\xff\xff\xff\x2b\x00\x00\x00\x00\xff\xff\xff\x2b" +
	"\x00\x00\x02\x75\x00\x00\x02\x75\xff\xff\xff\x2a\x00\x00\x00\x00\xff\xff\xff\x2a" +
	"\x00\x00\x02\x7d\x00\x00\x02\x7d\x00\x00\x29\xe7\x00\x00\x00\x00\x00\x00\x29\xe7" +
	"\x00\x00\x02\x80\x00\x00\x02\x80\xff\xff\xff\x26\x00\x00\x00\x00\xff\xff\xff\x26" +
	"\x00\x00\x02\x82\x00\x00\x02\x82\x00\x00\xa5\x43\x00\x00\x00\x00\x00\x00\xa5\x43" +
	"\x00\x00\x02\x83\x00\x00\x02\x83\xff\xff\xff\x26\x00\x00\x00\x00\xff\xff\xff\x26" +
	"\x00\x00\x02\x87\x00\x00\x02\x87\x00\x00\xa5\x2a\x00\x00\x00\x00\x00\x00\xa5\x2a" +
	"\x00\x00\x02\x88\x00\x00\x02\x88\xff\xff\xff\x26\x00\x00\x00\x00\xff\xff\xff\x26" +
	"\x00\x00\x02\x89\x00\x00\x02\x89\xff\xff\xff\xbb\x00\x00\x00\x00\xff\xff\xff\xbb" +
	"\x00\x00\x02\x8a\x00\x00\x02\x8b\xff\xff\xff\x27\x00\x00\x00\x00\xff\xff\xff\x27" +
	"\x00\x00\x02\x8c\x00\x00\x02\x8c\xff\xff\xff\xb9\x00\x00\x00\x00\xff\xff\xff\xb9" +
	"\x00\x00\x02\x92\x00\x00\x02\x92\xff\xff\xff\x25\x00\x00\x00\x00\xff\xff\xff\x25" +
	"\x00\x00\x02\x9d\x00\x00\x02\x9d\x00\x00\xa5\x15\x00\x00\x00\x00\x00\x00\xa5\x15" +
	"\x00\x00\x02\x9e\x00\x00\x02\x9e\x00\x00\xa5\x12\x00\x00\x00\x00\x00\x00\xa5\x12" +
	"\x00\x00\x03\x45\x00\x00\x03\x45\x00\x00\x00\x54\x00\x00\x00\x00\x00\x00\x00\x54" +
	"\x00\x00\x03\x70\x00\x00\x03\x73\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\x03\x76\x00\x00\x03\x77\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\x03\x7b\x00\x00\x03\x7d\x00\x00\x00\x82\x00\x00\x00\x00\x00\x00\x00\x82" +
	"\x00\x00\x03\x7f\x00\x00\x03\x7f\x00\x00\x00\x00\x00\x00\x00\x74\x00\x00\x00\x00" +
	"\x00\x00\x03\x86\x00\x00\x03\x86\x00\x00\x00\x00\x00\x00\x00\x26\x00\x00\x00\x00" +
	"\x00\x00\x03\x88\x00\x00\x03\x8a\x00\x00\x00\x00\x00\x00\x00\x25\x00\x00\x00\x00" +
	"\x00\x00\x03\x8c\x00\x00\x03\x8c\x00\x00\x00\x00\x00\x00\x00\x40\x00\x00\x00\x00" +
	"\x00\x00\x03\x8e\x00\x00\x03\x8f\x00\x00\x00\x00\x00\x00\x00\x3f\x00\x00\x00\x00" +
	"\x00\x00\x03\x91\x00\x00\x03\xa1\x00\x00\x00\x00\x00\x00\x00\x20\x00\x00\x00\x00" +
	"\x00\x00\x03\xa3\x00\x00\x03\xab\x00\x00\x00\x00\x00\x00\x00\x20\x00\x00\x00\x00" +
	"\x00\x00\x03\xac\x00\x00\x03\xac\xff\xff\xff\xda\x00\x00\x00\x00\xff\xff\xff\xda" +
	"\x00\x00\x03\xad\x00\x00\x03\xaf\xff\xff\xff\xdb\x00\x00\x00\x00\xff\xff\xff\xdb" +
	"\x00\x00\x03\xb1\x00\x00\x03\xc1\xff\xff\xff\xe0\x00\x00\x00\x00\xff\xff\xff\xe0" +
	"\x00\x00\x03\xc2\x00\x00\x03\xc2\xff\xff\xff\xe1\x00\x00\x00\x00\xff\xff\xff\xe1" +
	"\x00\x00\x03\xc3\x00\x00\x03\xcb\xff\xff\xff\xe0\x00\x00\x00\x00\xff\xff\xff\xe0" +
	"\x00\x00\x03\xcc\x00\x00\x03\xcc\xff\xff\xff\xc0\x00\x00\x00\x00\xff\xff\xff\xc0" +
	"\x00\x00\x03\xcd\x00\x00\x03\xce\xff\xff\xff\xc1\x00\x00\x00\x00\xff\xff\xff\xc1" +
	"\x00\x00\x03\xcf\x00\x00\x03\xcf\x00\x00\x00\x00\x00\x00\x00\x08\x00\x00\x00\x00" +
	"\x00\x00\x03\xd0\x00\x00\x03\xd0\xff\xff\xff\xc2\x00\x00\x00\x00\xff\xff\xff\xc2" +
	"\x00\x00\x03\xd1\x00\x00\x03\xd1\xff\xff\xff\xc7\x00\x00\x00\x00\xff\xff\xff\xc7" +
	"\x00\x00\x03\xd5\x00\x00\x03\xd5\xff\xff\xff\xd1\x00\x00\x00\x00\xff\xff\xff\xd1" +
	"\x00\x00\x03\xd6\x00\x00\x03\xd6\xff\xff\xff\xca\x00\x00\x00\x00\xff\xff\xff\xca" +
	"\x00\x00\x03\xd7\x00\x00\x03\xd7\xff\xff\xff\xf8\x00\x00\x00\x00\xff\xff\xff\xf8" +
	"\x00\x00\x03\xd8\x00\x00\x03\xef\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\x03\xf0\x00\x00\x03\xf0\xff\xff\xff\xaa\x00\x00\x00\x00\xff\xff\xff\xaa" +
	"\x00\x00\x03\xf1\x00\x00\x03\xf1\xff\xff\xff\xb0\x00\x00\x00\x00\xff\xff\xff\xb0" +
	"\x00\x00\x03\xf2\x00\x00\x03\xf2\x00\x00\x00\x07\x00\x00\x00\x00\x00\x00\x00\x07" +
	"\x00\x00\x03\xf3\x00\x00\x03\xf3\xff\xff\xff\x8c\x00\x00\x00\x00\xff\xff\xff\x8c" +
	"\x00\x00\x03\xf4\x00\x00\x03\xf4\x00\x00\x00\x00\xff\xff\xff\xc4\x00\x00\x00\x00" +
	"\x00\x00\x03\xf5\x00\x00\x03\xf5\xff\xff\xff\xa0\x00\x00\x00\x00\xff\xff\xff\xa0" +
	"\x00\x00\x03\xf7\x00\x00\x03\xf8\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\x03\xf9\x00\x00\x03\xf9\x00\x00\x00\x00\xff\xff\xff\xf9\x00\x00\x00\x00" +
	"\x00\x00\x03\xfa\x00\x00\x03\xfb\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\x03\xfd\x00\x00\x03\xff\x00\x00\x00\x00\xff\xff\xff\x7e\x00\x00\x00\x00" +
	"\x00\x00\x04\x00\x00\x00\x04\x0f\x00\x00\x00\x00\x00\x00\x00\x50\x00\x00\x00\x00" +
	"\x00\x00\x04\x10\x00\x00\x04\x2f\x00\x00\x00\x00\x00\x00\x00\x20\x00\x00\x00\x00" +
	"\x00\x00\x04\x30\x00\x00\x04\x4f\xff\xff\xff\xe0\x00\x00\x00\x00\xff\xff\xff\xe0" +
	"\x00\x00\x04\x50\x00\x00\x04\x5f\xff\xff\xff\xb0\x00\x00\x00\x00\xff\xff\xff\xb0" +
	"\x00\x00\x04\x60\x00\x00\x04\x81\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\x04\x8a\x00\x00\x04\xbf\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\x04\xc0\x00\x00\x04\xc0\x00\x00\x00\x00\x00\x00\x00\x0f\x00\x00\x00\x00" +
	"\x00\x00\x04\xc1\x00\x00\x04\xce\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\x04\xcf\x00\x00\x04\xcf\xff\xff\xff\xf1\x00\x00\x00\x00\xff\xff\xff\xf1" +
	"\x00\x00\x04\xd0\x00\x00\x05\x2f\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\x05\x31\x00\x00\x05\x56\x00\x00\x00\x00\x00\x00\x00\x30\x00\x00\x00\x00" +
	"\x00\x00\x05\x61\x00\x00\x05\x86\xff\xff\xff\xd0\x00\x00\x00\x00\xff\xff\xff\xd0" +
	"\x00\x00\x10\xa0\x00\x00\x10\xc5\x00\x00\x00\x00\x00\x00\x1c\x60\x00\x00\x00\x00" +
	"\x00\x00\x10\xc7\x00\x00\x10\xc7\x00\x00\x00\x00\x00\x00\x1c\x60\x00\x00\x00\x00" +
	"\x00\x00\x10\xcd\x00\x00\x10\xcd\x00\x00\x00\x00\x00\x00\x1c\x60\x00\x00\x00\x00" +
	"\x00\x00\x10\xd0\x00\x00\x10\xfa\x00\x00\x0b\xc0\x00\x00\x00\x00\x00\x00\x00\x00" +
	"\x00\x00\x10\xfd\x00\x00\x10\xff\x00\x00\x0b\xc0\x00\x00\x00\x00\x00\x00\x00\x00" +
	"\x00\x00\x13\xa0\x00\x00\x13\xef\x00\x00\x00\x00\x00\x00\x97\xd0\x00\x00\x00\x00" +
	"\x00\x00\x13\xf0\x00\x00\x13\xf5\x00\x00\x00\x00\x00\x00\x00\x08\x00\x00\x00\x00" +
	"\x00\x00\x13\xf8\x00\x00\x13\xfd\xff\xff\xff\xf8\x00\x00\x00\x00\xff\xff\xff\xf8" +
	"\x00\x00\x1c\x80\x00\x00\x1c\x80\xff\xff\xe7\x92\x00\x00\x00\x00\xff\xff\xe7\x92" +
	"\x00\x00\x1c\x81\x00\x00\x1c\x81\xff\xff\xe7\x93\x00\x00\x00\x00\xff\xff\xe7\x93" +
	"\x00\x00\x1c\x82\x00\x00\x1c\x82\xff\xff\xe7\x9c\x00\x00\x00\x00\xff\xff\xe7\x9c" +
	"\x00\x00\x1c\x83\x00\x00\x1c\x84\xff\xff\xe7\x9e\x00\x00\x00\x00\xff\xff\xe7\x9e" +
	"\x00\x00\x1c\x85\x00\x00\x1c\x85\xff\xff\xe7\x9d\x00\x00\x00\x00\xff\xff\xe7\x9d" +
	"\x00\x00\x1c\x86\x00\x00\x1c\x86\xff\xff\xe7\xa4\x00\x00\x00\x00\xff\xff\xe7\xa4" +
	"\x00\x00\x1c\x87\x00\x00\x1c\x87\xff\xff\xe7\xdb\x00\x00\x00\x00\xff\xff\xe7\xdb" +
	"\x00\x00\x1c\x88\x00\x00\x1c\x88\x00\x00\x89\xc2\x00\x00\x00\x00\x00\x00\x89\xc2" +
	"\x00\x00\x1c\x90\x00\x00\x1c\xba\x00\x00\x00\x00\xff\xff\xf4\x40\x00\x00\x00\x00" +
	"\x00\x00\x1c\xbd\x00\x00\x1c\xbf\x00\x00\x00\x00\xff\xff\xf4\x40\x00\x00\x00\x00" +
	"\x00\x00\x1d\x79\x00\x00\x1d\x79\x00\x00\x8a\x04\x00\x00\x00\x00\x00\x00\x8a\x04" +
	"\x00\x00\x1d\x7d\x00\x00\x1d\x7d\x00\x00\x0e\xe6\x00\x00\x00\x00\x00\x00\x0e\xe6" +
	"\x00\x00\x1d\x8e\x00\x00\x1d\x8e\x00\x00\x8a\x38\x00\x00\x00\x00\x00\x00\x8a\x38" +
	"\x00\x00\x1e\x00\x00\x00\x1e\x95\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\x1e\x9b\x00\x00\x1e\x9b\xff\xff\xff\xc5\x00\x00\x00\x00\xff\xff\xff\xc5" +
	"\x00\x00\x1e\x9e\x00\x00\x1e\x9e\x00\x00\x00\x00\xff\xff\xe2\x41\x00\x00\x00\x00" +
	"\x00\x00\x1e\xa0\x00\x00\x1e\xff\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\x1f\x00\x00\x00\x1f\x07\x00\x00\x00\x08\x00\x00\x00\x00\x00\x00\x00\x08" +
	"\x00\x00\x1f\x08\x00\x00\x1f\x0f\x00\x00\x00\x00\xff\xff\xff\xf8\x00\x00\x00\x00" +
	"\x00\x00\x1f\x10\x00\x00\x1f\x15\x00\x00\x00\x08\x00\x00\x00\x00\x00\x00\x00\x08" +
	"\x00\x00\x1f\x18\x00\x00\x1f\x1d\x00\x00\x00\x00\xff\xff\xff\xf8\x00\x00\x00\x00" +
	"\x00\x00\x1f\x20\x00\x00\x1f\x27\x00\x00\x00\x08\x00\x00\x00\x00\x00\x00\x00\x08" +
	"\x00\x00\x1f\x28\x00\x00\x1f\x2f\x00\x00\x00\x00\xff\xff\xff\xf8\x00\x00\x00\x00" +
	"\x00\x00\x1f\x30\x00\x00\x1f\x37\x00\x00\x00\x08\x00\x00\x00\x00\x00\x00\x00\x08" +
	"\x00\x00\x1f\x38\x00\x00\x1f\x3f\x00\x00\x00\x00\xff\xff\xff\xf8\x00\x00\x00\x00" +
	"\x00\x00\x1f\x40\x00\x00\x1f\x45\x00\x00\x00\x08\x00\x00\x00\x00\x00\x00\x00\x08" +
	"\x00\x00\x1f\x48\x00\x00\x1f\x4d\x00\x00\x00\x00\xff\xff\xff\xf8\x00\x00\x00\x00" +
	"\x00\x00\x1f\x51\x00\x00\x1f\x51\x00\x00\x00\x08\x00\x00\x00\x00\x00\x00\x00\x08" +
	"\x00\x00\x1f\x53\x00\x00\x1f\x53\x00\x00\x00\x08\x00\x00\x00\x00\x00\x00\x00\x08" +
	"\x00\x00\x1f\x55\x00\x00\x1f\x55\x00\x00\x00\x08\x00\x00\x00\x00\x00\x00\x00\x08" +
	"\x00\x00\x1f\x57\x00\x00\x1f\x57\x00\x00\x00\x08\x00\x00\x00\x00\x00\x00\x00\x08" +
	"\x00\x00\x1f\x59\x00\x00\x1f\x59\x00\x00\x00\x00\xff\xff\xff\xf8\x00\x00\x00\x00" +
	"\x00\x00\x1f\x5b\x00\x00\x1f\x5b\x00\x00\x00\x00\xff\xff\xff\xf8\x00\x00\x00\x00" +
	"\x00\x00\x1f\x5d\x00\x00\x1f\x5d\x00\x00\x00\x00\xff\xff\xff\xf8\x00\x00\x00\x00" +
	"\x00\x00\x1f\x5f\x00\x00\x1f\x5f\x00\x00\x00\x00\xff\xff\xff\xf8\x00\x00\x00\x00" +
	"\x00\x00\x1f\x60\x00\x00\x1f\x67\x00\x00\x00\x08\x00\x00\x00\x00\x00\x00\x00\x08" +
	"\x00\x00\x1f\x68\x00\x00\x1f\x6f\x00\x00\x00\x00\xff\xff\xff\xf8\x00\x00\x00\x00" +
	"\x00\x00\x1f\x70\x00\x00\x1f\x71\x00\x00\x00\x4a\x00\x00\x00\x00\x00\x00\x00\x4a" +
	"\x00\x00\x1f\x72\x00\x00\x1f\x75\x00\x00\x00\x56\x00\x00\x00\x00\x00\x00\x00\x56" +
	"\x00\x00\x1f\x76\x00\x00\x1f\x77\x00\x00\x00\x64\x00\x00\x00\x00\x00\x00\x00\x64" +
	"\x00\x00\x1f\x78\x00\x00\x1f\x79\x00\x00\x00\x80\x00\x00\x00\x00\x00\x00\x00\x80" +
	"\x00\x00\x1f\x7a\x00\x00\x1f\x7b\x00\x00\x00\x70\x00\x00\x00\x00\x00\x00\x00\x70" +
	"\x00\x00\x1f\x7c\x00\x00\x1f\x7d\x00\x00\x00\x7e\x00\x00\x00\x00\x00\x00\x00\x7e" +
	"\x00\x00\x1f\x80\x00\x00\x1f\x87\x00\x00\x00\x08\x00\x00\x00\x00\x00\x00\x00\x08" +
	"\x00\x00\x1f\x88\x00\x00\x1f\x8f\x00\x00\x00\x00\xff\xff\xff\xf8\x00\x00\x00\x00" +
	"\x00\x00\x1f\x90\x00\x00\x1f\x97\x00\x00\x00\x08\x00\x00\x00\x00\x00\x00\x00\x08" +
	"\x00\x00\x1f\x98\x00\x00\x1f\x9f\x00\x00\x00\x00\xff\xff\xff\xf8\x00\x00\x00\x00" +
	"\x00\x00\x1f\xa0\x00\x00\x1f\xa7\x00\x00\x00\x08\x00\x00\x00\x00\x00\x00\x00\x08" +
	"\x00\x00\x1f\xa8\x00\x00\x1f\xaf\x00\x00\x00\x00\xff\xff\xff\xf8\x00\x00\x00\x00" +
	"\x00\x00\x1f\xb0\x00\x00\x1f\xb1\x00\x00\x00\x08\x00\x00\x00\x00\x00\x00\x00\x08" +
	"\x00\x00\x1f\xb3\x00\x00\x1f\xb3\x00\x00\x00\x09\x00\x00\x00\x00\x00\x00\x00\x09" +
	"\x00\x00\x1f\xb8\x00\x00\x1f\xb9\x00\x00\x00\x00\xff\xff\xff\xf8\x00\x00\x00\x00" +
	"\x00\x00\x1f\xba\x00\x00\x1f\xbb\x00\x00\x00\x00\xff\xff\xff\xb6\x00\x00\x00\x00" +
	"\x00\x00\x1f\xbc\x00\x00\x1f\xbc\x00\x00\x00\x00\xff\xff\xff\xf7\x00\x00\x00\x00" +
	"\x00\x00\x1f\xbe\x00\x00\x1f\xbe\xff\xff\xe3\xdb\x00\x00\x00\x00\xff\xff\xe3\xdb" +
	"\x00\x00\x1f\xc3\x00\x00\x1f\xc3\x00\x00\x00\x09\x00\x00\x00\x00\x00\x00\x00\x09" +
	"\x00\x00\x1f\xc8\x00\x00\x1f\xcb\x00\x00\x00\x00\xff\xff\xff\xaa\x00\x00\x00\x00" +
	"\x00\x00\x1f\xcc\x00\x00\x1f\xcc\x00\x00\x00\x00\xff\xff\xff\xf7\x00\x00\x00\x00" +
	"\x00\x00\x1f\xd0\x00\x00\x1f\xd1\x00\x00\x00\x08\x00\x00\x00\x00\x00\x00\x00\x08" +
	"\x00\x00\x1f\xd8\x00\x00\x1f\xd9\x00\x00\x00\x00\xff\xff\xff\xf8\x00\x00\x00\x00" +
	"\x00\x00\x1f\xda\x00\x00\x1f\xdb\x00\x00\x00\x00\xff\xff\xff\x9c\x00\x00\x00\x00" +
	"\x00\x00\x1f\xe0\x00\x00\x1f\xe1\x00\x00\x00\x08\x00\x00\x00\x00\x00\x00\x00\x08" +
	"\x00\x00\x1f\xe5\x00\x00\x1f\xe5\x00\x00\x00\x07\x00\x00\x00\x00\x00\x00\x00\x07" +
	"\x00\x00\x1f\xe8\x00\x00\x1f\xe9\x00\x00\x00\x00\xff\xff\xff\xf8\x00\x00\x00\x00" +
	"\x00\x00\x1f\xea\x00\x00\x1f\xeb\x00\x00\x00\x00\xff\xff\xff\x90\x00\x00\x00\x00" +
	"\x00\x00\x1f\xec\x00\x00\x1f\xec\x00\x00\x00\x00\xff\xff\xff\xf9\x00\x00\x00\x00" +
	"\x00\x00\x1f\xf3\x00\x00\x1f\xf3\x00\x00\x00\x09\x00\x00\x00\x00\x00\x00\x00\x09" +
	"\x00\x00\x1f\xf8\x00\x00\x1f\xf9\x00\x00\x00\x00\xff\xff\xff\x80\x00\x00\x00\x00" +
	"\x00\x00\x1f\xfa\x00\x00\x1f\xfb\x00\x00\x00\x00\xff\xff\xff\x82\x00\x00\x00\x00" +
	"\x00\x00\x1f\xfc\x00\x00\x1f\xfc\x00\x00\x00\x00\xff\xff\xff\xf7\x00\x00\x00\x00" +
	"\x00\x00\x21\x26\x00\x00\x21\x26\x00\x00\x00\x00\xff\xff\xe2\xa3\x00\x00\x00\x00" +
	"\x00\x00\x21\x2a\x00\x00\x21\x2a\x00\x00\x00\x00\xff\xff\xdf\x41\x00\x00\x00\x00" +
	"\x00\x00\x21\x2b\x00\x00\x21\x2b\x00\x00\x00\x00\xff\xff\xdf\xba\x00\x00\x00\x00" +
	"\x00\x00\x21\x32\x00\x00\x21\x32\x00\x00\x00\x00\x00\x00\x00\x1c\x00\x00\x00\x00" +
	"\x00\x00\x21\x4e\x00\x00\x21\x4e\xff\xff\xff\xe4\x00\x00\x00\x00\xff\xff\xff\xe4" +
	"\x00\x00\x21\x60\x00\x00\x21\x6f\x00\x00\x00\x00\x00\x00\x00\x10\x00\x00\x00\x00" +
	"\x00\x00\x21\x70\x00\x00\x21\x7f\xff\xff\xff\xf0\x00\x00\x00\x00\xff\xff\xff\xf0" +
	"\x00\x00\x21\x83\x00\x00\x21\x84\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\x24\xb6\x00\x00\x24\xcf\x00\x00\x00\x00\x00\x00\x00\x1a\x00\x00\x00\x00" +
	"\x00\x00\x24\xd0\x00\x00\x24\xe9\xff\xff\xff\xe6\x00\x00\x00\x00\xff\xff\xff\xe6" +
	"\x00\x00\x2c\x00\x00\x00\x2c\x2f\x00\x00\x00\x00\x00\x00\x00\x30\x00\x00\x00\x00" +
	"\x00\x00\x2c\x30\x00\x00\x2c\x5f\xff\xff\xff\xd0\x00\x00\x00\x00\xff\xff\xff\xd0" +
	"\x00\x00\x2c\x60\x00\x00\x2c\x61\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\x2c\x62\x00\x00\x2c\x62\x00\x00\x00\x00\xff\xff\xd6\x09\x00\x00\x00\x00" +
	"\x00\x00\x2c\x63\x00\x00\x2c\x63\x00\x00\x00\x00\xff\xff\xf1\x1a\x00\x00\x00\x00" +
	"\x00\x00\x2c\x64\x00\x00\x2c\x64\x00\x00\x00\x00\xff\xff\xd6\x19\x00\x00\x00\x00" +
	"\x00\x00\x2c\x65\x00\x00\x2c\x65\xff\xff\xd5\xd5\x00\x00\x00\x00\xff\xff\xd5\xd5" +
	"\x00\x00\x2c\x66\x00\x00\x2c\x66\xff\xff\xd5\xd8\x00\x00\x00\x00\xff\xff\xd5\xd8" +
	"\x00\x00\x2c\x67\x00\x00\x2c\x6c\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\x2c\x6d\x00\x00\x2c\x6d\x00\x00\x00\x00\xff\xff\xd5\xe4\x00\x00\x00\x00" +
	"\x00\x00\x2c\x6e\x00\x00\x2c\x6e\x00\x00\x00\x00\xff\xff\xd6\x03\x00\x00\x00\x00" +
	"\x00\x00\x2c\x6f\x00\x00\x2c\x6f\x00\x00\x00\x00\xff\xff\xd5\xe1\x00\x00\x00\x00" +
	"\x00\x00\x2c\x70\x00\x00\x2c\x70\x00\x00\x00\x00\xff\xff\xd5\xe2\x00\x00\x00\x00" +
	"\x00\x00\x2c\x72\x00\x00\x2c\x73\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\x2c\x75\x00\x00\x2c\x76\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\x2c\x7e\x00\x00\x2c\x7f\x00\x00\x00\x00\xff\xff\xd5\xc1\x00\x00\x00\x00" +
	"\x00\x00\x2c\x80\x00\x00\x2c\xe3\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\x2c\xeb\x00\x00\x2c\xee\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\x2c\xf2\x00\x00\x2c\xf3\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\x2d\x00\x00\x00\x2d\x25\xff\xff\xe3\xa0\x00\x00\x00\x00\xff\xff\xe3\xa0" +
	"\x00\x00\x2d\x27\x00\x00\x2d\x27\xff\xff\xe3\xa0\x00\x00\x00\x00\xff\xff\xe3\xa0" +
	"\x00\x00\x2d\x2d\x00\x00\x2d\x2d\xff\xff\xe3\xa0\x00\x00\x00\x00\xff\xff\xe3\xa0" +
	"\x00\x00\xa6\x40\x00\x00\xa6\x6d\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\xa6\x80\x00\x00\xa6\x9b\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\xa7\x22\x00\x00\xa7\x2f\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\xa7\x32\x00\x00\xa7\x6f\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\xa7\x79\x00\x00\xa7\x7c\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\xa7\x7d\x00\x00\xa7\x7d\x00\x00\x00\x00\xff\xff\x75\xfc\x00\x00\x00\x00" +
	"\x00\x00\xa7\x7e\x00\x00\xa7\x87\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\xa7\x8b\x00\x00\xa7\x8c\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\xa7\x8d\x00\x00\xa7\x8d\x00\x00\x00\x00\xff\xff\x5a\xd8\x00\x00\x00\x00" +
	"\x00\x00\xa7\x90\x00\x00\xa7\x93\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\xa7\x94\x00\x00\xa7\x94\x00\x00\x00\x30\x00\x00\x00\x00\x00\x00\x00\x30" +
	"\x00\x00\xa7\x96\x00\x00\xa7\xa9\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\xa7\xaa\x00\x00\xa7\xaa\x00\x00\x00\x00\xff\xff\x5a\xbc\x00\x00\x00\x00" +
	"\x00\x00\xa7\xab\x00\x00\xa7\xab\x00\x00\x00\x00\xff\xff\x5a\xb1\x00\x00\x00\x00" +
	"\x00\x00\xa7\xac\x00\x00\xa7\xac\x00\x00\x00\x00\xff\xff\x5a\xb5\x00\x00\x00\x00" +
	"\x00\x00\xa7\xad\x00\x00\xa7\xad\x00\x00\x00\x00\xff\xff\x5a\xbf\x00\x00\x00\x00" +
	"\x00\x00\xa7\xae\x00\x00\xa7\xae\x00\x00\x00\x00\xff\xff\x5a\xbc\x00\x00\x00\x00" +
	"\x00\x00\xa7\xb0\x00\x00\xa7\xb0\x00\x00\x00\x00\xff\xff\x5a\xee\x00\x00\x00\x00" +
	"\x00\x00\xa7\xb1\x00\x00\xa7\xb1\x00\x00\x00\x00\xff\xff\x5a\xd6\x00\x00\x00\x00" +
	"\x00\x00\xa7\xb2\x00\x00\xa7\xb2\x00\x00\x00\x00\xff\xff\x5a\xeb\x00\x00\x00\x00" +
	"\x00\x00\xa7\xb3\x00\x00\xa7\xb3\x00\x00\x00\x00\x00\x00\x03\xa0\x00\x00\x00\x00" +
	"\x00\x00\xa7\xb4\x00\x00\xa7\xc3\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\xa7\xc4\x00\x00\xa7\xc4\x00\x00\x00\x00\xff\xff\xff\xd0\x00\x00\x00\x00" +
	"\x00\x00\xa7\xc5\x00\x00\xa7\xc5\x00\x00\x00\x00\xff\xff\x5a\xbd\x00\x00\x00\x00" +
	"\x00\x00\xa7\xc6\x00\x00\xa7\xc6\x00\x00\x00\x00\xff\xff\x75\xc8\x00\x00\x00\x00" +
	"\x00\x00\xa7\xc7\x00\x00\xa7\xca\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\xa7\xd0\x00\x00\xa7\xd1\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\xa7\xd6\x00\x00\xa7\xd9\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\xa7\xf5\x00\x00\xa7\xf6\x00\x11\x00\x00\x00\x11\x00\x00\x00\x11\x00\x00" +
	"\x00\x00\xab\x53\x00\x00\xab\x53\xff\xff\xfc\x60\x00\x00\x00\x00\xff\xff\xfc\x60" +
	"\x00\x00\xab\x70\x00\x00\xab\xbf\xff\xff\x68\x30\x00\x00\x00\x00\xff\xff\x68\x30" +
	"\x00\x00\xff\x21\x00\x00\xff\x3a\x00\x00\x00\x00\x00\x00\x00\x20\x00\x00\x00\x00" +
	"\x00\x00\xff\x41\x00\x00\xff\x5a\xff\xff\xff\xe0\x00\x00\x00\x00\xff\xff\xff\xe0" +
	"\x00\x01\x04\x00\x00\x01\x04\x27\x00\x00\x00\x00\x00\x00\x00\x28\x00\x00\x00\x00" +
	"\x00\x01\x04\x28\x00\x01\x04\x4f\xff\xff\xff\xd8\x00\x00\x00\x00\xff\xff\xff\xd8" +
	"\x00\x01\x04\xb0\x00\x01\x04\xd3\x00\x00\x00\x00\x00\x00\x00\x28\x00\x00\x00\x00" +
	"\x00\x01\x04\xd8\x00\x01\x04\xfb\xff\xff\xff\xd8\x00\x00\x00\x00\xff\xff\xff\xd8" +
	"\x00\x01\x05\x70\x00\x01\x05\x7a\x00\x00\x00\x00\x00\x00\x00\x27\x00\x00\x00\x00" +
	"\x00\x01\x05\x7c\x00\x01\x05\x8a\x00\x00\x00\x00\x00\x00\x00\x27\x00\x00\x00\x00" +
	"\x00\x01\x05\x8c\x00\x01\x05\x92\x00\x00\x00\x00\x00\x00\x00\x27\x00\x00\x00\x00" +
	"\x00\x01\x05\x94\x00\x01\x05\x95\x00\x00\x00\x00\x00\x00\x00\x27\x00\x00\x00\x00" +
	"\x00\x01\x05\x97\x00\x01\x05\xa1\xff\xff\xff\xd9\x00\x00\x00\x00\xff\xff\xff\xd9" +
	"\x00\x01\x05\xa3\x00\x01\x05\xb1\xff\xff\xff\xd9\x00\x00\x00\x00\xff\xff\xff\xd9" +
	"\x00\x01\x05\xb3\x00\x01\x05\xb9\xff\xff\xff\xd9\x00\x00\x00\x00\xff\xff\xff\xd9" +
	"\x00\x01\x05\xbb\x00\x01\x05\xbc\xff\xff\xff\xd9\x00\x00\x00\x00\xff\xff\xff\xd9" +
	"\x00\x01\x0c\x80\x00\x01\x0c\xb2\x00\x00\x00\x00\x00\x00\x00\x40\x00\x00\x00\x00" +
	"\x00\x01\x0c\xc0\x00\x01\x0c\xf2\xff\xff\xff\xc0\x00\x00\x00\x00\xff\xff\xff\xc0" +
	"\x00\x01\x18\xa0\x00\x01\x18\xbf\x00\x00\x00\x00\x00\x00\x00\x20\x00\x00\x00\x00" +
	"\x00\x01\x18\xc0\x00\x01\x18\xdf\xff\xff\xff\xe0\x00\x00\x00\x00\xff\xff\xff\xe0" +
	"\x00\x01\x6e\x40\x00\x01\x6e\x5f\x00\x00\x00\x00\x00\x00\x00\x20\x00\x00\x00\x00" +
	"\x00\x01\x6e\x60\x00\x01\x6e\x7f\xff\xff\xff\xe0\x00\x00\x00\x00\xff\xff\xff\xe0" +
	"\x00\x01\xe9\x00\x00\x01\xe9\x21\x00\x00\x00\x00\x00\x00\x00\x22\x00\x00\x00\x00" +
	"\x00\x01\xe9\x22\x00\x01\xe9\x43\xff\xff\xff\xde\x00\x00\x00\x00\xff\xff\xff\xde"

// CASE_ORBIT_DATA stores the exceptional simple-fold pairs.
const CASE_ORBIT_DATA = "" +
	"\x00\x4b\x00\x6b\x00\x53\x00\x73\x00\x6b\x21\x2a\x00\x73\x01\x7f\x00\xb5\x03\x9c" +
	"\x00\xc5\x00\xe5\x00\xdf\x1e\x9e\x00\xe5\x21\x2b\x01\x30\x01\x30\x01\x31\x01\x31" +
	"\x01\x7f\x00\x53\x01\xc4\x01\xc5\x01\xc5\x01\xc6\x01\xc6\x01\xc4\x01\xc7\x01\xc8" +
	"\x01\xc8\x01\xc9\x01\xc9\x01\xc7\x01\xca\x01\xcb\x01\xcb\x01\xcc\x01\xcc\x01\xca" +
	"\x01\xf1\x01\xf2\x01\xf2\x01\xf3\x01\xf3\x01\xf1\x03\x45\x03\x99\x03\x92\x03\xb2" +
	"\x03\x95\x03\xb5\x03\x98\x03\xb8\x03\x99\x03\xb9\x03\x9a\x03\xba\x03\x9c\x03\xbc" +
	"\x03\xa0\x03\xc0\x03\xa1\x03\xc1\x03\xa3\x03\xc2\x03\xa6\x03\xc6\x03\xa9\x03\xc9" +
	"\x03\xb2\x03\xd0\x03\xb5\x03\xf5\x03\xb8\x03\xd1\x03\xb9\x1f\xbe\x03\xba\x03\xf0" +
	"\x03\xbc\x00\xb5\x03\xc0\x03\xd6\x03\xc1\x03\xf1\x03\xc2\x03\xc3\x03\xc3\x03\xa3" +
	"\x03\xc6\x03\xd5\x03\xc9\x21\x26\x03\xd0\x03\x92\x03\xd1\x03\xf4\x03\xd5\x03\xa6" +
	"\x03\xd6\x03\xa0\x03\xf0\x03\x9a\x03\xf1\x03\xa1\x03\xf4\x03\x98\x03\xf5\x03\x95" +
	"\x04\x12\x04\x32\x04\x14\x04\x34\x04\x1e\x04\x3e\x04\x21\x04\x41\x04\x22\x04\x42" +
	"\x04\x2a\x04\x4a\x04\x32\x1c\x80\x04\x34\x1c\x81\x04\x3e\x1c\x82\x04\x41\x1c\x83" +
	"\x04\x42\x1c\x84\x04\x4a\x1c\x86\x04\x62\x04\x63\x04\x63\x1c\x87\x1c\x80\x04\x12" +
	"\x1c\x81\x04\x14\x1c\x82\x04\x1e\x1c\x83\x04\x21\x1c\x84\x1c\x85\x1c\x85\x04\x22" +
	"\x1c\x86\x04\x2a\x1c\x87\x04\x62\x1c\x88\xa6\x4a\x1e\x60\x1e\x61\x1e\x61\x1e\x9b" +
	"\x1e\x9b\x1e\x60\x1e\x9e\x00\xdf\x1f\xbe\x03\x45\x21\x26\x03\xa9\x21\x2a\x00\x4b" +
	"\x21\x2b\x00\xc5\xa6\x4a\xa6\x4b\xa6\x4b\x1c\x88"
