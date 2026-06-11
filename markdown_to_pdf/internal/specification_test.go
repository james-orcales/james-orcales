package markdown_to_pdf_test

import (
	"strings"
	"testing"

	"github.com/james-orcales/james-orcales/markdown_to_pdf/internal"
)

// Each leaf test feeds the renderer a Markdown fragment that exercises exactly
// the feature its heading names and asserts on the resulting PDF bytes. The
// assertions read observable output — visible text, the font-selection operator
// a feature must emit, a stroke for a drawn rule, the page count — never the
// renderer's internals, so the suite stays a black box over Render and Main.

// Test_Render_Headings verifies a number-sign line becomes a heading shown in
// the large bold heading font rather than body text.
func Test_Render_Headings(t *testing.T) {
	t.Parallel()
	document := render("# Title")
	if !strings.Contains(document, "Title") {
		t.Fatal("heading text is missing")
	}
	if !strings.Contains(document, "/F1 24 Tf") {
		t.Fatal("heading is not set in the 24pt bold font")
	}
	// A lone heading still carries a blank line above it, so it never sits flush
	// at the top margin (page_height - page_margin = 786).
	if strings.Contains(document, "1 0 0 1 56 786 Tm") {
		t.Fatal("heading has no line break above it")
	}
	// A top-level heading carries a rule beneath it, drawn like a horizontal
	// rule; a deep heading does not.
	if !strings.Contains(document, " l S\n") {
		t.Fatal("heading has no rule beneath it")
	}
	if strings.Contains(render("### Sub"), " l S\n") {
		t.Fatal("a level-three heading should not get a rule")
	}
}

// Test_Render_Paragraphs verifies consecutive prose words render as body text
// in the regular body font.
func Test_Render_Paragraphs(t *testing.T) {
	t.Parallel()
	document := render("alpha beta gamma")
	if !strings.Contains(document, "alpha") {
		t.Fatal("paragraph text is missing")
	}
	if !strings.Contains(document, "/F0 11 Tf") {
		t.Fatal("paragraph is not set in the 11pt regular font")
	}
}

// Test_Render_Emphasis verifies a bold span selects the bold font and an italic
// span the italic font, while unemphasized text selects neither.
func Test_Render_Emphasis(t *testing.T) {
	t.Parallel()
	bold := render("**stout**")
	if !strings.Contains(bold, "/F1 11 Tf") {
		t.Fatal("bold span does not select the bold font")
	}
	italic := render("*lean*")
	if !strings.Contains(italic, "/F2 11 Tf") {
		t.Fatal("italic span does not select the italic font")
	}
	plain := render("stout")
	if strings.Contains(plain, "/F1 11 Tf") {
		t.Fatal("unemphasized text wrongly selects the bold font")
	}
}

// Test_Render_Code verifies a fenced block renders in the fixed-width font at
// the code point size.
func Test_Render_Code(t *testing.T) {
	t.Parallel()
	document := render("```\nspruce\n```")
	if !strings.Contains(document, "spruce") {
		t.Fatal("code text is missing")
	}
	if !strings.Contains(document, "/F4 8 Tf") {
		t.Fatal("code is not set in the 8pt fixed-width font")
	}
	// Code sits in white text on a dark-gray panel.
	if !strings.Contains(document, "0.17 0.17 0.17 rg") {
		t.Fatal("code block has no dark-gray background")
	}
	if !strings.Contains(document, "1 1 1 rg") {
		t.Fatal("code text is not white")
	}
	inline := render("use `x` here")
	if !strings.Contains(inline, "0.17 0.17 0.17 rg") {
		t.Fatal("inline code has no dark-gray background")
	}
	// Box-drawing and arrows transliterate to ASCII, not question marks.
	diagram := render("```\n┌─→\n```")
	if !strings.Contains(diagram, "+->") {
		t.Fatal("box-drawing and arrows did not transliterate to ASCII")
	}
}

// Test_Render_Lists verifies an unordered item renders its text with the raw
// dash marker consumed rather than shown.
func Test_Render_Lists(t *testing.T) {
	t.Parallel()
	document := render("- alpha\n- bravo")
	if !strings.Contains(document, "alpha") {
		t.Fatal("first item text is missing")
	}
	if !strings.Contains(document, "bravo") {
		t.Fatal("second item text is missing")
	}
	if strings.Contains(document, "- alpha") {
		t.Fatal("list marker was not consumed")
	}
}

// Test_Render_Quote verifies a block quote renders its text with the raw
// greater-than marker consumed.
func Test_Render_Quote(t *testing.T) {
	t.Parallel()
	document := render("> wisdom")
	if !strings.Contains(document, "wisdom") {
		t.Fatal("quote text is missing")
	}
	if strings.Contains(document, "> wisdom") {
		t.Fatal("quote marker was not consumed")
	}
	// A GitHub-style quote draws a soft gray left bar and sets its text in gray.
	if !strings.Contains(document, "0.75 0.75 0.75 rg") {
		t.Fatal("quote has no gray bar")
	}
	if !strings.Contains(document, "0.4 0.4 0.4 rg") {
		t.Fatal("quote text is not gray")
	}
	if !strings.Contains(document, "0.95 0.95 0.95 rg") {
		t.Fatal("quote has no light gray background")
	}
}

// Test_Render_Link verifies a link shows a blue label, carries its target in a
// clickable annotation, and underlines a multi-word label with one stroke.
func Test_Render_Link(t *testing.T) {
	t.Parallel()
	document := render("[two words](https://example.com)")
	if !strings.Contains(document, "words") {
		t.Fatal("link label is missing")
	}
	if !strings.Contains(document, "0 0 0.93 rg") {
		t.Fatal("link text is not blue")
	}
	if !strings.Contains(document, "/Subtype /Link") {
		t.Fatal("link has no clickable annotation")
	}
	if !strings.Contains(document, "/URI (https://example.com)") {
		t.Fatal("link annotation does not carry its target")
	}
	if strings.Count(document, " l S\n") != 1 {
		t.Fatal("multi-word link underline is cut at the space")
	}
}

// Test_Render_Rule verifies a thematic break emits a stroked line.
func Test_Render_Rule(t *testing.T) {
	t.Parallel()
	document := render("---")
	if !strings.Contains(document, " S\n") {
		t.Fatal("thematic break drew no stroked line")
	}
}

// Test_Render_Tables verifies a pipe table renders every cell and strokes its
// grid, with the raw pipe markers consumed.
func Test_Render_Tables(t *testing.T) {
	t.Parallel()
	document := render("| north | south |\n| - | - |\n| up | down |")
	if !strings.Contains(document, "north") {
		t.Fatal("header cell is missing")
	}
	if !strings.Contains(document, "down") {
		t.Fatal("body cell is missing")
	}
	if strings.Contains(document, "| north") {
		t.Fatal("table markers were not consumed")
	}
	// GitHub style: a gray cell grid, a bold header row, and a shaded alternate row.
	if !strings.Contains(document, "0.82 0.82 0.82 RG") {
		t.Fatal("table has no gray grid")
	}
	if !strings.Contains(document, "/F1 11 Tf") {
		t.Fatal("table header is not bold")
	}
	if !strings.Contains(document, "0.97 0.97 0.97 rg") {
		t.Fatal("table has no shaded alternate row")
	}
	// Cells parse inline markdown: a backtick span in a body cell is a code panel.
	coded := render("| h | i |\n| - | - |\n| `x` | y |")
	if !strings.Contains(coded, "0.17 0.17 0.17 rg") {
		t.Fatal("table cell did not render inline code")
	}
	// A code span wider than its column breaks into multiple panels across
	// lines instead of overflowing into the neighboring cells.
	wide := "`" + strings.Repeat("x", 60) + "`"
	broken := render("| a | b |\n| - | - |\n| " + wide + " | y |")
	if strings.Count(broken, "0.17 0.17 0.17 rg") < 2 {
		t.Fatal("over-wide code span in a cell did not break across lines")
	}
}

// Test_Pages_Single verifies a short body produces exactly one page.
func Test_Pages_Single(t *testing.T) {
	t.Parallel()
	document := render("# Title\n\nA short paragraph.")
	if strings.Count(document, "/Contents ") != 1 {
		t.Fatal("a short body did not produce exactly one page")
	}
}

// Test_Pages_Overflow verifies a body taller than one page continues onto
// further pages.
func Test_Pages_Overflow(t *testing.T) {
	t.Parallel()
	document := render(strings.Repeat("Filler paragraph here.\n\n", 300))
	if strings.Count(document, "/Contents ") < 2 {
		t.Fatal("an overflowing body did not break onto further pages")
	}
}

// Test_Document_Header verifies the output opens with the PDF version header.
func Test_Document_Header(t *testing.T) {
	t.Parallel()
	document := render("# Title")
	if !strings.HasPrefix(document, "%PDF-1.") {
		t.Fatal("output does not open with a PDF header")
	}
}

// Test_Document_Trailer verifies the output carries a cross reference table and
// ends with the end-of-file marker.
func Test_Document_Trailer(t *testing.T) {
	t.Parallel()
	document := render("# Title")
	if !strings.Contains(document, "\nxref\n") {
		t.Fatal("output has no cross reference table")
	}
	if !strings.Contains(document, "\nstartxref\n") {
		t.Fatal("output has no startxref pointer")
	}
	if !strings.Contains(document, "%%EOF") {
		t.Fatal("output has no end-of-file marker")
	}
}

// Test_Main_Output verifies Main writes a valid PDF to its injected sink and
// returns a zero status code.
func Test_Main_Output(t *testing.T) {
	t.Parallel()
	var output strings.Builder
	var diagnostics strings.Builder
	status := markdown_to_pdf.Main(&markdown_to_pdf.Main_Input{
		Markdown: []byte("# Hello"),
		Output:   &output,
		Stderr:   &diagnostics,
	})
	if status != 0 {
		t.Fatalf("status = %d, want 0", status)
	}
	if !strings.HasPrefix(output.String(), "%PDF-1.") {
		t.Fatal("Main did not write a PDF to its output")
	}
}

// Renders the Markdown source to PDF bytes and returns them as a string for
// substring assertions.
func render(source string) (document string) {
	return string(markdown_to_pdf.Render([]byte(source)))
}
