// Package strings supplies bounded text operations used by lint implementation.
package strings

import (
	"local/james-orcales/shared/unicode/ucd"
	"local/james-orcales/shared/unicode/utf8"
)

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
		if character_is_space(character) {
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
	if len(separator) == 1 {
		return Index_Byte(source, separator[0])
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
	if len(separator) == 1 {
		return Last_Index_Byte(source, separator[0])
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
	for index, character := range source {
		mapped := character_to_lower(character)
		if mapped == character {
			continue
		}
		var builder Builder
		Builder_Write_Text(&builder, source[:index])
		Builder_Write_Text(&builder, string(mapped))
		next_offset := index + character_size(character)
		for _, suffix_character := range source[next_offset:] {
			Builder_Write_Text(&builder, string(character_to_lower(suffix_character)))
		}
		return builder.String()
	}
	return source
}

// To_Upper maps each Unicode character to upper case.
func To_Upper(source string) (upper string) {
	for index, character := range source {
		mapped := character_to_upper(character)
		if mapped == character {
			continue
		}
		var builder Builder
		Builder_Write_Text(&builder, source[:index])
		Builder_Write_Text(&builder, string(mapped))
		next_offset := index + character_size(character)
		for _, suffix_character := range source[next_offset:] {
			Builder_Write_Text(&builder, string(character_to_upper(suffix_character)))
		}
		return builder.String()
	}
	return source
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
	end_count := len(source)
	for end_count > 0 {
		character, size := text_final_character(source[:end_count])
		if !text_contains_character(cutset, character) {
			break
		}
		end_count -= size
	}
	return source[:end_count]
}

// Trim_Space removes leading and trailing Unicode whitespace.
func Trim_Space(source string) (trimmed string) {
	start_offset := 0
	for start_offset < len(source) {
		character, size := text_first_character(source[start_offset:])
		if !character_is_space(character) {
			break
		}
		start_offset += size
	}
	if start_offset == len(source) {
		return ""
	}
	end_count := len(source)
	for end_count > start_offset {
		character, size := text_final_character(source[start_offset:end_count])
		if !character_is_space(character) {
			break
		}
		end_count -= size
	}
	return source[start_offset:end_count]
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

func character_is_space(character rune) (space bool) {
	if character <= 0x7f {
		return ascii_byte_is_space(byte(character))
	}
	return bool(ucd.Is_Space(ucd.Character(character)))
}

func ascii_byte_is_space(value byte) (space bool) {
	switch value {
	case ' ', '\t', '\n', '\v', '\f', '\r':
		return true
	}
	return false
}

func character_to_lower(character rune) (mapped rune) {
	if 'A' <= character {
		if character <= 'Z' {
			return character + ('a' - 'A')
		}
	}
	if character <= 0x7f {
		return character
	}
	return rune(ucd.To_Lower(ucd.Character(character)))
}

func character_to_upper(character rune) (mapped rune) {
	if 'a' <= character {
		if character <= 'z' {
			return character - ('a' - 'A')
		}
	}
	if character <= 0x7f {
		return character
	}
	return rune(ucd.To_Upper(ucd.Character(character)))
}

func text_first_character(source string) (character rune, size int) {
	if source[0] < byte(utf8.CHARACTER_SELF) {
		return rune(source[0]), 1
	}
	end_offset := utf8.UTF_MAXIMUM
	if len(source) < end_offset {
		end_offset = len(source)
	}
	decoded, decoded_size := utf8.Decode_Character_Text(utf8.Text(source[:end_offset]))
	return rune(decoded), int(decoded_size)
}

func text_final_character(source string) (character rune, size int) {
	if source[len(source)-1] < byte(utf8.CHARACTER_SELF) {
		return rune(source[len(source)-1]), 1
	}
	start_offset := 0
	if len(source) > utf8.UTF_MAXIMUM {
		start_offset = len(source) - utf8.UTF_MAXIMUM
	}
	decoded, decoded_size := utf8.Decode_Final_Character_Text(utf8.Text(source[start_offset:]))
	return rune(decoded), int(decoded_size)
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
