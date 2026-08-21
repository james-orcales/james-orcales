// Package myers renders bounded diffs into caller-owned storage.
package myers

import (
	"local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/strings"
	"local/james-orcales/shared/unicode/utf8"
)

// TEXT_SIZE_MAXIMUM reuses repository text boundary.
const TEXT_SIZE_MAXIMUM = strings.TEXT_SIZE_MAXIMUM

// TEXT_SIZE_UNVALIDATED_MAXIMUM admits first rejected byte.
const TEXT_SIZE_UNVALIDATED_MAXIMUM = TEXT_SIZE_MAXIMUM + 1

// TEXT_SIZE_NONEMPTY_MINIMUM is first comparable text size.
const TEXT_SIZE_NONEMPTY_MINIMUM = strings.TEXT_SIZE_MINIMUM + 1

// RUNE_COUNT_MAXIMUM follows byte bound because each rune consumes at least one byte.
const RUNE_COUNT_MAXIMUM = TEXT_SIZE_MAXIMUM

// RUNE_STORAGE_COUNT_REQUIRED holds maximum decoded text.
const RUNE_STORAGE_COUNT_REQUIRED = RUNE_COUNT_MAXIMUM + 1

// LINE_COUNT_MAXIMUM includes one line per byte plus final empty line.
const LINE_COUNT_MAXIMUM = TEXT_SIZE_MAXIMUM + 1

// MATRIX_SIDE_COUNT includes the empty-prefix boundary around maximum line count.
const MATRIX_SIDE_COUNT = LINE_COUNT_MAXIMUM + 1

// MATRIX_COLUMN_COUNT_MINIMUM holds empty-destination boundary.
const MATRIX_COLUMN_COUNT_MINIMUM = 1

// MATRIX_STORAGE_COUNT_REQUIRED holds full bounded dynamic-programming matrix.
const MATRIX_STORAGE_COUNT_REQUIRED = MATRIX_SIDE_COUNT * MATRIX_SIDE_COUNT

// TEXT_PAIR_COUNT accounts for source and destination.
const TEXT_PAIR_COUNT = 2

// UTF8_EXPANSION_MAXIMUM accounts for invalid bytes decoded as replacement runes.
const UTF8_EXPANSION_MAXIMUM = utf8.CHARACTER_SIZE_THREE

// EDIT_COUNT_MAXIMUM permits one operation per rune from both texts.
const EDIT_COUNT_MAXIMUM = RUNE_COUNT_MAXIMUM * TEXT_PAIR_COUNT

// DIFF_SIZE_MAXIMUM includes encoded runes, quote escapes, and edit delimiters.
const DIFF_SIZE_MAXIMUM = TEXT_SIZE_MAXIMUM*TEXT_PAIR_COUNT*UTF8_EXPANSION_MAXIMUM +
	EDIT_COUNT_MAXIMUM + EDIT_COUNT_MAXIMUM*3

// DIFF_SIZE_UNREPRESENTABLE is first output count caller storage cannot hold.
const DIFF_SIZE_UNREPRESENTABLE = DIFF_SIZE_MAXIMUM + 1

// LINE_DIFF_SIZE_MAXIMUM includes both texts and one prefix per line.
const LINE_DIFF_SIZE_MAXIMUM = TEXT_SIZE_MAXIMUM*TEXT_PAIR_COUNT +
	LINE_COUNT_MAXIMUM*TEXT_PAIR_COUNT

// LINE_DIFF_SIZE_UNREPRESENTABLE is first line output count storage cannot hold.
const LINE_DIFF_SIZE_UNREPRESENTABLE = LINE_DIFF_SIZE_MAXIMUM + 1

// STATUS_OK reports complete output.
const STATUS_OK Status = 0

// STATUS_INPUT_INVALID reports oversized text.
const STATUS_INPUT_INVALID Status = 1

// STATUS_WORKSPACE_TOO_SMALL reports insufficient caller scratch storage.
const STATUS_WORKSPACE_TOO_SMALL Status = 2

// STATUS_OUTPUT_TOO_SMALL reports insufficient caller output storage.
const STATUS_OUTPUT_TOO_SMALL Status = 3

// Prepare_Status reports validation and workspace preparation outcome.
type Prepare_Status uint8

// Prepare_Status_Invariants lists preparation outcomes.
func Prepare_Status_Invariants(value Prepare_Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(PREPARE_STATUS_OK),
			uint8(PREPARE_STATUS_INPUT_INVALID),
			uint8(PREPARE_STATUS_WORKSPACE_TOO_SMALL),
		).
		Ensure()
}

// PREPARE_STATUS_OK reports prepared state.
const PREPARE_STATUS_OK Prepare_Status = 0

// PREPARE_STATUS_INPUT_INVALID reports rejected text.
const PREPARE_STATUS_INPUT_INVALID Prepare_Status = 1

// PREPARE_STATUS_WORKSPACE_TOO_SMALL reports rejected scratch storage.
const PREPARE_STATUS_WORKSPACE_TOO_SMALL Prepare_Status = 2

// Render_Status reports caller output exhaustion.
type Render_Status bool

// Render_Status_Invariants requires fitting and overflowing renders.
func Render_Status_Invariants(value Render_Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "Caller output is too small.").
		Ensure()
}

// EDIT_RETAIN marks text shared by both inputs.
const EDIT_RETAIN Edit_Kind = 10

// EDIT_DELETE marks text present only in source.
const EDIT_DELETE Edit_Kind = 20

// EDIT_INSERT marks text present only in destination.
const EDIT_INSERT Edit_Kind = 30

// EDIT_NONE marks writer with no open operation.
const EDIT_NONE Open_Edit_Kind = 0

// Status reports bounded operation outcome.
type Status uint8

// Status_Invariants lists every operation outcome.
func Status_Invariants(value Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value),
			uint8(STATUS_OK),
			uint8(STATUS_INPUT_INVALID),
			uint8(STATUS_WORKSPACE_TOO_SMALL),
			uint8(STATUS_OUTPUT_TOO_SMALL),
		).
		Ensure()
}

// Edit_Kind identifies rendered operation.
type Edit_Kind uint8

// Edit_Kind_Invariants lists three script operations.
func Edit_Kind_Invariants(value Edit_Kind, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(EDIT_RETAIN), uint8(EDIT_DELETE), uint8(EDIT_INSERT),
		).
		Ensure()
}

// Count is written size or first unrepresentable size.
type Count int

// Count_Invariants bounds character diff output count.
func Count_Invariants(value Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Holed_Int(
			int(value), strings.TEXT_SIZE_MINIMUM, DIFF_SIZE_UNREPRESENTABLE,
			1, 2, 3, 3,
		).
		Ensure()
}

// Line_Count is written size or first unrepresentable line output size.
type Line_Count int

// Line_Count_Invariants bounds line diff output count.
func Line_Count_Invariants(value Line_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Holed_Int(
			int(value), strings.TEXT_SIZE_MINIMUM, LINE_DIFF_SIZE_UNREPRESENTABLE,
			1, 1, 1, 1,
		).
		Ensure()
}

// Diff_Position counts bytes while character output is built.
type Diff_Position int

// Diff_Position_Invariants bounds every intermediate character position.
func Diff_Position_Invariants(value Diff_Position, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), strings.TEXT_SIZE_MINIMUM, DIFF_SIZE_UNREPRESENTABLE).
		Ensure()
}

// Line_Position counts bytes while line output is built.
type Line_Position int

// Line_Position_Invariants bounds every intermediate line position.
func Line_Position_Invariants(value Line_Position, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), strings.TEXT_SIZE_MINIMUM, LINE_DIFF_SIZE_UNREPRESENTABLE).
		Ensure()
}

// Output is caller-owned character diff storage.
type Output []byte

// Output_Invariants bounds caller-owned character output.
func Output_Invariants(value Output, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, DIFF_SIZE_MAXIMUM).
		Ensure()
}

// Line_Output is caller-owned line diff storage.
type Line_Output []byte

// Line_Output_Invariants bounds caller-owned line output.
func Line_Output_Invariants(value Line_Output, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, LINE_DIFF_SIZE_MAXIMUM).
		Ensure()
}

// Old_Text_Unvalidated is hostile source before size validation.
type Old_Text_Unvalidated string

// Old_Text_Unvalidated_Invariants admits first rejected source byte.
func Old_Text_Unvalidated_Invariants(
	value Old_Text_Unvalidated, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, TEXT_SIZE_UNVALIDATED_MAXIMUM).
		Ensure()
}

// New_Text_Unvalidated is hostile destination before size validation.
type New_Text_Unvalidated string

// New_Text_Unvalidated_Invariants admits first rejected destination byte.
func New_Text_Unvalidated_Invariants(
	value New_Text_Unvalidated, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, TEXT_SIZE_UNVALIDATED_MAXIMUM).
		Ensure()
}

// Old_Rune_Storage is caller-owned source decode storage.
type Old_Rune_Storage []rune

// Old_Rune_Storage_Invariants bounds source storage.
func Old_Rune_Storage_Invariants(value Old_Rune_Storage, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, RUNE_STORAGE_COUNT_REQUIRED).
		Ensure()
}

// New_Rune_Storage is caller-owned destination decode storage.
type New_Rune_Storage []rune

// New_Rune_Storage_Invariants bounds destination storage.
func New_Rune_Storage_Invariants(value New_Rune_Storage, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, RUNE_STORAGE_COUNT_REQUIRED).
		Ensure()
}

// Matrix_Storage is caller-owned LCS matrix.
type Matrix_Storage []int

// Matrix_Storage_Invariants bounds quadratic scratch storage.
func Matrix_Storage_Invariants(value Matrix_Storage, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, MATRIX_STORAGE_COUNT_REQUIRED).
		Ensure()
}

// Rune_Bytes owns one encoded character.
type Rune_Bytes [utf8.CHARACTER_SIZE_MAXIMUM]byte

// Rune_Bytes_Invariants fixes maximum UTF-8 character width.
func Rune_Bytes_Invariants(value Rune_Bytes, _ invariant.Namespace) {
	invariant.Always(
		len(value) == utf8.CHARACTER_SIZE_MAXIMUM,
		"Rune byte workspace holds widest UTF-8 character.",
	)
}

// Workspace owns every scratch byte, rune, and matrix cell.
type Workspace struct {
	// Old_Runes avoids source decode allocation.
	Old_Runes Old_Rune_Storage
	// New_Runes avoids destination decode allocation.
	New_Runes New_Rune_Storage
	// Matrix holds longest-common-subsequence lengths.
	Matrix Matrix_Storage
	// Rune_Bytes avoids encoded-rune temporary allocation.
	Rune_Bytes Rune_Bytes
}

// Workspace_Invariants composes caller-owned scratch storage.
func Workspace_Invariants(value Workspace, namespace invariant.Namespace) {
	Old_Rune_Storage_Invariants(value.Old_Runes, namespace)
	New_Rune_Storage_Invariants(value.New_Runes, namespace)
	Matrix_Storage_Invariants(value.Matrix, namespace)
	Rune_Bytes_Invariants(value.Rune_Bytes, namespace)
}

// Diff_Input carries character output, workspace, and hostile texts.
type Diff_Input struct {
	// Output receives rendered script.
	Output Output
	// Workspace owns all scratch state.
	Workspace *Workspace
	// Old is source text.
	Old Old_Text_Unvalidated
	// New is destination text.
	New New_Text_Unvalidated
}

// Diff_Input_Invariants composes character diff boundaries.
func Diff_Input_Invariants(value Diff_Input, namespace invariant.Namespace) {
	Output_Invariants(value.Output, namespace)
	Workspace_Invariants(*value.Workspace, namespace)
	Old_Text_Unvalidated_Invariants(value.Old, namespace)
	New_Text_Unvalidated_Invariants(value.New, namespace)
}

// Old_Count is decoded source rune count.
type Old_Count int

// Old_Count_Invariants bounds decoded source count.
func Old_Count_Invariants(value Old_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), strings.TEXT_SIZE_MINIMUM, RUNE_COUNT_MAXIMUM).
		Ensure()
}

// New_Count is decoded destination rune count.
type New_Count int

// New_Count_Invariants bounds decoded destination count.
func New_Count_Invariants(value New_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), strings.TEXT_SIZE_MINIMUM, RUNE_COUNT_MAXIMUM).
		Ensure()
}

// Column_Count includes destination empty-prefix boundary.
type Column_Count int

// Column_Count_Invariants bounds matrix row width.
func Column_Count_Invariants(value Column_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), MATRIX_COLUMN_COUNT_MINIMUM, MATRIX_SIDE_COUNT).
		Ensure()
}

// RUNE_MATRIX_SIDE_COUNT includes character empty-prefix boundary.
const RUNE_MATRIX_SIDE_COUNT = RUNE_COUNT_MAXIMUM + 1

// Rune_Column_Count is character matrix row width.
type Rune_Column_Count int

// Rune_Column_Count_Invariants bounds character matrix row width.
func Rune_Column_Count_Invariants(value Rune_Column_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), MATRIX_COLUMN_COUNT_MINIMUM, RUNE_MATRIX_SIDE_COUNT).
		Ensure()
}

// Validated_Old_Text is accepted source text.
type Validated_Old_Text string

// Validated_Old_Text_Invariants bounds accepted source text.
func Validated_Old_Text_Invariants(value Validated_Old_Text, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Validated_New_Text is accepted destination text.
type Validated_New_Text string

// Validated_New_Text_Invariants bounds accepted destination text.
func Validated_New_Text_Invariants(value Validated_New_Text, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Prepared_Matrix is validated matrix storage.
type Prepared_Matrix []int

// Prepared_Matrix_Invariants requires an empty-prefix cell.
func Prepared_Matrix_Invariants(value Prepared_Matrix, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), MATRIX_COLUMN_COUNT_MINIMUM, MATRIX_STORAGE_COUNT_REQUIRED).
		Ensure()
}

// Diff_State carries prepared character matrix dimensions.
type Diff_State struct {
	// Output retains caller destination.
	Output Output
	// Old_Runes retains decoded source storage.
	Old_Runes Old_Rune_Storage
	// New_Runes retains decoded destination storage.
	New_Runes New_Rune_Storage
	// Matrix retains validated matrix storage.
	Matrix Prepared_Matrix
	// Rune_Bytes retains encoded-character scratch.
	Rune_Bytes *Rune_Bytes
	// Old is validated source text.
	Old Validated_Old_Text
	// New is validated destination text.
	New Validated_New_Text
	// Old_Count is decoded source rune count.
	Old_Count Old_Count
	// New_Count is decoded destination rune count.
	New_Count New_Count
	// Column_Count is matrix row width.
	Column_Count Rune_Column_Count
}

// Diff_State_Invariants composes prepared character diff state.
func Diff_State_Invariants(value Diff_State, namespace invariant.Namespace) {
	Output_Invariants(value.Output, namespace)
	Old_Rune_Storage_Invariants(value.Old_Runes, namespace)
	New_Rune_Storage_Invariants(value.New_Runes, namespace)
	Prepared_Matrix_Invariants(value.Matrix, namespace)
	Rune_Bytes_Invariants(*value.Rune_Bytes, namespace)
	Validated_Old_Text_Invariants(value.Old, namespace)
	Validated_New_Text_Invariants(value.New, namespace)
	Old_Count_Invariants(value.Old_Count, namespace)
	New_Count_Invariants(value.New_Count, namespace)
	Rune_Column_Count_Invariants(value.Column_Count, namespace)
}

// Diff_Prepare_State retains hostile inputs until preparation succeeds.
type Diff_Prepare_State struct {
	// Input preserves caller bounds on every failure path.
	Input Diff_Input
	// Old_Count records decoded source progress.
	Old_Count Old_Count
	// New_Count records decoded destination progress.
	New_Count New_Count
	// Column_Count records prepared matrix width.
	Column_Count Rune_Column_Count
}

// Diff_Prepare_State_Invariants composes partial preparation state.
func Diff_Prepare_State_Invariants(
	value Diff_Prepare_State, namespace invariant.Namespace,
) {
	Diff_Input_Invariants(value.Input, namespace)
	Old_Count_Invariants(value.Old_Count, namespace)
	New_Count_Invariants(value.New_Count, namespace)
	Rune_Column_Count_Invariants(value.Column_Count, namespace)
}

// Overflow reports caller output exhaustion.
type Overflow bool

// Overflow_Invariants requires fitting and overflowing writes.
func Overflow_Invariants(value Overflow, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "Character diff output overflows.").
		Ensure()
}

// Open_Edit_Kind includes no open edit plus three operations.
type Open_Edit_Kind uint8

// Open_Edit_Kind_Invariants lists writer states.
func Open_Edit_Kind_Invariants(value Open_Edit_Kind, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value),
			uint8(EDIT_NONE),
			uint8(EDIT_RETAIN),
			uint8(EDIT_DELETE),
			uint8(EDIT_INSERT),
		).
		Ensure()
}

// Byte is one output byte.
type Byte uint8

// Byte_Invariants covers complete byte domain.
func Byte_Invariants(value Byte, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint8(uint8(value), strings.BYTE_MINIMUM, strings.BYTE_MAXIMUM).
		Ensure()
}

// Line_Prefix identifies retained, deleted, or inserted line.
type Line_Prefix uint8

// Line_Prefix_Invariants lists line operations.
func Line_Prefix_Invariants(value Line_Prefix, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(LINE_PREFIX_RETAIN),
			uint8(LINE_PREFIX_INSERT), uint8(LINE_PREFIX_DELETE),
		).
		Ensure()
}

// LINE_PREFIX_RETAIN marks a shared line.
const LINE_PREFIX_RETAIN Line_Prefix = ' '

// LINE_PREFIX_INSERT marks a destination-only line.
const LINE_PREFIX_INSERT Line_Prefix = '+'

// LINE_PREFIX_DELETE marks a source-only line.
const LINE_PREFIX_DELETE Line_Prefix = '-'

// Diff_Writer carries zero-allocation character rendering state.
type Diff_Writer struct {
	// Output receives bytes that fit.
	Output Output
	// Rune_Bytes owns one encoded character.
	Rune_Bytes *Rune_Bytes
	// Position counts required bytes.
	Position Diff_Position
	// Overflow records insufficient output.
	Overflow Overflow
	// Kind is currently open operation.
	Kind Open_Edit_Kind
}

// Diff_Writer_Invariants composes character rendering state.
func Diff_Writer_Invariants(value Diff_Writer, namespace invariant.Namespace) {
	Output_Invariants(value.Output, namespace)
	Rune_Bytes_Invariants(*value.Rune_Bytes, namespace)
	Diff_Position_Invariants(value.Position, namespace)
	Overflow_Invariants(value.Overflow, namespace)
	Open_Edit_Kind_Invariants(value.Kind, namespace)
}

func diff_writer_byte(writer *Diff_Writer, value Byte) {
	Diff_Writer_Invariants(*writer, "diff_writer_byte.writer")
	Byte_Invariants(value, "diff_writer_byte.value")
	if int(writer.Position) < len(writer.Output) {
		writer.Output[writer.Position] = byte(value)
	} else {
		writer.Overflow = true
	}
	writer.Position++
}

func diff_writer_rune(writer *Diff_Writer, character utf8.Decoded_Character) {
	Diff_Writer_Invariants(*writer, "diff_writer_rune.writer")
	utf8.Decoded_Character_Invariants(character, "diff_writer_rune.character")
	if character == '"' {
		diff_writer_byte(writer, '\\')
	}
	size := utf8.Encode_Character(writer.Rune_Bytes[:], utf8.Character(character))
	for index := 0; index < int(size); index++ {
		diff_writer_byte(writer, Byte(writer.Rune_Bytes[index]))
	}
}

func diff_writer_open(writer *Diff_Writer, kind Edit_Kind) {
	Diff_Writer_Invariants(*writer, "diff_writer_open.writer")
	Edit_Kind_Invariants(kind, "diff_writer_open.kind")
	if writer.Kind == Open_Edit_Kind(kind) {
		return
	}
	if writer.Kind != 0 {
		diff_writer_byte(writer, '"')
	}
	if kind == EDIT_RETAIN {
		diff_writer_byte(writer, ' ')
	} else if kind == EDIT_DELETE {
		diff_writer_byte(writer, '-')
	} else {
		diff_writer_byte(writer, '+')
	}
	diff_writer_byte(writer, '"')
	writer.Kind = Open_Edit_Kind(kind)
}

// Diff_Into renders minimal rune script without allocation.
func Diff_Into(input Diff_Input) (count Count, status Status) {
	defer func() {
		Count_Invariants(count, "diff_into.count")
		Status_Invariants(status, "diff_into.status")
	}()
	Diff_Input_Invariants(input, "diff_into.input")
	prepared, prepare_status := diff_prepare(input)
	if prepare_status != PREPARE_STATUS_OK {
		return 0, Status(prepare_status)
	}
	state := Diff_State{
		Output:     prepared.Input.Output,
		Old_Runes:  prepared.Input.Workspace.Old_Runes,
		New_Runes:  prepared.Input.Workspace.New_Runes,
		Matrix:     Prepared_Matrix(prepared.Input.Workspace.Matrix),
		Rune_Bytes: &prepared.Input.Workspace.Rune_Bytes,
		Old:        Validated_Old_Text(prepared.Input.Old),
		New:        Validated_New_Text(prepared.Input.New),
		Old_Count:  prepared.Old_Count, New_Count: prepared.New_Count,
		Column_Count: prepared.Column_Count,
	}
	count, render_status := diff_render(state)
	if render_status {
		return count, STATUS_OUTPUT_TOO_SMALL
	}
	return count, STATUS_OK
}

func diff_prepare(input Diff_Input) (
	state Diff_Prepare_State, status Prepare_Status,
) {
	defer func() {
		Diff_Prepare_State_Invariants(state, "diff_prepare.state")
		Prepare_Status_Invariants(status, "diff_prepare.status")
	}()
	Diff_Input_Invariants(input, "diff_prepare.input")
	state.Input = input
	state.Column_Count = Rune_Column_Count(MATRIX_COLUMN_COUNT_MINIMUM)
	if len(input.Old) > TEXT_SIZE_MAXIMUM {
		return state, PREPARE_STATUS_INPUT_INVALID
	}
	if len(input.New) > TEXT_SIZE_MAXIMUM {
		return state, PREPARE_STATUS_INPUT_INVALID
	}
	old_count := 0
	for _, character := range input.Old {
		if old_count == len(input.Workspace.Old_Runes) {
			return state, PREPARE_STATUS_WORKSPACE_TOO_SMALL
		}
		input.Workspace.Old_Runes[old_count] = character
		old_count++
	}
	new_count := 0
	for _, character := range input.New {
		if new_count == len(input.Workspace.New_Runes) {
			return state, PREPARE_STATUS_WORKSPACE_TOO_SMALL
		}
		input.Workspace.New_Runes[new_count] = character
		new_count++
	}
	column_count := new_count + 1
	matrix_count := (old_count + 1) * column_count
	if len(input.Workspace.Matrix) < matrix_count {
		return state, PREPARE_STATUS_WORKSPACE_TOO_SMALL
	}
	for old_index := old_count; old_index >= 0; old_index-- {
		input.Workspace.Matrix[old_index*column_count+new_count] = 0
	}
	for new_index := new_count; new_index >= 0; new_index-- {
		input.Workspace.Matrix[old_count*column_count+new_index] = 0
	}
	for old_index := old_count - 1; old_index >= 0; old_index-- {
		for new_index := new_count - 1; new_index >= 0; new_index-- {
			cell := old_index*column_count + new_index
			if input.Workspace.Old_Runes[old_index] ==
				input.Workspace.New_Runes[new_index] {
				input.Workspace.Matrix[cell] =
					input.Workspace.Matrix[cell+column_count+1] + 1
			} else {
				input.Workspace.Matrix[cell] = max(
					input.Workspace.Matrix[cell+column_count],
					input.Workspace.Matrix[cell+1],
				)
			}
		}
	}
	state.Old_Count = Old_Count(old_count)
	state.New_Count = New_Count(new_count)
	state.Column_Count = Rune_Column_Count(column_count)
	return state, PREPARE_STATUS_OK
}

func diff_render(state Diff_State) (count Count, status Render_Status) {
	defer func() {
		Count_Invariants(count, "diff_render.count")
		Render_Status_Invariants(status, "diff_render.status")
	}()
	Diff_State_Invariants(state, "diff_render.state")
	old_count := int(state.Old_Count)
	new_count := int(state.New_Count)
	column_count := int(state.Column_Count)
	writer := Diff_Writer{Output: state.Output, Rune_Bytes: state.Rune_Bytes}
	old_index := 0
	new_index := 0
	for old_index < old_count || new_index < new_count {
		if old_index < old_count {
			if new_index < new_count {
				if state.Old_Runes[old_index] == state.New_Runes[new_index] {
					diff_writer_open(&writer, EDIT_RETAIN)
					character := utf8.Decoded_Character(
						state.Old_Runes[old_index],
					)
					diff_writer_rune(&writer, character)
					old_index++
					new_index++
					continue
				}
			}
		}
		delete_next := new_index == new_count
		if old_index < old_count {
			if new_index < new_count {
				delete_reach :=
					state.Matrix[(old_index+1)*column_count+new_index]
				insert_reach :=
					state.Matrix[old_index*column_count+new_index+1]
				delete_next = delete_reach >= insert_reach
			}
		}
		if delete_next {
			diff_writer_open(&writer, EDIT_DELETE)
			diff_writer_rune(
				&writer, utf8.Decoded_Character(state.Old_Runes[old_index]),
			)
			old_index++
		} else {
			diff_writer_open(&writer, EDIT_INSERT)
			diff_writer_rune(
				&writer, utf8.Decoded_Character(state.New_Runes[new_index]),
			)
			new_index++
		}
	}
	if writer.Kind != EDIT_NONE {
		diff_writer_byte(&writer, '"')
	}
	if writer.Overflow {
		return DIFF_SIZE_UNREPRESENTABLE, true
	}
	return Count(writer.Position), false
}

// Line_Old_Text_Unvalidated is hostile line source before size validation.
type Line_Old_Text_Unvalidated string

// Line_Old_Text_Unvalidated_Invariants admits first rejected source byte.
func Line_Old_Text_Unvalidated_Invariants(
	value Line_Old_Text_Unvalidated, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, TEXT_SIZE_UNVALIDATED_MAXIMUM).
		Ensure()
}

// Line_New_Text_Unvalidated is hostile line destination before size validation.
type Line_New_Text_Unvalidated string

// Line_New_Text_Unvalidated_Invariants admits first rejected destination byte.
func Line_New_Text_Unvalidated_Invariants(
	value Line_New_Text_Unvalidated, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, TEXT_SIZE_UNVALIDATED_MAXIMUM).
		Ensure()
}

// Line_Diff_Input carries line output, workspace, and hostile texts.
type Line_Diff_Input struct {
	// Output receives prefixed lines.
	Output Line_Output
	// Workspace owns all scratch state.
	Workspace *Workspace
	// Old is source text.
	Old Line_Old_Text_Unvalidated
	// New is destination text.
	New Line_New_Text_Unvalidated
}

// Line_Diff_Input_Invariants composes line diff boundaries.
func Line_Diff_Input_Invariants(value Line_Diff_Input, namespace invariant.Namespace) {
	Line_Output_Invariants(value.Output, namespace)
	Workspace_Invariants(*value.Workspace, namespace)
	Line_Old_Text_Unvalidated_Invariants(value.Old, namespace)
	Line_New_Text_Unvalidated_Invariants(value.New, namespace)
}

// Old_Line_Count is source line count.
type Old_Line_Count int

// Old_Line_Count_Invariants bounds source lines.
func Old_Line_Count_Invariants(value Old_Line_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), strings.TEXT_SIZE_MINIMUM, LINE_COUNT_MAXIMUM).
		Ensure()
}

// New_Line_Count is destination line count.
type New_Line_Count int

// New_Line_Count_Invariants bounds destination lines.
func New_Line_Count_Invariants(value New_Line_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), strings.TEXT_SIZE_MINIMUM, LINE_COUNT_MAXIMUM).
		Ensure()
}

// Line_State carries prepared line matrix dimensions.
type Line_State struct {
	// Output retains caller destination.
	Output Line_Output
	// Old_Runes retains source line starts.
	Old_Runes Old_Rune_Storage
	// New_Runes retains destination line starts.
	New_Runes New_Rune_Storage
	// Matrix retains validated storage.
	Matrix Prepared_Matrix
	// Old avoids copying validated source text.
	Old Validated_Old_Text
	// New avoids copying validated destination text.
	New Validated_New_Text
	// Old_Count is source line count.
	Old_Count Old_Line_Count
	// New_Count is destination line count.
	New_Count New_Line_Count
	// Column_Count is matrix row width.
	Column_Count Column_Count
}

// Line_State_Invariants composes prepared line state.
func Line_State_Invariants(value Line_State, namespace invariant.Namespace) {
	Line_Output_Invariants(value.Output, namespace)
	Old_Rune_Storage_Invariants(value.Old_Runes, namespace)
	New_Rune_Storage_Invariants(value.New_Runes, namespace)
	Prepared_Matrix_Invariants(value.Matrix, namespace)
	Validated_Old_Text_Invariants(value.Old, namespace)
	Validated_New_Text_Invariants(value.New, namespace)
	Old_Line_Count_Invariants(value.Old_Count, namespace)
	New_Line_Count_Invariants(value.New_Count, namespace)
	Column_Count_Invariants(value.Column_Count, namespace)
}

// Line_Prepare_State retains hostile inputs until preparation succeeds.
type Line_Prepare_State struct {
	// Input preserves caller bounds on every failure path.
	Input Line_Diff_Input
	// Old_Count records source-line progress.
	Old_Count Old_Line_Count
	// New_Count records destination-line progress.
	New_Count New_Line_Count
	// Column_Count records prepared matrix width.
	Column_Count Column_Count
}

// Line_Prepare_State_Invariants composes partial preparation state.
func Line_Prepare_State_Invariants(
	value Line_Prepare_State, namespace invariant.Namespace,
) {
	Line_Diff_Input_Invariants(value.Input, namespace)
	Old_Line_Count_Invariants(value.Old_Count, namespace)
	New_Line_Count_Invariants(value.New_Count, namespace)
	Column_Count_Invariants(value.Column_Count, namespace)
}

// Old_Line_Index selects source line.
type Old_Line_Index int

// Old_Line_Index_Invariants bounds source line index.
func Old_Line_Index_Invariants(value Old_Line_Index, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), strings.TEXT_SIZE_MINIMUM, RUNE_COUNT_MAXIMUM).
		Ensure()
}

// New_Line_Index selects destination line.
type New_Line_Index int

// New_Line_Index_Invariants bounds destination line index.
func New_Line_Index_Invariants(value New_Line_Index, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), strings.TEXT_SIZE_MINIMUM, RUNE_COUNT_MAXIMUM).
		Ensure()
}

// Line_Equal reports equal line bytes.
type Line_Equal bool

// Line_Equal_Invariants requires equal and different line coverage.
func Line_Equal_Invariants(value Line_Equal, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "Compared lines are equal.").
		Ensure()
}

// Compared_Old_Runes excludes empty storage because comparison needs one line.
type Compared_Old_Runes []rune

// Compared_Old_Runes_Invariants bounds prepared source starts.
func Compared_Old_Runes_Invariants(
	value Compared_Old_Runes, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), MATRIX_COLUMN_COUNT_MINIMUM, LINE_COUNT_MAXIMUM).
		Ensure()
}

// Compared_New_Runes excludes empty storage because comparison needs one line.
type Compared_New_Runes []rune

// Compared_New_Runes_Invariants bounds prepared destination starts.
func Compared_New_Runes_Invariants(
	value Compared_New_Runes, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), MATRIX_COLUMN_COUNT_MINIMUM, LINE_COUNT_MAXIMUM).
		Ensure()
}

// Compared_Old_Text excludes empty text because it has no line to compare.
type Compared_Old_Text string

// Compared_Old_Text_Invariants bounds comparable source text.
func Compared_Old_Text_Invariants(
	value Compared_Old_Text, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), TEXT_SIZE_NONEMPTY_MINIMUM, TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Compared_New_Text excludes empty text because it has no line to compare.
type Compared_New_Text string

// Compared_New_Text_Invariants bounds comparable destination text.
func Compared_New_Text_Invariants(
	value Compared_New_Text, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), TEXT_SIZE_NONEMPTY_MINIMUM, TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Compared_Old_Count excludes zero because comparison selects a source line.
type Compared_Old_Count int

// Compared_Old_Count_Invariants bounds comparable source lines.
func Compared_Old_Count_Invariants(
	value Compared_Old_Count, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), MATRIX_COLUMN_COUNT_MINIMUM, LINE_COUNT_MAXIMUM).
		Ensure()
}

// Compared_New_Count excludes zero because comparison selects a destination line.
type Compared_New_Count int

// Compared_New_Count_Invariants bounds comparable destination lines.
func Compared_New_Count_Invariants(
	value Compared_New_Count, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), MATRIX_COLUMN_COUNT_MINIMUM, LINE_COUNT_MAXIMUM).
		Ensure()
}

// Lines_Equal_State carries only state required to compare lines.
type Lines_Equal_State struct {
	// Old_Runes permits locating source line boundaries.
	Old_Runes Compared_Old_Runes
	// New_Runes permits locating destination line boundaries.
	New_Runes Compared_New_Runes
	// Old permits comparing source bytes without copies.
	Old Compared_Old_Text
	// New permits comparing destination bytes without copies.
	New Compared_New_Text
	// Old_Count permits finding final source line end.
	Old_Count Compared_Old_Count
	// New_Count permits finding final destination line end.
	New_Count Compared_New_Count
}

// Lines_Equal_State_Invariants composes comparable line state.
func Lines_Equal_State_Invariants(value Lines_Equal_State, namespace invariant.Namespace) {
	Compared_Old_Runes_Invariants(value.Old_Runes, namespace)
	Compared_New_Runes_Invariants(value.New_Runes, namespace)
	Compared_Old_Text_Invariants(value.Old, namespace)
	Compared_New_Text_Invariants(value.New, namespace)
	Compared_Old_Count_Invariants(value.Old_Count, namespace)
	Compared_New_Count_Invariants(value.New_Count, namespace)
}

// Line_Overflow reports caller line output exhaustion.
type Line_Overflow bool

// Line_Overflow_Invariants requires fitting and overflowing line writes.
func Line_Overflow_Invariants(value Line_Overflow, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "Line diff output overflows.").
		Ensure()
}

// First_Line reports writer has emitted no line.
type First_Line bool

// First_Line_Invariants requires first and later line writes.
func First_Line_Invariants(value First_Line, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "Line writer has emitted no line.").
		Ensure()
}

// Line_Writer carries zero-allocation line rendering state.
type Line_Writer struct {
	// Output receives bytes that fit.
	Output Line_Output
	// Position counts required bytes.
	Position Line_Position
	// Overflow records insufficient output.
	Overflow Line_Overflow
	// First reports whether separator is needed.
	First First_Line
}

// Line_Writer_Invariants composes line rendering state.
func Line_Writer_Invariants(value Line_Writer, namespace invariant.Namespace) {
	Line_Output_Invariants(value.Output, namespace)
	Line_Position_Invariants(value.Position, namespace)
	Line_Overflow_Invariants(value.Overflow, namespace)
	First_Line_Invariants(value.First, namespace)
}

// Line_Text is borrowed line source.
type Line_Text string

// Line_Text_Invariants bounds borrowed line source.
func Line_Text_Invariants(value Line_Text, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Line_Start is inclusive byte boundary.
type Line_Start int

// Line_Start_Invariants bounds inclusive byte boundary.
func Line_Start_Invariants(value Line_Start, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), strings.TEXT_SIZE_MINIMUM, TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Line_End is exclusive byte boundary.
type Line_End int

// Line_End_Invariants bounds exclusive byte boundary.
func Line_End_Invariants(value Line_End, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), strings.TEXT_SIZE_MINIMUM, TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Line_Write_Input carries one prefixed line write.
type Line_Write_Input struct {
	// Writer receives line bytes.
	Writer *Line_Writer
	// Prefix identifies operation.
	Prefix Line_Prefix
	// Text contains borrowed line.
	Text Line_Text
	// Start is inclusive line boundary.
	Start Line_Start
	// End is exclusive line boundary.
	End Line_End
}

// Line_Write_Input_Invariants composes one line write.
func Line_Write_Input_Invariants(value Line_Write_Input, namespace invariant.Namespace) {
	Line_Writer_Invariants(*value.Writer, namespace)
	Line_Prefix_Invariants(value.Prefix, namespace)
	Line_Text_Invariants(value.Text, namespace)
	Line_Start_Invariants(value.Start, namespace)
	Line_End_Invariants(value.End, namespace)
}

// Completed_Line_Writer excludes incomplete result positions.
type Completed_Line_Writer struct {
	// Output retains caller destination.
	Output Line_Output
	// Position excludes incomplete one-byte line output.
	Position Line_Count
	// Overflow selects bounded sentinel result.
	Overflow Line_Overflow
	// First distinguishes empty result from emitted lines.
	First First_Line
}

// Completed_Line_Writer_Invariants composes completed writer state.
func Completed_Line_Writer_Invariants(
	value Completed_Line_Writer, namespace invariant.Namespace,
) {
	Line_Output_Invariants(value.Output, namespace)
	Line_Count_Invariants(value.Position, namespace)
	Line_Overflow_Invariants(value.Overflow, namespace)
	First_Line_Invariants(value.First, namespace)
}

func line_writer_byte(writer *Line_Writer, value Byte) {
	Line_Writer_Invariants(*writer, "line_writer_byte.writer")
	Byte_Invariants(value, "line_writer_byte.value")
	if int(writer.Position) < len(writer.Output) {
		writer.Output[writer.Position] = byte(value)
	} else {
		writer.Overflow = true
	}
	writer.Position++
}

func line_writer_line(input Line_Write_Input) {
	Line_Write_Input_Invariants(input, "line_writer_line.input")
	if input.Writer.Overflow {
		return
	}
	if !input.Writer.First {
		line_writer_byte(input.Writer, '\n')
	}
	input.Writer.First = false
	line_writer_byte(input.Writer, Byte(input.Prefix))
	for index := int(input.Start); index < int(input.End); index++ {
		line_writer_byte(input.Writer, Byte(input.Text[index]))
	}
}

func line_writer_result(writer Completed_Line_Writer) (
	count Line_Count, status Render_Status,
) {
	defer func() {
		Line_Count_Invariants(count, "line_writer_result.count")
		Render_Status_Invariants(status, "line_writer_result.status")
	}()
	Completed_Line_Writer_Invariants(writer, "line_writer_result.writer")
	if writer.Overflow {
		return LINE_DIFF_SIZE_UNREPRESENTABLE, true
	}
	return writer.Position, false
}

func lines_equal(
	state Lines_Equal_State, old_index Old_Line_Index, new_index New_Line_Index,
) (equal Line_Equal) {
	defer func() { Line_Equal_Invariants(equal, "lines_equal.equal") }()
	Lines_Equal_State_Invariants(state, "lines_equal.state")
	Old_Line_Index_Invariants(old_index, "lines_equal.old_index")
	New_Line_Index_Invariants(new_index, "lines_equal.new_index")
	old_start := int(state.Old_Runes[old_index])
	old_end_count := len(state.Old)
	if int(old_index)+1 < int(state.Old_Count) {
		old_end_count = int(state.Old_Runes[old_index+1]) - 1
	}
	new_start := int(state.New_Runes[new_index])
	new_end_count := len(state.New)
	if int(new_index)+1 < int(state.New_Count) {
		new_end_count = int(state.New_Runes[new_index+1]) - 1
	}
	if old_end_count-old_start != new_end_count-new_start {
		return false
	}
	for index := 0; index < old_end_count-old_start; index++ {
		if state.Old[old_start+index] != state.New[new_start+index] {
			return false
		}
	}
	return true
}

func old_line_end(
	runes Compared_Old_Runes, text Compared_Old_Text,
	count Compared_Old_Count, index Old_Line_Index,
) (result Line_End) {
	defer func() { Line_End_Invariants(result, "old_line_end.result") }()
	Compared_Old_Runes_Invariants(runes, "old_line_end.runes")
	Compared_Old_Text_Invariants(text, "old_line_end.text")
	Compared_Old_Count_Invariants(count, "old_line_end.count")
	Old_Line_Index_Invariants(index, "old_line_end.index")
	if int(index)+1 < int(count) {
		return Line_End(int(runes[index+1]) - 1)
	}
	return Line_End(len(text))
}

func new_line_end(
	runes Compared_New_Runes, text Compared_New_Text,
	count Compared_New_Count, index New_Line_Index,
) (result Line_End) {
	defer func() { Line_End_Invariants(result, "new_line_end.result") }()
	Compared_New_Runes_Invariants(runes, "new_line_end.runes")
	Compared_New_Text_Invariants(text, "new_line_end.text")
	Compared_New_Count_Invariants(count, "new_line_end.count")
	New_Line_Index_Invariants(index, "new_line_end.index")
	if int(index)+1 < int(count) {
		return Line_End(int(runes[index+1]) - 1)
	}
	return Line_End(len(text))
}

func line_writer_old(
	writer *Line_Writer, runes Compared_Old_Runes, text Compared_Old_Text,
	count Compared_Old_Count, index Old_Line_Index, prefix Line_Prefix,
) {
	Line_Writer_Invariants(*writer, "line_writer_old.writer")
	Compared_Old_Runes_Invariants(runes, "line_writer_old.runes")
	Compared_Old_Text_Invariants(text, "line_writer_old.text")
	Compared_Old_Count_Invariants(count, "line_writer_old.count")
	Old_Line_Index_Invariants(index, "line_writer_old.index")
	Line_Prefix_Invariants(prefix, "line_writer_old.prefix")
	line_writer_line(Line_Write_Input{
		Writer: writer, Prefix: prefix, Text: Line_Text(text),
		Start: Line_Start(runes[index]),
		End:   old_line_end(runes, text, count, index),
	})
}

func line_writer_new(
	writer *Line_Writer, runes Compared_New_Runes, text Compared_New_Text,
	count Compared_New_Count, index New_Line_Index, prefix Line_Prefix,
) {
	Line_Writer_Invariants(*writer, "line_writer_new.writer")
	Compared_New_Runes_Invariants(runes, "line_writer_new.runes")
	Compared_New_Text_Invariants(text, "line_writer_new.text")
	Compared_New_Count_Invariants(count, "line_writer_new.count")
	New_Line_Index_Invariants(index, "line_writer_new.index")
	Line_Prefix_Invariants(prefix, "line_writer_new.prefix")
	line_writer_line(Line_Write_Input{
		Writer: writer, Prefix: prefix, Text: Line_Text(text),
		Start: Line_Start(runes[index]),
		End:   new_line_end(runes, text, count, index),
	})
}

// Line_Diff_Into renders minimal line script without allocation.
func Line_Diff_Into(input Line_Diff_Input) (count Line_Count, status Status) {
	defer func() {
		Line_Count_Invariants(count, "line_diff_into.count")
		Status_Invariants(status, "line_diff_into.status")
	}()
	Line_Diff_Input_Invariants(input, "line_diff_into.input")
	prepared, prepare_status := line_prepare(input)
	if prepare_status != PREPARE_STATUS_OK {
		return 0, Status(prepare_status)
	}
	state := Line_State{
		Output:    prepared.Input.Output,
		Old_Runes: prepared.Input.Workspace.Old_Runes,
		New_Runes: prepared.Input.Workspace.New_Runes,
		Matrix:    Prepared_Matrix(prepared.Input.Workspace.Matrix),
		Old:       Validated_Old_Text(prepared.Input.Old),
		New:       Validated_New_Text(prepared.Input.New),
		Old_Count: prepared.Old_Count, New_Count: prepared.New_Count,
		Column_Count: prepared.Column_Count,
	}
	line_matrix(state)
	count, render_status := line_render(state)
	if render_status {
		return count, STATUS_OUTPUT_TOO_SMALL
	}
	return count, STATUS_OK
}

func line_prepare(input Line_Diff_Input) (
	state Line_Prepare_State, status Prepare_Status,
) {
	defer func() {
		Line_Prepare_State_Invariants(state, "line_prepare.state")
		Prepare_Status_Invariants(status, "line_prepare.status")
	}()
	Line_Diff_Input_Invariants(input, "line_prepare.input")
	state.Input = input
	state.Column_Count = MATRIX_COLUMN_COUNT_MINIMUM
	if len(input.Old) > TEXT_SIZE_MAXIMUM {
		return state, PREPARE_STATUS_INPUT_INVALID
	}
	if len(input.New) > TEXT_SIZE_MAXIMUM {
		return state, PREPARE_STATUS_INPUT_INVALID
	}
	old_line_count := 0
	if len(input.Old) > 0 {
		if len(input.Workspace.Old_Runes) == 0 {
			return state, PREPARE_STATUS_WORKSPACE_TOO_SMALL
		}
		old_line_count = 1
		input.Workspace.Old_Runes[0] = 0
		for index := 0; index < len(input.Old); index++ {
			if input.Old[index] == '\n' {
				if old_line_count == len(input.Workspace.Old_Runes) {
					return state, PREPARE_STATUS_WORKSPACE_TOO_SMALL
				}
				input.Workspace.Old_Runes[old_line_count] = rune(index + 1)
				old_line_count++
			}
		}
	}
	new_line_count := 0
	if len(input.New) > 0 {
		if len(input.Workspace.New_Runes) == 0 {
			return state, PREPARE_STATUS_WORKSPACE_TOO_SMALL
		}
		new_line_count = 1
		input.Workspace.New_Runes[0] = 0
		for index := 0; index < len(input.New); index++ {
			if input.New[index] == '\n' {
				if new_line_count == len(input.Workspace.New_Runes) {
					return state, PREPARE_STATUS_WORKSPACE_TOO_SMALL
				}
				input.Workspace.New_Runes[new_line_count] = rune(index + 1)
				new_line_count++
			}
		}
	}
	column_count := new_line_count + 1
	matrix_count := (old_line_count + 1) * column_count
	if len(input.Workspace.Matrix) < matrix_count {
		return state, PREPARE_STATUS_WORKSPACE_TOO_SMALL
	}
	state.Old_Count = Old_Line_Count(old_line_count)
	state.New_Count = New_Line_Count(new_line_count)
	state.Column_Count = Column_Count(column_count)
	return state, PREPARE_STATUS_OK
}

func line_matrix(state Line_State) {
	Line_State_Invariants(state, "line_matrix.state")
	old_line_count := int(state.Old_Count)
	new_line_count := int(state.New_Count)
	comparison := Lines_Equal_State{
		Old_Runes: Compared_Old_Runes(state.Old_Runes),
		New_Runes: Compared_New_Runes(state.New_Runes),
		Old:       Compared_Old_Text(state.Old), New: Compared_New_Text(state.New),
		Old_Count: Compared_Old_Count(state.Old_Count),
		New_Count: Compared_New_Count(state.New_Count),
	}
	column_count := int(state.Column_Count)
	for old_index := old_line_count; old_index >= 0; old_index-- {
		state.Matrix[old_index*column_count+new_line_count] = 0
	}
	for new_index := new_line_count; new_index >= 0; new_index-- {
		state.Matrix[old_line_count*column_count+new_index] = 0
	}
	for old_index := old_line_count - 1; old_index >= 0; old_index-- {
		for new_index := new_line_count - 1; new_index >= 0; new_index-- {
			cell := old_index*column_count + new_index
			if lines_equal(
				comparison, Old_Line_Index(old_index), New_Line_Index(new_index),
			) {
				state.Matrix[cell] = state.Matrix[cell+column_count+1] + 1
			} else {
				state.Matrix[cell] = max(
					state.Matrix[cell+column_count],
					state.Matrix[cell+1],
				)
			}
		}
	}
}

func line_render(state Line_State) (count Line_Count, status Render_Status) {
	defer func() {
		Line_Count_Invariants(count, "line_render.count")
		Render_Status_Invariants(status, "line_render.status")
	}()
	Line_State_Invariants(state, "line_render.state")
	column_count := int(state.Column_Count)
	old_line_count, new_line_count := int(state.Old_Count), int(state.New_Count)
	comparison := Lines_Equal_State{
		Old_Runes: Compared_Old_Runes(state.Old_Runes),
		New_Runes: Compared_New_Runes(state.New_Runes),
		Old:       Compared_Old_Text(state.Old), New: Compared_New_Text(state.New),
		Old_Count: Compared_Old_Count(state.Old_Count),
		New_Count: Compared_New_Count(state.New_Count),
	}
	writer := Line_Writer{Output: state.Output, First: true}
	old_index, new_index := 0, 0
	for old_index < old_line_count || new_index < new_line_count {
		if old_index < old_line_count {
			if new_index < new_line_count {
				if lines_equal(
					comparison,
					Old_Line_Index(old_index), New_Line_Index(new_index),
				) {
					line_writer_old(
						&writer, comparison.Old_Runes, comparison.Old,
						comparison.Old_Count,
						Old_Line_Index(old_index), ' ',
					)
					old_index++
					new_index++
					continue
				}
			}
		}
		delete_next := new_index == new_line_count
		if old_index < old_line_count {
			if new_index < new_line_count {
				delete_reach := state.Matrix[(old_index+1)*column_count+new_index]
				insert_reach := state.Matrix[old_index*column_count+new_index+1]
				delete_next = delete_reach >= insert_reach
			}
		}
		if delete_next {
			line_writer_old(
				&writer, comparison.Old_Runes, comparison.Old,
				comparison.Old_Count, Old_Line_Index(old_index), '-',
			)
			old_index++
		} else {
			line_writer_new(
				&writer, comparison.New_Runes, comparison.New,
				comparison.New_Count, New_Line_Index(new_index), '+',
			)
			new_index++
		}
	}
	return line_writer_result(Completed_Line_Writer{
		Output: writer.Output, Position: Line_Count(writer.Position),
		Overflow: writer.Overflow, First: writer.First,
	})
}

// Runes is borrowed bounded character run.
type Runes []rune

// Runes_Invariants bounds borrowed result.
func Runes_Invariants(value Runes, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, RUNE_COUNT_MAXIMUM).
		Ensure()
}

// Left_Runes is first bounded helper operand.
type Left_Runes []rune

// Left_Runes_Invariants bounds first helper operand.
func Left_Runes_Invariants(value Left_Runes, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, RUNE_COUNT_MAXIMUM).
		Ensure()
}

// Right_Runes is second bounded helper operand.
type Right_Runes []rune

// Right_Runes_Invariants bounds second helper operand.
func Right_Runes_Invariants(value Right_Runes, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, RUNE_COUNT_MAXIMUM).
		Ensure()
}

// Find_Common_Prefix_Input carries both prefix operands.
type Find_Common_Prefix_Input struct {
	// Left is first text.
	Left Left_Runes
	// Right is second text.
	Right Right_Runes
}

// Find_Common_Prefix_Input_Invariants composes prefix operands.
func Find_Common_Prefix_Input_Invariants(
	value Find_Common_Prefix_Input, namespace invariant.Namespace,
) {
	Left_Runes_Invariants(value.Left, namespace)
	Right_Runes_Invariants(value.Right, namespace)
}

// Find_Common_Prefix returns longest shared leading run without allocation.
func Find_Common_Prefix(input Find_Common_Prefix_Input) (result Runes) {
	defer func() { Runes_Invariants(result, "find_common_prefix.result") }()
	Find_Common_Prefix_Input_Invariants(input, "find_common_prefix.input")
	limit := min(len(input.Left), len(input.Right))
	for index := 0; index < limit; index++ {
		if input.Left[index] != input.Right[index] {
			return Runes(input.Left[:index])
		}
	}
	return Runes(input.Left[:limit])
}

// Find_Common_Suffix_Input carries both suffix operands.
type Find_Common_Suffix_Input struct {
	// Left is first text.
	Left Left_Runes
	// Right is second text.
	Right Right_Runes
}

// Find_Common_Suffix_Input_Invariants composes suffix operands.
func Find_Common_Suffix_Input_Invariants(
	value Find_Common_Suffix_Input, namespace invariant.Namespace,
) {
	Left_Runes_Invariants(value.Left, namespace)
	Right_Runes_Invariants(value.Right, namespace)
}

// Find_Common_Suffix returns longest shared trailing run without allocation.
func Find_Common_Suffix(input Find_Common_Suffix_Input) (result Runes) {
	defer func() { Runes_Invariants(result, "find_common_suffix.result") }()
	Find_Common_Suffix_Input_Invariants(input, "find_common_suffix.input")
	limit := min(len(input.Left), len(input.Right))
	for index := 0; index < limit; index++ {
		left_index := len(input.Left) - index - 1
		right_index := len(input.Right) - index - 1
		if input.Left[left_index] != input.Right[right_index] {
			return Runes(input.Left[len(input.Left)-index:])
		}
	}
	return Runes(input.Left[len(input.Left)-limit:])
}

// Find_Common_Run_Input carries both run operands.
type Find_Common_Run_Input struct {
	// Left is first text.
	Left Left_Runes
	// Right is second text.
	Right Right_Runes
}

// Find_Common_Run_Input_Invariants composes run operands.
func Find_Common_Run_Input_Invariants(
	value Find_Common_Run_Input, namespace invariant.Namespace,
) {
	Left_Runes_Invariants(value.Left, namespace)
	Right_Runes_Invariants(value.Right, namespace)
}

// Find_Common_Run returns longest qualifying shared run without allocation.
func Find_Common_Run(input Find_Common_Run_Input) (result Runes) {
	defer func() { Runes_Invariants(result, "find_common_run.result") }()
	Find_Common_Run_Input_Invariants(input, "find_common_run.input")
	left := Runes(input.Left)
	right := Runes(input.Right)
	if len(left) < len(right) {
		left, right = right, left
	}
	minimum := (len(left) + 1) / 2
	if len(right) < minimum {
		return nil
	}
	for run_count := len(right); run_count >= minimum; run_count-- {
		for left_index := 0; left_index <= len(left)-run_count; left_index++ {
			for right_index := 0; right_index <= len(right)-run_count; right_index++ {
				equal := true
				left_run := left[left_index:]
				right_run := right[right_index:]
				for run_index := 0; run_index < run_count; run_index++ {
					if left_run[run_index] != right_run[run_index] {
						equal = false
						break
					}
				}
				if equal {
					return left[left_index : left_index+run_count]
				}
			}
		}
	}
	return nil
}

// Boolean reports bounded rune predicate result.
type Boolean bool

// Boolean_Invariants requires both predicate outcomes.
func Boolean_Invariants(value Boolean, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "Rune predicate reports true.").
		Ensure()
}

// String_Runes is bounded searched run.
type String_Runes []rune

// String_Runes_Invariants bounds searched run.
func String_Runes_Invariants(value String_Runes, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, RUNE_COUNT_MAXIMUM).
		Ensure()
}

// Expected_Runes is bounded prefix or suffix.
type Expected_Runes []rune

// Expected_Runes_Invariants bounds expected run.
func Expected_Runes_Invariants(value Expected_Runes, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, RUNE_COUNT_MAXIMUM).
		Ensure()
}

// Runes_Have_Prefix_Input carries searched run and expected prefix.
type Runes_Have_Prefix_Input struct {
	// String is searched run.
	String String_Runes
	// Expect is expected prefix.
	Expect Expected_Runes
}

// Runes_Have_Prefix_Input_Invariants composes predicate operands.
func Runes_Have_Prefix_Input_Invariants(
	value Runes_Have_Prefix_Input, namespace invariant.Namespace,
) {
	String_Runes_Invariants(value.String, namespace)
	Expected_Runes_Invariants(value.Expect, namespace)
}

// Runes_Have_Prefix reports nonempty prefix match without allocation.
func Runes_Have_Prefix(input Runes_Have_Prefix_Input) (result Boolean) {
	defer func() { Boolean_Invariants(result, "runes_have_prefix.result") }()
	Runes_Have_Prefix_Input_Invariants(input, "runes_have_prefix.input")
	if len(input.Expect) == 0 {
		return false
	}
	if len(input.Expect) > len(input.String) {
		return false
	}
	for index := range input.Expect {
		if input.String[index] != input.Expect[index] {
			return false
		}
	}
	return true
}

// Runes_Have_Suffix_Input carries searched run and expected suffix.
type Runes_Have_Suffix_Input struct {
	// String is searched run.
	String String_Runes
	// Expect is expected suffix.
	Expect Expected_Runes
}

// Runes_Have_Suffix_Input_Invariants composes predicate operands.
func Runes_Have_Suffix_Input_Invariants(
	value Runes_Have_Suffix_Input, namespace invariant.Namespace,
) {
	String_Runes_Invariants(value.String, namespace)
	Expected_Runes_Invariants(value.Expect, namespace)
}

// Runes_Have_Suffix reports nonempty suffix match without allocation.
func Runes_Have_Suffix(input Runes_Have_Suffix_Input) (result Boolean) {
	defer func() { Boolean_Invariants(result, "runes_have_suffix.result") }()
	Runes_Have_Suffix_Input_Invariants(input, "runes_have_suffix.input")
	if len(input.Expect) == 0 {
		return false
	}
	if len(input.Expect) > len(input.String) {
		return false
	}
	start := len(input.String) - len(input.Expect)
	for index := range input.Expect {
		if input.String[start+index] != input.Expect[index] {
			return false
		}
	}
	return true
}
