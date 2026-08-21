// Package tabwriter aligns bounded tabular text into caller-owned storage.
package tabwriter

import (
	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/aver/default"
	"local/james-orcales/shared/unicode/utf8"
)

// SOURCE_SIZE_MINIMUM keeps empty input valid.
const SOURCE_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// SOURCE_SIZE_MAXIMUM follows the repository byte-slice boundary.
const SOURCE_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// SOURCE_SIZE_UNVALIDATED_MAXIMUM admits one rejected boundary witness.
const SOURCE_SIZE_UNVALIDATED_MAXIMUM = SOURCE_SIZE_MAXIMUM + utf8.CHARACTER_SIZE_MINIMUM

// OUTPUT_SIZE_MINIMUM keeps empty formatted output valid.
const OUTPUT_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// OUTPUT_SIZE_MAXIMUM follows caller byte-slice storage boundary.
const OUTPUT_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// OUTPUT_SIZE_UNREPRESENTABLE is the first result outside caller storage.
const OUTPUT_SIZE_UNREPRESENTABLE = OUTPUT_SIZE_MAXIMUM + utf8.CHARACTER_SIZE_MINIMUM

// WIDTH_MINIMUM accepts disabled padding and tab stops.
const WIDTH_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// WIDTH_MAXIMUM prevents configuration from multiplying work beyond input bound.
const WIDTH_MAXIMUM = SOURCE_SIZE_MAXIMUM

// CELL_COUNT_MAXIMUM occurs when every source byte terminates one cell.
const CELL_COUNT_MAXIMUM = SOURCE_SIZE_MAXIMUM

// LINE_COUNT_MAXIMUM includes the trailing line after every source byte is a line feed.
const LINE_COUNT_MAXIMUM = SOURCE_SIZE_MAXIMUM + utf8.CHARACTER_SIZE_MINIMUM

// ALIGNED_SOURCE_OVERHEAD needs one terminator and one trailing cell byte.
const ALIGNED_SOURCE_OVERHEAD = utf8.CHARACTER_SIZE_TWO

// ALIGNED_CELL_WIDTH_MAXIMUM leaves room for terminator and trailing cell.
const ALIGNED_CELL_WIDTH_MAXIMUM = SOURCE_SIZE_MAXIMUM - ALIGNED_SOURCE_OVERHEAD

// PARTIAL_CELL_WIDTH_MAXIMUM leaves room for one final escape opener.
const PARTIAL_CELL_WIDTH_MAXIMUM = SOURCE_SIZE_MAXIMUM - utf8.CHARACTER_SIZE_MINIMUM

// COLUMN_WIDTH_MAXIMUM adds maximum padding to maximum aligned cell width.
const COLUMN_WIDTH_MAXIMUM = ALIGNED_CELL_WIDTH_MAXIMUM + WIDTH_MAXIMUM

// CONFIGURATION_FIELD is the sole immutable-policy storage position.
const CONFIGURATION_FIELD = bytes.SLICE_SIZE_MINIMUM

// CONFIGURATION_FIELD_COUNT prevents caller mutation from changing storage shape.
const CONFIGURATION_FIELD_COUNT = CONFIGURATION_FIELD + utf8.CHARACTER_SIZE_MINIMUM

// WORKSPACE_FIELD is the sole caller-state pointer position.
const WORKSPACE_FIELD = bytes.SLICE_SIZE_MINIMUM

// WORKSPACE_FIELD_COUNT keeps unvalidated pointer storage fixed.
const WORKSPACE_FIELD_COUNT = WORKSPACE_FIELD + utf8.CHARACTER_SIZE_MINIMUM

// CELL_INDEX_MINIMUM is the first flat workspace cell.
const CELL_INDEX_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// CELL_COUNT_MINIMUM means no source cell was parsed.
const CELL_COUNT_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// NONZERO_CELL_COUNT_MINIMUM follows one successfully parsed cell.
const NONZERO_CELL_COUNT_MINIMUM = CELL_COUNT_MINIMUM + utf8.CHARACTER_SIZE_MINIMUM

// LINE_CELL_COUNT_MINIMUM means one source line is empty.
const LINE_CELL_COUNT_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// LINE_COUNT_MINIMUM is parser state before its first completed or final line.
const LINE_COUNT_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// PARSED_LINE_COUNT_MINIMUM follows the mandatory final parsed line.
const PARSED_LINE_COUNT_MINIMUM = LINE_COUNT_MINIMUM + utf8.CHARACTER_SIZE_MINIMUM

// LINE_INDEX_MINIMUM is the first parsed line.
const LINE_INDEX_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// LINE_INDEX_MAXIMUM is the last slot in line workspace.
const LINE_INDEX_MAXIMUM = LINE_COUNT_MAXIMUM - utf8.CHARACTER_SIZE_MINIMUM

// COLUMN_INDEX_MINIMUM is the first cell within one line.
const COLUMN_INDEX_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// CELL_INDEX_MAXIMUM is the final real cell slot before flat boundary.
const CELL_INDEX_MAXIMUM = CELL_COUNT_MAXIMUM - utf8.CHARACTER_SIZE_MINIMUM

// COLUMN_INDEX_MAXIMUM leaves one trailing cell outside aligned columns.
const COLUMN_INDEX_MAXIMUM = CELL_COUNT_MAXIMUM - ALIGNED_SOURCE_OVERHEAD

// STORED_CELL_START_MAXIMUM is the final source byte that can begin a cell.
const STORED_CELL_START_MAXIMUM = SOURCE_SIZE_MAXIMUM - utf8.CHARACTER_SIZE_MINIMUM

// NONEMPTY_CELL_END_MINIMUM follows the first retained source byte.
const NONEMPTY_CELL_END_MINIMUM = utf8.CHARACTER_SIZE_MINIMUM

// COLUMN_LINE_INDEX_MAXIMUM leaves bytes for an aligned line at the source tail.
const COLUMN_LINE_INDEX_MAXIMUM = SOURCE_SIZE_MAXIMUM - ALIGNED_SOURCE_OVERHEAD

// COLUMN_LINE_INDEX_MINIMUM is first line participating in one column width.
const COLUMN_LINE_INDEX_MINIMUM = LINE_INDEX_MINIMUM

// COLUMN_LINE_COUNT_MINIMUM includes one final source line.
const COLUMN_LINE_COUNT_MINIMUM = utf8.CHARACTER_SIZE_MINIMUM

// COLUMN_LINE_COUNT_MAXIMUM leaves one source byte beyond maximum aligned-line index.
const COLUMN_LINE_COUNT_MAXIMUM = SOURCE_SIZE_MAXIMUM

// LINE_TERMINATOR_NONE marks final incomplete source line.
const LINE_TERMINATOR_NONE byte = bits.WORD_8_MINIMUM

// LINE_TERMINATOR_NEWLINE preserves ordinary source line ending.
const LINE_TERMINATOR_NEWLINE byte = '\n'

// LINE_TERMINATOR_FORM_FEED resets the current column section.
const LINE_TERMINATOR_FORM_FEED byte = '\f'

// END_CHARACTER_HTML_TAG closes ignored HTML tag width.
const END_CHARACTER_HTML_TAG byte = '>'

// END_CHARACTER_HTML_ENTITY closes one-character HTML entity width.
const END_CHARACTER_HTML_ENTITY byte = ';'

// ESCAPE_START_HTML_TAG opens ignored HTML tag width.
const ESCAPE_START_HTML_TAG byte = '<'

// ESCAPE_START_HTML_ENTITY opens one-character HTML entity width.
const ESCAPE_START_HTML_ENTITY byte = '&'

// FILTER_HTML ignores tag width and treats each entity as one character.
const FILTER_HTML Flags = Flags(utf8.CHARACTER_SIZE_MINIMUM) << bytes.SLICE_SIZE_MINIMUM

// STRIP_ESCAPE removes Escape boundary bytes.
const STRIP_ESCAPE Flags = FILTER_HTML << utf8.CHARACTER_SIZE_MINIMUM

// ALIGN_RIGHT pads before cell text.
const ALIGN_RIGHT Flags = STRIP_ESCAPE << utf8.CHARACTER_SIZE_MINIMUM

// DISCARD_EMPTY_COLUMNS removes columns made only from empty soft cells.
const DISCARD_EMPTY_COLUMNS Flags = ALIGN_RIGHT << utf8.CHARACTER_SIZE_MINIMUM

// TAB_INDENT uses tabs for leading empty-cell padding.
const TAB_INDENT Flags = DISCARD_EMPTY_COLUMNS << utf8.CHARACTER_SIZE_MINIMUM

// DEBUG emits visible column and form-feed boundaries.
const DEBUG Flags = TAB_INDENT << utf8.CHARACTER_SIZE_MINIMUM

// FLAGS_MAXIMUM includes every defined formatting flag.
const FLAGS_MAXIMUM = FILTER_HTML | STRIP_ESCAPE | ALIGN_RIGHT |
	DISCARD_EMPTY_COLUMNS | TAB_INDENT | DEBUG

// ESCAPE cannot occur inside valid UTF-8 and therefore cannot collide with text characters.
const ESCAPE byte = bits.WORD_8_MAXIMUM

// STATUS_OK means the requested operation completed.
const STATUS_OK = bytes.SLICE_SIZE_MINIMUM

// STATUS_OUTPUT_TOO_SMALL means caller output cannot hold the exact bounded result.
const STATUS_OUTPUT_TOO_SMALL = STATUS_OK + utf8.CHARACTER_SIZE_MINIMUM

// STATUS_RESULT_TOO_LARGE means formatted output exceeds repository storage boundary.
const STATUS_RESULT_TOO_LARGE = STATUS_OUTPUT_TOO_SMALL + utf8.CHARACTER_SIZE_MINIMUM

// STATUS_STORAGE_INVALID means source and output overlap.
const STATUS_STORAGE_INVALID = STATUS_RESULT_TOO_LARGE + utf8.CHARACTER_SIZE_MINIMUM

// STATUS_WORKSPACE_INVALID means caller omitted required formatting state.
const STATUS_WORKSPACE_INVALID = STATUS_STORAGE_INVALID + utf8.CHARACTER_SIZE_MINIMUM

// STATUS_CONFIGURATION_INVALID means scalar policy lies outside bounded domain.
const STATUS_CONFIGURATION_INVALID = STATUS_WORKSPACE_INVALID + utf8.CHARACTER_SIZE_MINIMUM

// STATUS_INPUT_INVALID means hostile source exceeds repository boundary.
const STATUS_INPUT_INVALID = STATUS_CONFIGURATION_INVALID + utf8.CHARACTER_SIZE_MINIMUM

// Minimum_Width is hostile minimum column width before validation.
type Minimum_Width int

// Minimum_Width_Invariants keeps the complete caller scalar domain visible.
func Minimum_Width_Invariants(value Minimum_Width, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), bits.INTEGER_MINIMUM, bits.INTEGER_MAXIMUM).
		Ensure()
}

// Tab_Width is hostile display width of one tab before validation.
type Tab_Width int

// Tab_Width_Invariants keeps the complete caller scalar domain visible.
func Tab_Width_Invariants(value Tab_Width, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), bits.INTEGER_MINIMUM, bits.INTEGER_MAXIMUM).
		Ensure()
}

// Padding is hostile extra column width before validation.
type Padding int

// Padding_Invariants keeps the complete caller scalar domain visible.
func Padding_Invariants(value Padding, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), bits.INTEGER_MINIMUM, bits.INTEGER_MAXIMUM).
		Ensure()
}

// Pad_Character is the output byte used for non-tab padding.
type Pad_Character byte

// Pad_Character_Invariants covers every caller byte.
func Pad_Character_Invariants(value Pad_Character, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), bits.WORD_8_MINIMUM, bits.WORD_8_MAXIMUM).
		Ensure()
}

// Flags is hostile formatting policy before validation.
type Flags uint

// Flags_Invariants keeps the complete caller word domain visible.
func Flags_Invariants(value Flags, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint(uint(value), bits.WORD_MINIMUM, bits.WORD_MAXIMUM).
		Ensure()
}

// Configuration_Input is caller policy before bounded validation.
type Configuration_Input struct {
	// Minimum_Width is the least aligned cell width.
	Minimum_Width Minimum_Width
	// Tab_Width is the displayed width of tab padding.
	Tab_Width Tab_Width
	// Padding extends each aligned cell width.
	Padding Padding
	// Pad_Character fills non-tab padding.
	Pad_Character Pad_Character
	// Flags selects formatting rules.
	Flags Flags
}

// Configuration_Input_Invariants composes every hostile configuration scalar once.
func Configuration_Input_Invariants(
	value Configuration_Input, namespace aver.Namespace,
) {
	Minimum_Width_Invariants(value.Minimum_Width, namespace)
	Tab_Width_Invariants(value.Tab_Width, namespace)
	Padding_Invariants(value.Padding, namespace)
	Pad_Character_Invariants(value.Pad_Character, namespace)
	Flags_Invariants(value.Flags, namespace)
}

// Minimum_Width_Storage keeps caller mutation status-reportable.
type Minimum_Width_Storage interface{}

// Tab_Width_Storage keeps caller mutation status-reportable.
type Tab_Width_Storage interface{}

// Padding_Storage keeps caller mutation status-reportable.
type Padding_Storage interface{}

// Pad_Character_Storage keeps caller mutation status-reportable.
type Pad_Character_Storage interface{}

// Flags_Storage keeps caller mutation status-reportable.
type Flags_Storage interface{}

// Configuration stores validated formatting policy by value.
type Configuration struct {
	// Minimum_Width stays shape-safe when caller corrupts its value.
	Minimum_Width Minimum_Width_Storage
	// Tab_Width stays shape-safe when caller corrupts its value.
	Tab_Width Tab_Width_Storage
	// Padding stays shape-safe when caller corrupts its value.
	Padding Padding_Storage
	// Pad_Character stays shape-safe when caller corrupts its value.
	Pad_Character Pad_Character_Storage
	// Flags stays shape-safe when caller corrupts its value.
	Flags Flags_Storage
}

// Configuration_Invariants admits zero failure output and typed configured storage.
func Configuration_Invariants(value Configuration, _ aver.Namespace) {
	_, minimum_width_valid := value.Minimum_Width.(Minimum_Width)
	_, tab_width_valid := value.Tab_Width.(Tab_Width)
	_, padding_valid := value.Padding.(Padding)
	_, pad_character_valid := value.Pad_Character.(Pad_Character)
	_, flags_valid := value.Flags.(Flags)
	aver.Always(
		minimum_width_valid == (value.Minimum_Width != nil),
		"Configuration minimum width has expected type.",
	)
	aver.Always(
		tab_width_valid == (value.Tab_Width != nil),
		"Configuration tab width has expected type.",
	)
	aver.Always(
		padding_valid == (value.Padding != nil),
		"Configuration padding has expected type.",
	)
	aver.Always(
		pad_character_valid == (value.Pad_Character != nil),
		"Configuration pad character has expected type.",
	)
	aver.Always(
		flags_valid == (value.Flags != nil),
		"Configuration flags have expected type.",
	)
}

// Configuration_Validity reports whether visible storage remains bounded.
type Configuration_Validity bool

// Configuration_Validity_Invariants covers usable and corrupted policy.
func Configuration_Validity_Invariants(
	value Configuration_Validity, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Tabwriter configuration storage is valid.").
		Ensure()
}

// Configuration_Valid validates caller-visible policy before formatting work.
func Configuration_Valid(
	configuration Configuration,
) (valid Configuration_Validity) {
	defer func() {
		Configuration_Validity_Invariants(valid, "configuration_valid.valid")
	}()
	Configuration_Invariants(configuration, "configuration_valid.configuration")
	minimum_width, minimum_width_valid := configuration.Minimum_Width.(Minimum_Width)
	tab_width, tab_width_valid := configuration.Tab_Width.(Tab_Width)
	padding, padding_valid := configuration.Padding.(Padding)
	pad_character, pad_character_valid := configuration.Pad_Character.(Pad_Character)
	flags, flags_valid := configuration.Flags.(Flags)
	if !minimum_width_valid {
		return false
	}
	if !tab_width_valid {
		return false
	}
	if !padding_valid {
		return false
	}
	if !pad_character_valid {
		return false
	}
	if !flags_valid {
		return false
	}
	if minimum_width < WIDTH_MINIMUM {
		return false
	}
	if minimum_width > WIDTH_MAXIMUM {
		return false
	}
	if tab_width < WIDTH_MINIMUM {
		return false
	}
	if tab_width > WIDTH_MAXIMUM {
		return false
	}
	if padding < WIDTH_MINIMUM {
		return false
	}
	if padding > WIDTH_MAXIMUM {
		return false
	}
	if flags > FLAGS_MAXIMUM {
		return false
	}
	if pad_character == '\t' {
		if flags&ALIGN_RIGHT != Flags(bits.WORD_MINIMUM) {
			return false
		}
	}
	return true
}

// Configuration_Status reports valid or rejected policy.
type Configuration_Status uint8

// Configuration_Status_Invariants lists both construction outcomes.
func Configuration_Status_Invariants(
	value Configuration_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_CONFIGURATION_INVALID),
		).
		Ensure()
}

// Source_Unvalidated is hostile borrowed input before size validation.
type Source_Unvalidated []byte

// Source_Unvalidated_Invariants bounds validation work itself.
func Source_Unvalidated_Invariants(
	value Source_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), SOURCE_SIZE_MINIMUM, SOURCE_SIZE_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Source is validated borrowed tabular input.
type Source []byte

// Source_Invariants keeps parsing inside workspace capacity.
func Source_Invariants(value Source, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), SOURCE_SIZE_MINIMUM, SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Nonempty_Source is source known to contain a retained cell.
type Nonempty_Source []byte

// Nonempty_Source_Invariants excludes calls impossible for cell output.
func Nonempty_Source_Invariants(value Nonempty_Source, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), NONEMPTY_CELL_END_MINIMUM, SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Source_Status reports accepted or oversized input.
type Source_Status uint8

// Source_Status_Invariants lists both validation outcomes.
func Source_Status_Invariants(value Source_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(STATUS_OK), uint8(STATUS_INPUT_INVALID)).
		Ensure()
}

// Output is caller-owned formatted byte storage.
type Output []byte

// Output_Invariants follows repository byte-slice boundary.
func Output_Invariants(value Output, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), OUTPUT_SIZE_MINIMUM, OUTPUT_SIZE_MAXIMUM).
		Ensure()
}

// Output_Count is exact bounded size or first unrepresentable size.
type Output_Count int

// Output_Count_Invariants includes the result-too-large sentinel.
func Output_Count_Invariants(value Output_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), OUTPUT_SIZE_MINIMUM, OUTPUT_SIZE_UNREPRESENTABLE,
		).
		Ensure()
}

// Format_Status reports every formatting and storage outcome.
type Format_Status uint8

// Format_Status_Invariants covers the contiguous runtime outcome range.
func Format_Status_Invariants(value Format_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_CONFIGURATION_INVALID),
		).
		Ensure()
}

// Cell_Start is the first source byte in one cell.
type Cell_Start int

// Cell_Start_Invariants includes source end for empty trailing cells.
func Cell_Start_Invariants(value Cell_Start, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), SOURCE_SIZE_MINIMUM, SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Stored_Cell_Start is the first byte of a real workspace cell.
type Stored_Cell_Start int

// Stored_Cell_Start_Invariants excludes trailing source boundary without a cell.
func Stored_Cell_Start_Invariants(
	value Stored_Cell_Start, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), SOURCE_SIZE_MINIMUM, STORED_CELL_START_MAXIMUM).
		Ensure()
}

// Cell_End is the source boundary after one cell.
type Cell_End int

// Cell_End_Invariants includes every bounded source boundary.
func Cell_End_Invariants(value Cell_End, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), SOURCE_SIZE_MINIMUM, SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Nonempty_Cell_End follows at least one retained source byte.
type Nonempty_Cell_End int

// Nonempty_Cell_End_Invariants excludes empty cell boundary.
func Nonempty_Cell_End_Invariants(
	value Nonempty_Cell_End, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), NONEMPTY_CELL_END_MINIMUM, SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Cell_Width is the decoded display width before padding.
type Cell_Width int

// Cell_Width_Invariants admits one replacement character per invalid byte.
func Cell_Width_Invariants(value Cell_Width, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), WIDTH_MINIMUM, SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Partial_Cell_Width is width accumulated before final source segment.
type Partial_Cell_Width int

// Partial_Cell_Width_Invariants leaves room for a final escape opener.
func Partial_Cell_Width_Invariants(
	value Partial_Cell_Width, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), WIDTH_MINIMUM, PARTIAL_CELL_WIDTH_MAXIMUM).
		Ensure()
}

// Aligned_Cell_Width leaves room for terminator and trailing source cell.
type Aligned_Cell_Width int

// Aligned_Cell_Width_Invariants bounds padding input to reachable aligned text.
func Aligned_Cell_Width_Invariants(
	value Aligned_Cell_Width, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), WIDTH_MINIMUM, ALIGNED_CELL_WIDTH_MAXIMUM).
		Ensure()
}

// Cell_Output_Size is retained byte count after escape stripping.
type Cell_Output_Size int

// Cell_Output_Size_Invariants follows source byte count.
func Cell_Output_Size_Invariants(
	value Cell_Output_Size, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), OUTPUT_SIZE_MINIMUM, SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Cell_Hard_Tab records horizontal rather than vertical termination.
type Cell_Hard_Tab bool

// Cell_Hard_Tab_Invariants covers hard and soft cells.
func Cell_Hard_Tab_Invariants(value Cell_Hard_Tab, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "A cell uses a hard tab.").
		Ensure()
}

// Line_First_Cell is the first flat cell owned by one line.
type Line_First_Cell int

// Line_First_Cell_Invariants includes boundary after final source cell.
func Line_First_Cell_Invariants(
	value Line_First_Cell, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), CELL_INDEX_MINIMUM, CELL_COUNT_MAXIMUM).
		Ensure()
}

// Line_Cell_Boundary is the flat boundary after one line's cells.
type Line_Cell_Boundary int

// Line_Cell_Boundary_Invariants includes boundary after final source cell.
func Line_Cell_Boundary_Invariants(
	value Line_Cell_Boundary, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), CELL_INDEX_MINIMUM, CELL_COUNT_MAXIMUM).
		Ensure()
}

// Cell_Index selects one workspace cell.
type Cell_Index int

// Cell_Index_Invariants includes the boundary after final cell.
func Cell_Index_Invariants(value Cell_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), CELL_INDEX_MINIMUM, CELL_INDEX_MAXIMUM).
		Ensure()
}

// Cell_Count is a bounded number of parsed cells.
type Cell_Count int

// Cell_Count_Invariants includes empty input.
func Cell_Count_Invariants(value Cell_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), CELL_COUNT_MINIMUM, CELL_COUNT_MAXIMUM).
		Ensure()
}

// Nonzero_Cell_Count follows one successful cell insertion.
type Nonzero_Cell_Count int

// Nonzero_Cell_Count_Invariants excludes pre-parse zero state.
func Nonzero_Cell_Count_Invariants(
	value Nonzero_Cell_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), NONZERO_CELL_COUNT_MINIMUM, CELL_COUNT_MAXIMUM).
		Ensure()
}

// Line_Cell_Count is cell count belonging to one source line.
type Line_Cell_Count int

// Line_Cell_Count_Invariants includes empty lines.
func Line_Cell_Count_Invariants(
	value Line_Cell_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), LINE_CELL_COUNT_MINIMUM, CELL_COUNT_MAXIMUM).
		Ensure()
}

// Parsed_Line_Count includes the final line added after parsing.
type Parsed_Line_Count int

// Parsed_Line_Count_Invariants excludes accumulator zero state.
func Parsed_Line_Count_Invariants(
	value Parsed_Line_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), PARSED_LINE_COUNT_MINIMUM, LINE_COUNT_MAXIMUM).
		Ensure()
}

// Line_Index selects one parsed line.
type Line_Index int

// Line_Index_Invariants excludes boundary after final line.
func Line_Index_Invariants(value Line_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), LINE_INDEX_MINIMUM, LINE_INDEX_MAXIMUM).
		Ensure()
}

// Column_Index selects one cell position inside a line.
type Column_Index int

// Column_Index_Invariants follows maximum cells on one line.
func Column_Index_Invariants(value Column_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), COLUMN_INDEX_MINIMUM, COLUMN_INDEX_MAXIMUM).
		Ensure()
}

// Column_Line_Index is a line index with room for an aligned cell.
type Column_Line_Index int

// Column_Line_Index_Invariants excludes source tails too short for a column.
func Column_Line_Index_Invariants(
	value Column_Line_Index, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), LINE_INDEX_MINIMUM, COLUMN_LINE_INDEX_MAXIMUM).
		Ensure()
}

// Column_Line_Count is parsed line count in a source containing a column.
type Column_Line_Count int

// Column_Line_Count_Invariants excludes all-newline maximum source.
func Column_Line_Count_Invariants(
	value Column_Line_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), COLUMN_LINE_COUNT_MINIMUM, COLUMN_LINE_COUNT_MAXIMUM).
		Ensure()
}

// Column_Width is cell width after bounded padding.
type Column_Width int

// Column_Width_Invariants includes maximum cell and padding widths.
func Column_Width_Invariants(value Column_Width, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), WIDTH_MINIMUM, COLUMN_WIDTH_MAXIMUM).
		Ensure()
}

// Append_Offset is next writable byte before result-too-large sentinel.
type Append_Offset int

// Append_Offset_Invariants stops at maximum caller output boundary.
func Append_Offset_Invariants(value Append_Offset, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), OUTPUT_SIZE_MINIMUM, OUTPUT_SIZE_MAXIMUM).
		Ensure()
}

// APPENDED_OFFSET_MINIMUM follows one emitted byte beyond empty output.
const APPENDED_OFFSET_MINIMUM = OUTPUT_SIZE_MINIMUM + utf8.CHARACTER_SIZE_MINIMUM

// Appended_Offset is populated output after a mandatory write path.
type Appended_Offset int

// Appended_Offset_Invariants excludes empty output after mandatory emission.
func Appended_Offset_Invariants(value Appended_Offset, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), APPENDED_OFFSET_MINIMUM, OUTPUT_SIZE_MAXIMUM).
		Ensure()
}

// Line_Terminator distinguishes incomplete, newline, and form-feed lines.
type Line_Terminator byte

// Line_Terminator_Invariants lists every parsed line ending.
func Line_Terminator_Invariants(
	value Line_Terminator, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), LINE_TERMINATOR_NONE,
			LINE_TERMINATOR_NEWLINE, LINE_TERMINATOR_FORM_FEED,
		).
		Ensure()
}

// Cell_Starts borrows caller storage for source beginnings.
type Cell_Starts []Stored_Cell_Start

// Cell_Starts_Invariants fixes complete caller storage.
func Cell_Starts_Invariants(value Cell_Starts, _ aver.Namespace) {
	aver.Always(len(value) == CELL_COUNT_MAXIMUM, "Cell starts retain complete length.")
	aver.Always(cap(value) == CELL_COUNT_MAXIMUM, "Cell starts cannot grow.")
}

// Cell_Ends borrows caller storage for source endings.
type Cell_Ends []Cell_End

// Cell_Ends_Invariants fixes complete caller storage.
func Cell_Ends_Invariants(value Cell_Ends, _ aver.Namespace) {
	aver.Always(len(value) == CELL_COUNT_MAXIMUM, "Cell ends retain complete length.")
	aver.Always(cap(value) == CELL_COUNT_MAXIMUM, "Cell ends cannot grow.")
}

// Cell_Widths borrows caller storage for decoded widths.
type Cell_Widths []Cell_Width

// Cell_Widths_Invariants fixes complete caller storage.
func Cell_Widths_Invariants(value Cell_Widths, _ aver.Namespace) {
	aver.Always(len(value) == CELL_COUNT_MAXIMUM, "Cell widths retain complete length.")
	aver.Always(cap(value) == CELL_COUNT_MAXIMUM, "Cell widths cannot grow.")
}

// Cell_Output_Sizes borrows caller storage for retained byte sizes.
type Cell_Output_Sizes []Cell_Output_Size

// Cell_Output_Sizes_Invariants fixes complete caller storage.
func Cell_Output_Sizes_Invariants(value Cell_Output_Sizes, _ aver.Namespace) {
	aver.Always(len(value) == CELL_COUNT_MAXIMUM, "Cell output sizes retain complete length.")
	aver.Always(cap(value) == CELL_COUNT_MAXIMUM, "Cell output sizes cannot grow.")
}

// Cell_Hard_Tabs borrows caller storage for tab kind.
type Cell_Hard_Tabs []Cell_Hard_Tab

// Cell_Hard_Tabs_Invariants fixes complete caller storage.
func Cell_Hard_Tabs_Invariants(value Cell_Hard_Tabs, _ aver.Namespace) {
	aver.Always(len(value) == CELL_COUNT_MAXIMUM, "Cell tab kinds retain complete length.")
	aver.Always(cap(value) == CELL_COUNT_MAXIMUM, "Cell tab kinds cannot grow.")
}

// Line_First_Cells stores flat cell beginnings for each line.
type Line_First_Cells []Line_First_Cell

// Line_First_Cells_Invariants fixes complete caller storage.
func Line_First_Cells_Invariants(value Line_First_Cells, _ aver.Namespace) {
	aver.Always(
		len(value) == LINE_COUNT_MAXIMUM, "Line cell beginnings retain complete length.",
	)
	aver.Always(cap(value) == LINE_COUNT_MAXIMUM, "Line cell beginnings cannot grow.")
}

// Line_Cell_Counts stores flat cell counts for each line.
type Line_Cell_Counts []Line_Cell_Count

// Line_Cell_Counts_Invariants fixes complete caller storage.
func Line_Cell_Counts_Invariants(value Line_Cell_Counts, _ aver.Namespace) {
	aver.Always(len(value) == LINE_COUNT_MAXIMUM, "Line cell counts retain complete length.")
	aver.Always(cap(value) == LINE_COUNT_MAXIMUM, "Line cell counts cannot grow.")
}

// Line_Terminators stores source line-ending kind.
type Line_Terminators []Line_Terminator

// Line_Terminators_Invariants fixes complete caller storage.
func Line_Terminators_Invariants(value Line_Terminators, _ aver.Namespace) {
	aver.Always(len(value) == LINE_COUNT_MAXIMUM, "Line endings retain complete length.")
	aver.Always(cap(value) == LINE_COUNT_MAXIMUM, "Line endings cannot grow.")
}

// Workspace is caller-owned cell and line metadata.
type Workspace struct {
	// Cell_Starts keeps source beginnings in caller state.
	Cell_Starts Cell_Starts
	// Cell_Ends keeps source endings in caller state.
	Cell_Ends Cell_Ends
	// Cell_Widths keeps decoded widths in caller state.
	Cell_Widths Cell_Widths
	// Cell_Output_Size keeps retained byte sizes in caller state.
	Cell_Output_Size Cell_Output_Sizes
	// Cell_Hard_Tabs keeps horizontal-tab decisions in caller state.
	Cell_Hard_Tabs Cell_Hard_Tabs
	// Line_First_Cells keeps flat line beginnings in caller state.
	Line_First_Cells Line_First_Cells
	// Line_Cell_Counts keeps line sizes in caller state.
	Line_Cell_Counts Line_Cell_Counts
	// Line_Terminators keeps line boundary kinds in caller state.
	Line_Terminators Line_Terminators
}

// Workspace_Invariants verifies fixed state shape without reading stale content.
func Workspace_Invariants(value Workspace, namespace aver.Namespace) {
	Cell_Starts_Invariants(value.Cell_Starts, namespace)
	Cell_Ends_Invariants(value.Cell_Ends, namespace)
	Cell_Widths_Invariants(value.Cell_Widths, namespace)
	Cell_Output_Sizes_Invariants(value.Cell_Output_Size, namespace)
	Cell_Hard_Tabs_Invariants(value.Cell_Hard_Tabs, namespace)
	Line_First_Cells_Invariants(value.Line_First_Cells, namespace)
	Line_Cell_Counts_Invariants(value.Line_Cell_Counts, namespace)
	Line_Terminators_Invariants(value.Line_Terminators, namespace)
}

// Workspace_Pointer names validated mutable metadata state.
type Workspace_Pointer *Workspace

// Workspace_Pointer_Invariants composes present metadata state.
func Workspace_Pointer_Invariants(value Workspace_Pointer, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Workspace_Invariants(*value, namespace)
}

// Workspace_Validity reports complete caller storage without indexing it.
type Workspace_Validity bool

// Workspace_Validity_Invariants covers valid and rejected storage.
func Workspace_Validity_Invariants(value Workspace_Validity, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Workspace storage is complete.").
		Ensure()
}

// Cell_Starts_Input retains hostile borrowed source-beginning storage.
type Cell_Starts_Input []Stored_Cell_Start

// Cell_Starts_Input_Invariants states Go slice shape without scanning hostile storage.
func Cell_Starts_Input_Invariants(value Cell_Starts_Input, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Int(len(value), CELL_COUNT_MINIMUM, CELL_COUNT_MAXIMUM).
		Enum_Int(cap(value), CELL_COUNT_MINIMUM, CELL_COUNT_MAXIMUM).
		Ensure()
}

// Cell_Ends_Input retains hostile borrowed source-ending storage.
type Cell_Ends_Input []Cell_End

// Cell_Ends_Input_Invariants states Go slice shape without scanning hostile storage.
func Cell_Ends_Input_Invariants(value Cell_Ends_Input, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Int(len(value), CELL_COUNT_MINIMUM, CELL_COUNT_MAXIMUM).
		Enum_Int(cap(value), CELL_COUNT_MINIMUM, CELL_COUNT_MAXIMUM).
		Ensure()
}

// Cell_Widths_Input retains hostile borrowed display-width storage.
type Cell_Widths_Input []Cell_Width

// Cell_Widths_Input_Invariants states Go slice shape without scanning hostile storage.
func Cell_Widths_Input_Invariants(value Cell_Widths_Input, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Int(len(value), CELL_COUNT_MINIMUM, CELL_COUNT_MAXIMUM).
		Enum_Int(cap(value), CELL_COUNT_MINIMUM, CELL_COUNT_MAXIMUM).
		Ensure()
}

// Cell_Output_Sizes_Input retains hostile borrowed byte-count storage.
type Cell_Output_Sizes_Input []Cell_Output_Size

// Cell_Output_Sizes_Input_Invariants states Go slice shape without scanning hostile storage.
func Cell_Output_Sizes_Input_Invariants(
	value Cell_Output_Sizes_Input, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Int(len(value), CELL_COUNT_MINIMUM, CELL_COUNT_MAXIMUM).
		Enum_Int(cap(value), CELL_COUNT_MINIMUM, CELL_COUNT_MAXIMUM).
		Ensure()
}

// Cell_Hard_Tabs_Input retains hostile borrowed tab-kind storage.
type Cell_Hard_Tabs_Input []Cell_Hard_Tab

// Cell_Hard_Tabs_Input_Invariants states Go slice shape without scanning hostile storage.
func Cell_Hard_Tabs_Input_Invariants(
	value Cell_Hard_Tabs_Input, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Int(len(value), CELL_COUNT_MINIMUM, CELL_COUNT_MAXIMUM).
		Enum_Int(cap(value), CELL_COUNT_MINIMUM, CELL_COUNT_MAXIMUM).
		Ensure()
}

// Line_First_Cells_Input retains hostile borrowed line-beginning storage.
type Line_First_Cells_Input []Line_First_Cell

// Line_First_Cells_Input_Invariants states Go slice shape without scanning hostile storage.
func Line_First_Cells_Input_Invariants(
	value Line_First_Cells_Input, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Int(len(value), LINE_COUNT_MINIMUM, LINE_COUNT_MAXIMUM).
		Enum_Int(cap(value), LINE_COUNT_MINIMUM, LINE_COUNT_MAXIMUM).
		Ensure()
}

// Line_Cell_Counts_Input retains hostile borrowed line-size storage.
type Line_Cell_Counts_Input []Line_Cell_Count

// Line_Cell_Counts_Input_Invariants states Go slice shape without scanning hostile storage.
func Line_Cell_Counts_Input_Invariants(
	value Line_Cell_Counts_Input, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Int(len(value), LINE_COUNT_MINIMUM, LINE_COUNT_MAXIMUM).
		Enum_Int(cap(value), LINE_COUNT_MINIMUM, LINE_COUNT_MAXIMUM).
		Ensure()
}

// Line_Terminators_Input retains hostile borrowed line-ending storage.
type Line_Terminators_Input []Line_Terminator

// Line_Terminators_Input_Invariants states Go slice shape without scanning hostile storage.
func Line_Terminators_Input_Invariants(
	value Line_Terminators_Input, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Int(len(value), LINE_COUNT_MINIMUM, LINE_COUNT_MAXIMUM).
		Enum_Int(cap(value), LINE_COUNT_MINIMUM, LINE_COUNT_MAXIMUM).
		Ensure()
}

// Workspace_Valid checks every slice before parser indexes caller storage.
func Workspace_Valid(storage Workspace_Storage) (valid Workspace_Validity) {
	defer func() {
		Workspace_Validity_Invariants(valid, "workspace_valid.valid")
	}()
	Workspace_Storage_Invariants(storage, "workspace_valid.storage")
	return Workspace_Validity(
		len(storage.Cell_Starts) == CELL_COUNT_MAXIMUM &&
			cap(storage.Cell_Starts) == CELL_COUNT_MAXIMUM &&
			len(storage.Cell_Ends) == CELL_COUNT_MAXIMUM &&
			cap(storage.Cell_Ends) == CELL_COUNT_MAXIMUM &&
			len(storage.Cell_Widths) == CELL_COUNT_MAXIMUM &&
			cap(storage.Cell_Widths) == CELL_COUNT_MAXIMUM &&
			len(storage.Cell_Output_Size) == CELL_COUNT_MAXIMUM &&
			cap(storage.Cell_Output_Size) == CELL_COUNT_MAXIMUM &&
			len(storage.Cell_Hard_Tabs) == CELL_COUNT_MAXIMUM &&
			cap(storage.Cell_Hard_Tabs) == CELL_COUNT_MAXIMUM &&
			len(storage.Line_First_Cells) == LINE_COUNT_MAXIMUM &&
			cap(storage.Line_First_Cells) == LINE_COUNT_MAXIMUM &&
			len(storage.Line_Cell_Counts) == LINE_COUNT_MAXIMUM &&
			cap(storage.Line_Cell_Counts) == LINE_COUNT_MAXIMUM &&
			len(storage.Line_Terminators) == LINE_COUNT_MAXIMUM &&
			cap(storage.Line_Terminators) == LINE_COUNT_MAXIMUM,
	)
}

// Workspace_Storage retains hostile caller slices until validation.
type Workspace_Storage struct {
	// Cell_Starts remains borrowed until exact length validation.
	Cell_Starts Cell_Starts_Input
	// Cell_Ends remains borrowed until exact length validation.
	Cell_Ends Cell_Ends_Input
	// Cell_Widths remains borrowed until exact length validation.
	Cell_Widths Cell_Widths_Input
	// Cell_Output_Size remains borrowed until exact length validation.
	Cell_Output_Size Cell_Output_Sizes_Input
	// Cell_Hard_Tabs remains borrowed until exact length validation.
	Cell_Hard_Tabs Cell_Hard_Tabs_Input
	// Line_First_Cells remains borrowed until exact length validation.
	Line_First_Cells Line_First_Cells_Input
	// Line_Cell_Counts remains borrowed until exact length validation.
	Line_Cell_Counts Line_Cell_Counts_Input
	// Line_Terminators remains borrowed until exact length validation.
	Line_Terminators Line_Terminators_Input
}

// Workspace_Storage_Invariants composes hostile borrowed slice shapes.
func Workspace_Storage_Invariants(value Workspace_Storage, namespace aver.Namespace) {
	Cell_Starts_Input_Invariants(value.Cell_Starts, namespace)
	Cell_Ends_Input_Invariants(value.Cell_Ends, namespace)
	Cell_Widths_Input_Invariants(value.Cell_Widths, namespace)
	Cell_Output_Sizes_Input_Invariants(value.Cell_Output_Size, namespace)
	Cell_Hard_Tabs_Input_Invariants(value.Cell_Hard_Tabs, namespace)
	Line_First_Cells_Input_Invariants(value.Line_First_Cells, namespace)
	Line_Cell_Counts_Input_Invariants(value.Line_Cell_Counts, namespace)
	Line_Terminators_Input_Invariants(value.Line_Terminators, namespace)
}

// Workspace_Input retains absent hostile state until Format_Into can return status.
type Workspace_Input struct {
	// State borrows caller-owned parser metadata when present.
	State Workspace_Storage
}

// Workspace_Input_Invariants fixes unvalidated pointer storage shape.
func Workspace_Input_Invariants(value Workspace_Input, namespace aver.Namespace) {
	Workspace_Storage_Invariants(value.State, namespace)
}

// Write_Output separates sizing from the second mutating pass.
type Write_Output bool

// Write_Output_Invariants covers sizing and writing passes.
func Write_Output_Invariants(value Write_Output, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The formatter writes caller output.").
		Ensure()
}

// Aligned_Cell distinguishes a column cell from trailing line text.
type Aligned_Cell bool

// Aligned_Cell_Invariants covers aligned and trailing cell output.
func Aligned_Cell_Invariants(value Aligned_Cell, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "A cell belongs to an aligned column.").
		Ensure()
}

// Use_Tabs selects tab padding for leading empty indentation.
type Use_Tabs bool

// Use_Tabs_Invariants covers tab and configured-character padding.
func Use_Tabs_Invariants(value Use_Tabs, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Padding uses tab characters.").
		Ensure()
}

// Input_Byte is one byte examined by escape parsing.
type Input_Byte byte

// Input_Byte_Invariants covers the complete source byte domain.
func Input_Byte_Invariants(value Input_Byte, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), bits.WORD_8_MINIMUM, bits.WORD_8_MAXIMUM).
		Ensure()
}

// Escape_Start_Byte is one manual or HTML escape opener.
type Escape_Start_Byte byte

// Escape_Start_Byte_Invariants lists every byte passed to escape construction.
func Escape_Start_Byte_Invariants(
	value Escape_Start_Byte, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), ESCAPE_START_HTML_ENTITY, ESCAPE_START_HTML_TAG, ESCAPE,
		).
		Ensure()
}

// Escape_Output_Size is zero for stripped boundary and one otherwise.
type Escape_Output_Size int

// Escape_Output_Size_Invariants excludes impossible multi-byte single-step output.
func Escape_Output_Size_Invariants(
	value Escape_Output_Size, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Int(int(value), OUTPUT_SIZE_MINIMUM, NONEMPTY_CELL_END_MINIMUM).
		Ensure()
}

// Active_End_Character is one currently open escape boundary.
type Active_End_Character byte

// Active_End_Character_Invariants excludes inactive zero parser state.
func Active_End_Character_Invariants(
	value Active_End_Character, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), END_CHARACTER_HTML_ENTITY,
			END_CHARACTER_HTML_TAG, ESCAPE,
		).
		Ensure()
}

// End_Character is no escape or one supported escape boundary.
type End_Character byte

// End_Character_Invariants lists every escape parser state.
func End_Character_Invariants(value End_Character, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), LINE_TERMINATOR_NONE, END_CHARACTER_HTML_ENTITY,
			END_CHARACTER_HTML_TAG, ESCAPE,
		).
		Ensure()
}

// Character_Count is decoded width of one bounded source segment.
type Character_Count int

// Character_Count_Invariants admits one replacement character per invalid byte.
func Character_Count_Invariants(value Character_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), WIDTH_MINIMUM, SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Output_Byte is one source, padding, or marker byte.
type Output_Byte byte

// Output_Byte_Invariants covers the complete byte domain.
func Output_Byte_Invariants(value Output_Byte, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), bits.WORD_8_MINIMUM, bits.WORD_8_MAXIMUM).
		Ensure()
}

// Append_Status reports successful append or bounded result overflow.
type Append_Status uint8

// Append_Status_Invariants lists both append outcomes.
func Append_Status_Invariants(value Append_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(STATUS_OK), uint8(STATUS_RESULT_TOO_LARGE)).
		Ensure()
}

// New_Configuration validates formatting policy without retaining a driver.
func New_Configuration(
	input Configuration_Input,
) (configuration Configuration, status Configuration_Status) {
	defer func() {
		Configuration_Invariants(configuration, "new_configuration.configuration")
		Configuration_Status_Invariants(status, "new_configuration.status")
	}()
	Configuration_Input_Invariants(input, "new_configuration.input")
	if input.Minimum_Width < WIDTH_MINIMUM {
		return Configuration{}, STATUS_CONFIGURATION_INVALID
	}
	if input.Minimum_Width > WIDTH_MAXIMUM {
		return Configuration{}, STATUS_CONFIGURATION_INVALID
	}
	if input.Tab_Width < WIDTH_MINIMUM {
		return Configuration{}, STATUS_CONFIGURATION_INVALID
	}
	if input.Tab_Width > WIDTH_MAXIMUM {
		return Configuration{}, STATUS_CONFIGURATION_INVALID
	}
	if input.Padding < WIDTH_MINIMUM {
		return Configuration{}, STATUS_CONFIGURATION_INVALID
	}
	if input.Padding > WIDTH_MAXIMUM {
		return Configuration{}, STATUS_CONFIGURATION_INVALID
	}
	if input.Flags > FLAGS_MAXIMUM {
		return Configuration{}, STATUS_CONFIGURATION_INVALID
	}
	flags := input.Flags
	if input.Pad_Character == '\t' {
		flags &^= ALIGN_RIGHT
	}
	return Configuration{
		Minimum_Width: input.Minimum_Width,
		Tab_Width:     input.Tab_Width,
		Padding:       input.Padding,
		Pad_Character: input.Pad_Character,
		Flags:         flags,
	}, STATUS_OK
}

// Source_Validate refuses oversized input before workspace mutation.
func Source_Validate(
	unvalidated Source_Unvalidated,
) (source Source, status Source_Status) {
	defer func() {
		Source_Invariants(source, "source_validate.source")
		Source_Status_Invariants(status, "source_validate.status")
	}()
	Source_Unvalidated_Invariants(unvalidated, "source_validate.unvalidated")
	if len(unvalidated) > SOURCE_SIZE_MAXIMUM {
		return nil, STATUS_INPUT_INVALID
	}
	return Source(unvalidated), STATUS_OK
}

// Format_Into aligns one complete source without hidden IO or owned storage.
func Format_Into(
	destination Output,
	source Source,
	configuration Configuration,
	workspace Workspace_Input,
) (count Output_Count, status Format_Status) {
	defer func() {
		Output_Count_Invariants(count, "format_into.count")
		Format_Status_Invariants(status, "format_into.status")
	}()
	Output_Invariants(destination, "format_into.destination")
	Source_Invariants(source, "format_into.source")
	Configuration_Invariants(configuration, "format_into.configuration")
	Workspace_Input_Invariants(workspace, "format_into.workspace")
	if !bool(Workspace_Valid(workspace.State)) {
		return Output_Count(OUTPUT_SIZE_MINIMUM), STATUS_WORKSPACE_INVALID
	}
	state := Workspace{
		Cell_Starts:      Cell_Starts(workspace.State.Cell_Starts),
		Cell_Ends:        Cell_Ends(workspace.State.Cell_Ends),
		Cell_Widths:      Cell_Widths(workspace.State.Cell_Widths),
		Cell_Output_Size: Cell_Output_Sizes(workspace.State.Cell_Output_Size),
		Cell_Hard_Tabs:   Cell_Hard_Tabs(workspace.State.Cell_Hard_Tabs),
		Line_First_Cells: Line_First_Cells(workspace.State.Line_First_Cells),
		Line_Cell_Counts: Line_Cell_Counts(workspace.State.Line_Cell_Counts),
		Line_Terminators: Line_Terminators(workspace.State.Line_Terminators),
	}
	Workspace_Invariants(state, "format_into.workspace_state")
	if !bool(Configuration_Valid(configuration)) {
		return Output_Count(OUTPUT_SIZE_MINIMUM), STATUS_CONFIGURATION_INVALID
	}
	if bool(bytes.Overlap(bytes.Slice(destination), bytes.Slice(source))) {
		return Output_Count(OUTPUT_SIZE_MINIMUM), STATUS_STORAGE_INVALID
	}
	line_count := parse_unchecked(source, configuration, &state)
	var append_status Append_Status
	count, append_status = render_unchecked(
		nil, source, configuration, &state, line_count, false,
	)
	if append_status != STATUS_OK {
		return count, Format_Status(append_status)
	}
	if len(destination) < int(count) {
		return count, STATUS_OUTPUT_TOO_SMALL
	}
	count, append_status = render_unchecked(
		destination, source, configuration, &state, line_count, true,
	)
	return count, Format_Status(append_status)
}

func parse_unchecked(
	source Source, configuration Configuration, workspace Workspace_Pointer,
) (line_count Parsed_Line_Count) {
	defer func() {
		Parsed_Line_Count_Invariants(line_count, "parse_unchecked.line_count")
	}()
	Source_Invariants(source, "parse_unchecked.source")
	Configuration_Invariants(configuration, "parse_unchecked.configuration")
	Workspace_Pointer_Invariants(workspace, "parse_unchecked.workspace")
	cell_start, segment_start := CELL_INDEX_MINIMUM, CELL_INDEX_MINIMUM
	cell_width, cell_output_size := WIDTH_MINIMUM, OUTPUT_SIZE_MINIMUM
	line_first, line_index := CELL_INDEX_MINIMUM, Line_Index(LINE_INDEX_MINIMUM)
	cell_count, end_character := Cell_Count(CELL_COUNT_MINIMUM),
		End_Character(LINE_TERMINATOR_NONE)
	for index, character := range source {
		if end_character != End_Character(LINE_TERMINATOR_NONE) {
			cell_output_size += int(escaped_output_size(
				Input_Byte(character),
				Active_End_Character(end_character), configuration,
			))
			if Input_Byte(character) == Input_Byte(end_character) {
				cell_width += int(escaped_width(
					source, Cell_Start(segment_start),
					Cell_End(index), end_character,
				))
				end_character = End_Character(LINE_TERMINATOR_NONE)
				segment_start = index + utf8.CHARACTER_SIZE_MINIMUM
			}
			continue
		}
		terminates_cell := character == '\t' || character == '\v' ||
			character == '\n' || character == '\f'
		if terminates_cell {
			cell_width += int(character_count(source[segment_start:index]))
			cell_count = Cell_Count(add_cell_unchecked(
				workspace, Cell_Index(cell_count), Stored_Cell_Start(cell_start),
				Cell_End(index),
				Cell_Width(cell_width), Cell_Output_Size(cell_output_size),
				Cell_Hard_Tab(character == '\t'),
			))
			cell_start, segment_start = index+utf8.CHARACTER_SIZE_MINIMUM,
				index+utf8.CHARACTER_SIZE_MINIMUM
			cell_width, cell_output_size = WIDTH_MINIMUM, OUTPUT_SIZE_MINIMUM
			terminates_line := character == '\n' || character == '\f'
			if terminates_line {
				line_index = Line_Index(add_line_unchecked(
					workspace, line_index, Line_First_Cell(line_first),
					Line_Cell_Boundary(cell_count), Line_Terminator(character),
				))
				line_first = int(cell_count)
			}
			continue
		}
		start_end := escape_start_unchecked(Input_Byte(character), configuration)
		if start_end != End_Character(LINE_TERMINATOR_NONE) {
			cell_width += int(character_count(source[segment_start:index]))
			end_character = start_end
			segment_start = index + utf8.CHARACTER_SIZE_MINIMUM
			cell_output_size += int(escape_start_output_size(
				Escape_Start_Byte(character), configuration,
			))
			continue
		}
		cell_output_size++
	}
	return parse_finish_unchecked(
		source, workspace, cell_count, line_index,
		Cell_Start(cell_start), Cell_Start(segment_start),
		Partial_Cell_Width(cell_width), Cell_Output_Size(cell_output_size),
		Line_First_Cell(line_first), end_character,
	)
}

func parse_finish_unchecked(
	source Source,
	workspace Workspace_Pointer,
	cell_count Cell_Count,
	line_index Line_Index,
	cell_start Cell_Start,
	segment_start Cell_Start,
	cell_width Partial_Cell_Width,
	cell_output_size Cell_Output_Size,
	line_first Line_First_Cell,
	end_character End_Character,
) (line_count Parsed_Line_Count) {
	defer func() {
		Parsed_Line_Count_Invariants(line_count, "parse_finish_unchecked.line_count")
	}()
	Source_Invariants(source, "parse_finish_unchecked.source")
	Workspace_Pointer_Invariants(workspace, "parse_finish_unchecked.workspace")
	Cell_Count_Invariants(cell_count, "parse_finish_unchecked.input_cell_count")
	Line_Index_Invariants(line_index, "parse_finish_unchecked.line_index")
	Cell_Start_Invariants(cell_start, "parse_finish_unchecked.cell_start")
	Cell_Start_Invariants(segment_start, "parse_finish_unchecked.segment_start")
	Partial_Cell_Width_Invariants(cell_width, "parse_finish_unchecked.cell_width")
	Cell_Output_Size_Invariants(
		cell_output_size, "parse_finish_unchecked.cell_output_size",
	)
	Line_First_Cell_Invariants(line_first, "parse_finish_unchecked.line_first")
	End_Character_Invariants(end_character, "parse_finish_unchecked.end_character")
	final_cell_width := Cell_Width(cell_width) + escaped_width(
		source, segment_start, Cell_End(len(source)), end_character,
	)
	if cell_output_size > Cell_Output_Size(OUTPUT_SIZE_MINIMUM) {
		cell_count = Cell_Count(add_cell_unchecked(
			workspace, Cell_Index(cell_count), Stored_Cell_Start(cell_start),
			Cell_End(len(source)), final_cell_width, cell_output_size, false,
		))
	}
	line_count = add_line_unchecked(
		workspace, line_index, line_first,
		Line_Cell_Boundary(cell_count), Line_Terminator(LINE_TERMINATOR_NONE),
	)
	return line_count
}

func escape_start_unchecked(
	character Input_Byte, configuration Configuration,
) (end End_Character) {
	defer func() { End_Character_Invariants(end, "escape_start_unchecked.end") }()
	Input_Byte_Invariants(character, "escape_start_unchecked.character")
	Configuration_Invariants(configuration, "escape_start_unchecked.configuration")
	if character == Input_Byte(ESCAPE) {
		return End_Character(escape_end(Escape_Start_Byte(character)))
	}
	if configuration.Flags.(Flags)&FILTER_HTML == Flags(bits.WORD_MINIMUM) {
		return End_Character(LINE_TERMINATOR_NONE)
	}
	if character == Input_Byte(ESCAPE_START_HTML_TAG) {
		return End_Character(escape_end(Escape_Start_Byte(character)))
	}
	if character == Input_Byte(ESCAPE_START_HTML_ENTITY) {
		return End_Character(escape_end(Escape_Start_Byte(character)))
	}
	return End_Character(LINE_TERMINATOR_NONE)
}

func escaped_output_size(
	character Input_Byte,
	end_character Active_End_Character,
	configuration Configuration,
) (size Escape_Output_Size) {
	defer func() {
		Escape_Output_Size_Invariants(size, "escaped_output_size.size")
	}()
	Input_Byte_Invariants(character, "escaped_output_size.character")
	Active_End_Character_Invariants(end_character, "escaped_output_size.end_character")
	Configuration_Invariants(configuration, "escaped_output_size.configuration")
	if character == Input_Byte(end_character) {
		if end_character == Active_End_Character(ESCAPE) {
			if configuration.Flags.(Flags)&STRIP_ESCAPE !=
				Flags(bits.WORD_MINIMUM) {
				return Escape_Output_Size(OUTPUT_SIZE_MINIMUM)
			}
		}
	}
	return Escape_Output_Size(utf8.CHARACTER_SIZE_MINIMUM)
}

func escaped_width(
	source Source,
	start Cell_Start,
	end Cell_End,
	end_character End_Character,
) (width Cell_Width) {
	defer func() { Cell_Width_Invariants(width, "escaped_width.width") }()
	Source_Invariants(source, "escaped_width.source")
	Cell_Start_Invariants(start, "escaped_width.start")
	Cell_End_Invariants(end, "escaped_width.end")
	End_Character_Invariants(end_character, "escaped_width.end_character")
	switch end_character {
	case End_Character(ESCAPE), End_Character(LINE_TERMINATOR_NONE):
		return Cell_Width(character_count(source[start:end]))
	case ';':
		return Cell_Width(utf8.CHARACTER_SIZE_MINIMUM)
	default:
		return Cell_Width(WIDTH_MINIMUM)
	}
}

func character_count(source Source) (count Character_Count) {
	defer func() { Character_Count_Invariants(count, "character_count.count") }()
	Source_Invariants(source, "character_count.source")
	return Character_Count(utf8.Character_Count(utf8.Bytes(source)))
}

func escape_end(character Escape_Start_Byte) (end Active_End_Character) {
	defer func() { Active_End_Character_Invariants(end, "escape_end.end") }()
	Escape_Start_Byte_Invariants(character, "escape_end.character")
	switch character {
	case Escape_Start_Byte(ESCAPE):
		return Active_End_Character(ESCAPE)
	case Escape_Start_Byte(ESCAPE_START_HTML_TAG):
		return Active_End_Character(END_CHARACTER_HTML_TAG)
	default:
		return Active_End_Character(END_CHARACTER_HTML_ENTITY)
	}
}

func escape_start_output_size(
	character Escape_Start_Byte, configuration Configuration,
) (size Escape_Output_Size) {
	defer func() {
		Escape_Output_Size_Invariants(size, "escape_start_output_size.size")
	}()
	Escape_Start_Byte_Invariants(character, "escape_start_output_size.character")
	Configuration_Invariants(configuration, "escape_start_output_size.configuration")
	if character == Escape_Start_Byte(ESCAPE) {
		if configuration.Flags.(Flags)&STRIP_ESCAPE !=
			Flags(bits.WORD_MINIMUM) {
			return Escape_Output_Size(OUTPUT_SIZE_MINIMUM)
		}
	}
	return Escape_Output_Size(utf8.CHARACTER_SIZE_MINIMUM)
}

func add_cell_unchecked(
	workspace Workspace_Pointer,
	index Cell_Index,
	start Stored_Cell_Start,
	end Cell_End,
	width Cell_Width,
	output_size Cell_Output_Size,
	hard_tab Cell_Hard_Tab,
) (count Nonzero_Cell_Count) {
	defer func() { Nonzero_Cell_Count_Invariants(count, "add_cell_unchecked.count") }()
	Workspace_Pointer_Invariants(workspace, "add_cell_unchecked.workspace")
	Cell_Index_Invariants(index, "add_cell_unchecked.index")
	Stored_Cell_Start_Invariants(start, "add_cell_unchecked.start")
	Cell_End_Invariants(end, "add_cell_unchecked.end")
	Cell_Width_Invariants(width, "add_cell_unchecked.width")
	Cell_Output_Size_Invariants(output_size, "add_cell_unchecked.output_size")
	Cell_Hard_Tab_Invariants(hard_tab, "add_cell_unchecked.hard_tab")
	workspace.Cell_Starts[index] = start
	workspace.Cell_Ends[index] = end
	workspace.Cell_Widths[index] = width
	workspace.Cell_Output_Size[index] = output_size
	workspace.Cell_Hard_Tabs[index] = hard_tab
	return Nonzero_Cell_Count(index + utf8.CHARACTER_SIZE_MINIMUM)
}

func add_line_unchecked(
	workspace Workspace_Pointer,
	index Line_Index,
	first Line_First_Cell,
	cell_boundary Line_Cell_Boundary,
	terminator Line_Terminator,
) (count Parsed_Line_Count) {
	defer func() { Parsed_Line_Count_Invariants(count, "add_line_unchecked.count") }()
	Workspace_Pointer_Invariants(workspace, "add_line_unchecked.workspace")
	Line_Index_Invariants(index, "add_line_unchecked.index")
	Line_First_Cell_Invariants(first, "add_line_unchecked.first")
	Line_Cell_Boundary_Invariants(cell_boundary, "add_line_unchecked.cell_boundary")
	Line_Terminator_Invariants(terminator, "add_line_unchecked.terminator")
	workspace.Line_First_Cells[index] = first
	workspace.Line_Cell_Counts[index] = Line_Cell_Count(
		cell_boundary - Line_Cell_Boundary(first),
	)
	workspace.Line_Terminators[index] = terminator
	return Parsed_Line_Count(index + utf8.CHARACTER_SIZE_MINIMUM)
}

func render_unchecked(
	destination Output,
	source Source,
	configuration Configuration,
	workspace Workspace_Pointer,
	line_count Parsed_Line_Count,
	write Write_Output,
) (count Output_Count, status Append_Status) {
	defer func() {
		Output_Count_Invariants(count, "render_unchecked.count")
		Append_Status_Invariants(status, "render_unchecked.status")
	}()
	Output_Invariants(destination, "render_unchecked.destination")
	Source_Invariants(source, "render_unchecked.source")
	Configuration_Invariants(configuration, "render_unchecked.configuration")
	Workspace_Pointer_Invariants(workspace, "render_unchecked.workspace")
	Parsed_Line_Count_Invariants(line_count, "render_unchecked.line_count")
	Write_Output_Invariants(write, "render_unchecked.write")
	offset := Append_Offset(OUTPUT_SIZE_MINIMUM)
	for line_index := LINE_INDEX_MINIMUM; line_index < int(line_count); line_index++ {
		var append_status Append_Status
		offset, append_status = render_line_unchecked(
			destination, source, configuration, workspace,
			Line_Index(line_index), line_count, write, offset,
		)
		if append_status != STATUS_OK {
			return OUTPUT_SIZE_UNREPRESENTABLE, append_status
		}
		terminator := workspace.Line_Terminators[line_index]
		if terminator != Line_Terminator(LINE_TERMINATOR_NONE) {
			var appended Appended_Offset
			appended, append_status = append_byte_unchecked(
				destination, offset, Output_Byte('\n'), write,
			)
			offset = Append_Offset(appended)
			if append_status != STATUS_OK {
				return OUTPUT_SIZE_UNREPRESENTABLE, append_status
			}
		}
		if terminator == '\f' {
			if configuration.Flags.(Flags)&DEBUG !=
				Flags(bits.WORD_MINIMUM) {
				for _, character := range [...]byte{'-', '-', '-', '\n'} {
					var appended Appended_Offset
					appended, append_status = append_byte_unchecked(
						destination, offset, Output_Byte(character), write,
					)
					offset = Append_Offset(appended)
					if append_status != STATUS_OK {
						return OUTPUT_SIZE_UNREPRESENTABLE, append_status
					}
				}
			}
		}
	}
	return Output_Count(offset), STATUS_OK
}

func render_line_unchecked(
	destination Output,
	source Source,
	configuration Configuration,
	workspace Workspace_Pointer,
	line_index Line_Index,
	line_count Parsed_Line_Count,
	write Write_Output,
	count_value Append_Offset,
) (count Append_Offset, status Append_Status) {
	defer func() {
		Append_Offset_Invariants(count, "render_line_unchecked.count")
		Append_Status_Invariants(status, "render_line_unchecked.status")
	}()
	Output_Invariants(destination, "render_line_unchecked.destination")
	Source_Invariants(source, "render_line_unchecked.source")
	Configuration_Invariants(configuration, "render_line_unchecked.configuration")
	Workspace_Pointer_Invariants(workspace, "render_line_unchecked.workspace")
	Line_Index_Invariants(line_index, "render_line_unchecked.line_index")
	Parsed_Line_Count_Invariants(line_count, "render_line_unchecked.line_count")
	Write_Output_Invariants(write, "render_line_unchecked.write")
	Append_Offset_Invariants(count_value, "render_line_unchecked.count_value")
	count = Append_Offset(count_value)
	first := int(workspace.Line_First_Cells[line_index])
	cell_count := int(workspace.Line_Cell_Counts[line_index])
	flags := configuration.Flags.(Flags)
	use_tabs := flags&TAB_INDENT != Flags(bits.WORD_MINIMUM)
	for column_index := COLUMN_INDEX_MINIMUM; column_index < cell_count; column_index++ {
		index := first + column_index
		if column_index > COLUMN_INDEX_MINIMUM {
			if flags&DEBUG != Flags(bits.WORD_MINIMUM) {
				appended, append_status := append_byte_unchecked(
					destination, count, Output_Byte('|'), write,
				)
				count, status = Append_Offset(appended), append_status
				if status != STATUS_OK {
					return count, status
				}
			}
		}
		column_width := Column_Width(WIDTH_MINIMUM)
		if column_index < cell_count-utf8.CHARACTER_SIZE_MINIMUM {
			column_width = column_width_unchecked(
				workspace, configuration, Column_Line_Index(line_index),
				Column_Index(column_index), Column_Line_Count(line_count),
			)
		}
		if workspace.Cell_Output_Size[index] == Cell_Output_Size(OUTPUT_SIZE_MINIMUM) {
			if column_index < cell_count-utf8.CHARACTER_SIZE_MINIMUM {
				count, status = append_padding_unchecked(
					destination, configuration,
					Aligned_Cell_Width(workspace.Cell_Widths[index]),
					column_width, Use_Tabs(use_tabs), write, count,
				)
			}
		} else {
			use_tabs = false
			aligned := Aligned_Cell(
				column_index < cell_count-utf8.CHARACTER_SIZE_MINIMUM,
			)
			appended, append_status := append_nonempty_cell_unchecked(
				destination, Nonempty_Source(source), configuration,
				workspace, Cell_Index(index),
				column_width, aligned, write, count,
			)
			count, status = Append_Offset(appended), append_status
		}
		if status != STATUS_OK {
			return count, status
		}
	}
	return count, STATUS_OK
}

func append_nonempty_cell_unchecked(
	destination Output,
	source Nonempty_Source,
	configuration Configuration,
	workspace Workspace_Pointer,
	index Cell_Index,
	column_width Column_Width,
	aligned Aligned_Cell,
	write Write_Output,
	count_value Append_Offset,
) (count Appended_Offset, status Append_Status) {
	defer func() {
		Appended_Offset_Invariants(count, "append_nonempty_cell_unchecked.count")
		Append_Status_Invariants(status, "append_nonempty_cell_unchecked.status")
	}()
	Output_Invariants(destination, "append_nonempty_cell_unchecked.destination")
	Nonempty_Source_Invariants(source, "append_nonempty_cell_unchecked.source")
	Configuration_Invariants(configuration, "append_nonempty_cell_unchecked.configuration")
	Workspace_Pointer_Invariants(workspace, "append_nonempty_cell_unchecked.workspace")
	Cell_Index_Invariants(index, "append_nonempty_cell_unchecked.index")
	Column_Width_Invariants(column_width, "append_nonempty_cell_unchecked.column_width")
	Aligned_Cell_Invariants(aligned, "append_nonempty_cell_unchecked.aligned")
	Write_Output_Invariants(write, "append_nonempty_cell_unchecked.write")
	Append_Offset_Invariants(count_value, "append_nonempty_cell_unchecked.count_value")
	offset := Append_Offset(count_value)
	right := configuration.Flags.(Flags)&ALIGN_RIGHT !=
		Flags(bits.WORD_MINIMUM)
	if bool(aligned) {
		if right {
			offset, status = append_padding_unchecked(
				destination, configuration,
				Aligned_Cell_Width(workspace.Cell_Widths[index]),
				column_width, false, write, offset,
			)
			if status != STATUS_OK {
				return Appended_Offset(offset), status
			}
		}
	}
	count, status = append_cell_unchecked(
		destination, source, configuration,
		workspace.Cell_Starts[index],
		Nonempty_Cell_End(workspace.Cell_Ends[index]), write, offset,
	)
	offset = Append_Offset(count)
	if status != STATUS_OK {
		return count, status
	}
	if !bool(aligned) {
		return count, status
	}
	if right {
		return count, status
	}
	offset, status = append_padding_unchecked(
		destination, configuration,
		Aligned_Cell_Width(workspace.Cell_Widths[index]),
		column_width, false, write, offset,
	)
	return Appended_Offset(offset), status
}

func column_width_unchecked(
	workspace Workspace_Pointer,
	configuration Configuration,
	line_index Column_Line_Index,
	column_index Column_Index,
	line_count Column_Line_Count,
) (width Column_Width) {
	defer func() { Column_Width_Invariants(width, "column_width_unchecked.width") }()
	Workspace_Pointer_Invariants(workspace, "column_width_unchecked.workspace")
	Configuration_Invariants(configuration, "column_width_unchecked.configuration")
	Column_Line_Index_Invariants(line_index, "column_width_unchecked.line_index")
	Column_Index_Invariants(column_index, "column_width_unchecked.column_index")
	Column_Line_Count_Invariants(line_count, "column_width_unchecked.line_count")
	first := int(line_index)
	end := int(line_index) + COLUMN_LINE_COUNT_MINIMUM
	for first > COLUMN_LINE_INDEX_MINIMUM {
		if workspace.Line_Terminators[first-COLUMN_LINE_COUNT_MINIMUM] == '\f' {
			break
		}
		if int(workspace.Line_Cell_Counts[first-COLUMN_LINE_COUNT_MINIMUM]) <=
			int(column_index)+NONZERO_CELL_COUNT_MINIMUM {
			break
		}
		first--
	}
	for end < int(line_count) {
		if workspace.Line_Terminators[end-COLUMN_LINE_COUNT_MINIMUM] == '\f' {
			break
		}
		if int(workspace.Line_Cell_Counts[end]) <=
			int(column_index)+NONZERO_CELL_COUNT_MINIMUM {
			break
		}
		end++
	}
	width = Column_Width(configuration.Minimum_Width.(Minimum_Width))
	discardable := true
	for row := first; row < end; row++ {
		index := int(workspace.Line_First_Cells[row]) + int(column_index)
		candidate := int(workspace.Cell_Widths[index]) +
			int(configuration.Padding.(Padding))
		if candidate > int(width) {
			width = Column_Width(candidate)
		}
		if workspace.Cell_Widths[index] > Cell_Width(WIDTH_MINIMUM) {
			discardable = false
		}
		if bool(workspace.Cell_Hard_Tabs[index]) {
			discardable = false
		}
	}
	if discardable {
		if configuration.Flags.(Flags)&DISCARD_EMPTY_COLUMNS !=
			Flags(bits.WORD_MINIMUM) {
			return Column_Width(WIDTH_MINIMUM)
		}
	}
	return width
}

func append_padding_unchecked(
	destination Output,
	configuration Configuration,
	text_width Aligned_Cell_Width,
	cell_width Column_Width,
	use_tabs Use_Tabs,
	write Write_Output,
	count_value Append_Offset,
) (count Append_Offset, status Append_Status) {
	defer func() {
		Append_Offset_Invariants(count, "append_padding_unchecked.count")
		Append_Status_Invariants(status, "append_padding_unchecked.status")
	}()
	Output_Invariants(destination, "append_padding_unchecked.destination")
	Configuration_Invariants(configuration, "append_padding_unchecked.configuration")
	Aligned_Cell_Width_Invariants(text_width, "append_padding_unchecked.text_width")
	Column_Width_Invariants(cell_width, "append_padding_unchecked.cell_width")
	Use_Tabs_Invariants(use_tabs, "append_padding_unchecked.use_tabs")
	Write_Output_Invariants(write, "append_padding_unchecked.write")
	Append_Offset_Invariants(count_value, "append_padding_unchecked.count_value")
	count = Append_Offset(count_value)
	padding_character := Output_Byte(configuration.Pad_Character.(Pad_Character))
	padding_count := int(cell_width) - int(text_width)
	tab_padding := configuration.Pad_Character.(Pad_Character) == '\t'
	if bool(use_tabs) {
		tab_padding = true
	}
	if tab_padding {
		if configuration.Tab_Width.(Tab_Width) == WIDTH_MINIMUM {
			return count, STATUS_OK
		}
		tab_width := int(configuration.Tab_Width.(Tab_Width))
		rounded_width := (int(cell_width) + tab_width - utf8.CHARACTER_SIZE_MINIMUM) /
			tab_width * tab_width
		padding_count = (rounded_width - int(text_width) + tab_width -
			utf8.CHARACTER_SIZE_MINIMUM) / tab_width
		padding_character = '\t'
	}
	for index := WIDTH_MINIMUM; index < padding_count; index++ {
		appended, append_status := append_byte_unchecked(
			destination, count, padding_character, write,
		)
		count, status = Append_Offset(appended), append_status
		if status != STATUS_OK {
			return count, status
		}
	}
	return count, STATUS_OK
}

func append_cell_unchecked(
	destination Output,
	source Nonempty_Source,
	configuration Configuration,
	start Stored_Cell_Start,
	end Nonempty_Cell_End,
	write Write_Output,
	count_value Append_Offset,
) (count Appended_Offset, status Append_Status) {
	defer func() {
		Appended_Offset_Invariants(count, "append_cell_unchecked.count")
		Append_Status_Invariants(status, "append_cell_unchecked.status")
	}()
	Output_Invariants(destination, "append_cell_unchecked.destination")
	Nonempty_Source_Invariants(source, "append_cell_unchecked.source")
	Configuration_Invariants(configuration, "append_cell_unchecked.configuration")
	Stored_Cell_Start_Invariants(start, "append_cell_unchecked.start")
	Nonempty_Cell_End_Invariants(end, "append_cell_unchecked.end")
	Write_Output_Invariants(write, "append_cell_unchecked.write")
	Append_Offset_Invariants(count_value, "append_cell_unchecked.count_value")
	offset := Append_Offset(count_value)
	end_character := End_Character(LINE_TERMINATOR_NONE)
	flags := configuration.Flags.(Flags)
	for index := int(start); index < int(end); index++ {
		character := source[index]
		strip := false
		if end_character == End_Character(LINE_TERMINATOR_NONE) {
			if character == ESCAPE {
				end_character = End_Character(escape_end(
					Escape_Start_Byte(character),
				))
				strip = flags&STRIP_ESCAPE != Flags(bits.WORD_MINIMUM)
			} else {
				if flags&FILTER_HTML != Flags(bits.WORD_MINIMUM) {
					if character == ESCAPE_START_HTML_TAG {
						end_character = End_Character(escape_end(
							Escape_Start_Byte(character),
						))
					}
					if character == ESCAPE_START_HTML_ENTITY {
						end_character = End_Character(escape_end(
							Escape_Start_Byte(character),
						))
					}
				}
			}
		} else if Input_Byte(character) == Input_Byte(end_character) {
			if end_character == End_Character(ESCAPE) {
				if flags&STRIP_ESCAPE != Flags(bits.WORD_MINIMUM) {
					strip = true
				}
			}
			end_character = End_Character(LINE_TERMINATOR_NONE)
		}
		if strip {
			continue
		}
		var appended Appended_Offset
		appended, status = append_byte_unchecked(
			destination, offset, Output_Byte(character), write,
		)
		offset = Append_Offset(appended)
		if status != STATUS_OK {
			return appended, status
		}
	}
	return Appended_Offset(offset), STATUS_OK
}

func append_byte_unchecked(
	destination Output,
	count_value Append_Offset,
	character Output_Byte,
	write Write_Output,
) (count Appended_Offset, status Append_Status) {
	defer func() {
		Appended_Offset_Invariants(count, "append_byte_unchecked.count")
		Append_Status_Invariants(status, "append_byte_unchecked.status")
	}()
	Output_Invariants(destination, "append_byte_unchecked.destination")
	Append_Offset_Invariants(count_value, "append_byte_unchecked.count_value")
	Output_Byte_Invariants(character, "append_byte_unchecked.character")
	Write_Output_Invariants(write, "append_byte_unchecked.write")
	if count_value == OUTPUT_SIZE_MAXIMUM {
		return Appended_Offset(count_value), STATUS_RESULT_TOO_LARGE
	}
	if bool(write) {
		destination[count_value] = byte(character)
	}
	return Appended_Offset(count_value + utf8.CHARACTER_SIZE_MINIMUM), STATUS_OK
}
