// Package xml implements bounded XML validation and text transforms on caller storage.
package xml

import (
	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/aver/default"
	"local/james-orcales/shared/unicode/utf8"
)

// DOCUMENT_SIZE_MAXIMUM follows the repository byte-slice boundary.
const DOCUMENT_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// TEXT_SIZE_MAXIMUM follows the repository byte-slice boundary.
const TEXT_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// OUTPUT_SIZE_MAXIMUM follows the repository byte-slice boundary.
const OUTPUT_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// NONEMPTY_SIZE_MINIMUM is one addressable byte.
const NONEMPTY_SIZE_MINIMUM = 1

// SPECIAL_DOCUMENT_SIZE_MINIMUM holds markup opener and discriminator.
const SPECIAL_DOCUMENT_SIZE_MINIMUM = len("<!")

// NUMERIC_ENTITY_DOCUMENT_SIZE_MINIMUM holds the shortest numeric entity shape.
const NUMERIC_ENTITY_DOCUMENT_SIZE_MINIMUM = len("&#;")

// SPACE_DOCUMENT_SIZE_MINIMUM holds a start, name, and following space.
const SPACE_DOCUMENT_SIZE_MINIMUM = len("<a ")

// ATTRIBUTE_DOCUMENT_SIZE_MINIMUM holds a start tag and one attribute-name byte.
const ATTRIBUTE_DOCUMENT_SIZE_MINIMUM = len("<a b")

// COMMENT_DOCUMENT_SIZE_MINIMUM holds the complete comment opener.
const COMMENT_DOCUMENT_SIZE_MINIMUM = len("<!--")

// TERMINATED_INSTRUCTION_DOCUMENT_SIZE_MINIMUM holds one target and its terminator.
const TERMINATED_INSTRUCTION_DOCUMENT_SIZE_MINIMUM = len("<?a?>")

// END_TAG_DOCUMENT_SIZE_MINIMUM holds one open element and an incomplete end tag.
const END_TAG_DOCUMENT_SIZE_MINIMUM = len("<a></")

// XML_INSTRUCTION_DOCUMENT_SIZE_MINIMUM holds the XML target and its terminator.
const XML_INSTRUCTION_DOCUMENT_SIZE_MINIMUM = len("<?xml?>")

// DIRECTIVE_COMMENT_DOCUMENT_SIZE_MINIMUM reaches a nested comment opener.
const DIRECTIVE_COMMENT_DOCUMENT_SIZE_MINIMUM = len("<!a<!--")

// CDATA_DOCUMENT_SIZE_MINIMUM reaches a CDATA opener inside one root element.
const CDATA_DOCUMENT_SIZE_MINIMUM = len("<a><![CDATA[")

// ENTITY_NAME_SIZE_MAXIMUM leaves ampersand and semicolon framing.
const ENTITY_NAME_SIZE_MAXIMUM = DOCUMENT_SIZE_MAXIMUM - len("&;")

// POSITION_MAXIMUM includes unexpected end immediately after maximum input.
const POSITION_MAXIMUM = DOCUMENT_SIZE_MAXIMUM + 1

// DOCUMENT_INDEX_MAXIMUM is the last addressable document byte.
const DOCUMENT_INDEX_MAXIMUM = DOCUMENT_SIZE_MAXIMUM - 1

// START_TAG_POSITION_MINIMUM follows the opening less-than byte.
const START_TAG_POSITION_MINIMUM = 1

// END_TAG_POSITION_MINIMUM follows the end-tag opener.
const END_TAG_POSITION_MINIMUM = 2

// INSTRUCTION_POSITION_MINIMUM follows the instruction opener.
const INSTRUCTION_POSITION_MINIMUM = 2

// DIRECTIVE_POSITION_MINIMUM follows the directive opener.
const DIRECTIVE_POSITION_MINIMUM = 2

// COMMENT_POSITION_MINIMUM follows the comment opener.
const COMMENT_POSITION_MINIMUM = 4

// CDATA_POSITION_MINIMUM follows the CDATA opener.
const CDATA_POSITION_MINIMUM = 9

// DIRECTIVE_COMMENT_POSITION_MINIMUM follows a nested comment opener.
const DIRECTIVE_COMMENT_POSITION_MINIMUM = 7

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

// PATTERN_SIZE_MINIMUM is the shortest fixed XML token.
const PATTERN_SIZE_MINIMUM = len("<?")

// PATTERN_SIZE_MAXIMUM is the longest fixed XML token.
const PATTERN_SIZE_MAXIMUM = len("<![CDATA[")

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

// Nonempty_Document is parser input after one byte has been selected.
type Nonempty_Document []byte

// Nonempty_Document_Invariants rejects absent parser work.
func Nonempty_Document_Invariants(value Nonempty_Document, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), NONEMPTY_SIZE_MINIMUM, DOCUMENT_SIZE_MAXIMUM).
		Ensure()
}

// Special_Document contains markup opener and discriminator.
type Special_Document []byte

// Special_Document_Invariants states the shared special-markup prefix.
func Special_Document_Invariants(value Special_Document, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), SPECIAL_DOCUMENT_SIZE_MINIMUM, DOCUMENT_SIZE_MAXIMUM).
		Ensure()
}

// Numeric_Entity_Document contains the shortest numeric reference shape.
type Numeric_Entity_Document []byte

// Numeric_Entity_Document_Invariants rejects input that cannot reach numeric parsing.
func Numeric_Entity_Document_Invariants(
	value Numeric_Entity_Document, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), NUMERIC_ENTITY_DOCUMENT_SIZE_MINIMUM, DOCUMENT_SIZE_MAXIMUM).
		Ensure()
}

// Space_Document contains a start-tag name followed by space.
type Space_Document []byte

// Space_Document_Invariants rejects input that cannot reach space scanning.
func Space_Document_Invariants(value Space_Document, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), SPACE_DOCUMENT_SIZE_MINIMUM, DOCUMENT_SIZE_MAXIMUM).
		Ensure()
}

// Attribute_Document contains one reachable attribute-name byte.
type Attribute_Document []byte

// Attribute_Document_Invariants rejects input that cannot reach attribute parsing.
func Attribute_Document_Invariants(value Attribute_Document, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), ATTRIBUTE_DOCUMENT_SIZE_MINIMUM, DOCUMENT_SIZE_MAXIMUM).
		Ensure()
}

// Comment_Document contains the complete XML comment opener.
type Comment_Document []byte

// Comment_Document_Invariants rejects input that cannot reach comment scanning.
func Comment_Document_Invariants(value Comment_Document, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COMMENT_DOCUMENT_SIZE_MINIMUM, DOCUMENT_SIZE_MAXIMUM).
		Ensure()
}

// Terminated_Instruction_Document contains one target and its terminator.
type Terminated_Instruction_Document []byte

// Terminated_Instruction_Document_Invariants rejects incomplete instruction framing.
func Terminated_Instruction_Document_Invariants(
	value Terminated_Instruction_Document, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), TERMINATED_INSTRUCTION_DOCUMENT_SIZE_MINIMUM,
			DOCUMENT_SIZE_MAXIMUM,
		).
		Ensure()
}

// End_Tag_Document contains one open element before an end-tag scan.
type End_Tag_Document []byte

// End_Tag_Document_Invariants rejects input that cannot reach end-tag scanning.
func End_Tag_Document_Invariants(value End_Tag_Document, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), END_TAG_DOCUMENT_SIZE_MINIMUM, DOCUMENT_SIZE_MAXIMUM).
		Ensure()
}

// XML_Instruction_Document contains the XML target and its terminator.
type XML_Instruction_Document []byte

// XML_Instruction_Document_Invariants rejects input without a complete XML target.
func XML_Instruction_Document_Invariants(
	value XML_Instruction_Document, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), XML_INSTRUCTION_DOCUMENT_SIZE_MINIMUM, DOCUMENT_SIZE_MAXIMUM,
		).
		Ensure()
}

// Directive_Comment_Document reaches a nested directive comment opener.
type Directive_Comment_Document []byte

// Directive_Comment_Document_Invariants rejects input before nested comment dispatch.
func Directive_Comment_Document_Invariants(
	value Directive_Comment_Document, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), DIRECTIVE_COMMENT_DOCUMENT_SIZE_MINIMUM, DOCUMENT_SIZE_MAXIMUM,
		).
		Ensure()
}

// CDATA_Document reaches a CDATA opener inside one root element.
type CDATA_Document []byte

// CDATA_Document_Invariants rejects input before legal CDATA dispatch.
func CDATA_Document_Invariants(value CDATA_Document, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), CDATA_DOCUMENT_SIZE_MINIMUM, DOCUMENT_SIZE_MAXIMUM).
		Ensure()
}

// Entity_Name is reference content without ampersand or semicolon framing.
type Entity_Name []byte

// Entity_Name_Invariants leaves room for mandatory reference framing.
func Entity_Name_Invariants(value Entity_Name, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, ENTITY_NAME_SIZE_MAXIMUM).
		Ensure()
}

// Nonempty_Text is escape input after one character has been selected.
type Nonempty_Text []byte

// Nonempty_Text_Invariants rejects absent escape work.
func Nonempty_Text_Invariants(value Nonempty_Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), NONEMPTY_SIZE_MINIMUM, TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Nonempty_Escaped is entity input after one character has been selected.
type Nonempty_Escaped []byte

// Nonempty_Escaped_Invariants rejects absent unescape work.
func Nonempty_Escaped_Invariants(value Nonempty_Escaped, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), NONEMPTY_SIZE_MINIMUM, TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Nonempty_Output has one byte available for proven transform work.
type Nonempty_Output []byte

// Nonempty_Output_Invariants rejects storage that cannot accept selected work.
func Nonempty_Output_Invariants(value Nonempty_Output, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), NONEMPTY_SIZE_MINIMUM, OUTPUT_SIZE_MAXIMUM).
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

// Document_Position is one zero-based document boundary.
type Document_Position int

// Document_Position_Invariants bounds parser progress and absent zero.
func Document_Position_Invariants(value Document_Position, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), bytes.SLICE_SIZE_MINIMUM, DOCUMENT_SIZE_MAXIMUM).
		Ensure()
}

// Document_Index is one addressable document byte.
type Document_Index int

// Document_Index_Invariants excludes the boundary after input storage.
func Document_Index_Invariants(value Document_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), bytes.SLICE_SIZE_MINIMUM, DOCUMENT_INDEX_MAXIMUM).
		Ensure()
}

// Next_Position follows at least one consumed or written byte.
type Next_Position int

// Next_Position_Invariants bounds nonzero progress.
func Next_Position_Invariants(value Next_Position, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), NONEMPTY_SIZE_MINIMUM, DOCUMENT_SIZE_MAXIMUM).
		Ensure()
}

// Start_Tag_Position follows the opening less-than byte.
type Start_Tag_Position int

// Start_Tag_Position_Invariants bounds start-tag progress.
func Start_Tag_Position_Invariants(value Start_Tag_Position, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), START_TAG_POSITION_MINIMUM, DOCUMENT_SIZE_MAXIMUM).
		Ensure()
}

// End_Tag_Position follows the end-tag opener.
type End_Tag_Position int

// End_Tag_Position_Invariants bounds end-tag progress.
func End_Tag_Position_Invariants(value End_Tag_Position, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), END_TAG_POSITION_MINIMUM, DOCUMENT_SIZE_MAXIMUM).
		Ensure()
}

// Instruction_Position follows the instruction opener.
type Instruction_Position int

// Instruction_Position_Invariants bounds instruction progress.
func Instruction_Position_Invariants(
	value Instruction_Position, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), INSTRUCTION_POSITION_MINIMUM, DOCUMENT_SIZE_MAXIMUM).
		Ensure()
}

// Directive_Position follows the directive opener.
type Directive_Position int

// Directive_Position_Invariants bounds directive progress.
func Directive_Position_Invariants(value Directive_Position, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), DIRECTIVE_POSITION_MINIMUM, DOCUMENT_SIZE_MAXIMUM).
		Ensure()
}

// Comment_Position follows the comment opener.
type Comment_Position int

// Comment_Position_Invariants bounds comment progress.
func Comment_Position_Invariants(value Comment_Position, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), COMMENT_POSITION_MINIMUM, DOCUMENT_SIZE_MAXIMUM).
		Ensure()
}

// CDATA_Position follows the CDATA opener.
type CDATA_Position int

// CDATA_Position_Invariants bounds CDATA progress.
func CDATA_Position_Invariants(value CDATA_Position, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), CDATA_POSITION_MINIMUM, DOCUMENT_SIZE_MAXIMUM).
		Ensure()
}

// Directive_Comment_Position follows a nested comment opener.
type Directive_Comment_Position int

// Directive_Comment_Position_Invariants bounds nested comment progress.
func Directive_Comment_Position_Invariants(
	value Directive_Comment_Position, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), DIRECTIVE_COMMENT_POSITION_MINIMUM,
			DOCUMENT_SIZE_MAXIMUM,
		).
		Ensure()
}

// XML_Character is one decoded scalar considered by XML grammar.
type XML_Character int32

// XML_Character_Invariants covers the decoded scalar domain across parser stages.
func XML_Character_Invariants(value XML_Character, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), int32(bytes.SLICE_SIZE_MINIMUM), int32(utf8.RUNE_MAX)).
		Ensure()
}

// Character_Size is one decoded UTF-8 width.
type Character_Size int

// Character_Size_Invariants covers all decoded widths across transform stages.
func Character_Size_Invariants(value Character_Size, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Int(
			int(value), utf8.CHARACTER_SIZE_MINIMUM, utf8.CHARACTER_SIZE_TWO,
			utf8.CHARACTER_SIZE_THREE, utf8.CHARACTER_SIZE_MAXIMUM,
		).
		Ensure()
}

// Pattern is one fixed XML token searched inside a document.
type Pattern string

// Pattern_Invariants covers every fixed token width used by the parser.
func Pattern_Invariants(value Pattern, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), PATTERN_SIZE_MINIMUM, PATTERN_SIZE_MAXIMUM).
		Ensure()
}

// XML_Byte is one source byte considered for XML space.
type XML_Byte uint8

// XML_Byte_Invariants covers the complete wire-byte domain.
func XML_Byte_Invariants(value XML_Byte, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), uint8(bits.WORD_8_MINIMUM), uint8(bits.WORD_8_MAXIMUM)).
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

// Name_Starts retains one opening-name start per possible document depth.
type Name_Starts []int

// Name_Starts_Invariants fixes parser storage independently of input shape.
func Name_Starts_Invariants(value Name_Starts, _ aver.Namespace) {
	aver.Always(len(value) == DEPTH_MAXIMUM, "Every element depth has a name start slot.")
}

// Name_Ends retains one opening-name end per possible document depth.
type Name_Ends []int

// Name_Ends_Invariants fixes parser storage independently of input shape.
func Name_Ends_Invariants(value Name_Ends, _ aver.Namespace) {
	aver.Always(len(value) == DEPTH_MAXIMUM, "Every element depth has a name end slot.")
}

// Depth retains nesting inside fixed element-name storage.
type Depth int

// Depth_Invariants keeps nesting inside fixed element-name storage.
func Depth_Invariants(value Depth, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), bytes.SLICE_SIZE_MINIMUM, DEPTH_MAXIMUM).
		Ensure()
}

// Depth_Handle retains nesting depth across element transitions.
type Depth_Handle *Depth

// Depth_Handle_Invariants composes present parser state.
func Depth_Handle_Invariants(value Depth_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Depth_Invariants(*value, namespace)
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
		calculated += int(escaped_character_size(
			XML_Character(character), Character_Size(size),
		))
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
	if required > 0 {
		escape_unchecked(Nonempty_Output(destination), Nonempty_Text(source))
	}
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
	entity_output := Output(entity_storage[:])
	document := Document(source)
	for source_position := 0; source_position < len(source); {
		if source[source_position] == '&' {
			next, error_index, entity_size, valid := parse_entity(
				Nonempty_Document(document),
				Nonempty_Output(entity_output),
				Document_Position(source_position),
			)
			if !bool(valid) {
				return 0, Position(int(error_index) + 1), STATUS_INPUT_INVALID
			}
			calculated += int(entity_size)
			source_position = int(next)
			continue
		}
		character, size := utf8.Decode_Character(utf8.Bytes(source[source_position:]))
		if !bool(decoded_xml_character(XML_Character(character), Character_Size(size))) {
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
	if required > 0 {
		unescape_unchecked(Nonempty_Output(destination), Nonempty_Escaped(source))
	}
	return required, 0, STATUS_OK
}

func escaped_character_size(
	character XML_Character, decoded_size Character_Size,
) (size Escaped_Character_Size_Count) {
	defer func() {
		Escaped_Character_Size_Count_Invariants(size, "escaped_character_size.size")
	}()
	XML_Character_Invariants(character, "escaped_character_size.character")
	Character_Size_Invariants(decoded_size, "escaped_character_size.decoded_size")
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

func escape_unchecked(destination Nonempty_Output, source Nonempty_Text) {
	Nonempty_Output_Invariants(destination, "escape_unchecked.destination")
	Nonempty_Text_Invariants(source, "escape_unchecked.source")
	output_position := Document_Index(bytes.SLICE_SIZE_MINIMUM)
	for source_position := Document_Index(0); source_position < Document_Index(len(source)); {
		character, size := utf8.Decode_Character(utf8.Bytes(source[source_position:]))
		output_position = Document_Index(write_escaped_character(
			destination, source, source_position, output_position,
			XML_Character(character), Character_Size(size),
		))
		source_position += Document_Index(size)
	}
}

func write_escaped_character(
	destination Nonempty_Output, source Nonempty_Text,
	source_position Document_Index, output_position Document_Index,
	character XML_Character, decoded_size Character_Size,
) (next Next_Position) {
	defer func() { Next_Position_Invariants(next, "write_escaped_character.next") }()
	Nonempty_Output_Invariants(destination, "write_escaped_character.destination")
	Nonempty_Text_Invariants(source, "write_escaped_character.source")
	Document_Index_Invariants(source_position, "write_escaped_character.source_position")
	Document_Index_Invariants(output_position, "write_escaped_character.output_position")
	XML_Character_Invariants(character, "write_escaped_character.character")
	Character_Size_Invariants(decoded_size, "write_escaped_character.decoded_size")
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
		return Next_Position(output_position + Document_Index(written_count))
	}
	return Next_Position(output_position + Document_Index(copy(
		destination[output_position:],
		source[source_position:Document_Index(int(source_position)+int(decoded_size))],
	)))
}

func xml_character(character XML_Character) (valid Boolean) {
	defer func() { Boolean_Invariants(valid, "xml_character.valid") }()
	XML_Character_Invariants(character, "xml_character.character")
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

func decoded_xml_character(
	character XML_Character, decoded_size Character_Size,
) (valid Boolean) {
	defer func() { Boolean_Invariants(valid, "decoded_xml_character.valid") }()
	XML_Character_Invariants(character, "decoded_xml_character.character")
	Character_Size_Invariants(decoded_size, "decoded_xml_character.decoded_size")
	if int(decoded_size) == utf8.CHARACTER_SIZE_MINIMUM {
		if int32(character) == int32(utf8.REPLACEMENT_CHARACTER) {
			return false
		}
	}
	return xml_character(character)
}

func parse_entity(
	source Nonempty_Document, destination Nonempty_Output, start Document_Position,
) (
	next Document_Position, error_index Document_Position,
	size Entity_Size_Count, valid Boolean,
) {
	defer func() {
		Document_Position_Invariants(next, "parse_entity.next")
		Document_Position_Invariants(error_index, "parse_entity.error_index")
		Entity_Size_Count_Invariants(size, "parse_entity.size")
		Boolean_Invariants(valid, "parse_entity.valid")
	}()
	Nonempty_Document_Invariants(source, "parse_entity.source")
	Nonempty_Output_Invariants(destination, "parse_entity.destination")
	Document_Position_Invariants(start, "parse_entity.start")
	semicolon := int(start) + len("&")
	for semicolon < len(source) {
		if source[semicolon] == ';' {
			break
		}
		semicolon++
	}
	if semicolon >= len(source) {
		end := Document_Position(len(source))
		return end, end, 0, false
	}
	name_start := int(start) + len("&")
	name := Entity_Name(source[name_start:semicolon])
	if bool(text_equal(name, "lt")) {
		written_next, written_size := write_entity(
			destination, XML_Character('<'), Document_Index(semicolon),
		)
		next_position := Document_Position(written_next)
		return next_position, next_position, Entity_Size_Count(written_size), true
	}
	if bool(text_equal(name, "gt")) {
		written_next, written_size := write_entity(
			destination, XML_Character('>'), Document_Index(semicolon),
		)
		next_position := Document_Position(written_next)
		return next_position, next_position, Entity_Size_Count(written_size), true
	}
	if bool(text_equal(name, "amp")) {
		written_next, written_size := write_entity(
			destination, XML_Character('&'), Document_Index(semicolon),
		)
		next_position := Document_Position(written_next)
		return next_position, next_position, Entity_Size_Count(written_size), true
	}
	if bool(text_equal(name, "apos")) {
		written_next, written_size := write_entity(
			destination, XML_Character('\''), Document_Index(semicolon),
		)
		next_position := Document_Position(written_next)
		return next_position, next_position, Entity_Size_Count(written_size), true
	}
	if bool(text_equal(name, "quot")) {
		written_next, written_size := write_entity(
			destination, XML_Character('"'), Document_Index(semicolon),
		)
		next_position := Document_Position(written_next)
		return next_position, next_position, Entity_Size_Count(written_size), true
	}
	if name_start == semicolon {
		return start, start, 0, false
	}
	if source[name_start] != '#' {
		return start, start, 0, false
	}
	return parse_numeric_entity(
		Numeric_Entity_Document(source), destination,
		Document_Position(name_start+1), Document_Position(semicolon),
	)
}

func parse_numeric_entity(
	source Numeric_Entity_Document, destination Nonempty_Output,
	start Document_Position, semicolon Document_Position,
) (next Document_Position, error_index Document_Position, size Entity_Size_Count, valid Boolean) {
	defer func() {
		Document_Position_Invariants(next, "parse_numeric_entity.next")
		Document_Position_Invariants(error_index, "parse_numeric_entity.error_index")
		Entity_Size_Count_Invariants(size, "parse_numeric_entity.size")
		Boolean_Invariants(valid, "parse_numeric_entity.valid")
	}()
	Numeric_Entity_Document_Invariants(source, "parse_numeric_entity.source")
	Nonempty_Output_Invariants(destination, "parse_numeric_entity.destination")
	Document_Position_Invariants(start, "parse_numeric_entity.start")
	Document_Position_Invariants(semicolon, "parse_numeric_entity.semicolon")
	base := uint32(10)
	position := int(start)
	if position < int(semicolon) {
		if source[position] == 'x' {
			base = 16
			position++
		}
	}
	if position == int(semicolon) {
		return start, start, 0, false
	}
	var character uint32
	for position < int(semicolon) {
		value, failure_position := source[position], Document_Position(position)
		var digit uint32
		if value >= '0' {
			if value <= '9' {
				digit = uint32(value - '0')
			} else if base == 16 {
				if value >= 'a' {
					if value <= 'f' {
						digit = uint32(value-'a') + 10
					} else {
						return failure_position, failure_position, 0, false
					}
				} else if value >= 'A' {
					if value <= 'F' {
						digit = uint32(value-'A') + 10
					} else {
						return failure_position, failure_position, 0, false
					}
				} else {
					return failure_position, failure_position, 0, false
				}
			} else {
				return failure_position, failure_position, 0, false
			}
		} else {
			return failure_position, failure_position, 0, false
		}
		if character > (uint32(utf8.RUNE_MAX)-digit)/base {
			return failure_position, failure_position, 0, false
		}
		character = character*base + digit
		position++
	}
	if !bool(xml_character(XML_Character(character))) {
		return start, start, 0, false
	}
	written_next, written_size := write_entity(
		destination, XML_Character(character), Document_Index(semicolon),
	)
	next = Document_Position(written_next)
	return next, next, Entity_Size_Count(written_size), true
}

func write_entity(
	destination Nonempty_Output, character XML_Character, semicolon Document_Index,
) (next Next_Position, size Encoded_Entity_Size_Count) {
	defer func() {
		Next_Position_Invariants(next, "write_entity.next")
		Encoded_Entity_Size_Count_Invariants(size, "write_entity.size")
	}()
	Nonempty_Output_Invariants(destination, "write_entity.destination")
	XML_Character_Invariants(character, "write_entity.character")
	Document_Index_Invariants(semicolon, "write_entity.semicolon")
	character_size := utf8.Character_Size(utf8.Character(character))
	size = Encoded_Entity_Size_Count(character_size)
	if len(destination) >= int(size) {
		utf8.Encode_Character(
			utf8.Bytes(destination[:size]), utf8.Character(character),
		)
	}
	return Next_Position(semicolon + Document_Index(len(";"))), size
}

func text_equal(source Entity_Name, expected Pattern) (equal Boolean) {
	defer func() { Boolean_Invariants(equal, "text_equal.equal") }()
	Entity_Name_Invariants(source, "text_equal.source")
	Pattern_Invariants(expected, "text_equal.expected")
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

func unescape_unchecked(destination Nonempty_Output, source Nonempty_Escaped) {
	Nonempty_Output_Invariants(destination, "unescape_unchecked.destination")
	Nonempty_Escaped_Invariants(source, "unescape_unchecked.source")
	document := Document(source)
	output_position := bytes.SLICE_SIZE_MINIMUM
	for source_position := 0; source_position < len(source); {
		if source[source_position] == '&' {
			entity_destination := Nonempty_Output(destination[output_position:])
			next, _, entity_size, _ := parse_entity(
				Nonempty_Document(document), entity_destination,
				Document_Position(source_position),
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

func validate_unchecked(source Document) (position Position) {
	defer func() { Position_Invariants(position, "validate_unchecked.position") }()
	Document_Invariants(source, "validate_unchecked.source")
	var name_start_storage, name_end_storage [DEPTH_MAXIMUM]int
	name_starts := Name_Starts(name_start_storage[:])
	name_ends := Name_Ends(name_end_storage[:])
	index := bytes.SLICE_SIZE_MINIMUM
	depth := Depth(bytes.SLICE_SIZE_MINIMUM)
	root_seen := false
	root_closed := false
	for index < len(source) {
		nonempty_source := Nonempty_Document(source)
		if source[index] != '<' {
			next, error_index, valid := scan_document_text(
				nonempty_source, Document_Position(index), depth,
			)
			if !bool(valid) {
				return Position(int(error_index) + 1)
			}
			index = int(next)
			continue
		}
		next, error_index, matched, valid := scan_document_special(
			nonempty_source, Document_Position(index), depth,
		)
		if matched {
			if !bool(valid) {
				return Position(int(error_index) + 1)
			}
			index = int(next)
			continue
		}
		if bool(prefix_at(nonempty_source, Document_Position(index), "</")) {
			close_next, close_error, closed, close_valid := close_document_element(
				Special_Document(source), Document_Position(index),
				name_starts, name_ends, Depth_Handle(&depth),
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
		open_next, open_error, closed, valid := open_document_element(
			nonempty_source, Document_Position(index), name_starts, name_ends,
			Depth_Handle(&depth),
		)
		if !bool(valid) {
			return Position(int(open_error) + 1)
		}
		root_seen = true
		index = int(open_next)
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

func scan_document_text(
	source Nonempty_Document, start Document_Position, depth Depth,
) (next Document_Position, error_index Document_Position, valid Boolean) {
	defer func() {
		Document_Position_Invariants(next, "scan_document_text.next")
		Document_Position_Invariants(error_index, "scan_document_text.error_index")
		Boolean_Invariants(valid, "scan_document_text.valid")
	}()
	Nonempty_Document_Invariants(source, "scan_document_text.source")
	Document_Position_Invariants(start, "scan_document_text.start")
	Depth_Invariants(depth, "scan_document_text.depth")
	next, error_index, valid = scan_text(source, start)
	if !valid {
		return error_index, error_index, false
	}
	if depth != bytes.SLICE_SIZE_MINIMUM {
		return next, next, true
	}
	for outside := start; outside < next; outside++ {
		if !bool(xml_space(XML_Byte(source[outside]))) {
			return outside, outside, false
		}
	}
	return next, next, true
}

func scan_document_special(
	source Nonempty_Document, start Document_Position, depth Depth,
) (
	next Document_Position, error_index Document_Position,
	matched Boolean, valid Boolean,
) {
	defer func() {
		Document_Position_Invariants(next, "scan_document_special.next")
		Document_Position_Invariants(error_index, "scan_document_special.error_index")
		Boolean_Invariants(matched, "scan_document_special.matched")
		Boolean_Invariants(valid, "scan_document_special.valid")
	}()
	Nonempty_Document_Invariants(source, "scan_document_special.source")
	Document_Position_Invariants(start, "scan_document_special.start")
	Depth_Invariants(depth, "scan_document_special.depth")
	if bool(prefix_at(source, start, "<!--")) {
		comment_next, comment_error, comment_valid := scan_comment(
			Comment_Document(source), start,
		)
		return Document_Position(comment_next), Document_Position(comment_error),
			true, Boolean(comment_valid)
	}
	if bool(prefix_at(source, start, "<?")) {
		instruction_next, instruction_error, instruction_valid := scan_instruction(
			Special_Document(source), start,
		)
		return Document_Position(instruction_next),
			Document_Position(instruction_error), true, Boolean(instruction_valid)
	}
	if bool(prefix_at(source, start, "<![CDATA[")) {
		if depth == bytes.SLICE_SIZE_MINIMUM {
			return start, start, true, false
		}
		cdata_next, cdata_error, cdata_valid := scan_cdata(CDATA_Document(source), start)
		return Document_Position(cdata_next), Document_Position(cdata_error),
			true, Boolean(cdata_valid)
	}
	if bool(prefix_at(source, start, "<![")) {
		return start, start, true, false
	}
	if bool(prefix_at(source, start, "<!-")) {
		return start, start, true, false
	}
	if bool(prefix_at(source, start, "<!")) {
		directive_next, directive_error, directive_valid := scan_directive(
			Special_Document(source), start,
		)
		return Document_Position(directive_next), Document_Position(directive_error),
			true, Boolean(directive_valid)
	}
	return start, start, false, true
}

func close_document_element(
	source Special_Document, start Document_Position, name_starts Name_Starts,
	name_ends Name_Ends, depth Depth_Handle,
) (
	next Document_Position, error_index Document_Position,
	closed Boolean, valid Boolean,
) {
	defer func() {
		Document_Position_Invariants(next, "close_document_element.next")
		Document_Position_Invariants(error_index, "close_document_element.error_index")
		Boolean_Invariants(closed, "close_document_element.closed")
		Boolean_Invariants(valid, "close_document_element.valid")
	}()
	Special_Document_Invariants(source, "close_document_element.source")
	Document_Position_Invariants(start, "close_document_element.start")
	Name_Starts_Invariants(name_starts, "close_document_element.name_starts")
	Name_Ends_Invariants(name_ends, "close_document_element.name_ends")
	Depth_Handle_Invariants(depth, "close_document_element.depth")
	if *depth == bytes.SLICE_SIZE_MINIMUM {
		return start, start, false, false
	}
	end_next, name_start, name_end, end_error, valid := scan_end_tag(
		End_Tag_Document(source), start,
	)
	if !valid {
		failure_position := Document_Position(end_error)
		return failure_position, failure_position, false, false
	}
	*depth--
	if !bytes.Equal(
		bytes.Slice(source[name_start:name_end]),
		bytes.Slice(source[name_starts[*depth]:name_ends[*depth]]),
	) {
		position := Document_Position(name_start)
		return position, position, false, false
	}
	next = Document_Position(end_next)
	return next, next, Boolean(*depth == bytes.SLICE_SIZE_MINIMUM), true
}

func open_document_element(
	source Nonempty_Document, start Document_Position, name_starts Name_Starts,
	name_ends Name_Ends, depth Depth_Handle,
) (
	next Start_Tag_Position, error_index Start_Tag_Position,
	closed Boolean, valid Boolean,
) {
	defer func() {
		Start_Tag_Position_Invariants(next, "open_document_element.next")
		Start_Tag_Position_Invariants(error_index, "open_document_element.error_index")
		Boolean_Invariants(closed, "open_document_element.closed")
		Boolean_Invariants(valid, "open_document_element.valid")
	}()
	Nonempty_Document_Invariants(source, "open_document_element.source")
	Document_Position_Invariants(start, "open_document_element.start")
	Name_Starts_Invariants(name_starts, "open_document_element.name_starts")
	Name_Ends_Invariants(name_ends, "open_document_element.name_ends")
	Depth_Handle_Invariants(depth, "open_document_element.depth")
	next, name_start, name_end, error_index, empty, valid := scan_start_tag(
		source, start,
	)
	if !valid {
		return error_index, error_index, false, false
	}
	if empty {
		return next, next, Boolean(*depth == bytes.SLICE_SIZE_MINIMUM), true
	}
	if int(*depth) == DEPTH_MAXIMUM {
		position := Start_Tag_Position(start)
		return position, position, false, false
	}
	name_starts[*depth] = int(name_start)
	name_ends[*depth] = int(name_end)
	*depth++
	return next, next, false, true
}

func scan_text(
	source Nonempty_Document, start Document_Position,
) (next Document_Position, error_index Document_Position, valid Boolean) {
	defer func() {
		Document_Position_Invariants(next, "scan_text.next")
		Document_Position_Invariants(error_index, "scan_text.error_index")
		Boolean_Invariants(valid, "scan_text.valid")
	}()
	Nonempty_Document_Invariants(source, "scan_text.source")
	Document_Position_Invariants(start, "scan_text.start")
	index := start
	var entity_storage [utf8.CHARACTER_SIZE_MAXIMUM]byte
	entity_output := Output(entity_storage[:])
	for index < Document_Position(len(source)) {
		if source[index] == '<' {
			return index, index, true
		}
		if bool(prefix_at(source, index, "]]>")) {
			return index, index, false
		}
		if source[index] == '&' {
			entity_next, entity_error, _, entity_valid := parse_entity(
				source, Nonempty_Output(entity_output), index,
			)
			if !bool(entity_valid) {
				return entity_error, entity_error, false
			}
			index = entity_next
			continue
		}
		character, size := utf8.Decode_Character(utf8.Bytes(source[index:]))
		if !bool(decoded_xml_character(XML_Character(character), Character_Size(size))) {
			return index, index, false
		}
		index += Document_Position(size)
	}
	return index, index, true
}

func scan_start_tag(
	source Nonempty_Document, start Document_Position,
) (
	next Start_Tag_Position, name_start Start_Tag_Position,
	name_end Start_Tag_Position, error_index Start_Tag_Position,
	empty Boolean, valid Boolean,
) {
	defer func() {
		Start_Tag_Position_Invariants(next, "scan_start_tag.next")
		Start_Tag_Position_Invariants(name_start, "scan_start_tag.name_start")
		Start_Tag_Position_Invariants(name_end, "scan_start_tag.name_end")
		Start_Tag_Position_Invariants(error_index, "scan_start_tag.error_index")
		Boolean_Invariants(empty, "scan_start_tag.empty")
		Boolean_Invariants(valid, "scan_start_tag.valid")
	}()
	Nonempty_Document_Invariants(source, "scan_start_tag.source")
	Document_Position_Invariants(start, "scan_start_tag.start")
	if start >= Document_Position(len(source)) {
		end := Start_Tag_Position(len(source))
		return end, end, end, end, false, false
	}
	name_start = Start_Tag_Position(start + Document_Position(len("<")))
	qualified_end, qualified_error, qualified_valid := scan_qualified_name(
		source, Document_Position(name_start),
	)
	name_end = Start_Tag_Position(qualified_end)
	error_index = Start_Tag_Position(qualified_error)
	valid = qualified_valid
	if !valid {
		return error_index, error_index, error_index, error_index, false, false
	}
	next, error_index, empty, valid = scan_start_tag_attributes(source, name_end)
	if !valid {
		return error_index, error_index, error_index, error_index, false, false
	}
	return next, name_start, name_end, error_index, empty, true
}

func scan_start_tag_attributes(
	source Nonempty_Document, index Start_Tag_Position,
) (next Start_Tag_Position, error_index Start_Tag_Position, empty Boolean, valid Boolean) {
	defer func() {
		Start_Tag_Position_Invariants(next, "scan_start_tag_attributes.next")
		Start_Tag_Position_Invariants(error_index, "scan_start_tag_attributes.error_index")
		Boolean_Invariants(empty, "scan_start_tag_attributes.empty")
		Boolean_Invariants(valid, "scan_start_tag_attributes.valid")
	}()
	Nonempty_Document_Invariants(source, "scan_start_tag_attributes.source")
	Start_Tag_Position_Invariants(index, "scan_start_tag_attributes.index")
	var attribute_starts, attribute_ends [ATTRIBUTE_COUNT_MAXIMUM]int
	attribute_count := bytes.SLICE_SIZE_MINIMUM
	position := Document_Position(index)
	for position < Document_Position(len(source)) {
		if source[position] == '>' {
			next_position := Start_Tag_Position(position + 1)
			return next_position, next_position, false, true
		}
		if source[position] == '/' {
			if position+1 < Document_Position(len(source)) {
				if source[position+1] == '>' {
					next_position := Start_Tag_Position(position + 2)
					return next_position, next_position, true, true
				}
			}
			failure_position := Start_Tag_Position(position)
			return failure_position, failure_position, false, false
		}
		if !bool(xml_space(XML_Byte(source[position]))) {
			failure_position := Start_Tag_Position(position)
			return failure_position, failure_position, false, false
		}
		position = skip_xml_space(Space_Document(source), position)
		if position == Document_Position(len(source)) {
			failure_position := Start_Tag_Position(position)
			return failure_position, failure_position, false, false
		}
		if source[position] == '>' {
			next_position := Start_Tag_Position(position + 1)
			return next_position, next_position, false, true
		}
		if source[position] == '/' {
			continue
		}
		a_next, a_start, a_end, a_error, a_valid := scan_attribute(
			Attribute_Document(source), position,
		)
		if !a_valid {
			failure_position := Start_Tag_Position(a_error)
			return failure_position, failure_position, false, false
		}
		for previous_index := 0; previous_index < attribute_count; previous_index++ {
			previous_start, previous_end :=
				attribute_starts[previous_index], attribute_ends[previous_index]
			attribute := bytes.Slice(source[a_start:a_end])
			previous := bytes.Slice(source[previous_start:previous_end])
			if bytes.Equal(attribute, previous) {
				failure_position := Start_Tag_Position(a_start)
				return failure_position, failure_position, false, false
			}
		}
		attribute_starts[attribute_count], attribute_ends[attribute_count] =
			int(a_start), int(a_end)
		attribute_count++
		position = a_next
	}
	end := Start_Tag_Position(len(source))
	return end, end, false, false
}

func scan_attribute(
	source Attribute_Document, start Document_Position,
) (
	next Document_Position, name_start Document_Position,
	name_end Document_Position, error_index Document_Position, valid Boolean,
) {
	defer func() {
		Document_Position_Invariants(next, "scan_attribute.next")
		Document_Position_Invariants(name_start, "scan_attribute.name_start")
		Document_Position_Invariants(name_end, "scan_attribute.name_end")
		Document_Position_Invariants(error_index, "scan_attribute.error_index")
		Boolean_Invariants(valid, "scan_attribute.valid")
	}()
	Attribute_Document_Invariants(source, "scan_attribute.source")
	Document_Position_Invariants(start, "scan_attribute.start")
	name_start = start
	name_end, error_index, valid = scan_qualified_name(Nonempty_Document(source), start)
	if !valid {
		return error_index, error_index, error_index, error_index, false
	}
	index := skip_xml_space(Space_Document(source), name_end)
	if index >= Document_Position(len(source)) {
		end := Document_Position(len(source))
		return end, end, end, end, false
	}
	if source[index] != '=' {
		return index, index, index, index, false
	}
	index = skip_xml_space(Space_Document(source), index+1)
	if index == Document_Position(len(source)) {
		return index, index, index, index, false
	}
	quote := source[index]
	if quote != '\'' {
		if quote != '"' {
			return index, index, index, index, false
		}
	}
	index++
	var entity_storage [utf8.CHARACTER_SIZE_MAXIMUM]byte
	entity_output := Output(entity_storage[:])
	for index < Document_Position(len(source)) {
		if source[index] == quote {
			next_position := index + 1
			return next_position, name_start, name_end, next_position, true
		}
		if source[index] == '<' {
			return index, index, index, index, false
		}
		if source[index] == '&' {
			entity_next, entity_error, _, entity_valid := parse_entity(
				Nonempty_Document(source), Nonempty_Output(entity_output), index,
			)
			if !bool(entity_valid) {
				return entity_error, entity_error, entity_error, entity_error, false
			}
			index = entity_next
			continue
		}
		character, size := utf8.Decode_Character(utf8.Bytes(source[index:]))
		if !bool(decoded_xml_character(XML_Character(character), Character_Size(size))) {
			return index, index, index, index, false
		}
		index += Document_Position(size)
	}
	end := Document_Position(len(source))
	return end, end, end, end, false
}

func scan_end_tag(
	source End_Tag_Document, start Document_Position,
) (
	next End_Tag_Position, name_start End_Tag_Position,
	name_end End_Tag_Position, error_index End_Tag_Position, valid Boolean,
) {
	defer func() {
		End_Tag_Position_Invariants(next, "scan_end_tag.next")
		End_Tag_Position_Invariants(name_start, "scan_end_tag.name_start")
		End_Tag_Position_Invariants(name_end, "scan_end_tag.name_end")
		End_Tag_Position_Invariants(error_index, "scan_end_tag.error_index")
		Boolean_Invariants(valid, "scan_end_tag.valid")
	}()
	End_Tag_Document_Invariants(source, "scan_end_tag.source")
	Document_Position_Invariants(start, "scan_end_tag.start")
	if start >= Document_Position(len(source)) {
		end := End_Tag_Position(len(source))
		return end, end, end, end, false
	}
	name_start = End_Tag_Position(start + Document_Position(len("</")))
	qualified_end, qualified_error, qualified_valid := scan_qualified_name(
		Nonempty_Document(source), Document_Position(name_start),
	)
	name_end = End_Tag_Position(qualified_end)
	error_index = End_Tag_Position(qualified_error)
	valid = qualified_valid
	if !valid {
		return error_index, error_index, error_index, error_index, false
	}
	index := skip_xml_space(Space_Document(source), Document_Position(name_end))
	if index == Document_Position(len(source)) {
		position := End_Tag_Position(index)
		return position, position, position, position, false
	}
	if source[index] != '>' {
		position := End_Tag_Position(index)
		return position, position, position, position, false
	}
	next = End_Tag_Position(index + 1)
	return next, name_start, name_end, next, true
}

func scan_name(
	source Nonempty_Document, start Document_Position,
) (next Document_Position, error_index Document_Position, valid Boolean) {
	defer func() {
		Document_Position_Invariants(next, "scan_name.next")
		Document_Position_Invariants(error_index, "scan_name.error_index")
		Boolean_Invariants(valid, "scan_name.valid")
	}()
	Nonempty_Document_Invariants(source, "scan_name.source")
	Document_Position_Invariants(start, "scan_name.start")
	index := start
	if index >= Document_Position(len(source)) {
		end := Document_Position(len(source))
		return end, end, false
	}
	character, size := utf8.Decode_Character(utf8.Bytes(source[index:]))
	if !bool(decoded_xml_character(XML_Character(character), Character_Size(size))) {
		return index, index, false
	}
	if !bool(name_start_character(XML_Character(character))) {
		return index, index, false
	}
	index += Document_Position(size)
	for index < Document_Position(len(source)) {
		if source[index] < byte(utf8.CHARACTER_SELF) {
			if !bool(name_character(XML_Character(source[index]))) {
				break
			}
			index++
			continue
		}
		character, size = utf8.Decode_Character(utf8.Bytes(source[index:]))
		if !bool(decoded_xml_character(XML_Character(character), Character_Size(size))) {
			return index, index, false
		}
		if !bool(name_character(XML_Character(character))) {
			break
		}
		index += Document_Position(size)
	}
	return index, index, true
}

func scan_qualified_name(
	source Nonempty_Document, start Document_Position,
) (next Document_Position, error_index Document_Position, valid Boolean) {
	defer func() {
		Document_Position_Invariants(next, "scan_qualified_name.next")
		Document_Position_Invariants(error_index, "scan_qualified_name.error_index")
		Boolean_Invariants(valid, "scan_qualified_name.valid")
	}()
	Nonempty_Document_Invariants(source, "scan_qualified_name.source")
	Document_Position_Invariants(start, "scan_qualified_name.start")
	next, error_index, valid = scan_name(source, start)
	if !bool(valid) {
		return error_index, error_index, false
	}
	colon_seen := false
	for index := start; index < next; index++ {
		if source[index] != ':' {
			continue
		}
		if colon_seen {
			return index, index, false
		}
		colon_seen = true
	}
	return next, next, true
}

func scan_comment(
	source Comment_Document, start Document_Position,
) (next Comment_Position, error_index Comment_Position, valid Boolean) {
	defer func() {
		Comment_Position_Invariants(next, "scan_comment.next")
		Comment_Position_Invariants(error_index, "scan_comment.error_index")
		Boolean_Invariants(valid, "scan_comment.valid")
	}()
	Comment_Document_Invariants(source, "scan_comment.source")
	Document_Position_Invariants(start, "scan_comment.start")
	index := start + Document_Position(len("<!--"))
	for index < Document_Position(len(source)) {
		if bool(prefix_at(Nonempty_Document(source), index, "-->")) {
			next_position := Comment_Position(index + Document_Position(len("-->")))
			return next_position, next_position, true
		}
		if bool(prefix_at(Nonempty_Document(source), index, "--")) {
			position := Comment_Position(index)
			return position, position, false
		}
		index++
	}
	end := Comment_Position(len(source))
	return end, end, false
}

func scan_instruction(
	source Special_Document, start Document_Position,
) (next Instruction_Position, error_index Instruction_Position, valid Boolean) {
	defer func() {
		Instruction_Position_Invariants(next, "scan_instruction.next")
		Instruction_Position_Invariants(error_index, "scan_instruction.error_index")
		Boolean_Invariants(valid, "scan_instruction.valid")
	}()
	Special_Document_Invariants(source, "scan_instruction.source")
	Document_Position_Invariants(start, "scan_instruction.start")
	if start >= Document_Position(len(source)) {
		end := Instruction_Position(len(source))
		return end, end, false
	}
	name_start := start + Document_Position(len("<?"))
	name_end, name_error, valid := scan_name(Nonempty_Document(source), name_start)
	if !valid {
		position := Instruction_Position(name_error)
		return position, position, false
	}
	index := name_end
	for index < Document_Position(len(source)) {
		if bool(prefix_at(Nonempty_Document(source), index, "?>")) {
			instruction_error, instruction_valid := xml_instruction_valid(
				Terminated_Instruction_Document(source),
				name_start,
				name_end,
				index,
			)
			if !bool(instruction_valid) {
				position := Instruction_Position(instruction_error)
				return position, position, false
			}
			next_position := Instruction_Position(index + Document_Position(len("?>")))
			return next_position, next_position, true
		}
		index++
	}
	end := Instruction_Position(len(source))
	return end, end, false
}

func xml_instruction_valid(
	source Terminated_Instruction_Document, name_start Document_Position,
	name_end Document_Position, data_end Document_Position,
) (error_index Document_Position, valid Boolean) {
	defer func() {
		Document_Position_Invariants(error_index, "xml_instruction_valid.error_index")
		Boolean_Invariants(valid, "xml_instruction_valid.valid")
	}()
	Terminated_Instruction_Document_Invariants(source, "xml_instruction_valid.source")
	Document_Position_Invariants(name_start, "xml_instruction_valid.name_start")
	Document_Position_Invariants(name_end, "xml_instruction_valid.name_end")
	Document_Position_Invariants(data_end, "xml_instruction_valid.data_end")
	if name_end-name_start != Document_Position(len("xml")) {
		return name_start, true
	}
	if !bool(prefix_at(Nonempty_Document(source), name_start, "xml")) {
		return name_start, true
	}
	value_start, value_end, found := instruction_parameter(
		XML_Instruction_Document(source), name_end, data_end, "version=",
	)
	if bool(found) {
		if value_start != value_end {
			if value_end-value_start != Document_Position(len("1.0")) {
				return value_start, false
			}
			if !bool(prefix_at(Nonempty_Document(source), value_start, "1.0")) {
				return value_start, false
			}
		}
	}
	value_start, value_end, found = instruction_parameter(
		XML_Instruction_Document(source), name_end, data_end, "encoding=",
	)
	if bool(found) {
		if value_start != value_end {
			var utf8_name = "utf-8"
			if value_end-value_start != Document_Position(len(utf8_name)) {
				return value_start, false
			}
			for index := range len(utf8_name) {
				value := source[value_start+Document_Position(index)]
				if value >= 'A' {
					if value <= 'Z' {
						value += 'a' - 'A'
					}
				}
				if value != utf8_name[index] {
					return value_start, false
				}
			}
		}
	}
	return name_start, true
}

func instruction_parameter(
	source XML_Instruction_Document, start Document_Position,
	end Document_Position, parameter Pattern,
) (value_start Document_Position, value_end Document_Position, found Boolean) {
	defer func() {
		Document_Position_Invariants(value_start, "instruction_parameter.value_start")
		Document_Position_Invariants(value_end, "instruction_parameter.value_end")
		Boolean_Invariants(found, "instruction_parameter.found")
	}()
	XML_Instruction_Document_Invariants(source, "instruction_parameter.source")
	Document_Position_Invariants(start, "instruction_parameter.start")
	Document_Position_Invariants(end, "instruction_parameter.end")
	Pattern_Invariants(parameter, "instruction_parameter.parameter")
	for index := start; index+Document_Position(len(parameter)) < end; index++ {
		if !bool(prefix_at(Nonempty_Document(source), index, parameter)) {
			continue
		}
		quote_index := index + Document_Position(len(parameter))
		quote := source[quote_index]
		if quote != '\'' {
			if quote != '"' {
				continue
			}
		}
		value_start = quote_index + 1
		for value_index := value_start; value_index < end; value_index++ {
			if source[value_index] == quote {
				return value_start, value_index, true
			}
		}
	}
	return start, end, false
}

func scan_cdata(
	source CDATA_Document, start Document_Position,
) (next CDATA_Position, error_index CDATA_Position, valid Boolean) {
	defer func() {
		CDATA_Position_Invariants(next, "scan_cdata.next")
		CDATA_Position_Invariants(error_index, "scan_cdata.error_index")
		Boolean_Invariants(valid, "scan_cdata.valid")
	}()
	CDATA_Document_Invariants(source, "scan_cdata.source")
	Document_Position_Invariants(start, "scan_cdata.start")
	index := start + Document_Position(len("<![CDATA["))
	for index < Document_Position(len(source)) {
		if bool(prefix_at(Nonempty_Document(source), index, "]]>")) {
			next_position := CDATA_Position(index + Document_Position(len("]]>")))
			return next_position, next_position, true
		}
		character, size := utf8.Decode_Character(utf8.Bytes(source[index:]))
		if !bool(decoded_xml_character(XML_Character(character), Character_Size(size))) {
			position := CDATA_Position(index)
			return position, position, false
		}
		index += Document_Position(size)
	}
	end := CDATA_Position(len(source))
	return end, end, false
}

func scan_directive(
	source Special_Document, start Document_Position,
) (next Directive_Position, error_index Directive_Position, valid Boolean) {
	defer func() {
		Directive_Position_Invariants(next, "scan_directive.next")
		Directive_Position_Invariants(error_index, "scan_directive.error_index")
		Boolean_Invariants(valid, "scan_directive.valid")
	}()
	Special_Document_Invariants(source, "scan_directive.source")
	Document_Position_Invariants(start, "scan_directive.start")
	index := start + Document_Position(len("<!"))
	if index == Document_Position(len(source)) {
		position := Directive_Position(index)
		return position, position, false
	}
	// Encoding/xml dispatch consumes first directive byte before nesting starts.
	index++
	quote := byte(0)
	depth := bytes.SLICE_SIZE_MINIMUM
	for index < Document_Position(len(source)) {
		if quote != 0 {
			if source[index] == quote {
				quote = 0
				index++
				continue
			}
			index++
			continue
		}
		if bool(prefix_at(Nonempty_Document(source), index, "<!--")) {
			comment_next, comment_error, comment_valid := scan_directive_comment(
				Directive_Comment_Document(source), index,
			)
			if !bool(comment_valid) {
				position := Directive_Position(comment_error)
				return position, position, false
			}
			index = Document_Position(comment_next)
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
				next_position := Directive_Position(index + 1)
				return next_position, next_position, true
			}
			depth--
			index++
			continue
		}
		index++
	}
	end := Directive_Position(len(source))
	return end, end, false
}

func scan_directive_comment(
	source Directive_Comment_Document, start Document_Position,
) (
	next Directive_Comment_Position, error_index Directive_Comment_Position,
	valid Boolean,
) {
	defer func() {
		Directive_Comment_Position_Invariants(next, "scan_directive_comment.next")
		Directive_Comment_Position_Invariants(
			error_index, "scan_directive_comment.error_index",
		)
		Boolean_Invariants(valid, "scan_directive_comment.valid")
	}()
	Directive_Comment_Document_Invariants(source, "scan_directive_comment.source")
	Document_Position_Invariants(start, "scan_directive_comment.start")
	index := start + Document_Position(len("<!--"))
	for ; index < Document_Position(len(source)); index++ {
		if bool(prefix_at(Nonempty_Document(source), index, "-->")) {
			next_position := Directive_Comment_Position(
				index + Document_Position(len("-->")),
			)
			return next_position, next_position, true
		}
	}
	end := Directive_Comment_Position(len(source))
	return end, end, false
}

func prefix_at(
	source Nonempty_Document, start Document_Position, prefix Pattern,
) (matches Boolean) {
	defer func() { Boolean_Invariants(matches, "prefix_at.matches") }()
	Nonempty_Document_Invariants(source, "prefix_at.source")
	Document_Position_Invariants(start, "prefix_at.start")
	Pattern_Invariants(prefix, "prefix_at.prefix")
	if start+Document_Position(len(prefix)) > Document_Position(len(source)) {
		return false
	}
	for index := range len(prefix) {
		if source[start+Document_Position(index)] != prefix[index] {
			return false
		}
	}
	return true
}

func skip_xml_space(
	source Space_Document, start Document_Position,
) (next Document_Position) {
	defer func() { Document_Position_Invariants(next, "skip_xml_space.next") }()
	Space_Document_Invariants(source, "skip_xml_space.source")
	Document_Position_Invariants(start, "skip_xml_space.start")
	index := start
	for index < Document_Position(len(source)) {
		if !bool(xml_space(XML_Byte(source[index]))) {
			break
		}
		index++
	}
	return index
}

func xml_space(value XML_Byte) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "xml_space.yes") }()
	XML_Byte_Invariants(value, "xml_space.value")
	switch value {
	case ' ', '\t', '\r', '\n':
		return true
	default:
		return false
	}
}

func name_start_character(character XML_Character) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "name_start_character.yes") }()
	XML_Character_Invariants(character, "name_start_character.character")
	if character < 0x200c {
		return name_start_character_lower(character)
	}
	return name_start_character_upper(character)
}

func name_start_character_lower(character XML_Character) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "name_start_character_lower.yes") }()
	XML_Character_Invariants(character, "name_start_character_lower.character")
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

func name_start_character_upper(character XML_Character) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "name_start_character_upper.yes") }()
	XML_Character_Invariants(character, "name_start_character_upper.character")
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

func name_character(character XML_Character) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "name_character.yes") }()
	XML_Character_Invariants(character, "name_character.character")
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
