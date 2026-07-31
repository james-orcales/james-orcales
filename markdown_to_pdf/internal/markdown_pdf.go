// Package markdown_to_pdf renders a Markdown document to a PDF file. It is the
// pure tier of the markdown_to_pdf binary: every dependency on the outside world
// (the source bytes, the output sink, the diagnostic sink) arrives through
// Main_Input, so the renderer reads no files and touches no process globals.
package markdown_to_pdf

import (
	"bytes"
	"encoding/ascii85"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	bounded_zlib "local/james-orcales/shared/compress/zlib"
	"local/james-orcales/shared/math/fixedpoint"
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

// PDF_BYTES_MAX is the largest accepted source document and extracted Markdown.
const PDF_BYTES_MAX = 64 * 1024 * 1024

// PDF_OBJECT_COUNT_MAX bounds the indirect-object index.
const PDF_OBJECT_COUNT_MAX = 262144

// PDF_PAGE_COUNT_MAX bounds page-tree traversal and retained output chunks.
const PDF_PAGE_COUNT_MAX = 65536

// PDF_DEPTH_MAX bounds nested values, page trees, forms, and object resolution.
const PDF_DEPTH_MAX = 128

// PDF_STREAM_BYTES_MAX bounds each decoded stream.
const PDF_STREAM_BYTES_MAX = 64 * 1024 * 1024

// PDF_DECODED_BYTES_MAX bounds all decoded streams in one conversion.
const PDF_DECODED_BYTES_MAX = 256 * 1024 * 1024

// PDF_VALUE_NULL tags the PDF null object.
const PDF_VALUE_NULL = 0

// PDF_VALUE_BOOLEAN tags a PDF boolean.
const PDF_VALUE_BOOLEAN = 1

// PDF_VALUE_NUMBER tags an integer or fixed-point PDF number.
const PDF_VALUE_NUMBER = 2

// PDF_VALUE_NAME tags a slash-prefixed PDF name.
const PDF_VALUE_NAME = 3

// PDF_VALUE_STRING tags literal and hexadecimal PDF strings.
const PDF_VALUE_STRING = 4

// PDF_VALUE_ARRAY tags a nested PDF array.
const PDF_VALUE_ARRAY = 5

// PDF_VALUE_DICTIONARY tags a PDF dictionary and its optional stream.
const PDF_VALUE_DICTIONARY = 6

// PDF_VALUE_REFERENCE tags an indirect-object reference.
const PDF_VALUE_REFERENCE = 7

// PDF_VALUE_KEYWORD tags a content-stream operator or other bare token.
const PDF_VALUE_KEYWORD = 8

// Pdf_Value is a concrete tagged union, which avoids an interface allocation
// for every token.
type Pdf_Value struct {
	// Kind selects the one populated value shape.
	Kind int
	// Boolean stores a boolean token when Kind is PDF_VALUE_BOOLEAN.
	Boolean bool
	// Integer preserves exact values for offsets, counts, and object numbers.
	Integer int64
	// Number preserves fractions without platform-dependent floating point.
	Number fixedpoint.Number
	// Name stores names and bare keywords without their delimiters.
	Name string
	// String_Bytes preserves source codes before font decoding.
	String_Bytes []byte
	// Array preserves source order because PDF arrays are positional.
	Array []Pdf_Value
	// Dictionary permits direct lookup of named entries.
	Dictionary map[string]Pdf_Value
	// Reference_Object_Number selects the referenced indirect object.
	Reference_Object_Number int
	// Reference_Generation retains the declared reference generation.
	Reference_Generation int
	// Stream retains encoded bytes until the owning page needs them.
	Stream []byte
}

// Pdf_Indirect_Object pairs a value with incremental-update metadata.
type Pdf_Indirect_Object struct {
	// Value is the parsed indirect-object body.
	Value Pdf_Value
	// Generation preserves the generation declared by the object header.
	Generation int
	// Offset lets a later incremental update replace an earlier definition.
	Offset int
}

// Pdf_Document owns bounded indexes and the cumulative decode budget.
type Pdf_Document struct {
	// Source remains live while stream slices refer to encoded bytes.
	Source []byte
	// Objects indexes only the newest definition of each object.
	Objects map[int]Pdf_Indirect_Object
	// Decoded_Bytes_Count enforces the conversion-wide decode budget.
	Decoded_Bytes_Count int64
	// Object_Headers_Count also counts compressed objects against the cap.
	Object_Headers_Count int
	// Object_Streams_Decoded prevents repeated expansion during index growth.
	Object_Streams_Decoded map[int]bool
}

// Pdf_Parser is a bounded cursor over source or decoded object bytes.
type Pdf_Parser struct {
	// Source is the immutable byte sequence under the cursor.
	Source []byte
	// Offset is the next byte that value parsing will consume.
	Offset int
}

// Pdf_Object_Header identifies one top-level indirect object.
type Pdf_Object_Header struct {
	// Found distinguishes no match from object zero.
	Found bool
	// Object_Number is the indirect-object index.
	Object_Number int
	// Generation is the header's declared generation.
	Generation int
	// Value_Offset starts the object body after the obj keyword.
	Value_Offset int
	// Header_Offset orders incremental definitions by physical location.
	Header_Offset int
}

// PDF_To_Markdown converts one unencrypted PDF into MarkItDown-compatible
// Markdown while enforcing all parser and output bounds.
func PDF_To_Markdown(pdf []byte) (markdown []byte, err error) {
	if len(pdf) > PDF_BYTES_MAX {
		return nil, fmt.Errorf("PDF input exceeds 64 MiB")
	}
	if !pdf_header_valid(pdf) {
		return nil, fmt.Errorf("invalid PDF header")
	}
	if bytes.Contains(pdf, []byte("/Encrypt")) {
		return nil, fmt.Errorf("encrypted PDF is not supported")
	}
	document, parse_err := pdf_parse_document(pdf)
	if parse_err != nil {
		return nil, parse_err
	}
	markdown, extract_err := pdf_extract_document(document)
	if extract_err != nil {
		return nil, extract_err
	}
	if len(markdown) > PDF_BYTES_MAX {
		return nil, fmt.Errorf("Markdown output exceeds 64 MiB")
	}
	return markdown, nil
}

func pdf_header_valid(source []byte) (valid bool) {
	if len(source) < 8 {
		return false
	}
	if !bytes.HasPrefix(source, []byte("%PDF-1.")) {
		return false
	}
	version := source[7]
	return version >= '0' && version <= '9'
}

func pdf_parse_document(source []byte) (document *Pdf_Document, err error) {
	document = &Pdf_Document{
		Source:                 source,
		Objects:                make(map[int]Pdf_Indirect_Object),
		Object_Streams_Decoded: make(map[int]bool),
	}
	search_offset := 0
	for search_offset < len(source) {
		header := pdf_find_object_header(source, search_offset)
		if !header.Found {
			break
		}
		next_offset, parse_err := pdf_parse_indirect_object(document, &header)
		if parse_err != nil {
			return nil, parse_err
		}
		search_offset = next_offset
	}
	if len(document.Objects) == 0 {
		return nil, fmt.Errorf("PDF has no indirect objects")
	}
	if decode_err := pdf_decode_object_streams(document); decode_err != nil {
		return nil, decode_err
	}
	return document, nil
}

func pdf_find_object_header(source []byte, start int) (header Pdf_Object_Header) {
	for keyword_offset := start; keyword_offset+3 <= len(source); keyword_offset++ {
		if !bytes.Equal(source[keyword_offset:keyword_offset+3], []byte("obj")) {
			continue
		}
		parsed, valid := pdf_object_header_before(source, keyword_offset)
		if valid {
			return parsed
		}
	}
	return header
}

func pdf_object_header_before(source []byte, keyword_offset int) (
	header Pdf_Object_Header,
	valid bool,
) {
	if !pdf_keyword_boundary(&Pdf_Keyword_Boundary_Input{
		Source: source, Offset: keyword_offset, Size: 3,
	}) {
		return header, false
	}
	cursor := pdf_skip_space_reverse(source, keyword_offset-1)
	generation_end := cursor + 1
	generation_start := pdf_digits_start(source, cursor)
	if generation_start == generation_end {
		return header, false
	}
	cursor = pdf_skip_space_reverse(source, generation_start-1)
	object_end := cursor + 1
	object_start := pdf_digits_start(source, cursor)
	if object_start == object_end {
		return header, false
	}
	object_number, object_err := strconv.Atoi(string(source[object_start:object_end]))
	generation, generation_err := strconv.Atoi(string(source[generation_start:generation_end]))
	if object_err != nil {
		return header, false
	}
	if generation_err != nil {
		return header, false
	}
	return Pdf_Object_Header{
		Found: true, Object_Number: object_number, Generation: generation,
		Value_Offset: keyword_offset + 3, Header_Offset: object_start,
	}, true
}

func pdf_skip_space_reverse(source []byte, offset int) (result int) {
	for offset >= 0 && pdf_is_space(source[offset]) {
		offset--
	}
	return offset
}

func pdf_digits_start(source []byte, offset int) (start int) {
	for offset >= 0 && source[offset] >= '0' && source[offset] <= '9' {
		offset--
	}
	return offset + 1
}

// Pdf_Keyword_Boundary_Input groups the source range around one PDF keyword.
type Pdf_Keyword_Boundary_Input struct {
	// Source contains the PDF syntax.
	Source []byte
	// Offset locates the first keyword byte.
	Offset int
	// Size is the keyword byte count.
	Size int
}

func pdf_keyword_boundary(input *Pdf_Keyword_Boundary_Input) (valid bool) {
	if input.Offset > 0 {
		if !pdf_is_delimiter(input.Source[input.Offset-1]) {
			return false
		}
	}
	end_offset := input.Offset + input.Size
	if end_offset < len(input.Source) {
		if !pdf_is_delimiter(input.Source[end_offset]) {
			return false
		}
	}
	return true
}

func pdf_parse_indirect_object(
	document *Pdf_Document,
	header *Pdf_Object_Header,
) (next_offset int, err error) {
	document.Object_Headers_Count++
	if document.Object_Headers_Count > PDF_OBJECT_COUNT_MAX {
		return 0, fmt.Errorf("PDF exceeds %d indirect objects", PDF_OBJECT_COUNT_MAX)
	}
	parser := &Pdf_Parser{Source: document.Source, Offset: header.Value_Offset}
	value, value_err := pdf_parse_value(parser, 0)
	if value_err != nil {
		return 0, fmt.Errorf("PDF object %d: %w", header.Object_Number, value_err)
	}
	pdf_skip_space_and_comments(parser)
	if value.Kind == PDF_VALUE_DICTIONARY {
		if pdf_match_keyword(parser, "stream") {
			stream_err := pdf_parse_stream(parser, &value)
			if stream_err != nil {
				return 0, fmt.Errorf(
					"PDF object %d: %w", header.Object_Number, stream_err,
				)
			}
		}
	}
	end_offset := pdf_find_end_object(document.Source, parser.Offset)
	if end_offset < 0 {
		return 0, fmt.Errorf("PDF object %d has no endobj", header.Object_Number)
	}
	pdf_store_object(document, header, value)
	return end_offset + len("endobj"), nil
}

func pdf_store_object(
	document *Pdf_Document,
	header *Pdf_Object_Header,
	value Pdf_Value,
) {
	prior, exists := document.Objects[header.Object_Number]
	if exists {
		if prior.Offset > header.Header_Offset {
			return
		}
	}
	document.Objects[header.Object_Number] = Pdf_Indirect_Object{
		Value: value, Generation: header.Generation, Offset: header.Header_Offset,
	}
}

func pdf_find_end_object(source []byte, start int) (offset int) {
	relative_offset := bytes.Index(source[start:], []byte("endobj"))
	if relative_offset < 0 {
		return -1
	}
	return start + relative_offset
}

func pdf_parse_stream(parser *Pdf_Parser, value *Pdf_Value) (err error) {
	if parser.Offset < len(parser.Source) {
		if parser.Source[parser.Offset] == '\r' {
			parser.Offset++
		}
	}
	if parser.Offset < len(parser.Source) {
		if parser.Source[parser.Offset] == '\n' {
			parser.Offset++
		}
	}
	stream_start := parser.Offset
	stream_end := pdf_direct_stream_end(value, stream_start, parser.Source)
	if stream_end < 0 {
		relative_offset := bytes.Index(parser.Source[stream_start:], []byte("endstream"))
		if relative_offset < 0 {
			return fmt.Errorf("stream has no endstream")
		}
		stream_end = stream_start + relative_offset
		for stream_end > stream_start && pdf_is_line_end(parser.Source[stream_end-1]) {
			stream_end--
		}
	}
	value.Stream = parser.Source[stream_start:stream_end]
	end_relative_offset := bytes.Index(parser.Source[stream_end:], []byte("endstream"))
	if end_relative_offset < 0 {
		return fmt.Errorf("stream has no endstream")
	}
	parser.Offset = stream_end + end_relative_offset + len("endstream")
	return nil
}

func pdf_direct_stream_end(value *Pdf_Value, start int, source []byte) (end int) {
	stream_size, exists := value.Dictionary["Length"]
	if !exists {
		return -1
	}
	if stream_size.Kind != PDF_VALUE_NUMBER {
		return -1
	}
	if stream_size.Integer < 0 {
		return -1
	}
	if stream_size.Integer > int64(len(source)-start) {
		return -1
	}
	end = start + int(stream_size.Integer)
	probe := end
	for probe < len(source) && pdf_is_space(source[probe]) {
		probe++
	}
	if !bytes.HasPrefix(source[probe:], []byte("endstream")) {
		return -1
	}
	return end
}

// Pdf_Value_Frame retains one unfinished array or dictionary.
type Pdf_Value_Frame struct {
	// Value is the container under construction.
	Value Pdf_Value
	// Dictionary_Key is the pending dictionary key.
	Dictionary_Key string
	// Has_Dictionary_Key distinguishes an empty key from no pending key.
	Has_Dictionary_Key bool
}

func pdf_parse_value(parser *Pdf_Parser, depth int) (value Pdf_Value, err error) {
	frames := make([]Pdf_Value_Frame, 0, 8)
	operation_count_max := len(parser.Source)*2 + 1
	for operation_index := 0; operation_index < operation_count_max; operation_index++ {
		pdf_skip_space_and_comments(parser)
		if parser.Offset >= len(parser.Source) {
			return value, fmt.Errorf("unexpected end of PDF value")
		}
		container_value, handled, complete, container_err := pdf_parse_container_token(
			parser, &frames,
		)
		if container_err != nil {
			return value, container_err
		}
		if complete {
			return container_value, nil
		}
		if handled {
			continue
		}
		if bytes.HasPrefix(parser.Source[parser.Offset:], []byte("<<")) {
			if depth+len(frames)+1 > PDF_DEPTH_MAX {
				return value, fmt.Errorf(
					"PDF value nesting exceeds %d", PDF_DEPTH_MAX,
				)
			}
			parser.Offset += 2
			frames = append(frames, Pdf_Value_Frame{Value: Pdf_Value{
				Kind: PDF_VALUE_DICTIONARY, Dictionary: make(map[string]Pdf_Value),
			}})
			continue
		}
		if parser.Source[parser.Offset] == '[' {
			if depth+len(frames)+1 > PDF_DEPTH_MAX {
				return value, fmt.Errorf(
					"PDF value nesting exceeds %d", PDF_DEPTH_MAX,
				)
			}
			parser.Offset++
			frames = append(frames, Pdf_Value_Frame{
				Value: Pdf_Value{Kind: PDF_VALUE_ARRAY},
			})
			continue
		}
		value, err = pdf_parse_leaf_value(parser)
		if err != nil {
			return value, err
		}
		if len(frames) == 0 {
			return value, nil
		}
		pdf_attach_value(&frames[len(frames)-1], value)
	}
	return value, fmt.Errorf("PDF value parser operation limit exceeded")
}

func pdf_parse_container_token(
	parser *Pdf_Parser,
	frames *[]Pdf_Value_Frame,
) (value Pdf_Value, handled bool, complete bool, err error) {
	if len(*frames) == 0 {
		return value, false, false, nil
	}
	frame := &(*frames)[len(*frames)-1]
	if frame.Value.Kind == PDF_VALUE_DICTIONARY {
		if frame.Has_Dictionary_Key {
			return value, false, false, nil
		}
		if bytes.HasPrefix(parser.Source[parser.Offset:], []byte(">>")) {
			parser.Offset += 2
			return pdf_close_value_frame(frames)
		}
		if parser.Source[parser.Offset] != '/' {
			return value, false, false, fmt.Errorf("dictionary key is not a name")
		}
		key := pdf_parse_name(parser)
		frame.Dictionary_Key = key.Name
		frame.Has_Dictionary_Key = true
		return value, true, false, nil
	}
	if parser.Source[parser.Offset] != ']' {
		return value, false, false, nil
	}
	parser.Offset++
	return pdf_close_value_frame(frames)
}

func pdf_close_value_frame(
	frames *[]Pdf_Value_Frame,
) (value Pdf_Value, handled bool, complete bool, err error) {
	value = (*frames)[len(*frames)-1].Value
	*frames = (*frames)[:len(*frames)-1]
	if len(*frames) == 0 {
		return value, true, true, nil
	}
	pdf_attach_value(&(*frames)[len(*frames)-1], value)
	return value, true, false, nil
}

func pdf_attach_value(frame *Pdf_Value_Frame, value Pdf_Value) {
	if frame.Value.Kind == PDF_VALUE_ARRAY {
		frame.Value.Array = append(frame.Value.Array, value)
		return
	}
	frame.Value.Dictionary[frame.Dictionary_Key] = value
	frame.Dictionary_Key = ""
	frame.Has_Dictionary_Key = false
}

func pdf_parse_leaf_value(parser *Pdf_Parser) (value Pdf_Value, err error) {
	switch parser.Source[parser.Offset] {
	case '(':
		return pdf_parse_literal_string(parser)
	case '<':
		return pdf_parse_hexadecimal_string(parser)
	case '/':
		return pdf_parse_name(parser), nil
	}
	return pdf_parse_scalar(parser)
}

func pdf_parse_name(parser *Pdf_Parser) (value Pdf_Value) {
	parser.Offset++
	var name strings.Builder
	for parser.Offset < len(parser.Source) {
		current := parser.Source[parser.Offset]
		if pdf_is_delimiter(current) {
			break
		}
		if current == '#' {
			if parser.Offset+2 < len(parser.Source) {
				decoded, decode_err := hex.DecodeString(
					string(parser.Source[parser.Offset+1 : parser.Offset+3]),
				)
				if decode_err == nil {
					name.WriteByte(decoded[0])
					parser.Offset += 3
					continue
				}
			}
		}
		name.WriteByte(current)
		parser.Offset++
	}
	return Pdf_Value{Kind: PDF_VALUE_NAME, Name: name.String()}
}

func pdf_parse_literal_string(parser *Pdf_Parser) (value Pdf_Value, err error) {
	parser.Offset++
	depth := 1
	output := make([]byte, 0, 32)
	for parser.Offset < len(parser.Source) {
		current := parser.Source[parser.Offset]
		parser.Offset++
		if current == '\\' {
			output = pdf_parse_string_escape(parser, output)
			continue
		}
		if current == '(' {
			depth++
		}
		if current == ')' {
			depth--
			if depth == 0 {
				return Pdf_Value{Kind: PDF_VALUE_STRING, String_Bytes: output}, nil
			}
		}
		output = append(output, current)
	}
	return value, fmt.Errorf("unterminated literal string")
}

func pdf_parse_string_escape(parser *Pdf_Parser, output []byte) (result []byte) {
	if parser.Offset >= len(parser.Source) {
		return output
	}
	current := parser.Source[parser.Offset]
	parser.Offset++
	switch current {
	case 'n':
		return append(output, '\n')
	case 'r':
		return append(output, '\r')
	case 't':
		return append(output, '\t')
	case 'b':
		return append(output, '\b')
	case 'f':
		return append(output, '\f')
	case '\n':
		return output
	case '\r':
		if parser.Offset < len(parser.Source) {
			if parser.Source[parser.Offset] == '\n' {
				parser.Offset++
			}
		}
		return output
	}
	if current >= '0' {
		if current <= '7' {
			return pdf_parse_octal_escape(parser, output, current)
		}
	}
	return append(output, current)
}

func pdf_parse_octal_escape(parser *Pdf_Parser, output []byte, first byte) (result []byte) {
	value := int(first - '0')
	digit_count := 1
	for digit_count < 3 && parser.Offset < len(parser.Source) {
		current := parser.Source[parser.Offset]
		if current < '0' {
			break
		}
		if current > '7' {
			break
		}
		value = value*8 + int(current-'0')
		parser.Offset++
		digit_count++
	}
	return append(output, byte(value))
}

func pdf_parse_hexadecimal_string(parser *Pdf_Parser) (value Pdf_Value, err error) {
	parser.Offset++
	digits := make([]byte, 0, 32)
	for parser.Offset < len(parser.Source) && parser.Source[parser.Offset] != '>' {
		current := parser.Source[parser.Offset]
		parser.Offset++
		if !pdf_is_space(current) {
			digits = append(digits, current)
		}
	}
	if parser.Offset >= len(parser.Source) {
		return value, fmt.Errorf("unterminated hex string")
	}
	parser.Offset++
	if len(digits)%2 != 0 {
		digits = append(digits, '0')
	}
	decoded := make([]byte, hex.DecodedLen(len(digits)))
	_, decode_err := hex.Decode(decoded, digits)
	if decode_err != nil {
		return value, fmt.Errorf("invalid hex string: %w", decode_err)
	}
	return Pdf_Value{Kind: PDF_VALUE_STRING, String_Bytes: decoded}, nil
}

func pdf_parse_scalar(parser *Pdf_Parser) (value Pdf_Value, err error) {
	start := parser.Offset
	for parser.Offset < len(parser.Source) && !pdf_is_delimiter(parser.Source[parser.Offset]) {
		parser.Offset++
	}
	token := string(parser.Source[start:parser.Offset])
	if token == "" {
		return value, fmt.Errorf("empty PDF token")
	}
	if token == "true" {
		return Pdf_Value{Kind: PDF_VALUE_BOOLEAN, Boolean: true}, nil
	}
	if token == "false" {
		return Pdf_Value{Kind: PDF_VALUE_BOOLEAN, Boolean: token == "true"}, nil
	}
	if token == "null" {
		return Pdf_Value{Kind: PDF_VALUE_NULL}, nil
	}
	number, integer, number_ok := pdf_parse_number(token)
	if !number_ok {
		return Pdf_Value{Kind: PDF_VALUE_KEYWORD, Name: token}, nil
	}
	value = Pdf_Value{Kind: PDF_VALUE_NUMBER, Number: number, Integer: integer}
	return pdf_parse_reference(parser, value), nil
}

func pdf_parse_reference(parser *Pdf_Parser, first Pdf_Value) (value Pdf_Value) {
	saved := parser.Offset
	pdf_skip_space_and_comments(parser)
	second_start := parser.Offset
	for parser.Offset < len(parser.Source) && !pdf_is_delimiter(parser.Source[parser.Offset]) {
		parser.Offset++
	}
	second_text := string(parser.Source[second_start:parser.Offset])
	_, generation, generation_ok := pdf_parse_number(second_text)
	if !generation_ok {
		parser.Offset = saved
		return first
	}
	pdf_skip_space_and_comments(parser)
	if !pdf_match_keyword(parser, "R") {
		parser.Offset = saved
		return first
	}
	return Pdf_Value{
		Kind: PDF_VALUE_REFERENCE, Reference_Object_Number: int(first.Integer),
		Reference_Generation: int(generation),
	}
}

func pdf_parse_number(text string) (
	number fixedpoint.Number,
	integer int64,
	valid bool,
) {
	switch text {
	case "", "+", "-", ".":
		return 0, 0, false
	}
	negative := false
	if text[0] == '-' {
		negative = true
		text = text[1:]
	} else if text[0] == '+' {
		negative = text[0] == '-'
		text = text[1:]
	}
	parts := strings.SplitN(text, ".", 2)
	whole_text := parts[0]
	if whole_text == "" {
		whole_text = "0"
	}
	whole, whole_err := strconv.ParseInt(whole_text, 10, 64)
	if whole_err != nil {
		return 0, 0, false
	}
	fraction := int64(0)
	denominator := int64(1)
	if len(parts) == 2 {
		fraction, denominator, valid = pdf_parse_fraction(parts[1])
		if !valid {
			return 0, 0, false
		}
	}
	fraction_number := fixedpoint.From_Ratio(&fixedpoint.From_Ratio_Input{
		Numerator: fraction, Denominator: denominator,
	})
	number = fixedpoint.From_Integer(whole) + fraction_number
	if negative {
		number = -number
		whole = -whole
	}
	return number, whole, true
}

func pdf_parse_fraction(text string) (numerator int64, denominator int64, valid bool) {
	if text == "" {
		return 0, 1, true
	}
	if len(text) > 18 {
		text = text[:18]
	}
	numerator, parse_err := strconv.ParseInt(text, 10, 64)
	if parse_err != nil {
		return 0, 0, false
	}
	denominator = 1
	for index := 0; index < len(text); index++ {
		denominator *= 10
	}
	return numerator, denominator, true
}

func pdf_skip_space_and_comments(parser *Pdf_Parser) {
	for parser.Offset < len(parser.Source) {
		if pdf_is_space(parser.Source[parser.Offset]) {
			parser.Offset++
			continue
		}
		if parser.Source[parser.Offset] != '%' {
			return
		}
		for parser.Offset < len(parser.Source) &&
			!pdf_is_line_end(parser.Source[parser.Offset]) {
			parser.Offset++
		}
	}
}

func pdf_match_keyword(parser *Pdf_Parser, keyword string) (matched bool) {
	if !bytes.HasPrefix(parser.Source[parser.Offset:], []byte(keyword)) {
		return false
	}
	if !pdf_keyword_boundary(&Pdf_Keyword_Boundary_Input{
		Source: parser.Source, Offset: parser.Offset, Size: len(keyword),
	}) {
		return false
	}
	parser.Offset += len(keyword)
	return true
}

func pdf_is_space(value byte) (yes bool) {
	return value == 0 || value == '\t' || value == '\n' || value == '\f' ||
		value == '\r' || value == ' '
}

func pdf_is_line_end(value byte) (yes bool) {
	return value == '\n' || value == '\r'
}

func pdf_is_delimiter(value byte) (yes bool) {
	if pdf_is_space(value) {
		return true
	}
	return bytes.ContainsRune([]byte("()<>[]{}/%"), rune(value))
}

func pdf_resolve_value(
	document *Pdf_Document,
	value Pdf_Value,
	depth int,
) (resolved Pdf_Value, err error) {
	for value.Kind == PDF_VALUE_REFERENCE {
		if depth > PDF_DEPTH_MAX {
			return resolved, fmt.Errorf("PDF reference depth exceeds %d", PDF_DEPTH_MAX)
		}
		object, exists := document.Objects[value.Reference_Object_Number]
		if !exists {
			return resolved, fmt.Errorf(
				"PDF references absent_width object %d",
				value.Reference_Object_Number,
			)
		}
		value = object.Value
		depth++
	}
	return value, nil
}

func pdf_dictionary_value(
	document *Pdf_Document,
	dictionary Pdf_Value,
	key string,
) (value Pdf_Value, exists bool, err error) {
	if dictionary.Kind != PDF_VALUE_DICTIONARY {
		return value, false, nil
	}
	entry, exists := dictionary.Dictionary[key]
	if !exists {
		return value, false, nil
	}
	resolved, resolve_err := pdf_resolve_value(document, entry, 0)
	return resolved, true, resolve_err
}

func pdf_dictionary_name(
	document *Pdf_Document,
	dictionary Pdf_Value,
	key string,
) (name string) {
	value, exists, value_err := pdf_dictionary_value(document, dictionary, key)
	if value_err != nil {
		return ""
	}
	if !exists {
		return ""
	}
	if value.Kind != PDF_VALUE_NAME {
		return ""
	}
	return value.Name
}

func pdf_decode_object_streams(document *Pdf_Document) (err error) {
	for pass_count := 0; pass_count <= len(document.Objects); pass_count++ {
		decoded_one := false
		for object_number, object := range document.Objects {
			if document.Object_Streams_Decoded[object_number] {
				continue
			}
			if pdf_dictionary_name(document, object.Value, "Type") != "ObjStm" {
				continue
			}
			document.Object_Streams_Decoded[object_number] = true
			decode_err := pdf_decode_object_stream(document, &object)
			if decode_err != nil {
				return decode_err
			}
			decoded_one = true
		}
		if !decoded_one {
			return nil
		}
	}
	return fmt.Errorf("object-stream expansion did not converge")
}

func pdf_decode_object_stream(
	document *Pdf_Document,
	object *Pdf_Indirect_Object,
) (err error) {
	count_value, count_exists, count_err := pdf_dictionary_value(
		document, object.Value, "N",
	)
	first_value, first_exists, first_err := pdf_dictionary_value(
		document, object.Value, "First",
	)
	if count_err != nil {
		return fmt.Errorf("object stream lacks N or First")
	}
	if first_err != nil {
		return fmt.Errorf("object stream lacks N or First")
	}
	if !count_exists {
		return fmt.Errorf("object stream lacks N or First")
	}
	if !first_exists {
		return fmt.Errorf("object stream lacks N or First")
	}
	if count_value.Kind != PDF_VALUE_NUMBER {
		return fmt.Errorf("object stream N or First is not a number")
	}
	if first_value.Kind != PDF_VALUE_NUMBER {
		return fmt.Errorf("object stream N or First is not a number")
	}
	if count_value.Integer < 0 {
		return fmt.Errorf("invalid object stream count")
	}
	if count_value.Integer > PDF_OBJECT_COUNT_MAX {
		return fmt.Errorf("invalid object stream count")
	}
	decoded, decode_err := pdf_decode_stream(document, object.Value)
	if decode_err != nil {
		return decode_err
	}
	return pdf_parse_object_stream_entries(&Pdf_Parse_Object_Stream_Entries_Input{
		Document: document, Container: object, Decoded: decoded,
		Count: int(count_value.Integer), First: int(first_value.Integer),
	})
}

// Pdf_Parse_Object_Stream_Entries_Input describes one decoded object stream.
type Pdf_Parse_Object_Stream_Entries_Input struct {
	// Document owns the parsed objects.
	Document *Pdf_Document
	// Container supplies update-order precedence.
	Container *Pdf_Indirect_Object
	// Decoded contains the object index and values.
	Decoded []byte
	// Count is the declared number of contained objects.
	Count int
	// First locates the first contained value.
	First int
}

func pdf_parse_object_stream_entries(
	input *Pdf_Parse_Object_Stream_Entries_Input,
) (err error) {
	if input.First < 0 {
		return fmt.Errorf("invalid object stream First")
	}
	if input.First > len(input.Decoded) {
		return fmt.Errorf("invalid object stream First")
	}
	parser := &Pdf_Parser{Source: input.Decoded}
	object_numbers := make([]int, input.Count)
	offsets := make([]int, input.Count)
	for index := 0; index < input.Count; index++ {
		object_number, number_err := pdf_parse_required_integer(parser)
		object_offset, offset_err := pdf_parse_required_integer(parser)
		if number_err != nil {
			return fmt.Errorf("invalid object stream header")
		}
		if offset_err != nil {
			return fmt.Errorf("invalid object stream header")
		}
		object_numbers[index] = object_number
		offsets[index] = object_offset
	}
	for index := 0; index < input.Count; index++ {
		parse_err := pdf_parse_compressed_object(&Pdf_Parse_Compressed_Object_Input{
			Document: input.Document, Container: input.Container,
			Decoded: input.Decoded, First: input.First,
			Object_Number: object_numbers[index], Object_Offset: offsets[index],
		})
		if parse_err != nil {
			return parse_err
		}
	}
	return nil
}

func pdf_parse_required_integer(parser *Pdf_Parser) (integer int, err error) {
	value, parse_err := pdf_parse_value(parser, 0)
	if parse_err != nil {
		return 0, parse_err
	}
	if value.Kind != PDF_VALUE_NUMBER {
		return 0, fmt.Errorf("expected integer")
	}
	if !fixedpoint.Is_Integer(value.Number) {
		return 0, fmt.Errorf("expected integer")
	}
	return int(value.Integer), nil
}

// Pdf_Parse_Compressed_Object_Input identifies one object-stream member.
type Pdf_Parse_Compressed_Object_Input struct {
	// Document owns the parsed object.
	Document *Pdf_Document
	// Container supplies update-order precedence.
	Container *Pdf_Indirect_Object
	// Decoded contains the member syntax.
	Decoded []byte
	// First locates the object-stream value area.
	First int
	// Object_Number is the indirect object identifier.
	Object_Number int
	// Object_Offset is relative to First.
	Object_Offset int
}

func pdf_parse_compressed_object(input *Pdf_Parse_Compressed_Object_Input) (err error) {
	value_offset := input.First + input.Object_Offset
	if input.Object_Number < 0 {
		return fmt.Errorf("invalid compressed object offset")
	}
	if value_offset < input.First {
		return fmt.Errorf("invalid compressed object offset")
	}
	if value_offset >= len(input.Decoded) {
		return fmt.Errorf("invalid compressed object offset")
	}
	parser := &Pdf_Parser{Source: input.Decoded, Offset: value_offset}
	value, parse_err := pdf_parse_value(parser, 0)
	if parse_err != nil {
		return fmt.Errorf("compressed object %d: %w", input.Object_Number, parse_err)
	}
	input.Document.Object_Headers_Count++
	if input.Document.Object_Headers_Count > PDF_OBJECT_COUNT_MAX {
		return fmt.Errorf("PDF exceeds %d indirect objects", PDF_OBJECT_COUNT_MAX)
	}
	prior, exists := input.Document.Objects[input.Object_Number]
	if exists {
		if prior.Offset > input.Container.Offset {
			return nil
		}
	}
	input.Document.Objects[input.Object_Number] = Pdf_Indirect_Object{
		Value: value, Generation: 0, Offset: input.Container.Offset,
	}
	return nil
}

func pdf_decode_stream(
	document *Pdf_Document,
	stream Pdf_Value,
) (decoded []byte, err error) {
	decoded = stream.Stream
	filters, parameters, filters_err := pdf_stream_filters(document, stream)
	if filters_err != nil {
		return nil, filters_err
	}
	for index := 0; index < len(filters); index++ {
		parameter := Pdf_Value{Kind: PDF_VALUE_NULL}
		if index < len(parameters) {
			parameter = parameters[index]
		}
		decoded, err = pdf_apply_filter(filters[index], decoded, parameter)
		if err != nil {
			return nil, err
		}
	}
	document.Decoded_Bytes_Count += int64(len(decoded))
	if document.Decoded_Bytes_Count > PDF_DECODED_BYTES_MAX {
		return nil, fmt.Errorf("PDF decoded data exceeds 256 MiB")
	}
	return decoded, nil
}

func pdf_stream_filters(
	document *Pdf_Document,
	stream Pdf_Value,
) (filters []string, parameters []Pdf_Value, err error) {
	filter, exists, filter_err := pdf_dictionary_value(document, stream, "Filter")
	if filter_err != nil {
		return nil, nil, filter_err
	}
	if exists {
		filters, err = pdf_filter_names(document, filter)
		if err != nil {
			return nil, nil, err
		}
	}
	parameter, parameter_exists, parameter_err := pdf_dictionary_value(
		document, stream, "DecodeParms",
	)
	if parameter_err != nil {
		return nil, nil, parameter_err
	}
	if parameter_exists {
		parameters, err = pdf_filter_parameters(document, parameter)
	}
	return filters, parameters, err
}

func pdf_filter_names(
	document *Pdf_Document,
	value Pdf_Value,
) (filters []string, err error) {
	if value.Kind == PDF_VALUE_NAME {
		return []string{value.Name}, nil
	}
	if value.Kind != PDF_VALUE_ARRAY {
		return nil, fmt.Errorf("PDF Filter is not a name or array")
	}
	for _, entry := range value.Array {
		resolved, resolve_err := pdf_resolve_value(document, entry, 0)
		if resolve_err != nil {
			return nil, fmt.Errorf("PDF filter array contains a non-name")
		}
		if resolved.Kind != PDF_VALUE_NAME {
			return nil, fmt.Errorf("PDF filter array contains a non-name")
		}
		filters = append(filters, resolved.Name)
	}
	return filters, nil
}

func pdf_filter_parameters(
	document *Pdf_Document,
	value Pdf_Value,
) (parameters []Pdf_Value, err error) {
	if value.Kind != PDF_VALUE_ARRAY {
		return []Pdf_Value{value}, nil
	}
	for _, entry := range value.Array {
		resolved, resolve_err := pdf_resolve_value(document, entry, 0)
		if resolve_err != nil {
			return nil, resolve_err
		}
		parameters = append(parameters, resolved)
	}
	return parameters, nil
}

func pdf_apply_filter(
	filter string,
	encoded []byte,
	parameters Pdf_Value,
) (decoded []byte, err error) {
	switch filter {
	case "ASCIIHexDecode", "AHx":
		decoded, err = pdf_decode_ascii_hexadecimal(encoded)
	case "ASCII85Decode", "A85":
		decoded, err = pdf_decode_ascii85(encoded)
	case "FlateDecode", "Fl":
		decoded, err = pdf_decode_flate(encoded)
	case "LZWDecode", "LZW":
		early_change := pdf_parameter_integer(parameters, "EarlyChange", 1)
		decoded, err = pdf_decode_lzw(encoded, early_change)
	case "RunLengthDecode", "RL":
		decoded, err = pdf_decode_run_size(encoded)
	default:
		return nil, fmt.Errorf("unsupported PDF stream filter %s", filter)
	}
	if err != nil {
		return nil, err
	}
	return pdf_apply_predictor(decoded, parameters)
}

func pdf_decode_ascii_hexadecimal(encoded []byte) (decoded []byte, err error) {
	digits := make([]byte, 0, len(encoded))
	for _, value := range encoded {
		if value == '>' {
			break
		}
		if !pdf_is_space(value) {
			digits = append(digits, value)
		}
	}
	if len(digits)%2 != 0 {
		digits = append(digits, '0')
	}
	if len(digits)/2 > PDF_STREAM_BYTES_MAX {
		return nil, fmt.Errorf("decoded PDF stream exceeds 64 MiB")
	}
	decoded = make([]byte, hex.DecodedLen(len(digits)))
	_, decode_err := hex.Decode(decoded, digits)
	if decode_err != nil {
		return nil, fmt.Errorf("invalid ASCIIHex Stream: %w", decode_err)
	}
	return decoded, nil
}

func pdf_decode_ascii85(encoded []byte) (decoded []byte, err error) {
	compact := make([]byte, 0, len(encoded))
	for _, value := range encoded {
		if !pdf_is_space(value) {
			compact = append(compact, value)
		}
	}
	if bytes.HasPrefix(compact, []byte("<~")) {
		compact = compact[2:]
	}
	if bytes.HasSuffix(compact, []byte("~>")) {
		compact = compact[:len(compact)-2]
	}
	decoded_size := len(compact)*4/5 + 4
	if decoded_size > PDF_STREAM_BYTES_MAX {
		return nil, fmt.Errorf("decoded PDF stream exceeds 64 MiB")
	}
	decoded = make([]byte, decoded_size)
	count, _, decode_err := ascii85.Decode(decoded, compact, true)
	if decode_err != nil {
		return nil, fmt.Errorf("invalid ASCII85 Stream: %w", decode_err)
	}
	return decoded[:count], nil
}

func pdf_decode_flate(encoded []byte) (decoded []byte, err error) {
	reader, new_err := bounded_zlib.New_Reader(
		bytes.NewReader(encoded), PDF_STREAM_BYTES_MAX,
	)
	if new_err != nil {
		return nil, fmt.Errorf("invalid Flate Stream: %w", new_err)
	}
	defer reader.Close()
	return pdf_read_bounded(reader, PDF_STREAM_BYTES_MAX)
}

func pdf_read_bounded(reader io.Reader, bytes_max int) (contents []byte, err error) {
	contents = make([]byte, 0, 32768)
	var buffer [32768]byte
	for read_count := 0; read_count <= bytes_max+1; read_count++ {
		count, read_err := reader.Read(buffer[:])
		if len(contents)+count > bytes_max {
			return nil, fmt.Errorf("decoded PDF stream exceeds 64 MiB")
		}
		contents = append(contents, buffer[:count]...)
		if read_err == io.EOF {
			return contents, nil
		}
		if read_err != nil {
			return nil, read_err
		}
	}
	return nil, fmt.Errorf("decoded PDF stream made no bounded progress")
}

func pdf_decode_run_size(encoded []byte) (decoded []byte, err error) {
	decoded = make([]byte, 0, len(encoded))
	for offset := 0; offset < len(encoded); {
		run_size := int(encoded[offset])
		offset++
		if run_size == 128 {
			return decoded, nil
		}
		if run_size < 128 {
			literal_size := run_size + 1
			if offset+literal_size > len(encoded) {
				return nil, fmt.Errorf("truncated RunLength stream")
			}
			decoded = append(decoded, encoded[offset:offset+literal_size]...)
			offset += literal_size
		} else {
			if offset >= len(encoded) {
				return nil, fmt.Errorf("truncated RunLength stream")
			}
			count := 257 - run_size
			for index := 0; index < count; index++ {
				decoded = append(decoded, encoded[offset])
			}
			offset++
		}
		if len(decoded) > PDF_STREAM_BYTES_MAX {
			return nil, fmt.Errorf("decoded PDF stream exceeds 64 MiB")
		}
	}
	return nil, fmt.Errorf("RunLength stream lacks end marker")
}

// Pdf_Bit_Reader reads the variable-size, most-significant-bit-first codes
// required by PDF LZW streams.
type Pdf_Bit_Reader struct {
	// Source is the encoded LZW byte sequence.
	Source []byte
	// Bit_Offset is the next unread bit.
	Bit_Offset int
}

func pdf_read_bits(reader *Pdf_Bit_Reader, width int) (code int, ok bool) {
	if reader.Bit_Offset+width > len(reader.Source)*8 {
		return 0, false
	}
	for index := 0; index < width; index++ {
		absolute := reader.Bit_Offset + index
		value := reader.Source[absolute/8]
		code = code<<1 | int((value>>uint(7-absolute%8))&1)
	}
	reader.Bit_Offset += width
	return code, true
}

func pdf_decode_lzw(encoded []byte, early_change int64) (decoded []byte, err error) {
	reader := &Pdf_Bit_Reader{Source: encoded}
	dictionary := pdf_lzw_dictionary()
	width := 9
	next_code := 258
	var prior []byte
	for reader.Bit_Offset < len(encoded)*8 {
		code, code_ok := pdf_read_bits(reader, width)
		if !code_ok {
			return nil, fmt.Errorf("truncated LZW stream")
		}
		if code == 256 {
			dictionary = pdf_lzw_dictionary()
			width = 9
			next_code = 258
			prior = nil
			continue
		}
		if code == 257 {
			return decoded, nil
		}
		entry, entry_err := pdf_lzw_entry(&Pdf_Lzw_Entry_Input{
			Dictionary: dictionary, Code: code, Next_Code: next_code, Prior: prior,
		})
		if entry_err != nil {
			return nil, entry_err
		}
		if len(decoded)+len(entry) > PDF_STREAM_BYTES_MAX {
			return nil, fmt.Errorf("decoded PDF stream exceeds 64 MiB")
		}
		decoded = append(decoded, entry...)
		if prior != nil {
			if next_code < 4096 {
				dictionary[next_code] = append(append([]byte{}, prior...), entry[0])
				next_code++
				if width < 12 {
					if int64(next_code)+early_change == int64(1<<width) {
						width++
					}
				}
			}
		}
		prior = entry
	}
	return nil, fmt.Errorf("truncated LZW stream")
}

func pdf_lzw_dictionary() (dictionary [][]byte) {
	dictionary = make([][]byte, 4096)
	for value_index := 0; value_index < 256; value_index++ {
		dictionary[value_index] = []byte{byte(value_index)}
	}
	return dictionary
}

// Pdf_Lzw_Entry_Input groups the state needed to resolve one LZW code.
type Pdf_Lzw_Entry_Input struct {
	// Dictionary contains the current code entries.
	Dictionary [][]byte
	// Code is the encoded dictionary index.
	Code int
	// Next_Code is the first unassigned index.
	Next_Code int
	// Prior is needed for the special next-code case.
	Prior []byte
}

func pdf_lzw_entry(input *Pdf_Lzw_Entry_Input) (entry []byte, err error) {
	if input.Code >= 0 {
		if input.Code < input.Next_Code {
			if input.Dictionary[input.Code] != nil {
				return input.Dictionary[input.Code], nil
			}
		}
	}
	if input.Code == input.Next_Code {
		if input.Prior != nil {
			return append(append([]byte{}, input.Prior...), input.Prior[0]), nil
		}
	}
	return nil, fmt.Errorf("invalid LZW code %d", input.Code)
}

func pdf_parameter_integer(parameters Pdf_Value, key string, fallback int64) (value int64) {
	if parameters.Kind != PDF_VALUE_DICTIONARY {
		return fallback
	}
	entry, exists := parameters.Dictionary[key]
	if !exists {
		return fallback
	}
	if entry.Kind != PDF_VALUE_NUMBER {
		return fallback
	}
	return entry.Integer
}

func pdf_apply_predictor(encoded []byte, parameters Pdf_Value) (decoded []byte, err error) {
	predictor := pdf_parameter_integer(parameters, "Predictor", 1)
	if predictor == 1 {
		return encoded, nil
	}
	colors := pdf_parameter_integer(parameters, "Colors", 1)
	bits := pdf_parameter_integer(parameters, "BitsPerComponent", 8)
	columns := pdf_parameter_integer(parameters, "Columns", 1)
	if colors < 1 {
		return nil, fmt.Errorf("invalid PDF predictor parameters")
	}
	if bits < 1 {
		return nil, fmt.Errorf("invalid PDF predictor parameters")
	}
	if columns < 1 {
		return nil, fmt.Errorf("invalid PDF predictor parameters")
	}
	row_bytes_size := int((colors*columns*bits + 7) / 8)
	bytes_per_pixel := int((colors*bits + 7) / 8)
	if predictor == 2 {
		return pdf_apply_tiff_predictor(&Pdf_Apply_Tiff_Predictor_Input{
			Encoded: encoded, Row_Bytes_Size: row_bytes_size,
			Bytes_Per_Pixel: bytes_per_pixel,
		})
	}
	if predictor >= 10 {
		if predictor <= 15 {
			return pdf_apply_png_predictor(&Pdf_Apply_Png_Predictor_Input{
				Encoded: encoded, Row_Bytes_Size: row_bytes_size,
				Bytes_Per_Pixel: bytes_per_pixel,
			})
		}
	}
	return nil, fmt.Errorf("unsupported PDF predictor %d", predictor)
}

// Pdf_Apply_Tiff_Predictor_Input describes TIFF predictor row geometry.
type Pdf_Apply_Tiff_Predictor_Input struct {
	// Encoded contains predictor deltas.
	Encoded []byte
	// Row_Bytes_Size is the decoded row length.
	Row_Bytes_Size int
	// Bytes_Per_Pixel locates each left neighbor.
	Bytes_Per_Pixel int
}

func pdf_apply_tiff_predictor(
	input *Pdf_Apply_Tiff_Predictor_Input,
) (decoded []byte, err error) {
	if input.Row_Bytes_Size == 0 {
		return nil, fmt.Errorf("invalid TIFF predictor rows")
	}
	if len(input.Encoded)%input.Row_Bytes_Size != 0 {
		return nil, fmt.Errorf("invalid TIFF predictor rows")
	}
	decoded = append([]byte{}, input.Encoded...)
	for row_start := 0; row_start < len(decoded); row_start += input.Row_Bytes_Size {
		for column := input.Bytes_Per_Pixel; column < input.Row_Bytes_Size; column++ {
			index := row_start + column
			decoded[index] += decoded[index-input.Bytes_Per_Pixel]
		}
	}
	return decoded, nil
}

// Pdf_Apply_Png_Predictor_Input describes PNG predictor row geometry.
type Pdf_Apply_Png_Predictor_Input struct {
	// Encoded contains filter bytes and predictor deltas.
	Encoded []byte
	// Row_Bytes_Size is the decoded row length.
	Row_Bytes_Size int
	// Bytes_Per_Pixel locates each left neighbor.
	Bytes_Per_Pixel int
}

func pdf_apply_png_predictor(
	input *Pdf_Apply_Png_Predictor_Input,
) (decoded []byte, err error) {
	encoded_row_bytes := input.Row_Bytes_Size + 1
	if input.Row_Bytes_Size == 0 {
		return nil, fmt.Errorf("invalid PNG predictor rows")
	}
	if len(input.Encoded)%encoded_row_bytes != 0 {
		return nil, fmt.Errorf("invalid PNG predictor rows")
	}
	decoded = make([]byte, 0, len(input.Encoded))
	prior := make([]byte, input.Row_Bytes_Size)
	for row_start := 0; row_start < len(input.Encoded); row_start += encoded_row_bytes {
		filter := input.Encoded[row_start]
		row := append([]byte{}, input.Encoded[row_start+1:row_start+encoded_row_bytes]...)
		filter_err := pdf_apply_png_row(&Pdf_Apply_Png_Row_Input{
			Filter: filter, Row: row, Prior: prior,
			Bytes_Per_Pixel: input.Bytes_Per_Pixel,
		})
		if filter_err != nil {
			return nil, filter_err
		}
		decoded = append(decoded, row...)
		prior = row
	}
	return decoded, nil
}

// Pdf_Apply_Png_Row_Input contains one PNG predictor row and its neighbors.
type Pdf_Apply_Png_Row_Input struct {
	// Filter selects the PNG reconstruction rule.
	Filter byte
	// Row is reconstructed in place.
	Row []byte
	// Prior is the preceding decoded row.
	Prior []byte
	// Bytes_Per_Pixel locates each left neighbor.
	Bytes_Per_Pixel int
}

func pdf_apply_png_row(input *Pdf_Apply_Png_Row_Input) (err error) {
	for index := 0; index < len(input.Row); index++ {
		left := byte(0)
		if index >= input.Bytes_Per_Pixel {
			left = input.Row[index-input.Bytes_Per_Pixel]
		}
		up := input.Prior[index]
		upper_left := byte(0)
		if index >= input.Bytes_Per_Pixel {
			upper_left = input.Prior[index-input.Bytes_Per_Pixel]
		}
		switch input.Filter {
		case 0:
		case 1:
			input.Row[index] += left
		case 2:
			input.Row[index] += up
		case 3:
			input.Row[index] += byte((int(left) + int(up)) / 2)
		case 4:
			input.Row[index] += pdf_paeth(&Pdf_Paeth_Input{
				Left: left, Up: up, Upper_Left: upper_left,
			})
		default:
			return fmt.Errorf("invalid PNG predictor filter %d", input.Filter)
		}
	}
	return nil
}

// Pdf_Paeth_Input groups the three neighboring PNG predictor bytes.
type Pdf_Paeth_Input struct {
	// Left is the prior byte in this row.
	Left byte
	// Up is the corresponding byte in the prior row.
	Up byte
	// Upper_Left is the diagonal byte in the prior row.
	Upper_Left byte
}

func pdf_paeth(input *Pdf_Paeth_Input) (predictor byte) {
	estimate := int(input.Left) + int(input.Up) - int(input.Upper_Left)
	left_distance := pdf_absolute_integer(estimate - int(input.Left))
	up_distance := pdf_absolute_integer(estimate - int(input.Up))
	upper_left_distance := pdf_absolute_integer(estimate - int(input.Upper_Left))
	if left_distance <= up_distance {
		if left_distance <= upper_left_distance {
			return input.Left
		}
	}
	if up_distance <= upper_left_distance {
		return input.Up
	}
	return input.Upper_Left
}

func pdf_absolute_integer(value int) (absolute int) {
	if value < 0 {
		return -value
	}
	return value
}

// Pdf_Page holds only the resources and streams needed for one page.
type Pdf_Page struct {
	// Resources contains the inherited font and XObject dictionaries.
	Resources Pdf_Value
	// Contents retains ordered encoded content streams.
	Contents []Pdf_Value
	// Width is required by the form-density classifier.
	Width fixedpoint.Number
	// Height converts bottom-origin positions into top-origin rows.
	Height fixedpoint.Number
}

// Pdf_Page_Inheritance carries entries inherited by page-tree leaves.
type Pdf_Page_Inheritance struct {
	// Resources is the nearest ancestor resource dictionary.
	Resources Pdf_Value
	// Media_Box is the nearest ancestor page boundary.
	Media_Box Pdf_Value
}

// Pdf_Page_Candidate keeps both paths until form classification is complete.
type Pdf_Page_Candidate struct {
	// Form_Content is the position-aware Markdown candidate.
	Form_Content string
	// Plain_Content is the prose fallback for this page.
	Plain_Content string
	// Is_Form selects Form_Content in mixed documents.
	Is_Form bool
}

func pdf_extract_document(document *Pdf_Document) (markdown []byte, err error) {
	pages, pages_err := pdf_document_pages(document)
	if pages_err != nil {
		return nil, pages_err
	}
	candidates := make([]Pdf_Page_Candidate, 0, len(pages))
	form_page_count := 0
	for page_index := 0; page_index < len(pages); page_index++ {
		candidate, page_err := pdf_extract_page(document, &pages[page_index])
		if page_err != nil {
			return nil, fmt.Errorf("PDF page %d: %w", page_index+1, page_err)
		}
		if candidate.Is_Form {
			form_page_count++
		}
		candidates = append(candidates, candidate)
	}
	text := pdf_select_page_candidates(candidates, form_page_count)
	text = pdf_merge_partial_numbers(text)
	if len(text) > PDF_BYTES_MAX {
		return nil, fmt.Errorf("Markdown output exceeds 64 MiB")
	}
	return []byte(text), nil
}

func pdf_document_pages(document *Pdf_Document) (pages []Pdf_Page, err error) {
	catalog, catalog_err := pdf_document_catalog(document)
	if catalog_err != nil {
		return nil, catalog_err
	}
	pages_value, exists, pages_err := pdf_dictionary_value(document, catalog, "Pages")
	if pages_err != nil {
		return nil, fmt.Errorf("PDF catalog has no Pages tree")
	}
	if !exists {
		return nil, fmt.Errorf("PDF catalog has no Pages tree")
	}
	return pdf_collect_pages(document, pages_value, Pdf_Page_Inheritance{}, 0, pages)
}

func pdf_document_catalog(document *Pdf_Document) (catalog Pdf_Value, err error) {
	best_offset := -1
	for _, object := range document.Objects {
		if pdf_dictionary_name(document, object.Value, "Type") != "Catalog" {
			continue
		}
		if object.Offset > best_offset {
			catalog = object.Value
			best_offset = object.Offset
		}
	}
	if best_offset < 0 {
		return catalog, fmt.Errorf("PDF has no catalog")
	}
	return catalog, nil
}

// Pdf_Page_Node retains one pending page-tree node and its inherited values.
type Pdf_Page_Node struct {
	// Value is the unresolved page-tree node.
	Value Pdf_Value
	// Inherited contains the nearest ancestor resources and media box.
	Inherited Pdf_Page_Inheritance
	// Depth bounds malformed page trees.
	Depth int
}

func pdf_collect_pages(
	document *Pdf_Document,
	node Pdf_Value,
	inherited Pdf_Page_Inheritance,
	depth int,
	pages []Pdf_Page,
) (collected []Pdf_Page, err error) {
	stack := []Pdf_Page_Node{{Value: node, Inherited: inherited, Depth: depth}}
	collected = pages
	for len(stack) != 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if current.Depth > PDF_DEPTH_MAX {
			return nil, fmt.Errorf("PDF page-tree depth exceeds %d", PDF_DEPTH_MAX)
		}
		resolved, resolve_err := pdf_resolve_value(document, current.Value, 0)
		if resolve_err != nil {
			return nil, fmt.Errorf("invalid PDF page-tree node")
		}
		if resolved.Kind != PDF_VALUE_DICTIONARY {
			return nil, fmt.Errorf("invalid PDF page-tree node")
		}
		inheritance, inheritance_err := pdf_page_inherit(
			document, resolved, current.Inherited,
		)
		if inheritance_err != nil {
			return nil, inheritance_err
		}
		if pdf_dictionary_name(document, resolved, "Type") == "Page" {
			page, page_err := pdf_build_page(document, resolved, inheritance)
			if page_err != nil {
				return nil, page_err
			}
			collected = append(collected, page)
			if len(collected) > PDF_PAGE_COUNT_MAX {
				return nil, fmt.Errorf("PDF exceeds %d pages", PDF_PAGE_COUNT_MAX)
			}
			continue
		}
		kids, exists, kids_err := pdf_dictionary_value(document, resolved, "Kids")
		if kids_err != nil {
			return nil, fmt.Errorf("PDF Pages node has no Kids array")
		}
		if !exists {
			return nil, fmt.Errorf("PDF Pages node has no Kids array")
		}
		if kids.Kind != PDF_VALUE_ARRAY {
			return nil, fmt.Errorf("PDF Pages node has no Kids array")
		}
		for kid_index := len(kids.Array) - 1; kid_index >= 0; kid_index-- {
			stack = append(stack, Pdf_Page_Node{
				Value: kids.Array[kid_index], Inherited: inheritance,
				Depth: current.Depth + 1,
			})
		}
	}
	return collected, nil
}

func pdf_page_inherit(
	document *Pdf_Document,
	node Pdf_Value,
	inherited Pdf_Page_Inheritance,
) (result Pdf_Page_Inheritance, err error) {
	result = inherited
	resources, resources_exists, resources_err := pdf_dictionary_value(
		document, node, "Resources",
	)
	if resources_err != nil {
		return result, resources_err
	}
	if resources_exists {
		result.Resources = resources
	}
	media_box, media_exists, media_err := pdf_dictionary_value(document, node, "MediaBox")
	if media_err != nil {
		return result, media_err
	}
	if media_exists {
		result.Media_Box = media_box
	}
	return result, nil
}

func pdf_build_page(
	document *Pdf_Document,
	node Pdf_Value,
	inherited Pdf_Page_Inheritance,
) (page Pdf_Page, err error) {
	page.Resources = inherited.Resources
	page.Width, page.Height = pdf_page_dimensions(inherited.Media_Box)
	contents, exists, contents_err := pdf_dictionary_value(document, node, "Contents")
	if contents_err != nil {
		return page, contents_err
	}
	if !exists {
		return page, nil
	}
	if contents.Kind == PDF_VALUE_ARRAY {
		for _, entry := range contents.Array {
			resolved, resolve_err := pdf_resolve_value(document, entry, 0)
			if resolve_err != nil {
				return page, resolve_err
			}
			page.Contents = append(page.Contents, resolved)
		}
		return page, nil
	}
	page.Contents = append(page.Contents, contents)
	return page, nil
}

func pdf_page_dimensions(media_box Pdf_Value) (
	width fixedpoint.Number,
	height fixedpoint.Number,
) {
	width = fixedpoint.From_Integer(612)
	height = fixedpoint.From_Integer(792)
	if media_box.Kind != PDF_VALUE_ARRAY {
		return width, height
	}
	if len(media_box.Array) < 4 {
		return width, height
	}
	left, left_ok := pdf_value_number(media_box.Array[0])
	bottom, bottom_ok := pdf_value_number(media_box.Array[1])
	right, right_ok := pdf_value_number(media_box.Array[2])
	top, top_ok := pdf_value_number(media_box.Array[3])
	if !left_ok {
		return width, height
	}
	if !bottom_ok {
		return width, height
	}
	if !right_ok {
		return width, height
	}
	if !top_ok {
		return width, height
	}
	return right - left, top - bottom
}

func pdf_value_number(value Pdf_Value) (number fixedpoint.Number, ok bool) {
	if value.Kind != PDF_VALUE_NUMBER {
		return 0, false
	}
	return value.Number, true
}

func pdf_select_page_candidates(candidates []Pdf_Page_Candidate, form_count int) (text string) {
	chunks := make([]string, 0, len(candidates))
	if form_count == 0 {
		for _, candidate := range candidates {
			if strings.TrimSpace(candidate.Plain_Content) != "" {
				chunks = append(chunks, candidate.Plain_Content)
			}
		}
		if len(chunks) == 0 {
			return ""
		}
		return strings.Join(chunks, "\n\n") + "\n\n"
	}
	for _, candidate := range candidates {
		content := candidate.Plain_Content
		if candidate.Is_Form {
			content = candidate.Form_Content
		}
		if strings.TrimSpace(content) != "" {
			chunks = append(chunks, strings.TrimSpace(content))
		}
	}
	return strings.TrimSpace(strings.Join(chunks, "\n\n"))
}

func pdf_merge_partial_numbers(text string) (merged string) {
	lines := strings.Split(text, "\n")
	result := make([]string, 0, len(lines))
	for index := 0; index < len(lines); {
		stripped := strings.TrimSpace(lines[index])
		if !pdf_is_partial_number(stripped) {
			result = append(result, lines[index])
			index++
			continue
		}
		next := index + 1
		for next < len(lines) && strings.TrimSpace(lines[next]) == "" {
			next++
		}
		if next >= len(lines) {
			result = append(result, lines[index])
			index++
			continue
		}
		result = append(result, stripped+" "+strings.TrimSpace(lines[next]))
		index = next + 1
	}
	return strings.Join(result, "\n")
}

func pdf_is_partial_number(text string) (yes bool) {
	if len(text) < 2 {
		return false
	}
	if text[0] != '.' {
		return false
	}
	for index := 1; index < len(text); index++ {
		if text[index] < '0' {
			return false
		}
		if text[index] > '9' {
			return false
		}
	}
	return true
}

// Pdf_Font is the decoded information required for text and positions.
type Pdf_Font struct {
	// Subtype selects simple-font or Type 0 decoding rules.
	Subtype string
	// Base_Encoding maps one-byte codes when ToUnicode has no entry.
	Base_Encoding [256]string
	// Unicode_Map maps source-code byte strings to Unicode.
	Unicode_Map map[string]string
	// Code_Sizes permits greedy decoding of variable-size CMap codes.
	Code_Sizes []int
	// Widths advances glyph positions in thousandths of an em.
	Widths map[int]int64
	// Default_Width covers codes omitted from Widths.
	Default_Width int64
	// Type_Zero selects multibyte CID fallback behavior.
	Type_Zero bool
}

// Pdf_Glyph pairs decoded text with its source-code metrics.
type Pdf_Glyph struct {
	// Text is the Unicode text emitted for the source code.
	Text string
	// Code selects a width from the font metric table.
	Code int
	// Width is the advance in thousandths of an em.
	Width int64
	// Is_Space selects the PDF word-spacing adjustment.
	Is_Space bool
}

func pdf_font_from_resources(
	document *Pdf_Document,
	resources Pdf_Value,
	font_name string,
) (font Pdf_Font, err error) {
	fonts, exists, fonts_err := pdf_dictionary_value(document, resources, "Font")
	if fonts_err != nil {
		return font, fmt.Errorf("text uses missing Font resources")
	}
	if !exists {
		return font, fmt.Errorf("text uses missing Font resources")
	}
	if fonts.Kind != PDF_VALUE_DICTIONARY {
		return font, fmt.Errorf("text uses absent_width Font resources")
	}
	font_value, exists := fonts.Dictionary[font_name]
	if !exists {
		return font, fmt.Errorf("text uses absent_width font %s", font_name)
	}
	resolved, resolve_err := pdf_resolve_value(document, font_value, 0)
	if resolve_err != nil {
		return font, fmt.Errorf("invalid font %s", font_name)
	}
	if resolved.Kind != PDF_VALUE_DICTIONARY {
		return font, fmt.Errorf("invalid font %s", font_name)
	}
	return pdf_build_font(document, resolved)
}

func pdf_build_font(document *Pdf_Document, value Pdf_Value) (font Pdf_Font, err error) {
	font.Subtype = pdf_dictionary_name(document, value, "Subtype")
	font.Type_Zero = font.Subtype == "Type0"
	font.Default_Width = 0
	if font.Type_Zero {
		font.Default_Width = 1000
	}
	font.Widths = make(map[int]int64)
	font.Unicode_Map = make(map[string]string)
	font.Base_Encoding = pdf_base_encoding("StandardEncoding")
	if encoding_err := pdf_font_encoding(document, value, &font); encoding_err != nil {
		return font, encoding_err
	}
	if widths_err := pdf_font_widths(document, value, &font); widths_err != nil {
		return font, widths_err
	}
	if unicode_err := pdf_font_unicode_map(document, value, &font); unicode_err != nil {
		return font, unicode_err
	}
	if len(font.Code_Sizes) == 0 {
		if font.Type_Zero {
			font.Code_Sizes = []int{2}
		} else {
			font.Code_Sizes = []int{1}
		}
	}
	return font, nil
}

func pdf_font_encoding(
	document *Pdf_Document,
	value Pdf_Value,
	font *Pdf_Font,
) (err error) {
	encoding, exists, encoding_err := pdf_dictionary_value(document, value, "Encoding")
	if encoding_err != nil {
		return encoding_err
	}
	if !exists {
		return encoding_err
	}
	if encoding.Kind == PDF_VALUE_NAME {
		font.Base_Encoding = pdf_base_encoding(encoding.Name)
		return nil
	}
	if encoding.Kind != PDF_VALUE_DICTIONARY {
		return nil
	}
	base := pdf_dictionary_name(document, encoding, "BaseEncoding")
	if base != "" {
		font.Base_Encoding = pdf_base_encoding(base)
	}
	differences, differences_exists, differences_err := pdf_dictionary_value(
		document, encoding, "Differences",
	)
	if differences_err != nil {
		return differences_err
	}
	if !differences_exists {
		return differences_err
	}
	pdf_apply_encoding_differences(&font.Base_Encoding, differences)
	return nil
}

func pdf_apply_encoding_differences(encoding *[256]string, differences Pdf_Value) {
	if differences.Kind != PDF_VALUE_ARRAY {
		return
	}
	code := -1
	for _, entry := range differences.Array {
		if entry.Kind == PDF_VALUE_NUMBER {
			code = int(entry.Integer)
			continue
		}
		if entry.Kind != PDF_VALUE_NAME {
			continue
		}
		if code < 0 {
			continue
		}
		if code > 255 {
			continue
		}
		encoding[code] = pdf_glyph_name_text(entry.Name)
		code++
	}
}

func pdf_base_encoding(name string) (encoding [256]string) {
	for code := 32; code <= 126; code++ {
		encoding[code] = string(rune(code))
	}
	for code := 160; code <= 255; code++ {
		encoding[code] = string(rune(code))
	}
	if name == "WinAnsiEncoding" {
		pdf_apply_windows_ansi(&encoding)
	}
	if name == "MacRomanEncoding" {
		pdf_apply_mac_roman(&encoding)
	}
	return encoding
}

func pdf_apply_windows_ansi(encoding *[256]string) {
	values := []rune{
		'€', 0, '‚', 'ƒ', '„', '…', '†', '‡', 'ˆ', '‰', 'Š', '‹', 'Œ', 0, 'Ž', 0,
		0, '‘', '’', '“', '”', '•', '–', '—', '˜', '™', 'š', '›', 'œ', 0, 'ž', 'Ÿ',
	}
	for index, value := range values {
		if value != 0 {
			encoding[128+index] = string(value)
		}
	}
}

func pdf_apply_mac_roman(encoding *[256]string) {
	values := []rune{
		'Ä', 'Å', 'Ç', 'É', 'Ñ', 'Ö', 'Ü', 'á', 'à', 'â', 'ä', 'ã', 'å', 'ç', 'é', 'è',
		'ê', 'ë', 'í', 'ì', 'î', 'ï', 'ñ', 'ó', 'ò', 'ô', 'ö', 'õ', 'ú', 'ù', 'û', 'ü',
		'†', '°', '¢', '£', '§', '•', '¶', 'ß', '®', '©', '™', '´', '¨', '≠',
		'Æ', 'Ø', '∞', '±', '≤', '≥', '¥', 'µ', '∂', '∑', '∏', 'π', '∫', 'ª',
		'º', 'Ω', 'æ', 'ø',
		'¿', '¡', '¬', '√', 'ƒ', '≈', '∆', '«', '»', '…', 0, 'À', 'Ã', 'Õ', 'Œ', 'œ',
		'–', '—', '“', '”', '‘', '’', '÷', '◊', 'ÿ', 'Ÿ', '⁄', '€', '‹', '›',
		'ﬁ', 'ﬂ', '‡', '·', '‚', '„', '‰', 'Â', 'Ê', 'Á', 'Ë', 'È', 'Í', 'Î',
		'Ï', 'Ì', 'Ó', 'Ô',
		0, 'Ò', 'Ú', 'Û', 'Ù', 'ı', 'ˆ', '˜', '¯', '˘', '˙', '˚', '¸', '˝', '˛', 'ˇ',
	}
	for index, value := range values {
		if value != 0 {
			encoding[128+index] = string(value)
		}
	}
}

func pdf_glyph_name_text(name string) (text string) {
	if len(name) == 1 {
		return name
	}
	if strings.HasPrefix(name, "uni") {
		if len(name) >= 7 {
			return pdf_unicode_name(name[3:])
		}
	}
	if strings.HasPrefix(name, "u") {
		if len(name) >= 5 {
			return pdf_unicode_name(name[1:])
		}
	}
	text = pdf_glyph_punctuation(name)
	if text != "" {
		return text
	}
	return pdf_glyph_symbol(name)
}

func pdf_glyph_punctuation(name string) (text string) {
	switch name {
	case "space":
		return " "
	case "hyphen":
		return "-"
	case "period":
		return "."
	case "comma":
		return ","
	case "colon":
		return ":"
	case "semicolon":
		return ";"
	case "slash":
		return "/"
	case "backslash":
		return "\\"
	case "parenleft":
		return "("
	case "parenright":
		return ")"
	case "bracketleft":
		return "["
	case "bracketright":
		return "]"
	case "braceleft":
		return "{"
	case "braceright":
		return "}"
	case "quotedbl":
		return "\""
	case "quotesingle", "quoteright":
		return "'"
	case "quoteleft":
		return "‘"
	case "quotedblleft":
		return "“"
	case "quotedblright":
		return "”"
	case "endash":
		return "–"
	case "emdash":
		return "—"
	case "bullet":
		return "•"
	case "ellipsis":
		return "…"
	}
	return ""
}

func pdf_glyph_symbol(name string) (text string) {
	switch name {
	case "ampersand":
		return "&"
	case "percent":
		return "%"
	case "numbersign":
		return "#"
	case "dollar":
		return "$"
	case "at":
		return "@"
	case "underscore":
		return "_"
	case "plus":
		return "+"
	case "equal":
		return "="
	case "less":
		return "<"
	case "greater":
		return ">"
	case "fi":
		return "fi"
	case "fl":
		return "fl"
	}
	return ""
}

func pdf_unicode_name(hexadecimal string) (text string) {
	if len(hexadecimal)%4 != 0 {
		return ""
	}
	var output strings.Builder
	for offset := 0; offset < len(hexadecimal); offset += 4 {
		value, parse_err := strconv.ParseUint(hexadecimal[offset:offset+4], 16, 16)
		if parse_err != nil {
			return ""
		}
		output.WriteRune(rune(value))
	}
	return output.String()
}

func pdf_font_widths(
	document *Pdf_Document,
	value Pdf_Value,
	font *Pdf_Font,
) (err error) {
	if font.Type_Zero {
		return pdf_type_zero_widths(document, value, font)
	}
	first := int64(0)
	first_value, first_exists, first_err := pdf_dictionary_value(document, value, "FirstChar")
	if first_err != nil {
		return first_err
	}
	if first_exists {
		if first_value.Kind == PDF_VALUE_NUMBER {
			first = first_value.Integer
		}
	}
	widths, widths_exists, widths_err := pdf_dictionary_value(document, value, "Widths")
	if widths_err != nil {
		return widths_err
	}
	if widths_exists {
		if widths.Kind == PDF_VALUE_ARRAY {
			for index, width := range widths.Array {
				if width.Kind == PDF_VALUE_NUMBER {
					font.Widths[int(first)+index] = width.Integer
				}
			}
		}
	}
	font.Default_Width = pdf_font_missing_width(document, value)
	return nil
}

func pdf_font_missing_width(document *Pdf_Document, value Pdf_Value) (width int64) {
	descriptor, exists, descriptor_err := pdf_dictionary_value(
		document, value, "FontDescriptor",
	)
	if descriptor_err != nil {
		return 0
	}
	if !exists {
		return 0
	}
	absent_width, absent_exists, absent_err := pdf_dictionary_value(
		document, descriptor, "MissingWidth",
	)
	if absent_err != nil {
		return 0
	}
	if !absent_exists {
		return 0
	}
	if absent_width.Kind != PDF_VALUE_NUMBER {
		return 0
	}
	return absent_width.Integer
}

func pdf_type_zero_widths(
	document *Pdf_Document,
	value Pdf_Value,
	font *Pdf_Font,
) (err error) {
	descendants, exists, descendants_err := pdf_dictionary_value(
		document, value, "DescendantFonts",
	)
	if descendants_err != nil {
		return descendants_err
	}
	if !exists {
		return nil
	}
	if descendants.Kind != PDF_VALUE_ARRAY {
		return nil
	}
	if len(descendants.Array) == 0 {
		return descendants_err
	}
	descendant, resolve_err := pdf_resolve_value(document, descendants.Array[0], 0)
	if resolve_err != nil {
		return resolve_err
	}
	default_width, default_exists, default_err := pdf_dictionary_value(
		document, descendant, "DW",
	)
	if default_err != nil {
		return default_err
	}
	if default_exists {
		if default_width.Kind == PDF_VALUE_NUMBER {
			font.Default_Width = default_width.Integer
		}
	}
	widths, widths_exists, widths_err := pdf_dictionary_value(document, descendant, "W")
	if widths_err != nil {
		return widths_err
	}
	if !widths_exists {
		return nil
	}
	if widths.Kind != PDF_VALUE_ARRAY {
		return widths_err
	}
	pdf_apply_cid_widths(font, widths.Array)
	return nil
}

func pdf_apply_cid_widths(font *Pdf_Font, entries []Pdf_Value) {
	for index := 0; index < len(entries); {
		if entries[index].Kind != PDF_VALUE_NUMBER {
			index++
			continue
		}
		if index+1 >= len(entries) {
			index++
			continue
		}
		first := int(entries[index].Integer)
		next := entries[index+1]
		if next.Kind == PDF_VALUE_ARRAY {
			for width_index, width := range next.Array {
				if width.Kind == PDF_VALUE_NUMBER {
					font.Widths[first+width_index] = width.Integer
				}
			}
			index += 2
			continue
		}
		if index+2 >= len(entries) {
			index++
			continue
		}
		if next.Kind != PDF_VALUE_NUMBER {
			index++
			continue
		}
		if entries[index+2].Kind != PDF_VALUE_NUMBER {
			index++
			continue
		}
		last := int(next.Integer)
		width := entries[index+2].Integer
		for code := first; code <= last && code-first <= 65536; code++ {
			font.Widths[code] = width
		}
		index += 3
	}
}

func pdf_font_unicode_map(
	document *Pdf_Document,
	value Pdf_Value,
	font *Pdf_Font,
) (err error) {
	to_unicode, exists, unicode_err := pdf_dictionary_value(document, value, "ToUnicode")
	if unicode_err != nil {
		return unicode_err
	}
	if !exists {
		return unicode_err
	}
	if to_unicode.Kind != PDF_VALUE_DICTIONARY {
		return nil
	}
	if to_unicode.Stream == nil {
		return nil
	}
	decoded, decode_err := pdf_decode_stream(document, to_unicode)
	if decode_err != nil {
		return decode_err
	}
	return pdf_parse_unicode_cmap(decoded, font)
}

func pdf_parse_unicode_cmap(source []byte, font *Pdf_Font) (err error) {
	tokens, tokens_err := pdf_cmap_tokens(source)
	if tokens_err != nil {
		return tokens_err
	}
	for index := 0; index < len(tokens); index++ {
		if tokens[index].Kind != PDF_VALUE_KEYWORD {
			continue
		}
		switch tokens[index].Name {
		case "begincodespacerange":
			pdf_cmap_code_spaces(tokens, index+1, font)
		case "beginbfchar":
			pdf_cmap_characters(tokens, index+1, font)
		case "beginbfrange":
			pdf_cmap_ranges(tokens, index+1, font)
		}
	}
	return nil
}

func pdf_cmap_tokens(source []byte) (tokens []Pdf_Value, err error) {
	parser := &Pdf_Parser{Source: source}
	for parser.Offset < len(source) {
		pdf_skip_space_and_comments(parser)
		if parser.Offset >= len(source) {
			break
		}
		value, value_err := pdf_parse_value(parser, 0)
		if value_err != nil {
			return nil, value_err
		}
		tokens = append(tokens, value)
	}
	return tokens, nil
}

func pdf_cmap_code_spaces(tokens []Pdf_Value, start int, font *Pdf_Font) {
	for index := start; index+1 < len(tokens); index += 2 {
		if tokens[index].Kind == PDF_VALUE_KEYWORD {
			if tokens[index].Name == "endcodespacerange" {
				return
			}
		}
		if tokens[index].Kind != PDF_VALUE_STRING {
			continue
		}
		code_size := len(tokens[index].String_Bytes)
		if !pdf_integer_slice_contains(font.Code_Sizes, code_size) {
			font.Code_Sizes = append(font.Code_Sizes, code_size)
		}
	}
}

func pdf_cmap_characters(tokens []Pdf_Value, start int, font *Pdf_Font) {
	for index := start; index+1 < len(tokens); index += 2 {
		if tokens[index].Kind == PDF_VALUE_KEYWORD {
			if tokens[index].Name == "endbfchar" {
				return
			}
		}
		if tokens[index].Kind != PDF_VALUE_STRING {
			continue
		}
		if tokens[index+1].Kind != PDF_VALUE_STRING {
			continue
		}
		font.Unicode_Map[string(tokens[index].String_Bytes)] = pdf_utf16_text(
			tokens[index+1].String_Bytes,
		)
	}
}

func pdf_cmap_ranges(tokens []Pdf_Value, start int, font *Pdf_Font) {
	for index := start; index+2 < len(tokens); index += 3 {
		if tokens[index].Kind == PDF_VALUE_KEYWORD {
			if tokens[index].Name == "endbfrange" {
				return
			}
		}
		if tokens[index].Kind != PDF_VALUE_STRING {
			continue
		}
		if tokens[index+1].Kind != PDF_VALUE_STRING {
			continue
		}
		pdf_cmap_range(&Pdf_Cmap_Range_Input{
			Font: font, First: tokens[index], Last: tokens[index+1],
			Target: tokens[index+2],
		})
	}
}

// Pdf_Cmap_Range_Input describes one ToUnicode range mapping.
type Pdf_Cmap_Range_Input struct {
	// Font receives the expanded mappings.
	Font *Pdf_Font
	// First is the first encoded character.
	First Pdf_Value
	// Last is the last encoded character.
	Last Pdf_Value
	// Target contains sequential or explicit Unicode values.
	Target Pdf_Value
}

func pdf_cmap_range(input *Pdf_Cmap_Range_Input) {
	first_code := pdf_bytes_integer(input.First.String_Bytes)
	last_code := pdf_bytes_integer(input.Last.String_Bytes)
	if last_code < first_code {
		return
	}
	if last_code-first_code > 65536 {
		return
	}
	for code := first_code; code <= last_code; code++ {
		source := pdf_integer_bytes(&Pdf_Integer_Bytes_Input{
			Value: code, Size: len(input.First.String_Bytes),
		})
		if input.Target.Kind == PDF_VALUE_ARRAY {
			array_index := code - first_code
			if array_index < len(input.Target.Array) {
				if input.Target.Array[array_index].Kind == PDF_VALUE_STRING {
					input.Font.Unicode_Map[string(source)] = pdf_utf16_text(
						input.Target.Array[array_index].String_Bytes,
					)
				}
			}
			continue
		}
		if input.Target.Kind == PDF_VALUE_STRING {
			destination := pdf_increment_bytes(
				input.Target.String_Bytes, code-first_code,
			)
			input.Font.Unicode_Map[string(source)] = pdf_utf16_text(destination)
		}
	}
}

func pdf_utf16_text(source []byte) (text string) {
	if len(source) >= 2 {
		if source[0] == 0xfe {
			if source[1] == 0xff {
				source = source[2:]
			}
		}
	}
	if len(source)%2 != 0 {
		if utf8.Valid(source) {
			return string(source)
		}
		return ""
	}
	units := make([]uint16, len(source)/2)
	for index := range units {
		units[index] = uint16(source[index*2])<<8 | uint16(source[index*2+1])
	}
	return string(utf16.Decode(units))
}

func pdf_bytes_integer(source []byte) (value int) {
	for _, current := range source {
		value = value<<8 | int(current)
	}
	return value
}

// Pdf_Integer_Bytes_Input describes a fixed-width big-endian integer.
type Pdf_Integer_Bytes_Input struct {
	// Value is the integer to encode.
	Value int
	// Size is the required byte count.
	Size int
}

func pdf_integer_bytes(input *Pdf_Integer_Bytes_Input) (source []byte) {
	source = make([]byte, input.Size)
	value := input.Value
	for index := input.Size - 1; index >= 0; index-- {
		source[index] = byte(value)
		value >>= 8
	}
	return source
}

func pdf_increment_bytes(source []byte, increment int) (result []byte) {
	result = append([]byte{}, source...)
	for index := len(result) - 1; index >= 0 && increment > 0; index-- {
		value := int(result[index]) + increment
		result[index] = byte(value)
		increment = value >> 8
	}
	return result
}

func pdf_integer_slice_contains(values []int, target int) (contains bool) {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func pdf_decode_glyphs(font *Pdf_Font, source []byte) (glyphs []Pdf_Glyph) {
	for offset := 0; offset < len(source); {
		code_size := pdf_font_code_size(font, source[offset:])
		if offset+code_size > len(source) {
			code_size = 1
		}
		code_bytes := source[offset : offset+code_size]
		code := pdf_bytes_integer(code_bytes)
		text, mapped := font.Unicode_Map[string(code_bytes)]
		if !mapped {
			text = pdf_font_fallback_text(font, code_bytes, code)
		}
		width, width_exists := font.Widths[code]
		if !width_exists {
			width = font.Default_Width
		}
		glyphs = append(glyphs, Pdf_Glyph{
			Text: text, Code: code, Width: width, Is_Space: text == " ",
		})
		offset += code_size
	}
	return glyphs
}

func pdf_font_code_size(font *Pdf_Font, source []byte) (code_size int) {
	for _, candidate := range font.Code_Sizes {
		if candidate <= len(source) {
			if _, exists := font.Unicode_Map[string(source[:candidate])]; exists {
				if candidate > code_size {
					code_size = candidate
				}
			}
		}
	}
	if code_size != 0 {
		return code_size
	}
	for _, candidate := range font.Code_Sizes {
		if candidate <= len(source) {
			if candidate > code_size {
				code_size = candidate
			}
		}
	}
	if code_size == 0 {
		return 1
	}
	return code_size
}

func pdf_font_fallback_text(font *Pdf_Font, source []byte, code int) (text string) {
	if !font.Type_Zero {
		if code >= 0 {
			if code < 256 {
				return font.Base_Encoding[code]
			}
		}
	}
	if len(source) == 1 {
		if source[0] >= 32 {
			if source[0] <= 126 {
				return string(source)
			}
		}
	}
	if code >= 32 {
		if code <= utf8.MaxRune {
			return string(rune(code))
		}
	}
	return ""
}

// Pdf_Matrix is the six fixed-point coefficients of an affine transform.
type Pdf_Matrix struct {
	// A is the horizontal scale and rotation coefficient.
	A fixedpoint.Number
	// B is the vertical rotation coefficient.
	B fixedpoint.Number
	// C is the horizontal skew coefficient.
	C fixedpoint.Number
	// D is the vertical scale and rotation coefficient.
	D fixedpoint.Number
	// E is the horizontal translation.
	E fixedpoint.Number
	// F is the vertical translation.
	F fixedpoint.Number
}

// Pdf_Character is one decoded glyph with page coordinates.
type Pdf_Character struct {
	// Text is the decoded Unicode text.
	Text string
	// X0 is the left page coordinate.
	X0 fixedpoint.Number
	// X1 is the right page coordinate.
	X1 fixedpoint.Number
	// Top is the top-origin row coordinate.
	Top fixedpoint.Number
	// Bottom is the lower edge used to measure prose gaps.
	Bottom fixedpoint.Number
	// Order resolves equal positions without map-order dependence.
	Order int
}

// Pdf_Graphics_State retains only parameters that affect text extraction.
type Pdf_Graphics_State struct {
	// Ctm maps content coordinates into page coordinates.
	Ctm Pdf_Matrix
	// Text_Matrix locates the next glyph.
	Text_Matrix Pdf_Matrix
	// Line_Matrix anchors relative_offset line moves.
	Line_Matrix Pdf_Matrix
	// Font is the active decoded font.
	Font Pdf_Font
	// Font_Set prevents output before a valid Tf operator.
	Font_Set bool
	// Font_Size scales thousandth-em widths.
	Font_Size fixedpoint.Number
	// Character_Spacing is the Tc adjustment.
	Character_Spacing fixedpoint.Number
	// Word_Spacing is the Tw adjustment for space codes.
	Word_Spacing fixedpoint.Number
	// Horizontal_Scale is the Tz percentage expressed as a ratio.
	Horizontal_Scale fixedpoint.Number
	// Line_Advance is the TL distance used by T*.
	Line_Advance fixedpoint.Number
	// Rise is the Ts vertical baseline offset.
	Rise fixedpoint.Number
}

// Pdf_Content_Context owns only current-page interpreter and layout state.
type Pdf_Content_Context struct {
	// Document provides bounded object and stream resolution.
	Document *Pdf_Document
	// Page provides dimensions and inherited resources.
	Page *Pdf_Page
	// State is the active graphics and text state.
	State Pdf_Graphics_State
	// State_Stack implements q and Q without recursion.
	State_Stack []Pdf_Graphics_State
	// Characters is released after page candidates are built.
	Characters []Pdf_Character
	// Font_Cache prevents repeated ToUnicode decoding on one page.
	Font_Cache map[string]Pdf_Font
	// Character_Count makes equal-position sorting stable.
	Character_Count int
	// Content_Stack drives nested Form XObjects without recursive calls.
	Content_Stack []Pdf_Content_Frame
}

// Pdf_Content_Frame retains one active page or Form XObject content stream.
type Pdf_Content_Frame struct {
	// Parser is the suspended cursor for this content stream.
	Parser Pdf_Parser
	// Resources resolve this stream's fonts and nested forms.
	Resources Pdf_Value
	// Depth bounds nested Form XObjects.
	Depth int
	// Operands wait for the next content operator.
	Operands []Pdf_Value
	// Saved_State is restored after a Form XObject finishes.
	Saved_State Pdf_Graphics_State
	// Restore_State distinguishes forms from top-level page streams.
	Restore_State bool
}

func pdf_extract_page(
	document *Pdf_Document,
	page *Pdf_Page,
) (candidate Pdf_Page_Candidate, err error) {
	context := &Pdf_Content_Context{
		Document:   document,
		Page:       page,
		State:      pdf_default_graphics_state(),
		Font_Cache: make(map[string]Pdf_Font),
	}
	for _, content := range page.Contents {
		if content.Kind != PDF_VALUE_DICTIONARY {
			continue
		}
		if content.Stream == nil {
			continue
		}
		decoded, decode_err := pdf_decode_stream(document, content)
		if decode_err != nil {
			return candidate, decode_err
		}
		if interpret_err := pdf_interpret_content(
			context, decoded, page.Resources, 0,
		); interpret_err != nil {
			return candidate, interpret_err
		}
	}
	words := pdf_characters_to_words(context.Characters)
	candidate.Form_Content, candidate.Is_Form = pdf_form_content(words, page.Width)
	candidate.Plain_Content = pdf_prose_content(context.Characters)
	context.Characters = nil
	context.State_Stack = nil
	context.Font_Cache = nil
	context.Content_Stack = nil
	return candidate, nil
}

func pdf_default_graphics_state() (state Pdf_Graphics_State) {
	state.Ctm = pdf_identity_matrix()
	state.Text_Matrix = pdf_identity_matrix()
	state.Line_Matrix = pdf_identity_matrix()
	state.Horizontal_Scale = fixedpoint.From_Integer(1)
	return state
}

func pdf_identity_matrix() (matrix Pdf_Matrix) {
	return Pdf_Matrix{
		A: fixedpoint.From_Integer(1),
		D: fixedpoint.From_Integer(1),
	}
}

func pdf_interpret_content(
	context *Pdf_Content_Context,
	source []byte,
	resources Pdf_Value,
	depth int,
) (err error) {
	if depth > PDF_DEPTH_MAX {
		return fmt.Errorf("PDF Form depth exceeds %d", PDF_DEPTH_MAX)
	}
	context.Content_Stack = append(context.Content_Stack, Pdf_Content_Frame{
		Parser: Pdf_Parser{Source: source}, Resources: resources, Depth: depth,
		Operands: make([]Pdf_Value, 0, 8),
	})
	for len(context.Content_Stack) != 0 {
		frame_index := len(context.Content_Stack) - 1
		frame := &context.Content_Stack[frame_index]
		pdf_skip_space_and_comments(&frame.Parser)
		if frame.Parser.Offset >= len(frame.Parser.Source) {
			if frame.Restore_State {
				context.State = frame.Saved_State
			}
			context.Content_Stack = context.Content_Stack[:frame_index]
			continue
		}
		value, value_err := pdf_parse_value(&frame.Parser, 0)
		if value_err != nil {
			return value_err
		}
		if value.Kind != PDF_VALUE_KEYWORD {
			frame.Operands = append(frame.Operands, value)
			continue
		}
		if value.Name == "BI" {
			pdf_skip_inline_image(&frame.Parser)
			frame.Operands = frame.Operands[:0]
			continue
		}
		operands := frame.Operands
		frame.Operands = frame.Operands[:0]
		operator_err := pdf_apply_content_operator(
			context, frame.Resources, frame.Depth, value.Name, operands,
		)
		if operator_err != nil {
			return operator_err
		}
	}
	return nil
}

func pdf_skip_inline_image(parser *Pdf_Parser) {
	identifier_offset := bytes.Index(parser.Source[parser.Offset:], []byte(" ID"))
	if identifier_offset < 0 {
		parser.Offset = len(parser.Source)
		return
	}
	data_start := parser.Offset + identifier_offset + 3
	end_offset := bytes.Index(parser.Source[data_start:], []byte(" EI"))
	if end_offset < 0 {
		parser.Offset = len(parser.Source)
		return
	}
	parser.Offset = data_start + end_offset + 3
}

func pdf_apply_content_operator(
	context *Pdf_Content_Context,
	resources Pdf_Value,
	depth int,
	operator string,
	operands []Pdf_Value,
) (err error) {
	switch operator {
	case "q", "Q", "cm", "Do":
		return pdf_apply_graphics_operator(context, resources, depth, operator, operands)
	}
	return pdf_apply_text_operator(context, resources, operator, operands)
}

func pdf_apply_graphics_operator(
	context *Pdf_Content_Context,
	resources Pdf_Value,
	depth int,
	operator string,
	operands []Pdf_Value,
) (err error) {
	switch operator {
	case "q":
		context.State_Stack = append(context.State_Stack, context.State)
	case "Q":
		if len(context.State_Stack) != 0 {
			context.State = context.State_Stack[len(context.State_Stack)-1]
			context.State_Stack = context.State_Stack[:len(context.State_Stack)-1]
		}
	case "cm":
		matrix, matrix_ok := pdf_operands_matrix(operands)
		if matrix_ok {
			context.State.Ctm = pdf_matrix_multiply(&Pdf_Matrix_Multiply_Input{
				Left: matrix, Right: context.State.Ctm,
			})
		}
	case "Do":
		return pdf_apply_form(context, resources, depth, operands)
	}
	return nil
}

func pdf_apply_form(
	context *Pdf_Content_Context,
	resources Pdf_Value,
	depth int,
	operands []Pdf_Value,
) (err error) {
	if len(operands) == 0 {
		return nil
	}
	if operands[len(operands)-1].Kind != PDF_VALUE_NAME {
		return nil
	}
	xobjects, exists, xobjects_err := pdf_dictionary_value(
		context.Document, resources, "XObject",
	)
	if xobjects_err != nil {
		return xobjects_err
	}
	if !exists {
		return nil
	}
	if xobjects.Kind != PDF_VALUE_DICTIONARY {
		return xobjects_err
	}
	xobject_entry, exists := xobjects.Dictionary[operands[len(operands)-1].Name]
	if !exists {
		return nil
	}
	xobject, resolve_err := pdf_resolve_value(context.Document, xobject_entry, 0)
	if resolve_err != nil {
		return resolve_err
	}
	if pdf_dictionary_name(context.Document, xobject, "Subtype") != "Form" {
		return nil
	}
	return pdf_interpret_form(&Pdf_Interpret_Form_Input{
		Context: context, Parent_Resources: resources,
		Form: xobject, Depth: depth + 1,
	})
}

// Pdf_Interpret_Form_Input contains one nested Form XObject invocation.
type Pdf_Interpret_Form_Input struct {
	// Context retains the shared page output and graphics state.
	Context *Pdf_Content_Context
	// Parent_Resources are inherited when the form declares none.
	Parent_Resources Pdf_Value
	// Form is the resolved form stream.
	Form Pdf_Value
	// Depth bounds recursive forms.
	Depth int
}

func pdf_interpret_form(input *Pdf_Interpret_Form_Input) (err error) {
	if input.Depth > PDF_DEPTH_MAX {
		return fmt.Errorf("PDF Form depth exceeds %d", PDF_DEPTH_MAX)
	}
	decoded, decode_err := pdf_decode_stream(input.Context.Document, input.Form)
	if decode_err != nil {
		return decode_err
	}
	resources := input.Parent_Resources
	form_resources, resources_exists, resources_err := pdf_dictionary_value(
		input.Context.Document, input.Form, "Resources",
	)
	if resources_err != nil {
		return resources_err
	}
	if resources_exists {
		resources = form_resources
	}
	saved := input.Context.State
	matrix_value, matrix_exists, matrix_err := pdf_dictionary_value(
		input.Context.Document, input.Form, "Matrix",
	)
	if matrix_err != nil {
		return matrix_err
	}
	if matrix_exists {
		matrix, matrix_ok := pdf_value_matrix(matrix_value)
		if matrix_ok {
			input.Context.State.Ctm = pdf_matrix_multiply(&Pdf_Matrix_Multiply_Input{
				Left: matrix, Right: input.Context.State.Ctm,
			})
		}
	}
	input.Context.Content_Stack = append(
		input.Context.Content_Stack,
		Pdf_Content_Frame{
			Parser: Pdf_Parser{Source: decoded}, Resources: resources,
			Depth: input.Depth, Operands: make([]Pdf_Value, 0, 8),
			Saved_State: saved, Restore_State: true,
		},
	)
	return nil
}

func pdf_apply_text_operator(
	context *Pdf_Content_Context,
	resources Pdf_Value,
	operator string,
	operands []Pdf_Value,
) (err error) {
	switch operator {
	case "BT":
		context.State.Text_Matrix = pdf_identity_matrix()
		context.State.Line_Matrix = pdf_identity_matrix()
	case "Tf":
		return pdf_set_font(context, resources, operands)
	case "Tm":
		matrix, matrix_ok := pdf_operands_matrix(operands)
		if matrix_ok {
			context.State.Text_Matrix = matrix
			context.State.Line_Matrix = matrix
		}
	case "Td":
		pdf_move_text(context, operands, false)
	case "TD":
		pdf_move_text(context, operands, true)
	case "T*":
		pdf_text_new_line(context)
	case "Tj":
		pdf_show_operand(context, operands)
	case "TJ":
		pdf_show_array(context, operands)
	case "'":
		pdf_text_new_line(context)
		pdf_show_operand(context, operands)
	case "\"":
		pdf_set_quote_spacing(context, operands)
	case "Tc":
		context.State.Character_Spacing = pdf_last_number(operands)
	case "Tw":
		context.State.Word_Spacing = pdf_last_number(operands)
	case "Tz":
		context.State.Horizontal_Scale = pdf_number_ratio(pdf_last_number(operands), 100)
	case "TL":
		context.State.Line_Advance = pdf_last_number(operands)
	case "Ts":
		context.State.Rise = pdf_last_number(operands)
	}
	return nil
}

func pdf_set_font(
	context *Pdf_Content_Context,
	resources Pdf_Value,
	operands []Pdf_Value,
) (err error) {
	if len(operands) < 2 {
		return nil
	}
	if operands[len(operands)-2].Kind != PDF_VALUE_NAME {
		return nil
	}
	if operands[len(operands)-1].Kind != PDF_VALUE_NUMBER {
		return nil
	}
	name := operands[len(operands)-2].Name
	cache_key := pdf_font_cache_key(context.Document, resources, name)
	font, exists := context.Font_Cache[cache_key]
	if !exists {
		font, err = pdf_font_from_resources(context.Document, resources, name)
		if err != nil {
			return err
		}
		context.Font_Cache[cache_key] = font
	}
	context.State.Font = font
	context.State.Font_Set = true
	context.State.Font_Size = operands[len(operands)-1].Number
	return nil
}

func pdf_font_cache_key(document *Pdf_Document, resources Pdf_Value, name string) (key string) {
	fonts, exists, fonts_err := pdf_dictionary_value(document, resources, "Font")
	if fonts_err != nil {
		return name
	}
	if !exists {
		return name
	}
	if fonts.Kind != PDF_VALUE_DICTIONARY {
		return name
	}
	entry, exists := fonts.Dictionary[name]
	if exists {
		if entry.Kind == PDF_VALUE_REFERENCE {
			return strconv.Itoa(entry.Reference_Object_Number) + ":" + name
		}
	}
	return pdf_dictionary_name(document, resources, "BaseFont") + ":" + name
}

func pdf_move_text(context *Pdf_Content_Context, operands []Pdf_Value, set_line_advance bool) {
	if len(operands) < 2 {
		return
	}
	x, x_ok := pdf_value_number(operands[len(operands)-2])
	y, y_ok := pdf_value_number(operands[len(operands)-1])
	if !x_ok {
		return
	}
	if !y_ok {
		return
	}
	if set_line_advance {
		context.State.Line_Advance = -y
	}
	context.State.Line_Matrix = pdf_matrix_translate(&Pdf_Matrix_Translate_Input{
		Matrix: context.State.Line_Matrix, X: x, Y: y,
	})
	context.State.Text_Matrix = context.State.Line_Matrix
}

func pdf_text_new_line(context *Pdf_Content_Context) {
	context.State.Line_Matrix = pdf_matrix_translate(&Pdf_Matrix_Translate_Input{
		Matrix: context.State.Line_Matrix, Y: -context.State.Line_Advance,
	})
	context.State.Text_Matrix = context.State.Line_Matrix
}

func pdf_set_quote_spacing(context *Pdf_Content_Context, operands []Pdf_Value) {
	if len(operands) < 3 {
		return
	}
	context.State.Word_Spacing = pdf_number_at(operands, len(operands)-3)
	context.State.Character_Spacing = pdf_number_at(operands, len(operands)-2)
	pdf_text_new_line(context)
	pdf_show_value(context, operands[len(operands)-1])
}

func pdf_show_operand(context *Pdf_Content_Context, operands []Pdf_Value) {
	if len(operands) == 0 {
		return
	}
	pdf_show_value(context, operands[len(operands)-1])
}

func pdf_show_value(context *Pdf_Content_Context, value Pdf_Value) {
	if value.Kind != PDF_VALUE_STRING {
		return
	}
	if !context.State.Font_Set {
		return
	}
	glyphs := pdf_decode_glyphs(&context.State.Font, value.String_Bytes)
	for _, glyph := range glyphs {
		pdf_show_glyph(context, &glyph)
	}
}

func pdf_show_array(context *Pdf_Content_Context, operands []Pdf_Value) {
	if len(operands) == 0 {
		return
	}
	array := operands[len(operands)-1]
	if array.Kind != PDF_VALUE_ARRAY {
		return
	}
	for _, entry := range array.Array {
		if entry.Kind == PDF_VALUE_STRING {
			pdf_show_value(context, entry)
		}
		if entry.Kind == PDF_VALUE_NUMBER {
			pdf_adjust_text(context, entry.Number)
		}
	}
}

func pdf_show_glyph(context *Pdf_Content_Context, glyph *Pdf_Glyph) {
	glyph_width := pdf_glyph_width(context, glyph)
	advance := glyph_width + context.State.Character_Spacing
	if glyph.Is_Space {
		advance += context.State.Word_Spacing
	}
	glyph_width = pdf_fixed_multiply(&Pdf_Fixed_Multiply_Input{
		Left: glyph_width, Right: context.State.Horizontal_Scale,
	})
	advance = pdf_fixed_multiply(&Pdf_Fixed_Multiply_Input{
		Left: advance, Right: context.State.Horizontal_Scale,
	})
	if glyph.Text != "" {
		pdf_append_character(context, glyph.Text, glyph_width)
	}
	context.State.Text_Matrix = pdf_matrix_translate(&Pdf_Matrix_Translate_Input{
		Matrix: context.State.Text_Matrix, X: advance,
	})
}

func pdf_glyph_width(context *Pdf_Content_Context, glyph *Pdf_Glyph) (width fixedpoint.Number) {
	ratio := fixedpoint.From_Ratio(&fixedpoint.From_Ratio_Input{
		Numerator: glyph.Width, Denominator: 1000,
	})
	return pdf_fixed_multiply(&Pdf_Fixed_Multiply_Input{
		Left: context.State.Font_Size, Right: ratio,
	})
}

func pdf_append_character(
	context *Pdf_Content_Context,
	text string,
	width fixedpoint.Number,
) {
	origin_x, origin_y := pdf_matrix_point(&Pdf_Matrix_Point_Input{
		Matrix: context.State.Text_Matrix, Y: context.State.Rise,
	})
	end_x, end_y := pdf_matrix_point(&Pdf_Matrix_Point_Input{
		Matrix: context.State.Text_Matrix, X: width, Y: context.State.Rise,
	})
	origin_x, origin_y = pdf_matrix_point(&Pdf_Matrix_Point_Input{
		Matrix: context.State.Ctm, X: origin_x, Y: origin_y,
	})
	end_x, end_y = pdf_matrix_point(&Pdf_Matrix_Point_Input{
		Matrix: context.State.Ctm, X: end_x, Y: end_y,
	})
	x0 := pdf_fixed_min(&Pdf_Fixed_Input_Min{Left: origin_x, Right: end_x})
	x1 := pdf_fixed_max(&Pdf_Fixed_Input_Max{Left: origin_x, Right: end_x})
	baseline := pdf_fixed_max(&Pdf_Fixed_Input_Max{Left: origin_y, Right: end_y})
	height := pdf_fixed_absolute(context.State.Font_Size)
	context.Characters = append(context.Characters, Pdf_Character{
		Text: text, X0: x0, X1: x1,
		Top:    context.Page.Height - baseline - height,
		Bottom: context.Page.Height - baseline,
		Order:  context.Character_Count,
	})
	context.Character_Count++
}

func pdf_adjust_text(context *Pdf_Content_Context, adjustment fixedpoint.Number) {
	distance := pdf_fixed_multiply(&Pdf_Fixed_Multiply_Input{
		Left: context.State.Font_Size, Right: adjustment,
	})
	distance = pdf_number_ratio(distance, -1000)
	distance = pdf_fixed_multiply(&Pdf_Fixed_Multiply_Input{
		Left: distance, Right: context.State.Horizontal_Scale,
	})
	context.State.Text_Matrix = pdf_matrix_translate(&Pdf_Matrix_Translate_Input{
		Matrix: context.State.Text_Matrix, X: distance,
	})
}

func pdf_operands_matrix(operands []Pdf_Value) (matrix Pdf_Matrix, ok bool) {
	if len(operands) < 6 {
		return matrix, false
	}
	start := len(operands) - 6
	return pdf_value_matrix(Pdf_Value{
		Kind:  PDF_VALUE_ARRAY,
		Array: operands[start:],
	})
}

func pdf_value_matrix(value Pdf_Value) (matrix Pdf_Matrix, ok bool) {
	if value.Kind != PDF_VALUE_ARRAY {
		return matrix, false
	}
	if len(value.Array) < 6 {
		return matrix, false
	}
	numbers := make([]fixedpoint.Number, 6)
	for index := 0; index < 6; index++ {
		if value.Array[index].Kind != PDF_VALUE_NUMBER {
			return matrix, false
		}
		numbers[index] = value.Array[index].Number
	}
	return Pdf_Matrix{
		A: numbers[0], B: numbers[1], C: numbers[2],
		D: numbers[3], E: numbers[4], F: numbers[5],
	}, true
}

// Pdf_Matrix_Multiply_Input groups two PDF affine matrices.
type Pdf_Matrix_Multiply_Input struct {
	// Left is applied before Right.
	Left Pdf_Matrix
	// Right receives the transformed coordinates.
	Right Pdf_Matrix
}

func pdf_matrix_multiply(input *Pdf_Matrix_Multiply_Input) (product Pdf_Matrix) {
	product.A = pdf_fixed_multiply(&Pdf_Fixed_Multiply_Input{
		Left: input.Left.A, Right: input.Right.A,
	})
	product.A += pdf_fixed_multiply(&Pdf_Fixed_Multiply_Input{
		Left: input.Left.B, Right: input.Right.C,
	})
	product.B = pdf_fixed_multiply(&Pdf_Fixed_Multiply_Input{
		Left: input.Left.A, Right: input.Right.B,
	})
	product.B += pdf_fixed_multiply(&Pdf_Fixed_Multiply_Input{
		Left: input.Left.B, Right: input.Right.D,
	})
	product.C = pdf_fixed_multiply(&Pdf_Fixed_Multiply_Input{
		Left: input.Left.C, Right: input.Right.A,
	})
	product.C += pdf_fixed_multiply(&Pdf_Fixed_Multiply_Input{
		Left: input.Left.D, Right: input.Right.C,
	})
	product.D = pdf_fixed_multiply(&Pdf_Fixed_Multiply_Input{
		Left: input.Left.C, Right: input.Right.B,
	})
	product.D += pdf_fixed_multiply(&Pdf_Fixed_Multiply_Input{
		Left: input.Left.D, Right: input.Right.D,
	})
	product.E = pdf_fixed_multiply(&Pdf_Fixed_Multiply_Input{
		Left: input.Left.E, Right: input.Right.A,
	})
	product.E += pdf_fixed_multiply(&Pdf_Fixed_Multiply_Input{
		Left: input.Left.F, Right: input.Right.C,
	}) + input.Right.E
	product.F = pdf_fixed_multiply(&Pdf_Fixed_Multiply_Input{
		Left: input.Left.E, Right: input.Right.B,
	})
	product.F += pdf_fixed_multiply(&Pdf_Fixed_Multiply_Input{
		Left: input.Left.F, Right: input.Right.D,
	}) + input.Right.F
	return product
}

// Pdf_Matrix_Translate_Input describes a translation in matrix coordinates.
type Pdf_Matrix_Translate_Input struct {
	// Matrix is the coordinate basis.
	Matrix Pdf_Matrix
	// X is the horizontal translation.
	X fixedpoint.Number
	// Y is the vertical translation.
	Y fixedpoint.Number
}

func pdf_matrix_translate(input *Pdf_Matrix_Translate_Input) (translated Pdf_Matrix) {
	translated = input.Matrix
	translated.E += pdf_fixed_multiply(&Pdf_Fixed_Multiply_Input{
		Left: input.X, Right: input.Matrix.A,
	})
	translated.E += pdf_fixed_multiply(&Pdf_Fixed_Multiply_Input{
		Left: input.Y, Right: input.Matrix.C,
	})
	translated.F += pdf_fixed_multiply(&Pdf_Fixed_Multiply_Input{
		Left: input.X, Right: input.Matrix.B,
	})
	translated.F += pdf_fixed_multiply(&Pdf_Fixed_Multiply_Input{
		Left: input.Y, Right: input.Matrix.D,
	})
	return translated
}

// Pdf_Matrix_Point_Input describes one point transformed by a PDF matrix.
type Pdf_Matrix_Point_Input struct {
	// Matrix is the coordinate transform.
	Matrix Pdf_Matrix
	// X is the source horizontal coordinate.
	X fixedpoint.Number
	// Y is the source vertical coordinate.
	Y fixedpoint.Number
}

func pdf_matrix_point(
	input *Pdf_Matrix_Point_Input,
) (result_x fixedpoint.Number, result_y fixedpoint.Number) {
	result_x = pdf_fixed_multiply(&Pdf_Fixed_Multiply_Input{
		Left: input.X, Right: input.Matrix.A,
	})
	result_x += pdf_fixed_multiply(&Pdf_Fixed_Multiply_Input{
		Left: input.Y, Right: input.Matrix.C,
	}) + input.Matrix.E
	result_y = pdf_fixed_multiply(&Pdf_Fixed_Multiply_Input{
		Left: input.X, Right: input.Matrix.B,
	})
	result_y += pdf_fixed_multiply(&Pdf_Fixed_Multiply_Input{
		Left: input.Y, Right: input.Matrix.D,
	}) + input.Matrix.F
	return result_x, result_y
}

// Pdf_Fixed_Multiply_Input groups two fixed-point factors.
type Pdf_Fixed_Multiply_Input struct {
	// Left is the first factor.
	Left fixedpoint.Number
	// Right is the second factor.
	Right fixedpoint.Number
}

func pdf_fixed_multiply(input *Pdf_Fixed_Multiply_Input) (product fixedpoint.Number) {
	return fixedpoint.Multiply(&fixedpoint.Multiply_Input{A: input.Left, B: input.Right})
}

func pdf_number_ratio(value fixedpoint.Number, denominator int64) (result fixedpoint.Number) {
	ratio := fixedpoint.From_Ratio(&fixedpoint.From_Ratio_Input{
		Numerator: 1, Denominator: denominator,
	})
	return pdf_fixed_multiply(&Pdf_Fixed_Multiply_Input{Left: value, Right: ratio})
}

func pdf_number_at(values []Pdf_Value, index int) (number fixedpoint.Number) {
	if index < 0 {
		return 0
	}
	if index >= len(values) {
		return 0
	}
	if values[index].Kind != PDF_VALUE_NUMBER {
		return 0
	}
	return values[index].Number
}

func pdf_last_number(values []Pdf_Value) (number fixedpoint.Number) {
	return pdf_number_at(values, len(values)-1)
}

// Pdf_Fixed_Input_Min groups two fixed-point values for minimum selection.
type Pdf_Fixed_Input_Min struct {
	// Left is the first candidate.
	Left fixedpoint.Number
	// Right is the second candidate.
	Right fixedpoint.Number
}

func pdf_fixed_min(input *Pdf_Fixed_Input_Min) (minimum fixedpoint.Number) {
	if input.Left < input.Right {
		return input.Left
	}
	return input.Right
}

// Pdf_Fixed_Input_Max groups two fixed-point values for maximum selection.
type Pdf_Fixed_Input_Max struct {
	// Left is the first candidate.
	Left fixedpoint.Number
	// Right is the second candidate.
	Right fixedpoint.Number
}

func pdf_fixed_max(input *Pdf_Fixed_Input_Max) (maximum fixedpoint.Number) {
	if input.Left > input.Right {
		return input.Left
	}
	return input.Right
}

func pdf_fixed_absolute(value fixedpoint.Number) (absolute fixedpoint.Number) {
	if value < 0 {
		return -value
	}
	return value
}

// Pdf_Word is a pdfplumber-compatible cluster of nearby glyphs.
type Pdf_Word struct {
	// Text retains blanks because MarkItDown requests keep_blank_chars.
	Text string
	// X0 is the cluster's left coordinate.
	X0 fixedpoint.Number
	// X1 is the cluster's right coordinate.
	X1 fixedpoint.Number
	// Top selects the five-point form row.
	Top fixedpoint.Number
	// Bottom retains vertical extent for prose spacing.
	Bottom fixedpoint.Number
}

// Pdf_Form_Row holds the complete row-classification state.
type Pdf_Form_Row struct {
	// Words remain position-sorted for cell assignment.
	Words []Pdf_Word
	// Text is the fallback for a non-table row.
	Text string
	// X_Groups are candidate column starts more than 50 points apart.
	X_Groups []fixedpoint.Number
	// Is_Paragraph prevents wide prose from defining columns.
	Is_Paragraph bool
	// Has_Partial_Number keeps MasterFormat list items out of tables.
	Has_Partial_Number bool
	// Is_Table_Row requires alignment with at least two global columns.
	Is_Table_Row bool
}

// Pdf_Table_Region is one consecutive half-open range of table rows.
type Pdf_Table_Region struct {
	// Start is the first table-row index.
	Start int
	// End is one past the last table-row index.
	End int
}

func pdf_characters_to_words(characters []Pdf_Character) (words []Pdf_Word) {
	return pdf_characters_to_words_with_tolerance(
		characters, fixedpoint.From_Integer(3),
	)
}

func pdf_characters_to_words_with_tolerance(
	characters []Pdf_Character,
	horizontal_tolerance fixedpoint.Number,
) (words []Pdf_Word) {
	ordered := append([]Pdf_Character{}, characters...)
	pdf_sort_characters(ordered)
	for _, character := range ordered {
		if character.Text == "" {
			continue
		}
		if len(words) == 0 {
			words = append(words, Pdf_Word{
				Text: character.Text, X0: character.X0, X1: character.X1,
				Top: character.Top, Bottom: character.Bottom,
			})
			continue
		}
		if !pdf_character_joins_word(
			&character, &words[len(words)-1], horizontal_tolerance,
		) {
			words = append(words, Pdf_Word{
				Text: character.Text, X0: character.X0, X1: character.X1,
				Top: character.Top, Bottom: character.Bottom,
			})
			continue
		}
		word := &words[len(words)-1]
		word.Text += character.Text
		word.X0 = pdf_fixed_min(&Pdf_Fixed_Input_Min{Left: word.X0, Right: character.X0})
		word.X1 = pdf_fixed_max(&Pdf_Fixed_Input_Max{Left: word.X1, Right: character.X1})
		word.Top = pdf_fixed_min(&Pdf_Fixed_Input_Min{Left: word.Top, Right: character.Top})
		word.Bottom = pdf_fixed_max(&Pdf_Fixed_Input_Max{
			Left: word.Bottom, Right: character.Bottom,
		})
	}
	return words
}

func pdf_character_joins_word(
	character *Pdf_Character,
	word *Pdf_Word,
	horizontal_tolerance fixedpoint.Number,
) (joins bool) {
	vertical_distance := pdf_fixed_absolute(character.Top - word.Top)
	if vertical_distance > fixedpoint.From_Integer(3) {
		return false
	}
	horizontal_gap := character.X0 - word.X1
	return horizontal_gap <= horizontal_tolerance
}

func pdf_sort_characters(characters []Pdf_Character) {
	for index := 1; index < len(characters); index++ {
		current := characters[index]
		position := index
		for position > 0 && pdf_character_before(&Pdf_Character_Before_Input{
			Left: &current, Right: &characters[position-1],
		}) {
			characters[position] = characters[position-1]
			position--
		}
		characters[position] = current
	}
}

// Pdf_Character_Before_Input groups two characters for visual ordering.
type Pdf_Character_Before_Input struct {
	// Left is the candidate earlier character.
	Left *Pdf_Character
	// Right is the candidate later character.
	Right *Pdf_Character
}

func pdf_character_before(input *Pdf_Character_Before_Input) (before bool) {
	vertical_distance := pdf_fixed_absolute(input.Left.Top - input.Right.Top)
	if vertical_distance <= fixedpoint.From_Integer(3) {
		if input.Left.X0 != input.Right.X0 {
			return input.Left.X0 < input.Right.X0
		}
		return input.Left.Order < input.Right.Order
	}
	return input.Left.Top < input.Right.Top
}

func pdf_form_content(words []Pdf_Word, page_width fixedpoint.Number) (
	content string,
	is_form bool,
) {
	if len(words) == 0 {
		return "", false
	}
	rows := pdf_form_rows(words, page_width)
	table_positions := pdf_table_positions(rows)
	if len(table_positions) == 0 {
		return "", false
	}
	tolerance := pdf_adaptive_column_tolerance(table_positions)
	columns := pdf_cluster_positions(table_positions, tolerance)
	if !pdf_columns_valid(columns, page_width) {
		return "", false
	}
	pdf_classify_table_rows(rows, columns)
	regions, table_row_count := pdf_table_regions(rows)
	if len(rows) != 0 {
		if table_row_count*5 < len(rows) {
			return "", false
		}
	}
	return pdf_format_form_rows(rows, columns, regions), true
}

func pdf_form_rows(words []Pdf_Word, page_width fixedpoint.Number) (rows []Pdf_Form_Row) {
	for _, word := range words {
		key := pdf_round_position(&Pdf_Round_Position_Input{
			Value: word.Top, Step: fixedpoint.From_Integer(5),
		})
		if len(rows) == 0 {
			rows = append(rows, Pdf_Form_Row{})
		} else if pdf_round_position(&Pdf_Round_Position_Input{
			Value: rows[len(rows)-1].Words[0].Top,
			Step:  fixedpoint.From_Integer(5),
		}) != key {
			rows = append(rows, Pdf_Form_Row{})
		}
		rows[len(rows)-1].Words = append(rows[len(rows)-1].Words, word)
	}
	for index := range rows {
		pdf_sort_words_x(rows[index].Words)
		pdf_analyze_form_row(&rows[index], page_width)
	}
	return rows
}

func pdf_analyze_form_row(row *Pdf_Form_Row, page_width fixedpoint.Number) {
	texts := make([]string, 0, len(row.Words))
	for _, word := range row.Words {
		texts = append(texts, word.Text)
		if len(row.X_Groups) == 0 {
			row.X_Groups = append(row.X_Groups, word.X0)
		} else if word.X0-row.X_Groups[len(row.X_Groups)-1] >
			fixedpoint.From_Integer(50) {
			row.X_Groups = append(row.X_Groups, word.X0)
		}
	}
	row.Text = strings.TrimSpace(strings.Join(texts, " "))
	first := row.Words[0]
	last := row.Words[len(row.Words)-1]
	line_width := last.X1 - first.X0
	wide := line_width > pdf_number_ratio(page_width*55, 100)
	row.Is_Paragraph = wide && utf8.RuneCountInString(row.Text) > 60
	row.Has_Partial_Number = pdf_is_partial_number(strings.TrimSpace(first.Text))
}

func pdf_sort_words_x(words []Pdf_Word) {
	for index := 1; index < len(words); index++ {
		current := words[index]
		position := index
		for position > 0 && current.X0 < words[position-1].X0 {
			words[position] = words[position-1]
			position--
		}
		words[position] = current
	}
}

func pdf_table_positions(rows []Pdf_Form_Row) (positions []fixedpoint.Number) {
	for _, row := range rows {
		if len(row.X_Groups) >= 3 {
			if !row.Is_Paragraph {
				positions = append(positions, row.X_Groups...)
			}
		}
	}
	pdf_sort_numbers(positions)
	return positions
}

func pdf_adaptive_column_tolerance(positions []fixedpoint.Number) (
	tolerance fixedpoint.Number,
) {
	gaps := make([]fixedpoint.Number, 0, len(positions))
	for index := 0; index+1 < len(positions); index++ {
		gap := positions[index+1] - positions[index]
		if gap > fixedpoint.From_Integer(5) {
			gaps = append(gaps, gap)
		}
	}
	if len(gaps) < 3 {
		return fixedpoint.From_Integer(35)
	}
	pdf_sort_numbers(gaps)
	tolerance = gaps[len(gaps)*70/100]
	tolerance = pdf_fixed_max(&Pdf_Fixed_Input_Max{
		Left: tolerance, Right: fixedpoint.From_Integer(25),
	})
	return pdf_fixed_min(&Pdf_Fixed_Input_Min{
		Left: tolerance, Right: fixedpoint.From_Integer(50),
	})
}

func pdf_cluster_positions(
	positions []fixedpoint.Number,
	tolerance fixedpoint.Number,
) (columns []fixedpoint.Number) {
	for _, position := range positions {
		if len(columns) == 0 {
			columns = append(columns, position)
		} else if position-columns[len(columns)-1] > tolerance {
			columns = append(columns, position)
		}
	}
	return columns
}

func pdf_columns_valid(columns []fixedpoint.Number, page_width fixedpoint.Number) (valid bool) {
	if len(columns) <= 1 {
		return false
	}
	content_width := columns[len(columns)-1] - columns[0]
	average_width := pdf_number_ratio(content_width, int64(len(columns)))
	if average_width < fixedpoint.From_Integer(30) {
		return false
	}
	if content_width <= 0 {
		return false
	}
	if fixedpoint.From_Integer(int64(len(columns)*72)) > content_width*10 {
		return false
	}
	adaptive_max := int(fixedpoint.Whole(pdf_number_ratio(page_width*20, 612)))
	if adaptive_max < 15 {
		adaptive_max = 15
	}
	return len(columns) <= adaptive_max
}

func pdf_classify_table_rows(rows []Pdf_Form_Row, columns []fixedpoint.Number) {
	for row_index := range rows {
		row := &rows[row_index]
		if row.Is_Paragraph {
			continue
		}
		if row.Has_Partial_Number {
			continue
		}
		aligned := make([]bool, len(columns))
		aligned_count := 0
		for _, word := range row.Words {
			column := pdf_aligned_column(word.X0, columns)
			if column >= 0 {
				if !aligned[column] {
					aligned[column] = true
					aligned_count++
				}
			}
		}
		row.Is_Table_Row = aligned_count >= 2
	}
}

func pdf_aligned_column(x fixedpoint.Number, columns []fixedpoint.Number) (column int) {
	for index, column_x := range columns {
		if pdf_fixed_absolute(x-column_x) < fixedpoint.From_Integer(40) {
			return index
		}
	}
	return -1
}

func pdf_table_regions(rows []Pdf_Form_Row) (regions []Pdf_Table_Region, count int) {
	for index := 0; index < len(rows); {
		if !rows[index].Is_Table_Row {
			index++
			continue
		}
		start := index
		for index < len(rows) && rows[index].Is_Table_Row {
			index++
		}
		regions = append(regions, Pdf_Table_Region{Start: start, End: index})
		count += index - start
	}
	return regions, count
}

func pdf_format_form_rows(
	rows []Pdf_Form_Row,
	columns []fixedpoint.Number,
	regions []Pdf_Table_Region,
) (content string) {
	lines := make([]string, 0, len(rows)*2)
	for index := 0; index < len(rows); {
		region, starts := pdf_region_at(regions, index)
		if !starts {
			lines = append(lines, rows[index].Text)
			index++
			continue
		}
		table := pdf_region_cells(rows, columns, region)
		lines = append(lines, pdf_format_table(table)...)
		index = region.End
	}
	return strings.Join(lines, "\n")
}

func pdf_region_at(regions []Pdf_Table_Region, index int) (
	region Pdf_Table_Region,
	found bool,
) {
	for _, candidate := range regions {
		if candidate.Start == index {
			return candidate, true
		}
	}
	return region, false
}

func pdf_region_cells(
	rows []Pdf_Form_Row,
	columns []fixedpoint.Number,
	region Pdf_Table_Region,
) (table [][]string) {
	for row_index := region.Start; row_index < region.End; row_index++ {
		cells := make([]string, len(columns))
		for _, word := range rows[row_index].Words {
			column := pdf_cell_column(word.X0, columns)
			if cells[column] != "" {
				cells[column] += " "
			}
			cells[column] += word.Text
		}
		table = append(table, cells)
	}
	return table
}

func pdf_cell_column(x fixedpoint.Number, columns []fixedpoint.Number) (column int) {
	column = len(columns) - 1
	for index := 0; index+1 < len(columns); index++ {
		if x < columns[index+1]-fixedpoint.From_Integer(20) {
			return index
		}
	}
	return column
}

func pdf_format_table(table [][]string) (lines []string) {
	if len(table) == 0 {
		return nil
	}
	widths := make([]int, len(table[0]))
	for _, row := range table {
		for column, cell := range row {
			width_count := utf8.RuneCountInString(cell)
			if width_count > widths[column] {
				widths[column] = width_count
			}
		}
	}
	for index := range widths {
		if widths[index] < 3 {
			widths[index] = 3
		}
	}
	lines = append(lines, pdf_format_table_row(table[0], widths))
	separators := make([]string, len(widths))
	for index, width := range widths {
		separators[index] = strings.Repeat("-", width)
	}
	lines = append(lines, pdf_format_table_row(separators, widths))
	for index := 1; index < len(table); index++ {
		lines = append(lines, pdf_format_table_row(table[index], widths))
	}
	return lines
}

func pdf_format_table_row(cells []string, widths []int) (row string) {
	padded := make([]string, len(cells))
	for index, cell := range cells {
		padding := widths[index] - utf8.RuneCountInString(cell)
		padded[index] = cell + strings.Repeat(" ", padding)
	}
	return "| " + strings.Join(padded, " | ") + " |"
}

func pdf_sort_numbers(numbers []fixedpoint.Number) {
	for index := 1; index < len(numbers); index++ {
		current := numbers[index]
		position := index
		for position > 0 && current < numbers[position-1] {
			numbers[position] = numbers[position-1]
			position--
		}
		numbers[position] = current
	}
}

// Pdf_Round_Position_Input describes position rounding to a fixed grid.
type Pdf_Round_Position_Input struct {
	// Value is the source position.
	Value fixedpoint.Number
	// Step is the grid interval.
	Step fixedpoint.Number
}

func pdf_round_position(input *Pdf_Round_Position_Input) (rounded fixedpoint.Number) {
	quotient := int64(input.Value) / int64(input.Step)
	remainder := int64(input.Value) % int64(input.Step)
	if remainder < 0 {
		remainder = -remainder
	}
	double := remainder * 2
	round_up := double > int64(input.Step)
	if double == int64(input.Step) {
		if quotient%2 != 0 {
			round_up = true
		}
	}
	if round_up {
		if input.Value >= 0 {
			quotient++
		} else {
			quotient--
		}
	}
	return fixedpoint.Number(quotient * int64(input.Step))
}

// Pdf_Prose_Line retains geometry needed to reproduce pdfminer line gaps.
type Pdf_Prose_Line struct {
	// Text is the reconstructed visual line.
	Text string
	// Top is the smallest top-origin coordinate.
	Top fixedpoint.Number
	// Bottom is the largest lower edge.
	Bottom fixedpoint.Number
	// X1 tracks the right edge for separate numeric headings.
	X1 fixedpoint.Number
	// Force_Blank_Before preserves a separate numeric heading box.
	Force_Blank_Before bool
}

func pdf_prose_content(characters []Pdf_Character) (content string) {
	words := pdf_characters_to_words_with_tolerance(
		characters, fixedpoint.From_Integer(1),
	)
	if len(words) == 0 {
		return ""
	}
	lines := pdf_prose_lines(words)
	var output strings.Builder
	for index, line := range lines {
		if index != 0 {
			output.WriteByte('\n')
			prior := lines[index-1]
			prior_height := prior.Bottom - prior.Top
			gap := line.Top - prior.Bottom
			if pdf_prose_blank_before(&Pdf_Prose_Blank_Before_Input{
				Line: &line, Gap: gap, Prior_Height: prior_height,
			}) {
				output.WriteByte('\n')
			}
		}
		output.WriteString(strings.TrimRight(line.Text, " "))
	}
	return output.String()
}

// Pdf_Prose_Blank_Before_Input contains prose geometry around one line.
type Pdf_Prose_Blank_Before_Input struct {
	// Line is the current reconstructed line.
	Line *Pdf_Prose_Line
	// Gap is the vertical space after the prior line.
	Gap fixedpoint.Number
	// Prior_Height supplies a scale for proportional gaps.
	Prior_Height fixedpoint.Number
}

func pdf_prose_blank_before(input *Pdf_Prose_Blank_Before_Input) (blank bool) {
	if input.Line.Force_Blank_Before {
		return true
	}
	if input.Gap > pdf_number_ratio(input.Prior_Height*3, 5) {
		return true
	}
	if pdf_prose_rule(input.Line.Text) {
		if input.Gap > fixedpoint.From_Integer(4) {
			return true
		}
	}
	return pdf_prose_numbered_paragraph(input.Line.Text)
}

func pdf_prose_rule(text string) (rule bool) {
	text = strings.TrimSpace(text)
	if len(text) < 3 {
		return false
	}
	marker := text[0]
	if marker != '-' {
		if marker != '=' {
			return false
		}
	}
	for index := 1; index < len(text); index++ {
		if text[index] != marker {
			return false
		}
	}
	return true
}

func pdf_prose_numbered_paragraph(text string) (numbered bool) {
	space_offset := strings.IndexByte(text, ' ')
	if space_offset <= 0 {
		return false
	}
	if space_offset > 2 {
		return false
	}
	number, parse_err := strconv.Atoi(text[:space_offset])
	if parse_err != nil {
		return false
	}
	return number >= 2
}

func pdf_prose_lines(words []Pdf_Word) (lines []Pdf_Prose_Line) {
	for _, word := range words {
		if len(lines) == 0 {
			lines = append(lines, Pdf_Prose_Line{
				Text: word.Text, Top: word.Top, Bottom: word.Bottom, X1: word.X1,
			})
			continue
		}
		if pdf_fixed_absolute(word.Top-lines[len(lines)-1].Top) >
			fixedpoint.From_Integer(3) {
			lines = append(lines, Pdf_Prose_Line{
				Text: word.Text, Top: word.Top, Bottom: word.Bottom, X1: word.X1,
			})
			continue
		}
		line := &lines[len(lines)-1]
		if pdf_prose_numeric_prefix(line.Text) {
			if word.X0-line.X1 > fixedpoint.From_Integer(8) {
				lines = append(lines, Pdf_Prose_Line{
					Text: word.Text, Top: word.Top, Bottom: word.Bottom,
					X1: word.X1, Force_Blank_Before: true,
				})
				continue
			}
		}
		if line.Text != "" {
			if !strings.HasSuffix(line.Text, " ") {
				if !strings.HasPrefix(word.Text, " ") {
					line.Text += " "
				}
			}
		}
		line.Text += word.Text
		line.Top = pdf_fixed_min(&Pdf_Fixed_Input_Min{Left: line.Top, Right: word.Top})
		line.Bottom = pdf_fixed_max(&Pdf_Fixed_Input_Max{
			Left: line.Bottom, Right: word.Bottom,
		})
		line.X1 = pdf_fixed_max(&Pdf_Fixed_Input_Max{Left: line.X1, Right: word.X1})
	}
	return lines
}

func pdf_prose_numeric_prefix(text string) (numeric bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	for index := 0; index < len(text); index++ {
		if text[index] < '0' {
			return false
		}
		if text[index] > '9' {
			return false
		}
	}
	return true
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
