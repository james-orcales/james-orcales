package markdown_to_pdf

import (
	"bytes"
	standard_zlib "compress/zlib"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/ascii85"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"local/james-orcales/shared/math/fixedpoint"
)

// Test_Pdf_Stream_Filters covers every decoder reachable through PDF streams.
func Test_Pdf_Stream_Filters(t *testing.T) {
	t.Parallel()
	hexadecimal, hexadecimal_err := pdf_decode_ascii_hexadecimal([]byte("48656c6c6f>"))
	if hexadecimal_err != nil {
		t.Fatalf("ASCIIHex: %v", hexadecimal_err)
	}
	if !bytes.Equal(hexadecimal, []byte("Hello")) {
		t.Fatalf("ASCIIHex = %q", hexadecimal)
	}

	ascii85_buffer := make([]byte, ascii85.MaxEncodedLen(len("Hello")))
	ascii85_size := ascii85.Encode(ascii85_buffer, []byte("Hello"))
	ascii85_decoded, ascii85_err := pdf_decode_ascii85(ascii85_buffer[:ascii85_size])
	if ascii85_err != nil {
		t.Fatalf("ASCII85: %v", ascii85_err)
	}
	if !bytes.Equal(ascii85_decoded, []byte("Hello")) {
		t.Fatalf("ASCII85 = %q", ascii85_decoded)
	}

	var flate_buffer bytes.Buffer
	flate_writer := standard_zlib.NewWriter(&flate_buffer)
	if _, write_err := flate_writer.Write([]byte("Hello")); write_err != nil {
		t.Fatalf("Flate write: %v", write_err)
	}
	if close_err := flate_writer.Close(); close_err != nil {
		t.Fatalf("Flate close: %v", close_err)
	}
	flate, flate_err := pdf_decode_flate(flate_buffer.Bytes())
	if flate_err != nil {
		t.Fatalf("Flate: %v", flate_err)
	}
	if !bytes.Equal(flate, []byte("Hello")) {
		t.Fatalf("Flate = %q", flate)
	}

	lzw, lzw_err := pdf_decode_lzw(pdf_test_lzw_codes([]int{256, 'H', 'i', 257}), 1)
	if lzw_err != nil {
		t.Fatalf("LZW: %v", lzw_err)
	}
	if !bytes.Equal(lzw, []byte("Hi")) {
		t.Fatalf("LZW = %q", lzw)
	}

	run_size, run_size_err := pdf_decode_run_size([]byte{4, 'H', 'e', 'l', 'l', 'o', 128})
	if run_size_err != nil {
		t.Fatalf("RunLength: %v", run_size_err)
	}
	if !bytes.Equal(run_size, []byte("Hello")) {
		t.Fatalf("RunLength = %q", run_size)
	}
}

// Test_Pdf_Stream_Predictors covers TIFF and PNG row reconstruction.
func Test_Pdf_Stream_Predictors(t *testing.T) {
	t.Parallel()
	tiff_parameters := pdf_test_predictor_parameters(2)
	tiff, tiff_err := pdf_apply_predictor([]byte{1, 1, 1}, tiff_parameters)
	if tiff_err != nil {
		t.Fatalf("TIFF predictor: %v", tiff_err)
	}
	if !bytes.Equal(tiff, []byte{1, 2, 3}) {
		t.Fatalf("TIFF predictor = %v", tiff)
	}

	png_parameters := pdf_test_predictor_parameters(15)
	png, png_err := pdf_apply_predictor([]byte{1, 1, 1, 1}, png_parameters)
	if png_err != nil {
		t.Fatalf("PNG predictor: %v", png_err)
	}
	if !bytes.Equal(png, []byte{1, 2, 3}) {
		t.Fatalf("PNG predictor = %v", png)
	}
}

// Test_Pdf_Object_Indexes covers incremental definitions, classic xref text,
// xref streams, and compressed object streams.
func Test_Pdf_Object_Indexes(t *testing.T) {
	t.Parallel()
	source := []byte("%PDF-1.4\n" +
		"1 0 obj << /Flag true >> endobj\n" +
		"xref\n0 2\n0000000000 65535 f\n0000000009 00000 n\n" +
		"3 0 obj << /Type /XRef /Length 0 >> stream\n\nendstream\nendobj\n" +
		"1 0 obj << /Flag false >> endobj\n")
	document, parse_err := pdf_parse_document(source)
	if parse_err != nil {
		t.Fatalf("parse xref variants: %v", parse_err)
	}
	flag := document.Objects[1].Value.Dictionary["Flag"]
	if flag.Boolean {
		t.Fatal("incremental update did not replace the earlier object")
	}
	if pdf_dictionary_name(document, document.Objects[3].Value, "Type") != "XRef" {
		t.Fatal("xref stream object was not retained")
	}

	decoded := []byte("5 0 << /Answer 42 >>")
	object_stream := Pdf_Value{
		Kind: PDF_VALUE_DICTIONARY,
		Dictionary: map[string]Pdf_Value{
			"Type":  {Kind: PDF_VALUE_NAME, Name: "ObjStm"},
			"N":     {Kind: PDF_VALUE_NUMBER, Integer: 1},
			"First": {Kind: PDF_VALUE_NUMBER, Integer: 4},
		},
		Stream: decoded,
	}
	document = &Pdf_Document{
		Objects: map[int]Pdf_Indirect_Object{
			10: {Value: object_stream, Offset: 10},
		},
		Object_Streams_Decoded: make(map[int]bool),
	}
	if decode_err := pdf_decode_object_streams(document); decode_err != nil {
		t.Fatalf("object stream: %v", decode_err)
	}
	answer := document.Objects[5].Value.Dictionary["Answer"]
	if answer.Integer != 42 {
		t.Fatalf("compressed object answer = %d", answer.Integer)
	}
}

// Test_Pdf_Unicode_And_Form covers ToUnicode text and nested Form XObjects.
func Test_Pdf_Unicode_And_Form(t *testing.T) {
	t.Parallel()
	font := Pdf_Font{Unicode_Map: make(map[string]string)}
	cmap := []byte("1 begincodespacerange <00> <ff> endcodespacerange " +
		"2 beginbfchar <41> <0041> <42> <03a9> endbfchar")
	if cmap_err := pdf_parse_unicode_cmap(cmap, &font); cmap_err != nil {
		t.Fatalf("ToUnicode: %v", cmap_err)
	}
	if font.Unicode_Map["A"] != "A" {
		t.Fatalf("ToUnicode A = %q", font.Unicode_Map["A"])
	}
	if font.Unicode_Map["B"] != "Ω" {
		t.Fatalf("ToUnicode B = %q", font.Unicode_Map["B"])
	}

	pdf := pdf_test_document(
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] "+
			"/Resources << /Font << /F0 4 0 R >> /XObject << /Fm0 5 0 R >> >> "+
			"/Contents 6 0 R >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
		"<< /Type /XObject /Subtype /Form /BBox [0 0 100 100] /Length 38 >>\n"+
			"stream\nBT /F0 12 Tf 1 0 0 1 0 0 Tm (Nested) Tj ET\nendstream",
		"<< /Length 19 >>\nstream\nq /Fm0 Do Q\nendstream",
	)
	markdown, convert_err := PDF_To_Markdown(&PDF_To_Markdown_Input{PDF: pdf})
	if convert_err != nil {
		t.Fatalf("Form XObject: %v", convert_err)
	}
	if !strings.Contains(string(markdown), "Nested") {
		t.Fatalf("Form XObject output = %q", markdown)
	}
}

// Test_PDF_To_Markdown_Split_Content_Value verifies a Contents array acts as
// one byte sequence when a PDF value crosses an indirect-stream boundary.
func Test_PDF_To_Markdown_Split_Content_Value(t *testing.T) {
	t.Parallel()
	first := "BT /F0 12 Tf 1 0 0 1 72 720 Tm (Hel"
	second := "lo) Tj ET"
	pdf := pdf_test_document(
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] "+
			"/Resources << /Font << /F0 4 0 R >> >> /Contents [5 0 R 6 0 R] >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(first), first),
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(second), second),
	)
	markdown, convert_err := PDF_To_Markdown(&PDF_To_Markdown_Input{PDF: pdf})
	if convert_err != nil {
		t.Fatalf("split content value: %v", convert_err)
	}
	if !bytes.Contains(markdown, []byte("Hello")) {
		t.Fatalf("split content output = %q", markdown)
	}
}

// Test_Pdf_Form_Table_Continuation_Rows pins the source geometry that caused
// wrapped cells to escape as prose and footer positions to become columns.
func Test_Pdf_Form_Table_Continuation_Rows(t *testing.T) {
	t.Parallel()
	words := pdf_test_continuation_table_words()
	content, is_form := pdf_form_content(&Pdf_Form_Content_Input{
		Words: words, Page_Width: fixedpoint.From_Integer(612),
		Page_Height: fixedpoint.From_Integer(792),
	})
	if !is_form {
		t.Fatal("table page was classified as prose")
	}
	lines := strings.Split(content, "\n")
	separator_count := 0
	for _, line := range lines {
		if !strings.HasPrefix(line, "|") {
			continue
		}
		if strings.Contains(line, "---") {
			separator_count++
		}
		if strings.Count(line, "|") != 5 {
			t.Fatalf("table row has an invented column: %q", line)
		}
	}
	if separator_count != 1 {
		t.Fatalf("source table became %d Markdown tables:\n%s", separator_count, content)
	}
	row_cases := [][]string{
		{"Word", "(part of speech)", "Approved meaning/", "ALTERNATIVES"},
		{"CAN (v),", "COULD", "No other verb forms.", "EXPLOSION."},
		{"YOU CAN CLEAN THE DRAIN HOLES WITH THE CLEANING TOOL."},
		{"Do not use", "IF YOU DO NOT OBEY", "EXPLOSION CAN OCCUR."},
		{"WILL (v)", "No other verb forms.", "WILL HELP YOU."},
	}
	for _, parts := range row_cases {
		pdf_test_table_row_contains(&pdf_test_table_row_contains_input{
			T: t, Lines: lines, Parts: parts,
		})
	}
}

// Test_Pdf_Form_Table_Blank_Columns verifies that retained blank glyphs do not
// turn layout positions or fixed page furniture into semantic table cells.
func Test_Pdf_Form_Table_Blank_Columns(t *testing.T) {
	t.Parallel()
	words := pdf_test_blank_column_words()
	content, is_form := pdf_form_content(&Pdf_Form_Content_Input{
		Words: words, Page_Width: fixedpoint.From_Integer(612),
		Page_Height: fixedpoint.From_Integer(792),
	})
	if !is_form {
		t.Fatal("blank-cell table page was classified as prose")
	}
	separator_count := 0
	for _, line := range strings.Split(content, "\n") {
		if strings.Contains(line, "---") {
			separator_count++
		}
		if strings.HasPrefix(line, "|") {
			if strings.Count(line, "|") != 3 {
				t.Fatalf("blank layout columns were retained: %q", line)
			}
		}
		if strings.Contains(line, "Page 1") {
			if strings.HasPrefix(line, "|") {
				t.Fatalf("page furniture became a table row: %q", line)
			}
		}
		if strings.Contains(line, "Alphanumeric identifiers") {
			if strings.HasPrefix(line, "|") {
				t.Fatalf("section heading became a table row: %q", line)
			}
		}
		if strings.Contains(line, "writing.") {
			if strings.HasPrefix(line, "|") {
				t.Fatalf("wrapped prose fragment became a table row: %q", line)
			}
		}
	}
	if separator_count != 1 {
		t.Fatalf("page furniture created a second table:\n%s", content)
	}
	if !strings.Contains(content, "writing.\n\nPage 1") {
		t.Fatalf("page furniture joined the final content paragraph:\n%s", content)
	}
}

func pdf_test_blank_column_words() (words []Pdf_Word) {
	inputs := []pdf_test_table_word_input{
		{Text: "Example one", X: 72, Top: 100},
		{Text: " ", X: 172, Top: 100},
		{Text: " ", X: 272, Top: 100},
		{Text: " ", X: 372, Top: 100},
		{Text: " ", X: 472, Top: 100},
		{Text: "(5 words)", X: 552, Top: 100},
		{Text: "Example two", X: 72, Top: 120},
		{Text: " ", X: 172, Top: 120},
		{Text: " ", X: 272, Top: 120},
		{Text: " ", X: 372, Top: 120},
		{Text: " ", X: 472, Top: 120},
		{Text: "(7 words)", X: 552, Top: 120},
		{Text: "4. Alphanumeric identifiers", X: 72, Top: 140},
		{Text: " ", X: 172, Top: 140},
		{Text: " ", X: 272, Top: 140},
		{Text: " ", X: 372, Top: 140},
		{Text: " ", X: 472, Top: 140},
		{Text: " ", X: 552, Top: 140},
		{Text: "Prose row one", X: 72, Top: 160},
		{Text: "Prose row two", X: 72, Top: 180},
		{Text: "Prose row three", X: 72, Top: 200},
		{Text: "Prose row four", X: 72, Top: 220},
		{Text: "Prose row five", X: 72, Top: 240},
		{Text: "Prose row six", X: 72, Top: 260},
		{Text: "Prose row seven", X: 72, Top: 280},
		{Text: "Prose row eight", X: 72, Top: 300},
		{Text: "writing.", X: 72, Top: 340},
		{Text: " ", X: 172, Top: 340},
		{Text: " ", X: 272, Top: 340},
		{Text: " ", X: 372, Top: 340},
		{Text: " ", X: 472, Top: 340},
		{Text: " ", X: 552, Top: 340},
		{Text: "Page 1", X: 72, Top: 720},
		{Text: "Part 1", X: 272, Top: 720},
		{Text: "Issue 9", X: 552, Top: 720},
	}
	words = make([]Pdf_Word, 0, len(inputs))
	for input_index := range inputs {
		words = append(words, pdf_test_table_word(&inputs[input_index]))
	}
	return words
}

// Test_Pdf_Form_Table_Wide_Continuation_Row verifies that a wrapped cell can
// span most of the page without escaping from a blank-cell-preserving table.
func Test_Pdf_Form_Table_Wide_Continuation_Row(t *testing.T) {
	t.Parallel()
	row := Pdf_Form_Row{
		Words: []Pdf_Word{
			pdf_test_table_word(&pdf_test_table_word_input{
				Text: "A long explanation remains in the first " +
					"semantic table cell.",
				X: 72, Top: 100,
			}),
			pdf_test_table_word(&pdf_test_table_word_input{
				Text: " ", X: 172, Top: 100,
			}),
			pdf_test_table_word(&pdf_test_table_word_input{
				Text: " ", X: 272, Top: 100,
			}),
			pdf_test_table_word(&pdf_test_table_word_input{
				Text: " ", X: 372, Top: 100,
			}),
			pdf_test_table_word(&pdf_test_table_word_input{
				Text: " ", X: 472, Top: 100,
			}),
			pdf_test_table_word(&pdf_test_table_word_input{
				Text: " ", X: 552, Top: 100,
			}),
		},
		Is_Paragraph: true,
	}
	columns := []fixedpoint.Number{
		fixedpoint.From_Integer(72), fixedpoint.From_Integer(172),
		fixedpoint.From_Integer(272), fixedpoint.From_Integer(372),
		fixedpoint.From_Integer(472), fixedpoint.From_Integer(552),
	}
	if !pdf_table_continuation_row(&row, columns) {
		t.Fatal("wide wrapped cell escaped from its retained blank-cell table")
	}
}

// Test_Pdf_Form_Paragraph_Annotations verifies that right-aligned sentence
// counts do not turn descriptive prose into one-row Markdown tables.
func Test_Pdf_Form_Paragraph_Annotations(t *testing.T) {
	t.Parallel()
	inputs := []pdf_test_table_word_input{
		{Text: "The pressure is more than a set", X: 72, Top: 100},
		{Text: "value.", X: 72, Top: 112},
		{Text: " ", X: 180, Top: 112},
		{Text: " ", X: 310, Top: 112},
		{Text: "(one paragraph, 5 sentences)", X: 440, Top: 112},
		{Text: "The fittings attach the manifold.", X: 72, Top: 132},
		{Text: " ", X: 72, Top: 144},
		{Text: " ", X: 180, Top: 144},
		{Text: " ", X: 310, Top: 144},
		{Text: "(one paragraph, 1 sentence)", X: 440, Top: 144},
	}
	words := make([]Pdf_Word, 0, len(inputs))
	for input_index := range inputs {
		words = append(words, pdf_test_table_word(&inputs[input_index]))
	}
	content, is_form := pdf_form_content(&Pdf_Form_Content_Input{
		Words: words, Page_Width: fixedpoint.From_Integer(612),
		Page_Height: fixedpoint.From_Integer(792),
	})
	if !is_form {
		t.Fatal("annotated paragraph page was classified as plain extraction")
	}
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "|") {
			t.Fatalf("paragraph annotation became a table row: %q", line)
		}
	}
	normalized := strings.Join(strings.Fields(content), " ")
	if !strings.Contains(normalized, "value. (one paragraph, 5 sentences)") {
		t.Fatalf("sentence ending escaped from annotated prose:\n%s", content)
	}
	if !strings.Contains(content, "(one paragraph, 1 sentence)") {
		t.Fatalf("standalone paragraph annotation was lost:\n%s", content)
	}
}

// Test_Pdf_Ruled_Table_Top_Glyph_Overlap verifies that a glyph extending above
// its cell border is emitted only through the table that contains its center.
func Test_Pdf_Ruled_Table_Top_Glyph_Overlap(t *testing.T) {
	t.Parallel()
	label := pdf_test_table_word(&pdf_test_table_word_input{
		Text: "Non-STE:", X: 50, Top: 98,
	})
	sentence := pdf_test_table_word(&pdf_test_table_word_input{
		Text: "Use a cable.", X: 150, Top: 98,
	})
	explanation := pdf_test_table_word(&pdf_test_table_word_input{
		Text: "Cable explanation.", X: 150, Top: 112,
	})
	second_label := pdf_test_table_word(&pdf_test_table_word_input{
		Text: "Non-STE:", X: 50, Top: 148,
	})
	second_sentence := pdf_test_table_word(&pdf_test_table_word_input{
		Text: "Use a rope.", X: 150, Top: 148,
	})
	rows := []Pdf_Form_Row{
		{Words: []Pdf_Word{label, sentence}, Text: "Non-STE: Use a cable."},
		{Words: []Pdf_Word{explanation}, Text: "Cable explanation."},
		{Words: []Pdf_Word{second_label, second_sentence},
			Text: "Non-STE: Use a rope."},
	}
	table := Pdf_Ruled_Table{
		Top: fixedpoint.From_Integer(100), Bottom: fixedpoint.From_Integer(130),
		Rows: []Pdf_Ruled_Row{{
			Top: fixedpoint.From_Integer(100), Bottom: fixedpoint.From_Integer(130),
			Cells: []Pdf_Rectangle{
				{X0: fixedpoint.From_Integer(40), X1: fixedpoint.From_Integer(140),
					Top:    fixedpoint.From_Integer(100),
					Bottom: fixedpoint.From_Integer(130)},
				{X0: fixedpoint.From_Integer(140), X1: fixedpoint.From_Integer(300),
					Top:    fixedpoint.From_Integer(100),
					Bottom: fixedpoint.From_Integer(130)},
			},
		}},
	}
	second_table := Pdf_Ruled_Table{
		Top: fixedpoint.From_Integer(150), Bottom: fixedpoint.From_Integer(180),
		Rows: []Pdf_Ruled_Row{{
			Top: fixedpoint.From_Integer(150), Bottom: fixedpoint.From_Integer(180),
			Cells: []Pdf_Rectangle{
				{X0: fixedpoint.From_Integer(40), X1: fixedpoint.From_Integer(140),
					Top:    fixedpoint.From_Integer(150),
					Bottom: fixedpoint.From_Integer(180)},
				{X0: fixedpoint.From_Integer(140), X1: fixedpoint.From_Integer(300),
					Top:    fixedpoint.From_Integer(150),
					Bottom: fixedpoint.From_Integer(180)},
			},
		}},
	}
	content := pdf_format_ruled_page(&Pdf_Format_Ruled_Page_Input{
		Rows: rows,
		Words: []Pdf_Word{
			label, sentence, explanation, second_label, second_sentence,
		},
		Tables: []Pdf_Ruled_Table{table, second_table},
	})
	if strings.Count(content, "Use a cable.") != 1 {
		t.Fatalf("top cell text was emitted twice:\n%s", content)
	}
	if !strings.Contains(content, "|\n\n| Non-STE: | Use a rope.") {
		t.Fatalf("adjacent ruled tables were not separated:\n%s", content)
	}
}

// Test_PDF_To_Markdown_Ruled_Table_Rows verifies that cell rectangles, not
// equal text spacing, separate compact records from wrapped cell lines.
func Test_PDF_To_Markdown_Ruled_Table_Rows(t *testing.T) {
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
	pdf := pdf_test_document(
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] "+
			"/Resources << /Font << /F0 4 0 R >> >> /Contents 5 0 R >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content),
	)
	markdown, convert_err := PDF_To_Markdown(&PDF_To_Markdown_Input{PDF: pdf})
	if convert_err != nil {
		t.Fatalf("ruled table: %v", convert_err)
	}
	lines := strings.Split(string(markdown), "\n")
	if strings.Count(string(markdown), "\n| ---") != 1 {
		t.Fatalf("ruled table was split:\n%s", markdown)
	}
	for _, line := range lines {
		if strings.HasPrefix(line, "|") {
			if strings.Count(line, "|") != 4 {
				t.Fatalf("empty grid column was retained: %q", line)
			}
		}
	}
	pdf_test_table_row_contains(&pdf_test_table_row_contains_input{
		T: t, Lines: lines,
		Parts: []string{"1986-02-15", "Guide revised continued text"},
	})
	pdf_test_table_row_contains(&pdf_test_table_row_contains_input{
		T: t, Lines: lines, Parts: []string{"1987-06-01", "Words added"},
	})
	for _, line := range lines {
		if strings.Contains(line, "Page 1") {
			if strings.HasPrefix(line, "|") {
				t.Fatalf("page footer became a table row: %q", line)
			}
		}
	}
}

func pdf_test_continuation_table_words() (words []Pdf_Word) {
	inputs := pdf_test_table_definition_word_inputs()
	inputs = append(inputs, pdf_test_table_example_word_inputs()...)
	inputs = append(inputs, pdf_test_table_warning_word_inputs()...)
	for input_index := range inputs {
		words = append(words, pdf_test_table_word(&inputs[input_index]))
	}
	return words
}

func pdf_test_table_definition_word_inputs() (inputs []pdf_test_table_word_input) {
	return []pdf_test_table_word_input{
		{Text: "Example:", X: 72, Top: 100},
		{Text: "Word", X: 72, Top: 120},
		{Text: "Approved meaning/", X: 180, Top: 120},
		{Text: "STE EXAMPLE", X: 310, Top: 120},
		{Text: "Non-STE example", X: 440, Top: 120},
		{Text: "(part of speech)", X: 72, Top: 132},
		{Text: "ALTERNATIVES", X: 180, Top: 132},
		{Text: "CAN (v),", X: 72, Top: 150},
		{Text: "Auxiliary modal verb that", X: 180, Top: 150},
		{Text: "A MIXTURE OF FUEL", X: 310, Top: 150},
		{Text: " ", X: 440, Top: 150},
		{Text: "CAN", X: 72, Top: 162},
		{Text: "means to be possible, to", X: 180, Top: 162},
		{Text: "AND OXYGEN CAN", X: 310, Top: 162},
		{Text: "COULD", X: 72, Top: 174},
		{Text: "be able to, or to be", X: 180, Top: 178},
		{Text: "CAUSE AN", X: 310, Top: 174},
		{Text: "permitted to", X: 180, Top: 188},
		{Text: "EXPLOSION.", X: 310, Top: 188},
		{Text: "No other verb", X: 105, Top: 202},
		{Text: "forms.", X: 105, Top: 214},
	}
}

func pdf_test_table_example_word_inputs() (inputs []pdf_test_table_word_input) {
	return []pdf_test_table_word_input{
		{Text: " ", X: 72, Top: 232},
		{Text: " ", X: 180, Top: 232},
		{Text: "YOU CAN CLEAN THE", X: 310, Top: 232},
		{Text: " ", X: 440, Top: 232},
		{Text: "DRAIN HOLES WITH", X: 310, Top: 244},
		{Text: "THE CLEANING TOOL.", X: 310, Top: 256},
		{Text: " ", X: 72, Top: 274},
		{Text: " ", X: 180, Top: 274},
		{Text: "YOU CAN OPERATE", X: 310, Top: 274},
		{Text: " ", X: 440, Top: 274},
		{Text: "THE VEHICLE AFTER", X: 310, Top: 286},
		{Text: "THE INSPECTION IS COMPLETED.", X: 310, Top: 298},
		{Text: " ", X: 72, Top: 316},
		{Text: "Do not use", X: 220, Top: 316},
		{Text: "IF YOU DO NOT OBEY", X: 310, Top: 316},
		{Text: "If you do not obey this", X: 440, Top: 316},
		{Text: "COULD (v) to", X: 220, Top: 328},
		{Text: "THIS WARNING, AN", X: 310, Top: 328},
		{Text: "warning, an explosion", X: 440, Top: 328},
		{Text: "show possibility", X: 220, Top: 340},
		{Text: "EXPLOSION CAN", X: 310, Top: 340},
		{Text: "could occur.", X: 440, Top: 340},
		{Text: "OCCUR.", X: 310, Top: 352},
	}
}

func pdf_test_table_warning_word_inputs() (inputs []pdf_test_table_word_input) {
	return []pdf_test_table_word_input{
		{Text: "WILL (v)", X: 72, Top: 374},
		{Text: "Auxiliary modal verb that", X: 180, Top: 374},
		{Text: "WARNINGS AND", X: 310, Top: 374},
		{Text: " ", X: 440, Top: 374},
		{Text: "shows simple future tense", X: 180, Top: 386},
		{Text: "CAUTIONS IN THIS MANUAL", X: 310, Top: 386},
		{Text: "No other verb forms.", X: 105, Top: 398},
		{Text: "WILL HELP YOU.", X: 310, Top: 398},
		{Text: "Issue 9", X: 72, Top: 720},
		{Text: "Part 2 - Dictionary", X: 269, Top: 720},
		{Text: "Page 2-0-7", X: 491, Top: 720},
	}
}

type pdf_test_table_word_input struct {
	Text string
	X    int64
	Top  int64
}

func pdf_test_table_word(input *pdf_test_table_word_input) (word Pdf_Word) {
	word.Text = input.Text
	word.X0 = fixedpoint.From_Integer(input.X)
	word.X1 = fixedpoint.From_Integer(input.X + int64(len(input.Text))*5)
	word.Top = fixedpoint.From_Integer(input.Top)
	word.Bottom = fixedpoint.From_Integer(input.Top + 10)
	return word
}

type pdf_test_table_row_contains_input struct {
	T     *testing.T
	Lines []string
	Parts []string
}

func pdf_test_table_row_contains(input *pdf_test_table_row_contains_input) {
	input.T.Helper()
	for _, line := range input.Lines {
		contains_all := true
		for _, part := range input.Parts {
			if !strings.Contains(line, part) {
				contains_all = false
				break
			}
		}
		if contains_all {
			return
		}
	}
	input.T.Fatalf("no table row contains %q", input.Parts)
}

// Test_Pdf_Resource_Budgets exercises object, page, reference, stream, and
// cumulative decode limits without relying only on exported constants.
func Test_Pdf_Resource_Budgets(t *testing.T) {
	document := &Pdf_Document{
		Source: []byte("null endobj"), Objects: make(map[int]Pdf_Indirect_Object),
		Object_Headers_Count: PDF_OBJECT_COUNT_MAX,
	}
	_, object_err := pdf_parse_indirect_object(document, &Pdf_Object_Header{})
	if object_err == nil {
		t.Fatal("indirect-object limit did not fail")
	}

	page := Pdf_Value{
		Kind: PDF_VALUE_DICTIONARY,
		Dictionary: map[string]Pdf_Value{
			"Type": {Kind: PDF_VALUE_NAME, Name: "Page"},
		},
	}
	kids := make([]Pdf_Value, PDF_PAGE_COUNT_MAX+1)
	for page_index := range kids {
		kids[page_index] = page
	}
	pages_node := Pdf_Value{
		Kind: PDF_VALUE_DICTIONARY,
		Dictionary: map[string]Pdf_Value{
			"Type": {Kind: PDF_VALUE_NAME, Name: "Pages"},
			"Kids": {Kind: PDF_VALUE_ARRAY, Array: kids},
		},
	}
	document = &Pdf_Document{Objects: make(map[int]Pdf_Indirect_Object)}
	if _, page_err := pdf_collect_pages(
		document, pages_node, Pdf_Page_Inheritance{}, 0, nil,
	); page_err == nil {
		t.Fatal("page limit did not fail")
	}

	objects := make(map[int]Pdf_Indirect_Object)
	for reference_index := 0; reference_index <= PDF_DEPTH_MAX+1; reference_index++ {
		objects[reference_index] = Pdf_Indirect_Object{Value: Pdf_Value{
			Kind: PDF_VALUE_REFERENCE, Reference_Object_Number: reference_index + 1,
		}}
	}
	objects[PDF_DEPTH_MAX+2] = Pdf_Indirect_Object{
		Value: Pdf_Value{Kind: PDF_VALUE_NULL},
	}
	document = &Pdf_Document{Objects: objects}
	_, reference_err := pdf_resolve_value(document, objects[0].Value, 0)
	if reference_err == nil {
		t.Fatal("reference-depth limit did not fail")
	}

	if _, stream_err := pdf_read_bounded(strings.NewReader("ab"), 1); stream_err == nil {
		t.Fatal("decoded-stream limit did not fail")
	}
	document = &Pdf_Document{
		Objects:             make(map[int]Pdf_Indirect_Object),
		Decoded_Bytes_Count: PDF_DECODED_BYTES_MAX,
	}
	stream := Pdf_Value{
		Kind: PDF_VALUE_DICTIONARY, Dictionary: make(map[string]Pdf_Value),
		Stream: []byte("x"),
	}
	if _, decoded_err := pdf_decode_stream(document, stream); decoded_err == nil {
		t.Fatal("cumulative decoded-data limit did not fail")
	}
}

func pdf_test_document(objects ...string) (pdf []byte) {
	var output strings.Builder
	output.WriteString("%PDF-1.4\n")
	for object_index, object := range objects {
		output.WriteString(pdf_test_integer(object_index + 1))
		output.WriteString(" 0 obj\n")
		output.WriteString(object)
		output.WriteString("\nendobj\n")
	}
	return []byte(output.String())
}

func pdf_test_integer(value int) (text string) {
	const DIGITS = "0123456789"
	if value == 0 {
		return "0"
	}
	var reversed [20]byte
	count := len(reversed)
	for value > 0 {
		count--
		reversed[count] = DIGITS[value%10]
		value /= 10
	}
	return string(reversed[count:])
}

func pdf_test_lzw_codes(codes []int) (encoded []byte) {
	bit_count := len(codes) * 9
	encoded = make([]byte, (bit_count+7)/8)
	bit_offset := 0
	for _, code := range codes {
		for shift := 8; shift >= 0; shift-- {
			if code&(1<<shift) != 0 {
				encoded[bit_offset/8] |= 1 << uint(7-bit_offset%8)
			}
			bit_offset++
		}
	}
	return encoded
}

func pdf_test_predictor_parameters(predictor int64) (parameters Pdf_Value) {
	return Pdf_Value{
		Kind: PDF_VALUE_DICTIONARY,
		Dictionary: map[string]Pdf_Value{
			"Predictor": {Kind: PDF_VALUE_NUMBER, Integer: predictor},
			"Colors":    {Kind: PDF_VALUE_NUMBER, Integer: 1},
			"Columns":   {Kind: PDF_VALUE_NUMBER, Integer: 3},
			"BitsPerComponent": {
				Kind: PDF_VALUE_NUMBER, Integer: 8,
			},
		},
	}
}

// Test_Pdf_Encryption_Primitives verifies the ciphers and object-key derivation
// independently from PDF syntax and password authentication.
func Test_Pdf_Encryption_Primitives(t *testing.T) {
	t.Parallel()
	rc4_ciphertext, rc4_err := pdf_rc4_crypt(&Pdf_Rc4_Crypt_Input{
		Key: []byte("Key"), Source: []byte("Plaintext"),
	})
	if rc4_err != nil {
		t.Fatalf("RC4: %v", rc4_err)
	}
	if hexadecimal := hex.EncodeToString(rc4_ciphertext); hexadecimal !=
		"bbf316e8d940af0ad3" {
		t.Fatalf("RC4 ciphertext = %s", hexadecimal)
	}

	file_key := []byte{
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
	}
	rc4_key := pdf_object_key(&Pdf_Object_Key_Input{
		File_Key: file_key, Object_Number: 15, Generation: 2,
	})
	if hexadecimal := hex.EncodeToString(rc4_key); hexadecimal !=
		"44bda4ef900e49c82ce9d3c3cf86e8cc" {
		t.Fatalf("RC4 object key = %s", hexadecimal)
	}
	aes_key := pdf_object_key(&Pdf_Object_Key_Input{
		File_Key: file_key, Object_Number: 15, Generation: 2, Use_AES: true,
	})
	if hexadecimal := hex.EncodeToString(aes_key); hexadecimal !=
		"40e406e40b9150dca53d96487bc6e3b6" {
		t.Fatalf("AESV2 object key = %s", hexadecimal)
	}
	pdf_test_aes_primitives(t, aes_key)
}

func pdf_test_aes_primitives(t *testing.T, aes_key []byte) {
	t.Helper()
	aes_ciphertext, decode_err := hex.DecodeString(
		"00000000000000000000000000000000042dbe01027a650c746a5dc65db6be11",
	)
	if decode_err != nil {
		t.Fatalf("decode AES test value: %v", decode_err)
	}
	plaintext, decrypt_err := pdf_aes_cbc_decrypt(&Pdf_Aes_Cbc_Decrypt_Input{
		Key: aes_key[:0:0], Encoded: aes_ciphertext, Remove_Padding: true,
	})
	if decrypt_err == nil {
		t.Fatal("AES accepted an empty key")
	}
	plaintext, decrypt_err = pdf_aes_cbc_decrypt(&Pdf_Aes_Cbc_Decrypt_Input{
		Key: make([]byte, 16), Encoded: aes_ciphertext, Remove_Padding: true,
	})
	if decrypt_err != nil {
		t.Fatalf("AESV2: %v", decrypt_err)
	}
	if !bytes.Equal(plaintext, []byte("Hello")) {
		t.Fatalf("AESV2 plaintext = %x", plaintext)
	}
	if _, decrypt_err = pdf_aes_cbc_decrypt(&Pdf_Aes_Cbc_Decrypt_Input{
		Key: make([]byte, 16), Encoded: make([]byte, aes.BlockSize),
		Remove_Padding: true,
	}); decrypt_err == nil {
		t.Fatal("AES accepted a missing ciphertext block")
	}
	bad_padding := []byte{
		'H', 'e', 'l', 'l', 'o', 0x0a, 0x0b, 0x0b,
		0x0b, 0x0b, 0x0b, 0x0b, 0x0b, 0x0b, 0x0b, 0x0b,
	}
	bad_ciphertext := pdf_test_aes_encrypt_no_padding(
		t, &pdf_test_aes_encrypt_no_padding_input{
			Key: make([]byte, 16), Initialization_Vector: make([]byte, aes.BlockSize),
			Plaintext: bad_padding,
		},
	)
	bad_encoded := append(make([]byte, aes.BlockSize), bad_ciphertext...)
	if _, decrypt_err = pdf_aes_cbc_decrypt(&Pdf_Aes_Cbc_Decrypt_Input{
		Key: make([]byte, 16), Encoded: bad_encoded, Remove_Padding: true,
	}); decrypt_err == nil {
		t.Fatal("AES accepted invalid PKCS#7 padding")
	}
}

type pdf_test_legacy_password_case struct {
	Revision int
	Key_Size int
	Owner    string
	User     string
	File_Key string
}

// Test_Pdf_Legacy_Password_Authentication verifies revisions 2 through 4 with
// fixed owner values, user values, file keys, and both password paths.
func Test_Pdf_Legacy_Password_Authentication(t *testing.T) {
	t.Parallel()
	cases := []pdf_test_legacy_password_case{
		{
			Revision: 2, Key_Size: 5,
			Owner: "94e8094419662a774442fb072e3d9f19" +
				"e9d130ec09a4d0061e78fe920f7ab62f",
			User: "b1f625689bca1356d125453c43cfa750" +
				"c0296b5c271c03f1cc359fa658a4fa21",
			File_Key: "77867d7fae",
		},
		{
			Revision: 3, Key_Size: 16,
			Owner: "0ba3835f88f90388e74e54584125ce14" +
				"2be0de24c6b0d37746e075b891756671",
			User: "d798239dd347dc3afe3dd10515bb5843" +
				"00000000000000000000000000000000",
			File_Key: "2e76247e4b9d5c6aa46d224a2e740d27",
		},
		{
			Revision: 4, Key_Size: 16,
			Owner: "0ba3835f88f90388e74e54584125ce14" +
				"2be0de24c6b0d37746e075b891756671",
			User: "d798239dd347dc3afe3dd10515bb5843" +
				"00000000000000000000000000000000",
			File_Key: "2e76247e4b9d5c6aa46d224a2e740d27",
		},
	}
	pdf_test_legacy_password_cases(t, cases)
	pdf_test_legacy_password_encoding(t)
}

func pdf_test_legacy_password_cases(t *testing.T, cases []pdf_test_legacy_password_case) {
	t.Helper()
	for _, test_case := range cases {
		encryption := Pdf_Encryption{
			Revision: test_case.Revision, Key_Size: test_case.Key_Size,
			Permissions: -4,
			Owner:       pdf_test_hexadecimal(t, test_case.Owner),
			User:        pdf_test_hexadecimal(t, test_case.User),
			Identifier: pdf_test_hexadecimal(
				t, "00112233445566778899aabbccddeeff",
			),
			Encrypt_Metadata: true,
		}
		want := pdf_test_hexadecimal(t, test_case.File_Key)
		for _, password := range [][]byte{[]byte("user"), []byte("owner")} {
			got, authenticate_err := pdf_authenticate_password(&encryption, password)
			if authenticate_err != nil {
				t.Errorf(
					"R%d password %q: %v",
					test_case.Revision,
					password,
					authenticate_err,
				)
				continue
			}
			if !bytes.Equal(got, want) {
				t.Errorf(
					"R%d password %q key = %x",
					test_case.Revision,
					password,
					got,
				)
			}
		}
		if _, authenticate_err := pdf_authenticate_password(
			&encryption, []byte("wrong"),
		); !PDF_Incorrect_Password(authenticate_err) {
			t.Errorf(
				"R%d wrong password error = %v",
				test_case.Revision,
				authenticate_err,
			)
		}
	}
}

func pdf_test_legacy_password_encoding(t *testing.T) {
	t.Helper()
	legacy, legacy_err := pdf_legacy_password([]byte("€"))
	if legacy_err != nil {
		t.Fatalf("PDFDocEncoding password: %v", legacy_err)
	}
	if legacy[0] != 0xa0 {
		t.Fatalf("PDFDocEncoding euro = %x", legacy[0])
	}
}

type pdf_test_aes_256_password_case struct {
	Revision  int
	User      string
	User_Key  string
	Owner     string
	Owner_Key string
}

// Test_Pdf_Aes_256_Password_Authentication verifies R5 and R6 hashes, user and
// owner key recovery, UTF-8 password handling, and permission validation.
func Test_Pdf_Aes_256_Password_Authentication(t *testing.T) {
	t.Parallel()
	pdf_test_r6_known_hash(t)

	cases := []pdf_test_aes_256_password_case{
		{
			Revision: 5,
			User: "f90940351d2eddc7a5d9bf15695a0200" +
				"54f9854261486d8a51767ad0089e4c26000102030405060708090a0b0c0d0e0f",
			User_Key: "d276a2eea10241f7aaf7f51b83f0b922" +
				"f65701a67f9dc9ab48b9d0acf725d5a6",
			Owner: "183d5953a45564acf9763b996aecf087" +
				"d72bb04628bca9c9a6534795472edc34101112131415161718191a1b1c1d1e1f",
			Owner_Key: "aafa2092d04762f0d90126d46a708997" +
				"b2ac781f74bd97d079437365cdba6240",
		},
		{
			Revision: 6,
			User: "731758c09c8b0160a34721d18bdd2422" +
				"0abada0070aa3f05b8103fd5b8d05f17000102030405060708090a0b0c0d0e0f",
			User_Key: "1828690a0eac8bb8404a1ef461e29eff" +
				"0f275bac6dad126cda1624d3cc072770",
			Owner: "430fcaed602ced2ea5a8deaab9e32378" +
				"8ce324b8ae39b7d627f47fdc2c3f800d101112131415161718191a1b1c1d1e1f",
			Owner_Key: "96e5466324d50bf177e695e76e49baab" +
				"44fd04dab98b5b0ae725775d0d81e788",
		},
	}
	pdf_test_aes_256_password_cases(t, cases)
	pdf_test_aes_256_utf_8_password(t)
}

func pdf_test_aes_256_password_cases(t *testing.T, cases []pdf_test_aes_256_password_case) {
	t.Helper()
	want := make([]byte, 32)
	for index := range want {
		want[index] = byte(index)
	}
	for _, test_case := range cases {
		encryption := Pdf_Encryption{
			Revision: test_case.Revision, Key_Size: 32, Permissions: -4,
			Owner:               pdf_test_hexadecimal(t, test_case.Owner),
			User:                pdf_test_hexadecimal(t, test_case.User),
			Owner_Encrypted_Key: pdf_test_hexadecimal(t, test_case.Owner_Key),
			User_Encrypted_Key:  pdf_test_hexadecimal(t, test_case.User_Key),
			Permissions_Encrypted: pdf_test_hexadecimal(
				t, "d4d0080c1ed86563a9fe9f05f865f017",
			),
			Encrypt_Metadata: true,
		}
		for _, password := range [][]byte{[]byte("user"), []byte("owner")} {
			got, authenticate_err := pdf_authenticate_password(&encryption, password)
			if authenticate_err != nil {
				t.Errorf(
					"R%d password %q: %v",
					test_case.Revision,
					password,
					authenticate_err,
				)
				continue
			}
			if !bytes.Equal(got, want) {
				t.Errorf(
					"R%d password %q key = %x",
					test_case.Revision,
					password,
					got,
				)
			}
		}
	}
}

func pdf_test_aes_256_utf_8_password(t *testing.T) {
	t.Helper()
	prepared, prepare_err := pdf_aes_256_password([]byte("密碼"))
	if prepare_err != nil {
		t.Fatalf("UTF-8 password: %v", prepare_err)
	} else if !bytes.Equal(prepared, []byte("密碼")) {
		t.Fatalf("UTF-8 password = %x", prepared)
	}
}

func pdf_test_r6_known_hash(t *testing.T) {
	t.Helper()
	hash := pdf_r6_hash(&Pdf_R6_Hash_Input{
		Password: []byte("password"), Salt: []byte{1, 2, 3, 4, 5, 6, 7, 8},
	})
	if hexadecimal := hex.EncodeToString(hash); hexadecimal !=
		"22d08d1860cb92edcadda1451a4aebb49c1873722bbfca2aef1a7e5f51e69935" {
		t.Fatalf("R6 hash = %s", hexadecimal)
	}
}

// Test_PDF_To_Markdown_Standard_R4 verifies trailer discovery, an explicit
// AESV2 Crypt filter, and both accepted password paths in a complete PDF.
func Test_PDF_To_Markdown_Standard_R4(t *testing.T) {
	t.Parallel()
	pdf := pdf_test_r4_document(t)
	for _, password := range [][]byte{[]byte("user"), []byte("owner")} {
		markdown, convert_err := PDF_To_Markdown(&PDF_To_Markdown_Input{
			PDF: pdf, Password: password,
		})
		if convert_err != nil {
			t.Errorf("password %q: %v", password, convert_err)
			continue
		}
		if !bytes.Contains(markdown, []byte("Secret")) {
			t.Errorf("password %q output = %q", password, markdown)
		}
	}
	if _, convert_err := PDF_To_Markdown(&PDF_To_Markdown_Input{
		PDF: pdf, Password: []byte("wrong"),
	}); !PDF_Incorrect_Password(convert_err) {
		t.Fatalf("wrong password error = %v", convert_err)
	}
}

// Test_PDF_To_Markdown_Encrypt_Text verifies encryption detection reads the
// active trailer instead of searching unrelated object bytes.
func Test_PDF_To_Markdown_Encrypt_Text(t *testing.T) {
	t.Parallel()
	pdf := pdf_test_document(
		"<< /Type /Catalog /Pages 2 0 R /Marker (/Encrypt) >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] >>",
	)
	markdown, convert_err := PDF_To_Markdown(&PDF_To_Markdown_Input{PDF: pdf})
	if convert_err != nil {
		t.Fatalf("unencrypted PDF: %v", convert_err)
	}
	if len(markdown) != 0 {
		t.Fatalf("unencrypted PDF output = %q", markdown)
	}
}

// Test_PDF_To_Markdown_Standard_Password_Decryption verifies complete RC4 and
// AES-256 documents for each remaining Standard handler revision.
func Test_PDF_To_Markdown_Standard_Password_Decryption(t *testing.T) {
	t.Parallel()
	cases := []struct {
		Revision int
		User     []byte
		Owner    []byte
	}{
		{Revision: 2, User: nil, Owner: []byte("owner")},
		{Revision: 3, User: []byte("user"), Owner: []byte("owner")},
		{Revision: 4, User: []byte("user"), Owner: []byte("owner")},
		{Revision: 5, User: []byte("密碼"), Owner: []byte("owner")},
		{Revision: 6, User: []byte("user"), Owner: []byte("owner")},
	}
	for _, test_case := range cases {
		pdf := pdf_test_standard_document(t, &pdf_test_standard_document_input{
			Revision: test_case.Revision, User_Password: test_case.User,
			Owner_Password: test_case.Owner,
		})
		passwords := [][]byte{test_case.User, test_case.Owner}
		for _, password := range passwords {
			markdown, convert_err := PDF_To_Markdown(&PDF_To_Markdown_Input{
				PDF: pdf, Password: password,
			})
			if convert_err != nil {
				t.Errorf(
					"R%d password %q: %v",
					test_case.Revision,
					password,
					convert_err,
				)
				continue
			}
			if !bytes.Contains(markdown, []byte("Secret")) {
				t.Errorf(
					"R%d password %q output = %q",
					test_case.Revision,
					password,
					markdown,
				)
			}
		}
	}
	pdf_2 := pdf_test_standard_document(t, &pdf_test_standard_document_input{
		Revision: 6, User_Password: []byte("user"), Owner_Password: []byte("owner"),
	})
	copy(pdf_2[:8], []byte("%PDF-2.0"))
	if _, convert_err := PDF_To_Markdown(&PDF_To_Markdown_Input{
		PDF: pdf_2, Password: []byte("user"),
	}); convert_err != nil {
		t.Fatalf("PDF 2.0 R6: %v", convert_err)
	}
}

// Test_PDF_To_Markdown_Xref_Encryption_Trailers verifies encryption discovery
// through xref streams, hybrid XRefStm links, and incremental Prev links.
func Test_PDF_To_Markdown_Xref_Encryption_Trailers(t *testing.T) {
	t.Parallel()
	for _, layout := range []string{"stream", "hybrid", "incremental"} {
		pdf := pdf_test_r4_xref_layout(t, layout)
		markdown, convert_err := PDF_To_Markdown(&PDF_To_Markdown_Input{
			PDF: pdf, Password: []byte("user"),
		})
		if convert_err != nil {
			t.Errorf("%s: %v", layout, convert_err)
			continue
		}
		if !bytes.Contains(markdown, []byte("Secret")) {
			t.Errorf("%s output = %q", layout, markdown)
		}
	}
}

// Test_PDF_To_Markdown_Encrypted_Object_Stream verifies one encrypted object
// stream is decrypted before expansion and its contained strings stay clear.
func Test_PDF_To_Markdown_Encrypted_Object_Stream(t *testing.T) {
	t.Parallel()
	pdf := pdf_test_r4_object_stream_document(t)
	markdown, convert_err := PDF_To_Markdown(&PDF_To_Markdown_Input{
		PDF: pdf, Password: []byte("user"),
	})
	if convert_err != nil {
		t.Fatalf("object stream: %v", convert_err)
	}
	if !bytes.Contains(markdown, []byte("Secret")) {
		t.Fatalf("object stream output = %q", markdown)
	}
}

// Test_PDF_To_Markdown_Encryption_Exemptions verifies nested encrypted streams,
// direct strings, Identity, clear metadata, and signature Contents together.
func Test_PDF_To_Markdown_Encryption_Exemptions(t *testing.T) {
	t.Parallel()
	pdf := pdf_test_r4_feature_document(t)
	markdown, convert_err := PDF_To_Markdown(&PDF_To_Markdown_Input{
		PDF: pdf, Password: []byte("user"),
	})
	if convert_err != nil {
		t.Fatalf("feature PDF: %v", convert_err)
	}
	if !bytes.Contains(markdown, []byte("Ω")) {
		t.Fatalf("feature output = %q", markdown)
	}
	document, parse_err := pdf_parse_document_with_password(
		&Pdf_Parse_Document_With_Password_Input{Source: pdf, Password: []byte("user")},
	)
	if parse_err != nil {
		t.Fatalf("parse feature PDF: %v", parse_err)
	}
	title := document.Objects[9].Value.Dictionary["Title"].String_Bytes
	if !bytes.Equal(title, []byte("Title")) {
		t.Fatalf("encrypted string = %q", title)
	}
	metadata, metadata_err := pdf_decode_stream(document, document.Objects[10].Value)
	if metadata_err != nil {
		t.Fatalf("metadata: %v", metadata_err)
	}
	if !bytes.Equal(metadata, []byte("<metadata/>")) {
		t.Fatalf("metadata = %q", metadata)
	}
	signature := document.Objects[11].Value.Dictionary
	if !bytes.Equal(signature["Contents"].String_Bytes, []byte("signature")) {
		t.Fatalf("signature Contents = %q", signature["Contents"].String_Bytes)
	}
	if !bytes.Equal(signature["Reason"].String_Bytes, []byte("Reason")) {
		t.Fatalf("signature Reason = %q", signature["Reason"].String_Bytes)
	}
}

// Test_PDF_To_Markdown_Encryption_Validation verifies malformed and unsupported
// encryption state fails as a document error instead of a password error.
func Test_PDF_To_Markdown_Encryption_Validation(t *testing.T) {
	t.Parallel()
	r4 := pdf_test_r4_document(t)
	r3 := pdf_test_standard_document(t, &pdf_test_standard_document_input{
		Revision: 3, User_Password: []byte("user"), Owner_Password: []byte("owner"),
	})
	cases := [][]byte{
		pdf_test_replace_same_size(t, &pdf_test_replace_same_size_input{
			Source: r3, Old: []byte("/V 2 /R 3"), New: []byte("/V 3 /R 3"),
		}),
		pdf_test_replace_same_size(t, &pdf_test_replace_same_size_input{
			Source: r4, Old: []byte("/ID ["), New: []byte("/IX ["),
		}),
		pdf_test_replace_same_size(t, &pdf_test_replace_same_size_input{
			Source: r4, Old: []byte("/Length 128"), New: []byte("/Length 129"),
		}),
		pdf_test_replace_same_size(t, &pdf_test_replace_same_size_input{
			Source: r4, Old: []byte("/Filter /Standard"),
			New: []byte("/Filter /Badxxxxx"),
		}),
		pdf_test_replace_same_size(t, &pdf_test_replace_same_size_input{
			Source: r4, Old: []byte("/CFM /AESV2"), New: []byte("/CFM /AESV4"),
		}),
		pdf_test_replace_same_size(t, &pdf_test_replace_same_size_input{
			Source: r4, Old: []byte("/StmF /StdCF"), New: []byte("/StmF /NoWay"),
		}),
		pdf_test_invalid_owner_value(t, r4),
		pdf_test_invalid_permissions(t),
	}
	for case_index, pdf := range cases {
		_, convert_err := PDF_To_Markdown(&PDF_To_Markdown_Input{
			PDF: pdf, Password: []byte("user"),
		})
		if convert_err == nil {
			t.Errorf("case %d succeeded", case_index)
			continue
		}
		if PDF_Incorrect_Password(convert_err) {
			t.Errorf(
				"case %d returned incorrect-password status: %v",
				case_index,
				convert_err,
			)
		}
	}
}

type pdf_test_replace_same_size_input struct {
	Source []byte
	Old    []byte
	New    []byte
}

func pdf_test_replace_same_size(
	t *testing.T,
	input *pdf_test_replace_same_size_input,
) (result []byte) {
	t.Helper()
	if len(input.Old) != len(input.New) {
		t.Fatal("same-size replacement changed length")
	}
	offset := bytes.Index(input.Source, input.Old)
	if offset < 0 {
		t.Fatalf("replacement source lacks %q", input.Old)
	}
	result = append([]byte{}, input.Source...)
	copy(result[offset:offset+len(input.Old)], input.New)
	return result
}

func pdf_test_invalid_owner_value(t *testing.T, source []byte) (pdf []byte) {
	t.Helper()
	pdf = append([]byte{}, source...)
	prefix := []byte("/O <")
	offset := bytes.Index(pdf, prefix)
	if offset < 0 {
		t.Fatal("R4 dictionary has no O value")
	}
	value_start := offset + len("/O ")
	value_end := value_start + 65
	pdf[value_start] = '/'
	pdf[value_end] = 'A'
	return pdf
}

func pdf_test_invalid_permissions(t *testing.T) (pdf []byte) {
	t.Helper()
	pdf = pdf_test_standard_document(t, &pdf_test_standard_document_input{
		Revision: 5, User_Password: []byte("user"), Owner_Password: []byte("owner"),
	})
	prefix := []byte("/Perms <")
	offset := bytes.Index(pdf, prefix)
	if offset < 0 {
		t.Fatal("R5 dictionary has no Perms value")
	}
	digit := offset + len(prefix)
	if pdf[digit] == '0' {
		pdf[digit] = '1'
	} else {
		pdf[digit] = '0'
	}
	return pdf
}

func pdf_test_r4_document(t *testing.T) (pdf []byte) {
	t.Helper()
	return pdf_test_classic_xref(pdf_test_r4_objects(t), 6)
}

func pdf_test_r4_objects(t *testing.T) (objects [][]byte) {
	t.Helper()
	file_key := pdf_test_hexadecimal(t, "2e76247e4b9d5c6aa46d224a2e740d27")
	plaintext := []byte("BT /F0 12 Tf 1 0 0 1 72 720 Tm (Secret) Tj ET")
	ciphertext := pdf_test_aes_encrypt(t, &pdf_test_aes_encrypt_input{
		Key: pdf_object_key(&Pdf_Object_Key_Input{
			File_Key: file_key, Object_Number: 4, Use_AES: true,
		}),
		Initialization_Vector: make([]byte, aes.BlockSize), Plaintext: plaintext,
	})
	return [][]byte{
		[]byte("<< /Type /Catalog /Pages 2 0 R >>"),
		[]byte("<< /Type /Pages /Kids [3 0 R] /Count 1 >>"),
		[]byte("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] " +
			"/Resources << /Font << /F0 5 0 R >> >> /Contents 4 0 R >>"),
		append([]byte(fmt.Sprintf(
			"<< /Length %d /Filter /Crypt /DecodeParms << /Name /StdCF >> >>\nstream\n",
			len(ciphertext),
		)), append(ciphertext, []byte("\nendstream")...)...),
		[]byte("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>"),
		[]byte("<< /Filter /Standard /V 4 /R 4 /Length 128 /P -4 " +
			"/O <0ba3835f88f90388e74e54584125ce142be0de24c6b0d37746e075b891756671> " +
			"/U <d798239dd347dc3afe3dd10515bb584300000000000000000000000000000000> " +
			"/CF << /StdCF << /CFM /AESV2 /Length 16 >> >> " +
			"/StmF /StdCF /StrF /StdCF >>"),
	}
}

func pdf_test_r4_object_stream_document(t *testing.T) (pdf []byte) {
	t.Helper()
	encryption := pdf_test_r4_encryption(t, true)
	bodies := [][]byte{
		[]byte("<< /Type /Catalog /Pages 3 0 R /Label (clear) >>"),
		[]byte("<< /Type /Pages /Kids [4 0 R] /Count 1 >>"),
		[]byte("<< /Type /Page /Parent 3 0 R /MediaBox [0 0 612 792] " +
			"/Resources << /Font << /F0 5 0 R >> >> /Contents 6 0 R >>"),
		[]byte("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>"),
	}
	object_numbers := []int{2, 3, 4, 5}
	plain, first := pdf_test_object_stream_plaintext(object_numbers, bodies)
	object_stream := pdf_test_encrypt_object(t, &pdf_test_encrypt_object_input{
		Encryption: encryption, Object_Number: 1, Plaintext: plain,
	})
	content := pdf_test_encrypt_object(t, &pdf_test_encrypt_object_input{
		Encryption: encryption, Object_Number: 6,
		Plaintext: []byte("BT /F0 12 Tf 1 0 0 1 72 720 Tm (Secret) Tj ET"),
	})
	objects := []pdf_test_sparse_object{
		{
			Number: 1,
			Value: append([]byte(fmt.Sprintf(
				"<< /Type /ObjStm /N 4 /First %d /Length %d >>\nstream\n",
				first, len(object_stream),
			)), append(object_stream, []byte("\nendstream")...)...),
		},
		{
			Number: 6,
			Value: append([]byte(fmt.Sprintf(
				"<< /Length %d >>\nstream\n", len(content),
			)),
				append(content, []byte("\nendstream")...)...),
		},
		{Number: 7, Value: []byte(pdf_test_r4_encryption_dictionary(encryption))},
	}
	return pdf_test_sparse_classic_xref(&pdf_test_sparse_classic_xref_input{
		Objects: objects, Object_Count: 8, Encrypt_Object: 7,
	})
}

func pdf_test_object_stream_plaintext(
	object_numbers []int,
	bodies [][]byte,
) (plaintext []byte, first int) {
	var body bytes.Buffer
	offsets := make([]int, len(bodies))
	for index, value := range bodies {
		offsets[index] = body.Len()
		body.Write(value)
		body.WriteByte(' ')
	}
	var header bytes.Buffer
	for index, object_number := range object_numbers {
		fmt.Fprintf(&header, "%d %d ", object_number, offsets[index])
	}
	first = header.Len()
	header.Write(body.Bytes())
	return header.Bytes(), first
}

func pdf_test_r4_feature_document(t *testing.T) (pdf []byte) {
	t.Helper()
	encryption := pdf_test_r4_encryption(t, false)
	unicode_map := []byte(
		"1 begincodespacerange <00> <ff> endcodespacerange " +
			"1 beginbfchar <41> <03a9> endbfchar",
	)
	form := pdf_test_encrypt_object(t, &pdf_test_encrypt_object_input{
		Encryption: encryption, Object_Number: 6,
		Plaintext: []byte("BT /F0 12 Tf 1 0 0 1 0 0 Tm (A) Tj ET"),
	})
	content := pdf_test_encrypt_object(t, &pdf_test_encrypt_object_input{
		Encryption: encryption, Object_Number: 7, Plaintext: []byte("q /Fm0 Do Q"),
	})
	title := pdf_test_encrypt_object(t, &pdf_test_encrypt_object_input{
		Encryption: encryption, Object_Number: 9, Plaintext: []byte("Title"),
	})
	reason := pdf_test_encrypt_object(t, &pdf_test_encrypt_object_input{
		Encryption: encryption, Object_Number: 11, Plaintext: []byte("Reason"),
	})
	objects := [][]byte{
		[]byte("<< /Type /Catalog /Pages 2 0 R /Metadata 10 0 R >>"),
		[]byte("<< /Type /Pages /Kids [3 0 R] /Count 1 >>"),
		[]byte("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] " +
			"/Resources << /Font << /F0 4 0 R >> /XObject << /Fm0 6 0 R >> >> " +
			"/Contents 7 0 R >>"),
		[]byte("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /ToUnicode 5 0 R >>"),
		append([]byte(fmt.Sprintf(
			"<< /Length %d /Filter /Crypt /DecodeParms << /Name /Identity >> >>\n"+
				"stream\n", len(unicode_map),
		)), append(unicode_map, []byte("\nendstream")...)...),
		append([]byte(fmt.Sprintf(
			"<< /Type /XObject /Subtype /Form /BBox [0 0 100 100] "+
				"/Resources << /Font << /F0 4 0 R >> >> /Length %d >>\nstream\n",
			len(form),
		)), append(form, []byte("\nendstream")...)...),
		append([]byte(fmt.Sprintf("<< /Length %d >>\nstream\n", len(content))),
			append(content, []byte("\nendstream")...)...),
		[]byte(pdf_test_r4_encryption_dictionary(encryption)),
		[]byte(fmt.Sprintf("<< /Title <%x> >>", title)),
		[]byte("<< /Type /Metadata /Length 11 >>\nstream\n<metadata/>\nendstream"),
		[]byte(fmt.Sprintf(
			"<< /Type /Sig /Contents <7369676e6174757265> /Reason <%x> >>", reason,
		)),
	}
	return pdf_test_classic_xref(objects, 8)
}

func pdf_test_r4_encryption(t *testing.T, encrypt_metadata bool) (encryption *Pdf_Encryption) {
	t.Helper()
	encryption = pdf_test_standard_encryption(t, &pdf_test_standard_encryption_input{
		Revision: 4, User_Password: []byte("user"), Owner_Password: []byte("owner"),
	})
	encryption.Version = 4
	encryption.Encrypt_Metadata = encrypt_metadata
	prepared_user, prepare_err := pdf_legacy_password([]byte("user"))
	if prepare_err != nil {
		t.Fatalf("R4 user password: %v", prepare_err)
	}
	encryption.File_Key = pdf_legacy_file_key(encryption, prepared_user)
	encryption.User = pdf_test_legacy_user_value(t, encryption)
	encryption.Crypt_Filters = map[string]Pdf_Crypt_Filter{
		"StdCF": {Method: PDF_CRYPT_AES_128},
	}
	encryption.Stream_Filter = "StdCF"
	encryption.String_Filter = "StdCF"
	return encryption
}

func pdf_test_r4_encryption_dictionary(encryption *Pdf_Encryption) (dictionary string) {
	metadata := "true"
	if !encryption.Encrypt_Metadata {
		metadata = "false"
	}
	return fmt.Sprintf(
		"<< /Filter /Standard /V 4 /R 4 /Length 128 /P -4 "+
			"/O <%x> /U <%x> /EncryptMetadata %s "+
			"/CF << /StdCF << /CFM /AESV2 /Length 16 >> >> "+
			"/StmF /StdCF /StrF /StdCF >>",
		encryption.Owner, encryption.User, metadata,
	)
}

type pdf_test_sparse_object struct {
	Number int
	Value  []byte
}

type pdf_test_sparse_classic_xref_input struct {
	Objects        []pdf_test_sparse_object
	Object_Count   int
	Encrypt_Object int
}

func pdf_test_sparse_classic_xref(
	input *pdf_test_sparse_classic_xref_input,
) (pdf []byte) {
	var output bytes.Buffer
	output.WriteString("%PDF-1.7\n")
	offsets := make([]int, input.Object_Count)
	for _, object := range input.Objects {
		offsets[object.Number] = output.Len()
		fmt.Fprintf(&output, "%d 0 obj\n", object.Number)
		output.Write(object.Value)
		output.WriteString("\nendobj\n")
	}
	xref_size := output.Len()
	fmt.Fprintf(&output, "xref\n0 %d\n", input.Object_Count)
	output.WriteString("0000000000 65535 f \n")
	object_number_index := 1
	for object_number_index < input.Object_Count {
		if offsets[object_number_index] == 0 {
			output.WriteString("0000000000 00000 f \n")
			object_number_index++
			continue
		}
		fmt.Fprintf(&output, "%010d 00000 n \n", offsets[object_number_index])
		object_number_index++
	}
	fmt.Fprintf(
		&output,
		"trailer\n<< /Size %d /Root 2 0 R /Encrypt %d 0 R "+
			"/ID [<00112233445566778899aabbccddeeff> "+
			"<00112233445566778899aabbccddeeff>] >>\n"+
			"startxref\n%d\n%%%%EOF\n",
		input.Object_Count, input.Encrypt_Object, xref_size,
	)
	return output.Bytes()
}

func pdf_test_r4_xref_layout(t *testing.T, layout string) (pdf []byte) {
	t.Helper()
	objects := pdf_test_r4_objects(t)
	if layout == "incremental" {
		return pdf_test_incremental_encryption_trailer(t, objects)
	}
	var output bytes.Buffer
	offsets := pdf_test_write_objects(&output, objects)
	xref_stream_size := output.Len()
	xref_stream := []byte(
		"<< /Type /XRef /Size 8 /Root 1 0 R /Encrypt 6 0 R " +
			"/ID [<00112233445566778899aabbccddeeff> " +
			"<00112233445566778899aabbccddeeff>] /W [1 4 2] /Length 0 >>" +
			"\nstream\n\nendstream",
	)
	fmt.Fprintf(&output, "7 0 obj\n%s\nendobj\n", xref_stream)
	if layout == "stream" {
		fmt.Fprintf(&output, "startxref\n%d\n%%%%EOF\n", xref_stream_size)
		return output.Bytes()
	}
	if layout != "hybrid" {
		t.Fatalf("unknown xref layout %q", layout)
	}
	offsets = append(offsets, xref_stream_size)
	xref_size := output.Len()
	pdf_test_write_xref(&output, offsets)
	fmt.Fprintf(
		&output,
		"trailer\n<< /Size 8 /Root 1 0 R /XRefStm %d >>\n"+
			"startxref\n%d\n%%%%EOF\n",
		xref_stream_size, xref_size,
	)
	return output.Bytes()
}

func pdf_test_incremental_encryption_trailer(
	t *testing.T,
	objects [][]byte,
) (pdf []byte) {
	t.Helper()
	base := pdf_test_classic_xref(objects, 6)
	previous_offset := bytes.Index(base, []byte("\nxref\n"))
	if previous_offset < 0 {
		t.Fatal("base PDF has no xref")
	}
	previous_offset++
	var output bytes.Buffer
	output.Write(base)
	current_size := output.Len()
	output.WriteString("xref\n0 1\n0000000000 65535 f \n")
	fmt.Fprintf(
		&output,
		"trailer\n<< /Size 7 /Root 1 0 R /Prev %d >>\n"+
			"startxref\n%d\n%%%%EOF\n",
		previous_offset, current_size,
	)
	return output.Bytes()
}

func pdf_test_write_objects(output *bytes.Buffer, objects [][]byte) (offsets []int) {
	output.WriteString("%PDF-1.7\n")
	offsets = make([]int, len(objects)+1)
	for object_index, object := range objects {
		offsets[object_index+1] = output.Len()
		fmt.Fprintf(output, "%d 0 obj\n", object_index+1)
		output.Write(object)
		output.WriteString("\nendobj\n")
	}
	return offsets
}

func pdf_test_write_xref(output *bytes.Buffer, offsets []int) {
	fmt.Fprintf(output, "xref\n0 %d\n", len(offsets))
	output.WriteString("0000000000 65535 f \n")
	for object_number_index := 1; object_number_index < len(offsets); object_number_index++ {
		fmt.Fprintf(output, "%010d 00000 n \n", offsets[object_number_index])
	}
}

type pdf_test_standard_document_input struct {
	Revision       int
	User_Password  []byte
	Owner_Password []byte
}

func pdf_test_standard_document(
	t *testing.T,
	input *pdf_test_standard_document_input,
) (pdf []byte) {
	t.Helper()
	encryption := pdf_test_standard_encryption(t, &pdf_test_standard_encryption_input{
		Revision: input.Revision, User_Password: input.User_Password,
		Owner_Password: input.Owner_Password,
	})
	plaintext := []byte("BT /F0 12 Tf 1 0 0 1 72 720 Tm (Secret) Tj ET")
	encoded := pdf_test_encrypt_object(t, &pdf_test_encrypt_object_input{
		Encryption: encryption, Object_Number: 4, Plaintext: plaintext,
	})
	stream_dictionary := fmt.Sprintf("<< /Length %d >>", len(encoded))
	objects := [][]byte{
		[]byte("<< /Type /Catalog /Pages 2 0 R >>"),
		[]byte("<< /Type /Pages /Kids [3 0 R] /Count 1 >>"),
		[]byte("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] " +
			"/Resources << /Font << /F0 5 0 R >> >> /Contents 4 0 R >>"),
		append([]byte(stream_dictionary+"\nstream\n"),
			append(encoded, []byte("\nendstream")...)...),
		[]byte("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>"),
		[]byte(pdf_test_encryption_dictionary(encryption)),
	}
	return pdf_test_classic_xref(objects, 6)
}

type pdf_test_standard_encryption_input struct {
	Revision       int
	User_Password  []byte
	Owner_Password []byte
}

func pdf_test_standard_encryption(
	t *testing.T,
	input *pdf_test_standard_encryption_input,
) (encryption *Pdf_Encryption) {
	t.Helper()
	encryption = &Pdf_Encryption{
		Revision: input.Revision, Permissions: -4, Encrypt_Metadata: true,
		Identifier:    pdf_test_hexadecimal(t, "00112233445566778899aabbccddeeff"),
		Crypt_Filters: make(map[string]Pdf_Crypt_Filter),
	}
	if input.Revision <= 4 {
		pdf_test_legacy_encryption(t, &pdf_test_legacy_encryption_input{
			Encryption: encryption, User_Password: input.User_Password,
			Owner_Password: input.Owner_Password,
		})
		if input.Revision == 4 {
			encryption.Version = 4
			encryption.Crypt_Filters = map[string]Pdf_Crypt_Filter{
				"StdCF": {Method: PDF_CRYPT_RC4},
			}
			encryption.Stream_Filter = "StdCF"
			encryption.String_Filter = "StdCF"
		}
		return encryption
	}
	pdf_test_aes_256_encryption(t, &pdf_test_aes_256_encryption_input{
		Encryption: encryption, User_Password: input.User_Password,
		Owner_Password: input.Owner_Password,
	})
	return encryption
}

type pdf_test_legacy_encryption_input struct {
	Encryption     *Pdf_Encryption
	User_Password  []byte
	Owner_Password []byte
}

func pdf_test_legacy_encryption(
	t *testing.T,
	input *pdf_test_legacy_encryption_input,
) {
	t.Helper()
	encryption := input.Encryption
	encryption.Version = 2
	encryption.Key_Size = 16
	if encryption.Revision == 2 {
		encryption.Version = 1
		encryption.Key_Size = 5
	}
	prepared_user, user_err := pdf_legacy_password(input.User_Password)
	if user_err != nil {
		t.Fatalf("legacy user password: %v", user_err)
	}
	effective_owner := input.Owner_Password
	if len(effective_owner) == 0 {
		effective_owner = input.User_Password
	}
	prepared_owner, owner_err := pdf_legacy_password(effective_owner)
	if owner_err != nil {
		t.Fatalf("legacy owner password: %v", owner_err)
	}
	owner_key := pdf_legacy_owner_key(encryption, prepared_owner)
	encryption.Owner = pdf_test_rc4_iterations(t, &pdf_test_rc4_iterations_input{
		Source: prepared_user, Key: owner_key, Repeated: encryption.Revision >= 3,
	})
	encryption.File_Key = pdf_legacy_file_key(encryption, prepared_user)
	encryption.User = pdf_test_legacy_user_value(t, encryption)
	pdf_set_legacy_crypt_filters(encryption)
}

type pdf_test_rc4_iterations_input struct {
	Source   []byte
	Key      []byte
	Repeated bool
}

func pdf_test_rc4_iterations(
	t *testing.T,
	input *pdf_test_rc4_iterations_input,
) (result []byte) {
	t.Helper()
	result, _ = pdf_rc4_crypt(&Pdf_Rc4_Crypt_Input{
		Key: input.Key, Source: input.Source,
	})
	if !input.Repeated {
		return result
	}
	for iteration_index := 1; iteration_index < 20; iteration_index++ {
		result, _ = pdf_rc4_crypt(&Pdf_Rc4_Crypt_Input{
			Key: pdf_xor_key(input.Key, byte(iteration_index)), Source: result,
		})
	}
	return result
}

func pdf_test_legacy_user_value(
	t *testing.T,
	encryption *Pdf_Encryption,
) (user []byte) {
	t.Helper()
	if encryption.Revision == 2 {
		user, _ = pdf_rc4_crypt(&Pdf_Rc4_Crypt_Input{
			Key: encryption.File_Key, Source: []byte(PDF_PASSWORD_PADDING),
		})
		return user
	}
	material := append([]byte(PDF_PASSWORD_PADDING), encryption.Identifier...)
	digest := md5.Sum(material)
	user = pdf_test_rc4_iterations(t, &pdf_test_rc4_iterations_input{
		Source: digest[:], Key: encryption.File_Key, Repeated: true,
	})
	return append(user, make([]byte, 16)...)
}

type pdf_test_aes_256_encryption_input struct {
	Encryption     *Pdf_Encryption
	User_Password  []byte
	Owner_Password []byte
}

func pdf_test_aes_256_encryption(
	t *testing.T,
	input *pdf_test_aes_256_encryption_input,
) {
	t.Helper()
	encryption := input.Encryption
	encryption.Version = 5
	encryption.Key_Size = 32
	encryption.File_Key = make([]byte, 32)
	for index := range encryption.File_Key {
		encryption.File_Key[index] = byte(index)
	}
	user := pdf_test_aes_256_password(t, input.User_Password)
	owner := pdf_test_aes_256_password(t, input.Owner_Password)
	user_validation := []byte{0, 1, 2, 3, 4, 5, 6, 7}
	user_key := []byte{8, 9, 10, 11, 12, 13, 14, 15}
	encryption.User = append(append(pdf_aes_256_hash(&Pdf_Aes_256_Hash_Input{
		Revision: encryption.Revision, Password: user, Salt: user_validation,
	}), user_validation...), user_key...)
	encryption.User_Encrypted_Key = pdf_test_aes_encrypt_no_padding(
		t, &pdf_test_aes_encrypt_no_padding_input{
			Key: pdf_aes_256_hash(&Pdf_Aes_256_Hash_Input{
				Revision: encryption.Revision, Password: user, Salt: user_key,
			}),
			Initialization_Vector: make([]byte, aes.BlockSize),
			Plaintext:             encryption.File_Key,
		},
	)
	pdf_test_aes_256_owner_values(t, encryption, owner)
	pdf_test_aes_256_permissions(t, encryption)
	encryption.Crypt_Filters["StdCF"] = Pdf_Crypt_Filter{Method: PDF_CRYPT_AES_256}
	encryption.Stream_Filter = "StdCF"
	encryption.String_Filter = "StdCF"
}

func pdf_test_aes_256_owner_values(
	t *testing.T,
	encryption *Pdf_Encryption,
	owner []byte,
) {
	t.Helper()
	owner_validation := []byte{16, 17, 18, 19, 20, 21, 22, 23}
	owner_key := []byte{24, 25, 26, 27, 28, 29, 30, 31}
	encryption.Owner = append(append(pdf_aes_256_hash(&Pdf_Aes_256_Hash_Input{
		Revision: encryption.Revision, Password: owner,
		Salt: owner_validation, User: encryption.User,
	}), owner_validation...), owner_key...)
	encryption.Owner_Encrypted_Key = pdf_test_aes_encrypt_no_padding(
		t, &pdf_test_aes_encrypt_no_padding_input{
			Key: pdf_aes_256_hash(&Pdf_Aes_256_Hash_Input{
				Revision: encryption.Revision, Password: owner,
				Salt: owner_key, User: encryption.User,
			}),
			Initialization_Vector: make([]byte, aes.BlockSize),
			Plaintext:             encryption.File_Key,
		},
	)
}

func pdf_test_aes_256_permissions(t *testing.T, encryption *Pdf_Encryption) {
	t.Helper()
	clear := []byte{
		0xfc, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		'T', 'a', 'd', 'b', 0xa0, 0xa1, 0xa2, 0xa3,
	}
	block, block_err := aes.NewCipher(encryption.File_Key)
	if block_err != nil {
		t.Fatalf("Perms AES key: %v", block_err)
	}
	encryption.Permissions_Encrypted = make([]byte, aes.BlockSize)
	block.Encrypt(encryption.Permissions_Encrypted, clear)
}

func pdf_test_aes_256_password(t *testing.T, password []byte) (prepared []byte) {
	t.Helper()
	prepared, prepare_err := pdf_aes_256_password(password)
	if prepare_err != nil {
		t.Fatalf("AES-256 password: %v", prepare_err)
	}
	return prepared
}

type pdf_test_encrypt_object_input struct {
	Encryption    *Pdf_Encryption
	Object_Number int
	Generation    int
	Plaintext     []byte
}

func pdf_test_encrypt_object(
	t *testing.T,
	input *pdf_test_encrypt_object_input,
) (encoded []byte) {
	t.Helper()
	encryption := input.Encryption
	filter, exists := encryption.Crypt_Filters[encryption.Stream_Filter]
	if !exists {
		t.Fatal("test encryption has no stream crypt filter")
	}
	if filter.Method == PDF_CRYPT_RC4 {
		key := pdf_object_key(&Pdf_Object_Key_Input{
			File_Key: encryption.File_Key, Object_Number: input.Object_Number,
			Generation: input.Generation,
		})
		encoded, _ = pdf_rc4_crypt(&Pdf_Rc4_Crypt_Input{Key: key, Source: input.Plaintext})
		return encoded
	}
	key := encryption.File_Key
	if filter.Method == PDF_CRYPT_AES_128 {
		key = pdf_object_key(&Pdf_Object_Key_Input{
			File_Key: key, Object_Number: input.Object_Number,
			Generation: input.Generation, Use_AES: true,
		})
	}
	return pdf_test_aes_encrypt(t, &pdf_test_aes_encrypt_input{
		Key: key, Initialization_Vector: make([]byte, aes.BlockSize),
		Plaintext: input.Plaintext,
	})
}

type pdf_test_aes_encrypt_no_padding_input struct {
	Key                   []byte
	Initialization_Vector []byte
	Plaintext             []byte
}

func pdf_test_aes_encrypt_no_padding(
	t *testing.T,
	input *pdf_test_aes_encrypt_no_padding_input,
) (ciphertext []byte) {
	t.Helper()
	block, block_err := aes.NewCipher(input.Key)
	if block_err != nil {
		t.Fatalf("AES key: %v", block_err)
	}
	if len(input.Plaintext)%aes.BlockSize != 0 {
		t.Fatal("AES test plaintext is not block aligned")
	}
	ciphertext = make([]byte, len(input.Plaintext))
	cipher.NewCBCEncrypter(block, input.Initialization_Vector).CryptBlocks(
		ciphertext, input.Plaintext,
	)
	return ciphertext
}

func pdf_test_encryption_dictionary(encryption *Pdf_Encryption) (dictionary string) {
	if encryption.Revision == 2 {
		return fmt.Sprintf(
			"<< /Filter /Standard /V 1 /R 2 /P -4 /O <%x> /U <%x> >>",
			encryption.Owner, encryption.User,
		)
	}
	if encryption.Revision == 3 {
		return fmt.Sprintf(
			"<< /Filter /Standard /V 2 /R 3 /Length 128 /P -4 "+
				"/O <%x> /U <%x> >>", encryption.Owner, encryption.User,
		)
	}
	if encryption.Revision == 4 {
		return fmt.Sprintf(
			"<< /Filter /Standard /V 4 /R 4 /Length 128 /P -4 "+
				"/O <%x> /U <%x> "+
				"/CF << /StdCF << /CFM /V2 /Length 16 >> >> "+
				"/StmF /StdCF /StrF /StdCF >>",
			encryption.Owner, encryption.User,
		)
	}
	return fmt.Sprintf(
		"<< /Filter /Standard /V 5 /R %d /Length 256 /P -4 "+
			"/O <%x> /U <%x> /OE <%x> /UE <%x> /Perms <%x> "+
			"/CF << /StdCF << /CFM /AESV3 /Length 32 >> >> "+
			"/StmF /StdCF /StrF /StdCF >>",
		encryption.Revision, encryption.Owner, encryption.User,
		encryption.Owner_Encrypted_Key, encryption.User_Encrypted_Key,
		encryption.Permissions_Encrypted,
	)
}

type pdf_test_aes_encrypt_input struct {
	Key                   []byte
	Initialization_Vector []byte
	Plaintext             []byte
}

func pdf_test_aes_encrypt(
	t *testing.T,
	input *pdf_test_aes_encrypt_input,
) (encoded []byte) {
	t.Helper()
	block, block_err := aes.NewCipher(input.Key)
	if block_err != nil {
		t.Fatalf("AES key: %v", block_err)
	}
	padding_size := aes.BlockSize - len(input.Plaintext)%aes.BlockSize
	padded := append([]byte{}, input.Plaintext...)
	for index := 0; index < padding_size; index++ {
		padded = append(padded, byte(padding_size))
	}
	ciphertext := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, input.Initialization_Vector).CryptBlocks(ciphertext, padded)
	return append(append([]byte{}, input.Initialization_Vector...), ciphertext...)
}

func pdf_test_classic_xref(objects [][]byte, encrypt_object int) (pdf []byte) {
	var output bytes.Buffer
	output.WriteString("%PDF-1.7\n")
	offsets := make([]int, len(objects)+1)
	for object_index, object := range objects {
		offsets[object_index+1] = output.Len()
		fmt.Fprintf(&output, "%d 0 obj\n", object_index+1)
		output.Write(object)
		output.WriteString("\nendobj\n")
	}
	xref_size := output.Len()
	fmt.Fprintf(&output, "xref\n0 %d\n", len(objects)+1)
	output.WriteString("0000000000 65535 f \n")
	for object_number_index := 1; object_number_index <= len(objects); object_number_index++ {
		fmt.Fprintf(&output, "%010d 00000 n \n", offsets[object_number_index])
	}
	fmt.Fprintf(
		&output,
		"trailer\n<< /Size %d /Root 1 0 R /Encrypt %d 0 R "+
			"/ID [<00112233445566778899aabbccddeeff> "+
			"<00112233445566778899aabbccddeeff>] >>\nstartxref\n%d\n%%%%EOF\n",
		len(objects)+1, encrypt_object, xref_size,
	)
	return output.Bytes()
}

func pdf_test_hexadecimal(t *testing.T, value string) (decoded []byte) {
	t.Helper()
	decoded, decode_err := hex.DecodeString(value)
	if decode_err != nil {
		t.Fatalf("decode %q: %v", value, decode_err)
	}
	return decoded
}
