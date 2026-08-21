// Package json implements bounded RFC 8259 transforms on caller-owned storage.
package json

import (
	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/sim/aver/default"
)

// ENCODED_SIZE_MAXIMUM follows shared byte-slice boundary.
const ENCODED_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// OUTPUT_SIZE_MAXIMUM prevents formatting from growing caller work without bound.
const OUTPUT_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// PREFIX_SIZE_MAXIMUM keeps every formatting input on shared byte-slice boundary.
const PREFIX_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// INDENT_SIZE_MAXIMUM keeps every formatting input on shared byte-slice boundary.
const INDENT_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// POSITION_MAXIMUM includes unexpected end immediately after maximum input.
const POSITION_MAXIMUM = ENCODED_SIZE_MAXIMUM + 1

// SOURCE_INDEX_MAXIMUM is the last addressable input byte.
const SOURCE_INDEX_MAXIMUM = ENCODED_SIZE_MAXIMUM - 1

// STRING_POSITION_MINIMUM follows one opening quote and content byte.
const STRING_POSITION_MINIMUM = VALUE_SIZE_MINIMUM + 1

// ESCAPE_POSITION_MINIMUM follows slash and escape byte.
const ESCAPE_POSITION_MINIMUM = VALUE_SIZE_MINIMUM + 2

// OBJECT_POSITION_MINIMUM follows one object grammar byte.
const OBJECT_POSITION_MINIMUM = VALUE_SIZE_MINIMUM + 1

// LINE_POSITION_MINIMUM follows one structural byte and newline.
const LINE_POSITION_MINIMUM = VALUE_SIZE_MINIMUM + 1

// ARRAY_CONTAINER_KIND is an array opener.
const ARRAY_CONTAINER_KIND Container_Kind = '['

// OBJECT_CONTAINER_KIND is an object opener.
const OBJECT_CONTAINER_KIND Container_Kind = '{'

// NESTING_DEPTH_MAXIMUM follows maximum complete empty-container nesting.
const NESTING_DEPTH_MAXIMUM = ENCODED_SIZE_MAXIMUM / 2

// VALUE_SIZE_MINIMUM is the shortest complete JSON value.
const VALUE_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM + 1

// COMPOUND_VALUE_SIZE_MINIMUM reaches a second byte after one opening marker.
const COMPOUND_VALUE_SIZE_MINIMUM = VALUE_SIZE_MINIMUM + 1

// FORMATTED_OUTPUT_SIZE_MINIMUM is one value wrapped by two line breaks.
const FORMATTED_OUTPUT_SIZE_MINIMUM = len("[\n0\n]")

// FORMATTING_AFFIX_SIZE_MAXIMUM leaves room for two affixes and minimum structure.
const FORMATTING_AFFIX_SIZE_MAXIMUM = (OUTPUT_SIZE_MAXIMUM - FORMATTED_OUTPUT_SIZE_MINIMUM) / 2

// STATUS_OK means operation completed.
const STATUS_OK = 0

// STATUS_INPUT_INVALID means source is not one complete JSON value.
const STATUS_INPUT_INVALID = STATUS_OK + 1

// STATUS_OUTPUT_TOO_SMALL means caller output cannot hold exact result.
const STATUS_OUTPUT_TOO_SMALL = STATUS_INPUT_INVALID + 1

// STATUS_STORAGE_INVALID means output overlaps one read-only input.
const STATUS_STORAGE_INVALID = STATUS_OUTPUT_TOO_SMALL + 1

// STATUS_OUTPUT_TOO_LARGE means formatting exceeds package output bound.
const STATUS_OUTPUT_TOO_LARGE = STATUS_STORAGE_INVALID + 1

// ARRAY_VALUE_OR_END permits the empty-array closing delimiter.
const ARRAY_VALUE_OR_END byte = 0

// ARRAY_VALUE requires a value after a comma.
const ARRAY_VALUE = ARRAY_VALUE_OR_END + 1

// ARRAY_AFTER_VALUE requires a comma or closing delimiter.
const ARRAY_AFTER_VALUE = ARRAY_VALUE + 1

// OBJECT_KEY_OR_END permits the empty-object closing delimiter.
const OBJECT_KEY_OR_END = ARRAY_AFTER_VALUE + 1

// OBJECT_KEY requires a quoted key after a comma.
const OBJECT_KEY = OBJECT_KEY_OR_END + 1

// OBJECT_COLON requires the separator after a key.
const OBJECT_COLON = OBJECT_KEY + 1

// OBJECT_VALUE requires a value after the separator.
const OBJECT_VALUE = OBJECT_COLON + 1

// OBJECT_AFTER_VALUE requires a comma or closing delimiter.
const OBJECT_AFTER_VALUE = OBJECT_VALUE + 1

// BYTE_MINIMUM is the first JSON octet.
const BYTE_MINIMUM = 0

// BYTE_MAXIMUM is the final JSON octet.
const BYTE_MAXIMUM = 255

// CONTAINER_INDEX_MAXIMUM is the last parser stack slot.
const CONTAINER_INDEX_MAXIMUM = NESTING_DEPTH_MAXIMUM - 1

// ACTIVE_DEPTH_MINIMUM is one open container.
const ACTIVE_DEPTH_MINIMUM = 1

// LITERAL_SIZE_MINIMUM is the size of true and null.
const LITERAL_SIZE_MINIMUM = len("true")

// LITERAL_SIZE_MAXIMUM is the size of false.
const LITERAL_SIZE_MAXIMUM = len("false")

// LINE_SIZE_MAXIMUM is the largest prefix and indentation line expansion.
const LINE_SIZE_MAXIMUM = 1 + PREFIX_SIZE_MAXIMUM +
	NESTING_DEPTH_MAXIMUM*INDENT_SIZE_MAXIMUM

// FORMATTING_SIZE_MAXIMUM includes one pending line expansion after bounded output.
const FORMATTING_SIZE_MAXIMUM = OUTPUT_SIZE_MAXIMUM + LINE_SIZE_MAXIMUM + 2

// PENDING_FORMATTING_SIZE_MAXIMUM leaves one next symbol.
const PENDING_FORMATTING_SIZE_MAXIMUM = FORMATTING_SIZE_MAXIMUM - 1

// OUTPUT_INDEX_MAXIMUM is the last writable output byte.
const OUTPUT_INDEX_MAXIMUM = OUTPUT_SIZE_MAXIMUM - 1

// Encoded keeps malicious source bounded before syntax validation.
type Encoded []byte

// Encoded_Invariants enforces shared source boundary.
func Encoded_Invariants(value Encoded, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Output keeps caller destination bounded before capacity checks.
type Output []byte

// Output_Invariants enforces shared destination boundary.
func Output_Invariants(value Output, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, OUTPUT_SIZE_MAXIMUM).
		Ensure()
}

// Prefix keeps each repeated line prefix bounded.
type Prefix []byte

// Prefix_Invariants enforces shared formatting boundary.
func Prefix_Invariants(value Prefix, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, PREFIX_SIZE_MAXIMUM).
		Ensure()
}

// Indent keeps each repeated depth unit bounded.
type Indent []byte

// Indent_Invariants enforces shared formatting boundary.
func Indent_Invariants(value Indent, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, INDENT_SIZE_MAXIMUM).
		Ensure()
}

// Nonempty_Encoded is source after the parser sees one byte.
type Nonempty_Encoded []byte

// Nonempty_Encoded_Invariants excludes the empty parser state.
func Nonempty_Encoded_Invariants(value Nonempty_Encoded, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), VALUE_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Container_Encoded is source large enough to open and inspect a container.
type Container_Encoded []byte

// Container_Encoded_Invariants excludes sources shorter than one delimiter pair.
func Container_Encoded_Invariants(value Container_Encoded, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COMPOUND_VALUE_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Escape_Encoded is source containing a quote followed by an escape marker.
type Escape_Encoded []byte

// Escape_Encoded_Invariants excludes sources too short to reach an escape.
func Escape_Encoded_Invariants(value Escape_Encoded, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COMPOUND_VALUE_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Nonempty_Output is writable storage after exact size succeeds.
type Nonempty_Output []byte

// Nonempty_Output_Invariants excludes storage too short for one JSON value.
func Nonempty_Output_Invariants(value Nonempty_Output, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), VALUE_SIZE_MINIMUM, OUTPUT_SIZE_MAXIMUM).
		Ensure()
}

// Formatted_Output is storage after sizing proves at least one line break will be written.
type Formatted_Output []byte

// Formatted_Output_Invariants excludes destinations too short for formatted structure.
func Formatted_Output_Invariants(value Formatted_Output, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), FORMATTED_OUTPUT_SIZE_MINIMUM, OUTPUT_SIZE_MAXIMUM).
		Ensure()
}

// Formatting_Affix is one prefix or indent that can occur twice in bounded output.
type Formatting_Affix []byte

// Formatting_Affix_Invariants retains room for minimum formatted structure.
func Formatting_Affix_Invariants(value Formatting_Affix, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), bytes.SLICE_SIZE_MINIMUM, FORMATTING_AFFIX_SIZE_MAXIMUM,
		).
		Ensure()
}

// Count is exact bounded output bytes or zero on refusal.
type Count int

// Count_Invariants keeps public results inside writable output bound.
func Count_Invariants(value Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), bytes.SLICE_SIZE_MINIMUM, OUTPUT_SIZE_MAXIMUM).
		Ensure()
}

// Compact_Count excludes zero because valid JSON always contains one value byte.
type Compact_Count int

// Compact_Count_Invariants follows compact output limits for valid input.
func Compact_Count_Invariants(value Compact_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), VALUE_SIZE_MINIMUM, OUTPUT_SIZE_MAXIMUM).
		Ensure()
}

// Position is one-based invalid byte or zero when absent.
type Position int

// Position_Invariants includes unexpected end after maximum source.
func Position_Invariants(value Position, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), bytes.SLICE_SIZE_MINIMUM, POSITION_MAXIMUM).
		Ensure()
}

// Validate_Status reports whether source is one complete JSON value.
type Validate_Status bool

// Validate_Status_Invariants observes both syntax outcomes.
func Validate_Status_Invariants(value Validate_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "JSON source is valid.").
		Ensure()
}

// Compact_Size_Status reports whether compact size exists for source.
type Compact_Size_Status bool

// Compact_Size_Status_Invariants observes both syntax outcomes.
func Compact_Size_Status_Invariants(
	value Compact_Size_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Compact size exists.").
		Ensure()
}

// Indent_Size_Status admits bounded growth refusal after valid syntax.
type Indent_Size_Status uint8

// Indent_Size_Status_Invariants lists indentation sizing outcomes.
func Indent_Size_Status_Invariants(
	value Indent_Size_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_INPUT_INVALID),
			uint8(STATUS_OUTPUT_TOO_LARGE),
		).
		Ensure()
}

// Compact_Status includes syntax, output, and overlap refusals.
type Compact_Status uint8

// Compact_Status_Invariants lists every compact outcome.
func Compact_Status_Invariants(value Compact_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_INPUT_INVALID),
			uint8(STATUS_OUTPUT_TOO_SMALL), uint8(STATUS_STORAGE_INVALID),
		).
		Ensure()
}

// Indent_Status includes every shared transform refusal.
type Indent_Status uint8

// Indent_Status_Invariants covers contiguous indentation outcomes.
func Indent_Status_Invariants(value Indent_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), uint8(STATUS_OK), uint8(STATUS_OUTPUT_TOO_LARGE)).
		Ensure()
}

// Value_Result reports one scalar parse.
type Value_Result struct {
	// Position is continuation on success or one-based error on refusal.
	Position Parse_Position
	// Valid avoids error interfaces and owned messages.
	Valid Parse_Valid
}

// Value_Result_Invariants bounds one scalar parse result.
func Value_Result_Invariants(value Value_Result, namespace aver.Namespace) {
	Parse_Position_Invariants(value.Position, namespace)
	Parse_Valid_Invariants(value.Valid, namespace)
}

// String_Result reports a quoted value or escape parse.
type String_Result struct {
	// Position follows the opening quote.
	Position String_Position
	// Valid reports complete quoted syntax.
	Valid Parse_Valid
}

// String_Result_Invariants bounds quoted parser progress.
func String_Result_Invariants(value String_Result, namespace aver.Namespace) {
	String_Position_Invariants(value.Position, namespace)
	Parse_Valid_Invariants(value.Valid, namespace)
}

// Escape_Result reports a sequence after slash and escape byte.
type Escape_Result struct {
	// Position follows the escape or names its refusal.
	Position Escape_Position
	// Valid reports complete escape syntax.
	Valid Parse_Valid
}

// Escape_Result_Invariants bounds escape progress.
func Escape_Result_Invariants(value Escape_Result, namespace aver.Namespace) {
	Escape_Position_Invariants(value.Position, namespace)
	Parse_Valid_Invariants(value.Valid, namespace)
}

// Start_Result reports one value start and container opening.
type Start_Result struct {
	// Position continues parsing or names refusal.
	Position Parse_Position
	// Valid reports accepted value syntax.
	Valid Parse_Valid
	// Opened reports pushed container state.
	Opened Boolean
}

// Start_Result_Invariants bounds one value-start result.
func Start_Result_Invariants(value Start_Result, namespace aver.Namespace) {
	Parse_Position_Invariants(value.Position, namespace)
	Parse_Valid_Invariants(value.Valid, namespace)
	Boolean_Invariants(value.Opened, namespace)
}

// Push_Result reports one successful or refused container opening.
type Push_Result struct {
	// Position follows the opening byte.
	Position End_Position
	// Valid reports available nesting storage.
	Valid Parse_Valid
	// Opened reports stored container state.
	Opened Boolean
}

// Push_Result_Invariants bounds one container opening.
func Push_Result_Invariants(value Push_Result, namespace aver.Namespace) {
	End_Position_Invariants(value.Position, namespace)
	Parse_Valid_Invariants(value.Valid, namespace)
	Boolean_Invariants(value.Opened, namespace)
}

// Container_Result reports one container grammar step.
type Container_Result struct {
	// Position follows the grammar byte.
	Position End_Position
	// Valid reports accepted container syntax.
	Valid Parse_Valid
	// Need_Value selects value parsing next.
	Need_Value Need_Value
	// Closed reports completed container syntax.
	Closed Boolean
}

// Container_Result_Invariants bounds one container step.
func Container_Result_Invariants(value Container_Result, namespace aver.Namespace) {
	End_Position_Invariants(value.Position, namespace)
	Parse_Valid_Invariants(value.Valid, namespace)
	Need_Value_Invariants(value.Need_Value, namespace)
	Boolean_Invariants(value.Closed, namespace)
}

// Object_Result reports one object grammar step.
type Object_Result struct {
	// Position follows the object grammar byte.
	Position Object_Position
	// Valid reports accepted object syntax.
	Valid Parse_Valid
	// Need_Value selects member value parsing.
	Need_Value Need_Value
	// Closed reports completed object syntax.
	Closed Boolean
}

// Object_Result_Invariants bounds object grammar progress.
func Object_Result_Invariants(value Object_Result, namespace aver.Namespace) {
	Object_Position_Invariants(value.Position, namespace)
	Parse_Valid_Invariants(value.Valid, namespace)
	Need_Value_Invariants(value.Need_Value, namespace)
	Boolean_Invariants(value.Closed, namespace)
}

// Complete_Result reports trailing syntax inspection.
type Complete_Result struct {
	// Position ends the value or names trailing syntax.
	Position End_Position
	// Valid reports no trailing syntax.
	Valid Parse_Valid
}

// Complete_Result_Invariants bounds trailing syntax inspection.
func Complete_Result_Invariants(value Complete_Result, namespace aver.Namespace) {
	End_Position_Invariants(value.Position, namespace)
	Parse_Valid_Invariants(value.Valid, namespace)
}

// Container_Kinds retains one bounded parser kind per nesting level.
type Container_Kinds []byte

// Container_Kinds_Invariants fixes caller-independent parser stack storage.
func Container_Kinds_Invariants(value Container_Kinds, _ aver.Namespace) {
	aver.Always(len(value) == NESTING_DEPTH_MAXIMUM, "Kind stack has one slot per level.")
}

// Container_States retains one bounded grammar state per nesting level.
type Container_States []byte

// Container_States_Invariants fixes caller-independent parser stack storage.
func Container_States_Invariants(value Container_States, _ aver.Namespace) {
	aver.Always(len(value) == NESTING_DEPTH_MAXIMUM, "State stack has one slot per level.")
}

// Boolean gives lexical decisions independent coverage identity.
type Boolean bool

// Boolean_Invariants covers both lexical outcomes.
func Boolean_Invariants(value Boolean, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "JSON lexical decision is positive.").
		Ensure()
}

// Parse_Position is continuation on success or one-based error on refusal.
type Parse_Position int

// Parse_Position_Invariants admits both parser position meanings.
func Parse_Position_Invariants(value Parse_Position, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), VALUE_SIZE_MINIMUM, POSITION_MAXIMUM).
		Ensure()
}

// String_Position follows an opening quote or slash.
type String_Position int

// String_Position_Invariants excludes positions before quoted content.
func String_Position_Invariants(value String_Position, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), STRING_POSITION_MINIMUM, POSITION_MAXIMUM).
		Ensure()
}

// Escape_Position follows slash and its escape byte.
type Escape_Position int

// Escape_Position_Invariants excludes positions before one escape sequence.
func Escape_Position_Invariants(value Escape_Position, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), ESCAPE_POSITION_MINIMUM, POSITION_MAXIMUM).
		Ensure()
}

// Parse_Valid reports whether the parser stage succeeded.
type Parse_Valid bool

// Parse_Valid_Invariants covers parser success and refusal.
func Parse_Valid_Invariants(value Parse_Valid, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "JSON parser stage succeeds.").
		Ensure()
}

// Need_Value reports whether container grammar expects a value.
type Need_Value bool

// Need_Value_Invariants covers both container grammar states.
func Need_Value_Invariants(value Need_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "JSON container expects a value.").
		Ensure()
}

// Internal_Position is a zero-based parser or formatter position.
type Internal_Position int

// Internal_Position_Invariants keeps internal positions inside source boundary.
func Internal_Position_Invariants(value Internal_Position, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), bytes.SLICE_SIZE_MINIMUM, POSITION_MAXIMUM).
		Ensure()
}

// Source_Index is one addressable JSON byte.
type Source_Index int

// Source_Index_Invariants excludes the position after input storage.
func Source_Index_Invariants(value Source_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), bytes.SLICE_SIZE_MINIMUM, SOURCE_INDEX_MAXIMUM).
		Ensure()
}

// Container_Position is one byte after an opening delimiter.
type Container_Position int

// Container_Position_Invariants bounds active container grammar input.
func Container_Position_Invariants(value Container_Position, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), VALUE_SIZE_MINIMUM, SOURCE_INDEX_MAXIMUM).
		Ensure()
}

// End_Position is one continuation inside or immediately after input.
type End_Position int

// End_Position_Invariants excludes absence and unexpected-end errors.
func End_Position_Invariants(value End_Position, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), VALUE_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Object_Position follows one object grammar byte.
type Object_Position int

// Object_Position_Invariants bounds object grammar progress.
func Object_Position_Invariants(value Object_Position, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), OBJECT_POSITION_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Depth is active JSON container count.
type Depth int

// Depth_Invariants keeps parser and formatter nesting bounded.
func Depth_Invariants(value Depth, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), bytes.SLICE_SIZE_MINIMUM, NESTING_DEPTH_MAXIMUM).
		Ensure()
}

// Active_Depth is one or more open containers.
type Active_Depth int

// Active_Depth_Invariants excludes container-free parser state.
func Active_Depth_Invariants(value Active_Depth, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), ACTIVE_DEPTH_MINIMUM, NESTING_DEPTH_MAXIMUM).
		Ensure()
}

// Container_Index is one addressable parser stack slot.
type Container_Index int

// Container_Index_Invariants excludes the position after parser stack storage.
func Container_Index_Invariants(value Container_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), bytes.SLICE_SIZE_MINIMUM, CONTAINER_INDEX_MAXIMUM).
		Ensure()
}

// Container_Kind is one opening delimiter.
type Container_Kind byte

// Container_Kind_Invariants lists both opening delimiters.
func Container_Kind_Invariants(value Container_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(ARRAY_CONTAINER_KIND), uint8(OBJECT_CONTAINER_KIND)).
		Ensure()
}

// JSON_Byte is one untrusted source octet.
type JSON_Byte byte

// JSON_Byte_Invariants covers the complete octet domain.
func JSON_Byte_Invariants(value JSON_Byte, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), BYTE_MINIMUM, BYTE_MAXIMUM).
		Ensure()
}

// Literal is one fixed JSON word matched by the parser.
type Literal string

// Literal_Invariants covers the two literal lengths.
func Literal_Invariants(value Literal, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Int(len(value), LITERAL_SIZE_MINIMUM, LITERAL_SIZE_MAXIMUM).
		Ensure()
}

// Formatting_Size is an intermediate indentation size before overflow refusal.
type Formatting_Size int

// Formatting_Size_Invariants includes one completed size increment.
func Formatting_Size_Invariants(value Formatting_Size, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), VALUE_SIZE_MINIMUM, FORMATTING_SIZE_MAXIMUM,
		).
		Ensure()
}

// Pending_Formatting_Size leaves room for one completed increment.
type Pending_Formatting_Size int

// Pending_Formatting_Size_Invariants bounds pre-symbol size.
func Pending_Formatting_Size_Invariants(
	value Pending_Formatting_Size, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), bytes.SLICE_SIZE_MINIMUM,
			PENDING_FORMATTING_SIZE_MAXIMUM,
		).
		Ensure()
}

// Output_Index is one writable output byte.
type Output_Index int

// Output_Index_Invariants excludes the position after storage.
func Output_Index_Invariants(value Output_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), bytes.SLICE_SIZE_MINIMUM, OUTPUT_INDEX_MAXIMUM).
		Ensure()
}

// Nonzero_Output_Index follows one written structural byte.
type Nonzero_Output_Index int

// Nonzero_Output_Index_Invariants excludes initial output position.
func Nonzero_Output_Index_Invariants(
	value Nonzero_Output_Index, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), VALUE_SIZE_MINIMUM, OUTPUT_INDEX_MAXIMUM).
		Ensure()
}

// Written_Position follows one written output byte.
type Written_Position int

// Written_Position_Invariants bounds completed output progress.
func Written_Position_Invariants(value Written_Position, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), VALUE_SIZE_MINIMUM, OUTPUT_SIZE_MAXIMUM).
		Ensure()
}

// Line_Position follows one structural byte and newline.
type Line_Position int

// Line_Position_Invariants bounds completed newline progress.
func Line_Position_Invariants(value Line_Position, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), LINE_POSITION_MINIMUM, OUTPUT_SIZE_MAXIMUM).
		Ensure()
}

// Line_Size is one newline, prefix, and repeated indentation unit.
type Line_Size int

// Line_Size_Invariants keeps one formatting line expansion bounded.
func Line_Size_Invariants(value Line_Size, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), VALUE_SIZE_MINIMUM, LINE_SIZE_MAXIMUM).
		Ensure()
}

// Validate reports first invalid byte without owned diagnostics.
func Validate(source Encoded) (position Position, status Validate_Status) {
	defer func() {
		Position_Invariants(position, "Validate.position")
		Validate_Status_Invariants(status, "Validate.status")
	}()
	Encoded_Invariants(source, "Validate.source")
	result := validate_unchecked(source)
	if !result.Valid {
		return Position(result.Position), false
	}
	return 0, true
}

// Compact_Size reports exact output before any caller storage can change.
func Compact_Size(
	source Encoded,
) (count Count, position Position, status Compact_Size_Status) {
	defer func() {
		Count_Invariants(count, "Compact_Size.count")
		Position_Invariants(position, "Compact_Size.position")
		Compact_Size_Status_Invariants(status, "Compact_Size.status")
	}()
	Encoded_Invariants(source, "Compact_Size.source")
	result := validate_unchecked(source)
	if !result.Valid {
		return 0, Position(result.Position), false
	}
	return Count(compact_size_unchecked(Nonempty_Encoded(source))), 0, true
}

// Compact_Into rejects every refusal before changing caller output.
func Compact_Into(
	destination Output, source Encoded,
) (count Count, position Position, status Compact_Status) {
	defer func() {
		Count_Invariants(count, "Compact_Into.count")
		Position_Invariants(position, "Compact_Into.position")
		Compact_Status_Invariants(status, "Compact_Into.status")
	}()
	Output_Invariants(destination, "Compact_Into.destination")
	Encoded_Invariants(source, "Compact_Into.source")
	if bytes.Overlap(bytes.Slice(destination), bytes.Slice(source)) {
		return 0, 0, STATUS_STORAGE_INVALID
	}
	required, invalid_position, size_status := Compact_Size(source)
	if !bool(size_status) {
		return 0, invalid_position, STATUS_INPUT_INVALID
	}
	if len(destination) < int(required) {
		return 0, 0, STATUS_OUTPUT_TOO_SMALL
	}
	compact_unchecked(
		Nonempty_Output(destination), Nonempty_Encoded(source),
	)
	return required, 0, STATUS_OK
}

// Indent_Size refuses unbounded formatting growth before output access.
func Indent_Size(
	source Encoded, prefix Prefix, indent Indent,
) (count Count, position Position, status Indent_Size_Status) {
	defer func() {
		Count_Invariants(count, "Indent_Size.count")
		Position_Invariants(position, "Indent_Size.position")
		Indent_Size_Status_Invariants(status, "Indent_Size.status")
	}()
	Encoded_Invariants(source, "Indent_Size.source")
	Prefix_Invariants(prefix, "Indent_Size.prefix")
	Indent_Invariants(indent, "Indent_Size.indent")
	result := validate_unchecked(source)
	if !result.Valid {
		return 0, Position(result.Position), STATUS_INPUT_INVALID
	}
	required, too_large := indent_size_unchecked(
		Nonempty_Encoded(source), End_Position(result.Position), prefix, indent,
	)
	if too_large {
		return 0, 0, STATUS_OUTPUT_TOO_LARGE
	}
	return required, 0, STATUS_OK
}

// Indent_Into rejects every refusal before changing caller output.
func Indent_Into(
	destination Output, source Encoded, prefix Prefix, indent Indent,
) (count Count, position Position, status Indent_Status) {
	defer func() {
		Count_Invariants(count, "Indent_Into.count")
		Position_Invariants(position, "Indent_Into.position")
		Indent_Status_Invariants(status, "Indent_Into.status")
	}()
	Output_Invariants(destination, "Indent_Into.destination")
	Encoded_Invariants(source, "Indent_Into.source")
	Prefix_Invariants(prefix, "Indent_Into.prefix")
	Indent_Invariants(indent, "Indent_Into.indent")
	if transform_storage_overlaps(destination, source, prefix, indent) {
		return 0, 0, STATUS_STORAGE_INVALID
	}
	result := validate_unchecked(source)
	if !result.Valid {
		return 0, Position(result.Position), STATUS_INPUT_INVALID
	}
	value_end := End_Position(result.Position)
	required, too_large := indent_size_unchecked(
		Nonempty_Encoded(source), value_end, prefix, indent,
	)
	if too_large {
		return 0, 0, STATUS_OUTPUT_TOO_LARGE
	}
	if len(destination) < int(required) {
		return 0, 0, STATUS_OUTPUT_TOO_SMALL
	}
	indent_unchecked(
		Nonempty_Output(destination), Nonempty_Encoded(source),
		value_end, prefix, indent,
	)
	return required, 0, STATUS_OK
}

func validate_unchecked(source Encoded) (result Value_Result) {
	defer func() { Value_Result_Invariants(result, "validate_unchecked.result") }()
	Encoded_Invariants(source, "validate_unchecked.source")
	var kind_storage [NESTING_DEPTH_MAXIMUM]byte
	var state_storage [NESTING_DEPTH_MAXIMUM]byte
	kinds := Container_Kinds(kind_storage[:])
	states := Container_States(state_storage[:])
	position := skip_space(source, 0)
	if int(position) == len(source) {
		return Value_Result{Position: Parse_Position(len(source) + 1)}
	}
	depth := Depth(bytes.SLICE_SIZE_MINIMUM)
	need_value := true
	for int(position) <= len(source) {
		value_complete := false
		if need_value {
			start_result := parse_value_start(
				Nonempty_Encoded(source), Source_Index(position),
				kinds, states, depth,
			)
			if !start_result.Valid {
				return Value_Result{
					Position: start_result.Position,
				}
			}
			position = Internal_Position(start_result.Position)
			if bool(start_result.Opened) {
				depth++
				need_value = false
				continue
			}
			value_complete = true
		} else {
			position = skip_space(source, position)
			if int(position) == len(source) {
				return Value_Result{Position: Parse_Position(len(source) + 1)}
			}
			container_result := parse_container(
				Container_Encoded(source), Container_Position(position),
				kinds, states,
				Active_Depth(depth),
			)
			if !container_result.Valid {
				return Value_Result{
					Position: Parse_Position(container_result.Position),
				}
			}
			position = Internal_Position(container_result.Position)
			need_value = bool(container_result.Need_Value)
			value_complete = bool(container_result.Closed)
			if container_result.Closed {
				depth--
			}
		}
		if !value_complete {
			continue
		}
		if depth == bytes.SLICE_SIZE_MINIMUM {
			complete := validate_complete(
				Nonempty_Encoded(source), End_Position(position),
			)
			return Value_Result{
				Position: Parse_Position(complete.Position), Valid: complete.Valid,
			}
		}
		mark_parent_complete(kinds, states, Active_Depth(depth))
		need_value = false
	}
	return Value_Result{Position: Parse_Position(position + 1)}
}

func parse_value_start(
	source Nonempty_Encoded, start Source_Index, kinds Container_Kinds,
	states Container_States, depth Depth,
) (result Start_Result) {
	defer func() { Start_Result_Invariants(result, "parse_value_start.result") }()
	Nonempty_Encoded_Invariants(source, "parse_value_start.source")
	Source_Index_Invariants(start, "parse_value_start.start")
	Container_Kinds_Invariants(kinds, "parse_value_start.kinds")
	Container_States_Invariants(states, "parse_value_start.states")
	Depth_Invariants(depth, "parse_value_start.depth")
	position := skip_space(Encoded(source), Internal_Position(start))
	if int(position) >= len(source) {
		return Start_Result{Position: Parse_Position(len(source) + 1)}
	}
	switch source[position] {
	case '{':
		pushed := push_container(
			Source_Index(position), Container_Kind('{'), kinds, states, depth,
		)
		return Start_Result{
			Position: Parse_Position(pushed.Position), Valid: pushed.Valid,
			Opened: pushed.Opened,
		}
	case '[':
		pushed := push_container(
			Source_Index(position), Container_Kind('['), kinds, states, depth,
		)
		return Start_Result{
			Position: Parse_Position(pushed.Position), Valid: pushed.Valid,
			Opened: pushed.Opened,
		}
	case '"':
		value := parse_string(source, Source_Index(position))
		return Start_Result{Position: Parse_Position(value.Position), Valid: value.Valid}
	case 't':
		value := parse_literal(source, Source_Index(position), "true")
		return Start_Result{Position: value.Position, Valid: value.Valid}
	case 'f':
		value := parse_literal(source, Source_Index(position), "false")
		return Start_Result{Position: value.Position, Valid: value.Valid}
	case 'n':
		value := parse_literal(source, Source_Index(position), "null")
		return Start_Result{Position: value.Position, Valid: value.Valid}
	default:
		value := parse_number(source, Source_Index(position))
		return Start_Result{Position: value.Position, Valid: value.Valid}
	}
}

func push_container(
	position Source_Index, kind Container_Kind, kinds Container_Kinds,
	states Container_States, depth Depth,
) (result Push_Result) {
	defer func() { Push_Result_Invariants(result, "push_container.result") }()
	Source_Index_Invariants(position, "push_container.position")
	Container_Kind_Invariants(kind, "push_container.kind")
	Container_Kinds_Invariants(kinds, "push_container.kinds")
	Container_States_Invariants(states, "push_container.states")
	Depth_Invariants(depth, "push_container.depth")
	if depth == NESTING_DEPTH_MAXIMUM {
		return Push_Result{Position: End_Position(position + 1)}
	}
	kinds[depth] = byte(kind)
	if kind == Container_Kind('[') {
		states[depth] = ARRAY_VALUE_OR_END
	} else {
		states[depth] = OBJECT_KEY_OR_END
	}
	return Push_Result{
		Position: End_Position(position + 1), Valid: true, Opened: true,
	}
}

func parse_container(
	source Container_Encoded, position Container_Position, kinds Container_Kinds,
	states Container_States, depth Active_Depth,
) (result Container_Result) {
	defer func() { Container_Result_Invariants(result, "parse_container.result") }()
	Container_Encoded_Invariants(source, "parse_container.source")
	Container_Position_Invariants(position, "parse_container.position")
	Container_Kinds_Invariants(kinds, "parse_container.kinds")
	Container_States_Invariants(states, "parse_container.states")
	Active_Depth_Invariants(depth, "parse_container.depth")
	index := depth - 1
	if kinds[index] == '[' {
		return parse_array_state(source, position, states, depth)
	}
	object := parse_object_state(source, position, states, depth)
	return Container_Result{
		Position: End_Position(object.Position), Valid: object.Valid,
		Need_Value: object.Need_Value, Closed: object.Closed,
	}
}

func parse_array_state(
	source Container_Encoded, position Container_Position, states Container_States,
	depth Active_Depth,
) (result Container_Result) {
	defer func() { Container_Result_Invariants(result, "parse_array_state.result") }()
	Container_Encoded_Invariants(source, "parse_array_state.source")
	Container_Position_Invariants(position, "parse_array_state.position")
	Container_States_Invariants(states, "parse_array_state.states")
	Active_Depth_Invariants(depth, "parse_array_state.depth")
	index := depth - 1
	if states[index] == ARRAY_VALUE_OR_END {
		if source[position] == ']' {
			return Container_Result{
				Position: End_Position(position + 1), Valid: true, Closed: true,
			}
		}
		return Container_Result{
			Position: End_Position(position), Valid: true, Need_Value: true,
		}
	}
	if states[index] == ARRAY_VALUE {
		return Container_Result{
			Position: End_Position(position), Valid: true, Need_Value: true,
		}
	}
	if source[position] == ']' {
		return Container_Result{
			Position: End_Position(position + 1), Valid: true, Closed: true,
		}
	}
	if source[position] != ',' {
		return Container_Result{Position: End_Position(position + 1)}
	}
	states[index] = ARRAY_VALUE
	return Container_Result{Position: End_Position(position + 1), Valid: true}
}

func parse_object_state(
	source Container_Encoded, position Container_Position, states Container_States,
	depth Active_Depth,
) (result Object_Result) {
	defer func() { Object_Result_Invariants(result, "parse_object_state.result") }()
	Container_Encoded_Invariants(source, "parse_object_state.source")
	Container_Position_Invariants(position, "parse_object_state.position")
	Container_States_Invariants(states, "parse_object_state.states")
	Active_Depth_Invariants(depth, "parse_object_state.depth")
	index := depth - 1
	switch states[index] {
	case OBJECT_KEY_OR_END:
		if source[position] == '}' {
			return Object_Result{
				Position: Object_Position(position + 1), Valid: true, Closed: true,
			}
		}
		key := parse_object_key(source, position, states, Container_Index(index))
		return Object_Result{
			Position: Object_Position(key.Position), Valid: key.Valid,
		}
	case OBJECT_KEY:
		key := parse_object_key(source, position, states, Container_Index(index))
		return Object_Result{
			Position: Object_Position(key.Position), Valid: key.Valid,
		}
	case OBJECT_COLON:
		if source[position] != ':' {
			return Object_Result{Position: Object_Position(position + 1)}
		}
		states[index] = OBJECT_VALUE
		return Object_Result{
			Position: Object_Position(position + 1), Valid: true, Need_Value: true,
		}
	default:
		if source[position] == '}' {
			return Object_Result{
				Position: Object_Position(position + 1), Valid: true, Closed: true,
			}
		}
		if source[position] != ',' {
			return Object_Result{Position: Object_Position(position + 1)}
		}
		states[depth-1] = OBJECT_KEY
		return Object_Result{Position: Object_Position(position + 1), Valid: true}
	}
}

func parse_object_key(
	source Container_Encoded, position Container_Position,
	states Container_States, index Container_Index,
) (result String_Result) {
	defer func() { String_Result_Invariants(result, "parse_object_key.result") }()
	Container_Encoded_Invariants(source, "parse_object_key.source")
	Container_Position_Invariants(position, "parse_object_key.position")
	Container_States_Invariants(states, "parse_object_key.states")
	Container_Index_Invariants(index, "parse_object_key.index")
	if source[position] != '"' {
		return String_Result{Position: String_Position(position + 1)}
	}
	result = parse_string(Nonempty_Encoded(source), Source_Index(position))
	if result.Valid {
		states[index] = OBJECT_COLON
	}
	return result
}

func mark_parent_complete(
	kinds Container_Kinds, states Container_States, depth Active_Depth,
) {
	Container_Kinds_Invariants(kinds, "mark_parent_complete.kinds")
	Container_States_Invariants(states, "mark_parent_complete.states")
	Active_Depth_Invariants(depth, "mark_parent_complete.depth")
	index := depth - 1
	if kinds[index] == '[' {
		states[index] = ARRAY_AFTER_VALUE
	} else {
		states[index] = OBJECT_AFTER_VALUE
	}
}

func validate_complete(
	source Nonempty_Encoded, start End_Position,
) (result Complete_Result) {
	defer func() { Complete_Result_Invariants(result, "validate_complete.result") }()
	Nonempty_Encoded_Invariants(source, "validate_complete.source")
	End_Position_Invariants(start, "validate_complete.start")
	position := skip_space(Encoded(source), Internal_Position(start))
	if int(position) != len(source) {
		return Complete_Result{Position: End_Position(position + 1)}
	}
	return Complete_Result{Position: start, Valid: true}
}

func parse_string(
	source Nonempty_Encoded, start Source_Index,
) (result String_Result) {
	defer func() { String_Result_Invariants(result, "parse_string.result") }()
	Nonempty_Encoded_Invariants(source, "parse_string.source")
	Source_Index_Invariants(start, "parse_string.start")
	for position := start + 1; int(position) < len(source); position++ {
		value := source[position]
		if value == '"' {
			return String_Result{Position: String_Position(position + 1), Valid: true}
		}
		if value < ' ' {
			return String_Result{Position: String_Position(position + 1)}
		}
		if value == '\\' {
			escape := parse_escape(Escape_Encoded(source), Container_Position(position))
			if !escape.Valid {
				return String_Result{
					Position: String_Position(escape.Position),
				}
			}
			position = Source_Index(escape.Position - 1)
			continue
		}
	}
	return String_Result{Position: String_Position(len(source) + 1)}
}

func parse_escape(
	source Escape_Encoded, slash Container_Position,
) (result Escape_Result) {
	defer func() { Escape_Result_Invariants(result, "parse_escape.result") }()
	Escape_Encoded_Invariants(source, "parse_escape.source")
	Container_Position_Invariants(slash, "parse_escape.slash")
	position := slash + 1
	if int(position) >= len(source) {
		return Escape_Result{Position: Escape_Position(len(source) + 1)}
	}
	value := source[position]
	switch value {
	case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
		return Escape_Result{Position: Escape_Position(position + 1), Valid: true}
	}
	if value != 'u' {
		return Escape_Result{Position: Escape_Position(position + 1)}
	}
	for index := position + 1; index < position+5; index++ {
		if int(index) >= len(source) {
			return Escape_Result{Position: Escape_Position(len(source) + 1)}
		}
		if !bool(hexadecimal(JSON_Byte(source[index]))) {
			return Escape_Result{Position: Escape_Position(index + 1)}
		}
	}
	return Escape_Result{Position: Escape_Position(position + 5), Valid: true}
}

func parse_literal(
	source Nonempty_Encoded, start Source_Index, literal Literal,
) (result Value_Result) {
	defer func() { Value_Result_Invariants(result, "parse_literal.result") }()
	Nonempty_Encoded_Invariants(source, "parse_literal.source")
	Source_Index_Invariants(start, "parse_literal.start")
	Literal_Invariants(literal, "parse_literal.literal")
	for index := range len(literal) {
		position := Internal_Position(start) + Internal_Position(index)
		if int(position) >= len(source) {
			return Value_Result{Position: Parse_Position(len(source) + 1)}
		}
		if source[position] != literal[index] {
			return Value_Result{Position: Parse_Position(position + 1)}
		}
	}
	return Value_Result{
		Position: Parse_Position(
			Internal_Position(start) + Internal_Position(len(literal)),
		),
		Valid: true,
	}
}

func parse_number(
	source Nonempty_Encoded, start Source_Index,
) (result Value_Result) {
	defer func() { Value_Result_Invariants(result, "parse_number.result") }()
	Nonempty_Encoded_Invariants(source, "parse_number.source")
	Source_Index_Invariants(start, "parse_number.start")
	position := Internal_Position(start)
	if source[position] == '-' {
		position++
		if int(position) == len(source) {
			return Value_Result{Position: Parse_Position(len(source) + 1)}
		}
	}
	if source[position] == '0' {
		position++
	} else {
		if source[position] < '1' {
			return Value_Result{Position: Parse_Position(position + 1)}
		}
		if source[position] > '9' {
			return Value_Result{Position: Parse_Position(position + 1)}
		}
		position = digits_end(source, position)
	}
	if int(position) < len(source) {
		if source[position] == '.' {
			position++
			if int(position) == len(source) {
				return Value_Result{Position: Parse_Position(position + 1)}
			}
			if !bool(decimal(JSON_Byte(source[position]))) {
				return Value_Result{Position: Parse_Position(position + 1)}
			}
			position = digits_end(source, position)
		}
	}
	return parse_exponent(source, End_Position(position))
}

func parse_exponent(
	source Nonempty_Encoded, start End_Position,
) (result Value_Result) {
	defer func() { Value_Result_Invariants(result, "parse_exponent.result") }()
	Nonempty_Encoded_Invariants(source, "parse_exponent.source")
	End_Position_Invariants(start, "parse_exponent.start")
	position := Internal_Position(start)
	if int(position) >= len(source) {
		return Value_Result{Position: Parse_Position(position), Valid: true}
	}
	if source[position] != 'e' {
		if source[position] != 'E' {
			return Value_Result{Position: Parse_Position(position), Valid: true}
		}
	}
	position++
	if int(position) < len(source) {
		switch source[position] {
		case '+', '-':
			position++
		}
	}
	if int(position) == len(source) {
		return Value_Result{Position: Parse_Position(len(source) + 1)}
	}
	if !bool(decimal(JSON_Byte(source[position]))) {
		return Value_Result{Position: Parse_Position(position + 1)}
	}
	return Value_Result{Position: Parse_Position(digits_end(source, position)), Valid: true}
}

func skip_space(source Encoded, start Internal_Position) (next Internal_Position) {
	defer func() { Internal_Position_Invariants(next, "skip_space.next") }()
	Encoded_Invariants(source, "skip_space.source")
	Internal_Position_Invariants(start, "skip_space.start")
	position := start
	for int(position) < len(source) {
		if !bool(space(JSON_Byte(source[position]))) {
			break
		}
		position++
	}
	return position
}

func digits_end(source Nonempty_Encoded, start Internal_Position) (next Internal_Position) {
	defer func() { Internal_Position_Invariants(next, "digits_end.next") }()
	Nonempty_Encoded_Invariants(source, "digits_end.source")
	Internal_Position_Invariants(start, "digits_end.start")
	position := start
	for int(position) < len(source) {
		if !bool(decimal(JSON_Byte(source[position]))) {
			break
		}
		position++
	}
	return position
}

func space(value JSON_Byte) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "space.yes") }()
	JSON_Byte_Invariants(value, "space.value")
	switch value {
	case ' ', '\t', '\r', '\n':
		return true
	default:
		return false
	}
}

func decimal(value JSON_Byte) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "decimal.yes") }()
	JSON_Byte_Invariants(value, "decimal.value")
	if value < '0' {
		return false
	}
	return Boolean(value <= '9')
}

func hexadecimal(value JSON_Byte) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "hexadecimal.yes") }()
	JSON_Byte_Invariants(value, "hexadecimal.value")
	if value >= '0' {
		if value <= '9' {
			return true
		}
	}
	if value >= 'a' {
		if value <= 'f' {
			return true
		}
	}
	if value < 'A' {
		return false
	}
	return Boolean(value <= 'F')
}

func compact_size_unchecked(source Nonempty_Encoded) (size Compact_Count) {
	defer func() {
		Compact_Count_Invariants(size, "compact_size_unchecked.size")
	}()
	Nonempty_Encoded_Invariants(source, "compact_size_unchecked.source")
	quoted := false
	escaped := false
	for _, value := range source {
		if quoted {
			size++
			if escaped {
				escaped = false
				continue
			}
			if value == '\\' {
				escaped = true
				continue
			}
			if value == '"' {
				quoted = false
			}
			continue
		}
		if value == '"' {
			quoted = true
			size++
			continue
		}
		if !bool(space(JSON_Byte(value))) {
			size++
		}
	}
	return size
}

func compact_unchecked(destination Nonempty_Output, source Nonempty_Encoded) {
	Nonempty_Output_Invariants(destination, "compact_unchecked.destination")
	Nonempty_Encoded_Invariants(source, "compact_unchecked.source")
	position := bytes.SLICE_SIZE_MINIMUM
	quoted := false
	escaped := false
	for _, value := range source {
		if quoted {
			destination[position] = value
			position++
		} else if !bool(space(JSON_Byte(value))) {
			destination[position] = value
			position++
		}
		if quoted {
			if escaped {
				escaped = false
				continue
			}
			if value == '\\' {
				escaped = true
				continue
			}
			if value == '"' {
				quoted = false
			}
		} else if value == '"' {
			quoted = true
		}
	}
}

func indent_size_unchecked(
	source Nonempty_Encoded, value_end End_Position, prefix Prefix, indent Indent,
) (size Count, too_large Boolean) {
	defer func() {
		Count_Invariants(size, "indent_size_unchecked.size")
		Boolean_Invariants(too_large, "indent_size_unchecked.too_large")
	}()
	Nonempty_Encoded_Invariants(source, "indent_size_unchecked.source")
	End_Position_Invariants(value_end, "indent_size_unchecked.value_end")
	Prefix_Invariants(prefix, "indent_size_unchecked.prefix")
	Indent_Invariants(indent, "indent_size_unchecked.indent")
	calculated := Pending_Formatting_Size(bytes.SLICE_SIZE_MINIMUM)
	quoted := Boolean(false)
	escaped := Boolean(false)
	depth := Depth(bytes.SLICE_SIZE_MINIMUM)
	need_indent := Boolean(false)
	for _, value := range source[:int(value_end)] {
		if quoted {
			calculated++
			quoted, escaped = quote_state(JSON_Byte(value), quoted, escaped)
			continue
		}
		if bool(space(JSON_Byte(value))) {
			continue
		}
		if value == '"' {
			quoted = true
		}
		if need_indent {
			if value != '}' {
				if value != ']' {
					need_indent = false
					depth++
					line_size := newline_size(prefix, indent, depth)
					calculated += Pending_Formatting_Size(line_size)
				}
			}
		}
		next_size, next_depth, next_need_indent := indent_symbol_size(
			calculated, depth, need_indent, JSON_Byte(value), prefix, indent,
		)
		calculated = Pending_Formatting_Size(next_size)
		depth = next_depth
		need_indent = next_need_indent
		if int(calculated) > OUTPUT_SIZE_MAXIMUM {
			return 0, true
		}
	}
	calculated += Pending_Formatting_Size(len(source) - int(value_end))
	if int(calculated) > OUTPUT_SIZE_MAXIMUM {
		return 0, true
	}
	return Count(calculated), false
}

func indent_unchecked(
	destination Nonempty_Output, source Nonempty_Encoded, value_end End_Position,
	prefix Prefix, indent Indent,
) {
	Nonempty_Output_Invariants(destination, "indent_unchecked.destination")
	Nonempty_Encoded_Invariants(source, "indent_unchecked.source")
	End_Position_Invariants(value_end, "indent_unchecked.value_end")
	Prefix_Invariants(prefix, "indent_unchecked.prefix")
	Indent_Invariants(indent, "indent_unchecked.indent")
	position := Output_Index(bytes.SLICE_SIZE_MINIMUM)
	quoted := Boolean(false)
	escaped := Boolean(false)
	depth := Depth(bytes.SLICE_SIZE_MINIMUM)
	need_indent := Boolean(false)
	for _, value := range source[:int(value_end)] {
		if quoted {
			destination[position] = value
			position++
			quoted, escaped = quote_state(JSON_Byte(value), quoted, escaped)
			continue
		}
		if bool(space(JSON_Byte(value))) {
			continue
		}
		if value == '"' {
			quoted = true
		}
		if need_indent {
			if value != '}' {
				if value != ']' {
					need_indent = false
					depth++
					position = Output_Index(append_newline(
						Formatted_Output(destination),
						Nonzero_Output_Index(position),
						Formatting_Affix(prefix),
						Formatting_Affix(indent),
						depth,
					))
				}
			}
		}
		next_position, next_depth, next_need_indent := indent_symbol_write(
			destination, position, depth, need_indent, JSON_Byte(value), prefix, indent,
		)
		position = Output_Index(next_position)
		depth = next_depth
		need_indent = next_need_indent
	}
	copy(destination[position:], source[int(value_end):])
}

func quote_state(
	value JSON_Byte, quoted Boolean, escaped Boolean,
) (next_quoted Boolean, next_escaped Boolean) {
	defer func() {
		Boolean_Invariants(next_quoted, "quote_state.next_quoted")
		Boolean_Invariants(next_escaped, "quote_state.next_escaped")
	}()
	JSON_Byte_Invariants(value, "quote_state.value")
	Boolean_Invariants(quoted, "quote_state.quoted")
	Boolean_Invariants(escaped, "quote_state.escaped")
	if escaped {
		return quoted, false
	}
	if value == '\\' {
		return quoted, true
	}
	if value == '"' {
		return false, false
	}
	return quoted, false
}

func newline_size(
	prefix Prefix, indent Indent, depth Depth,
) (size Line_Size) {
	defer func() { Line_Size_Invariants(size, "newline_size.size") }()
	Prefix_Invariants(prefix, "newline_size.prefix")
	Indent_Invariants(indent, "newline_size.indent")
	Depth_Invariants(depth, "newline_size.depth")
	return Line_Size(1 + len(prefix) + int(depth)*len(indent))
}

func indent_symbol_size(
	size Pending_Formatting_Size, depth Depth, need_indent Boolean, value JSON_Byte,
	prefix Prefix, indent Indent,
) (next_size Formatting_Size, next_depth Depth, next_need_indent Boolean) {
	defer func() {
		Formatting_Size_Invariants(next_size, "indent_symbol_size.next_size")
		Depth_Invariants(next_depth, "indent_symbol_size.next_depth")
		Boolean_Invariants(next_need_indent, "indent_symbol_size.next_need_indent")
	}()
	Pending_Formatting_Size_Invariants(size, "indent_symbol_size.size")
	Depth_Invariants(depth, "indent_symbol_size.depth")
	Boolean_Invariants(need_indent, "indent_symbol_size.need_indent")
	JSON_Byte_Invariants(value, "indent_symbol_size.value")
	Prefix_Invariants(prefix, "indent_symbol_size.prefix")
	Indent_Invariants(indent, "indent_symbol_size.indent")
	next_size = Formatting_Size(size + 1)
	next_depth = depth
	next_need_indent = need_indent
	switch value {
	case '{', '[':
		next_need_indent = true
	case ',':
		next_size += Formatting_Size(newline_size(prefix, indent, depth))
	case ':':
		next_size++
	case '}', ']':
		if need_indent {
			next_need_indent = false
		} else {
			next_depth--
			next_size += Formatting_Size(newline_size(prefix, indent, next_depth))
		}
	}
	return next_size, next_depth, next_need_indent
}

func append_newline(
	destination Formatted_Output, start Nonzero_Output_Index,
	prefix Formatting_Affix, indent Formatting_Affix, depth Depth,
) (next Line_Position) {
	defer func() { Line_Position_Invariants(next, "append_newline.next") }()
	Formatted_Output_Invariants(destination, "append_newline.destination")
	Nonzero_Output_Index_Invariants(start, "append_newline.start")
	Formatting_Affix_Invariants(prefix, "append_newline.prefix")
	Formatting_Affix_Invariants(indent, "append_newline.indent")
	Depth_Invariants(depth, "append_newline.depth")
	position := Line_Position(start)
	destination[position] = '\n'
	position++
	position += Line_Position(copy(destination[position:], prefix))
	for range depth {
		position += Line_Position(copy(destination[position:], indent))
	}
	return position
}

func indent_symbol_write(
	destination Nonempty_Output, position Output_Index, depth Depth,
	need_indent Boolean, value JSON_Byte,
	prefix Prefix, indent Indent,
) (next Written_Position, next_depth Depth, next_need_indent Boolean) {
	defer func() {
		Written_Position_Invariants(next, "indent_symbol_write.next")
		Depth_Invariants(next_depth, "indent_symbol_write.next_depth")
		Boolean_Invariants(next_need_indent, "indent_symbol_write.next_need_indent")
	}()
	Nonempty_Output_Invariants(destination, "indent_symbol_write.destination")
	Output_Index_Invariants(position, "indent_symbol_write.position")
	Depth_Invariants(depth, "indent_symbol_write.depth")
	Boolean_Invariants(need_indent, "indent_symbol_write.need_indent")
	JSON_Byte_Invariants(value, "indent_symbol_write.value")
	Prefix_Invariants(prefix, "indent_symbol_write.prefix")
	Indent_Invariants(indent, "indent_symbol_write.indent")
	next = Written_Position(position)
	next_depth = depth
	next_need_indent = need_indent
	switch value {
	case '{', '[':
		destination[next] = byte(value)
		next++
		next_need_indent = true
	case ',':
		destination[next] = byte(value)
		next++
		next = Written_Position(append_newline(
			Formatted_Output(destination), Nonzero_Output_Index(next),
			Formatting_Affix(prefix), Formatting_Affix(indent), depth,
		))
	case ':':
		destination[next] = byte(value)
		destination[next+1] = ' '
		next += 2
	case '}', ']':
		if need_indent {
			next_need_indent = false
		} else {
			next_depth--
			next = Written_Position(append_newline(
				Formatted_Output(destination), Nonzero_Output_Index(next),
				Formatting_Affix(prefix), Formatting_Affix(indent), next_depth,
			))
		}
		destination[next] = byte(value)
		next++
	default:
		destination[next] = byte(value)
		next++
	}
	return next, next_depth, next_need_indent
}

func transform_storage_overlaps(
	destination Output, source Encoded, prefix Prefix, indent Indent,
) (overlaps Boolean) {
	defer func() { Boolean_Invariants(overlaps, "transform_storage_overlaps.overlaps") }()
	Output_Invariants(destination, "transform_storage_overlaps.destination")
	Encoded_Invariants(source, "transform_storage_overlaps.source")
	Prefix_Invariants(prefix, "transform_storage_overlaps.prefix")
	Indent_Invariants(indent, "transform_storage_overlaps.indent")
	if bytes.Overlap(bytes.Slice(destination), bytes.Slice(source)) {
		return true
	}
	if bytes.Overlap(bytes.Slice(destination), bytes.Slice(prefix)) {
		return true
	}
	return Boolean(bytes.Overlap(bytes.Slice(destination), bytes.Slice(indent)))
}
