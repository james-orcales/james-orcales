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

	"local/james-orcales/shared/fixedpoint"
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
		return EXIT_WRITE_FAILURE
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
	state.Pages = append(state.Pages, Page_Content{
		Stream: state.Stream.String(),
		Links:  state.Links,
	})
	return assemble_document(state.Pages)
}

// The header's second line is a comment of bytes above 0x7f, the conventional
// marker telling tools the file carries binary data and must not be munged as
// text by a transport that rewrites line endings.
const PDF_HEADER = "%PDF-1.7\n%\xE2\xE3\xCF\xD3\n"

// PAGE_WIDTH is A4's 210mm portrait width in points; every media box spans it.
const PAGE_WIDTH fixedpoint.Number = 595 * fixedpoint.SCALE

// PAGE_HEIGHT is A4's 297mm portrait height in points; the cursor descends from its top.
const PAGE_HEIGHT fixedpoint.Number = 842 * fixedpoint.SCALE

// PAGE_MARGIN is the whitespace frame; text lands inside it on all four sides.
const PAGE_MARGIN fixedpoint.Number = 56 * fixedpoint.SCALE

// BODY_SIZE is the point size of ordinary prose, the baseline the body ratios scale from.
const BODY_SIZE fixedpoint.Number = 11 * fixedpoint.SCALE

// The largest whole point size at which a 100-column Courier line fits the A4
// text column: 100 glyphs * 0.6 * 8pt = 480pt within the 483pt margin span.
const CODE_SIZE fixedpoint.Number = 8 * fixedpoint.SCALE

// HEADING_SIZE_1 is the point size of a top-level (#) heading, the largest on a page.
const HEADING_SIZE_1 fixedpoint.Number = 24 * fixedpoint.SCALE

// HEADING_SIZE_2 is the point size of a level-2 (##) heading.
const HEADING_SIZE_2 fixedpoint.Number = 20 * fixedpoint.SCALE

// HEADING_SIZE_3 is the point size of a level-3 (###) heading.
const HEADING_SIZE_3 fixedpoint.Number = 16 * fixedpoint.SCALE

// HEADING_SIZE_4 is the point size of a level-4 (####) heading.
const HEADING_SIZE_4 fixedpoint.Number = 14 * fixedpoint.SCALE

// HEADING_SIZE_5 is the point size of a level-5 (#####) heading.
const HEADING_SIZE_5 fixedpoint.Number = 12 * fixedpoint.SCALE

// HEADING_SIZE_6 is the point size of a level-6 (######) heading, matching body size.
const HEADING_SIZE_6 fixedpoint.Number = 11 * fixedpoint.SCALE

// LINE_LEADING_RATIO is line height as a multiple of font size; 1.3 gives legible leading.
const LINE_LEADING_RATIO fixedpoint.Ratio = 13 * fixedpoint.SCALE / 10

// PARAGRAPH_GAP_RATIO is the blank space after a paragraph, as a fraction of body size.
const PARAGRAPH_GAP_RATIO fixedpoint.Ratio = 3 * fixedpoint.SCALE / 5

// HEADING_GAP_RATIO is the space below a heading before its body, per heading size.
const HEADING_GAP_RATIO fixedpoint.Ratio = 2 * fixedpoint.SCALE / 5

// HEADING_RULE_LEVEL_MAX is the deepest heading level drawn with an underline rule.
const HEADING_RULE_LEVEL_MAX = 2

// HEADING_RULE_GAP_RATIO lifts the heading rule into the gap below its baseline, per size.
const HEADING_RULE_GAP_RATIO fixedpoint.Ratio = fixedpoint.SCALE / 2

// TABLE_ROW_RATIO is a single-line table row's height as a multiple of body size.
const TABLE_ROW_RATIO fixedpoint.Ratio = 2 * fixedpoint.SCALE

// Courier is monospaced at 600 units per 1000-em, so each of its glyphs
// advances exactly 0.6 of the point size.
const COURIER_ADVANCE_RATIO fixedpoint.Ratio = 3 * fixedpoint.SCALE / 5

// LIST_INDENT is the left offset of a list item's text column from the page margin.
const LIST_INDENT fixedpoint.Number = 18 * fixedpoint.SCALE

// QUOTE_INDENT is the left offset of quote text from the margin, clearing the bar.
const QUOTE_INDENT fixedpoint.Number = 18 * fixedpoint.SCALE

// QUOTE_BAR_WIDTH is the width of the bar drawn down a block quote's left edge.
const QUOTE_BAR_WIDTH fixedpoint.Number = 3 * fixedpoint.SCALE

// QUOTE_BAR_RISE_RATIO is the ascent above baseline sizing the quote box and first line.
const QUOTE_BAR_RISE_RATIO fixedpoint.Ratio = 4 * fixedpoint.SCALE / 5

// QUOTE_BAR_DROP_RATIO is the descent below the last baseline inside the quote box.
const QUOTE_BAR_DROP_RATIO fixedpoint.Ratio = fixedpoint.SCALE / 4

// QUOTE_VERTICAL_INSET is the top and bottom padding between the quote box and its text.
const QUOTE_VERTICAL_INSET fixedpoint.Number = 8 * fixedpoint.SCALE

// A block quote sits on a light gray background with a darker left bar and gray
// body text.
const QUOTE_BACK_FILL = "0.95 0.95 0.95 rg"

// QUOTE_BAR_FILL is the PDF fill operator for the bar down the quote's left edge.
const QUOTE_BAR_FILL = "0.75 0.75 0.75 rg"

// QUOTE_TEXT_FILL is the PDF fill operator for a quote's muted gray body text.
const QUOTE_TEXT_FILL = "0.4 0.4 0.4 rg"

// TABLE_PADDING is the inset between a table cell's border and its text, on every side.
const TABLE_PADDING fixedpoint.Number = 4 * fixedpoint.SCALE

// TABLE_TOP_BASELINE_RATIO drops the first cell baseline below the row top, per body size.
const TABLE_TOP_BASELINE_RATIO fixedpoint.Ratio = 63 * fixedpoint.SCALE / 50

// A quote or table ends at a drawn box edge, not at a dropped text baseline
// like a paragraph, so its trailing gap must also clear the next line's ascent —
// hence larger than PARAGRAPH_GAP_RATIO, or following prose laps the edge.
const BOX_GAP_RATIO fixedpoint.Ratio = 9 * fixedpoint.SCALE / 5

// A GitHub table draws a soft gray cell grid and shades alternate rows.
const TABLE_BORDER_STROKE = "0.82 0.82 0.82 RG"

// TABLE_SHADE_FILL is the PDF fill operator shading alternate table rows faint gray.
const TABLE_SHADE_FILL = "0.97 0.97 0.97 rg"

// UNDERLINE_DROP is how far below the baseline a link underline is stroked.
const UNDERLINE_DROP fixedpoint.Number = 2 * fixedpoint.SCALE

// Code renders as white glyphs on a JetBrains Darcula gray panel (#2B2B2B);
// these are the PDF fill-color operators for the panel, the code text, and
// ordinary text.
const CODE_PANEL_FILL = "0.17 0.17 0.17 rg"

// CODE_TEXT_FILL is the PDF fill operator for white code glyphs on the dark panel.
const CODE_TEXT_FILL = "1 1 1 rg"

// NORMAL_TEXT_FILL is the PDF fill operator for ordinary black body text.
const NORMAL_TEXT_FILL = "0 0 0 rg"

// A link is the browser blue, both as the text fill and the underline stroke;
// the normal stroke restores black for rules and table borders afterward.
const LINK_FILL = "0 0 0.93 rg"

// LINK_STROKE is the PDF stroke operator painting a link's blue underline.
const LINK_STROKE = "0 0 0.93 RG"

// NORMAL_STROKE is the PDF stroke operator restoring black after a colored stroke.
const NORMAL_STROKE = "0 0 0 RG"

// LINK_ASCENT_RATIO sets the clickable rectangle's top above the baseline, per size.
const LINK_ASCENT_RATIO fixedpoint.Ratio = 4 * fixedpoint.SCALE / 5

// LINK_DESCENT_RATIO sets the clickable rectangle's bottom below the baseline, per size.
const LINK_DESCENT_RATIO fixedpoint.Ratio = fixedpoint.SCALE / 4

// CODE_PANEL_INSET pads an inline code panel out past its glyphs on the left and right.
const CODE_PANEL_INSET fixedpoint.Number = 3 * fixedpoint.SCALE / 2

// CODE_INLINE_DESCENT_RATIO drops an inline code panel below the baseline, per font size.
const CODE_INLINE_DESCENT_RATIO fixedpoint.Ratio = fixedpoint.SCALE / 4

// CODE_INLINE_HEIGHT_RATIO is an inline code panel's height as a multiple of font size.
const CODE_INLINE_HEIGHT_RATIO fixedpoint.Ratio = 6 * fixedpoint.SCALE / 5

// CODE_BAND_DESCENT_RATIO drops a fenced code line's band below the baseline, per code size.
const CODE_BAND_DESCENT_RATIO fixedpoint.Ratio = 3 * fixedpoint.SCALE / 10

// HEADING_LEVEL_MAX is the deepest heading (######) parsed; more hashes read as body text.
const HEADING_LEVEL_MAX = 6

// BULLET_GLYPH is the marker rendered for an unordered list item.
const BULLET_GLYPH = "•"

// FENCE_MARKER is the triple backtick that opens and closes a fenced code block.
const FENCE_MARKER = "```"

// FONT_REGULAR indexes the Helvetica face; ids match the /F0../F4 page resources.
const FONT_REGULAR = 0

// FONT_BOLD indexes the Helvetica-Bold face.
const FONT_BOLD = 1

// FONT_ITALIC indexes the Helvetica-Oblique face.
const FONT_ITALIC = 2

// FONT_BOLD_ITALIC indexes the Helvetica-BoldOblique face.
const FONT_BOLD_ITALIC = 3

// FONT_CODE indexes the Courier face used for code spans and fenced blocks.
const FONT_CODE = 4

// BLOCK_HEADING tags a heading block; the kind routes layout_block to its handler.
const BLOCK_HEADING = 1

// BLOCK_PARAGRAPH tags a paragraph block.
const BLOCK_PARAGRAPH = 2

// BLOCK_LIST_ITEM tags a list-item block.
const BLOCK_LIST_ITEM = 3

// BLOCK_CODE tags a fenced-code block.
const BLOCK_CODE = 4

// BLOCK_QUOTE tags a block-quote block.
const BLOCK_QUOTE = 5

// BLOCK_RULE tags a horizontal-rule block.
const BLOCK_RULE = 6

// BLOCK_TABLE tags a table block.
const BLOCK_TABLE = 7

// EXIT_WRITE_FAILURE is the exit code Main returns when writing the PDF to Output fails.
const EXIT_WRITE_FAILURE = 1

// Text_Run is a span of inline text sharing one font and link target.
type Text_Run struct {
	// Text is the span's literal characters.
	Text string
	// Font is the FONT_* code the span renders in.
	Font int
	// Link is the target URL, or empty when the span is not a link.
	Link string
}

// Block is one parsed Markdown block, tagged by Kind with only the fields
// its kind uses populated.
type Block struct {
	// Kind is the BLOCK_* tag selecting how the block is laid out.
	Kind int
	// Level is the heading depth, for a heading block.
	Level int
	// Ordered marks a numbered list item, for a list-item block.
	Ordered bool
	// Number is the ordinal shown, for an ordered list item.
	Number int
	// Runs holds the block's inline text spans.
	Runs []Text_Run
	// Lines holds the raw source lines, for a code block.
	Lines []string
	// Cells holds the row-major cell text, for a table block.
	Cells [][]string
}

// Word_Piece is the smallest unit line wrapping places: a word or an
// unbroken code span, carrying its own font and link.
type Word_Piece struct {
	// Text is the piece's literal characters.
	Text string
	// Font is the FONT_* code the piece renders in.
	Font int
	// Link is the target URL, or empty when the piece is not a link.
	Link string
}

// Inline_State accumulates runs as the inline scanner walks a line,
// tracking the emphasis and link context in effect at the cursor.
type Inline_State struct {
	// Runs holds the spans closed so far.
	Runs []Text_Run
	// Buffer collects the current span until its style changes.
	Buffer strings.Builder
	// Bold reports whether a bold span is open.
	Bold bool
	// Italic reports whether an italic span is open.
	Italic bool
	// Link is the target URL of the open link span, or empty.
	Link string
}

// Link_Box is a link target paired with the page rectangle that activates it.
type Link_Box struct {
	// Target is the destination URL.
	Target string
	// X1 is the rectangle's left edge.
	X1 fixedpoint.Number
	// Y1 is the rectangle's bottom edge.
	Y1 fixedpoint.Number
	// X2 is the rectangle's right edge.
	X2 fixedpoint.Number
	// Y2 is the rectangle's top edge.
	Y2 fixedpoint.Number
}

// Page_Content is one finished page: its content stream and link rectangles.
type Page_Content struct {
	// Stream is the page's PDF content stream.
	Stream string
	// Links holds the clickable rectangles on the page.
	Links []Link_Box
}

// Layout is the running state of the layout pass: the pages closed so far
// plus the stream, links, and cursor of the page in progress.
type Layout struct {
	// Pages holds the pages already finished.
	Pages []Page_Content
	// Stream is the content stream of the page being built.
	Stream *strings.Builder
	// Links holds the link rectangles collected on the current page.
	Links []Link_Box
	// Cursor is the current vertical baseline, descending down the page.
	Cursor fixedpoint.Number
}

func parse_blocks(markdown []byte) (blocks []Block) {
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
			blocks = append(blocks, Block{Kind: BLOCK_RULE})
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
	return strings.HasPrefix(strings.TrimSpace(line), FENCE_MARKER)
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
	if hash_index > HEADING_LEVEL_MAX {
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

func parse_heading(line string) (heading Block) {
	level_index := 0
	for level_index < len(line) {
		if line[level_index] != '#' {
			break
		}
		level_index++
	}
	rest := strings.TrimLeft(line[level_index:], " ")
	return Block{Kind: BLOCK_HEADING, Level: level_index, Runs: parse_inline(rest)}
}

func parse_code(lines []string, index int) (code Block, next int) {
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
	return Block{Kind: BLOCK_CODE, Lines: collected}, cursor
}

func parse_quote(lines []string, index int) (quote Block, next int) {
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
	return Block{Kind: BLOCK_QUOTE, Runs: parse_inline(joined.String())}, cursor
}

func strip_quote_marker(line string) (stripped string) {
	without := strings.TrimPrefix(line, ">")
	return strings.TrimPrefix(without, " ")
}

func parse_list_item(line string) (item Block) {
	if is_unordered_marker(line) {
		text := strings.TrimSpace(line[2:])
		return Block{Kind: BLOCK_LIST_ITEM, Runs: parse_inline(text)}
	}
	number, rest := split_ordered_marker(line)
	return Block{Kind: BLOCK_LIST_ITEM, Ordered: true, Number: number, Runs: parse_inline(rest)}
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

func parse_paragraph(lines []string, index int) (paragraph Block, next int) {
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
	return Block{Kind: BLOCK_PARAGRAPH, Runs: parse_inline(joined.String())}, cursor
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

func parse_table(lines []string, index int) (table Block, next int) {
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
	return Block{Kind: BLOCK_TABLE, Cells: rows}, cursor
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

func parse_inline(text string) (runs []Text_Run) {
	state := &Inline_State{}
	scan_index := 0
	for scan_index < len(text) {
		scan_index = inline_state_step(state, text, scan_index)
	}
	inline_state_flush(state)
	return state.Runs
}

func inline_state_step(state *Inline_State, text string, index int) (next int) {
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

func inline_state_star(state *Inline_State, text string, index int) (next int) {
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

func inline_state_code(state *Inline_State, text string, index int) (next int) {
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
	state.Runs = append(state.Runs, Text_Run{
		Text: content,
		Font: FONT_CODE,
		Link: state.Link,
	})
	return close_index + 1
}

// A [label](target) span emits the label as a run carrying its target; on a
// malformed span it returns index unchanged so the caller shows the bracket.
func inline_state_link(state *Inline_State, text string, index int) (next int) {
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

func inline_state_flush(state *Inline_State) {
	if state.Buffer.Len() == 0 {
		return
	}
	state.Runs = append(state.Runs, Text_Run{
		Text: state.Buffer.String(),
		Font: inline_state_font(state),
		Link: state.Link,
	})
	state.Buffer.Reset()
}

func inline_state_font(state *Inline_State) (font int) {
	if state.Bold {
		if state.Italic {
			return FONT_BOLD_ITALIC
		}
		return FONT_BOLD
	}
	if state.Italic {
		return FONT_ITALIC
	}
	return FONT_REGULAR
}

func layout_new() (state *Layout) {
	return &Layout{Stream: &strings.Builder{}, Cursor: PAGE_HEIGHT - PAGE_MARGIN}
}

func layout_block(state *Layout, current *Block) {
	if current.Kind == BLOCK_HEADING {
		layout_heading(state, current)
		return
	}
	if current.Kind == BLOCK_PARAGRAPH {
		layout_paragraph(state, current)
		return
	}
	if current.Kind == BLOCK_LIST_ITEM {
		layout_list_item(state, current)
		return
	}
	if current.Kind == BLOCK_CODE {
		layout_code(state, current)
		return
	}
	if current.Kind == BLOCK_QUOTE {
		layout_quote(state, current)
		return
	}
	if current.Kind == BLOCK_RULE {
		layout_rule(state)
		return
	}
	if current.Kind == BLOCK_TABLE {
		layout_table(state, current)
	}
}

func layout_heading(state *Layout, current *Block) {
	var runs []Text_Run
	for _, run := range current.Runs {
		runs = append(runs, Text_Run{
			Text: run.Text,
			Font: FONT_BOLD,
			Link: run.Link,
		})
	}
	size := heading_size(current.Level)
	// A heading opens with a blank line above it, setting it apart from the
	// content it follows.
	state.Cursor -= fixedpoint.Apply(size, LINE_LEADING_RATIO)
	layout_prose(state, &Layout_Prose_Input{
		Runs:        runs,
		Size:        size,
		Line_Height: fixedpoint.Apply(size, LINE_LEADING_RATIO),
	})
	if current.Level <= HEADING_RULE_LEVEL_MAX {
		// The rule sits in the gap below the last baseline, a touch under the
		// descent, like GitHub's heading border.
		rule_y := state.Cursor + fixedpoint.Apply(size, LINE_LEADING_RATIO) -
			fixedpoint.Apply(size, HEADING_RULE_GAP_RATIO)
		emit_heading_rule(state.Stream, rule_y)
	}
	state.Cursor -= fixedpoint.Apply(size, HEADING_GAP_RATIO)
}

// A heading rule is drawn like a horizontal rule: a plain stroke across the
// column beneath the top heading levels.
func emit_heading_rule(stream *strings.Builder, y fixedpoint.Number) {
	emit_stroke(stream, &Emit_Stroke_Input{
		X1: PAGE_MARGIN,
		Y1: y,
		X2: PAGE_WIDTH - PAGE_MARGIN,
		Y2: y,
	})
}

func heading_size(level int) (size fixedpoint.Number) {
	if level <= 1 {
		return HEADING_SIZE_1
	}
	if level == 2 {
		return HEADING_SIZE_2
	}
	if level == 3 {
		return HEADING_SIZE_3
	}
	if level == 4 {
		return HEADING_SIZE_4
	}
	if level == 5 {
		return HEADING_SIZE_5
	}
	return HEADING_SIZE_6
}

func layout_paragraph(state *Layout, current *Block) {
	layout_prose(state, &Layout_Prose_Input{
		Runs:        current.Runs,
		Size:        BODY_SIZE,
		Line_Height: fixedpoint.Apply(BODY_SIZE, LINE_LEADING_RATIO),
	})
	state.Cursor -= fixedpoint.Apply(BODY_SIZE, PARAGRAPH_GAP_RATIO)
}

func layout_list_item(state *Layout, current *Block) {
	marker := Text_Run{Text: list_marker(current.Ordered, current.Number), Font: FONT_REGULAR}
	combined := append([]Text_Run{marker}, current.Runs...)
	layout_prose(state, &Layout_Prose_Input{
		Runs:        combined,
		Size:        BODY_SIZE,
		Indent:      LIST_INDENT,
		Line_Height: fixedpoint.Apply(BODY_SIZE, LINE_LEADING_RATIO),
	})
}

func list_marker(ordered bool, number int) (marker string) {
	if ordered {
		return strconv.Itoa(number) + "."
	}
	return BULLET_GLYPH
}

func layout_code(state *Layout, current *Block) {
	for _, code_line := range current.Lines {
		layout_need(state, fixedpoint.Apply(CODE_SIZE, LINE_LEADING_RATIO))
		emit_code_band(state.Stream, state.Cursor)
		emit_glyphs(state.Stream, &Emit_Glyphs_Input{
			Text: code_line,
			Font: FONT_CODE,
			Size: CODE_SIZE,
			X:    PAGE_MARGIN,
			Y:    state.Cursor,
		})
		state.Cursor -= fixedpoint.Apply(CODE_SIZE, LINE_LEADING_RATIO)
	}
	state.Cursor -= fixedpoint.Apply(BODY_SIZE, PARAGRAPH_GAP_RATIO)
}

func layout_quote(state *Layout, current *Block) {
	line_height := fixedpoint.Apply(BODY_SIZE, LINE_LEADING_RATIO)
	lines := wrap_pieces(&Wrap_Pieces_Input{
		Runs:      current.Runs,
		Size:      BODY_SIZE,
		Width_Max: PAGE_WIDTH - 2*PAGE_MARGIN - QUOTE_INDENT,
	})
	ascent := fixedpoint.Apply(BODY_SIZE, QUOTE_BAR_RISE_RATIO)
	descent := fixedpoint.Apply(BODY_SIZE, QUOTE_BAR_DROP_RATIO)
	box_height := 2*QUOTE_VERTICAL_INSET + ascent + descent
	box_height += fixedpoint.Number(len(lines)-1) * line_height
	// Keep the padded box on one page so its background is not split by a break
	// the text would cross; the panels are drawn before the text lands on them.
	layout_need(state, box_height)
	box_top := state.Cursor
	box_bottom := box_top - box_height
	emit_panel(state.Stream, &Emit_Panel_Input{
		Fill:   QUOTE_BACK_FILL,
		X:      PAGE_MARGIN,
		Y:      box_bottom,
		Width:  PAGE_WIDTH - 2*PAGE_MARGIN,
		Height: box_height,
	})
	emit_panel(state.Stream, &Emit_Panel_Input{
		Fill:   QUOTE_BAR_FILL,
		X:      PAGE_MARGIN,
		Y:      box_bottom,
		Width:  QUOTE_BAR_WIDTH,
		Height: box_height,
	})
	// The first baseline sits one inset plus an ascent below the box top, so the
	// glyph caps clear the padding.
	state.Cursor = box_top - QUOTE_VERTICAL_INSET - ascent
	for _, visual_line := range lines {
		layout_line(state, &Layout_Line_Input{
			Pieces:      visual_line,
			Size:        BODY_SIZE,
			Indent:      QUOTE_INDENT,
			Line_Height: line_height,
			Color:       QUOTE_TEXT_FILL,
		})
	}
	state.Cursor = box_bottom - fixedpoint.Apply(BODY_SIZE, BOX_GAP_RATIO)
}

func layout_rule(state *Layout) {
	layout_need(state, BODY_SIZE)
	state.Cursor -= fixedpoint.Apply(BODY_SIZE, PARAGRAPH_GAP_RATIO)
	emit_stroke(state.Stream, &Emit_Stroke_Input{
		X1: PAGE_MARGIN,
		Y1: state.Cursor,
		X2: PAGE_WIDTH - PAGE_MARGIN,
		Y2: state.Cursor,
	})
	state.Cursor -= fixedpoint.Apply(BODY_SIZE, PARAGRAPH_GAP_RATIO)
}

func layout_table(state *Layout, current *Block) {
	columns := table_column_count(current.Cells)
	if columns == 0 {
		return
	}
	column_width := (PAGE_WIDTH - 2*PAGE_MARGIN) / fixedpoint.Number(columns)
	row_index := 0
	for row_index < len(current.Cells) {
		layout_table_row(state, &Layout_Table_Row_Input{
			Cells:        current.Cells[row_index],
			Column_Width: column_width,
			Columns:      columns,
			Header:       row_index == 0,
			Shaded:       row_index%2 == 1,
		})
		row_index++
	}
	state.Cursor -= fixedpoint.Apply(BODY_SIZE, BOX_GAP_RATIO)
}

func table_column_count(rows [][]string) (columns int) {
	for _, row := range rows {
		if len(row) > columns {
			columns = len(row)
		}
	}
	return columns
}

// Layout_Table_Row_Input carries one table row plus the geometry that places it.
type Layout_Table_Row_Input struct {
	// Cells is the row's cell text.
	Cells []string
	// Column_Width is the width allotted to each column.
	Column_Width fixedpoint.Number
	// Columns is the number of columns in the table.
	Columns int
	// Header marks the row as the bold header row.
	Header bool
	// Shaded marks the row for an alternating shaded background.
	Shaded bool
}

func layout_table_row(state *Layout, input *Layout_Table_Row_Input) {
	line_height := fixedpoint.Apply(BODY_SIZE, LINE_LEADING_RATIO)
	cell_width_max := input.Column_Width - 2*TABLE_PADDING
	var cell_lines [][][]Word_Piece
	line_count_max := 1
	cell_index := 0
	for cell_index < input.Columns {
		runs := table_cell_runs(table_cell_text(input.Cells, cell_index), input.Header)
		lines := wrap_pieces(&Wrap_Pieces_Input{
			Runs:      runs,
			Size:      BODY_SIZE,
			Width_Max: cell_width_max,
		})
		cell_lines = append(cell_lines, lines)
		line_count := len(lines)
		if line_count > line_count_max {
			line_count_max = line_count
		}
		cell_index++
	}
	row_height := fixedpoint.Apply(BODY_SIZE, TABLE_ROW_RATIO) +
		fixedpoint.Number(line_count_max-1)*line_height
	layout_need(state, row_height)
	row_top := state.Cursor
	row_bottom := row_top - row_height
	if input.Shaded {
		emit_panel(state.Stream, &Emit_Panel_Input{
			Fill:   TABLE_SHADE_FILL,
			X:      PAGE_MARGIN,
			Y:      row_bottom,
			Width:  PAGE_WIDTH - 2*PAGE_MARGIN,
			Height: row_height,
		})
	}
	emit_table_grid(state.Stream, &Emit_Table_Grid_Input{
		Top:          row_top,
		Bottom:       row_bottom,
		Column_Width: input.Column_Width,
		Columns:      input.Columns,
	})
	top_baseline := row_top - fixedpoint.Apply(BODY_SIZE, TABLE_TOP_BASELINE_RATIO)
	cell_index = 0
	for cell_index < input.Columns {
		cell_x := PAGE_MARGIN +
			fixedpoint.Number(cell_index)*input.Column_Width + TABLE_PADDING
		layout_table_cell(state, &Layout_Table_Cell_Input{
			Lines:        cell_lines[cell_index],
			X:            cell_x,
			Top_Baseline: top_baseline,
			Line_Height:  line_height,
		})
		cell_index++
	}
	state.Cursor = row_bottom
}

// Layout_Table_Cell_Input carries one cell's wrapped lines and where to stack them.
type Layout_Table_Cell_Input struct {
	// Lines is the cell's wrapped lines of pieces.
	Lines [][]Word_Piece
	// X is the cell's left text edge.
	X fixedpoint.Number
	// Top_Baseline is the baseline of the cell's first line.
	Top_Baseline fixedpoint.Number
	// Line_Height is the vertical step between lines.
	Line_Height fixedpoint.Number
}

// Stacks a cell's wrapped lines downward from its top baseline, collecting any
// link rectangles the lines produce.
func layout_table_cell(state *Layout, input *Layout_Table_Cell_Input) {
	line_index := 0
	for line_index < len(input.Lines) {
		links := emit_text_line(state.Stream, &Emit_Text_Line_Input{
			Pieces: input.Lines[line_index],
			Size:   BODY_SIZE,
			X:      input.X,
			Y: input.Top_Baseline -
				fixedpoint.Number(line_index)*input.Line_Height,
		})
		state.Links = append(state.Links, links...)
		line_index++
	}
}

// Parses a cell's inline markdown; a header cell forces its prose runs bold so
// the header row reads as a header while code spans stay monospace.
func table_cell_runs(cell_text string, header bool) (runs []Text_Run) {
	cell_runs := parse_inline(cell_text)
	if !header {
		return cell_runs
	}
	for _, run := range cell_runs {
		font := run.Font
		if font == FONT_REGULAR {
			font = FONT_BOLD
		}
		if font == FONT_ITALIC {
			font = FONT_BOLD_ITALIC
		}
		runs = append(runs, Text_Run{Text: run.Text, Font: font, Link: run.Link})
	}
	return runs
}

// The GitHub gray cell grid for one row: its top and bottom edges plus a
// vertical at every column boundary, then the stroke color is restored.
type Emit_Table_Grid_Input struct {
	// Top is the row's top edge.
	Top fixedpoint.Number
	// Bottom is the row's bottom edge.
	Bottom fixedpoint.Number
	// Column_Width is the width of each column.
	Column_Width fixedpoint.Number
	// Columns is the number of columns whose verticals are drawn.
	Columns int
}

func emit_table_grid(stream *strings.Builder, input *Emit_Table_Grid_Input) {
	stream.WriteString(TABLE_BORDER_STROKE)
	stream.WriteByte('\n')
	emit_stroke(stream, &Emit_Stroke_Input{
		X1: PAGE_MARGIN,
		Y1: input.Top,
		X2: PAGE_WIDTH - PAGE_MARGIN,
		Y2: input.Top,
	})
	emit_stroke(stream, &Emit_Stroke_Input{
		X1: PAGE_MARGIN,
		Y1: input.Bottom,
		X2: PAGE_WIDTH - PAGE_MARGIN,
		Y2: input.Bottom,
	})
	line_index := 0
	for line_index <= input.Columns {
		column_x := PAGE_MARGIN + fixedpoint.Number(line_index)*input.Column_Width
		emit_stroke(stream, &Emit_Stroke_Input{
			X1: column_x,
			Y1: input.Top,
			X2: column_x,
			Y2: input.Bottom,
		})
		line_index++
	}
	stream.WriteString(NORMAL_STROKE)
	stream.WriteByte('\n')
}

func table_cell_text(cells []string, index int) (text string) {
	if index >= len(cells) {
		return ""
	}
	return cells[index]
}

// Layout_Prose_Input carries a paragraph's runs and the metrics that place them.
type Layout_Prose_Input struct {
	// Runs holds the paragraph's inline spans.
	Runs []Text_Run
	// Size is the font size in points.
	Size fixedpoint.Number
	// Indent is the left indent from the page margin.
	Indent fixedpoint.Number
	// Line_Height is the vertical step between lines.
	Line_Height fixedpoint.Number
	// Color is the base text fill, or empty for the default.
	Color string
}

func layout_prose(state *Layout, input *Layout_Prose_Input) {
	available_width := PAGE_WIDTH - 2*PAGE_MARGIN - input.Indent
	lines := wrap_pieces(&Wrap_Pieces_Input{
		Runs:      input.Runs,
		Size:      input.Size,
		Width_Max: available_width,
	})
	for _, visual_line := range lines {
		layout_line(state, &Layout_Line_Input{
			Pieces:      visual_line,
			Size:        input.Size,
			Indent:      input.Indent,
			Line_Height: input.Line_Height,
			Color:       input.Color,
		})
	}
}

// Wrap_Pieces_Input carries the runs to wrap and the column width they must fit.
type Wrap_Pieces_Input struct {
	// Runs holds the spans to break into pieces.
	Runs []Text_Run
	// Size is the font size the widths are measured at.
	Size fixedpoint.Number
	// Width_Max is the widest a line may be.
	Width_Max fixedpoint.Number
}

func wrap_pieces(input *Wrap_Pieces_Input) (lines [][]Word_Piece) {
	var pieces []Word_Piece
	for _, run := range input.Runs {
		// A code span stays one piece so its panel is unbroken at normal
		// widths; prose splits into words. Either is broken at character
		// boundaries below only when a single piece cannot fit the column.
		if run.Font == FONT_CODE {
			pieces = append(pieces, Word_Piece{
				Text: run.Text,
				Font: run.Font,
				Link: run.Link,
			})
			continue
		}
		for _, field := range strings.Fields(run.Text) {
			pieces = append(pieces, Word_Piece{
				Text: field,
				Font: run.Font,
				Link: run.Link,
			})
		}
	}
	var current []Word_Piece
	used_width := fixedpoint.Number(0)
	space_width := text_width(" ", FONT_REGULAR, input.Size)
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
			chunks := split_piece(&Split_Piece_Input{
				Piece:     piece,
				Width_Max: input.Width_Max,
				Size:      input.Size,
			})
			for chunk_index := 0; chunk_index < len(chunks)-1; chunk_index++ {
				lines = append(lines, []Word_Piece{chunks[chunk_index]})
			}
			last_chunk := chunks[len(chunks)-1]
			current = []Word_Piece{last_chunk}
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

// Split_Piece_Input carries one over-wide piece and the width its chunks must fit.
type Split_Piece_Input struct {
	// Piece is the piece too wide for a single line.
	Piece Word_Piece
	// Width_Max is the widest a chunk may be.
	Width_Max fixedpoint.Number
	// Size is the font size the widths are measured at.
	Size fixedpoint.Number
}

// Breaks one piece into the longest rune-prefixes that each fit Width_Max,
// taking at least one rune per chunk so a single over-wide glyph still
// advances. Each chunk inherits the piece's font and link, so a broken code
// span renders as a stack of panels.
func split_piece(input *Split_Piece_Input) (chunks []Word_Piece) {
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
		chunks = append(chunks, Word_Piece{
			Text: string(runes[start_index:end_index]),
			Font: input.Piece.Font,
			Link: input.Piece.Link,
		})
		start_index = end_index
	}
	return chunks
}

// Layout_Line_Input carries one wrapped line and the metrics that place it.
type Layout_Line_Input struct {
	// Pieces holds the line's laid-out pieces.
	Pieces []Word_Piece
	// Size is the font size in points.
	Size fixedpoint.Number
	// Indent is the left indent from the page margin.
	Indent fixedpoint.Number
	// Line_Height is the vertical step consumed after the line.
	Line_Height fixedpoint.Number
	// Color is the base text fill, or empty for the default.
	Color string
}

func layout_line(state *Layout, input *Layout_Line_Input) {
	layout_need(state, input.Line_Height)
	links := emit_text_line(state.Stream, &Emit_Text_Line_Input{
		Pieces: input.Pieces,
		Size:   input.Size,
		X:      PAGE_MARGIN + input.Indent,
		Y:      state.Cursor,
		Color:  input.Color,
	})
	state.Links = append(state.Links, links...)
	state.Cursor -= input.Line_Height
}

func layout_need(state *Layout, height fixedpoint.Number) {
	if state.Cursor-height >= PAGE_MARGIN {
		return
	}
	layout_break(state)
}

func layout_break(state *Layout) {
	state.Pages = append(state.Pages, Page_Content{
		Stream: state.Stream.String(),
		Links:  state.Links,
	})
	state.Stream.Reset()
	state.Links = nil
	state.Cursor = PAGE_HEIGHT - PAGE_MARGIN
}

// Courier measures by glyph count; Helvetica weights measure off their own
// metric face via glyph_advance. These widths place the panels and underlines,
// so they must match the advances the viewer draws with, bold included.
func text_width(text string, font int, size fixedpoint.Number) (width fixedpoint.Number) {
	if font == FONT_CODE {
		return fixedpoint.Apply(
			fixedpoint.Number(len([]rune(text)))*size, COURIER_ADVANCE_RATIO)
	}
	advance_units := 0
	for _, glyph := range text {
		advance_units += glyph_advance(font, winansi_byte(glyph))
	}
	// The advance units are 1000-em; divide by 1000 to reach point space.
	return fixedpoint.Number(advance_units) * size / 1000
}

// Routes to the metric face for the font: the bold weights off Helvetica-Bold,
// regular and oblique off Helvetica, whose widths the oblique face shares.
func glyph_advance(font int, code byte) (advance int) {
	if font == FONT_BOLD {
		return helvetica_bold_advance(code)
	}
	if font == FONT_BOLD_ITALIC {
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
type Emit_Text_Line_Input struct {
	// Pieces holds the line's pieces, shown left to right.
	Pieces []Word_Piece
	// Size is the font size in points.
	Size fixedpoint.Number
	// X is the line's left edge.
	X fixedpoint.Number
	// Y is the line's baseline.
	Y fixedpoint.Number
	// Color is the base text fill, or empty for the default.
	Color string
}

func emit_text_line(stream *strings.Builder, input *Emit_Text_Line_Input) (links []Link_Box) {
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
// yields a Link_Box, the clickable rectangle the page turns into an annotation.
func emit_line_decorations(
	stream *strings.Builder, input *Emit_Text_Line_Input,
) (links []Link_Box) {
	cursor_x := input.X
	first := true
	run_start := input.X
	run_target := ""
	run_active := false
	for _, piece := range input.Pieces {
		space_before := fixedpoint.Number(0)
		if !first {
			space_before = text_width(" ", piece.Font, input.Size)
		}
		first = false
		text_start := cursor_x + space_before
		text_end := text_start + text_width(piece.Text, piece.Font, input.Size)
		emit_code_panel(stream, &Emit_Code_Panel_Input{
			Piece: piece,
			Start: text_start,
			End:   text_end,
			Y:     input.Y,
			Size:  input.Size,
		})
		if run_active {
			if piece.Link == "" {
				links = append(links, emit_link_run(stream, &Emit_Link_Run_Input{
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
		links = append(links, emit_link_run(stream, &Emit_Link_Run_Input{
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
type Emit_Link_Run_Input struct {
	// Target is the run's destination URL.
	Target string
	// Start is the run's left edge.
	Start fixedpoint.Number
	// End is the run's right edge.
	End fixedpoint.Number
	// Y is the baseline the underline drops from.
	Y fixedpoint.Number
	// Size is the font size in points.
	Size fixedpoint.Number
}

func emit_link_run(stream *strings.Builder, input *Emit_Link_Run_Input) (box Link_Box) {
	stream.WriteString(LINK_STROKE)
	stream.WriteByte('\n')
	emit_stroke(stream, &Emit_Stroke_Input{
		X1: input.Start,
		Y1: input.Y - UNDERLINE_DROP,
		X2: input.End,
		Y2: input.Y - UNDERLINE_DROP,
	})
	stream.WriteString(NORMAL_STROKE)
	stream.WriteByte('\n')
	return Link_Box{
		Target: input.Target,
		X1:     input.Start,
		Y1:     input.Y - fixedpoint.Apply(input.Size, LINK_DESCENT_RATIO),
		X2:     input.End,
		Y2:     input.Y + fixedpoint.Apply(input.Size, LINK_ASCENT_RATIO),
	}
}

// Emit_Code_Panel_Input carries one piece and the span it occupies, so a code
// piece can be backed by a panel.
type Emit_Code_Panel_Input struct {
	// Piece is the piece that may need a panel.
	Piece Word_Piece
	// Start is the piece's left edge.
	Start fixedpoint.Number
	// End is the piece's right edge.
	End fixedpoint.Number
	// Y is the piece's baseline.
	Y fixedpoint.Number
	// Size is the font size in points.
	Size fixedpoint.Number
}

func emit_code_panel(stream *strings.Builder, input *Emit_Code_Panel_Input) {
	if input.Piece.Font != FONT_CODE {
		return
	}
	emit_panel(stream, &Emit_Panel_Input{
		Fill:   CODE_PANEL_FILL,
		X:      input.Start - CODE_PANEL_INSET,
		Y:      input.Y - fixedpoint.Apply(input.Size, CODE_INLINE_DESCENT_RATIO),
		Width:  input.End - input.Start + 2*CODE_PANEL_INSET,
		Height: fixedpoint.Apply(input.Size, CODE_INLINE_HEIGHT_RATIO),
	})
}

// Emit_Stroke_Input carries the two endpoints of a straight stroke.
type Emit_Stroke_Input struct {
	// X1 is the start point's x.
	X1 fixedpoint.Number
	// Y1 is the start point's y.
	Y1 fixedpoint.Number
	// X2 is the end point's x.
	X2 fixedpoint.Number
	// Y2 is the end point's y.
	Y2 fixedpoint.Number
}

func emit_stroke(stream *strings.Builder, input *Emit_Stroke_Input) {
	fmt.Fprintf(stream, "%s %s m %s %s l S\n",
		format_number(input.X1), format_number(input.Y1),
		format_number(input.X2), format_number(input.Y2))
}

// Emit_Panel_Input carries a filled rectangle's color and geometry.
type Emit_Panel_Input struct {
	// Fill is the PDF fill-color operator for the panel.
	Fill string
	// X is the rectangle's left edge.
	X fixedpoint.Number
	// Y is the rectangle's bottom edge.
	Y fixedpoint.Number
	// Width is the rectangle's width.
	Width fixedpoint.Number
	// Height is the rectangle's height.
	Height fixedpoint.Number
}

func emit_panel(stream *strings.Builder, input *Emit_Panel_Input) {
	stream.WriteString(input.Fill)
	stream.WriteByte('\n')
	fmt.Fprintf(stream, "%s %s %s %s re\nf\n",
		format_number(input.X), format_number(input.Y),
		format_number(input.Width), format_number(input.Height))
}

// A code line's panel spans the full text column so consecutive lines tile into
// one continuous band; the descent offset keeps each line's glyphs inside it.
func emit_code_band(stream *strings.Builder, baseline fixedpoint.Number) {
	emit_panel(stream, &Emit_Panel_Input{
		Fill:   CODE_PANEL_FILL,
		X:      PAGE_MARGIN,
		Y:      baseline - fixedpoint.Apply(CODE_SIZE, CODE_BAND_DESCENT_RATIO),
		Width:  PAGE_WIDTH - 2*PAGE_MARGIN,
		Height: fixedpoint.Apply(CODE_SIZE, LINE_LEADING_RATIO),
	})
}

// The fill set here is what the following Tj paints with: white for code on its
// panel, blue for a link, else the line's base color or black. Every text show
// sets its own color, so a panel or link fill never bleeds onto later glyphs.
func emit_text_color(stream *strings.Builder, font int, is_link bool, base_color string) {
	if font == FONT_CODE {
		stream.WriteString(CODE_TEXT_FILL)
		stream.WriteByte('\n')
		return
	}
	if is_link {
		stream.WriteString(LINK_FILL)
		stream.WriteByte('\n')
		return
	}
	if base_color == "" {
		stream.WriteString(NORMAL_TEXT_FILL)
		stream.WriteByte('\n')
		return
	}
	stream.WriteString(base_color)
	stream.WriteByte('\n')
}

// Emit_Glyphs_Input carries a single run of glyphs and where to draw them.
type Emit_Glyphs_Input struct {
	// Text is the literal characters to draw.
	Text string
	// Font is the FONT_* code to draw them in.
	Font int
	// Size is the font size in points.
	Size fixedpoint.Number
	// X is the run's left edge.
	X fixedpoint.Number
	// Y is the run's baseline.
	Y fixedpoint.Number
}

func emit_glyphs(stream *strings.Builder, input *Emit_Glyphs_Input) {
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

func assemble_document(pages []Page_Content) (document []byte) {
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
	annot_number := FIRST_DYNAMIC_OBJECT + 2*len(pages)
	page_index := 0
	for page_index < len(pages) {
		objects = append(objects, page_object(&Page_Object_Input{
			Content:     FIRST_DYNAMIC_OBJECT + len(pages) + page_index,
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
const FIRST_DYNAMIC_OBJECT = 8

func pages_object(count int) (body string) {
	var kids strings.Builder
	page_index := 0
	for page_index < count {
		if page_index > 0 {
			kids.WriteByte(' ')
		}
		fmt.Fprintf(&kids, "%d 0 R", FIRST_DYNAMIC_OBJECT+page_index)
		page_index++
	}
	return fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", kids.String(), count)
}

// Page_Object_Input carries the object numbers a page dictionary must reference.
type Page_Object_Input struct {
	// Content is the object number of the page's content stream.
	Content int
	// Annot_First is the object number of the page's first annotation.
	Annot_First int
	// Annot_Count is how many annotation objects the page has.
	Annot_Count int
}

func page_object(input *Page_Object_Input) (body string) {
	resources := "<< /Font << /F0 3 0 R /F1 4 0 R /F2 5 0 R /F3 6 0 R /F4 7 0 R >> >>"
	annots := page_annots(&Page_Annots_Input{
		First: input.Annot_First,
		Count: input.Annot_Count,
	})
	return fmt.Sprintf(
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %s %s] "+
			"/Resources %s%s /Contents %d 0 R >>",
		format_number(PAGE_WIDTH), format_number(PAGE_HEIGHT),
		resources, annots, input.Content)
}

// Page_Annots_Input carries the object-number range of a page's annotations.
type Page_Annots_Input struct {
	// First is the object number of the first annotation.
	First int
	// Count is how many annotation objects follow.
	Count int
}

func page_annots(input *Page_Annots_Input) (clause string) {
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

func link_box_annotation(link Link_Box) (body string) {
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
	output.WriteString(PDF_HEADER)
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

func format_number(value fixedpoint.Number) (text string) {
	if fixedpoint.Is_Integer(value) {
		return strconv.FormatInt(fixedpoint.Whole(value), 10)
	}
	return fixedpoint.Format(value, 2)
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
