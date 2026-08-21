package myers

import (
	"testing"

	"local/james-orcales/shared/strings"
	"local/james-orcales/shared/unicode/utf8"
)

// Test_Internal_Diff_Boundaries avoids quadratic public runs for writer states.
func Test_Internal_Diff_Boundaries(t *testing.T) {
	rune_bytes := Rune_Bytes{}
	for _, character := range []utf8.Decoded_Character{
		1, 2, utf8.Decoded_Character(utf8.DECODED_CHARACTER_MAXIMUM),
	} {
		writer := Diff_Writer{
			Output: make(Output, DIFF_SIZE_MAXIMUM), Rune_Bytes: &rune_bytes,
		}
		diff_writer_rune(&writer, character)
	}
	for _, position := range []Diff_Position{1, DIFF_SIZE_UNREPRESENTABLE} {
		writer := Diff_Writer{
			Output: make(Output, DIFF_SIZE_MAXIMUM), Rune_Bytes: &rune_bytes,
			Position: position,
		}
		diff_writer_rune(&writer, 1)
	}
	for _, value := range []Byte{1, 2, Byte(strings.BYTE_MAXIMUM)} {
		writer := Diff_Writer{
			Output: make(Output, DIFF_SIZE_MAXIMUM), Rune_Bytes: &rune_bytes,
		}
		diff_writer_byte(&writer, value)
	}
	for _, position := range []Diff_Position{1, 2, DIFF_SIZE_UNREPRESENTABLE} {
		writer := Diff_Writer{
			Output: make(Output, DIFF_SIZE_MAXIMUM), Rune_Bytes: &rune_bytes,
			Position: position, Kind: Open_Edit_Kind(EDIT_INSERT),
		}
		diff_writer_open(&writer, EDIT_INSERT)
	}
}

// Test_Internal_Line_Boundaries avoids quadratic public runs for line states.
func Test_Internal_Line_Boundaries(t *testing.T) {
	for _, value := range []Byte{1, 2, Byte(strings.BYTE_MAXIMUM)} {
		line_writer := Line_Writer{
			Output: make(Line_Output, LINE_DIFF_SIZE_MAXIMUM), First: true,
		}
		line_writer_byte(&line_writer, value)
	}
	for _, position := range []Line_Count{0, LINE_DIFF_SIZE_UNREPRESENTABLE} {
		writer := Completed_Line_Writer{
			Output: make(Line_Output, LINE_DIFF_SIZE_MAXIMUM), Position: position,
			First: true,
		}
		line_writer_result(writer)
	}
	writer := Line_Writer{
		Output:   make(Line_Output, LINE_DIFF_SIZE_MAXIMUM),
		Position: LINE_DIFF_SIZE_UNREPRESENTABLE, First: true,
	}
	line_writer_byte(&writer, 1)
	for _, prefix := range []Line_Prefix{' ', '-', '+'} {
		line_writer := Line_Writer{
			Output: make(Line_Output, LINE_DIFF_SIZE_MAXIMUM), First: true,
		}
		line_writer_line(Line_Write_Input{Writer: &line_writer, Prefix: prefix})
	}
	writer = Line_Writer{
		Output:   make(Line_Output, LINE_DIFF_SIZE_MAXIMUM),
		Position: LINE_DIFF_SIZE_UNREPRESENTABLE, First: true,
	}
	line_writer_line(Line_Write_Input{Writer: &writer, Prefix: '+'})
}

// Test_Internal_Line_Comparison_Boundaries avoids a quadratic maximum matrix run.
func Test_Internal_Line_Comparison_Boundaries(t *testing.T) {
	minimum_state := Line_State{
		Output:    make(Line_Output, 1),
		Old_Runes: make(Old_Rune_Storage, 1), New_Runes: make(New_Rune_Storage, 1),
		Matrix: make(Prepared_Matrix, 1), Column_Count: 1,
		Old: "a", New: "a", Old_Count: 1, New_Count: 1,
	}
	lines_equal(compared_state(minimum_state), 0, 0)
	for _, prefix := range []Line_Prefix{' ', '-', '+'} {
		old_writer := Line_Writer{
			Output: make(Line_Output, 1), Position: LINE_DIFF_SIZE_UNREPRESENTABLE,
			Overflow: true, First: true,
		}
		line_writer_old(&old_writer, []rune{0}, "a", 1, 0, prefix)
		new_writer := Line_Writer{
			Output: make(Line_Output, 1), Position: LINE_DIFF_SIZE_UNREPRESENTABLE,
			Overflow: true, First: true,
		}
		line_writer_new(&new_writer, []rune{0}, "a", 1, 0, prefix)
	}
	two_state := minimum_state
	two_state.Matrix = make(Prepared_Matrix, 2)
	two_state.Old_Runes = Old_Rune_Storage{0, 1}
	two_state.New_Runes = New_Rune_Storage{0, 1}
	two_state.Old = "\n"
	two_state.New = "\n"
	two_state.Old_Count = 2
	two_state.New_Count = 2
	lines_equal(compared_state(two_state), 1, 1)

	three_lines := Validated_Old_Text("\n\n")
	three_new_lines := Validated_New_Text(three_lines)
	three_starts := []rune{0, 1, 2}
	index_state := Line_State{
		Output:    make(Line_Output, 1),
		Old_Runes: Old_Rune_Storage(three_starts),
		New_Runes: New_Rune_Storage(three_starts),
		Matrix:    make(Prepared_Matrix, 16), Old: three_lines, New: three_new_lines,
		Old_Count: 3, New_Count: 3, Column_Count: 4,
	}
	lines_equal(compared_state(index_state), 2, 2)

	maximum_text := make([]byte, TEXT_SIZE_MAXIMUM)
	maximum_starts := make([]rune, LINE_COUNT_MAXIMUM)
	for index := range maximum_text {
		maximum_text[index] = '\n'
		maximum_starts[index] = rune(index)
	}
	maximum_starts[LINE_COUNT_MAXIMUM-1] = TEXT_SIZE_MAXIMUM
	maximum_state := Line_State{
		Output:    make(Line_Output, LINE_DIFF_SIZE_MAXIMUM),
		Old_Runes: Old_Rune_Storage(maximum_starts),
		New_Runes: New_Rune_Storage(maximum_starts),
		Matrix:    make(Prepared_Matrix, MATRIX_STORAGE_COUNT_REQUIRED),
		Old:       Validated_Old_Text(maximum_text), New: Validated_New_Text(maximum_text),
		Old_Count: LINE_COUNT_MAXIMUM, New_Count: LINE_COUNT_MAXIMUM,
		Column_Count: MATRIX_SIDE_COUNT,
	}
	lines_equal(
		compared_state(maximum_state), Old_Line_Index(RUNE_COUNT_MAXIMUM),
		New_Line_Index(RUNE_COUNT_MAXIMUM),
	)
}

func compared_state(state Line_State) (result Lines_Equal_State) {
	return Lines_Equal_State{
		Old_Runes: Compared_Old_Runes(state.Old_Runes),
		New_Runes: Compared_New_Runes(state.New_Runes),
		Old:       Compared_Old_Text(state.Old), New: Compared_New_Text(state.New),
		Old_Count: Compared_Old_Count(state.Old_Count),
		New_Count: Compared_New_Count(state.New_Count),
	}
}
