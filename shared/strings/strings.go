// Package strings supplies bounded string views and queries. Owned results stay absent because
// caller-independent storage cannot remain zero allocation.
package strings

import (
	"unsafe"

	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/aver/default"
	"local/james-orcales/shared/unicode/ucd"
	"local/james-orcales/shared/unicode/utf8"
)

// TEXT_SIZE_MINIMUM keeps empty text valid.
const TEXT_SIZE_MINIMUM = utf8.SEQUENCE_SIZE_MINIMUM

// TEXT_SIZE_MAXIMUM bounds malicious input before scan work.
const TEXT_SIZE_MAXIMUM = utf8.SEQUENCE_SIZE_MAXIMUM

// INDEX_ABSENT separates absence from first byte.
const INDEX_ABSENT = -1

// INDEX_MAXIMUM names final valid byte index.
const INDEX_MAXIMUM = TEXT_SIZE_MAXIMUM - 1

// BOUNDARY_INDEX_MAXIMUM includes boundary after final byte.
const BOUNDARY_INDEX_MAXIMUM = TEXT_SIZE_MAXIMUM

// COUNT_VALUE_MINIMUM permits no match.
const COUNT_VALUE_MINIMUM = 0

// COUNT_VALUE_MAXIMUM includes every empty UTF-8 boundary.
const COUNT_VALUE_MAXIMUM = TEXT_SIZE_MAXIMUM + 1

// ORDER_BEFORE normalizes every negative lexical result.
const ORDER_BEFORE = -1

// ORDER_EQUAL identifies equal text.
const ORDER_EQUAL = 0

// ORDER_AFTER normalizes every positive lexical result.
const ORDER_AFTER = 1

// CHARACTER_MINIMUM includes every rune storage value.
const CHARACTER_MINIMUM int32 = bits.INTEGER_32_MINIMUM

// CHARACTER_MAXIMUM includes every rune storage value.
const CHARACTER_MAXIMUM int32 = bits.INTEGER_32_MAXIMUM

// BYTE_MINIMUM names first byte value.
const BYTE_MINIMUM uint8 = bits.WORD_8_MINIMUM

// BYTE_MAXIMUM names final byte value.
const BYTE_MAXIMUM uint8 = bits.WORD_8_MAXIMUM

// Text gives each string boundary one bounded identity.
type Text string

// Text_Invariants rejects oversized text before scan work.
func Text_Invariants(value Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), TEXT_SIZE_MINIMUM, TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Index_Value keeps absence beside valid byte indices.
type Index_Value int

// Index_Value_Invariants rejects end boundaries from nonempty searches.
func Index_Value_Invariants(value Index_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), INDEX_ABSENT, INDEX_MAXIMUM).
		Ensure()
}

// Boundary_Index admits end boundary from final empty match.
type Boundary_Index int

// Boundary_Index_Invariants includes absence and final end boundary.
func Boundary_Index_Invariants(value Boundary_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), INDEX_ABSENT, BOUNDARY_INDEX_MAXIMUM).
		Ensure()
}

// Count_Value identifies nonoverlapping match count.
type Count_Value int

// Count_Value_Invariants includes every empty boundary in largest Text.
func Count_Value_Invariants(value Count_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), COUNT_VALUE_MINIMUM, COUNT_VALUE_MAXIMUM).
		Ensure()
}

// Order gives lexical results three stable values.
type Order int

// Order_Invariants hides implementation comparison magnitude.
func Order_Invariants(value Order, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Int(int(value), ORDER_BEFORE, ORDER_EQUAL, ORDER_AFTER).
		Ensure()
}

// Boolean gives query result its own coverage identity.
type Boolean bool

// Boolean_Invariants requires both query results.
func Boolean_Invariants(value Boolean, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "A Boolean report is true.").
		Ensure()
}

// Character keeps rune argument distinct from integer argument.
type Character rune

// Character_Invariants covers complete rune storage domain.
func Character_Invariants(value Character, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), CHARACTER_MINIMUM, CHARACTER_MAXIMUM).
		Ensure()
}

// Byte keeps byte argument distinct from character argument.
type Byte byte

// Byte_Invariants covers all byte values.
func Byte_Invariants(value Byte, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), BYTE_MINIMUM, BYTE_MAXIMUM).
		Ensure()
}

// Compare normalizes lexical order.
func Compare(left Text, right Text) (order Order) {
	defer func() { Order_Invariants(order, "compare.order") }()
	Text_Invariants(left, "compare.left")
	Text_Invariants(right, "compare.right")
	if left < right {
		return ORDER_BEFORE
	}
	if left > right {
		return ORDER_AFTER
	}
	return ORDER_EQUAL
}

// Contains reports substring presence.
func Contains(source Text, separator Text) (contained Boolean) {
	defer func() { Boolean_Invariants(contained, "contains.contained") }()
	Text_Invariants(source, "contains.source")
	Text_Invariants(separator, "contains.separator")
	return Boolean(Index(source, separator) >= 0)
}

// Contains_Any reports character-set presence.
func Contains_Any(source Text, characters Text) (contained Boolean) {
	defer func() { Boolean_Invariants(contained, "contains_any.contained") }()
	Text_Invariants(source, "contains_any.source")
	Text_Invariants(characters, "contains_any.characters")
	return Boolean(Index_Any(source, characters) >= 0)
}

// Contains_Rune reports character presence.
func Contains_Rune(source Text, character Character) (contained Boolean) {
	defer func() { Boolean_Invariants(contained, "contains_rune.contained") }()
	Text_Invariants(source, "contains_rune.source")
	Character_Invariants(character, "contains_rune.character")
	return Boolean(Index_Rune(source, character) >= 0)
}

// Contains_Function reports predicate match presence.
func Contains_Function(
	source Text, predicate func(rune) (matches bool),
) (contained Boolean) {
	defer func() { Boolean_Invariants(contained, "contains_function.contained") }()
	Text_Invariants(source, "contains_function.source")
	return Boolean(Index_Function(source, predicate) >= 0)
}

// Count reports nonoverlapping matches, including empty UTF-8 boundaries.
func Count(source Text, separator Text) (count Count_Value) {
	defer func() { Count_Value_Invariants(count, "count.count") }()
	Text_Invariants(source, "count.source")
	Text_Invariants(separator, "count.separator")
	if len(separator) == 0 {
		return Count_Value(utf8.Character_Count_Text(utf8.Text(source))) + 1
	}
	tail := source
	for len(tail) >= len(separator) {
		separator_index := Index(tail, separator)
		if separator_index == INDEX_ABSENT {
			break
		}
		count++
		tail = tail[int(separator_index)+len(separator):]
	}
	return count
}

// Index returns first substring byte index or INDEX_ABSENT.
func Index(source Text, separator Text) (index Index_Value) {
	defer func() { Index_Value_Invariants(index, "index.index") }()
	Text_Invariants(source, "index.source")
	Text_Invariants(separator, "index.separator")
	if len(separator) == 0 {
		return 0
	}
	if len(separator) > len(source) {
		return INDEX_ABSENT
	}
	final_index := len(source) - len(separator)
	for source_index := 0; source_index <= final_index; source_index++ {
		if source[source_index:source_index+len(separator)] == separator {
			return Index_Value(source_index)
		}
	}
	return INDEX_ABSENT
}

// Last_Index returns final substring byte index, including final empty boundary.
func Last_Index(source Text, separator Text) (index Boundary_Index) {
	defer func() { Boundary_Index_Invariants(index, "last_index.index") }()
	Text_Invariants(source, "last_index.source")
	Text_Invariants(separator, "last_index.separator")
	if len(separator) == 0 {
		return Boundary_Index(len(source))
	}
	if len(separator) > len(source) {
		return INDEX_ABSENT
	}
	for source_index := len(source) - len(separator); source_index >= 0; source_index-- {
		if source[source_index:source_index+len(separator)] == separator {
			return Boundary_Index(source_index)
		}
	}
	return INDEX_ABSENT
}

// Index_Byte returns first byte index or INDEX_ABSENT.
func Index_Byte(source Text, value Byte) (index Index_Value) {
	defer func() { Index_Value_Invariants(index, "index_byte.index") }()
	Text_Invariants(source, "index_byte.source")
	Byte_Invariants(value, "index_byte.value")
	for source_index := 0; source_index < len(source); source_index++ {
		if Byte(source[source_index]) == value {
			return Index_Value(source_index)
		}
	}
	return INDEX_ABSENT
}

// Index_Byte_Or_Non_ASCII finds selected ASCII or first non-ASCII byte.
func Index_Byte_Or_Non_ASCII(source Text, values Text) (index Index_Value) {
	defer func() {
		Index_Value_Invariants(index, "index_byte_or_non_ascii.index")
	}()
	Text_Invariants(source, "index_byte_or_non_ascii.source")
	Text_Invariants(values, "index_byte_or_non_ascii.values")
	for source_index := 0; source_index < len(source); source_index++ {
		value := source[source_index]
		if value >= byte(utf8.CHARACTER_SELF) {
			return Index_Value(source_index)
		}
		if Index_Byte(values, Byte(value)) >= 0 {
			return Index_Value(source_index)
		}
	}
	return INDEX_ABSENT
}

// Last_Index_Byte returns final byte index or INDEX_ABSENT.
func Last_Index_Byte(source Text, value Byte) (index Index_Value) {
	defer func() { Index_Value_Invariants(index, "last_index_byte.index") }()
	Text_Invariants(source, "last_index_byte.source")
	Byte_Invariants(value, "last_index_byte.value")
	for source_index := len(source) - 1; source_index >= 0; source_index-- {
		if Byte(source[source_index]) == value {
			return Index_Value(source_index)
		}
	}
	return INDEX_ABSENT
}

// Index_Rune returns first character byte index or INDEX_ABSENT.
func Index_Rune(source Text, character Character) (index Index_Value) {
	defer func() { Index_Value_Invariants(index, "index_rune.index") }()
	Text_Invariants(source, "index_rune.source")
	Character_Invariants(character, "index_rune.character")
	if !utf8.Valid_Character(utf8.Character(character)) {
		return INDEX_ABSENT
	}
	for source_index, source_character := range source {
		if Character(source_character) == character {
			return Index_Value(source_index)
		}
	}
	return INDEX_ABSENT
}

// Index_Any returns first character-set byte index or INDEX_ABSENT.
func Index_Any(source Text, characters Text) (index Index_Value) {
	defer func() { Index_Value_Invariants(index, "index_any.index") }()
	Text_Invariants(source, "index_any.source")
	Text_Invariants(characters, "index_any.characters")
	if len(characters) == 0 {
		return INDEX_ABSENT
	}
	for source_index, source_character := range source {
		if Index_Rune(characters, Character(source_character)) >= 0 {
			return Index_Value(source_index)
		}
	}
	return INDEX_ABSENT
}

// Last_Index_Any returns final character-set byte index or INDEX_ABSENT.
func Last_Index_Any(source Text, characters Text) (index Index_Value) {
	defer func() { Index_Value_Invariants(index, "last_index_any.index") }()
	Text_Invariants(source, "last_index_any.source")
	Text_Invariants(characters, "last_index_any.characters")
	if len(characters) == 0 {
		return INDEX_ABSENT
	}
	for boundary_count := len(source); boundary_count > 0; {
		character, size := utf8.Decode_Final_Character_Text(
			utf8.Text(source[:boundary_count]),
		)
		boundary_count -= int(size)
		if Index_Rune(characters, Character(character)) >= 0 {
			return Index_Value(boundary_count)
		}
	}
	return INDEX_ABSENT
}

// Index_Function returns first predicate byte index or INDEX_ABSENT.
func Index_Function(
	source Text, predicate func(rune) (matches bool),
) (index Index_Value) {
	defer func() { Index_Value_Invariants(index, "index_function.index") }()
	Text_Invariants(source, "index_function.source")
	for source_index, source_character := range source {
		if predicate(source_character) {
			return Index_Value(source_index)
		}
	}
	return INDEX_ABSENT
}

// Last_Index_Function returns final predicate byte index or INDEX_ABSENT.
func Last_Index_Function(
	source Text, predicate func(rune) (matches bool),
) (index Index_Value) {
	defer func() { Index_Value_Invariants(index, "last_index_function.index") }()
	Text_Invariants(source, "last_index_function.source")
	for boundary_count := len(source); boundary_count > 0; {
		character, size := utf8.Decode_Final_Character_Text(
			utf8.Text(source[:boundary_count]),
		)
		boundary_count -= int(size)
		if predicate(rune(character)) {
			return Index_Value(boundary_count)
		}
	}
	return INDEX_ABSENT
}

// Has_Prefix reports prefix presence.
func Has_Prefix(source Text, prefix Text) (present Boolean) {
	defer func() { Boolean_Invariants(present, "has_prefix.present") }()
	Text_Invariants(source, "has_prefix.source")
	Text_Invariants(prefix, "has_prefix.prefix")
	if len(prefix) > len(source) {
		return false
	}
	return Boolean(source[:len(prefix)] == prefix)
}

// Has_Suffix reports suffix presence.
func Has_Suffix(source Text, suffix Text) (present Boolean) {
	defer func() { Boolean_Invariants(present, "has_suffix.present") }()
	Text_Invariants(source, "has_suffix.source")
	Text_Invariants(suffix, "has_suffix.suffix")
	if len(suffix) > len(source) {
		return false
	}
	return Boolean(source[len(source)-len(suffix):] == suffix)
}

// Trim returns source view without surrounding cutset characters.
func Trim(source Text, cutset Text) (trimmed Text) {
	defer func() { Text_Invariants(trimmed, "trim.trimmed") }()
	Text_Invariants(source, "trim.source")
	Text_Invariants(cutset, "trim.cutset")
	left := Trim_Left(source, cutset)
	return Trim_Right(left, cutset)
}

// Trim_Left returns source view without leading cutset characters.
func Trim_Left(source Text, cutset Text) (trimmed Text) {
	defer func() { Text_Invariants(trimmed, "trim_left.trimmed") }()
	Text_Invariants(source, "trim_left.source")
	Text_Invariants(cutset, "trim_left.cutset")
	if len(cutset) == 0 {
		return source
	}
	for source_index, source_character := range source {
		if !Contains_Rune(cutset, Character(source_character)) {
			return source[source_index:]
		}
	}
	return ""
}

// Trim_Right returns source view without trailing cutset characters.
func Trim_Right(source Text, cutset Text) (trimmed Text) {
	defer func() { Text_Invariants(trimmed, "trim_right.trimmed") }()
	Text_Invariants(source, "trim_right.source")
	Text_Invariants(cutset, "trim_right.cutset")
	if len(cutset) == 0 {
		return source
	}
	boundary_count := len(source)
	for boundary_count > 0 {
		character, size := utf8.Decode_Final_Character_Text(
			utf8.Text(source[:boundary_count]),
		)
		if !Contains_Rune(cutset, Character(character)) {
			break
		}
		boundary_count -= int(size)
	}
	return source[:boundary_count]
}

// Trim_Function returns source view without surrounding predicate matches.
func Trim_Function(
	source Text, predicate func(rune) (matches bool),
) (trimmed Text) {
	defer func() { Text_Invariants(trimmed, "trim_function.trimmed") }()
	Text_Invariants(source, "trim_function.source")
	left := Trim_Left_Function(source, predicate)
	return Trim_Right_Function(left, predicate)
}

// Trim_Left_Function returns source view without leading predicate matches.
func Trim_Left_Function(
	source Text, predicate func(rune) (matches bool),
) (trimmed Text) {
	defer func() { Text_Invariants(trimmed, "trim_left_function.trimmed") }()
	Text_Invariants(source, "trim_left_function.source")
	for source_index, source_character := range source {
		if !predicate(source_character) {
			return source[source_index:]
		}
	}
	return ""
}

// Trim_Right_Function returns source view without trailing predicate matches.
func Trim_Right_Function(
	source Text, predicate func(rune) (matches bool),
) (trimmed Text) {
	defer func() { Text_Invariants(trimmed, "trim_right_function.trimmed") }()
	Text_Invariants(source, "trim_right_function.source")
	boundary_count := len(source)
	for boundary_count > 0 {
		character, size := utf8.Decode_Final_Character_Text(
			utf8.Text(source[:boundary_count]),
		)
		if !predicate(rune(character)) {
			break
		}
		boundary_count -= int(size)
	}
	return source[:boundary_count]
}

// Trim_Space returns source view without surrounding Unicode space.
func Trim_Space(source Text) (trimmed Text) {
	defer func() { Text_Invariants(trimmed, "trim_space.trimmed") }()
	Text_Invariants(source, "trim_space.source")
	left_index := 0
	for left_index < len(source) {
		character, size := utf8.Decode_Character_Text(utf8.Text(source[left_index:]))
		if !ucd.Is_Space(ucd.Character(character)) {
			break
		}
		left_index += int(size)
	}
	boundary_count := len(source)
	for boundary_count > left_index {
		character, size := utf8.Decode_Final_Character_Text(
			utf8.Text(source[:boundary_count]),
		)
		if !ucd.Is_Space(ucd.Character(character)) {
			break
		}
		boundary_count -= int(size)
	}
	return source[left_index:boundary_count]
}

// Trim_Prefix returns source view without present prefix.
func Trim_Prefix(source Text, prefix Text) (trimmed Text) {
	defer func() { Text_Invariants(trimmed, "trim_prefix.trimmed") }()
	Text_Invariants(source, "trim_prefix.source")
	Text_Invariants(prefix, "trim_prefix.prefix")
	if Has_Prefix(source, prefix) {
		return source[len(prefix):]
	}
	return source
}

// Trim_Suffix returns source view without present suffix.
func Trim_Suffix(source Text, suffix Text) (trimmed Text) {
	defer func() { Text_Invariants(trimmed, "trim_suffix.trimmed") }()
	Text_Invariants(source, "trim_suffix.source")
	Text_Invariants(suffix, "trim_suffix.suffix")
	if Has_Suffix(source, suffix) {
		return source[:len(source)-len(suffix)]
	}
	return source
}

// Equal_Fold reports Unicode simple-fold equality without normalized copy.
func Equal_Fold(left Text, right Text) (equal Boolean) {
	defer func() { Boolean_Invariants(equal, "equal_fold.equal") }()
	Text_Invariants(left, "equal_fold.left")
	Text_Invariants(right, "equal_fold.right")
	left_index := 0
	right_index := 0
	for left_index < len(left) {
		if right_index == len(right) {
			return false
		}
		left_character, left_size := utf8.Decode_Character_Text(
			utf8.Text(left[left_index:]),
		)
		right_character, right_size := utf8.Decode_Character_Text(
			utf8.Text(right[right_index:]),
		)
		// Invalid bytes decode to the same replacement character as valid U+FFFD. Byte
		// identity must remain observable or malformed input compares equal to valid text.
		if left_character == utf8.REPLACEMENT_CHARACTER {
			if right_character == utf8.REPLACEMENT_CHARACTER {
				if left_size != right_size {
					return false
				}
				left_byte_index := left_index
				right_byte_index := right_index
				for index := 0; index < int(left_size); index++ {
					if left[left_byte_index] != right[right_byte_index] {
						return false
					}
					left_byte_index++
					right_byte_index++
				}
			}
		}
		left_value := ucd.Character(left_character)
		right_value := ucd.Character(right_character)
		if left_value != right_value {
			folded := ucd.Simple_Fold(left_value)
			for folded != left_value && folded != right_value {
				folded = ucd.Simple_Fold(folded)
			}
			if folded != right_value {
				return false
			}
		}
		left_index += int(left_size)
		right_index += int(right_size)
	}
	return Boolean(right_index == len(right))
}

// Cut returns source views around first separator.
func Cut(source Text, separator Text) (before Text, after Text, found Boolean) {
	defer func() {
		Text_Invariants(before, "cut.before")
		Text_Invariants(after, "cut.after")
		Boolean_Invariants(found, "cut.found")
	}()
	Text_Invariants(source, "cut.source")
	Text_Invariants(separator, "cut.separator")
	separator_index := Index(source, separator)
	if separator_index == INDEX_ABSENT {
		return source, "", false
	}
	return source[:separator_index], source[int(separator_index)+len(separator):], true
}

// Cut_Prefix returns source view after present prefix.
func Cut_Prefix(source Text, prefix Text) (after Text, found Boolean) {
	defer func() {
		Text_Invariants(after, "cut_prefix.after")
		Boolean_Invariants(found, "cut_prefix.found")
	}()
	Text_Invariants(source, "cut_prefix.source")
	Text_Invariants(prefix, "cut_prefix.prefix")
	if Has_Prefix(source, prefix) {
		return source[len(prefix):], true
	}
	return source, false
}

// Cut_Suffix returns source view before present suffix.
func Cut_Suffix(source Text, suffix Text) (before Text, found Boolean) {
	defer func() {
		Text_Invariants(before, "cut_suffix.before")
		Boolean_Invariants(found, "cut_suffix.found")
	}()
	Text_Invariants(source, "cut_suffix.source")
	Text_Invariants(suffix, "cut_suffix.suffix")
	if Has_Suffix(source, suffix) {
		return source[:len(source)-len(suffix)], true
	}
	return source, false
}

// TEXT_COUNT_MAXIMUM includes every empty UTF-8 boundary.
const TEXT_COUNT_MAXIMUM = TEXT_SIZE_MAXIMUM + 1

// FIELD_COUNT_MAXIMUM alternates one-byte fields with one-byte separators.
const FIELD_COUNT_MAXIMUM = TEXT_SIZE_MAXIMUM / 2

// LINE_COUNT_MAXIMUM occurs when every source byte ends one line.
const LINE_COUNT_MAXIMUM = TEXT_SIZE_MAXIMUM

// LIMIT_MINIMUM requests every result.
const LIMIT_MINIMUM = -1

// LIMIT_MAXIMUM permits every possible result.
const LIMIT_MAXIMUM = TEXT_COUNT_MAXIMUM

// REPEAT_COUNT_MINIMUM requests no copies.
const REPEAT_COUNT_MINIMUM = 0

// REPEAT_COUNT_MAXIMUM permits one-byte source to fill destination.
const REPEAT_COUNT_MAXIMUM = TEXT_SIZE_MAXIMUM

// REPLACEMENT_COUNT_MINIMUM requests every replacement.
const REPLACEMENT_COUNT_MINIMUM = -1

// REPLACEMENT_COUNT_MAXIMUM permits one replacement per input byte.
const REPLACEMENT_COUNT_MAXIMUM = TEXT_SIZE_MAXIMUM

// SPECIAL_CASE_COUNT_MINIMUM keeps zero value on standard Unicode mapping.
const SPECIAL_CASE_COUNT_MINIMUM = ucd.SPECIAL_CASE_COUNT_MINIMUM

// SPECIAL_CASE_COUNT_MAXIMUM matches complete language override data.
const SPECIAL_CASE_COUNT_MAXIMUM = ucd.SPECIAL_CASE_COUNT_MAXIMUM

// SIZE_VALUE_MINIMUM permits empty content.
const SIZE_VALUE_MINIMUM = 0

// SIZE_VALUE_MAXIMUM permits full content.
const SIZE_VALUE_MAXIMUM = TEXT_SIZE_MAXIMUM

// Bytes views caller-owned writable storage.
type Bytes []byte

// Bytes_Invariants bounds destination and result storage.
func Bytes_Invariants(value Bytes, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), TEXT_SIZE_MINIMUM, TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Texts views caller-owned Text slots.
type Texts []Text

// Texts_Invariants bounds every possible split result.
func Texts_Invariants(value Texts, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), TEXT_SIZE_MINIMUM, TEXT_COUNT_MAXIMUM).
		Ensure()
}

// Character_Count counts decoded source characters.
type Character_Count int

// Character_Count_Invariants excludes impossible trailing empty view.
func Character_Count_Invariants(value Character_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), TEXT_SIZE_MINIMUM, TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Fields views caller-owned slots holding non-space input views.
type Fields []Text

// Fields_Invariants applies maximum alternating field count.
func Fields_Invariants(value Fields, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), TEXT_SIZE_MINIMUM, FIELD_COUNT_MAXIMUM).
		Ensure()
}

// Lines views caller-owned slots holding line input views.
type Lines []Text

// Lines_Invariants applies maximum one-line-per-byte count.
func Lines_Invariants(value Lines, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), TEXT_SIZE_MINIMUM, LINE_COUNT_MAXIMUM).
		Ensure()
}

// Limit bounds requested split result count.
type Limit int

// Limit_Invariants admits all-results sentinel beside bounded counts.
func Limit_Invariants(value Limit, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), LIMIT_MINIMUM, LIMIT_MAXIMUM).
		Ensure()
}

// Split_Limit excludes zero after split short-circuit.
type Split_Limit int

// Split_Limit_Invariants applies nonzero split-helper domain.
func Split_Limit_Invariants(value Split_Limit, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Holed_Int(
			int(value), LIMIT_MINIMUM, LIMIT_MAXIMUM,
			0, 0, 0, 0,
		).
		Ensure()
}

// Repeat_Count bounds requested source copies.
type Repeat_Count int

// Repeat_Count_Invariants prevents unbounded work.
func Repeat_Count_Invariants(value Repeat_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), REPEAT_COUNT_MINIMUM, REPEAT_COUNT_MAXIMUM).
		Ensure()
}

// Replacement_Count bounds requested substitutions.
type Replacement_Count int

// Replacement_Count_Invariants admits all-results sentinel.
func Replacement_Count_Invariants(
	value Replacement_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), REPLACEMENT_COUNT_MINIMUM, REPLACEMENT_COUNT_MAXIMUM,
		).
		Ensure()
}

// Size_Value counts bytes inside caller or fixed storage.
type Size_Value int

// Size_Value_Invariants applies Text byte bounds.
func Size_Value_Invariants(value Size_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), SIZE_VALUE_MINIMUM, SIZE_VALUE_MAXIMUM).
		Ensure()
}

// Special_Case_Count separates active prefix from fixed named rule storage.
type Special_Case_Count int

// Special_Case_Count_Invariants bounds work to complete Unicode override data.
func Special_Case_Count_Invariants(value Special_Case_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), SPECIAL_CASE_COUNT_MINIMUM, SPECIAL_CASE_COUNT_MAXIMUM).
		Ensure()
}

// First_Case_Range gives first rule separate invariant identity.
type First_Case_Range ucd.Case_Range

// First_Case_Range_Invariants rejects reversed caller rule ranges.
func First_Case_Range_Invariants(value First_Case_Range, _ aver.Namespace) {
	aver.Always(
		uint32(value.Minimum) <= uint32(value.Maximum),
		"First special case range minimum does not exceed maximum.",
	)
}

// Second_Case_Range gives second rule separate invariant identity.
type Second_Case_Range ucd.Case_Range

// Second_Case_Range_Invariants rejects reversed caller rule ranges.
func Second_Case_Range_Invariants(value Second_Case_Range, _ aver.Namespace) {
	aver.Always(
		uint32(value.Minimum) <= uint32(value.Maximum),
		"Second special case range minimum does not exceed maximum.",
	)
}

// Third_Case_Range gives third rule separate invariant identity.
type Third_Case_Range ucd.Case_Range

// Third_Case_Range_Invariants rejects reversed caller rule ranges.
func Third_Case_Range_Invariants(value Third_Case_Range, _ aver.Namespace) {
	aver.Always(
		uint32(value.Minimum) <= uint32(value.Maximum),
		"Third special case range minimum does not exceed maximum.",
	)
}

// Fourth_Case_Range gives fourth rule separate invariant identity.
type Fourth_Case_Range ucd.Case_Range

// Fourth_Case_Range_Invariants rejects reversed caller rule ranges.
func Fourth_Case_Range_Invariants(value Fourth_Case_Range, _ aver.Namespace) {
	aver.Always(
		uint32(value.Minimum) <= uint32(value.Maximum),
		"Fourth special case range minimum does not exceed maximum.",
	)
}

// Special_Case uses named rules because four entries form bounded shape.
type Special_Case struct {
	// Count makes zero value select standard Unicode mapping.
	Count Special_Case_Count
	// First retains first ordered language rule.
	First First_Case_Range
	// Second retains second ordered language rule.
	Second Second_Case_Range
	// Third retains third ordered language rule.
	Third Third_Case_Range
	// Fourth retains fourth ordered language rule.
	Fourth Fourth_Case_Range
}

// Special_Case_Invariants composes active count and fixed rule shape.
func Special_Case_Invariants(value Special_Case, namespace aver.Namespace) {
	Special_Case_Count_Invariants(value.Count, namespace)
	First_Case_Range_Invariants(value.First, namespace)
	Second_Case_Range_Invariants(value.Second, namespace)
	Third_Case_Range_Invariants(value.Third, namespace)
	Fourth_Case_Range_Invariants(value.Fourth, namespace)
}

// Special_Case_Of copies four bounded rules so caller slices cannot escape.
func Special_Case_Of(
	count Special_Case_Count,
	first First_Case_Range,
	second Second_Case_Range,
	third Third_Case_Range,
	fourth Fourth_Case_Range,
) (special Special_Case) {
	defer func() { Special_Case_Invariants(special, "special_case_of.special") }()
	Special_Case_Count_Invariants(count, "special_case_of.count")
	First_Case_Range_Invariants(first, "special_case_of.first")
	Second_Case_Range_Invariants(second, "special_case_of.second")
	Third_Case_Range_Invariants(third, "special_case_of.third")
	Fourth_Case_Range_Invariants(fourth, "special_case_of.fourth")
	return Special_Case{
		Count:  count,
		First:  first,
		Second: second,
		Third:  third,
		Fourth: fourth,
	}
}

// Clone_Into copies source into caller storage.
func Clone_Into(destination Bytes, source Text) (clone Bytes) {
	defer func() { Bytes_Invariants(clone, "clone_into.clone") }()
	Bytes_Invariants(destination, "clone_into.destination")
	Text_Invariants(source, "clone_into.source")
	if len(source) > len(destination) {
		panic("strings: destination too small")
	}
	copy(destination, source)
	return destination[:len(source)]
}

// Split_Into fills caller slots and returns populated slot count.
func Split_Into(destination Texts, source Text, separator Text) (count Count_Value) {
	defer func() { Count_Value_Invariants(count, "split_into.count") }()
	Texts_Invariants(destination, "split_into.destination")
	Text_Invariants(source, "split_into.source")
	Text_Invariants(separator, "split_into.separator")
	return Split_N_Into(destination, source, separator, LIMIT_MINIMUM)
}

// Split_N_Into fills at most limit caller slots and returns populated slot count.
func Split_N_Into(
	destination Texts, source Text, separator Text, limit Limit,
) (count Count_Value) {
	defer func() { Count_Value_Invariants(count, "split_n_into.count") }()
	Texts_Invariants(destination, "split_n_into.destination")
	Text_Invariants(source, "split_n_into.source")
	Text_Invariants(separator, "split_n_into.separator")
	Limit_Invariants(limit, "split_n_into.limit")
	return split_into(destination, source, separator, limit, false)
}

// Split_After_Into retains each separator in preceding source view.
func Split_After_Into(
	destination Texts, source Text, separator Text,
) (count Count_Value) {
	defer func() { Count_Value_Invariants(count, "split_after_into.count") }()
	Texts_Invariants(destination, "split_after_into.destination")
	Text_Invariants(source, "split_after_into.source")
	Text_Invariants(separator, "split_after_into.separator")
	return Split_After_N_Into(destination, source, separator, LIMIT_MINIMUM)
}

// Split_After_N_Into retains separators while applying result limit.
func Split_After_N_Into(
	destination Texts, source Text, separator Text, limit Limit,
) (count Count_Value) {
	defer func() { Count_Value_Invariants(count, "split_after_n_into.count") }()
	Texts_Invariants(destination, "split_after_n_into.destination")
	Text_Invariants(source, "split_after_n_into.source")
	Text_Invariants(separator, "split_after_n_into.separator")
	Limit_Invariants(limit, "split_after_n_into.limit")
	return split_into(destination, source, separator, limit, true)
}

func split_into(
	destination Texts, source Text, separator Text, limit Limit, after Boolean,
) (count Count_Value) {
	defer func() { Count_Value_Invariants(count, "split_internal.count") }()
	Texts_Invariants(destination, "split_internal.destination")
	Text_Invariants(source, "split_internal.source")
	Text_Invariants(separator, "split_internal.separator")
	Limit_Invariants(limit, "split_internal.limit")
	Boolean_Invariants(after, "split_internal.after")
	if limit == 0 {
		return 0
	}
	if len(separator) == 0 {
		return Count_Value(split_empty_into(
			destination, source, Split_Limit(limit),
		))
	}
	tail := source
	for limit < 0 || int(count) < int(limit)-1 {
		separator_index := Index(tail, separator)
		if separator_index == INDEX_ABSENT {
			break
		}
		end := int(separator_index)
		if after {
			end += len(separator)
		}
		if int(count) == len(destination) {
			panic("strings: destination too small")
		}
		destination[int(count)] = tail[:end]
		count++
		tail = tail[int(separator_index)+len(separator):]
	}
	if int(count) == len(destination) {
		panic("strings: destination too small")
	}
	destination[int(count)] = tail
	return count + 1
}

func split_empty_into(
	destination Texts, source Text, limit Split_Limit,
) (count Character_Count) {
	defer func() { Character_Count_Invariants(count, "split_empty_into.count") }()
	Texts_Invariants(destination, "split_empty_into.destination")
	Text_Invariants(source, "split_empty_into.source")
	Split_Limit_Invariants(limit, "split_empty_into.limit")
	if len(source) == 0 {
		return 0
	}
	tail := source
	for len(tail) > 0 {
		if limit > 0 {
			if int(count)+1 == int(limit) {
				break
			}
		}
		if int(count) == len(destination) {
			panic("strings: destination too small")
		}
		_, size := utf8.Decode_Character_Text(utf8.Text(tail))
		destination[int(count)] = tail[:size]
		count++
		tail = tail[size:]
	}
	if len(tail) == 0 {
		return count
	}
	if int(count) == len(destination) {
		panic("strings: destination too small")
	}
	destination[int(count)] = tail
	return count + 1
}

// Fields_Into fills caller slots with Unicode-space-delimited source views.
func Fields_Into(destination Texts, source Text) (fields Fields) {
	defer func() { Fields_Invariants(fields, "fields_into.fields") }()
	Texts_Invariants(destination, "fields_into.destination")
	Text_Invariants(source, "fields_into.source")
	fields = Fields(destination[:0])
	start := INDEX_ABSENT
	for source_index, character := range source {
		if ucd.Is_Space(ucd.Character(character)) {
			if start >= 0 {
				if len(fields) == len(destination) {
					panic("strings: destination too small")
				}
				destination[len(fields)] = source[start:source_index]
				fields = Fields(destination[:len(fields)+1])
				start = INDEX_ABSENT
			}
		} else if start == INDEX_ABSENT {
			start = source_index
		}
	}
	if start == INDEX_ABSENT {
		return fields
	}
	if len(fields) == len(destination) {
		panic("strings: destination too small")
	}
	destination[len(fields)] = source[start:]
	return Fields(destination[:len(fields)+1])
}

// Fields_Function_Into fills caller slots around injected separators.
func Fields_Function_Into(
	destination Texts, source Text, predicate func(rune) (matches bool),
) (fields Fields) {
	defer func() { Fields_Invariants(fields, "fields_function_into.fields") }()
	Texts_Invariants(destination, "fields_function_into.destination")
	Text_Invariants(source, "fields_function_into.source")
	fields = Fields(destination[:0])
	start := INDEX_ABSENT
	for source_index, character := range source {
		if predicate(character) {
			if start >= 0 {
				if len(fields) == len(destination) {
					panic("strings: destination too small")
				}
				destination[len(fields)] = source[start:source_index]
				fields = Fields(destination[:len(fields)+1])
				start = INDEX_ABSENT
			}
		} else if start == INDEX_ABSENT {
			start = source_index
		}
	}
	if start == INDEX_ABSENT {
		return fields
	}
	if len(fields) == len(destination) {
		panic("strings: destination too small")
	}
	destination[len(fields)] = source[start:]
	return Fields(destination[:len(fields)+1])
}

// Join_Into writes Text collection with separator into caller storage.
func Join_Into(destination Bytes, parts Texts, separator Text) (joined Bytes) {
	defer func() { Bytes_Invariants(joined, "join_into.joined") }()
	Bytes_Invariants(destination, "join_into.destination")
	Texts_Invariants(parts, "join_into.parts")
	Text_Invariants(separator, "join_into.separator")
	written := 0
	for part_index, part := range parts {
		Text_Invariants(part, "join_into.part")
		if part_index > 0 {
			if len(separator) > len(destination)-written {
				panic("strings: destination too small")
			}
			written += copy(destination[written:], separator)
		}
		if len(part) > len(destination)-written {
			panic("strings: destination too small")
		}
		written += copy(destination[written:], part)
	}
	return destination[:written]
}

// Lines_Into fills caller slots with newline-terminated source views.
func Lines_Into(destination Texts, source Text) (lines Lines) {
	defer func() { Lines_Invariants(lines, "lines_into.lines") }()
	Texts_Invariants(destination, "lines_into.destination")
	Text_Invariants(source, "lines_into.source")
	lines = Lines(destination[:0])
	tail := source
	for len(tail) > 0 {
		if len(lines) == len(destination) {
			panic("strings: destination too small")
		}
		line_end := Index_Byte(tail, '\n')
		if line_end == INDEX_ABSENT {
			destination[len(lines)] = tail
			return Lines(destination[:len(lines)+1])
		}
		boundary := int(line_end) + 1
		destination[len(lines)] = tail[:boundary]
		lines = Lines(destination[:len(lines)+1])
		tail = tail[boundary:]
	}
	return lines
}

// Map_Into writes mapped characters into caller storage and drops negative mappings.
func Map_Into(
	destination Bytes, source Text, mapping func(rune) (mapped rune),
) (mapped Bytes) {
	defer func() { Bytes_Invariants(mapped, "map_into.mapped") }()
	Bytes_Invariants(destination, "map_into.destination")
	Text_Invariants(source, "map_into.source")
	written := 0
	for _, character := range source {
		mapped_character := mapping(character)
		if mapped_character < 0 {
			continue
		}
		size := utf8.Character_Size(utf8.Character(mapped_character))
		if size == utf8.CHARACTER_SIZE_INVALID {
			size = utf8.CHARACTER_SIZE_THREE
		}
		if int(size) > len(destination)-written {
			panic("strings: destination too small")
		}
		encoded := utf8.Encode_Character(
			utf8.Bytes(destination[written:]), utf8.Character(mapped_character),
		)
		written += int(encoded)
	}
	return destination[:written]
}

// Repeat_Into writes count consecutive source copies into caller storage.
func Repeat_Into(
	destination Bytes, source Text, count Repeat_Count,
) (repeated Bytes) {
	defer func() { Bytes_Invariants(repeated, "repeat_into.repeated") }()
	Bytes_Invariants(destination, "repeat_into.destination")
	Text_Invariants(source, "repeat_into.source")
	Repeat_Count_Invariants(count, "repeat_into.count")
	repeated_size := len(source) * int(count)
	if len(source) > 0 {
		if int(count) > len(destination)/len(source) {
			panic("strings: destination too small")
		}
	}
	written := 0
	for copy_index := Repeat_Count(0); copy_index < count; copy_index++ {
		written += copy(destination[written:], source)
	}
	return destination[:repeated_size]
}

// To_Upper_Into writes Unicode uppercase mapping into caller storage.
func To_Upper_Into(destination Bytes, source Text) (upper Bytes) {
	defer func() { Bytes_Invariants(upper, "to_upper_into.upper") }()
	Bytes_Invariants(destination, "to_upper_into.destination")
	Text_Invariants(source, "to_upper_into.source")
	return map_case_into(destination, source, ucd.UPPER_CASE, Special_Case{}, false)
}

// To_Lower_Into writes Unicode lowercase mapping into caller storage.
func To_Lower_Into(destination Bytes, source Text) (lower Bytes) {
	defer func() { Bytes_Invariants(lower, "to_lower_into.lower") }()
	Bytes_Invariants(destination, "to_lower_into.destination")
	Text_Invariants(source, "to_lower_into.source")
	return map_case_into(destination, source, ucd.LOWER_CASE, Special_Case{}, false)
}

// To_Title_Into writes Unicode title mapping into caller storage.
func To_Title_Into(destination Bytes, source Text) (title Bytes) {
	defer func() { Bytes_Invariants(title, "to_title_into.title") }()
	Bytes_Invariants(destination, "to_title_into.destination")
	Text_Invariants(source, "to_title_into.source")
	return map_case_into(destination, source, ucd.TITLE_CASE, Special_Case{}, false)
}

// To_Upper_Special_Into applies language override before uppercase mapping.
func To_Upper_Special_Into(
	destination Bytes, special Special_Case, source Text,
) (upper Bytes) {
	defer func() { Bytes_Invariants(upper, "to_upper_special_into.upper") }()
	Bytes_Invariants(destination, "to_upper_special_into.destination")
	Special_Case_Invariants(special, "to_upper_special_into.special")
	Text_Invariants(source, "to_upper_special_into.source")
	return map_case_into(destination, source, ucd.UPPER_CASE, special, true)
}

// To_Lower_Special_Into applies language override before lowercase mapping.
func To_Lower_Special_Into(
	destination Bytes, special Special_Case, source Text,
) (lower Bytes) {
	defer func() { Bytes_Invariants(lower, "to_lower_special_into.lower") }()
	Bytes_Invariants(destination, "to_lower_special_into.destination")
	Special_Case_Invariants(special, "to_lower_special_into.special")
	Text_Invariants(source, "to_lower_special_into.source")
	return map_case_into(destination, source, ucd.LOWER_CASE, special, true)
}

// To_Title_Special_Into applies language override before title mapping.
func To_Title_Special_Into(
	destination Bytes, special Special_Case, source Text,
) (title Bytes) {
	defer func() { Bytes_Invariants(title, "to_title_special_into.title") }()
	Bytes_Invariants(destination, "to_title_special_into.destination")
	Special_Case_Invariants(special, "to_title_special_into.special")
	Text_Invariants(source, "to_title_special_into.source")
	return map_case_into(destination, source, ucd.TITLE_CASE, special, true)
}

func map_case_into(
	destination Bytes, source Text, mapping ucd.Case,
	special Special_Case, use_special Boolean,
) (mapped Bytes) {
	defer func() { Bytes_Invariants(mapped, "map_case_into.mapped") }()
	Bytes_Invariants(destination, "map_case_into.destination")
	Text_Invariants(source, "map_case_into.source")
	ucd.Case_Invariants(mapping, "map_case_into.mapping")
	Special_Case_Invariants(special, "map_case_into.special")
	Boolean_Invariants(use_special, "map_case_into.use_special")
	special_ranges := [...]ucd.Case_Range{
		ucd.Case_Range(special.First),
		ucd.Case_Range(special.Second),
		ucd.Case_Range(special.Third),
		ucd.Case_Range(special.Fourth),
	}
	active_special := ucd.Special_Case(special_ranges[:int(special.Count)])
	written := 0
	for _, character := range source {
		mapped_character := ucd.To(mapping, ucd.Character(character))
		if use_special {
			switch mapping {
			case ucd.UPPER_CASE:
				mapped_character = ucd.Special_Case_To_Upper(
					active_special, ucd.Character(character),
				)
			case ucd.LOWER_CASE:
				mapped_character = ucd.Special_Case_To_Lower(
					active_special, ucd.Character(character),
				)
			case ucd.TITLE_CASE:
				mapped_character = ucd.Special_Case_To_Title(
					active_special, ucd.Character(character),
				)
			}
		}
		size := utf8.Character_Size(utf8.Character(mapped_character))
		if size == utf8.CHARACTER_SIZE_INVALID {
			size = utf8.CHARACTER_SIZE_THREE
		}
		if int(size) > len(destination)-written {
			panic("strings: destination too small")
		}
		encoded := utf8.Encode_Character(
			utf8.Bytes(destination[written:]), utf8.Character(mapped_character),
		)
		written += int(encoded)
	}
	return destination[:written]
}

// To_Valid_UTF8_Into replaces each invalid-byte run in caller storage.
func To_Valid_UTF8_Into(
	destination Bytes, source Text, replacement Text,
) (valid Bytes) {
	defer func() { Bytes_Invariants(valid, "to_valid_utf8_into.valid") }()
	Bytes_Invariants(destination, "to_valid_utf8_into.destination")
	Text_Invariants(source, "to_valid_utf8_into.source")
	Text_Invariants(replacement, "to_valid_utf8_into.replacement")
	written := 0
	invalid := false
	for source_index := 0; source_index < len(source); {
		character, size := utf8.Decode_Character_Text(utf8.Text(source[source_index:]))
		if character == utf8.REPLACEMENT_CHARACTER {
			if size == 1 {
				source_index++
				if invalid {
					continue
				}
				invalid = true
				if len(replacement) > len(destination)-written {
					panic("strings: destination too small")
				}
				written += copy(destination[written:], replacement)
				continue
			}
		}
		invalid = false
		if int(size) > len(destination)-written {
			panic("strings: destination too small")
		}
		boundary := source_index + int(size)
		written += copy(destination[written:], source[source_index:boundary])
		source_index = boundary
	}
	return destination[:written]
}

// Title_Into writes legacy Unicode word-start title mapping into caller storage.
func Title_Into(destination Bytes, source Text) (title Bytes) {
	defer func() { Bytes_Invariants(title, "title_into.title") }()
	Bytes_Invariants(destination, "title_into.destination")
	Text_Invariants(source, "title_into.source")
	written := 0
	previous := rune(' ')
	for _, character := range source {
		mapped_character := character
		if title_separator(Decoded_Character(previous)) {
			mapped_character = rune(ucd.To_Title(ucd.Character(character)))
		}
		previous = character
		size := utf8.Character_Size(utf8.Character(mapped_character))
		if size == utf8.CHARACTER_SIZE_INVALID {
			size = utf8.CHARACTER_SIZE_THREE
		}
		if int(size) > len(destination)-written {
			panic("strings: destination too small")
		}
		encoded := utf8.Encode_Character(
			utf8.Bytes(destination[written:]), utf8.Character(mapped_character),
		)
		written += int(encoded)
	}
	return destination[:written]
}

func title_separator(character Decoded_Character) (separator Boolean) {
	defer func() { Boolean_Invariants(separator, "title_separator.separator") }()
	Decoded_Character_Invariants(character, "title_separator.character")
	if character <= 0x7F {
		if '0' <= character {
			if character <= '9' {
				return false
			}
		}
		if 'a' <= character {
			if character <= 'z' {
				return false
			}
		}
		if 'A' <= character {
			if character <= 'Z' {
				return false
			}
		}
		return Boolean(character != '_')
	}
	ucd_character := ucd.Character(character)
	if ucd.Is_Letter(ucd_character) {
		return false
	}
	if ucd.Is_Digit(ucd_character) {
		return false
	}
	return Boolean(ucd.Is_Space(ucd_character))
}

// Replace_Into writes non-overlapping substitutions into caller storage.
func Replace_Into(
	destination Bytes, source Text, old Text, replacement Text,
	count Replacement_Count,
) (replaced Bytes) {
	defer func() { Bytes_Invariants(replaced, "replace_into.replaced") }()
	Bytes_Invariants(destination, "replace_into.destination")
	Text_Invariants(source, "replace_into.source")
	Text_Invariants(old, "replace_into.old")
	Text_Invariants(replacement, "replace_into.replacement")
	Replacement_Count_Invariants(count, "replace_into.count")
	match_count := Count(source, old)
	if count < 0 {
		count = Replacement_Count(match_count)
	} else if Count_Value(count) > match_count {
		count = Replacement_Count(match_count)
	}
	written := 0
	source_index := 0
	for replacement_index := Replacement_Count(0); replacement_index < count; {
		prefix_size := 0
		if len(old) == 0 {
			if replacement_index > 0 {
				_, size := utf8.Decode_Character_Text(
					utf8.Text(source[source_index:]),
				)
				prefix_size = int(size)
			}
		} else {
			prefix_size = int(Index(source[source_index:], old))
		}
		if prefix_size > len(destination)-written {
			panic("strings: destination too small")
		}
		boundary := source_index
		boundary += prefix_size
		written += copy(destination[written:], source[source_index:boundary])
		if len(replacement) > len(destination)-written {
			panic("strings: destination too small")
		}
		written += copy(destination[written:], replacement)
		source_index = boundary + len(old)
		replacement_index++
	}
	if len(source)-source_index > len(destination)-written {
		panic("strings: destination too small")
	}
	written += copy(destination[written:], source[source_index:])
	return destination[:written]
}

// Replace_All_Into writes every non-overlapping substitution into caller storage.
func Replace_All_Into(
	destination Bytes, source Text, old Text, replacement Text,
) (replaced Bytes) {
	defer func() { Bytes_Invariants(replaced, "replace_all_into.replaced") }()
	Bytes_Invariants(destination, "replace_all_into.destination")
	Text_Invariants(source, "replace_all_into.source")
	Text_Invariants(old, "replace_all_into.old")
	Text_Invariants(replacement, "replace_all_into.replacement")
	return Replace_Into(
		destination, source, old, replacement, REPLACEMENT_COUNT_MINIMUM,
	)
}

// RULE_COUNT_MINIMUM permits identity Replacer.
const RULE_COUNT_MINIMUM = 0

// RULE_COUNT_MAXIMUM prevents unbounded rule search.
const RULE_COUNT_MAXIMUM = TEXT_SIZE_MAXIMUM

// BUILDER_STORAGE_SCALAR_BYTE_COUNT matches fixed-width opaque scalar.
const BUILDER_STORAGE_SCALAR_BYTE_COUNT = 8

// Builder_Storage uses scalar fields because fixed arrays hide element shape.
type Builder_Storage struct {
	// Block_00 groups bytes 0 through 63 so field list stays reviewable.
	Block_00 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_01 groups bytes 64 through 127 so field list stays reviewable.
	Block_01 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_02 groups bytes 128 through 191 so field list stays reviewable.
	Block_02 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_03 groups bytes 192 through 255 so field list stays reviewable.
	Block_03 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_04 groups bytes 256 through 319 so field list stays reviewable.
	Block_04 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_05 groups bytes 320 through 383 so field list stays reviewable.
	Block_05 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_06 groups bytes 384 through 447 so field list stays reviewable.
	Block_06 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_07 groups bytes 448 through 511 so field list stays reviewable.
	Block_07 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_08 groups bytes 512 through 575 so field list stays reviewable.
	Block_08 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_09 groups bytes 576 through 639 so field list stays reviewable.
	Block_09 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_10 groups bytes 640 through 703 so field list stays reviewable.
	Block_10 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_11 groups bytes 704 through 767 so field list stays reviewable.
	Block_11 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_12 groups bytes 768 through 831 so field list stays reviewable.
	Block_12 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_13 groups bytes 832 through 895 so field list stays reviewable.
	Block_13 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_14 groups bytes 896 through 959 so field list stays reviewable.
	Block_14 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_15 groups bytes 960 through 1023 so field list stays reviewable.
	Block_15 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_16 groups bytes 1024 through 1087 so field list stays reviewable.
	Block_16 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_17 groups bytes 1088 through 1151 so field list stays reviewable.
	Block_17 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_18 groups bytes 1152 through 1215 so field list stays reviewable.
	Block_18 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_19 groups bytes 1216 through 1279 so field list stays reviewable.
	Block_19 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_20 groups bytes 1280 through 1343 so field list stays reviewable.
	Block_20 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_21 groups bytes 1344 through 1407 so field list stays reviewable.
	Block_21 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_22 groups bytes 1408 through 1471 so field list stays reviewable.
	Block_22 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_23 groups bytes 1472 through 1535 so field list stays reviewable.
	Block_23 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_24 groups bytes 1536 through 1599 so field list stays reviewable.
	Block_24 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_25 groups bytes 1600 through 1663 so field list stays reviewable.
	Block_25 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_26 groups bytes 1664 through 1727 so field list stays reviewable.
	Block_26 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_27 groups bytes 1728 through 1791 so field list stays reviewable.
	Block_27 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_28 groups bytes 1792 through 1855 so field list stays reviewable.
	Block_28 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_29 groups bytes 1856 through 1919 so field list stays reviewable.
	Block_29 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_30 groups bytes 1920 through 1983 so field list stays reviewable.
	Block_30 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_31 groups bytes 1984 through 2047 so field list stays reviewable.
	Block_31 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_32 groups bytes 2048 through 2111 so field list stays reviewable.
	Block_32 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_33 groups bytes 2112 through 2175 so field list stays reviewable.
	Block_33 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_34 groups bytes 2176 through 2239 so field list stays reviewable.
	Block_34 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_35 groups bytes 2240 through 2303 so field list stays reviewable.
	Block_35 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_36 groups bytes 2304 through 2367 so field list stays reviewable.
	Block_36 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_37 groups bytes 2368 through 2431 so field list stays reviewable.
	Block_37 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_38 groups bytes 2432 through 2495 so field list stays reviewable.
	Block_38 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_39 groups bytes 2496 through 2559 so field list stays reviewable.
	Block_39 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_40 groups bytes 2560 through 2623 so field list stays reviewable.
	Block_40 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_41 groups bytes 2624 through 2687 so field list stays reviewable.
	Block_41 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_42 groups bytes 2688 through 2751 so field list stays reviewable.
	Block_42 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_43 groups bytes 2752 through 2815 so field list stays reviewable.
	Block_43 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_44 groups bytes 2816 through 2879 so field list stays reviewable.
	Block_44 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_45 groups bytes 2880 through 2943 so field list stays reviewable.
	Block_45 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_46 groups bytes 2944 through 3007 so field list stays reviewable.
	Block_46 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_47 groups bytes 3008 through 3071 so field list stays reviewable.
	Block_47 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_48 groups bytes 3072 through 3135 so field list stays reviewable.
	Block_48 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_49 groups bytes 3136 through 3199 so field list stays reviewable.
	Block_49 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_50 groups bytes 3200 through 3263 so field list stays reviewable.
	Block_50 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_51 groups bytes 3264 through 3327 so field list stays reviewable.
	Block_51 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_52 groups bytes 3328 through 3391 so field list stays reviewable.
	Block_52 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_53 groups bytes 3392 through 3455 so field list stays reviewable.
	Block_53 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_54 groups bytes 3456 through 3519 so field list stays reviewable.
	Block_54 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_55 groups bytes 3520 through 3583 so field list stays reviewable.
	Block_55 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_56 groups bytes 3584 through 3647 so field list stays reviewable.
	Block_56 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_57 groups bytes 3648 through 3711 so field list stays reviewable.
	Block_57 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_58 groups bytes 3712 through 3775 so field list stays reviewable.
	Block_58 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_59 groups bytes 3776 through 3839 so field list stays reviewable.
	Block_59 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_60 groups bytes 3840 through 3903 so field list stays reviewable.
	Block_60 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_61 groups bytes 3904 through 3967 so field list stays reviewable.
	Block_61 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_62 groups bytes 3968 through 4031 so field list stays reviewable.
	Block_62 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
	// Block_63 groups bytes 4032 through 4095 so field list stays reviewable.
	Block_63 struct {
		// Bytes_00_07 keeps byte positions explicit without banned array storage.
		Bytes_00_07 bits.Word_64
		// Bytes_08_15 keeps byte positions explicit without banned array storage.
		Bytes_08_15 bits.Word_64
		// Bytes_16_23 keeps byte positions explicit without banned array storage.
		Bytes_16_23 bits.Word_64
		// Bytes_24_31 keeps byte positions explicit without banned array storage.
		Bytes_24_31 bits.Word_64
		// Bytes_32_39 keeps byte positions explicit without banned array storage.
		Bytes_32_39 bits.Word_64
		// Bytes_40_47 keeps byte positions explicit without banned array storage.
		Bytes_40_47 bits.Word_64
		// Bytes_48_55 keeps byte positions explicit without banned array storage.
		Bytes_48_55 bits.Word_64
		// Bytes_56_63 keeps byte positions explicit without banned array storage.
		Bytes_56_63 bits.Word_64
	}
}

// Builder_Storage_Invariants proves scalar layout matches complete text capacity.
func Builder_Storage_Invariants(value Builder_Storage, _ aver.Namespace) {
	aver.Always(
		unsafe.Sizeof(bits.Word_64(0)) == uintptr(BUILDER_STORAGE_SCALAR_BYTE_COUNT),
		"Builder storage scalar owns eight bytes.",
	)
	aver.Always(
		unsafe.Sizeof(value) == uintptr(TEXT_SIZE_MAXIMUM),
		"Builder storage owns complete text capacity without padding.",
	)
}

// Builder_Storage_View prevents fixed-capacity bytes from widening into general Bytes domain.
type Builder_Storage_View []byte

// Builder_Storage_View_Invariants fixes unsafe view to complete backing storage.
func Builder_Storage_View_Invariants(value Builder_Storage_View, _ aver.Namespace) {
	aver.Always(
		len(value) == TEXT_SIZE_MAXIMUM,
		"Builder storage view spans complete backing storage.",
	)
}

// Builder storage byte view exists only after layout proof keeps unsafe span bounded.
func builder_storage_bytes(storage *Builder_Storage) (view Builder_Storage_View) {
	defer func() {
		Builder_Storage_View_Invariants(view, "builder_storage_bytes.view")
	}()
	Builder_Storage_Invariants(*storage, "builder_storage_bytes.storage")
	return unsafe.Slice((*byte)(unsafe.Pointer(storage)), TEXT_SIZE_MAXIMUM)
}

// Builder holds fixed caller-stack-compatible storage.
type Builder struct {
	// Storage keeps ownership visible in value.
	Storage Builder_Storage
	// Size selects initialized storage prefix.
	Size Size_Value
}

// Builder_Invariants composes fixed storage and current content size.
func Builder_Invariants(value *Builder, namespace aver.Namespace) {
	Builder_Storage_Invariants(value.Storage, namespace)
	Size_Value_Invariants(value.Size, namespace)
}

// Builder_Capacity_Value is fixed storage byte count.
type Builder_Capacity_Value int

// Builder_Capacity_Value_Invariants fixes one compile-time capacity.
func Builder_Capacity_Value_Invariants(
	value Builder_Capacity_Value, _ aver.Namespace,
) {
	aver.Always(
		int(value) == TEXT_SIZE_MAXIMUM,
		"Builder capacity equals fixed Text capacity.",
	)
}

// Builder_Bytes returns current fixed-storage view.
func Builder_Bytes(builder *Builder) (content Bytes) {
	defer func() { Bytes_Invariants(content, "builder_bytes.content") }()
	Builder_Invariants(builder, "builder_bytes.builder")
	return Bytes(builder_storage_bytes(&builder.Storage)[:builder.Size])
}

// Builder_Size reports current encoded byte count.
func Builder_Size(builder *Builder) (size Size_Value) {
	defer func() { Size_Value_Invariants(size, "builder_size.size") }()
	Builder_Invariants(builder, "builder_size.builder")
	return builder.Size
}

// Builder_Capacity reports fixed storage capacity.
func Builder_Capacity(builder *Builder) (capacity Builder_Capacity_Value) {
	defer func() {
		Builder_Capacity_Value_Invariants(capacity, "builder_capacity.capacity")
	}()
	Builder_Invariants(builder, "builder_capacity.builder")
	return TEXT_SIZE_MAXIMUM
}

// Builder_Reset removes content without replacing storage.
func Builder_Reset(builder *Builder) {
	Builder_Invariants(builder, "builder_reset.builder")
	builder.Size = 0
}

// Builder_Write copies bytes into remaining fixed storage.
func Builder_Write(builder *Builder, source Bytes) (written Size_Value) {
	defer func() { Size_Value_Invariants(written, "builder_write.written") }()
	Builder_Invariants(builder, "builder_write.builder")
	Bytes_Invariants(source, "builder_write.source")
	if len(source) > TEXT_SIZE_MAXIMUM-int(builder.Size) {
		panic("strings: destination too small")
	}
	written = Size_Value(copy(builder_storage_bytes(&builder.Storage)[builder.Size:], source))
	builder.Size += written
	return written
}

// Builder_Write_Text copies Text into remaining fixed storage.
func Builder_Write_Text(builder *Builder, source Text) (written Size_Value) {
	defer func() { Size_Value_Invariants(written, "builder_write_text.written") }()
	Builder_Invariants(builder, "builder_write_text.builder")
	Text_Invariants(source, "builder_write_text.source")
	if len(source) > TEXT_SIZE_MAXIMUM-int(builder.Size) {
		panic("strings: destination too small")
	}
	written = Size_Value(copy(builder_storage_bytes(&builder.Storage)[builder.Size:], source))
	builder.Size += written
	return written
}

// Builder_Write_Byte writes one byte into fixed storage.
func Builder_Write_Byte(builder *Builder, value Byte) {
	Builder_Invariants(builder, "builder_write_byte.builder")
	Byte_Invariants(value, "builder_write_byte.value")
	if builder.Size == TEXT_SIZE_MAXIMUM {
		panic("strings: destination too small")
	}
	builder_storage_bytes(&builder.Storage)[builder.Size] = byte(value)
	builder.Size++
}

// Builder_Write_Character writes one UTF-8 encoding into fixed storage.
func Builder_Write_Character(builder *Builder, character Character) {
	Builder_Invariants(builder, "builder_write_character.builder")
	Character_Invariants(character, "builder_write_character.character")
	size := utf8.Character_Size(utf8.Character(character))
	if size == utf8.CHARACTER_SIZE_INVALID {
		size = utf8.CHARACTER_SIZE_THREE
	}
	if int(size) > TEXT_SIZE_MAXIMUM-int(builder.Size) {
		panic("strings: destination too small")
	}
	written := utf8.Encode_Character(
		utf8.Bytes(builder_storage_bytes(&builder.Storage)[builder.Size:]),
		utf8.Character(character),
	)
	builder.Size += Size_Value(written)
}

// Decoded_Character is one valid code point Reader returns.
type Decoded_Character rune

// Decoded_Character_Invariants applies Unicode code-point domain.
func Decoded_Character_Invariants(
	value Decoded_Character, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int32(
			int32(value),
			utf8.DECODED_CHARACTER_MINIMUM,
			utf8.DECODED_CHARACTER_MAXIMUM,
		).
		Ensure()
}

// Reader holds borrowed Text and byte cursor without wrapper allocation.
type Reader struct {
	// Source remains caller-owned.
	Source Text
	// Position selects unread suffix boundary.
	Position Size_Value
	// Previous records character boundary eligible for unread.
	Previous Index_Value
}

// Reader_Invariants composes borrowed source and cursor state.
func Reader_Invariants(value Reader, namespace aver.Namespace) {
	Text_Invariants(value.Source, namespace)
	Size_Value_Invariants(value.Position, namespace)
	Index_Value_Invariants(value.Previous, namespace)
	aver.Always(
		int(value.Position) <= len(value.Source),
		"Reader position does not exceed source size.",
	)
}

// Reader_Reset replaces borrowed source and resets cursor.
func Reader_Reset(reader *Reader, source Text) {
	Reader_Invariants(*reader, "reader_reset.reader")
	Text_Invariants(source, "reader_reset.source")
	reader.Source = source
	reader.Position = 0
	reader.Previous = INDEX_ABSENT
}

// Reader_Size reports unread byte count.
func Reader_Size(reader *Reader) (size Size_Value) {
	defer func() { Size_Value_Invariants(size, "reader_size.size") }()
	Reader_Invariants(*reader, "reader_size.reader")
	return Size_Value(len(reader.Source) - int(reader.Position))
}

// Reader_Read_Into copies unread bytes into caller storage.
func Reader_Read_Into(reader *Reader, destination Bytes) (read Bytes) {
	defer func() { Bytes_Invariants(read, "reader_read_into.read") }()
	Reader_Invariants(*reader, "reader_read_into.reader")
	Bytes_Invariants(destination, "reader_read_into.destination")
	unread := len(reader.Source) - int(reader.Position)
	count := len(destination)
	if unread < count {
		count = unread
	}
	boundary := int(reader.Position) + count
	copy(destination, reader.Source[reader.Position:boundary])
	reader.Position += Size_Value(count)
	reader.Previous = INDEX_ABSENT
	return destination[:count]
}

// Reader_Read_Byte returns next byte and presence.
func Reader_Read_Byte(reader *Reader) (value Byte, found Boolean) {
	defer func() {
		Byte_Invariants(value, "reader_read_byte.value")
		Boolean_Invariants(found, "reader_read_byte.found")
	}()
	Reader_Invariants(*reader, "reader_read_byte.reader")
	if int(reader.Position) == len(reader.Source) {
		return 0, false
	}
	value = Byte(reader.Source[reader.Position])
	reader.Position++
	reader.Previous = INDEX_ABSENT
	return value, true
}

// Reader_Unread_Byte moves cursor back one byte.
func Reader_Unread_Byte(reader *Reader) {
	Reader_Invariants(*reader, "reader_unread_byte.reader")
	if reader.Position == 0 {
		panic("strings: no byte to unread")
	}
	reader.Position--
	reader.Previous = INDEX_ABSENT
}

// Reader_Read_Character returns next decoded character and presence.
func Reader_Read_Character(
	reader *Reader,
) (character Decoded_Character, found Boolean) {
	defer func() {
		Decoded_Character_Invariants(character, "reader_read_character.character")
		Boolean_Invariants(found, "reader_read_character.found")
	}()
	Reader_Invariants(*reader, "reader_read_character.reader")
	if int(reader.Position) == len(reader.Source) {
		return 0, false
	}
	reader.Previous = Index_Value(reader.Position)
	decoded, size := utf8.Decode_Character_Text(
		utf8.Text(reader.Source[reader.Position:]),
	)
	reader.Position += Size_Value(size)
	return Decoded_Character(decoded), true
}

// Reader_Unread_Character restores most recent character boundary.
func Reader_Unread_Character(reader *Reader) {
	Reader_Invariants(*reader, "reader_unread_character.reader")
	if reader.Previous == INDEX_ABSENT {
		panic("strings: no character to unread")
	}
	reader.Position = Size_Value(reader.Previous)
	reader.Previous = INDEX_ABSENT
}

// Old_Text identifies one Replacer match pattern.
type Old_Text string

// Old_Text_Invariants applies Text byte bounds.
func Old_Text_Invariants(value Old_Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), TEXT_SIZE_MINIMUM, TEXT_SIZE_MAXIMUM).
		Ensure()
}

// New_Text identifies one Replacer substitution.
type New_Text string

// New_Text_Invariants applies Text byte bounds.
func New_Text_Invariants(value New_Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), TEXT_SIZE_MINIMUM, TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Rule holds one borrowed ordered replacement pair.
type Rule struct {
	// Old selects matched bytes.
	Old Old_Text
	// New selects emitted bytes.
	New New_Text
}

// Rule_Invariants composes distinct old and new Text identities.
func Rule_Invariants(value Rule, namespace aver.Namespace) {
	Old_Text_Invariants(value.Old, namespace)
	New_Text_Invariants(value.New, namespace)
}

// Rules views caller-owned ordered replacement pairs.
type Rules []Rule

// Rules_Invariants prevents unbounded ordered search.
func Rules_Invariants(value Rules, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), RULE_COUNT_MINIMUM, RULE_COUNT_MAXIMUM).
		Ensure()
}

// Replacer borrows ordered caller rules.
type Replacer struct {
	// Rules remains caller-owned and ordered.
	Rules Rules
}

// Replacer_Invariants composes borrowed bounded rules.
func Replacer_Invariants(value Replacer, namespace aver.Namespace) {
	Rules_Invariants(value.Rules, namespace)
}

// Replacer_Init borrows caller rules without copying storage.
func Replacer_Init(rules Rules) (replacer Replacer) {
	defer func() { Replacer_Invariants(replacer, "replacer_init.replacer") }()
	Rules_Invariants(rules, "replacer_init.rules")
	for _, rule := range rules {
		Rule_Invariants(rule, "replacer_init.rule")
	}
	return Replacer{Rules: rules}
}

// Replacer_Replace_Into applies ordered caller rules into caller storage.
func Replacer_Replace_Into(
	destination Bytes, replacer Replacer, source Text,
) (replaced Bytes) {
	defer func() { Bytes_Invariants(replaced, "replacer_replace_into.replaced") }()
	Bytes_Invariants(destination, "replacer_replace_into.destination")
	Replacer_Invariants(replacer, "replacer_replace_into.replacer")
	Text_Invariants(source, "replacer_replace_into.source")
	written := 0
	source_index := 0
	previous_empty := false
	for source_index <= len(source) {
		matched := false
		for _, rule := range replacer.Rules {
			Rule_Invariants(rule, "replacer_replace_into.rule")
			old := Text(rule.Old)
			if len(old) == 0 {
				if previous_empty {
					continue
				}
			}
			if !Has_Prefix(source[source_index:], old) {
				continue
			}
			if len(rule.New) > len(destination)-written {
				panic("strings: destination too small")
			}
			written += copy(destination[written:], Text(rule.New))
			source_index += len(old)
			previous_empty = len(old) == 0
			matched = true
			break
		}
		if matched {
			continue
		}
		previous_empty = false
		if source_index == len(source) {
			break
		}
		if written == len(destination) {
			panic("strings: destination too small")
		}
		destination[written] = source[source_index]
		written++
		source_index++
	}
	return destination[:written]
}
