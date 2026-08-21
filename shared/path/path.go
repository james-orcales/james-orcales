// Package path keeps slash-separated paths bounded. Caller-owned output avoids allocation.
package path

import (
	"errors"

	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/unicode/utf8"
)

// ELEMENT_COUNT_MINIMUM keeps zero-element Join_Into valid.
const ELEMENT_COUNT_MINIMUM = 0

// ELEMENT_COUNT_MAXIMUM caps empty-element work to one element per pathname boundary.
const ELEMENT_COUNT_MAXIMUM = PATH_SIZE_MAXIMUM + NONEMPTY_SIZE_MINIMUM

// PATH_SIZE_MINIMUM keeps empty input valid.
const PATH_SIZE_MINIMUM = bytes.TEXT_SIZE_MINIMUM

// NONEMPTY_SIZE_MINIMUM accounts for mandatory dot output and active scan input.
const NONEMPTY_SIZE_MINIMUM = 1

// PATH_TERMINATOR_BYTES reserves one kernel C-string NUL outside pathname text.
const PATH_TERMINATOR_BYTES = 1

// TAIL_SIZE_MAXIMUM subtracts one byte because tail follows consumed input.
const TAIL_SIZE_MAXIMUM = PATH_SIZE_MAXIMUM - NONEMPTY_SIZE_MINIMUM

// DIRECTORY_SIZE_MAXIMUM subtracts final element or slash from full input.
const DIRECTORY_SIZE_MAXIMUM = TAIL_SIZE_MAXIMUM

// CLASS_TAIL_SIZE_MAXIMUM leaves opening bracket and one member byte.
const CLASS_TAIL_SIZE_MAXIMUM = TAIL_SIZE_MAXIMUM - NONEMPTY_SIZE_MINIMUM

// CLASS_MATCH_TAIL_SIZE_MAXIMUM leaves opening bracket, member, and closing bracket.
const CLASS_MATCH_TAIL_SIZE_MAXIMUM = CLASS_TAIL_SIZE_MAXIMUM - NONEMPTY_SIZE_MINIMUM

// Error_Bad_Pattern keeps malformed shell grammar error stable.
var Error_Bad_Pattern = errors.New("syntax error in pattern")

// Boolean separates path decision coverage from unrelated booleans.
type Boolean bool

// Boolean_Invariants proves both path decision states occur.
func Boolean_Invariants(value Boolean, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "A path decision is true.").
		Ensure()
}

// Elements bounds even empty Join_Into inputs, because empty values still consume work.
type Elements []bytes.Text

// Elements_Invariants caps Join_Into work independently from output size.
func Elements_Invariants(value Elements, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), ELEMENT_COUNT_MINIMUM, ELEMENT_COUNT_MAXIMUM).
		Ensure()
}

// Nonempty_Count excludes zero because empty lexical result becomes dot.
type Nonempty_Count int

// Nonempty_Count_Invariants proves mandatory dot through maximum path size.
func Nonempty_Count_Invariants(value Nonempty_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), NONEMPTY_SIZE_MINIMUM, PATH_SIZE_MAXIMUM).
		Ensure()
}

// Boundary gives generic byte boundary host pathname limits.
type Boundary bytes.Boundary

// Boundary_Invariants proves every written path count stays inside host pathname capacity.
func Boundary_Invariants(value Boundary, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), PATH_SIZE_MINIMUM, PATH_SIZE_MAXIMUM).
		Ensure()
}

// Directory_Count excludes zero because cleaned directory always contains dot or root.
type Directory_Count int

// Directory_Count_Invariants excludes full size because directory loses final element or slash.
func Directory_Count_Invariants(value Directory_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), NONEMPTY_SIZE_MINIMUM, DIRECTORY_SIZE_MAXIMUM).
		Ensure()
}

// Text gives generic bounded text host pathname limits.
type Text bytes.Text

// Text_Invariants proves clipped intermediate views stay inside path bound.
func Text_Invariants(value Text, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), PATH_SIZE_MINIMUM, PATH_SIZE_MAXIMUM).
		Ensure()
}

// Nonempty_Text excludes loop state after Match exhausts pattern.
type Nonempty_Text string

// Nonempty_Text_Invariants proves Match scans only active pattern tails.
func Nonempty_Text_Invariants(value Nonempty_Text, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), NONEMPTY_SIZE_MINIMUM, PATH_SIZE_MAXIMUM).
		Ensure()
}

// Tail_Text preserves proof that scan consumed at least one byte.
type Tail_Text string

// Tail_Text_Invariants subtracts one consumed byte from maximum path.
func Tail_Text_Invariants(value Tail_Text, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), PATH_SIZE_MINIMUM, TAIL_SIZE_MAXIMUM).
		Ensure()
}

// Class_Text preserves proof that opening bracket was consumed.
type Class_Text string

// Class_Text_Invariants subtracts opening bracket consumed by matcher.
func Class_Text_Invariants(value Class_Text, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), PATH_SIZE_MINIMUM, TAIL_SIZE_MAXIMUM).
		Ensure()
}

// Class_Tail preserves proof that opening bracket and one member were consumed.
type Class_Tail string

// Class_Tail_Invariants subtracts opening bracket and one consumed member byte.
func Class_Tail_Invariants(value Class_Tail, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), PATH_SIZE_MINIMUM, CLASS_TAIL_SIZE_MAXIMUM).
		Ensure()
}

// Class_Match_Tail preserves proof that one complete character class was consumed.
type Class_Match_Tail string

// Class_Match_Tail_Invariants subtracts shortest complete class grammar.
func Class_Match_Tail_Invariants(value Class_Match_Tail, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), PATH_SIZE_MINIMUM, CLASS_MATCH_TAIL_SIZE_MAXIMUM).
		Ensure()
}

// Clean_Into uses caller storage because returning cleaned text would allocate.
func Clean_Into(destination bytes.Slice, value Text) (count Nonempty_Count) {
	defer func() { Nonempty_Count_Invariants(count, "clean_into.count") }()
	bytes.Slice_Invariants(destination, "clean_into.destination")
	Text_Invariants(value, "clean_into.value")
	var storage [PATH_SIZE_MAXIMUM]byte
	count = clean_text(&storage, value)
	if len(destination) < int(count) {
		panic("path: destination too small")
	}
	copy(destination, storage[:int(count)])
	return count
}

// Split returns input views because component copies would allocate.
func Split(value Text) (directory Text, file Text) {
	defer func() {
		Text_Invariants(directory, "split.directory")
		Text_Invariants(file, "split.file")
	}()
	Text_Invariants(value, "split.value")
	index := len(value) - 1
	for index >= 0 && value[index] != '/' {
		index--
	}
	return value[:index+1], value[index+1:]
}

// Join_Into uses caller storage because joined output has no source view.
func Join_Into(destination bytes.Slice, elements Elements) (count Boundary) {
	defer func() { Boundary_Invariants(count, "join_into.count") }()
	bytes.Slice_Invariants(destination, "join_into.destination")
	Elements_Invariants(elements, "join_into.elements")
	var storage [PATH_SIZE_MAXIMUM]byte
	raw_count := 0
	for _, element := range elements {
		Text_Invariants(Text(element), "join_into.element")
		if raw_count == 0 {
			if element == "" {
				continue
			}
		} else {
			if raw_count == len(storage) {
				panic("path: joined path outside bounds")
			}
			storage[raw_count] = '/'
			raw_count++
		}
		if len(element) > len(storage)-raw_count {
			panic("path: joined path outside bounds")
		}
		raw_count += copy(storage[raw_count:], element)
	}
	if raw_count == 0 {
		return 0
	}
	count = Boundary(clean_joined(&storage, Nonempty_Count(raw_count)))
	if len(destination) < int(count) {
		panic("path: destination too small")
	}
	copy(destination, storage[:count])
	return count
}

// Extension returns source view because suffix already exists in input.
func Extension(value Text) (extension Text) {
	defer func() { Text_Invariants(extension, "extension.extension") }()
	Text_Invariants(value, "extension.value")
	for index := len(value) - 1; index >= 0 && value[index] != '/'; index-- {
		if value[index] == '.' {
			return value[index:]
		}
	}
	return ""
}

// Base returns source view where possible because component copies would allocate.
func Base(value Text) (base Text) {
	defer func() { Text_Invariants(base, "base.base") }()
	Text_Invariants(value, "base.value")
	if value == "" {
		return "."
	}
	for len(value) > 0 && value[len(value)-1] == '/' {
		value = value[:len(value)-1]
	}
	index := len(value) - 1
	for index >= 0 && value[index] != '/' {
		index--
	}
	value = value[index+1:]
	if value == "" {
		return "/"
	}
	return value
}

// Is_Absolute stays lexical so no filesystem dependency enters path.
func Is_Absolute(value Text) (absolute Boolean) {
	defer func() { Boolean_Invariants(absolute, "is_absolute.absolute") }()
	Text_Invariants(value, "is_absolute.value")
	return len(value) > 0 && value[0] == '/'
}

// Directory_Into uses caller storage because directory cleaning cannot always return input view.
func Directory_Into(destination bytes.Slice, value Text) (count Directory_Count) {
	defer func() { Directory_Count_Invariants(count, "directory_into.count") }()
	bytes.Slice_Invariants(destination, "directory_into.destination")
	Text_Invariants(value, "directory_into.value")
	directory, _ := Split(value)
	var storage [PATH_SIZE_MAXIMUM]byte
	count = Directory_Count(clean_text(&storage, directory))
	if len(destination) < int(count) {
		panic("path: destination too small")
	}
	copy(destination, storage[:int(count)])
	return count
}

// Match scans bounded input directly because compiled matcher state would allocate.
func Match(pattern Text, name Text) (matched Boolean, err error) {
	defer func() { Boolean_Invariants(matched, "match.matched") }()
	Text_Invariants(pattern, "match.pattern")
	Text_Invariants(name, "match.name")
	pattern_tail := pattern
	name_tail := name

Pattern:
	for len(pattern_tail) > 0 {
		star, chunk, rest := scan_chunk(Nonempty_Text(pattern_tail))
		pattern_tail = Text(rest)
		if star {
			if chunk == "" {
				// Star cannot cross slash.
				for index := 0; index < len(name_tail); index++ {
					if name_tail[index] == '/' {
						return false, nil
					}
				}
				return true, nil
			}
		}
		tail, okay, match_error := match_chunk(chunk, name_tail)
		if okay {
			if len(tail) == 0 {
				name_tail = Text(tail)
				continue
			}
			if len(pattern_tail) > 0 {
				name_tail = Text(tail)
				continue
			}
		}
		if match_error != nil {
			return false, match_error
		}
		if star {
			for index := 0; index < len(name_tail) && name_tail[index] != '/'; index++ {
				tail, okay, match_error = match_chunk(chunk, name_tail[index+1:])
				if okay {
					if len(pattern_tail) == 0 {
						if len(tail) > 0 {
							continue
						}
					}
					name_tail = Text(tail)
					continue Pattern
				}
				if match_error != nil {
					return false, match_error
				}
			}
		}
		// Remaining grammar still needs validation because malformed pattern outranks
		// mismatch.
		for len(pattern_tail) > 0 {
			_, chunk, rest = scan_chunk(Nonempty_Text(pattern_tail))
			pattern_tail = Text(rest)
			if _, _, match_error = match_chunk(chunk, ""); match_error != nil {
				return false, match_error
			}
		}
		return false, nil
	}
	return Boolean(len(name_tail) == 0), nil
}

func clean_text(storage *[PATH_SIZE_MAXIMUM]byte, value Text) (count Nonempty_Count) {
	defer func() { Nonempty_Count_Invariants(count, "clean_text.count") }()
	Text_Invariants(value, "clean_text.value")
	if value == "" {
		storage[0] = '.'
		return 1
	}
	copy(storage[:], value)
	return clean_joined(storage, Nonempty_Count(len(value)))
}

func clean_joined(storage *[PATH_SIZE_MAXIMUM]byte, size Nonempty_Count) (count Nonempty_Count) {
	defer func() { Nonempty_Count_Invariants(count, "clean_joined.count") }()
	Nonempty_Count_Invariants(size, "clean_joined.size")
	rooted := storage[0] == '/'
	read, write, dot_dot := 0, 0, 0
	if rooted {
		read, write, dot_dot = 1, 1, 1
	}
	for read < int(size) {
		dot_element := false
		dot_dot_element := false
		if storage[read] == '.' {
			if read+1 == int(size) {
				dot_element = true
			} else if storage[read+1] == '/' {
				dot_element = true
			} else if storage[read+1] == '.' {
				if read+2 == int(size) {
					dot_dot_element = true
				} else if storage[read+2] == '/' {
					dot_dot_element = true
				}
			}
		}
		switch {
		case storage[read] == '/':
			read++
		case dot_element:
			read++
		case dot_dot_element:
			read += 2
			if write > dot_dot {
				write--
				for write > dot_dot && storage[write] != '/' {
					write--
				}
			} else if !rooted {
				// Root blocks parents; relative paths retain them.
				if write > 0 {
					storage[write] = '/'
					write++
				}
				storage[write] = '.'
				storage[write+1] = '.'
				write += 2
				dot_dot = write
			}
		default:
			if rooted {
				if write != 1 {
					storage[write] = '/'
					write++
				}
			} else if write != 0 {
				storage[write] = '/'
				write++
			}
			for read < int(size) && storage[read] != '/' {
				storage[write] = storage[read]
				write++
				read++
			}
		}
	}
	if write == 0 {
		storage[0] = '.'
		write = 1
	}
	return Nonempty_Count(write)
}

func scan_chunk(
	pattern Nonempty_Text,
) (star Boolean, chunk Text, rest Tail_Text) {
	defer func() {
		Boolean_Invariants(star, "scan_chunk.star")
		Text_Invariants(chunk, "scan_chunk.chunk")
		Tail_Text_Invariants(rest, "scan_chunk.rest")
	}()
	Nonempty_Text_Invariants(pattern, "scan_chunk.pattern")
	for len(pattern) > 0 && pattern[0] == '*' {
		pattern = pattern[1:]
		star = true
	}
	in_range := false
	for index := 0; index < len(pattern); index++ {
		switch pattern[index] {
		case '\\':
			if index+1 < len(pattern) {
				index++
			}
		case '[':
			in_range = true
		case ']':
			in_range = false
		case '*':
			if !in_range {
				return star, Text(pattern[:index]), Tail_Text(pattern[index:])
			}
		}
	}
	return star, Text(pattern), ""
}

func match_chunk(
	chunk Text, name Text,
) (rest Tail_Text, okay Boolean, err error) {
	defer func() {
		Tail_Text_Invariants(rest, "match_chunk.rest")
		Boolean_Invariants(okay, "match_chunk.okay")
	}()
	Text_Invariants(chunk, "match_chunk.chunk")
	Text_Invariants(name, "match_chunk.name")
	failed := false
	for len(chunk) > 0 {
		failed = failed || len(name) == 0
		switch chunk[0] {
		case '[':
			var character utf8.Decoded_Character
			if !failed {
				var size utf8.Decoded_Size
				character, size = utf8.Decode_Character_Text(utf8.Text(name))
				name = name[size:]
			}
			class_tail, member, negated, class_error := match_class(
				Class_Text(chunk[1:]), character,
			)
			if class_error != nil {
				return "", false, class_error
			}
			chunk = Text(class_tail)
			failed = failed || member == negated
		case '?':
			if !failed {
				failed = name[0] == '/'
				_, size := utf8.Decode_Character_Text(utf8.Text(name))
				name = name[size:]
			}
			chunk = chunk[1:]
		case '\\':
			chunk = chunk[1:]
			if len(chunk) == 0 {
				return "", false, Error_Bad_Pattern
			}
			if !failed {
				failed = chunk[0] != name[0]
				name = name[1:]
			}
			chunk = chunk[1:]
		default:
			if !failed {
				failed = chunk[0] != name[0]
				name = name[1:]
			}
			chunk = chunk[1:]
		}
	}
	if failed {
		return "", false, nil
	}
	return Tail_Text(name), true, nil
}

func match_class(
	chunk Class_Text, character utf8.Decoded_Character,
) (rest Class_Match_Tail, member Boolean, negated Boolean, err error) {
	defer func() {
		Class_Match_Tail_Invariants(rest, "match_class.rest")
		Boolean_Invariants(member, "match_class.member")
		Boolean_Invariants(negated, "match_class.negated")
	}()
	Class_Text_Invariants(chunk, "match_class.chunk")
	utf8.Decoded_Character_Invariants(character, "match_class.character")
	if len(chunk) > 0 {
		if chunk[0] == '^' {
			negated = true
			chunk = chunk[1:]
		}
	}
	range_count := 0
	// Extra pass turns missing closing bracket into stable syntax error without sentinel
	// storage.
	for range len(chunk) + 1 {
		if len(chunk) > 0 {
			if chunk[0] == ']' {
				if range_count > 0 {
					return Class_Match_Tail(chunk[1:]), member, negated, nil
				}
			}
		}
		low, tail, escape_error := escaped_character(chunk)
		if escape_error != nil {
			return "", false, false, escape_error
		}
		chunk = Class_Text(tail)
		high := low
		if chunk[0] == '-' {
			high, tail, escape_error = escaped_character(Class_Text(chunk[1:]))
			chunk = Class_Text(tail)
			if escape_error != nil {
				return "", false, false, escape_error
			}
		}
		member = member || low <= character && character <= high
		range_count++
	}
	return "", false, false, Error_Bad_Pattern
}

func escaped_character(
	chunk Class_Text,
) (character utf8.Decoded_Character, rest Class_Tail, err error) {
	defer func() {
		utf8.Decoded_Character_Invariants(character, "escaped_character.character")
		Class_Tail_Invariants(rest, "escaped_character.rest")
	}()
	Class_Text_Invariants(chunk, "escaped_character.chunk")
	if len(chunk) == 0 {
		return 0, "", Error_Bad_Pattern
	}
	if chunk[0] == '-' {
		return 0, "", Error_Bad_Pattern
	}
	if chunk[0] == ']' {
		return 0, "", Error_Bad_Pattern
	}
	if chunk[0] == '\\' {
		chunk = chunk[1:]
		if len(chunk) == 0 {
			return 0, "", Error_Bad_Pattern
		}
	}
	character, size := utf8.Decode_Character_Text(utf8.Text(chunk))
	if character == utf8.REPLACEMENT_CHARACTER {
		if size == 1 {
			return 0, "", Error_Bad_Pattern
		}
	}
	rest = Class_Tail(chunk[size:])
	if len(rest) == 0 {
		return 0, "", Error_Bad_Pattern
	}
	return character, rest, nil
}
