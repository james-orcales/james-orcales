package markdown_to_pdf_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"local/james-orcales/markdown_to_pdf/internal"
)

// Test_PDF_To_Markdown_Golden_Documents compares raw output bytes for every
// PDF/Markdown pair copied from MarkItDown 0.1.7.
func Test_PDF_To_Markdown_Golden_Documents(t *testing.T) {
	t.Parallel()
	fixture_names := []string{
		"academic_paper",
		"movie_booking",
		"receipt",
		"repair_estimate",
		"sparse_table",
		"empty_medical_scan",
	}
	for _, fixture_name := range fixture_names {
		pdf := read_fixture(t, fixture_name+".pdf", 64*1024*1024)
		want := read_fixture(t, fixture_name+".golden", 64*1024*1024)
		markdown, convert_err := markdown_to_pdf.PDF_To_Markdown(
			&markdown_to_pdf.PDF_To_Markdown_Input{PDF: pdf},
		)
		if convert_err != nil {
			t.Errorf("%s: PDF_To_Markdown: %v", fixture_name, convert_err)
			continue
		}
		if !bytes.Equal(markdown, want) {
			difference := first_difference(&First_Difference_Input{
				Got: markdown, Want: want,
			})
			t.Errorf(
				"%s: markdown differs: got %d bytes, want %d bytes; %s",
				fixture_name,
				len(markdown),
				len(want),
				difference,
			)
		}
	}
}

// Test_PDF_To_Markdown_Master_Format_Numbers verifies that the MasterFormat
// fixture keeps its content and joins dot-prefixed numbers to their text.
func Test_PDF_To_Markdown_Master_Format_Numbers(t *testing.T) {
	t.Parallel()
	pdf := read_fixture(t, "masterformat_partial_numbering.pdf", 64*1024*1024)
	markdown, convert_err := markdown_to_pdf.PDF_To_Markdown(
		&markdown_to_pdf.PDF_To_Markdown_Input{PDF: pdf},
	)
	if convert_err != nil {
		t.Fatalf("PDF_To_Markdown: %v", convert_err)
	}
	text := string(markdown)
	wanted := []string{
		"RFP for Construction Management Services",
		"Section 00 00 43",
		"Instructions to Respondents",
		"Ken Sargent House",
		"INTENT",
		".1 The intent",
		".2 Available information",
		"GRANDE PRAIRIE, ALBERTA",
		"Section 00 00 45",
	}
	for _, fragment := range wanted {
		if !strings.Contains(text, fragment) {
			t.Errorf("missing %q", fragment)
		}
	}
	for _, line := range strings.Split(text, "\n") {
		stripped := strings.TrimSpace(strings.ReplaceAll(line, "|", ""))
		if partial_number(stripped) {
			t.Errorf("isolated partial numbering %q", line)
		}
	}
}

// Test_PDF_To_Markdown_Table_Reconstruction verifies that painted cell
// boundaries preserve logical records without turning blank layout columns or
// page furniture into semantic table content.
func Test_PDF_To_Markdown_Table_Reconstruction(t *testing.T) {
	t.Parallel()
	content := "50 700 50 30 re f 100 700 50 30 re f " +
		"150 700 100 30 re f 250 700 300 30 re f " +
		"50 660 50 40 re f 100 660 50 40 re f " +
		"150 660 100 40 re f 250 660 300 40 re f " +
		"50 50 50 610 re f 100 50 50 610 re f " +
		"150 50 100 610 re f 250 50 300 610 re f " +
		"50 30 50 20 re f 100 30 50 20 re f " +
		"150 30 100 20 re f 250 30 300 20 re f " +
		"BT /F0 10 Tf " +
		"1 0 0 1 55 712 Tm (Release date) Tj " +
		"1 0 0 1 155 712 Tm (Identification) Tj " +
		"1 0 0 1 255 712 Tm (Scope) Tj " +
		"1 0 0 1 55 682 Tm (1986-02-15) Tj " +
		"1 0 0 1 155 682 Tm (First release) Tj " +
		"1 0 0 1 255 682 Tm (Guide revised) Tj " +
		"1 0 0 1 255 670 Tm (continued text) Tj " +
		"1 0 0 1 55 642 Tm (1987-06-01) Tj " +
		"1 0 0 1 155 642 Tm (Change 2) Tj " +
		"1 0 0 1 255 642 Tm (Words added) Tj " +
		"1 0 0 1 55 42 Tm (Page 1) Tj " +
		"1 0 0 1 255 42 Tm (History) Tj " +
		"1 0 0 1 500 42 Tm (Issue 9) Tj ET"
	pdf := pdf_specification_document(
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] "+
			"/Resources << /Font << /F0 4 0 R >> >> /Contents 5 0 R >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content),
	)
	markdown, convert_err := markdown_to_pdf.PDF_To_Markdown(
		&markdown_to_pdf.PDF_To_Markdown_Input{PDF: pdf},
	)
	if convert_err != nil {
		t.Fatalf("PDF_To_Markdown: %v", convert_err)
	}
	text := string(markdown)
	if strings.Count(text, "\n| ---") != 1 {
		t.Fatalf("source grid was not one table:\n%s", text)
	}
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "|") {
			if strings.Count(line, "|") != 4 {
				t.Fatalf("blank layout column became a table column: %q", line)
			}
		}
	}
	compact := strings.ReplaceAll(text, " ", "")
	if !strings.Contains(compact, "|1986-02-15|Firstrelease|Guiderevisedcontinuedtext|") {
		t.Fatalf("wrapped cell did not remain in its source row:\n%s", text)
	}
	if !strings.Contains(compact, "|1987-06-01|Change2|Wordsadded|") {
		t.Fatalf("source row boundaries were lost:\n%s", text)
	}
	if strings.Contains(text, "| Page 1") {
		t.Fatalf("page furniture became table content:\n%s", text)
	}
}

// Test_PDF_To_Markdown_Parser_Validation verifies invalid headers, malformed
// objects, and encryption fail instead of producing partial Markdown.
func Test_PDF_To_Markdown_Parser_Validation(t *testing.T) {
	t.Parallel()
	cases := [][]byte{
		[]byte("not a PDF"),
		[]byte("%PDF-1.4\n1 0 obj <<"),
		[]byte("%PDF-1.4\n/Encrypt 2 0 R\n"),
	}
	for _, source := range cases {
		markdown, convert_err := markdown_to_pdf.PDF_To_Markdown(
			&markdown_to_pdf.PDF_To_Markdown_Input{PDF: source},
		)
		if convert_err == nil {
			t.Errorf("source %q succeeded with %q", source, markdown)
		}
	}
}

// Test_PDF_To_Markdown_Resource_Limits pins each public parser bound and
// exercises the input-size and nesting checks without an RSS assertion.
func Test_PDF_To_Markdown_Resource_Limits(t *testing.T) {
	if markdown_to_pdf.PDF_OBJECT_COUNT_MAX != 262144 {
		t.Fatal("indirect-object limit changed")
	}
	if markdown_to_pdf.PDF_PAGE_COUNT_MAX != 65536 {
		t.Fatal("page limit changed")
	}
	if markdown_to_pdf.PDF_DEPTH_MAX != 128 {
		t.Fatal("depth limit changed")
	}
	if markdown_to_pdf.PDF_STREAM_BYTES_MAX != 64*1024*1024 {
		t.Fatal("decoded-stream limit changed")
	}
	if markdown_to_pdf.PDF_DECODED_BYTES_MAX != 256*1024*1024 {
		t.Fatal("cumulative decoded-data limit changed")
	}
	too_large := make([]byte, markdown_to_pdf.PDF_BYTES_MAX+1)
	copy(too_large, "%PDF-1.4")
	if _, convert_err := markdown_to_pdf.PDF_To_Markdown(
		&markdown_to_pdf.PDF_To_Markdown_Input{PDF: too_large},
	); convert_err == nil {
		t.Fatal("oversized PDF input succeeded")
	}
	nested := "%PDF-1.4\n1 0 obj " + strings.Repeat("[", 130) + "null" +
		strings.Repeat("]", 130) + " endobj\n"
	if _, convert_err := markdown_to_pdf.PDF_To_Markdown(
		&markdown_to_pdf.PDF_To_Markdown_Input{PDF: []byte(nested)},
	); convert_err == nil {
		t.Fatal("over-deep PDF value succeeded")
	}
}

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
	// at the top margin (PAGE_HEIGHT - PAGE_MARGIN = 786).
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

// Test_Main_Dependencies verifies that the temporary path and external opener
// stay injectable because neither is deterministic process state.
func Test_Main_Dependencies(t *testing.T) {
	t.Parallel()
	written_path := ""
	opened_path := ""
	status := markdown_to_pdf.Main(&markdown_to_pdf.Main_Input{
		Arguments:    []string{"markdown_to_pdf", "golden"},
		Output:       io.Discard,
		Error_Output: io.Discard,
		Write_File: func(path string, document []byte) (err error) {
			written_path = path
			return nil
		},
		Temporary_Directory: "/injected",
		Open_Path: func(path string) (err error) {
			opened_path = path
			return nil
		},
	})
	if status != 0 {
		t.Fatalf("status = %d, want 0", status)
	}
	want_path := "/injected/markdown_to_pdf_golden.pdf"
	if written_path != want_path {
		t.Fatalf("written path = %q, want %q", written_path, want_path)
	}
	if opened_path != want_path {
		t.Fatalf("opened path = %q, want %q", opened_path, want_path)
	}
}

// Test_Main_Output verifies that package main only binds capabilities because
// the injected entry point owns the complete command policy.
func Test_Main_Output(t *testing.T) {
	t.Parallel()
	var output strings.Builder
	var diagnostics strings.Builder
	written_path := ""
	var written_document []byte
	status := markdown_to_pdf.Main(&markdown_to_pdf.Main_Input{
		Arguments:    []string{"markdown_to_pdf", "render", "source.md", "-out=result.pdf"},
		Output:       &output,
		Error_Output: &diagnostics,
		Open_File: func(path string) (file io.ReadCloser, err error) {
			if path != "source.md" {
				t.Fatalf("opened path = %q, want source.md", path)
			}
			return io.NopCloser(strings.NewReader("# Hello")), nil
		},
		Write_File: func(path string, document []byte) (err error) {
			written_path = path
			written_document = append([]byte{}, document...)
			return nil
		},
		Path_Exists:         func(string) (exists bool) { return false },
		Temporary_Directory: "/tmp",
		Open_Path:           func(string) (err error) { return nil },
	})
	if status != 0 {
		t.Fatalf("status = %d, want 0", status)
	}
	if written_path != "result.pdf" {
		t.Fatalf("written path = %q, want result.pdf", written_path)
	}
	if !bytes.HasPrefix(written_document, []byte("%PDF-1.")) {
		t.Fatal("Main did not write a PDF document")
	}
	if diagnostics.Len() != 0 {
		t.Fatalf("diagnostics = %q, want empty", diagnostics.String())
	}
}

// First_Difference_Input groups two byte slices for mismatch reporting.
type First_Difference_Input struct {
	// Got is the converter output.
	Got []byte
	// Want is the golden output.
	Want []byte
}

func first_difference(input *First_Difference_Input) (description string) {
	shared_size := len(input.Got)
	if len(input.Want) < shared_size {
		shared_size = len(input.Want)
	}
	offset := 0
	for offset < shared_size {
		if input.Got[offset] != input.Want[offset] {
			break
		}
		offset++
	}
	got_end := offset + 80
	if got_end > len(input.Got) {
		got_end = len(input.Got)
	}
	want_end := offset + 80
	if want_end > len(input.Want) {
		want_end = len(input.Want)
	}
	return fmt.Sprintf(
		"offset %d: got %q, want %q",
		offset,
		input.Got[offset:got_end],
		input.Want[offset:want_end],
	)
}

func partial_number(text string) (yes bool) {
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

func read_fixture(t *testing.T, name string, bytes_max int64) (contents []byte) {
	t.Helper()
	path := "testdata/" + name
	file, open_err := os.Open(path)
	if open_err != nil {
		t.Fatalf("open %s: %v", path, open_err)
	}
	defer file.Close()
	info, stat_err := file.Stat()
	if stat_err != nil {
		t.Fatalf("stat %s: %v", path, stat_err)
	}
	if info.Size() > bytes_max {
		t.Fatalf("%s exceeds %d bytes", path, bytes_max)
	}
	contents = make([]byte, info.Size())
	_, read_err := io.ReadFull(file, contents)
	if read_err != nil {
		t.Fatalf("read %s: %v", path, read_err)
	}
	return contents
}

func pdf_specification_document(objects ...string) (pdf []byte) {
	var output strings.Builder
	output.WriteString("%PDF-1.4\n")
	for object_index, object := range objects {
		fmt.Fprintf(&output, "%d 0 obj\n%s\nendobj\n", object_index+1, object)
	}
	return []byte(output.String())
}

// Renders the Markdown source to PDF bytes and returns them as a string for
// substring assertions.
func render(source string) (document string) {
	return string(markdown_to_pdf.Render([]byte(source)))
}
