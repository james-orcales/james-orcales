package main

import (
	"bytes"
	"crypto/md5"
	"crypto/rc4"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"local/james-orcales/markdown_to_pdf/internal"
	"local/james-orcales/shared/cli"
)

// Test_Main_Direction_And_Paths verifies case-insensitive PDF detection and
// direction-specific render paths while previews remain PDF documents.
func Test_Main_Direction_And_Paths(t *testing.T) {
	t.Parallel()
	if !main_input_is_pdf("document.PDF") {
		t.Fatal("uppercase PDF suffix was not detected")
	}
	if main_input_is_pdf("document.md") {
		t.Fatal("Markdown input was detected as PDF")
	}
	if got := main_output_path(&Main_Output_Path_Input{Input: "a.PDF"}); got != "a.md" {
		t.Fatalf("PDF output path = %q, want a.md", got)
	}
	if got := main_output_path(&Main_Output_Path_Input{Input: "a.md"}); got != "a.pdf" {
		t.Fatalf("Markdown output path = %q, want a.pdf", got)
	}
	if filepath.Ext(main_preview_path("a.PDF")) != ".pdf" {
		t.Fatal("PDF input does not produce a PDF preview")
	}
}

// Test_Main_Preview_Document verifies both inputs produce a PDF for the system
// previewer and password failures stop before a document exists.
func Test_Main_Preview_Document(t *testing.T) {
	t.Parallel()
	markdown_document, markdown_status := main_preview_document(
		&Main_Preview_Document_Input{Source: []byte("# Title")},
	)
	if markdown_status != 0 {
		t.Fatalf("Markdown preview status = %d", markdown_status)
	}
	if !bytes.HasPrefix(markdown_document, []byte("%PDF-1.")) {
		t.Fatal("Markdown preview is not a PDF")
	}
	pdf_document, pdf_status := main_preview_document(&Main_Preview_Document_Input{
		Source: main_test_r2_pdf(t), Is_PDF: true, Password: []byte("user"),
	})
	if pdf_status != 0 {
		t.Fatalf("PDF preview status = %d", pdf_status)
	}
	if !bytes.HasPrefix(pdf_document, []byte("%PDF-1.")) {
		t.Fatal("PDF input preview is not a PDF")
	}
	failed_document, failed_status := main_preview_document(&Main_Preview_Document_Input{
		Source: main_test_r2_pdf(t), Is_PDF: true, Password: []byte("wrong"),
	})
	if failed_status != EXIT_USAGE {
		t.Fatalf("incorrect preview password status = %d", failed_status)
	}
	if len(failed_document) != 0 {
		t.Fatal("incorrect preview password produced a document")
	}
}

// Test_Main_Help verifies the command describes both conversion directions
// without advertising unsupported standard-output behavior.
func Test_Main_Help(t *testing.T) {
	t.Parallel()
	var output strings.Builder
	program := main_program()
	cli.Print_Help(&output, program)
	help := output.String()
	if !strings.Contains(help, "PDF to Markdown") {
		t.Fatal("help does not describe PDF to Markdown conversion")
	}
	if strings.Contains(help, "standard output") {
		t.Fatal("help advertises unsupported standard-output behavior")
	}
	if strings.Contains(help, "stdout") {
		t.Fatal("help advertises unsupported standard-output behavior")
	}
	if !strings.Contains(help, "password") {
		t.Fatal("help does not describe the PDF password flag")
	}
}

// Test_Main_Password_Usage verifies both conversion commands accept the flag,
// Markdown rejects it with status two, and conversion errors keep distinct statuses.
func Test_Main_Password_Usage(t *testing.T) {
	directory := t.TempDir()
	source_path := filepath.Join(directory, "source.md")
	source, create_err := os.Create(source_path)
	if create_err != nil {
		t.Fatalf("create Markdown: %v", create_err)
	}
	if _, write_err := source.Write([]byte("# Title")); write_err != nil {
		t.Fatalf("write Markdown: %v", write_err)
	}
	if close_err := source.Close(); close_err != nil {
		t.Fatalf("close Markdown: %v", close_err)
	}
	output_path := filepath.Join(directory, "output.pdf")
	program := main_program()
	for _, password := range []string{"secret"} {
		render, render_err := cli.Program_Parse(&program, []string{
			"markdown_to_pdf", "render", source_path,
			"-out=" + output_path, "-password=" + password,
		})
		if render_err != nil {
			t.Fatalf("parse render password: %v", render_err)
		}
		if status := main_render_command(render); status != EXIT_USAGE {
			t.Fatalf("Markdown password %q status = %d", password, status)
		}
	}
	if _, render_err := cli.Program_Parse(&program, []string{
		"markdown_to_pdf", "render", source_path,
		"-out=" + output_path, "-password=",
	}); render_err == nil {
		t.Fatal("empty Markdown password flag parsed")
	}
	if main_path_exists(output_path) {
		t.Fatal("Markdown password created output")
	}
	if main_conversion_error_status(markdown_to_pdf.Pdf_Incorrect_Password_Error{}) !=
		EXIT_USAGE {
		t.Fatal("incorrect password did not map to status two")
	}
	if main_conversion_error_status(errors.New("malformed")) != EXIT_FAILURE {
		t.Fatal("malformed encryption did not map to status one")
	}
	for _, password := range []string{"secret"} {
		preview, preview_err := cli.Program_Parse(&program, []string{
			"markdown_to_pdf", "preview", source_path, "-password=" + password,
		})
		if preview_err != nil {
			t.Fatalf("parse preview password: %v", preview_err)
		}
		if status := main_preview(preview); status != EXIT_USAGE {
			t.Fatalf("Markdown preview password %q status = %d", password, status)
		}
	}
	if _, preview_err := cli.Program_Parse(&program, []string{
		"markdown_to_pdf", "preview", source_path, "-password=",
	}); preview_err == nil {
		t.Fatal("empty Markdown preview password flag parsed")
	}
}

// Test_Main_Incorrect_Password_Preserves_Output verifies authentication finishes
// before the explicit output path opens or changes.
func Test_Main_Incorrect_Password_Preserves_Output(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	output_path := filepath.Join(directory, "output.md")
	output, create_err := os.Create(output_path)
	if create_err != nil {
		t.Fatalf("create output: %v", create_err)
	}
	if _, write_err := output.Write([]byte("sentinel")); write_err != nil {
		t.Fatalf("write output: %v", write_err)
	}
	if close_err := output.Close(); close_err != nil {
		t.Fatalf("close output: %v", close_err)
	}
	status := main_convert_to_path_with_password(&Main_Convert_To_Path_With_Password_Input{
		Source: main_test_r2_pdf(t), Is_PDF: true, Output_Path: output_path,
		Password: []byte("wrong"),
	})
	if status != EXIT_USAGE {
		t.Fatalf("incorrect password status = %d", status)
	}
	if got := read_test_file(t, output_path); !bytes.Equal(got, []byte("sentinel")) {
		t.Fatalf("incorrect password changed output to %q", got)
	}
	absent_path := filepath.Join(directory, "absent.md")
	status = main_convert_to_path_with_password(&Main_Convert_To_Path_With_Password_Input{
		Source: main_test_r2_pdf(t), Is_PDF: true, Output_Path: absent_path,
		Password: []byte("wrong"),
	})
	if status != EXIT_USAGE {
		t.Fatalf("incorrect password absent status = %d", status)
	}
	if main_path_exists(absent_path) {
		t.Fatal("incorrect password created output")
	}
}

func main_test_r2_pdf(t *testing.T) (pdf []byte) {
	t.Helper()
	file_key, key_err := hex.DecodeString("77867d7fae")
	if key_err != nil {
		t.Fatalf("decode file key: %v", key_err)
	}
	key_material := append(append([]byte{}, file_key...), 4, 0, 0, 0, 0)
	object_digest := md5.Sum(key_material)
	stream, stream_err := rc4.NewCipher(object_digest[:10])
	if stream_err != nil {
		t.Fatalf("RC4 key: %v", stream_err)
	}
	plaintext := []byte("BT /F0 12 Tf 1 0 0 1 72 720 Tm (Secret) Tj ET")
	ciphertext := make([]byte, len(plaintext))
	stream.XORKeyStream(ciphertext, plaintext)
	objects := [][]byte{
		[]byte("<< /Type /Catalog /Pages 2 0 R >>"),
		[]byte("<< /Type /Pages /Kids [3 0 R] /Count 1 >>"),
		[]byte("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] " +
			"/Resources << /Font << /F0 5 0 R >> >> /Contents 4 0 R >>"),
		append([]byte(fmt.Sprintf("<< /Length %d >>\nstream\n", len(ciphertext))),
			append(ciphertext, []byte("\nendstream")...)...),
		[]byte("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>"),
		[]byte("<< /Filter /Standard /V 1 /R 2 /P -4 " +
			"/O <94e8094419662a774442fb072e3d9f19e9d130ec09a4d0061e78fe920f7ab62f> " +
			"/U <b1f625689bca1356d125453c43cfa750c0296b5c271c03f1cc359fa658a4fa21> >>"),
	}
	return main_test_classic_pdf(objects)
}

func main_test_classic_pdf(objects [][]byte) (pdf []byte) {
	var output bytes.Buffer
	output.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects)+1)
	for object_index, object := range objects {
		offsets[object_index+1] = output.Len()
		fmt.Fprintf(&output, "%d 0 obj\n", object_index+1)
		output.Write(object)
		output.WriteString("\nendobj\n")
	}
	xref_size := output.Len()
	fmt.Fprintf(&output, "xref\n0 %d\n", len(offsets))
	output.WriteString("0000000000 65535 f \n")
	for object_number_index := 1; object_number_index < len(offsets); object_number_index++ {
		fmt.Fprintf(&output, "%010d 00000 n \n", offsets[object_number_index])
	}
	fmt.Fprintf(
		&output,
		"trailer\n<< /Size 7 /Root 1 0 R /Encrypt 6 0 R "+
			"/ID [<00112233445566778899aabbccddeeff> "+
			"<00112233445566778899aabbccddeeff>] >>\n"+
			"startxref\n%d\n%%%%EOF\n",
		xref_size,
	)
	return output.Bytes()
}

// Test_Main_Output_Preservation verifies PDF parsing finishes before the output
// opens, so an invalid PDF cannot truncate an existing explicit path.
func Test_Main_Output_Preservation(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	path := filepath.Join(directory, "existing.md")
	file, create_err := os.Create(path)
	if create_err != nil {
		t.Fatalf("create sentinel: %v", create_err)
	}
	if _, write_err := file.Write([]byte("sentinel")); write_err != nil {
		t.Fatalf("write sentinel: %v", write_err)
	}
	if close_err := file.Close(); close_err != nil {
		t.Fatalf("close sentinel: %v", close_err)
	}
	status := main_convert_to_path([]byte("not a PDF"), true, path)
	if status != EXIT_FAILURE {
		t.Fatalf("status = %d, want %d", status, EXIT_FAILURE)
	}
	if got := read_test_file(t, path); !bytes.Equal(got, []byte("sentinel")) {
		t.Fatalf("existing output changed to %q", got)
	}
}

// Test_Main_Conversion_Directions verifies Markdown still renders to PDF and
// PDF input extracts to Markdown through the common output transaction.
func Test_Main_Conversion_Directions(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	pdf_path := filepath.Join(directory, "rendered.pdf")
	if status := main_convert_to_path([]byte("# Hello"), false, pdf_path); status != 0 {
		t.Fatalf("Markdown status = %d", status)
	}
	if !bytes.HasPrefix(read_test_file(t, pdf_path), []byte("%PDF-1.")) {
		t.Fatal("Markdown conversion did not write a PDF")
	}
	source := read_test_file(t, "internal/testdata/movie_booking.pdf")
	markdown_path := filepath.Join(directory, "extracted.md")
	if status := main_convert_to_path(source, true, markdown_path); status != 0 {
		t.Fatalf("PDF status = %d", status)
	}
	want := read_test_file(t, "internal/testdata/movie_booking.golden")
	if !bytes.Equal(read_test_file(t, markdown_path), want) {
		t.Fatal("PDF conversion output differs from its golden")
	}
}

// Test_Main_Overwrite_Statuses verifies derived outputs are protected, while
// explicit outputs are replaced and ordinary read failures return status one.
func Test_Main_Overwrite_Statuses(t *testing.T) {
	directory := t.TempDir()
	program := main_program()
	source_path := filepath.Join(directory, "source.PDF")
	derived_path := filepath.Join(directory, "source.md")
	derived, create_err := os.Create(derived_path)
	if create_err != nil {
		t.Fatalf("create derived output: %v", create_err)
	}
	if close_err := derived.Close(); close_err != nil {
		t.Fatalf("close derived output: %v", close_err)
	}
	command, parse_err := cli.Program_Parse(
		&program, []string{"markdown_to_pdf", "render", source_path},
	)
	if parse_err != nil {
		t.Fatalf("parse derived command: %v", parse_err)
	}
	if status := main_render_command(command); status != EXIT_EXISTS {
		t.Fatalf("derived collision status = %d", status)
	}

	source, source_create_err := os.Create(source_path)
	if source_create_err != nil {
		t.Fatalf("create PDF source: %v", source_create_err)
	}
	if _, write_err := source.Write(
		read_test_file(t, "internal/testdata/movie_booking.pdf"),
	); write_err != nil {
		t.Fatalf("write PDF source: %v", write_err)
	}
	if close_err := source.Close(); close_err != nil {
		t.Fatalf("close PDF source: %v", close_err)
	}
	explicit_path := filepath.Join(directory, "explicit.md")
	explicit, explicit_create_err := os.Create(explicit_path)
	if explicit_create_err != nil {
		t.Fatalf("create explicit output: %v", explicit_create_err)
	}
	if _, write_err := explicit.Write([]byte("sentinel")); write_err != nil {
		t.Fatalf("write explicit output: %v", write_err)
	}
	if close_err := explicit.Close(); close_err != nil {
		t.Fatalf("close explicit output: %v", close_err)
	}
	command, parse_err = cli.Program_Parse(&program, []string{
		"markdown_to_pdf", "render", source_path, "-out=" + explicit_path,
	})
	if parse_err != nil {
		t.Fatalf("parse explicit command: %v", parse_err)
	}
	if status := main_render_command(command); status != 0 {
		t.Fatalf("explicit output status = %d", status)
	}
	if bytes.Equal(read_test_file(t, explicit_path), []byte("sentinel")) {
		t.Fatal("explicit output was not replaced")
	}

	missing_path := filepath.Join(directory, "missing.pdf")
	command, parse_err = cli.Program_Parse(&program, []string{
		"markdown_to_pdf", "render", missing_path,
		"-out=" + filepath.Join(directory, "missing.md"),
	})
	if parse_err != nil {
		t.Fatalf("parse missing command: %v", parse_err)
	}
	if status := main_render_command(command); status != EXIT_FAILURE {
		t.Fatalf("read failure status = %d", status)
	}
}

func read_test_file(t *testing.T, path string) (contents []byte) {
	t.Helper()
	file, open_err := os.Open(path)
	if open_err != nil {
		t.Fatalf("open %s: %v", path, open_err)
	}
	defer file.Close()
	info, stat_err := file.Stat()
	if stat_err != nil {
		t.Fatalf("stat %s: %v", path, stat_err)
	}
	contents = make([]byte, info.Size())
	if _, read_err := io.ReadFull(file, contents); read_err != nil {
		t.Fatalf("read %s: %v", path, read_err)
	}
	return contents
}
