// Package strings supplies bounded text operations used by lint implementation.
package strings

import "local/james-orcales/shared/unicode/ucd"

// TEXT_SIZE_MAXIMUM matches largest bounded lint input.
const TEXT_SIZE_MAXIMUM = 1 << 20

// Builder owns bounded diagnostic or fixture text.
type Builder struct {
	// Content is exported only because house naming forbids lowercase fields.
	Content []byte
}

// Builder_Invariants bounds owned text.
func Builder_Invariants(builder Builder) {
	if len(builder.Content) > TEXT_SIZE_MAXIMUM {
		panic("lint strings: text too large")
	}
}

// Builder_Size returns owned byte count.
func Builder_Size(builder *Builder) (size int) {
	Builder_Invariants(*builder)
	return len(builder.Content)
}

// Write satisfies io.Writer without permitting unbounded growth.
func (builder *Builder) Write(source []byte) (count int, err error) {
	Builder_Invariants(*builder)
	if len(source) > TEXT_SIZE_MAXIMUM-len(builder.Content) {
		panic("lint strings: text too large")
	}
	builder.Content = append(builder.Content, source...)
	return len(source), nil
}

// String satisfies fmt.Stringer without exposing mutable storage.
func (builder *Builder) String() (text string) {
	Builder_Invariants(*builder)
	return string(builder.Content)
}

// Builder_Write_Text appends text under Builder bound.
func Builder_Write_Text(builder *Builder, source string) {
	count, err := builder.Write([]byte(source))
	if err != nil {
		panic(err)
	}
	if count != len(source) {
		panic("lint strings: short write")
	}
}

// Builder_Write_Byte appends one byte under Builder bound.
func Builder_Write_Byte(builder *Builder, value byte) {
	count, err := builder.Write([]byte{value})
	if err != nil {
		panic(err)
	}
	if count != 1 {
		panic("lint strings: short write")
	}
}

// Contains reports whether separator occurs in source.
func Contains(source string, separator string) (contained bool) {
	return Index(source, separator) >= 0
}

// Contains_Any reports whether any character occurs in source.
func Contains_Any(source string, characters string) (contained bool) {
	for _, character := range source {
		if text_contains_character(characters, character) {
			return true
		}
	}
	return false
}

// Contains_Rune reports whether character occurs in source.
func Contains_Rune(source string, character rune) (contained bool) {
	for _, candidate := range source {
		if candidate == character {
			return true
		}
	}
	return false
}

// Count reports non-overlapping separator matches.
func Count(source string, separator string) (count int) {
	if separator == "" {
		count = 1
		for range source {
			count++
		}
		return count
	}
	for offset := 0; offset <= len(source)-len(separator); {
		index := Index(source[offset:], separator)
		if index < 0 {
			return count
		}
		count++
		offset += index + len(separator)
	}
	return count
}

// Equal_Fold reports Unicode simple-fold equality.
func Equal_Fold(left string, right string) (equal bool) {
	left_characters := []rune(left)
	right_characters := []rune(right)
	if len(left_characters) != len(right_characters) {
		return false
	}
	for index, left_character := range left_characters {
		if !character_equal_fold(left_character, right_characters[index]) {
			return false
		}
	}
	return true
}

// Fields splits source around Unicode whitespace.
func Fields(source string) (fields []string) {
	start := -1
	for index, character := range source {
		if bool(ucd.Is_Space(ucd.Character(character))) {
			if start >= 0 {
				fields = append(fields, source[start:index])
				start = -1
			}
			continue
		}
		if start < 0 {
			start = index
		}
	}
	if start >= 0 {
		fields = append(fields, source[start:])
	}
	return fields
}

// Has_Prefix reports whether source begins with prefix.
func Has_Prefix(source string, prefix string) (present bool) {
	if len(prefix) > len(source) {
		return false
	}
	return source[:len(prefix)] == prefix
}

// Has_Suffix reports whether source ends with suffix.
func Has_Suffix(source string, suffix string) (present bool) {
	if len(suffix) > len(source) {
		return false
	}
	return source[len(source)-len(suffix):] == suffix
}

// Index reports first separator byte index.
func Index(source string, separator string) (index int) {
	if separator == "" {
		return 0
	}
	for index = 0; index <= len(source)-len(separator); index++ {
		if source[index:index+len(separator)] == separator {
			return index
		}
	}
	return -1
}

// Index_Byte reports first byte index.
func Index_Byte(source string, value byte) (index int) {
	for index = 0; index < len(source); index++ {
		if source[index] == value {
			return index
		}
	}
	return -1
}

// Join concatenates parts with separator under lint input bound.
func Join(parts []string, separator string) (joined string) {
	size := 0
	for index, part := range parts {
		if len(part) > TEXT_SIZE_MAXIMUM-size {
			panic("lint strings: text too large")
		}
		size += len(part)
		if index > 0 {
			if len(separator) > TEXT_SIZE_MAXIMUM-size {
				panic("lint strings: text too large")
			}
			size += len(separator)
		}
	}
	buffer := make([]byte, 0, size)
	for index, part := range parts {
		if index > 0 {
			buffer = append(buffer, separator...)
		}
		buffer = append(buffer, part...)
	}
	return string(buffer)
}

// Last_Index reports final separator byte index.
func Last_Index(source string, separator string) (index int) {
	if separator == "" {
		return len(source)
	}
	for index = len(source) - len(separator); index >= 0; index-- {
		if source[index:index+len(separator)] == separator {
			return index
		}
	}
	return -1
}

// Last_Index_Byte reports final byte index.
func Last_Index_Byte(source string, value byte) (index int) {
	for index = len(source) - 1; index >= 0; index-- {
		if source[index] == value {
			return index
		}
	}
	return -1
}

// Repeat repeats source count times under lint input bound.
func Repeat(source string, count int) (repeated string) {
	if count < 0 {
		panic("lint strings: negative repeat count")
	}
	if len(source) > 0 {
		if count > TEXT_SIZE_MAXIMUM/len(source) {
			panic("lint strings: text too large")
		}
	}
	size := len(source) * count
	buffer := make([]byte, 0, size)
	for index := 0; index < count; index++ {
		buffer = append(buffer, source...)
	}
	return string(buffer)
}

// Replace substitutes first count non-overlapping matches; negative count replaces all.
func Replace(source string, old string, replacement string, count int) (result string) {
	if count == 0 {
		return source
	}
	parts := Split(source, old)
	match_count := len(parts) - 1
	if count >= 0 {
		if count < match_count {
			match_count = count
		}
	}
	var builder Builder
	for index, part := range parts {
		Builder_Write_Text(&builder, part)
		if index < match_count {
			Builder_Write_Text(&builder, replacement)
			continue
		}
		if index == match_count {
			if index+1 < len(parts) {
				Builder_Write_Text(&builder, old)
				Builder_Write_Text(&builder, Join(parts[index+1:], old))
			}
			break
		}
	}
	return builder.String()
}

// Replace_All substitutes every non-overlapping match.
func Replace_All(source string, old string, replacement string) (result string) {
	return Replace(source, old, replacement, -1)
}

// Split divides source after each separator match.
func Split(source string, separator string) (parts []string) {
	if separator == "" {
		if source == "" {
			return []string{}
		}
		parts = append(parts, "")
		start := 0
		for index := range source {
			if index == 0 {
				continue
			}
			parts = append(parts, source[start:index])
			start = index
		}
		parts = append(parts, source[start:])
		return append(parts, "")
	}
	start := 0
	for start <= len(source) {
		index := Index(source[start:], separator)
		if index < 0 {
			return append(parts, source[start:])
		}
		parts = append(parts, source[start:start+index])
		start += index + len(separator)
	}
	return parts
}

// To_Lower maps each Unicode character to lower case.
func To_Lower(source string) (lower string) {
	var builder Builder
	for _, character := range source {
		mapped := rune(ucd.To_Lower(ucd.Character(character)))
		Builder_Write_Text(&builder, string(mapped))
	}
	return builder.String()
}

// To_Upper maps each Unicode character to upper case.
func To_Upper(source string) (upper string) {
	var builder Builder
	for _, character := range source {
		mapped := rune(ucd.To_Upper(ucd.Character(character)))
		Builder_Write_Text(&builder, string(mapped))
	}
	return builder.String()
}

// Trim removes cutset characters from both ends.
func Trim(source string, cutset string) (trimmed string) {
	return Trim_Right(Trim_Left(source, cutset), cutset)
}

// Trim_Left removes leading cutset characters.
func Trim_Left(source string, cutset string) (trimmed string) {
	start := 0
	for index, character := range source {
		if !text_contains_character(cutset, character) {
			return source[index:]
		}
		start = index + character_size(character)
	}
	return source[start:]
}

// Trim_Prefix removes prefix when present.
func Trim_Prefix(source string, prefix string) (trimmed string) {
	if Has_Prefix(source, prefix) {
		return source[len(prefix):]
	}
	return source
}

// Trim_Right removes trailing cutset characters.
func Trim_Right(source string, cutset string) (trimmed string) {
	end := 0
	for index, character := range source {
		if !text_contains_character(cutset, character) {
			end = index + character_size(character)
		}
	}
	return source[:end]
}

// Trim_Space removes leading and trailing Unicode whitespace.
func Trim_Space(source string) (trimmed string) {
	start_count := len(source)
	end := 0
	for index, character := range source {
		if bool(ucd.Is_Space(ucd.Character(character))) {
			continue
		}
		if start_count == len(source) {
			start_count = index
		}
		end = index + character_size(character)
	}
	if end == 0 {
		return ""
	}
	return source[start_count:end]
}

// Trim_Suffix removes suffix when present.
func Trim_Suffix(source string, suffix string) (trimmed string) {
	if Has_Suffix(source, suffix) {
		return source[:len(source)-len(suffix)]
	}
	return source
}

func character_equal_fold(left rune, right rune) (equal bool) {
	if left == right {
		return true
	}
	folded := rune(ucd.Simple_Fold(ucd.Character(left)))
	for folded != left {
		if folded == right {
			return true
		}
		folded = rune(ucd.Simple_Fold(ucd.Character(folded)))
	}
	return false
}

func character_size(character rune) (size int) {
	switch {
	case character <= 0x7F:
		return 1
	case character <= 0x7FF:
		return 2
	case character <= 0xFFFF:
		return 3
	default:
		return 4
	}
}

func text_contains_character(text string, character rune) (contained bool) {
	for _, candidate := range text {
		if candidate == character {
			return true
		}
	}
	return false
}
