// Package path keeps slash-separated paths bounded. Caller-owned output avoids allocation.
package path

import (
	"errors"

	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/sim/aver/default"
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

// MATCH_STAR_INDEX_ABSENT keeps simple classification distinct from first-byte star.
const MATCH_STAR_INDEX_ABSENT Match_Star_Index = -NONEMPTY_SIZE_MINIMUM

// MATCH_STAR_INDEX_MAXIMUM keeps final star inside bounded pattern.
const MATCH_STAR_INDEX_MAXIMUM = PATH_SIZE_MAXIMUM - NONEMPTY_SIZE_MINIMUM

// MATCH_PATTERN_SIZE_MINIMUM gives two operators enough bytes to interact.
const MATCH_PATTERN_SIZE_MINIMUM = 2

// MATCH_CLASSIFICATION_GENERIC_ABSENT keeps literal and single wildcard paths direct.
const MATCH_CLASSIFICATION_GENERIC_ABSENT Match_Generic = 0

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

// CLASS_RANGE_REST_SIZE_MAXIMUM leaves opening bracket, two members, and closing bracket.
const CLASS_RANGE_REST_SIZE_MAXIMUM = CLASS_MATCH_TAIL_SIZE_MAXIMUM - NONEMPTY_SIZE_MINIMUM

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

// Match_Star_Index identifies first star or explicit absence.
type Match_Star_Index int

// Match_Star_Index_Invariants spans absence through final pattern byte.
func Match_Star_Index_Invariants(value Match_Star_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), int(MATCH_STAR_INDEX_ABSENT), MATCH_STAR_INDEX_MAXIMUM).
		Ensure()
}

// Match_Generic selects matcher needed beyond literal, star, or question fast paths.
type Match_Generic uint8

// Match_Generic_Invariants covers every generic matcher class.
func Match_Generic_Invariants(value Match_Generic, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), uint8(MATCH_CLASSIFICATION_GENERIC_ABSENT),
			uint8(MATCH_CLASSIFICATION_GENERIC_PATTERN),
			uint8(MATCH_CLASSIFICATION_GENERIC_CLASS),
			uint8(MATCH_CLASSIFICATION_GENERIC_ESCAPE),
		).
		Ensure()
}

// Match_Text holds a fragment of one validated matcher input.
type Match_Text string

// Match_Text_Invariants spans every fragment of one validated matcher input.
func Match_Text_Invariants(value Match_Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), PATH_SIZE_MINIMUM, PATH_SIZE_MAXIMUM).
		Ensure()
}

// Match_Nonempty_Text holds a fragment known to contain matcher work.
type Match_Nonempty_Text string

// Match_Nonempty_Text_Invariants excludes exhausted matcher input.
func Match_Nonempty_Text_Invariants(value Match_Nonempty_Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), NONEMPTY_SIZE_MINIMUM, PATH_SIZE_MAXIMUM).
		Ensure()
}

// Match_Pattern_Text holds generic matcher input with interacting operators.
type Match_Pattern_Text string

// Match_Pattern_Text_Invariants requires the two bytes needed to interact.
func Match_Pattern_Text_Invariants(value Match_Pattern_Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), MATCH_PATTERN_SIZE_MINIMUM, PATH_SIZE_MAXIMUM).
		Ensure()
}

// Match_Tail holds matcher input after at least one consumed byte.
type Match_Tail string

// Match_Tail_Invariants accounts for the byte preceding every tail.
func Match_Tail_Invariants(value Match_Tail, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), PATH_SIZE_MINIMUM, TAIL_SIZE_MAXIMUM).
		Ensure()
}

// Match_Nonempty_Tail holds remaining matcher input after one consumed byte.
type Match_Nonempty_Tail string

// Match_Nonempty_Tail_Invariants excludes exhausted matcher tails.
func Match_Nonempty_Tail_Invariants(
	value Match_Nonempty_Tail, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), NONEMPTY_SIZE_MINIMUM, TAIL_SIZE_MAXIMUM).
		Ensure()
}

// Match_Class_Tail holds class input after opening bracket and one member byte.
type Match_Class_Tail string

// Match_Class_Tail_Invariants accounts for both consumed class bytes.
func Match_Class_Tail_Invariants(value Match_Class_Tail, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), PATH_SIZE_MINIMUM, CLASS_TAIL_SIZE_MAXIMUM).
		Ensure()
}

// Match_Class_Rest holds pattern input after one complete class.
type Match_Class_Rest string

// Match_Class_Rest_Invariants accounts for opening bracket, member, and closing bracket.
func Match_Class_Rest_Invariants(value Match_Class_Rest, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), PATH_SIZE_MINIMUM, CLASS_MATCH_TAIL_SIZE_MAXIMUM).
		Ensure()
}

// Match_Range_Rest holds pattern input after one class handled by the range parser.
type Match_Range_Rest string

// Match_Range_Rest_Invariants accounts for the range parser's two members and delimiters.
func Match_Range_Rest_Invariants(value Match_Range_Rest, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), PATH_SIZE_MINIMUM, CLASS_RANGE_REST_SIZE_MAXIMUM).
		Ensure()
}

// Match_Character_Text holds one encoded matcher character or failed empty output.
type Match_Character_Text string

// Match_Character_Text_Invariants spans malformed empty output through one encoded character.
func Match_Character_Text_Invariants(
	value Match_Character_Text, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), PATH_SIZE_MINIMUM, int(utf8.CHARACTER_SIZE_MAXIMUM),
		).
		Ensure()
}

// Clean_Storage holds one complete bounded path during normalization.
type Clean_Storage bytes.Slice

// Clean_Storage_Invariants preserves the scratch capacity required by every cleaner branch.
func Clean_Storage_Invariants(value Clean_Storage, _ aver.Namespace) {
	aver.Always(len(value) == PATH_SIZE_MAXIMUM, "Clean storage has exact path capacity.")
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
	count = clean_text(Clean_Storage(storage[:]), value)
	aver.Always(
		len(destination) >= int(count),
		"Path clean destination holds complete result.",
	)
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
			aver.Always(
				raw_count < len(storage),
				"Path join storage has room for separator.",
			)
			storage[raw_count] = '/'
			raw_count++
		}
		aver.Always(
			len(element) <= len(storage)-raw_count,
			"Path join storage has room for element.",
		)
		raw_count += copy(storage[raw_count:], element)
	}
	if raw_count == 0 {
		return 0
	}
	count = Boundary(clean_joined(Clean_Storage(storage[:]), Nonempty_Count(raw_count)))
	aver.Always(
		len(destination) >= int(count),
		"Path join destination holds complete result.",
	)
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
	count = Directory_Count(clean_text(Clean_Storage(storage[:]), directory))
	aver.Always(
		len(destination) >= int(count),
		"Path directory destination holds complete result.",
	)
	copy(destination, storage[:int(count)])
	return count
}

// Match scans bounded input directly because compiled matcher state would allocate.
func Match(pattern Text, name Text) (matched Boolean, err error) {
	defer func() { Boolean_Invariants(matched, "match.matched") }()
	Text_Invariants(pattern, "match.pattern")
	Text_Invariants(name, "match.name")
	matched, malformed := match(pattern, name)
	if malformed {
		return false, Error_Bad_Pattern
	}
	return matched, nil
}

func match(pattern Text, name Text) (matched Boolean, malformed Boolean) {
	defer func() {
		Boolean_Invariants(matched, "match_internal.matched")
		Boolean_Invariants(malformed, "match_internal.malformed")
	}()
	Text_Invariants(pattern, "match_internal.pattern")
	Text_Invariants(name, "match_internal.name")
	if pattern == "*" {
		return match_trailing_star(Match_Text(name)), false
	}
	star_index, question, generic := classify_match(pattern)
	if star_index >= 0 {
		if question {
			generic = MATCH_CLASSIFICATION_GENERIC_PATTERN
		}
	}
	if generic == MATCH_CLASSIFICATION_GENERIC_CLASS {
		result, handled, match_malformed := match_simple_class(
			Match_Nonempty_Text(pattern), Match_Text(name),
		)
		if handled {
			return result, match_malformed
		}
	}
	if generic == MATCH_CLASSIFICATION_GENERIC_ESCAPE {
		result, handled, match_malformed := match_simple_escape(
			Match_Nonempty_Text(pattern), Match_Text(name),
		)
		if handled {
			return result, match_malformed
		}
	}
	if generic != MATCH_CLASSIFICATION_GENERIC_ABSENT {
		return match_pattern(Match_Pattern_Text(pattern), Match_Text(name))
	}
	if star_index >= 0 {
		prefix := pattern[:star_index]
		suffix := pattern[star_index+NONEMPTY_SIZE_MINIMUM:]
		if len(name) < len(pattern)-NONEMPTY_SIZE_MINIMUM {
			return false, false
		}
		if name[:len(prefix)] != prefix {
			return false, false
		}
		suffix_start := len(name) - len(suffix)
		if name[suffix_start:] != suffix {
			return false, false
		}
		for count := len(prefix); count < suffix_start; count++ {
			if name[count] == '/' {
				return false, false
			}
		}
		return true, false
	}
	if question {
		return match_simple_question(Match_Nonempty_Text(pattern), Match_Text(name)), false
	}
	return pattern == name, false
}

func match_pattern(
	pattern Match_Pattern_Text, name Match_Text,
) (matched Boolean, malformed Boolean) {
	defer func() {
		Boolean_Invariants(matched, "match_pattern.matched")
		Boolean_Invariants(malformed, "match_pattern.malformed")
	}()
	Match_Pattern_Text_Invariants(pattern, "match_pattern.pattern")
	Match_Text_Invariants(name, "match_pattern.name")
	remaining_pattern := Match_Text(pattern)
Pattern:
	for len(remaining_pattern) > 0 {
		star, chunk, pattern_result := scan_match_chunk(
			Match_Nonempty_Text(remaining_pattern),
		)
		remaining_pattern = Match_Text(pattern_result)
		if star {
			if chunk == "" {
				return match_trailing_star(name), false
			}
		}
		rest, okay, chunk_malformed := match_text_chunk(chunk, name)
		if okay {
			if len(rest) == 0 {
				name = Match_Text(rest)
				continue
			}
			if len(remaining_pattern) > 0 {
				name = Match_Text(rest)
				continue
			}
		}
		if chunk_malformed {
			return false, true
		}
		if star {
			for index := 0; index < len(name); index++ {
				if name[index] == '/' {
					break
				}
				rest, okay, chunk_malformed = match_text_chunk(
					chunk, name[index+1:],
				)
				if okay {
					if len(remaining_pattern) == 0 {
						if len(rest) > 0 {
							continue
						}
					}
					name = Match_Text(rest)
					continue Pattern
				}
				if chunk_malformed {
					return false, true
				}
			}
		}
		invalid_rest := match_validate_rest(Match_Tail(remaining_pattern))
		if invalid_rest {
			return false, true
		}
		return false, false
	}
	return Boolean(len(name) == 0), false
}

func classify_match(
	pattern Text,
) (star_index Match_Star_Index, question Boolean, generic Match_Generic) {
	defer func() {
		Match_Star_Index_Invariants(star_index, "classify_match.star_index")
		Boolean_Invariants(question, "classify_match.question")
		Match_Generic_Invariants(generic, "classify_match.generic")
	}()
	Text_Invariants(pattern, "classify_match.pattern")
	star_index = MATCH_STAR_INDEX_ABSENT
	for index := 0; index < len(pattern); index++ {
		switch pattern[index] {
		case '\\':
			return star_index, question, MATCH_CLASSIFICATION_GENERIC_ESCAPE
		case '[':
			return star_index, question, MATCH_CLASSIFICATION_GENERIC_CLASS
		case '?':
			question = true
		case '*':
			if star_index >= 0 {
				return star_index, question, MATCH_CLASSIFICATION_GENERIC_PATTERN
			}
			star_index = Match_Star_Index(index)
		}
	}
	return star_index, question, MATCH_CLASSIFICATION_GENERIC_ABSENT
}

func match_simple_escape(
	pattern Match_Nonempty_Text, name Match_Text,
) (matched Boolean, handled Boolean, malformed Boolean) {
	defer func() {
		Boolean_Invariants(matched, "match_simple_escape.matched")
		Boolean_Invariants(handled, "match_simple_escape.handled")
		Boolean_Invariants(malformed, "match_simple_escape.malformed")
	}()
	Match_Nonempty_Text_Invariants(pattern, "match_simple_escape.pattern")
	Match_Text_Invariants(name, "match_simple_escape.name")
	name_index := 0
	failed := false
	for pattern_index := 0; pattern_index < len(pattern); pattern_index++ {
		character := pattern[pattern_index]
		if character == '\\' {
			pattern_index++
			if pattern_index == len(pattern) {
				return false, true, true
			}
			character = pattern[pattern_index]
		} else {
			switch character {
			case '*', '?', '[':
				return false, false, false
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
	return matched_value, true, false
}

func match_simple_class(
	pattern Match_Nonempty_Text, name Match_Text,
) (matched Boolean, handled Boolean, malformed Boolean) {
	defer func() {
		Boolean_Invariants(matched, "match_simple_class.matched")
		Boolean_Invariants(handled, "match_simple_class.handled")
		Boolean_Invariants(malformed, "match_simple_class.malformed")
	}()
	Match_Nonempty_Text_Invariants(pattern, "match_simple_class.pattern")
	Match_Text_Invariants(name, "match_simple_class.name")
	class_start := 0
	for pattern[class_start] != '[' {
		class_start++
	}
	candidate := Match_Text("")
	if class_start < len(name) {
		candidate = name[class_start:]
	}
	class_rest, member_result, negated_result, class_malformed := match_class(
		Match_Tail(pattern[class_start+NONEMPTY_SIZE_MINIMUM:]), candidate,
	)
	if class_malformed {
		return false, true, true
	}
	rest := Match_Text(class_rest)
	for index := 0; index < len(rest); index++ {
		switch rest[index] {
		case '*', '?', '\\', '[':
			return false, false, false
		}
	}
	if len(name) <= class_start {
		return false, true, false
	}
	if name[:class_start] != Match_Text(pattern[:class_start]) {
		return false, true, false
	}
	candidate_count := int(utf8.CHARACTER_SIZE_MINIMUM)
	if name[class_start] >= byte(utf8.CHARACTER_SELF) {
		_, decoded_size := utf8.Decode_Character_Text(utf8.Text(name[class_start:]))
		candidate_count = int(decoded_size)
	}
	if member_result == negated_result {
		return false, true, false
	}
	name_rest := name[class_start+candidate_count:]
	return Boolean(name_rest == rest), true, false
}

func match_simple_question(
	pattern Match_Nonempty_Text, name Match_Text,
) (matched Boolean) {
	defer func() { Boolean_Invariants(matched, "match_simple_question.matched") }()
	Match_Nonempty_Text_Invariants(pattern, "match_simple_question.pattern")
	Match_Text_Invariants(name, "match_simple_question.name")
	pattern_index := 0
	name_index := 0
	for pattern_index < len(pattern) {
		if name_index == len(name) {
			return false
		}
		if pattern[pattern_index] == '?' {
			if name[name_index] == '/' {
				return false
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
			return false
		}
		pattern_index++
		name_index++
	}
	return Boolean(name_index == len(name))
}

func match_trailing_star(name Match_Text) (matched Boolean) {
	defer func() { Boolean_Invariants(matched, "match_trailing_star.matched") }()
	Match_Text_Invariants(name, "match_trailing_star.name")
	for index := 0; index < len(name); index++ {
		if name[index] == '/' {
			return false
		}
	}
	return true
}

func match_validate_rest(pattern Match_Tail) (malformed Boolean) {
	defer func() { Boolean_Invariants(malformed, "match_validate_rest.malformed") }()
	Match_Tail_Invariants(pattern, "match_validate_rest.pattern")
	remainder := Match_Text(pattern)
	for len(remainder) > 0 {
		_, chunk, rest := scan_match_chunk(Match_Nonempty_Text(remainder))
		remainder = Match_Text(rest)
		_, _, chunk_malformed := match_text_chunk(chunk, "")
		if chunk_malformed {
			return true
		}
	}
	return false
}

func scan_match_chunk(
	pattern Match_Nonempty_Text,
) (star Boolean, chunk Match_Text, rest Match_Tail) {
	defer func() {
		Boolean_Invariants(star, "scan_match_chunk.star")
		Match_Text_Invariants(chunk, "scan_match_chunk.chunk")
		Match_Tail_Invariants(rest, "scan_match_chunk.rest")
	}()
	Match_Nonempty_Text_Invariants(pattern, "scan_match_chunk.pattern")
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
				return star_value, Match_Text(pattern[:index]),
					Match_Tail(pattern[index:])
			}
		}
	}
	return star_value, Match_Text(pattern), ""
}

func match_text_chunk(chunk Match_Text, name Match_Text) (
	rest Match_Tail, okay Boolean, malformed Boolean,
) {
	defer func() {
		Match_Tail_Invariants(rest, "match_text_chunk.rest")
		Boolean_Invariants(okay, "match_text_chunk.okay")
		Boolean_Invariants(malformed, "match_text_chunk.malformed")
	}()
	Match_Text_Invariants(chunk, "match_text_chunk.chunk")
	Match_Text_Invariants(name, "match_text_chunk.name")
	failed := false
	for len(chunk) > 0 {
		if len(name) == 0 {
			failed = true
		}
		switch chunk[0] {
		case '[':
			candidate := Match_Text("")
			if !failed {
				_, decoded_size := utf8.Decode_Character_Text(
					utf8.Text(name),
				)
				size := int(decoded_size)
				candidate = name[:size]
				name = name[size:]
			}
			class_result, member_result, negated_result, class_malformed := match_class(
				Match_Tail(chunk[1:]), candidate,
			)
			if class_malformed {
				return "", false, true
			}
			chunk = Match_Text(class_result)
			if member_result == negated_result {
				failed = true
			}
		case '?':
			if !failed {
				failed = name[0] == '/'
				_, decoded_size := utf8.Decode_Character_Text(
					utf8.Text(name),
				)
				size := int(decoded_size)
				name = name[size:]
			}
			chunk = chunk[1:]
		case '\\':
			chunk = chunk[1:]
			if len(chunk) == 0 {
				return "", false, true
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
		return "", false, false
	}
	return Match_Tail(name), true, false
}

func match_character(value Match_Text) (character utf8.Decoded_Character) {
	defer func() {
		utf8.Decoded_Character_Invariants(character, "match_character.character")
	}()
	Match_Text_Invariants(value, "match_character.value")
	if len(value) == 0 {
		return 0
	}
	if value[0] < byte(utf8.CHARACTER_SELF) {
		return utf8.Decoded_Character(value[0])
	}
	decoded, _ := utf8.Decode_Character_Text(utf8.Text(value))
	return decoded
}

func match_class(
	chunk Match_Tail, candidate_text Match_Text,
) (
	rest Match_Class_Rest, member Boolean, negated Boolean, malformed Boolean,
) {
	defer func() {
		Match_Class_Rest_Invariants(rest, "match_class.rest")
		Boolean_Invariants(member, "match_class.member")
		Boolean_Invariants(negated, "match_class.negated")
		Boolean_Invariants(malformed, "match_class.malformed")
	}()
	Match_Tail_Invariants(chunk, "match_class.chunk")
	Match_Text_Invariants(candidate_text, "match_class.candidate_text")
	candidate := match_character(candidate_text)
	negated = Boolean(len(chunk) > 0 && chunk[0] == '^')
	if negated {
		chunk = chunk[1:]
	}
	if len(chunk) == 0 {
		return "", false, false, true
	}
	literal := len(chunk) >= 2 && chunk[1] == ']' && chunk[0] != '\\' && chunk[0] != '-'
	if literal {
		character := utf8.Decoded_Character(chunk[0])
		member = Boolean(candidate == character)
		return Match_Class_Rest(chunk[2:]), member, negated, false
	}
	escaped_literal := len(chunk) == 3 && chunk[0] == '\\' && chunk[2] == ']'
	if escaped_literal {
		character := utf8.Decoded_Character(chunk[1])
		member = Boolean(candidate == character)
		return "", member, negated, false
	}
	range_literal := len(chunk) >= 4 && chunk[1] == '-' && chunk[3] == ']'
	range_literal = range_literal && chunk[0] != '\\' && chunk[2] != '\\'
	if range_literal {
		low := utf8.Decoded_Character(chunk[0])
		high := utf8.Decoded_Character(chunk[2])
		member = Boolean(low <= candidate)
		member = member && Boolean(candidate <= high)
		return Match_Class_Rest(chunk[4:]), member, negated, false
	}
	if len(chunk) == 4 {
		if chunk[3] == ']' {
			if chunk[0] == '\\' {
				member = Boolean(candidate == utf8.Decoded_Character(chunk[1]))
				second := utf8.Decoded_Character(chunk[2])
				member = member || Boolean(candidate == second)
				return "", member, negated, false
			}
			if chunk[1] == '\\' {
				member = Boolean(candidate == utf8.Decoded_Character(chunk[0]))
				second := utf8.Decoded_Character(chunk[2])
				member = member || Boolean(candidate == second)
				return "", member, negated, false
			}
		}
	}
	range_rest, range_member, range_negated, range_malformed := match_class_ranges(
		Match_Nonempty_Tail(chunk), candidate_text, negated,
	)
	return Match_Class_Rest(range_rest), range_member, range_negated, range_malformed
}

func match_class_ranges(
	chunk Match_Nonempty_Tail, candidate_text Match_Text, negated Boolean,
) (
	rest Match_Range_Rest, member Boolean, negated_output Boolean, malformed Boolean,
) {
	defer func() {
		Match_Range_Rest_Invariants(rest, "match_class_ranges.rest")
		Boolean_Invariants(member, "match_class_ranges.member")
		Boolean_Invariants(negated_output, "match_class_ranges.negated_output")
		Boolean_Invariants(malformed, "match_class_ranges.malformed")
	}()
	Match_Nonempty_Tail_Invariants(chunk, "match_class_ranges.chunk")
	Match_Text_Invariants(candidate_text, "match_class_ranges.candidate_text")
	Boolean_Invariants(negated, "match_class_ranges.negated")
	candidate := match_character(candidate_text)
	remainder := Match_Text(chunk)
	range_count := 0
	for range len(remainder) + NONEMPTY_SIZE_MINIMUM {
		if len(remainder) > 0 {
			if remainder[0] == ']' {
				if range_count > 0 {
					return Match_Range_Rest(remainder[1:]), member,
						negated, false
				}
			}
		}
		if len(remainder) == 0 {
			return "", false, false, true
		}
		low_tail := Match_Nonempty_Tail(remainder)
		escaped_result, low_result, escaped_malformed := match_escaped_character(
			low_tail,
		)
		if escaped_malformed {
			return "", false, false, true
		}
		remainder = Match_Text(escaped_result)
		low := match_character(Match_Text(low_result))
		high := low
		if remainder[0] == '-' {
			if len(remainder) == NONEMPTY_SIZE_MINIMUM {
				return "", false, false, true
			}
			high_tail := Match_Nonempty_Tail(remainder[1:])
			high_rest, high_result, high_malformed := match_escaped_character(
				high_tail,
			)
			if high_malformed {
				return "", false, false, true
			}
			remainder = Match_Text(high_rest)
			high = match_character(Match_Text(high_result))
		}
		if low <= candidate {
			if candidate <= high {
				member = true
			}
		}
		range_count++
	}
	return "", false, false, true
}

func match_escaped_character(chunk Match_Nonempty_Tail) (
	rest Match_Class_Tail, character Match_Character_Text, malformed Boolean,
) {
	defer func() {
		Match_Class_Tail_Invariants(rest, "match_escaped_character.rest")
		Match_Character_Text_Invariants(character, "match_escaped_character.character")
		Boolean_Invariants(malformed, "match_escaped_character.malformed")
	}()
	Match_Nonempty_Tail_Invariants(chunk, "match_escaped_character.chunk")
	size := int(utf8.CHARACTER_SIZE_MINIMUM)
	if len(chunk) == 0 {
		return "", "", true
	}
	if chunk[0] == '-' {
		return "", "", true
	}
	if chunk[0] == ']' {
		return "", "", true
	}
	if chunk[0] == '\\' {
		chunk = chunk[1:]
		if len(chunk) == 0 {
			return "", "", true
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
			return "", "", true
		}
	}
	character = Match_Character_Text(chunk[:size])
	rest = Match_Class_Tail(chunk[size:])
	if len(rest) == 0 {
		return "", "", true
	}
	return rest, character, false
}

func clean_text(storage Clean_Storage, value Text) (count Nonempty_Count) {
	defer func() { Nonempty_Count_Invariants(count, "clean_text.count") }()
	Clean_Storage_Invariants(storage, "clean_text.storage")
	Text_Invariants(value, "clean_text.value")
	if value == "" {
		storage[0] = '.'
		return 1
	}
	copy(storage, value)
	return clean_joined(storage, Nonempty_Count(len(value)))
}

func clean_joined(storage Clean_Storage, size Nonempty_Count) (count Nonempty_Count) {
	defer func() { Nonempty_Count_Invariants(count, "clean_joined.count") }()
	Clean_Storage_Invariants(storage, "clean_joined.storage")
	Nonempty_Count_Invariants(size, "clean_joined.size")
	read, write, dot_dot := 0, 0, 0
	if storage[0] == '/' {
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
			} else if storage[0] != '/' {
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
			if storage[0] == '/' {
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
