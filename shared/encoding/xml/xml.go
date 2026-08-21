// Package xml implements bounded XML validation and text transforms on caller storage.
package xml

import (
	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/sim/aver/default"
	"local/james-orcales/shared/unicode/utf8"
)

// DOCUMENT_SIZE_MAXIMUM follows the repository byte-slice boundary.
const DOCUMENT_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// TEXT_SIZE_MAXIMUM follows the repository byte-slice boundary.
const TEXT_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// OUTPUT_SIZE_MAXIMUM follows the repository byte-slice boundary.
const OUTPUT_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// POSITION_MAXIMUM includes unexpected end immediately after maximum input.
const POSITION_MAXIMUM = DOCUMENT_SIZE_MAXIMUM + 1

// OPEN_ELEMENT_SIZE_MINIMUM is the shortest nonempty opening tag.
const OPEN_ELEMENT_SIZE_MINIMUM = len("<a>")

// CLOSE_ELEMENT_SIZE_MINIMUM is the matching shortest closing tag.
const CLOSE_ELEMENT_SIZE_MINIMUM = len("</a>")

// DEPTH_MAXIMUM follows the shortest complete nested element pair.
const DEPTH_MAXIMUM = DOCUMENT_SIZE_MAXIMUM /
	(OPEN_ELEMENT_SIZE_MINIMUM + CLOSE_ELEMENT_SIZE_MINIMUM)

// ATTRIBUTE_SIZE_MINIMUM is one separating space, name, separator, and empty value.
const ATTRIBUTE_SIZE_MINIMUM = len(" a=''")

// ATTRIBUTE_COUNT_MAXIMUM follows the shortest complete attribute.
const ATTRIBUTE_COUNT_MAXIMUM = DOCUMENT_SIZE_MAXIMUM / ATTRIBUTE_SIZE_MINIMUM

// ESCAPE_QUOT uses the standard library's shorter numeric spelling.
const ESCAPE_QUOT = "&#34;"

// ESCAPE_APOS uses the standard library's shorter numeric spelling.
const ESCAPE_APOS = "&#39;"

// ESCAPE_AMP prevents entity introduction.
const ESCAPE_AMP = "&amp;"

// ESCAPE_LESS prevents markup introduction.
const ESCAPE_LESS = "&lt;"

// ESCAPE_GREATER keeps text symmetric with the standard transform.
const ESCAPE_GREATER = "&gt;"

// ESCAPE_TAB preserves attribute-safe control spelling.
const ESCAPE_TAB = "&#x9;"

// ESCAPE_LINE_FEED preserves attribute-safe control spelling.
const ESCAPE_LINE_FEED = "&#xA;"

// ESCAPE_CARRIAGE_RETURN avoids parser newline normalization.
const ESCAPE_CARRIAGE_RETURN = "&#xD;"

// REPLACEMENT_CHARACTER repairs invalid UTF-8 and forbidden XML characters.
const REPLACEMENT_CHARACTER = "\uFFFD"

// ESCAPED_CHARACTER_SIZE_MAXIMUM is the widest standard replacement.
const ESCAPED_CHARACTER_SIZE_MAXIMUM = len(ESCAPE_CARRIAGE_RETURN)

// STATUS_OK means the operation completed.
const STATUS_OK = 0

// STATUS_INPUT_INVALID means XML or an entity reference is malformed.
const STATUS_INPUT_INVALID = STATUS_OK + 1

// STATUS_OUTPUT_TOO_SMALL means caller output cannot hold the exact result.
const STATUS_OUTPUT_TOO_SMALL = STATUS_INPUT_INVALID + 1

// STATUS_STORAGE_INVALID means output overlaps read-only input.
const STATUS_STORAGE_INVALID = STATUS_OUTPUT_TOO_SMALL + 1

// STATUS_OUTPUT_TOO_LARGE means escaped text exceeds bounded output.
const STATUS_OUTPUT_TOO_LARGE = STATUS_STORAGE_INVALID + 1

// Document is one bounded XML document.
type Document []byte

// Document_Invariants enforces the shared source boundary.
func Document_Invariants(value Document, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, DOCUMENT_SIZE_MAXIMUM).
		Ensure()
}

// Text is caller plain text before escaping.
type Text []byte

// Text_Invariants enforces the shared text boundary.
func Text_Invariants(value Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Escaped is bounded text containing XML entity references.
type Escaped []byte

// Escaped_Invariants enforces the shared encoded boundary.
func Escaped_Invariants(value Escaped, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Output is caller-owned transformed text storage.
type Output []byte

// Output_Invariants enforces the shared destination boundary.
func Output_Invariants(value Output, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, OUTPUT_SIZE_MAXIMUM).
		Ensure()
}

// Count is exact output bytes or zero on refusal.
type Count int

// Count_Invariants keeps results inside caller output.
func Count_Invariants(value Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), bytes.SLICE_SIZE_MINIMUM, OUTPUT_SIZE_MAXIMUM).
		Ensure()
}

// Position is one-based invalid input or zero when absent.
type Position int

// Position_Invariants includes unexpected end after maximum input.
func Position_Invariants(value Position, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), bytes.SLICE_SIZE_MINIMUM, POSITION_MAXIMUM).
		Ensure()
}

// Boolean gives parser decisions independent coverage identity.
type Boolean bool

// Boolean_Invariants covers both parser outcomes.
func Boolean_Invariants(value Boolean, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "An XML parser decision is positive.").
		Ensure()
}

// Entity_Size_Count is zero on refusal or one UTF-8 character.
type Entity_Size_Count uint8

// Entity_Size_Count_Invariants covers invalidity and every UTF-8 width.
func Entity_Size_Count_Invariants(
	value Entity_Size_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value), uint8(utf8.DECODED_SIZE_MINIMUM),
			uint8(utf8.CHARACTER_SIZE_MAXIMUM),
		).
		Ensure()
}

// Escaped_Character_Size_Count is one source character after escaping.
type Escaped_Character_Size_Count uint8

// Escaped_Character_Size_Count_Invariants covers direct and replacement widths.
func Escaped_Character_Size_Count_Invariants(
	value Escaped_Character_Size_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value), uint8(utf8.CHARACTER_SIZE_MINIMUM),
			uint8(ESCAPED_CHARACTER_SIZE_MAXIMUM),
		).
		Ensure()
}

// Encoded_Entity_Size_Count is one valid UTF-8 character width.
type Encoded_Entity_Size_Count uint8

// Encoded_Entity_Size_Count_Invariants excludes invalid entity width.
func Encoded_Entity_Size_Count_Invariants(
	value Encoded_Entity_Size_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), uint8(utf8.CHARACTER_SIZE_MINIMUM),
			uint8(utf8.CHARACTER_SIZE_TWO), uint8(utf8.CHARACTER_SIZE_THREE),
			uint8(utf8.CHARACTER_SIZE_MAXIMUM),
		).
		Ensure()
}

// Validate_Status lists well-formed and malformed documents.
type Validate_Status uint8

// Validate_Status_Invariants covers both validation outcomes.
func Validate_Status_Invariants(value Validate_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), STATUS_OK, STATUS_INPUT_INVALID).
		Ensure()
}

// Escape_Size_Status reports bounded growth.
type Escape_Size_Status uint8

// Escape_Size_Status_Invariants lists successful and oversized text.
func Escape_Size_Status_Invariants(
	value Escape_Size_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), STATUS_OK, STATUS_OUTPUT_TOO_LARGE).
		Ensure()
}

// Escape_Status adds caller storage refusals.
type Escape_Status uint8

// Escape_Status_Invariants lists every escape outcome.
func Escape_Status_Invariants(value Escape_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), STATUS_OK, STATUS_OUTPUT_TOO_SMALL,
			STATUS_STORAGE_INVALID, STATUS_OUTPUT_TOO_LARGE,
		).
		Ensure()
}

// Unescape_Size_Status reports valid or malformed entity input.
type Unescape_Size_Status uint8

// Unescape_Size_Status_Invariants covers both sizing outcomes.
func Unescape_Size_Status_Invariants(
	value Unescape_Size_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), STATUS_OK, STATUS_INPUT_INVALID).
		Ensure()
}

// Unescape_Status adds output and overlap refusals.
type Unescape_Status uint8

// Unescape_Status_Invariants lists every unescape outcome.
func Unescape_Status_Invariants(value Unescape_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), STATUS_OK, STATUS_INPUT_INVALID,
			STATUS_OUTPUT_TOO_SMALL, STATUS_STORAGE_INVALID,
		).
		Ensure()
}

// Validate requires one root while permitting surrounding comments and instructions.
func Validate(source Document) (position Position, status Validate_Status) {
	defer func() {
		Position_Invariants(position, "Validate.position")
		Validate_Status_Invariants(status, "Validate.status")
	}()
	Document_Invariants(source, "Validate.source")
	invalid_position := validate_unchecked(source)
	if invalid_position != 0 {
		return Position(invalid_position), STATUS_INPUT_INVALID
	}
	return 0, STATUS_OK
}

// Escape_Size calculates exact standard text replacement storage.
func Escape_Size(source Text) (count Count, status Escape_Size_Status) {
	defer func() {
		Count_Invariants(count, "Escape_Size.count")
		Escape_Size_Status_Invariants(status, "Escape_Size.status")
	}()
	Text_Invariants(source, "Escape_Size.source")
	calculated := bytes.SLICE_SIZE_MINIMUM
	for position := 0; position < len(source); {
		character, size := utf8.Decode_Character(utf8.Bytes(source[position:]))
		calculated += int(escaped_character_size(character, size))
		if calculated > OUTPUT_SIZE_MAXIMUM {
			return 0, STATUS_OUTPUT_TOO_LARGE
		}
		position += int(size)
	}
	return Count(calculated), STATUS_OK
}

// Escape_Into rejects every refusal before changing caller output.
func Escape_Into(destination Output, source Text) (
	count Count, status Escape_Status,
) {
	defer func() {
		Count_Invariants(count, "Escape_Into.count")
		Escape_Status_Invariants(status, "Escape_Into.status")
	}()
	Output_Invariants(destination, "Escape_Into.destination")
	Text_Invariants(source, "Escape_Into.source")
	if bytes.Overlap(bytes.Slice(destination), bytes.Slice(source)) {
		return 0, STATUS_STORAGE_INVALID
	}
	required, size_status := Escape_Size(source)
	if size_status == STATUS_OUTPUT_TOO_LARGE {
		return 0, STATUS_OUTPUT_TOO_LARGE
	}
	if len(destination) < int(required) {
		return 0, STATUS_OUTPUT_TOO_SMALL
	}
	escape_unchecked(destination, source)
	return required, STATUS_OK
}

// Unescape_Size validates every reference before output access.
func Unescape_Size(source Escaped) (
	count Count, position Position, status Unescape_Size_Status,
) {
	defer func() {
		Count_Invariants(count, "Unescape_Size.count")
		Position_Invariants(position, "Unescape_Size.position")
		Unescape_Size_Status_Invariants(status, "Unescape_Size.status")
	}()
	Escaped_Invariants(source, "Unescape_Size.source")
	calculated := bytes.SLICE_SIZE_MINIMUM
	var entity_storage [utf8.CHARACTER_SIZE_MAXIMUM]byte
	for source_position := 0; source_position < len(source); {
		if source[source_position] == '&' {
			next, error_index, entity_size, valid := parse_entity(
				source, entity_storage[:], source_position,
			)
			if !bool(valid) {
				return 0, Position(int(error_index) + 1), STATUS_INPUT_INVALID
			}
			calculated += int(entity_size)
			source_position = int(next)
			continue
		}
		character, size := utf8.Decode_Character(utf8.Bytes(source[source_position:]))
		if !bool(decoded_xml_character(character, size)) {
			return 0, Position(source_position + 1), STATUS_INPUT_INVALID
		}
		calculated += int(size)
		source_position += int(size)
	}
	return Count(calculated), 0, STATUS_OK
}

// Unescape_Into rejects malformed input, overlap, and capacity before mutation.
func Unescape_Into(destination Output, source Escaped) (
	count Count, position Position, status Unescape_Status,
) {
	defer func() {
		Count_Invariants(count, "Unescape_Into.count")
		Position_Invariants(position, "Unescape_Into.position")
		Unescape_Status_Invariants(status, "Unescape_Into.status")
	}()
	Output_Invariants(destination, "Unescape_Into.destination")
	Escaped_Invariants(source, "Unescape_Into.source")
	if bytes.Overlap(bytes.Slice(destination), bytes.Slice(source)) {
		return 0, 0, STATUS_STORAGE_INVALID
	}
	required, error_position, size_status := Unescape_Size(source)
	if size_status == STATUS_INPUT_INVALID {
		return 0, error_position, STATUS_INPUT_INVALID
	}
	if len(destination) < int(required) {
		return 0, 0, STATUS_OUTPUT_TOO_SMALL
	}
	unescape_unchecked(destination, source)
	return required, 0, STATUS_OK
}

func escaped_character_size[Character ~int32, Decoded_Size ~int](
	character Character, decoded_size Decoded_Size,
) (size Escaped_Character_Size_Count) {
	defer func() {
		Escaped_Character_Size_Count_Invariants(size, "escaped_character_size.size")
	}()
	if int(decoded_size) == utf8.CHARACTER_SIZE_MINIMUM {
		if int32(character) == int32(utf8.REPLACEMENT_CHARACTER) {
			return Escaped_Character_Size_Count(utf8.CHARACTER_SIZE_THREE)
		}
	}
	switch character {
	case '"':
		return Escaped_Character_Size_Count(len(ESCAPE_QUOT))
	case '\'':
		return Escaped_Character_Size_Count(len(ESCAPE_APOS))
	case '&':
		return Escaped_Character_Size_Count(len(ESCAPE_AMP))
	case '<':
		return Escaped_Character_Size_Count(len(ESCAPE_LESS))
	case '>':
		return Escaped_Character_Size_Count(len(ESCAPE_GREATER))
	case '\t':
		return Escaped_Character_Size_Count(len(ESCAPE_TAB))
	case '\n':
		return Escaped_Character_Size_Count(len(ESCAPE_LINE_FEED))
	case '\r':
		return Escaped_Character_Size_Count(len(ESCAPE_CARRIAGE_RETURN))
	default:
		if !bool(xml_character(character)) {
			return Escaped_Character_Size_Count(utf8.CHARACTER_SIZE_THREE)
		}
		return Escaped_Character_Size_Count(decoded_size)
	}
}

func escape_unchecked[Destination ~[]byte, Source ~[]byte](
	destination Destination, source Source,
) {
	output_position := bytes.SLICE_SIZE_MINIMUM
	for source_position := 0; source_position < len(source); {
		character, size := utf8.Decode_Character(utf8.Bytes(source[source_position:]))
		output_position = int(write_escaped_character(
			destination, source, source_position, output_position, character, size,
		))
		source_position += int(size)
	}
}

func write_escaped_character[
	Destination ~[]byte, Source ~[]byte, Index ~int,
	Character ~int32, Decoded_Size ~int,
](
	destination Destination, source Source, source_position Index, output_position Index,
	character Character, decoded_size Decoded_Size,
) (next Index) {
	replacement := ""
	if int(decoded_size) == utf8.CHARACTER_SIZE_MINIMUM {
		if int32(character) == int32(utf8.REPLACEMENT_CHARACTER) {
			replacement = REPLACEMENT_CHARACTER
		}
	}
	if len(replacement) == 0 {
		switch character {
		case '"':
			replacement = ESCAPE_QUOT
		case '\'':
			replacement = ESCAPE_APOS
		case '&':
			replacement = ESCAPE_AMP
		case '<':
			replacement = ESCAPE_LESS
		case '>':
			replacement = ESCAPE_GREATER
		case '\t':
			replacement = ESCAPE_TAB
		case '\n':
			replacement = ESCAPE_LINE_FEED
		case '\r':
			replacement = ESCAPE_CARRIAGE_RETURN
		default:
			if !bool(xml_character(character)) {
				replacement = REPLACEMENT_CHARACTER
			}
		}
	}
	if len(replacement) > 0 {
		written_count := copy(destination[output_position:], replacement)
		return Index(int(output_position) + written_count)
	}
	return Index(int(output_position) + copy(
		destination[output_position:],
		source[source_position:int(source_position)+int(decoded_size)],
	))
}

func xml_character[Character ~int32](character Character) (valid Boolean) {
	defer func() { Boolean_Invariants(valid, "xml_character.valid") }()
	if character == '\t' {
		return true
	}
	if character == '\n' {
		return true
	}
	if character == '\r' {
		return true
	}
	if character < 0x20 {
		return false
	}
	if character <= 0xd7ff {
		return true
	}
	if character < 0xe000 {
		return false
	}
	if character <= 0xfffd {
		return true
	}
	if character < 0x10000 {
		return false
	}
	return Boolean(int32(character) <= int32(utf8.RUNE_MAX))
}

func decoded_xml_character[Character ~int32, Decoded_Size ~int](
	character Character, decoded_size Decoded_Size,
) (valid Boolean) {
	defer func() { Boolean_Invariants(valid, "decoded_xml_character.valid") }()
	if int(decoded_size) == utf8.CHARACTER_SIZE_MINIMUM {
		if int32(character) == int32(utf8.REPLACEMENT_CHARACTER) {
			return false
		}
	}
	return xml_character(character)
}

func parse_entity[Source ~[]byte, Destination ~[]byte, Index ~int](
	source Source, destination Destination, start Index,
) (
	next Index, error_index Index, size Entity_Size_Count, valid Boolean,
) {
	defer func() {
		Entity_Size_Count_Invariants(size, "parse_entity.size")
		Boolean_Invariants(valid, "parse_entity.valid")
	}()
	semicolon := int(start) + len("&")
	for semicolon < len(source) {
		if source[semicolon] == ';' {
			break
		}
		semicolon++
	}
	if semicolon == len(source) {
		return 0, Index(len(source)), 0, false
	}
	name_start := int(start) + len("&")
	if bool(text_equal(source[name_start:semicolon], "lt")) {
		written_next, written_error, written_size := write_entity(
			destination, uint32('<'), Index(semicolon),
		)
		return written_next, written_error, Entity_Size_Count(written_size), true
	}
	if bool(text_equal(source[name_start:semicolon], "gt")) {
		written_next, written_error, written_size := write_entity(
			destination, uint32('>'), Index(semicolon),
		)
		return written_next, written_error, Entity_Size_Count(written_size), true
	}
	if bool(text_equal(source[name_start:semicolon], "amp")) {
		written_next, written_error, written_size := write_entity(
			destination, uint32('&'), Index(semicolon),
		)
		return written_next, written_error, Entity_Size_Count(written_size), true
	}
	if bool(text_equal(source[name_start:semicolon], "apos")) {
		written_next, written_error, written_size := write_entity(
			destination, uint32('\''), Index(semicolon),
		)
		return written_next, written_error, Entity_Size_Count(written_size), true
	}
	if bool(text_equal(source[name_start:semicolon], "quot")) {
		written_next, written_error, written_size := write_entity(
			destination, uint32('"'), Index(semicolon),
		)
		return written_next, written_error, Entity_Size_Count(written_size), true
	}
	if name_start == semicolon {
		return 0, start, 0, false
	}
	if source[name_start] != '#' {
		return 0, start, 0, false
	}
	return parse_numeric_entity(source, destination, Index(name_start+1), Index(semicolon))
}

func parse_numeric_entity[
	Source ~[]byte, Destination ~[]byte, Index ~int,
](
	source Source, destination Destination, start Index, semicolon Index,
) (
	next Index, error_index Index, size Entity_Size_Count, valid Boolean,
) {
	defer func() {
		Entity_Size_Count_Invariants(size, "parse_numeric_entity.size")
		Boolean_Invariants(valid, "parse_numeric_entity.valid")
	}()
	base := uint32(10)
	position := int(start)
	if position < int(semicolon) {
		if source[position] == 'x' {
			base = 16
			position++
		}
	}
	if position == int(semicolon) {
		return 0, start, 0, false
	}
	var character uint32
	for position < int(semicolon) {
		value := source[position]
		var digit uint32
		if value >= '0' {
			if value <= '9' {
				digit = uint32(value - '0')
			} else if base == 16 {
				if value >= 'a' {
					if value <= 'f' {
						digit = uint32(value-'a') + 10
					} else {
						return 0, Index(position), 0, false
					}
				} else if value >= 'A' {
					if value <= 'F' {
						digit = uint32(value-'A') + 10
					} else {
						return 0, Index(position), 0, false
					}
				} else {
					return 0, Index(position), 0, false
				}
			} else {
				return 0, Index(position), 0, false
			}
		} else {
			return 0, Index(position), 0, false
		}
		if character > (uint32(utf8.RUNE_MAX)-digit)/base {
			return 0, Index(position), 0, false
		}
		character = character*base + digit
		position++
	}
	if !bool(xml_character(int32(character))) {
		return 0, start, 0, false
	}
	written_next, written_error, written_size := write_entity(
		destination, character, semicolon,
	)
	return written_next, written_error, Entity_Size_Count(written_size), true
}

func write_entity[Destination ~[]byte, Character ~uint32, Index ~int](
	destination Destination, character Character, semicolon Index,
) (next Index, error_index Index, size Encoded_Entity_Size_Count) {
	defer func() {
		Encoded_Entity_Size_Count_Invariants(size, "write_entity.size")
	}()
	character_size := utf8.Character_Size(utf8.Character(character))
	size = Encoded_Entity_Size_Count(character_size)
	if len(destination) >= int(size) {
		utf8.Encode_Character(
			utf8.Bytes(destination[:size]), utf8.Character(character),
		)
	}
	return Index(int(semicolon) + len(";")), 0, size
}

func text_equal[Source ~[]byte, Expected ~string](
	source Source, expected Expected,
) (equal Boolean) {
	defer func() { Boolean_Invariants(equal, "text_equal.equal") }()
	if len(source) != len(expected) {
		return false
	}
	for index := range source {
		if source[index] != expected[index] {
			return false
		}
	}
	return true
}

func unescape_unchecked[Destination ~[]byte, Source ~[]byte](
	destination Destination, source Source,
) {
	output_position := bytes.SLICE_SIZE_MINIMUM
	for source_position := 0; source_position < len(source); {
		if source[source_position] == '&' {
			next, _, entity_size, _ := parse_entity(
				source, destination[output_position:], source_position,
			)
			output_position += int(entity_size)
			source_position = int(next)
			continue
		}
		_, size := utf8.Decode_Character(utf8.Bytes(source[source_position:]))
		output_position += copy(
			destination[output_position:],
			source[source_position:source_position+int(size)],
		)
		source_position += int(size)
	}
}

func validate_unchecked[Source ~[]byte](source Source) (position Position) {
	defer func() { Position_Invariants(position, "validate_unchecked.position") }()
	var name_starts [DEPTH_MAXIMUM]int
	var name_ends [DEPTH_MAXIMUM]int
	index := bytes.SLICE_SIZE_MINIMUM
	depth := bytes.SLICE_SIZE_MINIMUM
	root_seen := false
	root_closed := false
	for index < len(source) {
		if source[index] != '<' {
			next, error_index, valid := scan_document_text(source, index, depth)
			if !bool(valid) {
				return Position(int(error_index) + 1)
			}
			index = int(next)
			continue
		}
		next, error_index, matched, valid := scan_document_special(source, index, depth)
		if matched {
			if !bool(valid) {
				return Position(int(error_index) + 1)
			}
			index = int(next)
			continue
		}
		if bool(prefix_at(source, index, "</")) {
			close_next, close_error, closed, close_valid := close_document_element(
				source, index, &name_starts, &name_ends, &depth,
			)
			if !bool(close_valid) {
				return Position(int(close_error) + 1)
			}
			index = int(close_next)
			if closed {
				root_closed = true
			}
			continue
		}
		if root_closed {
			return Position(index + 1)
		}
		next, error_index, closed, valid := open_document_element(
			source, index, &name_starts, &name_ends, &depth,
		)
		if !bool(valid) {
			return Position(int(error_index) + 1)
		}
		root_seen = true
		index = int(next)
		if closed {
			root_closed = true
		}
	}
	if depth != bytes.SLICE_SIZE_MINIMUM {
		return Position(len(source) + 1)
	}
	if !root_seen {
		return Position(len(source) + 1)
	}
	return 0
}

func scan_document_text[Source ~[]byte, Index ~int, Depth ~int](
	source Source, start Index, depth Depth,
) (next Index, error_index Index, valid Boolean) {
	defer func() { Boolean_Invariants(valid, "scan_document_text.valid") }()
	next, error_index, valid = scan_text(source, start)
	if !valid {
		return 0, error_index, false
	}
	if int(depth) != bytes.SLICE_SIZE_MINIMUM {
		return next, 0, true
	}
	for outside := int(start); outside < int(next); outside++ {
		if !bool(xml_space(source[outside])) {
			return 0, Index(outside), false
		}
	}
	return next, 0, true
}

func scan_document_special[Source ~[]byte, Index ~int, Depth ~int](
	source Source, start Index, depth Depth,
) (
	next Index, error_index Index, matched Boolean, valid Boolean,
) {
	defer func() {
		Boolean_Invariants(matched, "scan_document_special.matched")
		Boolean_Invariants(valid, "scan_document_special.valid")
	}()
	if bool(prefix_at(source, start, "<!--")) {
		next, error_index, valid = scan_comment(source, start)
		return next, error_index, true, valid
	}
	if bool(prefix_at(source, start, "<?")) {
		next, error_index, valid = scan_instruction(source, start)
		return next, error_index, true, valid
	}
	if bool(prefix_at(source, start, "<![CDATA[")) {
		if int(depth) == bytes.SLICE_SIZE_MINIMUM {
			return 0, start, true, false
		}
		next, error_index, valid = scan_cdata(source, start)
		return next, error_index, true, valid
	}
	if bool(prefix_at(source, start, "<![")) {
		return 0, start, true, false
	}
	if bool(prefix_at(source, start, "<!-")) {
		return 0, start, true, false
	}
	if bool(prefix_at(source, start, "<!")) {
		next, error_index, valid = scan_directive(source, start)
		return next, error_index, true, valid
	}
	return 0, 0, false, true
}

func close_document_element[Source ~[]byte, Index ~int, Depth ~int](
	source Source, start Index, name_starts *[DEPTH_MAXIMUM]int,
	name_ends *[DEPTH_MAXIMUM]int, depth *Depth,
) (next Index, error_index Index, closed Boolean, valid Boolean) {
	defer func() {
		Boolean_Invariants(closed, "close_document_element.closed")
		Boolean_Invariants(valid, "close_document_element.valid")
	}()
	if int(*depth) == bytes.SLICE_SIZE_MINIMUM {
		return 0, start, false, false
	}
	next, name_start, name_end, error_index, valid := scan_end_tag(source, start)
	if !valid {
		return 0, error_index, false, false
	}
	*depth--
	if !bytes.Equal(
		bytes.Slice(source[name_start:name_end]),
		bytes.Slice(source[name_starts[*depth]:name_ends[*depth]]),
	) {
		return 0, name_start, false, false
	}
	return next, 0, Boolean(int(*depth) == bytes.SLICE_SIZE_MINIMUM), true
}

func open_document_element[Source ~[]byte, Index ~int, Depth ~int](
	source Source, start Index, name_starts *[DEPTH_MAXIMUM]int,
	name_ends *[DEPTH_MAXIMUM]int, depth *Depth,
) (next Index, error_index Index, closed Boolean, valid Boolean) {
	defer func() {
		Boolean_Invariants(closed, "open_document_element.closed")
		Boolean_Invariants(valid, "open_document_element.valid")
	}()
	next, name_start, name_end, error_index, empty, valid := scan_start_tag(
		source, start,
	)
	if !valid {
		return 0, error_index, false, false
	}
	if empty {
		return next, 0, Boolean(int(*depth) == bytes.SLICE_SIZE_MINIMUM), true
	}
	if int(*depth) == DEPTH_MAXIMUM {
		return 0, start, false, false
	}
	name_starts[*depth] = int(name_start)
	name_ends[*depth] = int(name_end)
	*depth++
	return next, 0, false, true
}

func scan_text[Source ~[]byte, Index ~int](
	source Source, start Index,
) (next Index, error_index Index, valid Boolean) {
	defer func() { Boolean_Invariants(valid, "scan_text.valid") }()
	index := int(start)
	var entity_storage [utf8.CHARACTER_SIZE_MAXIMUM]byte
	for index < len(source) {
		if source[index] == '<' {
			return Index(index), 0, true
		}
		if bool(prefix_at(source, index, "]]>")) {
			return 0, Index(index), false
		}
		if source[index] == '&' {
			entity_next, entity_error, _, entity_valid := parse_entity(
				source, entity_storage[:], index,
			)
			if !bool(entity_valid) {
				return 0, Index(entity_error), false
			}
			index = int(entity_next)
			continue
		}
		character, size := utf8.Decode_Character(utf8.Bytes(source[index:]))
		if !bool(decoded_xml_character(character, size)) {
			return 0, Index(index), false
		}
		index += int(size)
	}
	return Index(index), 0, true
}

func scan_start_tag[Source ~[]byte, Index ~int](
	source Source, start Index,
) (
	next Index, name_start Index, name_end Index, error_index Index,
	empty Boolean, valid Boolean,
) {
	defer func() {
		Boolean_Invariants(empty, "scan_start_tag.empty")
		Boolean_Invariants(valid, "scan_start_tag.valid")
	}()
	name_start = start + Index(len("<"))
	name_end, error_index, valid = scan_qualified_name(source, name_start)
	if !valid {
		return 0, 0, 0, error_index, false, false
	}
	index := int(name_end)
	var attribute_starts [ATTRIBUTE_COUNT_MAXIMUM]int
	var attribute_ends [ATTRIBUTE_COUNT_MAXIMUM]int
	attribute_count := bytes.SLICE_SIZE_MINIMUM
	for index < len(source) {
		if source[index] == '>' {
			return Index(index + 1), name_start, name_end, 0, false, true
		}
		if source[index] == '/' {
			if index+1 < len(source) {
				if source[index+1] == '>' {
					return Index(index + 2), name_start, name_end, 0, true, true
				}
			}
			return 0, 0, 0, Index(index), false, false
		}
		if !bool(xml_space(source[index])) {
			return 0, 0, 0, Index(index), false, false
		}
		index = int(skip_xml_space(source, index))
		if index == len(source) {
			return 0, 0, 0, Index(len(source)), false, false
		}
		if source[index] == '>' {
			return Index(index + 1), name_start, name_end, 0, false, true
		}
		if source[index] == '/' {
			continue
		}
		attribute_next, attribute_start, attribute_end,
			attribute_error, attribute_valid := scan_attribute(
			source, Index(index),
		)
		if !attribute_valid {
			return 0, 0, 0, attribute_error, false, false
		}
		for previous_index := 0; previous_index < attribute_count; previous_index++ {
			previous_start := attribute_starts[previous_index]
			previous_end := attribute_ends[previous_index]
			if bytes.Equal(
				bytes.Slice(source[attribute_start:attribute_end]),
				bytes.Slice(source[previous_start:previous_end]),
			) {
				return 0, 0, 0, attribute_start, false, false
			}
		}
		attribute_starts[attribute_count] = int(attribute_start)
		attribute_ends[attribute_count] = int(attribute_end)
		attribute_count++
		index = int(attribute_next)
	}
	return 0, 0, 0, Index(len(source)), false, false
}

func scan_attribute[Source ~[]byte, Index ~int](
	source Source, start Index,
) (
	next Index, name_start Index, name_end Index,
	error_index Index, valid Boolean,
) {
	defer func() { Boolean_Invariants(valid, "scan_attribute.valid") }()
	name_start = start
	name_end, error_index, valid = scan_qualified_name(source, start)
	if !valid {
		return 0, 0, 0, error_index, false
	}
	index := int(skip_xml_space(source, name_end))
	if index == len(source) {
		return 0, 0, 0, Index(len(source)), false
	}
	if source[index] != '=' {
		return 0, 0, 0, Index(index), false
	}
	index = int(skip_xml_space(source, index+1))
	if index == len(source) {
		return 0, 0, 0, Index(len(source)), false
	}
	quote := source[index]
	if quote != '\'' {
		if quote != '"' {
			return 0, 0, 0, Index(index), false
		}
	}
	index++
	var entity_storage [utf8.CHARACTER_SIZE_MAXIMUM]byte
	for index < len(source) {
		if source[index] == quote {
			return Index(index + 1), name_start, name_end, 0, true
		}
		if source[index] == '<' {
			return 0, 0, 0, Index(index), false
		}
		if source[index] == '&' {
			entity_next, entity_error, _, entity_valid := parse_entity(
				source, entity_storage[:], index,
			)
			if !bool(entity_valid) {
				return 0, 0, 0, Index(entity_error), false
			}
			index = int(entity_next)
			continue
		}
		character, size := utf8.Decode_Character(utf8.Bytes(source[index:]))
		if !bool(decoded_xml_character(character, size)) {
			return 0, 0, 0, Index(index), false
		}
		index += int(size)
	}
	return 0, 0, 0, Index(len(source)), false
}

func scan_end_tag[Source ~[]byte, Index ~int](
	source Source, start Index,
) (
	next Index, name_start Index, name_end Index,
	error_index Index, valid Boolean,
) {
	defer func() { Boolean_Invariants(valid, "scan_end_tag.valid") }()
	name_start = start + Index(len("</"))
	name_end, error_index, valid = scan_qualified_name(source, name_start)
	if !valid {
		return 0, 0, 0, error_index, false
	}
	index := int(skip_xml_space(source, name_end))
	if index == len(source) {
		return 0, 0, 0, Index(len(source)), false
	}
	if source[index] != '>' {
		return 0, 0, 0, Index(index), false
	}
	return Index(index + 1), name_start, name_end, 0, true
}

func scan_name[Source ~[]byte, Index ~int](
	source Source, start Index,
) (next Index, error_index Index, valid Boolean) {
	defer func() { Boolean_Invariants(valid, "scan_name.valid") }()
	index := int(start)
	if index == len(source) {
		return 0, Index(len(source)), false
	}
	character, size := utf8.Decode_Character(utf8.Bytes(source[index:]))
	if !bool(decoded_xml_character(character, size)) {
		return 0, Index(index), false
	}
	if !bool(name_start_character(character)) {
		return 0, Index(index), false
	}
	index += int(size)
	for index < len(source) {
		if source[index] < byte(utf8.CHARACTER_SELF) {
			if !bool(name_character(int32(source[index]))) {
				break
			}
			index++
			continue
		}
		character, size = utf8.Decode_Character(utf8.Bytes(source[index:]))
		if !bool(decoded_xml_character(character, size)) {
			return 0, Index(index), false
		}
		if !bool(name_character(character)) {
			break
		}
		index += int(size)
	}
	return Index(index), 0, true
}

func scan_qualified_name[Source ~[]byte, Index ~int](
	source Source, start Index,
) (next Index, error_index Index, valid Boolean) {
	defer func() { Boolean_Invariants(valid, "scan_qualified_name.valid") }()
	next, error_index, valid = scan_name(source, start)
	if !bool(valid) {
		return 0, error_index, false
	}
	colon_seen := false
	for index := int(start); index < int(next); index++ {
		if source[index] != ':' {
			continue
		}
		if colon_seen {
			return 0, Index(index), false
		}
		colon_seen = true
	}
	return next, 0, true
}

func scan_comment[Source ~[]byte, Index ~int](
	source Source, start Index,
) (next Index, error_index Index, valid Boolean) {
	defer func() { Boolean_Invariants(valid, "scan_comment.valid") }()
	index := int(start) + len("<!--")
	for index < len(source) {
		if bool(prefix_at(source, index, "-->")) {
			return Index(index + len("-->")), 0, true
		}
		if bool(prefix_at(source, index, "--")) {
			return 0, Index(index), false
		}
		index++
	}
	return 0, Index(len(source)), false
}

func scan_instruction[Source ~[]byte, Index ~int](
	source Source, start Index,
) (next Index, error_index Index, valid Boolean) {
	defer func() { Boolean_Invariants(valid, "scan_instruction.valid") }()
	name_start := start + Index(len("<?"))
	name_end, error_index, valid := scan_name(source, name_start)
	if !valid {
		return 0, error_index, false
	}
	index := int(name_end)
	for index < len(source) {
		if bool(prefix_at(source, index, "?>")) {
			instruction_error, instruction_valid := xml_instruction_valid(
				source, name_start, name_end, Index(index),
			)
			if !bool(instruction_valid) {
				return 0, instruction_error, false
			}
			return Index(index + len("?>")), 0, true
		}
		index++
	}
	return 0, Index(len(source)), false
}

func xml_instruction_valid[Source ~[]byte, Index ~int](
	source Source, name_start Index, name_end Index, data_end Index,
) (error_index Index, valid Boolean) {
	defer func() { Boolean_Invariants(valid, "xml_instruction_valid.valid") }()
	if int(name_end)-int(name_start) != len("xml") {
		return 0, true
	}
	if !bool(prefix_at(source, name_start, "xml")) {
		return 0, true
	}
	value_start, value_end, found := instruction_parameter(
		source, name_end, data_end, "version=",
	)
	if bool(found) {
		if value_start != value_end {
			if int(value_end)-int(value_start) != len("1.0") {
				return value_start, false
			}
			if !bool(prefix_at(source, value_start, "1.0")) {
				return value_start, false
			}
		}
	}
	value_start, value_end, found = instruction_parameter(
		source, name_end, data_end, "encoding=",
	)
	if bool(found) {
		if value_start != value_end {
			var utf8_name = [...]byte{'u', 't', 'f', '-', '8'}
			encoding_name := bytes.Slice(source[value_start:value_end])
			if !bool(bytes.Equal_Fold(
				encoding_name, bytes.Slice(utf8_name[:]),
			)) {
				return value_start, false
			}
		}
	}
	return 0, true
}

func instruction_parameter[
	Source ~[]byte, Index ~int, Parameter ~string,
](
	source Source, start Index, end Index, parameter Parameter,
) (value_start Index, value_end Index, found Boolean) {
	defer func() { Boolean_Invariants(found, "instruction_parameter.found") }()
	for index := int(start); index+len(parameter) < int(end); index++ {
		if !bool(prefix_at(source, index, parameter)) {
			continue
		}
		quote_index := index + len(parameter)
		quote := source[quote_index]
		if quote != '\'' {
			if quote != '"' {
				continue
			}
		}
		value_start = Index(quote_index + 1)
		for value_index := int(value_start); value_index < int(end); value_index++ {
			if source[value_index] == quote {
				return value_start, Index(value_index), true
			}
		}
	}
	return 0, 0, false
}

func scan_cdata[Source ~[]byte, Index ~int](
	source Source, start Index,
) (next Index, error_index Index, valid Boolean) {
	defer func() { Boolean_Invariants(valid, "scan_cdata.valid") }()
	index := int(start) + len("<![CDATA[")
	for index < len(source) {
		if bool(prefix_at(source, index, "]]>")) {
			return Index(index + len("]]>")), 0, true
		}
		character, size := utf8.Decode_Character(utf8.Bytes(source[index:]))
		if !bool(decoded_xml_character(character, size)) {
			return 0, Index(index), false
		}
		index += int(size)
	}
	return 0, Index(len(source)), false
}

func scan_directive[Source ~[]byte, Index ~int](
	source Source, start Index,
) (next Index, error_index Index, valid Boolean) {
	defer func() { Boolean_Invariants(valid, "scan_directive.valid") }()
	index := int(start) + len("<!")
	if index == len(source) {
		return 0, Index(len(source)), false
	}
	// Encoding/xml dispatch consumes first directive byte before nesting starts.
	index++
	quote := byte(0)
	depth := bytes.SLICE_SIZE_MINIMUM
	for index < len(source) {
		if quote != 0 {
			if source[index] == quote {
				quote = 0
				index++
				continue
			}
			index++
			continue
		}
		if bool(prefix_at(source, index, "<!--")) {
			comment_next, comment_error, comment_valid := scan_directive_comment(
				source, Index(index),
			)
			if !bool(comment_valid) {
				return 0, comment_error, false
			}
			index = int(comment_next)
			continue
		}
		value := source[index]
		if value == '\'' {
			quote = value
			index++
			continue
		}
		if value == '"' {
			quote = value
			index++
			continue
		}
		if value == '<' {
			depth++
			index++
			continue
		}
		if value == '>' {
			if depth == bytes.SLICE_SIZE_MINIMUM {
				return Index(index + 1), 0, true
			}
			depth--
			index++
			continue
		}
		index++
	}
	return 0, Index(len(source)), false
}

func scan_directive_comment[Source ~[]byte, Index ~int](
	source Source, start Index,
) (next Index, error_index Index, valid Boolean) {
	defer func() { Boolean_Invariants(valid, "scan_directive_comment.valid") }()
	for index := int(start) + len("<!--"); index < len(source); index++ {
		if bool(prefix_at(source, index, "-->")) {
			return Index(index + len("-->")), 0, true
		}
	}
	return 0, Index(len(source)), false
}

func prefix_at[Source ~[]byte, Index ~int, Prefix ~string](
	source Source, start Index, prefix Prefix,
) (matches Boolean) {
	defer func() { Boolean_Invariants(matches, "prefix_at.matches") }()
	if int(start)+len(prefix) > len(source) {
		return false
	}
	for index := range len(prefix) {
		if source[int(start)+index] != prefix[index] {
			return false
		}
	}
	return true
}

func skip_xml_space[Source ~[]byte, Index ~int](
	source Source, start Index,
) (next Index) {
	index := int(start)
	for index < len(source) {
		if !bool(xml_space(source[index])) {
			break
		}
		index++
	}
	return Index(index)
}

func xml_space[Value ~byte](value Value) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "xml_space.yes") }()
	switch value {
	case ' ', '\t', '\r', '\n':
		return true
	default:
		return false
	}
}

func name_start_character[Character ~int32](character Character) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "name_start_character.yes") }()
	if character < 0x200c {
		return name_start_character_lower(character)
	}
	return name_start_character_upper(character)
}

func name_start_character_lower[Character ~int32](character Character) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "name_start_character_lower.yes") }()
	if character == '_' {
		return true
	}
	if character == ':' {
		return true
	}
	if character >= 'A' {
		if character <= 'Z' {
			return true
		}
	}
	if character >= 'a' {
		if character <= 'z' {
			return true
		}
	}
	if character >= 0xc0 {
		if character <= 0xd6 {
			return true
		}
	}
	if character >= 0xd8 {
		if character <= 0xf6 {
			return true
		}
	}
	if character >= 0xf8 {
		if character <= 0x2ff {
			return true
		}
	}
	if character >= 0x370 {
		if character <= 0x37d {
			return true
		}
	}
	if character < 0x37f {
		return false
	}
	return Boolean(character <= 0x1fff)
}

func name_start_character_upper[Character ~int32](character Character) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "name_start_character_upper.yes") }()
	if character <= 0x200d {
		return true
	}
	if character >= 0x2070 {
		if character <= 0x218f {
			return true
		}
	}
	if character >= 0x2c00 {
		if character <= 0x2fef {
			return true
		}
	}
	if character >= 0x3001 {
		if character <= 0xd7ff {
			return true
		}
	}
	if character >= 0xf900 {
		if character <= 0xfdcf {
			return true
		}
	}
	if character >= 0xfdf0 {
		if character <= 0xfffd {
			return true
		}
	}
	if character < 0x10000 {
		return false
	}
	return Boolean(character <= 0xeffff)
}

func name_character[Character ~int32](character Character) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "name_character.yes") }()
	if bool(name_start_character(character)) {
		return true
	}
	if character == '-' {
		return true
	}
	if character == '.' {
		return true
	}
	if character >= '0' {
		if character <= '9' {
			return true
		}
	}
	if character == 0xb7 {
		return true
	}
	if character >= 0x300 {
		if character <= 0x36f {
			return true
		}
	}
	if character < 0x203f {
		return false
	}
	return Boolean(character <= 0x2040)
}
