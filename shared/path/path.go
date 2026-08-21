// Package path keeps slash-separated paths bounded. Caller-owned output avoids allocation.
package path

import (
	"errors"

	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/simulation/aver/default"
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

// MATCH_RESULT_INDEX locates sole bounded helper result.
const MATCH_RESULT_INDEX = 0

// MATCH_RESULT_COUNT gives internal helpers one bounded result slot.
const MATCH_RESULT_COUNT = MATCH_RESULT_INDEX + NONEMPTY_SIZE_MINIMUM

// MATCH_STAR_INDEX_ABSENT keeps simple classification distinct from first-byte star.
const MATCH_STAR_INDEX_ABSENT = -NONEMPTY_SIZE_MINIMUM

// MATCH_CLASSIFICATION_STAR_INDEX locates simple star position.
const MATCH_CLASSIFICATION_STAR_INDEX = 0

// MATCH_CLASSIFICATION_QUESTION_INDEX follows star position.
const MATCH_CLASSIFICATION_QUESTION_INDEX = MATCH_CLASSIFICATION_STAR_INDEX + 1

// MATCH_CLASSIFICATION_GENERIC_INDEX follows question presence.
const MATCH_CLASSIFICATION_GENERIC_INDEX = MATCH_CLASSIFICATION_QUESTION_INDEX + 1

// MATCH_CLASSIFICATION_COUNT bounds classification state.
const MATCH_CLASSIFICATION_COUNT = MATCH_CLASSIFICATION_GENERIC_INDEX + 1

// MATCH_CLASSIFICATION_GENERIC_ABSENT keeps literal and single wildcard paths direct.
const MATCH_CLASSIFICATION_GENERIC_ABSENT = 0

// MATCH_CLASSIFICATION_GENERIC_PATTERN routes interacting wildcard operators.
const MATCH_CLASSIFICATION_GENERIC_PATTERN = MATCH_CLASSIFICATION_GENERIC_ABSENT + 1

// MATCH_CLASSIFICATION_GENERIC_CLASS gives one class a direct bounded parse first.
const MATCH_CLASSIFICATION_GENERIC_CLASS = MATCH_CLASSIFICATION_GENERIC_PATTERN + 1

// MATCH_CLASSIFICATION_GENERIC_ESCAPE gives literal escapes a direct bounded scan first.
const MATCH_CLASSIFICATION_GENERIC_ESCAPE = MATCH_CLASSIFICATION_GENERIC_CLASS + 1

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
func Boolean_Invariants(value Boolean, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "A path decision is true.").
		Ensure()
}

// Elements bounds even empty Join_Into inputs, because empty values still consume work.
type Elements []bytes.Text

// Elements_Invariants caps Join_Into work independently from output size.
func Elements_Invariants(value Elements, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), ELEMENT_COUNT_MINIMUM, ELEMENT_COUNT_MAXIMUM).
		Ensure()
}

// Nonempty_Count excludes zero because empty lexical result becomes dot.
type Nonempty_Count int

// Nonempty_Count_Invariants proves mandatory dot through maximum path size.
func Nonempty_Count_Invariants(value Nonempty_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), NONEMPTY_SIZE_MINIMUM, PATH_SIZE_MAXIMUM).
		Ensure()
}

// Boundary gives generic byte boundary host pathname limits.
type Boundary bytes.Boundary

// Boundary_Invariants proves every written path count stays inside host pathname capacity.
func Boundary_Invariants(value Boundary, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), PATH_SIZE_MINIMUM, PATH_SIZE_MAXIMUM).
		Ensure()
}

// Directory_Count excludes zero because cleaned directory always contains dot or root.
type Directory_Count int

// Directory_Count_Invariants excludes full size because directory loses final element or slash.
func Directory_Count_Invariants(value Directory_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), NONEMPTY_SIZE_MINIMUM, DIRECTORY_SIZE_MAXIMUM).
		Ensure()
}

// Text gives generic bounded text host pathname limits.
type Text bytes.Text

// Text_Invariants proves clipped intermediate views stay inside path bound.
func Text_Invariants(value Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), PATH_SIZE_MINIMUM, PATH_SIZE_MAXIMUM).
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
	if pattern == "*" {
		result := match_trailing_star([MATCH_RESULT_COUNT]Text{name})
		return result[MATCH_RESULT_INDEX], nil
	}
	classification := classify_match(
		[MATCH_RESULT_COUNT]Text{pattern},
	)
	star_index := classification[MATCH_CLASSIFICATION_STAR_INDEX]
	question := classification[MATCH_CLASSIFICATION_QUESTION_INDEX] != 0
	generic := classification[MATCH_CLASSIFICATION_GENERIC_INDEX]
	if star_index >= 0 {
		if question {
			generic = MATCH_CLASSIFICATION_GENERIC_PATTERN
		}
	}
	if generic == MATCH_CLASSIFICATION_GENERIC_CLASS {
		result, handled, match_error := match_simple_class(
			[MATCH_RESULT_COUNT]Text{pattern}, [MATCH_RESULT_COUNT]Text{name},
		)
		if handled[MATCH_RESULT_INDEX] {
			return result[MATCH_RESULT_INDEX], match_error
		}
	}
	if generic == MATCH_CLASSIFICATION_GENERIC_ESCAPE {
		result, handled, match_error := match_simple_escape(
			[MATCH_RESULT_COUNT]Text{pattern}, [MATCH_RESULT_COUNT]Text{name},
		)
		if handled[MATCH_RESULT_INDEX] {
			return result[MATCH_RESULT_INDEX], match_error
		}
	}
	if generic != MATCH_CLASSIFICATION_GENERIC_ABSENT {
		result, match_error := match_pattern(
			[MATCH_RESULT_COUNT]Text{pattern}, [MATCH_RESULT_COUNT]Text{name},
		)
		return result[MATCH_RESULT_INDEX], match_error
	}
	if star_index >= 0 {
		prefix := pattern[:star_index]
		suffix := pattern[star_index+NONEMPTY_SIZE_MINIMUM:]
		if len(name) < len(pattern)-NONEMPTY_SIZE_MINIMUM {
			return false, nil
		}
		if name[:len(prefix)] != prefix {
			return false, nil
		}
		suffix_start := len(name) - len(suffix)
		if name[suffix_start:] != suffix {
			return false, nil
		}
		for count := len(prefix); count < suffix_start; count++ {
			if name[count] == '/' {
				return false, nil
			}
		}
		return true, nil
	}
	if question {
		result := match_simple_question(
			[MATCH_RESULT_COUNT]Text{pattern}, [MATCH_RESULT_COUNT]Text{name},
		)
		return result[MATCH_RESULT_INDEX], nil
	}
	return pattern == name, nil
}

func match_pattern(
	pattern_value [MATCH_RESULT_COUNT]Text, name_value [MATCH_RESULT_COUNT]Text,
) (matched [MATCH_RESULT_COUNT]Boolean, err error) {
	pattern := pattern_value[MATCH_RESULT_INDEX]
	name := name_value[MATCH_RESULT_INDEX]
Pattern:
	for len(pattern) > 0 {
		star_result, chunk_result, pattern_result := scan_match_chunk(
			[MATCH_RESULT_COUNT]Text{pattern},
		)
		star := star_result[MATCH_RESULT_INDEX]
		chunk := chunk_result[MATCH_RESULT_INDEX]
		pattern = pattern_result[MATCH_RESULT_INDEX]
		if star {
			if chunk == "" {
				result := match_trailing_star([MATCH_RESULT_COUNT]Text{name})
				return result, nil
			}
		}
		rest_result, okay_result, chunk_error := match_text_chunk(
			[MATCH_RESULT_COUNT]Text{chunk}, [MATCH_RESULT_COUNT]Text{name},
		)
		rest := rest_result[MATCH_RESULT_INDEX]
		okay := okay_result[MATCH_RESULT_INDEX]
		if okay {
			if len(rest) == 0 {
				name = rest
				continue
			}
			if len(pattern) > 0 {
				name = rest
				continue
			}
		}
		if chunk_error != nil {
			return [MATCH_RESULT_COUNT]Boolean{false}, chunk_error
		}
		if star {
			for index := 0; index < len(name); index++ {
				if name[index] == '/' {
					break
				}
				rest_result, okay_result, chunk_error = match_text_chunk(
					[MATCH_RESULT_COUNT]Text{chunk},
					[MATCH_RESULT_COUNT]Text{name[index+1:]},
				)
				rest = rest_result[MATCH_RESULT_INDEX]
				okay = okay_result[MATCH_RESULT_INDEX]
				if okay {
					if len(pattern) == 0 {
						if len(rest) > 0 {
							continue
						}
					}
					name = rest
					continue Pattern
				}
				if chunk_error != nil {
					return [MATCH_RESULT_COUNT]Boolean{false}, chunk_error
				}
			}
		}
		validation_input := [MATCH_RESULT_COUNT]Text{pattern}
		validation_error := match_validate_rest(validation_input)
		if validation_error != nil {
			return [MATCH_RESULT_COUNT]Boolean{false}, validation_error
		}
		return [MATCH_RESULT_COUNT]Boolean{false}, nil
	}
	return [MATCH_RESULT_COUNT]Boolean{Boolean(len(name) == 0)}, nil
}

func classify_match(pattern_value [MATCH_RESULT_COUNT]Text) (
	classification [MATCH_CLASSIFICATION_COUNT]int,
) {
	pattern := pattern_value[MATCH_RESULT_INDEX]
	star_index := MATCH_STAR_INDEX_ABSENT
	question := 0
	for index := 0; index < len(pattern); index++ {
		switch pattern[index] {
		case '\\':
			return [MATCH_CLASSIFICATION_COUNT]int{
				star_index, question, MATCH_CLASSIFICATION_GENERIC_ESCAPE,
			}
		case '[':
			return [MATCH_CLASSIFICATION_COUNT]int{
				star_index, question, MATCH_CLASSIFICATION_GENERIC_CLASS,
			}
		case '?':
			question = NONEMPTY_SIZE_MINIMUM
		case '*':
			if star_index >= 0 {
				return [MATCH_CLASSIFICATION_COUNT]int{
					star_index, question, MATCH_CLASSIFICATION_GENERIC_PATTERN,
				}
			}
			star_index = index
		}
	}
	return [MATCH_CLASSIFICATION_COUNT]int{
		star_index, question, MATCH_CLASSIFICATION_GENERIC_ABSENT,
	}
}

func match_simple_escape(
	pattern_value, name_value [MATCH_RESULT_COUNT]Text,
) (matched, handled [MATCH_RESULT_COUNT]Boolean, err error) {
	pattern := pattern_value[MATCH_RESULT_INDEX]
	name := name_value[MATCH_RESULT_INDEX]
	name_index := 0
	failed := false
	for pattern_index := 0; pattern_index < len(pattern); pattern_index++ {
		character := pattern[pattern_index]
		if character == '\\' {
			pattern_index++
			if pattern_index == len(pattern) {
				return [MATCH_RESULT_COUNT]Boolean{false},
					[MATCH_RESULT_COUNT]Boolean{true}, Error_Bad_Pattern
			}
			character = pattern[pattern_index]
		} else {
			switch character {
			case '*', '?', '[':
				return [MATCH_RESULT_COUNT]Boolean{false},
					[MATCH_RESULT_COUNT]Boolean{false}, nil
			}
		}
		if name_index == len(name) {
			failed = true
			continue
		}
		failed = failed || character != name[name_index]
		name_index++
	}
	matched_value := Boolean(name_index == len(name))
	matched_value = matched_value && Boolean(!failed)
	return [MATCH_RESULT_COUNT]Boolean{matched_value},
		[MATCH_RESULT_COUNT]Boolean{true}, nil
}

func match_simple_class(
	pattern_value, name_value [MATCH_RESULT_COUNT]Text,
) (matched, handled [MATCH_RESULT_COUNT]Boolean, err error) {
	pattern := pattern_value[MATCH_RESULT_INDEX]
	name := name_value[MATCH_RESULT_INDEX]
	class_start := 0
	for pattern[class_start] != '[' {
		class_start++
	}
	candidate := Text("")
	if class_start < len(name) {
		candidate = name[class_start:]
	}
	class_rest, member_result, negated_result, class_error := match_class(
		[MATCH_RESULT_COUNT]Text{pattern[class_start+NONEMPTY_SIZE_MINIMUM:]},
		[MATCH_RESULT_COUNT]Text{candidate},
	)
	if class_error != nil {
		return [MATCH_RESULT_COUNT]Boolean{false},
			[MATCH_RESULT_COUNT]Boolean{true}, class_error
	}
	rest := class_rest[MATCH_RESULT_INDEX]
	for index := 0; index < len(rest); index++ {
		switch rest[index] {
		case '*', '?', '\\', '[':
			return [MATCH_RESULT_COUNT]Boolean{false},
				[MATCH_RESULT_COUNT]Boolean{false}, nil
		}
	}
	if len(name) <= class_start {
		return [MATCH_RESULT_COUNT]Boolean{false}, [MATCH_RESULT_COUNT]Boolean{true}, nil
	}
	if name[:class_start] != pattern[:class_start] {
		return [MATCH_RESULT_COUNT]Boolean{false}, [MATCH_RESULT_COUNT]Boolean{true}, nil
	}
	candidate_count := int(utf8.CHARACTER_SIZE_MINIMUM)
	if name[class_start] >= byte(utf8.CHARACTER_SELF) {
		_, decoded_size := utf8.Decode_Character_Text(utf8.Text(name[class_start:]))
		candidate_count = int(decoded_size)
	}
	if member_result[MATCH_RESULT_INDEX] == negated_result[MATCH_RESULT_INDEX] {
		return [MATCH_RESULT_COUNT]Boolean{false}, [MATCH_RESULT_COUNT]Boolean{true}, nil
	}
	name_rest := name[class_start+candidate_count:]
	return [MATCH_RESULT_COUNT]Boolean{Boolean(name_rest == rest)},
		[MATCH_RESULT_COUNT]Boolean{true}, nil
}

func match_simple_question(
	pattern_value [MATCH_RESULT_COUNT]Text,
	name_value [MATCH_RESULT_COUNT]Text,
) (matched [MATCH_RESULT_COUNT]Boolean) {
	pattern := pattern_value[MATCH_RESULT_INDEX]
	name := name_value[MATCH_RESULT_INDEX]
	pattern_index := 0
	name_index := 0
	for pattern_index < len(pattern) {
		if name_index == len(name) {
			return [MATCH_RESULT_COUNT]Boolean{false}
		}
		if pattern[pattern_index] == '?' {
			if name[name_index] == '/' {
				return [MATCH_RESULT_COUNT]Boolean{false}
			}
			size := int(utf8.CHARACTER_SIZE_MINIMUM)
			if name[name_index] >= byte(utf8.CHARACTER_SELF) {
				_, decoded_size := utf8.Decode_Character_Text(
					utf8.Text(name[name_index:]),
				)
				size = int(decoded_size)
			}
			name_index += size
			pattern_index++
			continue
		}
		if pattern[pattern_index] != name[name_index] {
			return [MATCH_RESULT_COUNT]Boolean{false}
		}
		pattern_index++
		name_index++
	}
	return [MATCH_RESULT_COUNT]Boolean{Boolean(name_index == len(name))}
}

func match_trailing_star(
	name_value [MATCH_RESULT_COUNT]Text,
) (matched [MATCH_RESULT_COUNT]Boolean) {
	name := name_value[MATCH_RESULT_INDEX]
	for index := 0; index < len(name); index++ {
		if name[index] == '/' {
			return [MATCH_RESULT_COUNT]Boolean{false}
		}
	}
	return [MATCH_RESULT_COUNT]Boolean{true}
}

func match_validate_rest(pattern_value [MATCH_RESULT_COUNT]Text) (err error) {
	pattern := pattern_value[MATCH_RESULT_INDEX]
	for len(pattern) > 0 {
		_, chunk_result, rest_result := scan_match_chunk(
			[MATCH_RESULT_COUNT]Text{pattern},
		)
		pattern = rest_result[MATCH_RESULT_INDEX]
		_, _, chunk_error := match_text_chunk(
			chunk_result, [MATCH_RESULT_COUNT]Text{""},
		)
		if chunk_error != nil {
			return chunk_error
		}
	}
	return nil
}

func scan_match_chunk(pattern_value [MATCH_RESULT_COUNT]Text) (
	star [MATCH_RESULT_COUNT]Boolean,
	chunk [MATCH_RESULT_COUNT]Text,
	rest [MATCH_RESULT_COUNT]Text,
) {
	pattern := pattern_value[MATCH_RESULT_INDEX]
	star_value := Boolean(false)
	for len(pattern) > 0 {
		if pattern[0] != '*' {
			break
		}
		pattern = pattern[1:]
		star_value = true
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
				return [MATCH_RESULT_COUNT]Boolean{star_value},
					[MATCH_RESULT_COUNT]Text{pattern[:index]},
					[MATCH_RESULT_COUNT]Text{pattern[index:]}
			}
		}
	}
	return [MATCH_RESULT_COUNT]Boolean{star_value},
		[MATCH_RESULT_COUNT]Text{pattern}, [MATCH_RESULT_COUNT]Text{""}
}

func match_text_chunk(chunk_value, name_value [MATCH_RESULT_COUNT]Text) (
	rest [MATCH_RESULT_COUNT]Text, okay [MATCH_RESULT_COUNT]Boolean, err error,
) {
	chunk := chunk_value[MATCH_RESULT_INDEX]
	name := name_value[MATCH_RESULT_INDEX]
	failed := false
	for len(chunk) > 0 {
		if len(name) == 0 {
			failed = true
		}
		switch chunk[0] {
		case '[':
			candidate := Text("")
			if !failed {
				size := int(utf8.CHARACTER_SIZE_MINIMUM)
				if name[0] >= byte(utf8.CHARACTER_SELF) {
					text := utf8.Text(name)
					_, decoded_size := utf8.Decode_Character_Text(text)
					size = int(decoded_size)
				}
				candidate = name[:size]
				name = name[size:]
			}
			class_result, member_result, negated_result, err := match_class(
				[MATCH_RESULT_COUNT]Text{chunk[1:]},
				[MATCH_RESULT_COUNT]Text{candidate},
			)
			if err != nil {
				return [MATCH_RESULT_COUNT]Text{""},
					[MATCH_RESULT_COUNT]Boolean{false}, err
			}
			chunk = class_result[MATCH_RESULT_INDEX]
			if member_result[MATCH_RESULT_INDEX] == negated_result[MATCH_RESULT_INDEX] {
				failed = true
			}
		case '?':
			if !failed {
				failed = name[0] == '/'
				size := int(utf8.CHARACTER_SIZE_MINIMUM)
				if name[0] >= byte(utf8.CHARACTER_SELF) {
					text := utf8.Text(name)
					_, decoded_size := utf8.Decode_Character_Text(text)
					size = int(decoded_size)
				}
				name = name[size:]
			}
			chunk = chunk[1:]
		case '\\':
			chunk = chunk[1:]
			if len(chunk) == 0 {
				return [MATCH_RESULT_COUNT]Text{""},
					[MATCH_RESULT_COUNT]Boolean{false}, Error_Bad_Pattern
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
		return [MATCH_RESULT_COUNT]Text{""},
			[MATCH_RESULT_COUNT]Boolean{false}, nil
	}
	return [MATCH_RESULT_COUNT]Text{name}, [MATCH_RESULT_COUNT]Boolean{true}, nil
}

func match_character(
	value_result [MATCH_RESULT_COUNT]Text,
) (character [MATCH_RESULT_COUNT]utf8.Decoded_Character) {
	value := value_result[MATCH_RESULT_INDEX]
	if len(value) == 0 {
		return [MATCH_RESULT_COUNT]utf8.Decoded_Character{0}
	}
	if value[0] < byte(utf8.CHARACTER_SELF) {
		return [MATCH_RESULT_COUNT]utf8.Decoded_Character{
			utf8.Decoded_Character(value[0]),
		}
	}
	decoded, _ := utf8.Decode_Character_Text(utf8.Text(value))
	return [MATCH_RESULT_COUNT]utf8.Decoded_Character{decoded}
}

func match_class(
	chunk_result [MATCH_RESULT_COUNT]Text,
	candidate_result [MATCH_RESULT_COUNT]Text,
) (
	rest [MATCH_RESULT_COUNT]Text,
	member_result, negated_result [MATCH_RESULT_COUNT]Boolean,
	err error,
) {
	chunk := chunk_result[MATCH_RESULT_INDEX]
	candidate := match_character(candidate_result)[MATCH_RESULT_INDEX]
	negated := Boolean(len(chunk) > 0 && chunk[0] == '^')
	if negated {
		chunk = chunk[1:]
	}
	if len(chunk) == 0 {
		return [MATCH_RESULT_COUNT]Text{""}, [MATCH_RESULT_COUNT]Boolean{false},
			[MATCH_RESULT_COUNT]Boolean{false}, Error_Bad_Pattern
	}
	literal := len(chunk) >= 2 && chunk[1] == ']' && chunk[0] != '\\' && chunk[0] != '-'
	if literal {
		character := utf8.Decoded_Character(chunk[0])
		member := Boolean(candidate == character)
		return [MATCH_RESULT_COUNT]Text{chunk[2:]},
			[MATCH_RESULT_COUNT]Boolean{member},
			[MATCH_RESULT_COUNT]Boolean{negated}, nil
	}
	escaped_literal := len(chunk) == 3 && chunk[0] == '\\' && chunk[2] == ']'
	if escaped_literal {
		character := utf8.Decoded_Character(chunk[1])
		member := Boolean(candidate == character)
		return [MATCH_RESULT_COUNT]Text{""},
			[MATCH_RESULT_COUNT]Boolean{member},
			[MATCH_RESULT_COUNT]Boolean{negated}, nil
	}
	range_literal := len(chunk) >= 4 && chunk[1] == '-' && chunk[3] == ']'
	range_literal = range_literal && chunk[0] != '\\' && chunk[2] != '\\'
	if range_literal {
		low := utf8.Decoded_Character(chunk[0])
		high := utf8.Decoded_Character(chunk[2])
		member := Boolean(low <= candidate)
		member = member && Boolean(candidate <= high)
		return [MATCH_RESULT_COUNT]Text{chunk[4:]},
			[MATCH_RESULT_COUNT]Boolean{member},
			[MATCH_RESULT_COUNT]Boolean{negated}, nil
	}
	if len(chunk) == 4 {
		if chunk[3] == ']' {
			if chunk[0] == '\\' {
				member := Boolean(candidate == utf8.Decoded_Character(chunk[1]))
				second := utf8.Decoded_Character(chunk[2])
				member = member || Boolean(candidate == second)
				return [MATCH_RESULT_COUNT]Text{""},
					[MATCH_RESULT_COUNT]Boolean{member},
					[MATCH_RESULT_COUNT]Boolean{negated}, nil
			}
			if chunk[1] == '\\' {
				member := Boolean(candidate == utf8.Decoded_Character(chunk[0]))
				second := utf8.Decoded_Character(chunk[2])
				member = member || Boolean(candidate == second)
				return [MATCH_RESULT_COUNT]Text{""},
					[MATCH_RESULT_COUNT]Boolean{member},
					[MATCH_RESULT_COUNT]Boolean{negated}, nil
			}
		}
	}
	return match_class_ranges(
		[MATCH_RESULT_COUNT]Text{chunk}, candidate_result,
		[MATCH_RESULT_COUNT]Boolean{negated},
	)
}

func match_class_ranges(
	chunk_result, candidate_result [MATCH_RESULT_COUNT]Text,
	negated_result [MATCH_RESULT_COUNT]Boolean,
) (
	rest [MATCH_RESULT_COUNT]Text, member_result,
	negated_output [MATCH_RESULT_COUNT]Boolean, err error,
) {
	chunk := chunk_result[MATCH_RESULT_INDEX]
	candidate := match_character(candidate_result)[MATCH_RESULT_INDEX]
	negated := negated_result[MATCH_RESULT_INDEX]
	if len(chunk) >= 5 {
		if chunk[1] == '-' {
			if chunk[2] >= byte(utf8.CHARACTER_SELF) {
				high, size := utf8.Decode_Character_Text(utf8.Text(chunk[2:]))
				close_index := int(size) + 2
				if close_index < len(chunk) {
					if chunk[close_index] == ']' {
						low := utf8.Decoded_Character(chunk[0])
						member := Boolean(low <= candidate)
						member = member && Boolean(candidate <= high)
						tail := chunk[close_index+1:]
						return [MATCH_RESULT_COUNT]Text{tail},
							[MATCH_RESULT_COUNT]Boolean{member},
							[MATCH_RESULT_COUNT]Boolean{negated}, nil
					}
				}
			}
		}
	}
	member := Boolean(false)
	range_count := 0
	for range len(chunk) + NONEMPTY_SIZE_MINIMUM {
		if len(chunk) > 0 {
			if chunk[0] == ']' {
				if range_count > 0 {
					return [MATCH_RESULT_COUNT]Text{chunk[1:]},
						[MATCH_RESULT_COUNT]Boolean{member},
						[MATCH_RESULT_COUNT]Boolean{negated}, nil
				}
			}
		}
		escaped_input := [MATCH_RESULT_COUNT]Text{chunk}
		escaped_result, low_result, escaped_error := match_escaped_character(escaped_input)
		if escaped_error != nil {
			return [MATCH_RESULT_COUNT]Text{""},
				[MATCH_RESULT_COUNT]Boolean{false},
				[MATCH_RESULT_COUNT]Boolean{false}, escaped_error
		}
		chunk = escaped_result[MATCH_RESULT_INDEX]
		low_text := low_result[MATCH_RESULT_INDEX]
		low_character := match_character([MATCH_RESULT_COUNT]Text{low_text})
		low := low_character[MATCH_RESULT_INDEX]
		high := low
		if chunk[0] == '-' {
			high_input := [MATCH_RESULT_COUNT]Text{chunk[1:]}
			high_rest, high_result, high_error := match_escaped_character(high_input)
			if high_error != nil {
				return [MATCH_RESULT_COUNT]Text{""},
					[MATCH_RESULT_COUNT]Boolean{false},
					[MATCH_RESULT_COUNT]Boolean{false}, high_error
			}
			chunk = high_rest[MATCH_RESULT_INDEX]
			high_text := high_result[MATCH_RESULT_INDEX]
			high_character := match_character([MATCH_RESULT_COUNT]Text{high_text})
			high = high_character[MATCH_RESULT_INDEX]
		}
		if low <= candidate {
			if candidate <= high {
				member = true
			}
		}
		range_count++
	}
	return [MATCH_RESULT_COUNT]Text{""}, [MATCH_RESULT_COUNT]Boolean{false},
		[MATCH_RESULT_COUNT]Boolean{false}, Error_Bad_Pattern
}

func match_escaped_character(chunk_result [MATCH_RESULT_COUNT]Text) (
	rest_result [MATCH_RESULT_COUNT]Text,
	character_result [MATCH_RESULT_COUNT]Text,
	err error,
) {
	chunk := chunk_result[MATCH_RESULT_INDEX]
	size := int(utf8.CHARACTER_SIZE_MINIMUM)
	if len(chunk) == 0 {
		return [MATCH_RESULT_COUNT]Text{""},
			[MATCH_RESULT_COUNT]Text{""}, Error_Bad_Pattern
	}
	if chunk[0] == '-' {
		return [MATCH_RESULT_COUNT]Text{""},
			[MATCH_RESULT_COUNT]Text{""}, Error_Bad_Pattern
	}
	if chunk[0] == ']' {
		return [MATCH_RESULT_COUNT]Text{""},
			[MATCH_RESULT_COUNT]Text{""}, Error_Bad_Pattern
	}
	if chunk[0] == '\\' {
		chunk = chunk[1:]
		if len(chunk) == 0 {
			return [MATCH_RESULT_COUNT]Text{""},
				[MATCH_RESULT_COUNT]Text{""}, Error_Bad_Pattern
		}
	}
	decoded := utf8.Decoded_Character(chunk[0])
	if chunk[0] >= byte(utf8.CHARACTER_SELF) {
		decoded_character, decoded_size := utf8.Decode_Character_Text(utf8.Text(chunk))
		decoded = decoded_character
		size = int(decoded_size)
	}
	if decoded == utf8.REPLACEMENT_CHARACTER {
		if size == int(utf8.CHARACTER_SIZE_MINIMUM) {
			return [MATCH_RESULT_COUNT]Text{""},
				[MATCH_RESULT_COUNT]Text{""}, Error_Bad_Pattern
		}
	}
	character := chunk[:size]
	rest := chunk[size:]
	if len(rest) == 0 {
		return [MATCH_RESULT_COUNT]Text{""},
			[MATCH_RESULT_COUNT]Text{""}, Error_Bad_Pattern
	}
	return [MATCH_RESULT_COUNT]Text{rest}, [MATCH_RESULT_COUNT]Text{character}, nil
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
