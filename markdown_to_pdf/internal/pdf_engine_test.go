package markdown_to_pdf

import (
	"bytes"
	standard_zlib "compress/zlib"
	"encoding/ascii85"
	"strings"
	"testing"
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
	markdown, convert_err := PDF_To_Markdown(pdf)
	if convert_err != nil {
		t.Fatalf("Form XObject: %v", convert_err)
	}
	if !strings.Contains(string(markdown), "Nested") {
		t.Fatalf("Form XObject output = %q", markdown)
	}
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
