package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"local/james-orcales/shared/cli"
)

// Test_Main_Direction_And_Paths verifies case-insensitive PDF detection and
// direction-specific derived and preview suffixes.
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
	if filepath.Ext(main_preview_path("a.PDF")) != ".md" {
		t.Fatal("PDF preview does not use an .md file")
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
