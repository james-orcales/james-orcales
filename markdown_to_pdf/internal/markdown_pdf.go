// Package markdown_to_pdf renders a Markdown document to a PDF file. It is the
// pure tier of the markdown_to_pdf binary: every dependency on the outside world
// (the source bytes, the output sink, the diagnostic sink) arrives through
// Main_Input, so the renderer reads no files and touches no process globals.
package markdown_to_pdf

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Main_Input carries the injected dependencies Main needs.
type Main_Input struct {
	// Markdown is the source document, already read into memory by the caller.
	Markdown []byte
	// Output receives the rendered PDF bytes.
	Output io.Writer
	// Stderr receives a diagnostic line when writing the output fails.
	Stderr io.Writer
}

// Main renders input.Markdown and writes the resulting PDF to input.Output,
// returning a process exit code: zero on success, non-zero when the write
// fails. It is the binary's single entry point, kept here so package main
// stays a thin, untested shell.
func Main(input *Main_Input) (status_code int) {
	document := Render(input.Markdown)
	_, write_err := input.Output.Write(document)
	if write_err != nil {
		fmt.Fprintf(input.Stderr, "markdown_to_pdf: %v\n", write_err)
		return exit_write_failure
	}
	return 0
}

// Render turns Markdown source into the bytes of a self-contained PDF file.
func Render(markdown []byte) (document []byte) {
	blocks := parse_blocks(markdown)
	state := layout_new()
	block_index := 0
	for block_index < len(blocks) {
		layout_block(state, &blocks[block_index])
		block_index++
	}
	// The in-progress builder always holds one final, possibly empty page that
	// no break has flushed yet; appending it guarantees at least one page.
	state.Pages = append(state.Pages, page_content{
		Stream: state.Stream.String(),
		Links:  state.Links,
	})
	return assemble_document(state.Pages)
}

// The header's second line is a comment of bytes above 0x7f, the conventional
// marker telling tools the file carries binary data and must not be munged as
// text by a transport that rewrites line endings.
const pdf_header = "%PDF-1.7\n%\xE2\xE3\xCF\xD3\n"

const page_width = 595.0
const page_height = 842.0
const page_margin = 56.0

const body_size = 11.0

// The largest whole point size at which a 100-column Courier line fits the A4
// text column: 100 glyphs * 0.6 * 8pt = 480pt within the 483pt margin span.
const code_size = 8.0
const heading_size_1 = 24.0
const heading_size_2 = 20.0
const heading_size_3 = 16.0
const heading_size_4 = 14.0
const heading_size_5 = 12.0
const heading_size_6 = 11.0

const line_leading_ratio = 1.3
const paragraph_gap_ratio = 0.6
const heading_gap_ratio = 0.4
const heading_rule_level_max = 2
const heading_rule_gap_ratio = 0.5
const table_row_ratio = 2.0

// Courier is monospaced at 600 units per 1000-em, so each of its glyphs
// advances exactly 0.6 of the point size.
const courier_advance_ratio = 0.6

const list_indent = 18.0
const quote_indent = 18.0
const quote_bar_width = 3.0
const quote_bar_rise_ratio = 0.8
const quote_bar_drop_ratio = 0.25
const quote_vertical_inset = 8.0

// A block quote sits on a light gray background with a darker left bar and gray
// body text.
const quote_back_fill = "0.95 0.95 0.95 rg"
const quote_bar_fill = "0.75 0.75 0.75 rg"
const quote_text_fill = "0.4 0.4 0.4 rg"
const table_padding = 4.0
const table_top_baseline_ratio = 1.26

// A quote or table ends at a drawn box edge, not at a dropped text baseline
// like a paragraph, so its trailing gap must also clear the next line's ascent —
// hence larger than paragraph_gap_ratio, or following prose laps the edge.
const box_gap_ratio = 1.8

// A GitHub table draws a soft gray cell grid and shades alternate rows.
const table_border_stroke = "0.82 0.82 0.82 RG"
const table_shade_fill = "0.97 0.97 0.97 rg"
const underline_drop = 2.0

// Code renders as white glyphs on a JetBrains Darcula gray panel (#2B2B2B);
// these are the PDF fill-color operators for the panel, the code text, and
// ordinary text.
const code_panel_fill = "0.17 0.17 0.17 rg"
const code_text_fill = "1 1 1 rg"
const normal_text_fill = "0 0 0 rg"

// A link is the browser blue, both as the text fill and the underline stroke;
// the normal stroke restores black for rules and table borders afterward.
const link_fill = "0 0 0.93 rg"
const link_stroke = "0 0 0.93 RG"
const normal_stroke = "0 0 0 RG"
const link_ascent_ratio = 0.8
const link_descent_ratio = 0.25

const code_panel_inset = 1.5
const code_inline_descent_ratio = 0.25
const code_inline_height_ratio = 1.2
const code_band_descent_ratio = 0.3

const heading_level_max = 6

const bullet_glyph = "•"
const fence_marker = "```"

const font_regular = 0
const font_bold = 1
const font_italic = 2
const font_bold_italic = 3
const font_code = 4

const block_heading = 1
const block_paragraph = 2
const block_list_item = 3
const block_code = 4
const block_quote = 5
const block_rule = 6
const block_table = 7

const exit_write_failure = 1

type text_run struct {
	Text string
	Font int
	Link string
}

type block struct {
	Kind    int
	Level   int
	Ordered bool
	Number  int
	Runs    []text_run
	Lines   []string
	Cells   [][]string
}

type word_piece struct {
	Text string
	Font int
	Link string
}

type inline_state struct {
	Runs   []text_run
	Buffer strings.Builder
	Bold   bool
	Italic bool
	Link   string
}

type link_box struct {
	Target string
	X1     float64
	Y1     float64
	X2     float64
	Y2     float64
}

type page_content struct {
	Stream string
	Links  []link_box
}

type layout struct {
	Pages  []page_content
	Stream *strings.Builder
	Links  []link_box
	Cursor float64
}

func parse_blocks(markdown []byte) (blocks []block) {
	lines := split_lines(markdown)
	index := 0
	for index < len(lines) {
		line := lines[index]
		if is_blank(line) {
			index++
			continue
		}
		if is_fence(line) {
			parsed, next := parse_code(lines, index)
			blocks = append(blocks, parsed)
			index = next
			continue
		}
		if is_rule_line(line) {
			blocks = append(blocks, block{Kind: block_rule})
			index++
			continue
		}
		if is_heading(line) {
			blocks = append(blocks, parse_heading(line))
			index++
			continue
		}
		if is_quote(line) {
			parsed, next := parse_quote(lines, index)
			blocks = append(blocks, parsed)
			index = next
			continue
		}
		if is_table_start(lines, index) {
			parsed, next := parse_table(lines, index)
			blocks = append(blocks, parsed)
			index = next
			continue
		}
		if is_list_item(line) {
			blocks = append(blocks, parse_list_item(line))
			index++
			continue
		}
		parsed, next := parse_paragraph(lines, index)
		blocks = append(blocks, parsed)
		index = next
	}
	return blocks
}

func split_lines(markdown []byte) (lines []string) {
	text := strings.ReplaceAll(string(markdown), "\r\n", "\n")
	return strings.Split(text, "\n")
}

func is_blank(line string) (blank bool) {
	return strings.TrimSpace(line) == ""
}

func is_fence(line string) (fence bool) {
	return strings.HasPrefix(strings.TrimSpace(line), fence_marker)
}

func is_heading(line string) (heading bool) {
	hash_index := 0
	for hash_index < len(line) {
		if line[hash_index] != '#' {
			break
		}
		hash_index++
	}
	if hash_index == 0 {
		return false
	}
	if hash_index > heading_level_max {
		return false
	}
	if hash_index == len(line) {
		return true
	}
	return line[hash_index] == ' '
}

func is_quote(line string) (quote bool) {
	return strings.HasPrefix(line, ">")
}

func is_rule_line(line string) (rule bool) {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) < 3 {
		return false
	}
	if is_all_byte(trimmed, '-') {
		return true
	}
	if is_all_byte(trimmed, '*') {
		return true
	}
	return is_all_byte(trimmed, '_')
}

func is_all_byte(text string, target byte) (uniform bool) {
	scan_index := 0
	for scan_index < len(text) {
		if text[scan_index] != target {
			return false
		}
		scan_index++
	}
	return len(text) > 0
}

func is_list_item(line string) (item bool) {
	if is_unordered_marker(line) {
		return true
	}
	return is_ordered_marker(line)
}

func is_unordered_marker(line string) (marker bool) {
	if strings.HasPrefix(line, "- ") {
		return true
	}
	if strings.HasPrefix(line, "* ") {
		return true
	}
	return strings.HasPrefix(line, "+ ")
}

func is_ordered_marker(line string) (marker bool) {
	digit_index := 0
	for digit_index < len(line) {
		if !is_digit(line[digit_index]) {
			break
		}
		digit_index++
	}
	if digit_index == 0 {
		return false
	}
	return strings.HasPrefix(line[digit_index:], ". ")
}

func is_digit(code byte) (digit bool) {
	if code < '0' {
		return false
	}
	return code <= '9'
}

func is_table_start(lines []string, index int) (table bool) {
	if !strings.Contains(lines[index], "|") {
		return false
	}
	if index+1 >= len(lines) {
		return false
	}
	return is_table_separator(lines[index+1])
}

func is_table_separator(line string) (separator bool) {
	if !strings.Contains(line, "|") {
		return false
	}
	if !strings.Contains(line, "-") {
		return false
	}
	return is_separator_run(line)
}

func is_separator_run(line string) (clean bool) {
	scan_index := 0
	for scan_index < len(line) {
		if !is_separator_byte(line[scan_index]) {
			return false
		}
		scan_index++
	}
	return true
}

func is_separator_byte(code byte) (allowed bool) {
	if code == '|' {
		return true
	}
	if code == '-' {
		return true
	}
	if code == ':' {
		return true
	}
	return code == ' '
}

func parse_heading(line string) (heading block) {
	level_index := 0
	for level_index < len(line) {
		if line[level_index] != '#' {
			break
		}
		level_index++
	}
	rest := strings.TrimLeft(line[level_index:], " ")
	return block{Kind: block_heading, Level: level_index, Runs: parse_inline(rest)}
}

func parse_code(lines []string, index int) (code block, next int) {
	var collected []string
	cursor := index + 1
	for cursor < len(lines) {
		if is_fence(lines[cursor]) {
			cursor++
			break
		}
		collected = append(collected, lines[cursor])
		cursor++
	}
	return block{Kind: block_code, Lines: collected}, cursor
}

func parse_quote(lines []string, index int) (quote block, next int) {
	var joined strings.Builder
	cursor := index
	for cursor < len(lines) {
		if !is_quote(lines[cursor]) {
			break
		}
		if joined.Len() > 0 {
			joined.WriteByte(' ')
		}
		joined.WriteString(strip_quote_marker(lines[cursor]))
		cursor++
	}
	return block{Kind: block_quote, Runs: parse_inline(joined.String())}, cursor
}

func strip_quote_marker(line string) (stripped string) {
	without := strings.TrimPrefix(line, ">")
	return strings.TrimPrefix(without, " ")
}

func parse_list_item(line string) (item block) {
	if is_unordered_marker(line) {
		text := strings.TrimSpace(line[2:])
		return block{Kind: block_list_item, Runs: parse_inline(text)}
	}
	number, rest := split_ordered_marker(line)
	return block{Kind: block_list_item, Ordered: true, Number: number, Runs: parse_inline(rest)}
}

func split_ordered_marker(line string) (number int, rest string) {
	digit_index := 0
	for digit_index < len(line) {
		if !is_digit(line[digit_index]) {
			break
		}
		digit_index++
	}
	number, _ = strconv.Atoi(line[:digit_index])
	rest = strings.TrimPrefix(line[digit_index:], ". ")
	return number, rest
}

func parse_paragraph(lines []string, index int) (paragraph block, next int) {
	var joined strings.Builder
	cursor := index
	for cursor < len(lines) {
		if is_paragraph_break(lines, cursor) {
			break
		}
		if joined.Len() > 0 {
			joined.WriteByte(' ')
		}
		joined.WriteString(strings.TrimSpace(lines[cursor]))
		cursor++
	}
	return block{Kind: block_paragraph, Runs: parse_inline(joined.String())}, cursor
}

func is_paragraph_break(lines []string, index int) (split bool) {
	line := lines[index]
	if is_blank(line) {
		return true
	}
	if is_heading(line) {
		return true
	}
	if is_fence(line) {
		return true
	}
	if is_quote(line) {
		return true
	}
	if is_rule_line(line) {
		return true
	}
	if is_list_item(line) {
		return true
	}
	return is_table_start(lines, index)
}

func parse_table(lines []string, index int) (table block, next int) {
	var rows [][]string
	rows = append(rows, split_table_row(lines[index]))
	// The line at index+1 is the dash separator the recognizer matched; the
	// body begins two lines down.
	cursor := index + 2
	for cursor < len(lines) {
		if !strings.Contains(lines[cursor], "|") {
			break
		}
		rows = append(rows, split_table_row(lines[cursor]))
		cursor++
	}
	return block{Kind: block_table, Cells: rows}, cursor
}

func split_table_row(line string) (cells []string) {
	trimmed := strings.TrimSpace(line)
	trimmed = strings.TrimPrefix(trimmed, "|")
	trimmed = strings.TrimSuffix(trimmed, "|")
	for _, part := range strings.Split(trimmed, "|") {
		cells = append(cells, strings.TrimSpace(part))
	}
	return cells
}

func parse_inline(text string) (runs []text_run) {
	state := &inline_state{}
	scan_index := 0
	for scan_index < len(text) {
		scan_index = inline_state_step(state, text, scan_index)
	}
	inline_state_flush(state)
	return state.Runs
}

func inline_state_step(state *inline_state, text string, index int) (next int) {
	if text[index] == '`' {
		return inline_state_code(state, text, index)
	}
	if text[index] == '*' {
		return inline_state_star(state, text, index)
	}
	if text[index] == '[' {
		consumed := inline_state_link(state, text, index)
		if consumed > index {
			return consumed
		}
	}
	state.Buffer.WriteByte(text[index])
	return index + 1
}

func inline_state_star(state *inline_state, text string, index int) (next int) {
	inline_state_flush(state)
	if inline_is_double_star(text, index) {
		state.Bold = !state.Bold
		return index + 2
	}
	state.Italic = !state.Italic
	return index + 1
}

func inline_is_double_star(text string, index int) (double bool) {
	if index+1 >= len(text) {
		return false
	}
	return text[index+1] == '*'
}

func inline_state_code(state *inline_state, text string, index int) (next int) {
	inline_state_flush(state)
	close_index := index + 1
	for close_index < len(text) {
		if text[close_index] == '`' {
			break
		}
		close_index++
	}
	if close_index >= len(text) {
		state.Buffer.WriteByte(text[index])
		return index + 1
	}
	content := text[index+1 : close_index]
	state.Runs = append(state.Runs, text_run{
		Text: content,
		Font: font_code,
		Link: state.Link,
	})
	return close_index + 1
}

// A [label](target) span emits the label as a run carrying its target; on a
// malformed span it returns index unchanged so the caller shows the bracket.
func inline_state_link(state *inline_state, text string, index int) (next int) {
	label_end_index := index + 1
	for label_end_index < len(text) {
		if text[label_end_index] == ']' {
			break
		}
		label_end_index++
	}
	if label_end_index >= len(text) {
		return index
	}
	open_index := label_end_index + 1
	if open_index >= len(text) {
		return index
	}
	if text[open_index] != '(' {
		return index
	}
	close_index := open_index + 1
	for close_index < len(text) {
		if text[close_index] == ')' {
			break
		}
		close_index++
	}
	if close_index >= len(text) {
		return index
	}
	inline_state_flush(state)
	previous_link := state.Link
	state.Link = text[open_index+1 : close_index]
	state.Buffer.WriteString(text[index+1 : label_end_index])
	inline_state_flush(state)
	state.Link = previous_link
	return close_index + 1
}

func inline_state_flush(state *inline_state) {
	if state.Buffer.Len() == 0 {
		return
	}
	state.Runs = append(state.Runs, text_run{
		Text: state.Buffer.String(),
		Font: inline_state_font(state),
		Link: state.Link,
	})
	state.Buffer.Reset()
}

func inline_state_font(state *inline_state) (font int) {
	if state.Bold {
		if state.Italic {
			return font_bold_italic
		}
		return font_bold
	}
	if state.Italic {
		return font_italic
	}
	return font_regular
}

func layout_new() (state *layout) {
	return &layout{Stream: &strings.Builder{}, Cursor: page_height - page_margin}
}

func layout_block(state *layout, current *block) {
	if current.Kind == block_heading {
		layout_heading(state, current)
		return
	}
	if current.Kind == block_paragraph {
		layout_paragraph(state, current)
		return
	}
	if current.Kind == block_list_item {
		layout_list_item(state, current)
		return
	}
	if current.Kind == block_code {
		layout_code(state, current)
		return
	}
	if current.Kind == block_quote {
		layout_quote(state, current)
		return
	}
	if current.Kind == block_rule {
		layout_rule(state)
		return
	}
	if current.Kind == block_table {
		layout_table(state, current)
	}
}

func layout_heading(state *layout, current *block) {
	var runs []text_run
	for _, run := range current.Runs {
		runs = append(runs, text_run{
			Text: run.Text,
			Font: font_bold,
			Link: run.Link,
		})
	}
	size := heading_size(current.Level)
	// A heading opens with a blank line above it, setting it apart from the
	// content it follows.
	state.Cursor -= size * line_leading_ratio
	layout_prose(state, &layout_prose_input{
		Runs:        runs,
		Size:        size,
		Line_Height: size * line_leading_ratio,
	})
	if current.Level <= heading_rule_level_max {
		// The rule sits in the gap below the last baseline, a touch under the
		// descent, like GitHub's heading border.
		rule_y := state.Cursor + size*line_leading_ratio - size*heading_rule_gap_ratio
		emit_heading_rule(state.Stream, rule_y)
	}
	state.Cursor -= size * heading_gap_ratio
}

// A heading rule is drawn like a horizontal rule: a plain stroke across the
// column beneath the top heading levels.
func emit_heading_rule(stream *strings.Builder, y float64) {
	emit_stroke(stream, &emit_stroke_input{
		X1: page_margin,
		Y1: y,
		X2: page_width - page_margin,
		Y2: y,
	})
}

func heading_size(level int) (size float64) {
	if level <= 1 {
		return heading_size_1
	}
	if level == 2 {
		return heading_size_2
	}
	if level == 3 {
		return heading_size_3
	}
	if level == 4 {
		return heading_size_4
	}
	if level == 5 {
		return heading_size_5
	}
	return heading_size_6
}

func layout_paragraph(state *layout, current *block) {
	layout_prose(state, &layout_prose_input{
		Runs:        current.Runs,
		Size:        body_size,
		Line_Height: body_size * line_leading_ratio,
	})
	state.Cursor -= body_size * paragraph_gap_ratio
}

func layout_list_item(state *layout, current *block) {
	marker := text_run{Text: list_marker(current.Ordered, current.Number), Font: font_regular}
	combined := append([]text_run{marker}, current.Runs...)
	layout_prose(state, &layout_prose_input{
		Runs:        combined,
		Size:        body_size,
		Indent:      list_indent,
		Line_Height: body_size * line_leading_ratio,
	})
}

func list_marker(ordered bool, number int) (marker string) {
	if ordered {
		return strconv.Itoa(number) + "."
	}
	return bullet_glyph
}

func layout_code(state *layout, current *block) {
	for _, code_line := range current.Lines {
		layout_need(state, code_size*line_leading_ratio)
		emit_code_band(state.Stream, state.Cursor)
		emit_glyphs(state.Stream, &emit_glyphs_input{
			Text: code_line,
			Font: font_code,
			Size: code_size,
			X:    page_margin,
			Y:    state.Cursor,
		})
		state.Cursor -= code_size * line_leading_ratio
	}
	state.Cursor -= body_size * paragraph_gap_ratio
}

func layout_quote(state *layout, current *block) {
	line_height := body_size * line_leading_ratio
	lines := wrap_pieces(&wrap_pieces_input{
		Runs:      current.Runs,
		Size:      body_size,
		Width_Max: page_width - 2*page_margin - quote_indent,
	})
	ascent := body_size * quote_bar_rise_ratio
	descent := body_size * quote_bar_drop_ratio
	box_height := 2*quote_vertical_inset + ascent + descent
	box_height += float64(len(lines)-1) * line_height
	// Keep the padded box on one page so its background is not split by a break
	// the text would cross; the panels are drawn before the text lands on them.
	layout_need(state, box_height)
	box_top := state.Cursor
	box_bottom := box_top - box_height
	emit_panel(state.Stream, &emit_panel_input{
		Fill:   quote_back_fill,
		X:      page_margin,
		Y:      box_bottom,
		Width:  page_width - 2*page_margin,
		Height: box_height,
	})
	emit_panel(state.Stream, &emit_panel_input{
		Fill:   quote_bar_fill,
		X:      page_margin,
		Y:      box_bottom,
		Width:  quote_bar_width,
		Height: box_height,
	})
	// The first baseline sits one inset plus an ascent below the box top, so the
	// glyph caps clear the padding.
	state.Cursor = box_top - quote_vertical_inset - ascent
	for _, visual_line := range lines {
		layout_line(state, &layout_line_input{
			Pieces:      visual_line,
			Size:        body_size,
			Indent:      quote_indent,
			Line_Height: line_height,
			Color:       quote_text_fill,
		})
	}
	state.Cursor = box_bottom - body_size*box_gap_ratio
}

func layout_rule(state *layout) {
	layout_need(state, body_size)
	state.Cursor -= body_size * paragraph_gap_ratio
	emit_stroke(state.Stream, &emit_stroke_input{
		X1: page_margin,
		Y1: state.Cursor,
		X2: page_width - page_margin,
		Y2: state.Cursor,
	})
	state.Cursor -= body_size * paragraph_gap_ratio
}

func layout_table(state *layout, current *block) {
	columns := table_column_count(current.Cells)
	if columns == 0 {
		return
	}
	column_width := (page_width - 2*page_margin) / float64(columns)
	row_index := 0
	for row_index < len(current.Cells) {
		layout_table_row(state, &layout_table_row_input{
			Cells:        current.Cells[row_index],
			Column_Width: column_width,
			Columns:      columns,
			Header:       row_index == 0,
			Shaded:       row_index%2 == 1,
		})
		row_index++
	}
	state.Cursor -= body_size * box_gap_ratio
}

func table_column_count(rows [][]string) (columns int) {
	for _, row := range rows {
		if len(row) > columns {
			columns = len(row)
		}
	}
	return columns
}

type layout_table_row_input struct {
	Cells        []string
	Column_Width float64
	Columns      int
	Header       bool
	Shaded       bool
}

func layout_table_row(state *layout, input *layout_table_row_input) {
	line_height := body_size * line_leading_ratio
	cell_width_max := input.Column_Width - 2*table_padding
	var cell_lines [][][]word_piece
	line_count_max := 1
	cell_index := 0
	for cell_index < input.Columns {
		runs := table_cell_runs(table_cell_text(input.Cells, cell_index), input.Header)
		lines := wrap_pieces(&wrap_pieces_input{
			Runs:      runs,
			Size:      body_size,
			Width_Max: cell_width_max,
		})
		cell_lines = append(cell_lines, lines)
		line_count := len(lines)
		if line_count > line_count_max {
			line_count_max = line_count
		}
		cell_index++
	}
	row_height := body_size*table_row_ratio + float64(line_count_max-1)*line_height
	layout_need(state, row_height)
	row_top := state.Cursor
	row_bottom := row_top - row_height
	if input.Shaded {
		emit_panel(state.Stream, &emit_panel_input{
			Fill:   table_shade_fill,
			X:      page_margin,
			Y:      row_bottom,
			Width:  page_width - 2*page_margin,
			Height: row_height,
		})
	}
	emit_table_grid(state.Stream, &emit_table_grid_input{
		Top:          row_top,
		Bottom:       row_bottom,
		Column_Width: input.Column_Width,
		Columns:      input.Columns,
	})
	top_baseline := row_top - body_size*table_top_baseline_ratio
	cell_index = 0
	for cell_index < input.Columns {
		cell_x := page_margin + float64(cell_index)*input.Column_Width + table_padding
		layout_table_cell(state, &layout_table_cell_input{
			Lines:        cell_lines[cell_index],
			X:            cell_x,
			Top_Baseline: top_baseline,
			Line_Height:  line_height,
		})
		cell_index++
	}
	state.Cursor = row_bottom
}

type layout_table_cell_input struct {
	Lines        [][]word_piece
	X            float64
	Top_Baseline float64
	Line_Height  float64
}

// Stacks a cell's wrapped lines downward from its top baseline, collecting any
// link rectangles the lines produce.
func layout_table_cell(state *layout, input *layout_table_cell_input) {
	line_index := 0
	for line_index < len(input.Lines) {
		links := emit_text_line(state.Stream, &emit_text_line_input{
			Pieces: input.Lines[line_index],
			Size:   body_size,
			X:      input.X,
			Y:      input.Top_Baseline - float64(line_index)*input.Line_Height,
		})
		state.Links = append(state.Links, links...)
		line_index++
	}
}

// Parses a cell's inline markdown; a header cell forces its prose runs bold so
// the header row reads as a header while code spans stay monospace.
func table_cell_runs(cell_text string, header bool) (runs []text_run) {
	cell_runs := parse_inline(cell_text)
	if !header {
		return cell_runs
	}
	for _, run := range cell_runs {
		font := run.Font
		if font == font_regular {
			font = font_bold
		}
		if font == font_italic {
			font = font_bold_italic
		}
		runs = append(runs, text_run{Text: run.Text, Font: font, Link: run.Link})
	}
	return runs
}

// The GitHub gray cell grid for one row: its top and bottom edges plus a
// vertical at every column boundary, then the stroke color is restored.
type emit_table_grid_input struct {
	Top          float64
	Bottom       float64
	Column_Width float64
	Columns      int
}

func emit_table_grid(stream *strings.Builder, input *emit_table_grid_input) {
	stream.WriteString(table_border_stroke)
	stream.WriteByte('\n')
	emit_stroke(stream, &emit_stroke_input{
		X1: page_margin,
		Y1: input.Top,
		X2: page_width - page_margin,
		Y2: input.Top,
	})
	emit_stroke(stream, &emit_stroke_input{
		X1: page_margin,
		Y1: input.Bottom,
		X2: page_width - page_margin,
		Y2: input.Bottom,
	})
	line_index := 0
	for line_index <= input.Columns {
		column_x := page_margin + float64(line_index)*input.Column_Width
		emit_stroke(stream, &emit_stroke_input{
			X1: column_x,
			Y1: input.Top,
			X2: column_x,
			Y2: input.Bottom,
		})
		line_index++
	}
	stream.WriteString(normal_stroke)
	stream.WriteByte('\n')
}

func table_cell_text(cells []string, index int) (text string) {
	if index >= len(cells) {
		return ""
	}
	return cells[index]
}

type layout_prose_input struct {
	Runs        []text_run
	Size        float64
	Indent      float64
	Line_Height float64
	Color       string
}

func layout_prose(state *layout, input *layout_prose_input) {
	available_width := page_width - 2*page_margin - input.Indent
	lines := wrap_pieces(&wrap_pieces_input{
		Runs:      input.Runs,
		Size:      input.Size,
		Width_Max: available_width,
	})
	for _, visual_line := range lines {
		layout_line(state, &layout_line_input{
			Pieces:      visual_line,
			Size:        input.Size,
			Indent:      input.Indent,
			Line_Height: input.Line_Height,
			Color:       input.Color,
		})
	}
}

type wrap_pieces_input struct {
	Runs      []text_run
	Size      float64
	Width_Max float64
}

func wrap_pieces(input *wrap_pieces_input) (lines [][]word_piece) {
	var pieces []word_piece
	for _, run := range input.Runs {
		// A code span stays one piece so its panel is unbroken at normal
		// widths; prose splits into words. Either is broken at character
		// boundaries below only when a single piece cannot fit the column.
		if run.Font == font_code {
			pieces = append(pieces, word_piece{
				Text: run.Text,
				Font: run.Font,
				Link: run.Link,
			})
			continue
		}
		for _, field := range strings.Fields(run.Text) {
			pieces = append(pieces, word_piece{
				Text: field,
				Font: run.Font,
				Link: run.Link,
			})
		}
	}
	var current []word_piece
	used_width := 0.0
	space_width := text_width(" ", font_regular, input.Size)
	for _, piece := range pieces {
		piece_width := text_width(piece.Text, piece.Font, input.Size)
		if piece_width > input.Width_Max {
			// Too wide for any line: flush the line in progress, then break the
			// piece at character boundaries so a code span or over-long word
			// wraps like prose instead of overflowing into the next column.
			if len(current) > 0 {
				lines = append(lines, current)
				current = nil
			}
			chunks := split_piece(&split_piece_input{
				Piece:     piece,
				Width_Max: input.Width_Max,
				Size:      input.Size,
			})
			for chunk_index := 0; chunk_index < len(chunks)-1; chunk_index++ {
				lines = append(lines, []word_piece{chunks[chunk_index]})
			}
			last_chunk := chunks[len(chunks)-1]
			current = []word_piece{last_chunk}
			used_width = text_width(last_chunk.Text, last_chunk.Font, input.Size)
			continue
		}
		if len(current) > 0 {
			projected := used_width + space_width + piece_width
			if projected > input.Width_Max {
				lines = append(lines, current)
				current = nil
				used_width = 0
			}
		}
		if len(current) > 0 {
			used_width += space_width
		}
		current = append(current, piece)
		used_width += piece_width
	}
	if len(current) > 0 {
		lines = append(lines, current)
	}
	return lines
}

type split_piece_input struct {
	Piece     word_piece
	Width_Max float64
	Size      float64
}

// Breaks one piece into the longest rune-prefixes that each fit Width_Max,
// taking at least one rune per chunk so a single over-wide glyph still
// advances. Each chunk inherits the piece's font and link, so a broken code
// span renders as a stack of panels.
func split_piece(input *split_piece_input) (chunks []word_piece) {
	runes := []rune(input.Piece.Text)
	start_index := 0
	for start_index < len(runes) {
		end_index := start_index + 1
		for end_index < len(runes) {
			candidate := string(runes[start_index : end_index+1])
			if text_width(candidate, input.Piece.Font, input.Size) > input.Width_Max {
				break
			}
			end_index++
		}
		chunks = append(chunks, word_piece{
			Text: string(runes[start_index:end_index]),
			Font: input.Piece.Font,
			Link: input.Piece.Link,
		})
		start_index = end_index
	}
	return chunks
}

type layout_line_input struct {
	Pieces      []word_piece
	Size        float64
	Indent      float64
	Line_Height float64
	Color       string
}

func layout_line(state *layout, input *layout_line_input) {
	layout_need(state, input.Line_Height)
	links := emit_text_line(state.Stream, &emit_text_line_input{
		Pieces: input.Pieces,
		Size:   input.Size,
		X:      page_margin + input.Indent,
		Y:      state.Cursor,
		Color:  input.Color,
	})
	state.Links = append(state.Links, links...)
	state.Cursor -= input.Line_Height
}

func layout_need(state *layout, height float64) {
	if state.Cursor-height >= page_margin {
		return
	}
	layout_break(state)
}

func layout_break(state *layout) {
	state.Pages = append(state.Pages, page_content{
		Stream: state.Stream.String(),
		Links:  state.Links,
	})
	state.Stream.Reset()
	state.Links = nil
	state.Cursor = page_height - page_margin
}

// Courier measures by glyph count; Helvetica weights measure off their own
// metric face via glyph_advance. These widths place the panels and underlines,
// so they must match the advances the viewer draws with, bold included.
func text_width(text string, font int, size float64) (width float64) {
	if font == font_code {
		return float64(len([]rune(text))) * size * courier_advance_ratio
	}
	advance_units := 0
	for _, glyph := range text {
		advance_units += glyph_advance(font, winansi_byte(glyph))
	}
	return float64(advance_units) * size / 1000.0
}

// Routes to the metric face for the font: the bold weights off Helvetica-Bold,
// regular and oblique off Helvetica, whose widths the oblique face shares.
func glyph_advance(font int, code byte) (advance int) {
	if font == font_bold {
		return helvetica_bold_advance(code)
	}
	if font == font_bold_italic {
		return helvetica_bold_advance(code)
	}
	return helvetica_advance(code)
}

// Adobe Helvetica metrics give a WinAnsi byte's advance in 1000-em units.
// Controls measure zero; the upper range routes to the punctuation widths the
// renderer emits, falling back to a common letter width.
func helvetica_advance(code byte) (advance int) {
	if code < 0x20 {
		return 0
	}
	if code > 0x7e {
		return helvetica_punctuation_advance(code)
	}
	if code < 0x50 {
		return helvetica_advance_low(code)
	}
	return helvetica_advance_high(code)
}

// WinAnsi punctuation the renderer emits, at real Helvetica widths so the panels
// and underlines line up; the em dash is a full em, far from the 556 fallback.
func helvetica_punctuation_advance(code byte) (advance int) {
	switch code {
	case 0x91, 0x92:
		return 222
	case 0x93, 0x94:
		return 333
	case 0x95:
		return 350
	case 0x97:
		return 1000
	}
	return 556
}

func helvetica_advance_low(code byte) (advance int) {
	switch code {
	case ' ', '!', ',', '.', '/', ':', ';', 'I':
		return 278
	case '"':
		return 355
	case '#', '$', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '?', 'L':
		return 556
	case '%':
		return 889
	case '&', 'A', 'B', 'E', 'K':
		return 667
	case '\'':
		return 191
	case '(', ')', '-':
		return 333
	case '*':
		return 389
	case '+', '<', '=', '>':
		return 584
	case '@':
		return 1015
	case 'C', 'D', 'H', 'N':
		return 722
	case 'F':
		return 611
	case 'G', 'O':
		return 778
	case 'M':
		return 833
	case 'J':
		return 500
	}
	return 556
}

func helvetica_advance_high(code byte) (advance int) {
	switch code {
	case 'P', 'S', 'V', 'X', 'Y':
		return 667
	case 'Q':
		return 778
	case 'R', 'U', 'w':
		return 722
	case 'T', 'Z':
		return 611
	case 'W':
		return 944
	case '[', '\\', ']', 'f', 't':
		return 278
	case '^':
		return 469
	case '_', 'a', 'b', 'd', 'e', 'g', 'h', 'n', 'o', 'p', 'q', 'u':
		return 556
	case '`', 'r':
		return 333
	case 'c', 'k', 's', 'v', 'x', 'y', 'z':
		return 500
	case 'i', 'j', 'l':
		return 222
	case 'm':
		return 833
	case '|':
		return 260
	case '{', '}':
		return 334
	}
	return 584
}

// Adobe Helvetica-Bold metrics, used for bold and bold-oblique text so its
// panels and underlines line up with the wider glyphs.
func helvetica_bold_advance(code byte) (advance int) {
	if code < 0x20 {
		return 0
	}
	if code > 0x7e {
		return helvetica_bold_punctuation_advance(code)
	}
	if code < 0x50 {
		return helvetica_bold_advance_low(code)
	}
	return helvetica_bold_advance_high(code)
}

func helvetica_bold_punctuation_advance(code byte) (advance int) {
	switch code {
	case 0x91, 0x92:
		return 278
	case 0x93, 0x94:
		return 500
	case 0x95:
		return 350
	case 0x97:
		return 1000
	}
	return 556
}

func helvetica_bold_advance_low(code byte) (advance int) {
	switch code {
	case ' ', ',', '.', '/', 'I':
		return 278
	case '!', '(', ')', '-', ':', ';':
		return 333
	case '"':
		return 474
	case '#', '$', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'J':
		return 556
	case '%':
		return 889
	case '&', 'A', 'B', 'C', 'D', 'H', 'K', 'N':
		return 722
	case '\'':
		return 238
	case '*':
		return 389
	case '+', '<', '=', '>':
		return 584
	case '@':
		return 975
	case 'E':
		return 667
	case 'F', 'L':
		return 611
	case 'G', 'O':
		return 778
	case 'M':
		return 833
	}
	return 611
}

func helvetica_bold_advance_high(code byte) (advance int) {
	switch code {
	case 'P', 'S', 'V', 'X', 'Y':
		return 667
	case 'Q', 'w':
		return 778
	case 'R', 'U':
		return 722
	case 'T', 'Z', 'b', 'd', 'g', 'h', 'n', 'o', 'p', 'q', 'u':
		return 611
	case 'W':
		return 944
	case '[', ']', '`', 'f', 't':
		return 333
	case '\\', 'i', 'j', 'l':
		return 278
	case '^', '~':
		return 584
	case '_', 'a', 'c', 'e', 'k', 's', 'v', 'x', 'y':
		return 556
	case 'r', '{', '}':
		return 389
	case 'm':
		return 889
	case 'z':
		return 500
	case '|':
		return 280
	}
	return 584
}

// The wrapper drops the gaps between tokens, so each piece after the first is
// shown with a leading space; the font switches per piece inside one text run.
type emit_text_line_input struct {
	Pieces []word_piece
	Size   float64
	X      float64
	Y      float64
	Color  string
}

func emit_text_line(stream *strings.Builder, input *emit_text_line_input) (links []link_box) {
	if len(input.Pieces) == 0 {
		return nil
	}
	links = emit_line_decorations(stream, input)
	stream.WriteString("BT\n")
	fmt.Fprintf(stream, "1 0 0 1 %s %s Tm\n", format_number(input.X), format_number(input.Y))
	first := true
	for _, piece := range input.Pieces {
		shown := piece.Text
		if !first {
			shown = " " + piece.Text
		}
		first = false
		emit_text_color(stream, piece.Font, piece.Link != "", input.Color)
		fmt.Fprintf(stream, "/F%d %s Tf\n", piece.Font, format_number(input.Size))
		fmt.Fprintf(stream, "(%s) Tj\n", escape_pdf_text(shown))
	}
	stream.WriteString("ET\n")
	return links
}

// Behind a line's glyphs go two kinds of path: a panel under each code piece,
// and one blue underline per run of consecutive link pieces. Each run also
// yields a link_box, the clickable rectangle the page turns into an annotation.
func emit_line_decorations(
	stream *strings.Builder, input *emit_text_line_input,
) (links []link_box) {
	cursor_x := input.X
	first := true
	run_start := input.X
	run_target := ""
	run_active := false
	for _, piece := range input.Pieces {
		space_before := 0.0
		if !first {
			space_before = text_width(" ", piece.Font, input.Size)
		}
		first = false
		text_start := cursor_x + space_before
		text_end := text_start + text_width(piece.Text, piece.Font, input.Size)
		emit_code_panel(stream, &emit_code_panel_input{
			Piece: piece,
			Start: text_start,
			End:   text_end,
			Y:     input.Y,
			Size:  input.Size,
		})
		if run_active {
			if piece.Link == "" {
				links = append(links, emit_link_run(stream, &emit_link_run_input{
					Target: run_target,
					Start:  run_start,
					End:    cursor_x,
					Y:      input.Y,
					Size:   input.Size,
				}))
				run_active = false
			}
		}
		if piece.Link != "" {
			if !run_active {
				run_start = text_start
				run_target = piece.Link
				run_active = true
			}
		}
		cursor_x = text_end
	}
	if run_active {
		links = append(links, emit_link_run(stream, &emit_link_run_input{
			Target: run_target,
			Start:  run_start,
			End:    cursor_x,
			Y:      input.Y,
			Size:   input.Size,
		}))
	}
	return links
}

// A link run's underline strokes blue, then the stroke color is restored; the
// run also yields the clickable rectangle covering its glyphs.
type emit_link_run_input struct {
	Target string
	Start  float64
	End    float64
	Y      float64
	Size   float64
}

func emit_link_run(stream *strings.Builder, input *emit_link_run_input) (box link_box) {
	stream.WriteString(link_stroke)
	stream.WriteByte('\n')
	emit_stroke(stream, &emit_stroke_input{
		X1: input.Start,
		Y1: input.Y - underline_drop,
		X2: input.End,
		Y2: input.Y - underline_drop,
	})
	stream.WriteString(normal_stroke)
	stream.WriteByte('\n')
	return link_box{
		Target: input.Target,
		X1:     input.Start,
		Y1:     input.Y - input.Size*link_descent_ratio,
		X2:     input.End,
		Y2:     input.Y + input.Size*link_ascent_ratio,
	}
}

type emit_code_panel_input struct {
	Piece word_piece
	Start float64
	End   float64
	Y     float64
	Size  float64
}

func emit_code_panel(stream *strings.Builder, input *emit_code_panel_input) {
	if input.Piece.Font != font_code {
		return
	}
	emit_panel(stream, &emit_panel_input{
		Fill:   code_panel_fill,
		X:      input.Start - code_panel_inset,
		Y:      input.Y - input.Size*code_inline_descent_ratio,
		Width:  input.End - input.Start + 2*code_panel_inset,
		Height: input.Size * code_inline_height_ratio,
	})
}

type emit_stroke_input struct {
	X1 float64
	Y1 float64
	X2 float64
	Y2 float64
}

func emit_stroke(stream *strings.Builder, input *emit_stroke_input) {
	fmt.Fprintf(stream, "%s %s m %s %s l S\n",
		format_number(input.X1), format_number(input.Y1),
		format_number(input.X2), format_number(input.Y2))
}

type emit_panel_input struct {
	Fill   string
	X      float64
	Y      float64
	Width  float64
	Height float64
}

func emit_panel(stream *strings.Builder, input *emit_panel_input) {
	stream.WriteString(input.Fill)
	stream.WriteByte('\n')
	fmt.Fprintf(stream, "%s %s %s %s re\nf\n",
		format_number(input.X), format_number(input.Y),
		format_number(input.Width), format_number(input.Height))
}

// A code line's panel spans the full text column so consecutive lines tile into
// one continuous band; the descent offset keeps each line's glyphs inside it.
func emit_code_band(stream *strings.Builder, baseline float64) {
	emit_panel(stream, &emit_panel_input{
		Fill:   code_panel_fill,
		X:      page_margin,
		Y:      baseline - code_size*code_band_descent_ratio,
		Width:  page_width - 2*page_margin,
		Height: code_size * line_leading_ratio,
	})
}

// The fill set here is what the following Tj paints with: white for code on its
// panel, blue for a link, else the line's base color or black. Every text show
// sets its own color, so a panel or link fill never bleeds onto later glyphs.
func emit_text_color(stream *strings.Builder, font int, is_link bool, base_color string) {
	if font == font_code {
		stream.WriteString(code_text_fill)
		stream.WriteByte('\n')
		return
	}
	if is_link {
		stream.WriteString(link_fill)
		stream.WriteByte('\n')
		return
	}
	if base_color == "" {
		stream.WriteString(normal_text_fill)
		stream.WriteByte('\n')
		return
	}
	stream.WriteString(base_color)
	stream.WriteByte('\n')
}

type emit_glyphs_input struct {
	Text string
	Font int
	Size float64
	X    float64
	Y    float64
}

func emit_glyphs(stream *strings.Builder, input *emit_glyphs_input) {
	if input.Text == "" {
		return
	}
	stream.WriteString("BT\n")
	fmt.Fprintf(stream, "1 0 0 1 %s %s Tm\n", format_number(input.X), format_number(input.Y))
	emit_text_color(stream, input.Font, false, "")
	fmt.Fprintf(stream, "/F%d %s Tf\n", input.Font, format_number(input.Size))
	fmt.Fprintf(stream, "(%s) Tj\n", escape_pdf_text(input.Text))
	stream.WriteString("ET\n")
}

func assemble_document(pages []page_content) (document []byte) {
	var objects []string
	objects = append(objects, "<< /Type /Catalog /Pages 2 0 R >>")
	objects = append(objects, pages_object(len(pages)))
	objects = append(objects, font_object("Helvetica"))
	objects = append(objects, font_object("Helvetica-Bold"))
	objects = append(objects, font_object("Helvetica-Oblique"))
	objects = append(objects, font_object("Helvetica-BoldOblique"))
	objects = append(objects, font_object("Courier"))
	// After the fonts come every page object, then every content stream, then
	// every link annotation, so each block's object numbers are predictable.
	annot_number := first_dynamic_object + 2*len(pages)
	page_index := 0
	for page_index < len(pages) {
		objects = append(objects, page_object(&page_object_input{
			Content:     first_dynamic_object + len(pages) + page_index,
			Annot_First: annot_number,
			Annot_Count: len(pages[page_index].Links),
		}))
		annot_number += len(pages[page_index].Links)
		page_index++
	}
	content_index := 0
	for content_index < len(pages) {
		objects = append(objects, content_object(pages[content_index].Stream))
		content_index++
	}
	link_page_index := 0
	for link_page_index < len(pages) {
		link_index := 0
		for link_index < len(pages[link_page_index].Links) {
			box := pages[link_page_index].Links[link_index]
			objects = append(objects, link_box_annotation(box))
			link_index++
		}
		link_page_index++
	}
	return serialize_pdf(objects)
}

// The first per-page object is number eight: one catalog, one page tree, and
// five fonts precede it. Page objects come first, then content streams, then
// link annotations.
const first_dynamic_object = 8

func pages_object(count int) (body string) {
	var kids strings.Builder
	page_index := 0
	for page_index < count {
		if page_index > 0 {
			kids.WriteByte(' ')
		}
		fmt.Fprintf(&kids, "%d 0 R", first_dynamic_object+page_index)
		page_index++
	}
	return fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", kids.String(), count)
}

type page_object_input struct {
	Content     int
	Annot_First int
	Annot_Count int
}

func page_object(input *page_object_input) (body string) {
	resources := "<< /Font << /F0 3 0 R /F1 4 0 R /F2 5 0 R /F3 6 0 R /F4 7 0 R >> >>"
	annots := page_annots(&page_annots_input{
		First: input.Annot_First,
		Count: input.Annot_Count,
	})
	return fmt.Sprintf(
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %s %s] "+
			"/Resources %s%s /Contents %d 0 R >>",
		format_number(page_width), format_number(page_height),
		resources, annots, input.Content)
}

type page_annots_input struct {
	First int
	Count int
}

func page_annots(input *page_annots_input) (clause string) {
	if input.Count == 0 {
		return ""
	}
	var refs strings.Builder
	annot_index := 0
	for annot_index < input.Count {
		if annot_index > 0 {
			refs.WriteByte(' ')
		}
		fmt.Fprintf(&refs, "%d 0 R", input.First+annot_index)
		annot_index++
	}
	return fmt.Sprintf(" /Annots [%s]", refs.String())
}

func link_box_annotation(link link_box) (body string) {
	return fmt.Sprintf(
		"<< /Type /Annot /Subtype /Link /Rect [%s %s %s %s] "+
			"/Border [0 0 0] /A << /S /URI /URI (%s) >> >>",
		format_number(link.X1), format_number(link.Y1),
		format_number(link.X2), format_number(link.Y2), escape_pdf_text(link.Target))
}

func content_object(stream string) (body string) {
	return fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", len(stream), stream)
}

func font_object(base_font string) (body string) {
	return fmt.Sprintf(
		"<< /Type /Font /Subtype /Type1 /BaseFont /%s "+
			"/Encoding /WinAnsiEncoding >>", base_font)
}

func serialize_pdf(objects []string) (document []byte) {
	var output strings.Builder
	output.WriteString(pdf_header)
	offsets := make([]int, len(objects))
	object_index := 0
	for object_index < len(objects) {
		offsets[object_index] = output.Len()
		fmt.Fprintf(&output, "%d 0 obj\n%s\nendobj\n",
			object_index+1, objects[object_index])
		object_index++
	}
	// The current length is the byte offset where the cross reference table
	// begins; the trailer's startxref points a reader straight to it.
	xref_size := output.Len()
	serialize_xref(&output, offsets)
	fmt.Fprintf(&output, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n",
		len(objects)+1, xref_size)
	return []byte(output.String())
}

func serialize_xref(output *strings.Builder, offsets []int) {
	fmt.Fprintf(output, "xref\n0 %d\n", len(offsets)+1)
	output.WriteString("0000000000 65535 f \n")
	entry_index := 0
	for entry_index < len(offsets) {
		fmt.Fprintf(output, "%010d 00000 n \n", offsets[entry_index])
		entry_index++
	}
}

func format_number(value float64) (text string) {
	if value == float64(int64(value)) {
		return strconv.FormatInt(int64(value), 10)
	}
	return strconv.FormatFloat(value, 'f', 2, 64)
}

// A PDF literal string escapes the three syntax bytes and writes any byte above
// 0x7f as octal, so the content stream stays plain ASCII.
func escape_pdf_text(text string) (escaped string) {
	var output strings.Builder
	for _, glyph := range text {
		code := winansi_byte(glyph)
		if code == '\\' {
			output.WriteString("\\\\")
			continue
		}
		if code == '(' {
			output.WriteString("\\(")
			continue
		}
		if code == ')' {
			output.WriteString("\\)")
			continue
		}
		if code >= 0x80 {
			fmt.Fprintf(&output, "\\%03o", code)
			continue
		}
		output.WriteByte(code)
	}
	return output.String()
}

// WinAnsi passes ASCII and Latin-1 through, and the punctuation runes the
// renderer emits fold onto their WinAnsi code points. Box-drawing and arrows,
// which WinAnsi lacks, transliterate to one ASCII byte each so monospace
// diagrams stay aligned; anything still unmapped becomes a question mark.
func winansi_byte(glyph rune) (code byte) {
	if glyph < 0x80 {
		return byte(glyph)
	}
	switch glyph {
	case 0x2022:
		return 0x95
	case 0x2014:
		return 0x97
	case 0x2013:
		return 0x96
	case 0x2019:
		return 0x92
	case 0x201C:
		return 0x93
	case 0x201D:
		return 0x94
	case 0x2500:
		return '-'
	case 0x2502:
		return '|'
	case 0x250C, 0x2510, 0x2514, 0x2518, 0x251C, 0x2524, 0x252C, 0x2534, 0x253C:
		return '+'
	case 0x2192, 0x25B6:
		return '>'
	case 0x2190, 0x25C0:
		return '<'
	case 0x2191, 0x25B2:
		return '^'
	case 0x2193, 0x25BC:
		return 'v'
	}
	if glyph < 0x100 {
		return byte(glyph)
	}
	return '?'
}
